package meshery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meshery/meshkit/logger"
)

// HeaderKeychainScope is sent with every request when a keychain scope is
// configured, so Cloud evaluates the token against the deployment's
// requested keychains. The header name is pending Cloud's contract.
const HeaderKeychainScope = "Meshery-Keychain-Scope"

// HeaderAPIToken is the cookie Meshery Server reads for API token
// authentication. Cloud uses Authorization: Bearer instead.
const HeaderAPIToken = "token"

type callerIDKey struct{}

// WithCallerID returns a context carrying the authenticated user id that
// keys Cloud's per-user rate limits (client IP when there is no user).
func WithCallerID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, callerIDKey{}, id)
}

// callerIDFromContext returns the caller id from ctx, defaulting to "default".
func callerIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(callerIDKey{}).(string); ok && id != "" {
		return id
	}
	return "default"
}

type Option func(*Client)

// WithKeychainScope configures the narrowest keychain scope the deployment
// needs. Empty (the default) requests no keychain scope: previous behavior.
func WithKeychainScope(scope string) Option {
	return func(c *Client) { c.keychainScope = scope }
}

// WithMaxConcurrentCalls bounds concurrent Cloud calls per tool (default 4).
func WithMaxConcurrentCalls(limit int) Option {
	return func(c *Client) {
		if limit >= 1 {
			c.semaphore = NewSemaphore(limit)
		}
	}
}

// WithMaxRetries bounds 429/503 retries (default 3).
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithBackoff configures the 503/no-Retry-After backoff.
func WithBackoff(b Backoff) Option {
	return func(c *Client) { c.backoff = b }
}

// WithLogger attaches a meshkit logger for diagnostic debug output.
func WithLogger(log logger.Handler) Option {
	return func(c *Client) { c.log = log }
}

// WithAuthHeader configures the Authorization header name and value format.
// Use "Authorization" (default) for Bearer token auth (Cloud API), or
// "token" for Meshery Server API key header auth (no "Bearer " prefix).
func WithAuthHeader(header string) Option {
	return func(c *Client) {
		c.authHeader = header
		if header == HeaderAPIToken {
			c.authValue = func(token string) string { return token }
		} else {
			c.authValue = func(token string) string { return "Bearer " + token }
		}
	}
}

// WithAuthMode selects cookie auth (Meshery Server) or bearer auth (Cloud).
func WithAuthMode(mode AuthMode) Option {
	return func(c *Client) { c.authMode = mode }
}

// WithProvider sets the optional meshery-provider value sent with cookie auth.
func WithProvider(provider string) Option {
	return func(c *Client) { c.provider = provider }
}

// WithUserAgent overrides the User-Agent sent with every request.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// WithTimeout overrides the overall request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.http.Timeout = timeout }
}

// Client calls the Meshery Cloud API with token handling, rate-limit and
// revocation behavior per issues #6333, #6366, #6369 and #6381.
type Client struct {
	baseURL       string
	http          *http.Client
	tokens        TokenStore
	keychainScope string
	limiter       *Limiter
	semaphore     *Semaphore
	maxRetries    int
	backoff       Backoff
	log           logger.Handler
	authHeader    string                    // Header name for token auth; defaults to "Authorization" with Bearer prefix
	authValue     func(token string) string // Formats the header value; defaults to "Bearer <token>"
	authMode      AuthMode
	provider      string
	userAgent     string
}

// NewClient builds a Cloud client for baseURL using token. The token is
// held in a memory store; Revoke empties it (and any other store wired in).
func NewClient(baseURL string, token string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		http:       newHTTPClient(30 * time.Second),
		tokens:     &MemoryStore{},
		limiter:    NewLimiter(),
		semaphore:  NewSemaphore(4),
		maxRetries: 3,
		backoff:    Backoff{Base: DefaultBaseBackoff, Cap: DefaultCapBackoff},
		authHeader: "Authorization",
		authValue:  func(token string) string { return "Bearer " + token },
		authMode:   AuthBearer,
		userAgent:  defaultUserAgent,
	}
	c.tokens.Set(token)
	for _, o := range opts {
		o(c)
	}
	return c
}

