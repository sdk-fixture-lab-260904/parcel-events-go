// Idempotency-key injection (doc 38 §3.2 row 8, `idempotency:` in
// `doctorine.sdk.yml`). The key is resolved ONCE per logical request,
// BEFORE the retry loop, so every retry of one call carries the SAME key —
// that is the whole point of the header. Auto-generation (uuid4) applies
// only to POST; an explicit caller key is honored on any method. The uuid
// is runtime-unique by design (never hashed into builds).
//
// Vendored kernel file — imports nothing (stdlib only).

package kernel

import (
	"crypto/rand"
	"encoding/hex"
)

// DefaultIdempotencyHeader is the header name used when the config sets none.
const DefaultIdempotencyHeader = "Idempotency-Key"

// IdempotencyConfig mirrors the `idempotency:` config block. The zero value
// means: default header, no auto-generation.
type IdempotencyConfig struct {
	// Header is the header name (config `idempotency.header`);
	// "" means DefaultIdempotencyHeader.
	Header string
	// AutoGenerate opts POST requests into a generated key (like the
	// config block, auto-generation is opt-in).
	AutoGenerate bool
	// Generate overrides the key factory (default `doctorine-go-<uuid4>`).
	Generate func() string
}

func (c IdempotencyConfig) headerName() string {
	if c.Header == "" {
		return DefaultIdempotencyHeader
	}
	return c.Header
}

// DefaultIdempotencyKey returns `doctorine-go-<uuid4>`.
func DefaultIdempotencyKey() string {
	return "doctorine-go-" + newUUID4()
}

// ResolveIdempotencyKey resolves the key for one logical request: an
// explicit key always wins; otherwise auto-generation covers POST only (the
// one non-idempotent-by-spec verb). "" means no key.
func ResolveIdempotencyKey(method string, explicitKey string, config IdempotencyConfig) string {
	if explicitKey != "" {
		return explicitKey
	}
	if config.AutoGenerate && method == "POST" {
		if config.Generate != nil {
			return config.Generate()
		}
		return DefaultIdempotencyKey()
	}
	return ""
}

// newUUID4 renders an RFC 4122 version-4 UUID from crypto/rand.
func newUUID4() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		// crypto/rand failure means the platform's CSPRNG is broken —
		// unrecoverable, and never reachable on supported targets.
		panic("kernel: crypto/rand unavailable: " + err.Error())
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
}
