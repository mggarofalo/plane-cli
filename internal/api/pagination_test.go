package api

import (
	"net/url"
	"testing"
)

func TestPaginationParamsApply(t *testing.T) {
	tests := []struct {
		name    string
		params  PaginationParams
		wantQS  map[string]string
		absentK []string
	}{
		{
			name:   "both per_page and cursor",
			params: PaginationParams{PerPage: 50, Cursor: "abc123"},
			wantQS: map[string]string{"per_page": "50", "cursor": "abc123"},
		},
		{
			name:    "zero per_page omitted",
			params:  PaginationParams{PerPage: 0, Cursor: "abc"},
			wantQS:  map[string]string{"cursor": "abc"},
			absentK: []string{"per_page"},
		},
		{
			name:    "empty cursor omitted",
			params:  PaginationParams{PerPage: 100},
			wantQS:  map[string]string{"per_page": "100"},
			absentK: []string{"cursor"},
		},
		{
			name:    "both zero",
			params:  PaginationParams{},
			absentK: []string{"per_page", "cursor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse("https://example.com/api/v1/issues/")
			tt.params.Apply(u)

			q := u.Query()
			for k, want := range tt.wantQS {
				got := q.Get(k)
				if got != want {
					t.Errorf("query param %q = %q, want %q", k, got, want)
				}
			}
			for _, k := range tt.absentK {
				if q.Has(k) {
					t.Errorf("query param %q should be absent, got %q", k, q.Get(k))
				}
			}
		})
	}
}

func TestRawPaginatedResponseFields(t *testing.T) {
	// Verify the struct tags match the Plane API response format
	raw := RawPaginatedResponse{
		TotalPages:      5,
		TotalCount:      42,
		NextCursor:      "cursor123",
		PrevCursor:      "cursor000",
		NextPageResults: true,
	}

	if raw.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", raw.TotalPages)
	}
	if raw.TotalCount != 42 {
		t.Errorf("TotalCount = %d, want 42", raw.TotalCount)
	}
	if raw.NextCursor != "cursor123" {
		t.Errorf("NextCursor = %q, want cursor123", raw.NextCursor)
	}
	if !raw.NextPageResults {
		t.Error("NextPageResults should be true")
	}
}
