package models

import (
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

type Link struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	IssueID     string    `json:"issue"`
	ProjectID   string    `json:"project"`
	WorkspaceID string    `json:"workspace"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
}

type LinkCreate struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url"`
}

type LinkUpdate struct {
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`
}

type LinkList struct {
	Results []Link `json:"results"`
}

func (ll LinkList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Title", "URL", "Created"}
	rows := make([][]string, len(ll.Results))
	for i, l := range ll.Results {
		rows[i] = []string{l.ID, l.Title, l.URL, l.CreatedAt.Format(time.DateOnly)}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
