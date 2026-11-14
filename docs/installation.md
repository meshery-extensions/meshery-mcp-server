# Installation

meshery-mcp-server can be installed from source, from pre-built binaries, or as
a Docker image. All installation methods produce a binary that connects to a
Meshery server via HTTP(S) using the `MESHERY_SERVER_URL` and
`MESHERY_API_TOKEN` environment variables.

## Prerequisites

- **Go 1.27.1 or newer** (as declared in [`go.mod`](go.mod)) — required for
  building from source
- **Docker** — required for the container image
- **cosign** (optional) — required for verifying signatures on release artifacts

## Install from source

Clone the repository and build with `make`:

```sh
git clone https://github.com/meshery-extensions/meshery-mcp-server.git
cd meshery-mcp-server
make build           # writes bin/meshery-mcp-server
```

The binary is built with `-ldflags` that embed the version and commit SHA from
git tags. The version is displayed in the startup log line.

## Install pre-built binaries

Pre-built binaries are published as GitHub Release assets for every tagged
release (`v0.1.0` and later). Download the appropriate archive for your
platform from the [Releases page](https://github.com/meshery-extensions/meshery-mcp-server/releases).

Each release includes:

- `linux_amd64.tar.gz`
- `linux_arm64.tar.gz`
- `darwin_amd64.tar.gz`
- `darwin_arm64.tar.gz`
- `windows_amd64.zip`

Verify the checksums against `checksums.txt`, and verify the signature with
Cosign if available:

```sh
# Verify checksums
sha256sum -c checksums.txt

# Verify signature (requires cosign)
cosign verify \
  ghcr.io/meshery-extensions/meshery-mcp-server@<digest> \
  --certificate-identity=ghcr.io/meshery-extensions/meshery-mcp-server \
  --certificate-issuer=https://token.actions.githubusercontent.com
```

## Install as a container image

The Docker image is published to
[ghcr.io/meshery-extensions/meshery-mcp-server](https://github.com/meshery-extensions/meshery-mcp-server/pkgs/container/meshery-mcp-server)
for every release tag.

```sh
docker pull ghcr.io/meshery-extensions/meshery-mcp-server:v0.1.0
```

Run the image over stdio:

```sh
docker run -i --rm \
  --add-host=host.docker.internal:host-gateway \
  -e MESHERY_SERVER_URL=http://localhost:9081 \
  -e MESHERY_API_TOKEN \
  ghcr.io/meshery-extensions/meshery-mcp-server:v0.1.0
```

The image runs as a non-root user (UID 65532) and is built from
[`Dockerfile`](Dockerfile) using `CGO_ENABLED=0` for a static, minimal
runtime image.

## Verify the installation

Run the binary to confirm it starts correctly:

```sh
meshery-mcp-server
# time="..." level=info msg="starting meshery-mcp-server version=v0.1.0 ..." app=meshery-mcp-server
```

The `--version` flag is not implemented as a standalone flag; the version is
emitted in the startup log line via build-time `-ldflags`.

## Next steps

Once the server is running, connect an MCP client such as Claude Desktop using
the configuration described in [docs/config.md](docs/config.md#from-an-ai-app-mcp-client).
