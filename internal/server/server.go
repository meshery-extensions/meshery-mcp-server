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
	"github.com/mark3labs/mcp-go/server"

	"github.com/meshery-extensions/meshery-mcp-server/internal/tools"
	"github.com/meshery-extensions/meshery-mcp-server/internal/version"
)

// New creates an MCP server with all registered tools.
func New() *server.MCPServer {
	s := server.NewMCPServer(version.Name, version.Version)
	tools.Register(s)
	return s
}

// Serve runs the MCP server over stdio until a client disconnects or the
// process is interrupted.
func Serve(s *server.MCPServer) error {
	return server.ServeStdio(s)
}
