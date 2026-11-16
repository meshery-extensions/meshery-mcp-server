package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofrs/uuid"
	componentv1beta3 "github.com/meshery/schemas/models/v1beta3/component"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ComponentRegistry interface {
	// An empty query or model does not filter. Results may omit schemas.
	SearchComponents(ctx context.Context, query, model string, p Page) (Result[componentv1beta3.ComponentDefinition], error)
	GetComponent(ctx context.Context, id uuid.UUID) (componentv1beta3.ComponentDefinition, error)
}

type componentHandler struct {
	registry ComponentRegistry
}

func componentToolset(r ComponentRegistry) toolset {
	h := &componentHandler{registry: r}
	return toolset{
		name: "components",
		tools: []toolEntry{
			newToolEntry(searchComponentsTool, h.searchComponents),
			newToolEntry(getComponentTool, h.getComponent),
		},
	}
}

type ComponentSummary struct {
	ID           string `json:"id" jsonschema:"ID of the component; pass it to get_component"`
	Kind         string `json:"kind" jsonschema:"kind of the component, for example Pod"`
	APIVersion   string `json:"apiVersion" jsonschema:"API version of the component, for example v1"`
	DisplayName  string `json:"displayName" jsonschema:"human-readable name of the component"`
	Model        string `json:"model" jsonschema:"name of the Meshery model the component belongs to"`
	ModelVersion string `json:"modelVersion" jsonschema:"version of that model"`
}

func componentSummary(c componentv1beta3.ComponentDefinition) ComponentSummary {
	return ComponentSummary{
		ID:           c.ID.String(),
		Kind:         c.Component.Kind,
		APIVersion:   c.Component.Version,
		DisplayName:  c.DisplayName,
		Model:        c.ModelReference.Name,
		ModelVersion: c.ModelReference.Model.Version,
	}
}

var searchComponentsTool = &mcp.Tool{
	Name:  "search_components",
	Title: "Search components",
	Description: "Search components by display name, across all Meshery models or within one. " +
		"Results leave out each component's schema; get it with get_component. " + notImplementedNote,
	Annotations: readOnly(),
}

type SearchComponentsIn struct {
	Query string `json:"query,omitempty" jsonschema:"text to match against component display names, for example Deployment; set query, model or both"`
	Model string `json:"model,omitempty" jsonschema:"only return components of the Meshery model with this name, for example kubernetes; set query, model or both"`
	PageIn
}

type SearchComponentsOut struct {
	Components []ComponentSummary `json:"components" jsonschema:"matching components on this page"`
	PageOut
}

func (h *componentHandler) searchComponents(ctx context.Context, _ *mcp.CallToolRequest, in SearchComponentsIn) (*mcp.CallToolResult, SearchComponentsOut, error) {
	if in.Query == "" && in.Model == "" {
		return nil, SearchComponentsOut{}, ErrRequiredOneOf("query", "model")
	}
	p, err := in.resolve()
	if err != nil {
		return nil, SearchComponentsOut{}, err
	}
	res, err := h.registry.SearchComponents(ctx, in.Query, in.Model, p)
	if err != nil {
		return nil, SearchComponentsOut{}, fromBackend(err, ErrSearchComponents)
	}
	return nil, SearchComponentsOut{
		Components: mapItems(res.Items, componentSummary),
		PageOut:    p.out(res.Total),
	}, nil
}

var getComponentTool = &mcp.Tool{
	Name:        "get_component",
	Title:       "Get component",
	Description: "Get a component's details, including the JSON Schema of its configuration. " + notImplementedNote,
	Annotations: readOnly(),
}

type GetComponentIn struct {
	ID string `json:"id" jsonschema:"ID of the component, from search_components"`
}

type GetComponentOut struct {
	Component   ComponentSummary `json:"component" jsonschema:"the component"`
	Description string           `json:"description" jsonschema:"what the component is for"`
	// An object, not Meshery's JSON text, which would reach the AI escaped.
	// json.RawMessage is not used: the SDK infers it as an array of bytes.
	Schema map[string]any `json:"schema,omitempty" jsonschema:"JSON Schema of the component's configuration; absent if the component has none"`
}

func (h *componentHandler) getComponent(ctx context.Context, _ *mcp.CallToolRequest, in GetComponentIn) (*mcp.CallToolResult, GetComponentOut, error) {
	id, err := parseID("id", in.ID)
	if err != nil {
		return nil, GetComponentOut{}, err
	}
	c, err := h.registry.GetComponent(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, GetComponentOut{}, ErrComponentNotFound(id.String())
	}
	if err != nil {
		return nil, GetComponentOut{}, fromBackend(err, func(err error) error { return ErrGetComponent(err, id.String()) })
	}
	schema, err := decodeSchema(c.Component.Schema)
	if err != nil {
		return nil, GetComponentOut{}, ErrInvalidComponentSchema(err, id.String())
	}
	return nil, GetComponentOut{
		Component:   componentSummary(c),
		Description: c.Description,
		Schema:      schema,
	}, nil
}

// Annotation components have an empty schema, which decodes to nil.
func decodeSchema(text string) (map[string]any, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(text), &schema); err != nil {
		return nil, err
	}
	return schema, nil
}
