package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

// Performance test status values. There is no "pending" state: RunTest
// starts the underlying Fortio-backed load test immediately and the test is
// "running" from the moment its ID is handed back.
const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// PerformanceTestParams are the validated inputs to start a load test.
type PerformanceTestParams struct {
	Name               string
	URL                string
	DurationSeconds    int
	RPS                int
	ConcurrentRequests int
}

// LatencyPercentiles holds latency percentiles in milliseconds.
//
// Field names carry a unit suffix (p50_ms, not p50; throughput_rps, not
// throughput) rather than matching issue #14's bare wording verbatim. This
// is an intentional, documented deviation for clarity in tool output an AI
// agent will read directly - open to renaming to match the issue exactly if
// maintainers prefer strict field-name parity.
type LatencyPercentiles struct {
	P50  float64 `json:"p50_ms"`
	P90  float64 `json:"p90_ms"`
	P99  float64 `json:"p99_ms"`
	P999 float64 `json:"p99_9_ms"`
}

// PerformanceTestResult is the full record of one test run.
type PerformanceTestResult struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	URL           string             `json:"url"`
	Status        string             `json:"status"`
	ErrorMessage  string             `json:"error_message,omitempty"`
	StartedAt     time.Time          `json:"started_at"`
	CompletedAt   *time.Time         `json:"completed_at,omitempty"`
	Latency       LatencyPercentiles `json:"latency"`
	ThroughputRPS float64            `json:"throughput_rps"`
	ErrorRate     float64            `json:"error_rate"`
	TotalRequests int64              `json:"total_requests"`
}

// PerformanceTestSummary is the shape returned by list_performance_tests.
type PerformanceTestSummary struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// PerformanceTestPage is one page of test summaries.
type PerformanceTestPage struct {
	Page       int                      `json:"page"`
	PageSize   int                      `json:"page_size"`
	TotalCount int                      `json:"total_count"`
	Tests      []PerformanceTestSummary `json:"tests"`
}

// PerformanceDelta is test B minus test A across every comparable metric.
type PerformanceDelta struct {
	LatencyP50Ms  float64 `json:"latency_p50_ms_delta"`
	LatencyP90Ms  float64 `json:"latency_p90_ms_delta"`
	LatencyP99Ms  float64 `json:"latency_p99_ms_delta"`
	LatencyP999Ms float64 `json:"latency_p99_9_ms_delta"`
	ThroughputRPS float64 `json:"throughput_rps_delta"`
	ErrorRate     float64 `json:"error_rate_delta"`
	TotalRequests int64   `json:"total_requests_delta"`
}

// PerformanceClient starts, reads, lists, and deletes performance tests. The
// concrete implementation wraps Meshery's Fortio-backed load test endpoint;
// tests use a fake implementation.
type PerformanceClient interface {
	RunTest(ctx context.Context, params PerformanceTestParams) (string, error)
	GetTest(ctx context.Context, testID string) (*PerformanceTestResult, error)
	ListTests(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error)
	DeleteTest(ctx context.Context, testID string) error
}

