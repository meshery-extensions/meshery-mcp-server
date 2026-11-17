package meshery

import "github.com/meshery-extensions/meshery-mcp-server/internal/tools"

type Registry struct {
	*Components
	*Models
	*Relationships
}

func NewRegistry(c *Client) *Registry {
	return &Registry{
		Components:    &Components{client: c},
		Models:        &Models{client: c},
		Relationships: &Relationships{client: c},
	}
}

var _ tools.Backend = (*Registry)(nil)
