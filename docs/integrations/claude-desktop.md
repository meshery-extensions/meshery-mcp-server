# Claude Desktop Integration

> **Status:** Meshery MCP Server is currently under active development.

This guide will document how to connect Claude Desktop to Meshery MCP Server once the server implementation and supported transport are finalized.

## Prerequisites

- Claude Desktop
- A running Meshery instance
- A working Meshery MCP Server installation

See the [Installation Guide](../installation.md) for project setup.

## MCP Server Configuration

Claude Desktop uses an MCP server configuration to start or connect to MCP servers.

The final Meshery MCP Server configuration will be documented here after the supported command and transport are finalized.

Expected structure:

    {
      "mcpServers": {
        "meshery": {
          "command": "<meshery-mcp-server-command>",
          "args": []
        }
      }
    }

Replace the placeholder command with the actual published Meshery MCP Server command.

## Configuration

Meshery connection settings should be configured according to the [Configuration Guide](../configuration.md).

Do not place real API tokens in documentation or source control.

## Verify the Connection

After configuring the client:

1. Restart Claude Desktop.
2. Open the MCP/tools interface.
3. Confirm that the Meshery server is available.
4. Test an available Meshery MCP tool.

## Troubleshooting

If the server cannot be discovered:

- Confirm the executable path.
- Confirm the Meshery server is reachable.
- Check the configured environment variables.
- Check the server logs.
- Verify that the MCP transport configured by the client matches the server implementation.

The final troubleshooting commands will be added after the implementation is finalized.