// RegisterPerformanceTools registers the performance-testing MCP tools. If
// client is nil, a default client backed by the Meshery server configured in
// the process environment is used.
func RegisterPerformanceTools(s *server.MCPServer, client PerformanceClient) {
	if client == nil {
		client = NewDefaultPerformanceClient()
	}

	runTool := mcp.NewTool("run_performance_test",
		mcp.WithDescription("Start a Meshery performance (load) test against a URL and return the generated test ID along with its initial status. The test runs asynchronously; poll get_performance_test with the returned test ID to check progress and results."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("name", mcp.Required(), mcp.Description("A name for this test run.")),
		mcp.WithString("url", mcp.Required(), mcp.Description("Target URL to load test, e.g. https://example.com.")),
		mcp.WithNumber("duration", mcp.Required(), mcp.Description("Test duration in seconds (positive integer).")),
		mcp.WithNumber("rps", mcp.Required(), mcp.Description("Target requests per second (positive integer).")),
		mcp.WithNumber("concurrent_requests", mcp.Required(), mcp.Description("Number of concurrent connections to use (positive integer).")),
	)
	s.AddTool(runTool, runPerformanceTestHandler(client))

	getTool := mcp.NewTool("get_performance_test",
		mcp.WithDescription("Get the status and results of a performance test by ID: running/completed/failed, latency percentiles (p50, p90, p99, p99.9), throughput, error rate, and total requests."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("test_id", mcp.Required(), mcp.Description("ID of the performance test returned by run_performance_test.")),
	)
	s.AddTool(getTool, getPerformanceTestHandler(client))

	listTool := mcp.NewTool("list_performance_tests",
		mcp.WithDescription("List performance test runs known to this MCP server, most recently started first."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithNumber("page", mcp.Description("Page number, 0-indexed (default 0).")),
		mcp.WithNumber("page_size", mcp.Description("Number of results per page (default 25).")),
	)
	s.AddTool(listTool, listPerformanceTestsHandler(client))

	compareTool := mcp.NewTool("compare_performance_tests",
		mcp.WithDescription("Compare two completed performance tests and return the delta (test B minus test A) across latency percentiles, throughput, error rate, and total requests."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithString("test_id_a", mcp.Required(), mcp.Description("ID of the baseline performance test.")),
		mcp.WithString("test_id_b", mcp.Required(), mcp.Description("ID of the performance test to compare against the baseline.")),
	)
	s.AddTool(compareTool, comparePerformanceTestsHandler(client))

	deleteTool := mcp.NewTool("delete_performance_test",
		mcp.WithDescription("Delete a performance test run tracked by this MCP server."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithString("test_id", mcp.Required(), mcp.Description("ID of the performance test to delete.")),
	)
	s.AddTool(deleteTool, deletePerformanceTestHandler(client))
}

// runPerformanceTestHandler validates run_performance_test's inputs and
// starts a load test via client.
func runPerformanceTestHandler(client PerformanceClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("name")
		if err != nil || strings.TrimSpace(name) == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		rawURL, err := req.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError("url is required"), nil
		}
		if err := validateTestURL(rawURL); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid url: %v", err)), nil
		}

		duration, err := req.RequireInt("duration")
		if err != nil {
			return mcp.NewToolResultError("duration is required and must be a number of seconds"), nil
		}
		if duration <= 0 {
			return mcp.NewToolResultError("duration must be a positive number of seconds"), nil
		}

		rps, err := req.RequireInt("rps")
		if err != nil {
			return mcp.NewToolResultError("rps is required and must be a number"), nil
		}
		if rps <= 0 {
			return mcp.NewToolResultError("rps must be a positive number"), nil
		}

		concurrent, err := req.RequireInt("concurrent_requests")
		if err != nil {
			return mcp.NewToolResultError("concurrent_requests is required and must be a number"), nil
		}
		if concurrent <= 0 {
			return mcp.NewToolResultError("concurrent_requests must be a positive number"), nil
		}

		testID, err := client.RunTest(ctx, PerformanceTestParams{
			Name:               name,
			URL:                rawURL,
			DurationSeconds:    duration,
			RPS:                rps,
			ConcurrentRequests: concurrent,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to start performance test: %v", err)), nil
		}

		return jsonToolResult(map[string]interface{}{
			"test_id": testID,
			"status":  StatusRunning,
			"name":    name,
			"url":     rawURL,
		})
	}
}

// getPerformanceTestHandler returns one test's current status and results.
func getPerformanceTestHandler(client PerformanceClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		testID, err := req.RequireString("test_id")
		if err != nil || strings.TrimSpace(testID) == "" {
			return mcp.NewToolResultError("test_id is required"), nil
		}

		result, err := client.GetTest(ctx, testID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get performance test: %v", err)), nil
		}
		return jsonToolResult(result)
	}
}

// listPerformanceTestsHandler returns one page of tracked test summaries.
func listPerformanceTestsHandler(client PerformanceClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		page := req.GetInt("page", 0)
		pageSize := req.GetInt("page_size", 25)
		if page < 0 {
			return mcp.NewToolResultError("page must be zero or a positive number"), nil
		}
		if pageSize <= 0 {
			return mcp.NewToolResultError("page_size must be a positive number"), nil
		}

		result, err := client.ListTests(ctx, page, pageSize)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list performance tests: %v", err)), nil
		}
		return jsonToolResult(result)
	}
}

// comparePerformanceTestsHandler diffs two completed tests, rejecting the
// request if either is missing or not yet completed.
func comparePerformanceTestsHandler(client PerformanceClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idA, err := req.RequireString("test_id_a")
		if err != nil || strings.TrimSpace(idA) == "" {
			return mcp.NewToolResultError("test_id_a is required"), nil
		}
		idB, err := req.RequireString("test_id_b")
		if err != nil || strings.TrimSpace(idB) == "" {
			return mcp.NewToolResultError("test_id_b is required"), nil
		}

		testA, err := client.GetTest(ctx, idA)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get test_id_a: %v", err)), nil
		}
		testB, err := client.GetTest(ctx, idB)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get test_id_b: %v", err)), nil
		}
		if testA.Status != StatusCompleted {
			return mcp.NewToolResultError(fmt.Sprintf("test_id_a (%s) is not completed yet (status: %s)", idA, testA.Status)), nil
		}
		if testB.Status != StatusCompleted {
			return mcp.NewToolResultError(fmt.Sprintf("test_id_b (%s) is not completed yet (status: %s)", idB, testB.Status)), nil
		}

		delta := computePerformanceDelta(testA, testB)
		return jsonToolResult(map[string]interface{}{
			"test_a": testA,
			"test_b": testB,
			"delta":  delta,
		})
	}
}

