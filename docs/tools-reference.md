# MCP Resources and Tools Reference

> **Status:** Meshery MCP Server is currently under active development.

This document describes the MCP resources currently present in the development implementation.

## MCP Resources

The current resource implementation exposes:

| Resource URI | Purpose |
| --- | --- |
| `meshery://connections` | Returns Meshery connection information |
| `meshery://providers` | Returns Meshery provider information |
| `meshery://adapters` | Returns Meshery adapter information |
| `meshery://health` | Returns server health information |
| `meshery://environments` | Returns Meshery environment information |

Unsupported resource URIs return the implementation's `ErrInvalidURI` error.

## `meshery://connections`

Returns connection information.

### Response

The response contains:

- `timestamp` — time at which the resource was read.
- `connections` — list of connections.

Each connection contains:

| Field | Description |
| --- | --- |
| `id` | Connection identifier |
| `name` | Connection name |
| `type` | Connection type |
| `status` | Connection status |

### Current Development Response

The current implementation returns static sample development data containing two Kubernetes connections.

This example represents development/sample data only and does not represent live production Meshery state.

Example:

    {
      "timestamp": "2026-08-16T00:00:00Z",
      "connections": [
        {
          "id": "conn-1",
          "name": "local-cluster",
          "type": "kubernetes",
          "status": "connected"
        },
        {
          "id": "conn-2",
          "name": "production-cluster",
          "type": "kubernetes",
          "status": "connected"
        }
      ]
    }

The timestamp is generated when the resource is read.

## `meshery://providers`

Returns Meshery provider information.

Example:

    {
      "timestamp": "2026-08-16T00:00:00Z",
      "providers": [
        {
          "name": "Meshery",
          "status": "active",
          "url": "http://localhost"
        }
      ]
    }

The timestamp is generated when the resource is read.

## `meshery://adapters`

Returns Meshery adapter information.

Example:

    {
      "timestamp": "2026-08-16T00:00:00Z",
      "adapters": [
        {
          "name": "Istio",
          "version": "1.18",
          "status": "running"
        },
        {
          "name": "Linkerd",
          "version": "2.13",
          "status": "stopped"
        }
      ]
    }

The timestamp is generated when the resource is read.

## `meshery://health`

Returns development server health information.

Example:

    {
      "timestamp": "2026-08-16T00:00:00Z",
      "status": "healthy",
      "components": {
        "server": "ok",
        "database": "ok",
        "adapters": "ok"
      }
    }

The timestamp is generated when the resource is read.

## `meshery://environments`

Returns Meshery environment information.

Each environment contains an environment ID, name, and connection IDs.

The documented connection IDs below correspond to the IDs in the `meshery://connections` example.

Example:

    {
      "timestamp": "2026-08-16T00:00:00Z",
      "environments": [
        {
          "id": "env-1",
          "name": "development",
          "connections": ["conn-1"]
        },
        {
          "id": "env-2",
          "name": "production",
          "connections": ["conn-2"]
        }
      ]
    }

The timestamp is generated when the resource is read.

## Reading Resources

Resources are selected using their URI.

Supported URIs:

    meshery://connections
    meshery://providers
    meshery://adapters
    meshery://health
    meshery://environments

The resource router dispatches each supported URI to its corresponding handler.

An unsupported URI returns `ErrInvalidURI`.

## MCP Tools

The MCP tool surface is still under active development.

The resources documented above are resource handlers and should not be described as MCP tools.

As MCP tools are implemented, this section should be expanded with:

- Tool name
- Purpose
- Input schema
- Output schema
- Error behavior
- Examples
- Corresponding tests

## Current Limitations

The resource handlers in the current development implementation return sample/static data.

They should therefore be treated as development functionality rather than a complete representation of live Meshery state.

As the implementation evolves to retrieve live data through the Meshery API client, this documentation should be updated to describe the actual API-backed behavior.

## Keeping This Reference Up to Date

When a resource or tool changes:

1. Update this document.
2. Document the exact URI or tool name.
3. Document request/input fields.
4. Document response/output fields.
5. Document errors.
6. Add a working example where applicable.
7. Keep the documentation synchronized with tests and implementation.
