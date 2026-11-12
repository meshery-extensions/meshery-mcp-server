# Configuration reference

meshery-mcp-server reads its configuration once, at startup. To change a
setting, restart the server; there is no hot reload.

> [!NOTE]
> **Early development.** Only the settings below exist in the current code.
> Settings such as keychain scope, per-tool concurrency limits, and bounded
> retries (from the accepted [MCP SDK ADR](adr/0001-mcp-sdk-choice.md)) are
> planned and not yet implemented.

## Settings

| Setting | Environment variable | Config file key | Default | Accepted values |
|---------|----------------------|-----------------|---------|-----------------|
| Meshery server URL | `MESHERY_SERVER_URL` | `meshery.url` | `http://localhost:9081` | An `http://` or `https://` URL with a host |
| Meshery provider token | `MESHERY_API_TOKEN` | `meshery.token` | *(none)* | Any string; optional |
| Log level | `LOG_LEVEL` | `log.level` | `info` | `debug`, `info`, `warn`, `error` (case-insensitive) |

## Where settings come from

Each setting is resolved independently, from lowest to highest priority:

```
built-in defaults  <  config file (--config)  <  environment variables
```

- A higher source overrides only the settings it actually sets. For example, a
  config file can provide the token while `MESHERY_SERVER_URL` overrides only
  the URL.
- An environment variable set to an empty string counts as unset, so the file
  value or the default applies.
- Invalid values stop the server at startup with an error that names the
  setting and how to fix it (see [Errors](#errors)).

## The config file

The config file is optional and is only read when you pass its path:

```sh
meshery-mcp-server --config config.local.yaml
```

It is YAML. [`config_example.yml`](../config_example.yml) lists every setting
with comments:

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

`*.local.yaml`, `/config.yaml` and `*.env` are in `.gitignore`, so files with
these names cannot be committed by accident.

If `--config` points to a file that does not exist or is not valid YAML, the
server exits with an error rather than silently falling back to the defaults.

## Handling the token

The token is a credential. The server never prints it: it is redacted as
`[REDACTED]` in logs and errors. Keeping it safe outside the server is up to
how you supply it.

- **Prefer the environment variable** (`MESHERY_API_TOKEN`) over the config
  file.
- **If you put it in a file**, restrict the file to your user (`chmod 600`)
  and never commit it.
- **With Docker**, pass it as `-e MESHERY_API_TOKEN` with no `=value`. Docker
  then copies the value from its own environment, so the token never appears
  in the command line or in process listings.
- **Never bake it into an image** with `ENV` or `--build-arg`. Build arguments
  are recorded in the image and readable with `docker history`.

## Multi-instance / multi-context setup

Not yet available. The server reads a single Meshery server URL and a single
token at startup. There is no named-context (`current-context`, `contexts:`)
mechanism, and no context switching. To talk to multiple Meshery instances,
run separate server instances, each with its own `MESHERY_SERVER_URL` and
`MESHERY_API_TOKEN`.

This is tracked as future work; see the
[MCP SDK ADR](adr/0001-mcp-sdk-choice.md) and related issues #5, #13.

## Docker

The image runs as a non-root user (UID 65532), which cannot read a file with
`600` permissions owned by you. Either keep the token out of the mounted file
and pass it with `-e` as shown (so the file can be `644`), or add
`--user "$(id -u):$(id -g)"`.

`make docker-run` sets the URL from `DOCKER_MESHERY_SERVER_URL` (default
`http://host.docker.internal:9081`), not from `MESHERY_SERVER_URL`. That way a
`localhost` URL exported for native runs does not leak into the container,
where it would point at the container itself. To override it:

```sh
DOCKER_MESHERY_SERVER_URL=http://other-host:9081 make docker-run
```

Use `make docker-run` only to try the image by hand; do not point an AI app at
it. Make prints its commands to stdout, which would corrupt the MCP stream.

## Logging

Logs are written to **stderr**. Stdout is reserved for MCP protocol messages,
so nothing else may be written there. `LOG_LEVEL` controls verbosity:

| Level | Shows |
|-------|-------|
| `debug` | Everything, including diagnostic detail |
| `info` | Startup information and normal operation (default) |
| `warn` | Warnings and errors only |
| `error` | Errors only |

## Errors

Configuration problems stop the server at startup. Each error is logged with a
short description, the probable cause and a suggested fix.

| Short description | Cause | Fix |
|-------------------|-------|-----|
| Unable to read the config file | The `--config` path does not exist, is not readable, or is not valid YAML | Check the path and the YAML syntax |
| Unable to decode configuration | A setting in the file has the wrong type | Make every setting a plain string |
| Invalid Meshery server URL | Malformed URL, a scheme other than `http`/`https`, or no host | Use a URL like `http://localhost:9081` |
| Invalid log level | `LOG_LEVEL` / `log.level` is not a supported level | Use `debug`, `info`, `warn` or `error` |

Error messages never include the token or any password embedded in the URL.

Error codes follow Meshery's
[error conventions](https://docs.meshery.io/project/contributing/contributing-error):
each package defines its errors in `error.go` with `"replace_me"` codes, which
Meshery's `errorutil` tool replaces with real ones. To check your errors
locally:

```sh
go run github.com/meshery/meshkit/cmd/errorutil -d . analyze -i ./helpers -o ./helpers
```

## For contributors

### How configuration is wired

- `internal/config` is the only package that knows about Viper, environment
  variables or the config file. `config.Load(path)` returns a validated
  `*config.Config` or an error.
- `main` parses `--config`, calls `config.Load` and passes plain values to the
  packages that need them. Other packages never read configuration themselves.
- The log level is stored as a plain name (`"info"`, …), independent of any
  logging library. `main` translates it for meshkit's logger in one function,
  `logrusLevel`.
- The token is a `config.Secret`, which redacts itself when printed or
  JSON-encoded. Convert it with `string(token)` only where it is actually used.

### Adding a setting

1. In `internal/config/config.go`, add a key constant (`"section.name"`) and an
   env var constant. The env var name is public: changing it breaks users'
   configs.
2. Add a field with a `mapstructure` tag to the right struct.
3. Call `v.SetDefault` for the key, even if the default is empty. Viper ignores
   environment variables for keys it has no default for.
4. Call `v.MustBindEnv(key, envVar)`.
5. Validate the value in `validate()`, and add an error to
   `internal/config/error.go` following meshkit's conventions.
6. Add test cases to `config_test.go`.
7. Add the setting to `config_example.yml`. A test loads that file, so a broken
   example fails the tests.
8. Update this document.

## Not yet available

The following are planned (see the [MCP SDK ADR](adr/0001-mcp-sdk-choice.md))
but not yet implemented:

- Keychain scope (`meshery.keychain_scope` / `MESHERY_KEYCHAIN_SCOPE`)
- Per-tool concurrency limit (`meshery.max_concurrent_calls`)
- Bounded retries (`meshery.max_retries`)
