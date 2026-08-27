// Copyright Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func mustNewMesheryHTTPClient(t *testing.T, baseURL, token, provider string, httpClient *http.Client) *MesheryHTTPClient {
	t.Helper()
	c, err := NewMesheryHTTPClient(baseURL, token, provider, httpClient)
	if err != nil {
		t.Fatalf("mustNewMesheryHTTPClient: %v", err)
	}
	return c
}

func TestListClusters_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes/contexts" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"totalCount": 2,
				"contexts": [
					{
						"id": "ctx-1",
						"name": "minikube-dev",
						"server": "https://192.168.49.2:8443",
						"kubernetesServerId": "ksid-101",
						"connectionId": "conn-001",
						"deploymentType": "out_cluster",
						"version": "v1.30.0",
						"reachable": true,
						"createdAt": "2026-08-01T10:00:00Z"
					},
					{
						"id": "ctx-2",
						"name": "prod-gke",
						"server": "https://35.200.1.1:443",
						"kubernetesServerId": "ksid-102",
						"connectionId": "conn-002",
						"deploymentType": "in_cluster",
						"version": "v1.29.2",
						"reachable": false,
						"createdAt": "2026-08-05T12:00:00Z"
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "test-token", "", ts.Client())
	handler := listClustersHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "list_clusters",
			Arguments: map[string]interface{}{
				"page":      float64(0),
				"page_size": float64(10),
			},
		},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected successful result, got error: %+v", result.Content)
	}

	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &parsed); err != nil {
		t.Fatalf("failed to parse result json: %v", err)
	}

	if parsed["total_count"].(float64) != 2 {
		t.Errorf("expected total_count 2, got %v", parsed["total_count"])
	}

	clusters, ok := parsed["clusters"].([]interface{})
	if !ok || len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %v", parsed["clusters"])
	}

	firstCluster := clusters[0].(map[string]interface{})
	if firstCluster["id"] != "ctx-1" || firstCluster["name"] != "minikube-dev" {
		t.Errorf("unexpected first cluster: %+v", firstCluster)
	}
	if firstCluster["kubernetes_server_id"] != "ksid-101" {
		t.Errorf("expected ksid-101, got %v", firstCluster["kubernetes_server_id"])
	}
}

func TestListClusters_AuthRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/provider?error=unauthorized", http.StatusFound)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "", "", ts.Client())
	handler := listClustersHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "list_clusters",
		},
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned go error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected error result on auth redirect, got success: %+v", result.Content)
	}

	textContent, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	if !strings.Contains(textContent.Text, "authentication required") {
		t.Errorf("expected auth error message, got %q", textContent.Text)
	}
}

func TestGetCluster_ByID_And_ConnectionID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes/contexts" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"totalCount": 1,
				"contexts": [
					{
						"id": "ctx-1",
						"name": "staging-cluster",
						"server": "https://10.0.0.1:6443",
						"kubernetesServerId": "ksid-staging",
						"connectionId": "conn-staging-123",
						"deploymentType": "out_cluster",
						"version": "v1.30.2",
						"reachable": true
					}
				]
			}`))
			return
		}
		if r.URL.Path == "/api/system/meshsync/resources" {
			// Verify clusterIds was passed as a valid JSON array string containing the exact cluster ID
			clusterIDs := r.URL.Query().Get("clusterIds")
			var ids []string
			if err := json.Unmarshal([]byte(clusterIDs), &ids); err != nil || len(ids) != 1 || ids[0] != "ksid-staging" {
				http.Error(w, "missing or invalid clusterIds", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"page": 1,
				"page_size": 100,
				"total_count": 2,
				"resources": [
					{"id": "node-1", "kind": "Node"},
					{"id": "node-2", "kind": "Node"}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "token", "", ts.Client())
	handler := getClusterHandler(client)

	// Test 1: Resolve by Context ID
	reqByID := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_cluster",
			Arguments: map[string]interface{}{
				"cluster_id": "ctx-1",
			},
		},
	}
	result, err := handler(context.Background(), reqByID)
	if err != nil || result.IsError {
		t.Fatalf("failed to get cluster by ID: %v, result: %+v", err, result)
	}

	text, _ := mcp.AsTextContent(result.Content[0])
	var details map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &details); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if details["name"] != "staging-cluster" {
		t.Errorf("expected name 'staging-cluster', got %v", details["name"])
	}
	if details["node_count"].(float64) != 2 {
		t.Errorf("expected 2 nodes, got %v", details["node_count"])
	}
	if details["health_status"] != "Healthy" {
		t.Errorf("expected 'Healthy', got %v", details["health_status"])
	}

	// Test 2: Resolve by Connection ID
	reqByConn := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_cluster",
			Arguments: map[string]interface{}{
				"cluster_id": "conn-staging-123",
			},
		},
	}
	resultConn, err := handler(context.Background(), reqByConn)
	if err != nil || resultConn.IsError {
		t.Fatalf("failed to get cluster by ConnectionID: %v", err)
	}
}

func TestGetCluster_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalCount": 0, "contexts": []}`))
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "token", "", ts.Client())
	handler := getClusterHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_cluster",
			Arguments: map[string]interface{}{
				"cluster_id": "non-existent-cluster",
			},
		},
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for missing cluster, got success")
	}

	text, _ := mcp.AsTextContent(result.Content[0])
	if !strings.Contains(text.Text, "not found in Meshery registered contexts") {
		t.Errorf("expected not found message, got: %s", text.Text)
	}
}

