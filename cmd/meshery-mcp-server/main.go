// Command meshery-mcp-server runs the Meshery MCP server over stdio.
package main

import (
	"log"
	"os"

	"github.com/meshery-extensions/meshery-mcp-server/internal/server"
)

// main wires up and serves the Meshery MCP server over stdio.
func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.SetOutput(os.Stderr)

	srv, err := server.New()
	if err != nil {
		log.Fatalf("create MCP server: %v", err)
	}
	if err := server.Serve(srv); err != nil {
		log.Fatalf("serve MCP server: %v", err)
	}
}
