package tools

import (
	"context"
	"errors"
	"testing"

	relationshipv1alpha3 "github.com/meshery/schemas/models/v1alpha3/relationship"
	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"
)

func TestRelationshipSummary(t *testing.T) {
	description := "A parent contains its children"
	r := relationshipv1alpha3.RelationshipDefinition{
		ID:               mustUUID(t, "11111111-2222-4333-8444-555555555555"),
		Kind:             relationshipv1alpha3.Hierarchical,
		RelationshipType: "parent",
		SubType:          "inventory",
		Model:            modelv1beta1.ModelReference{Name: "kubernetes"},
		Metadata:         &relationshipv1alpha3.RelationshipMetadata{Description: &description},
	}

	got := relationshipSummary(r)
	want := RelationshipSummary{
		ID:          "11111111-2222-4333-8444-555555555555",
		Kind:        "hierarchical",
		Type:        "parent",
		SubType:     "inventory",
		Model:       "kubernetes",
		Description: description,
	}
	if got != want {
		t.Errorf("relationshipSummary = %+v, want %+v", got, want)
	}

	// A definition without metadata must not panic.
	r.Metadata = nil
	if got := relationshipSummary(r); got.Description != "" {
		t.Errorf("Description = %q, want empty", got.Description)
	}
}

func TestListRelationships(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("nil result becomes an empty list", func(t *testing.T) {
		h := &relationshipHandler{registry: &fakeRegistry{}}

		_, out, err := h.listRelationships(context.Background(), nil, ListRelationshipsIn{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Relationships == nil || len(out.Relationships) != 0 {
			t.Errorf("Relationships = %#v, want empty non-nil slice", out.Relationships)
		}
	})

	t.Run("passes the model filter through", func(t *testing.T) {
		fake := &fakeRegistry{}
		h := &relationshipHandler{registry: fake}

		in := ListRelationshipsIn{Model: "kubernetes"}
		if _, _, err := h.listRelationships(context.Background(), nil, in); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotModel != in.Model {
			t.Errorf("registry got model %q, want %q", fake.gotModel, in.Model)
		}
	})

	t.Run("wraps registry errors", func(t *testing.T) {
		h := &relationshipHandler{registry: &fakeRegistry{err: errBoom}}

		_, _, err := h.listRelationships(context.Background(), nil, ListRelationshipsIn{})
		wantError(t, err, ErrListRelationshipsCode, "boom")
	})
}