func TestGetClusterNodes_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes/contexts" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"totalCount": 1,
				"contexts": [
					{
						"id": "ctx-k8s",
						"name": "k8s-cluster",
						"kubernetesServerId": "ksid-prod",
						"reachable": true
					}
				]
			}`))
			return
		}
		if r.URL.Path == "/api/system/meshsync/resources" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"page": 0,
				"pageSize": 100,
				"totalCount": 1,
				"resources": [
					{
						"id": "node-master-1",
						"kind": "Node",
						"apiVersion": "v1",
						"metadata": {
							"name": "control-plane-01",
							"labels": {
								"node-role.kubernetes.io/control-plane": "",
								"kubernetes.io/arch": "amd64",
								"kubernetes.io/os": "linux"
							}
						},
						"status": {
							"conditions": [
								{"type": "Ready", "status": "True"}
							],
							"nodeInfo": {
								"kubeletVersion": "v1.30.0",
								"osImage": "Ubuntu 22.04"
							},
							"capacity": {
								"cpu": "8",
								"memory": "16384Mi"
							},
							"addresses": [
								{"type": "InternalIP", "address": "10.0.1.10"}
							]
						}
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "token", "", ts.Client())
	handler := getClusterNodesHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_cluster_nodes",
			Arguments: map[string]interface{}{
				"cluster_id": "ctx-k8s",
			},
		},
	}
	result, err := handler(context.Background(), req)
	if err != nil || result.IsError {
		t.Fatalf("unexpected error fetching nodes: %v, result: %+v", err, result)
	}

	text, _ := mcp.AsTextContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &resp); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	nodes := resp["nodes"].([]interface{})
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	node := nodes[0].(map[string]interface{})
	if node["name"] != "control-plane-01" {
		t.Errorf("expected node name 'control-plane-01', got %v", node["name"])
	}
	if node["status"] != "Ready" {
		t.Errorf("expected status 'Ready', got %v", node["status"])
	}
	if node["kubelet_version"] != "v1.30.0" {
		t.Errorf("expected v1.30.0, got %v", node["kubelet_version"])
	}
	roles := node["roles"].([]interface{})
	if len(roles) != 1 || roles[0] != "control-plane" {
		t.Errorf("expected role 'control-plane', got %v", roles)
	}
}

func TestGetClusterResources_Pods(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes/contexts" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"totalCount": 1,
				"contexts": [
					{
						"id": "ctx-dev",
						"name": "dev-cluster",
						"kubernetesServerId": "ksid-dev"
					}
				]
			}`))
			return
		}
		if r.URL.Path == "/api/system/meshsync/resources" {
			if r.URL.Query().Get("kind") != "Pod" {
				http.Error(w, "expected kind=Pod", http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("namespace") != "kube-system" {
				http.Error(w, "expected namespace=kube-system", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"page": 0,
				"pageSize": 25,
				"totalCount": 2,
				"resources": [
					{
						"id": "res-1",
						"kind": "Pod",
						"apiVersion": "v1",
						"metadata": {
							"name": "coredns-78f",
							"namespace": "kube-system",
							"creationTimestamp": "2026-08-10T12:00:00Z"
						},
						"status": {
							"phase": "Running"
						}
					},
					{
						"id": "res-2",
						"kind": "Pod",
						"apiVersion": "v1",
						"metadata": {
							"name": "kube-proxy-abc",
							"namespace": "kube-system",
							"creationTimestamp": "2026-08-10T12:00:00Z"
						},
						"status": {
							"phase": "Running"
						}
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "token", "", ts.Client())
	handler := getClusterResourcesHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_cluster_resources",
			Arguments: map[string]interface{}{
				"cluster_id":    "ctx-dev",
				"resource_kind": "Pod",
				"namespace":     "kube-system",
			},
		},
	}
	result, err := handler(context.Background(), req)
	if err != nil || result.IsError {
		t.Fatalf("failed to fetch resources: %v, result: %+v", err, result)
	}

	text, _ := mcp.AsTextContent(result.Content[0])
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &resp); err != nil {
		t.Fatalf("json decode failed: %v", err)
	}

	if resp["total_count"].(float64) != 2 {
		t.Errorf("expected total_count 2, got %v", resp["total_count"])
	}
	resources := resp["resources"].([]interface{})
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	r0 := resources[0].(map[string]interface{})
	if r0["name"] != "coredns-78f" || r0["status"] != "Running" {
		t.Errorf("unexpected resource summary: %+v", r0)
	}
}

func TestGetClusterResources_MissingClusterID(t *testing.T) {
	client := mustNewMesheryHTTPClient(t, "http://localhost:9081", "token", "", nil)
	handler := getClusterResourcesHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get_cluster_resources",
			Arguments: map[string]interface{}{},
		},
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error when cluster_id is missing")
	}
	text, _ := mcp.AsTextContent(result.Content[0])
	if !strings.Contains(text.Text, "cluster_id parameter is required") {
		t.Errorf("expected validation message, got %s", text.Text)
	}
}

