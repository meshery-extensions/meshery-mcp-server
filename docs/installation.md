# Installation

> **Status:** Meshery MCP Server is currently under active development.

This guide documents installation methods supported by the current development repository.

## Prerequisites

For building from source, install:

- Git
- Go 1.26.0 or a compatible Go release supported by the repository

Verify Go:

    go version

The repository's `go.mod` declares Go 1.26.0.

## Clone the Repository

Clone the official repository:

    git clone https://github.com/meshery-extensions/meshery-mcp-server.git
    cd meshery-mcp-server

## Build from Source

The repository provides a Makefile build target:

    make build

This builds the Meshery MCP Server executable at:

    bin/meshery-mcp-server

The server entrypoint used by the build is:

    ./cmd/meshery-mcp-server

The CI workflow separately validates that the repository builds successfully with:

    go build ./...

The `go build ./...` command validates the Go packages; it is not equivalent to the `make build` target.

## Verify the Built Server

After running:

    make build

verify that the executable exists:

PowerShell:

    Test-Path .\bin\meshery-mcp-server.exe

Linux/macOS:

    test -f ./bin/meshery-mcp-server

## Run from Source

Run the development server with:

    make run

The Makefile builds the binary and starts it over stdio.

## Docker

The repository includes a Dockerfile and a Makefile target for building the container image.

Build the image with:

    make docker-build

The image is tagged:

    meshery/meshery-mcp-server

The Dockerfile builds the server from:

    ./cmd/meshery-mcp-server

and uses:

    /meshery-mcp-server

as the container entrypoint.

A verified Docker stdio invocation is not currently established in the repository documentation or CI workflow. Therefore, Docker stdio client integration is deferred until a tested invocation and verification flow are added.

For now, the supported documented Docker operation is building the image locally with:

    make docker-build

## Pre-built Binaries

The repository contains release/build configuration, but this guide does not claim a public pre-built binary download URL because a verified release artifact location was not established from the implementation reviewed for this documentation.

For a guaranteed reproducible local installation, build from source with:

    make build

## Homebrew

A Homebrew formula was not identified in the reviewed Meshery MCP Server implementation.

Therefore, Homebrew installation is currently **not documented as a supported installation method**.

Use the source build or Docker method described above.

## Server Version

The build embeds the Git version and commit SHA into the server binary.

The Makefile derives these values during the build.

A verified end-user `--version` command is not currently established in the repository documentation.

To identify the source revision used for a development build, run:

    git describe --tags --always --dirty

and:

    git rev-parse --short HEAD

## Configuration

Meshery connection and authentication settings are documented in the [Configuration Guide](configuration.md).

## Verify the Installation

Build and validate the server:

    make build
    make test
    make vet

Then run:

    make run

For Docker:

    make docker-build

## Troubleshooting

### Go is not installed

Install Go and verify:

    go version

### Build fails

Run from the repository root and verify dependencies:

    go mod download

Then:

    make build

### Tests fail

Run:

    make test

### Linting fails

Run:

    make lint

## Related Documentation

- [Configuration](configuration.md)
- [Tools and Resources Reference](tools-reference.md)
- [Development Guide](development.md)
