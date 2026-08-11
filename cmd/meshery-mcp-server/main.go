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
	"log"
	"os"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/server"
	"github.com/meshery-extensions/meshery-mcp-server/internal/version"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetOutput(os.Stderr)

	// Initialize configuration manager with multi-context support
	mgr := config.NewManager()
	if err := mgr.Load(); err != nil {
		log.Printf("warning: %v", err)
		log.Print(config.SetupPrompt())
	}

	// Get active configuration for logging
	cfg := mgr.GetConfig()
	ctx, ctxName, _ := mgr.GetCurrentContext()

	if mgr.IsFromFile() {
		log.Printf("starting %s %s (commit %s)", version.Name, version.Version, version.CommitSHA)
		log.Printf("  context: %s, server: %s, token: %s", ctxName, cfg.RedactedURL(), config.MaskToken(ctx.Token))
	} else {
		log.Printf("starting %s %s (commit %s, Meshery Server: %s)", version.Name, version.Version, version.CommitSHA, cfg.RedactedURL())
		log.Print("  (using environment variables; create ~/.meshery/mcp-config.yaml for multi-context support)")
	}

	srv := server.New(mgr)
	if err := server.Serve(srv); err != nil {
		log.Fatalf("serve MCP server: %v", err)
	}
}
