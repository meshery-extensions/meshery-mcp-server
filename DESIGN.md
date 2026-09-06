# Meshery MCP Server – Design

## Goals

The Meshery MCP Server extension is intended to:

- Provide an AI-native interface to Meshery through the Model Context Protocol (MCP).
- Enable AI agents and other MCP clients to create, inspect, export, and analyze Meshery designs and related artifacts.
- Establish clear, stable tool contracts that can grow with Meshery capabilities without duplicating server or client infrastructure.

## Architecture Overview

At a high level, the MCP Server consists of:

- **MCP Server**: The server process that implements MCP and exposes tools to MCP clients.
- **Shared Meshery client**: The common client layer used by MCP tools to communicate with Meshery Server.
- **MCP tools**: Focused tool implementations for Meshery designs, deployment workflows, and test results.

The MCP Server is a thin layer that translates MCP tool calls into Meshery client operations and returns structured, predictable results suitable for MCP clients.

The implementation builds on the repository foundation established by the MCP Server scaffold. This includes the selected Go MCP SDK, an initial stdio transport, the shared server and tool-registration structure, and common configuration support. Tool implementations should extend this foundation instead of introducing parallel server or registration patterns.

## Transport Considerations

The initial MCP Server transport is stdio. This aligns with the repository foundation and provides a focused starting point for local MCP client integrations.

The MCP Server core should remain transport-agnostic so additional transports can be considered later when they are supported by the project foundation and maintainer priorities. The existing proof of concept provides useful implementation input for future transport work, but it does not determine the main project transport strategy.

## Meshery REST Integration

The initial MCP tools will use Meshery's existing REST APIs through the shared Meshery client.

REST is the practical initial integration path because the required Meshery Server APIs, external-client workflows, and JSON response shapes already exist. Introducing a separate gRPC integration would require new protobuf contracts and corresponding server-side services for the required resources.

Streaming-oriented capabilities, including MeshSync or other live-state workflows, are outside the initial scope. They can be evaluated later without changing the core MCP tool architecture.

### Shared client responsibilities

The shared Meshery client is the single integration layer between MCP tools and Meshery Server. It is responsible for:

- Meshery Server base-URL configuration.
- Authentication configuration and request handling.
- REST request execution and error handling.
- Explicit mapping of Meshery API response fields.

MCP tools must use the shared client rather than directly construct REST requests, authorization headers, or authentication cookies. This keeps individual tools independent of the final authentication implementation and consistent with the repository configuration contract.

### Authentication contract

The shared Meshery client is the only layer that applies authentication to outbound REST requests. MCP tools must not read credentials, construct authorization headers, or manage cookies directly.

The initial authentication contract uses Meshery session cookies:

- `token`
- `meshery-provider`

Meshery CLI login writes these credentials to `~/.meshery/auth.json`. The shared client is responsible for loading configured credentials, attaching the required cookies to applicable REST requests, and keeping authentication behavior consistent across tools.

The initial MCP tool scope does not use `Authorization: Bearer <token>` for the Meshery data routes. If another endpoint requires a different mechanism later, that mechanism must be implemented in the shared client rather than separately by individual tools.

The client must not log authentication cookies, credential values, authorization headers, or complete URLs containing credentials.

### Data Shape and Pagination

For the list-designs API, the `/api/pattern` REST response fields include `page`, `pageSize`, `totalCount`, and `patterns`.

The shared Meshery client should use explicit JSON tags or equivalent field mapping to correctly parse Meshery's camelCase response fields. The `list_designs` MCP tool should expose a documented, stable response contract that identifies returned design data and pagination metadata.

Where a tool transforms a REST response, the transformation should be explicit and documented so MCP clients and AI agents receive predictable, stable results.

## Initial MCP Tools

The first implementation slice is a read-only design-listing tool. The remaining tools below are planned contracts and should be implemented only after their Meshery API behavior is validated.

| MCP tool | Required inputs | Optional inputs | Structured result | Safety | Error behavior |
|---|---|---|---|---|---|
| `server_info` | None | None | Meshery Server version and supported capability metadata | Read-only | Authentication, connectivity, and upstream failures return a tool error without exposing credentials. |
| `list_designs` | None | `page` (integer, minimum 1), `page_size` (integer, 1–100), `search` (string) | `designs` array plus `page`, `page_size`, and `total_count` | Read-only | Invalid input returns an invalid-params error. Authentication, connectivity, and upstream failures return a tool error. |
| `export_meshery_design` | `design_id` (string) | `format` (`yaml` or `json`; default `yaml`) | `design_id`, `format`, and exported `content` | Read-only | Invalid input or unsupported format returns invalid params. Not-found, authentication, and upstream failures return a tool error. |
| `snapshot_meshery_design` | `design_id` (string) | `name` (string) | Snapshot identifier and metadata | Pending API confirmation | Invalid input, not-found, authentication, and upstream failures return a tool error. |
| `get_deployment_dry_run` | `design_id` (string) | None | `design_id`, status, and dry-run output | Read-only | Invalid input, not-found, authentication, and upstream failures return a tool error. |
| `get_performance_test_results` | `design_id` (string) | `page` (integer, minimum 1), `page_size` (integer, 1–100) | `results` array plus `page`, `page_size`, and `total_count` | Read-only | Invalid input returns invalid params. Authentication and upstream failures return a tool error. |

The final safety classification for `snapshot_meshery_design` depends on whether the Meshery API persists a snapshot. It must be classified as state-changing if it creates or stores a snapshot; otherwise it may be classified as read-only.

Tool names and input fields are MCP-facing contracts and should remain stable once released. The shared Meshery client owns REST endpoint paths, HTTP status handling, request serialization, and Meshery API response decoding. Individual tools own MCP input validation and mapping client/domain results into the documented MCP result shape.

Each concrete tool registers through the server's `Registrant` interface. The first tools may receive the concrete shared Meshery client during construction.
When `list_designs` is implemented, the project should evaluate whether a narrow tool-facing client interface materially improves testability or separation before introducing another abstraction.

## Future Tool Candidates

The Meshery MCP proof of concept also demonstrates read-only access to MeshSync-discovered Kubernetes resources and Kubernetes cluster connections. These are promising future tool candidates.

Before they are added to the main MCP Server scope, they should be proposed as separate issues and aligned with maintainer priorities, the shared Meshery client, and the project transport strategy.

## Tool Safety and Confirmation

Tool descriptions and registrations must declare whether a tool is read-only, state-changing, or potentially destructive.

Read-only tools must not modify Meshery state. The MCP SDK's read-only defaults should be used where applicable. A tool that changes state or has destructive effects must explicitly override those defaults and describe its side effects.

Potentially destructive operations must require explicit confirmation behavior before execution when they are introduced.

Safety metadata belongs with the concrete MCP tool definition so clients and AI agents can discover it programmatically; this document defines the intended contract for those annotations.
