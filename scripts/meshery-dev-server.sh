#!/usr/bin/env bash
#
# Builds and runs a Meshery Server from source, for testing an MCP server
# against something real.
#
# The published image is amd64 only and crashes under emulation on arm64 during
# content seeding, which is why so much of this work gets tested against mocks.
# The source builds and runs natively.
#
#   ./scripts/meshery-dev-server.sh                 # clones into ./.meshery-src
#   ./scripts/meshery-dev-server.sh /path/to/meshery
#
# Setup needs network: it clones meshery/meshery when the checkout is absent,
# and the build downloads a large module tree. Once built, running it needs no
# remote provider and no credentials, because PROVIDER=Local selects the
# built-in provider.
#
# It listens on every interface, not loopback. Meshery has no bind-address
# option: server/router/server.go passes fmt.Sprintf(":%d", port) straight to
# http.ListenAndServe. Treat it as a development server and do not run it on a
# network you do not trust.
set -euo pipefail

PORT="${PORT:-9081}"
SRC="${1:-.meshery-src}"

# The revision every observation in docs/INTEGRATION.md was made against.
# Override with MESHERY_REF to test a newer Meshery, knowing the seed counts
# and endpoint behaviour recorded there may then differ.
MESHERY_REF="${MESHERY_REF:-e6ed2de164b42d805b78dd1cdb3c4b415e8686eb}"

if [ ! -d "$SRC" ]; then
  echo "fetching meshery/meshery at $MESHERY_REF into $SRC"
  mkdir -p "$SRC"
  git -C "$SRC" init -q
  git -C "$SRC" remote add origin https://github.com/meshery/meshery.git
  git -C "$SRC" fetch -q --depth 1 origin "$MESHERY_REF"
  git -C "$SRC" checkout -q FETCH_HEAD
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required; meshery/meshery targets the version in its go.mod" >&2
  exit 1
fi

BIN="$(cd "$SRC" && pwd)/meshery-server"
echo "building the server (first build pulls a large dependency tree)"
(cd "$SRC/server/cmd" && go build -o "$BIN" .)

echo "serving on port $PORT with the Local provider"
echo "note: Meshery binds every interface, it has no loopback option; dev use only"
cd "$SRC/server/cmd"
PORT="$PORT" \
PROVIDER=Local \
USE_GO_POLICY_ENGINE=true \
LOG_LEVEL="${LOG_LEVEL:-3}" \
APP_PATH=./apps.json \
KEYS_PATH=../../server/permissions/keys.csv \
MESHSYNC_DEFAULT_DEPLOYMENT_MODE=operator \
exec "$BIN"
