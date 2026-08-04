package meshery

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ListOptions provides common pagination, search, and sorting parameters for REST calls.
type ListOptions struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Search   string `json:"search"`
	Order    string `json:"order"`
	Sort     string `json:"sort"`
}

// EncodeValues converts ListOptions into url.Values for query string construction using canonical API parameters.
func (o ListOptions) EncodeValues() url.Values {
	q := url.Values{}
	if o.Page > 0 {
		q.Set("page", strconv.Itoa(o.Page))
	}
	if o.PageSize > 0 {
		q.Set("pageSize", strconv.Itoa(o.PageSize))
	}
	if o.Search != "" {
		q.Set("search", o.Search)
	}
	if o.Order != "" {
		q.Set("order", o.Order)
	}
	if o.Sort != "" {
		q.Set("sort", o.Sort)
	}
	return q
}

// APIError represents an HTTP error response returned by the Meshery REST API.
type APIError struct {
	StatusCode int    `json:"statusCode"`
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
	CommitSHA      string `json:"commitSha,omitempty"`
	ReleaseChannel string `json:"releaseChannel,omitempty"`
}

// Connection represents a connection object returned by the Meshery REST API.
// Wire format strictly mirrors connection.yaml schema (mixing camelCase and snake_case properties).
type Connection struct {
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name,omitempty"`
	Kind         string         `json:"kind,omitempty"`
	Type         string         `json:"type,omitempty"`
	SubType      string         `json:"subType,omitempty"`
	Status       string         `json:"status,omitempty"`
	CredentialID string         `json:"credentialId,omitempty"`
	UserID       string         `json:"user_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at,omitempty"`
	UpdatedAt    time.Time      `json:"updated_at,omitempty"`
}

// ConnectionResponse represents the envelope response for connections list endpoint.
type ConnectionResponse struct {
	Page        int          `json:"page"`
	PageSize    int          `json:"pageSize"`
	TotalCount  int          `json:"totalCount"`
	Connections []Connection `json:"connections"`
}

// Pattern represents a design pattern object returned by the Meshery REST API.
type Pattern struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name,omitempty"`
	PatternFile string         `json:"patternFile,omitempty"`
	UserID      string         `json:"user_id,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at,omitempty"`
}

// PatternResponse represents the envelope response for patterns list endpoint.
type PatternResponse struct {
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
	TotalCount int       `json:"totalCount"`
	Patterns   []Pattern `json:"patterns"`
}

// Workspace represents a workspace object returned by the Meshery REST API.
type Workspace struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt,omitempty"`
	UpdatedAt   time.Time      `json:"updatedAt,omitempty"`
}

// WorkspaceResponse represents the envelope response for workspaces list endpoint.
type WorkspaceResponse struct {
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalCount int         `json:"totalCount"`
	Workspaces []Workspace `json:"workspaces"`
}

// Environment represents an environment object returned by the Meshery REST API.
type Environment struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt,omitempty"`
	UpdatedAt   time.Time      `json:"updatedAt,omitempty"`
}

// EnvironmentResponse represents the envelope response for environments list endpoint.
type EnvironmentResponse struct {
	Page         int           `json:"page"`
	PageSize     int           `json:"pageSize"`
	TotalCount   int           `json:"totalCount"`
	Environments []Environment `json:"environments"`
}

// Model represents a MeshModel registry model.
type Model struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Version     string         `json:"version,omitempty"`
	DisplayName string         `json:"displayName,omitempty"`
	Category    string         `json:"category,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ModelResponse represents the envelope response for models list endpoint.
type ModelResponse struct {
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalCount int     `json:"totalCount"`
	Models     []Model `json:"models"`
}

// PerformanceProfile represents a performance profile object returned by the Meshery REST API.
type PerformanceProfile struct {
	ID                 string         `json:"id,omitempty"`
	Name               string         `json:"name,omitempty"`
	ServiceMesh        string         `json:"serviceMesh,omitempty"`
	ConcurrentRequests int            `json:"concurrentRequests,omitempty"`
	Duration           string         `json:"duration,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	CreatedAt          time.Time      `json:"createdAt,omitempty"`
	UpdatedAt          time.Time      `json:"updatedAt,omitempty"`
}

// PerformanceProfileResponse represents the envelope response for performance profiles list endpoint.
type PerformanceProfileResponse struct {
	Page       int                  `json:"page"`
	PageSize   int                  `json:"pageSize"`
	TotalCount int                  `json:"totalCount"`
	Profiles   []PerformanceProfile `json:"profiles"`
}
