package meshery

import (
	"context"

	"github.com/gofrs/uuid"
	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"

	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
)

type Models struct {
	client *Client
}

// TODO: GET /api/registry/models?search=&page=&pageSize=; search matches the
// name and display name.
func (m *Models) ListModels(_ context.Context, _ string, _ tools.Page) (tools.Result[modelv1beta1.ModelDefinition], error) {
	return tools.Result[modelv1beta1.ModelDefinition]{}, ErrNotImplemented("list_models")
}

// TODO: GET /api/registry/models?id=<id>. An empty list means
// tools.ErrNotFound.
func (m *Models) GetModel(_ context.Context, _ uuid.UUID) (modelv1beta1.ModelDefinition, error) {
	return modelv1beta1.ModelDefinition{}, ErrNotImplemented("get_model")
}
