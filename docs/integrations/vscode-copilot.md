# VS Code / GitHub Copilot Integration

> **Status:** Meshery MCP Server is currently under active development.

This guide will document how to connect Meshery MCP Server to VS Code and GitHub Copilot after the supported MCP configuration is finalized.

## Prerequisites

- A recent VS Code installation with the required MCP/Copilot functionality.
- GitHub Copilot access where required.
- A running Meshery instance.
- A working Meshery MCP Server installation.

## MCP Configuration

The final configuration format depends on the MCP support provided by the VS Code version in use.

The repository will document the verified configuration once the Meshery MCP Server command and transport are finalized.

Expected configuration concept:

    {
      "servers": {
        "meshery": {
          "command": "<meshery-mcp-server-command>",
          "args": []
        }
      }
    }

Do not copy this placeholder configuration into a production setup until it has been verified against the implementation.

## Meshery Configuration

Configure the Meshery server connection according to the [Configuration Guide](../configuration.md).

## Verify the Connection

After configuring the MCP server:

1. Restart or reload VS Code if required.
2. Open the MCP/Copilot tools interface.
3. Confirm that Meshery MCP Server is available.
4. Invoke an available Meshery tool.

## Troubleshooting

If the server is unavailable:

- Check the configured command.
- Check the server logs.
- Verify the Meshery URL.
- Verify authentication settings.
- Confirm the MCP configuration format supported by your VS Code version.
