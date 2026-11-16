package meshery

import (
	"context"

	relationshipv1alpha3 "github.com/meshery/schemas/models/v1alpha3/relationship"

	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
)

type Relationships struct {
	client *Client
}

// TODO: GET /api/registry/relationships (or
// /api/registry/models/{model}/relationships) with page and pageSize.
func (r *Relationships) ListRelationships(_ context.Context, _ string, _ tools.Page) (tools.Result[relationshipv1alpha3.RelationshipDefinition], error) {
	return tools.Result[relationshipv1alpha3.RelationshipDefinition]{}, ErrNotImplemented("list_relationships")
}
