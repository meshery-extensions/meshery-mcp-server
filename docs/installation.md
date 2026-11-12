# Installation

Install the Meshery MCP Server binary, then point an MCP client (such as
Claude Desktop) at it. The server connects to a Meshery instance using the
`MESHERY_SERVER_URL` and `MESHERY_API_TOKEN` environment variables (or a config
file).

> [!NOTE]
> **Early development.** The MCP server itself is not yet implemented: the
> binary loads and validates its configuration, logs a startup line, and exits.
> See [docs/config.md](docs/config.md) and the [project status](../README.md#project-status)
> section of the README for what is built and what is planned.

## Install methods

### From source with Go (recommended)

The project is a Go module at
`github.com/meshery-extensions/meshery-mcp-server`. You need Go 1.27.1 or newer
(as declared in [`go.mod`](https://github.com/meshery-extensions/meshery-mcp-server/blob/master/go.mod)).

**Build from a clone:**

```sh
git clone https://github.com/meshery-extensions/meshery-mcp-server.git
cd meshery-mcp-server
make build            # writes bin/meshery-mcp-server
```

**Install with `go install`:**

```sh
go install github.com/meshery-extensions/meshery-mcp-server@latest
```

This puts `meshery-mcp-server` in your `$GOBIN` (usually
`$HOME/go/bin`). Make sure that directory is on your `$PATH`.

### With Docker

A multi-stage Dockerfile builds a distroless, non-root image:

```sh
make docker-build
make docker-run       # runs over stdio; forwards MESHERY_API_TOKEN from your shell
```

Inside a container, `localhost` refers to the container itself, so
`make docker-run` points the server at
`http://host.docker.internal:9081` by default. Override it with
`DOCKER_MESHERY_SERVER_URL`. `MESHERY_API_TOKEN` is forwarded from your shell.

See the [Docker section of the configuration reference](docs/config.md#docker)
for details.

### Not yet available

The following install methods are configured or planned but not yet published:

- **Pre-built release binaries** — [`goreleaser.yml`](.github/.goreleaser.yml)
  is configured to build darwin/linux/windows archives, but no releases or tags
  have been published yet.
- **Homebrew tap** — none exists yet.

## Verify the install

Run the binary with no arguments. It prints a startup line to stderr and exits:

```sh
meshery-mcp-server
# time="2026-..." level=info msg="starting meshery-mcp-server version=v0.0.0-unknown commit=unknown meshery_url=http://localhost:9081 token_configured=false" app=meshery-mcp-server
```

A `--version` flag is not yet implemented; the version and commit SHA are
embedded at build time via `-ldflags` and appear in the startup log.

## Next step

Configure the server (see [docs/configuration.md](docs/configuration.md)),
then connect an MCP client (see [docs/integrations/](docs/integrations/)).