// SetToken replaces the stored token (used by the OAuth flow in #66).
func (c *Client) SetToken(token string) {
	c.tokens.Set(token)
}

// Token returns the stored token, if any.
func (c *Client) Token() (string, bool) {
	return c.tokens.Get()
}

// applyAuth injects the credentials the target service actually accepts.
// Meshery Server reads the token cookie; Meshery Cloud reads Authorization:
// Bearer. An empty token sends no credentials.
func (c *Client) applyAuth(req *http.Request, token string) {
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if token == "" {
		return
	}
	if c.authMode == AuthCookie {
		//nolint:gosec // G124: Secure/HttpOnly/SameSite are response attributes only.
		req.AddCookie(&http.Cookie{Name: HeaderAPIToken, Value: token})
		if c.provider != "" {
			//nolint:gosec // G124: Secure/HttpOnly/SameSite are response attributes only.
			req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: c.provider})
		}
		return
	}
	req.Header.Set(c.authHeader, c.authValue(token))
}

// Call performs one Cloud request for tool. It applies the per-tool
// concurrency bound, the per-(family, caller) rate budget, classifies the
// response, and retries 429/503 a bounded number of times with context
// awareness and jitter. 401 and 403 are returned immediately, never retried.
func (c *Client) Call(ctx context.Context, method, path string, body []byte, tool string) ([]byte, error) {
	return c.call(ctx, method, path, body, tool, true)
}

// call is the single HTTP path for every request. Call requires a token;
// read-only wrappers and Ping may call without one because a Meshery Server
// in local provider mode has no authentication.
func (c *Client) call(ctx context.Context, method, path string, body []byte, tool string, requireToken bool) ([]byte, error) {
	if deadline, ok := ctx.Deadline(); !ok || (c.http.Timeout > 0 && time.Until(deadline) > c.http.Timeout) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.http.Timeout)
		defer cancel()
	}

	family := FamilyForRoute(method, path)

	if err := c.semaphore.Acquire(ctx, tool); err != nil {
		return nil, err
	}
	defer c.semaphore.Release(tool)

	if err := c.limiter.Take(ctx, family, callerIDFromContext(ctx)); err != nil {
		return nil, err
	}

	token, ok := c.tokens.Get()
	if requireToken && !ok {
		return nil, ErrNoToken()
	}

	url := c.baseURL + path
	var lastStatus int

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("building Cloud request: %w", err)
		}
		c.applyAuth(req, token)
		if c.keychainScope != "" {
			req.Header.Set(HeaderKeychainScope, c.keychainScope)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("cloud request failed: %w", err)
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("reading Cloud response: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("closing Cloud response: %w", closeErr)
		}
		lastStatus = resp.StatusCode

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return responseBody, nil
		}

		class := Classify(resp, responseBody)
		switch class.Class {
		case ClassReauth:
			return nil, ErrReauth(fmt.Errorf("HTTP %d", resp.StatusCode))
		case ClassInsufficientScope:
			message := strings.TrimSpace(string(responseBody))
			missingScope := class.MissingScope
			if token != "" {
				message = strings.ReplaceAll(message, token, "[REDACTED]")
				missingScope = strings.ReplaceAll(missingScope, token, "[REDACTED]")
			}
			return nil, &ErrInsufficientScope{
				MissingScope: missingScope,
				Underlying:   fmt.Errorf("HTTP %d %s", resp.StatusCode, message),
			}
		case ClassRateLimited:
			if attempt == c.maxRetries {
				retryAfter := class.RetryAfter
				if retryAfter == 0 {
					retryAfter = c.backoff.Cap
				}
				return nil, &ErrRateLimited{
					RetryAfter: retryAfter,
					Underlying: fmt.Errorf("HTTP 429 after %d attempts", c.maxRetries+1),
				}
			}
			if err := c.sleep(ctx, class.RetryAfter, attempt); err != nil {
				return nil, err
			}
			continue
		case ClassServiceUnavailable:
			if attempt == c.maxRetries {
				return nil, ErrServiceUnavailable(fmt.Errorf("HTTP 503 after %d attempts", c.maxRetries+1))
			}
			if err := c.sleep(ctx, class.RetryAfter, attempt); err != nil {
				return nil, err
			}
			continue
		default:
			return nil, c.apiError(method, path, resp.StatusCode, responseBody, token)
		}
	}
	return nil, fmt.Errorf("cloud request failed: HTTP %d", lastStatus)
}

