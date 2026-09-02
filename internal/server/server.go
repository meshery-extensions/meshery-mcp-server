// Package server wires up the Meshery MCP server: it creates the underlying
// mcp-go server, registers every tool surface, and serves it over stdio.
package server

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
	"github.com/meshery-extensions/meshery-mcp-server/internal/version"
)

// New builds a Meshery MCP server with every tool surface registered.
func New() (*server.MCPServer, error) {
	// WithRecovery keeps a panic in one tool call (a bug we failed to
	// catch, or a future tool we haven't hardened yet) from taking down
	// the whole process, which would otherwise drop every other in-flight
	// or future tool call in this stdio session.
	s := server.NewMCPServer(version.Name, version.Version, server.WithRecovery())

	registry := NewRegistry(
		RegistrantFunc(func(s *server.MCPServer) error {
			tools.Register(s)
			return nil
		}),
	)

	if err := registry.RegisterAll(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Serve runs s over stdio until the client disconnects.
func Serve(s *server.MCPServer) error {
	return server.ServeStdio(s)
}
