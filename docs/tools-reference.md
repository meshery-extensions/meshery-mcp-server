# MCP tools reference

This page lists the MCP tools exposed by the Meshery MCP Server, together with
their inputs, outputs, and example calls.

> [!NOTE]
> **No tools are registered yet.** The MCP server is not implemented; the
> [`internal/tools`](https://github.com/meshery-extensions/meshery-mcp-server/tree/master/internal/tools)
> package is an empty stub. This page documents the *planned* tools and will be
> populated as tools are added. A test
> ([test/tools_reference_test.go](#drift-test)) fails if a registered tool is
> missing from this page, keeping the doc and the registry in sync.

## Planned tools

In priority order (from the [README](README.md#planned-scope)):

| # | Tool | Purpose |
|---|------|---------|
| 1 | Create a Meshery design | Create a new Meshery design |
| 2 | Retrieve a Meshery design | Get a design in JSON or YAML |
| 3 | Snapshot a Meshery design | Snapshot a design |
| 4 | Dry-run a design deployment | Dry-run results for a design deployment |
| 5 | Define and run a performance test | Run a performance test and retrieve results |

These will be exposed through MCP's three building blocks: **Tools** (actions
the AI can invoke), **Resources** (read-only data the AI can load as context),
and **Prompts** (reusable prompt templates). See [docs/adr/0001-mcp-sdk-choice.md](adr/0001-mcp-sdk-choice.md)
for the SDK and protocol-version decision.

## Drift test

A Go test asserts that every tool registered in the server is listed on this
page. As long as no tools are registered, the test passes trivially. When a new
tool is added, its name must be added to the table above (or the drift-test
exception list) before the change can be merged.
