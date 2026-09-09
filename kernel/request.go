// Per-request configuration (the second half of the functional-options
// pattern): RequestOption values collected by Client.Do/DoRaw, and the
// preparation step that resolves them into ONE immutable prepared request
// BEFORE the retry loop — so the URL, headers, body bytes, and the
// idempotency key are stable across every retry of a logical call.
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import (
	"io"
	"strings"
	"time"
)

type requestConfig struct {
	parameters       []serializedParameter
	query            []QueryParam
	header           map[string]string
	jsonBody         any
	hasJSONBody      bool
	rawBody          []byte
	readerBody       io.Reader
	contentType      string
	multipart        []MultipartField
	hasMultipart     bool
	timeout          time.Duration
	idempotencyKey   string
	validation       *bool
	bodyContract     Contract
	responseContract Contract
	eventContract    Contract
}

// RequestOption configures one request (the functional options pattern).
type RequestOption func(*requestConfig)

// WithQuery adds one query param in the default `form` style. Order is
// preserved — identical calls yield identical URLs.
func WithQuery(name string, value any) RequestOption {
	return func(r *requestConfig) { r.query = append(r.query, QueryParam{Name: name, Value: value}) }
}

// WithQueryStyle adds one query param with an explicit OpenAPI style.
func WithQueryStyle(name string, value any, style QueryStyle) RequestOption {
	return func(r *requestConfig) {
		r.query = append(r.query, QueryParam{Name: name, Value: value, Style: style})
	}
}

// WithHeader sets one request header.
func WithHeader(name string, value string) RequestOption {
	return func(r *requestConfig) { r.header[strings.ToLower(name)] = value }
}

// WithJSONBody JSON-encodes a request body (content-type application/json).
func WithJSONBody(value any) RequestOption {
	return func(r *requestConfig) { r.jsonBody, r.hasJSONBody = value, true }
}

// WithRawBody sends pre-encoded bytes with an explicit content type
// (replayable across retries by construction).
func WithRawBody(contentType string, body []byte) RequestOption {
	return func(r *requestConfig) { r.rawBody, r.contentType = body, contentType }
}

// WithReaderBody streams an upload from an io.Reader. A reader that also
// implements io.Seeker is rewound for retries; a one-shot reader makes the
// request non-retryable after its first attempt.
func WithReaderBody(contentType string, body io.Reader) RequestOption {
	return func(r *requestConfig) { r.readerBody, r.contentType = body, contentType }
}

// WithMultipartBody sends `multipart/form-data` fields (scalar values and
// io.Reader UploadParts), buffered so retries can replay them.
func WithMultipartBody(fields ...MultipartField) RequestOption {
	return func(r *requestConfig) { r.multipart, r.hasMultipart = fields, true }
}

// WithRequestTimeout overrides the client's per-request timeout for this
// call.
func WithRequestTimeout(timeout time.Duration) RequestOption {
	return func(r *requestConfig) { r.timeout = timeout }
}

// WithIdempotencyKey pins this request's idempotency key (honored on any
// method; wins over auto-generation).
func WithIdempotencyKey(key string) RequestOption {
	return func(r *requestConfig) { r.idempotencyKey = key }
}

func newRequestConfig(options []RequestOption) *requestConfig {
	config := &requestConfig{header: map[string]string{}}
	for _, option := range options {
		option(config)
	}
	return config
}

// preparedRequest is one logical request, fully resolved BEFORE the retry
// loop (so the idempotency key is stable across every retry of this call).
type preparedRequest struct {
	method  string
	url     string
	header  map[string]string
	body    preparedBody
	timeout time.Duration
}

func (c *Client) prepare(method string, path string, config *requestConfig) (*preparedRequest, error) {
	path, bindErr := bindParameters(path, config)
	if bindErr != nil {
		return nil, bindErr
	}
	if c.validationEnabled(config) && (config.hasJSONBody || config.bodyContract.Schema != nil) {
		if err := Validate(config.jsonBody, config.bodyContract); err != nil {
			return nil, err
		}
	}
	header := map[string]string{
		"accept":     "application/json",
		"user-agent": UserAgent(c.sdkName, c.sdkVersion),
	}
	for name, value := range c.defaultHeader {
		header[strings.ToLower(name)] = value
	}
	for name, value := range config.header {
		header[name] = value
	}
	body, err := resolveBody(config, header)
	if err != nil {
		return nil, err
	}
	// ONE key per logical request — retries of this call all reuse it.
	if key := ResolveIdempotencyKey(method, config.idempotencyKey, c.idempotency); key != "" {
		header[strings.ToLower(c.idempotency.headerName())] = key
	}
	requestURL, err := c.buildURL(path, config.query)
	if err != nil {
		return nil, err
	}
	timeout := config.timeout
	if timeout == 0 {
		timeout = c.timeout
	}
	return &preparedRequest{method: method, url: requestURL, header: header, body: body, timeout: timeout}, nil
}

func resolveBody(config *requestConfig, header map[string]string) (preparedBody, error) {
	switch {
	case config.hasMultipart:
		encoded, contentType, err := EncodeMultipart(config.multipart)
		if err != nil {
			return preparedBody{}, err
		}
		header["content-type"] = contentType
		return preparedBody{buffered: encoded}, nil
	case config.hasJSONBody:
		encoded, contentType, err := JSONBody(config.jsonBody)
		if err != nil {
			return preparedBody{}, err
		}
		setDefaultHeader(header, "content-type", contentType)
		return preparedBody{buffered: encoded}, nil
	case config.rawBody != nil:
		setDefaultHeader(header, "content-type", config.contentType)
		return preparedBody{buffered: config.rawBody}, nil
	case config.readerBody != nil:
		setDefaultHeader(header, "content-type", config.contentType)
		return preparedBody{reader: config.readerBody}, nil
	default:
		return preparedBody{}, nil
	}
}

func setDefaultHeader(header map[string]string, name string, value string) {
	if _, exists := header[name]; !exists && value != "" {
		header[name] = value
	}
}

func (c *Client) buildURL(path string, query []QueryParam) (string, error) {
	base := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		separator := "/"
		if strings.HasPrefix(path, "/") {
			separator = ""
		}
		base = c.baseURL + separator + path
	}
	encoded, err := EncodeQuery(query)
	if err != nil {
		return "", err
	}
	return AppendQuery(base, encoded), nil
}
