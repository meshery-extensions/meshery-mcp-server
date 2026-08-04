package meshery

import (
	"context"
	"net/http"
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
func (c *Client) ListConnections(ctx context.Context, opts ListOptions) (*ConnectionResponse, error) {
	var resp ConnectionResponse
	err := c.do(ctx, http.MethodGet, "/api/integrations/connections", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListPatterns retrieves a paginated list of design patterns (GET /api/pattern).
func (c *Client) ListPatterns(ctx context.Context, opts ListOptions) (*PatternResponse, error) {
	var resp PatternResponse
	err := c.do(ctx, http.MethodGet, "/api/pattern", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListWorkspaces retrieves a paginated list of workspaces (GET /api/workspaces).
func (c *Client) ListWorkspaces(ctx context.Context, opts ListOptions) (*WorkspaceResponse, error) {
	var resp WorkspaceResponse
	err := c.do(ctx, http.MethodGet, "/api/workspaces", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListEnvironments retrieves a paginated list of environments (GET /api/environments).
func (c *Client) ListEnvironments(ctx context.Context, opts ListOptions) (*EnvironmentResponse, error) {
	var resp EnvironmentResponse
	err := c.do(ctx, http.MethodGet, "/api/environments", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListModels retrieves a paginated list of MeshModel registry models (GET /api/registry/models).
func (c *Client) ListModels(ctx context.Context, opts ListOptions) (*ModelResponse, error) {
	var resp ModelResponse
	err := c.do(ctx, http.MethodGet, "/api/registry/models", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListPerformanceProfiles retrieves a paginated list of performance test profiles (GET /api/user/performance/profiles).
func (c *Client) ListPerformanceProfiles(ctx context.Context, opts ListOptions) (*PerformanceProfileResponse, error) {
	var resp PerformanceProfileResponse
	err := c.do(ctx, http.MethodGet, "/api/user/performance/profiles", opts.EncodeValues(), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
