package meshery

import (
	"context"
	"encoding/json"
	"net/url"
)

// Workspace is a verified subset of v1beta3/workspace.Workspace.
type Workspace struct {
	ID             string          `json:"id"`
	Name           string          `json:"name,omitempty"`
	Description    string          `json:"description,omitempty"`
	OrganizationID string          `json:"organizationId,omitempty"`
	OrgName        string          `json:"orgName,omitempty"`
	OwnerID        string          `json:"ownerId,omitempty"`
	CreatedAt      string          `json:"createdAt,omitempty"`
	UpdatedAt      string          `json:"updatedAt,omitempty"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

// WorkspacePage is the GET /api/workspaces envelope.
type WorkspacePage struct {
	Workspaces []Workspace `json:"workspaces,omitempty"`
	Page       int         `json:"page,omitempty"`
	PageSize   int         `json:"pageSize,omitempty"`
	TotalCount int         `json:"totalCount,omitempty"`
}

// ListWorkspaces lists workspaces visible to the current user.
func (c *Client) ListWorkspaces(ctx context.Context, orgID string, opts ListOptions) (WorkspacePage, error) {
	q := url.Values{}
	opts.apply(q)
	if orgID != "" {
		q.Set("orgId", orgID)
	}
	var page WorkspacePage
	err := c.getJSON(ctx, "/api/workspaces", q, "list_workspaces", &page)
	return page, err
}

// GetWorkspace returns one workspace by ID.
func (c *Client) GetWorkspace(ctx context.Context, id string) (Workspace, error) {
	var ws Workspace
	err := c.getJSON(ctx, "/api/workspaces/"+url.PathEscape(id), nil, "get_workspace", &ws)
	return ws, err
}
