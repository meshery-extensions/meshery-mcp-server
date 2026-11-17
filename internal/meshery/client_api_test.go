package meshery

import (
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordedRequest struct {
	method string
	path   string
	query  string
	cookie string
	auth   string
}

type apiMock struct {
	mu       sync.Mutex
	requests []recordedRequest
	status   int
	body     string
	steps    []scenario
	delay    time.Duration
}

func (m *apiMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, _ = io.ReadAll(r.Body)
	m.mu.Lock()
	m.requests = append(m.requests, recordedRequest{
		method: r.Method,
		path:   r.URL.Path,
		query:  r.URL.RawQuery,
		cookie: r.Header.Get("Cookie"),
		auth:   r.Header.Get("Authorization"),
	})
	steps := m.steps
	n := len(m.requests)
	status, response := m.status, m.body
	if len(steps) > 0 {
		step := steps[min(n-1, len(steps)-1)]
		status, response = step.status, step.body
	}
	m.mu.Unlock()
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-r.Context().Done():
			return
		}
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(response))
}

func (m *apiMock) calls() []recordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]recordedRequest(nil), m.requests...)
}

func newAPIMock(t *testing.T, status int, body string) (*httptest.Server, *apiMock) {
	t.Helper()
	m := &apiMock{status: status, body: body}
	return httptest.NewServer(m), m
}

func newTestClient(t *testing.T, srv *httptest.Server, token string, opts ...Option) *Client {
	t.Helper()
	base := []Option{WithAuthMode(AuthCookie), WithTimeout(2 * time.Second)}
	base = append(base, opts...)
	return NewClient(srv.URL, token, base...)
}

func TestPingReturnsVersionInfo(t *testing.T) {
	srv, mock := newAPIMock(t, 200, `{"build":"v0.8.0","commitsha":"abc123","latest":"v0.9.0","outdated":true,"releaseChannel":"stable","version":"cloud"}`)
	defer srv.Close()

	c := newTestClient(t, srv, "secret-token")
	info, err := c.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Build != "v0.8.0" || info.Commitsha != "abc123" || !info.Outdated || info.ReleaseChannel != "stable" {
		t.Fatalf("unexpected version info: %+v", info)
	}
	reqs := mock.calls()
	if len(reqs) != 1 || reqs[0].path != "/api/system/version" {
		t.Fatalf("unexpected requests: %+v", reqs)
	}
	// /api/system/version is unauthenticated: no credentials may be sent.
	if reqs[0].cookie != "" || reqs[0].auth != "" {
		t.Fatalf("Ping leaked credentials: %+v", reqs[0])
	}
}

func TestPingClosedPortIsDescriptiveAndBounded(t *testing.T) {
	srv, _ := newAPIMock(t, 200, `{}`)
	url := srv.URL
	srv.Close()

	c := NewClient(url, "secret-token", WithTimeout(500*time.Millisecond), WithAuthMode(AuthCookie))
	start := time.Now()
	_, err := c.Ping(context.Background())
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error")
	}
	var ce *ConnectError
	if !stderrors.As(err, &ce) {
		t.Fatalf("expected ConnectError, got %T: %v", err, ce)
	}
	if ce.Kind != ConnectRefused && ce.Kind != ConnectOther {
		t.Fatalf("expected refused/other, got %s", ce.Kind)
	}
	if !strings.Contains(err.Error(), url) {
		t.Fatalf("error must name URL: %v", err)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error leaked token: %v", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Ping took %v, expected prompt failure", elapsed)
	}
}

func TestPingRespectsTimeoutOnSlowHeaders(t *testing.T) {
	m := &apiMock{status: 200, body: `{}`, delay: 3 * time.Second}
	srv := httptest.NewServer(m)
	defer srv.Close()

	c := NewClient(srv.URL, "secret-token", WithTimeout(250*time.Millisecond))
	start := time.Now()
	_, err := c.Ping(context.Background())
	elapsed := time.Since(start)
	var ce *ConnectError
	if !stderrors.As(err, &ce) || ce.Kind != ConnectTimeout {
		t.Fatalf("expected timeout ConnectError, got %v", err)
	}
	if elapsed < 200*time.Millisecond || elapsed > 1500*time.Millisecond {
		t.Fatalf("elapsed %v, expected timeout to bound the call", elapsed)
	}
}

func TestPingRespectsTimeoutOnSlowBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		select {
		case <-time.After(3 * time.Second):
			_, _ = w.Write([]byte(`{"build":"late"}`))
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", WithTimeout(250*time.Millisecond))
	start := time.Now()
	_, err := c.Ping(context.Background())
	if err == nil {
		t.Fatal("expected timeout reading body")
	}
	if time.Since(start) > 1500*time.Millisecond {
		t.Fatalf("slow body did not respect timeout: %v", time.Since(start))
	}
}

func TestPingContextCancellation(t *testing.T) {
	m := &apiMock{status: 200, body: `{}`, delay: 3 * time.Second}
	srv := httptest.NewServer(m)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient(srv.URL, "token", WithTimeout(2*time.Second))
	if _, err := c.Ping(ctx); !stderrors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPingErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		code int
		body string
		want error
	}{
		{"not found", 404, `{"error":"missing"}`, ErrNotFound},
		{"conflict", 409, `{"error":"busy"}`, ErrConflict},
		{"validation", 422, `{"error":"bad"}`, ErrValidation},
		{"server", 500, `{"error":"boom"}`, ErrServer},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := newAPIMock(t, tt.code, tt.body)
			defer srv.Close()
			c := newTestClient(t, srv, "token")
			_, err := c.Ping(context.Background())
			if !stderrors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			var apiErr *APIError
			if !stderrors.As(err, &apiErr) || apiErr.Status != tt.code {
				t.Fatalf("expected APIError with status %d, got %v", tt.code, err)
			}
		})
	}
}
