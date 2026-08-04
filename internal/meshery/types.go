package meshery

import (
	"fmt"
	"net/url"
	"strconv"
)

// ListOptions provides common pagination and filtering parameters for REST calls.
type ListOptions struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pagesize"`
	Search   string `json:"search"`
}

// EncodeValues converts ListOptions into url.Values for query string construction.
func (o ListOptions) EncodeValues() url.Values {
	q := url.Values{}
	if o.Page > 0 {
		q.Set("page", strconv.Itoa(o.Page))
	}
	if o.PageSize > 0 {
		q.Set("pagesize", strconv.Itoa(o.PageSize))
	}
	if o.Search != "" {
		q.Set("search", o.Search)
	}
	return q
}

// APIError represents an HTTP error response returned by the Meshery REST API.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	Message    string `json:"message"`
	RawBody    []byte `json:"-"`
}

// Error implements the standard error interface for APIError.
func (e *APIError) Error() string {
	return fmt.Sprintf("meshery API [%s %s] failed (%d): %s", e.Method, e.URL, e.StatusCode, e.Message)
}

// Version represents the system version metadata returned by /api/system/version.
type Version struct {
	Build          string `json:"build,omitempty"`
	CommitSHA      string `json:"commit_sha,omitempty"`
	ReleaseChannel string `json:"release_channel,omitempty"`
}

// Connection represents a connection object returned by the Meshery REST API.
type Connection struct {
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Kind      string                 `json:"kind,omitempty"`
	Type      string                 `json:"type,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt string                 `json:"created_at,omitempty"`
	UpdatedAt string                 `json:"updated_at,omitempty"`
}

// ConnectionResponse represents the envelope response for connections list endpoint.
type ConnectionResponse struct {
	Page        int          `json:"page"`
	PageSize    int          `json:"page_size"`
	TotalCount  int          `json:"total_count"`
	Connections []Connection `json:"connections"`
}

// Pattern represents a design pattern object returned by the Meshery REST API.
type Pattern struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	PatternFile string                 `json:"pattern_file,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"created_at,omitempty"`
	UpdatedAt   string                 `json:"updated_at,omitempty"`
}

// PatternResponse represents the envelope response for patterns list endpoint.
type PatternResponse struct {
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalCount int       `json:"total_count"`
	Patterns   []Pattern `json:"patterns"`
}

// Workspace represents a workspace object returned by the Meshery REST API.
type Workspace struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"created_at,omitempty"`
	UpdatedAt   string                 `json:"updated_at,omitempty"`
}

// WorkspaceResponse represents the envelope response for workspaces list endpoint.
type WorkspaceResponse struct {
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalCount int         `json:"total_count"`
	Workspaces []Workspace `json:"workspaces"`
}

// Environment represents an environment object returned by the Meshery REST API.
type Environment struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   string                 `json:"created_at,omitempty"`
	UpdatedAt   string                 `json:"updated_at,omitempty"`
}

// EnvironmentResponse represents the envelope response for environments list endpoint.
type EnvironmentResponse struct {
	Page         int           `json:"page"`
	PageSize     int           `json:"page_size"`
	TotalCount   int           `json:"total_count"`
	Environments []Environment `json:"environments"`
}

// Model represents a MeshModel component object returned by the Meshery REST API.
type Model struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Version     string                 `json:"version,omitempty"`
	DisplayName string                 `json:"display_name,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ModelResponse represents the envelope response for models list endpoint.
type ModelResponse struct {
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalCount int     `json:"total_count"`
	Models     []Model `json:"models"`
}

// PerformanceProfile represents a performance profile object returned by the Meshery REST API.
type PerformanceProfile struct {
	ID                 string                 `json:"id,omitempty"`
	Name               string                 `json:"name,omitempty"`
	ServiceMesh        string                 `json:"service_mesh,omitempty"`
	ConcurrentRequests int                    `json:"concurrent_requests,omitempty"`
	Duration           string                 `json:"duration,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          string                 `json:"created_at,omitempty"`
	UpdatedAt          string                 `json:"updated_at,omitempty"`
}

// PerformanceProfileResponse represents the envelope response for performance profiles list endpoint.
type PerformanceProfileResponse struct {
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalCount int                  `json:"total_count"`
	Profiles   []PerformanceProfile `json:"profiles"`
}
