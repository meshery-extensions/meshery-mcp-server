package tools

import (
	"context"
	"errors"
	"testing"

	categoryv1beta1 "github.com/meshery/schemas/models/v1beta1/category"
	modelv1beta1 "github.com/meshery/schemas/models/v1beta1/model"
)

const modelID = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"

func kubernetesModel(t *testing.T) modelv1beta1.ModelDefinition {
	t.Helper()
	return modelv1beta1.ModelDefinition{
		ID:                 mustUUID(t, modelID),
		Name:               "kubernetes",
		DisplayName:        "Kubernetes",
		Description:        "Container orchestration",
		Model:              modelv1beta1.Model{Version: "v1.31.0"},
		Category:           categoryv1beta1.CategoryDefinition{Name: "Orchestration & Management"},
		ComponentsCount:    3,
		RelationshipsCount: 2,
	}
}

func TestModelSummary(t *testing.T) {
	got := modelSummary(kubernetesModel(t))
	want := ModelSummary{
		ID:          modelID,
		Name:        "kubernetes",
		DisplayName: "Kubernetes",
		Version:     "v1.31.0",
		Category:    "Orchestration & Management",
	}
	if got != want {
		t.Errorf("modelSummary = %+v, want %+v", got, want)
	}
}

func TestListModels(t *testing.T) {
	t.Run("passes the page through and reports paging", func(t *testing.T) {
		fake := &fakeBackend{models: []modelv1beta1.ModelDefinition{kubernetesModel(t)}, total: 3}
		h := &modelHandler{registry: fake}

		_, out, err := h.listModels(context.Background(), nil, ListModelsIn{PageIn: PageIn{Page: 2, PageSize: 1}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotPage != (Page{Index: 2, Size: 1}) {
			t.Errorf("registry got page %+v", fake.gotPage)
		}
		if out.PageOut != (PageOut{Page: 2, PageSize: 1, TotalCount: 3, HasMore: false}) {
			t.Errorf("PageOut = %+v", out.PageOut)
		}
	})

	t.Run("invalid page skips the registry", func(t *testing.T) {
		fake := &fakeBackend{}
		h := &modelHandler{registry: fake}

		_, _, err := h.listModels(context.Background(), nil, ListModelsIn{PageIn: PageIn{PageSize: -1}})
		wantError(t, err, ErrNegativeArgumentCode, `"pageSize"`)
		if fake.calls != 0 {
			t.Errorf("registry calls = %d, want 0", fake.calls)
		}
	})

	t.Run("nil result becomes an empty list", func(t *testing.T) {
		h := &modelHandler{registry: &fakeBackend{}}

		_, out, err := h.listModels(context.Background(), nil, ListModelsIn{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Models == nil || len(out.Models) != 0 {
			t.Errorf("Models = %#v, want empty non-nil slice", out.Models)
		}
	})
}

func TestGetModel(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("invalid id skips the registry", func(t *testing.T) {
		for _, tt := range []struct{ id, code string }{
			{id: "", code: ErrRequiredArgumentCode},
			{id: "m1", code: ErrInvalidIDCode},
		} {
			fake := &fakeBackend{}
			h := &modelHandler{registry: fake}

			_, _, err := h.getModel(context.Background(), nil, GetModelIn{ID: tt.id})
			wantError(t, err, tt.code, `"id"`)
			if fake.calls != 0 {
				t.Errorf("%q: registry calls = %d, want 0", tt.id, fake.calls)
			}
		}
	})

	t.Run("returns summary, description and counts", func(t *testing.T) {
		fake := &fakeBackend{model: kubernetesModel(t)}
		h := &modelHandler{registry: fake}

		_, out, err := h.getModel(context.Background(), nil, GetModelIn{ID: modelID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fake.gotID.String() != modelID {
			t.Errorf("registry got id %s, want %s", fake.gotID, modelID)
		}
		if out.Model.Name != "kubernetes" || out.Description != "Container orchestration" ||
			out.ComponentsCount != 3 || out.RelationshipsCount != 2 {
			t.Errorf("out = %+v", out)
		}
	})

	t.Run("not found is reported as such", func(t *testing.T) {
		h := &modelHandler{registry: &fakeBackend{err: ErrNotFound}}

		_, _, err := h.getModel(context.Background(), nil, GetModelIn{ID: modelID})
		wantError(t, err, ErrModelNotFoundCode, modelID)
	})

	t.Run("wraps registry errors", func(t *testing.T) {
		h := &modelHandler{registry: &fakeBackend{err: errBoom}}

		_, _, err := h.getModel(context.Background(), nil, GetModelIn{ID: modelID})
		wantError(t, err, ErrGetModelCode, modelID, "boom")
	})
}

func TestModelTools_Spec(t *testing.T) {
	t.Run("list_models", func(t *testing.T) {
		wantTool(t, "list_models", toolSpec{readOnly: true, notImplemented: true})
	})
	t.Run("get_model", func(t *testing.T) {
		wantTool(t, "get_model", toolSpec{required: []string{"id"}, readOnly: true, notImplemented: true})
	})
}
