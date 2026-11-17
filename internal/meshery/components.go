package meshery

import (
	"context"

	"github.com/gofrs/uuid"
	componentv1beta3 "github.com/meshery/schemas/models/v1beta3/component"

	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
)

type Components struct {
	client *Client
}

// TODO: GET /api/registry/components?search=&trim=true (or
// /api/registry/models/{model}/components) with page and pageSize. trim drops
// the schemas. Items decode into ComponentDefinition; the {page, pageSize,
// totalCount, components} envelope needs its own struct.
func (c *Components) SearchComponents(_ context.Context, _, _ string, _ tools.Page) (tools.Result[componentv1beta3.ComponentDefinition], error) {
	return tools.Result[componentv1beta3.ComponentDefinition]{}, ErrNotImplemented("search_components")
}

// TODO: GET /api/registry/components?id=<id>; there is no /components/{id}
// route. An empty list means tools.ErrNotFound.
func (c *Components) GetComponent(_ context.Context, _ uuid.UUID) (componentv1beta3.ComponentDefinition, error) {
	return componentv1beta3.ComponentDefinition{}, ErrNotImplemented("get_component")
}