// sleep waits for the retry delay, honoring Retry-After when present and
// falling back to jittered exponential backoff. It returns immediately on
// context cancellation so retries are never a tight loop.
func (c *Client) sleep(ctx context.Context, retryAfter time.Duration, attempt int) error {
	delay := retryAfter
	if delay == 0 {
		delay = c.backoff.Delay(attempt)
	}
	if c.log != nil {
		c.log.Debugf("Cloud throttled the request; retrying in %v (attempt %d)", delay, attempt+1)
	}
	timer := time.NewTimer(delay)
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	}
}

// Revoke retires the stored token with Cloud (POST /revoke). The token is
// dropped from every client-side store immediately, regardless of the
// server's response. Revocation is confirmed only on HTTP 200 (RFC 7009:
// unknown or already-revoked tokens also answer 200). 503 is retried with
// backoff; 401 is the only re-auth signal.
func (c *Client) Revoke(ctx context.Context) error {
	if deadline, ok := ctx.Deadline(); !ok || (c.http.Timeout > 0 && time.Until(deadline) > c.http.Timeout) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.http.Timeout)
		defer cancel()
	}

	token, _ := c.tokens.Get()
	c.tokens.Delete() // evict before the request: regardless of the response

	if token == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/revoke", nil)
	if err != nil {
		return fmt.Errorf("building revocation request: %w", err)
	}
	c.applyAuth(req, token)

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, err := c.http.Do(req)
		if err != nil {
			return fmt.Errorf("revocation request failed: %w", err)
		}
		responseBody, readErr := io.ReadAll(resp.Body)
		closeErr := resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("reading revocation response: %w", readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("closing revocation response: %w", closeErr)
		}

		switch resp.StatusCode {
		case http.StatusOK:
			// RFC 7009: 200 means the token is retired (or was unknown).
			if c.log != nil {
				c.log.Debugf("Cloud confirmed revocation")
			}
			return nil
		case http.StatusServiceUnavailable:
			if attempt == c.maxRetries {
				return ErrServiceUnavailable(fmt.Errorf("revocation returned 503 after %d attempts: %s",
					c.maxRetries+1, redact(token, string(responseBody))))
			}
			if err := c.sleep(ctx, classRetryAfter(resp), attempt); err != nil {
				return err
			}
			continue
		case http.StatusUnauthorized:
			return ErrReauth(fmt.Errorf("revocation returned 401"))
		default:
			return fmt.Errorf("revocation failed: HTTP %d %s", resp.StatusCode,
				redact(token, string(responseBody)))
		}
	}
	return nil
}

// classRetryAfter extracts Retry-After from a response for sleeping.
func classRetryAfter(resp *http.Response) time.Duration {
	delay, _ := parseRetryAfter(resp.Header.Get("Retry-After"))
	return delay
}

type tokenKey struct{}

// credentials is stored in the context by pointer so that printing the
// context never shows the token. A context prints its values with their
// String method, and fmt's %#v prints a pointer held inside another value as
// an address. A string, even of a type with a String method, would appear
// in full under %#v.
type credentials struct {
	token string
}

func (*credentials) String() string { return "[REDACTED]" }

// WithToken returns a copy of ctx carrying the Meshery token for this request.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey{}, &credentials{token: token})
}

// TokenFromContext returns the Meshery token carried by ctx, if any.
func TokenFromContext(ctx context.Context) (string, bool) {
	c, ok := ctx.Value(tokenKey{}).(*credentials)
	if !ok || c.token == "" {
		return "", false
	}
	return c.token, true
}
