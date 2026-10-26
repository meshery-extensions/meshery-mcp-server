package meshery

import (
	"context"

	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"
)

// Models will call Meshery's model endpoints.
type Models struct {
	client *Client
}

func (m *Models) ListModels(_ context.Context, _ string, _, _ int) ([]modelv1beta1.ModelDefinition, error) {
	return nil, ErrNotImplemented("list_models")
}

func (m *Models) GetModel(_ context.Context, _ string) (modelv1beta1.ModelDefinition, error) {
	return modelv1beta1.ModelDefinition{}, ErrNotImplemented("get_model")
}
