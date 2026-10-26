// Package mcpera probes an MCP server over stdio and reports which protocol
// era it actually serves.
//
// Protocol revision 2026-07-28 is the first breaking change to MCP. It removes
// the initialize handshake, makes every request carry its own version in _meta,
// and adds server/discover. That splits implementations into a legacy era
// (2025-11-25 and earlier) and a modern one, and the specification's own
// compatibility table says a modern client against a legacy server "may reject
// the request with an implementation-defined error, stay silent, or even
// process an era-ambiguous method under legacy semantics".
//
// That last outcome is the one worth detecting. A legacy server can answer a
// modern request by running it, returning a legacy-shaped result with no error
// and no version acknowledgement. The client believes it negotiated
// 2026-07-28; the server never negotiated anything. Nothing in the exchange
// says so unless the client checks for the modern result markers, and a client
// that reads only content will not.
package mcpera

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Version is the protocol revision the modern probes declare.
const Version = "2026-07-28"

// The well-known _meta keys revision 2026-07-28 defines. protocolVersion and
// clientCapabilities are required on every modern request.
const (
	MetaProtocolVersion    = "io.modelcontextprotocol/protocolVersion"
	MetaClientInfo         = "io.modelcontextprotocol/clientInfo"
	MetaClientCapabilities = "io.modelcontextprotocol/clientCapabilities"
	MetaServerInfo         = "io.modelcontextprotocol/serverInfo"
)

// Era is the protocol era a server serves.
type Era string

const (
	// Legacy answers initialize and does not serve modern requests correctly.
	Legacy Era = "legacy"
	// Modern serves modern requests and does not answer initialize.
	Modern Era = "modern"
	// Dual answers both, which is the only posture that works with every client.
	Dual Era = "dual-era"
	// Unknown covers a server that answered neither opening.
	Unknown Era = "unknown"
)

// Report is what a probe found.
type Report struct {
	Era Era

	// AnswersInitialize is true when the legacy handshake succeeds.
	AnswersInitialize bool
	// NegotiatedVersion is the revision the legacy handshake settled on.
	NegotiatedVersion string
	// AnswersDiscover is true when server/discover succeeds. The spec says a
	// stdio client SHOULD send it first so an era mismatch fails deterministically.
	AnswersDiscover bool
	// DiscoverError is what server/discover returned when it did not succeed.
	DiscoverError string
	// DiscoverAnswered is true when the server replied to server/discover at
	// all, even to refuse it. A refusal is a fact about the method; silence is
	// only a fact about the process, and the two cannot be reported alike.
	DiscoverAnswered bool
	// ServesModernCall is true when a modern tools/call ran without an error.
	ServesModernCall bool
	// ModernResultIsModern is true when that result carried the markers a
	// modern result must have, rather than a legacy-shaped body.
	ModernResultIsModern bool

	// SilentDowngrade is the hazard: the server ran a modern request and
	// answered in the legacy shape, so the exchange looks successful to a
	// client that does not inspect the result markers.
	SilentDowngrade bool

	Notes []string
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	ID     any             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func modernMeta() map[string]any {
	return map[string]any{
		MetaProtocolVersion:    Version,
		MetaClientInfo:         map[string]any{"name": "mcpera", "version": "0"},
		MetaClientCapabilities: map[string]any{},
	}
}

// Probe starts the server, runs each opening against a fresh process, and
// reports what it found. Each exchange gets its own process because the two
// eras are mutually exclusive openings and a server may hold session state.
func Probe(ctx context.Context, timeout time.Duration, name string, args ...string) (*Report, error) {
	rep := &Report{Era: Unknown}

	init, _, err := exchange(ctx, timeout, name, args, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "mcpera", "version": "0"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("probing initialize: %w", err)
	}
	if init != nil && init.Error == nil && init.Result != nil {
		rep.AnswersInitialize = true
		var r struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if json.Unmarshal(init.Result, &r) == nil {
			rep.NegotiatedVersion = r.ProtocolVersion
		}
	}

	disc, discSilence, err := exchange(ctx, timeout, name, args, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "server/discover",
		"params": map[string]any{"_meta": modernMeta()},
	})
	if err != nil {
		return nil, fmt.Errorf("probing server/discover: %w", err)
	}
	switch {
	case disc == nil:
		// Not "it refused". The server said nothing, and why it said nothing is
		// the difference between a finding about the method and one about the
		// server being up at all.
		rep.DiscoverError = string(discSilence)
		rep.DiscoverAnswered = false
	case disc.Error != nil:
		rep.DiscoverError = fmt.Sprintf("%d: %s", disc.Error.Code, disc.Error.Message)
		rep.DiscoverAnswered = true
	case !describesAServer(disc.Result):
		// A reply with no error but nothing in it is not an answer to
		// server/discover. Counting it as one is how a server that merely
		// tolerates the method gets published as dual-era, which is the row a
		// client would trust most. The modern tools/list check below already
		// requires a result; this is the same bar.
		rep.DiscoverError = "answered without describing a server"
		rep.DiscoverAnswered = true
	default:
		rep.AnswersDiscover = true
	}

	call, _, err := exchange(ctx, timeout, name, args, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/list",
		"params": map[string]any{"_meta": modernMeta()},
	})
	if err != nil {
		return nil, fmt.Errorf("probing modern tools/list: %w", err)
	}
	if call != nil && call.Error == nil && call.Result != nil {
		rep.ServesModernCall = true
		rep.ModernResultIsModern = isModernResult(call.Result)
	}

	rep.classify()
	return rep, nil
}

