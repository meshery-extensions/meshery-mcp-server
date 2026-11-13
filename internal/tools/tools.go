// Package tools implements the MCP tools the server exposes: actions the AI can invoke.
package tools

import (
	"context"
	"fmt"
)

// Tool is an SDK-agnostic description of an MCP tool.
type Tool struct {
	// Name is the tool's identifier, used in tools/call requests.
	Name string
	// Description is shown to clients to help them choose tools.
	Description string
	// InputSchema is a JSON Schema (draft-07) describing the tool's input.
	InputSchema map[string]any
	// Handler executes the tool. args is the decoded input (a map[string]any
	// matching InputSchema). It returns the result or an error.
	Handler func(ctx context.Context, args any) (any, error)
}

// Register registers every tool with register. Called once at startup from
// cmd/meshery-mcp-server/main.go. The register callback adapts the portable
// Tool to the MCP server SDK (e.g. mcp.AddTool).
func Register(tools []Tool, register func(*Tool) error) error {
	for i := range tools {
		if err := register(&tools[i]); err != nil {
			return fmt.Errorf("registering tool %q: %w", tools[i].Name, err)
		}
	}
	return nil
}
