package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

// --- helpers ---------------------------------------------------------------

// resultText extracts the text of a tool result's single content item,
// failing the test if the result doesn't have exactly one text item.
func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("expected exactly 1 content item, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

// callToolRequest builds an mcp.CallToolRequest carrying args as its
// arguments, matching the shape a real MCP client sends.
func callToolRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

// fakePerformanceClient is a PerformanceClient whose behavior is set per
// test via its function fields, for handler-level tests that don't need a
// real memoryPerformanceClient.
type fakePerformanceClient struct {
	runTestFunc    func(ctx context.Context, params PerformanceTestParams) (string, error)
	getTestFunc    func(ctx context.Context, testID string) (*PerformanceTestResult, error)
	listTestsFunc  func(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error)
	deleteTestFunc func(ctx context.Context, testID string) error
}

// RunTest delegates to f.runTestFunc.
func (f *fakePerformanceClient) RunTest(ctx context.Context, params PerformanceTestParams) (string, error) {
	return f.runTestFunc(ctx, params)
}

// GetTest delegates to f.getTestFunc.
func (f *fakePerformanceClient) GetTest(ctx context.Context, testID string) (*PerformanceTestResult, error) {
	return f.getTestFunc(ctx, testID)
}

// ListTests delegates to f.listTestsFunc.
func (f *fakePerformanceClient) ListTests(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error) {
	return f.listTestsFunc(ctx, page, pageSize)
}

// DeleteTest delegates to f.deleteTestFunc.
func (f *fakePerformanceClient) DeleteTest(ctx context.Context, testID string) error {
	return f.deleteTestFunc(ctx, testID)
}

// --- validateTestURL ---------------------------------------------------------

func TestValidateTestURL(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com", false},
		{"valid http with path", "http://example.com/foo?bar=baz", false},
		{"empty", "", true},
		{"no scheme", "example.com", true},
		{"bad scheme", "ftp://example.com", true},
		{"no host", "https://", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTestURL(tc.url)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateTestURL(%q) error = %v, wantErr %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

// --- parseFortioResult -------------------------------------------------------

func TestParseFortioResult(t *testing.T) {
	fixture := `{
		"ActualQPS": 998.5,
		"DurationHistogram": {
			"Count": 1000,
			"Percentiles": [
				{"Percentile": 50, "Value": 0.0012},
				{"Percentile": 90, "Value": 0.0025},
				{"Percentile": 99, "Value": 0.0041},
				{"Percentile": 99.9, "Value": 0.0067}
			]
		},
		"RetCodes": {
			"200": 950,
			"503": 40,
			"429": 10
		}
	}`
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(fixture), &raw); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	latency, throughput, errorRate, totalRequests, err := parseFortioResult(raw)
	if err != nil {
		t.Fatalf("parseFortioResult returned error: %v", err)
	}

	if throughput != 998.5 {
		t.Errorf("throughput = %v, want 998.5", throughput)
	}
	if totalRequests != 1000 {
		t.Errorf("totalRequests = %v, want 1000", totalRequests)
	}
	wantErrorRate := 50.0 / 1000.0
	if errorRate != wantErrorRate {
		t.Errorf("errorRate = %v, want %v", errorRate, wantErrorRate)
	}

	const tolerance = 1e-9
	checkClose := func(name string, got, want float64) {
		if diff := got - want; diff > tolerance || diff < -tolerance {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	checkClose("p50", latency.P50, 1.2)
	checkClose("p90", latency.P90, 2.5)
	checkClose("p99", latency.P99, 4.1)
	checkClose("p99.9", latency.P999, 6.7)
}

func TestParseFortioResult_MissingHistogram(t *testing.T) {
	if _, _, _, _, err := parseFortioResult(map[string]interface{}{"ActualQPS": 1.0}); err == nil {
		t.Fatal("expected error when DurationHistogram is missing")
	}
}

func TestParseFortioResult_Nil(t *testing.T) {
	if _, _, _, _, err := parseFortioResult(nil); err == nil {
		t.Fatal("expected error for nil runner results")
	}
}

// --- computePerformanceDelta -------------------------------------------------

func TestComputePerformanceDelta(t *testing.T) {
	testA := &PerformanceTestResult{
		Status: StatusCompleted,
		Latency: LatencyPercentiles{
			P50: 10, P90: 20, P99: 40, P999: 60,
		},
		ThroughputRPS: 500,
		ErrorRate:     0.01,
		TotalRequests: 15000,
	}
	testB := &PerformanceTestResult{
		Status: StatusCompleted,
		Latency: LatencyPercentiles{
			P50: 15, P90: 18, P99: 50, P999: 90,
		},
		ThroughputRPS: 480,
		ErrorRate:     0.04,
		TotalRequests: 14400,
	}

	delta := computePerformanceDelta(testA, testB)

	want := PerformanceDelta{
		LatencyP50Ms:  5,   // 15 - 10
		LatencyP90Ms:  -2,  // 18 - 20
		LatencyP99Ms:  10,  // 50 - 40
		LatencyP999Ms: 30,  // 90 - 60
		ThroughputRPS: -20, // 480 - 500
		ErrorRate:     0.03,
		TotalRequests: -600, // 14400 - 15000
	}

	if delta.LatencyP50Ms != want.LatencyP50Ms {
		t.Errorf("LatencyP50Ms = %v, want %v", delta.LatencyP50Ms, want.LatencyP50Ms)
	}
	if delta.LatencyP90Ms != want.LatencyP90Ms {
		t.Errorf("LatencyP90Ms = %v, want %v", delta.LatencyP90Ms, want.LatencyP90Ms)
	}
	if delta.LatencyP99Ms != want.LatencyP99Ms {
		t.Errorf("LatencyP99Ms = %v, want %v", delta.LatencyP99Ms, want.LatencyP99Ms)
	}
	if delta.LatencyP999Ms != want.LatencyP999Ms {
		t.Errorf("LatencyP999Ms = %v, want %v", delta.LatencyP999Ms, want.LatencyP999Ms)
	}
	if delta.ThroughputRPS != want.ThroughputRPS {
		t.Errorf("ThroughputRPS = %v, want %v", delta.ThroughputRPS, want.ThroughputRPS)
	}
	const tolerance = 1e-9
	if diff := delta.ErrorRate - want.ErrorRate; diff > tolerance || diff < -tolerance {
		t.Errorf("ErrorRate = %v, want %v", delta.ErrorRate, want.ErrorRate)
	}
	if delta.TotalRequests != want.TotalRequests {
		t.Errorf("TotalRequests = %v, want %v", delta.TotalRequests, want.TotalRequests)
	}
}

// --- memoryPerformanceClient --------------------------------------------------

// fakeRunner blocks in RunLoadTest until told to proceed, so tests can
// observe the "running" state before the test completes.
type fakeRunner struct {
	proceed chan struct{}
	raw     *meshery.RawResult
	err     error
}

// newFakeRunner builds a fakeRunner blocked until its proceed channel is closed.
func newFakeRunner() *fakeRunner {
	return &fakeRunner{proceed: make(chan struct{})}
}

// RunLoadTest blocks until f.proceed is closed, then returns the
// preconfigured f.raw/f.err.
func (f *fakeRunner) RunLoadTest(ctx context.Context, p meshery.RunLoadTestParams) (*meshery.RawResult, error) {
	<-f.proceed
	return f.raw, f.err
}

// waitForStatus polls client.GetTest until testID reaches status or a
// 2-second deadline passes, failing the test in the latter case.
func waitForStatus(t *testing.T, client PerformanceClient, testID, status string) *PerformanceTestResult {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		result, err := client.GetTest(context.Background(), testID)
		if err != nil {
			t.Fatalf("GetTest: %v", err)
		}
		if result.Status == status {
			return result
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("test %s did not reach status %q in time", testID, status)
	return nil
}

// fortioRawResult returns a meshery.RawResult wrapping a realistic Fortio
// runner-results fixture (500 requests, all 2xx, p50-p99.9 latencies set).
func fortioRawResult(t *testing.T) *meshery.RawResult {
	t.Helper()
	fixture := `{
		"ActualQPS": 100,
		"DurationHistogram": {
			"Count": 500,
			"Percentiles": [
				{"Percentile": 50, "Value": 0.001},
				{"Percentile": 90, "Value": 0.002},
				{"Percentile": 99, "Value": 0.004},
				{"Percentile": 99.9, "Value": 0.006}
			]
		},
		"RetCodes": {"200": 500}
	}`
	var runnerResults map[string]interface{}
	if err := json.Unmarshal([]byte(fixture), &runnerResults); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return &meshery.RawResult{TestID: "irrelevant", RunnerResults: runnerResults}
}

func TestMemoryPerformanceClient_RunningThenCompleted(t *testing.T) {
	runner := newFakeRunner()
	runner.raw = fortioRawResult(t)
	client := newMemoryPerformanceClient(runner)

	id, err := client.RunTest(context.Background(), PerformanceTestParams{
		Name: "smoke", URL: "https://example.com", DurationSeconds: 10, RPS: 100, ConcurrentRequests: 5,
	})
	if err != nil {
		t.Fatalf("RunTest: %v", err)
	}

	running, err := client.GetTest(context.Background(), id)
	if err != nil {
		t.Fatalf("GetTest: %v", err)
	}
	if running.Status != StatusRunning {
		t.Fatalf("status before completion = %q, want %q", running.Status, StatusRunning)
	}

	close(runner.proceed)

	completed := waitForStatus(t, client, id, StatusCompleted)
	if completed.TotalRequests != 500 {
		t.Errorf("TotalRequests = %d, want 500", completed.TotalRequests)
	}
	if completed.ThroughputRPS != 100 {
		t.Errorf("ThroughputRPS = %v, want 100", completed.ThroughputRPS)
	}
	if completed.CompletedAt == nil {
		t.Error("CompletedAt should be set once completed")
	}
}

func TestMemoryPerformanceClient_Failed(t *testing.T) {
	runner := newFakeRunner()
	runner.err = errors.New("boom")
	client := newMemoryPerformanceClient(runner)

	id, err := client.RunTest(context.Background(), PerformanceTestParams{
		Name: "will-fail", URL: "https://example.com", DurationSeconds: 5, RPS: 10, ConcurrentRequests: 1,
	})
	if err != nil {
		t.Fatalf("RunTest: %v", err)
	}
	close(runner.proceed)

	failed := waitForStatus(t, client, id, StatusFailed)
	if failed.ErrorMessage != "boom" {
		t.Errorf("ErrorMessage = %q, want %q", failed.ErrorMessage, "boom")
	}
}

func TestMemoryPerformanceClient_GetTest_NotFound(t *testing.T) {
	client := newMemoryPerformanceClient(newFakeRunner())
	if _, err := client.GetTest(context.Background(), "does-not-exist"); err == nil {
		t.Fatal("expected error for unknown test ID")
	}
}

func TestMemoryPerformanceClient_ListTests_Pagination(t *testing.T) {
	runner := newFakeRunner()
	close(runner.proceed) // let every run complete immediately
	runner.raw = fortioRawResult(t)
	client := newMemoryPerformanceClient(runner)

	var ids []string
	for i := 0; i < 5; i++ {
		id, err := client.RunTest(context.Background(), PerformanceTestParams{
			Name: "t", URL: "https://example.com", DurationSeconds: 1, RPS: 1, ConcurrentRequests: 1,
		})
		if err != nil {
			t.Fatalf("RunTest: %v", err)
		}
		ids = append(ids, id)
		waitForStatus(t, client, id, StatusCompleted)
	}

	page, err := client.ListTests(context.Background(), 0, 2)
	if err != nil {
		t.Fatalf("ListTests: %v", err)
	}
	if page.TotalCount != 5 {
		t.Errorf("TotalCount = %d, want 5", page.TotalCount)
	}
	if len(page.Tests) != 2 {
		t.Fatalf("len(Tests) = %d, want 2", len(page.Tests))
	}
	// Most recently started test (last id created) should come first.
	if page.Tests[0].ID != ids[len(ids)-1] {
		t.Errorf("first result ID = %q, want most recent %q", page.Tests[0].ID, ids[len(ids)-1])
	}

	lastPage, err := client.ListTests(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("ListTests: %v", err)
	}
	if len(lastPage.Tests) != 1 {
		t.Fatalf("len(Tests) on last page = %d, want 1", len(lastPage.Tests))
	}
}

// TestMemoryPerformanceClient_ListTests_LargePageNoPanic is a regression
// test for a page*pageSize int-overflow bug: a large enough page made the
// multiplication wrap around to a negative start, which then panicked on
// c.order[start:end]. An MCP client fully controls page, so this must
// degrade to an empty result, never panic.
func TestMemoryPerformanceClient_ListTests_LargePageNoPanic(t *testing.T) {
	runner := newFakeRunner()
	close(runner.proceed)
	runner.raw = fortioRawResult(t)
	client := newMemoryPerformanceClient(runner)

	id, err := client.RunTest(context.Background(), PerformanceTestParams{
		Name: "t", URL: "https://example.com", DurationSeconds: 1, RPS: 1, ConcurrentRequests: 1,
	})
	if err != nil {
		t.Fatalf("RunTest: %v", err)
	}
	waitForStatus(t, client, id, StatusCompleted)

	cases := []int{
		math.MaxInt,
		math.MaxInt / 2,
		1_000_000_000_000_000_000,
	}
	for _, page := range cases {
		page, pageSize := page, 25
		t.Run(fmt.Sprintf("page=%d", page), func(t *testing.T) {
			result, err := client.ListTests(context.Background(), page, pageSize)
			if err != nil {
				t.Fatalf("ListTests: %v", err)
			}
			if len(result.Tests) != 0 {
				t.Errorf("len(Tests) = %d, want 0 for an out-of-range page", len(result.Tests))
			}
			if result.TotalCount != 1 {
				t.Errorf("TotalCount = %d, want 1", result.TotalCount)
			}
		})
	}
}

// TestMemoryPerformanceClient_RetentionCap verifies that tracking more than
// maxTrackedTests completed tests evicts the oldest ones rather than
// growing the store without bound.
func TestMemoryPerformanceClient_RetentionCap(t *testing.T) {
	runner := newFakeRunner()
	close(runner.proceed)
	runner.raw = fortioRawResult(t)
	client := newMemoryPerformanceClient(runner)

	const extra = 5
	var ids []string
	for i := 0; i < maxTrackedTests+extra; i++ {
		id, err := client.RunTest(context.Background(), PerformanceTestParams{
			Name: "t", URL: "https://example.com", DurationSeconds: 1, RPS: 1, ConcurrentRequests: 1,
		})
		if err != nil {
			t.Fatalf("RunTest: %v", err)
		}
		ids = append(ids, id)
		waitForStatus(t, client, id, StatusCompleted)
	}

	page, err := client.ListTests(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("ListTests: %v", err)
	}
	if page.TotalCount != maxTrackedTests {
		t.Errorf("TotalCount = %d, want %d (capped)", page.TotalCount, maxTrackedTests)
	}

	// The oldest `extra` tests should have been evicted.
	for _, id := range ids[:extra] {
		if _, err := client.GetTest(context.Background(), id); err == nil {
			t.Errorf("GetTest(%s) succeeded, want it evicted as one of the oldest entries", id)
		}
	}
	// The most recent test must still be tracked.
	if _, err := client.GetTest(context.Background(), ids[len(ids)-1]); err != nil {
		t.Errorf("GetTest on the most recent test failed: %v", err)
	}
}

func TestMemoryPerformanceClient_DeleteTest(t *testing.T) {
	runner := newFakeRunner()
	close(runner.proceed)
	runner.raw = fortioRawResult(t)
	client := newMemoryPerformanceClient(runner)

	id, err := client.RunTest(context.Background(), PerformanceTestParams{
		Name: "t", URL: "https://example.com", DurationSeconds: 1, RPS: 1, ConcurrentRequests: 1,
	})
	if err != nil {
		t.Fatalf("RunTest: %v", err)
	}
	waitForStatus(t, client, id, StatusCompleted)

	if err := client.DeleteTest(context.Background(), id); err != nil {
		t.Fatalf("DeleteTest: %v", err)
	}
	if _, err := client.GetTest(context.Background(), id); err == nil {
		t.Fatal("expected GetTest to fail after delete")
	}
	if err := client.DeleteTest(context.Background(), id); err == nil {
		t.Fatal("expected DeleteTest to fail for an already-deleted test")
	}
}

// cancelAwareRunner blocks in RunLoadTest until its context is canceled,
// then reports the context's error on done - used to verify that deleting a
// running test actually cancels the load test's context.
type cancelAwareRunner struct {
	started chan struct{}
	done    chan error
}

// newCancelAwareRunner builds a cancelAwareRunner.
func newCancelAwareRunner() *cancelAwareRunner {
	return &cancelAwareRunner{started: make(chan struct{}), done: make(chan error, 1)}
}

// RunLoadTest closes r.started, blocks until ctx is canceled, then reports
// ctx.Err() on r.done and returns it as the call's error.
func (r *cancelAwareRunner) RunLoadTest(ctx context.Context, p meshery.RunLoadTestParams) (*meshery.RawResult, error) {
	close(r.started)
	<-ctx.Done()
	err := ctx.Err()
	r.done <- err
	return nil, err
}

// TestMemoryPerformanceClient_DeleteTest_CancelsRunning verifies that
// deleting a still-running test cancels its context, so the client stops
// waiting on it promptly instead of holding the connection/goroutine open
// for the test's full remaining duration.
func TestMemoryPerformanceClient_DeleteTest_CancelsRunning(t *testing.T) {
	runner := newCancelAwareRunner()
	client := newMemoryPerformanceClient(runner)

	id, err := client.RunTest(context.Background(), PerformanceTestParams{
		Name: "t", URL: "https://example.com", DurationSeconds: 300, RPS: 1, ConcurrentRequests: 1,
	})
	if err != nil {
		t.Fatalf("RunTest: %v", err)
	}

	select {
	case <-runner.started:
	case <-time.After(2 * time.Second):
		t.Fatal("RunLoadTest was never called")
	}

	if err := client.DeleteTest(context.Background(), id); err != nil {
		t.Fatalf("DeleteTest: %v", err)
	}

	select {
	case err := <-runner.done:
		if err != context.Canceled {
			t.Errorf("RunLoadTest's context error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("DeleteTest did not cancel the in-flight RunLoadTest call")
	}
}

// --- tool handlers ------------------------------------------------------------

func TestRunPerformanceTestHandler_Validation(t *testing.T) {
	client := &fakePerformanceClient{
		runTestFunc: func(ctx context.Context, params PerformanceTestParams) (string, error) {
			return "should-not-be-called", nil
		},
	}
	handler := runPerformanceTestHandler(client)

	cases := []struct {
		name        string
		args        map[string]interface{}
		wantErrText string
	}{
		{"missing name", map[string]interface{}{"url": "https://example.com", "duration": float64(10), "rps": float64(10), "concurrent_requests": float64(1)}, "name"},
		{"invalid url", map[string]interface{}{"name": "t", "url": "not-a-url", "duration": float64(10), "rps": float64(10), "concurrent_requests": float64(1)}, "url"},
		{"zero duration", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(0), "rps": float64(10), "concurrent_requests": float64(1)}, "duration"},
		{"duration too large", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(maxDurationSeconds + 1), "rps": float64(10), "concurrent_requests": float64(1)}, "duration"},
		{"negative rps", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(10), "rps": float64(-1), "concurrent_requests": float64(1)}, "rps"},
		{"rps too large", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(10), "rps": float64(maxRPS + 1), "concurrent_requests": float64(1)}, "rps"},
		{"zero concurrent_requests", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(10), "rps": float64(10), "concurrent_requests": float64(0)}, "concurrent_requests"},
		{"concurrent_requests too large", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(10), "rps": float64(10), "concurrent_requests": float64(maxConcurrentRequests + 1)}, "concurrent_requests"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := handler(context.Background(), callToolRequest(tc.args))
			if err != nil {
				t.Fatalf("handler returned Go error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected IsError=true for invalid input, text: %s", resultText(t, res))
			}
			text := resultText(t, res)
			if !strings.Contains(text, tc.wantErrText) {
				t.Errorf("error text = %q, want it to mention %q (checking the right field was rejected, not a coincidental failure)", text, tc.wantErrText)
			}
		})
	}
}

