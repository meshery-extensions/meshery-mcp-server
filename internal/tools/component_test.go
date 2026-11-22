package tools

import (
	"context"
	"errors"
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

	t.Run("empty query skips the registry", func(t *testing.T) {
		fake := &fakeRegistry{}
		h := &componentHandler{registry: fake}

		_, _, err := h.searchComponents(context.Background(), nil, SearchComponentsIn{})
		wantError(t, err, ErrRequiredArgumentCode, `"query"`)
		if fake.calls != 0 {
			t.Errorf("registry calls = %d, want 0", fake.calls)
		}
	})

	t.Run("nil result becomes an empty list", func(t *testing.T) {
		h := &componentHandler{registry: &fakeRegistry{}}

		_, out, err := h.searchComponents(context.Background(), nil, SearchComponentsIn{Query: "pod"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Components == nil || len(out.Components) != 0 {
			t.Errorf("Components = %#v, want empty non-nil slice", out.Components)
		}
	})

	t.Run("passes filters through and maps results", func(t *testing.T) {
		fake := &fakeRegistry{components: []componentv1beta3.ComponentDefinition{podDefinition(t)}}
		h := &componentHandler{registry: fake}

		in := SearchComponentsIn{Query: "pod", Kind: "Pod", Model: "kubernetes"}
		_, out, err := h.searchComponents(context.Background(), nil, in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotQuery != "pod" || fake.gotKind != "Pod" || fake.gotModel != "kubernetes" {
			t.Errorf("registry got query=%q kind=%q model=%q", fake.gotQuery, fake.gotKind, fake.gotModel)
		}
		if len(out.Components) != 1 || out.Components[0].ID != componentID {
			t.Errorf("Components = %+v, want the mapped pod", out.Components)
		}
	})

	t.Run("wraps registry errors", func(t *testing.T) {
		h := &componentHandler{registry: &fakeRegistry{err: errBoom}}

		_, _, err := h.searchComponents(context.Background(), nil, SearchComponentsIn{Query: "pod"})
		wantError(t, err, ErrSearchComponentsCode, "boom")
	})
}

func TestGetComponent(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("empty id skips the registry", func(t *testing.T) {
		fake := &fakeRegistry{}
		h := &componentHandler{registry: fake}

		_, _, err := h.getComponent(context.Background(), nil, GetComponentIn{})
		wantError(t, err, ErrRequiredArgumentCode, `"id"`)
		if fake.calls != 0 {
			t.Errorf("registry calls = %d, want 0", fake.calls)
		}
	})

	t.Run("returns summary, description and schema", func(t *testing.T) {
		fake := &fakeRegistry{component: podDefinition(t)}
		h := &componentHandler{registry: fake}

		_, out, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: componentID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotID != componentID {
			t.Errorf("registry got id %q, want %q", fake.gotID, componentID)
		}
		if out.Component.Kind != "Pod" || out.Description != "A group of containers" || out.Schema != `{"type":"object"}` {
			t.Errorf("out = %+v", out)
		}
	})

	t.Run("wraps registry errors", func(t *testing.T) {
		h := &componentHandler{registry: &fakeRegistry{err: errBoom}}

		_, _, err := h.getComponent(context.Background(), nil, GetComponentIn{ID: "c1"})
		wantError(t, err, ErrGetComponentCode, `"c1"`, "boom")
	})
}
