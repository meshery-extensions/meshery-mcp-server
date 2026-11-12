# Cline integration

Connect Cline to a local Meshery instance through the Meshery MCP Server.

> Source: [MCP — Cline Docs](https://docs.cline.bot/mcp/mcp-overview)

## Prerequisites

- Meshery Server running locally (default `http://localhost:9081`)
- Meshery MCP Server built and on your `PATH` (see [docs/installation.md](docs/installation.md))
- A Meshery provider token (`MESHERY_API_TOKEN`)

## Configure Cline

Cline reads MCP servers from its settings JSON:

| Interface | Config file path |
|-----------|------------------|
| Cline CLI | `~/.cline/data/settings/cline_mcp_settings.json` (overridable with `CLINE_MCP_SETTINGS_PATH`) |
| Cline IDE extension | Open the MCP panel → **Configure** → **Configure MCP Servers** |

Add the `meshery` server under `mcpServers`:

```json
{
  "mcpServers": {
    "meshery": {
      "command": "/absolute/path/to/meshery-mcp-server",
      "args": [],
      "env": {
        "MESHERY_SERVER_URL": "http://localhost:9081",
        "MESHERY_API_TOKEN": "<your-meshery-token>"
      },
      "disabled": false,
      "autoApprove": []
    }
  }
}
```

- `command` must be an absolute path to the built `meshery-mcp-server`
  binary.
- `MESHERY_API_TOKEN` is your Meshery provider token. Do not commit it in a
  shared file.
- `disabled: false` enables the server; `autoApprove: []` means no tools are
  auto-approved (each tool call requires approval).

Using the Cline CLI, you can also add the server without editing JSON:

```sh
cline mcp add meshery --yes -- /absolute/path/to/meshery-mcp-server
```

Then set `MESHERY_SERVER_URL` and `MESHERY_API_TOKEN` in the server's env.

## Verify the connection

1. Open Cline and go to the MCP Servers panel.
2. Confirm `meshery` is listed and enabled.
3. Ask Cline to list available tools.
4. You should see the Meshery MCP Server's tools once they are implemented
   (currently none are registered — see
   [docs/tools-reference.md](docs/tools-reference.md)).

## Troubleshooting

| Problem | Likely cause | Fix |
|---------|--------------|-----|
| Server doesn't appear | Wrong config file or `mcpServers` key | Use `cline_mcp_settings.json`; confirm top-level `mcpServers` |
| Token not picked up | Env var not set | Set `MESHERY_API_TOKEN` in `env` or your shell |
| `token_configured=false` | No token | Provide `MESHERY_API_TOKEN` |
| Tools missing | Not yet implemented | Zero tools are registered on master — see [docs/tools-reference.md](docs/tools-reference.md) |

The server logs to `stderr`; run it directly to see startup errors:

```sh
MESHERY_SERVER_URL=http://localhost:9081 MESHERY_API_TOKEN=<token> meshery-mcp-server