func TestRunPerformanceTestHandler_Success(t *testing.T) {
	var gotParams PerformanceTestParams
	client := &fakePerformanceClient{
		runTestFunc: func(ctx context.Context, params PerformanceTestParams) (string, error) {
			gotParams = params
			return "test-123", nil
		},
	}
	handler := runPerformanceTestHandler(client)

	res, err := handler(context.Background(), callToolRequest(map[string]interface{}{
		"name": "load-test-1", "url": "https://example.com", "duration": float64(30), "rps": float64(200), "concurrent_requests": float64(20),
	}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, res))
	}

	var body map[string]interface{}
	if err := json.Unmarshal([]byte(resultText(t, res)), &body); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if body["test_id"] != "test-123" {
		t.Errorf("test_id = %v, want test-123", body["test_id"])
	}
	if body["status"] != StatusRunning {
		t.Errorf("status = %v, want %v", body["status"], StatusRunning)
	}

	if gotParams.DurationSeconds != 30 || gotParams.RPS != 200 || gotParams.ConcurrentRequests != 20 {
		t.Errorf("unexpected params passed to client: %+v", gotParams)
	}
}

func TestGetPerformanceTestHandler(t *testing.T) {
	client := &fakePerformanceClient{
		getTestFunc: func(ctx context.Context, testID string) (*PerformanceTestResult, error) {
			if testID != "abc" {
				return nil, errors.New("not found")
			}
			return &PerformanceTestResult{ID: "abc", Status: StatusCompleted, TotalRequests: 42}, nil
		},
	}
	handler := getPerformanceTestHandler(client)

	res, err := handler(context.Background(), callToolRequest(map[string]interface{}{"test_id": "abc"}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, res))
	}
	var got PerformanceTestResult
	if err := json.Unmarshal([]byte(resultText(t, res)), &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got.Status != StatusCompleted || got.TotalRequests != 42 {
		t.Errorf("unexpected result: %+v", got)
	}

	res, err = handler(context.Background(), callToolRequest(map[string]interface{}{"test_id": "missing"}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for unknown test_id")
	}
}

func TestComparePerformanceTestsHandler_Delta(t *testing.T) {
	testA := &PerformanceTestResult{
		ID: "a", Status: StatusCompleted,
		Latency:       LatencyPercentiles{P50: 10, P90: 20, P99: 40, P999: 60},
		ThroughputRPS: 500, ErrorRate: 0.01, TotalRequests: 15000,
	}
	testB := &PerformanceTestResult{
		ID: "b", Status: StatusCompleted,
		Latency:       LatencyPercentiles{P50: 15, P90: 18, P99: 50, P999: 90},
		ThroughputRPS: 480, ErrorRate: 0.04, TotalRequests: 14400,
	}
	client := &fakePerformanceClient{
		getTestFunc: func(ctx context.Context, testID string) (*PerformanceTestResult, error) {
			switch testID {
			case "a":
				return testA, nil
			case "b":
				return testB, nil
			}
			return nil, errors.New("not found")
		},
	}
	handler := comparePerformanceTestsHandler(client)

	res, err := handler(context.Background(), callToolRequest(map[string]interface{}{"test_id_a": "a", "test_id_b": "b"}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, res))
	}

	var body struct {
		Delta PerformanceDelta `json:"delta"`
	}
	if err := json.Unmarshal([]byte(resultText(t, res)), &body); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if body.Delta.LatencyP50Ms != 5 || body.Delta.LatencyP90Ms != -2 || body.Delta.LatencyP99Ms != 10 || body.Delta.LatencyP999Ms != 30 {
		t.Errorf("unexpected latency delta: %+v", body.Delta)
	}
	if body.Delta.ThroughputRPS != -20 {
		t.Errorf("ThroughputRPS delta = %v, want -20", body.Delta.ThroughputRPS)
	}
	if body.Delta.TotalRequests != -600 {
		t.Errorf("TotalRequests delta = %v, want -600", body.Delta.TotalRequests)
	}
}

