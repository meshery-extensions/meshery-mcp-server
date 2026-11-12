# Configuration

meshery-mcp-server reads its configuration once, at startup. To change a setting, restart the server; there is no hot reload.

## Settings

| Setting | Environment variable | Config file key | Default | Accepted values |
|---|---|---|---|---|
| Meshery server URL | `MESHERY_SERVER_URL` | `meshery.url` | `http://localhost:9081` | An `http://` or `https://` URL with a host |
| Meshery provider token | `MESHERY_API_TOKEN` | `meshery.token` | *(none)* | Any string; optional |
| Log level | `LOG_LEVEL` | `log.level` | `info` | `debug`, `info`, `warn`, `error` (case-insensitive) |

The **Meshery server URL** is the address of the Meshery server this MCP server sends requests to. It is not an address this server listens on: over stdio, the server has no network address of its own.

The **token** is optional at startup. Without one, requests that Meshery requires authentication for will fail.

## Where settings come from

Each setting is resolved independently, from lowest to highest priority:

```
built-in defaults  <  config file (--config)  <  environment variables
```

- A higher source overrides only the settings it actually sets. For example, a config file can provide the token while `MESHERY_SERVER_URL` overrides only the URL.
- An environment variable set to an empty string counts as unset, so the file value or the default applies.
- Invalid values stop the server at startup with an error that names the setting and how to fix it (see [Errors](#errors)).

## The config file

The config file is optional and is only read when you pass its path:

```sh
meshery-mcp-server --config config.local.yaml
```

It is YAML. [`config_example.yml`](../config_example.yml) lists every setting with comments:

```yaml
meshery:
  url: http://localhost:9081
  # token: <your-meshery-token>

log:
  level: info
```

To start your own, copy the example to a gitignored name:

```sh
cp config_example.yml config.local.yaml
```

`*.local.yaml`, `/config.yaml` and `*.env` are in `.gitignore`, so files with these names cannot be committed by accident.

If `--config` points to a file that does not exist or is not valid YAML, the server exits with an error rather than silently falling back to the defaults.

## Handling the token

The token is a credential. The server never prints it: it is redacted as `[REDACTED]` in logs and errors. Keeping it safe outside the server is up to how you supply it.

- **Prefer the environment variable** (`MESHERY_API_TOKEN`) over the config file.
- **If you put it in a file**, restrict the file to your user (`chmod 600`) and never commit it.
- **With Docker**, pass it as `-e MESHERY_API_TOKEN` with no `=value`. Docker then copies the value from its own environment, so the token never appears in the command line or in process listings.
- **Never bake it into an image** with `ENV` or `--build-arg`. Build arguments are recorded in the image and readable with `docker history`.

## Usage

### From an AI app (MCP client)

MCP clients such as Claude Desktop, Claude Code, VS Code and Cursor start the server themselves and talk to it over stdin/stdout. You configure the command they run and its environment. Each client uses its own config file format; the examples below use Claude Desktop's.

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
|---|---|
| `make run` | Builds and runs the binary. Uses your shell's environment variables. |
| `make run ARGS="--config config.local.yaml"` | Same, with flags passed through. |
| `make docker-run` | Builds the image and runs it over stdio. Forwards `MESHERY_API_TOKEN` from your shell. |

`make docker-run` sets the URL from `DOCKER_MESHERY_SERVER_URL` (default `http://host.docker.internal:9081`), not from `MESHERY_SERVER_URL`. That way a `localhost` URL exported for native runs does not leak into the container, where it would point at the container itself. To override it:

```sh
DOCKER_MESHERY_SERVER_URL=http://other-host:9081 make docker-run
```

Use `make docker-run` only to try the image by hand; do not point an AI app at it. Make prints its commands to stdout, which would corrupt the MCP stream.

## Logging

Logs are written to **stderr**. Stdout is reserved for MCP protocol messages, so nothing else may be written there. `LOG_LEVEL` controls verbosity:

| Level | Shows |
|---|---|
| `debug` | Everything, including diagnostic detail |
| `info` | Startup information and normal operation (default) |
| `warn` | Warnings and errors only |
| `error` | Errors only |

## MCP server

The MCP server is built on the official
[github.com/modelcontextprotocol/go-sdk](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)
v1.8.0, targeting protocol revision **2026-07-28** with backward-compatible
negotiation down to 2025-11-25 (see [docs/adr/0001-mcp-sdk-choice.md](../adr/0001-mcp-sdk-choice.md)).

### Transports

| Flag | Default | Description |
|------|---------|-------------|
| `--transport stdio` | `stdio` | JSON-RPC over stdin/stdout. For local AI clients such as Claude Desktop. Logs go to stderr. |
| `--transport http` | | Streamable HTTP (stateless mode) on `--port`. For remote clients. |

Examples:

```sh
# stdio (default)
meshery-mcp-server

# Streamable HTTP on port 8080
meshery-mcp-server --transport http --port 8080
```

### Protocol version

The server advertises `2026-07-28`. The SDK negotiates down to `2025-11-25`
for older clients, so existing MCP clients continue to work.

## Errors

Configuration problems stop the server at startup. Each error is logged with a short description, the probable cause and a suggested fix.

| Short description | Cause | Fix |
|---|---|---|
| Unable to read the config file | The `--config` path does not exist, is not readable, or is not valid YAML | Check the path and the YAML syntax |
| Unable to decode configuration | A setting in the file has the wrong type | Make every setting a plain string |
| Invalid Meshery server URL | Malformed URL, a scheme other than `http`/`https`, or no host | Use a URL like `http://localhost:9081` |
| Invalid log level | `LOG_LEVEL` / `log.level` is not a supported level | Use `debug`, `info`, `warn` or `error` |

Error messages never include the token or any password embedded in the URL.

Error codes follow Meshery's conventions (`meshery-mcp-server-<number>`) and are assigned by Meshery's `errorutil` tool. Until then they read `replace_me`.

## For contributors

### How configuration is wired

- `internal/config` is the only package that knows about Viper, environment variables or the config file. `config.Load(path)` returns a validated `*config.Config` or an error.
- `main` parses `--config`, calls `config.Load` and passes plain values to the packages that need them, for example the Meshery URL and token to the Meshery client. Other packages never read configuration themselves.
- The log level is stored as a plain name (`"info"`, …), independent of any logging library. `main` translates it for meshkit's logger in one function, `logrusLevel`.
- The token is a `config.Secret`, which redacts itself when printed or JSON-encoded. Convert it with `string(token)` only where it is actually used.

### Adding a setting

1. In `internal/config/config.go`, add a key constant (`"section.name"`) and an env var constant. The env var name is public: changing it breaks users' configs.
2. Add a field with a `mapstructure` tag to the right struct.
3. Call `v.SetDefault` for the key, even if the default is empty. Viper ignores environment variables for keys it has no default for.
4. Call `v.MustBindEnv(key, envVar)`.
5. Validate the value in `validate()`, and add an error to `internal/config/error.go` following meshkit's conventions.
6. Add test cases to `config_test.go`.
7. Add the setting to `config_example.yml`. A test loads that file, so a broken example fails the tests.
8. Update this document.
