package models

import (
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

// Label represents a project label.
type Label struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description,omitempty"`
	ParentID    string    `json:"parent,omitempty"`
	ProjectID   string    `json:"project"`
	WorkspaceID string    `json:"workspace"`
	SortOrder   float64   `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LabelCreate holds fields for creating a label.
type LabelCreate struct {
	Name        string `json:"name"`
	Color       string `json:"color,omitempty"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent,omitempty"`
}

// LabelUpdate holds fields for updating a label.
type LabelUpdate struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
	ParentID    *string `json:"parent,omitempty"`
}

// LabelList wraps a slice of labels for table rendering.
type LabelList struct {
	Results []Label `json:"results"`
}

func (ll LabelList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Name", "Color", "Parent"}
	rows := make([][]string, len(ll.Results))
	for i, l := range ll.Results {
		rows[i] = []string{l.ID, l.Name, l.Color, l.ParentID}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
