package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

func TestListWorkspacesTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces" {
			t.Errorf("expected path /api/workspaces, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"totalCount": 2, "workspaces": [{"id": "ws-1", "name": "workspace-a"}, {"id": "ws-2", "name": "workspace-b"}]}`))
	}))
	defer ts.Close()

	client := meshery.NewClient(ts.URL)
	tool := NewListWorkspacesTool(client)

	if tool.Name() != "list_workspaces" {
		t.Errorf("expected tool name list_workspaces, got %s", tool.Name())
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
		"pageSize": 10,
	}

	res, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error executing ListWorkspacesTool: %v", err)
	}

	workspaces, ok := res["workspaces"].([]interface{})
	if !ok || len(workspaces) != 2 {
		t.Errorf("expected 2 workspaces in result, got %v", res["workspaces"])
	}
}

func TestSwitchWorkspaceTool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces/ws-active" {
			t.Errorf("expected path /api/workspaces/ws-active, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "ws-active", "name": "production-workspace"}`))
	}))
	defer ts.Close()

	client := meshery.NewClient(ts.URL)
	tool := NewSwitchWorkspaceTool(client)

	if tool.Name() != "switch_workspace" {
		t.Errorf("expected tool name switch_workspace, got %s", tool.Name())
	}

	// Missing parameter check
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Errorf("expected error when passing nil params, got nil")
	}

	// Valid execution
	params := map[string]interface{}{
		"workspace_id": "ws-active",
	}

	res, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error executing SwitchWorkspaceTool: %v", err)
	}

	if res["status"] != "active" {
		t.Errorf("expected status active, got %v", res["status"])
	}
	if res["workspace_id"] != "ws-active" {
		t.Errorf("expected workspace_id ws-active, got %v", res["workspace_id"])
	}

	wsMap, ok := res["workspace"].(map[string]interface{})
	if !ok || wsMap["name"] != "production-workspace" {
		t.Errorf("expected workspace name production-workspace, got %v", res["workspace"])
	}
}
