# Cline Integration

> **Status:** Cline integration is not yet verified for the current development implementation.

The repository currently provides a Go-based development server with stdio support. A tested Cline configuration was not established from the implementation and CI reviewed for this documentation.

## Current Development Server

The repository entrypoint is:

    cmd/meshery-mcp-server/main.go

Run it through:

    make run

The server uses stdio for its development interface. Do not use an unverified Cline configuration as a production setup.

## Prerequisites

- VS Code with Cline installed
- Meshery MCP Server built from this repository
- A running Meshery instance when testing Meshery integration

See the [Installation Guide](../installation.md).

## Configuration

Configure Meshery connection and authentication according to the [Configuration Guide](../configuration.md).

Never commit real API tokens.

## Cline Support

A complete, tested Cline configuration requires a verified MCP protocol request/response flow and client configuration.

Those details are not established by the current implementation evidence reviewed for this documentation. Therefore this guide intentionally does not provide a fake command, transport, environment-variable list, or unverified JSON configuration.

## Verification

When Cline support is implemented and tested, this guide should document:

1. The exact executable or command.
2. The required transport.
3. The exact Cline configuration.
4. Required environment variables.
5. A verified Meshery resource or tool invocation.
6. Expected output.

## Troubleshooting

For the current development server:

- Verify Go with `go version`.
- Run `make build`.
- Run `make run`.
- Check **stderr** for server errors.
- Keep stdout reserved for the server's protocol/data output.
- Verify Meshery configuration separately using the [Configuration Guide](../configuration.md).

## Related Documentation

- [Installation](../installation.md)
- [Configuration](../configuration.md)
- [Tools and Resources Reference](../tools-reference.md)
- [Development Guide](../development.md)
