# Cursor Integration

> **Status:** Cursor integration is not yet verified for the current development implementation.

The repository currently provides a Go-based development server with stdio support. A tested Cursor configuration was not established from the implementation and CI reviewed for this documentation.

## Current Development Server

The repository entrypoint is:

    cmd/meshery-mcp-server/main.go

Run it through:

    make run

Do not use an unverified Cursor configuration as a production setup.

## Prerequisites

- Cursor
- Meshery MCP Server built from this repository
- A running Meshery instance when testing Meshery integration

See the [Installation Guide](../installation.md).

## Configuration

Configure Meshery connection and authentication according to the [Configuration Guide](../configuration.md).

Never commit real API tokens.

## Cursor Support

Cursor integration is currently deferred.

The repository provides a development stdio server through:

    make run

However, a tested Cursor configuration, required environment-variable set, and verified Cursor MCP request/response flow have not been established in the current repository implementation and CI.

Therefore, this documentation does not provide an unverified Cursor configuration.

The Cursor integration acceptance criterion remains deferred until the following are verified:

1. Exact stdio transport configuration.
2. Exact Cursor MCP configuration.
3. Required environment variables.
4. Successful server startup from Cursor.
5. A verified MCP request/response or resource/tool invocation.

## Verification

When Cursor support is implemented and tested, this guide should document:

1. The exact executable or command.
2. The required transport.
3. The exact Cursor configuration.
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
