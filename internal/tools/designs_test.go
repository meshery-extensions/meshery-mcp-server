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

// scenario describes one canned response from the mock Cloud server.
type scenario struct {
	status     int
	body       string
	retryAfter string
}

// mockCloud answers a fixed queue of scenarios, one per request, so tests
// can count exactly how many requests Cloud saw (no tight retry loops).
type mockCloud struct {
	mu     sync.Mutex
	steps  []scenario
	idx    int
	counts map[string]int
	auths  map[string][]string
}

func (m *mockCloud) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := r.URL.Path
	m.counts[path]++
	m.auths[path] = append(m.auths[path], r.Header.Get(meshery.HeaderToken))

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

func newMockCloud(t *testing.T, steps ...scenario) (*httptest.Server, *mockCloud) {
	t.Helper()
	m := &mockCloud{steps: steps, counts: map[string]int{}, auths: map[string][]string{}}
	return httptest.NewServer(m), m
}

func TestListDesigns(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"patterns":[{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled"},{"id":"d2","name":"istio","version":"v1beta2","displayName":"Istio","status":"enabled"}],"totalCount":2}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewDesignTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "list_designs" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("list_designs tool not found")
	}

	out, err := tool.Handler(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("list_designs handler failed: %v", err)
	}

	result, ok := out.(ListDesignsResult)
	if !ok {
		t.Fatalf("expected ListDesignsResult, got %T", out)
	}
	if len(result.Designs) != 2 {
		t.Errorf("expected 2 designs, got %d", len(result.Designs))
	}
	if result.Designs[0].Name != "bookinfo" {
		t.Errorf("expected first design bookinfo, got %s", result.Designs[0].Name)
	}
	if result.Designs[1].Name != "istio" {
		t.Errorf("expected second design istio, got %s", result.Designs[1].Name)
	}
	if mock.count("/api/content/patterns") != 1 {
		t.Errorf("expected 1 call to /api/content/patterns, got %d", mock.count("/api/content/patterns"))
	}
}

func TestGetDesign(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled","components":[{"name":"productpage","type":"Deployment"}],"relationships":[{"kind":"hierarchical","type":"parent","subType":"inventory"}],"patternFile":"apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: productpage\nspec:\n  selector:\n    app: productpage\n  ports:\n  - port: 8080\n    targetPort: 8080"}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewDesignTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "get_design" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("get_design tool not found")
	}

	out, err := tool.Handler(context.Background(), map[string]any{"design_id": "d1"})
	if err != nil {
		t.Fatalf("get_design handler failed: %v", err)
	}

	result, ok := out.(GetDesignResult)
	if !ok {
		t.Fatalf("expected GetDesignResult, got %T", out)
	}
	if result.Design.Name != "bookinfo" {
		t.Errorf("expected design name bookinfo, got %s", result.Design.Name)
	}
	if mock.count("/api/content/patterns/d1") != 1 {
		t.Errorf("expected 1 call to /api/content/patterns/d1, got %d", mock.count("/api/content/patterns/d1"))
	}
}

func TestGetDesignNotFound(t *testing.T) {
	srv, _ := newMockCloud(t, scenario{
		status: 404,
		body:   `{"error":"design not found"}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewDesignTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "get_design" {
			tool = &t
			break
		}
	}

	_, err := tool.Handler(context.Background(), map[string]any{"design_id": "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent design")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestCreateDesign(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"id":"d1","name":"my-design","version":"v1beta2","displayName":"My Design","status":"enabled"}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewDesignTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "create_design" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("create_design tool not found")
	}

	// Without confirm, should return pending confirmation
	_, err := tool.Handler(context.Background(), map[string]any{
		"name":        "my-design",
		"design_yaml": "apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: my-service",
		"confirm":     false,
	})
	if err == nil {
		t.Fatal("expected error when confirm is false")
	}
	if !strings.Contains(err.Error(), "confirm: true") {
		t.Errorf("expected confirm message, got %v", err)
	}
	if mock.count("/api/content/patterns") != 0 {
		t.Errorf("expected 0 calls to /api/content/patterns without confirm, got %d", mock.count("/api/content/patterns"))
	}

	// With confirm, should make the API call
	out, err := tool.Handler(context.Background(), map[string]any{
		"name":        "my-design",
		"design_yaml": "apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: my-service",
		"confirm":     true,
	})
	if err != nil {
		t.Fatalf("create_design handler failed: %v", err)
	}

	result, ok := out.(CreateDesignResult)
	if !ok {
		t.Fatalf("expected CreateDesignResult, got %T", out)
	}
	if result.ID != "d1" {
		t.Errorf("expected design ID d1, got %s", result.ID)
	}
	if result.Name != "my-design" {
		t.Errorf("expected design name my-design, got %s", result.Name)
	}
	if mock.count("/api/content/patterns") != 1 {
		t.Errorf("expected 1 call to /api/content/patterns, got %d", mock.count("/api/content/patterns"))
	}
}

func TestDeployDesign(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{
			status: 200,
			body:   `{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled","patternFile":"apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: productpage"}`,
		},
		scenario{
			status: 200,
			body:   `{"id":"d1","name":"bookinfo","version":"v1beta2"}`,
		},
	)
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewDesignTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "deploy_design" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("deploy_design tool not found")
	}

	// Without confirm, should return pending confirmation
	_, err := tool.Handler(context.Background(), map[string]any{
		"design_id":  "d1",
		"cluster_id": "cluster-1",
		"confirm":    false,
	})
	if err == nil {
		t.Fatal("expected error when confirm is false")
	}
	if !strings.Contains(err.Error(), "confirm: true") {
		t.Errorf("expected confirm message, got %v", err)
	}
	if mock.count("/api/content/patterns/d1") != 0 {
		t.Errorf("expected 0 calls to /api/content/patterns/d1 without confirm, got %d", mock.count("/api/content/patterns/d1"))
	}
	if mock.count("/api/pattern/deploy") != 0 {
		t.Errorf("expected 0 calls to /api/pattern/deploy without confirm, got %d", mock.count("/api/pattern/deploy"))
	}

	// With confirm, should make both API calls
	out, err := tool.Handler(context.Background(), map[string]any{
		"design_id":  "d1",
		"cluster_id": "cluster-1",
		"confirm":    true,
	})
	if err != nil {
		t.Fatalf("deploy_design handler failed: %v", err)
	}

	result, ok := out.(DeployDesignResult)
	if !ok {
		t.Fatalf("expected DeployDesignResult, got %T", out)
	}
	if result.DesignID != "d1" {
		t.Errorf("expected design ID d1, got %s", result.DesignID)
	}
	if result.ClusterID != "cluster-1" {
		t.Errorf("expected cluster ID cluster-1, got %s", result.ClusterID)
	}
	if result.Status != "deployed" {
		t.Errorf("expected status deployed, got %s", result.Status)
	}
	if mock.count("/api/content/patterns/d1") != 1 {
		t.Errorf("expected 1 call to /api/content/patterns/d1, got %d", mock.count("/api/content/patterns/d1"))
	}
	if mock.count("/api/pattern/deploy") != 1 {
		t.Errorf("expected 1 call to /api/pattern/deploy, got %d", mock.count("/api/pattern/deploy"))
	}
}

