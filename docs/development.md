# Development

Build, test, and contribute to meshery-mcp-server.

## Prerequisites

| Tool | Version | Needed for |
| :--- | :------ | :---------- |
| [Go](https://go.dev/doc/install) | as declared in [`go.mod`](go.mod) | Building and testing |
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

## CI overview

Every pull request runs three CI jobs (`.github/workflows/build-and-test.yml`):

1. **golangci-lint** — `golangci-lint run ./...`
2. **Build and unit tests** — `go mod tidy` (verified clean), `make build`,
   `make coverage`. Coverage runs with `go test -race -coverprofile=coverage.txt
   -covermode=atomic ./...`.
3. **Test Docker Image Build** — `make docker-run < /dev/null`

The test job runs on a matrix of Go versions (currently `1.27` and `1.26`).

## Release process

Releases are cut by pushing a version tag (`v0.1.0`, `v0.1.1`, etc.) to
`main`. The [`.github/workflows/release.yml`](.github/workflows/release.yml)
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
