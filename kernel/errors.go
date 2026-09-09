// The kernel error taxonomy (doc 38 §3.2 row 6): a typed *APIError carrying
// the verbatim response (status / headers / body / x-request-id), reachable
// with errors.As, plus status-family SENTINELS reachable with errors.Is
// (`errors.Is(err, kernel.ErrRateLimit)`), and the non-HTTP failures —
// *ConnectionError (network) with a timeout flavor (ErrConnectionTimeout).
// Mirrors the TS kernel's errors.ts / the Python kernel's _errors.py field
// for field.
//
// Vendored kernel file — imports only the stdlib.

package kernel

import (
	"errors"
	"fmt"
	"time"
)

// Status-family sentinels: `errors.Is(err, kernel.ErrNotFound)` matches any
// *APIError whose status falls in that family (ErrInternalServer matches the
// whole 5XX family, mirroring the Python taxonomy's InternalServerError).
var (
	ErrBadRequest          = errors.New("doctorine: bad request (HTTP 400)")
	ErrAuthentication      = errors.New("doctorine: authentication failed (HTTP 401)")
	ErrPermissionDenied    = errors.New("doctorine: permission denied (HTTP 403)")
	ErrNotFound            = errors.New("doctorine: not found (HTTP 404)")
	ErrConflict            = errors.New("doctorine: conflict (HTTP 409)")
	ErrUnprocessableEntity = errors.New("doctorine: unprocessable entity (HTTP 422)")
	ErrRateLimit           = errors.New("doctorine: rate limited (HTTP 429)")
	ErrInternalServer      = errors.New("doctorine: internal server error (HTTP 5XX)")
)

// ErrConnection matches any *ConnectionError; ErrConnectionTimeout matches
// only the per-request-timeout flavor (both via errors.Is).
var (
	ErrConnection        = errors.New("doctorine: connection error")
	ErrConnectionTimeout = errors.New("doctorine: connection timeout")
)

// APIError is every non-2xx HTTP response the API returned. Reach it with
// `var apiErr *kernel.APIError; errors.As(err, &apiErr)`.
type APIError struct {
	// Status is the HTTP status code.
	Status int
	// Header holds the response headers, keys lower-cased.
	Header map[string]string
	// Body is the parsed (JSON) or raw (string) error body, captured verbatim.
	Body any
	// RequestID is the `x-request-id` response header, when the server sent one.
	RequestID string
	// Message is the human-readable error text (defaulted from the body).
	Message string
}

// NewAPIError builds an *APIError; an empty message is defaulted from
// common `{error:{message}}` body shapes.
func NewAPIError(status int, header map[string]string, body any, message string) *APIError {
	if header == nil {
		header = map[string]string{}
	}
	if message == "" {
		message = defaultErrorMessage(status, body)
	}
	return &APIError{
		Status:    status,
		Header:    header,
		Body:      body,
		RequestID: header["x-request-id"],
		Message:   message,
	}
}

// Error implements the error interface.
func (e *APIError) Error() string { return e.Message }

// Is maps the status onto the family sentinels so errors.Is works without
// subclassing: 400→ErrBadRequest … 429→ErrRateLimit, >=500→ErrInternalServer.
func (e *APIError) Is(target error) bool {
	sentinel, ok := statusSentinels[e.Status]
	if !ok && e.Status >= 500 && e.Status < 600 {
		sentinel = ErrInternalServer
	}
	return sentinel != nil && target == sentinel
}

var statusSentinels = map[int]error{
	400: ErrBadRequest,
	401: ErrAuthentication,
	403: ErrPermissionDenied,
	404: ErrNotFound,
	409: ErrConflict,
	422: ErrUnprocessableEntity,
	429: ErrRateLimit,
}

// ConnectionError is a request that never produced an HTTP response (DNS,
// TLS, socket, per-request timeout, ...). Reach it with errors.As; test the
// timeout flavor with `errors.Is(err, kernel.ErrConnectionTimeout)` or
// Timeout().
type ConnectionError struct {
	// Message describes the failure.
	Message string
	// Cause is the underlying transport error, when one exists (Unwrap).
	Cause error
	// TimedOut is true when the per-request timeout elapsed.
	TimedOut bool
	// TimeoutAfter is the timeout that elapsed (zero unless TimedOut).
	TimeoutAfter time.Duration
}

// NewConnectionError wraps a transport failure that was not a timeout.
func NewConnectionError(message string, cause error) *ConnectionError {
	return &ConnectionError{Message: message, Cause: cause}
}

// NewConnectionTimeoutError marks a per-request timeout.
func NewConnectionTimeoutError(timeout time.Duration) *ConnectionError {
	return &ConnectionError{
		Message:      fmt.Sprintf("request timed out after %s", timeout),
		TimedOut:     true,
		TimeoutAfter: timeout,
	}
}

// Error implements the error interface.
func (e *ConnectionError) Error() string { return e.Message }

// Unwrap exposes the underlying transport error to errors.Is/As chains.
func (e *ConnectionError) Unwrap() error { return e.Cause }

// Timeout reports whether the failure was the per-request timeout (the
// net.Error convention).
func (e *ConnectionError) Timeout() bool { return e.TimedOut }

// Is matches the ErrConnection sentinel (and ErrConnectionTimeout for the
// timeout flavor) so callers can branch without errors.As.
func (e *ConnectionError) Is(target error) bool {
	if target == ErrConnection {
		return true
	}
	return target == ErrConnectionTimeout && e.TimedOut
}

// defaultErrorMessage builds a best-effort human message from common
// `{error:{message}}` bodies.
func defaultErrorMessage(status int, body any) string {
	detail := extractMessage(body)
	if detail == "" {
		return fmt.Sprintf("HTTP %d", status)
	}
	return fmt.Sprintf("HTTP %d: %s", status, detail)
}

func extractMessage(body any) string {
	switch value := body.(type) {
	case string:
		return value
	case map[string]any:
		if nested, ok := value["error"].(string); ok && nested != "" {
			return nested
		}
		if errObj, ok := value["error"].(map[string]any); ok {
			if nested, ok := errObj["message"].(string); ok && nested != "" {
				return nested
			}
		}
		if message, ok := value["message"].(string); ok && message != "" {
			return message
		}
	}
	return ""
}
