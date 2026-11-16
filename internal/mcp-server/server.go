// Package mcpserver builds the MCP server and serves it over a transport.
package mcpserver

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
)

// CredentialsFunc runs before every request; an error rejects it.
type CredentialsFunc func(ctx context.Context, req mcp.Request) (context.Context, error)

type Options struct {
	Name        string
	Version     string
	Logger      *slog.Logger
	Backend     tools.Backend
	Credentials CredentialsFunc
}

func newServer(opts Options) (*mcp.Server, error) {
	server := mcp.NewServer(
		&mcp.Implementation{Name: opts.Name, Version: opts.Version},
		&mcp.ServerOptions{
			Logger: opts.Logger,
			// With nil Capabilities the SDK also advertises logging, which
			// this server does not use and protocol 2026-07-28 deprecates.
			Capabilities: &mcp.ServerCapabilities{},
		},
	)
	if opts.Credentials != nil {
		server.AddReceivingMiddleware(credentialsMiddleware(opts.Credentials))
	}
	toolNames, err := tools.Register(server, opts.Backend)
	if err != nil {
		return nil, err
	}
	if opts.Logger != nil {
		// Nothing registers resources or prompts yet.
		opts.Logger.Info("registered capabilities",
			"tools", toolNames,
			"resources", []string{},
			"prompts", []string{},
		)
	}
	return server, nil
}

func credentialsMiddleware(resolve CredentialsFunc) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			ctx, err := resolve(ctx, req)
			if err != nil {
				return nil, err
			}
			return next(ctx, method, req)
		}
	}
}
