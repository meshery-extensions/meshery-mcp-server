package meshery

import (
	"context"
	"encoding/json"
)

// PerformanceProfile is a verified subset of a performance profile.
type PerformanceProfile struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// PerformanceProfilePage is the GET /api/performance/profiles envelope.
type PerformanceProfilePage struct {
	Profiles   []PerformanceProfile `json:"profiles,omitempty"`
	Page       int                  `json:"page,omitempty"`
	PageSize   int                  `json:"pageSize,omitempty"`
	TotalCount int                  `json:"totalCount,omitempty"`
}

// PerformanceResultPage is the GET /api/performance/results envelope.
type PerformanceResultPage struct {
	Results    []json.RawMessage `json:"results,omitempty"`
	Page       int               `json:"page,omitempty"`
	PageSize   int               `json:"pageSize,omitempty"`
	TotalCount int               `json:"totalCount,omitempty"`
}

// ListPerformanceProfiles lists saved performance test profiles.
func (c *Client) ListPerformanceProfiles(ctx context.Context, opts ListOptions) (PerformanceProfilePage, error) {
	var page PerformanceProfilePage
	err := c.getJSON(ctx, "/api/performance/profiles", queryFor(opts), "list_performance_profiles", &page)
	return page, err
}

// ListPerformanceResults lists recorded performance test results.
func (c *Client) ListPerformanceResults(ctx context.Context, opts ListOptions) (PerformanceResultPage, error) {
	var page PerformanceResultPage
	err := c.getJSON(ctx, "/api/performance/results", queryFor(opts), "list_performance_results", &page)
	return page, err
}
