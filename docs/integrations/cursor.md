# Cursor integration

Connect Cursor to a local Meshery instance through the Meshery MCP Server.

> Source: [MCP — Cursor Docs](https://cursor.com/docs/mcp)

## Prerequisites

- Meshery Server running locally (default `http://localhost:9081`)
- Meshery MCP Server built and on your `PATH` (see [docs/installation.md](docs/installation.md))
- A Meshery provider token (`MESHERY_API_TOKEN`)

## Configure Cursor

Cursor reads MCP servers from an `mcp.json` file:

| Scope | Config file path |
|-------|------------------|
| Global (all projects) | `~/.cursor/mcp.json` |
| Workspace (one project) | `.cursor/mcp.json` in the project root |

Add the `meshery` server under `mcpServers`:

```json
{
  "mcpServers": {
    "meshery": {
      "type": "stdio",
      "command": "/absolute/path/to/meshery-mcp-server",
      "args": [],
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
- `env` supports interpolation (`${env:NAME}`, `${userHome}`,
  `${workspaceFolder}`). Do not commit tokens.

Save the file. Cursor picks up MCP servers from **Customize** in the sidebar;
toggle the `meshery` server on if needed.

## Verify the connection

1. Open Chat in Cursor.
2. Ask Cursor to list available tools.
3. You should see the Meshery MCP Server's tools once they are implemented
   (currently none are registered — see
   [docs/tools-reference.md](docs/tools-reference.md)).

## Troubleshooting

| Problem | Likely cause | Fix |
|---------|--------------|-----|
| Server doesn't appear | Wrong config file or `mcpServers` key | Use `~/.cursor/mcp.json` or `.cursor/mcp.json`; confirm top-level `mcpServers` |
| Token not picked up | Env var not set | Set `MESHERY_API_TOKEN` in your shell or inline it in `env` |
| `token_configured=false` | No token | Provide `MESHERY_API_TOKEN` |
| Tools missing | Not yet implemented | Zero tools are registered on master — see [docs/tools-reference.md](docs/tools-reference.md) |

The server logs to `stderr`; run it directly to see startup errors:

```sh
MESHERY_SERVER_URL=http://localhost:9081 MESHERY_API_TOKEN=<token> meshery-mcp-server
