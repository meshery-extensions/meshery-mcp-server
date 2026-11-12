# Configuration

The full configuration reference has moved to
[docs/configuration.md](docs/configuration.md). This page remains as a short
overview and pointer; `docs/configuration.md` is the source of truth for every
setting, environment variable, config file key, default, and error message.

meshery-mcp-server reads its configuration once, at startup. To change a
setting, restart the server; there is no hot reload.

## Settings (summary)

| Setting | Environment variable | Config file key | Default |
|---------|----------------------|-----------------|---------|
| Meshery server URL | `MESHERY_SERVER_URL` | `meshery.url` | `http://localhost:9081` |
| Meshery provider token | `MESHERY_API_TOKEN` | `meshery.token` | *(none)* |
| Log level | `LOG_LEVEL` | `log.level` | `info` |

See [docs/configuration.md](docs/configuration.md) for accepted values,
precedence rules, the config file format, token handling, Docker notes,
logging, errors, and the contributor guide for adding a setting.

## Usage

### From an AI app (MCP client)

MCP clients such as Claude Desktop, Claude Code, VS Code and Cursor start the
server themselves and talk to it over stdin/stdout. You configure the command
they run and its environment. Each client uses its own config file format; the
examples below use Claude Desktop's. Client-specific guides live in
[docs/integrations/](docs/integrations/).

**Native binary:**

```json
{
  "mcpServers": {
    "meshery": {
      "command": "/usr/local/bin/meshery-mcp-server",
      "env": {
        "MESHERY_SERVER_URL": "http://localhost:9081",
        "MESHERY_API_TOKEN": "<token>"
      }
    }
  }
}
```

To use a config file instead, add `"args": ["--config", "/absolute/path/to/config.local.yaml"]`.

**Docker image:**

```json
{
  "mcpServers": {
    "meshery": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "--add-host=host.docker.internal:host-gateway",
        "-e", "MESHERY_SERVER_URL",
        "-e", "MESHERY_API_TOKEN",
        "meshery-extensions/meshery-mcp-server:<version>"
      ],
      "env": {
        "MESHERY_SERVER_URL": "http://host.docker.internal:9081",
        "MESHERY_API_TOKEN": "<token>"
      }
    }
  }
}
```

Three details matter when running in a container:

- **`-i` is required.** stdin is the connection to the AI app.
- **`localhost` is the container itself.** Use `http://host.docker.internal:9081` to reach a Meshery server running on your machine. `--add-host=host.docker.internal:host-gateway` makes that name work on Linux; Docker Desktop on macOS and Windows provides it already.
- **Variables must be passed explicitly.** A container does not inherit the environment of whoever runs `docker run`. Only variables passed with `-e` reach the server. A missing `-e` produces no error: the default applies instead.

### From a terminal

**Native binary:**

```sh
MESHERY_SERVER_URL=http://localhost:9081 MESHERY_API_TOKEN=<token> meshery-mcp-server
meshery-mcp-server --config config.local.yaml
```

**Docker, with a config file:** the path given to `--config` is read inside the container, so the file must be mounted there first:

```sh
docker run -i --rm \
  --add-host=host.docker.internal:host-gateway \
  -v "$PWD/config.local.yaml:/config/config.yaml:ro" \
  -e MESHERY_API_TOKEN \
  meshery-extensions/meshery-mcp-server:<version> --config /config/config.yaml
```

The image runs as a non-root user (UID 65532), which cannot read a file with `600` permissions owned by you. Either keep the token out of the mounted file and pass it with `-e` as shown, so the file can be `644`, or add `--user "$(id -u):$(id -g)"`.

### During development (Makefile)

| Command | What it does |
|---------|--------------|
| `make run` | Builds and runs the binary. Uses your shell's environment variables. |
| `make run ARGS="--config config.local.yaml"` | Same, with flags passed through. |
| `make docker-run` | Builds the image and runs it over stdio. Forwards `MESHERY_API_TOKEN` from your shell. |
| `make docs-lint` | Lints Markdown files with markdownlint. |

`make docker-run` sets the URL from `DOCKER_MESHERY_SERVER_URL` (default `http://host.docker.internal:9081`), not from `MESHERY_SERVER_URL`. That way a `localhost` URL exported for native runs does not leak into the container, where it would point at the container itself. To override it:

```sh
DOCKER_MESHERY_SERVER_URL=http://other-host:9081 make docker-run
```

Use `make docker-run` only to try the image by hand; do not point an AI app at it. Make prints its commands to stdout, which would corrupt the MCP stream.

## For contributors

The detailed contributor guide for adding a setting lives in
[docs/configuration.md](docs/configuration.md#for-contributors).
