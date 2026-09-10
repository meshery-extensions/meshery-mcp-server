// Package meshery provides internal HTTP client implementations for communicating with Meshery Server.
package meshery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/meshery-extensions/meshery-mcp-server/pkg/security"
)

// Client defines the interface for communicating with Meshery Server APIs.
type Client interface {
	ListDesigns(ctx context.Context, page, pageSize int, search string) ([]map[string]interface{}, int, error)
}

type mesheryClient struct {
	baseURL    string
	token      string
	provider   string
	httpClient *http.Client
}

// NewClient returns a new Meshery API client instance with optional token and provider authentication values.
// Usage: NewClient(baseURL, [token], [provider])
func NewClient(baseURL string, args ...string) Client {
	if baseURL == "" {
		baseURL = "http://localhost:9081"
	}
	t := ""
	p := "None"
	if len(args) > 0 {
		t = args[0]
	}
	if len(args) > 1 && args[1] != "" {
		p = args[1]
	}

	return &mesheryClient{
		baseURL:  baseURL,
		token:    t,
		provider: p,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Prevent automatic redirects to HTML provider login pages on unauthenticated API calls
				return http.ErrUseLastResponse
			},
		},
	}
}

// ListDesigns retrieves available design patterns from Meshery Server /api/pattern endpoint with 0-indexed pagination & search.
func (c *mesheryClient) ListDesigns(ctx context.Context, page, pageSize int, search string) ([]map[string]interface{}, int, error) {
	baseURL := fmt.Sprintf("%s/api/pattern", c.baseURL)
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse list_designs endpoint URL: %w", err)
	}

	q := u.Query()
	if page >= 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("pagesize", strconv.Itoa(pageSize))
	}
	if search != "" {
		q.Set("search", search)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create list_designs request: %w", err)
	}

	// CWE-319 Cleartext Transmission Prevention:
	// Only attach session credentials over HTTPS or secure loopback connections (localhost / 127.0.0.1)
	isSecureScheme := u.Scheme == "https"
	isLoopbackHost := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"

	if c.token != "" && (isSecureScheme || isLoopbackHost) {
		cleanToken := strings.TrimPrefix(c.token, "Bearer ")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cleanToken))

		// Send both token and meshery-provider cookies required for full Meshery session auth
		req.AddCookie(&http.Cookie{Name: "token", Value: cleanToken})
		req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: c.provider})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, 0, fmt.Errorf("failed to execute list_designs HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	// Check for auth redirects (302 Found or 307 Temporary Redirect to /provider)
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, fmt.Errorf("unauthenticated request: Meshery Server returned status %d (authentication required)", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		// Bound error body reading to 64KB max to prevent memory exhaustion on large upstream responses
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, 0, fmt.Errorf("meshery API returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	// Meshery Server server/models/meshery_patterns_api_response.go tags this field as totalCount
	var payload struct {
		TotalCount int                      `json:"totalCount"`
		Patterns   []map[string]interface{} `json:"patterns"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, 0, fmt.Errorf("failed to decode list_designs JSON response: %w", err)
	}

	if payload.TotalCount == 0 {
		payload.TotalCount = len(payload.Patterns)
	}

	return payload.Patterns, payload.TotalCount, nil
}
