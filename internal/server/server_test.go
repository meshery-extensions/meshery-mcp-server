// Copyright Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/version"
)

func TestServerInfoTool(t *testing.T) {
	s := New(nil) // nil manager for basic server tests without context tools

	mcpClient, err := client.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("new in-process client: %v", err)
	}
	defer func() {
		if err := mcpClient.Close(); err != nil {
			t.Errorf("close in-process client: %v", err)
		}
	}()

	ctx := context.Background()

	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test-client", Version: "0.0.1"},
		},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}

	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(toolsResp.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolsResp.Tools))
	}

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "server_info",
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok || text == nil {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	expected := fmt.Sprintf("%s %s (commit %s)", version.Name, version.Version, version.CommitSHA)
	if text.Text != expected {
		t.Fatalf("expected %q, got %q", expected, text.Text)
	}
}

// setupTestServerWithConfig creates an MCP server with a test config file.
func setupTestServerWithConfig(t *testing.T) (*client.Client, *config.Manager) {
	t.Helper()

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp-config.yaml")

	configContent := `current-context: dev
contexts:
  dev:
    server: http://localhost:9081
    token: dev-token-secret
  staging:
    server: https://meshery.staging.example.com
    token: staging-token-secret
  production:
    server: https://meshery.example.com
    token: production-token-secret
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	t.Setenv("MESHERY_CONFIG_PATH", configPath)
	t.Setenv("MESHERY_SERVER_URL", "")
	t.Setenv("MESHERY_API_TOKEN", "")

	mgr := config.NewManager()
	if err := mgr.Load(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	s := New(mgr)
	mcpClient, err := client.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("new in-process client: %v", err)
	}

	ctx := context.Background()
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test-client", Version: "0.0.1"},
		},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}

	return mcpClient, mgr
}

// ListContextsResponse mirrors the response from list_contexts tool.
type ListContextsResponse struct {
	Contexts       []ContextInfo `json:"contexts"`
	CurrentContext string        `json:"current_context"`
	FromFile       bool          `json:"from_file"`
}

// ContextInfo represents a context in the list response.
type ContextInfo struct {
	Name   string `json:"name"`
	Server string `json:"server"`
	Token  string `json:"token"`
	Active bool   `json:"active"`
}

// SwitchContextResponse mirrors the response from switch_context tool.
type SwitchContextResponse struct {
	Success         bool   `json:"success"`
	PreviousContext string `json:"previous_context"`
	CurrentContext  string `json:"current_context"`
	Server          string `json:"server"`
	Message         string `json:"message"`
}

func TestListContextsTool(t *testing.T) {
	mcpClient, _ := setupTestServerWithConfig(t)
	defer mcpClient.Close()

	ctx := context.Background()

	// Verify the tool is registered
	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	var hasListContexts bool
	for _, tool := range toolsResp.Tools {
		if tool.Name == "list_contexts" {
			hasListContexts = true
			break
		}
	}
	if !hasListContexts {
		t.Fatal("list_contexts tool not registered")
	}

	// Call the tool
	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "list_contexts",
		},
	})
	if err != nil {
		t.Fatalf("call list_contexts: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok || text == nil {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}

	// Parse the JSON response
	var response ListContextsResponse
	if err := json.Unmarshal([]byte(text.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.CurrentContext != "dev" {
		t.Errorf("CurrentContext = %q, want %q", response.CurrentContext, "dev")
	}

	if len(response.Contexts) != 3 {
		t.Errorf("len(Contexts) = %d, want %d", len(response.Contexts), 3)
	}

	// Verify tokens are masked
	for _, ctx := range response.Contexts {
		if ctx.Token != "***" && ctx.Token != "<not set>" && !containsAsterisks(ctx.Token) {
			t.Errorf("Token should be masked, got %q", ctx.Token)
		}
	}

	// Verify active context is marked
	var foundActive bool
	for _, ctx := range response.Contexts {
		if ctx.Name == "dev" && ctx.Active {
			foundActive = true
			break
		}
	}
	if !foundActive {
		t.Error("dev context should be marked as active")
	}
}

func TestSwitchContextTool(t *testing.T) {
	mcpClient, mgr := setupTestServerWithConfig(t)
	defer mcpClient.Close()

	ctx := context.Background()

	// Verify initial context
	_, name, _ := mgr.GetCurrentContext()
	if name != "dev" {
		t.Errorf("initial context = %q, want %q", name, "dev")
	}

	// Call switch_context to switch to staging
	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "switch_context",
			Arguments: map[string]any{
				"context": "staging",
			},
		},
	})
	if err != nil {
		t.Fatalf("call switch_context: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok || text == nil {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}

	// Parse the JSON response
	var response SwitchContextResponse
	if err := json.Unmarshal([]byte(text.Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Success = %v, want true", response.Success)
	}

	if response.PreviousContext != "dev" {
		t.Errorf("PreviousContext = %q, want %q", response.PreviousContext, "dev")
	}

	if response.CurrentContext != "staging" {
		t.Errorf("CurrentContext = %q, want %q", response.CurrentContext, "staging")
	}

	// Verify the manager's state changed
	_, newName, _ := mgr.GetCurrentContext()
	if newName != "staging" {
		t.Errorf("manager context = %q, want %q", newName, "staging")
	}
}

func TestSwitchContextToolInvalidContext(t *testing.T) {
	mcpClient, _ := setupTestServerWithConfig(t)
	defer mcpClient.Close()

	ctx := context.Background()

	// Try to switch to non-existent context
	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "switch_context",
			Arguments: map[string]any{
				"context": "nonexistent",
			},
		},
	})
	if err != nil {
		t.Fatalf("call switch_context: %v", err)
	}

	// Should return an error result
	if !result.IsError {
		t.Error("expected error result for non-existent context")
	}
}

func containsAsterisks(s string) bool {
	for i := 0; i <= len(s)-3; i++ {
		if s[i:i+3] == "***" {
			return true
		}
	}
	return false
}