func TestKubernetesTools_InProcessMCPServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/system/kubernetes/contexts" {
			_, _ = w.Write([]byte(`{"totalCount": 1, "contexts": [{"id": "c1", "name": "test-cluster", "version": "v1.30.0", "reachable": true}]}`))
			return
		}
		if r.URL.Path == "/api/system/meshsync/resources" {
			_, _ = w.Write([]byte(`{"page": 1, "page_size": 10, "total_count": 0, "resources": []}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	s := server.NewMCPServer("meshery-test", "0.1.0")
	client := mustNewMesheryHTTPClient(t, ts.URL, "token", "", ts.Client())
	RegisterKubernetesTools(s, client)

	inProcClient, err := mcpclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("failed to create in-process client: %v", err)
	}
	defer inProcClient.Close()

	ctx := context.Background()
	_, err = inProcClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test", Version: "1.0"},
		},
	})
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	toolsResp, err := inProcClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools failed: %v", err)
	}

	expectedTools := map[string]struct {
		readOnly    bool
		destructive bool
	}{
		"list_clusters":         {readOnly: true, destructive: false},
		"get_cluster":           {readOnly: true, destructive: false},
		"get_cluster_nodes":     {readOnly: true, destructive: false},
		"get_cluster_resources": {readOnly: true, destructive: false},
		"connect_cluster":       {readOnly: false, destructive: false},
		"disconnect_cluster":    {readOnly: false, destructive: true},
	}

	foundTools := make(map[string]bool)

	for _, tool := range toolsResp.Tools {
		if expected, exists := expectedTools[tool.Name]; exists {
			foundTools[tool.Name] = true
			if expected.readOnly {
				if tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint {
					t.Errorf("tool %s should have ReadOnlyHint set to true", tool.Name)
				}
			} else {
				if tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint {
					t.Errorf("tool %s should not have ReadOnlyHint set to true", tool.Name)
				}
			}

			if expected.destructive {
				if tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
					t.Errorf("tool %s should have DestructiveHint set to true", tool.Name)
				}
			} else {
				if tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint {
					t.Errorf("tool %s should not have DestructiveHint set to true", tool.Name)
				}
			}
		}
	}

	for toolName := range expectedTools {
		if !foundTools[toolName] {
			t.Errorf("expected tool %s was not registered", toolName)
		}
	}

	// Call list_clusters through the client
	callRes, err := inProcClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "list_clusters",
		},
	})
	if err != nil || callRes.IsError {
		t.Fatalf("CallTool list_clusters failed: %v, result: %+v", err, callRes)
	}
}

func TestGetClusterNodes_MissingKubernetesServerID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes/contexts" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"totalCount": 1,
				"contexts": [
					{
						"id": "ctx-noserverid",
						"name": "cluster-no-server-id",
						"kubernetesServerId": "",
						"reachable": false
					}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "token", "", ts.Client())
	handler := getClusterNodesHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "get_cluster_nodes",
			Arguments: map[string]interface{}{
				"cluster_id": "ctx-noserverid",
			},
		},
	}
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error when kubernetesServerId is empty")
	}

	text, _ := mcp.AsTextContent(result.Content[0])
	if !strings.Contains(text.Text, "has no associated kubernetesServerId") {
		t.Errorf("expected missing kubernetesServerId message, got: %s", text.Text)
	}
}

func TestResolveClusterContext_MultiPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes/contexts" {
			w.Header().Set("Content-Type", "application/json")
			page := r.URL.Query().Get("page")
			if page == "0" {
				_, _ = w.Write([]byte(`{
					"totalCount": 2,
					"contexts": [
						{"id": "ctx-page-1", "name": "cluster-1"}
					]
				}`))
				return
			}
			if page == "1" {
				_, _ = w.Write([]byte(`{
					"totalCount": 2,
					"contexts": [
						{"id": "ctx-page-2", "name": "cluster-2"}
					]
				}`))
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "token", "", ts.Client())
	ctx, err := resolveClusterContext(context.Background(), client, "ctx-page-2")
	if err != nil {
		t.Fatalf("failed to resolve context across pages: %v", err)
	}
	if ctx.Name != "cluster-2" {
		t.Errorf("expected cluster-2, got %s", ctx.Name)
	}
}

func TestNewMesheryHTTPClient_Security(t *testing.T) {
	// Remote HTTP with token should be rejected
	_, err := NewMesheryHTTPClient("http://remote-meshery.example.com:9081", "secret-token", "", nil)
	if err == nil {
		t.Error("expected error for remote HTTP with token, got nil")
	}

	// Remote HTTPS with token should be allowed
	_, err = NewMesheryHTTPClient("https://remote-meshery.example.com:9081", "secret-token", "", nil)
	if err != nil {
		t.Errorf("expected success for remote HTTPS with token, got: %v", err)
	}

	// Loopback HTTP with token should be allowed (localhost, 127.0.0.1, ::1)
	loopbacks := []string{
		"http://localhost:9081",
		"http://127.0.0.1:9081",
		"http://[::1]:9081",
	}
	for _, lb := range loopbacks {
		_, err = NewMesheryHTTPClient(lb, "secret-token", "", nil)
		if err != nil {
			t.Errorf("expected success for loopback %s with token, got: %v", lb, err)
		}
	}

	// Remote HTTP without token should be allowed
	_, err = NewMesheryHTTPClient("http://remote-meshery.example.com:9081", "", "", nil)
	if err != nil {
		t.Errorf("expected success for remote HTTP without token, got: %v", err)
	}
}

func TestMesheryProviderCookie_SentWithToken(t *testing.T) {
	var capturedCookie *http.Cookie
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCookie = nil
		for _, c := range r.Cookies() {
			if c.Name == "meshery-provider" {
				capturedCookie = c
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalCount": 0, "contexts": []}`))
	}))
	defer ts.Close()

	// 1. Explicit provider passed
	client := mustNewMesheryHTTPClient(t, ts.URL, "token-123", "CustomProvider", ts.Client())
	_, _ = client.GetK8sContexts(context.Background(), 0, 10, "")
	if capturedCookie == nil || capturedCookie.Value != "CustomProvider" {
		t.Fatalf("expected meshery-provider cookie CustomProvider, got: %v", capturedCookie)
	}

	// 2. Unset provider defaults to "Meshery"
	clientDefault := mustNewMesheryHTTPClient(t, ts.URL, "token-123", "", ts.Client())
	_, _ = clientDefault.GetK8sContexts(context.Background(), 0, 10, "")
	if capturedCookie == nil || capturedCookie.Value != "Meshery" {
		t.Fatalf("expected default meshery-provider cookie Meshery, got: %v", capturedCookie)
	}
}

