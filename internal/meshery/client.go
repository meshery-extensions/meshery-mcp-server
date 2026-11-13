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

const HeaderToken = "meshery"

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
	return func(c *Client) { c.semaphore = NewSemaphore(limit) }
}

// WithMaxRetries bounds 429/503 retries (default 3).
func WithMaxRetries(n int) Option {
	return func(c *Client) { c.maxRetries = n }
}

// WithBackoff configures the 503/no-Retry-After backoff.
func WithBackoff(b Backoff) Option {
	return func(c *Client) { c.backoff = b }
}

// WithLogger attaches a meshkit logger for diagnostic debug output.
func WithLogger(log logger.Handler) Option {
	return func(c *Client) { c.log = log }
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
}

// NewClient builds a Cloud client for baseURL using token. The token is
// held in a memory store; Revoke empties it (and any other store wired in).
func NewClient(baseURL string, token string, opts ...Option) *Client {
	c := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		http:       &http.Client{Timeout: 30 * time.Second},
		tokens:     &MemoryStore{},
		limiter:    NewLimiter(),
		semaphore:  NewSemaphore(4),
		maxRetries: 3,
		backoff:    Backoff{Base: DefaultBaseBackoff, Cap: DefaultCapBackoff},
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

// Call performs one Cloud request for tool. It applies the per-tool
// concurrency bound, the per-(family, caller) rate budget, classifies the
// response, and retries 429/503 a bounded number of times with context
// awareness and jitter. 401 and 403 are returned immediately, never retried.
func (c *Client) Call(ctx context.Context, method, path string, body []byte, tool string) ([]byte, error) {
	family := FamilyForRoute(method, path)

	if err := c.semaphore.Acquire(ctx, tool); err != nil {
		return nil, err
	}
	defer c.semaphore.Release(tool)

	if err := c.limiter.Take(ctx, family, callerIDFromContext(ctx)); err != nil {
		return nil, err
	}

	token, ok := c.tokens.Get()
	if !ok {
		return nil, ErrNoToken()
	}

	url := c.baseURL + path
	var lastStatus int

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("building Cloud request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
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
			return nil, &ErrInsufficientScope{
				MissingScope: class.MissingScope,
				Underlying:   fmt.Errorf("HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(responseBody))),
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
			return nil, fmt.Errorf("cloud request failed: HTTP %d %s", resp.StatusCode,
				strings.TrimSpace(string(responseBody)))
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
	token, _ := c.tokens.Get()
	c.tokens.Delete() // evict before the request: regardless of the response

	if token == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/revoke", nil)
	if err != nil {
		return fmt.Errorf("building revocation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

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
					c.maxRetries+1, strings.TrimSpace(string(responseBody))))
			}
			if err := c.sleep(ctx, classRetryAfter(resp), attempt); err != nil {
				return err
			}
			continue
		case http.StatusUnauthorized:
			return ErrReauth(fmt.Errorf("revocation returned 401"))
		default:
			return fmt.Errorf("revocation failed: HTTP %d %s", resp.StatusCode,
				strings.TrimSpace(string(responseBody)))
		}
	}
	return nil
}

// classRetryAfter extracts Retry-After from a response for sleeping.
func classRetryAfter(resp *http.Response) time.Duration {
	delay, _ := parseRetryAfter(resp.Header.Get("Retry-After"))
	return delay
}
