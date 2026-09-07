package mcpera_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meshery-extensions/meshery-mcp-server/internal/mcpera"
)

// The fake servers run as this test binary re-invoked with an era argument,
// so the suite stays hermetic and needs no external SDK.
func helper(era string) (string, []string) {
	return os.Args[0], []string{"-test.run=TestFakeServer", "--", era}
}

func probe(t *testing.T, era string) *mcpera.Report {
	t.Helper()
	return probeWithin(t, era, 4*time.Second)
}

func probeWithin(t *testing.T, era string, timeout time.Duration) *mcpera.Report {
	t.Helper()
	name, args := helper(era)
	rep, err := mcpera.Probe(context.Background(), timeout, name, args...)
	if err != nil {
		t.Fatalf("probing %s: %v", era, err)
	}
	return rep
}

// TestLegacyServerIsDetected covers a server that answers initialize, refuses
// server/discover, and runs a modern request anyway while answering in the old
// shape. That is the era-ambiguous hazard the specification names.
func TestLegacyServerIsDetected(t *testing.T) {
	rep := probe(t, "legacy")

	if rep.Era != mcpera.Legacy {
		t.Errorf("era = %s, want %s", rep.Era, mcpera.Legacy)
	}
	if !rep.AnswersInitialize {
		t.Error("expected the legacy handshake to succeed")
	}
	if rep.AnswersDiscover {
		t.Error("a legacy server should not answer server/discover")
	}
	if !rep.SilentDowngrade {
		t.Fatal("a legacy server that runs a modern request must be reported as a silent downgrade")
	}
	if !strings.Contains(strings.Join(rep.Notes, " "), "cannot tell it was downgraded") {
		t.Errorf("the notes should say what the client cannot see: %v", rep.Notes)
	}
}

// TestDualEraServerIsDetected covers the posture that works with every client.
func TestDualEraServerIsDetected(t *testing.T) {
	rep := probe(t, "dual")

	if rep.Era != mcpera.Dual {
		t.Errorf("era = %s, want %s", rep.Era, mcpera.Dual)
	}
	if rep.SilentDowngrade {
		t.Error("a modern-shaped result is not a downgrade")
	}
}

// TestModernOnlyServerIsDetected covers a server that refuses initialize. A
// legacy client has no fall-forward mechanism, so this is a real posture rather
// than a broken one.
func TestModernOnlyServerIsDetected(t *testing.T) {
	rep := probe(t, "modern")

	if rep.Era != mcpera.Modern {
		t.Errorf("era = %s, want %s", rep.Era, mcpera.Modern)
	}
	if rep.AnswersInitialize {
		t.Error("a modern-only server should refuse initialize")
	}
}

// TestSilenceIsAFindingNotAnError covers a server that never answers. The spec
// lists staying silent as one of the three ways a modern-to-legacy exchange
// fails, so the probe has to survive it.
func TestSilenceIsAFindingNotAnError(t *testing.T) {
	// A short timeout here: the point is that silence is survived, and three
	// probes at the default would dominate the suite's runtime.
	rep := probeWithin(t, "silent", 300*time.Millisecond)

	if rep.Era != mcpera.Unknown {
		t.Errorf("era = %s, want %s", rep.Era, mcpera.Unknown)
	}
	if rep.SilentDowngrade {
		t.Error("silence is not a downgrade")
	}
}

// TestModernResultMarkersAreWhatDistinguish pins the detection itself. A result
// carrying resultType, or serverInfo in _meta, is modern; the bare legacy body
// is not, and that difference is the only thing on the wire that separates a
// downgraded exchange from a real one.
func TestModernResultMarkersAreWhatDistinguish(t *testing.T) {
	withMarkers := probe(t, "dual")
	if !withMarkers.ModernResultIsModern {
		t.Error("a result carrying resultType should read as modern")
	}
	without := probe(t, "legacy")
	if without.ModernResultIsModern {
		t.Error("a bare legacy body should not read as modern")
	}
	if !without.ServesModernCall {
		t.Error("the legacy fake did serve the request, which is the point")
	}
}

// TestFakeServer is the fake server. It is a normal test when run without the
// era argument and a JSON-RPC server over stdio when given one.
func TestFakeServer(t *testing.T) {
	era := ""
	for i, a := range os.Args {
		if a == "--" && i+1 < len(os.Args) {
			era = os.Args[i+1]
		}
	}
	if era == "" {
		t.Skip("not running as a fake server")
	}
	serve(era)
	os.Exit(0)
}

func serve(era string) {
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(sc.Bytes(), &req) != nil {
			continue
		}
		if era == "silent" {
			continue
		}
		resp := respond(era, req.Method, req.ID)
		if resp == nil {
			continue
		}
		b, _ := json.Marshal(resp)
		out.Write(append(b, '\n'))
		out.Flush()
	}
}

func respond(era, method string, id any) map[string]any {
	ok := func(result any) map[string]any {
		return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
	}
	fail := func(code int, msg string) map[string]any {
		return map[string]any{"jsonrpc": "2.0", "id": id,
			"error": map[string]any{"code": code, "message": msg}}
	}
	legacyResult := map[string]any{"tools": []any{}}
	modernResult := map[string]any{
		"tools": []any{}, "resultType": "complete",
		"_meta": map[string]any{mcpera.MetaServerInfo: map[string]any{"name": "fake"}},
	}

	switch method {
	case "initialize":
		if era == "modern" {
			return fail(-32601, "method not found: initialize")
		}
		return ok(map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{},
			"serverInfo":      map[string]any{"name": "fake", "version": "0"},
		})
	case "server/discover":
		if era == "legacy" {
			return fail(-32601, "Method server/discover not found")
		}
		return ok(modernResult)
	case "tools/list":
		if era == "legacy" {
			// The hazard: it runs the modern request and answers in the old shape.
			return ok(legacyResult)
		}
		return ok(modernResult)
	}
	return nil
}
