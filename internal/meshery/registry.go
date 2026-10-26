package meshery

import "github.com/meshery-extensions/meshery-mcp-server/internal/tools"

// Registry implements tools.Registry on top of a Meshery client. Each embedded
// type owns one domain's endpoints and response mapping.
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

var _ tools.Registry = (*Registry)(nil)
