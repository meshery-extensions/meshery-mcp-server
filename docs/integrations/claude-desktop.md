# Claude Desktop integration

Connect Claude Desktop to a local Meshery instance through the Meshery MCP
Server.

## Prerequisites

- Meshery Server running locally (default `http://localhost:9081`)
- Meshery MCP Server built and on your `PATH` (see [docs/installation.md](docs/installation.md))
- A Meshery provider token (`MESHERY_API_TOKEN`)

## Configure Claude Desktop

Open Claude Desktop's configuration file:

| OS | Config file path |
|----|------------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `$XDG_CONFIG_HOME/Claude/claude_desktop_config.json`, or `~/.config/Claude/claude_desktop_config.json` |

Add the `meshery` server under `mcpServers`:

```json
{
  "mcpServers": {
    "meshery": {
      "command": "/absolute/path/to/meshery-mcp-server",
      "env": {
        "MESHERY_SERVER_URL": "http://localhost:9081",
        "MESHERY_API_TOKEN": "<your-meshery-token>"
      }
    }
  }
}
```

- `command` must be the absolute path to the built `meshery-mcp-server`
  binary (e.g. `$HOME/go/bin/meshery-mcp-server` or
  `/path/to/meshery-mcp-server/bin/meshery-mcp-server`).
- `MESHERY_SERVER_URL` points at your Meshery Server (default
  `http://localhost:9081`).
- `MESHERY_API_TOKEN` is your Meshery provider token. Obtain it from Meshery
  (UI or CLI), never hard-code it in a shared file.

Save the file and restart Claude Desktop. Claude Desktop starts the server as a
child process over stdio.

## Verify the connection

1. Open a chat in Claude Desktop.
2. Ask Claude to list available tools: *"What tools do you have access to?"*
3. You should see the Meshery MCP Server's tools once they are implemented
   (currently none are registered — see
   [docs/tools-reference.md](docs/tools-reference.md)).

## Troubleshooting

| Problem | Likely cause | Fix |
|---------|--------------|-----|
| Server doesn't appear | Wrong `command` path | Use an absolute path; confirm the binary is executable |
| `token_configured=false` | No token | Set `MESHERY_API_TOKEN` in the config or run with the env var |
| Server exits immediately | Meshery unreachable | Confirm Meshery Server is running at `MESHERY_SERVER_URL` |
| Tools missing | Not yet implemented | Zero tools are registered on master — see [docs/tools-reference.md](docs/tools-reference.md) |

Because `stdout` carries the MCP protocol, the server logs to `stderr`. Run the
binary directly to see startup errors:

```sh
MESHERY_SERVER_URL=http://localhost:9081 MESHERY_API_TOKEN=<token> meshery-mcp-server
```