func TestComparePerformanceTestsHandler_RejectsRunningTest(t *testing.T) {
	client := &fakePerformanceClient{
		getTestFunc: func(ctx context.Context, testID string) (*PerformanceTestResult, error) {
			return &PerformanceTestResult{ID: testID, Status: StatusRunning}, nil
		},
	}
	handler := comparePerformanceTestsHandler(client)

	res, err := handler(context.Background(), callToolRequest(map[string]interface{}{"test_id_a": "a", "test_id_b": "b"}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when comparing a still-running test")
	}
}

// TestComparePerformanceTestsHandler_NotFound covers comparing against a
// test_id that doesn't exist (e.g. it was deleted), not just one that's
// still running.
func TestComparePerformanceTestsHandler_NotFound(t *testing.T) {
	client := &fakePerformanceClient{
		getTestFunc: func(ctx context.Context, testID string) (*PerformanceTestResult, error) {
			if testID == "a" {
				return &PerformanceTestResult{ID: "a", Status: StatusCompleted}, nil
			}
			return nil, fmt.Errorf("performance test %q not found", testID)
		},
	}
	handler := comparePerformanceTestsHandler(client)

	res, err := handler(context.Background(), callToolRequest(map[string]interface{}{"test_id_a": "a", "test_id_b": "missing"}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when comparing against a not-found test_id")
	}
}

func TestListPerformanceTestsHandler(t *testing.T) {
	client := &fakePerformanceClient{
		listTestsFunc: func(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error) {
			if page != 1 || pageSize != 10 {
				t.Errorf("unexpected page/pageSize: %d/%d", page, pageSize)
			}
			return &PerformanceTestPage{Page: page, PageSize: pageSize, TotalCount: 25}, nil
		},
	}
	handler := listPerformanceTestsHandler(client)

	res, err := handler(context.Background(), callToolRequest(map[string]interface{}{"page": float64(1), "page_size": float64(10)}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, res))
	}
}

// TestListPerformanceTestsHandler_Validation covers the bounds checks that
// were previously untested: a negative page, a non-positive page_size, and
// an oversized page_size getting clamped to maxPageSize rather than passed
// through as-is.
func TestListPerformanceTestsHandler_Validation(t *testing.T) {
	t.Run("negative page", func(t *testing.T) {
		client := &fakePerformanceClient{
			listTestsFunc: func(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error) {
				t.Fatal("ListTests should not be called for a negative page")
				return nil, nil
			},
		}
		res, err := listPerformanceTestsHandler(client)(context.Background(), callToolRequest(map[string]interface{}{"page": float64(-1)}))
		if err != nil {
			t.Fatalf("handler returned Go error: %v", err)
		}
		if !res.IsError {
			t.Fatal("expected error result for a negative page")
		}
	})

	t.Run("zero page_size", func(t *testing.T) {
		client := &fakePerformanceClient{
			listTestsFunc: func(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error) {
				t.Fatal("ListTests should not be called for a zero page_size")
				return nil, nil
			},
		}
		res, err := listPerformanceTestsHandler(client)(context.Background(), callToolRequest(map[string]interface{}{"page_size": float64(0)}))
		if err != nil {
			t.Fatalf("handler returned Go error: %v", err)
		}
		if !res.IsError {
			t.Fatal("expected error result for a zero page_size")
		}
	})

	t.Run("oversized page_size is clamped", func(t *testing.T) {
		var gotPageSize int
		client := &fakePerformanceClient{
			listTestsFunc: func(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error) {
				gotPageSize = pageSize
				return &PerformanceTestPage{Page: page, PageSize: pageSize}, nil
			},
		}
		res, err := listPerformanceTestsHandler(client)(context.Background(), callToolRequest(map[string]interface{}{"page_size": float64(1_000_000)}))
		if err != nil {
			t.Fatalf("handler returned Go error: %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected error result: %s", resultText(t, res))
		}
		if gotPageSize != maxPageSize {
			t.Errorf("page_size passed to client = %d, want it clamped to %d", gotPageSize, maxPageSize)
		}
	})
}

func TestDeletePerformanceTestHandler(t *testing.T) {
	var deletedID string
	client := &fakePerformanceClient{
		deleteTestFunc: func(ctx context.Context, testID string) error {
			deletedID = testID
			return nil
		},
	}
	handler := deletePerformanceTestHandler(client)

	res, err := handler(context.Background(), callToolRequest(map[string]interface{}{"test_id": "xyz"}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", resultText(t, res))
	}
	if deletedID != "xyz" {
		t.Errorf("deletedID = %q, want xyz", deletedID)
	}

	var body map[string]interface{}
	if err := json.Unmarshal([]byte(resultText(t, res)), &body); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if body["deleted"] != true {
		t.Errorf("deleted = %v, want true", body["deleted"])
	}
}

// TestDeletePerformanceTestHandler_Error covers the client rejecting the
// delete (e.g. an unknown test_id) - previously only the success path was
// exercised at the handler level.
func TestDeletePerformanceTestHandler_Error(t *testing.T) {
	client := &fakePerformanceClient{
		deleteTestFunc: func(ctx context.Context, testID string) error {
			return fmt.Errorf("performance test %q not found", testID)
		},
	}
	handler := deletePerformanceTestHandler(client)

	res, err := handler(context.Background(), callToolRequest(map[string]interface{}{"test_id": "missing"}))
	if err != nil {
		t.Fatalf("handler returned Go error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result when the client rejects the delete")
	}
}
