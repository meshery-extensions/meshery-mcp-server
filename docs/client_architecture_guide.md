# Deep Dive: Meshery REST Client Architecture (`internal/meshery`)

This guide provides a comprehensive, educational walkthrough of the internal HTTP REST client implemented for `meshery-mcp-server`. It explains **what** was built, **why** specific architectural patterns were chosen, **how** the components work together, and **what work remains** for future phases.

---

## 1. Context & Core Objective

### What is `meshery-mcp-server`?
`meshery-mcp-server` is a **Model Context Protocol (MCP)** server that exposes Meshery's capabilities (cloud-native infrastructure management, service mesh operations, design blueprints, connections, and performance benchmarks) to LLM assistants and automated tools.

### The Challenge
Before this implementation, the MCP server had no way to communicate with a live Meshery Server instance (`http://localhost:9081`). 

### The Solution
We built an internal Go client package in `internal/meshery` that:
1. Manages HTTP transport mechanics (headers, timeouts, zero-dependency retries, serialization).
2. Provides clean, thin Go methods for querying Meshery REST APIs (`Ping()`, `ListConnections()`, `ListPatterns()`).
3. Satisfies all acceptance criteria of **Issue #6**.

---

## 2. High-Level Architecture

The flow of a request follows a strict single-direction pipeline:

```text
  ┌─────────────────────────────────────────────────────────────┐
  │                 MCP Tools (e.g. list_connections)           │
  └──────────────────────────────┬──────────────────────────────┘
                                 │ Invokes thin method (e.g. ListConnections)
                                 ▼
  ┌─────────────────────────────────────────────────────────────┐
  │                 internal/meshery/methods.go                 │
  │     (Constructs path & query url.Values, calls do())        │
  └──────────────────────────────┬──────────────────────────────┘
                                 │ Calls generic transport: do(ctx, method, path, query, req, resp)
                                 ▼
  ┌─────────────────────────────────────────────────────────────┐
  │                 internal/meshery/client.go                  │
  │  - Validates URL & Query via url.Parse                      │
  │  - Injects headers (Bearer token, Accept, User-Agent)       │
  │  - Executes zero-dependency retry loop for 502/503/504      │
  │  - Deserializes JSON or maps 4xx/5xx to APIError            │
  └──────────────────────────────┬──────────────────────────────┘
                                 │ Standard http.Request
                                 ▼
  ┌─────────────────────────────────────────────────────────────┐
  │                    Meshery Server API                       │
  │                 (http://localhost:9081/api/...)             │
  └──────────────────────────────┬──────────────────────────────┘
```

---

## 3. Package File Layout & Responsibilities

Instead of creating dozens of tiny files (over-fragmentation) or putting everything into one 500-line file (god-file), we organized the package into **5 single-responsibility files**:

```text
internal/meshery/
├── config.go      # Configuration data struct & DefaultConfig()
├── types.go       # Query options (ListOptions), APIError, and response envelopes
├── client.go      # Client struct, NewClient constructor, and generic do() transport
├── methods.go     # Phase 1 domain wrapper methods (Ping, ListConnections, ListPatterns)
└── client_test.go # Acceptance-mapped unit test suite using httptest.Server
```

---

## 4. Deep-Dive: Architectural Decisions & Design Rationale

### A. Why `internal/meshery` instead of `pkg/client`?
* **Go Module Boundary**: Go enforces a special rule for directories named `internal/`: code inside `internal/` cannot be imported by any external Go module.
* **App-Specific vs SDK**: `meshery-mcp-server` is an application server, not a general-purpose public client library. Keeping code in `internal/` prevents accidental API stability commitments to third parties.

---

### B. Pure Config Data Struct (Decoupled Configuration)
* **Design**: `Config` in `config.go` is a pure struct. It contains **no** file-reading (`os.ReadFile`) or environment-reading (`os.Getenv`) logic.
* **Why**: The transport client should only care about **making HTTP requests**, not **discovering configuration**. 
* **Precedence**: Flag parsing and environment loading happen at the application entry point (`main.go`):
  ```text
  DefaultConfig()  <  LoadFromFile(--config flag)  <  LoadFromEnv()
  ```
  Environment variables take highest precedence (ideal for containerized deployments).

