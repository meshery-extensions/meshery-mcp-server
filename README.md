<div>
    <!-- Top section -->
    <div>
        <img src="https://raw.githubusercontent.com/meshery-extensions/.github/master/profile/assets/img/meshery-extensions-github.png" usemap="#workmap" />
    </div>
    <!-- Overview section -->
    <div align="center">
        <h3>Meshery is an extensible, self-service engineering platform for the collaborative management of cloud and cloud native infrastructure.</h3>
        <h3 align="center"><a href="https://meshery.io/extensions">Browse all extensions</a></h3>
        <h5 align="center">
            <a href="https://meshery.io#getting-started">Installation</a> |
            <a href="https://docs.meshery.io">Documentation</a> |
            <a href="https://discuss.meshery.io">Forum</a> |
            <a href="https://play.meshery.io">Playground</a> |
            <a href="https://meshery.io/catalog">Catalog</a>
        </h5>
        <br />
        <a href="https://github.com/meshery-extensions/meshery-mcp-server/graphs/contributors"><img src="https://img.shields.io/github/contributors/meshery-extensions/meshery-mcp-server.svg" /></a>
        <a href="https://github.com/meshery-extensions/meshery-mcp-server/blob/master/go.mod"><img alt="GitHub go.mod Go version" src="https://img.shields.io/github/go-mod/go-version/meshery-extensions/meshery-mcp-server" /></a>
        <a href="https://modelcontextprotocol.io"><img alt="Model Context Protocol" src="https://img.shields.io/badge/MCP-Server-000000?logo=modelcontextprotocol&logoColor=white" /></a>
        <a href="https://slack.meshery.io"><img src="https://img.shields.io/badge/Slack-Join%20the%20community-00B39F?logo=slack&logoColor=white" /></a>
    </div>

<h1>Meshery MCP Server</h1>

