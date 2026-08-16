# Development Guide

> **Status:** Meshery MCP Server is currently under active development.

The current implementation is a Go-based Meshery MCP Server with a stdio entrypoint.

## Repository Setup

Clone the repository:

    git clone https://github.com/meshery-extensions/meshery-mcp-server.git
    cd meshery-mcp-server

## Project Structure

The server executable is located at:

    cmd/meshery-mcp-server/main.go

The repository also contains internal packages for MCP, Meshery client, resources, and version information.

## Build

The repository Makefile provides the tested build target:

    make build

The binary is written to:

    bin/meshery-mcp-server

The CI workflow also verifies that the repository builds with:

    go build ./...

## Run

Run the server through the Makefile:

    make run

The `run` target builds the binary and starts the Meshery MCP Server over stdio.

## Test

Run the repository test target:

    make test

The test target runs:

    go test --short ./... -race -coverprofile=coverage.txt -covermode=atomic

The CI workflow uses the same test command.

## Vet

Run:

    make vet

The underlying command is:

    go vet ./...

## Formatting

Run:

    make fmt

The repository uses `golangci-lint fmt` for formatting.

## Linting

Run:

    make lint

The underlying command is:

    golangci-lint run --timeout=10m

## Docker

Build the development container image with:

    make docker-build

The Makefile tags the image as:

    meshery/meshery-mcp-server

The Dockerfile builds the `cmd/meshery-mcp-server` binary and uses it as the container entrypoint.

## Clean Build Artifacts

Run:

    make clean

This removes the `bin` directory and `coverage.txt`.

## CI Validation

The repository build-and-test workflow verifies:

    go build ./...
    go vet ./...
    go test --short ./... -race -coverprofile=coverage.txt -covermode=atomic

The workflow also runs golangci-lint.

A README Markdown linting command is not currently configured in the repository CI workflow. Therefore, README Markdown linting as a merge acceptance requirement is deferred until a corresponding CI check is added.

Do not document or claim an unconfigured Markdown linting command as a repository requirement.

## Development Workflow

When implementing a change:

1. Create a focused branch.
2. Make the smallest change needed for the issue.
3. Add or update tests.
4. Run formatting.
5. Run linting.
6. Run `make build`.
7. Run `make test`.
8. Run `make vet`.
9. Review the Git diff.
10. Update documentation when behavior changes.
11. Open a pull request that references the related issue.

## Current MCP Development Status

MCP functionality is under active development. Keep documentation aligned with the implementation actually present in the branch being documented.

Do not document a client integration, transport, tool, resource, or command as production-ready unless it has been implemented and verified.

## Documentation

Update the relevant documentation when behavior changes:

- [Installation](installation.md)
- [Configuration](configuration.md)
- [Tools and Resources Reference](tools-reference.md)

## Related Documentation

- [Installation](installation.md)
- [Configuration](configuration.md)
- [Tools and Resources Reference](tools-reference.md)
