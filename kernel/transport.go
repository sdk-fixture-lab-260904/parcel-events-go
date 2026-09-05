// Low-level transport glue over net/http: the mapping from transport
// failures onto the kernel error taxonomy, header flattening, response body
// parsing (JSON → value, text → string, else bytes), and the one-attempt
// send with a headers-deadline timeout. The per-attempt timeout covers the
// exchange up to response headers; the body read is governed by the
// caller's context, so long-lived streams are never killed by the request
// timeout (httpx's read-timeout shape, one runtime over). One send is one
// attempt; the retry loop lives in the client.
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import (
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
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("kernel: decode JSON response body: %w", err)
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
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

// sendAttempt performs ONE HTTP exchange: per-attempt timeout as a
// headers-deadline timer (stopped once the response arrives, so body reads
// answer only to ctx), taxonomy-mapped failures, and a body wrapper that
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
	timer.Stop()
	if err != nil {
		cancel()
		if timedOut.Load() {
			return nil, NewConnectionTimeoutError(timeout)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, mapTransportError(err, timeout)
	}
	response.Body = &cancelOnClose{ReadCloser: response.Body, cancel: cancel}
	return response, nil
}
