# Meshery MCP Server – Design

## Goals

The Meshery MCP Server extension is intended to:

- Provide an AI‑native interface to Meshery via the Model Context Protocol (MCP).
- Focus on Meshery designs and related artifacts (snapshots, deployment dry‑runs, performance test results).
- Enable tools that make it easy for AI agents and other MCP clients to create, inspect, export, and analyze Meshery designs.

## Architecture Overview

At a high level, the MCP Server will consist of:

- **MCP Server**: The server process that implements the MCP protocol and exposes tools.
- **Meshery REST client**: A shared client (as implemented in the Meshery REST client work) used by MCP tools to talk to Meshery over REST.
- **MCP tools**: Individual tools for working with Meshery designs and test results.

The MCP Server acts as a thin layer that translates MCP tool calls into Meshery REST API calls and returns structured results suitable for MCP clients.

## Transport considerations

The MCP Server will support MCP clients through a transport strategy that is aligned with the MCP specification and maintainer direction.

The existing Meshery MCP proof-of-concept demonstrates both stdio and Streamable HTTP transports. This provides useful implementation input, while the transport adopted by the main MCP Server will remain aligned with the project foundation work and maintainer decisions.

## Meshery REST Integration

The MCP Server will interact with Meshery using **REST APIs only** (no GraphQL), aligned with the direction set by the Meshery maintainers.

Key aspects:

- Use the shared Meshery REST client as the single integration layer between the MCP Server and Meshery.
- Configure the client with the appropriate Meshery base URL and authentication.
- Keep the MCP Server’s design and tools aligned with any changes in the REST client and Meshery’s REST APIs.

### Data shape and pagination

MCP tools will use Meshery REST APIs through the shared REST client. Existing proof-of-concept work shows that some Meshery API responses include camelCase fields such as `pageSize` and `totalCount`.

Each MCP tool should document its response shape and pagination behavior. Where a tool transforms a REST response, the transformation should be explicit so MCP clients and AI agents receive predictable, stable results.

The tool definitions will continue to incorporate relevant findings from the proof-of-concept, including the existing `list_designs` implementation, while remaining aligned with future Meshery REST client improvements.

## Initial MCP Tools

The initial scope of MCP tools is expected to cover:

- **List Meshery designs**: Retrieve a list of designs available in Meshery.
- **Export a design**: Export a selected design in a requested file format (for example, YAML or JSON).
- **Snapshot a design**: Create a snapshot of a design at a point in time.
- **Retrieve deployment dry‑run results**: Access dry‑run outputs associated with a design.
- **Retrieve performance test results**: Access performance test results associated with a design.

These tools will all be implemented on top of the shared Meshery REST client and will return structured JSON responses and human‑readable error messages, suitable for use by MCP clients and AI agents.

## Future tool candidates

The Meshery MCP proof-of-concept also demonstrates read-only access to MeshSync-discovered Kubernetes resources and Kubernetes cluster connections. These are promising future MCP tool candidates.

Before they are added to the main MCP Server scope, they should be proposed as separate issues and aligned with maintainer priorities, the shared REST client, and the project transport strategy.
