package tools

import (
	"math"
	"testing"
)

func TestPageIn_Resolve(t *testing.T) {
	tests := []struct {
		name    string
		in      PageIn
		want    Page
		wantErr string // argument named in the error; "" = expect success
	}{
		{name: "defaults", in: PageIn{}, want: Page{Index: 0, Size: defaultPageSize}},
		{name: "explicit values", in: PageIn{Page: 2, PageSize: 10}, want: Page{Index: 2, Size: 10}},
		{name: "page size is capped", in: PageIn{PageSize: 500}, want: Page{Index: 0, Size: maxPageSize}},
		{name: "negative page", in: PageIn{Page: -1}, wantErr: "page"},
		{name: "negative page size", in: PageIn{PageSize: -1}, wantErr: "pageSize"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.resolve()
			if tt.wantErr != "" {
				wantError(t, err, ErrNegativeArgumentCode, `"`+tt.wantErr+`"`)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolve() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPage_Out(t *testing.T) {
	tests := []struct {
		name  string
		page  Page
		total int
		want  bool // HasMore
	}{
		{name: "no results", page: Page{Index: 0, Size: 25}, total: 0, want: false},
		{name: "one partial page", page: Page{Index: 0, Size: 25}, total: 10, want: false},
		{name: "exactly one full page", page: Page{Index: 0, Size: 25}, total: 25, want: false},
		{name: "one more than a page", page: Page{Index: 0, Size: 25}, total: 26, want: true},
		{name: "last page", page: Page{Index: 1, Size: 25}, total: 26, want: false},
		{name: "middle page", page: Page{Index: 1, Size: 10}, total: 35, want: true},
		{name: "past the end", page: Page{Index: 9, Size: 10}, total: 35, want: false},
		{name: "huge index does not overflow", page: Page{Index: math.MaxInt, Size: 100}, total: 1000, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.page.out(tt.total)
			if got.HasMore != tt.want {
				t.Errorf("HasMore = %v, want %v", got.HasMore, tt.want)
			}
			if got.Page != tt.page.Index || got.PageSize != tt.page.Size || got.TotalCount != tt.total {
				t.Errorf("out = %+v, want page %d, size %d, total %d", got, tt.page.Index, tt.page.Size, tt.total)
			}
		})
	}
}

func TestMapItems_EmptyIsNotNil(t *testing.T) {
	got := mapItems([]int(nil), func(i int) int { return i })
	if got == nil || len(got) != 0 {
		t.Errorf("mapItems(nil) = %#v, want empty non-nil slice", got)
	}
}