func TestCheckRedirect_RejectInsecureHTTPRedirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect to an external unencrypted HTTP address
		http.Redirect(w, r, "http://external-insecure.example.com/api/test", http.StatusFound)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "secret-token", "", ts.Client())
	_, err := client.GetK8sContexts(context.Background(), 0, 10, "")
	if err == nil {
		t.Fatal("expected error on insecure plaintext redirect with token, got nil")
	}
	if !strings.Contains(err.Error(), "insecure redirect") {
		t.Fatalf("expected 'insecure redirect' error, got: %v", err)
	}
}

func TestConnectCluster_ViaPath_Success(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig.yaml")
	sampleConfig := []byte("apiVersion: v1\nclusters:\n- cluster:\n    server: https://cluster.example.com\n  name: test-cluster\n")
	if err := os.WriteFile(kubeconfigPath, sampleConfig, 0600); err != nil {
		t.Fatalf("failed to write sample kubeconfig: %v", err)
	}

	var receivedFileBytes []byte
	var receivedContextName string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes" && r.Method == http.MethodPost {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			file, _, err := r.FormFile("k8sfile")
			if err != nil {
				http.Error(w, "missing k8sfile", http.StatusBadRequest)
				return
			}
			defer file.Close()
			receivedFileBytes, _ = io.ReadAll(file)
			receivedContextName = r.FormValue("contextName")

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"registeredContexts": [],
				"connectedContexts": [
					{
						"id": "ctx-new-123",
						"name": "test-cluster",
						"server": "https://cluster.example.com"
					}
				],
				"ignoredContexts": [],
				"erroredContexts": []
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "test-token", "", ts.Client())
	handler := connectClusterHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "connect_cluster",
			Arguments: map[string]interface{}{
				"kubeconfig_path": kubeconfigPath,
				"context_name":    "test-cluster",
			},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil || res.IsError {
		t.Fatalf("handler failed: %v, result: %+v", err, res)
	}

	if string(receivedFileBytes) != string(sampleConfig) {
		t.Errorf("expected uploaded kubeconfig bytes %q, got %q", string(sampleConfig), string(receivedFileBytes))
	}
	if receivedContextName != "test-cluster" {
		t.Errorf("expected contextName 'test-cluster', got %q", receivedContextName)
	}

	text, _ := mcp.AsTextContent(res.Content[0])
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	connected, ok := parsed["connected_contexts"].([]interface{})
	if !ok || len(connected) != 1 {
		t.Fatalf("expected 1 connected context, got: %v", parsed["connected_contexts"])
	}
}

