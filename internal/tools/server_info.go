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

package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/version"
)

// Register registers all tools exposed by the Meshery MCP server.
// The config.Manager is optional; if nil, context management tools are not registered.
func Register(s *server.MCPServer, mgr *config.Manager) {
	// Server info tool
	serverInfo := mcp.NewTool("server_info",
		mcp.WithDescription("Return metadata about the Meshery MCP server."),
	)
	s.AddTool(serverInfo, serverInfoHandler)

	// Context management tools (only if manager is provided)
	if mgr != nil {
		registerContextTools(s, mgr)
	}
}

// registerContextTools registers the context management MCP tools.
func registerContextTools(s *server.MCPServer, mgr *config.Manager) {
	// list_contexts tool
	listContexts := mcp.NewTool("list_contexts",
		mcp.WithDescription("List all configured Meshery instances and show which is active."),
	)
	s.AddTool(listContexts, listContextsHandler(mgr))

	// switch_context tool
	switchContext := mcp.NewTool("switch_context",
		mcp.WithDescription("Switch the active Meshery instance context. Subsequent tool calls will use the new instance."),
		mcp.WithString("context",
			mcp.Required(),
			mcp.Description("Name of the context to switch to (e.g., 'dev', 'staging', 'production')"),
		),
	)
	s.AddTool(switchContext, switchContextHandler(mgr))
}

func serverInfoHandler(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(fmt.Sprintf("%s %s (commit %s)", version.Name, version.Version, version.CommitSHA)), nil
}
