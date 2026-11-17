package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"
	componentv1beta3 "github.com/meshery/schemas/models/v1beta3/component"
)

const componentID = "5f4e2a1c-8d3b-4c6e-9a7f-1b2c3d4e5f60"

func podDefinition(t *testing.T) componentv1beta3.ComponentDefinition {
	t.Helper()
	return componentv1beta3.ComponentDefinition{
		ID:          mustUUID(t, componentID),
		DisplayName: "Pod",
		Description: "A group of containers",
		Component: componentv1beta3.Component{
			Kind:    "Pod",
			Version: "v1",
			Schema:  `{"type":"object"}`,
		},
		ModelReference: modelv1beta1.ModelReference{
			Name:  "kubernetes",
			Model: modelv1beta1.Model{Version: "v1.31.0"},
		},
	}
}

func TestComponentSummary(t *testing.T) {
	got := componentSummary(podDefinition(t))
	want := ComponentSummary{
		ID:           componentID,
		Kind:         "Pod",
		APIVersion:   "v1",
		DisplayName:  "Pod",
		Model:        "kubernetes",
		ModelVersion: "v1.31.0",
	}
	if got != want {
		t.Errorf("componentSummary = %+v, want %+v", got, want)
	}
}

func TestSearchComponents(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("query or model is required", func(t *testing.T) {
		fake := &fakeBackend{}
		h := &componentHandler{registry: fake}

		_, _, err := h.searchComponents(context.Background(), nil, SearchComponentsIn{})
		wantError(t, err, ErrRequiredOneOfCode, "query", "model")
		if fake.calls != 0 {
			t.Errorf("registry calls = %d, want 0", fake.calls)
		}
	})

	t.Run("model alone is enough", func(t *testing.T) {
		fake := &fakeBackend{}
		h := &componentHandler{registry: fake}

		if _, _, err := h.searchComponents(context.Background(), nil, SearchComponentsIn{Model: "kubernetes"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotModel != "kubernetes" || fake.gotQuery != "" {
			t.Errorf("registry got query=%q model=%q", fake.gotQuery, fake.gotModel)
		}
	})

	t.Run("invalid page skips the registry", func(t *testing.T) {
		fake := &fakeBackend{}
		h := &componentHandler{registry: fake}

		_, _, err := h.searchComponents(context.Background(), nil, SearchComponentsIn{Query: "pod", PageIn: PageIn{Page: -1}})
		wantError(t, err, ErrNegativeArgumentCode, `"page"`)
		if fake.calls != 0 {
			t.Errorf("registry calls = %d, want 0", fake.calls)
		}
	})

	t.Run("nil result becomes an empty list", func(t *testing.T) {
		h := &componentHandler{registry: &fakeBackend{}}

		_, out, err := h.searchComponents(context.Background(), nil, SearchComponentsIn{Query: "pod"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Components == nil || len(out.Components) != 0 {
			t.Errorf("Components = %#v, want empty non-nil slice", out.Components)
		}
	})

	t.Run("passes filters and page through, and reports paging", func(t *testing.T) {
		fake := &fakeBackend{components: []componentv1beta3.ComponentDefinition{podDefinition(t)}, total: 30}
		h := &componentHandler{registry: fake}

		in := SearchComponentsIn{Query: "pod", Model: "kubernetes", PageIn: PageIn{Page: 1, PageSize: 10}}
		_, out, err := h.searchComponents(context.Background(), nil, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotQuery != "pod" || fake.gotModel != "kubernetes" || fake.gotPage != (Page{Index: 1, Size: 10}) {
			t.Errorf("registry got query=%q model=%q page=%+v", fake.gotQuery, fake.gotModel, fake.gotPage)
		}
		if len(out.Components) != 1 || out.Components[0].ID != componentID {
			t.Errorf("Components = %+v, want the mapped pod", out.Components)
		}
		if out.PageOut != (PageOut{Page: 1, PageSize: 10, TotalCount: 30, HasMore: true}) {
			t.Errorf("PageOut = %+v", out.PageOut)
		}
	})

	t.Run("wraps registry errors", func(t *testing.T) {
		h := &componentHandler{registry: &fakeBackend{err: errBoom}}

		_, _, err := h.searchComponents(context.Background(), nil, SearchComponentsIn{Query: "pod"})
		wantError(t, err, ErrSearchComponentsCode, "boom")
	})
}

func TestGetComponent(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("invalid id skips the registry", func(t *testing.T) {
		for _, tt := range []struct{ id, code string }{
			{id: "", code: ErrRequiredArgumentCode},
			{id: "c1", code: ErrInvalidIDCode},
		} {
			fake := &fakeBackend{}
			h := &componentHandler{registry: fake}

			_, _, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: tt.id})
			wantError(t, err, tt.code, `"id"`)
			if fake.calls != 0 {
				t.Errorf("%q: registry calls = %d, want 0", tt.id, fake.calls)
			}
		}
	})

	t.Run("passes the id on in canonical form", func(t *testing.T) {
		fake := &fakeBackend{component: podDefinition(t)}
		h := &componentHandler{registry: fake}

		if _, _, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: strings.ToUpper(componentID)}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotID.String() != componentID {
			t.Errorf("registry got id %s, want %s", fake.gotID, componentID)
		}
	})

	t.Run("returns summary, description and schema as an object", func(t *testing.T) {
		h := &componentHandler{registry: &fakeBackend{component: podDefinition(t)}}

		_, out, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: componentID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Component.Kind != "Pod" || out.Description != "A group of containers" {
			t.Errorf("out = %+v", out)
		}
		if out.Schema["type"] != "object" {
			t.Errorf("Schema = %#v, want the decoded object", out.Schema)
		}
	})

	t.Run("empty schema is left out", func(t *testing.T) {
		c := podDefinition(t)
		c.Component.Schema = ""
		h := &componentHandler{registry: &fakeBackend{component: c}}

		_, out, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: componentID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Schema != nil {
			t.Errorf("Schema = %#v, want nil", out.Schema)
		}
	})

	t.Run("schema that is not a JSON object is an error", func(t *testing.T) {
		for _, schema := range []string{"{not json", `"a string"`, "[1, 2]"} {
			c := podDefinition(t)
			c.Component.Schema = schema
			h := &componentHandler{registry: &fakeBackend{component: c}}

			_, _, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: componentID})
			wantError(t, err, ErrInvalidComponentSchemaCode, componentID)
		}
	})

	t.Run("not found is reported as such", func(t *testing.T) {
		h := &componentHandler{registry: &fakeBackend{err: fmt.Errorf("lookup: %w", ErrNotFound)}}

		_, _, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: componentID})
		wantError(t, err, ErrComponentNotFoundCode, componentID)
	})

	t.Run("wraps registry errors", func(t *testing.T) {
		h := &componentHandler{registry: &fakeBackend{err: errBoom}}

		_, _, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: componentID})
		wantError(t, err, ErrGetComponentCode, componentID, "boom")
	})
}

func TestComponentTools_Spec(t *testing.T) {
	t.Run("search_components", func(t *testing.T) {
		// The handler, not the schema, requires query or model.
		wantTool(t, "search_components", toolSpec{readOnly: true, notImplemented: true})
	})
	t.Run("get_component", func(t *testing.T) {
		wantTool(t, "get_component", toolSpec{required: []string{"id"}, readOnly: true, notImplemented: true})
	})
}
