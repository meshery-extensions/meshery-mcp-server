# Cursor Integration

> **Status:** Meshery MCP Server is currently under active development.

This guide will document how to connect Cursor to Meshery MCP Server once the server command, transport, and supported Cursor configuration have been finalized and tested.

## Prerequisites

- Cursor
- A running Meshery instance
- A working Meshery MCP Server installation

## MCP Configuration

Cursor provides MCP server configuration for connecting external MCP servers.

The final Meshery configuration will be documented here after the implementation is finalized.

Expected configuration concept:

    {
      "mcpServers": {
        "meshery": {
          "command": "<meshery-mcp-server-command>",
          "args": []
        }
      }
    }

Replace the placeholder command with the actual Meshery MCP Server command.

## Meshery Configuration

Configure the Meshery connection according to the [Configuration Guide](../configuration.md).

Never commit real credentials.

## Verify the Connection

After adding the server:

1. Restart Cursor if required.
2. Open the MCP tools interface.
3. Confirm that Meshery MCP Server is available.
4. Test an available Meshery MCP tool.

## Troubleshooting

If the connection fails:

- Verify the server command.
- Verify the Meshery server URL.
- Verify authentication.
- Check the MCP server logs.
- Confirm the configuration format supported by your Cursor version.
