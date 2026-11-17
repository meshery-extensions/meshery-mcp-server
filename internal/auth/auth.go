// Package auth attaches each request's Meshery identity to its context.
package auth

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

// Resolver gives every stdio request the identity of the user who started
// the server. It must not be used for http, where each client has its own.
// The return type is unnamed so this package need not import the server.
func Resolver(cfg config.StdioConfig) func(context.Context, mcp.Request) (context.Context, error) {
	creds := meshery.Credentials{Provider: cfg.Provider, Token: string(cfg.Token)}
	if creds == (meshery.Credentials{}) {
		return nil
	}
	return func(ctx context.Context, _ mcp.Request) (context.Context, error) {
		return meshery.WithCredentials(ctx, creds), nil
	}
}
