package meshery

import (
	"context"
	"net/http"

	connection "github.com/meshery/schemas/models/v1beta3/connection"
	design "github.com/meshery/schemas/models/v1beta3/design"
	environment "github.com/meshery/schemas/models/v1beta3/environment"
	// MeshModel schemas are currently generated under v1beta2.
	// Update this import if a newer schema version becomes available.
	model "github.com/meshery/schemas/models/v1beta2/model"
	performanceprofile "github.com/meshery/schemas/models/v1beta3/performance_profile"
	workspace "github.com/meshery/schemas/models/v1beta3/workspace"
)

// Ping checks server connectivity and returns system version metadata (GET /api/system/version).
func (c *Client) Ping(ctx context.Context) (*Version, error) {
	var v Version
	err := c.do(ctx, http.MethodGet, "/api/system/version", nil, nil, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListConnections retrieves a paginated list of connections (GET /api/integrations/connections).
func (c *Client) ListConnections(ctx context.Context, opts ListOptions) (*connection.ConnectionPage, error) {
	var resp connection.ConnectionPage
	err := c.do(ctx, http.MethodGet, "/api/integrations/connections", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListPatterns retrieves a paginated list of design patterns (GET /api/pattern).
func (c *Client) ListPatterns(ctx context.Context, opts ListOptions) (*design.MesheryPatternPage, error) {
	var resp design.MesheryPatternPage
	err := c.do(ctx, http.MethodGet, "/api/pattern", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListWorkspaces retrieves a paginated list of workspaces (GET /api/workspaces).
func (c *Client) ListWorkspaces(ctx context.Context, opts ListOptions) (*workspace.WorkspacePage, error) {
	var resp workspace.WorkspacePage
	err := c.do(ctx, http.MethodGet, "/api/workspaces", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListEnvironments retrieves a paginated list of environments (GET /api/environments).
func (c *Client) ListEnvironments(ctx context.Context, opts ListOptions) (*environment.EnvironmentPage, error) {
	var resp environment.EnvironmentPage
	err := c.do(ctx, http.MethodGet, "/api/environments", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListModels retrieves a paginated list of MeshModel registry models (GET /api/registry/models).
func (c *Client) ListModels(ctx context.Context, opts ListOptions) (*model.MeshModelModelsPage, error) {
	var resp model.MeshModelModelsPage
	err := c.do(ctx, http.MethodGet, "/api/registry/models", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListPerformanceProfiles retrieves a paginated list of performance test profiles (GET /api/user/performance/profiles).
func (c *Client) ListPerformanceProfiles(ctx context.Context, opts ListOptions) (*performanceprofile.PerformanceProfilePage, error) {
	var resp performanceprofile.PerformanceProfilePage
	err := c.do(ctx, http.MethodGet, "/api/user/performance/profiles", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
