package meshery

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// Connection is a verified subset of v1beta3/connection.Connection.
type Connection struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Kind             string          `json:"kind,omitempty"`
	Type             string          `json:"type,omitempty"`
	SubType          string          `json:"subType,omitempty"`
	Status           string          `json:"status,omitempty"`
	CredentialID     string          `json:"credentialId,omitempty"`
	OrganizationID   string          `json:"organizationId,omitempty"`
	CreatedAt        string          `json:"createdAt,omitempty"`
	UpdatedAt        string          `json:"updatedAt,omitempty"`
	ConnectionSchema json.RawMessage `json:"connectionSchema,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	Environments     json.RawMessage `json:"environments,omitempty"`
}

// ConnectionPage is the GET /api/integrations/connections envelope.
type ConnectionPage struct {
	Connections []Connection `json:"connections,omitempty"`
	Page        int          `json:"page,omitempty"`
	PageSize    int          `json:"pageSize,omitempty"`
	TotalCount  int          `json:"totalCount,omitempty"`
}

// ConnectionListOptions adds the repeated filters the connection route reads.
type ConnectionListOptions struct {
	ListOptions
	Statuses []string
	Kinds    []string
	Types    []string
	Name     string
}

// ListConnections lists connections through Meshery Server. The canonical
// page-size parameter is pageSize; pagesize is accepted by older servers.
func (c *Client) ListConnections(ctx context.Context, opts ConnectionListOptions) (ConnectionPage, error) {
	q := url.Values{}
	opts.apply(q)
	q.Del("pagesize")
	_, pageSize := opts.bounds()
	q.Set("pageSize", strconv.Itoa(pageSize))
	for _, s := range opts.Statuses {
		q.Add("status", s)
	}
	for _, k := range opts.Kinds {
		q.Add("kind", k)
	}
	for _, t := range opts.Types {
		q.Add("type", t)
	}
	if opts.Name != "" {
		q.Set("name", opts.Name)
	}
	var page ConnectionPage
	err := c.getJSON(ctx, "/api/integrations/connections", q, "list_connections", &page)
	return page, err
}

// GetConnection returns one connection by ID.
func (c *Client) GetConnection(ctx context.Context, id string) (Connection, error) {
	var conn Connection
	err := c.getJSON(ctx, "/api/integrations/connections/"+url.PathEscape(id), nil, "get_connection", &conn)
	return conn, err
}

// KubernetesContext is a verified subset of a Meshery Kubernetes context.
type KubernetesContext struct {
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Status   string          `json:"status,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// KubernetesContextPage is the GET /api/system/kubernetes/contexts envelope.
type KubernetesContextPage struct {
	Contexts   []KubernetesContext `json:"contexts,omitempty"`
	Page       int                 `json:"page,omitempty"`
	PageSize   int                 `json:"pageSize,omitempty"`
	TotalCount int                 `json:"totalCount,omitempty"`
}

// ListKubernetesContexts lists the legacy Kubernetes contexts known to the
// running Meshery Server.
func (c *Client) ListKubernetesContexts(ctx context.Context, opts ListOptions) (KubernetesContextPage, error) {
	var page KubernetesContextPage
	err := c.getJSON(ctx, "/api/system/kubernetes/contexts", queryFor(opts), "list_kubernetes_contexts", &page)
	return page, err
}
