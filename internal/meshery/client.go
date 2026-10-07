// Package meshery provides internal HTTP client implementations for communicating with Meshery Server.
package meshery

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/meshery-extensions/meshery-mcp-server/pkg/errors"
	"github.com/meshery-extensions/meshery-mcp-server/pkg/security"
)

// Client defines the interface for communicating with Meshery Server APIs.
type Client interface {
	Ping(ctx context.Context) (map[string]interface{}, error)
	ListDesigns(ctx context.Context, page, pageSize int, search string) ([]map[string]interface{}, int, error)
	GetEnvironments(ctx context.Context, orgID string, page, pageSize int) ([]map[string]interface{}, int, error)
	GetEnvironmentByID(ctx context.Context, environmentID string) (map[string]interface{}, error)
	CreateEnvironment(ctx context.Context, name, description, orgID string) (map[string]interface{}, error)
	ListWorkspaces(ctx context.Context, orgID string, page, pageSize int) ([]map[string]interface{}, int, error)
	GetWorkspaceByID(ctx context.Context, workspaceID string) (map[string]interface{}, error)
	GetConnections(ctx context.Context, page, pageSize int) ([]map[string]interface{}, int, error)
	GetAdapters(ctx context.Context) ([]map[string]interface{}, error)
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
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
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
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Prevent automatic redirects to HTML provider login pages on unauthenticated API calls
				return http.ErrUseLastResponse
			},
		},
	}
}

// attachCredentials attaches token authorization header and dual session cookies over HTTPS or loopback transport.
func (c *mesheryClient) attachCredentials(req *http.Request, u *url.URL) {
	if c.token == "" || u == nil {
		return
	}
	isSecureScheme := u.Scheme == "https"
	isLoopbackHost := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if isSecureScheme || isLoopbackHost {
		cleanToken := strings.TrimPrefix(c.token, "Bearer ")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cleanToken))
		req.AddCookie(&http.Cookie{Name: "token", Value: cleanToken})
		req.AddCookie(&http.Cookie{Name: "meshery-provider", Value: c.provider})
	}
}

// ListDesigns retrieves available design patterns from Meshery Server /api/pattern endpoint with 0-indexed pagination & search.
func (c *mesheryClient) ListDesigns(ctx context.Context, page, pageSize int, search string) ([]map[string]interface{}, int, error) {
	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/pattern", cleanBaseURL)
	u, err := url.Parse(endpointURL)
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

	c.attachCredentials(req, u)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, 0, fmt.Errorf("failed to execute list_designs HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	// Check for auth redirects (302 Found or 307 Temporary Redirect to /provider)
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, errors.ErrUnauthenticated(fmt.Errorf("status %d", resp.StatusCode))
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

// Ping checks connectivity to Meshery Server by calling GET /api/system/version.
func (c *mesheryClient) Ping(ctx context.Context) (map[string]interface{}, error) {
	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/system/version", cleanBaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping request: %w", err)
	}

	u, err := url.Parse(endpointURL)
	if err == nil {
		c.attachCredentials(req, u)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("failed to execute ping HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.ErrUnauthenticated(fmt.Errorf("ping status %d", resp.StatusCode))
	}

	if resp.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, fmt.Errorf("meshery API ping returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ping JSON response: %w", err)
	}

	return result, nil
}

// GetEnvironments retrieves environments for an organization from /api/environments?orgId=... with pagination.
func (c *mesheryClient) GetEnvironments(ctx context.Context, orgID string, page, pageSize int) ([]map[string]interface{}, int, error) {
	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/environments", cleanBaseURL)
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse environments endpoint URL: %w", err)
	}

	q := u.Query()
	if orgID != "" {
		q.Set("orgId", orgID)
	}
	if page >= 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("pagesize", strconv.Itoa(pageSize))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create environments request: %w", err)
	}

	c.attachCredentials(req, u)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, 0, fmt.Errorf("failed to execute environments HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, errors.ErrUnauthenticated(fmt.Errorf("environments status %d", resp.StatusCode))
	}

	if resp.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, 0, fmt.Errorf("meshery API environments returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	var payload struct {
		TotalCount   int                      `json:"totalCount"`
		AltTotal     int                      `json:"total_count"`
		Environments []map[string]interface{} `json:"environments"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, 0, fmt.Errorf("failed to decode environments JSON response: %w", err)
	}

	total := payload.TotalCount
	if total == 0 {
		total = payload.AltTotal
	}
	if total == 0 {
		total = len(payload.Environments)
	}

	return payload.Environments, total, nil
}

// GetConnections retrieves MeshSync connection state from /api/system/meshsync/connections with pagination.
func (c *mesheryClient) GetConnections(ctx context.Context, page, pageSize int) ([]map[string]interface{}, int, error) {
	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/system/meshsync/connections", cleanBaseURL)
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse connections endpoint URL: %w", err)
	}

	q := u.Query()
	if page >= 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("pagesize", strconv.Itoa(pageSize))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create connections request: %w", err)
	}

	c.attachCredentials(req, u)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, 0, fmt.Errorf("failed to execute connections HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, errors.ErrUnauthenticated(fmt.Errorf("connections status %d", resp.StatusCode))
	}

	if resp.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, 0, fmt.Errorf("meshery API connections returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	var payload struct {
		TotalCount  int                      `json:"totalCount"`
		AltTotal    int                      `json:"total_count"`
		Connections []map[string]interface{} `json:"connections"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, 0, fmt.Errorf("failed to decode connections JSON response: %w", err)
	}

	total := payload.TotalCount
	if total == 0 {
		total = payload.AltTotal
	}
	if total == 0 {
		total = len(payload.Connections)
	}

	return payload.Connections, total, nil
}

