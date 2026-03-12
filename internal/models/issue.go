package models

import (
	"fmt"
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

// Issue represents a work item in Plane.
type Issue struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	DescriptionHTML  string    `json:"description_html,omitempty"`
	Priority         string    `json:"priority"` // urgent, high, medium, low, none
	StartDate        string    `json:"start_date,omitempty"`
	TargetDate       string    `json:"target_date,omitempty"`
	SequenceID       int       `json:"sequence_id"`
	SortOrder        float64   `json:"sort_order"`
	StateID          string    `json:"state"`
	StateDetail      *State    `json:"state_detail,omitempty"`
	ProjectID        string    `json:"project"`
	ProjectDetail    *Project  `json:"project_detail,omitempty"`
	WorkspaceID      string    `json:"workspace"`
	ParentID         string    `json:"parent,omitempty"`
	AssigneeIDs      []string  `json:"assignees"`
	LabelIDs         []string  `json:"labels"`
	CycleID          string    `json:"cycle,omitempty"`
	ModuleIDs        []string  `json:"modules,omitempty"`
	CompletedAt      string    `json:"completed_at,omitempty"`
	ArchivedAt       string    `json:"archived_at,omitempty"`
	IsDraft          bool      `json:"is_draft"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedBy        string    `json:"created_by"`
	UpdatedBy        string    `json:"updated_by"`
	EstimatePoint    *int      `json:"estimate_point,omitempty"`
	SubIssuesCount   int       `json:"sub_issues_count"`
	LinkCount        int       `json:"link_count"`
	AttachmentCount  int       `json:"attachment_count"`
}

// IssueCreate holds fields for creating an issue.
type IssueCreate struct {
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	DescriptionHTML string   `json:"description_html,omitempty"`
	Priority        string   `json:"priority,omitempty"`
	StateID         string   `json:"state,omitempty"`
	ParentID        string   `json:"parent,omitempty"`
	AssigneeIDs     []string `json:"assignees,omitempty"`
	LabelIDs        []string `json:"labels,omitempty"`
	StartDate       string   `json:"start_date,omitempty"`
	TargetDate      string   `json:"target_date,omitempty"`
	EstimatePoint   *int     `json:"estimate_point,omitempty"`
}

// IssueUpdate holds fields for updating an issue.
type IssueUpdate struct {
	Name            *string  `json:"name,omitempty"`
	Description     *string  `json:"description,omitempty"`
	DescriptionHTML *string  `json:"description_html,omitempty"`
	Priority        *string  `json:"priority,omitempty"`
	StateID         *string  `json:"state,omitempty"`
	ParentID        *string  `json:"parent,omitempty"`
	AssigneeIDs     []string `json:"assignees,omitempty"`
	LabelIDs        []string `json:"labels,omitempty"`
	StartDate       *string  `json:"start_date,omitempty"`
	TargetDate      *string  `json:"target_date,omitempty"`
	EstimatePoint   *int     `json:"estimate_point,omitempty"`
}

// IssueList wraps a slice of issues for table rendering.
type IssueList struct {
	Results        []Issue `json:"results"`
	TotalCount     int     `json:"total_count,omitempty"`
	NextCursor     string  `json:"next_cursor,omitempty"`
	PrevCursor     string  `json:"prev_cursor,omitempty"`
	NextPageResults bool   `json:"next_page_results,omitempty"`
}

func (il IssueList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Seq", "Name", "Priority", "State"}
	rows := make([][]string, len(il.Results))
	for i, issue := range il.Results {
		state := issue.StateID
		if issue.StateDetail != nil {
			state = issue.StateDetail.Name
		}
		rows[i] = []string{
			issue.ID,
			fmt.Sprintf("%d", issue.SequenceID),
			truncate(issue.Name, 60),
			issue.Priority,
			state,
		}
	}
	output.WriteTable(w, headers, rows)
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
