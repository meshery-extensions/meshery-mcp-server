package tools

import (
	"context"

	componentv1beta3 "github.com/meshery/schemas/models/v1beta3/component"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ComponentRegistry is what the component tools need from Meshery.
type ComponentRegistry interface {
	// SearchComponents returns components whose name matches query. kind and
	// model narrow the results; an empty string means no filter.
	SearchComponents(ctx context.Context, query, kind, model string) ([]componentv1beta3.ComponentDefinition, error)
	GetComponent(ctx context.Context, id string) (componentv1beta3.ComponentDefinition, error)
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

// ComponentSummary is the part of a component definition the AI sees.
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
	Name:        "search_components",
	Title:       "Search components",
	Description: "Search components across all Meshery models. " + notImplementedNote,
	Annotations: readOnly(),
}

type SearchComponentsIn struct {
	Query string `json:"query" jsonschema:"text to match against component names"`
	Kind  string `json:"kind,omitempty" jsonschema:"only return components of this kind, for example Deployment"`
	Model string `json:"model,omitempty" jsonschema:"only return components of the Meshery model with this name, for example kubernetes"`
}

type SearchComponentsOut struct {
	Components []ComponentSummary `json:"components" jsonschema:"matching components"`
}

func (h *componentHandler) searchComponents(ctx context.Context, _ *mcp.CallToolRequest, in SearchComponentsIn) (*mcp.CallToolResult, SearchComponentsOut, error) {
	if in.Query == "" {
		return nil, SearchComponentsOut{}, ErrRequiredArgument("query")
	}
	components, err := h.registry.SearchComponents(ctx, in.Query, in.Kind, in.Model)
	if err != nil {
		return nil, SearchComponentsOut{}, fromRegistry(err, ErrSearchComponents)
	}
	out := SearchComponentsOut{Components: make([]ComponentSummary, 0, len(components))}
	for _, c := range components {
		out.Components = append(out.Components, componentSummary(c))
	}
	return nil, out, nil
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
	Schema      string           `json:"schema" jsonschema:"JSON Schema of the component's configuration"`
}

func (h *componentHandler) getComponent(ctx context.Context, _ *mcp.CallToolRequest, in GetComponentIn) (*mcp.CallToolResult, GetComponentOut, error) {
	if in.ID == "" {
		return nil, GetComponentOut{}, ErrRequiredArgument("id")
	}
	c, err := h.registry.GetComponent(ctx, in.ID)
	if err != nil {
		return nil, GetComponentOut{}, fromRegistry(err, func(err error) error { return ErrGetComponent(err, in.ID) })
	}
	return nil, GetComponentOut{
		Component:   componentSummary(c),
		Description: c.Description,
		Schema:      c.Component.Schema,
	}, nil
}
