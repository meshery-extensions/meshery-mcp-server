package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

// mockCloud answers a fixed queue of scenarios, one per request, so tests
// can count exactly how many requests Cloud saw (no tight retry loops).
type mockCloud struct {
	mu     sync.Mutex
	steps  []scenario
	idx    int
	counts map[string]int
	auths  map[string][]string
}

type scenario struct {
	status     int
	body       string
	retryAfter string
}

func (m *mockCloud) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := r.URL.Path
	m.counts[path]++
	m.auths[path] = append(m.auths[path], r.Header.Get("Cookie"))

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

func (m *mockCloud) lastAuth(path string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	auths := m.auths[path]
	if len(auths) == 0 {
		return ""
	}
	return auths[len(auths)-1]
}

func newMockCloud(t *testing.T, steps ...scenario) (*httptest.Server, *mockCloud) {
	t.Helper()
	m := &mockCloud{steps: steps, counts: map[string]int{}, auths: map[string][]string{}}
	return httptest.NewServer(m), m
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// deployOK is the pattern handler's per-component response for a successful
// single-component deploy/undeploy.
const deployOK = `{"c1":[{"Summary":[{"Kind":"Service","Model":"Kubernetes","CompName":"productpage","Success":true,"Message":"Deployed design/productpage"}],"SystemName":"k8s","Location":"https://cluster"}]}`

func TestListDesigns(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"patterns":[{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled"},{"id":"d2","name":"istio","version":"v1beta2","displayName":"Istio","status":"enabled"}],"totalCount":2}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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

	out, err := tool.Handler(context.Background(), map[string]any{
		"page":      0,
		"page_size": 10,
		"search":    "",
	})
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
	if result.Total != 2 {
		t.Errorf("expected totalCount 2, got %d", result.Total)
	}
	if mock.count("/api/pattern") != 1 {
		t.Errorf("expected 1 call to /api/pattern, got %d", mock.count("/api/pattern"))
	}
	if got := mock.lastAuth("/api/pattern"); got != "token=test-token" {
		t.Errorf("expected cookie token=test-token, got %q", got)
	}
}

func TestListDesignsWithSearch(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"patterns":[{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled"}],"totalCount":1}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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

	out, err := tool.Handler(context.Background(), map[string]any{
		"search": "bookinfo",
	})
	if err != nil {
		t.Fatalf("list_designs handler failed: %v", err)
	}

	result, ok := out.(ListDesignsResult)
	if !ok {
		t.Fatalf("expected ListDesignsResult, got %T", out)
	}
	if len(result.Designs) != 1 {
		t.Errorf("expected 1 design, got %d", len(result.Designs))
	}
	if mock.count("/api/pattern") != 1 {
		t.Errorf("expected 1 call to /api/pattern, got %d", mock.count("/api/pattern"))
	}
	if got := mock.lastAuth("/api/pattern"); got != "token=test-token" {
		t.Errorf("expected cookie token=test-token, got %q", got)
	}
}

func TestGetDesign(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled","patternFile":"apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: productpage\nspec:\n  selector:\n    app: productpage\n  ports:\n  - port: 8080\n    targetPort: 8080"}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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
	if result.Design.PatternFile == "" {
		t.Errorf("expected patternFile to be populated")
	}
	if mock.count("/api/pattern/d1") != 1 {
		t.Errorf("expected 1 call to /api/pattern/d1, got %d", mock.count("/api/pattern/d1"))
	}
}

func TestGetDesignNotFound(t *testing.T) {
	srv, _ := newMockCloud(t, scenario{
		status: 404,
		body:   `{"error":"design not found"}`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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
		body:   `[{"id":"d1","name":"my-design","version":"v1beta2","displayName":"My Design","status":"enabled"}]`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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
	if mock.count("/api/pattern") != 1 {
		t.Errorf("expected 1 call to /api/pattern, got %d", mock.count("/api/pattern"))
	}
}

func TestCreateDesignMissingConfirm(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{
		status: 200,
		body:   `[{"id":"d1","name":"my-design","version":"v1beta2","displayName":"My Design","status":"enabled"}]`,
	})
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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

	_, err := tool.Handler(context.Background(), map[string]any{
		"name":        "my-design",
		"design_yaml": "apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: my-service",
	})
	if err == nil {
		t.Fatal("expected error when confirm is false")
	}
	if !strings.Contains(err.Error(), "confirm: true") {
		t.Errorf("expected confirm message, got %v", err)
	}
	if mock.count("/api/pattern") != 0 {
		t.Errorf("expected 0 calls to /api/pattern without confirm, got %d", mock.count("/api/pattern"))
	}
}

func TestDeployDesign(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{
			status: 200,
			body:   `{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled","patternFile":"apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: productpage\nspec:\n  selector:\n    app: productpage\n  ports:\n  - port: 8080\n    targetPort: 8080"}`,
		},
		scenario{
			status: 200,
			body:   deployOK,
		},
	)
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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
	if mock.count("/api/pattern/d1") != 0 {
		t.Errorf("expected 0 calls to /api/pattern/d1 without confirm, got %d", mock.count("/api/pattern/d1"))
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
	if mock.count("/api/pattern/d1") != 1 {
		t.Errorf("expected 1 call to /api/pattern/d1, got %d", mock.count("/api/pattern/d1"))
	}
	if mock.count("/api/pattern/deploy") != 1 {
		t.Errorf("expected 1 call to /api/pattern/deploy, got %d", mock.count("/api/pattern/deploy"))
	}
}

func TestUndeployDesign(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{
			status: 200,
			body:   `{"id":"d1","name":"bookinfo","version":"v1beta2","displayName":"Bookinfo","status":"enabled","patternFile":"apiVersion: serving.knative.dev/v1\nkind: Service\nmetadata:\n  name: productpage\nspec:\n  selector:\n    app: productpage\n  ports:\n  - port: 8080\n    targetPort: 8080"}`,
		},
		scenario{
			status: 200,
			body:   deployOK,
		},
	)
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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
	if mock.count("/api/pattern/d1") != 0 {
		t.Errorf("expected 0 calls to /api/pattern/d1 without confirm, got %d", mock.count("/api/pattern/d1"))
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
	if mock.count("/api/pattern/d1") != 1 {
		t.Errorf("expected 1 call to /api/pattern/d1, got %d", mock.count("/api/pattern/d1"))
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

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
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
	if mock.count("/api/pattern/d1") != 0 {
		t.Errorf("expected 0 calls to /api/pattern/d1 without confirm, got %d", mock.count("/api/pattern/d1"))
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
	if mock.count("/api/pattern/d1") != 1 {
		t.Errorf("expected 1 call to /api/pattern/d1, got %d", mock.count("/api/pattern/d1"))
	}
}

func TestDeployDesignMixedResult(t *testing.T) {
	mixed := `{"c1":[{"Summary":[
		{"Kind":"Service","Model":"Kubernetes","CompName":"productpage","Success":true,"Message":"Deployed design/productpage"},
		{"Kind":"Deployment","Model":"Kubernetes","CompName":"details","Success":false,"Message":"Error deploying design/details"},
		{"Kind":"ConfigMap","Model":"Kubernetes","CompName":"prereq","Success":false,"IsPrerequisite":true,"Message":"failed install"}
	],"SystemName":"k8s","Location":"https://cluster"}]}`
	srv, mock := newMockCloud(t,
		scenario{
			status: 200,
			body:   `{"id":"d1","name":"bookinfo","patternFile":"apiVersion: v1\nkind: Service\nmetadata:\n  name: productpage"}`,
		},
		scenario{status: 200, body: mixed},
	)
	defer srv.Close()

	c := meshery.NewClient(srv.URL, "test-token", meshery.WithAuthMode(meshery.AuthCookie))
	for _, tool := range NewDesignTools(c) {
		if tool.Name != "deploy_design" {
			continue
		}
		out, err := tool.Handler(context.Background(), map[string]any{
			"design_id": "d1", "cluster_id": "cluster-1", "confirm": true,
		})
		if err != nil {
			t.Fatalf("deploy_design handler failed: %v", err)
		}
		result, ok := out.(DeployDesignResult)
		if !ok {
			t.Fatalf("expected DeployDesignResult, got %T", out)
		}
		if result.Status != "failed" {
			t.Errorf("status = %q, want failed", result.Status)
		}
		if len(result.Failed) != 1 || result.Failed[0].Component != "details" {
			t.Errorf("failed = %+v, want only details", result.Failed)
		}
		if mock.count("/api/pattern/deploy") != 1 {
			t.Errorf("deploy calls = %d, want 1", mock.count("/api/pattern/deploy"))
		}
		return
	}
	t.Fatal("deploy_design tool not found")
}

func TestCreateDesignBody(t *testing.T) {
	body, err := createDesignBody("my-design", "name: doc\nschemaVersion: v1beta3\ncomponents: []\n")
	if err != nil {
		t.Fatalf("createDesignBody failed: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if got["name"] != "my-design" {
		t.Errorf("name = %v, want my-design", got["name"])
	}
	designFile, ok := got["designFile"].(map[string]any)
	if !ok {
		t.Fatalf("designFile must be an object for Meshery Server, got %T", got["designFile"])
	}
	if designFile["name"] != "doc" {
		t.Errorf("designFile.name = %v, want doc", designFile["name"])
	}
}

func TestCreateDesignBodyRejectsNonObject(t *testing.T) {
	if _, err := createDesignBody("x", "just a string"); err == nil {
		t.Fatal("expected a design_yaml object error")
	}
}
