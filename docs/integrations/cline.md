# Cline Integration

> **Status:** Meshery MCP Server is currently under active development.

The current Meshery MCP Server implementation provides a development stdio foundation. A complete, verified Cline integration is not yet documented because the final MCP protocol support and client configuration are still under development.

## Current Status

The current development server can be started with:

    go run cmd/server/main.go

The current foundation reads JSON from standard input and writes JSON to standard output.

The current development ping request is:

    {"method":"ping"}

Expected response:

    {"result":"pong"}

This development interface should not be treated as the final Cline MCP integration.

## Prerequisites

- VS Code with Cline installed
- A working Go development environment
- A running Meshery instance when testing Meshery integration
- A local checkout of Meshery MCP Server

See the [Installation Guide](../installation.md).

## Configuration

Meshery connection and authentication settings are documented in the [Configuration Guide](../configuration.md).

Never commit real credentials.

## Cline Integration

A verified Cline configuration will be documented after the server's MCP protocol support and supported transport are finalized.

At the current development stage, this repository does not provide a production-ready Cline configuration command or configuration snippet.

Do not use an unverified placeholder configuration as a production setup.

## Verification

When Cline support is finalized, this guide should document:

1. The exact server command.
2. The supported MCP transport.
3. The required Cline configuration.
4. Required environment variables.
5. A verified tool or resource invocation.
6. Expected output.

## Troubleshooting

For the current development foundation:

- Verify that Go is installed with `go version`.
- Run the server from the repository root.
- Verify that `go run cmd/server/main.go` starts successfully.
- Verify the development ping request and response.
- Check server output for errors.

## Related Documentation

- [Installation](../installation.md)
- [Configuration](../configuration.md)
- [Tools and Resources Reference](../tools-reference.md)
- [Development Guide](../development.md)
