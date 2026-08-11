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

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
)

// SwitchContextResponse is the response format for switch_context.
type SwitchContextResponse struct {
	Success         bool   `json:"success"`
	PreviousContext string `json:"previous_context"`
	CurrentContext  string `json:"current_context"`
	Server          string `json:"server"`
	Message         string `json:"message"`
}

// switchContextHandler handles the switch_context MCP tool call.
func switchContextHandler(mgr *config.Manager) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract the context name from arguments
		contextName, err := req.RequireString("context")
		if err != nil || contextName == "" {
			return mcp.NewToolResultError("context parameter is required"), nil
		}

		// Get current context before switching
		_, previousContext, _ := mgr.GetCurrentContext()

		// Attempt to switch context
		if err := mgr.SwitchContext(contextName); err != nil {
			// List available contexts for better error message
			contexts, _, _ := mgr.ListContexts()
			var available []string
			for name := range contexts {
				available = append(available, name)
			}
			return mcp.NewToolResultError(fmt.Sprintf(
				"failed to switch context: %v. Available contexts: %v",
				err, available,
			)), nil
		}

		// Get the new context details
		newCtx, currentContext, err := mgr.GetCurrentContext()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get new context: %v", err)), nil
		}

		response := SwitchContextResponse{
			Success:         true,
			PreviousContext: previousContext,
			CurrentContext:  currentContext,
			Server:          newCtx.Server,
			Message:         fmt.Sprintf("Successfully switched from '%s' to '%s'", previousContext, currentContext),
		}

		jsonBytes, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}
