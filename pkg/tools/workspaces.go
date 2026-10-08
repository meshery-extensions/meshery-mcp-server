// Package tools provides Model Context Protocol (MCP) tool contracts and execution handlers.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
	"github.com/meshery-extensions/meshery-mcp-server/pkg/security"
)

// ListWorkspacesTool implements the MCP tool contract for listing Meshery workspaces (Issue #7).
type ListWorkspacesTool struct {
	client meshery.Client
}

// NewListWorkspacesTool constructs a new ListWorkspacesTool instance with the shared Meshery client.
func NewListWorkspacesTool(client meshery.Client) *ListWorkspacesTool {
	return &ListWorkspacesTool{client: client}
}

// Name returns the MCP tool identifier.
func (t *ListWorkspacesTool) Name() string {
	return "list_workspaces"
}

// Description returns a human-readable explanation of what the tool accomplishes.
func (t *ListWorkspacesTool) Description() string {
	return "Lists all available Meshery workspaces with optional organization filtering and pagination."
}

// Schema returns the JSON schema definition for tool input arguments.
func (t *ListWorkspacesTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"orgId": map[string]interface{}{
				"type":        "string",
				"description": "Optional organization ID to filter workspaces.",
			},
			"page": map[string]interface{}{
				"type":        "integer",
				"description": "Page number for paginated results (default: 0, 0-indexed).",
			},
			"pageSize": map[string]interface{}{
				"type":        "integer",
				"description": "Number of workspaces per page (default: 10, max: 100).",
			},
		},
	}
}

// Execute queries the Meshery API for workspaces and returns response-boundary sanitized workspace objects.
func (t *ListWorkspacesTool) Execute(ctx context.Context, params map[string]interface{}) (res map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("list_workspaces execution recovered from panic: %v", r)
		}
	}()

	orgID := ""
	page := 0
	pageSize := 10

	if params != nil {
		if val, ok := params["orgId"].(string); ok {
			orgID = sanitizeStringParam(val, 128)
		}
		if val, ok := params["page"]; ok {
			page = parseNumericInt(val, 0)
		}
		if val, ok := params["pageSize"]; ok {
			pageSize = parseNumericInt(val, 10)
		}
	}

	if page < 0 {
		page = 0
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	workspaces, totalCount, err := t.client.ListWorkspaces(ctx, orgID, page, pageSize)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("list_workspaces failure: %s", sanitizedErr)
	}

	sanitizedWorkspaces := make([]interface{}, len(workspaces))
	for i, w := range workspaces {
		sanitizedWorkspaces[i] = security.SanitizeMap(w)
	}

	return map[string]interface{}{
		"page":       page,
		"pageSize":   pageSize,
		"total_count": totalCount,
		"workspaces": sanitizedWorkspaces,
	}, nil
}

// SwitchWorkspaceTool implements the MCP tool contract for switching the active workspace (Issue #7).
type SwitchWorkspaceTool struct {
	client meshery.Client
}

// NewSwitchWorkspaceTool constructs a new SwitchWorkspaceTool instance with the shared Meshery client.
func NewSwitchWorkspaceTool(client meshery.Client) *SwitchWorkspaceTool {
	return &SwitchWorkspaceTool{client: client}
}

// Name returns the MCP tool identifier.
func (t *SwitchWorkspaceTool) Name() string {
	return "switch_workspace"
}

// Description returns a human-readable explanation of what the tool accomplishes.
func (t *SwitchWorkspaceTool) Description() string {
	return "Sets the active workspace context in Meshery by workspace ID and returns confirmation details."
}

// Schema returns the JSON schema definition for tool input arguments.
func (t *SwitchWorkspaceTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"workspace_id": map[string]interface{}{
				"type":        "string",
				"description": "Unique identifier of the target workspace to activate.",
			},
		},
		"required": []string{"workspace_id"},
	}
}

// Execute validates workspace target ID, verifies existence via API query, and returns active workspace confirmation.
func (t *SwitchWorkspaceTool) Execute(ctx context.Context, params map[string]interface{}) (res map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("switch_workspace execution recovered from panic: %v", r)
		}
	}()

	if params == nil {
		return nil, fmt.Errorf("missing required parameters map")
	}

	rawID, ok := params["workspace_id"].(string)
	if !ok || strings.TrimSpace(rawID) == "" {
		return nil, fmt.Errorf("parameter 'workspace_id' is required and must be a non-empty string")
	}

	workspaceID := sanitizeStringParam(rawID, 128)

	ws, err := t.client.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("switch_workspace failure: %s", sanitizedErr)
	}

	t.client.SetActiveWorkspaceID(workspaceID)

	sanitizedWS := security.SanitizeMap(ws)

	return map[string]interface{}{
		"status":       "active",
		"workspace_id": workspaceID,
		"message":      fmt.Sprintf("Successfully set active workspace to %q", workspaceID),
		"workspace":    sanitizedWS,
	}, nil
}
