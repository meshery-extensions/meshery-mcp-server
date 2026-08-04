package meshery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient_ValidatesBaseURLSchemeAndHost(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid http URL",
			cfg:     Config{BaseURL: "http://localhost:9081"},
			wantErr: false,
		},
		{
			name:    "valid https URL",
			cfg:     Config{BaseURL: "https://cloud.meshery.io"},
			wantErr: false,
		},
		{
			name:    "missing scheme",
			cfg:     Config{BaseURL: "localhost:9081"},
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			cfg:     Config{BaseURL: "ftp://localhost:9081"},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			cfg:     Config{BaseURL: "http://localhost:9081", Timeout: -1 * time.Second},
			wantErr: true,
		},
		{
			name:    "negative retry count",
			cfg:     Config{BaseURL: "http://localhost:9081", RetryCount: -1},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient(tc.cfg)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %s, got nil", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for %s: %v", tc.name, err)
			}
			if !tc.wantErr && client == nil {
				t.Errorf("expected non-nil client for %s", tc.name)
			}
		})
	}
}

func TestPing_ReachableServer_ReturnsVersionInfo(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/version" {
			t.Errorf("expected path /api/system/version, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Version{
			Build:          "v0.7.0",
			CommitSHA:      "abcdef123456",
			ReleaseChannel: "stable",
		})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	v, err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if v.Build != "v0.7.0" || v.CommitSHA != "abcdef123456" || v.ReleaseChannel != "stable" {
		t.Errorf("unexpected version response: %+v", v)
	}
}

func TestPing_UnreachableServer_ReturnsDescriptiveError(t *testing.T) {
	t.Parallel()

	// Connect to closed port to trigger network error
	client, err := NewClient(Config{
		BaseURL:    "http://127.0.0.1:59999",
		Timeout:    100 * time.Millisecond,
		RetryCount: 0,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error for unreachable server, got nil")
	}

	if !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "refused") && !strings.Contains(err.Error(), "dial") {
		t.Errorf("expected descriptive network error, got: %v", err)
	}
}

func TestClient_AllHTTPCalls_IncludeAuthorizationHeader(t *testing.T) {
	t.Parallel()

	token := "test-secret-token"
	receivedToken := ""

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Version{Build: "v1.0"})
	}))
	defer ts.Close()

	client, err := NewClient(Config{
		BaseURL: ts.URL,
		Token:   token,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Ping(context.Background())
	if err != nil {
		t.Fatalf("expected ping success, got: %v", err)
	}

	expectedAuth := "Bearer " + token
	if receivedToken != expectedAuth {
		t.Errorf("expected Authorization header %q, got %q", expectedAuth, receivedToken)
	}
}

func TestClient_RespectsConfiguredTimeout_DoesNotHang(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, err := NewClient(Config{
		BaseURL: ts.URL,
		Timeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	start := time.Now()
	_, err = client.Ping(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if elapsed >= 250*time.Millisecond {
		t.Errorf("expected call to abort near 50ms, elapsed time was %v", elapsed)
	}
}

func TestClient_InjectsStandardHeaders(t *testing.T) {
	t.Parallel()

	var acceptHeader, userAgentHeader string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")
		userAgentHeader = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	client, err := NewClient(Config{
		BaseURL:   ts.URL,
		UserAgent: "custom-agent/1.0",
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Ping(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if acceptHeader != "application/json" {
		t.Errorf("expected Accept header 'application/json', got %q", acceptHeader)
	}
	if userAgentHeader != "custom-agent/1.0" {
		t.Errorf("expected User-Agent header 'custom-agent/1.0', got %q", userAgentHeader)
	}
}

func TestClient_PreservesRequestBodyOnRetry(t *testing.T) {
	t.Parallel()

	var attempts int32
	var receivedBodies []string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		body, _ := io.ReadAll(r.Body)
		receivedBodies = append(receivedBodies, string(body))

		if count == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	client, err := NewClient(Config{
		BaseURL:    ts.URL,
		RetryCount: 1,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	reqPayload := map[string]string{"foo": "bar"}
	var respPayload map[string]string

	err = client.do(context.Background(), http.MethodPost, "/test", nil, reqPayload, &respPayload)
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}

	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}

	expectedBody := `{"foo":"bar"}`
	for i, body := range receivedBodies {
		if body != expectedBody {
			t.Errorf("attempt %d: expected body %q, got %q", i+1, expectedBody, body)
		}
	}
}

func TestClient_RetriesOn5xx_NoRetryOn4xx(t *testing.T) {
	t.Parallel()

	t.Run("retry on 502 Bad Gateway", func(t *testing.T) {
		var attempts int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer ts.Close()

		client, err := NewClient(Config{
			BaseURL:    ts.URL,
			RetryCount: 2,
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.Ping(context.Background())
		if err == nil {
			t.Fatal("expected error on 502, got nil")
		}

		if atomic.LoadInt32(&attempts) != 3 { // Initial + 2 retries
			t.Errorf("expected 3 total attempts, got %d", attempts)
		}
	})

	t.Run("no retry on 401 Unauthorized", func(t *testing.T) {
		var attempts int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
		}))
		defer ts.Close()

		client, err := NewClient(Config{
			BaseURL:    ts.URL,
			RetryCount: 2,
		})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.Ping(context.Background())
		if err == nil {
			t.Fatal("expected error on 401, got nil")
		}

		if atomic.LoadInt32(&attempts) != 1 { // Should NOT retry 401
			t.Errorf("expected 1 attempt for 401, got %d", attempts)
		}
	})
}

func TestAPIError_StandardErrorHandling(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"connection not found"}`))
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ListConnections(context.Background(), ListOptions{})
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected error of type *APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected StatusCode 404, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "connection not found" {
		t.Errorf("expected parsed JSON message 'connection not found', got %q", apiErr.Message)
	}
}

func TestListConnections_EncodesQueryParameters(t *testing.T) {
	t.Parallel()

	var receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ConnectionResponse{
			Page:        1,
			PageSize:    10,
			TotalCount:  1,
			Connections: []Connection{{ID: "conn-1", Name: "k8s-cluster"}},
		})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.ListConnections(context.Background(), ListOptions{
		Page:     1,
		PageSize: 10,
		Search:   "k8s",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(receivedQuery, "page=1") || !strings.Contains(receivedQuery, "pagesize=10") || !strings.Contains(receivedQuery, "search=k8s") {
		t.Errorf("unexpected query string encoding: %s", receivedQuery)
	}

	if resp.TotalCount != 1 || len(resp.Connections) != 1 || resp.Connections[0].Name != "k8s-cluster" {
		t.Errorf("unexpected connection response: %+v", resp)
	}
}

func TestListPatterns_EncodesQueryParameters(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PatternResponse{
			Page:       1,
			PageSize:   10,
			TotalCount: 1,
			Patterns:   []Pattern{{ID: "pat-1", Name: "istio-app"}},
		})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.ListPatterns(context.Background(), ListOptions{
		Page:     1,
		PageSize: 10,
		Search:   "istio",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalCount != 1 || len(resp.Patterns) != 1 || resp.Patterns[0].Name != "istio-app" {
		t.Errorf("unexpected pattern response: %+v", resp)
	}
}
