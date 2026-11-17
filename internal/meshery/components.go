package meshery

import (
	"context"

	componentv1beta3 "github.com/meshery/schemas/models/v1beta3/component"
)

// Components will call Meshery's component endpoints.
type Components struct {
	client *Client
}

func (c *Components) SearchComponents(_ context.Context, _, _, _ string) ([]componentv1beta3.ComponentDefinition, error) {
	return nil, ErrNotImplemented("search_components")
}

func (c *Components) GetComponent(_ context.Context, _ string) (componentv1beta3.ComponentDefinition, error) {
	return componentv1beta3.ComponentDefinition{}, ErrNotImplemented("get_component")
}
