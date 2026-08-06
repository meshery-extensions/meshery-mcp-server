package meshery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestServer(t *testing.T, handler http.HandlerFunc, opts ...func(*Config)) (*Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	cfg := Config{BaseURL: ts.URL, Timeout: 5 * time.Second}
	for _, opt := range opts {
		opt(&cfg)
	}
	client, err := NewClient(cfg)
	if err != nil {
		ts.Close()
		t.Fatalf("failed to create test client: %v", err)
	}
	return client, ts
}

func TestNewClient_ValidatesBaseURLSchemeAndHost(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{name: "valid http URL", cfg: Config{BaseURL: "http://localhost:9081", Timeout: 10 * time.Second}},
		{name: "valid https URL", cfg: Config{BaseURL: "https://cloud.meshery.io", Timeout: 10 * time.Second}},
		{name: "http localhost with token", cfg: Config{BaseURL: "http://localhost:9081", Token: "secret", Timeout: 10 * time.Second}},
		{name: "http 127.0.0.1 with token", cfg: Config{BaseURL: "http://127.0.0.1:9081", Token: "secret", Timeout: 10 * time.Second}},
		{name: "http ipv6 loopback with token", cfg: Config{BaseURL: "http://[::1]:9081", Token: "secret", Timeout: 10 * time.Second}},
		{name: "http non-loopback with token fails", cfg: Config{BaseURL: "http://example.com:9081", Token: "secret", Timeout: 10 * time.Second}, wantErr: true},
		{name: "http non-loopback without token succeeds", cfg: Config{BaseURL: "http://example.com:9081", Timeout: 10 * time.Second}},
		{name: "https non-loopback with token succeeds", cfg: Config{BaseURL: "https://example.com:9081", Token: "secret", Timeout: 10 * time.Second}},
		{name: "missing scheme", cfg: Config{BaseURL: "localhost:9081", Timeout: 10 * time.Second}, wantErr: true},
		{name: "invalid scheme", cfg: Config{BaseURL: "ftp://localhost:9081", Timeout: 10 * time.Second}, wantErr: true},
		{name: "negative timeout", cfg: Config{BaseURL: "http://localhost:9081", Timeout: -1 * time.Second}, wantErr: true},
		{name: "zero timeout", cfg: Config{BaseURL: "http://localhost:9081", Timeout: 0}, wantErr: true},
		{name: "negative retry count", cfg: Config{BaseURL: "http://localhost:9081", Timeout: 10 * time.Second, RetryCount: -1}, wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, err := NewClient(tc.cfg)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for %s, got nil", tc.name)
			}
			if !tc.wantErr && (err != nil || client == nil) {
				t.Errorf("unexpected error or nil client for %s: %v", tc.name, err)
			}
		})
	}
}

func TestNewClient_CustomHTTPClient_RespectsTimeout(t *testing.T) {
	t.Parallel()

	customClient := &http.Client{Timeout: 1 * time.Second}
	client, err := NewClient(Config{
		BaseURL:    "http://localhost:9081",
		Timeout:    10 * time.Second,
		HTTPClient: customClient,
	})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if client.http != customClient {
		t.Fatal("expected supplied HTTPClient instance to be reused")
	}
	if client.http.Timeout != 10*time.Second {
		t.Errorf("expected custom client timeout to be set to 10s, got %v", client.http.Timeout)
	}
}

func TestPing_ReachableServer_ReturnsVersionInfo(t *testing.T) {
	t.Parallel()

	client, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/system/version" {
			t.Errorf("expected path /api/system/version, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Version{Build: "v0.7.0", CommitSHA: "abcdef123456", ReleaseChannel: "stable"})
	})
	defer ts.Close()

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
	var receivedToken string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Version{Build: "v1.0"})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL, Token: token, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Ping(context.Background())
	if err != nil {
		t.Fatalf("expected ping success, got: %v", err)
	}
	if expectedAuth := "Bearer " + token; receivedToken != expectedAuth {
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

	client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	start := time.Now()
	_, err = client.Ping(context.Background())
	if elapsed := time.Since(start); err == nil {
		t.Fatal("expected timeout error, got nil")
	} else if elapsed >= 150*time.Millisecond {
		t.Errorf("expected call to abort near 50ms, elapsed time was %v", elapsed)
	}
}

