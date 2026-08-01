# Copyright Meshery Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

include build/Makefile.core.mk
include build/Makefile.show-help.mk

BINARY        = meshery-mcp-server
DOCKER_IMAGE  = meshery-extensions/meshery-mcp-server

# ---------------------------------------------------------------------------
# MAINTENANCE
# ---------------------------------------------------------------------------

## Format source files with gofmt.
fmt:
	gofmt -l -s -w .

## Run golangci-lint across the module.
lint:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not found — see CONTRIBUTING.md"; exit 1; }
	golangci-lint run ./...

## Run unit tests.
test:
	go test ./...

## Remove build output.
clean:
	rm -rf bin

# ---------------------------------------------------------------------------
# LOCAL BUILDS
# ---------------------------------------------------------------------------

## Build meshery-mcp-server for the host platform.
build:
	go build \
		-ldflags="-s -w -X main.version=$(GIT_VERSION) -X main.commitsha=$(GIT_COMMITSHA)" \
		-o bin/$(BINARY) \
		./cmd/meshery-mcp-server

## Build and run the server locally.
run:
	./bin/$(BINARY)

# ---------------------------------------------------------------------------
# DOCKER BUILDS
# ---------------------------------------------------------------------------

## Build the container image via multi-stage Docker build.
docker-build:
	docker build \
		-t $(DOCKER_IMAGE):$(GIT_VERSION) \
		--build-arg GIT_VERSION=$(GIT_VERSION) \
		--build-arg GIT_COMMITSHA=$(GIT_COMMITSHA) \
		.

.PHONY: \
	fmt \
	lint \
	test \
	clean \
	build \
	run \
	docker-build
