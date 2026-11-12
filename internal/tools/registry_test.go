package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

// mockCloud answers a fixed queue of scenarios, one per request.
type mockCloud struct {
	mu     sync.Mutex
	steps  []scenario
	idx    int
	counts map[string]int
}

func (m *mockCloud) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := r.URL.Path
	m.counts[path]++

	step := m.steps[min(m.idx, len(m.steps)-1)]
	m.idx++

	if step.retryAfter != "" {
		w.Header().Set("Retry-After", step.retryAfter)
	}
	w.WriteHeader(step.status)
	_, _ = w.Write([]byte(step.body))
}

func (m *mockCloud) count(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counts[path]
}

type scenario struct {
	status     int
	body       string
	retryAfter string
}

func newMockCloud(t *testing.T, steps ...scenario) (*httptest.Server, *mockCloud) {
	t.Helper()
	m := &mockCloud{steps: steps, counts: map[string]int{}}
	return httptest.NewServer(m), m
}

func TestListModels(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"models":[{"id":"m1","name":"kubernetes","version":"v1beta2","displayName":"Kubernetes","status":"enabled"},{"id":"m2","name":"istio","version":"v1beta2","displayName":"Istio","status":"enabled"}],"totalCount":2}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "list_models" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("list_models tool not found")
	}

	out, err := tool.Handler(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("list_models handler failed: %v", err)
	}

	result, ok := out.(ListModelsResult)
	if !ok {
		t.Fatalf("expected ListModelsResult, got %T", out)
	}
	if len(result.Claims) != 2 {
		t.Errorf("expected 2 models, got %d", len(result.Claims))
	}
	if result.Claims[0].Name != "kubernetes" {
		t.Errorf("expected first model kubernetes, got %s", result.Claims[0].Name)
	}
	if result.Claims[1].Name != "istio" {
		t.Errorf("expected second model istio, got %s", result.Claims[1].Name)
	}
	if mock.count("/api/registry/models") != 1 {
		t.Errorf("expected 1 call to /api/registry/models, got %d", mock.count("/api/registry/models"))
	}
}

func TestGetModel(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"models":[{"id":"m1","name":"kubernetes","version":"v1beta2","displayName":"Kubernetes","status":"enabled","componentsCount":3,"relationshipsCount":2}],"totalCount":1}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "get_model" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("get_model tool not found")
	}

	out, err := tool.Handler(context.Background(), map[string]any{"model_id": "kubernetes"})
	if err != nil {
		t.Fatalf("get_model handler failed: %v", err)
	}

	result, ok := out.(GetModelResult)
	if !ok {
		t.Fatalf("expected GetModelResult, got %T", out)
	}
	if result.Model.Name != "kubernetes" {
		t.Errorf("expected model name kubernetes, got %s", result.Model.Name)
	}
	if mock.count("/api/registry/models/kubernetes") != 1 {
		t.Errorf("expected 1 call to /api/registry/models/kubernetes, got %d", mock.count("/api/registry/models/kubernetes"))
	}
}

func TestGetModelNotFound(t *testing.T) {
	srv, _ := newMockCloud(t, scenario{
		status: 200,
		body:   `{"models":[],"totalCount":0}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "get_model" {
			tool = &t
			break
		}
	}

	_, err := tool.Handler(context.Background(), map[string]any{"model_id": "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestSearchComponents(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"components":[{"id":"c1","displayName":"Deployment","version":"v1beta3","model":{"name":"kubernetes"}}],"totalCount":1}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "search_components" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("search_components tool not found")
	}

	out, err := tool.Handler(context.Background(), map[string]any{"search": "Deployment", "model": "kubernetes"})
	if err != nil {
		t.Fatalf("search_components handler failed: %v", err)
	}

	result, ok := out.(SearchContentItemsResult)
	if !ok {
		t.Fatalf("expected SearchContentItemsResult, got %T", out)
	}
	if len(result.ContentItems) != 1 {
		t.Errorf("expected 1 content item, got %d", len(result.ContentItems))
	}
	if result.ContentItems[0].Name != "Deployment" {
		t.Errorf("expected Deployment, got %s", result.ContentItems[0].Name)
	}
	// Should have called /api/registry/models/kubernetes/components
	if mock.count("/api/registry/models/kubernetes/components") != 1 {
		t.Errorf("expected 1 call to /api/registry/models/kubernetes/components, got %d", mock.count("/api/registry/models/kubernetes/components"))
	}
}

func TestGetComponent(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"components":[{"id":"c1","displayName":"Deployment","version":"v1beta3","format":"JSON","status":"enabled","model":{"name":"kubernetes"},"component":{"type":"object","properties":{"replicas":{"type":"integer"}}}}],"totalCount":1}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "get_component" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("get_component tool not found")
	}

	out, err := tool.Handler(context.Background(), map[string]any{"component_id": "Deployment"})
	if err != nil {
		t.Fatalf("get_component handler failed: %v", err)
	}

	result, ok := out.(GetContentItemResult)
	if !ok {
		t.Fatalf("expected GetContentItemResult, got %T", out)
	}
	if result.Component.Name != "Deployment" {
		t.Errorf("expected Deployment, got %s", result.Component.Name)
	}
	if mock.count("/api/registry/components/Deployment") != 1 {
		t.Errorf("expected 1 call to /api/registry/components/Deployment, got %d", mock.count("/api/registry/components/Deployment"))
	}
}

func TestListRelationships(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"relationships":[{"id":"r1","kind":"hierarchical","type":"parent","subType":"inventory"},{"id":"r2","kind":"edge","type":"binding","subType":"network"}],"totalCount":2}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "list_relationships" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("list_relationships tool not found")
	}

	out, err := tool.Handler(context.Background(), map[string]any{"model": "kubernetes"})
	if err != nil {
		t.Fatalf("list_relationships handler failed: %v", err)
	}

	result, ok := out.(ListLedgersResult)
	if !ok {
		t.Fatalf("expected ListLedgersResult, got %T", out)
	}
	if len(result.Relationships) != 2 {
		t.Errorf("expected 2 relationships, got %d", len(result.Relationships))
	}
	if result.Relationships[0].Kind != "hierarchical" {
		t.Errorf("expected hierarchical, got %s", result.Relationships[0].Kind)
	}
	if mock.count("/api/registry/models/kubernetes/relationships") != 1 {
		t.Errorf("expected 1 call to /api/registry/models/kubernetes/relationships, got %d", mock.count("/api/registry/models/kubernetes/relationships"))
	}
}

func TestListModelsEmpty(t *testing.T) {
	srv, _ := newMockCloud(t, scenario{
		status: 200,
		body:   `{"models":[],"totalCount":0}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "list_models" {
			tool = &t
			break
		}
	}

	out, err := tool.Handler(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("list_models handler failed: %v", err)
	}

	result, ok := out.(ListModelsResult)
	if !ok {
		t.Fatalf("expected ListModelsResult, got %T", out)
	}
	if len(result.Claims) != 0 {
		t.Errorf("expected 0 models, got %d", len(result.Claims))
	}
}
