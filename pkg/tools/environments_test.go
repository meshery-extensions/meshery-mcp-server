package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

func TestListEnvironmentsTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/environments" {
			t.Errorf("expected path /api/environments, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"totalCount": 2, "environments": [{"id": "e-1", "name": "staging"}, {"id": "e-2", "name": "prod"}]}`))
	}))
	defer ts.Close()

	client := meshery.NewClient(ts.URL)
	tool := NewListEnvironmentsTool(client)

	if tool.Name() != "list_environments" {
		t.Errorf("expected tool name list_environments, got %s", tool.Name())
	}
	if tool.Description() == "" {
		t.Errorf("expected non-empty description")
	}
	if tool.Schema() == nil {
		t.Errorf("expected non-nil schema")
	}

	params := map[string]interface{}{
		"orgId":    "org-test",
		"page":     0,
		"pageSize": 5,
	}

	res, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error executing ListEnvironmentsTool: %v", err)
	}

	envs, ok := res["environments"].([]interface{})
	if !ok || len(envs) != 2 {
		t.Errorf("expected 2 environments in result, got %v", res["environments"])
	}
}

func TestGetEnvironmentTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/environments/env-456" {
			t.Errorf("expected path /api/environments/env-456, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "env-456", "name": "production-cluster-1"}`))
	}))
	defer ts.Close()

	client := meshery.NewClient(ts.URL)
	tool := NewGetEnvironmentTool(client)

	if tool.Name() != "get_environment" {
		t.Errorf("expected tool name get_environment, got %s", tool.Name())
	}

	// Missing parameter check
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Errorf("expected error when passing nil params, got nil")
	}

	// Valid execution
	params := map[string]interface{}{
		"environment_id": "env-456",
	}

	res, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error executing GetEnvironmentTool: %v", err)
	}

	if res["name"] != "production-cluster-1" {
		t.Errorf("expected environment name production-cluster-1, got %v", res["name"])
	}
}

func TestCreateEnvironmentTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": "env-created-1", "name": "dev-env", "description": "development"}`))
	}))
	defer ts.Close()

	client := meshery.NewClient(ts.URL)
	tool := NewCreateEnvironmentTool(client)

	if tool.Name() != "create_environment" {
		t.Errorf("expected tool name create_environment, got %s", tool.Name())
	}

	// Missing name parameter check
	_, err := tool.Execute(context.Background(), map[string]interface{}{"description": "missing name"})
	if err == nil {
		t.Errorf("expected error when passing missing name, got nil")
	}

	// Valid execution
	params := map[string]interface{}{
		"name":        "dev-env",
		"description": "development",
		"orgId":       "org-dev",
	}

	res, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error executing CreateEnvironmentTool: %v", err)
	}

	if res["id"] != "env-created-1" {
		t.Errorf("expected created environment ID env-created-1, got %v", res["id"])
	}
}
