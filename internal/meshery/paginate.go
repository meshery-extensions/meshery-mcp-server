package meshery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	defaultPageSize = 10
	maxPageSize     = 100
)

// ListOptions are the common list parameters. Zero values select the server's
// defaults; PageSize is clamped to 1..100 so a tool cannot ask for everything.
type ListOptions struct {
	Page     int
	PageSize int
	Search   string
	Order    string
	Filter   string
}

func (o ListOptions) bounds() (page, pageSize int) {
	page = o.Page
	if page < 0 {
		page = 0
	}
	pageSize = o.PageSize
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func (o ListOptions) apply(q url.Values) {
	page, pageSize := o.bounds()
	q.Set("page", strconv.Itoa(page))
	q.Set("pagesize", strconv.Itoa(pageSize))
	if o.Search != "" {
		q.Set("search", o.Search)
	}
	if o.Order != "" {
		q.Set("order", o.Order)
	}
	if o.Filter != "" {
		q.Set("filter", o.Filter)
	}
}

func queryFor(opts ListOptions) url.Values {
	q := url.Values{}
	opts.apply(q)
	return q
}

// getJSON performs a read-only request without requiring a token (local
// provider mode needs none) and decodes the response body into out.
func (c *Client) getJSON(ctx context.Context, path string, q url.Values, tool string, out any) error {
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	body, err := c.call(ctx, http.MethodGet, path, nil, tool, false)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decoding %s response: %w", tool, err)
	}
	return nil
}
