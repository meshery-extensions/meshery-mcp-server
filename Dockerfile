ARG GO_VERSION=1.27.1

# ---------------------------------------------------------------------------
# Build stage: has the Go toolchain; produces a static binary.
# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION} AS builder
WORKDIR /src

# Dependencies first, so this layer is reused until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Declared late: a new commit SHA only invalidates the steps below this line.
ARG GIT_VERSION=v0.0.0-unknown
ARG GIT_COMMITSHA=unknown
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${GIT_VERSION} -X main.commitsha=${GIT_COMMITSHA}" \
    -o /out/meshery-mcp-server ./cmd/meshery-mcp-server

# ---------------------------------------------------------------------------
# Runtime stage: only the binary, no toolchain, no shell, non-root user.
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/meshery-mcp-server /meshery-mcp-server
ENTRYPOINT ["/meshery-mcp-server"]
