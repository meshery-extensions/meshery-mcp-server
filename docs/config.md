# Configuration

meshery-mcp-server reads its configuration once, at startup. To change a setting, restart the server; there is no hot reload.

## Settings

| Setting | Flag | Environment variable | Config file key | Default | Accepted values |
|---|---|---|---|---|---|
| Meshery server URL | | `MESHERY_SERVER_URL` | `meshery.url` | `http://localhost:9081` | An `http://` or `https://` URL with a host |
| Meshery provider token | | `MESHERY_API_TOKEN` | *(not allowed)* | *(none)* | Any string |
| Transport | `--transport` | `MESHERY_MCP_TRANSPORT` | `transport.mode` | `stdio` | `stdio`, `http` (case-insensitive) |
| HTTP host | `--host` | `MESHERY_MCP_HOST` | `transport.host` | `127.0.0.1` | A loopback address: `127.0.0.1`, `::1`, `localhost` |
| HTTP port | `--port` | `MESHERY_MCP_PORT` | `transport.port` | `8080` | `1`–`65535` |
| HTTP access key | | `MESHERY_MCP_ACCESS_KEY` | *(not allowed)* | *(none)* | At least 32 characters; required for `http` |
| Log level | | `LOG_LEVEL` | `log.level` | `info` | `debug`, `info`, `warn`, `error` (case-insensitive) |
| Log format | | `LOG_FORMAT` | `log.format` | `text` | `text`, `json` (case-insensitive) |

The **Meshery server URL** is the address of the Meshery server this MCP server sends requests to. It is not an address this server listens on.

The **transport** is how MCP clients reach this server:

