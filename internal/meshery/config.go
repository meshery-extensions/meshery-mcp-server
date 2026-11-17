package meshery

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthMode selects how the client authenticates.
type AuthMode string

const (
	// AuthCookie authenticates to a Meshery Server with the cookie the server
	// actually reads: token (and optionally meshery-provider).
	AuthCookie AuthMode = "cookie"

	// AuthBearer authenticates to Meshery Cloud with Authorization: Bearer.
	AuthBearer AuthMode = "bearer"
)

const (
	defaultTimeout            = 30 * time.Second
	defaultMaxConcurrentCalls = 4
	defaultUserAgent          = "meshery-mcp-server"
)

// Config is the validated client configuration.
type Config struct {
	BaseURL string
	Token   string
	// Provider is the optional meshery-provider value (cookie auth only).
	Provider string
	AuthMode AuthMode
	// Timeout bounds each HTTP request, including reading the body.
	Timeout time.Duration
	// MaxRetries bounds 429/503 retries. Zero means no retries.
	MaxRetries int
	// MaxConcurrentCalls bounds concurrent requests per tool.
	MaxConcurrentCalls int
	KeychainScope      string
	UserAgent          string
}

// New builds a validated client. It keeps one HTTP client with explicit
// transport timeouts; nothing may hang indefinitely.
func New(cfg Config) (*Client, error) {
	normalized, err := cfg.validate()
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL:       strings.TrimRight(normalized.BaseURL, "/"),
		http:          newHTTPClient(normalized.Timeout),
		tokens:        &MemoryStore{},
		limiter:       NewLimiter(),
		semaphore:     NewSemaphore(normalized.MaxConcurrentCalls),
		maxRetries:    normalized.MaxRetries,
		backoff:       Backoff{Base: DefaultBaseBackoff, Cap: DefaultCapBackoff},
		keychainScope: normalized.KeychainScope,
		userAgent:     normalized.UserAgent,
		authMode:      normalized.AuthMode,
		provider:      normalized.Provider,
		authHeader:    "Authorization",
		authValue:     func(token string) string { return "Bearer " + token },
	}
	c.tokens.Set(normalized.Token)
	return c, nil
}

func (c Config) validate() (Config, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Config{}, ErrInvalidBaseURL()
	}
	if u.User != nil {
		return Config{}, ErrInvalidBaseURL()
	}

	switch c.AuthMode {
	case "":
		c.AuthMode = AuthCookie
	case AuthCookie, AuthBearer:
	default:
		return Config{}, ErrInvalidAuthMode(string(c.AuthMode))
	}

	if c.Timeout < 0 {
		return Config{}, ErrInvalidTimeout(c.Timeout)
	}
	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
	if c.MaxRetries < 0 {
		return Config{}, ErrInvalidMaxRetries(c.MaxRetries)
	}
	if c.MaxConcurrentCalls < 0 {
		return Config{}, ErrInvalidMaxConcurrentCalls(c.MaxConcurrentCalls)
	}
	if c.MaxConcurrentCalls == 0 {
		c.MaxConcurrentCalls = defaultMaxConcurrentCalls
	}
	if c.UserAgent == "" {
		c.UserAgent = defaultUserAgent
	}
	return c, nil
}

func newHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Never carry credentials to another host.
			if len(via) > 0 && req.URL.Host != via[0].URL.Host {
				req.Header.Del("Cookie")
				req.Header.Del("Authorization")
				req.Header.Del(HeaderAPIToken)
			}
			return nil
		},
	}
}
