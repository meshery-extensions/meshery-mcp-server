// Package tools provides Model Context Protocol (MCP) tool contracts and execution handlers.
package tools

import (
	"context"
	"fmt"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
	"github.com/meshery-extensions/meshery-mcp-server/pkg/security"
)

// ListDesignsTool implements the MCP tool contract for listing Meshery design patterns (Issue #30).
type ListDesignsTool struct {
	client meshery.Client
}

// NewListDesignsTool constructs a new ListDesignsTool instance with the provided shared Meshery client.
func NewListDesignsTool(client meshery.Client) *ListDesignsTool {
	return &ListDesignsTool{client: client}
}

// Name returns the MCP tool identifier.
func (t *ListDesignsTool) Name() string {
	return "list_designs"
}

// Description returns a human-readable explanation of what the tool accomplishes.
func (t *ListDesignsTool) Description() string {
	return "Lists available Meshery cloud-native design patterns and infrastructure specifications with optional pagination and search filters."
}

// Schema returns the JSON schema definition for tool input arguments.
func (t *ListDesignsTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"page": map[string]interface{}{
				"type":        "integer",
				"description": "Page number for paginated results (default: 0, 0-indexed).",
			},
			"pageSize": map[string]interface{}{
				"type":        "integer",
				"description": "Number of design patterns per page (default: 10, max: 100).",
			},
			"search": map[string]interface{}{
				"type":        "string",
				"description": "Optional search term to filter design patterns by name or description.",
			},
		},
	}
}

// Execute queries Meshery API, validates pagination bounds, and returns response-boundary sanitized design objects.
func (t *ListDesignsTool) Execute(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	page := 0
	pageSize := 10
	search := ""

	if params != nil {
		if p, ok := params["page"].(float64); ok {
			page = int(p)
		} else if pInt, ok := params["page"].(int); ok {
			page = pInt
		}

		if ps, ok := params["pageSize"].(float64); ok {
			pageSize = int(ps)
		} else if psInt, ok := params["pageSize"].(int); ok {
			pageSize = psInt
		}

		if s, ok := params["search"].(string); ok {
			search = s
		}
	}

	// Clamp edge-case parameters safely (0-indexed Meshery Server pager)
	if page < 0 {
		page = 0
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	designs, totalCount, err := t.client.ListDesigns(ctx, page, pageSize, search)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("list_designs failure: %s", sanitizedErr)
	}

	sanitizedDesigns := make([]interface{}, len(designs))
	for i, d := range designs {
		sanitizedDesigns[i] = security.SanitizeMap(d)
	}

	return map[string]interface{}{
		"page":        page,
		"pageSize":    pageSize,
		"total_count": totalCount,
		"designs":     sanitizedDesigns,
	}, nil
}
