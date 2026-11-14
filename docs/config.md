# Configuration

meshery-mcp-server reads its configuration once, at startup. To change a setting, restart the server; there is no hot reload.

## Settings

| Setting | Environment variable | Config file key | Default | Accepted values |
|---|---|---|---|---|
| Meshery server URL | `MESHERY_SERVER_URL` | `meshery.url` | `http://localhost:9081` | An `http://` or `https://` URL with a host |
| Meshery provider token | `MESHERY_API_TOKEN` | `meshery.token` | *(none)* | Any string; optional |
| Keychain scope | `MESHERY_KEYCHAIN_SCOPE` | `meshery.keychain_scope` | *(none)* | Comma-separated keychain ids/names; empty = unscoped token (previous behavior) |
| Max concurrent Cloud calls per tool | `MESHERY_MAX_CONCURRENT_CALLS` | `meshery.max_concurrent_calls` | `4` | Integer >= 1 |
| Max Cloud retries per call | `MESHERY_MAX_RETRIES` | `meshery.max_retries` | `3` | Integer >= 0 |
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

## Cloud API-token behavior

When `MESHERY_SERVER_URL` points at Meshery Cloud, the client behaves as follows:

**Keychain-scoped tokens (#6369).** `meshery.keychain_scope` / `MESHERY_KEYCHAIN_SCOPE` requests the narrowest
keychain scope the deployment needs; it defaults to empty, which means no
keychain scope and the previous, unscoped behavior. A scoped token receives
403 on any route that makes no key check.

A Cloud 403 is treated as "insufficient scope", never as "bad token", so it
never triggers sign-in. When Cloud's response identifies the missing scope
or key, the error names it and tells the operator to grant it (or to set
`MESHERY_KEYCHAIN_SCOPE`); when it does not, the error says so.

**Per-user rate limits (#6333).** Cloud enforces a token bucket per (route
family, caller). Concurrent Cloud calls are bound per tool
(`meshery.max_concurrent_calls`, default 4) so one tool invocation—or a
batch—cannot flood a route family. On 429 the client honors `Retry-After`
(delta-seconds or HTTP-date); without it, it backs off exponentially with
jitter (capped at 8s). Retries are bounded (default 3), context-aware, and
never a tight loop; exhausted retries surface
"rate limited by Cloud; retry after N seconds". 429 is not an auth failure.

**Revocation (#6381).** `POST /revoke` retires Cloud-issued tokens
immediately. As soon as the user revokes, the cached token is dropped
client-side (memory; disk/keyring/session stores are emptied the same way)
regardless of the server's response. Revocation succeeds only when Cloud
answers 200; 503 is retried with backoff, never reported as success.
401 remains the only re-auth signal.

**IdP outage (#6366).** Cloud answers 503 on an identity-provider outage
instead of 401. The client backs off on 503 (bounded retries with jitter)
and surfaces "service unavailable". A 503 is never treated as "token
revoked" or "token invalid": no stored credentials are cleared and no sign-in
is forced. Cloud caches active introspection for at most 60s, so the client
adds no longer client-side validity cache; only safe caches exist, evicted
on revoke.

## Troubleshooting

| Status | Meaning | What to do |
|--------|---------|------------|
| 401 | Token missing, expired or revoked | Sign in again (Cloud OAuth); this is the only status that triggers re-auth |
| 403 | Insufficient keychain scope | Grant the missing scope named in the error, or set `MESHERY_KEYCHAIN_SCOPE` to the narrowest scope the deployment needs; do not re-auth |
| 429 | Rate limited | Wait for `Retry-After` (or the backoff) and retry; reduce `meshery.max_concurrent_calls` if batches keep hitting it |
| 503 | Cloud service unavailable | Retry later; no credential changes are made |

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
