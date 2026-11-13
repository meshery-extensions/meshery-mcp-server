// Package meshery provides the client used to call the Meshery server API.
package meshery

import "context"

// Client knows where Meshery is. It holds no identity: the caller's token
// travels with each request's context (see WithToken).
type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL}
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
