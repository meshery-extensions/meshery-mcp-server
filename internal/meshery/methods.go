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
