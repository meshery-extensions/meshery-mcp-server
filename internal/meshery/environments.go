package meshery

import (
	"context"
	"encoding/json"
	"net/url"
)

// Environment is a verified subset of v1beta3/environment.Environment.
type Environment struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	OrganizationID string          `json:"organizationId,omitempty"`
	Purpose        string          `json:"purpose,omitempty"`
	CreatedAt      string          `json:"createdAt,omitempty"`
	UpdatedAt      string          `json:"updatedAt,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// EnvironmentPage is the GET /api/environments envelope.
type EnvironmentPage struct {
	Environments []Environment `json:"environments,omitempty"`
	Page         int           `json:"page,omitempty"`
	PageSize     int           `json:"pageSize,omitempty"`
	TotalCount   int           `json:"totalCount,omitempty"`
}

// ListEnvironments lists environments in an organization. orgId is required
// by GET /api/environments.
func (c *Client) ListEnvironments(ctx context.Context, orgID string, opts ListOptions) (EnvironmentPage, error) {
	q := url.Values{}
	opts.apply(q)
	q.Set("orgId", orgID)
	var page EnvironmentPage
	err := c.getJSON(ctx, "/api/environments", q, "list_environments", &page)
	return page, err
}

// GetEnvironment returns one environment by ID.
func (c *Client) GetEnvironment(ctx context.Context, id string) (Environment, error) {
	var env Environment
	err := c.getJSON(ctx, "/api/environments/"+url.PathEscape(id), nil, "get_environment", &env)
	return env, err
}
