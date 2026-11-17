// Package tools implements the MCP tools the server exposes: actions the AI can invoke.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
	"gopkg.in/yaml.v3"
)

// NewDesignTools returns the Meshery Server design tools: list, get, create,
// deploy, undeploy, and delete designs.
func NewDesignTools(client *meshery.Client) []Tool {
	return []Tool{
		{
			Name:        "list_designs",
			Description: "List designs in the workspace.",
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
						"description": "Filter designs by case-insensitive substring match on name.",
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

				pageResp, err := client.ListDesigns(ctx, meshery.DesignListOptions{
					ListOptions: meshery.ListOptions{
						Page: page, PageSize: pageSize, Search: search,
					},
				})
				if err != nil {
					return nil, err
				}

				designs := make([]DesignSummary, 0, len(pageResp.Patterns))
				for _, p := range pageResp.Patterns {
					designs = append(designs, DesignSummary{ID: p.ID, Name: p.Name})
				}
				return ListDesignsResult{Designs: designs, Total: pageResp.TotalCount}, nil
			},
		},
		{
			Name:        "get_design",
			Description: "Get design details by ID from Meshery Server.",
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

				design, err := client.GetDesign(ctx, designID)
				if err != nil {
					return nil, err
				}
				return GetDesignResult{Design: designDetail(design)}, nil
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
						"description": "Design content as YAML or JSON.",
						"minLength":   1,
						"maxLength":   1000000, // 1 MB max
					},
					"confirm": map[string]any{
						"type":        "boolean",
						"description": "Must be true to confirm creation.",
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

				body, err := createDesignBody(name, designYAML)
				if err != nil {
					return nil, err
				}

				resp, err := client.Call(ctx, http.MethodPost, "/api/pattern", body, "create_design")
				if err != nil {
					return nil, err
				}

				var created struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				if err := json.Unmarshal(resp, &created); err != nil {
					return nil, fmt.Errorf("decoding create design response: %w", err)
				}

				return CreateDesignResult{
					ID:   created.ID,
					Name: created.Name,
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
						"description": "The Kubernetes cluster ID (context ID) to deploy to.",
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
				design, err := getDesignByID(ctx, client, designID)
				if err != nil {
					return nil, fmt.Errorf("fetching design %q for deploy: %w", designID, err)
				}

				deployBody, err := json.Marshal(map[string]any{
					"patternFile": design.PatternFile,
					"patternID":   designID,
				})
				if err != nil {
					return nil, fmt.Errorf("marshaling deploy request: %w", err)
				}

				q := url.Values{}
				q.Add("contexts", clusterID)

				resp, err := client.Call(ctx, http.MethodPost, "/api/pattern/deploy?"+q.Encode(), deployBody, "deploy_design")
				if err != nil {
					return nil, err
				}

				var deployResp struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}
				if err := json.Unmarshal(resp, &deployResp); err != nil {
					return nil, fmt.Errorf("decoding deploy response: %w", err)
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
						"description": "The Kubernetes cluster ID (context ID) to undeploy from.",
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
				if !confirm {
					return UndeployDesignResult{
						DesignID:  designID,
						ClusterID: clusterID,
						Status:    "pending_confirmation",
					}, fmt.Errorf("undeploy_design would undeploy design %q from cluster %q. Set confirm: true to proceed", designID, clusterID)
				}
				if designID == "" {
					return nil, fmt.Errorf("design_id is required")
				}
				if clusterID == "" {
					return nil, fmt.Errorf("cluster_id is required")
				}

				// First, get the design to obtain its patternFile
				design, err := getDesignByID(ctx, client, designID)
				if err != nil {
					return nil, fmt.Errorf("fetching design %q for undeploy: %w", designID, err)
				}

				undeployBody, err := json.Marshal(map[string]any{
					"patternFile": design.PatternFile,
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

				_, err := client.Call(ctx, http.MethodDelete, "/api/pattern/"+url.PathEscape(designID), nil, "delete_design")
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

// createDesignBody builds the POST /api/pattern payload. Meshery Server decodes
// designFile into a structured design.PatternFile, so YAML/JSON text from the
// tool is parsed into an object rather than sent as a JSON string.
func createDesignBody(name, designYAML string) ([]byte, error) {
	var content any
	if err := yaml.Unmarshal([]byte(designYAML), &content); err != nil {
		return nil, fmt.Errorf("parsing design_yaml: %w", err)
	}
	if _, ok := content.(map[string]any); !ok {
		return nil, fmt.Errorf("design_yaml must be a YAML or JSON object")
	}
	body, err := json.Marshal(map[string]any{"name": name, "designFile": content})
	if err != nil {
		return nil, fmt.Errorf("marshaling create_design request: %w", err)
	}
	return body, nil
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
}

type GetDesignResult struct {
	Design DesignDetail `json:"design"`
}

type DesignDetail struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Version            string `json:"version"`
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
	Visibility         string `json:"visibility,omitempty"`
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

// getDesignByID fetches a design by ID from Meshery Server.
func getDesignByID(ctx context.Context, client *meshery.Client, designID string) (DesignDetail, error) {
	design, err := client.GetDesign(ctx, designID)
	if err != nil {
		return DesignDetail{}, err
	}
	return designDetail(design), nil
}

func designDetail(design meshery.Design) DesignDetail {
	return DesignDetail{
		ID:          design.ID,
		Name:        design.Name,
		PatternFile: patternFileString(design.PatternFile),
		Visibility:  design.Visibility,
		CreatedAt:   design.CreatedAt,
		UpdatedAt:   design.UpdatedAt,
	}
}

// patternFileString converts the design's JSON patternFile (a string on
// Cloud, an object on some servers) to the string the tools return.
func patternFileString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}
