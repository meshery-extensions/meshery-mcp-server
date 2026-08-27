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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
)

// K8sContext represents a Kubernetes cluster context registered in Meshery.
type K8sContext struct {
	ID                 string `json:"id,omitempty"`
	Name               string `json:"name,omitempty"`
	Server             string `json:"server,omitempty"`
	KubernetesServerID string `json:"kubernetesServerId,omitempty"`
	ConnectionID       string `json:"connectionId,omitempty"`
	DeploymentType     string `json:"deploymentType,omitempty"`
	Version            string `json:"version,omitempty"`
	Reachable          bool   `json:"reachable"`
	CreatedAt          string `json:"createdAt,omitempty"`
	UpdatedAt          string `json:"updatedAt,omitempty"`
}

// MesheryK8sContextPage represents a paginated list of Kubernetes contexts returned by Meshery.
type MesheryK8sContextPage struct {
	TotalCount int          `json:"totalCount"`
	Contexts   []K8sContext `json:"contexts"`
}

// MeshSyncResource represents an individual Kubernetes resource discovered by MeshSync.
type MeshSyncResource struct {
	ID         string                 `json:"id"`
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	ClusterID  string                 `json:"cluster_id"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Spec       map[string]interface{} `json:"spec,omitempty"`
	Status     map[string]interface{} `json:"status,omitempty"`
}

// MeshSyncResourcesResponse represents the API response for discovered MeshSync resources.
type MeshSyncResourcesResponse struct {
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalCount int64              `json:"total_count"`
	Resources  []MeshSyncResource `json:"resources"`
}

// NodeSummary contains details of a Kubernetes cluster node discovered by MeshSync.
type NodeSummary struct {
	Name           string                 `json:"name"`
	Status         string                 `json:"status"`
	Roles          []string               `json:"roles,omitempty"`
	KubeletVersion string                 `json:"kubelet_version,omitempty"`
	OSImage        string                 `json:"os_image,omitempty"`
	Architecture   string                 `json:"architecture,omitempty"`
	Capacity       map[string]interface{} `json:"capacity,omitempty"`
	Allocatable    map[string]interface{} `json:"allocatable,omitempty"`
	Addresses      []map[string]string    `json:"addresses,omitempty"`
}

// ResourceSummary contains high-level information about a discovered Kubernetes resource.
type ResourceSummary struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Namespace         string `json:"namespace,omitempty"`
	Kind              string `json:"kind"`
	APIVersion        string `json:"apiVersion"`
	CreationTimestamp string `json:"creation_timestamp,omitempty"`
	Status            string `json:"status,omitempty"`
}

// KubernetesClient defines the interface for communicating with Meshery's Kubernetes and MeshSync APIs.
type KubernetesClient interface {
	GetK8sContexts(ctx context.Context, page, pageSize int, search string) (*MesheryK8sContextPage, error)
	GetMeshSyncResources(ctx context.Context, kubernetesServerID string, kind string, namespace string, page, pageSize int) (*MeshSyncResourcesResponse, error)
	Ping(ctx context.Context) error
}

// MesheryHTTPClient implements KubernetesClient by calling Meshery Server's REST endpoints.
type MesheryHTTPClient struct {
	baseURL    string
	token      string
	provider   string
	httpClient *http.Client
}

// errClient implements KubernetesClient by returning an initialization error.
type errClient struct {
	err error
}

func (e *errClient) GetK8sContexts(ctx context.Context, page, pageSize int, search string) (*MesheryK8sContextPage, error) {
	return nil, e.err
}

func (e *errClient) GetMeshSyncResources(ctx context.Context, kubernetesServerID string, kind string, namespace string, page, pageSize int) (*MeshSyncResourcesResponse, error) {
	return nil, e.err
}

func (e *errClient) Ping(ctx context.Context) error {
	return e.err
}

// NewMesheryHTTPClient creates a new Meshery HTTP client.
// Rejects non-loopback HTTP endpoints when a token is provided to prevent credential exposure.
func NewMesheryHTTPClient(baseURL, token, provider string, httpClient *http.Client) (*MesheryHTTPClient, error) {
	if baseURL == "" {
		baseURL = config.DefaultMeshServerURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	if token != "" && u.Scheme == "http" {
		host := u.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return nil, errors.New("bearer tokens require HTTPS unless using a loopback address")
		}
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 15 * time.Second,
		}
	} else {
		copied := *httpClient
		httpClient = &copied
	}

	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if strings.Contains(req.URL.Path, "/provider") || strings.Contains(req.URL.Path, "/auth/login") {
			return errors.New("authentication required: Meshery session expired or absent; please run 'mesheryctl system login' or set MESHERY_API_TOKEN")
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}

	return &MesheryHTTPClient{
		baseURL:    baseURL,
		token:      token,
		provider:   provider,
		httpClient: httpClient,
	}, nil
}

// NewDefaultKubernetesClient initializes a client from the process environment config.
func NewDefaultKubernetesClient() KubernetesClient {
	cfg := config.Load()
	c, err := NewMesheryHTTPClient(cfg.MeshServerURL, cfg.MeshAPIToken, "", nil)
	if err != nil {
		return &errClient{err: err}
	}
	return c
}

// GetK8sContexts calls GET /api/system/kubernetes/contexts.
func (c *MesheryHTTPClient) GetK8sContexts(ctx context.Context, page, pageSize int, search string) (*MesheryK8sContextPage, error) {
	endpoint := c.baseURL + "/api/system/kubernetes/contexts"
	q := url.Values{}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("pagesize", strconv.Itoa(pageSize))
	}
	if search != "" {
		q.Set("search", search)
	}
	if len(q) > 0 {
		endpoint += "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create contexts request: %w", err)
	}

	c.applyAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch kubernetes contexts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("meshery GET /api/system/kubernetes/contexts returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rawPage struct {
		TotalCount int `json:"totalCount"`
		Contexts   []struct {
			ID                 string          `json:"id"`
			Name               string          `json:"name"`
			Server             string          `json:"server"`
			KubernetesServerID json.RawMessage `json:"kubernetesServerId"`
			ConnectionID       string          `json:"connectionId"`
			DeploymentType     string          `json:"deploymentType"`
			Version            string          `json:"version"`
			Reachable          bool            `json:"reachable"`
			CreatedAt          json.RawMessage `json:"createdAt"`
			UpdatedAt          json.RawMessage `json:"updatedAt"`
		} `json:"contexts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawPage); err != nil {
		return nil, fmt.Errorf("decode kubernetes contexts: %w", err)
	}

	result := &MesheryK8sContextPage{
		TotalCount: rawPage.TotalCount,
		Contexts:   make([]K8sContext, len(rawPage.Contexts)),
	}

	for i, rc := range rawPage.Contexts {
		result.Contexts[i] = K8sContext{
			ID:                 rc.ID,
			Name:               rc.Name,
			Server:             rc.Server,
			KubernetesServerID: parseUUIDOrString(rc.KubernetesServerID),
			ConnectionID:       rc.ConnectionID,
			DeploymentType:     rc.DeploymentType,
			Version:            rc.Version,
			Reachable:          rc.Reachable,
			CreatedAt:          parseStringOrDate(rc.CreatedAt),
			UpdatedAt:          parseStringOrDate(rc.UpdatedAt),
		}
	}

	return result, nil
}

