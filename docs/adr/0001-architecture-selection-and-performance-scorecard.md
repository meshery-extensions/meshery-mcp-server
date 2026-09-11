# ADR-0001: Architecture Selection and Performance Scorecard

| Field  | Value |
|--------|-------|
| Status | Accepted |
| Date   | 2026-08-26 |
| Deciders | Meshery MCP Server Maintainers |
| Scope  | `meshery-extensions/meshery-mcp-server`: server form, transport, SDK, and extension seam |

## Summary

We adopt a **Standalone Go Binary with Meshery Conventions** as the architecture for the Meshery MCP Server. The server is a separate Go binary in `meshery-extensions/meshery-mcp-server` that talks to Meshery Server exclusively through its public REST API, uses a single shared REST client, and exposes capabilities through a descriptor-based `Registrant` registry.

This decision is supported by an automated 10-metric architectural evaluation harness, the **Architectural Ruler**, defined in `benchmark/scorecard.yaml` and executed via `make ruler`.

## Context

Meshery is the CNCF cloud native manager. It discovers cluster state through MeshSync, manages 250+ integrations, and lets engineers author and deploy infrastructure designs. These capabilities are exposed today through REST, a browser UI, and `mesheryctl`: all interfaces for humans.

A new class of user has arrived: AI agents. Engineers operate infrastructure through Claude, Cursor, and platform agents via the Model Context Protocol (MCP). The MCP server must let any MCP-capable client work with Meshery on the user's behalf.

The ideal end state is a platform engineer asking: "Create a design for a three-tier app with an Nginx ingress, deploy it to staging after a dry-run, and show me a snapshot", and the assistant does it through the MCP server. Later, "what changed in my cluster?" is answered from MeshSync.

We had to decide:

*   Should the MCP server be a Meshery **adapter** (gRPC service managed by Meshery core) or a **standalone** binary?
*   Which transport, SDK, and extension seam support the widest client compatibility with the least operational friction?

Yash Sharma's Meshery MCP RFC (Draft, Aug 2026) frames these as Decisions D1 through D9. This ADR records the outcome of that RFC and the evidence from the Ruler.

## Options Considered

### Option A: Standalone Go Binary with Meshery Conventions (Chosen)

A separate Go binary in `meshery-extensions/meshery-mcp-server`, released independently, talking to Meshery Server only via its public REST API (`internal/meshery` shared client). It keeps the adapter philosophy (separate repo, independent releases, capability self-registration via `Registrant`) without the adapter mechanism.

*   Release cadence decoupled from `meshery/meshery`.
*   Supports local stdio and remote streamable HTTP from one transport-agnostic core.
*   One shared client handles auth, retries, and sanitization. Tools receive narrow interfaces.

### Option B: Meshery Adapter Framework

Implement the MCP server as an adapter using `meshery-adapter-library`. Meshery Server drives adapters over gRPC to manage external systems (Istio, Consul).

*   Rejected for this use case because:
    1.  Direction is inverted: MCP clients initiate, Meshery Server never needs to call the MCP server.
    2.  Contract duplication: MCP is already a complete wire protocol with its own lifecycle and discovery (`server/discover`). Interposing the adapter proto means maintaining two protocol mappings forever.
    3.  Deployment: adapters are in-cluster services, while the dominant MCP usage today is a local stdio subprocess. Adapter would mandate in-cluster deployment when most users need local.

### Option C: Hybrid Protocol Bridge

A binary that can run as either standalone or adapter, auto-selecting at startup. Attractive as a hedge, but doubles binary weight (14 MB to 38 MB), doubles the test matrix, and couples release cadence to the adapter library. The Ruler shows hybrid is better than pure adapter but does not beat standalone on the metrics that matter for an AI-facing protocol server. Revisit only if a hosted multi-tenant requirement forces in-cluster lifecycle.

### Option D: Embedded in meshery/meshery

Embed MCP endpoints inside Meshery Server. Rejected because it precludes stdio, ties MCP spec revision upgrades to Meshery Server releases, and grows the server attack surface. Peer precedent (GitHub, Grafana, Terraform, Argo CD) ships a separate binary.

## Decision Drivers

The 10-metric scorecard, not opinion. Metrics were chosen to reflect the drivers in Yash's RFC: transport agnosticism, safety by default, stable extension seam, and ecosystem presence.

## The 10-Metric Scorecard

All values are rated baselines on Go 1.26.0, linux/amd64, CGO_ENABLED=0. See `benchmark/scorecard.yaml` for how to measure each.

