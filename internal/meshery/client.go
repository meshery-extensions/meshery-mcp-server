// Package meshery provides the client used to call the Meshery server API.
package meshery

import (
	"context"
	"fmt"
)

// Client holds no identity: credentials travel with each request's context.
//
// TODO: send Provider and Token as the "meshery-provider" and "token"
// cookies, the only places Meshery reads them. Report a 302 to /provider or
// an HTML body as an authentication error naming the provider sent.
type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL}
}

// Credentials come from whoever issued the token; the provider is never
// inferred from it.
type Credentials struct {
	Provider string // for example Local or Meshery
	Token    string // empty with the Local provider
}

// String and GoString keep the token out of logs and test failures.
func (c Credentials) String() string {
	token := ""
	if c.Token != "" {
		token = "[REDACTED]"
	}
	return fmt.Sprintf("{Provider:%s Token:%s}", c.Provider, token)
}

func (c Credentials) GoString() string { return "meshery.Credentials" + c.String() }

type credentialsKey struct{}

// Stored by pointer: fmt's %#v prints a context's values from unexported
// fields without their String methods, but prints a pointer as an address.
type storedCredentials struct {
	c Credentials
}

func (*storedCredentials) String() string { return "[REDACTED]" }

func WithCredentials(ctx context.Context, c Credentials) context.Context {
	return context.WithValue(ctx, credentialsKey{}, &storedCredentials{c: c})
}

// CredentialsFromContext reports false for missing or empty credentials.
func CredentialsFromContext(ctx context.Context) (Credentials, bool) {
	s, ok := ctx.Value(credentialsKey{}).(*storedCredentials)
	if !ok || s.c == (Credentials{}) {
		return Credentials{}, false
	}
	return s.c, true
}
