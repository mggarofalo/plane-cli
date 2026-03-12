package models

import (
	"fmt"
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

// Cycle represents a project cycle (sprint).
type Cycle struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	StartDate   string    `json:"start_date,omitempty"`
	EndDate     string    `json:"end_date,omitempty"`
	Status      string    `json:"status,omitempty"`
	OwnedBy     string    `json:"owned_by,omitempty"`
	ProjectID   string    `json:"project"`
	WorkspaceID string    `json:"workspace"`
	IsArchived  bool      `json:"is_archived"`
	SortOrder   float64   `json:"sort_order"`
	TotalIssues int       `json:"total_issues"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CycleCreate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	OwnedBy     string `json:"owned_by,omitempty"`
	ProjectID   string `json:"project_id"`
}

type CycleUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
	OwnedBy     *string `json:"owned_by,omitempty"`
}

type CycleList struct {
	Results []Cycle `json:"results"`
}

func (cl CycleList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Name", "Status", "Start", "End", "Issues"}
	rows := make([][]string, len(cl.Results))
	for i, c := range cl.Results {
		rows[i] = []string{c.ID, c.Name, c.Status, c.StartDate, c.EndDate, fmt.Sprintf("%d", c.TotalIssues)}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
