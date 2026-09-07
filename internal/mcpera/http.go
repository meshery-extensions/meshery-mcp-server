package mcpera

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HeaderMismatchCode is the JSON-RPC error revision 2026-07-28 reserves for a
// request whose headers disagree with its body, or which is missing a required
// header.
const HeaderMismatchCode = -32020

// HTTPReport is what an HTTP probe found.
type HTTPReport struct {
	// ServesModern is true when a request declaring the modern version was
	// accepted rather than refused.
	ServesModern bool
	// ModernResultIsModern is true when that result carried the modern markers.
	ModernResultIsModern bool
	// RefusedModern carries why a modern request was turned away, when it was.
	RefusedModern string

	// ValidatesHeaderBody is true when a request whose Mcp-Name header disagrees
	// with the tool named in its body is rejected.
	ValidatesHeaderBody bool
	// MismatchStatus and MismatchCode are what that rejection looked like.
	MismatchStatus int
	MismatchCode   int
	// ExecutedMismatched is the hazard: the server ran the tool named in the
	// body while the header named a different one.
	ExecutedMismatched bool
	// BodyTruncated is true when a response was larger than this probe reads.
	// The verdicts that depend on parsing it are withheld rather than guessed,
	// because a cut-off body parses as nothing and would otherwise be reported
	// as the server having answered nonsense.
	BodyTruncated bool

	Notes []string
}

