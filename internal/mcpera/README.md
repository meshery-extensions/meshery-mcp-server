# mcpera

Reports which MCP protocol era a server actually serves, by talking to it.

```bash
go build -o mcpera ./cmd/mcpera
./mcpera ./my-mcp-server
```

## Why

Revision `2026-07-28` is the first breaking change to MCP. It removes the
`initialize` handshake, makes every request carry its own version in `_meta`,
and adds `server/discover`. That splits the world into a legacy era
(`2025-11-25` and earlier) and a modern one, and the specification's own
compatibility table is blunt about the crossing:

> **Modern client, Legacy server.** Fails. The server may reject the request
> with an implementation-defined error, stay silent, or even process an
> era-ambiguous method under legacy semantics.

The third outcome is the one worth detecting, because it does not look like a
failure. A legacy server can answer a modern request by running it and returning
a legacy-shaped result: no error, no version acknowledgement, nothing on the wire
saying the eras did not match. The client believes it negotiated `2026-07-28`.
The server never negotiated anything.

The only signal is the shape of the result. Revision `2026-07-28` puts
`resultType` on a result and `io.modelcontextprotocol/serverInfo` in its `_meta`;
the legacy body has neither. A client that reads `content` and stops, which is
most of them, cannot tell.

## Measured

Four servers, each a one-tool stdio server built against a different SDK, probed
with a request declaring `protocolVersion: 2026-07-28` and the
`clientCapabilities` the revision requires:

| server | `initialize` | `server/discover` | modern request | result shape | era |
|---|---|---|---|---|---|
| `mark3labs/mcp-go` v0.57.0 | yes | `-32601` | **runs it** | **legacy** | legacy |
| `mark3labs/mcp-go` v0.58.0 | yes | `-32601` | **runs it** | **legacy** | legacy |
| `mark3labs/mcp-go` v1.0.0-beta.1 | yes | yes | runs it | modern | dual-era |
| `modelcontextprotocol/go-sdk` v1.7.0 | yes | yes | runs it | modern | dual-era |

Raw, for the row that matters:

```text
request  {"method":"tools/call","params":{"name":"ping","arguments":{},
          "_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28", ...}}}

v0.58.0  {"result":{"content":[{"type":"text","text":"pong"}]}}
v1.7.0   {"result":{"_meta":{"io.modelcontextprotocol/serverInfo":{...}},
                    "content":[{"type":"text","text":"pong"}],"resultType":"complete"}}
```

v0.58.0 executed the tool for a client it never handshook with, and answered in
the old shape.

The spec's own mitigation holds up: it says stdio clients SHOULD send
`server/discover` first so the mismatch fails deterministically, and on v0.58.0
that probe does fail cleanly with `-32601`. The hazard is only for a client that
skips it.

## Reading the output

`era` is one of `legacy`, `modern`, `dual-era` or `unknown`. Dual-era is the only
posture that works with every client: a modern client cannot fall back to a
legacy server, and a legacy client has no fall-forward mechanism at all.

The command exits non-zero when it finds a silent downgrade, so it can gate CI.

## Over HTTP

Streamable HTTP adds a rule with a security rationale behind it. Revision
`2026-07-28` requires a server to reject a request whose headers disagree with
its body, with `400` and error `-32020`, and the spec says why:

> This prevents potential security vulnerabilities when different components in
> the network rely on different sources of truth (e.g., a load balancer routing
> on the header value while the MCP server executes based on the body value).

`ProbeHTTP` calls a tool twice, once with `Mcp-Name` agreeing with the body and
once naming a different tool, and reports which happened. Measured against the
same servers, each exposing a `ping` and a `danger` tool:

| server | serves modern | header/body mismatch |
|---|---|---|
| `go-sdk` v1.7.0, `Stateless: true` | yes, modern shape | rejected, `400` / `-32020` |
| `mcp-go` v1.0.0-beta.1 | yes, modern shape | rejected, `400` / `-32020` |
| `mcp-go` v0.57.0, legacy session | yes, **legacy shape** | **ran `danger`, `200`** |

Both current SDKs enforce it correctly. v0.57.0 predates the rule, so it ignores
`Mcp-Name` and runs whatever the body asked for:

```text
Mcp-Name: ping                      <- what an intermediary would route on
body:     {"name":"danger"}         <- what the server executed
->        200 {"content":[{"type":"text","text":"DANGER EXECUTED"}]}
```

That is not v0.57.0 violating its own revision, since the rule is newer than it.
It is the deployment shape the spec's own note warns intermediaries about: if the
`MCP-Protocol-Version` is older than the rule, or absent, an intermediary
**SHOULD** reject rather than trust header values the server never validated.

One thing worth knowing before reaching for the go-sdk over HTTP: its default
`NewStreamableHTTPHandler` refuses `2026-07-28` outright with
`protocol version "2026-07-28" is only supported on stateless HTTP servers`.
It needs `&mcp.StreamableHTTPOptions{Stateless: true}`, which the README setup
does not set.

## Scope

The stdio probe opens a fresh process per exchange. The HTTP probe calls the
endpoint directly and needs a server already listening, plus the names of two
tools it is safe to run.

The rows above are one and two-tool servers, not the SDKs in full. They say what
these SDKs do on a default setup, plus the one documented flag the go-sdk needs
before it will speak `2026-07-28` over HTTP at all.


