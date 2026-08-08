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
- Configure the client with the appropriate Meshery base URL and cookie-based authentication. Meshery data routes use the `token` and `meshery-provider` cookies written by `mesheryctl system login`; the shared client should not assume an `Authorization: Bearer` flow for these routes.- Keep the MCP Server’s design and tools aligned with any changes in the REST client and Meshery’s REST APIs.

### Data shape and pagination

MCP tools will use Meshery REST APIs through the shared REST client. For the list-designs API, the known REST response fields are `page`, `pageSize`, `totalCount`, and `patterns`.

The shared REST client should use explicit JSON tags, or equivalent field mapping, to correctly parse Meshery’s camelCase response fields. The `list_designs` MCP tool should expose a documented, stable response contract that identifies the returned design data and pagination metadata.

Where a tool transforms a REST response, the transformation should be explicit and documented so MCP clients and AI agents receive predictable, stable results. Tool definitions will continue to incorporate relevant findings from the existing proof-of-concept while remaining aligned with future Meshery REST client improvements.

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
