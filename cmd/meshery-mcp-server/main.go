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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/server"
	"github.com/meshery-extensions/meshery-mcp-server/internal/version"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetOutput(os.Stderr)

	transport := flag.String("transport", "", "MCP transport: stdio, http, or sse")
	port := flag.Int("port", 0, "Port for http/sse transport")
	flag.Parse()

	cfg := config.Load()
	if *transport != "" {
		cfg.Transport = *transport
	}
	if *port != 0 {
		cfg.HTTPAddr = fmt.Sprintf("0.0.0.0:%d", *port)
	}
	log.Printf("starting %s %s (commit %s, Meshery Server: %s, transport: %s, addr: %s)", version.Name, version.Version, version.CommitSHA, cfg.RedactedURL(), cfg.Transport, cfg.HTTPAddr)

	srv, err := server.New()
	if err != nil {
		log.Fatalf("create MCP server: %v", err)
	}

	if err := server.Serve(srv, cfg); err != nil {
		log.Fatalf("serve MCP server: %v", err)
	}
}
