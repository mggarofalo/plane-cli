package models

import (
	"fmt"
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

// Module represents a project module.
type Module struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	StartDate   string    `json:"start_date,omitempty"`
	TargetDate  string    `json:"target_date,omitempty"`
	Status      string    `json:"status,omitempty"`
	LeadID      string    `json:"lead,omitempty"`
	MemberIDs   []string  `json:"members,omitempty"`
	ProjectID   string    `json:"project"`
	WorkspaceID string    `json:"workspace"`
	IsArchived  bool      `json:"is_archived"`
	SortOrder   float64   `json:"sort_order"`
	TotalIssues int       `json:"total_issues"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ModuleCreate struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	StartDate   string   `json:"start_date,omitempty"`
	TargetDate  string   `json:"target_date,omitempty"`
	Status      string   `json:"status,omitempty"`
	LeadID      string   `json:"lead,omitempty"`
	MemberIDs   []string `json:"members,omitempty"`
	ProjectID   string   `json:"project_id"`
}

type ModuleUpdate struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	StartDate   *string  `json:"start_date,omitempty"`
	TargetDate  *string  `json:"target_date,omitempty"`
	Status      *string  `json:"status,omitempty"`
	LeadID      *string  `json:"lead,omitempty"`
	MemberIDs   []string `json:"members,omitempty"`
}

type ModuleList struct {
	Results []Module `json:"results"`
}

func (ml ModuleList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Name", "Status", "Start", "Target", "Issues"}
	rows := make([][]string, len(ml.Results))
	for i, m := range ml.Results {
		rows[i] = []string{m.ID, m.Name, m.Status, m.StartDate, m.TargetDate, fmt.Sprintf("%d", m.TotalIssues)}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
