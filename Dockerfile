ARG GO_VERSION=1.26.4

# Binary base
FROM golang:${GO_VERSION} AS base
ARG GIT_VERSION
ARG GIT_COMMITSHA
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=$GIT_VERSION -X main.commitsha=$GIT_COMMITSHA" \
    -o /out/meshery-mcp-server ./cmd/meshery-mcp-server

# Runtime image
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=base /out/meshery-mcp-server /meshery-mcp-server
ENTRYPOINT ["/meshery-mcp-server"]