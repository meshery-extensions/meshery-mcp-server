package tools

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"
	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ModelRegistry interface {
	// An empty search matches every model.
	ListModels(ctx context.Context, search string, p Page) (Result[modelv1beta1.ModelDefinition], error)
	GetModel(ctx context.Context, id uuid.UUID) (modelv1beta1.ModelDefinition, error)
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
	Search string `json:"search,omitempty" jsonschema:"text to match against model names"`
	PageIn
}

type ListModelsOut struct {
	Models []ModelSummary `json:"models" jsonschema:"models on this page"`
	PageOut
}

func (h *modelHandler) listModels(ctx context.Context, _ *mcp.CallToolRequest, in ListModelsIn) (*mcp.CallToolResult, ListModelsOut, error) {
	p, err := in.resolve()
	if err != nil {
		return nil, ListModelsOut{}, err
	}
	res, err := h.registry.ListModels(ctx, in.Search, p)
	if err != nil {
		return nil, ListModelsOut{}, fromBackend(err, ErrListModels)
	}
	return nil, ListModelsOut{
		Models:  mapItems(res.Items, modelSummary),
		PageOut: p.out(res.Total),
	}, nil
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
	id, err := parseID("id", in.ID)
	if err != nil {
		return nil, GetModelOut{}, err
	}
	m, err := h.registry.GetModel(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, GetModelOut{}, ErrModelNotFound(id.String())
	}
	if err != nil {
		return nil, GetModelOut{}, fromBackend(err, func(err error) error { return ErrGetModel(err, id.String()) })
	}
	return nil, GetModelOut{
		Model:              modelSummary(m),
		Description:        m.Description,
		ComponentsCount:    m.ComponentsCount,
		RelationshipsCount: m.RelationshipsCount,
	}, nil
}
