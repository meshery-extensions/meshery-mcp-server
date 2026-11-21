# Go MCP SDK comparison

Input for the design discussion, covering the criteria raised in the MCP Server
channel: specification support, tools/resources/prompts, transports,
authentication, maintenance maturity, testability, observability and Meshery
ecosystem fit.

SDK-specific claims below were read from the tagged source of each SDK rather
than from release notes or documentation. What v0.57 does with a modern request
was measured against a one-tool stdio server built on it. Protocol claims come
from the MCP specification, cited where used. The star and issue counts are
GitHub metadata, dated. Versions compared are `mark3labs/mcp-go` v0.57.0
(the version the scaffold in #28 uses) and `modelcontextprotocol/go-sdk` v1.7.0.
Both are good libraries; the differences that matter here are narrow.

## Summary

| Criterion | mark3labs/mcp-go v0.57.0 | modelcontextprotocol/go-sdk v1.7.0 |
|---|---|---|
| Latest protocol revision | `2025-11-25` | `2026-07-28` |
| `server/discover` | not present | present |
| Transports | stdio, HTTP+SSE (legacy), Streamable HTTP | stdio, HTTP+SSE (legacy), Streamable HTTP |
| Tools, resources, prompts | all three | all three |
| Resource templates and subscriptions | yes | yes |
| Authentication helpers | OAuth client (`client/transport/oauth.go`), serves Protected Resource Metadata | the same, plus server-side bearer verification (`auth.RequireBearerToken`) and `oauthex/` |
| Conformance suite | none in-tree | `conformance/` with baseline fixtures |
| Observability | `otel/` and `tracing/` packages | none in-tree |
| Test helpers | `mcptest/` package | in-memory transport |
| Stars (19 Aug 2026) | 9.0k | 5.0k |
| Open issues (19 Aug 2026) | 30 | 97 |
| Licence | MIT | Apache-2.0 (transitioning from MIT) |
| Status | community | maintained by the MCP project |

## Protocol revision

This is the one asymmetry worth a deliberate decision.

`mcp-go` v0.57.0 and v0.58.0 both declare, in `mcp/types.go`:

```go
const LATEST_PROTOCOL_VERSION = "2025-11-25"

var ValidProtocolVersions = []string{
	LATEST_PROTOCOL_VERSION,
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}
```

The official SDK declares, in `mcp/shared.go`:

```go
latestProtocolVersion   = protocolVersion20260728
protocolVersion20260728 = "2026-07-28"
```

`2026-07-28` is a breaking revision. It removes the `initialize` handshake and
carries the protocol version in `_meta` on every request, adds `server/discover`
for capability discovery (optional: a client may probe with it or invoke any RPC
inline), and replaces `resources/subscribe` with `subscriptions/listen`.

**A legacy client loses nothing by a server staying on `2025-11-25` today.**
Legacy clients negotiate down, and `mcp-go` handles an unknown version gracefully
rather than erroring:

```go
if slices.Contains(mcp.ValidProtocolVersions, clientVersion) {
	return clientVersion
}
return mcp.LATEST_PROTOCOL_VERSION
```

The gap is on the other side: a client speaking the current revision sends no
`initialize` at all. `server/discover` is one way it can probe, and the spec
marks that optional, so a client may equally invoke any RPC inline and expect an
`UnsupportedProtocolVersionError` back. Measured against a one-tool v0.57 stdio
server: `server/discover` fails cleanly with `-32601`, but the inline RPC is
**executed**, and answered in the legacy result shape with no error and no
version acknowledgement. That is the third outcome in the spec's own
modern-client-to-legacy-server row: "Fails. The server may reject the request
with an implementation-defined error, stay silent, or even process an
era-ambiguous method under legacy semantics." A client that reads only `content`
cannot tell it was downgraded.

`mcp-go` added the revision in `v1.0.0-beta.1` on 12 August, which is both a beta
and a v1 API break from the 0.x line.

This matters now specifically because the parts of the spec that changed are the
lifecycle and capability advertisement, which is what a registration contract
abstracts. Deciding after the contract exists means changing the contract and
everything registered through it.

## Transports

Both SDKs ship stdio, HTTP+SSE and Streamable HTTP. The naming is worth getting
right in any comparison table, because "SSE" on its own is now ambiguous:
HTTP+SSE is the separate transport that Streamable HTTP **replaced** in the
`2025-03-26` revision, while Streamable HTTP still uses SSE for its own response
streams. New work should target Streamable HTTP and treat HTTP+SSE as
backwards-compatibility only.

## Tool annotations

A behavioural difference with a practical consequence. `mcp.NewTool` in `mcp-go`
seeds annotations rather than leaving them unset:

```go
Annotations: ToolAnnotation{
	ReadOnlyHint:    ToBoolPtr(false),
	DestructiveHint: ToBoolPtr(true),
	IdempotentHint:  ToBoolPtr(false),
	OpenWorldHint:   ToBoolPtr(true),
},
```

They are `*bool` with `omitempty`, and `omitempty` only drops a nil pointer, so
those defaults are serialized. A read-only tool declared with a bare `NewTool`
therefore goes out as `readOnlyHint: false, destructiveHint: true`, and clients
use those hints to decide what needs confirmation. Both #28 and #34 hit this and
have since set the annotations explicitly.

Annotations cannot carry the read-only decision on their own. The spec is
explicit that they are hints and should "be untrusted unless they come from
trusted servers", so a client must not derive authorization or safety behaviour
from them. A read-only mode belongs in the registration path, by not registering
mutating tools at all, with the annotations following that decision rather than
defining it. Either way the seeded default is worth a build-time guard so a new
tool cannot silently ship as `destructiveHint: true`.

## Where each is stronger

**mcp-go** has more adoption, a permissive MIT licence, a much smaller open-issue
count, in-tree OpenTelemetry and tracing packages, and an `mcptest` package for
driving a server in tests. For a project that wants observability out of the box
and does not need the newest revision, it is a reasonable default, and it is
already the scaffold's choice.

**The official SDK** tracks the specification, ships `auth/` and `oauthex/` for
the OAuth 2.1 resource-server flow the spec defines, and includes a conformance
suite with baseline fixtures. Being maintained by the MCP project means revision
support lands there first.

## Suggestion

Not a recommendation of one over the other, since both are defensible, but a
suggestion about sequencing: decide the target protocol revision explicitly and
record it in the decision log, then pick the SDK that serves it. If the answer is
"`2025-11-25` is fine for the MVP", `mcp-go` is the sensible choice and the
scaffold already reflects it. If the answer is "we should be current", that
points at the official SDK today, or at waiting for `mcp-go` v1.0.0 to leave beta
and absorbing its API break.

Either way the decision is much cheaper before the registration contract lands
than after.

## References

- Tracking issue for the protocol revision decision: #42
- `mcp-go` protocol constants: `mcp/types.go`, `server/server.go`
- Official SDK protocol constants: `mcp/shared.go`; methods in `mcp/protocol.go`
- MCP specification revisions: https://modelcontextprotocol.io/specification/versioning
