package models

import (
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

type Sticky struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Color       string    `json:"color,omitempty"`
	WorkspaceID string    `json:"workspace"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
}

type StickyCreate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

type StickyUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type StickyList struct {
	Results []Sticky `json:"results"`
}

func (sl StickyList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Name", "Color", "Created"}
	rows := make([][]string, len(sl.Results))
	for i, s := range sl.Results {
		rows[i] = []string{s.ID, truncate(s.Name, 40), s.Color, s.CreatedAt.Format(time.DateOnly)}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
