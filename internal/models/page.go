package models

import (
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

type Page struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description_stripped,omitempty"`
	BodyHTML     string    `json:"description_html,omitempty"`
	OwnedBy      string    `json:"owned_by"`
	IsLocked     bool      `json:"is_locked"`
	IsArchived   bool      `json:"archived_at,omitempty"` // presence indicates archived
	ProjectID    string    `json:"project"`
	WorkspaceID  string    `json:"workspace"`
	Access       int       `json:"access"` // 0=public, 1=private
	Color        string    `json:"color,omitempty"`
	Labels       []string  `json:"labels,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PageCreate struct {
	Name        string   `json:"name"`
	Description string   `json:"description_html,omitempty"`
	Access      *int     `json:"access,omitempty"`
	Color       string   `json:"color,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

type PageUpdate struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description_html,omitempty"`
	Access      *int     `json:"access,omitempty"`
	Color       *string  `json:"color,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

type PageList struct {
	Results []Page `json:"results"`
}

func (pl PageList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Name", "Owner", "Access", "Created"}
	rows := make([][]string, len(pl.Results))
	for i, p := range pl.Results {
		access := "public"
		if p.Access == 1 {
			access = "private"
		}
		rows[i] = []string{p.ID, truncate(p.Name, 40), p.OwnedBy, access, p.CreatedAt.Format(time.DateOnly)}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
