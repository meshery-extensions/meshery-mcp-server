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
<a href="https://github.com/meshery-extensions/meshery-mcp-server/actions/workflows/build-and-test.yml"><img src="https://github.com/meshery-extensions/meshery-mcp-server/actions/workflows/build-and-test.yml/badge.svg" alt="Build and Test" /></a>
<a href="https://github.com/meshery-extensions/meshery-mcp-server/blob/master/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/meshery-extensions/meshery-mcp-server" alt="Go version" /></a>
<a href="https://modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-server-000000" alt="Model Context Protocol" /></a>
<a href="https://slack.meshery.io"><img src="https://img.shields.io/badge/Slack-join%20the%20community-00B39F?logo=slack&logoColor=white" alt="Slack" /></a>
</div>

# Meshery MCP Server

Meshery MCP Server connects AI assistants to [Meshery](https://meshery.io) through the [Model Context Protocol](https://modelcontextprotocol.io) (MCP), an open standard for connecting AI applications to external systems. Any MCP-capable client, such as Claude Desktop, Claude Code, or an IDE agent, can then work with Meshery designs through one consistent interface instead of a custom integration per assistant.

It is an official [Meshery Extension](https://meshery.io/extensions), written in Go, and runs as a standalone binary or container next to your Meshery deployment.

## Project status

> [!NOTE]
> **Early development.** The MCP server runs over stdio and over Streamable HTTP on localhost, but it does not call Meshery yet: its tools are placeholders that return a "not implemented" error.

| Area | Status |
|---|---|
| Configuration (env vars, config file, flags), logging, Meshery-style errors | ✅ Done |
| Build, lint, tests, container image, CI | ✅ Done |
| MCP server over stdio | ✅ Done |
| MCP server over Streamable HTTP, localhost only, with an access key | ✅ Done |
| Remote HTTP with MCP authorization (OAuth) | Planned |
| Meshery API client | Planned |
| MCP tools | Placeholders (return "not implemented") |
| MCP resources and prompts | Planned |

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

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│ MCP client: Claude Desktop, Claude Code, IDE agents, ...     │
│ starts the server (stdio) or connects to it (HTTP)           │
└──────────────────────────────────────────────────────────────┘
                                │  JSON-RPC over stdio, or Streamable HTTP on localhost
                                ▼
┌──────────────────────────────────────────────────────────────┐
│ meshery-mcp-server                                           │
│                                                              │
│ cmd/meshery-mcp-server  entrypoint: flags, logger, wiring    │
│ internal/config         settings from env vars and file      │
│ internal/logging        one log format for server and SDK    │
│ internal/auth           attaches the caller's token per call │
│ internal/mcp-server     MCP server over stdio or HTTP        │
│                                                              │
│ internal/tools          MCP tools          (placeholders)    │
│ internal/resources      MCP resources      (planned)         │
│ internal/prompts        MCP prompts        (planned)         │
│ internal/meshery        Meshery API client (stubbed)         │
│                                                              │
│ logs go to stderr; stdout is reserved for the protocol       │
└──────────────────────────────────────────────────────────────┘
                                │  HTTP(S) + provider token
                                ▼
┌──────────────────────────────────────────────────────────────┐
│ Meshery Server  (MESHERY_SERVER_URL)                         │
└──────────────────────────────────────────────────────────────┘
```

By default the MCP client launches the server and talks to it over stdin/stdout, so the server does not listen on any network address. With `--transport http` it listens instead at `http://127.0.0.1:8080/mcp`, on loopback only, and every request must carry an access key. Because stdout carries the protocol over stdio, all logs go to stderr.

Only `cmd/meshery-mcp-server` wires the packages together. `internal/config` is the only package that reads environment variables or the config file; the other packages receive plain values from `main`. See [docs/config.md](docs/config.md#for-contributors) for details.

## Repository structure

```text
meshery-mcp-server/
├── .github/workflows/          # CI: lint, build and test, Docker image
├── build/                      # Shared Makefile includes
├── cmd/meshery-mcp-server/     # Entrypoint
├── docs/
│   └── config.md               # Configuration reference
├── helpers/
│   └── component_info.json     # Error-code metadata for Meshery's errorutil
├── internal/
│   ├── auth/                   # Per-request credentials
│   ├── config/                 # Configuration loading and validation
│   ├── logging/                # Log format and level for the server and the MCP SDK
│   ├── mcp-server/             # MCP server construction and transport
│   ├── meshery/                # Meshery API client
│   ├── prompts/                # MCP prompts
│   ├── resources/              # MCP resources
│   └── tools/                  # MCP tools
├── .dockerignore
├── .golangci.yml               # Lint configuration
├── config_example.yml          # Example config file
├── Dockerfile                  # Multi-stage build, distroless non-root runtime
├── go.mod / go.sum
└── Makefile                    # Build and development commands
```

## Quick start

### Prerequisites

| Tool | Version | Needed for |
|---|---|---|
| [Go](https://go.dev/doc/install) | as declared in [`go.mod`](go.mod) | Building and testing |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | v2.x | `make lint` |
| [Docker](https://docs.docker.com/get-docker/) | any recent version | Container builds only |

### 1. Clone and build

```sh
git clone https://github.com/meshery-extensions/meshery-mcp-server.git
cd meshery-mcp-server
make build            # writes bin/meshery-mcp-server
```

### 2. Configure

The server works with no configuration if Meshery runs at `http://localhost:9081`. Otherwise, set environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `MESHERY_SERVER_URL` | `http://localhost:9081` | Meshery server to connect to |
| `MESHERY_API_TOKEN` | *(none)* | Meshery provider token. Environment only; never in the config file |
| `MESHERY_MCP_TRANSPORT` | `stdio` | `stdio` or `http`; also `--transport` |
| `MESHERY_MCP_HOST` | `127.0.0.1` | HTTP only, loopback addresses only; also `--host` |
| `MESHERY_MCP_PORT` | `8080` | HTTP only; also `--port` |
| `MESHERY_MCP_ACCESS_KEY` | *(none)* | HTTP only, required: the bearer token clients must send. Environment only |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `LOG_FORMAT` | `text` | `text` for development, `json` for log collectors |

To use a config file instead:

```sh
cp config_example.yml config.local.yaml   # *.local.yaml is gitignored
```

and pass it with `--config config.local.yaml`. Environment variables override the file.

[docs/config.md](docs/config.md) is the full reference: precedence rules, token handling, Docker specifics and every error message.

### 3. Run

```sh
make run
make run ARGS="--config config.local.yaml"
```

This prints a startup line to stderr, then serves MCP over stdin/stdout until stdin closes or you press Ctrl+C:

```text
time="..." level=info msg="starting meshery-mcp-server version=... commit=... meshery_url=http://localhost:9081 token_configured=false transport=stdio" app=meshery-mcp-server
```

To serve Streamable HTTP on localhost instead:

```sh
export MESHERY_MCP_ACCESS_KEY="$(openssl rand -hex 32)"
make run ARGS="--transport http --port 8080"   # http://127.0.0.1:8080/mcp
```

Clients must send `Authorization: Bearer $MESHERY_MCP_ACCESS_KEY`. To call the tools by hand, use the MCP Inspector: `npx @modelcontextprotocol/inspector ./bin/meshery-mcp-server` for stdio, or start it without arguments and connect to the HTTP URL with that header.

### 4. Run in a container

```sh
make docker-run
```

This builds the image and runs it over stdio. Inside a container, `localhost` refers to the container itself, so `make docker-run` points the server at `http://host.docker.internal:9081` by default. Override it with `DOCKER_MESHERY_SERVER_URL`. `MESHERY_API_TOKEN` is forwarded from your shell.

### 5. Connect an MCP client

> [!NOTE]
> The tools are placeholders for now: every call returns a "not implemented" error, and their descriptions say so.

MCP clients start the server themselves. For example, in Claude Desktop's configuration:

```json
{
  "mcpServers": {
    "meshery": {
      "command": "/path/to/meshery-mcp-server",
      "env": {
        "MESHERY_SERVER_URL": "http://localhost:9081",
        "MESHERY_API_TOKEN": "<token>"
      }
    }
  }
}
```

Point the client at the binary or at `docker run -i ...`, never at `make`: Make prints its commands to stdout, which would corrupt the MCP stream. [docs/config.md](docs/config.md#from-an-ai-app-mcp-client) has the Docker version.

## Development

| Command | Description |
|---|---|
| `make build` | Build the binary for your platform into `bin/` |
| `make run` | Build and run the binary; pass flags with `ARGS="..."` |
| `make test` | Run unit tests |
| `make coverage` | Run tests with the race detector and write `coverage.txt` |
| `make lint` | Run golangci-lint |
| `make fmt` | Format the code with gofmt |
| `make clean` | Remove `bin/` and `coverage.txt` |
| `make docker-build` | Build the container image |
| `make docker-run` | Build the image and run it over stdio |
| `make` or `make show-help` | List all targets |

Every pull request runs three CI jobs: **golangci-lint**, **Build and unit tests** (tidy check, build, tests with coverage), and **Test Docker Image Build** (builds the image and starts it). Run `make lint` and `make coverage` before opening one.

Errors follow Meshery's [error conventions](https://docs.meshery.io/project/contributing/contributing-error): each package defines its errors in `error.go`, with codes `meshery-mcp-server-<number>`. To add an error, use the `next_error_code` in [`helpers/component_info.json`](helpers/component_info.json) and increase it, so no number is used twice. To check your errors locally:

```sh
go run github.com/meshery/meshkit/cmd/errorutil -d . analyze -i ./helpers -o ./helpers
```

## About Meshery Extensions

[Meshery Extensions](https://meshery.io/extensions) add capabilities to Meshery beyond its core platform. They live in the [meshery-extensions](https://github.com/meshery-extensions) organization, separate from the core [meshery](https://github.com/meshery) organization; the [ecosystem expansion post](https://meshery.io/blog/2025/meshery-ecosystem-expansion) explains why and how that organization is governed.

Related repositories:

- [meshery/meshery](https://github.com/meshery/meshery): Meshery core
- [meshery/meshkit](https://github.com/meshery/meshkit): shared Go library used here for logging and errors

<div>
<h2>Community and Contributing</h2>
<p>Please do! Code and non-code contributions are welcome. This project is community-built and fosters collaboration. Contributors are expected to adhere to the <a href="https://github.com/cncf/foundation/blob/main/code-of-conduct.md">CNCF Code of Conduct</a>.</p>
<p>Jump into our <a href="https://slack.meshery.io">Slack</a>! Submit the <a href="https://meshery.io/newcomers">community member form</a> to access additional resources. Don't forget to join the <a href="https://meshery.io/calendar">Newcomers meeting</a> held every Thursday!</p>
<img src="https://raw.githubusercontent.com/meshery/meshery/refs/heads/master/.github/assets/images/readme/community.png" style="margin:10px;" width="180px" alt="Community" align="right" />
<p>
✔️ <b>Star</b> ⭐ the main <a href="https://github.com/meshery/meshery">meshery repo</a><br />
✔️ <b>Join</b> any or all of the weekly meetings on the <a href="https://meet.meshery.io">community calendar</a><br />
✔️ <b>Watch</b> <a href="https://www.youtube.com/@mesheryio?sub_confirmation=1">community meeting recordings</a><br />
✔️ <b>Access</b> resources by completing a <a href="https://meshery.io/newcomers">community member form</a><br />
✔️ <b>Discuss</b> in the Meshery <a href="https://discuss.meshery.io">community forum</a><br />
✔️ Not sure where to start? <b>Grab</b> an open issue with the <a href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+(org%3Ameshery+OR+org%3Aservice-mesh-performance+OR+org%3Aservice-mesh-patterns+OR+org%3Ameshery-extensions)+label%3A%22help+wanted%22">help-wanted label</a>
</p>
</div>

<img src="https://raw.githubusercontent.com/meshery/.github/master/profile/assets/img/footer.png" alt="" />
