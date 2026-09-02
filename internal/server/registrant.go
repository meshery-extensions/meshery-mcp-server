package server

import "github.com/mark3labs/mcp-go/server"

// Registrant registers one MCP surface, such as tools, resources, or prompts.
type Registrant interface {
	Register(*server.MCPServer) error
}

// RegistrantFunc adapts a registration function to the Registrant interface.
type RegistrantFunc func(*server.MCPServer) error

func (f RegistrantFunc) Register(s *server.MCPServer) error {
	return f(s)
}

// Registry registers each configured MCP surface in order.
type Registry struct {
	registrants []Registrant
}

func NewRegistry(registrants ...Registrant) *Registry {
	return &Registry{registrants: registrants}
}

func (r *Registry) Add(registrant Registrant) {
	r.registrants = append(r.registrants, registrant)
}

func (r *Registry) RegisterAll(s *server.MCPServer) error {
	for _, registrant := range r.registrants {
		if err := registrant.Register(s); err != nil {
			return err
		}
	}
	return nil
}
