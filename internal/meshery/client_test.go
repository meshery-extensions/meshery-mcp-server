package meshery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
)

// testClient builds a Client pointed at an httptest.Server URL, failing the
// test if construction is rejected.
func testClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	c, err := NewClient(&config.Config{MeshServerURL: serverURL}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// writeSSEFrame marshals frame and writes it as one Server-Sent-Events
// "data:" line, flushing immediately, matching how meshery/meshery's
// loadTestHelperHandler streams results.
func writeSSEFrame(w http.ResponseWriter, frame LoadTestFrame) {
	b, err := json.Marshal(frame)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func TestNewClient_RejectsPlaintextNonLoopback(t *testing.T) {
	_, err := NewClient(&config.Config{MeshServerURL: "http://meshery.example.com:9081"}, nil)
	if err == nil {
		t.Fatal("expected NewClient to reject a non-loopback http URL")
	}
	if !strings.Contains(err.Error(), "plaintext") {
		t.Errorf("error = %v, want it to mention plaintext", err)
	}
}

func TestNewClient_AllowsLoopbackHTTP(t *testing.T) {
	cases := []string{
		"http://localhost:9081",
		"http://127.0.0.1:9081",
		"http://[::1]:9081",
	}
	for _, u := range cases {
		if _, err := NewClient(&config.Config{MeshServerURL: u}, nil); err != nil {
			t.Errorf("NewClient(%q) = %v, want no error", u, err)
		}
	}
}

func TestNewClient_AllowsHTTPS(t *testing.T) {
	if _, err := NewClient(&config.Config{MeshServerURL: "https://meshery.example.com"}, nil); err != nil {
		t.Errorf("NewClient with https URL = %v, want no error", err)
	}
}

func TestNewClient_RejectsInvalidURL(t *testing.T) {
	cases := []string{"", "not-a-url", "://bad"}
	for _, u := range cases {
		if _, err := NewClient(&config.Config{MeshServerURL: u}, nil); err == nil {
			t.Errorf("NewClient(%q) expected error, got nil", u)
		}
	}
}

func TestNewClient_RequiresConfig(t *testing.T) {
	if _, err := NewClient(nil, nil); err == nil {
		t.Fatal("expected error for nil config")
	}
}

// TestRunLoadTest_Success covers a normal SSE stream: an info frame followed
// by a success frame carrying the result.
func TestRunLoadTest_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSEFrame(w, LoadTestFrame{Status: "info", Message: "Initiating load test . . . "})
		writeSSEFrame(w, LoadTestFrame{
			Status: "success",
			Result: &RawResult{
				TestID:        "abc-123",
				Name:          "my-test",
				RunnerResults: map[string]interface{}{"ActualQPS": 100.5},
			},
		})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL)
	result, err := c.RunLoadTest(context.Background(), RunLoadTestParams{
		Name: "my-test", URL: "https://example.com", DurationSeconds: 1, RPS: 1, ConcurrentRequests: 1,
	})
	if err != nil {
		t.Fatalf("RunLoadTest: %v", err)
	}
	if result.TestID != "abc-123" {
		t.Errorf("TestID = %q, want abc-123", result.TestID)
	}
	if qps, _ := result.RunnerResults["ActualQPS"].(float64); qps != 100.5 {
		t.Errorf("ActualQPS = %v, want 100.5", qps)
	}
}

// TestRunLoadTest_ErrorFrame covers the server reporting failure in-band via
// a status:"error" SSE frame instead of an HTTP error status.
func TestRunLoadTest_ErrorFrame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSEFrame(w, LoadTestFrame{Status: "info", Message: "Initiating load test . . . "})
		writeSSEFrame(w, LoadTestFrame{Status: "error", Message: "unable to perform"})
	}))
	defer srv.Close()

	c := testClient(t, srv.URL)
	_, err := c.RunLoadTest(context.Background(), RunLoadTestParams{
		Name: "t", URL: "https://example.com", DurationSeconds: 1, RPS: 1, ConcurrentRequests: 1,
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "unable to perform") {
		t.Errorf("error = %v, want it to include the frame's message", err)
	}
}