func TestConnectCluster_ViaBase64_Success(t *testing.T) {
	rawConfig := "apiVersion: v1\nkind: Config\n"
	b64Config := base64.StdEncoding.EncodeToString([]byte(rawConfig))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"registeredContexts": [
					{
						"id": "ctx-b64",
						"name": "b64-cluster",
						"server": "https://b64.example.com"
					}
				],
				"connectedContexts": [],
				"ignoredContexts": [],
				"erroredContexts": []
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "test-token", "", ts.Client())
	handler := connectClusterHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "connect_cluster",
			Arguments: map[string]interface{}{
				"kubeconfig_base64": b64Config,
			},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil || res.IsError {
		t.Fatalf("handler failed: %v, res: %+v", err, res)
	}

	text, _ := mcp.AsTextContent(res.Content[0])
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	registered, ok := parsed["registered_contexts"].([]interface{})
	if !ok || len(registered) != 1 {
		t.Fatalf("expected 1 registered context, got: %v", parsed["registered_contexts"])
	}
}

func TestConnectCluster_MissingInputs(t *testing.T) {
	client := mustNewMesheryHTTPClient(t, "http://localhost:9081", "", "", nil)
	handler := connectClusterHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "connect_cluster",
			Arguments: map[string]interface{}{},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when missing both kubeconfig inputs")
	}
}

func TestDisconnectCluster_Success(t *testing.T) {
	var deletedContextID string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes/contexts" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"totalCount": 1,
				"contexts": [
					{
						"id": "ctx-delete-target",
						"name": "cluster-to-delete",
						"server": "https://k8s.example.com"
					}
				]
			}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/system/kubernetes/contexts/") && r.Method == http.MethodDelete {
			deletedContextID = strings.TrimPrefix(r.URL.Path, "/api/system/kubernetes/contexts/")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "test-token", "", ts.Client())
	handler := disconnectClusterHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "disconnect_cluster",
			Arguments: map[string]interface{}{
				"cluster_id": "cluster-to-delete",
			},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil || res.IsError {
		t.Fatalf("handler failed: %v, res: %+v", err, res)
	}

	if deletedContextID != "ctx-delete-target" {
		t.Errorf("expected deletedContextID 'ctx-delete-target', got %q", deletedContextID)
	}

	text, _ := mcp.AsTextContent(res.Content[0])
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(text.Text), &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed["status"] != "disconnected" {
		t.Errorf("expected status 'disconnected', got %v", parsed["status"])
	}
	if parsed["context_id"] != "ctx-delete-target" {
		t.Errorf("expected context_id 'ctx-delete-target', got %v", parsed["context_id"])
	}
}

func TestDisconnectCluster_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/system/kubernetes/contexts" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"totalCount": 0, "contexts": []}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := mustNewMesheryHTTPClient(t, ts.URL, "test-token", "", ts.Client())
	handler := disconnectClusterHandler(client)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "disconnect_cluster",
			Arguments: map[string]interface{}{
				"cluster_id": "nonexistent-cluster",
			},
		},
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when cluster is not found")
	}
}
