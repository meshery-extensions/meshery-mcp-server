# Cline Integration

> **Status:** Meshery MCP Server is currently under active development.

This guide will document how to connect Cline to Meshery MCP Server once the server command, transport, and Cline configuration have been finalized and tested.

## Prerequisites

- VS Code with Cline installed.
- A running Meshery instance.
- A working Meshery MCP Server installation.

## MCP Configuration

Cline supports connecting external MCP servers through its MCP configuration.

The final Meshery MCP Server configuration will be added here after the implementation is finalized.

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

After configuring Cline:

1. Restart or reload the client if required.
2. Open the MCP server/tool interface.
3. Confirm that Meshery MCP Server is available.
4. Test an available Meshery MCP tool.

## Troubleshooting

If Cline cannot connect:

- Verify the configured command.
- Check the Meshery server URL.
- Verify authentication.
- Check the MCP server logs.
- Confirm that the configuration format matches your installed Cline version.
