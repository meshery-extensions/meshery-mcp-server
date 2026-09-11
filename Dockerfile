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

# syntax=docker/dockerfile:1

FROM golang:1.26.0-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG GIT_VERSION=v0.1.0
ARG GIT_COMMITSHA=unknown
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags "-s -w -X 'github.com/meshery-extensions/meshery-mcp-server/internal/version.Version=${GIT_VERSION}' -X 'github.com/meshery-extensions/meshery-mcp-server/internal/version.CommitSHA=${GIT_COMMITSHA}'" \
    -o /out/meshery-mcp-server ./cmd/meshery-mcp-server

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/meshery-mcp-server /meshery-mcp-server
COPY --from=build /src/web /web

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/meshery-mcp-server"]