---

### C. Core Transport Method (`do()`) Deep Dive

The `do()` method in `client.go` is the central engine. Its exact signature is:

```go
func (c *Client) do(
    ctx context.Context,
    method string,
    path string,
    query url.Values,
    reqBody any,
    respBody any,
) error
```

#### Key Transport Invariants:

1. **Safe URL Construction**:
   ```go
   u, err := url.Parse(c.cfg.BaseURL)
   u.Path = path
   if query != nil {
       u.RawQuery = query.Encode()
   }
   ```
   * **Why**: Concatenating strings like `fmt.Sprintf("%s%s", baseURL, path)` causes double-slash bugs (`http://localhost:9081//api/pattern`) or missing slashes. Using `url.Parse` and `url.Values` handles slashes, encoding, and special query characters safely.

2. **Payload Preservation Across Retries (Solving the Body-Drain Bug)**:
   * **The Problem**: Standard Go `http.Request.Body` is an `io.ReadCloser`. Once `http.Client.Do()` reads it during the first attempt, the reader stream is exhausted (drained). If attempt #1 fails with a `502 Bad Gateway`, attempt #2 will send an **empty** request body!
   * **The Solution**: We pre-marshal `reqBody` into `payload []byte` **once** outside the retry loop. Inside each retry iteration, we create a fresh reader:
     ```go
     var bodyReader io.Reader
     if payload != nil {
         bodyReader = bytes.NewReader(payload) // Fresh reader for every attempt
     }
     ```

3. **Standard Header Injections**:
   * `Accept: application/json`
   * `User-Agent: <c.cfg.UserAgent>` (default: `meshery-mcp-server`)
   * `Content-Type: application/json` (only when `reqBody != nil`)
   * `Authorization: Bearer <token>` (only when `c.cfg.Token != ""`)

4. **Resource Leak Prevention**:
   * Every successful `http.Do` returns a `resp.Body` that **must** be closed to prevent socket leaks.
   * `do()` executes:
     ```go
     respData, readErr := io.ReadAll(resp.Body)
     resp.Body.Close()
     ```
     This ensures connection sockets return to Go's HTTP connection pool immediately.

---

### D. Zero-Dependency Retry Policy

We implemented a minimal, robust retry policy without external dependencies:

```go
func isRetryableErr(err error) bool {
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        return false // Never retry canceled contexts!
    }
    var netErr net.Error
    if errors.As(err, &netErr) {
        return netErr.Timeout() // Retry network timeouts
    }
    return errors.Is(err, io.EOF)
}

func isRetryableStatusCode(statusCode int) bool {
    return statusCode == http.StatusBadGateway ||          // 502
           statusCode == http.StatusServiceUnavailable ||  // 503
           statusCode == http.StatusGatewayTimeout         // 504
}
```

* **What gets retried**: Network timeouts (`net.Error`), `502 Bad Gateway`, `503 Service Unavailable`, `504 Gateway Timeout`.
* **What DOES NOT get retried**: `401 Unauthorized`, `403 Forbidden`, `404 Not Found`, JSON marshal errors, or context cancellations.

---

### E. Lean Error Handling (`APIError`)

When Meshery Server returns HTTP status `>= 400`, `do()` constructs an `APIError`:

```go
type APIError struct {
    StatusCode int    `json:"status_code"`
    Method     string `json:"method"`
    URL        string `json:"url"`
    Message    string `json:"message"`
    RawBody    []byte `json:"-"`
}
```

#### Why standard Go error checking?
Instead of adding non-standard helper methods like `err.IsNotFound()`, callers check errors using standard Go idioms:

```go
var apiErr *meshery.APIError
if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
    // Handle 404 Not Found
}
```

---

### F. Query Options Pattern (`ListOptions`)

Instead of writing function signatures with long parameter lists (`ListConnections(ctx, page, pageSize, search, sort, filter...)`), we use an options struct:

