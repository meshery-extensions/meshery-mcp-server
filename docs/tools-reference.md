# MCP Tools Reference

> **Status:** The MCP tool set is currently under active development.

This document is the reference for tools exposed by Meshery MCP Server.

Tool names, inputs, outputs, and examples must be kept synchronized with the server implementation.

## Tool Documentation Format

Each tool should document:

| Field | Description |
| --- | --- |
| Tool | MCP tool name |
| Purpose | What the tool does |
| Inputs | Required and optional arguments |
| Output | Returned data |
| Errors | Expected error conditions |
| Example | Working usage example |

## Planned Tool Areas

The project is expected to expose Meshery capabilities through MCP tools in areas including:

- Designs
- Connections
- Environments
- Workspaces
- Models
- Performance capabilities

The exact registered tool names and schemas will be added here after they are implemented and tested.

## Designs

Design-related tools will allow MCP clients to work with Meshery designs/patterns.

The final documentation should include:

- Tool name
- Input parameters
- Pagination behavior
- Returned design fields
- Error behavior
- Working example

## Connections

Connection-related tools will document how an MCP client can inspect or manage Meshery connections when the corresponding tools are implemented.

## Environments

Environment-related tools will document the available environment operations after implementation.

## Workspaces

Workspace-related tools will document the available workspace operations after implementation.

## Models

Model-related tools will document access to the Meshery model/component registry after implementation.

## Performance

Performance-related tools will document supported performance-test operations after implementation.

## Keeping This Reference Up to Date

Whenever an MCP tool is added or changed:

1. Update this document.
2. Document every input.
3. Document the output.
4. Document expected errors.
5. Add a working example.
6. Verify the example against the implementation.
7. Add or update the corresponding tests.