- `stdio`: the AI app starts the server and talks to it over stdin/stdout. Only that app can reach it.
- `http`: the server listens for [Streamable HTTP](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http) at `http://<host>:<port>/mcp`, on a loopback address only, and every request must carry the **HTTP access key**. See [Over HTTP (localhost)](#over-http-localhost). The deprecated HTTP+SSE transport is not supported.

**HTTP host**, **HTTP port** and **HTTP access key** apply to the `http` transport only; the access key is not even read otherwise.

The **token** is how the server identifies itself to Meshery:

- **Set:** every request to Meshery is made as the user the token belongs to. That is correct only when the server serves that one user: over stdio, the app that started it; over HTTP, clients that hold the access key.
- **Not set:** requests carry no token. Use this with Meshery's Local ("None") provider.

The token is read from the environment only, never from the config file. It is not stored on the Meshery client: it is attached to each request as it is handled, so a later transport that takes each caller's own identity needs no change to the client or the tools.

## Where settings come from

Each setting is resolved independently, from lowest to highest priority:

```
built-in defaults  <  config file (--config)  <  environment variables  <  flags
```

- Only the transport settings have flags.
- A higher source overrides only the settings it actually sets. For example, a config file can set the log level while `MESHERY_SERVER_URL` overrides only the URL.
- Secrets are the exception: the token comes from `MESHERY_API_TOKEN` only and the HTTP access key from `MESHERY_MCP_ACCESS_KEY` only. Neither has a flag, because command lines are visible to other users in process listings.
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

transport:
  mode: stdio
  host: 127.0.0.1
  port: 8080

log:
  level: info
  format: text
```

To start your own, copy the example to a gitignored name:

```sh
cp config_example.yml config.local.yaml
```

`*.local.yaml`, `/config.yaml` and `*.env` are in `.gitignore`, so files with these names cannot be committed by accident.

If `--config` points to a file that does not exist or is not valid YAML, the server exits with an error rather than silently falling back to the defaults. The same applies if the file contains a secret (`meshery.token`, `auth.token` or `transport.access_key`).

## Handling the token

The token is a credential. The server never prints it: it is redacted as `[REDACTED]` in logs and errors. Keeping it safe outside the server is up to how you supply it.

- **Set it in the environment** (`MESHERY_API_TOKEN`). The config file cannot hold it: config files get copied, shared and committed, and the MCP specification expects stdio servers to take credentials from the environment.
- **With Docker**, pass it as `-e MESHERY_API_TOKEN` with no `=value`. Docker then copies the value from its own environment, so the token never appears in the command line or in process listings.
- **Never bake it into an image** with `ENV` or `--build-arg`. Build arguments are recorded in the image and readable with `docker history`.

The same applies to `MESHERY_MCP_ACCESS_KEY`: environment only, never in the config file or on the command line.

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

The image runs as a non-root user (UID 65532), so the mounted file must be readable by others (`644`). It holds no token, so that is safe; the token is passed with `-e` as shown.

### Over HTTP (localhost)

```sh
export MESHERY_MCP_ACCESS_KEY="$(openssl rand -hex 32)"
meshery-mcp-server --transport http              # http://127.0.0.1:8080/mcp
meshery-mcp-server --transport http --port 9000  # http://127.0.0.1:9000/mcp
```

Point an MCP client that supports Streamable HTTP at the URL and configure it to send this header with every request:

```
Authorization: Bearer <the value of MESHERY_MCP_ACCESS_KEY>
```

The server stops cleanly on Ctrl+C or SIGTERM; a second Ctrl+C exits immediately.

The HTTP transport:

- **Listens on loopback only.** `--host` must be `127.0.0.1`, `::1` or `localhost`; anything else stops the server at startup. Remote access needs MCP authorization (OAuth), which is not implemented yet.
- **Requires the access key on every request.** Loopback keeps other machines out, but not other users or processes on this one. Without the key a request gets `401 Unauthorized`. This matters because every caller acts with your Meshery token, if one is set.
- **Rejects requests from web pages.** Cross-origin browser requests and requests whose `Host` header is not localhost (DNS rebinding) get `403 Forbidden`, so a website you visit cannot call the server through your browser.
- **Is stateless.** Each request stands alone; there are no sessions. `GET` and `DELETE` return `405 Method Not Allowed`, as protocol version 2026-07-28 expects. Clients on 2025 protocol versions are served the same way.

Docker is not supported for the HTTP transport yet: inside a container, loopback is the container itself, and listening on any other address is refused.

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

`LOG_FORMAT` controls the shape of each line:

| Format | Use it for | Example |
|---|---|---|
| `text` | Development: easy to read in a terminal (default) | `time="2026-10-16T23:56:59-04:00" level=info msg="server run start" app=meshery-mcp-server` |
| `json` | Production: one JSON object per line, for log collectors | `{"app":"meshery-mcp-server","level":"info","msg":"server run start","time":"2026-10-16T23:56:59-04:00"}` |

Every line has the same four fields, `time`, `level`, `msg` and `app`, whether the server or the MCP SDK wrote it. Errors also carry `code`, `severity`, `short-description`, `probable-cause` and `suggested-remediation`.

At startup the server logs what it offers to MCP clients:

```text
level=info msg="registered capabilities" tools="[search_components get_component list_models get_model list_relationships]" resources=[] prompts=[]
```

An error in the configuration itself is logged as `text` at `info` level, because the configured format and level are not known until the configuration has loaded.

## Errors

Configuration problems and HTTP listen failures stop the server at startup; a slow HTTP shutdown is reported when the server stops. Each error is logged with a short description, the probable cause and a suggested fix.

| Code | Short description | Cause | Fix |
|---|---|---|---|
| `meshery-mcp-server-1010` | Unable to read the config file | The `--config` path does not exist, is not readable, or is not valid YAML | Check the path and the YAML syntax |
| `meshery-mcp-server-1011` | Unable to decode configuration | A setting in the file or environment has the wrong type | Make `transport.port` / `MESHERY_MCP_PORT` a number and every other setting a plain string |
| `meshery-mcp-server-1012` | Invalid Meshery server URL | Malformed URL, a scheme other than `http`/`https`, or no host | Use a URL like `http://localhost:9081` |
| `meshery-mcp-server-1013` | Invalid log level | `LOG_LEVEL` / `log.level` is not a supported level | Use `debug`, `info`, `warn` or `error` |
| `meshery-mcp-server-1014` | Token is not allowed in the config file | The config file sets `meshery.token` or `auth.token` | Remove it and set `MESHERY_API_TOKEN` instead |
| `meshery-mcp-server-1015` | Invalid transport | The transport is not `stdio` or `http` (for example the deprecated `sse`) | Use `stdio` or `http` |
| `meshery-mcp-server-1016` | HTTP transport must listen on a loopback address | `--host` / `MESHERY_MCP_HOST` is not `127.0.0.1`, `::1` or `localhost` | Use a loopback address; remote access is not supported yet |
| `meshery-mcp-server-1017` | Invalid port | `--port` / `MESHERY_MCP_PORT` is outside 1–65535 | Use a port between 1 and 65535 |
| `meshery-mcp-server-1018` | HTTP access key is required | The transport is `http` and `MESHERY_MCP_ACCESS_KEY` is unset or shorter than 32 characters | Set it, for example to the output of `openssl rand -hex 32` |
| `meshery-mcp-server-1019` | Access key is not allowed in the config file | The config file sets `transport.access_key` | Remove it and set `MESHERY_MCP_ACCESS_KEY` instead |
| `meshery-mcp-server-1020` | Unable to listen for HTTP connections | The port is in use or the address is unavailable | Pick a free port or stop the process using it |
| `meshery-mcp-server-1021` | HTTP access key is required | `RunHTTP` was called without an access key (config validation normally catches this first) | Set `MESHERY_MCP_ACCESS_KEY` |
| `meshery-mcp-server-1022` | HTTP server did not shut down cleanly | Requests were still running when the 5-second shutdown timeout ended | None needed; remaining connections were closed |
| `meshery-mcp-server-1023` | Invalid log format | `LOG_FORMAT` / `log.format` is not a supported format | Use `text` or `json` |

Logs and error messages never include the token, the access key, or any password embedded in the URL: the startup line logs the URL with the password replaced by `xxxxx`.

Error codes follow Meshery's conventions (`meshery-mcp-server-<number>`).

## For contributors

### How configuration is wired

- `internal/config` is the only package that knows about Viper, environment variables or the config file. `config.Load(path, flags)` returns a validated `*config.Config` or an error.
- `main` parses the flags, passes only the ones given on the command line to `config.Load` as `config.Flags`, and passes plain values to the packages that need them. Other packages never read configuration themselves.
- `internal/mcp-server` knows nothing about `config`: `main` picks `RunStdio` or `RunHTTP` from `config.Transport` and builds `mcpserver.HTTPOptions`.
- `config.Meshery` says where Meshery is; `config.Auth` says who is calling. They are kept apart on purpose: the Meshery client gets only the URL, and `auth.Resolver` turns `config.Auth` into a function the MCP server runs on every request to attach the caller's token to the request's `context`. The Meshery client reads it from there with `meshery.TokenFromContext`. Nothing built at startup holds a token, except that function when a token is set.
- The log level and format are stored as plain names (`"info"`, `"json"`, …), independent of any logging library. `internal/logging` turns them into the MeshKit logger the server uses and the `slog` logger the MCP SDK takes, both writing the same fields.
- The token is a `config.Secret`, which redacts itself when printed or JSON-encoded. Convert it with `string(token)` only where it is actually used.

### Adding a setting

1. In `internal/config/config.go`, add a key constant (`"section.name"`) and an env var constant. The env var name is public: changing it breaks users' configs.
2. Add a field with a `mapstructure` tag to the right struct.
3. Call `v.SetDefault` for the key, with `""` if it has no default, so every setting and its default are listed in one place.
4. Call `v.MustBindEnv(key, envVar)`. If the setting also has a flag, add a field to `config.Flags`, apply it with `v.Set` in `Load`, and register the flag in `main`. Secrets get no flag and no config file key.
5. Validate the value in `validate()`, and add an error to `internal/config/error.go` following meshkit's conventions, with the next code from `helpers/component_info.json`.
6. Add test cases to `config_test.go`.
7. Add the setting to `config_example.yml`. A test loads that file, so a broken example fails the tests.
8. Update this document.