The Meshery MCP Server exposes Meshery to AI clients through the [Model Context Protocol](https://modelcontextprotocol.io/), an open standard for connecting AI assistants to external systems. Rather than each assistant requiring a bespoke Meshery integration, any MCP-capable client — AI assistants, IDE agents, and automation tooling — can query and operate a Meshery deployment through a single, consistent interface.

Meshery MCP Server is a [Meshery Extension](https://meshery.io/extensions). Built in Go on the [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk), it runs as a standalone binary or container alongside your Meshery deployment(s).

<h2>Overview</h2>

The Model Context Protocol defines three primitives that a server offers to its clients. Meshery MCP Server implements all three on top of the Meshery API:

- **Tools** — Meshery operations a model can invoke on the user's behalf.
- **Resources** — Meshery state exposed as read-only context for the client to draw on.
- **Prompts** — Reusable prompt templates that clients can surface to users for common Meshery workflows.

Supporting capabilities:

- **Dual transport** — stdio for local AI client integrations; SSE over HTTP for remote and web-based clients.
- **Meshery API client** — A single, shared client for communicating with a Meshery deployment.
- **Configuration** — CLI flags and environment-based configuration, with structured logging in JSON or text mode.
- **Container-ready** — Multi-stage Docker build producing a minimal, non-root runtime image.

<h2>Architecture</h2>

<pre>
┌──────────────────────────────────────────────────────────┐
│  AI Clients                                              │
│  AI assistants · IDE agents · MCP-capable tooling        │
└────────────────────────────┬─────────────────────────────┘
                             │  JSON-RPC over stdio or SSE
┌────────────────────────────▼─────────────────────────────┐
│  meshery-mcp-server                                      │
│                                                          │
│  cmd/meshery-mcp-server    Entrypoint, flag parsing,     │
│                            transport selection,          │
│                            graceful shutdown             │
│                              │                           │
│  internal/config           Configuration loading         │
│                              │                           │
│  internal/tools            MCP tools                     │
│  internal/resources        MCP resources                 │
│  internal/prompts          MCP prompts                   │
│                              │                           │
│  internal/meshery          Meshery API client            │
└────────────────────────────┬─────────────────────────────┘
                             │  REST
┌────────────────────────────▼─────────────────────────────┐
│  Meshery Server                                          │
└──────────────────────────────────────────────────────────┘
</pre>

The transport layer is the only component that differs between local and remote deployments; tools, resources, prompts, and the Meshery API client are shared across both.

<h2>Repository Structure</h2>

<pre>
meshery-mcp-server/
├── .github/                         # GitHub-related resources and automation
│   └── workflows/                   # CI/CD pipelines
├── build/                           # Makefile includes
├── cmd/
│   └── meshery-mcp-server/          # Server entrypoint
├── internal/
│   ├── config/                      # Configuration loading
│   ├── meshery/                     # Meshery API client
│   ├── prompts/                     # MCP prompt implementations
│   ├── resources/                   # MCP resource implementations
│   └── tools/                       # MCP tool implementations
├── .golangci.yml                    # Lint configuration
├── Dockerfile                       # Multi-stage container build
├── go.mod / go.sum                  # Go module dependencies
└── Makefile                         # Build & development commands
</pre>

<h2>Quick Start</h2>

<h3>Prerequisites</h3>

| Tool | Version | Link |
|------|---------|------|
| **Go** | 1.26.4 | [Install Go](https://go.dev/doc/install) |
| **golangci-lint** | v2.x | [Install golangci-lint](https://golangci-lint.run/docs/welcome/install/local/) |
| **Docker** | 29.5.2 | [Install Docker](https://docs.docker.com/get-docker/) |

*(Note: Docker is only required for container builds; the server can be built and run directly with Go).*

<h3>1. Fork & Clone</h3>

```bash
# Fork this repository on GitHub, then clone your fork
git clone https://github.com/<your-username>/meshery-mcp-server.git
cd meshery-mcp-server
```

<h3>2. Build the Server</h3>

```bash
make build
```

The binary is written to `bin/meshery-mcp-server`.

<h3>3. Run the Server</h3>

```bash
make run
```

<h3>4. Build the Container Image</h3>

```bash
make docker-build
```

This produces a minimal, non-root image tagged with the current Git version:

```bash
docker run --rm -i meshery-extensions/meshery-mcp-server:<tag>
```

> **Note:** `-i` keeps stdin open, which the stdio transport requires.

<h3>5. Other Useful Commands</h3>

| Command | Description |
|---------|-------------|
| `make build` | Build the server binary for the host platform |
| `make run` | Build and run the server locally |
| `make test` | Run unit tests |
| `make lint` | Run `golangci-lint` across the module |
| `make fmt` | Format source files with `gofmt` |
| `make clean` | Remove build output |
| `make docker-build` | Build the container image via multi-stage Docker build |

Run `make lint` and `make test` before opening a pull request; both are enforced by CI on every PR.

<h2>Related Repositories</h2>

- [meshery/meshery](https://github.com/meshery/meshery) – Meshery core project
- [meshery-extensions/meshery-mcp-server](https://github.com/meshery-extensions/meshery-mcp-server) – this repo

    <!-- Contributing and Guidelines -->
    <div>
        <h2>Community and Contributing</h2>
        <p>Please do! Code and non-code contributions are welcome. This project is community-built and fosters collaboration. Contributors are expected to adhere to the <a href="https://github.com/cncf/foundation/blob/main/code-of-conduct.md"> CNCF Code of Conduct</a>.
        </p>
        <p>Jump into our <a href="https://slack.meshery.io">Slack</a>! Submit your <a href="https://meshery.io/newcomers">community member form</a> access to additional resources. Don't forget to join the <a href="https://meshery.io/calendar">Newcomers meeting</a> held every Thursday!
        </p>
        <img src="https://raw.githubusercontent.com/meshery/meshery/refs/heads/master/.github/assets/images/readme/community.png"
            style="margin:10px;" width="180px" alt="Community" align="right" />
        <ul>
            ✔️ <b>Star</b> ⭐ the main <a href="https://github.com/meshery/meshery">meshery repo</a><br />
            ✔️ <b>Join</b> any or all of the weekly meetings on the <a href="https://meet.meshery.io">community
                    calendar</a><br />
            ✔️ <b>Watch</b> <a
                    href="https://www.youtube.com/@mesheryio?sub_confirmation=1">community meeting
                    recordings</a><br />
            <p>✔️ <b>Access</b> resources by completing a <a href="https://meshery.io/newcomers"> community member form
                </a><br />
            ✔️ <b>Discuss</b> in a Meshery <a href="https://discuss.meshery.io">Community forum</a><br />
            ✔️ Not sure where to start? <b>Grab</b> an open issue with the <a
                    href="https://github.com/issues?q=is%3Aopen+is%3Aissue+archived%3Afalse+(org%3Ameshery+OR+org%3Aservice-mesh-performance+OR+org%3Aservice-mesh-patterns+OR+org%3Ameshery-extensions)+label%3A%22help+wanted%22">help-wanted
                    label</a><br />
        </ul>
    </div>
</div>
