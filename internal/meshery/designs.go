package meshery

import (
	"context"
	"encoding/json"
	"net/url"
)

// Design is a verified subset of a Meshery pattern/design. Schema-derived
// fields not needed by tools stay in Raw.
type Design struct {
	ID          string          `json:"id,omitempty"`
	Name        string          `json:"name,omitempty"`
	PatternFile json.RawMessage `json:"patternFile,omitempty"`
	Visibility  string          `json:"visibility,omitempty"`
	CreatedAt   string          `json:"createdAt,omitempty"`
	UpdatedAt   string          `json:"updatedAt,omitempty"`
}

// DesignPage is the GET /api/pattern envelope.
type DesignPage struct {
	Patterns   []Design `json:"patterns,omitempty"`
	Page       int      `json:"page,omitempty"`
	PageSize   int      `json:"pageSize,omitempty"`
	TotalCount int      `json:"totalCount,omitempty"`
}

// DesignListOptions adds the design-specific filters to ListOptions.
type DesignListOptions struct {
	ListOptions
	UpdatedAfter string
	Metrics      string
}

// ListDesigns lists designs from the current provider through Meshery Server.
func (c *Client) ListDesigns(ctx context.Context, opts DesignListOptions) (DesignPage, error) {
	q := url.Values{}
	opts.apply(q)
	if opts.UpdatedAfter != "" {
		q.Set("updated_after", opts.UpdatedAfter)
	}
	if opts.Metrics != "" {
		q.Set("metrics", opts.Metrics)
	}
	var page DesignPage
	err := c.getJSON(ctx, "/api/pattern", q, "list_designs", &page)
	return page, err
}

// GetDesign returns one design by ID.
func (c *Client) GetDesign(ctx context.Context, id string) (Design, error) {
	var design Design
	err := c.getJSON(ctx, "/api/pattern/"+url.PathEscape(id), nil, "get_design", &design)
	return design, err
}
