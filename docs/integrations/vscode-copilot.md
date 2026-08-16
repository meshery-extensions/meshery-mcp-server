# VS Code / GitHub Copilot Integration

> **Status:** VS Code / GitHub Copilot integration is not yet verified for the current development implementation.

The repository currently provides a Go-based development server with stdio support. A tested VS Code / GitHub Copilot configuration was not established from the implementation and CI reviewed for this documentation.

## Current Development Server

The repository entrypoint is:

    cmd/meshery-mcp-server/main.go

Run it through:

    make run

Do not use an unverified VS Code / GitHub Copilot configuration as a production setup.

## Prerequisites

- VS Code
- GitHub Copilot access where required
- Meshery MCP Server built from this repository
- A running Meshery instance when testing Meshery integration

See the [Installation Guide](../installation.md).

## Configuration

Configure Meshery connection and authentication according to the [Configuration Guide](../configuration.md).

Never commit real API tokens.

## VS Code / GitHub Copilot Support

A complete, tested configuration requires a verified MCP protocol request/response flow and client configuration.

Those details are not established by the current implementation evidence reviewed for this documentation. Therefore this guide intentionally does not provide a fake command, transport, environment-variable list, or unverified JSON configuration.

## Verification

When VS Code / GitHub Copilot support is implemented and tested, this guide should document:

1. The exact executable or command.
2. The required transport.
3. The exact VS Code configuration.
4. Required environment variables.
5. A verified Meshery resource or tool invocation.
6. Expected output.

## Troubleshooting

For the current development server:

- Verify Go with `go version`.
- Run `make build`.
- Run `make run`.
- Check stderr for server errors.
- Verify Meshery configuration separately using the [Configuration Guide](../configuration.md).

## Related Documentation

- [Installation](../installation.md)
- [Configuration](../configuration.md)
- [Tools and Resources Reference](../tools-reference.md)
- [Development Guide](../development.md)
