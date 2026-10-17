package tools

import (
	"context"

	relationshipv1alpha3 "github.com/meshery/schemas/models/v1alpha3/relationship"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RelationshipRegistry is what the relationship tools need from Meshery.
type RelationshipRegistry interface {
	// ListRelationships returns relationship definitions of the Meshery model
	// with this name; an empty model returns them for every model.
	ListRelationships(ctx context.Context, model string) ([]relationshipv1alpha3.RelationshipDefinition, error)
}

type relationshipHandler struct {
	registry RelationshipRegistry
}

func relationshipToolset(r RelationshipRegistry) toolset {
	h := &relationshipHandler{registry: r}
	return toolset{
		name: "relationships",
		tools: []toolEntry{
			newToolEntry(listRelationshipsTool, h.listRelationships),
		},
	}
}

// RelationshipSummary is the part of a relationship definition the AI sees.
type RelationshipSummary struct {
	ID          string `json:"id" jsonschema:"ID of the relationship"`
	Kind        string `json:"kind" jsonschema:"kind of relationship: hierarchical, edge or sibling"`
	Type        string `json:"type" jsonschema:"type of the relationship within its kind"`
	SubType     string `json:"subType" jsonschema:"subtype of the relationship"`
	Model       string `json:"model" jsonschema:"name of the Meshery model that defines the relationship"`
	Description string `json:"description" jsonschema:"what the relationship means"`
}

func relationshipSummary(r relationshipv1alpha3.RelationshipDefinition) RelationshipSummary {
	s := RelationshipSummary{
		ID:      r.ID.String(),
		Kind:    string(r.Kind),
		Type:    r.RelationshipType,
		SubType: r.SubType,
		Model:   r.Model.Name,
	}
	if r.Metadata != nil && r.Metadata.Description != nil {
		s.Description = *r.Metadata.Description
	}
	return s
}

var listRelationshipsTool = &mcp.Tool{
	Name:        "list_relationships",
	Title:       "List relationships",
	Description: "List the relationships components can have, such as hierarchical or edge relationships. " + notImplementedNote,
	Annotations: readOnly(),
}

type ListRelationshipsIn struct {
	Model string `json:"model,omitempty" jsonschema:"only return relationships of the Meshery model with this name, for example kubernetes"`
}

type ListRelationshipsOut struct {
	Relationships []RelationshipSummary `json:"relationships" jsonschema:"relationship definitions"`
}

func (h *relationshipHandler) listRelationships(ctx context.Context, _ *mcp.CallToolRequest, in ListRelationshipsIn) (*mcp.CallToolResult, ListRelationshipsOut, error) {
	relationships, err := h.registry.ListRelationships(ctx, in.Model)
	if err != nil {
		return nil, ListRelationshipsOut{}, fromRegistry(err, ErrListRelationships)
	}
	out := ListRelationshipsOut{Relationships: make([]RelationshipSummary, 0, len(relationships))}
	for _, r := range relationships {
		out.Relationships = append(out.Relationships, relationshipSummary(r))
	}
	return nil, out, nil
}
