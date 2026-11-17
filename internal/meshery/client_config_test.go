package meshery

import (
	"context"
	neterrors "errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewValidationAndDefaults(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"defaults", Config{BaseURL: "http://localhost:9081"}, false},
		{"https", Config{BaseURL: "https://meshery.example", AuthMode: AuthBearer}, false},
		{"bad scheme", Config{BaseURL: "ftp://localhost:9081"}, true},
		{"no host", Config{BaseURL: "http:///path"}, true},
		{
			"embedded credentials",
			Config{BaseURL: "http://user:pass@localhost:9081"}, //nolint:gosec // G101: deliberately invalid URL fixture
			true,
		},
		{"bad auth mode", Config{BaseURL: "http://localhost:9081", AuthMode: "magic"}, true},
		{"negative timeout", Config{BaseURL: "http://x", Timeout: -time.Second}, true},
		{"negative retries", Config{BaseURL: "http://x", MaxRetries: -1}, true},
		{"negative concurrency", Config{BaseURL: "http://x", MaxConcurrentCalls: -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			wantMode := AuthCookie
			if tt.cfg.AuthMode != "" {
				wantMode = tt.cfg.AuthMode
			}
			if c.authMode != wantMode || c.http.Timeout != defaultTimeout || c.maxRetries != 0 || c.semaphore.limit != defaultMaxConcurrentCalls {
				t.Fatalf("unexpected defaults: %+v", c)
			}
		})
	}
}

func TestCallRetriesAreBounded(t *testing.T) {
	srv, mock := newAPIMock(t, 200, `{}`)
	defer srv.Close()
	mock.steps = []scenario{{status: 429}, {status: 429}, {status: 200, body: `{}`}}
	c := NewClient(srv.URL, "token", WithMaxRetries(2),
		WithBackoff(Backoff{Base: time.Millisecond, Cap: time.Millisecond}))

	if _, err := c.Call(context.Background(), http.MethodGet, "/api/claims", nil, "tool"); err != nil {
		t.Fatal(err)
	}
	if got := len(mock.calls()); got != 3 {
		t.Fatalf("got %d requests, want 3", got)
	}
}

func TestCallErrorDoesNotLeakToken(t *testing.T) {
	srv, _ := newAPIMock(t, 500, `{"error":"token secret-token was rejected"}`)
	defer srv.Close()
	c := newTestClient(t, srv, "secret-token")

	_, err := c.Call(context.Background(), http.MethodGet, "/api/claims?token=secret-token", nil, "tool")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("token leaked: %v", err)
	}
	var apiErr *APIError
	if !neterrors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if strings.Contains(apiErr.Path, "?") {
		t.Fatalf("path contains query string: %q", apiErr.Path)
	}
}

func TestBearerModeStillSendsAuthorization(t *testing.T) {
	srv, mock := newAPIMock(t, 200, `{}`)
	defer srv.Close()
	c := NewClient(srv.URL, "cloud-token", WithAuthMode(AuthBearer))

	if _, err := c.Call(context.Background(), http.MethodGet, "/api/claims", nil, "tool"); err != nil {
		t.Fatal(err)
	}
	req := mock.calls()[0]
	if req.auth != "Bearer cloud-token" || req.cookie != "" {
		t.Fatalf("unexpected auth: %+v", req)
	}
}

func TestCrossHostRedirectDropsCredentials(t *testing.T) {
	var destination *apiMock
	destSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destination.ServeHTTP(w, r)
	}))
	defer destSrv.Close()
	destination = &apiMock{status: 200, body: `{}`}

	sourceSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destSrv.URL+"/api/claims", http.StatusFound)
	}))
	defer sourceSrv.Close()

	c := NewClient(sourceSrv.URL, "secret-token", WithAuthMode(AuthCookie), WithProvider("cloud"))
	if _, err := c.Call(context.Background(), http.MethodGet, "/api/claims", nil, "tool"); err != nil {
		t.Fatal(err)
	}
	reqs := destination.calls()
	if len(reqs) != 1 {
		t.Fatalf("destination saw %d requests, want 1", len(reqs))
	}
	if reqs[0].cookie != "" || reqs[0].auth != "" {
		t.Fatalf("credentials followed redirect: %+v", reqs[0])
	}
}