// deletePerformanceTestHandler removes a tracked test from client.
func deletePerformanceTestHandler(client PerformanceClient) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		testID, err := req.RequireString("test_id")
		if err != nil || strings.TrimSpace(testID) == "" {
			return mcp.NewToolResultError("test_id is required"), nil
		}

		if err := client.DeleteTest(ctx, testID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete performance test: %v", err)), nil
		}
		return jsonToolResult(map[string]interface{}{
			"test_id": testID,
			"deleted": true,
		})
	}
}

// computePerformanceDelta returns test B's metrics minus test A's, for
// every latency percentile plus throughput, error rate, and total requests.
func computePerformanceDelta(a, b *PerformanceTestResult) PerformanceDelta {
	return PerformanceDelta{
		LatencyP50Ms:  b.Latency.P50 - a.Latency.P50,
		LatencyP90Ms:  b.Latency.P90 - a.Latency.P90,
		LatencyP99Ms:  b.Latency.P99 - a.Latency.P99,
		LatencyP999Ms: b.Latency.P999 - a.Latency.P999,
		ThroughputRPS: b.ThroughputRPS - a.ThroughputRPS,
		ErrorRate:     b.ErrorRate - a.ErrorRate,
		TotalRequests: b.TotalRequests - a.TotalRequests,
	}
}

// validateTestURL requires an absolute http(s) URL with a host.
func validateTestURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("could not parse url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("url must include a host")
	}
	return nil
}

// jsonToolResult marshals v as indented JSON and wraps it in a text tool result.
func jsonToolResult(v interface{}) (*mcp.CallToolResult, error) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// loadTestRunner is the minimal surface memoryPerformanceClient needs from
// the Meshery backend; *meshery.Client satisfies it. Kept as an interface so
// tests can supply a fake without a real Meshery server.
type loadTestRunner interface {
	RunLoadTest(ctx context.Context, p meshery.RunLoadTestParams) (*meshery.RawResult, error)
}

// memoryPerformanceClient implements PerformanceClient. Meshery's load-test
// endpoint has no server-generated run ID or poll-by-ID model of its own
// (starting a test opens a blocking SSE stream that runs for the test's
// full duration) - this type builds the "start test, get an ID back
// immediately, poll for status" contract on top of that by generating the
// ID itself, running the SSE call in a background goroutine, and tracking
// each test's state in memory.
type memoryPerformanceClient struct {
	runner loadTestRunner

	mu    sync.Mutex
	tests map[string]*PerformanceTestResult
	order []string // most recently started test ID first
}

// newMemoryPerformanceClient builds a memoryPerformanceClient that runs load
// tests through runner.
func newMemoryPerformanceClient(runner loadTestRunner) *memoryPerformanceClient {
	return &memoryPerformanceClient{
		runner: runner,
		tests:  make(map[string]*PerformanceTestResult),
	}
}

// RunTest generates a test ID, records it as running, and starts the load
// test in the background, returning the ID immediately.
func (c *memoryPerformanceClient) RunTest(ctx context.Context, params PerformanceTestParams) (string, error) {
	id := uuid.NewString()
	result := &PerformanceTestResult{
		ID:        id,
		Name:      params.Name,
		URL:       params.URL,
		Status:    StatusRunning,
		StartedAt: time.Now().UTC(),
	}

	c.mu.Lock()
	c.tests[id] = result
	c.order = append([]string{id}, c.order...)
	c.mu.Unlock()

	// The load test outlives the MCP tool call that started it, so it runs
	// against a background context rather than the request's context.
	go c.runTest(id, params)

	return id, nil
}

// runTest runs the load test identified by id to completion (or failure)
// and records the outcome. Intended to be called via `go c.runTest(...)`.
func (c *memoryPerformanceClient) runTest(id string, params PerformanceTestParams) {
	raw, err := c.runner.RunLoadTest(context.Background(), meshery.RunLoadTestParams{
		TestUUID:           id,
		Name:               params.Name,
		URL:                params.URL,
		DurationSeconds:    params.DurationSeconds,
		RPS:                params.RPS,
		ConcurrentRequests: params.ConcurrentRequests,
	})

	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.tests[id]
	if !ok {
		return // deleted while the test was running
	}

	completedAt := time.Now().UTC()
	result.CompletedAt = &completedAt

	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = err.Error()
		return
	}

	latency, throughput, errorRate, totalRequests, err := parseFortioResult(raw.RunnerResults)
	if err != nil {
		result.Status = StatusFailed
		result.ErrorMessage = err.Error()
		return
	}

	result.Status = StatusCompleted
	result.Latency = latency
	result.ThroughputRPS = throughput
	result.ErrorRate = errorRate
	result.TotalRequests = totalRequests
}

