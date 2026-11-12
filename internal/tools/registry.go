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

// NewTools returns the registry MCP tools backed by the Meshery client.
// All tools are read-only: they only GET from the Meshery registry.
func NewTools(client *meshery.Client) []Tool {
	return []Tool{
		{
			Name:        "list_models",
			Description: "List registered infrastructure models from the Meshery registry.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"search": map[string]any{
						"type":        "string",
						"description": "Filter results by a case-insensitive substring match on model name.",
					},
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

				body, err := client.Call(ctx, http.MethodGet, "/api/registry/models?"+q.Encode(), "list_models")
				if err != nil {
					return nil, err
				}

				var pageResp struct {
					Page     int            `json:"page"`
					PageSize int            `json:"pageSize"`
					Total    int            `json:"totalCount"`
					Models   []ModelSummary `json:"models"`
				}
				if err := json.Unmarshal(body, &pageResp); err != nil {
					return nil, fmt.Errorf("decoding models response: %w", err)
				}

				claims := make([]ModelSummary, 0, len(pageResp.Models))
				for _, m := range pageResp.Models {
					claims = append(claims, ModelSummary{
						ID:       m.ID,
						Name:     m.Name,
						Version:  m.Version,
						Category: m.Category,
					})
				}

				return ListModelsResult{
					Claims: claims,
					Total:  pageResp.Total,
				}, nil
			},
		},
		{
			Name:        "get_model",
			Description: "Get full model details from the Meshery registry.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"model_id"},
				"properties": map[string]any{
					"model_id": map[string]any{
						"type":        "string",
						"description": "The model ID (name) to look up.",
						"minLength":   1,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				modelID, _ := params["model_id"].(string)
				if modelID == "" {
					return nil, fmt.Errorf("model_id is required")
				}

				body, err := client.Call(ctx, http.MethodGet, "/api/registry/models/"+url.PathEscape(modelID), "get_model")
				if err != nil {
					return nil, err
				}

				var pageResp struct {
					Models []ModelDetail `json:"models"`
				}
				if err := json.Unmarshal(body, &pageResp); err != nil {
					return nil, fmt.Errorf("decoding model response: %w", err)
				}
				if len(pageResp.Models) == 0 {
					return nil, fmt.Errorf("model %q not found", modelID)
				}

				return GetModelResult{Model: pageResp.Models[0]}, nil
			},
		},
		{
			Name:        "search_components",
			Description: "Search components across all models in the Meshery registry.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"search": map[string]any{
						"type":        "string",
						"description": "Filter results by a case-insensitive substring match on component name.",
					},
					"kind": map[string]any{
						"type":        "string",
						"description": "Filter by component kind (e.g. Deployment).",
					},
					"model": map[string]any{
						"type":        "string",
						"description": "Filter by the model the component belongs to.",
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				search, _ := params["search"].(string)
				kind, _ := params["kind"].(string)
				model, _ := params["model"].(string)

				var path string
				if model != "" {
					path = "/api/registry/models/" + url.PathEscape(model) + "/components"
				} else if kind != "" {
					path = "/api/registry/components/" + url.PathEscape(kind)
				} else {
					path = "/api/registry/components"
				}

				q := url.Values{}
				if search != "" {
					q.Set("search", search)
				}
				q.Set("page", "0")
				q.Set("page_size", "100")

				body, err := client.Call(ctx, http.MethodGet, path+"?"+q.Encode(), "search_components")
				if err != nil {
					return nil, err
				}

				var pageResp struct {
					Components []struct {
						ID      string `json:"id"`
						Name    string `json:"displayName"`
						Version string `json:"version"`
					} `json:"components"`
					Total int `json:"totalCount"`
				}
				if err := json.Unmarshal(body, &pageResp); err != nil {
					return nil, fmt.Errorf("decoding components response: %w", err)
				}

				items := make([]ContentItemSummary, 0, len(pageResp.Components))
				for _, c := range pageResp.Components {
					items = append(items, ContentItemSummary{
						ID:      c.ID,
						Name:    c.Name,
						Version: c.Version,
					})
				}

				return SearchContentItemsResult{
					ContentItems: items,
					Total:        pageResp.Total,
				}, nil
			},
		},
		{
			Name:        "get_component",
			Description: "Get detailed component schema from the Meshery registry.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"component_id"},
				"properties": map[string]any{
					"component_id": map[string]any{
						"type":        "string",
						"description": "The component kind/name (e.g. Deployment) to look up.",
						"minLength":   1,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				componentID, _ := params["component_id"].(string)
				if componentID == "" {
					return nil, fmt.Errorf("component_id is required")
				}

				body, err := client.Call(ctx, http.MethodGet, "/api/registry/components/"+url.PathEscape(componentID), "get_component")
				if err != nil {
					return nil, err
				}

				var pageResp struct {
					Components []ComponentDetail `json:"components"`
				}
				if err := json.Unmarshal(body, &pageResp); err != nil {
					return nil, fmt.Errorf("decoding component response: %w", err)
				}
				if len(pageResp.Components) == 0 {
					return nil, fmt.Errorf("component %q not found", componentID)
				}

				return GetContentItemResult{Component: pageResp.Components[0]}, nil
			},
		},
		{
			Name:        "list_relationships",
			Description: "List available component relationship types for a model.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"model"},
				"properties": map[string]any{
					"model": map[string]any{
						"type":        "string",
						"description": "The model whose relationships to list.",
						"minLength":   1,
					},
				},
			},
			Handler: func(ctx context.Context, args any) (any, error) {
				params := map[string]any{}
				if err := decodeArgs(args, params); err != nil {
					return nil, err
				}
				model, _ := params["model"].(string)
				if model == "" {
					return nil, fmt.Errorf("model is required")
				}

				q := url.Values{}
				q.Set("page", "0")
				q.Set("page_size", "100")

				body, err := client.Call(ctx, http.MethodGet, "/api/registry/models/"+url.PathEscape(model)+"/relationships?"+q.Encode(), "list_relationships")
				if err != nil {
					return nil, err
				}

				var pageResp struct {
					Relationships []RelationshipSummary `json:"relationships"`
					Total         int                   `json:"totalCount"`
				}
				if err := json.Unmarshal(body, &pageResp); err != nil {
					return nil, fmt.Errorf("decoding relationships response: %w", err)
				}

				rels := make([]RelationshipSummary, 0, len(pageResp.Relationships))
				for _, r := range pageResp.Relationships {
					rels = append(rels, RelationshipSummary{
						ID:      r.ID,
						Kind:    r.Kind,
						Type:    r.Type,
						SubType: r.SubType,
					})
				}

				return ListLedgersResult{
					Relationships: rels,
					Total:         pageResp.Total,
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

type ListModelsResult struct {
	Claims []ModelSummary `json:"claims"`
	Total  int            `json:"total"`
}

type ComponentSummary struct {
	ID      string `json:"id"`
	Name    string `json:"displayName"`
	Version string `json:"version"`
}

type ModelSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Category string `json:"category,omitempty"`
}

type GetModelResult struct {
	Model ModelDetail `json:"model"`
}

type ModelDetail struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Category    string `json:"category,omitempty"`
}

type SearchContentItemsResult struct {
	ContentItems []ContentItemSummary `json:"content_items"`
	Total        int                  `json:"total"`
}

type ContentItemSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type GetContentItemResult struct {
	Component ComponentDetail `json:"component"`
}

type ComponentDetail struct {
	ID          string `json:"id"`
	Name        string `json:"displayName"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Format      string `json:"format,omitempty"`
	Status      string `json:"status,omitempty"`
	Schema      any    `json:"component,omitempty"`
}

type ListLedgersResult struct {
	Relationships []RelationshipSummary `json:"relationships"`
	Total         int                   `json:"total"`
}

type RelationshipSummary struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Type    string `json:"type,omitempty"`
	SubType string `json:"subType,omitempty"`
}
