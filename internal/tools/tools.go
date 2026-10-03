// Package tools implements the MCP tools exposed by the Meshery MCP server.
package tools

import "github.com/mark3labs/mcp-go/server"

// Register registers every tool exposed by the Meshery MCP server.
func Register(s *server.MCPServer) {
	RegisterPerformanceTools(s, nil)
}
