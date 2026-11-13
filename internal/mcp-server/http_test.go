package mcpserver

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testAccessKey = "0123456789abcdef0123456789abcdef"

// bearer adds an Authorization header to every request.
type bearer struct {
	value string
	base  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", b.value)
	return b.base.RoundTrip(r)
}

// startHTTP serves opts on a random loopback port and returns the endpoint
// URL, the cancel func for the server's context, and a channel that receives
// serveHTTP's return value.
func startHTTP(t *testing.T, opts Options) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	server, err := newServer(opts)
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- serveHTTP(ctx, server, ln, testAccessKey, nil) }()
	return "http://" + ln.Addr().String() + HTTPPath, cancel, done
}

func connectHTTP(t *testing.T, endpoint, protocolVersion string) *mcp.ClientSession {
	t.Helper()
	c := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	cs, err := c.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
		HTTPClient:           &http.Client{Transport: bearer{value: "Bearer " + testAccessKey, base: http.DefaultTransport}},
		DisableStandaloneSSE: true,
	}, &mcp.ClientSessionOptions{ProtocolVersion: protocolVersion})
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestHTTP_ServesToolsWithCredentials(t *testing.T) {
	for _, version := range protocolVersions {
		t.Run(version, func(t *testing.T) {
			fake := &fakeRegistry{}
			endpoint, _, _ := startHTTP(t, Options{Name: "test", Registry: fake, Credentials: staticToken("abc")})
			cs := connectHTTP(t, endpoint, version)

			tools, err := cs.ListTools(context.Background(), nil)
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}
			if len(tools.Tools) != 5 {
				t.Errorf("got %d tools, want 5", len(tools.Tools))
			}

			res, err := callGetComponent(cs)
			if err != nil || res.IsError {
				t.Fatalf("CallTool: err=%v result=%v", err, res)
			}
			if !fake.gotOK || fake.gotToken != "abc" {
				t.Errorf("registry saw token (%q, %v), want (%q, true)", fake.gotToken, fake.gotOK, "abc")
			}
		})
	}
}

func TestHTTP_RejectsUntrustedRequests(t *testing.T) {
	endpoint, _, _ := startHTTP(t, Options{Name: "test", Registry: &fakeRegistry{}})
	validAuth := "Bearer " + testAccessKey

	tests := []struct {
		name       string
		header     http.Header
		host       string
		wantStatus int
	}{
		{name: "no access key",
			wantStatus: http.StatusUnauthorized},
		{name: "wrong access key", header: http.Header{"Authorization": {"Bearer wrong"}},
			wantStatus: http.StatusUnauthorized},
		{name: "access key without Bearer scheme", header: http.Header{"Authorization": {testAccessKey}},
			wantStatus: http.StatusUnauthorized},
		{name: "cross-origin browser request", header: http.Header{"Authorization": {validAuth}, "Origin": {"http://evil.example"}},
			wantStatus: http.StatusForbidden},
		{name: "DNS rebinding (foreign Host header)", header: http.Header{"Authorization": {validAuth}}, host: "evil.example",
			wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, body)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			for k, v := range tt.header {
				req.Header[k] = v
			}
			if tt.host != "" {
				req.Host = tt.host
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUnauthorized && !strings.HasPrefix(resp.Header.Get("WWW-Authenticate"), "Bearer") {
				t.Errorf("WWW-Authenticate = %q, want a Bearer challenge", resp.Header.Get("WWW-Authenticate"))
			}
		})
	}
}

// postJSON sends body to endpoint with the access key and the headers a
// Streamable HTTP client sends, plus extra. It closes the response and
// returns its status code and headers.
func postJSON(t *testing.T, endpoint, body string, extra http.Header) (int, http.Header) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+testAccessKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range extra {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode, resp.Header
}

func TestHTTP_Stateless(t *testing.T) {
	endpoint, _, _ := startHTTP(t, Options{Name: "test", Registry: &fakeRegistry{}})

	t.Run("GET is not allowed", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+testAccessKey)
		req.Header.Set("Accept", "text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
		}
	})

	t.Run("2025-11-25 initialize gets no session", func(t *testing.T) {
		status, header := postJSON(t, endpoint,
			`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`, nil)
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d", status, http.StatusOK)
		}
		if id := header.Get("Mcp-Session-Id"); id != "" {
			t.Errorf("Mcp-Session-Id = %q, want none", id)
		}
	})

	t.Run("2026-07-28 header mismatch is rejected", func(t *testing.T) {
		status, _ := postJSON(t, endpoint,
			`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`,
			http.Header{"Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/call"}})
		if status != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", status, http.StatusBadRequest)
		}
	})

	t.Run("responses are not buffered by proxies", func(t *testing.T) {
		status, header := postJSON(t, endpoint,
			`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`,
			http.Header{"Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/list"}})
		if status != http.StatusOK {
			t.Fatalf("status = %d, want %d", status, http.StatusOK)
		}
		if got := header.Get("X-Accel-Buffering"); got != "no" {
			t.Errorf("X-Accel-Buffering = %q, want %q", got, "no")
		}
	})
}

func TestRequireAccessKey_SchemeIsCaseInsensitive(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := requireAccessKey(testAccessKey, ok)

	for _, auth := range []string{"Bearer " + testAccessKey, "bearer " + testAccessKey, "BEARER " + testAccessKey} {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, HTTPPath, nil)
		req.Header.Set("Authorization", auth)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("%q: status = %d, want %d", auth, rec.Code, http.StatusNoContent)
		}
	}
}

func TestHTTP_CleanShutdown(t *testing.T) {
	endpoint, cancel, done := startHTTP(t, Options{Name: "test", Registry: &fakeRegistry{}})
	connectHTTP(t, endpoint, protocolVersions[0])

	cancel()
	if err := wait(t, done); err != nil {
		t.Errorf("serveHTTP returned %v, want nil", err)
	}
}

func TestRunHTTP_RequiresAccessKey(t *testing.T) {
	err := RunHTTP(context.Background(), Options{Name: "test", Registry: &fakeRegistry{}}, HTTPOptions{Addr: "127.0.0.1:0"})
	if err == nil {
		t.Fatal("want error without an access key, got nil")
	}
}

func TestRunHTTP_PortInUse(t *testing.T) {
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	err = RunHTTP(context.Background(), Options{Name: "test", Registry: &fakeRegistry{}},
		HTTPOptions{Addr: ln.Addr().String(), AccessKey: testAccessKey})
	if err == nil {
		t.Fatal("want error when the port is in use, got nil")
	}
}