// GetTest returns a copy of the tracked test's current state.
func (c *memoryPerformanceClient) GetTest(ctx context.Context, testID string) (*PerformanceTestResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	result, ok := c.tests[testID]
	if !ok {
		return nil, fmt.Errorf("performance test %q not found", testID)
	}
	resultCopy := *result
	return &resultCopy, nil
}

// ListTests returns one page of tracked test summaries, most recently
// started first.
func (c *memoryPerformanceClient) ListTests(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	total := len(c.order)
	summaries := []PerformanceTestSummary{}
	start := page * pageSize
	if start < total {
		end := start + pageSize
		if end > total {
			end = total
		}
		for _, id := range c.order[start:end] {
			r := c.tests[id]
			summaries = append(summaries, PerformanceTestSummary{
				ID:        r.ID,
				Name:      r.Name,
				URL:       r.URL,
				Status:    r.Status,
				StartedAt: r.StartedAt,
			})
		}
	}

	return &PerformanceTestPage{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
		Tests:      summaries,
	}, nil
}

// DeleteTest removes a tracked test, returning an error if it is unknown.
func (c *memoryPerformanceClient) DeleteTest(ctx context.Context, testID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.tests[testID]; !ok {
		return fmt.Errorf("performance test %q not found", testID)
	}
	delete(c.tests, testID)
	for i, id := range c.order {
		if id == testID {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
	return nil
}

// NewDefaultPerformanceClient builds a PerformanceClient backed by the
// Meshery server configured in the process environment (MESHERY_SERVER_URL,
// MESHERY_API_TOKEN, MESHERY_PROVIDER).
func NewDefaultPerformanceClient() PerformanceClient {
	cfg := config.Load()
	client, err := meshery.NewClient(cfg, nil)
	if err != nil {
		return &errPerformanceClient{err: err}
	}
	return newMemoryPerformanceClient(client)
}

// errPerformanceClient returns a fixed error from every call; used when the
// default client could not be constructed (e.g. bad MESHERY_SERVER_URL).
type errPerformanceClient struct{ err error }

// RunTest always returns the client construction error.
func (c *errPerformanceClient) RunTest(context.Context, PerformanceTestParams) (string, error) {
	return "", c.err
}

// GetTest always returns the client construction error.
func (c *errPerformanceClient) GetTest(context.Context, string) (*PerformanceTestResult, error) {
	return nil, c.err
}

// ListTests always returns the client construction error.
func (c *errPerformanceClient) ListTests(context.Context, int, int) (*PerformanceTestPage, error) {
	return nil, c.err
}

// DeleteTest always returns the client construction error.
func (c *errPerformanceClient) DeleteTest(context.Context, string) error {
	return c.err
}

// parseFortioResult extracts latency percentiles, throughput, error rate,
// and total request count from a raw Fortio JSON result blob (the value of
// MesheryResult.RunnerResults / PerformanceResult.RunnerResults on the
// Meshery side). p99.9 is only populated if the Fortio run requested that
// percentile; Meshery's typed performance-profile schema does not carry it,
// so it is read directly from the raw histogram here.
func parseFortioResult(raw map[string]interface{}) (LatencyPercentiles, float64, float64, int64, error) {
	var latency LatencyPercentiles

	if raw == nil {
		return latency, 0, 0, 0, fmt.Errorf("empty runner results")
	}

	var throughput float64
	if v, ok := raw["ActualQPS"].(float64); ok {
		throughput = v
	}

	hist, ok := raw["DurationHistogram"].(map[string]interface{})
	if !ok {
		return latency, throughput, 0, 0, fmt.Errorf("runner results missing DurationHistogram")
	}

	var totalRequests int64
	if v, ok := hist["Count"].(float64); ok {
		totalRequests = int64(v)
	}

	if percentiles, ok := hist["Percentiles"].([]interface{}); ok {
		for _, entry := range percentiles {
			pm, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			pct, _ := pm["Percentile"].(float64)
			valueSeconds, _ := pm["Value"].(float64)
			ms := valueSeconds * 1000

			switch pct {
			case 50:
				latency.P50 = ms
			case 90:
				latency.P90 = ms
			case 99:
				latency.P99 = ms
			case 99.9:
				latency.P999 = ms
			}
		}
	}

	var errorCount int64
	if retCodes, ok := raw["RetCodes"].(map[string]interface{}); ok {
		for code, countRaw := range retCodes {
			count, _ := countRaw.(float64)
			if !strings.HasPrefix(code, "2") {
				errorCount += int64(count)
			}
		}
	}

	var errorRate float64
	if totalRequests > 0 {
		errorRate = float64(errorCount) / float64(totalRequests)
	}

	return latency, throughput, errorRate, totalRequests, nil
}
