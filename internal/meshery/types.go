package meshery

import (
	"net/url"
	"strconv"
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

// Version represents the system version metadata returned by /api/system/version.
type Version struct {
	Build          string `json:"build,omitempty"`
	CommitSHA      string `json:"commitsha,omitempty"`
	Latest         string `json:"latest,omitempty"`
	Outdated       bool   `json:"outdated,omitempty"`
	ReleaseChannel string `json:"releaseChannel,omitempty"`
	Version        string `json:"version,omitempty"`
}
