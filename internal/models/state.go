package models

import (
	"fmt"
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

// State represents a workflow state in a project.
type State struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Group       string    `json:"group"` // backlog, unstarted, started, completed, cancelled
	Description string    `json:"description,omitempty"`
	Sequence    float64   `json:"sequence"`
	IsDefault   bool      `json:"is_default"`
	ProjectID   string    `json:"project"`
	WorkspaceID string    `json:"workspace"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StateCreate holds fields for creating a state.
type StateCreate struct {
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	Group       string  `json:"group"`
	Description string  `json:"description,omitempty"`
	Sequence    float64 `json:"sequence,omitempty"`
}

// StateUpdate holds fields for updating a state.
type StateUpdate struct {
	Name        *string  `json:"name,omitempty"`
	Color       *string  `json:"color,omitempty"`
	Group       *string  `json:"group,omitempty"`
	Description *string  `json:"description,omitempty"`
	Sequence    *float64 `json:"sequence,omitempty"`
}

// StateList wraps a slice of states for table rendering.
type StateList struct {
	Results []State `json:"results"`
}

func (sl StateList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Name", "Group", "Color", "Default"}
	rows := make([][]string, len(sl.Results))
	for i, s := range sl.Results {
		rows[i] = []string{
			s.ID,
			s.Name,
			s.Group,
			s.Color,
			fmt.Sprintf("%t", s.IsDefault),
		}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
