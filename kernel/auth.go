// Auth providers (doc 38 §3.2 row 5, `auth:` in `doctorine.sdk.yml`):
// apiKey (header or query) / bearer / basic / oauth2 client-credentials
// with a token cache, early refresh, and a configurable token endpoint. The
// client calls Authorize per ATTEMPT — fresh material after refresh — and
// Invalidate once on a 401: a cached-but-revoked token earns exactly one
// transparent re-auth retry.
//
// Vendored kernel file — imports only sibling kernel files (stdlib only).

package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AuthMaterial is one attempt's credentials: headers merged onto the
// request (keys lower-cased), query params appended to the URL.
type AuthMaterial struct {
	Header map[string]string
	Query  []QueryParam
}

// AuthProvider yields per-attempt auth material. Providers that perform
// I/O (oauth2) honor ctx; static providers ignore it.
type AuthProvider interface {
	// Authorize returns material for one attempt — may fetch/refresh tokens.
	Authorize(ctx context.Context) (AuthMaterial, error)
	// Invalidate is the 401 hook: drop cached material. It reports true
	// when a single re-auth retry is worthwhile (something WAS cached and
	// may simply have expired).
	Invalidate() bool
}

type staticAuth struct {
	material AuthMaterial
}

func (a staticAuth) Authorize(context.Context) (AuthMaterial, error) { return a.material, nil }
func (a staticAuth) Invalidate() bool                                { return false }

// APIKeyPlacement says where an API key travels: a header or a query param.
type APIKeyPlacement string

// The two placements of the config's `auth.api_key.placement`.
const (
	APIKeyInHeader APIKeyPlacement = "header"
	APIKeyInQuery  APIKeyPlacement = "query"
)

// APIKeyAuth authenticates with a static API key in a header or query param.
func APIKeyAuth(placement APIKeyPlacement, name string, key string) AuthProvider {
	if placement == APIKeyInQuery {
		return staticAuth{material: AuthMaterial{Query: []QueryParam{{Name: name, Value: key}}}}
	}
	return staticAuth{material: AuthMaterial{Header: map[string]string{strings.ToLower(name): key}}}
}

// BearerAuth authenticates with a static bearer token.
func BearerAuth(token string) AuthProvider {
	return staticAuth{material: AuthMaterial{Header: map[string]string{"authorization": "Bearer " + token}}}
}

// BasicAuth authenticates with HTTP basic credentials.
func BasicAuth(username string, password string) AuthProvider {
	encoded := EncodeBase64([]byte(username + ":" + password))
	return staticAuth{material: AuthMaterial{Header: map[string]string{"authorization": "Basic " + encoded}}}
}

// ── oauth2 client-credentials ───────────────────────────────────────────────

// OAuth2Config configures the client-credentials provider.
type OAuth2Config struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
	// HTTPClient performs the token request (share the client's);
	// nil means a fresh default client.
	HTTPClient *http.Client
	// Now is a clock injection point for tests (default time.Now).
	Now func() time.Time
	// EarlyRefresh refreshes this long BEFORE nominal expiry (default 60s).
	EarlyRefresh time.Duration
}

type cachedToken struct {
	accessToken string
	expiresAt   time.Time
}

// OAuth2ClientCredentials is a client-credentials token cache with
// SINGLE-FLIGHT refresh: concurrent Authorize calls that miss the cache
// serialize on the mutex and re-check it inside, so one fetch serves the
// whole burst (a cold-cache fan-out must never stampede the token endpoint
// — the mutex is deliberately held across the fetch).
type OAuth2ClientCredentials struct {
	config OAuth2Config
	mu     sync.Mutex
	token  *cachedToken
}

// NewOAuth2ClientCredentials builds the provider (defaults: fresh
// http.Client, time.Now, 60s early refresh).
func NewOAuth2ClientCredentials(config OAuth2Config) *OAuth2ClientCredentials {
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{}
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.EarlyRefresh == 0 {
		config.EarlyRefresh = time.Minute
	}
	return &OAuth2ClientCredentials{config: config}
}

// Authorize returns a bearer header from the cache, fetching (single-
// flight) when the cached token is absent or inside the early-refresh
// window.
func (o *OAuth2ClientCredentials) Authorize(ctx context.Context) (AuthMaterial, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	token := o.cachedLocked()
	if token == nil {
		fetched, err := o.fetch(ctx)
		if err != nil {
			return AuthMaterial{}, err
		}
		o.token = fetched
		token = fetched
	}
	return AuthMaterial{Header: map[string]string{"authorization": "Bearer " + token.accessToken}}, nil
}

// Invalidate drops the cached token (the 401 hook) and reports whether
// anything was cached.
func (o *OAuth2ClientCredentials) Invalidate() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	hadMaterial := o.token != nil
	o.token = nil
	return hadMaterial
}

func (o *OAuth2ClientCredentials) cachedLocked() *cachedToken {
	if o.token != nil && o.token.expiresAt.Add(-o.config.EarlyRefresh).After(o.config.Now()) {
		return o.token
	}
	return nil
}

func (o *OAuth2ClientCredentials) fetch(ctx context.Context) (*cachedToken, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {o.config.ClientID},
		"client_secret": {o.config.ClientSecret},
	}
	if len(o.config.Scopes) > 0 {
		form.Set("scope", strings.Join(o.config.Scopes, " "))
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, o.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, NewConnectionError("invalid OAuth token endpoint URL: "+err.Error(), err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	response, err := o.config.HTTPClient.Do(request)
	if err != nil {
		return nil, NewConnectionError("connection error while contacting the OAuth token endpoint", err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, NewConnectionError("connection error while reading the OAuth token response", err)
	}
	return o.storeFromResponse(response, raw)
}

func (o *OAuth2ClientCredentials) storeFromResponse(response *http.Response, raw []byte) (*cachedToken, error) {
	var body any
	if len(raw) == 0 || json.Unmarshal(raw, &body) != nil {
		body = string(raw)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := fmt.Sprintf("OAuth token endpoint responded %d", response.StatusCode)
		return nil, NewAPIError(response.StatusCode, headerToMap(response.Header), body, message)
	}
	record, _ := body.(map[string]any)
	accessToken, _ := record["access_token"].(string)
	if accessToken == "" {
		return nil, NewConnectionError("OAuth token endpoint returned a malformed body (no access_token)", nil)
	}
	expiresInSeconds := 3600.0
	if expiresIn, isNumber := record["expires_in"].(float64); isNumber && expiresIn > 0 {
		expiresInSeconds = expiresIn
	}
	expiresAt := o.config.Now().Add(time.Duration(expiresInSeconds * float64(time.Second)))
	return &cachedToken{accessToken: accessToken, expiresAt: expiresAt}, nil
}
