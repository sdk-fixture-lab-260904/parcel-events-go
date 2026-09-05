// The core client (doc 38 §1 "Emitters + kernels" row): the net/http-based
// transport orchestrator every generated resource method calls into.
// context.Context is the FIRST parameter of every call; configuration is
// functional options at both levels (ClientOption on the client,
// RequestOption per request). One request = prepare (URL, headers, JSON
// body, ONE idempotency key) → the retry loop (auth per attempt /
// per-attempt timeout / backoff with Retry-After / one transparent re-auth
// on 401) → typed error or parsed body. Also the PageFetcher pagination
// iterates through, and the raw response source streaming reads from.
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the transport orchestrator. Build one with NewClient.
type Client struct {
	baseURL       string
	auth          AuthProvider
	httpClient    *http.Client
	timeout       time.Duration
	retry         RetryPolicy
	defaultHeader map[string]string
	sdkName       string
	sdkVersion    string
	idempotency   IdempotencyConfig
	sleep         func(ctx context.Context, delay time.Duration) error
	now           func() time.Time
}

// ClientOption configures a Client (the functional options pattern).
type ClientOption func(*Client)

// WithAuth installs the auth provider consulted on every attempt.
func WithAuth(provider AuthProvider) ClientOption { return func(c *Client) { c.auth = provider } }

// WithHTTPClient swaps the underlying *http.Client (transports, proxies,
// test servers).
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithTimeout sets the default per-request timeout (config
// `timeouts.default_ms`).
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) { c.timeout = timeout }
}

// WithRetryPolicy replaces the whole retry policy (start from
// DefaultRetryPolicy()).
func WithRetryPolicy(policy RetryPolicy) ClientOption { return func(c *Client) { c.retry = policy } }

// WithDefaultHeader stamps a header onto every request.
func WithDefaultHeader(name string, value string) ClientOption {
	return func(c *Client) { c.defaultHeader[strings.ToLower(name)] = value }
}

// WithSDKIdentity sets the generated SDK's name and sdkSha-stamped version,
// folded into the User-Agent next to the kernel token.
func WithSDKIdentity(name string, version string) ClientOption {
	return func(c *Client) { c.sdkName, c.sdkVersion = name, version }
}

// WithIdempotency configures idempotency-key injection.
func WithIdempotency(config IdempotencyConfig) ClientOption {
	return func(c *Client) { c.idempotency = config }
}

// WithClock injects the clock and sleeper (tests); production uses
// time.Now and a ctx-aware timer sleep.
func WithClock(now func() time.Time, sleep func(ctx context.Context, delay time.Duration) error) ClientOption {
	return func(c *Client) { c.now, c.sleep = now, sleep }
}

