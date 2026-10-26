package mcpera_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meshery-extensions/meshery-mcp-server/internal/mcpera"
)

// fakeHTTP serves tools/call. When validate is true it enforces the header
// against the body the way revision 2026-07-28 requires; when false it ignores
// the header and runs whatever the body asked for, which is what a server
// predating the rule does.
func fakeHTTP(t *testing.T, validate, modernShape bool) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		var req struct {
			ID     any `json:"id"`
			Params struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		_ = json.Unmarshal(raw, &req)

		w.Header().Set("Content-Type", "application/json")
		if validate && r.Header.Get("Mcp-Name") != req.Params.Name {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": req.ID,
				"error": map[string]any{
					"code": mcpera.HeaderMismatchCode,
					"message": "header mismatch: Mcp-Name header value '" +
						r.Header.Get("Mcp-Name") + "' does not match body value '" + req.Params.Name + "'",
				},
			})
			return
		}
		result := map[string]any{"content": []any{map[string]any{"type": "text", "text": req.Params.Name}}}
		if modernShape {
			result["resultType"] = "complete"
			result["_meta"] = map[string]any{mcpera.MetaServerInfo: map[string]any{"name": "fake"}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// TestHTTPConformantServerPasses covers a server that enforces the rule.
func TestHTTPConformantServerPasses(t *testing.T) {
	rep, err := mcpera.ProbeHTTP(context.Background(), 5*time.Second,
		fakeHTTP(t, true, true), "ping", "danger", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.ValidatesHeaderBody {
		t.Error("expected the mismatch to be caught")
	}
	if rep.MismatchCode != mcpera.HeaderMismatchCode {
		t.Errorf("code = %d, want %d", rep.MismatchCode, mcpera.HeaderMismatchCode)
	}
	if rep.ExecutedMismatched {
		t.Error("a rejected request did not execute")
	}
	if !rep.ServesModern || !rep.ModernResultIsModern {
		t.Error("expected a modern result on the agreeing call")
	}
}

// TestHTTPServerThatIgnoresTheHeader covers the hazard. The server runs the
// tool the body named while the header named another, so an intermediary
// routing on that header has a different idea of what happened.
func TestHTTPServerThatIgnoresTheHeader(t *testing.T) {
	rep, err := mcpera.ProbeHTTP(context.Background(), 5*time.Second,
		fakeHTTP(t, false, false), "ping", "danger", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ValidatesHeaderBody {
		t.Error("this server does not validate, and should not be reported as doing so")
	}
	if !rep.ExecutedMismatched {
		t.Fatal("running the body's tool under a different header name is the finding")
	}
	if rep.ModernResultIsModern {
		t.Error("a legacy-shaped result should not read as modern")
	}
	if !strings.Contains(strings.Join(rep.Notes, " "), "disagree about what just executed") {
		t.Errorf("the notes should name the consequence: %v", rep.Notes)
	}
}

// TestHTTPSSEFramingIsUnderstood checks a response delivered as a single SSE
// event is read the same as a plain body, since Streamable HTTP may use either.
func TestHTTPSSEFramingIsUnderstood(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		var req struct {
			Params struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		_ = json.Unmarshal(raw, &req)
		if r.Header.Get("Mcp-Name") != req.Params.Name {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-32020,\"message\":\"header mismatch\"}}\n\n"))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"resultType\":\"complete\",\"content\":[]}}\n\n"))
	}))
	defer srv.Close()

	rep, err := mcpera.ProbeHTTP(context.Background(), 5*time.Second, srv.URL, "ping", "danger", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.ValidatesHeaderBody {
		t.Error("an SSE-framed error should still be read as a rejection")
	}
	if !rep.ModernResultIsModern {
		t.Error("an SSE-framed result should still be read as modern")
	}
}

// TestHTTPRefusalIsReported covers a server that turns the modern request away
// entirely, which is what a stateful legacy endpoint does to a stateless call.
func TestHTTPRefusalIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Invalid session ID", http.StatusNotFound)
	}))
	defer srv.Close()

	rep, err := mcpera.ProbeHTTP(context.Background(), 5*time.Second, srv.URL, "ping", "danger", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ServesModern {
		t.Error("a 404 is not serving the modern request")
	}
	if !strings.Contains(rep.RefusedModern, "Invalid session ID") {
		t.Errorf("the refusal should carry the server's reason, got %q", rep.RefusedModern)
	}
	if rep.ExecutedMismatched {
		t.Error("nothing executed")
	}
}

// JSON-RPC carries its errors in the body, so a server may reject the mismatch
// and still answer 200. Reporting that as the dangerous tool having run is a
// false accusation, and the report is the whole product here.
func TestMismatchRejectedWithHTTP200IsNotAnExecution(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Mcp-Name") != "ping" {
			// A rejection that is not the header-mismatch code: still an
			// error, still a 200, still not an execution.
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"invalid params"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"resultType":"complete","content":[]}}`))
	}))
	defer srv.Close()

	rep, err := mcpera.ProbeHTTP(context.Background(), 5*time.Second, srv.URL, "ping", "danger", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ExecutedMismatched {
		t.Error("reported the tool as executed, but the server answered with a JSON-RPC error")
	}
	if rep.ValidatesHeaderBody {
		t.Error("it rejected with -32602, not the code the revision specifies, so this is not conformance")
	}
}

// A pretty-printed body is one JSON object across many lines. Reading only the
// first line starting with { yields a bare brace, which parses as nothing.
func TestPrettyPrintedBodyIsParsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Mcp-Name") != "ping" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("{\n  \"jsonrpc\": \"2.0\",\n  \"id\": 1,\n  \"error\": {\n    \"code\": -32020,\n    \"message\": \"header mismatch\"\n  }\n}"))
			return
		}
		_, _ = w.Write([]byte("{\n  \"jsonrpc\": \"2.0\",\n  \"id\": 1,\n  \"result\": {\n    \"resultType\": \"complete\"\n  }\n}"))
	}))
	defer srv.Close()

	rep, err := mcpera.ProbeHTTP(context.Background(), 5*time.Second, srv.URL, "ping", "danger", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.ModernResultIsModern {
		t.Error("a pretty-printed modern result was not recognised as modern")
	}
	if rep.MismatchCode != mcpera.HeaderMismatchCode {
		t.Errorf("mismatch code = %d, want %d from a pretty-printed error body", rep.MismatchCode, mcpera.HeaderMismatchCode)
	}
}

// When the endpoint refuses both calls for an unrelated reason, the report must
// not speak as though the server evaluated the header at all.
func TestRefusedBothCallsIsNotAMismatchVerdict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Invalid session ID", http.StatusNotFound)
	}))
	defer srv.Close()

	rep, err := mcpera.ProbeHTTP(context.Background(), 5*time.Second, srv.URL, "ping", "danger", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ValidatesHeaderBody || rep.ExecutedMismatched {
		t.Error("the endpoint answered nothing, so neither verdict is available")
	}
	joined := strings.Join(rep.Notes, " ")
	if !strings.Contains(joined, "did not serve") && !strings.Contains(joined, "refused") {
		t.Errorf("the notes should say the endpoint refused the request, got: %v", rep.Notes)
	}
}
