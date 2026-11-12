# VS Code + GitHub Copilot integration

Connect VS Code's GitHub Copilot to a local Meshery instance through the
Meshery MCP Server.

> Source: [MCP configuration reference — VS Code](https://code.visualstudio.com/docs/agents/reference/mcp-configuration)

## Prerequisites

- Meshery Server running locally (default `http://localhost:9081`)
- Meshery MCP Server built and on your `PATH` (see [docs/installation.md](docs/installation.md))
- A Meshery provider token (`MESHERY_API_TOKEN`)
- VS Code with GitHub Copilot enabled

## Configure Copilot

Use the **native Copilot MCP configuration** (not the deprecated editor-managed
`.vscode/mcp.json`). Choose one of these destinations:

| Goal | Configuration file |
|------|--------------------|
| Use across your workspaces | `$COPILOT_HOME/mcp-config.json`, or `~/.copilot/mcp-config.json` when `COPILOT_HOME` is unset |
| Share with your team | `.mcp.json` at the workspace root |

Add the `meshery` server under `mcpServers`:

```json
{
  "mcpServers": {
    "meshery": {
      "type": "stdio",
      "command": "/absolute/path/to/meshery-mcp-server",
      "env": {
        "MESHERY_SERVER_URL": "http://localhost:9081",
        "MESHERY_API_TOKEN": "${env:MESHERY_API_TOKEN}"
      }
    }
  }
}
```

- `type: "stdio"` — the server speaks MCP over stdin/stdout.
- `command` must be an absolute path to the built `meshery-mcp-server`
  binary.
- `env` references are resolved in the host environment; do not commit tokens.
  You can also inline the value: `"MESHERY_API_TOKEN": "<your-token>"`.

Save the file, then in VS Code run **Chat: Open Customizations** → **MCP
Servers** to confirm the server is listed, and verify a tool call.

## Verify the connection

1. Open a Copilot chat in VS Code.
2. Ask Copilot to list available tools.
3. You should see the Meshery MCP Server's tools once they are implemented
   (currently none are registered — see
   [docs/tools-reference.md](docs/tools-reference.md)).

## Troubleshooting

| Problem | Likely cause | Fix |
|---------|--------------|-----|
| Server doesn't appear | Wrong config file or path | Use `~/.copilot/mcp-config.json` or workspace `.mcp.json`; confirm absolute `command` path |
| Token not picked up | Env var not set | Set `MESHERY_API_TOKEN` in your shell or inline it in `env` |
| `token_configured=false` | No token | Provide `MESHERY_API_TOKEN` |
| Tools missing | Not yet implemented | Zero tools are registered on master — see [docs/tools-reference.md](docs/tools-reference.md) |

The server logs to `stderr`; run it directly to see startup errors:

```sh
MESHERY_SERVER_URL=http://localhost:9081 MESHERY_API_TOKEN=<token> meshery-mcp-server