func TestClient_InjectsStandardHeaders(t *testing.T) {
	t.Parallel()

	var acceptHeader, userAgentHeader string
	client, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")
		userAgentHeader = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}, func(cfg *Config) {
		cfg.UserAgent = "meshery-mcp-server/0.1.0"
	})
	defer ts.Close()

	_, err := client.Ping(context.Background())
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

func TestClient_HandlesBaseURLWithPathPrefix(t *testing.T) {
	t.Parallel()

	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Version{Build: "v1.0.0"})
	}))
	defer ts.Close()

	client, err := NewClient(Config{BaseURL: ts.URL + "/meshery-prefix", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Ping(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expectedPath := "/meshery-prefix/api/system/version"; receivedPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, receivedPath)
	}
}

func TestClient_Handles204NoContent(t *testing.T) {
	t.Parallel()

	client, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	defer ts.Close()

	var resp map[string]string
	if err := client.do(context.Background(), http.MethodDelete, "/api/resource", nil, nil, &resp); err != nil {
		t.Fatalf("expected 204 No Content to return nil error, got: %v", err)
	}
}

func TestAPIError_StandardErrorHandling(t *testing.T) {
	t.Parallel()

	client, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"connection not found"}`))
	})
	defer ts.Close()

	_, err := client.ListConnections(context.Background(), ListOptions{})
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

func TestClient_RetriesSafeMethods(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		method := method
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			var attempts int32

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if atomic.AddInt32(&attempts, 1) == 1 {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				w.WriteHeader(http.StatusOK)
				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(`{"status":"ok"}`))
				}
			}))
			defer ts.Close()

			client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second, RetryCount: 1})
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			var respPayload map[string]string
			var respPtr any
			if method == http.MethodGet {
				respPtr = &respPayload
			}
			if err := client.do(context.Background(), method, "/test", nil, nil, respPtr); err != nil {
				t.Fatalf("expected retry success for %s, got error: %v", method, err)
			}
			if atomic.LoadInt32(&attempts) != 2 {
				t.Errorf("expected 2 attempts for %s, got %d", method, attempts)
			}
		})
	}
}

func TestClient_NoRetryOnUnsafeMethods(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		method := method
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			var attempts int32

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attempts, 1)
				w.WriteHeader(http.StatusBadGateway)
			}))
			defer ts.Close()

			client, err := NewClient(Config{BaseURL: ts.URL, Timeout: 5 * time.Second, RetryCount: 3})
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			if err := client.do(context.Background(), method, "/test", nil, nil, nil); err == nil {
				t.Fatalf("expected error for %s on 502, got nil", method)
			}
			if atomic.LoadInt32(&attempts) != 1 {
				t.Errorf("expected exactly 1 attempt for unsafe method %s, got %d", method, attempts)
			}
		})
	}
}

func TestClient_RetriesOn5xx_NoRetryOn4xx(t *testing.T) {
	t.Parallel()

	t.Run("retry on 502 Bad Gateway", func(t *testing.T) {
		var attempts int32
		client, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusBadGateway)
		}, func(cfg *Config) {
			cfg.RetryCount = 2
		})
		defer ts.Close()

		if _, err := client.Ping(context.Background()); err == nil {
			t.Fatal("expected error on 502, got nil")
		}
		if atomic.LoadInt32(&attempts) != 3 {
			t.Errorf("expected 3 total attempts, got %d", attempts)
		}
	})

	t.Run("no retry on 401 Unauthorized", func(t *testing.T) {
		var attempts int32
		client, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&attempts, 1)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
		}, func(cfg *Config) {
			cfg.RetryCount = 2
		})
		defer ts.Close()

		if _, err := client.Ping(context.Background()); err == nil {
			t.Fatal("expected error on 401, got nil")
		}
		if atomic.LoadInt32(&attempts) != 1 {
			t.Errorf("expected 1 attempt for 401, got %d", attempts)
		}
	})
}

func TestListWrappers_EncodesQueryParametersAndParsesResponse(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		path       string
		sampleBody string
		call       func(ctx context.Context, c *Client) (int, error)
	}{
		{
			name:       "ListConnections",
			path:       "/api/integrations/connections",
			sampleBody: `{"page":1,"pageSize":10,"totalCount":1,"connections":[{"id":"conn-1","name":"k8s cluster"}]}`,
			call: func(ctx context.Context, c *Client) (int, error) {
				resp, err := c.ListConnections(ctx, ListOptions{Page: 1, PageSize: 10, Search: "k8s cluster"})
				if err != nil {
					return 0, err
				}
				return resp.TotalCount, nil
			},
		},
		{
			name:       "ListPatterns",
			path:       "/api/pattern",
			sampleBody: `{"page":1,"pageSize":10,"totalCount":1,"patterns":[{"id":"pat-1","name":"istio-app"}]}`,
			call: func(ctx context.Context, c *Client) (int, error) {
				resp, err := c.ListPatterns(ctx, ListOptions{Page: 1, PageSize: 10, Search: "istio"})
				if err != nil {
					return 0, err
				}
				return resp.TotalCount, nil
			},
		},
		{
			name:       "ListWorkspaces",
			path:       "/api/workspaces",
			sampleBody: `{"page":1,"pageSize":10,"totalCount":1,"workspaces":[{"id":"ws-1","name":"staging-ws"}]}`,
			call: func(ctx context.Context, c *Client) (int, error) {
				resp, err := c.ListWorkspaces(ctx, ListOptions{Page: 1, PageSize: 10})
				if err != nil {
					return 0, err
				}
				return resp.TotalCount, nil
			},
		},
		{
			name:       "ListEnvironments",
			path:       "/api/environments",
			sampleBody: `{"page":1,"pageSize":10,"totalCount":1,"environments":[{"id":"env-1","name":"prod-env"}]}`,
			call: func(ctx context.Context, c *Client) (int, error) {
				resp, err := c.ListEnvironments(ctx, ListOptions{Page: 1, PageSize: 10})
				if err != nil {
					return 0, err
				}
				return resp.TotalCount, nil
			},
		},
		{
			name:       "ListModels",
			path:       "/api/registry/models",
			sampleBody: `{"page":1,"pageSize":10,"totalCount":1,"models":[{"id":"mod-1","name":"kubernetes"}]}`,
			call: func(ctx context.Context, c *Client) (int, error) {
				resp, err := c.ListModels(ctx, ListOptions{Page: 1, PageSize: 10})
				if err != nil {
					return 0, err
				}
				return resp.TotalCount, nil
			},
		},
		{
			name:       "ListPerformanceProfiles",
			path:       "/api/user/performance/profiles",
			sampleBody: `{"page":1,"pageSize":10,"totalCount":1,"profiles":[{"id":"prof-1","name":"soak-test"}]}`,
			call: func(ctx context.Context, c *Client) (int, error) {
				resp, err := c.ListPerformanceProfiles(ctx, ListOptions{Page: 1, PageSize: 10})
				if err != nil {
					return 0, err
				}
				return resp.TotalCount, nil
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var receivedQuery, receivedPath string
			client, ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				receivedQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.sampleBody))
			})
			defer ts.Close()

			count, err := tc.call(context.Background(), client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if receivedPath != tc.path {
				t.Errorf("expected path %q, got %q", tc.path, receivedPath)
			}
			if !strings.Contains(receivedQuery, "page=1") || !strings.Contains(receivedQuery, "pageSize=10") {
				t.Errorf("expected query parameters page=1 & pageSize=10 in %q", receivedQuery)
			}
			if count != 1 {
				t.Errorf("expected total count 1, got %d", count)
			}
		})
	}
}
