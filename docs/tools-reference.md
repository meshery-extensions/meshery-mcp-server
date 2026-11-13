# MCP tools reference

This page lists the MCP tools exposed by the Meshery MCP Server, together with
their inputs, outputs, and example calls.

## Implemented tools

The following tools are registered against the Meshery server
(`MESHERY_SERVER_URL`). Tools marked "read-only" only `GET` from the Meshery
server and never modify data. Mutating tools (`create_design`,
`deploy_design`, `undeploy_design`, `delete_design`) require a `confirm: true`
input to prevent accidental execution.

### list_designs

List all designs in the workspace.

**Input:**

| Field       | Type     | Required | Description                                         |
|-------------|----------|----------|-----------------------------------------------------|
| page        | integer  | No       | Zero-indexed page number of the result set. Default: 0 |
| page_size   | integer  | No       | Number of items per page (1-100). Default: 10       |
| search      | string   | No       | Filter results by case-insensitive substring match on name. Max length: 255 |

**Output:**

| Field   | Type     | Description                     |
|---------|----------|---------------------------------|
| designs | array    | List of design summaries        |
| total   | integer  | Total number of matching designs|

**Example call:**

```json
{
  "name": "list_designs",
  "arguments": {
    "search": "bookinfo",
    "page": 0,
    "page_size": 10
  }
}
```

### get_design

Get full design details by ID from Meshery Cloud.

**Input:**

| Field      | Type   | Required | Description                             |
|------------|--------|----------|-----------------------------------------|
| design_id  | string | Yes      | The design ID to look up. Min length: 1 |

**Output:**

| Field     | Type     | Description                              |
|-----------|----------|------------------------------------------|
| design    | object   | Full design object (name, components, YAML) |

**Example call:**

```json
{
  "name": "get_design",
  "arguments": {
    "design_id": "d1"
  }
}
```

### create_design

Create a new design from YAML/JSON content. This is a mutating tool — it
creates a new design in Meshery Cloud. Requires `confirm: true`.

**Input:**

| Field        | Type   | Required | Description                                               |
|--------------|--------|----------|-----------------------------------------------------------|
| name         | string | Yes      | Human-readable name for the design. Min length: 1, max: 255 |
| design_yaml  | string | Yes      | The design body as YAML or JSON text. Max length: 16777216 |
| confirm      | boolean| No       | Must be `true` to confirm creation. Default: false        |

**Output:**

| Field | Type   | Description         |
|-------|--------|---------------------|
| id    | string | Created design ID   |
| name  | string | Design name         |

**Example call:**

```json
{
  "name": "create_design",
  "arguments": {
    "name": "my-design",
    "design_yaml": "apiVersion: meshery.io/v1beta1\nkind: Design\nmetadata:\n  name: my-design\nspec:\n  services:\n    - name: productpage\n      type: Deployment",
    "confirm": true
  }
}
```

### deploy_design

Deploy a design to a Kubernetes cluster. This is a mutating tool — it modifies
cluster state. Requires `confirm: true`.

**Input:**

| Field      | Type   | Required | Description                                        |
|------------|--------|----------|----------------------------------------------------|
| design_id  | string | Yes      | The design ID to deploy. Min length: 1             |
| cluster_id | string | Yes      | The K8sContext ID of the target cluster. Min: 1    |
| confirm    | boolean| No       | Must be `true` to confirm deployment. Default: false |

**Output:**

| Field     | Type   | Description                |
|-----------|--------|----------------------------|
| design_id | string | The deployed design ID     |
| cluster_id| string | The target cluster ID      |
| status    | string | Deployment status ("deployed") |

**Example call:**

```json
{
  "name": "deploy_design",
  "arguments": {
    "design_id": "d1",
    "cluster_id": "cluster-1",
    "confirm": true
  }
}
```

### undeploy_design

Undeploy a design from a Kubernetes cluster. This is a mutating tool — it
modifies cluster state. Requires `confirm: true`.

**Input:**

| Field      | Type   | Required | Description                                         |
|------------|--------|----------|-----------------------------------------------------|
| design_id  | string | Yes      | The design ID to undeploy. Min length: 1            |
| cluster_id | string | Yes      | The K8sContext ID of the target cluster. Min: 1     |
| confirm    | boolean| No       | Must be `true` to confirm undeployment. Default: false |

**Output:**

| Field     | Type   | Description                |
|-----------|--------|----------------------------|
| design_id | string | The undeployed design ID   |
| cluster_id| string | The cluster ID             |
| status    | string | Undeployment status ("undeployed") |

**Example call:**

```json
{
  "name": "undeploy_design",
  "arguments": {
    "design_id": "d1",
    "cluster_id": "cluster-1",
    "confirm": true
  }
}
```

### delete_design

Permanently delete a design. This is a mutating tool — it permanently deletes
design data. Requires `confirm: true`.

**Input:**

| Field      | Type   | Required | Description                              |
|------------|--------|----------|------------------------------------------|
| design_id  | string | Yes      | The design ID to delete. Min length: 1   |
| confirm    | boolean| No       | Must be `true` to confirm deletion. Default: false |

**Output:**

| Field | Type   | Description         |
|-------|--------|---------------------|
| id    | string | The deleted design ID |
| name  | string | The deleted design name |

**Example call:**

```json
{
  "name": "delete_design",
  "arguments": {
    "design_id": "d1",
    "confirm": true
  }
}
```

## Safety and read-only mode

### Read-only tools

The following tools are read-only — they only `GET` from Meshery Cloud and
never modify data:

- `list_designs`
- `get_design`

### Mutating tools

The following tools modify data in Meshery Cloud:

- `create_design` — creates a new design
- `deploy_design` — deploys a design to a Kubernetes cluster
- `undeploy_design` — undeploys a design from a cluster
- `delete_design` — permanently deletes a design

All mutating tools require a `confirm: true` input. When `confirm` is not `true`,
the tool returns a description of what it would do without making any changes to
the server. This prevents accidental mutations from malformed or exploratory
requests.

## Drift test

A Go test (`test/tools_reference_test.go`) asserts that every tool registered
in the server is listed on this page. When a new tool is added, its name must
be added to the tables above before the change can be merged.

## Not yet available

The following tool from the original issue scope is not yet implemented:

- `export_design` — The Cloud API does not expose a `format` parameter for
  design export; `GET /api/content/patterns/download/{designId}` returns raw
  design content but does not accept a format parameter. Export will be added
  when the Cloud API supports format selection.