// GetMeshSyncResources calls GET /api/system/meshsync/resources.
// cluster_id scoping is mandatory to avoid Meshery returning silent empty results.
func (c *MesheryHTTPClient) GetMeshSyncResources(ctx context.Context, kubernetesServerID string, kind string, namespace string, page, pageSize int) (*MeshSyncResourcesResponse, error) {
	if kubernetesServerID == "" {
		return nil, errors.New("kubernetesServerID is required for MeshSync queries")
	}

	endpoint := c.baseURL + "/api/system/meshsync/resources"
	q := url.Values{}

	// Meshery expects clusterIds as a JSON-encoded array string: e.g. ["ksid-123"]
	clusterIDsJSON, err := json.Marshal([]string{kubernetesServerID})
	if err != nil {
		return nil, fmt.Errorf("encode clusterIds: %w", err)
	}
	q.Set("clusterIds", string(clusterIDsJSON))

	if kind != "" {
		q.Set("kind", kind)
	}
	if namespace != "" {
		q.Set("namespace", namespace)
	}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("pagesize", strconv.Itoa(pageSize))
	}
	q.Set("status", "true")
	q.Set("spec", "true")

	endpoint += "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create meshsync request: %w", err)
	}

	c.applyAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch meshsync resources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("meshery GET /api/system/meshsync/resources returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var response MeshSyncResourcesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode meshsync resources: %w", err)
	}

	return &response, nil
}

