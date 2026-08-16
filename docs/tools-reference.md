# MCP Resources and Tools Reference

> **Status:** Meshery MCP Server is currently under active development.

This document describes the MCP resources currently present in the development implementation.

## MCP Resources

The current resource implementation exposes the following resource URIs:

| Resource URI | Purpose |
| --- | --- |
| `meshery://connections` | Returns Meshery connection information |
| `meshery://providers` | Returns Meshery provider information |
| `meshery://adapters` | Returns Meshery adapter information |
| `meshery://health` | Returns server health information |
| `meshery://environments` | Returns Meshery environment information |

These resources are registered by the resource implementation and are read by URI.

Unsupported resource URIs return an `ErrInvalidURI` error.

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

The current implementation returns sample development data containing a Kubernetes connection named `local-cluster`.

Example:

    {
      "timestamp": "2026-08-16T00:00:00Z",
      "connections": [
        {
          "id": "1",
          "name": "local-cluster",
          "type": "kubernetes",
          "status": "connected"
        }
      ]
    }

The timestamp is generated when the resource is read.

## `meshery://providers`

Returns Meshery provider information.

### Response

The response contains:

- `timestamp` — time at which the resource was read.
- `providers` — list of providers.

Each provider contains:

| Field | Description |
| --- | --- |
| `name` | Provider name |
| `status` | Provider status |
| `url` | Provider URL |

### Current Development Response

The current implementation returns sample data for the `Meshery` provider.

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

### Response

The response contains:

- `timestamp` — time at which the resource was read.
- `adapters` — list of adapters.

Each adapter contains:

| Field | Description |
| --- | --- |
| `name` | Adapter name |
| `version` | Adapter version |
| `status` | Adapter status |

### Current Development Response

The current implementation returns sample data for Istio and Linkerd.

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

Returns health information for the development server.

### Response

The response contains:

- `timestamp` — time at which the resource was read.
- `status` — overall health status.
- `components` — health status of individual components.

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

### Response

The response contains:

- `timestamp` — time at which the resource was read.
- `environments` — list of environments.

Each environment contains:

| Field | Description |
| --- | --- |
| `id` | Environment identifier |
| `name` | Environment name |
| `connections` | List of connection identifiers associated with the environment |

### Current Development Response

The current implementation returns sample `development` and `production` environments.

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

An unsupported URI returns an invalid-resource-URI error.

## MCP Tools

The MCP tool surface is still under active development.

The resources documented above are implemented as resource handlers and should not be described as MCP tools.

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