// TestRunLoadTest_DroppedStream covers the server closing the connection
// (e.g. it crashed, or the process was killed) without ever sending a
// success or error frame.
func TestRunLoadTest_DroppedStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSEFrame(w, LoadTestFrame{Status: "info", Message: "Initiating load test . . . "})
		// Handler returns without a terminal frame; the connection closes
		// and RunLoadTest sees EOF.
	}))
	defer srv.Close()

	c := testClient(t, srv.URL)
	_, err := c.RunLoadTest(context.Background(), RunLoadTestParams{
		Name: "t", URL: "https://example.com", DurationSeconds: 1, RPS: 1, ConcurrentRequests: 1,
	})
	if err == nil {
		t.Fatal("expected an error for a stream that closed with no result")
	}
	if !strings.Contains(err.Error(), "closed before a result") {
		t.Errorf("error = %v, want it to say the stream closed before a result", err)
	}
}

// TestRunLoadTest_NonOKStatus covers the endpoint rejecting the request
// outright (e.g. auth failure, bad request) before any SSE stream starts.
func TestRunLoadTest_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := testClient(t, srv.URL)
	_, err := c.RunLoadTest(context.Background(), RunLoadTestParams{
		Name: "t", URL: "https://example.com", DurationSeconds: 1, RPS: 1, ConcurrentRequests: 1,
	})
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %v, want it to mention status 403", err)
	}
}

// TestRunLoadTest_SendsExpectedRequest asserts the query parameters and
// auth headers/cookies sent match what meshery/meshery's load test endpoint
// and applyAuthHeaders are documented to expect.
func TestRunLoadTest_SendsExpectedRequest(t *testing.T) {
	var gotQuery url.Values
	var gotAuthHeader, gotTokenCookie, gotProviderCookie string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		gotAuthHeader = r.Header.Get("Authorization")
		if c, err := r.Cookie("token"); err == nil {
			gotTokenCookie = c.Value
		}
		if c, err := r.Cookie("meshery-provider"); err == nil {
			gotProviderCookie = c.Value
		}
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSEFrame(w, LoadTestFrame{Status: "success", Result: &RawResult{RunnerResults: map[string]interface{}{}}})
	}))
	defer srv.Close()

	c, err := NewClient(&config.Config{MeshServerURL: srv.URL, MeshAPIToken: "tok-123", MeshProvider: "Meshery"}, nil)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.RunLoadTest(context.Background(), RunLoadTestParams{
		TestUUID: "uuid-1", Name: "my-test", URL: "https://target.example",
		DurationSeconds: 30, RPS: 50, ConcurrentRequests: 5,
	})
	if err != nil {
		t.Fatalf("RunLoadTest: %v", err)
	}

	if got := gotQuery.Get("name"); got != "my-test" {
		t.Errorf("query name = %q, want my-test", got)
	}
	if got := gotQuery.Get("url"); got != "https://target.example" {
		t.Errorf("query url = %q, want https://target.example", got)
	}
	if got := gotQuery.Get("t"); got != "30" {
		t.Errorf("query t = %q, want 30", got)
	}
	if got := gotQuery.Get("qps"); got != "50" {
		t.Errorf("query qps = %q, want 50", got)
	}
	if got := gotQuery.Get("c"); got != "5" {
		t.Errorf("query c = %q, want 5", got)
	}
	if got := gotQuery.Get("uuid"); got != "uuid-1" {
		t.Errorf("query uuid = %q, want uuid-1", got)
	}
	if gotAuthHeader != "Bearer tok-123" {
		t.Errorf("Authorization header = %q, want Bearer tok-123", gotAuthHeader)
	}
	if gotTokenCookie != "tok-123" {
		t.Errorf("token cookie = %q, want tok-123", gotTokenCookie)
	}
	if gotProviderCookie != "Meshery" {
		t.Errorf("meshery-provider cookie = %q, want Meshery", gotProviderCookie)
	}
}
