package meshery

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// scenario describes one canned response from the mock Cloud server.
type scenario struct {
	status     int
	body       string
	retryAfter string
}

// mockCloud answers a fixed queue of scenarios, one per request, so tests
// can count exactly how many requests Cloud saw (no tight retry loops).
type mockCloud struct {
	mu     sync.Mutex
	steps  []scenario
	idx    int
	counts map[string]int
	auths  map[string][]string
}

func (m *mockCloud) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	path := r.URL.Path
	m.counts[path]++
	m.auths[path] = append(m.auths[path], r.Header.Get(HeaderToken))

	step := m.steps[min(m.idx, len(m.steps)-1)]
	m.idx++

	if step.retryAfter != "" {
		w.Header().Set("Retry-After", step.retryAfter)
	}
	w.WriteHeader(step.status)
	_, _ = w.Write([]byte(step.body))
	_, _ = io.ReadAll(r.Body)
	_ = r.Body.Close()
}

func (m *mockCloud) count(path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counts[path]
}

func (m *mockCloud) lastAuth(path string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	auths := m.auths[path]
	if len(auths) == 0 {
		return ""
	}
	return auths[len(auths)-1]
}

func newMockCloud(t *testing.T, steps ...scenario) (*httptest.Server, *mockCloud) {
	t.Helper()
	m := &mockCloud{steps: steps, counts: map[string]int{}, auths: map[string][]string{}}
	return httptest.NewServer(m), m
}

func TestCallReturnsBody(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{status: 200, body: `{"ok": true}`})
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	out, err := c.Call(context.Background(), http.MethodPost, "/api/claims", "tool-a")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"ok": true`) {
		t.Errorf("unexpected body: %s", out)
	}
	if mock.count("/api/claims") != 1 {
		t.Errorf("Cloud called %d times, want 1", mock.count("/api/claims"))
	}
	if got := mock.lastAuth("/api/claims"); got != "test-token" {
		t.Errorf("authorization header = %q, want Bearer test-token", got)
	}
}

func TestCall401IsReauthNoRetry(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{status: 401, body: `{"error":"invalid_token"}`})
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	_, err := c.Call(context.Background(), http.MethodGet, "/api/claims", "tool-a")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Authentication required") {
		t.Errorf("expected reauth error, got: %v", err)
	}
	if mock.count("/api/claims") != 1 {
		t.Errorf("Cloud called %d times after 401, want 1 (no retry)", mock.count("/api/claims"))
	}
}

func TestCall403MissingScope(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{status: 403, body: `{"missing_scope":"org:admin"}`})
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	_, err := c.Call(context.Background(), http.MethodPost, "/api/ledgers", "tool-a")
	var insufficient *ErrInsufficientScope
	if !errors.As(err, &insufficient) {
		t.Fatalf("expected ErrInsufficientScope, got %v", err)
	}
	if insufficient.MissingScope != "org:admin" {
		t.Errorf("MissingScope = %q, want org:admin", insufficient.MissingScope)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected 403 in message: %v", err)
	}
	if mock.count("/api/ledgers") != 1 {
		t.Errorf("Cloud called %d times after 403, want 1 (no retry)", mock.count("/api/ledgers"))
	}
}

func TestCall403UnidentifiedScope(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{status: 403, body: `{"error":"forbidden"}`})
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	_, err := c.Call(context.Background(), http.MethodGet, "/api/ledgers", "tool-a")
	var insufficient *ErrInsufficientScope
	if !errors.As(err, &insufficient) {
		t.Fatalf("expected ErrInsufficientScope, got %v", err)
	}
	if insufficient.MissingScope != "" {
		t.Errorf("MissingScope = %q, want empty", insufficient.MissingScope)
	}
	if !strings.Contains(err.Error(), "did not identify") {
		t.Errorf("expected 'did not identify' wording: %v", err)
	}
	if mock.count("/api/ledgers") != 1 {
		t.Errorf("Cloud called %d times after 403, want 1 (no retry)", mock.count("/api/ledgers"))
	}
}

func TestCall429HonorsRetryAfter(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{status: 429, retryAfter: "1"},
		scenario{status: 200, body: `{"ok": true}`},
	)
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	start := time.Now()
	out, err := c.Call(context.Background(), http.MethodPost, "/api/events", "tool-a")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"ok": true`) {
		t.Errorf("unexpected body: %s", out)
	}
	if mock.count("/api/events") != 2 {
		t.Errorf("Cloud called %d times, want 2 (one retry)", mock.count("/api/events"))
	}
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed %v, expected Retry-After 1s to be honored", elapsed)
	}
}