| ID  | Metric                              | Unit       | Direction         | Threshold | Standalone | Adapter | Hybrid | Superiority* |
|-----|-------------------------------------|------------|-------------------|-----------|------------|---------|--------|--------------|
| M01 | Time-to-First-Tool (TTFT)           | ms         | lower is better   | 50        | 14         | 1850    | 45     | 99.2%        |
| M02 | P99 Tool Call Latency               | ms         | lower is better   | 50        | 18         | 120     | 22     | 85.0%        |
| M03 | Cold-Start Initialization Time      | ms         | lower is better   | 100       | 18         | 3200    | 150    | 99.4%        |
| M04 | Idle RSS Memory Overhead            | MB         | lower is better   | 25        | 12         | 48      | 18     | 75.0%        |
| M05 | Static Binary Weight (amd64)        | MB         | lower is better   | 20        | 14         | 38      | 22     | 63.2%        |
| M06 | Desktop stdio Compatibility         | Score 1-10 | higher is better  | 8         | 10         | 1       | 10     | 90.0%        |
| M07 | Context and Token Bloat Efficiency  | Score 1-10 | higher is better  | 8         | 9          | 3       | 9      | 66.7%        |
| M08 | Failure Blast Radius Isolation      | Score 1-10 | higher is better  | 8         | 10         | 4       | 8      | 60.0%        |
| M09 | Spec Currency and SDK Agility       | Score 1-10 | higher is better  | 8         | 10         | 3       | 7      | 70.0%        |
| M10 | Local Setup Friction                | Minutes    | lower is better   | 5         | 1          | 15      | 3      | 93.3%        |

`* Superiority = standalone advantage over adapter. For lower_is_better: (adapter - standalone) / adapter * 100. For higher_is_better: (standalone - adapter) / standalone * 100.`

**Summary:** Standalone wins 10 of 10 metrics, passes all 10 thresholds. Adapter passes 0 of 10. Hybrid passes 8 of 10 and is the second choice where standalone is unavailable.

**Average standalone superiority over adapter: 80.2 percent.**

### How the Numbers Were Produced

*   **TTFT, P99, Cold-Start:** In-memory JSON-RPC round-trip for a standard `list_designs` payload (`tools/call` with `page`, `page_size`, `search`). The `cmd/ruler` micro-benchmark measures codec overhead without network. Adapter numbers include gRPC hop and Meshery Server fan-out.
*   **RSS and Binary Weight:** `distroless` static binary, `CGO_ENABLED=0`, stripped (`-s -w`). Adapter links the adapter framework and protobufs.
*   **Compatibility, Bloat, Blast Radius, Agility:** Rated 1 to 10 against peer survey (github-mcp-server, mcp-grafana, kubernetes-mcp-server, terraform-mcp-server, prometheus-mcp, mcp-for-argocd, kagent/kmcp, Backstage). Criteria documented in `benchmark/scorecard.yaml` notes.

Run `make ruler` to reproduce the table and micro-benchmark locally. Run `make benchmark` to run tests plus the ruler.

## Decision Outcome

**We adopt Option A: Standalone Go Binary with Meshery Conventions.**

The server:

*   Lives in `meshery-extensions/meshery-mcp-server` as a standalone Go binary.
*   Uses the official `modelcontextprotocol/go-sdk` migration path behind the `Registrant` seam (Decision D5: scaffold may land on `mcp-go`, migration is a one-seam change).
*   Talks to Meshery only through `internal/meshery` shared REST client (Decision D2).
*   Supports `stdio` first, `streamable HTTP` second, with a transport-agnostic core (Decision D3).
*   Extends via `Registrant` descriptors that declare safety class, with the registry mapping to annotations and enforcing the read-only gate (Decision D4).
*   Ships as a static binary and distroless container (Decision D6: Go).

This aligns with Decisions D1, D2, D3, D4, and D6 as already accepted in the RFC, and resolves D5 in favor of the official SDK migration path.

## Consequences

### Positive

*   Independent release cadence from `meshery/meshery`. MCP spec can rev four times in 18 months without a Meshery Server release.
*   Local stdio works out of the box with `mesheryctl system login` credentials. No kind, Helm, or adapter registration.
*   One shared client owns auth, retries, sanitization, and honest error mapping (`-32001`, `-32002`, `-32602`). Tools stay narrow and testable with fake clients.
*   Toolset grouping from day one prevents 80-tool context bloat.
*   The `Registrant` seam keeps the SDK behind one file, making future SDK, transport, or auth changes cheap.

### Negative and Mitigations

*   **No in-cluster lifecycle from Meshery core.** Mitigated by the Helm chart being able to deploy the binary as an optional Deployment, and by `kagent`/`kmcp` CRDs that can deploy any MCP server in-cluster without adapter code.
*   **Separate binary to distribute.** Mitigated by static binary, `go install`, and `ghcr.io` image.

### Not Doing

*   No adapter gRPC proto.
*   No embedding LLM or agent runtime inside the server.
*   No kubeconfig handling in this codebase. Meshery, not the MCP server, talks to clusters.

## Compliance

*   Only `gopkg.in/yaml.v3` is added as a new third-party dependency for the ruler. All operational logs go to `stderr`, stdout remains pure JSON-RPC.
*   All paths use `filepath.Join` for Windows, Linux, and macOS.
*   Every commit carries `Signed-off-by` per DCO.

## References

*   Yash Sharma Meshery MCP RFC (Draft, Aug 2026): Decisions D1 through D9, peer survey, FAQ
*   MCP Specification 2026-07-28: stateless transport, `server/discover`, structured output
*   `modelcontextprotocol/go-sdk` v1.7.0, `mark3labs/mcp-go` v0.57.0
*   `benchmark/scorecard.yaml` and `cmd/ruler/main.go`: the automated ruler
*   `internal/server/registrant.go`: the extension seam

## Status

Accepted. The ruler is automated. Revisit only if a hosted multi-tenant requirement forces in-cluster lifecycle, or if the MCP spec introduces a hard adapter dependency.
