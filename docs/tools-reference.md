# MCP tools reference

This page lists the MCP tools exposed by the Meshery MCP Server, together with
their inputs, outputs, and example calls.

## Implemented tools

The following tools are registered against the Meshery Cloud API
(`meshery.cloud`). All are read-only: they only `GET` from the Meshery Cloud
API and never modify data.

### claims

List identities (claims) registered in Meshery Cloud.

**Input:**

| Field       | Type     | Required | Description                                         |
|-------------|----------|----------|-----------------------------------------------------|
| search      | string   | No       | Filter by case-insensitive substring match on name  |
| page        | integer  | No       | Zero-indexed page number (default 0)                |
| page_size   | integer  | No       | Items per page, 1-100 (default 10)                  |

**Output:**

| Field   | Type     | Description                    |
|---------|----------|--------------------------------|
| claims  | array    | List of claim summaries        |
| total   | integer  | Total number of matching claims|

**Example call:**

```json
{
  "name": "list_models",
  "arguments": {
    "search": "kubernetes",
    "page": 0,
    "page_size": 10
  }
}
```

### get_model

Get full model details from Meshery Cloud.

**Input:**

| Field     | Type   | Required | Description                            |
|-----------|--------|----------|----------------------------------------|
| model_id  | string | Yes      | The model ID (name) to look up         |

**Output:**

| Field   | Type     | Description                        |
|---------|----------|------------------------------------|
| model   | object   | Model definition including all components |

**Example call:**

```json
{
  "name": "get_model",
  "arguments": {
    "model_id": "kubernetes"
  }
}
```

### search_components

Search components across all models in Meshery Cloud.

**Input:**

| Field     | Type   | Required | Description                                               |
|-----------|--------|----------|-----------------------------------------------------------|
| search    | string | No       | Filter by case-insensitive substring match on component name |
| kind      | string | No       | Filter by component kind (e.g. `Deployment`)              |
| model     | string | No       | Filter by the model the component belongs to              |

**Output:**

| Field         | Type     | Description                        |
|---------------|----------|------------------------------------|
| content_items | array    | List of matching component summaries |
| total         | integer  | Total number of matching components |

**Example call:**

```json
{
  "name": "search_components",
  "arguments": {
    "search": "Deployment",
    "model": "kubernetes"
  }
}
```

### get_component

Get detailed component schema from Meshery Cloud.

**Input:**

| Field         | Type   | Required | Description                                                       |
|---------------|--------|----------|-------------------------------------------------------------------|
| component_id  | string | Yes      | The component kind/name (e.g. `Deployment`) to look up          |

**Output:**

| Field     | Type     | Description                              |
|-----------|----------|------------------------------------------|
| component | object   | Full component definition including JSON Schema |

**Example call:**

```json
{
  "name": "get_component",
  "arguments": {
    "component_id": "Deployment"
  }
}
```

### list_relationships

List available component relationship types for a model.

**Input:**

| Field   | Type   | Required | Description                                   |
|---------|--------|----------|-----------------------------------------------|
| model   | string | Yes      | The model whose relationships to list         |

**Output:**

| Field         | Type     | Description                    |
|---------------|----------|----------------------------------|
| relationships | array    | List of relationship definitions |
| total         | integer  | Total number of relationships    |

**Example call:**

```json
{
  "name": "list_relationships",
  "arguments": {
    "model": "kubernetes"
  }
}
```

## Drift test

A Go test (`test/tools_reference_test.go`) asserts that every tool registered
in the server is listed on this page. When a new tool is added, its name must
be added to the table above before the change can be merged.

## Not yet available

The Meshery MCP Server is in early development. The following planned tool
categories from the [README](README.md#planned-scope) are not yet implemented:

- Design management (create, retrieve, snapshot, dry-run)
- Performance testing (define and run)

These will be added as the MCP server implementation progresses.
