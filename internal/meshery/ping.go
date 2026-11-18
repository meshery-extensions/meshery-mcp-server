package meshery

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// VersionInfo is the subset of GET /api/system/version the MCP server uses.
type VersionInfo struct {
	Build          string `json:"build,omitempty"`
	Commitsha      string `json:"commitsha,omitempty"`
	Latest         string `json:"latest,omitempty"`
	Outdated       bool   `json:"outdated,omitempty"`
	ReleaseChannel string `json:"releaseChannel,omitempty"`
	Version        string `json:"version,omitempty"`
}

// ConnectErrorKind classifies a transport failure.
type ConnectErrorKind string

const (
	ConnectRefused ConnectErrorKind = "connection refused"
	ConnectDNS     ConnectErrorKind = "dns failure"
	ConnectTimeout ConnectErrorKind = "timeout"
	ConnectTLS     ConnectErrorKind = "tls failure"
	ConnectOther   ConnectErrorKind = "transport failure"
)

// ConnectError reports that the server could not be reached. It names the
// URL but never includes credentials.
type ConnectError struct {
	URL  string
	Kind ConnectErrorKind
	Err  error
}

func (e *ConnectError) Error() string {
	return fmt.Sprintf("Meshery at %s is unreachable (%s): %v", e.URL, e.Kind, e.Err)
}

func (e *ConnectError) Unwrap() error { return e.Err }

// Ping checks that Meshery is reachable and returns its version. The route is
// unauthenticated, so a local server works without a token.
func (c *Client) Ping(ctx context.Context) (VersionInfo, error) {
	const path = "/api/system/version"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return VersionInfo{}, &ConnectError{URL: c.baseURL, Kind: ConnectOther, Err: err}
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return VersionInfo{}, &ConnectError{URL: c.baseURL, Kind: classifyConnect(err), Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return VersionInfo{}, &ConnectError{URL: c.baseURL, Kind: classifyConnect(err), Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return VersionInfo{}, c.apiError(http.MethodGet, path, resp.StatusCode, body, "")
	}

	var info VersionInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return VersionInfo{}, &APIError{
			Status:  resp.StatusCode,
			Method:  http.MethodGet,
			Path:    path,
			Message: "response was not valid JSON: " + err.Error(),
			Kind:    ErrServer,
		}
	}
	return info, nil
}

func classifyConnect(err error) ConnectErrorKind {
	var dnsErr *net.DNSError
	var netErr net.Error
	switch {
	case stderrors.As(err, &dnsErr):
		return ConnectDNS
	case stderrors.Is(err, context.DeadlineExceeded):
		return ConnectTimeout
	case stderrors.As(err, &netErr) && netErr.Timeout():
		return ConnectTimeout
	case strings.Contains(err.Error(), "connection refused"):
		return ConnectRefused
	case strings.Contains(err.Error(), "tls"):
		return ConnectTLS
	default:
		return ConnectOther
	}
}
