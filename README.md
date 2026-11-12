<div>
<img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-github.png" alt="Meshery Extensions" />
</div>
<div align="center">
<h3>Meshery is an extensible, self-service engineering platform for the collaborative management of cloud and cloud native infrastructure.</h3>
<h3><a href="https://meshery.io/extensions">Browse all extensions</a></h3>
<h5>
<a href="https://meshery.io#getting-started">Installation</a> |
<a href="https://docs.meshery.io">Documentation</a> |
<a href="https://discuss.meshery.io">Forum</a> |
<a href="https://play.meshery.io">Playground</a> |
<a href="https://meshery.io/catalog">Catalog</a>
</h5>
<a href="https://github.com/meshery/meshery/blob/master/GOVERNANCE.md#extensions-githubcommeshery-extensions"><img src="https://img.shields.io/badge/support-community-00B39F?style=flat-square&logo=meshery&logoColor=white" alt="Level of support for this repo" /></a>
<a href="https://github.com/meshery-extensions/meshery-mcp-server/actions/workflows/build-and-test.yml"><img src="https://img.shields.io/github/actions/workflow/status/meshery-extensions/meshery-mcp-server/build-and-test.yml?label=build%20and%20test" alt="Build and Test" /></a>
<a href="https://github.com/meshery-extensions/meshery-mcp-server/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/meshery-extensions/meshery-mcp-server" alt="Go version" /></a>
<a href="https://modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-server-000000" alt="Model Context Protocol" /></a>
<a href="https://slack.meshery.io"><img src="https://img.shields.io/badge/Slack-join%20the%20community-00B39F?logo=slack&logoColor=white" alt="Slack" /></a>
</div>

# Meshery MCP Server