// Ping checks connectivity with Meshery Kubernetes endpoint.
func (c *MesheryHTTPClient) Ping(ctx context.Context) error {
	endpoint := c.baseURL + "/api/system/kubernetes/ping"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	c.applyAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("kubernetes ping returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *MesheryHTTPClient) applyAuthHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.AddCookie(&http.Cookie{Name: "token", Value: c.token})
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.provider != "" {
		req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: c.provider})
	}
}

// RegisterKubernetesTools registers all read-only Kubernetes MCP tools.
func RegisterKubernetesTools(s *server.MCPServer, client KubernetesClient) {
	if client == nil {
		client = NewDefaultKubernetesClient()
	}

	// 1. list_clusters
	listClustersTool := mcp.NewTool("list_clusters",
		mcp.WithDescription("List all connected Kubernetes clusters registered in Meshery."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithNumber("page",
			mcp.Description("Page number for paginated results (default: 1)"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Number of cluster summaries per page (default: 25)"),
		),
		mcp.WithString("search",
			mcp.Description("Optional search string to filter clusters by name"),
		),
	)
	s.AddTool(listClustersTool, listClustersHandler(client))

	// 2. get_cluster
	getClusterTool := mcp.NewTool("get_cluster",
		mcp.WithDescription("Get details for a specific Kubernetes cluster (supports Context ID, Connection ID, or Kubernetes Server ID)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("cluster_id",
			mcp.Required(),
			mcp.Description("Identifier of the cluster (Context ID, Connection ID, or Kubernetes Server ID)"),
		),
	)
	s.AddTool(getClusterTool, getClusterHandler(client))

	// 3. get_cluster_nodes
	getClusterNodesTool := mcp.NewTool("get_cluster_nodes",
		mcp.WithDescription("List cluster nodes and their status, roles, and resource capacity."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("cluster_id",
			mcp.Required(),
			mcp.Description("Identifier of the cluster (Context ID, Connection ID, or Kubernetes Server ID)"),
		),
	)
	s.AddTool(getClusterNodesTool, getClusterNodesHandler(client))

	// 4. get_cluster_resources
	getClusterResourcesTool := mcp.NewTool("get_cluster_resources",
		mcp.WithDescription("List Kubernetes resources on a cluster discovered by MeshSync (e.g. Pods, Services, Deployments)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("cluster_id",
			mcp.Required(),
			mcp.Description("Identifier of the cluster (Context ID, Connection ID, or Kubernetes Server ID)"),
		),
		mcp.WithString("resource_kind",
			mcp.Description("Kubernetes resource kind filter (e.g., 'Pod', 'Service', 'Deployment')"),
		),
		mcp.WithString("namespace",
			mcp.Description("Namespace filter (e.g., 'default', 'kube-system')"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number for paginated results (default: 1)"),
		),
		mcp.WithNumber("page_size",
			mcp.Description("Number of resources per page (default: 25)"),
		),
	)
	s.AddTool(getClusterResourcesTool, getClusterResourcesHandler(client))
}

// listClustersHandler returns a tool handler that lists Kubernetes clusters.
func listClustersHandler(client KubernetesClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArguments(req)
		page := getIntArg(args, "page", 1)
		pageSize := getIntArg(args, "page_size", 0)
		if pageSize <= 0 {
			pageSize = getIntArg(args, "pageSize", 25)
		}
		search := getStringArg(args, "search")

		contextPage, err := client.GetK8sContexts(ctx, page, pageSize, search)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list clusters: %v", err)), nil
		}

		type clusterItem struct {
			ID                 string `json:"id"`
			Name               string `json:"name"`
			Server             string `json:"server"`
			KubernetesServerID string `json:"kubernetes_server_id,omitempty"`
			ConnectionID       string `json:"connection_id,omitempty"`
			DeploymentType     string `json:"deployment_type,omitempty"`
			Version            string `json:"version,omitempty"`
			Reachable          bool   `json:"reachable"`
			CreatedAt          string `json:"created_at,omitempty"`
			UpdatedAt          string `json:"updated_at,omitempty"`
		}

		clusters := make([]clusterItem, len(contextPage.Contexts))
		for i, c := range contextPage.Contexts {
			clusters[i] = clusterItem{
				ID:                 c.ID,
				Name:               c.Name,
				Server:             c.Server,
				KubernetesServerID: c.KubernetesServerID,
				ConnectionID:       c.ConnectionID,
				DeploymentType:     c.DeploymentType,
				Version:            c.Version,
				Reachable:          c.Reachable,
				CreatedAt:          c.CreatedAt,
				UpdatedAt:          c.UpdatedAt,
			}
		}

		response := map[string]interface{}{
			"total_count": contextPage.TotalCount,
			"page":        page,
			"page_size":   pageSize,
			"clusters":    clusters,
		}

		out, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal clusters: %v", err)), nil
		}

		return mcp.NewToolResultText(string(out)), nil
	}
}

// getClusterHandler returns a tool handler that gets details for a specific cluster.
func getClusterHandler(client KubernetesClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArguments(req)
		clusterID := getStringArg(args, "cluster_id")
		if clusterID == "" {
			return mcp.NewToolResultError("cluster_id parameter is required"), nil
		}

		targetCtx, err := resolveClusterContext(ctx, client, clusterID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		nodeCount := 0
		healthStatus := "Unknown"
		if targetCtx.Reachable {
			healthStatus = "Reachable"
		} else {
			healthStatus = "Unreachable"
		}

		// If we have a kubernetesServerID, query MeshSync for node count
		if targetCtx.KubernetesServerID != "" {
			nodeRes, err := client.GetMeshSyncResources(ctx, targetCtx.KubernetesServerID, "Node", "", 1, 1)
			if err == nil && nodeRes != nil {
				if nodeRes.TotalCount > 0 {
					nodeCount = int(nodeRes.TotalCount)
				} else {
					nodeCount = len(nodeRes.Resources)
				}
				if targetCtx.Reachable && nodeCount > 0 {
					healthStatus = "Healthy"
				}
			}
		}

		clusterDetails := map[string]interface{}{
			"id":                   targetCtx.ID,
			"name":                 targetCtx.Name,
			"server":               targetCtx.Server,
			"kubernetes_server_id": targetCtx.KubernetesServerID,
			"connection_id":        targetCtx.ConnectionID,
			"version":              targetCtx.Version,
			"deployment_type":      targetCtx.DeploymentType,
			"node_count":           nodeCount,
			"health_status":        healthStatus,
			"reachable":            targetCtx.Reachable,
			"created_at":           targetCtx.CreatedAt,
			"updated_at":           targetCtx.UpdatedAt,
		}

		out, err := json.MarshalIndent(clusterDetails, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal cluster details: %v", err)), nil
		}

		return mcp.NewToolResultText(string(out)), nil
	}
}

// getClusterNodesHandler returns a tool handler that lists nodes for a cluster.
func getClusterNodesHandler(client KubernetesClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArguments(req)
		clusterID := getStringArg(args, "cluster_id")
		if clusterID == "" {
			return mcp.NewToolResultError("cluster_id parameter is required"), nil
		}

		targetCtx, err := resolveClusterContext(ctx, client, clusterID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		k8sServerID := targetCtx.KubernetesServerID
		if k8sServerID == "" {
			return mcp.NewToolResultError(fmt.Sprintf("cluster %q has no associated kubernetesServerId; ensure MeshSync is deployed and active for this cluster", clusterID)), nil
		}

		nodes := make([]NodeSummary, 0)
		page := 1
		pageSize := 50

		for {
			nodeRes, err := client.GetMeshSyncResources(ctx, k8sServerID, "Node", "", page, pageSize)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to fetch nodes for cluster %s: %v", clusterID, err)), nil
			}

			for _, r := range nodeRes.Resources {
				nodes = append(nodes, parseNodeResource(r))
			}

			if len(nodeRes.Resources) == 0 || int64(len(nodes)) >= nodeRes.TotalCount {
				break
			}
			page++
		}

		response := map[string]interface{}{
			"cluster_id":           clusterID,
			"kubernetes_server_id": k8sServerID,
			"node_count":           len(nodes),
			"nodes":                nodes,
		}

		out, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal nodes: %v", err)), nil
		}

		return mcp.NewToolResultText(string(out)), nil
	}
}

