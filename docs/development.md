# Development

Build, test, and contribute to the Meshery MCP Server.

> [!NOTE]
> **Early development.** The MCP server itself is not yet implemented: the
> binary loads and validates its configuration, logs a startup line, and exits.
> The [`internal/tools`](internal/tools),
> [`internal/resources`](internal/resources), and
> [`internal/prompts`](internal/prompts) packages are empty stubs waiting for
> tool/resource/prompt registrations.

## Prerequisites

| Tool | Version | Needed for |
|------|---------|------------|
| [Go](https://go.dev/doc/install) | as declared in [`go.mod`](go.mod) | Building and testing |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | v2.x | `make lint` |
| [Docker](https://docs.docker.com/get-docker/) | any recent version | Container builds only |
| [markdownlint](https://github.com/DavidAnson/markdownlint) | any recent version | `make docs-lint` |

## Build

```sh
make build            # writes bin/meshery-mcp-server
```

Equivalent:

```sh
go build -o bin/meshery-mcp-server ./cmd/meshery-mcp-server
```

## Run

```sh
make run                        # uses your shell's environment variables
make run ARGS="--config config.local.yaml"
```

The binary logs a startup line to stderr and exits:

```text
time="..." level=info msg="starting meshery-mcp-server version=v0.0.0-unknown commit=unknown meshery_url=http://localhost:9081 token_configured=false" app=meshery-mcp-server
```

## Test

```sh
make test              # go test ./...
make coverage          # go test -race -coverprofile=coverage.txt -covermode=atomic ./...
```

## Lint and format

```sh
make lint              # golangci-lint run ./...
make fmt               # gofmt -l -s -w .
make docs-lint         # markdownlint on README.md and docs/
```

## Docker

```sh
make docker-build      # builds meshery-extensions/meshery-mcp-server
make docker-run        # runs over stdio; forwards MESHERY_API_TOKEN
```

Inside a container, `localhost` refers to the container, so the default URL is
`http://host.docker.internal:9081`. Override with `DOCKER_MESHERY_SERVER_URL`.

## CI

Every pull request runs three jobs (see
[`.github/workflows/build-and-test.yml`](.github/workflows/build-and-test.yml):

1. **golangci-lint** — `golangci-lint run ./...`
2. **Build and unit tests** — `go mod tidy` (checked clean), `make build`,
   `make coverage`
3. **Test Docker Image Build** — `make docker-run < /dev/null`

Markdown-only changes are ignored by CI (`paths-ignore: ["**/*.md", "LICENSE"]`).

## Adding a new MCP tool / resource / prompt

The [`internal/tools`](internal/tools), [`internal/resources`](internal/resources),
and [`internal/prompts`](internal/prompts) packages are empty stubs on `master`.
When the MCP server is implemented (see the [MCP SDK ADR](adr/0001-mcp-sdk-choice.md)),
each package will expose a `Register` function and an `all*()` list:

- `internal/tools` — `allTools() []Tool` + `Register(s *mcp.Server) error`
- `internal/resources` — `allResources() []Resource` +
  `allTemplates() []ResourceTemplate` + `Register(...)`
- `internal/prompts` — `allPrompts() []Prompt` + `Register(...)`

To add a capability:

1. Add the item to the corresponding `all*()` slice in its package.
2. Implement its handler (SDK-agnostic signature: `context` + decoded args
   → result/error).
3. Wire it through `Register`, which the entrypoint
   (`cmd/meshery-mcp-server/main.go`) calls at startup.
4. Add the item to [docs/tools-reference.md](docs/tools-reference.md). The drift
   test (`test/tools_reference_test.go`) fails if a registered tool is missing
   from the doc.
5. Add unit tests and an end-to-end test over an in-memory client session
   (see `internal/tools/tools_test.go` for the pattern).

## Contribution guidelines

- Follow [Meshery's contributing overview](https://docs.meshery.io/project/contributing).
- Sign off every commit with `git commit -s` (`Signed-off-by:` trailer) per
  the [DCO](https://github.com/meshery/meshery/blob/master/CONTRIBUTING.md#signing-off-on-commits-developer-certificate-of-origin).
- Use conventional commit prefixes: `feat`, `fix`, `refactor`, `docs`, `test`,
  `chore`, `perf`, `ci`, `build`.
- Prefix PR titles with the component, e.g.
  `[meshery] Add MCP tools registration`.
- Use "Related to #<issue>" rather than "Closes" unless the issue is resolved.
- Follow the repo's [PR template](.github/PULL_REQUEST_TEMPLATE.md), including
  the "Signed commits" checkbox.
- Run `make lint`, `make fmt`, and `make coverage` before opening a PR.
- Docs changes (this directory, `README.md`) skip CI's `paths-ignore` but still
  must pass `make docs-lint` (zero markdownlint errors on the README).

## Community

- [Contributor Guides](https://docs.meshery.io/project/contributing)
- [Community Handbook](https://layer5.io/community/handbook)
- [Discussion Forum](https://meshery.io/community#community-forums)
- [Community Slack](https://slack.meshery.io/)