func TestUndeployDesign(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{
			status: 200,
			body:   `{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled","patternFile":"apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: productpage"}`,
		},
		scenario{
			status: 200,
			body:   ``,
		},
	)
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewDesignTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "undeploy_design" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("undeploy_design tool not found")
	}

	// Without confirm, should return pending confirmation
	_, err := tool.Handler(context.Background(), map[string]any{
		"design_id":  "d1",
		"cluster_id": "cluster-1",
		"confirm":    false,
	})
	if err == nil {
		t.Fatal("expected error when confirm is false")
	}
	if !strings.Contains(err.Error(), "confirm: true") {
		t.Errorf("expected confirm message, got %v", err)
	}

	// With confirm, should make both API calls
	out, err := tool.Handler(context.Background(), map[string]any{
		"design_id":  "d1",
		"cluster_id": "cluster-1",
		"confirm":    true,
	})
	if err != nil {
		t.Fatalf("undeploy_design handler failed: %v", err)
	}

	result, ok := out.(UndeployDesignResult)
	if !ok {
		t.Fatalf("expected UndeployDesignResult, got %T", out)
	}
	if result.DesignID != "d1" {
		t.Errorf("expected design ID d1, got %s", result.DesignID)
	}
	if result.ClusterID != "cluster-1" {
		t.Errorf("expected cluster ID cluster-1, got %s", result.ClusterID)
	}
	if result.Status != "undeployed" {
		t.Errorf("expected status undeployed, got %s", result.Status)
	}
	if mock.count("/api/content/patterns/d1") != 1 {
		t.Errorf("expected 1 call to /api/content/patterns/d1, got %d", mock.count("/api/content/patterns/d1"))
	}
	if mock.count("/api/pattern/deploy") != 1 {
		t.Errorf("expected 1 call to /api/pattern/deploy, got %d", mock.count("/api/pattern/deploy"))
	}
}

func TestDeleteDesign(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   ``,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token")
	tools := NewDesignTools(c)

	var tool *Tool
	for _, t := range tools {
		if t.Name == "delete_design" {
			tool = &t
			break
		}
	}
	if tool == nil {
		t.Fatal("delete_design tool not found")
	}

	// Without confirm, should return pending confirmation
	_, err := tool.Handler(context.Background(), map[string]any{"design_id": "d1", "confirm": false})
	if err == nil {
		t.Fatal("expected error when confirm is false")
	}
	if !strings.Contains(err.Error(), "confirm: true") {
		t.Errorf("expected confirm message, got %v", err)
	}
	if mock.count("/api/content/patterns/d1") != 0 {
		t.Errorf("expected 0 calls to /api/content/patterns/d1 without confirm, got %d", mock.count("/api/content/patterns/d1"))
	}

	// With confirm, should make the API call
	out, err := tool.Handler(context.Background(), map[string]any{"design_id": "d1", "confirm": true})
	if err != nil {
		t.Fatalf("delete_design handler failed: %v", err)
	}

	result, ok := out.(DeleteDesignResult)
	if !ok {
		t.Fatalf("expected DeleteDesignResult, got %T", out)
	}
	if result.ID != "d1" {
		t.Errorf("expected design ID d1, got %s", result.ID)
	}
	if mock.count("/api/content/patterns/d1") != 1 {
		t.Errorf("expected 1 call to /api/content/patterns/d1, got %d", mock.count("/api/content/patterns/d1"))
	}
}
