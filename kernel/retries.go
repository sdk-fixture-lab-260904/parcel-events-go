// Retry policy + backoff math (doc 38 §3.2 row 4). Retries are ON BY
// DEFAULT (max 2), mirroring the `retries:` block of `doctorine.sdk.yml`
// field for field — config only tunes what the kernel already does. Backoff
// is pure exponential WITHOUT jitter: the kernel never draws randomness, so
// retry schedules are reproducible (the determinism doctrine extended to
// runtime). `Retry-After` (delta-seconds or HTTP-date) is respected when the
// server sends it.
//
// Vendored kernel file — everything here is a pure function (stdlib only).

package kernel

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// StatusMatcher is a literal status code ("429") or a whole family
// ("4XX" / "5XX"), the verbatim grammar of the config's `status_codes:`.
type StatusMatcher string

// Matches reports whether a response status falls under this matcher.
func (m StatusMatcher) Matches(status int) bool {
	switch m {
	case "4XX":
		return status >= 400 && status < 500
	case "5XX":
		return status >= 500 && status < 600
	default:
		code, err := strconv.Atoi(string(m))
		return err == nil && code == status
	}
}

// RetryPolicy mirrors the `retries:` defaults in the sdk-config schema.
// Start from DefaultRetryPolicy() and overwrite fields — the zero value is
// NOT a usable policy.
type RetryPolicy struct {
	Enabled               bool
	MaxRetries            int
	InitialDelay          time.Duration
	MaxDelay              time.Duration
	MaxElapsed            time.Duration
	Exponent              float64
	StatusCodes           []StatusMatcher
	RetryConnectionErrors bool
	// RetryUnsafeRequests explicitly opts non-idempotent writes into automatic replay.
	RetryUnsafeRequests bool
}

// DefaultRetryPolicy is the kernel's default-on policy (max 2 retries,
// 500ms → exponential ×2 capped at 8s, 60s total budget, 408/429/5XX +
// connection errors).
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		Enabled:               true,
		MaxRetries:            2,
		InitialDelay:          500 * time.Millisecond,
		MaxDelay:              8 * time.Second,
		MaxElapsed:            60 * time.Second,
		Exponent:              2,
		StatusCodes:           []StatusMatcher{"408", "429", "5XX"},
		RetryConnectionErrors: true,
	}
}

// RetryDecision is the retry loop's verdict for one failed attempt.
type RetryDecision struct {
	Retry bool
	Delay time.Duration
}

// IsRetryable reports whether a failure is retryable under the policy.
// status == 0 means the attempt failed without an HTTP response (a
// connection error — HTTP status codes are never 0).
func IsRetryable(policy RetryPolicy, status int) bool {
	if status == 0 {
		return policy.RetryConnectionErrors
	}
	for _, matcher := range policy.StatusCodes {
		if matcher.Matches(status) {
			return true
		}
	}
	return false
}

// BackoffDelay is the delay before retry number retryIndex (0-based). A
// server Retry-After (hasRetryAfter) overrides the exponential schedule
// (clamped to [0, MaxElapsed]); otherwise initial·exponentⁱ capped at
// MaxDelay.
func BackoffDelay(policy RetryPolicy, retryIndex int, retryAfter time.Duration, hasRetryAfter bool) time.Duration {
	if hasRetryAfter {
		return min(max(retryAfter, 0), policy.MaxElapsed)
	}
	delay := float64(policy.InitialDelay) * math.Pow(policy.Exponent, float64(retryIndex))
	return min(time.Duration(delay), policy.MaxDelay)
}

// NextRetryDecision is the single retry-decision entry point the client
// transport consults. attempt is the 0-based count of retries already
// performed; elapsed is wall-clock time since the first attempt started;
// status == 0 means no HTTP response arrived.
func NextRetryDecision(
	policy RetryPolicy,
	attempt int,
	elapsed time.Duration,
	status int,
	retryAfter time.Duration,
	hasRetryAfter bool,
) RetryDecision {
	if !policy.Enabled || attempt >= policy.MaxRetries || !IsRetryable(policy, status) {
		return RetryDecision{}
	}
	delay := BackoffDelay(policy, attempt, retryAfter, hasRetryAfter)
	if elapsed+delay > policy.MaxElapsed {
		return RetryDecision{}
	}
	return RetryDecision{Retry: true, Delay: delay}
}

// ParseRetryAfter parses a Retry-After header value: delta-seconds ("3") or
// an IMF-fixdate ("Wed, 01 Jul 2026 10:00:00 GMT"), returned as a
// non-negative duration from now. Unparseable values yield ok == false
// (fall back to backoff).
func ParseRetryAfter(value string, now time.Time) (delay time.Duration, ok bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	if seconds, err := strconv.ParseUint(trimmed, 10, 32); err == nil {
		return time.Duration(seconds) * time.Second, true
	}
	at, err := http.ParseTime(trimmed)
	if err != nil {
		return 0, false
	}
	return max(at.Sub(now), 0), true
}

func safeToRetry(method string, key string, policy RetryPolicy) bool {
	if strings.TrimSpace(key) != "" || policy.RetryUnsafeRequests {
		return true
	}
	switch strings.ToUpper(method) {
	case "GET", "HEAD", "OPTIONS", "PUT", "DELETE", "TRACE":
		return true
	default:
		return false
	}
}
