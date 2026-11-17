include build/Makefile.core.mk
include build/Makefile.show-help.mk

BINARY       := meshery-mcp-server
DOCKER_IMAGE := meshery-extensions/meshery-mcp-server

# Inside a container, localhost is the container
# itself; host.docker.internal reaches a Meshery server on this machine.
DOCKER_MESHERY_SERVER_URL ?= http://host.docker.internal:9081

ifeq ($(strip $(GIT_VERSION)),)
GIT_VERSION := v0.0.0-unknown
endif
ifeq ($(strip $(GIT_COMMITSHA)),)
GIT_COMMITSHA := unknown
endif

LDFLAGS := -s -w -X main.version=$(GIT_VERSION) -X main.commitsha=$(GIT_COMMITSHA)

# ---------------------------------------------------------------------------
# MAINTENANCE
# ---------------------------------------------------------------------------

.PHONY: fmt
## Format source files with gofmt.
fmt:
	gofmt -l -s -w .

.PHONY: lint
## Run golangci-lint across the module.
lint:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not found: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run ./...

.PHONY: docs-lint
## Lint Markdown files.
docs-lint:
	@command -v markdownlint >/dev/null || { echo "markdownlint not found: npm install -g markdownlint-cli"; exit 1; }
	markdownlint README.md docs/

.PHONY: test
## Run unit tests.
test:
	go test ./...

.PHONY: coverage
## Run unit tests with the race detector and write coverage.txt.
coverage:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: clean
## Remove build output.
clean:
	rm -rf bin coverage.txt

# ---------------------------------------------------------------------------
# LOCAL BUILDS
# ---------------------------------------------------------------------------

.PHONY: build
## Build meshery-mcp-server for the host platform.
build:
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/meshery-mcp-server

.PHONY: run
## Build and run the server locally. Pass flags with ARGS="--config dev.yaml".
run: build
	./bin/$(BINARY) $(ARGS)

# ---------------------------------------------------------------------------
# DOCKER BUILDS
# ---------------------------------------------------------------------------

.PHONY: docker-build
## Build the container image via multi-stage Docker build.
docker-build:
	docker build \
		-t $(DOCKER_IMAGE):$(GIT_VERSION) \
		--build-arg GIT_VERSION=$(GIT_VERSION) \
		--build-arg GIT_COMMITSHA=$(GIT_COMMITSHA) \
		.

.PHONY: docker-run
## Build and run the image over stdio. Pass flags with ARGS="--config ...".
docker-run: docker-build
	docker run -i --rm \
		--add-host=host.docker.internal:host-gateway \
		-e MESHERY_SERVER_URL=$(DOCKER_MESHERY_SERVER_URL) \
		-e MESHERY_API_TOKEN \
		$(DOCKER_IMAGE):$(GIT_VERSION) $(ARGS)
