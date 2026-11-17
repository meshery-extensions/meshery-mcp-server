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
		h := &relationshipHandler{registry: &fakeBackend{}}

		_, out, err := h.listRelationships(context.Background(), nil, ListRelationshipsIn{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Relationships == nil || len(out.Relationships) != 0 {
			t.Errorf("Relationships = %#v, want empty non-nil slice", out.Relationships)
		}
	})

	t.Run("passes the model and page through, and reports paging", func(t *testing.T) {
		fake := &fakeBackend{total: 120}
		h := &relationshipHandler{registry: fake}

		in := ListRelationshipsIn{Model: "kubernetes", PageIn: PageIn{Page: 1}}
		_, out, err := h.listRelationships(context.Background(), nil, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotModel != in.Model || fake.gotPage != (Page{Index: 1, Size: defaultPageSize}) {
			t.Errorf("registry got model %q page %+v", fake.gotModel, fake.gotPage)
		}
		if out.PageOut != (PageOut{Page: 1, PageSize: defaultPageSize, TotalCount: 120, HasMore: true}) {
			t.Errorf("PageOut = %+v", out.PageOut)
		}
	})

	t.Run("invalid page skips the registry", func(t *testing.T) {
		fake := &fakeBackend{}
		h := &relationshipHandler{registry: fake}

		_, _, err := h.listRelationships(context.Background(), nil, ListRelationshipsIn{PageIn: PageIn{Page: -2}})
		wantError(t, err, ErrNegativeArgumentCode, `"page"`)
		if fake.calls != 0 {
			t.Errorf("registry calls = %d, want 0", fake.calls)
		}
	})

	t.Run("wraps registry errors", func(t *testing.T) {
		h := &relationshipHandler{registry: &fakeBackend{err: errBoom}}

		_, _, err := h.listRelationships(context.Background(), nil, ListRelationshipsIn{})
		wantError(t, err, ErrListRelationshipsCode, "boom")
	})
}

func TestRelationshipTools_Spec(t *testing.T) {
	wantTool(t, "list_relationships", toolSpec{readOnly: true, notImplemented: true})
}
