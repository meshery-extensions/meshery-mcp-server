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

## Meshery REST Integration

The MCP Server will interact with Meshery using **REST APIs only** (no GraphQL), aligned with the direction set by the Meshery maintainers.

Key aspects:

- Use the shared Meshery REST client as the single integration layer between the MCP Server and Meshery.
- Configure the client with the appropriate Meshery base URL and authentication.
- Keep the MCP Server’s design and tools aligned with any changes in the REST client and Meshery’s REST APIs.

## Initial MCP Tools

The initial scope of MCP tools is expected to cover:

- **List Meshery designs**: Retrieve a list of designs available in Meshery.
- **Export a design**: Export a selected design in a requested file format (for example, YAML or JSON).
- **Snapshot a design**: Create a snapshot of a design at a point in time.
- **Retrieve deployment dry‑run results**: Access dry‑run outputs associated with a design.
- **Retrieve performance test results**: Access performance test results associated with a design.

These tools will all be implemented on top of the shared Meshery REST client and will return structured JSON responses and human‑readable error messages, suitable for use by MCP clients and AI agents.
