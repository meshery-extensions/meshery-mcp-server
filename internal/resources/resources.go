// Package resources implements the MCP resources the server exposes: read-only
// data the AI can load as context.
//
// Resources and resource templates are described by portable types. Registration
// adapts them to github.com/modelcontextprotocol/go-sdk's Server.AddResource
// and Server.AddResourceTemplate, so the handler signature stays SDK-agnostic; only the
// registration wrapper touches go-sdk types. See docs/adr/0001-mcp-sdk-choice.md.
package resources

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Resource is a fixed-address MCP resource.
type Resource struct {
	// URI is the resource's identifier, used in resources/read requests.
	URI string
	// Name is a human-readable name for the resource.
	Name string
	// Description is shown to clients to help them choose resources.
	Description string
	// MIMEType is the resource's media type, if known.
	MIMEType string
	// Handler reads the resource. It returns the contents as a string, or an
	// error.
	Handler func(ctx context.Context, uri string) (string, error)
}

// ResourceTemplate is a URI-templated MCP resource (RFC 6570).
type ResourceTemplate struct {
	// URITemplate is the URI template, e.g. "meshery://clusters/{cluster_id}/topology".
	URITemplate string
	// Name is a human-readable name for the template.
	Name string
	// Description is shown to clients to help them choose templates.
	Description string
	// MIMEType is the resource's media type, if known.
	MIMEType string
	// Handler reads a resource matching the template. uri is the fully-expanded
	// URI requested by the client.
	Handler func(ctx context.Context, uri string) (string, error)
}

// Register registers every known resource and template with s. Called once at
// startup from cmd/meshery-mcp-server/main.go.
func Register(s *mcp.Server) error {
	for _, r := range allResources() {
		if err := registerResource(s, r); err != nil {
			return err
		}
	}
	for _, t := range allTemplates() {
		if err := registerTemplate(s, t); err != nil {
			return err
		}
	}
	return nil
}

// registerResource adapts a portable Resource to go-sdk's Server.AddResource.
func registerResource(s *mcp.Server, r Resource) error {
	s.AddResource(&mcp.Resource{
		URI:         r.URI,
		Name:        r.Name,
		Description: r.Description,
		MIMEType:    r.MIMEType,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		content, err := r.Handler(ctx, req.Params.URI)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: r.MIMEType,
				Text:     content,
			}},
		}, nil
	})
	return nil
}

// registerTemplate adapts a portable ResourceTemplate to go-sdk's
// Server.AddResourceTemplate.
func registerTemplate(s *mcp.Server, t ResourceTemplate) error {
	s.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: t.URITemplate,
		Name:        t.Name,
		Description: t.Description,
		MIMEType:    t.MIMEType,
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		content, err := t.Handler(ctx, req.Params.URI)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      req.Params.URI,
				MIMEType: t.MIMEType,
				Text:     content,
			}},
		}, nil
	})
	return nil
}

// allResources returns the canonical list of fixed-address resources.
// Initially empty: resources are added as the MVP surface is built.
func allResources() []Resource {
	return nil
}

// allTemplates returns the canonical list of URI-templated resources.
// Initially empty: templates are added as the MVP surface is built.
func allTemplates() []ResourceTemplate {
	return nil
}
