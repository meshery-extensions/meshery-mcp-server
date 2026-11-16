// Package tools implements the MCP tools the server exposes.
//
// Backend methods return Meshery's schema types; each tool's In and Out types
// are its contract with the AI and carry only what the AI needs. "Model"
// always means a Meshery model, never the AI model.
package tools

import (
	"errors"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Backend is everything the tools need from Meshery, one interface per
// domain. Get methods return an error wrapping ErrNotFound for an unknown ID.
type Backend interface {
	Registry
}

// Registry covers Meshery's capabilities registry (/api/registry).
type Registry interface {
	ComponentRegistry
	ModelRegistry
	RelationshipRegistry
}

// Appended to a tool's description until its Meshery call exists.
const notImplementedNote = "Not implemented yet: always returns an error."

type toolset struct {
	name  string
	tools []toolEntry
}

// add hides the handler's type parameters, so different tools fit in one slice.
type toolEntry struct {
	tool *mcp.Tool
	add  func(*mcp.Server)
}

func newToolEntry[In, Out any](t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) toolEntry {
	return toolEntry{
		tool: t,
		add:  func(s *mcp.Server) { mcp.AddTool(s, t, h) },
	}
}

// Register adds every tool to s and returns their names; the SDK offers no
// way to list them afterwards.
func Register(s *mcp.Server, b Backend) ([]string, error) {
	if s == nil {
		return nil, ErrServerRequired()
	}
	if b == nil {
		return nil, ErrBackendRequired()
	}

	var names []string
	for _, ts := range toolsets(b) {
		for _, entry := range ts.tools {
			entry.add(s)
			names = append(names, entry.tool.Name)
		}
	}
	return names, nil
}

func toolsets(b Backend) []toolset {
	return []toolset{
		componentToolset(b),
		modelToolset(b),
		relationshipToolset(b),
	}
}

func readOnly() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true}
}

var ErrNotFound = errors.New("not found")

// parseID validates an ID argument. Meshery compares IDs as strings against
// their canonical form, so a malformed or differently written ID would
// silently match nothing; uuid.UUID.String gives the canonical form.
func parseID(argument, value string) (uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil, ErrRequiredArgument(argument)
	}
	id, err := uuid.FromString(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrInvalidID(argument, value)
	}
	return id, nil
}
