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

GO ?= go
BINARY := meshery-mcp-server
BIN_DIR := bin

GIT_VERSION ?= $(shell git describe --tags --always --dirty)
GIT_COMMITSHA ?= $(shell git rev-parse --short HEAD)
LDFLAGS := -s -w \
	-X 'github.com/meshery-extensions/meshery-mcp-server/internal/version.Version=$(GIT_VERSION)' \
	-X 'github.com/meshery-extensions/meshery-mcp-server/internal/version.CommitSHA=$(GIT_COMMITSHA)'

.DEFAULT_GOAL := build

.PHONY: build run test vet lint fmt clean docker-build help

## Build the Meshery MCP server binary.
build:
	mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/meshery-mcp-server

## Run the Meshery MCP server over stdio.
run: build
	$(BIN_DIR)/$(BINARY)

## Run all Go tests.
test:
	$(GO) test --short ./... -race -coverprofile=coverage.txt -covermode=atomic

## Vet the Go source.
vet:
	$(GO) vet ./...

## Format the Go source with gofmt and goimports.
fmt:
	golangci-lint fmt

## Lint the Go source with golangci-lint.
lint:
	golangci-lint run --timeout=10m

## Run the architectural ruler (10-metric scorecard + micro-benchmark).
.PHONY: ruler
ruler:
	$(GO) run ./cmd/ruler/main.go

## Run tests and the architectural ruler.
.PHONY: benchmark
benchmark: test ruler

## Remove build artifacts.
clean:
	rm -rf $(BIN_DIR) coverage.txt

## Build the container image.
docker-build:
	docker build \
		--build-arg GIT_VERSION=$(GIT_VERSION) \
		--build-arg GIT_COMMITSHA=$(GIT_COMMITSHA) \
		-t meshery/meshery-mcp-server .

## Show this help.
help:
	@echo "Usage: make [target]"; echo; grep -hE '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | column -t -s '	'
