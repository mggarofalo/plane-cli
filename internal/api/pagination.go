package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// PaginationParams holds pagination query parameters.
type PaginationParams struct {
	PerPage int
	Cursor  string
}

// Apply adds pagination query parameters to a URL.
func (p PaginationParams) Apply(u *url.URL) {
	q := u.Query()
	if p.PerPage > 0 {
		q.Set("per_page", fmt.Sprintf("%d", p.PerPage))
	}
	if p.Cursor != "" {
		q.Set("cursor", p.Cursor)
	}
	u.RawQuery = q.Encode()
}

// RawPaginatedResponse is used for initial unmarshalling before we know T.
type RawPaginatedResponse struct {
	GroupedResults json.RawMessage `json:"grouped_results"`
	Results        json.RawMessage `json:"results"`
	TotalPages     int             `json:"total_pages"`
	TotalCount     int             `json:"total_count"`
	NextCursor     string          `json:"next_cursor"`
	PrevCursor     string          `json:"prev_cursor"`
	NextPageResults bool           `json:"next_page_results"`
	ExtraStats     json.RawMessage `json:"extra_stats"`
}

// AutoPaginate fetches all pages from a paginated endpoint.
// makeURL should return the URL for a given cursor (empty string for first page).
func AutoPaginate[T any](ctx context.Context, client *Client, makeURL func(cursor string) (string, error)) ([]T, error) {
	var allResults []T
	cursor := ""

	for {
		rawURL, err := makeURL(cursor)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		resp, err := client.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		if resp.StatusCode >= 400 {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Status:     resp.Status,
				Body:       string(body),
				URL:        rawURL,
			}
		}

		var raw RawPaginatedResponse
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, fmt.Errorf("parsing pagination envelope: %w", err)
		}

		var page []T
		if raw.Results != nil {
			if err := json.Unmarshal(raw.Results, &page); err != nil {
				return nil, fmt.Errorf("parsing results: %w", err)
			}
		}

		allResults = append(allResults, page...)

		if !raw.NextPageResults || raw.NextCursor == "" {
			break
		}
		cursor = raw.NextCursor
	}

	return allResults, nil
}
