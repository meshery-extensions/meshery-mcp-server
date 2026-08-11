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
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
)

// ContextInfo represents a context in the list response.
type ContextInfo struct {
	Name   string `json:"name"`
	Server string `json:"server"`
	Token  string `json:"token"`
	Active bool   `json:"active"`
}

// ListContextsResponse is the response format for list_contexts.
type ListContextsResponse struct {
	Contexts       []ContextInfo `json:"contexts"`
	CurrentContext string        `json:"current_context"`
	FromFile       bool          `json:"from_file"`
}

// listContextsHandler handles the list_contexts MCP tool call.
func listContextsHandler(mgr *config.Manager) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		contexts, currentContext, err := mgr.ListContexts()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list contexts: %v", err)), nil
		}

		// Build response with sorted context names for consistent output
		var contextInfos []ContextInfo
		var names []string
		for name := range contexts {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			ctx := contexts[name]
			contextInfos = append(contextInfos, ContextInfo{
				Name:   name,
				Server: ctx.Server,
				Token:  config.MaskToken(ctx.Token),
				Active: name == currentContext,
			})
		}

		response := ListContextsResponse{
			Contexts:       contextInfos,
			CurrentContext: currentContext,
			FromFile:       mgr.IsFromFile(),
		}

		jsonBytes, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}
