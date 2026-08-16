# Installation

> **Status:** Meshery MCP Server is currently under active development.

Meshery MCP Server is being developed as an official Meshery extension that provides an MCP interface for Meshery.

## Prerequisites

Before running the current development server, install:

- Git
- Go
- A running Meshery instance when testing Meshery integration

The current MCP server foundation is implemented as a Go application and provides a stdio entrypoint under `cmd/server/`.

## Clone the Repository

Clone the official repository:

    git clone https://github.com/meshery-extensions/meshery-mcp-server.git
    cd meshery-mcp-server

## Run the Development Server

The current development implementation can be started with:

    go run cmd/server/main.go

The server reads JSON input from standard input and writes JSON responses to standard output.

## Test the Ping Tool

The current development foundation includes a simple `ping` operation.

Start the server:

    go run cmd/server/main.go

Then provide:

    {"method":"ping"}

The expected response from the current foundation is:

    {"result":"pong"}

This is a development bootstrap and is not yet a complete MCP client integration.

## Current Transport

The current foundation implements communication over standard input and standard output (stdio).

Streamable HTTP transport is planned as future work.

## Current Development Status

The current implementation is a minimal foundation. It includes:

- A Go module.
- A `cmd/server/` entrypoint.
- Internal MCP server logic.
- Stdio input/output handling.
- A basic `ping` operation.
- Initial unit tests.

Full MCP protocol support, additional tools, resources, prompts, and production-ready client integrations are still under development.

## Configuration

Meshery connection and authentication settings are documented in the [Configuration Guide](configuration.md).

## Troubleshooting

### Go command is not available

Verify that Go is installed:

    go version

### Server does not start

Run the command from the repository root:

    go run cmd/server/main.go

### Ping does not return the expected response

Make sure the input is valid JSON and uses the current development request format:

    {"method":"ping"}

### MCP client cannot connect

The current development foundation uses stdio and is not yet a finalized MCP client integration. Verify that the client configuration matches the server transport and implementation available in your checkout.

## Next Steps

After running the development server:

1. Review the [Configuration Guide](configuration.md).
2. Review the [Tools Reference](tools-reference.md).
3. Review the [Development Guide](development.md).
4. Follow the project's ongoing implementation work for complete MCP support.
