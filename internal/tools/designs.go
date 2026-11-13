package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

// NewDesignTools returns the design lifecycle MCP tools backed by the Meshery client.
// Mutating tools (create_design, deploy_design, undeploy_design, delete_design)
// require a confirm: true input to prevent accidental execution.
func NewDesignTools(client *meshery.Client) []Tool {
	return []Tool{
		{
			Name:        "list_designs",
			Description: "List all designs in the workspace.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page": map[string]any{
						"type":        "integer",
						"description": "Zero-indexed page number of the result set.",
						"minimum":     0,
						"default":     0,
					},
					"page_size": map[string]any{
						"type":        "integer",
						"description": "Number of items per page (1-100).",
						"minimum":     1,
						"maximum":     100,
						"default":     10,
					},
					"search": map[string]any{
						"type":        "string",
						"description": "Filter results by case-insensitive substring match on design name.",
						"maxLength":   255,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				search, _ := params["search"].(string)
				page := 0
				if v, ok := params["page"].(float64); ok {
					page = int(v)
				}
				pageSize := 10
				if v, ok := params["page_size"].(float64); ok {
					pageSize = int(v)
				}
				if page < 0 {
					page = 0
				}
				if pageSize < 1 {
					pageSize = 1
				}
				if pageSize > 100 {
					pageSize = 100
				}

				q := url.Values{}
				if search != "" {
					q.Set("search", search)
				}
				q.Set("page", strconv.Itoa(page))
				q.Set("page_size", strconv.Itoa(pageSize))
				q.Set("trim", "true")

				body, err := client.Call(ctx, http.MethodGet, "/api/content/patterns?"+q.Encode(), nil, "list_designs")
				if err != nil {
					return nil, err
				}

				var pageResp struct {
					Page     int             `json:"page"`
					PageSize int             `json:"pageSize"`
					Total    int             `json:"totalCount"`
					Patterns []DesignSummary `json:"patterns"`
				}
				if err := json.Unmarshal(body, &pageResp); err != nil {
					return nil, fmt.Errorf("decoding designs response: %w", err)
				}

				items := make([]DesignSummary, 0, len(pageResp.Patterns))
				for _, p := range pageResp.Patterns {
					items = append(items, DesignSummary{
						ID:          p.ID,
						Name:        p.Name,
						Version:     p.Version,
						Description: p.Description,
						Status:      p.Status,
						Visibility:  p.Visibility,
						CreatedAt:   p.CreatedAt,
						UpdatedAt:   p.UpdatedAt,
					})
				}

				return ListDesignsResult{
					Designs: items,
					Total:   pageResp.Total,
				}, nil
			},
		},
		{
			Name:        "get_design",
			Description: "Get full design details by ID from Meshery Cloud.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"design_id"},
				"properties": map[string]any{
					"design_id": map[string]any{
						"type":        "string",
						"description": "The design ID to look up.",
						"minLength":   1,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				designID, _ := params["design_id"].(string)
				if designID == "" {
					return nil, fmt.Errorf("design_id is required")
				}

				body, err := client.Call(ctx, http.MethodGet, "/api/content/patterns/"+url.PathEscape(designID), nil, "get_design")
				if err != nil {
					return nil, err
				}

				var patternResp struct {
					ID                 string `json:"id"`
					Name               string `json:"name"`
					Version            string `json:"version"`
					SchemaVersion      string `json:"schemaVersion,omitempty"`
					DisplayName        string `json:"displayName,omitempty"`
					Description        string `json:"description,omitempty"`
					Status             string `json:"status,omitempty"`
					Registrant         string `json:"registrant,omitempty"`
					Category           string `json:"category,omitempty"`
					Metadata           any    `json:"metadata,omitempty"`
					Model              string `json:"model,omitempty"`
					ModelID            string `json:"modelId,omitempty"`
					Components         []any  `json:"components,omitempty"`
					Relationships      []any  `json:"relationships,omitempty"`
					ComponentsCount    int    `json:"componentsCount,omitempty"`
					RelationshipsCount int    `json:"relationshipsCount,omitempty"`
					PatternFile        string `json:"patternFile,omitempty"`
					CreatedAt          string `json:"createdAt,omitempty"`
					UpdatedAt          string `json:"updatedAt,omitempty"`
				}
				if err := json.Unmarshal(body, &patternResp); err != nil {
					return nil, fmt.Errorf("decoding design response: %w", err)
				}

				return GetDesignResult{
					Design: DesignDetail{
						ID:                 patternResp.ID,
						Name:               patternResp.Name,
						Version:            patternResp.Version,
						SchemaVersion:      patternResp.SchemaVersion,
						DisplayName:        patternResp.DisplayName,
						Description:        patternResp.Description,
						Status:             patternResp.Status,
						Registrant:         patternResp.Registrant,
						Category:           patternResp.Category,
						Metadata:           patternResp.Metadata,
						Model:              patternResp.Model,
						ModelID:            patternResp.ModelID,
						Components:         patternResp.Components,
						Relationships:      patternResp.Relationships,
						ComponentsCount:    patternResp.ComponentsCount,
						RelationshipsCount: patternResp.RelationshipsCount,
						PatternFile:        patternResp.PatternFile,
						CreatedAt:          patternResp.CreatedAt,
						UpdatedAt:          patternResp.UpdatedAt,
					},
				}, nil
			},
		},
		{
			Name:        "create_design",
			Description: "Create a new design from YAML/JSON content.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"name", "design_yaml"},
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Human-readable name for the design.",
						"minLength":   1,
						"maxLength":   255,
					},
					"design_yaml": map[string]any{
						"type":        "string",
						"description": "The design body as YAML or JSON text.",
						"maxLength":   16777216,
					},
					"confirm": map[string]any{
						"type":        "boolean",
						"description": "Must be true to confirm creation of the design.",
						"default":     false,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				name, _ := params["name"].(string)
				designYAML, _ := params["design_yaml"].(string)
				confirm, _ := params["confirm"].(bool)

				if name == "" {
					return nil, fmt.Errorf("name is required")
				}
				if designYAML == "" {
					return nil, fmt.Errorf("design_yaml is required")
				}
				if !confirm {
					return CreateDesignResult{
						ID:   "",
						Name: name,
					}, fmt.Errorf("create_design would create design %q. Set confirm: true to proceed", name)
				}

				body, err := json.Marshal(map[string]any{
					"patternData": map[string]any{
						"patternFile": designYAML,
						"name":        name,
					},
					"name": name,
					"save": true,
				})
				if err != nil {
					return nil, fmt.Errorf("marshaling create_design request: %w", err)
				}

				respBody, err := client.Call(ctx, http.MethodPost, "/api/content/patterns", body, "create_design")
				if err != nil {
					return nil, err
				}

				var createResp struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				if err := json.Unmarshal(respBody, &createResp); err != nil {
					return nil, fmt.Errorf("decoding create_design response: %w", err)
				}

				return CreateDesignResult{
					ID:   createResp.ID,
					Name: createResp.Name,
				}, nil
			},
		},
		{
			Name:        "deploy_design",
			Description: "Deploy a design to a Kubernetes cluster.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"design_id", "cluster_id"},
				"properties": map[string]any{
					"design_id": map[string]any{
						"type":        "string",
						"description": "The design ID to deploy.",
						"minLength":   1,
					},
					"cluster_id": map[string]any{
						"type":        "string",
						"description": "The K8sContext ID of the target cluster.",
						"minLength":   1,
					},
					"confirm": map[string]any{
						"type":        "boolean",
						"description": "Must be true to confirm deployment.",
						"default":     false,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				designID, _ := params["design_id"].(string)
				clusterID, _ := params["cluster_id"].(string)
				confirm, _ := params["confirm"].(bool)

				if designID == "" {
					return nil, fmt.Errorf("design_id is required")
				}
				if clusterID == "" {
					return nil, fmt.Errorf("cluster_id is required")
				}
				if !confirm {
					return DeployDesignResult{
						DesignID:  designID,
						ClusterID: clusterID,
						Status:    "pending_confirmation",
					}, fmt.Errorf("deploy_design would deploy design %q to cluster %q. Set confirm: true to proceed", designID, clusterID)
				}

				// First, get the design to obtain its patternFile
				body, err := client.Call(ctx, http.MethodGet, "/api/content/patterns/"+url.PathEscape(designID), nil, "get_design")
				if err != nil {
					return nil, fmt.Errorf("fetching design %q for deployment: %w", designID, err)
				}

				var patternResp struct {
					ID          string `json:"id"`
					PatternFile string `json:"patternFile"`
				}
				if err := json.Unmarshal(body, &patternResp); err != nil {
					return nil, fmt.Errorf("decoding design response for deploy: %w", err)
				}
				if patternResp.PatternFile == "" {
					return nil, fmt.Errorf("design %q has no pattern content to deploy", designID)
				}

				deployBody, err := json.Marshal(map[string]any{
					"patternFile": patternResp.PatternFile,
					"patternID":   designID,
				})
				if err != nil {
					return nil, fmt.Errorf("marshaling deploy request: %w", err)
				}

				q := url.Values{}
				q.Add("contexts", clusterID)

				deployResp, err := client.Call(ctx, http.MethodPost, "/api/pattern/deploy?"+q.Encode(), deployBody, "deploy_design")
				if err != nil {
					return nil, err
				}

				var deployResult struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				if err := json.Unmarshal(deployResp, &deployResult); err != nil {
					// If we can't parse the response, still return success info
					return DeployDesignResult{
						DesignID:  designID,
						ClusterID: clusterID,
						Status:    "deployed",
					}, nil
				}

				return DeployDesignResult{
					DesignID:  designID,
					ClusterID: clusterID,
					Status:    "deployed",
				}, nil
			},
		},
		{
			Name:        "undeploy_design",
			Description: "Undeploy a design from a Kubernetes cluster.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"design_id", "cluster_id"},
				"properties": map[string]any{
					"design_id": map[string]any{
						"type":        "string",
						"description": "The design ID to undeploy.",
						"minLength":   1,
					},
					"cluster_id": map[string]any{
						"type":        "string",
						"description": "The K8sContext ID of the target cluster.",
						"minLength":   1,
					},
					"confirm": map[string]any{
						"type":        "boolean",
						"description": "Must be true to confirm undeployment.",
						"default":     false,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				designID, _ := params["design_id"].(string)
				clusterID, _ := params["cluster_id"].(string)
				confirm, _ := params["confirm"].(bool)

				if designID == "" {
					return nil, fmt.Errorf("design_id is required")
				}
				if clusterID == "" {
					return nil, fmt.Errorf("cluster_id is required")
				}
				if !confirm {
					return UndeployDesignResult{
						DesignID:  designID,
						ClusterID: clusterID,
						Status:    "pending_confirmation",
					}, fmt.Errorf("undeploy_design would undeploy design %q from cluster %q. Set confirm: true to proceed", designID, clusterID)
				}

				// First, get the design to obtain its patternFile
				body, err := client.Call(ctx, http.MethodGet, "/api/content/patterns/"+url.PathEscape(designID), nil, "get_design")
				if err != nil {
					return nil, fmt.Errorf("fetching design %q for undeploy: %w", designID, err)
				}

				var patternResp struct {
					ID          string `json:"id"`
					PatternFile string `json:"patternFile"`
				}
				if err := json.Unmarshal(body, &patternResp); err != nil {
					return nil, fmt.Errorf("decoding design response for undeploy: %w", err)
				}
				if patternResp.PatternFile == "" {
					return nil, fmt.Errorf("design %q has no pattern content to undeploy", designID)
				}

				undeployBody, err := json.Marshal(map[string]any{
					"patternFile": patternResp.PatternFile,
					"patternID":   designID,
				})
				if err != nil {
					return nil, fmt.Errorf("marshaling undeploy request: %w", err)
				}

				q := url.Values{}
				q.Add("contexts", clusterID)

				_, err = client.Call(ctx, http.MethodDelete, "/api/pattern/deploy?"+q.Encode(), undeployBody, "undeploy_design")
				if err != nil {
					return nil, err
				}

				return UndeployDesignResult{
					DesignID:  designID,
					ClusterID: clusterID,
					Status:    "undeployed",
				}, nil
			},
		},
		{
			Name:        "delete_design",
			Description: "Permanently delete a design.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"design_id"},
				"properties": map[string]any{
					"design_id": map[string]any{
						"type":        "string",
						"description": "The design ID to delete.",
						"minLength":   1,
					},
					"confirm": map[string]any{
						"type":        "boolean",
						"description": "Must be true to confirm deletion.",
						"default":     false,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				designID, _ := params["design_id"].(string)
				confirm, _ := params["confirm"].(bool)

				if designID == "" {
					return nil, fmt.Errorf("design_id is required")
				}
				if !confirm {
					return DeleteDesignResult{
						ID:   designID,
						Name: designID,
					}, fmt.Errorf("delete_design would delete design %q. Set confirm: true to proceed", designID)
				}

				_, err := client.Call(ctx, http.MethodDelete, "/api/content/patterns/"+url.PathEscape(designID), nil, "delete_design")
				if err != nil {
					return nil, err
				}

				return DeleteDesignResult{
					ID:   designID,
					Name: designID,
				}, nil
			},
		},
	}
}

// decodeArgs decodes a map[string]any into the target pointer. The MCP server
// decodes JSON-RPC arguments into map[string]any.
func decodeArgs(args any, target map[string]any) error {
	m, ok := args.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid arguments: expected object, got %T", args)
	}
	for k, v := range m {
		target[k] = v
	}
	return nil
}

// ---- Result types ----

type ListDesignsResult struct {
	Designs []DesignSummary `json:"designs"`
	Total   int             `json:"total"`
}

type DesignSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type GetDesignResult struct {
	Design DesignDetail `json:"design"`
}

type DesignDetail struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Version            string `json:"version"`
	SchemaVersion      string `json:"schemaVersion,omitempty"`
	DisplayName        string `json:"displayName,omitempty"`
	Description        string `json:"description,omitempty"`
	Status             string `json:"status,omitempty"`
	Registrant         string `json:"registrant,omitempty"`
	Category           string `json:"category,omitempty"`
	Metadata           any    `json:"metadata,omitempty"`
	Model              string `json:"model,omitempty"`
	ModelID            string `json:"modelId,omitempty"`
	Components         []any  `json:"components,omitempty"`
	Relationships      []any  `json:"relationships,omitempty"`
	ComponentsCount    int    `json:"componentsCount,omitempty"`
	RelationshipsCount int    `json:"relationshipsCount,omitempty"`
	PatternFile        string `json:"patternFile,omitempty"`
	CreatedAt          string `json:"createdAt,omitempty"`
	UpdatedAt          string `json:"updatedAt,omitempty"`
}

type CreateDesignResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeployDesignResult struct {
	DesignID  string `json:"design_id"`
	ClusterID string `json:"cluster_id"`
	Status    string `json:"status"`
}

type UndeployDesignResult struct {
	DesignID  string `json:"design_id"`
	ClusterID string `json:"cluster_id"`
	Status    string `json:"status"`
}

type DeleteDesignResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
