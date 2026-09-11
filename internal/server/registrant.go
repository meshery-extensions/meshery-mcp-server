// Copyright Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
)

// Registrant registers one MCP surface, such as tools, resources, or prompts.
type Registrant interface {
	Register(*server.MCPServer) error
}

// RegistrantFunc adapts a registration function to the Registrant interface.
type RegistrantFunc func(*server.MCPServer) error

// Register invokes f with s.
func (f RegistrantFunc) Register(s *server.MCPServer) error {
	return f(s)
}

// named pairs a registrant with its surface name so registration errors
// attribute failures to the right surface.
type named struct {
	name       string
	registrant Registrant
}

// Register delegates to the wrapped registrant.
func (n named) Register(s *server.MCPServer) error {
	return n.registrant.Register(s)
}

// Named labels a registrant with its surface name so RegisterAll can identify
// which surface failed to register while preserving the original error.
func Named(name string, registrant Registrant) Registrant {
	return named{name: name, registrant: registrant}
}

// Registry registers each configured MCP surface in order.
type Registry struct {
	registrants []Registrant
}

// NewRegistry creates a Registry from registrants.
func NewRegistry(registrants ...Registrant) *Registry {
	return &Registry{registrants: registrants}
}

// Add appends a registrant to the registry.
func (r *Registry) Add(registrant Registrant) {
	r.registrants = append(r.registrants, registrant)
}

// RegisterAll registers every configured surface in order and stops at the
// first registration error. The returned error names the surface that failed
// (its label when wrapped with Named, otherwise its type) and wraps the
// original error for inspection.
func (r *Registry) RegisterAll(s *server.MCPServer) error {
	for _, registrant := range r.registrants {
		if err := registrant.Register(s); err != nil {
			return fmt.Errorf("register %s: %w", registrantName(registrant), err)
		}
	}
	return nil
}

// registrantName returns the surface label for error attribution, falling back
// to the registrant's Go type.
func registrantName(r Registrant) string {
	if n, ok := r.(named); ok && n.name != "" {
		return n.name
	}
	return fmt.Sprintf("%T", r)
}
