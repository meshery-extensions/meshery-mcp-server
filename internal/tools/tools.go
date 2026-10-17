// Package tools implements the MCP tools the server exposes: actions the AI can invoke.
//
// Two kinds of types meet here. Meshery's types (from github.com/meshery/schemas)
// are what the registry interfaces return. Each tool's In and Out types are
// what the AI sends and sees; they are part of the tool's contract and carry
// only the fields the AI needs. Handlers map the first to the second.
//
// "Model" in this package always means a Meshery model, never the AI model.
package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Registry is everything the tools need from Meshery. Each domain declares
// its own narrow interface; an implementation satisfies all of them.
//
// Implementations should return MeshKit errors that say what failed: tools
// pass those to the AI unchanged, and wrap any other error.
type Registry interface {
	ComponentRegistry
	ModelRegistry
	RelationshipRegistry
}

// notImplementedNote is appended to the description of every tool whose
// Meshery calls do not exist yet, so the AI knows not to rely on it.
// Remove it from a tool when that tool is implemented.
const notImplementedNote = "Not implemented yet: always returns an error."

type toolset struct {
	name  string
	tools []toolEntry
}

// toolEntry pairs a tool's metadata with the call that adds it to a server.
// The add func hides the handler's In and Out type parameters, so tools with
// different types fit in one slice.
type toolEntry struct {
	tool *mcp.Tool
	add  func(*mcp.Server)
}

// newToolEntry returns the toolEntry for tool t handled by h.
func newToolEntry[In, Out any](t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) toolEntry {
	return toolEntry{
		tool: t,
		add:  func(s *mcp.Server) { mcp.AddTool(s, t, h) },
	}
}

// Register adds every tool to s and returns their names, in registration
// order. The SDK has no way to list a server's tools, so this is how the
// server learns what it registered.
func Register(s *mcp.Server, r Registry) ([]string, error) {
	if s == nil {
		return nil, ErrServerRequired()
	}
	if r == nil {
		return nil, ErrRegistryRequired()
	}

	var names []string
	for _, ts := range toolsets(r) {
		for _, entry := range ts.tools {
			entry.add(s)
			names = append(names, entry.tool.Name)
		}
	}
	return names, nil
}

func toolsets(r Registry) []toolset {
	return []toolset{
		componentToolset(r),
		modelToolset(r),
		relationshipToolset(r),
	}
}

func readOnly() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true}
}
