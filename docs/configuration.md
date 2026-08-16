# Configuration

> **Status:** Meshery MCP Server is currently under active development.

This document describes the configuration options planned for Meshery MCP Server. Configuration details should be kept in sync with the server implementation.

## Meshery Server URL

The Meshery server URL identifies the Meshery instance that the MCP server communicates with.

Environment variable:

`MESHERY_SERVER_URL`

Default:

`http://localhost:9081`

Example:

    MESHERY_SERVER_URL=http://localhost:9081

On PowerShell:

    $env:MESHERY_SERVER_URL="http://localhost:9081"

## Meshery API Token

Authentication can be configured with the Meshery provider/API token.

Environment variable:

`MESHERY_API_TOKEN`

Example:

    MESHERY_API_TOKEN=your-token

Do not commit tokens or other credentials to source control.

## Configuration File

A configuration-file based setup is planned for the server.

The supported file path, format, fields, and precedence rules should be documented here once the configuration implementation is finalized.

## Command-Line Configuration

Command-line flags will be documented here when they are implemented.

## Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `MESHERY_SERVER_URL` | URL of the Meshery server | `http://localhost:9081` |
| `MESHERY_API_TOKEN` | Meshery provider/API token | Not set |

## Configuration Precedence

The final configuration precedence will be documented when the implementation is finalized.

## Security

- Never commit API tokens to Git.
- Do not place credentials in public documentation examples.
- Prefer environment variables or a local configuration file for development credentials.
- Use appropriate secret-management facilities for production deployments.

## Related Documentation

- [Installation](installation.md)
- [Tools Reference](tools-reference.md)
- [Development Guide](development.md)