// GetAdapters retrieves available mesh adapters from /api/system/adapters.
func (c *mesheryClient) GetAdapters(ctx context.Context) ([]map[string]interface{}, error) {
	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/system/adapters", cleanBaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapters request: %w", err)
	}

	u, err := url.Parse(endpointURL)
	if err == nil {
		c.attachCredentials(req, u)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("failed to execute adapters HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.ErrUnauthenticated(fmt.Errorf("adapters status %d", resp.StatusCode))
	}

	if resp.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, fmt.Errorf("meshery API adapters returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	var adapters []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&adapters); err != nil {
		return nil, fmt.Errorf("failed to decode adapters JSON response: %w", err)
	}

	return adapters, nil
}

// GetEnvironmentByID fetches details of a single environment by ID from /api/environments/{id}.
func (c *mesheryClient) GetEnvironmentByID(ctx context.Context, environmentID string) (map[string]interface{}, error) {
	environmentID = strings.TrimSpace(environmentID)
	if environmentID == "" {
		return nil, fmt.Errorf("environmentID cannot be empty")
	}

	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/environments/%s", cleanBaseURL, url.PathEscape(environmentID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get_environment request: %w", err)
	}

	u, err := url.Parse(endpointURL)
	if err == nil {
		c.attachCredentials(req, u)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("failed to execute get_environment HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.ErrUnauthenticated(fmt.Errorf("get_environment status %d", resp.StatusCode))
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("environment with ID %q not found", environmentID)
	}

	if resp.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, fmt.Errorf("meshery API get_environment returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	var env map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("failed to decode get_environment JSON response: %w", err)
	}

	return env, nil
}

// CreateEnvironment creates a new environment via POST /api/environments.
func (c *mesheryClient) CreateEnvironment(ctx context.Context, name, description, orgID string) (map[string]interface{}, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("environment name cannot be empty")
	}

	payload := map[string]interface{}{
		"name": name,
	}
	if description != "" {
		payload["description"] = description
	}
	if orgID != "" {
		payload["organization_id"] = orgID
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create_environment payload: %w", err)
	}

	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/environments", cleanBaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create create_environment request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	u, err := url.Parse(endpointURL)
	if err == nil {
		c.attachCredentials(req, u)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("failed to execute create_environment HTTP request: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.ErrUnauthenticated(fmt.Errorf("create_environment status %d", resp.StatusCode))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, fmt.Errorf("meshery API create_environment returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode create_environment JSON response: %w", err)
	}

	return result, nil
}

// ListWorkspaces retrieves available workspaces from /api/workspaces with pagination.
func (c *mesheryClient) ListWorkspaces(ctx context.Context, orgID string, page, pageSize int) ([]map[string]interface{}, int, error) {
	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/workspaces", cleanBaseURL)
	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse workspaces endpoint URL: %w", err)
	}

	q := u.Query()
	if orgID != "" {
		q.Set("orgId", orgID)
	}
	if page >= 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("pagesize", strconv.Itoa(pageSize))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create workspaces request: %w", err)
	}

	c.attachCredentials(req, u)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, 0, fmt.Errorf("failed to execute workspaces HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, errors.ErrUnauthenticated(fmt.Errorf("workspaces status %d", resp.StatusCode))
	}

	if resp.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, 0, fmt.Errorf("meshery API workspaces returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	var payload struct {
		TotalCount int                      `json:"totalCount"`
		AltTotal   int                      `json:"total_count"`
		Workspaces []map[string]interface{} `json:"workspaces"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, 0, fmt.Errorf("failed to decode workspaces JSON response: %w", err)
	}

	total := payload.TotalCount
	if total == 0 {
		total = payload.AltTotal
	}
	if total == 0 {
		total = len(payload.Workspaces)
	}

	return payload.Workspaces, total, nil
}

// GetWorkspaceByID fetches details of a single workspace by ID from /api/workspaces/{id}.
func (c *mesheryClient) GetWorkspaceByID(ctx context.Context, workspaceID string) (map[string]interface{}, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("workspaceID cannot be empty")
	}

	cleanBaseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	endpointURL := fmt.Sprintf("%s/api/workspaces/%s", cleanBaseURL, url.PathEscape(workspaceID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get_workspace request: %w", err)
	}

	u, err := url.Parse(endpointURL)
	if err == nil {
		c.attachCredentials(req, u)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		sanitizedErr := security.SanitizeString(err.Error())
		return nil, fmt.Errorf("failed to execute get_workspace HTTP query: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusSeeOther || resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.ErrUnauthenticated(fmt.Errorf("get_workspace status %d", resp.StatusCode))
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workspace with ID %q not found", workspaceID)
	}

	if resp.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(resp.Body, 64*1024)
		body, _ := io.ReadAll(limitedReader)
		sanitizedBody := security.SanitizeString(string(body))
		return nil, fmt.Errorf("meshery API get_workspace returned status %d: %s", resp.StatusCode, sanitizedBody)
	}

	var ws map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
		return nil, fmt.Errorf("failed to decode get_workspace JSON response: %w", err)
	}

	return ws, nil
}
