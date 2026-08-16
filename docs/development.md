# Development Guide

## Repository

Clone the official repository:

    git clone https://github.com/meshery-extensions/meshery-mcp-server.git
    cd meshery-mcp-server

## Project Status

Meshery MCP Server is currently under active development.

The project is being developed incrementally, including:

- MCP server functionality
- Meshery API client functionality
- MCP tools
- MCP resources# Development Guide

Meshery MCP Server is currently under active development.

The current implementation is a Go-based development foundation with a stdio entrypoint and an initial `ping` operation.

## Repository Setup

Clone the repository:

    git clone https://github.com/meshery-extensions/meshery-mcp-server.git
    cd meshery-mcp-server

## Project Structure

The current MCP foundation includes:

    cmd/
      server/
        main.go

    internal/
      mcp/
        server.go
        server_test.go

The `cmd/server/` directory contains the executable entrypoint.

The `internal/mcp/` directory contains the server logic and its tests.

## Run the Server

Start the current development server with:

    go run cmd/server/main.go

The server reads JSON input from standard input and writes JSON responses to standard output.

For the current development ping operation, send:

    {"method":"ping"}

Expected response:

    {"result":"pong"}

## Format the Code

Format Go source files with:

    go fmt ./...

## Static Analysis

Run Go's vet checks with:

    go vet ./...

## Run Tests

Run the Go test suite with:

    go test ./...

The current MCP foundation includes an initial unit test for the server logic.

## Development Workflow

When implementing a change:

1. Create a focused branch.
2. Make the smallest change needed for the issue.
3. Add or update tests.
4. Format the Go code.
5. Run `go vet`.
6. Run `go test`.
7. Update documentation when behavior changes.
8. Review the Git diff before committing.
9. Open a pull request that references the related issue.

## Adding MCP Functionality

The project is being expanded from the initial stdio foundation toward full MCP functionality.

Future implementation areas include:

- MCP protocol support.
- Streamable HTTP transport.
- MCP tools.
- MCP resources.
- MCP prompts.
- Meshery API integration.
- Tests for client and server behavior.

When adding functionality, keep implementation, tests, and documentation synchronized.

## Testing Stdio Behavior

The current development server can be tested manually by starting:

    go run cmd/server/main.go

Then sending:

    {"method":"ping"}

Expected output:

    {"result":"pong"}

As the MCP protocol implementation evolves, tests should cover the actual protocol request/response path used by the executable.

## Validation Before a Pull Request

Run:

    go fmt ./...
    go vet ./...
    go test ./...

Also run:

    git diff --check

Review the output of:

    git status
    git diff

before creating the pull request.

## Documentation

Update the relevant documentation when changing user-visible behavior:

- [Installation](installation.md)
- [Configuration](configuration.md)
- [Tools Reference](tools-reference.md)

## Current Limitations

The current development foundation is not the completed Meshery MCP Server.

In particular, the initial stdio foundation does not yet represent the final MCP feature set. Streamable HTTP transport, complete MCP protocol behavior, and the complete Meshery tool/resource/prompt surface are part of the ongoing implementation.

Do not document development-only behavior as a production-ready feature.

## Community

Contributions should follow the repository's contribution guidelines and the CNCF Code of Conduct.

See:

- [Contributing Guide](../CONTRIBUTING.md)
- [Code of Conduct](../CODE_OF_CONDUCT.md)

## Prerequisites

For development, install:

- Git
- Go, when building the Go implementation
- A running Meshery instance when testing Meshery API integration

See the [Meshery Quick Start](https://docs.meshery.io/installation/quick-start/) for Meshery setup information.

## Building

Build instructions will be updated when the Go application structure and official build target are finalized.

Do not assume a build command that is not provided by the repository's current implementation.

## Testing

Tests should be added alongside implementation changes.

Before opening a pull request:

1. Run the repository's available tests.
2. Run formatting and linting checks.
3. Verify documentation examples.
4. Confirm that changes do not introduce unrelated failures.

The exact project test commands will be documented here once the implementation is available.

## Adding an MCP Tool

When the MCP tool architecture is finalized, a new tool should generally:

1. Define the tool's purpose.
2. Define its input schema.
3. Implement the handler.
4. Connect it to the appropriate Meshery API/client functionality.
5. Register the tool with the MCP server.
6. Add unit tests.
7. Document the tool in `tools-reference.md`.
8. Add a working example where appropriate.

## Adding an MCP Resource

A new MCP resource should:

1. Define its URI.
2. Define the returned data.
3. Implement the resource handler.
4. Register it with the server.
5. Add tests.
6. Document it.

## Adding an MCP Prompt

A new MCP prompt should:

1. Define its purpose.
2. Define its arguments.
3. Implement the prompt.
4. Register it with the server.
5. Add tests.
6. Document its usage.

## Pull Requests

Before opening a pull request:

1. Keep the change focused.
2. Add or update tests when applicable.
3. Update documentation.
4. Run available validation and linting commands.
5. Write a clear pull request description.
6. Reference the related GitHub issue.

## Code of Conduct

Contributors are expected to follow the project's [Code of Conduct](../CODE_OF_CONDUCT.md).