func TestCall429ExponentialBackoff(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{status: 429},
		scenario{status: 429},
		scenario{status: 200, body: `{"ok": true}`},
	)
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	start := time.Now()
	_, err := c.Call(context.Background(), http.MethodGet, "/api/schedules", "tool-a")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if mock.count("/api/schedules") != 3 {
		t.Errorf("Cloud called %d times, want 3 (two retries)", mock.count("/api/schedules"))
	}
	if elapsed < 500*time.Millisecond {
		t.Errorf("elapsed %v, expected two jittered backoffs", elapsed)
	}
}

func TestCall429Exhausted(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{status: 429},
		scenario{status: 429},
		scenario{status: 429},
		scenario{status: 429},
	)
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	_, err := c.Call(context.Background(), http.MethodGet, "/api/content_items", "tool-a")
	var limited *ErrRateLimited
	if !errors.As(err, &limited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
	if !strings.Contains(err.Error(), "retry after") {
		t.Errorf("expected 'retry after N seconds' wording: %v", err)
	}
	if mock.count("/api/content_items") != 4 {
		t.Errorf("Cloud called %d times, want 4 (maxRetries 3)", mock.count("/api/content_items"))
	}
}

func TestCall429RespectsContext(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{status: 429, retryAfter: "10"})
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := c.Call(ctx, http.MethodPost, "/api/events", "tool-a")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != context.DeadlineExceeded && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected context error, got %v", err)
	}
	if mock.count("/api/events") != 1 {
		t.Errorf("Cloud called %d times, want 1 (bounded retries)", mock.count("/api/events"))
	}
}

func TestCall503RetriesThenSuccess(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{status: 503},
		scenario{status: 200, body: `{"ok": true}`},
	)
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	out, err := c.Call(context.Background(), http.MethodGet, "/api/ledgers", "tool-a")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"ok": true`) {
		t.Errorf("unexpected body: %s", out)
	}
	if mock.count("/api/ledgers") != 2 {
		t.Errorf("Cloud called %d times, want 2 (one retry)", mock.count("/api/ledgers"))
	}
}

func TestCall503ExhaustedNoCredentialChange(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{status: 503},
		scenario{status: 503},
		scenario{status: 503},
		scenario{status: 503},
	)
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	_, err := c.Call(context.Background(), http.MethodGet, "/api/claims", "tool-a")
	if !strings.Contains(err.Error(), "service unavailable") {
		t.Errorf("expected service unavailable wording: %v", err)
	}
	if mock.count("/api/claims") != 4 {
		t.Errorf("Cloud called %d times, want 4 (maxRetries 3)", mock.count("/api/claims"))
	}
	// 503 must never clear stored credentials.
	token, ok := c.Token()
	if !ok || token != "test-token" {
		t.Errorf("token was cleared on 503: (%q, %v)", token, ok)
	}
}

func TestRevokeOK200EvictsToken(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{status: 200, body: ``})
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	if err := c.Revoke(context.Background()); err != nil {
		t.Fatal(err)
	}
	if mock.count("/revoke") != 1 {
		t.Errorf("POST /revoke called %d times, want 1", mock.count("/revoke"))
	}
	if token, ok := c.Token(); ok {
		t.Errorf("token not evicted after revoke: %q", token)
	}
	if _, err := c.Call(context.Background(), http.MethodGet, "/api/claims", "tool-a"); !strings.Contains(err.Error(), "No API token configured") {
		t.Errorf("expected no-token error after revoke, got %v", err)
	}
}

func TestRevoke503RetriesEvictsAnyway(t *testing.T) {
	srv, mock := newMockCloud(t,
		scenario{status: 503},
		scenario{status: 503},
		scenario{status: 503},
		scenario{status: 503},
	)
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	err := c.Revoke(context.Background())
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("expected revoke 503 error after retries, got %v", err)
	}
	if mock.count("/revoke") != 4 {
		t.Errorf("POST /revoke called %d times, want 4 (maxRetries 3)", mock.count("/revoke"))
	}
	if token, ok := c.Token(); ok {
		t.Errorf("token not evicted after revoke failure: %q", token)
	}
}

func TestRevoke401IsReauthNoRetry(t *testing.T) {
	srv, mock := newMockCloud(t, scenario{status: 401})
	defer srv.Close()

	c := NewClient(srv.URL, "test-token")
	err := c.Revoke(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Authentication required") {
		t.Errorf("expected revoke reauth error, got %v", err)
	}
	if mock.count("/revoke") != 1 {
		t.Errorf("POST /revoke called %d times after 401, want 1 (no retry)", mock.count("/revoke"))
	}
	if token, ok := c.Token(); ok {
		t.Errorf("token not evicted after revoke 401: %q", token)
	}
}
