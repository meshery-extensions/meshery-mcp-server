package mcpserver

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RunStdio returns nil when the client closes stdin or ctx is cancelled.
func RunStdio(ctx context.Context, opts Options) error {
	return runTransport(ctx, opts, &mcp.StdioTransport{})
}

func runTransport(ctx context.Context, opts Options, t mcp.Transport) error {
	server, err := newServer(opts)
	if err != nil {
		return err
	}

	err = server.Run(ctx, t)
	if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
		return nil
	}
	return err
}