// getClusterResourcesHandler returns a tool handler that lists Kubernetes resources scoped by cluster.
func getClusterResourcesHandler(client KubernetesClient) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := parseArguments(req)
		clusterID := getStringArg(args, "cluster_id")
		if clusterID == "" {
			return mcp.NewToolResultError("cluster_id parameter is required; run list_clusters to discover available cluster IDs"), nil
		}

		kind := getStringArg(args, "resource_kind")
		namespace := getStringArg(args, "namespace")
		page := getIntArg(args, "page", 1)
		pageSize := getIntArg(args, "page_size", 25)

		targetCtx, err := resolveClusterContext(ctx, client, clusterID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		k8sServerID := targetCtx.KubernetesServerID
		if k8sServerID == "" {
			return mcp.NewToolResultError(fmt.Sprintf("cluster %q has no associated kubernetesServerId; ensure MeshSync is deployed and active for this cluster", clusterID)), nil
		}

		meshSyncResp, err := client.GetMeshSyncResources(ctx, k8sServerID, kind, namespace, page, pageSize)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch cluster resources: %v", err)), nil
		}

		summaries := make([]ResourceSummary, 0, len(meshSyncResp.Resources))
		for _, r := range meshSyncResp.Resources {
			summaries = append(summaries, parseResourceSummary(r))
		}

		response := map[string]interface{}{
			"cluster_id":           clusterID,
			"kubernetes_server_id": k8sServerID,
			"resource_kind":        kind,
			"namespace":            namespace,
			"page":                 page,
			"page_size":            pageSize,
			"total_count":          meshSyncResp.TotalCount,
			"resources":            summaries,
		}

		out, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal resources: %v", err)), nil
		}

		return mcp.NewToolResultText(string(out)), nil
	}
}

