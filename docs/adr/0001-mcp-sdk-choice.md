# ADR-0001: MCP SDK choice (mcp-go 0.x vs official go-sdk)

## Status

Accepted

## Context

The meshery-mcp-server scaffold (internal/tools, internal/resources, internal/prompts) is a set of empty stubs; the MCP server itself is not yet implemented. `cmd/meshery-mcp-server/main.go` currently loads config, logs a startup line, and exits. `go.mod` contains no MCP SDK dependency at all — there is no existing mcp-go or go-sdk usage to migrate.

Issue #28 was to build the MCP server foundation. The issue this ADR addresses asks for an explicit MCP SDK decision before a registration contract is built on top of that scaffold.

## Decision problem

Choose an MCP SDK for the server implementation:

1. Stay on mcp-go 0.x (v0.58.0), target protocol revision 2025-11-25 for the MVP.
2. Move to the official github.com/modelcontextprotocol/go-sdk (v1.8.0 stable), targeting protocol revision 2026-07-28.
3. Wait for mcp-go v1.0.0 to leave beta, then absorb its API break.

## Verified facts

### MCP spec revision 2026-07-28

CONFIRMED via the official MCP blog post "The 2026-07-28 Specification"
(https://blog.modelcontextprotocol.io/posts/2026-07-28/) and the go-sdk v1.7.0 release notes:

- Removes the `initialize`/`notifications/initialized` handshake and protocol-level sessions.
- Each request carries `_meta.io.modelcontextprotocol/...` (protocolVersion, clientInfo, clientCapabilities).
- Adds `server/discover` RPC for advertising supported versions and capabilities (mandatory for discovery).
- Replaces `resources/subscribe` with `subscriptions/listen`.
- Removes server-initiated requests; replaces with Multi Round-Trip Requests (MRTR, SEP-2322).
- Roots, sampling, and logging are formally deprecated (SEP-2577).
- Backward-compatible clients/servers negotiate down to 2025-11-25.

### mcp-go (github.com/mark3labs/mcp-go)

CONFIRMED via https://github.com/mark3labs/mcp-go/releases:

- v0.57.0 and v0.58.0 cap at LATEST_PROTOCOL_VERSION = "2025-11-25" and have no `server/discover`. v0.58.0 (11 Aug 2026) is the last 0.x release.
- v1.0.0-beta.1 (12 Aug 2026) added 2026-07-28 support ("feat(mcp,server,client): support the 2026-07-28 specification").
- v1.0.0 (2 Sep 2026) is **stable, not in beta**, with 2026-07-28 support.
- v1.5.0 (30 Oct 2026) is the current stable release.

Correction to the issue: option 3's premise ("wait for mcp-go v1.0.0 to leave beta") is outdated — v1.0.0 is already stable.

### Official go-sdk (github.com/modelcontextprotocol/go-sdk)

CONFIRMED via https://github.com/modelcontextprotocol/go-sdk/releases and pkg.go.dev:

- v1.7.0 (28 Jul 2026) brought full 2026-07-28 support; v1.8.0 (4 Sep 2026) is the current stable release.
- `latestProtocolVersion = "2026-07-28"` (mcp/shared.go); `SupportedProtocolVersions()` returns the supported set.
- Registration API: `mcp.AddTool`, `mcp.AddResource`, `mcp.AddResourceTemplate`, `mcp.AddPrompt` (top-level helpers), plus `Server.AddTool`, `Server.AddResource`, `Server.AddResourceTemplate`, `Server.AddPrompt`.
- Transports: `StdioTransport`, `StreamableHTTPHandler`/`StreamableServerTransport`, `SSEHandler`/`SSEServerTransport`, `CommandTransport` (client).
- Stateless mode for 2026-07-28 over Streamable HTTP via `StreamableHTTPOptions.Stateless = true`.
- Maintained by modelcontextprotocol in collaboration with Google; Apache-2.0.

### PoC repo (github.com/singhharsh1708/meshery-mcp-poc)

CONFIRMED via https://github.com/singhharsh1708/meshery-mcp-poc:

- Read-only MCP server PoC for Meshery, Apache-2.0 (licensed so it can be folded into this repo).
- Uses official go-sdk v1.7.0, Go 1.25+.
- Covers: 4 tools, 4 resources (incl. URI templates via `github.com/yosida95/uritemplate/v3`), 3 prompts, cookie-based auth (`token` + `meshery-provider`), stdio and Streamable HTTP transports, tests (33 tests, 73.6% server coverage).
- Does NOT cover: SSE transport, auth beyond cookie-based, a registration contract design, mutating tools, the full server surface.
- Not verified against a live Meshery Server (uses a mock serving real payload shapes).

### Claim about PR #28 / mcp-go v0.57.0

NOT FOUND. `go.mod` has no mcp-go dependency. PR #28 is not in this repo's git history. The scaffold is empty stubs. The issue's claim that "#28 uses mcp-go v0.57.0" is incorrect for the current state of the repo.

## Options evaluated

### Option 1 — Stay on mcp-go 0.x (v0.58.0), target 2025-11-25

Trade-offs:
- Pros: mcp-go 0.x is stable and well understood; 2025-11-25 clients and servers interoperate via initialize handshake; ecosystem has the most examples and tutorials for this revision.
- Cons: Targets an older spec revision. A client speaking only 2026-07-28 sends no `initialize` (removed in that revision) and has no `server/discover` to fall back to in mcp-go 0.x — so such clients cannot connect. mcp-go 0.x is effectively superseded by v1; no new spec revisions will be backported.

Verdict: viable only if the MVP intentionally targets 2025-11-25 and accepts that 2026-07-28-only clients cannot connect.

### Option 2 — Move to official go-sdk now (v1.8.0), target 2026-07-28

Trade-offs:
- Pros: go-sdk v1.8.0 is stable with full 2026-07-28 support (`server/discover`, `subscriptions/listen`, MRTR, stateless core); official SDK maintained by modelcontextprotocol with Google; stdio and Streamable HTTP transports; PoC already validates it covers this project's real surface (tools, templated resources, prompts, stdio + Streamable HTTP, cookie auth). No existing mcp-go dependency in the repo, so there is no migration break.
- Cons: The PoC is a read-only vertical slice — it does not cover mutating tools or a registration-contract design; the team would need to learn the go-sdk API (it differs from mcp-go 0.x).

Verdict: recommended. Aligns with the current spec, is officially maintained, and the PoC de-risks the integration.

### Option 3 — Wait for mcp-go v1.0.0 to leave beta

Trade-offs:
- Pros: None actionable today.
- Cons: Premise is outdated — mcp-go v1.0.0 is already stable (released 2 Sep 2026), and v1.5.0 is current. Waiting buys nothing; mcp-go v1 and go-sdk v1 both support 2026-07-28 now.

Verdict: reject. The trigger condition (v1.0.0 leaving beta) is already satisfied.

## Decision

Recommend **Option 2: move to the official go-sdk now**, targeting protocol revision 2026-07-28.

Rationale: The repo has no MCP SDK dependency yet, so adopting go-sdk v1.8.0 is a clean addition with no migration break. It is the official, actively maintained SDK with full 2026-07-28 support, and the existing PoC demonstrates it covers this project's surface. mcp-go v1 is an acceptable fallback if the team later prefers it, but both SDKs support 2026-07-28, so the spec-revision decision is the more important one.

## Consequences

- The registration contract will be built on go-sdk's `Server.AddTool`/`AddResource`/`AddResourceTemplate`/`AddPrompt` API. Where practical, keep the contract SDK-agnostic (plain Go interfaces for handlers) so a later switch to mcp-go v1 would only require an adapter layer; the protocol-version handling itself cannot be SDK-agnostic.
- Transports: stdio (default, for Claude Desktop etc.) and Streamable HTTP (stateless mode for 2026-07-28). SSE is not required by 2026-07-28.
- Auth: integrate with the Cloud client layer (internal/meshery) for token handling; cookie-based auth from the PoC can inform the design.
- Advertised protocol version will be 2026-07-28, with negotiation down to 2025-11-25 handled by the SDK.

## What would change this decision

- If the PoC is found not to cover transports or auth the project needs (e.g., SSE, non-cookie auth) before implementation begins.
- If the go-sdk introduces a breaking change between v1.8.0 and the version used at implementation time — pin the minor version in go.mod.

## Related

- Related to #28 (MCP server foundation), #5 (transports), #13 (auth and configuration).
- PoC: https://github.com/singhharsh1708/meshery-mcp-poc
