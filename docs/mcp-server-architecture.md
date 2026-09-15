# Meshery MCP Server: Architecture and Registration Design

## 1. Overview

`meshery-mcp-server` is a standalone Model Context Protocol (MCP) extension that enables MCP-capable AI clients, such as Claude Desktop, Cursor, and VS Code extensions, to safely consume Meshery capabilities.

The server is a client-facing protocol boundary. It translates MCP tool and resource requests into calls to Meshery Server; it does not become a Meshery adapter, access Kubernetes directly, or duplicate Meshery Server business logic.

The initial release is deliberately read-only. Its purpose is to prove authenticated connectivity, safe response handling, and a maintainable extension model before adding any infrastructure-changing capability.

## 2. Transport and Request Flow

The initial transport is local `stdio`, allowing an MCP client to launch the server as a local subprocess.

Streamable HTTP is a future transport and deployment milestone. Tool, resource, and client-boundary logic must remain transport-agnostic so that later HTTP support does not require rewriting capability implementations.

```text
AI Client / IDE
      |
      | MCP protocol over stdio
      v
Meshery MCP Server
      |
      | Tool or resource handler
      v
Shared Meshery Server Client
      |
      | Authenticated request to Meshery Server API
      v
Meshery Server
      |
      v
Meshery-managed capabilities, adapters, MeshSync, and connected infrastructure
```

## 3. Shared Meshery Server Client Boundary

All MCP tools and resources communicate with Meshery Server through one shared client boundary, located under `internal/meshery`.

Individual tools and resources must not:

- Construct their own HTTP clients.
- Read raw tokens, session cookies, kubeconfig files, passwords, or private keys.
- Implement their own retry, timeout, redirect, TLS, or error-handling behavior.
- Shell out to `mesheryctl`.
- Bypass Meshery Server to communicate directly with adapters or Kubernetes clusters.

The shared client boundary is responsible for:

- Resolving Meshery Server configuration.
- Constructing authenticated requests using the existing Meshery authentication context.
- Applying centralized timeout, TLS, retry/backoff, redirect, and health-check behavior.
- Normalizing Meshery Server API errors, including known non-standard responses such as application error bodies returned with successful HTTP status codes or login redirects.
- Returning typed, safe results to MCP tools and resources.

### Authentication and Context

Meshery Server remains the authority for authentication, authorization, provider selection, and infrastructure routing.

MCP tools receive an already-authenticated Meshery Server client; they never receive raw credentials. The exact v0.1 credential mechanism and endpoint compatibility requirements must be validated against a running Meshery Server and recorded in the shared-client ADR before implementation is treated as final.

For local interactive use, the server will consume the user’s existing Meshery authentication context. CI may provide equivalent configuration through environment variables. Secrets must never appear in MCP tool input schemas, responses, errors, or logs.

### Required Client-Boundary Investigation

Before the shared-client design is finalized, the project must document:

- The existing package and symbols responsible for Meshery Server URL/configuration resolution.
- Server discovery and connection setup behavior.
- Session/token loading and authenticated request construction.
- TLS, timeout, retry/backoff, and health-check handling.
- Existing Meshery API error-normalization behavior.
- Which components can be reused independently of the adapter interface.

The investigation outcome must be captured in an ADR with exact repository paths, symbols, findings, trade-offs, and a recommendation for reuse, extraction, or a narrow new client implementation.

## 4. Registration Model

Tools, resources, and future prompts are registered through a descriptor-based Registrant/Registry seam.

Each registrant declares:

- Name.
- Input and output schemas.
- Handler.
- Safety classification.
- MCP annotations derived from that classification.

The registration layer centrally enforces the read-only gate and maps safety classifications to MCP annotations. Tool packages must not independently decide how to apply safety policy.

MCP SDK-specific types remain behind this registration seam so that SDK migration does not require rewriting individual tools.

## 5. Tool and Resource Scope

### v0.1: Read-only Tooling

The first mandatory vertical slice is:

```text
list_designs
```

`list_designs` is complete only when it:

- Uses the shared authenticated Meshery Server client.
- Runs against a real or Dockerized Meshery Server.
- Supports the selected Meshery API pagination model.
- Produces structured, token-conscious output.
- Normalizes Meshery Server failures into honest, actionable errors.
- Sanitizes successful and failed results before MCP serialization.
- Has unit and integration test coverage.

`server_info` may be considered as a smaller preliminary connectivity check if maintainers choose it, but it does not replace the need for one complete read-only vertical slice.

### Future Candidate Tools

Future read-only candidates may include design retrieval/export, dry-run or validation result retrieval, MeshSync resource inspection, environment/workspace information, and performance-test result retrieval.

Each future tool requires its own issue or tool contract defining API mapping, input/output schemas, safety classification, error behavior, and acceptance criteria.

### Explicitly Deferred

The following are not part of v0.1:

- Deploy, undeploy, create, update, or delete operations.
- Direct Kubernetes access or kubeconfig handling.
- Credential or connection mutation.
- Prompts, until a concrete Meshery-backed prompt surface is defined.
- Remote hosted deployment.
- In-cluster/Helm packaging.
- Streamable HTTP implementation.
- Long-running-operation behavior.

Any future mutating operation requires explicit server-side enforcement, user confirmation/elicitation where appropriate, authorization design, auditability, idempotency requirements, and rate-limit considerations.

## 6. Security and Sanitization

All tool and resource outputs pass through a shared response boundary before they reach an MCP client.

The response boundary must sanitize:

- Successful API payloads.
- Error bodies.
- Log entries.
- Task or operation state.
- Any discovered infrastructure data that could contain credentials, Kubernetes Secret content, connection configuration, tokens, passwords, or private keys.

Sanitization is a shared enforcement point, not a responsibility delegated to individual tool implementations.

Errors must use Meshery’s actual MeshKit error framework and preserve safe, actionable information without leaking sensitive data.

## 7. Implementation Roadmap

### Milestone 0: Repository Foundation

- Establish the repository structure.
- Add module configuration, Makefile targets, linting, Docker build, CI, and README.
- Document local development and contribution workflows.
- Keep transport, full auth behavior, and capability implementation outside the scaffold scope.

### Milestone 1: Shared Client and First Vertical Slice

- Complete the shared-client investigation and ADR.
- Implement the selected shared Meshery Server client boundary.
- Implement `list_designs`.
- Add centralized response/error/log sanitization.
- Add MeshKit-aligned error mapping.
- Add unit tests and integration tests against a real or Dockerized Meshery Server.
- Confirm the selected authentication mechanism against the v0.1 endpoint set.

### Milestone 2: Additional Read-only Capabilities

- Add individually specified read-only tools and resources.
- Add tool-schema snapshot tests.
- Add safe resource exposure where the Server API and sanitization rules are established.
- Evaluate streamable HTTP only after the core read-only capability path is stable.

### Milestone 3: Write-capability Proposal

- Produce a separate design proposal for write operations.
- Define authorization, confirmation, auditing, idempotency, rate limiting, and dry-run expectations.
- Do not implement mutating tools without explicit maintainer approval.