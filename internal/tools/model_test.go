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

func TestListModels_Pagination(t *testing.T) {
	tests := []struct {
		name         string
		in           ListModelsIn
		wantPage     int
		wantPageSize int
		wantErr      string // argument named in the error; "" = expect success
	}{
		{name: "defaults", in: ListModelsIn{}, wantPage: 0, wantPageSize: defaultPageSize},
		{name: "explicit values", in: ListModelsIn{Page: 2, PageSize: 10}, wantPage: 2, wantPageSize: 10},
		{name: "page size is capped", in: ListModelsIn{PageSize: 500}, wantPageSize: maxPageSize},
		{name: "negative page", in: ListModelsIn{Page: -1}, wantErr: "page"},
		{name: "negative page size", in: ListModelsIn{PageSize: -1}, wantErr: "pageSize"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeRegistry{}
			h := &modelHandler{registry: fake}

			_, _, err := h.listModels(context.Background(), nil, tt.in)
			if tt.wantErr != "" {
				wantError(t, err, ErrNegativeArgumentCode, `"`+tt.wantErr+`"`)
				if fake.calls != 0 {
					t.Errorf("registry calls = %d, want 0", fake.calls)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fake.gotPage != tt.wantPage || fake.gotPageSize != tt.wantPageSize {
				t.Errorf("registry got page=%d pageSize=%d, want %d/%d",
					fake.gotPage, fake.gotPageSize, tt.wantPage, tt.wantPageSize)
			}
		})
	}
}

func TestListModels_NilBecomesEmpty(t *testing.T) {
	h := &modelHandler{registry: &fakeRegistry{}}

	_, out, err := h.listModels(context.Background(), nil, ListModelsIn{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Models == nil || len(out.Models) != 0 {
		t.Errorf("Models = %#v, want empty non-nil slice", out.Models)
	}
}

func TestGetModel(t *testing.T) {
	errBoom := errors.New("boom")

	t.Run("empty id skips the registry", func(t *testing.T) {
		fake := &fakeRegistry{}
		h := &modelHandler{registry: fake}

		_, _, err := h.getModel(context.Background(), nil, GetModelIn{})
		wantError(t, err, ErrRequiredArgumentCode, `"id"`)
		if fake.calls != 0 {
			t.Errorf("registry calls = %d, want 0", fake.calls)
		}
	})

	t.Run("returns summary, description and counts", func(t *testing.T) {
		h := &modelHandler{registry: &fakeRegistry{model: kubernetesModel(t)}}

		_, out, err := h.getModel(context.Background(), nil, GetModelIn{ID: modelID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Model.Name != "kubernetes" || out.Description != "Container orchestration" ||
			out.ComponentsCount != 3 || out.RelationshipsCount != 2 {
			t.Errorf("out = %+v", out)
		}
	})

	t.Run("wraps registry errors", func(t *testing.T) {
		h := &modelHandler{registry: &fakeRegistry{err: errBoom}}

		_, _, err := h.getModel(context.Background(), nil, GetModelIn{ID: "m1"})
		wantError(t, err, ErrGetModelCode, `"m1"`, "boom")
	})
}
