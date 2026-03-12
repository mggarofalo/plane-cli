package models

import (
	"fmt"
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

// Project represents a Plane project.
type Project struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	DescriptionHTML string    `json:"description_html,omitempty"`
	Identifier      string    `json:"identifier"`
	Network         int       `json:"network"`
	Emoji           string    `json:"emoji,omitempty"`
	IconProp        any       `json:"icon_prop,omitempty"`
	CoverImage      string    `json:"cover_image,omitempty"`
	Archive         bool      `json:"is_archived"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedBy       string    `json:"created_by"`
	UpdatedBy       string    `json:"updated_by"`
	Workspace       string    `json:"workspace"`
	WorkspaceDetail any       `json:"workspace_detail,omitempty"`
	DefaultAssignee string    `json:"default_assignee,omitempty"`
	ProjectLead     string    `json:"project_lead,omitempty"`
	EstimateID      string    `json:"estimate,omitempty"`
	DefaultStateID  string    `json:"default_state,omitempty"`
	TotalMembers    int       `json:"total_members"`
	TotalCycles     int       `json:"total_cycles"`
	TotalModules    int       `json:"total_modules"`
	IsMember        bool      `json:"is_member"`
	MemberRole      int       `json:"member_role,omitempty"`
}

// ProjectCreate holds fields for creating a project.
type ProjectCreate struct {
	Name        string `json:"name"`
	Identifier  string `json:"identifier,omitempty"`
	Description string `json:"description,omitempty"`
	Network     *int   `json:"network,omitempty"`
	Emoji       string `json:"emoji,omitempty"`
}

// ProjectUpdate holds fields for updating a project.
type ProjectUpdate struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Network     *int    `json:"network,omitempty"`
	Emoji       *string `json:"emoji,omitempty"`
}

// ProjectList wraps a slice of projects for table rendering.
type ProjectList struct {
	Results    []Project `json:"results"`
	TotalCount int       `json:"total_count,omitempty"`
}

func (pl ProjectList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Identifier", "Name", "Members", "Archived"}
	rows := make([][]string, len(pl.Results))
	for i, p := range pl.Results {
		rows[i] = []string{
			p.ID,
			p.Identifier,
			p.Name,
			fmt.Sprintf("%d", p.TotalMembers),
			fmt.Sprintf("%t", p.Archive),
		}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
