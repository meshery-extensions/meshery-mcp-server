// Package meshery is a minimal HTTP client for the subset of the Meshery
// Server REST API needed to drive performance (load) tests.
//
// Meshery's performance testing backend uses Fortio as its load generator
// (https://github.com/fortio/fortio) — there is no "Nighthawk" integration
// anywhere in Meshery core. A load test is started with a GET request to
// /api/perf/profile, which the server keeps open as a Server-Sent-Events
// stream until the test finishes, at which point it emits a final frame
// carrying the raw Fortio result JSON. There is no separate "submit, then
// poll by ID" endpoint on the Meshery side; internal/tools.performance.go
// builds that model on top of this streaming call.
package meshery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/meshery-extensions/meshery-mcp-server/internal/config"
)

// Client talks to a Meshery Server instance.
type Client struct {
	baseURL    string
	token      string
	provider   string
	httpClient *http.Client
}

// NewClient builds a Client from cfg. If httpClient is nil, a client with no
// request timeout is used, since load tests can legitimately run for
// minutes and the request stays open for the test's full duration.
func NewClient(cfg *config.Config, httpClient *http.Client) (*Client, error) {
	if cfg == nil || cfg.MeshServerURL == "" {
		return nil, fmt.Errorf("meshery server URL is required")
	}
	u, err := url.Parse(cfg.MeshServerURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid meshery server URL %q", cfg.MeshServerURL)
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		baseURL:    strings.TrimRight(cfg.MeshServerURL, "/"),
		token:      cfg.MeshAPIToken,
		provider:   cfg.MeshProvider,
		httpClient: httpClient,
	}, nil
}

func (c *Client) applyAuthHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json, text/event-stream")
	if c.token != "" {
		req.AddCookie(&http.Cookie{Name: "token", Value: c.token})
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.provider != "" {
		req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: c.provider})
	}
}

// RunLoadTestParams mirrors the query parameters accepted by Meshery's
// /api/perf/profile load-test endpoint (see server/handlers/load_test_handler.go
// in meshery/meshery).
type RunLoadTestParams struct {
	TestUUID           string
	Name               string
	URL                string
	DurationSeconds    int
	RPS                int
	ConcurrentRequests int
}

// LoadTestFrame mirrors one Server-Sent-Event frame Meshery streams back
// while a load test runs (models.LoadTestResponse in meshery/meshery).
type LoadTestFrame struct {
	Status  string     `json:"status"`
	Message string     `json:"message"`
	Result  *RawResult `json:"result"`
}

// RawResult mirrors the fields of models.MesheryResult this client needs.
// RunnerResults is the raw Fortio JSON payload, passed through unparsed.
type RawResult struct {
	TestID        string                 `json:"testId"`
	Name          string                 `json:"name"`
	RunnerResults map[string]interface{} `json:"runnerResults"`
}

// RunLoadTest starts a load test against Meshery's SSE load-test endpoint
// and blocks until the stream reports success or error. Callers that want
// asynchronous behavior (as internal/tools.performance.go does) should run
// this in a goroutine.
func (c *Client) RunLoadTest(ctx context.Context, p RunLoadTestParams) (*RawResult, error) {
	q := url.Values{}
	q.Set("name", p.Name)
	q.Set("url", p.URL)
	q.Set("t", strconv.Itoa(p.DurationSeconds))
	q.Set("dur", "s")
	q.Set("qps", strconv.Itoa(p.RPS))
	q.Set("c", strconv.Itoa(p.ConcurrentRequests))
	q.Set("uuid", p.TestUUID)

	reqURL := c.baseURL + "/api/perf/profile?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build load test request: %w", err)
	}
	c.applyAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call meshery load test endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("meshery load test endpoint returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		data, ok := sseData(scanner.Text())
		if !ok {
			continue
		}

		var frame LoadTestFrame
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			continue
		}

		switch frame.Status {
		case "error":
			return nil, fmt.Errorf("load test failed: %s", frame.Message)
		case "success":
			if frame.Result != nil {
				return frame.Result, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read load test stream: %w", err)
	}
	return nil, fmt.Errorf("load test stream closed before a result was received")
}

// sseData extracts the payload from a "data: ..." Server-Sent-Events line.
func sseData(line string) (string, bool) {
	data, ok := strings.CutPrefix(line, "data:")
	if !ok {
		return "", false
	}
	data = strings.TrimSpace(data)
	if data == "" {
		return "", false
	}
	return data, true
}