// resolveClusterContext matches a cluster_id to a registered K8sContext.
// It paginates through all available contexts and checks ID, ConnectionID,
// KubernetesServerID, and Name.
func resolveClusterContext(ctx context.Context, client KubernetesClient, clusterID string) (*K8sContext, error) {
	page := 1
	pageSize := 50
	scanned := 0

	for {
		res, err := client.GetK8sContexts(ctx, page, pageSize, "")
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve contexts for cluster resolution: %w", err)
		}

		for _, c := range res.Contexts {
			if c.ID == clusterID || c.ConnectionID == clusterID || c.KubernetesServerID == clusterID || c.Name == clusterID {
				return &c, nil
			}
		}

		scanned += len(res.Contexts)
		if len(res.Contexts) == 0 || scanned >= res.TotalCount {
			break
		}
		page++
	}

	return nil, fmt.Errorf("cluster %q not found in Meshery registered contexts; run list_clusters to see available clusters", clusterID)
}

// parseNodeResource extracts NodeSummary from a MeshSync Node resource.
func parseNodeResource(r MeshSyncResource) NodeSummary {
	node := NodeSummary{
		Name:      r.ID,
		Status:    "Unknown",
		Roles:     []string{},
		Addresses: nil,
	}

	if meta := r.Metadata; meta != nil {
		if name, ok := meta["name"].(string); ok && name != "" {
			node.Name = name
		}
		if labels, ok := meta["labels"].(map[string]interface{}); ok {
			for k := range labels {
				if strings.HasPrefix(k, "node-role.kubernetes.io/") {
					role := strings.TrimPrefix(k, "node-role.kubernetes.io/")
					if role != "" {
						node.Roles = append(node.Roles, role)
					}
				}
			}
			if arch, ok := labels["kubernetes.io/arch"].(string); ok {
				node.Architecture = arch
			}
		}
	}

	if status := r.Status; status != nil {
		if conditions, ok := status["conditions"].([]interface{}); ok {
			for _, cond := range conditions {
				if cMap, ok := cond.(map[string]interface{}); ok {
					if cMap["type"] == "Ready" {
						if cMap["status"] == "True" {
							node.Status = "Ready"
						} else {
							node.Status = "NotReady"
						}
					}
				}
			}
		}
		if nodeInfo, ok := status["nodeInfo"].(map[string]interface{}); ok {
			if kv, ok := nodeInfo["kubeletVersion"].(string); ok {
				node.KubeletVersion = kv
			}
			if osImg, ok := nodeInfo["osImage"].(string); ok {
				node.OSImage = osImg
			}
			if arch, ok := nodeInfo["architecture"].(string); ok {
				node.Architecture = arch
			}
		}
		if cap, ok := status["capacity"].(map[string]interface{}); ok {
			node.Capacity = cap
		}
		if alloc, ok := status["allocatable"].(map[string]interface{}); ok {
			node.Allocatable = alloc
		}
		if addrs, ok := status["addresses"].([]interface{}); ok {
			for _, a := range addrs {
				if aMap, ok := a.(map[string]interface{}); ok {
					t, _ := aMap["type"].(string)
					addr, _ := aMap["address"].(string)
					if t != "" && addr != "" {
						node.Addresses = append(node.Addresses, map[string]string{
							"type":    t,
							"address": addr,
						})
					}
				}
			}
		}
	}

	return node
}

