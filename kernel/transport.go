// Low-level transport glue over net/http: the mapping from transport
// failures onto the kernel error taxonomy, header flattening, response body
// parsing (JSON → value, text → string, else bytes), and the one-attempt
// send with a per-attempt deadline covering headers and buffered bodies.
// Successful DoRaw streams release that deadline at headers; their lifetime
// remains governed by the caller context. The retry loop lives in the client.
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// DefaultTimeout is the per-request timeout used when the config sets none
// (config `timeouts.default_ms`).
const DefaultTimeout = 60 * time.Second

// headerToMap flattens response headers into a plain lower-cased map
// (multi-valued headers join with ", ", the fetch Headers semantics).
func headerToMap(header http.Header) map[string]string {
	out := make(map[string]string, len(header))
	for key, values := range header {
		out[strings.ToLower(key)] = strings.Join(values, ", ")
	}
	return out
}

// mapTransportError maps a net/http transport failure onto the kernel
// taxonomy: a timeout → the ErrConnectionTimeout flavor; anything else
// without an HTTP response → *ConnectionError.
func mapTransportError(err error, timeout time.Duration) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return NewConnectionTimeoutError(timeout)
	}
	return NewConnectionError("connection error while contacting the API", err)
}

// ParseResponseBody parses a fully-read response body by content type:
// JSON → any, text → string, else []byte. 204 → nil; an empty JSON body →
// nil.
func ParseResponseBody(status int, contentType string, raw []byte) (any, error) {
	if status == http.StatusNoContent {
		return nil, nil
	}
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "+json") {
		if len(raw) == 0 {
			return nil, nil
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("kernel: decode JSON response body: %w", err)
		}
		if decoder.Decode(new(any)) != io.EOF {
			return nil, fmt.Errorf("kernel: trailing JSON response data")
		}
		return value, nil
	}
	if strings.HasPrefix(contentType, "text/") || contentType == "" {
		return string(raw), nil
	}
	return raw, nil
}

// cancelOnClose ties an attempt's context to the response body's lifetime:
// closing the body releases the attempt context (sendAttempt hands the
// caller an open stream whose cancel it can no longer call itself).
type cancelOnClose struct {
	io.ReadCloser
	cancel    context.CancelFunc
	stopTimer func() bool
	timedOut  *atomic.Bool
	parent    context.Context
	timeout   time.Duration
}

func (c *cancelOnClose) Close() error {
	c.stopTimer()
	c.cancel()
	return c.ReadCloser.Close()
}

func (c *cancelOnClose) Read(buffer []byte) (int, error) {
	n, err := c.ReadCloser.Read(buffer)
	if err != nil && err != io.EOF {
		if c.parent.Err() != nil {
			return n, c.parent.Err()
		}
		if c.timedOut.Load() {
			return n, NewConnectionTimeoutError(c.timeout)
		}
		return n, mapTransportError(err, c.timeout)
	}
	return n, err
}

// sendAttempt performs ONE HTTP exchange: per-attempt timeout as a
// deadline timer, taxonomy-mapped failures, and a body wrapper that
// releases the attempt context on Close. Caller context errors pass
// through untranslated — the retry loop treats them as terminal.
func sendAttempt(
	ctx context.Context,
	client *http.Client,
	method string,
	url string,
	header map[string]string,
	body io.Reader,
	timeout time.Duration,
) (*http.Response, error) {
	attemptCtx, cancel := context.WithCancel(ctx)
	var timedOut atomic.Bool
	timer := time.AfterFunc(timeout, func() {
		timedOut.Store(true)
		cancel()
	})
	request, err := http.NewRequestWithContext(attemptCtx, method, url, body)
	if err != nil {
		timer.Stop()
		cancel()
		return nil, NewConnectionError("invalid request: "+err.Error(), err)
	}
	for key, value := range header {
		request.Header.Set(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		timer.Stop()
		cancel()
		if timedOut.Load() {
			return nil, NewConnectionTimeoutError(timeout)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, mapTransportError(err, timeout)
	}
	response.Body = &cancelOnClose{ReadCloser: response.Body, cancel: cancel, stopTimer: timer.Stop, timedOut: &timedOut, parent: ctx, timeout: timeout}
	return response, nil
}