// ProbeHTTP sends a tool call to an MCP endpoint twice, once with the headers
// agreeing with the body and once with the Mcp-Name header naming a different
// tool, and reports whether the disagreement was caught.
//
// Revision 2026-07-28 requires servers to reject a header that disagrees with
// the body, and says why: an intermediary may route on the header while the
// server executes the body. A server predating the rule ignores the header and
// runs whatever the body asked for, so the two sources of truth diverge without
// anything on the wire saying so.
//
// agreeing and mismatched name two tools the server exposes. Both are called,
// so point this at a server whose tools are safe to run.
func ProbeHTTP(ctx context.Context, timeout time.Duration, url, agreeing, mismatched string, headers map[string]string) (*HTTPReport, error) {
	rep := &HTTPReport{}
	client := &http.Client{
		Timeout: timeout,
		// A redirect is not an answer about protocol eras, and following one
		// rewrites this POST into a GET against a different URL, so whatever
		// came back would describe neither the request nor the endpoint asked
		// about.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	status, body, truncated, err := callTool(ctx, client, url, agreeing, agreeing, headers)
	if err != nil {
		return nil, fmt.Errorf("calling %s: %w", agreeing, err)
	}
	switch {
	case status != http.StatusOK:
		rep.RefusedModern = fmt.Sprintf("%d: %s", status, firstLine(body))
	default:
		// JSON-RPC carries its errors in the body, so a 200 is not by itself an
		// acceptance. A server refusing the version answers 200 with an error
		// object, and scoring that as service would publish it in the
		// silent-downgrade row: serving the modern era with a legacy result.
		if code, ok := rpcErrorCode(body); ok {
			rep.RefusedModern = fmt.Sprintf("200 with JSON-RPC error %d: %s", code, firstLine(body))
			rep.note(fmt.Sprintf("Refused the agreeing call in the body rather than the status line, JSON-RPC error %d under HTTP 200.", code))
			break
		}
		if truncated {
			// The call was served; only the shape is unknown, so the result
			// markers are not judged either way.
			rep.ServesModern = true
			rep.BodyTruncated = true
			rep.note(fmt.Sprintf("Served the agreeing call with a body larger than the %d bytes this probe reads, so the result shape was not judged.", maxBody))
			break
		}
		if jsonPayload(body) == nil {
			rep.RefusedModern = fmt.Sprintf("200 with a body that is not JSON-RPC: %s", firstLine(body))
			rep.note("Answered the agreeing call 200 with a body that is not JSON-RPC, so nothing here describes an MCP server.")
			break
		}
		rep.ServesModern = true
		rep.ModernResultIsModern = resultIsModern(body)
	}

	// The header names one tool, the body names another.
	status, body, mismatchTruncated, err := callTool(ctx, client, url, mismatched, agreeing, headers)
	if err != nil {
		return nil, fmt.Errorf("calling %s with a mismatched header: %w", agreeing, err)
	}
	rep.MismatchStatus = status
	rep.MismatchCode = errorCode(body)

	switch {
	case !rep.ServesModern:
		// The endpoint would not serve the agreeing call either, so it never
		// judged the header and no verdict about it is available.
		rep.note(fmt.Sprintf("Refused the agreeing call too (%s), so this says nothing about header validation; the endpoint did not serve either request.",
			rep.RefusedModern))
	case rep.MismatchCode == HeaderMismatchCode && status == http.StatusBadRequest:
		rep.ValidatesHeaderBody = true
		rep.note(fmt.Sprintf("Rejects a header that disagrees with the body, %d with %d, as revision %s requires.",
			status, HeaderMismatchCode, Version))
	case rep.MismatchCode == HeaderMismatchCode:
		// The server caught it, so an intermediary and this server agree about
		// what ran. The revision requires both the status and the code from a
		// server, though, and a client keying on the status alone would read
		// this as a success.
		rep.note(fmt.Sprintf("Caught the mismatch with %d, but under HTTP %d rather than the 400 revision %s requires alongside it.",
			HeaderMismatchCode, status, Version))
	case rep.MismatchCode != 0:
		// JSON-RPC carries its errors in the body, so an error here is a
		// rejection whatever the status line says. It is simply not the code
		// the revision specifies.
		rep.note(fmt.Sprintf("Turned the mismatch away with JSON-RPC error %d (HTTP %d), not the %d the revision specifies.",
			rep.MismatchCode, status, HeaderMismatchCode))
	case mismatchTruncated:
		rep.BodyTruncated = true
		rep.note(fmt.Sprintf("Answered the mismatched call with a body larger than the %d bytes this probe reads, so whether the tool ran is unknown.", maxBody))
	case status == http.StatusOK && jsonPayload(body) == nil:
		// errorCode cannot tell a server that answered without an error from
		// one whose body it could not read at all, and both arrive here as 0.
		// Claiming an execution on the strength of an unreadable body would put
		// a proxy error page or an SPA fallback in the hazard row.
		rep.note("Answered the mismatched call 200 with a body that is not JSON-RPC, so whether the tool ran is unknown.")
	case status == http.StatusOK:
		rep.ExecutedMismatched = true
		rep.note("Ran the tool named in the body while the Mcp-Name header named a different one. " +
			"An intermediary routing on that header and this server disagree about what just executed.")
	default:
		rep.note(fmt.Sprintf("Turned the mismatch away with %d, but not with the %d the revision specifies.",
			status, HeaderMismatchCode))
	}
	return rep, nil
}

// encodeHeaderValue renders a value for Mcp-Name or Mcp-Param-{Name}.
//
// RFC 9110 limits header field values to visible ASCII, space and horizontal
// tab, and revision 2026-07-28 says a client MUST carry anything outside that
// set, or with surrounding whitespace, in a Base64 sentinel. A probe that
// tests conformance has to be conformant itself, or it measures a server
// against a request the spec does not permit.
//
// The markers are lowercase and exact, since servers match on them literally
// before comparing the decoded value to the body.
func encodeHeaderValue(v string) string {
	if headerSafe(v) {
		return v
	}
	return "=?base64?" + base64.StdEncoding.EncodeToString([]byte(v)) + "?="
}

// headerSafe reports whether a value can travel as a plain header value.
//
// A value already shaped like the sentinel is not safe even when it is plain
// ASCII. The revision requires encoding those too, because a server decodes
// anything wearing the markers before comparing it to the body, so a tool
// genuinely named "=?base64?literal?=" would be decoded into something else and
// the agreeing call would be rejected as a mismatch this probe caused.
func headerSafe(v string) bool {
	if v == "" {
		return true
	}
	if strings.TrimSpace(v) != v {
		// Leading or trailing whitespace does not survive the wire intact.
		return false
	}
	if strings.HasPrefix(v, "=?base64?") && strings.HasSuffix(v, "?=") {
		return false
	}
	for _, r := range v {
		if r == ' ' || r == '\t' {
			continue
		}
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}

// maxBody bounds a response read. It matches the stdio probe's scanner limit so
// the two paths agree on what counts as too large to judge.
const maxBody = 8 << 20

func callTool(ctx context.Context, c *http.Client, url, headerName, bodyName string, extra map[string]string) (int, []byte, bool, error) {
	payload := map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{
			"name": bodyName, "arguments": map[string]any{},
			"_meta": modernMeta(),
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", Version)
	req.Header.Set("Mcp-Method", "tools/call")
	req.Header.Set("Mcp-Name", encodeHeaderValue(headerName))
	for k, v := range extra {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, false, err
	}
	defer resp.Body.Close()
	// One byte past the cap, so a body that reached it can be told from one that
	// merely ended there. A truncated read parses as nothing, and reporting that
	// as a malformed answer would blame the server for this probe's limit.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	truncated := len(body) > maxBody
	if truncated {
		body = body[:maxBody]
	}
	return resp.StatusCode, body, truncated, err
}

// jsonPayload pulls the JSON object out of a response, whether it arrived as a
// plain body or as SSE events.
//
// The whole body is tried first, because a pretty-printed object spans many
// lines and taking the first line that opens a brace would return just that
// brace. Only if that fails are data: lines gathered, which is how an SSE event
// carries a payload, one event's lines joined.
func jsonPayload(body []byte) []byte {
	if trimmed := bytes.TrimSpace(body); json.Valid(trimmed) && len(trimmed) > 0 && trimmed[0] == '{' {
		return trimmed
	}
	var event []string
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Blank line ends an event. Take the first one that parses.
			if joined := strings.Join(event, ""); json.Valid([]byte(joined)) && strings.HasPrefix(joined, "{") {
				return []byte(joined)
			}
			event = nil
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, "data:"); ok {
			event = append(event, strings.TrimSpace(rest))
		}
	}
	if joined := strings.Join(event, ""); json.Valid([]byte(joined)) && strings.HasPrefix(joined, "{") {
		return []byte(joined)
	}
	return nil
}

func errorCode(body []byte) int {
	code, _ := rpcErrorCode(body)
	return code
}

// rpcErrorCode reports the JSON-RPC error a body carries, and whether it
// carried one at all.
//
// The two are different questions and the code alone cannot separate them: a
// body with no error, and a body that is not JSON-RPC in the first place, both
// have no code. Reading a missing code as "the call went through" is what puts
// a proxy error page in the executed-the-mismatch row.
func rpcErrorCode(body []byte) (int, bool) {
	raw := jsonPayload(body)
	if raw == nil {
		return 0, false
	}
	var r struct {
		Error *rpcError `json:"error"`
	}
	if json.Unmarshal(raw, &r) != nil || r.Error == nil {
		return 0, false
	}
	return r.Error.Code, true
}

func resultIsModern(body []byte) bool {
	raw := jsonPayload(body)
	if raw == nil {
		return false
	}
	var r struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal(raw, &r) != nil || r.Result == nil {
		return false
	}
	return isModernResult(r.Result)
}

func firstLine(body []byte) string {
	s := strings.TrimSpace(string(body))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

func (r *HTTPReport) note(s string) { r.Notes = append(r.Notes, s) }
