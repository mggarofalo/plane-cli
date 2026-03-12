package models

// PaginatedResponse is the generic envelope for paginated API responses.
type PaginatedResponse[T any] struct {
	Results        []T    `json:"results"`
	TotalPages     int    `json:"total_pages"`
	TotalCount     int    `json:"total_count"`
	NextCursor     string `json:"next_cursor"`
	PrevCursor     string `json:"prev_cursor"`
	NextPageResults bool  `json:"next_page_results"`
}