// parseResourceSummary formats an individual MeshSync resource into ResourceSummary.
func parseResourceSummary(r MeshSyncResource) ResourceSummary {
	summary := ResourceSummary{
		ID:         r.ID,
		Kind:       r.Kind,
		APIVersion: r.APIVersion,
		Status:     "Unknown",
	}

	if meta := r.Metadata; meta != nil {
		if name, ok := meta["name"].(string); ok {
			summary.Name = name
		}
		if ns, ok := meta["namespace"].(string); ok {
			summary.Namespace = ns
		}
		if ct, ok := meta["creationTimestamp"].(string); ok {
			summary.CreationTimestamp = ct
		}
	}
	if summary.Name == "" {
		summary.Name = r.ID
	}

	if status := r.Status; status != nil {
		if phase, ok := status["phase"].(string); ok {
			summary.Status = phase
		}
	}

	return summary
}

// parseArguments normalizes tool call arguments map.
func parseArguments(req mcp.CallToolRequest) map[string]interface{} {
	if args, ok := req.Params.Arguments.(map[string]interface{}); ok {
		return args
	}
	if len(req.Params.RawArguments) > 0 {
		var args map[string]interface{}
		_ = json.Unmarshal(req.Params.RawArguments, &args)
		return args
	}
	return make(map[string]interface{})
}

// getStringArg extracts a trimmed string argument.
func getStringArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// getIntArg extracts an integer argument with fallback default.
func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if v, ok := args[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case string:
			if i, err := strconv.Atoi(n); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

// parseUUIDOrString extracts a UUID string whether serialized as a string or struct.
func parseUUIDOrString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		UUID string `json:"UUID"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.UUID != "" {
		return obj.UUID
	}
	return strings.Trim(string(raw), "\"")
}

// parseStringOrDate extracts a date string whether serialized as string or object.
func parseStringOrDate(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return strings.Trim(string(raw), "\"")
}
