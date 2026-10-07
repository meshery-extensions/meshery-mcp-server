// Package tools provides Model Context Protocol (MCP) tool contracts and execution handlers.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
	"github.com/meshery-extensions/meshery-mcp-server/pkg/security"
)

// ListEnvironmentsTool implements the MCP tool contract for listing Meshery environments (Issue #7).
type ListEnvironmentsTool struct {
	client meshery.Client
}

// NewListEnvironmentsTool constructs a new ListEnvironmentsTool instance with the shared Meshery client.
func NewListEnvironmentsTool(client meshery.Client) *ListEnvironmentsTool {
	return &ListEnvironmentsTool{client: client}
}

// Name returns the MCP tool identifier.
func (t *ListEnvironmentsTool) Name() string {
	return "list_environments"
}

// Description returns a human-readable explanation of what the tool accomplishes.
func (t *ListEnvironmentsTool) Description() string {
	return "Lists all available Meshery environments with optional organization ID filtering and pagination."
}

// Schema returns the JSON schema definition for tool input arguments.
func (t *ListEnvironmentsTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"orgId": map[string]interface{}{
				"type":        "string",
				"description": "Optional organization ID to filter environments.",
			},
			"page": map[string]interface{}{
				"type":        "integer",
				"description": "Page number for paginated results (default: 0, 0-indexed).",
			},
			"pageSize": map[string]interface{}{
				"type":        "integer",
				"description": "Number of environments per page (default: 10, max: 100).",
			},
		},
	}
}

// Execute queries the Meshery API for environments and returns response-boundary sanitized environment objects.
func (t *ListEnvironmentsTool) Execute(ctx context.Context, params map[string]interface{}) (res map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("list_environments execution recovered from panic: %v", r)
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

	envs, totalCount, err := t.client.GetEnvironments(ctx, orgID, page, pageSize)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("list_environments failure: %s", sanitizedErr)
	}

	sanitizedEnvs := make([]interface{}, len(envs))
	for i, e := range envs {
		sanitizedEnvs[i] = security.SanitizeMap(e)
	}

	return map[string]interface{}{
		"page":         page,
		"pageSize":     pageSize,
		"total_count":  totalCount,
		"environments": sanitizedEnvs,
	}, nil
}

// GetEnvironmentTool implements the MCP tool contract for retrieving details of a single environment (Issue #7).
type GetEnvironmentTool struct {
	client meshery.Client
}

// NewGetEnvironmentTool constructs a new GetEnvironmentTool instance with the shared Meshery client.
func NewGetEnvironmentTool(client meshery.Client) *GetEnvironmentTool {
	return &GetEnvironmentTool{client: client}
}

// Name returns the MCP tool identifier.
func (t *GetEnvironmentTool) Name() string {
	return "get_environment"
}

// Description returns a human-readable explanation of what the tool accomplishes.
func (t *GetEnvironmentTool) Description() string {
	return "Retrieves detailed information and connections for a specific Meshery environment by ID."
}

// Schema returns the JSON schema definition for tool input arguments.
func (t *GetEnvironmentTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"environment_id": map[string]interface{}{
				"type":        "string",
				"description": "Unique identifier of the environment to retrieve.",
			},
		},
		"required": []string{"environment_id"},
	}
}

// Execute queries the Meshery API for environment details and returns a sanitized result map.
func (t *GetEnvironmentTool) Execute(ctx context.Context, params map[string]interface{}) (res map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("get_environment execution recovered from panic: %v", r)
		}
	}()

	if params == nil {
		return nil, fmt.Errorf("missing required parameters map")
	}

	rawID, ok := params["environment_id"].(string)
	if !ok || strings.TrimSpace(rawID) == "" {
		return nil, fmt.Errorf("parameter 'environment_id' is required and must be a non-empty string")
	}

	envID := sanitizeStringParam(rawID, 128)

	env, err := t.client.GetEnvironmentByID(ctx, envID)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("get_environment failure: %s", sanitizedErr)
	}

	return security.SanitizeMap(env), nil
}

// CreateEnvironmentTool implements the MCP tool contract for creating a new environment (Issue #7).
type CreateEnvironmentTool struct {
	client meshery.Client
}

// NewCreateEnvironmentTool constructs a new CreateEnvironmentTool instance with the shared Meshery client.
func NewCreateEnvironmentTool(client meshery.Client) *CreateEnvironmentTool {
	return &CreateEnvironmentTool{client: client}
}

// Name returns the MCP tool identifier.
func (t *CreateEnvironmentTool) Name() string {
	return "create_environment"
}

// Description returns a human-readable explanation of what the tool accomplishes.
func (t *CreateEnvironmentTool) Description() string {
	return "Creates a new Meshery environment with specified name, description, and organization."
}

// Schema returns the JSON schema definition for tool input arguments.
func (t *CreateEnvironmentTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the new environment (required).",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Optional description of the environment.",
			},
			"orgId": map[string]interface{}{
				"type":        "string",
				"description": "Optional organization ID to associate with the environment.",
			},
		},
		"required": []string{"name"},
	}
}

// Execute validates input arguments, sends creation request to Meshery Server, and returns sanitized result.
func (t *CreateEnvironmentTool) Execute(ctx context.Context, params map[string]interface{}) (res map[string]interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("create_environment execution recovered from panic: %v", r)
		}
	}()

	if params == nil {
		return nil, fmt.Errorf("missing required parameters map")
	}

	rawName, ok := params["name"].(string)
	if !ok || strings.TrimSpace(rawName) == "" {
		return nil, fmt.Errorf("parameter 'name' is required and must be a non-empty string")
	}

	name := sanitizeStringParam(rawName, 128)
	desc := ""
	if d, ok := params["description"].(string); ok {
		desc = sanitizeStringParam(d, 512)
	}
	orgID := ""
	if o, ok := params["orgId"].(string); ok {
		orgID = sanitizeStringParam(o, 128)
	}

	created, err := t.client.CreateEnvironment(ctx, name, desc, orgID)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("create_environment failure: %s", sanitizedErr)
	}

	return security.SanitizeMap(created), nil
}

// sanitizeStringParam enforces UTF-8 rune bounds and strips control characters for safety.
func sanitizeStringParam(s string, maxRunes int) string {
	if len(s) > maxRunes {
		runes := []rune(s)
		if len(runes) > maxRunes {
			s = string(runes[:maxRunes])
		}
	}
	s = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}
