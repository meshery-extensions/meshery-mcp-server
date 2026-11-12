package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestToolsReferenceDrift asserts that docs/tools-reference.md documents the
// current set of registered MCP tools. With no tools registered on master it
// passes trivially; when tools are added, every registered tool name must
// appear in the doc (or be added to the undocumented list below) before the
// change can be merged.
func TestToolsReferenceDrift(t *testing.T) {
	// Exception list: tool names that are intentionally not documented yet.
	// Add names here only with a corresponding comment explaining why.
	undocumented := map[string]bool{}

	registered := registeredTools()
	for _, name := range registered {
		if undocumented[name] {
			continue
		}
		if !toolDocumented(t, name) {
			t.Errorf("tool %q is registered but not documented in docs/tools-reference.md", name)
		}
	}
}

// registeredTools returns the names of all registered MCP tools. On master no
// tools are registered (internal/tools is an empty stub), so this returns nil.
// When tools are registered, enumerate them here so the drift test enforces
// that each one is documented.
func registeredTools() []string {
	return nil
}

// toolDocumented reports whether the given tool name appears in the tools
// reference doc. It looks for the name in a Markdown table cell to avoid
// matching incidental mentions.
func toolDocumented(t *testing.T, name string) bool {
	path, err := docPath()
	if err != nil {
		t.Logf("could not locate docs/tools-reference.md: %v", err)
		return false
	}
	//nolint:gosec // G304: path is derived from the repo root (go.mod), not user input
	data, err := os.ReadFile(path)
	if err != nil {
		t.Logf("could not read docs/tools-reference.md: %v", err)
		return false
	}
	content := string(data)

	if !strings.Contains(content, "No tools are registered yet") {
		t.Log("tools-reference.md does not state that no tools are registered")
		return false
	}

	lookahead := "| " + name + " |"
	if strings.Contains(content, lookahead) {
		return true
	}
	if strings.Contains(content, "| **"+name+"** |") {
		return true
	}
	return false
}

// docPath returns the path to docs/tools-reference.md relative to the repo
// root. It walks up from the test's working directory until it finds go.mod.
func docPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "docs", "tools-reference.md"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
