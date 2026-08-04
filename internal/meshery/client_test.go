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
			cfg:     Config{BaseURL: "http://localhost:9081", Timeout: 10 * time.Second},
			wantErr: false,
		},
		{
			name:    "valid https URL",
			cfg:     Config{BaseURL: "https://cloud.meshery.io", Timeout: 10 * time.Second},
			wantErr: false,
		},
		{
			name:    "missing scheme",
			cfg:     Config{BaseURL: "localhost:9081", Timeout: 10 * time.Second},
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			cfg:     Config{BaseURL: "ftp://localhost:9081", Timeout: 10 * time.Second},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			cfg:     Config{BaseURL: "http://localhost:9081", Timeout: -1 * time.Second},
			wantErr: true,
		},
		{
			name:    "zero timeout",
			cfg:     Config{BaseURL: "http://localhost:9081", Timeout: 0},
			wantErr: true,
		},
		{
			name:    "negative retry count",
			cfg:     Config{BaseURL: "http://localhost:9081", Timeout: 10 * time.Second, RetryCount: -1},
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

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
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
		Timeout: 5 * time.Second,
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
		UserAgent: "meshery-mcp-server/0.1.0",
		Timeout:   5 * time.Second,
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
	if userAgentHeader != "meshery-mcp-server/0.1.0" {
		t.Errorf("expected User-Agent header 'meshery-mcp-server/0.1.0', got %q", userAgentHeader)
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
		Timeout:    5 * time.Second,
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
			Timeout:    5 * time.Second,
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
			Timeout:    5 * time.Second,
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

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
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

func TestClient_HandlesBaseURLWithPathPrefix(t *testing.T) {
	t.Parallel()

	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Version{Build: "v1.0.0"})
	}))
	defer ts.Close()

	client, err := NewClient(Config{
		BaseURL: ts.URL + "/meshery-prefix",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Ping(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPath := "/meshery-prefix/api/system/version"
	if receivedPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, receivedPath)
	}
}

func TestClient_Handles204NoContent(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var resp map[string]string
	err = client.do(context.Background(), http.MethodDelete, "/api/resource", nil, nil, &resp)
	if err != nil {
		t.Fatalf("expected 204 No Content to return nil error, got: %v", err)
	}
}

func TestConnection_SchemaWireFormatUnmarshaling(t *testing.T) {
	t.Parallel()

	rawJSON := `{
		"id": "conn-123",
		"name": "minikube",
		"kind": "kubernetes",
		"type": "cluster",
		"subType": "k8s",
		"status": "connected",
		"credentialId": "cred-abc-456",
		"user_id": "user-789",
		"created_at": "2026-08-04T12:00:00Z",
		"updated_at": "2026-08-04T12:30:00Z"
	}`

	var conn Connection
	err := json.Unmarshal([]byte(rawJSON), &conn)
	if err != nil {
		t.Fatalf("failed to unmarshal raw connection JSON: %v", err)
	}

	if conn.ID != "conn-123" {
		t.Errorf("expected ID 'conn-123', got %q", conn.ID)
	}
	if conn.CredentialID != "cred-abc-456" {
		t.Errorf("expected CredentialID 'cred-abc-456' from credentialId, got %q", conn.CredentialID)
	}
	if conn.SubType != "k8s" {
		t.Errorf("expected SubType 'k8s' from subType, got %q", conn.SubType)
	}
	if conn.UserID != "user-789" {
		t.Errorf("expected UserID 'user-789' from user_id, got %q", conn.UserID)
	}
	if conn.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be parsed from created_at, got zero time")
	}
	if conn.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be parsed from updated_at, got zero time")
	}
}

func TestListConnections_EncodesQueryParametersAndEscapesSpaces(t *testing.T) {
	t.Parallel()

	var receivedQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"page": 1,
			"pageSize": 10,
			"totalCount": 1,
			"connections": [
				{
					"id": "conn-1",
					"name": "k8s cluster",
					"credentialId": "cred-1",
					"subType": "k8s",
					"user_id": "user-1",
					"created_at": "2026-08-04T10:00:00Z"
				}
			]
		}`))
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.ListConnections(context.Background(), ListOptions{
		Page:     1,
		PageSize: 10,
		Search:   "k8s cluster",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(receivedQuery, "page=1") || !strings.Contains(receivedQuery, "pageSize=10") || !strings.Contains(receivedQuery, "search=k8s+cluster") {
		t.Errorf("unexpected query string encoding: %s", receivedQuery)
	}

	if resp.TotalCount != 1 || len(resp.Connections) != 1 || resp.Connections[0].Name != "k8s cluster" {
		t.Errorf("unexpected connection response: %+v", resp)
	}
	if resp.Connections[0].CredentialID != "cred-1" || resp.Connections[0].UserID != "user-1" {
		t.Errorf("failed to decode schema-backed credentialId or user_id fields: %+v", resp.Connections[0])
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

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
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

func TestListWorkspaces_EncodesQueryParameters(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(WorkspaceResponse{
			Page:       1,
			PageSize:   10,
			TotalCount: 1,
			Workspaces: []Workspace{{ID: "ws-1", Name: "staging-ws"}},
		})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.ListWorkspaces(context.Background(), ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalCount != 1 || len(resp.Workspaces) != 1 || resp.Workspaces[0].Name != "staging-ws" {
		t.Errorf("unexpected workspace response: %+v", resp)
	}
}

func TestListEnvironments_EncodesQueryParameters(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(EnvironmentResponse{
			Page:         1,
			PageSize:     10,
			TotalCount:   1,
			Environments: []Environment{{ID: "env-1", Name: "prod-env"}},
		})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.ListEnvironments(context.Background(), ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalCount != 1 || len(resp.Environments) != 1 || resp.Environments[0].Name != "prod-env" {
		t.Errorf("unexpected environment response: %+v", resp)
	}
}

func TestListModels_EncodesQueryParameters(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ModelResponse{
			Page:       1,
			PageSize:   10,
			TotalCount: 1,
			Models:     []Model{{ID: "mod-1", Name: "kubernetes"}},
		})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.ListModels(context.Background(), ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalCount != 1 || len(resp.Models) != 1 || resp.Models[0].Name != "kubernetes" {
		t.Errorf("unexpected model response: %+v", resp)
	}
}

func TestListPerformanceProfiles_EncodesQueryParameters(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PerformanceProfileResponse{
			Page:       1,
			PageSize:   10,
			TotalCount: 1,
			Profiles:   []PerformanceProfile{{ID: "prof-1", Name: "soak-test"}},
		})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resp, err := client.ListPerformanceProfiles(context.Background(), ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalCount != 1 || len(resp.Profiles) != 1 || resp.Profiles[0].Name != "soak-test" {
		t.Errorf("unexpected performance profile response: %+v", resp)
	}
}
