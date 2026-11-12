// Package tools implements the MCP tools the server exposes: actions the AI can invoke.
//
// Tools are described by the portable Tool type. Registration adapts them to
// github.com/modelcontextprotocol/go-sdk's mcp.AddTool, so the handler
// signature stays SDK-agnostic; only the registration wrapper touches go-sdk
// types. See docs/adr/0001-mcp-sdk-choice.md.
package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// Register registers every known tool with s. Called once at startup from
// cmd/meshery-mcp-server/main.go.
func Register(s *mcp.Server) error {
	for _, t := range allTools() {
		if err := register(s, t); err != nil {
			return err
		}
	}
	return nil
}

// register adapts a portable Tool to go-sdk's mcp.AddTool. go-sdk decodes the
// JSON-RPC arguments into a map[string]any matching the declared InputSchema.
func register(s *mcp.Server, t Tool) error {
	mcp.AddTool[map[string]any, any](s, &mcp.Tool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: t.InputSchema,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args map[string]any) (*mcp.CallToolResult, any, error) {
		out, err := t.Handler(ctx, args)
		if err != nil {
			return nil, nil, err
		}
		result := &mcp.CallToolResult{}
		if text, ok := out.(string); ok {
			result.Content = []mcp.Content{&mcp.TextContent{Text: text}}
		}
		return result, nil, nil
	})
	return nil
}

// allTools returns the canonical list of tools. Initially empty: tools are
// added as the MVP surface is built (see README "Planned scope").
func allTools() []Tool {
	return nil
}