```go
type ListOptions struct {
    Page     int
    PageSize int
    Search   string
}

func (o ListOptions) EncodeValues() url.Values {
    q := url.Values{}
    if o.Page > 0 {
        q.Set("page", strconv.Itoa(o.Page))
    }
    if o.PageSize > 0 {
        q.Set("pagesize", strconv.Itoa(o.PageSize))
    }
    if o.Search != "" {
        q.Set("search", o.Search)
    }
    return q
}
```
Adding new query parameters in the future requires zero changes to method signatures.

---

## 5. Phase 1 Implemented Domain Methods (`methods.go`)

### 1. Health Ping (`Ping`)
* **Endpoint**: `GET /api/system/version`
* **Signature**: `Ping(ctx context.Context) (*Version, error)`
* **Returns**: Version details (`build`, `commit_sha`, `release_channel`).

### 2. List Connections (`ListConnections`)
* **Endpoint**: `GET /api/integrations/connections`
* **Signature**: `ListConnections(ctx context.Context, opts ListOptions) (*ConnectionResponse, error)`
* **Returns**: Paginated envelope containing `page`, `page_size`, `total_count`, and `connections []Connection`.

### 3. List Designs/Patterns (`ListPatterns`)
* **Endpoint**: `GET /api/pattern`
* **Signature**: `ListPatterns(ctx context.Context, opts ListOptions) (*PatternResponse, error)`
* **Returns**: Paginated envelope containing `page`, `page_size`, `total_count`, and `patterns []Pattern`.

---

## 6. Testing Strategy (`client_test.go`)

All 11 unit tests use Go's standard `httptest.Server` to mock HTTP responses without calling real external servers.

### Acceptance Test Mapping:
1. **`TestPing_ReachableServer_ReturnsVersionInfo`**: Verifies `Ping()` parses JSON version when reachable.
2. **`TestPing_UnreachableServer_ReturnsDescriptiveError`**: Connects to a closed port and asserts a network error is returned.
3. **`TestClient_AllHTTPCalls_IncludeAuthorizationHeader`**: Verifies `Authorization: Bearer <token>` is present on every request.
4. **`TestClient_RespectsConfiguredTimeout_DoesNotHang`**: Configures a `50ms` client timeout against an `httptest` handler that sleeps for `300ms`, verifying the client aborts within ~50ms and does not hang.
5. **`TestClient_PreservesRequestBodyOnRetry`**: Retries a POST request after a 502 status and asserts the second attempt receives the exact payload body.
6. **`TestClient_RetriesOn5xx_NoRetryOn4xx`**: Asserts 502 retries up to `RetryCount`, while 401 returns immediately without retrying.

---

## 7. What's Left (Roadmap & Next Steps)

Now that the core client foundation (Phase 1) is built and verified, here is what remains for future iterations:

### 1. Application Entry Point Wiring (`main.go`)
* Parse `--config` CLI flag.
* Apply configuration precedence: `DefaultConfig()` -> file -> env vars.
* Instantiate `client, err := meshery.NewClient(cfg)`.

### 2. Phase 2 Domain Wrapper Methods (Follow-up PRs)
Add domain methods as corresponding MCP tools are introduced:
* `ListWorkspaces(ctx context.Context, opts ListOptions)` (`GET /api/workspaces`)
* `ListEnvironments(ctx context.Context, opts ListOptions)` (`GET /api/environments`)
* `ListModels(ctx context.Context, opts ListOptions)` (`GET /api/meshmodels/models`)
* `ListPerformanceProfiles(ctx context.Context, opts ListOptions)` (`GET /api/user/performance/profiles`)

### 3. MCP Tool Handler Integration
* Connect MCP tool handlers (e.g. `ping_tool`, `list_connections_tool`, `list_patterns_tool`) to call `internal/meshery` client methods.

---

## Summary

By focusing Phase 1 on **transport excellence, safety, and strict alignment with Go best practices**, we have created a clean, zero-dependency REST client foundation that is robust, easy to test, and ready for MCP tool integration.