// isModernResult reports whether a result carries the markers revision
// 2026-07-28 puts on one. A legacy server answering a modern request returns
// the old body, which has neither.
func isModernResult(raw json.RawMessage) bool {
	var r struct {
		ResultType string         `json:"resultType"`
		Meta       map[string]any `json:"_meta"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return false
	}
	if r.ResultType != "" {
		return true
	}
	_, ok := r.Meta[MetaServerInfo]
	return ok
}

// describesAServer reports whether a server/discover result is one.
//
// The revision replaces the initialize handshake with this method, so the
// answer is what a client negotiates from: DiscoverResult requires
// supportedVersions, capabilities, resultType, cacheScope and ttlMs
// (schema/2026-07-28/schema.json). Accepting any non-empty object instead would
// let a server that merely tolerates the method be published as dual-era, which
// is the row a client trusts most.
//
// supportedVersions is checked for this revision specifically. A server that
// answers the probe but does not list the version the probe declared has not
// agreed to speak it, and the list is what the spec tells a client to choose
// from.
func describesAServer(result json.RawMessage) bool {
	if len(result) == 0 {
		return false
	}
	var r struct {
		SupportedVersions []string         `json:"supportedVersions"`
		Capabilities      *json.RawMessage `json:"capabilities"`
		ResultType        *string          `json:"resultType"`
		CacheScope        *string          `json:"cacheScope"`
		TTLMs             *int64           `json:"ttlMs"`
	}
	if json.Unmarshal(result, &r) != nil {
		return false
	}
	if r.Capabilities == nil || r.ResultType == nil || r.CacheScope == nil || r.TTLMs == nil {
		return false
	}
	for _, v := range r.SupportedVersions {
		if v == Version {
			return true
		}
	}
	return false
}

func (r *Report) classify() {
	modern := r.AnswersDiscover || (r.ServesModernCall && r.ModernResultIsModern)

	switch {
	case modern && r.AnswersInitialize:
		r.Era = Dual
		r.note("Serves both eras, which is the only posture that works with every client.")
	case modern:
		r.Era = Modern
		r.note("Modern only. A legacy client has no fall-forward mechanism and will fail.")
	case r.AnswersInitialize:
		r.Era = Legacy
		r.note("Legacy only. A modern client that skips the server/discover probe is at risk.")
	default:
		r.Era = Unknown
		r.note("Answered neither opening.")
	}

	if r.ServesModernCall && !r.ModernResultIsModern {
		r.SilentDowngrade = true
		r.note("Ran a request declaring protocolVersion " + Version +
			" and answered in the legacy shape, with no error and no version acknowledgement. " +
			"A client reading only content cannot tell it was downgraded.")
	}
	switch {
	case r.AnswersDiscover || r.DiscoverError == "":
	case r.DiscoverAnswered:
		r.note("server/discover fails deterministically (" + r.DiscoverError +
			"), which is the probe the spec tells stdio clients to send first.")
	default:
		// Nothing came back, so nothing was learned about the method. Calling
		// that a deterministic failure would credit the server with a refusal
		// it never made.
		r.note("server/discover got no answer (" + r.DiscoverError +
			"), so this says nothing about whether the method is implemented.")
	}
}

func (r *Report) note(s string) { r.Notes = append(r.Notes, s) }

// exchange runs one request against a fresh server process and returns the
// first response, or nil when the server stayed silent.
// silence explains why a probe got nothing back. The three cases look alike to
// a caller and mean different things: a server that refused is a finding about
// the method, and a server that hung or died is a finding about the server.
type silence string

const (
	silentTimeout silence = "the server did not answer within the timeout"
	silentEOF     silence = "the server ended its output without answering"
)

func exchange(ctx context.Context, timeout time.Duration, name string, args []string, req map[string]any) (*rpcResponse, silence, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", err
	}
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, "", err
	}
	if _, err := stdin.Write(append(body, '\n')); err != nil {
		// The pipe closed under us, which means the process is already gone.
		return nil, silentEOF, nil
	}

	type result struct {
		resp *rpcResponse
		err  error
	}
	done := make(chan result, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || !strings.HasPrefix(line, "{") {
				continue
			}
			var resp rpcResponse
			if json.Unmarshal([]byte(line), &resp) != nil {
				continue
			}
			if resp.Result == nil && resp.Error == nil {
				continue // a notification, not our answer
			}
			done <- result{resp: &resp}
			return
		}
		done <- result{err: sc.Err()}
	}()

	select {
	case r := <-done:
		if r.err != nil && !errors.Is(r.err, context.DeadlineExceeded) {
			return nil, "", r.err
		}
		if r.resp == nil {
			// The scanner reached the end of the server's output without a
			// reply, so the process answered nothing and then stopped.
			return nil, silentEOF, nil
		}
		return r.resp, "", nil
	case <-ctx.Done():
		return nil, silentTimeout, nil // silence is a finding, not an error
	}
}
