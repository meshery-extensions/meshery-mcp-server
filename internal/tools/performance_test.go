package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/meshery-extensions/meshery-mcp-server/internal/meshery"
)

// --- helpers ---------------------------------------------------------------

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

func callToolRequest(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

type fakePerformanceClient struct {
	runTestFunc    func(ctx context.Context, params PerformanceTestParams) (string, error)
	getTestFunc    func(ctx context.Context, testID string) (*PerformanceTestResult, error)
	listTestsFunc  func(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error)
	deleteTestFunc func(ctx context.Context, testID string) error
}

func (f *fakePerformanceClient) RunTest(ctx context.Context, params PerformanceTestParams) (string, error) {
	return f.runTestFunc(ctx, params)
}

func (f *fakePerformanceClient) GetTest(ctx context.Context, testID string) (*PerformanceTestResult, error) {
	return f.getTestFunc(ctx, testID)
}

func (f *fakePerformanceClient) ListTests(ctx context.Context, page, pageSize int) (*PerformanceTestPage, error) {
	return f.listTestsFunc(ctx, page, pageSize)
}

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

func newFakeRunner() *fakeRunner {
	return &fakeRunner{proceed: make(chan struct{})}
}

func (f *fakeRunner) RunLoadTest(ctx context.Context, p meshery.RunLoadTestParams) (*meshery.RawResult, error) {
	<-f.proceed
	return f.raw, f.err
}

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

// --- tool handlers ------------------------------------------------------------

func TestRunPerformanceTestHandler_Validation(t *testing.T) {
	client := &fakePerformanceClient{
		runTestFunc: func(ctx context.Context, params PerformanceTestParams) (string, error) {
			return "should-not-be-called", nil
		},
	}
	handler := runPerformanceTestHandler(client)

	cases := []struct {
		name string
		args map[string]interface{}
	}{
		{"missing name", map[string]interface{}{"url": "https://example.com", "duration": float64(10), "rps": float64(10), "concurrent_requests": float64(1)}},
		{"invalid url", map[string]interface{}{"name": "t", "url": "not-a-url", "duration": float64(10), "rps": float64(10), "concurrent_requests": float64(1)}},
		{"zero duration", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(0), "rps": float64(10), "concurrent_requests": float64(1)}},
		{"negative rps", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(10), "rps": float64(-1), "concurrent_requests": float64(1)}},
		{"zero concurrent_requests", map[string]interface{}{"name": "t", "url": "https://example.com", "duration": float64(10), "rps": float64(10), "concurrent_requests": float64(0)}},
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
