package models

import (
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

type IntakeIssue struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Source       string   `json:"source,omitempty"`
	Status      string    `json:"status"` // pending, accepted, declined, snoozed, duplicate
	ProjectID   string    `json:"project"`
	WorkspaceID string    `json:"workspace"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type IntakeIssueCreate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
}

type IntakeIssueUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

type IntakeIssueList struct {
	Results []IntakeIssue `json:"results"`
}

func (il IntakeIssueList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Name", "Status", "Source", "Created"}
	rows := make([][]string, len(il.Results))
	for i, item := range il.Results {
		rows[i] = []string{item.ID, truncate(item.Name, 40), item.Status, item.Source, item.CreatedAt.Format(time.DateOnly)}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
