package tools

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

// PageIn and PageOut are embedded in list tools' input and output; the SDK
// flattens embedded fields into the schema.
type PageIn struct {
	Page     int `json:"page,omitempty" jsonschema:"zero-based page index; defaults to 0"`
	PageSize int `json:"pageSize,omitempty" jsonschema:"results per page; defaults to 25, at most 100"`
}

type PageOut struct {
	Page       int  `json:"page" jsonschema:"zero-based index of this page"`
	PageSize   int  `json:"pageSize" jsonschema:"results per page used for this response"`
	TotalCount int  `json:"totalCount" jsonschema:"number of results across all pages"`
	HasMore    bool `json:"hasMore" jsonschema:"true if another page follows; to get it, call the tool again with page set to page + 1"`
}

type Page struct {
	Index int // zero-based
	Size  int // 1 to maxPageSize
}

// Total counts results across all pages, like Meshery's totalCount.
type Result[T any] struct {
	Items []T
	Total int
}

// A page size above the maximum is lowered rather than rejected.
func (p PageIn) resolve() (Page, error) {
	if p.Page < 0 {
		return Page{}, ErrNegativeArgument("page", p.Page)
	}
	switch {
	case p.PageSize < 0:
		return Page{}, ErrNegativeArgument("pageSize", p.PageSize)
	case p.PageSize == 0:
		p.PageSize = defaultPageSize
	case p.PageSize > maxPageSize:
		p.PageSize = maxPageSize
	}
	return Page{Index: p.Page, Size: p.PageSize}, nil
}

func (p Page) out(total int) PageOut {
	return PageOut{
		Page:       p.Index,
		PageSize:   p.Size,
		TotalCount: total,
		// A division, so a huge page index cannot overflow.
		HasMore: total > 0 && p.Index < (total-1)/p.Size,
	}
}

// mapItems never returns nil, so the output has [] rather than null.
func mapItems[T, S any](items []T, f func(T) S) []S {
	out := make([]S, 0, len(items))
	for _, item := range items {
		out = append(out, f(item))
	}
	return out
}
