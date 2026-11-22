package meshery

import (
	"context"

	relationshipv1alpha3 "github.com/meshery/schemas/models/v1alpha3/relationship"
)

// Relationships will call Meshery's relationship endpoints.
type Relationships struct {
	client *Client
}

func (r *Relationships) ListRelationships(_ context.Context, _ string) ([]relationshipv1alpha3.RelationshipDefinition, error) {
	return nil, ErrNotImplemented("list_relationships")
}
