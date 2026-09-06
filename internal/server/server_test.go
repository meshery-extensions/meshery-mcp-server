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
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/version"
)

// TestServerInfoTool verifies the server_info tool through the full
// initialize, list tools, and call tool flow over an in-process client.
func TestServerInfoTool(t *testing.T) {
	s := New()

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
