package meshery

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// RegistryPage is the shared envelope of /api/registry/models and
// /api/registry/components. Entries stay raw: model and component definitions
// are open-ended and owned by meshery/schemas.
type RegistryPage struct {
	Entries    []json.RawMessage `json:"-"`
	Page       int               `json:"page"`
	PageSize   int               `json:"pageSize"`
	TotalCount int64             `json:"totalCount"`
}

type modelPage struct {
	RegistryPage
	Models []json.RawMessage `json:"models"`
}

type componentPage struct {
	RegistryPage
	Components []json.RawMessage `json:"components"`
}

// RegistryListOptions are the filters both registry list routes accept.
type RegistryListOptions struct {
	ListOptions
	Version       string
	Sort          string
	Status        string
	Annotations   string
	Components    bool
	Relationships bool
	Trim          bool
	ID            string
	APIVersion    string
}

func (o RegistryListOptions) values() url.Values {
	q := url.Values{}
	o.apply(q)
	q.Del("pagesize")
	_, pageSize := o.bounds()
	q.Set("pageSize", strconv.Itoa(pageSize))
	if o.Version != "" {
		q.Set("version", o.Version)
	}
	if o.Sort != "" {
		q.Set("sort", o.Sort)
	}
	if o.Status != "" {
		q.Set("status", o.Status)
	}
	if o.Annotations != "" {
		q.Set("annotations", o.Annotations)
	}
	if o.Components {
		q.Set("components", "true")
	}
	if o.Relationships {
		q.Set("relationships", "true")
	}
	if o.Trim {
		q.Set("trim", "true")
	}
	if o.ID != "" {
		q.Set("id", o.ID)
	}
	if o.APIVersion != "" {
		q.Set("apiVersion", o.APIVersion)
	}
	return q
}

// ListModels lists models from the component/model registry.
func (c *Client) ListModels(ctx context.Context, opts RegistryListOptions) (RegistryPage, error) {
	var raw modelPage
	err := c.getJSON(ctx, "/api/registry/models", opts.values(), "list_models", &raw)
	page := raw.RegistryPage
	page.Entries = raw.Models
	return page, err
}

// ListComponents lists components from the component/model registry.
func (c *Client) ListComponents(ctx context.Context, opts RegistryListOptions) (RegistryPage, error) {
	var raw componentPage
	err := c.getJSON(ctx, "/api/registry/components", opts.values(), "list_components", &raw)
	page := raw.RegistryPage
	page.Entries = raw.Components
	return page, err
}
