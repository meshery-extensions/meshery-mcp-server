package tools

import (
	"context"
	"fmt"
	"testing"
)

type mockMesheryClient struct {
	designs      []map[string]interface{}
	err          error
	lastPage     int
	lastPageSize int
	lastSearch   string
}

func (m *mockMesheryClient) ListDesigns(ctx context.Context, page, pageSize int, search string) ([]map[string]interface{}, int, error) {
	m.lastPage = page
	m.lastPageSize = pageSize
	m.lastSearch = search

	if m.err != nil {
		return nil, 0, m.err
	}
	return m.designs, len(m.designs), nil
}

func TestListDesignsTool_Execute_Success(t *testing.T) {
	mockClient := &mockMesheryClient{
		designs: []map[string]interface{}{
			{
				"id":     "design-001",
				"name":   "K8s-Deployment-Pattern",
				"secret": "bearer-token-secret-123",
			},
		},
	}

	tool := NewListDesignsTool(mockClient)
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"page":     2,
		"pageSize": 25,
		"search":   "K8s",
	})
	if err != nil {
		t.Fatalf("unexpected error executing list_designs tool: %v", err)
	}

	if mockClient.lastPage != 2 {
		t.Errorf("expected page 2 passed to client, got %d", mockClient.lastPage)
	}

	if mockClient.lastPageSize != 25 {
		t.Errorf("expected pageSize 25 passed to client, got %d", mockClient.lastPageSize)
	}

	if mockClient.lastSearch != "K8s" {
		t.Errorf("expected search K8s passed to client, got %s", mockClient.lastSearch)
	}

	if result["total_count"] != 1 {
		t.Errorf("expected total_count 1, got %v", result["total_count"])
	}

	designs := result["designs"].([]interface{})
	d0 := designs[0].(map[string]interface{})

	if d0["id"] != "design-001" {
		t.Errorf("expected id design-001, got %v", d0["id"])
	}

	if d0["secret"] != "[REDACTED_SECRET]" {
		t.Errorf("expected secret to be redacted, got %v", d0["secret"])
	}
}

func TestListDesignsTool_Execute_ParameterClamping(t *testing.T) {
	mockClient := &mockMesheryClient{
		designs: []map[string]interface{}{},
	}

	tool := NewListDesignsTool(mockClient)

	// Test negative page and oversized pageSize clamping (0-indexed Meshery Server pager)
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"page":     -5,
		"pageSize": 500,
	})
	if err != nil {
		t.Fatalf("unexpected error executing list_designs tool: %v", err)
	}

	if mockClient.lastPage != 0 {
		t.Errorf("expected clamped page 0 (0-indexed), got %d", mockClient.lastPage)
	}

	if mockClient.lastPageSize != 100 {
		t.Errorf("expected clamped pageSize 100, got %d", mockClient.lastPageSize)
	}
}

func TestListDesignsTool_Execute_ErrorPath(t *testing.T) {
	mockClient := &mockMesheryClient{
		err: fmt.Errorf("connection refused: token=raw-secret-123"),
	}

	tool := NewListDesignsTool(mockClient)
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error from list_designs execution, got nil")
	}

	if err.Error() == "list_designs failure: connection refused: token=raw-secret-123" {
		t.Errorf("error path unredacted token leak detected: %v", err)
	}
}