Meshery MCP Server connects AI assistants to [Meshery](https://meshery.io)
through the [Model Context Protocol](https://modelcontextprotocol.io) (MCP), an
open standard for connecting AI applications to external systems. Any
MCP-capable client, such as Claude Desktop, Claude Code, or an IDE agent, can
then work with Meshery designs through one consistent interface instead of a
custom integration per assistant.

It is an official [Meshery Extension](https://meshery.io/extensions), written
in Go, and runs as a standalone binary or container next to your Meshery
deployment.

## Architecture

```mermaid
flowchart LR
    Client["MCP client<br/>Claude Desktop · Claude Code · Cursor · VS Code Copilot"]
    Server["meshery-mcp-server<br/>(stdio · JSON-RPC)"]
    Meshery["Meshery Server<br/>MESHERY_SERVER_URL:9081"]
    K8s["Kubernetes<br/>clusters · designs · policies"]

    Client -->|JSON-RPC over stdin/stdout| Server
    Server -->|HTTP(S) + provider token| Meshery
    Meshery -->|manages| K8s
```

The MCP client launches the server as a child process and talks to it over
stdin/stdout, so the server does not listen on any network address. Because
stdout carries the protocol, all logs go to stderr.

Only `cmd/meshery-mcp-server` wires the packages together. `internal/config` is
the only package that reads environment variables or the config file; the other
packages receive plain values from `main`. See
[docs/config.md](docs/config.md#for-contributors) for details.

## Project status

> [!NOTE]
> **Early development.** The project scaffold is in place, but the MCP server
> itself is not implemented yet: the binary loads and validates its
> configuration, logs a startup line, and exits.

| Area | Status |
|------|--------|
| Configuration (env vars, config file), logging, Meshery-style errors | ✅ Done |
| Build, lint, tests, container image, CI | ✅ Done |
| MCP server over stdio | Planned |
| MCP server over Streamable HTTP | Planned |
| Meshery API client | Planned |
| MCP tools, resources and prompts | Planned |

### Planned scope

In priority order, the server will support:

1. Creating Meshery designs.
2. Retrieving designs in the format the user asks for (for example JSON or YAML).
3. Snapshotting Meshery designs.
4. Dry-run results for design deployments.
5. Defining and running performance tests, and retrieving their results.

These will be exposed through MCP's three building blocks:

- **Tools**: Meshery operations the AI can invoke on the user's behalf.
- **Resources**: Meshery data the AI can read as context.
- **Prompts**: reusable prompt templates for common Meshery workflows.

See [docs/adr/0001-mcp-sdk-choice.md](docs/adr/0001-mcp-sdk-choice.md) for the
MCP SDK and protocol-version decision.

## Quick start (10 minutes)

### 1. Install

**From source (Go):**

```sh
git clone https://github.com/meshery-extensions/meshery-mcp-server.git
cd meshery-mcp-server
make build              # writes bin/meshery-mcp-server
```

Or install with `go install`:

```sh
go install github.com/meshery-extensions/meshery-mcp-server@latest
```

**With Docker:**

```sh
make docker-build
make docker-run
```

### 2. Configure

Set the Meshery server URL and your provider token:

```sh
export MESHERY_SERVER_URL=http://localhost:9081
export MESHERY_API_TOKEN=<your-meshery-token>
```

Or create a config file:

```sh
cp config_example.yml config.local.yaml
meshery-mcp-server --config config.local.yaml
```

See [docs/configuration.md](docs/configuration.md) for the full reference.

### 3. Run

```sh
make run
# time="..." level=info msg="starting meshery-mcp-server ... token_configured=false" app=meshery-mcp-server
```

### 4. Connect Claude Desktop

Open Claude Desktop's configuration file:

| OS | Config file path |
|----|------------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| Linux | `$XDG_CONFIG_HOME/Claude/claude_desktop_config.json`, or `~/.config/Claude/claude_desktop_config.json` |

Add the `meshery` server:

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

Use the absolute path to the built binary. Restart Claude Desktop, then ask
*"What tools do you have access to?"* in a chat. See
[docs/integrations/claude-desktop.md](docs/integrations/claude-desktop.md) for
the full walkthrough, plus guides for
[VS Code Copilot](docs/integrations/vscode-copilot.md),
[Cursor](docs/integrations/cursor.md), and
[Cline](docs/integrations/cline.md).

## Full documentation

| Document | What it covers |
|----------|----------------|
| [docs/installation.md](docs/installation.md) | Install from source, Go, or Docker |
| [docs/configuration.md](docs/configuration.md) | Full config schema, env vars, Docker notes |
| [docs/tools-reference.md](docs/tools-reference.md) | MCP tools registry (planned) |
| [docs/development.md](docs/development.md) | Build, test, contribute |
| [docs/integrations/](docs/integrations/) | Claude Desktop, VS Code Copilot, Cursor, Cline |

## Development

| Command | Description |
|---------|-------------|
| `make build` | Build the binary into `bin/` |
| `make run` | Build and run; pass flags with `ARGS="..."` |
| `make test` | Run unit tests |
| `make coverage` | Run tests with the race detector |
| `make lint` | Run golangci-lint |
| `make fmt` | Format with gofmt |
| `make docs-lint` | Lint Markdown with markdownlint |
| `make docker-build` | Build the container image |
| `make docker-run` | Run the image over stdio |

Errors follow Meshery's
[error conventions](https://docs.meshery.io/project/contributing/contributing-error).
Every pull request runs lint, build/test, and a Docker smoke test in CI.

## Community and Contributing

Please do! Code and non-code contributions are welcome. This project is
community-built and fosters collaboration. Contributors are expected to adhere
to the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).

- [Contributor Guides](https://docs.meshery.io/project/contributing)
- [Community Handbook](https://layer5.io/community/handbook)
- [Discussion Forum](https://meshery.io/community#community-forums)
- [Community Slack](https://slack.meshery.io/)

Jump into our Slack! Submit the
[community member form](https://meshery.io/newcomers) to access additional
resources. Don't forget to join the [Newcomers meeting](https://meet.meshery.io)
held every Thursday!

<img src="https://raw.githubusercontent.com/meshery/meshery/refs/heads/master/.github/assets/images/readme/community.png" style="margin:10px;" width="180px" alt="Community" align="right" />

- ✔️ [Star](https://github.com/meshery-extensions/meshery-mcp-server/stargazers)
  the repo
- ✔️ [Join](https://meshery.io/newcomers) the community
- ✔️ [Watch](https://www.youtube.com/@mesheryio?sub_confirmation=1) community
  meeting recordings
- ✔️ [Discuss](https://discuss.meshery.io) in the Meshery forum
- ✔️ Grab an open [help-wanted
  issue](https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+(org%3Ameshery+OR+org%3Aservice-mesh-performance+OR+org%3Aservice-mesh-patterns+OR+org%3Ameshery-extensions)+label%3A%22help+wanted%22)
