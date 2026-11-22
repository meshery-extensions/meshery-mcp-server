// Package auth decides whose Meshery identity each request uses, and attaches
// it to the request's context.
package auth

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

// Resolver returns the function the MCP server runs before every request.
// With a token, that function attaches it to the request's context, so every
// request uses the identity of the user who started the server. Without one,
// Resolver returns nil and requests carry no token, as with Meshery's Local
// provider.
//
// Transports that serve several users will need a resolver that takes the
// identity from each request instead; that resolver goes here.
//
// The returned type is unnamed so it can be assigned to the server's
// credentials option without this package importing the server.
func Resolver(cfg config.AuthConfig) func(context.Context, mcp.Request) (context.Context, error) {
	if cfg.Token == "" {
		return nil
	}
	token := string(cfg.Token)
	return func(ctx context.Context, _ mcp.Request) (context.Context, error) {
		return meshery.WithToken(ctx, token), nil
	}
}
