# Development

Build, test, and contribute to meshery-mcp-server.

## Prerequisites

| Tool | Version | Needed for |
| :--- | :------ | :---------- |
| [Go](https://go.dev/doc/install) | as declared in [`go.mod`](../go.mod) | Building and testing |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | v2.x | `make lint` |
| [Docker](https://docs.docker.com/get-docker/) | any recent version | Container builds |
| [markdownlint](https://github.com/DavidAnson/markdownlint-cli) | any recent version | `make docs-lint` |

## Build

```sh
make build           # writes bin/meshery-mcp-server
go build -o bin/meshery-mcp-server ./cmd/meshery-mcp-server   # equivalent
```

## Test

```sh
make test               # go test ./...
make coverage           # go test -race ...
```

## Lint

```sh
make lint            # golangci-lint run ./...
make fmt             # gofmt -l -s -w .
```

## Markdown lint

```sh
make docs-lint       # markdownlint README.md docs/
```

## Docker

```sh
make docker-build    # Build the container image
make docker-run      # Build and run over stdio; forwards MESHERY_API_TOKEN
```

Inside a container, `localhost` refers to the container itself, so
`make docker-run` points the server at `http://host.docker.internal:9081` by
default. Override it with `DOCKER_MESHERY_SERVER_URL`.

## Client architecture

`internal/meshery` holds one HTTP client. `New(cfg)` validates the URL, timeout and limits, then builds a single `*http.Client` with dial, TLS and response-header timeouts. `Client.call` is the only HTTP path: it applies the per-tool semaphore, per-route rate budget, credentials and user agent, then classifies 401/403/429/503 with the existing status-code contract. `Call` requires a token; read-only wrappers and `Ping` do not, because a local Meshery Server needs none.

`Ping` calls the unauthenticated `GET /api/system/version` and returns a typed `VersionInfo`. Transport failures become a `ConnectError` that names the URL without credentials. Other non-2xx responses become an `APIError` with status, method, path (without query), the server's message and a category sentinel for `errors.Is` (`ErrNotFound`, `ErrConflict`, `ErrValidation`, `ErrServer`).

The domain wrappers (`environments.go`, `workspaces.go`, `designs.go`, `connections.go`, `registry.go`, `performance.go`) are thin read-only methods over `call`. They use Meshery Server routes and return typed page envelopes. GraphQL is deferred: none of these domains needs it today.

To add a wrapper: define the typed model from `meshery/schemas`, add a method that builds the route and query in a new file, call `c.getJSON(ctx, path, query, toolName, &out)`, and add `httptest` cases for success, empty results, malformed JSON and each relevant status. Never expose a token in an error or log line.

## CI overview

Every pull request runs three CI jobs (`.github/workflows/build-and-test.yml`):

1. **golangci-lint** — `golangci-lint run ./...`
2. **Build and unit tests** — `go mod tidy` (verified clean), `make build`,
   `make coverage`. Coverage runs with `go test -race -coverprofile=coverage.txt
   -covermode=atomic ./...`.
3. **Test Docker Image Build** — `make docker-run < /dev/null`

The test job runs on Go 1.27 (add versions to the matrix in `build-and-test.yml` once `go.mod` allows them).

## Release process

Releases are cut by pushing a version tag (`v0.1.0`, `v0.1.1`, etc.) to
`main`. The [`.github/workflows/release.yml`](../.github/workflows/release.yml)
workflow runs GoReleaser, which:

1. Builds binaries for `linux/amd64`, `linux/arm64`,
   `darwin/amd64`, `darwin/arm64`, `windows/amd64`
2. Creates `tar.gz` archives (`.zip` for Windows)
3. Generates a CycloneDX SBOM
4. Computes `checksums.txt`
5. Signs artifacts with Cosign (keyless, via GitHub OIDC)
6. Builds and publishes a multi-arch Docker image to
   `ghcr.io/meshery-extensions/meshery-mcp-server`
7. Attaches artifacts to the existing GitHub Release draft created by
   Release Drafter

### Cutting a release

1. On `main`, create and push a version tag:

   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. The release workflow runs automatically and publishes the release.
3. Verify the release at [GitHub Releases](https://github.com/meshery-extensions/meshery-mcp-server/releases).

Do not create tags on feature branches; only push tags to `main` from a clean,
merged state.

## Contributing

See the contributing guidelines in the [Meshery documentation](
https://docs.meshery.io/project/contributing). Key conventions:

- Conventional commit prefixes: `feat`, `fix`, `refactor`, `docs`, `test`,
  `chore`, `perf`, `ci`, `build`
- PR titles prefixed with the component: `[meshery]`, `[docs]`, `[ci]`, etc.
- Signed commits (`git commit -s`)
- Run `make lint`, `make test`, and `make docs-lint` before opening a PR
