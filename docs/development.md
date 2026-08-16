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
- MCP resources
- MCP prompts
- Tests
- Documentation

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
