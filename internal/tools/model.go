package tools

import (
	"context"

	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

// ModelRegistry is what the model tools need from Meshery.
type ModelRegistry interface {
	// ListModels returns one page of models whose name matches search; an
	// empty search matches every model.
	ListModels(ctx context.Context, search string, page, pageSize int) ([]modelv1beta1.ModelDefinition, error)
	GetModel(ctx context.Context, id string) (modelv1beta1.ModelDefinition, error)
}

type modelHandler struct {
	registry ModelRegistry
}

func modelToolset(r ModelRegistry) toolset {
	h := &modelHandler{registry: r}
	return toolset{
		name: "models",
		tools: []toolEntry{
			newToolEntry(listModelsTool, h.listModels),
			newToolEntry(getModelTool, h.getModel),
		},
	}
}

// ModelSummary is the part of a Meshery model definition the AI sees.
type ModelSummary struct {
	ID          string `json:"id" jsonschema:"ID of the model; pass it to get_model"`
	Name        string `json:"name" jsonschema:"name of the model, for example kubernetes"`
	DisplayName string `json:"displayName" jsonschema:"human-readable name of the model"`
	Version     string `json:"version" jsonschema:"version of the model"`
	Category    string `json:"category" jsonschema:"category of the model"`
}

func modelSummary(m modelv1beta1.ModelDefinition) ModelSummary {
	return ModelSummary{
		ID:          m.ID.String(),
		Name:        m.Name,
		DisplayName: m.DisplayName,
		Version:     m.Model.Version,
		Category:    string(m.Category.Name),
	}
}

var listModelsTool = &mcp.Tool{
	Name:        "list_models",
	Title:       "List models",
	Description: "List registered Meshery models, the packages that define components and relationships. " + notImplementedNote,
	Annotations: readOnly(),
}

type ListModelsIn struct {
	Search   string `json:"search,omitempty" jsonschema:"text to match against model names"`
	Page     int    `json:"page,omitempty" jsonschema:"zero-based page index; defaults to 0"`
	PageSize int    `json:"pageSize,omitempty" jsonschema:"results per page; defaults to 25, at most 100"`
}

type ListModelsOut struct {
	Models []ModelSummary `json:"models" jsonschema:"models on this page"`
}

func (h *modelHandler) listModels(ctx context.Context, _ *mcp.CallToolRequest, in ListModelsIn) (*mcp.CallToolResult, ListModelsOut, error) {
	if in.Page < 0 {
		return nil, ListModelsOut{}, ErrNegativeArgument("page", in.Page)
	}
	switch {
	case in.PageSize < 0:
		return nil, ListModelsOut{}, ErrNegativeArgument("pageSize", in.PageSize)
	case in.PageSize == 0:
		in.PageSize = defaultPageSize
	case in.PageSize > maxPageSize:
		in.PageSize = maxPageSize
	}

	models, err := h.registry.ListModels(ctx, in.Search, in.Page, in.PageSize)
	if err != nil {
		return nil, ListModelsOut{}, fromRegistry(err, ErrListModels)
	}
	out := ListModelsOut{Models: make([]ModelSummary, 0, len(models))}
	for _, m := range models {
		out.Models = append(out.Models, modelSummary(m))
	}
	return nil, out, nil
}

var getModelTool = &mcp.Tool{
	Name:        "get_model",
	Title:       "Get model",
	Description: "Get a Meshery model's details. " + notImplementedNote,
	Annotations: readOnly(),
}

type GetModelIn struct {
	ID string `json:"id" jsonschema:"ID of the model, from list_models"`
}

type GetModelOut struct {
	Model              ModelSummary `json:"model" jsonschema:"the model"`
	Description        string       `json:"description" jsonschema:"what the model covers"`
	ComponentsCount    int          `json:"componentsCount" jsonschema:"number of components the model defines"`
	RelationshipsCount int          `json:"relationshipsCount" jsonschema:"number of relationships the model defines"`
}

func (h *modelHandler) getModel(ctx context.Context, _ *mcp.CallToolRequest, in GetModelIn) (*mcp.CallToolResult, GetModelOut, error) {
	if in.ID == "" {
		return nil, GetModelOut{}, ErrRequiredArgument("id")
	}
	m, err := h.registry.GetModel(ctx, in.ID)
	if err != nil {
		return nil, GetModelOut{}, fromRegistry(err, func(err error) error { return ErrGetModel(err, in.ID) })
	}
	return nil, GetModelOut{
		Model:              modelSummary(m),
		Description:        m.Description,
		ComponentsCount:    m.ComponentsCount,
		RelationshipsCount: m.RelationshipsCount,
	}, nil
}