// NewClient builds a Client for a base URL with the kernel defaults (60s
// timeout, default-on retries, no auth).
func NewClient(baseURL string, options ...ClientOption) *Client {
	client := &Client{
		baseURL:       strings.TrimRight(baseURL, "/"),
		httpClient:    &http.Client{},
		timeout:       DefaultTimeout,
		retry:         DefaultRetryPolicy(),
		defaultHeader: map[string]string{},
		sdkName:       "doctorine-sdk",
		sdkVersion:    "0.0.0",
		sleep:         sleepWithContext,
		now:           time.Now,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

// Response is one parsed API response.
type Response struct {
	Status int
	Header map[string]string
	// Data is the parsed body: JSON → any, text → string, else []byte.
	Data any
}

// Do performs a request through the full auth/retry pipeline and parses the
// body. Non-2xx responses surface as *APIError (errors.As / errors.Is).
func (c *Client) Do(ctx context.Context, method string, path string, options ...RequestOption) (*Response, error) {
	response, err := c.DoRaw(ctx, method, path, options...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, NewConnectionError("connection error while reading the response body", err)
	}
	data, err := ParseResponseBody(response.StatusCode, response.Header.Get("Content-Type"), raw)
	if err != nil {
		return nil, err
	}
	return &Response{Status: response.StatusCode, Header: headerToMap(response.Header), Data: data}, nil
}

// DoRaw runs the full auth/retry pipeline and returns the UNREAD ok
// *http.Response — the entry point streaming and download methods build on
// (`kernel.NewSSEStreamFromResponse(client.DoRaw(...))`). The caller owns
// Body.Close.
func (c *Client) DoRaw(ctx context.Context, method string, path string, options ...RequestOption) (*http.Response, error) {
	config := newRequestConfig(options)
	prepared, err := c.prepare(method, path, config)
	if err != nil {
		return nil, err
	}
	start := c.now()
	attempt := 0
	reauthorized := false
	for {
		response, attemptErr := c.attemptOnce(ctx, prepared)
		if attemptErr == nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			return response, nil
		}
		if attemptErr != nil && ctx.Err() != nil {
			return nil, attemptErr // user cancellation/deadline — terminal, never retried
		}
		status, retryAfterHeader := 0, ""
		if attemptErr == nil {
			status, retryAfterHeader = response.StatusCode, response.Header.Get("Retry-After")
		}
		if status == http.StatusUnauthorized && !reauthorized && c.invalidateAuth() {
			reauthorized = true
			discardBody(response)
			continue
		}
		decision := c.decide(status, retryAfterHeader, attempt, c.now().Sub(start))
		if !decision.Retry || !prepared.body.replayable() {
			if attemptErr != nil {
				return nil, attemptErr
			}
			return nil, errorFromResponse(response)
		}
		discardBody(response)
		if sleepErr := c.sleep(ctx, decision.Delay); sleepErr != nil {
			return nil, sleepErr
		}
		attempt++
	}
}

// RequestPage is the pagination hook (PageFetcher).
func (c *Client) RequestPage(ctx context.Context, request PageRequest) (any, error) {
	options := make([]RequestOption, 0, len(request.Query)+len(request.Header)+1)
	for _, param := range request.Query {
		options = append(options, WithQueryStyle(param.Name, param.Value, param.Style))
	}
	for name, value := range request.Header {
		options = append(options, WithHeader(name, value))
	}
	if request.Body != nil {
		options = append(options, WithJSONBody(request.Body))
	}
	response, err := c.Do(ctx, request.Method, request.URL, options...)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

// attemptOnce folds one attempt's fresh auth material into the prepared
// request and performs one send. Auth failures are terminal (mirrors the
// TS/Python kernels: an oauth fetch failure never enters the retry loop).
func (c *Client) attemptOnce(ctx context.Context, prepared *preparedRequest) (*http.Response, error) {
	material := AuthMaterial{}
	if c.auth != nil {
		var err error
		if material, err = c.auth.Authorize(ctx); err != nil {
			return nil, err
		}
	}
	requestURL := prepared.url
	if len(material.Query) > 0 {
		encoded, err := EncodeQuery(material.Query)
		if err != nil {
			return nil, err
		}
		requestURL = AppendQuery(requestURL, encoded)
	}
	header := make(map[string]string, len(prepared.header)+len(material.Header))
	for name, value := range prepared.header {
		header[name] = value
	}
	for name, value := range material.Header {
		header[strings.ToLower(name)] = value
	}
	body, err := prepared.body.attempt()
	if err != nil {
		return nil, err
	}
	return sendAttempt(ctx, c.httpClient, prepared.method, requestURL, header, body, prepared.timeout)
}

func (c *Client) decide(status int, retryAfterHeader string, attempt int, elapsed time.Duration) RetryDecision {
	retryAfter, hasRetryAfter := ParseRetryAfter(retryAfterHeader, c.now())
	return NextRetryDecision(c.retry, attempt, elapsed, status, retryAfter, hasRetryAfter)
}

func (c *Client) invalidateAuth() bool {
	if c.auth == nil {
		return false
	}
	return c.auth.Invalidate()
}

// errorFromResponse turns a non-2xx response into the most specific typed
// error, reading and closing the body.
func errorFromResponse(response *http.Response) error {
	defer func() { _ = response.Body.Close() }()
	raw, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		raw = nil // an unreadable error body is still an APIError
	}
	body, parseErr := ParseResponseBody(response.StatusCode, response.Header.Get("Content-Type"), raw)
	if parseErr != nil {
		body = string(raw)
	}
	return NewAPIError(response.StatusCode, headerToMap(response.Header), body, "")
}

// discardBody drains a little and closes, so the connection can be reused.
func discardBody(response *http.Response) {
	if response == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	_ = response.Body.Close()
}

// sleepWithContext waits out a backoff delay, honoring cancellation.
func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// preparedBody is one logical request's body, replayable across retries
// when it is buffered bytes or a seekable reader.
type preparedBody struct {
	buffered []byte
	reader   io.Reader
	used     bool
}

func (b *preparedBody) attempt() (io.Reader, error) {
	if b.buffered != nil {
		return bytes.NewReader(b.buffered), nil
	}
	if b.reader == nil {
		return nil, nil
	}
	if !b.used {
		b.used = true
		return b.reader, nil
	}
	seeker, isSeekable := b.reader.(io.Seeker)
	if !isSeekable {
		return nil, NewConnectionError("request body is a one-shot io.Reader and cannot be replayed", nil)
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return nil, NewConnectionError("failed to rewind the request body for a retry", err)
	}
	return b.reader, nil
}

func (b *preparedBody) replayable() bool {
	if b.reader == nil || !b.used {
		return true
	}
	_, isSeekable := b.reader.(io.Seeker)
	return isSeekable
}
