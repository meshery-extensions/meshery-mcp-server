package mcpserver

import (
	"context"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
	go func() { done <- serveHTTP(ctx, server, ln, nil) }()
	return "http://" + ln.Addr().String() + HTTPPath, cancel, done
}

func connectHTTP(t *testing.T, endpoint, protocolVersion string) *mcp.ClientSession {
	t.Helper()
	c := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	cs, err := c.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
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
			endpoint, _, _ := startHTTP(t, Options{Name: "test", Registry: fake, Credentials: staticCredentials(testCreds)})
			cs := connectHTTP(t, endpoint, version)

			tools, err := cs.ListTools(context.Background(), nil)
			if err != nil {
				t.Fatalf("ListTools: %v", err)
			}
			if !slices.ContainsFunc(tools.Tools, func(tool *mcp.Tool) bool { return tool.Name == "get_component" }) {
				t.Fatalf("get_component is not served over HTTP")
			}

			res, err := callGetComponent(cs)
			if err != nil || res.IsError {
				t.Fatalf("CallTool: err=%v result=%v", err, res)
			}
			if !fake.gotOK || fake.gotCreds != testCreds {
				t.Errorf("backend saw credentials (%v, %v), want (%v, true)", fake.gotCreds, fake.gotOK, testCreds)
			}
		})
	}
}

func TestHTTP_RejectsBrowserRequests(t *testing.T) {
	endpoint, _, _ := startHTTP(t, Options{Name: "test", Registry: &fakeRegistry{}})

	tests := []struct {
		name       string
		header     http.Header
		host       string
		wantStatus int
	}{
		{name: "cross-origin browser request", header: http.Header{"Origin": {"http://evil.example"}},
			wantStatus: http.StatusForbidden},
		{name: "DNS rebinding (foreign Host header)", host: "evil.example",
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
		})
	}
}

func postJSON(t *testing.T, endpoint, body string, extra http.Header) (int, http.Header) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
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

func TestHTTP_CleanShutdown(t *testing.T) {
	endpoint, cancel, done := startHTTP(t, Options{Name: "test", Registry: &fakeRegistry{}})
	connectHTTP(t, endpoint, protocolVersions[0])

	cancel()
	if err := wait(t, done); err != nil {
		t.Errorf("serveHTTP returned %v, want nil", err)
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
		HTTPOptions{Addr: ln.Addr().String()})
	if err == nil {
		t.Fatal("want error when the port is in use, got nil")
	}
}
