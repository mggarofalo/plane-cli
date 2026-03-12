package models

import (
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

type Activity struct {
	ID          string    `json:"id"`
	Verb        string    `json:"verb"`
	Field       string    `json:"field,omitempty"`
	OldValue    string    `json:"old_value,omitempty"`
	NewValue    string    `json:"new_value,omitempty"`
	Comment     string    `json:"comment,omitempty"`
	IssueID     string    `json:"issue,omitempty"`
	ProjectID   string    `json:"project"`
	WorkspaceID string    `json:"workspace"`
	ActorDetail *User     `json:"actor_detail,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type ActivityList struct {
	Results []Activity `json:"results"`
}

func (al ActivityList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Verb", "Field", "Old", "New", "Time"}
	rows := make([][]string, len(al.Results))
	for i, a := range al.Results {
		rows[i] = []string{
			a.ID,
			a.Verb,
			a.Field,
			truncate(a.OldValue, 30),
			truncate(a.NewValue, 30),
			a.CreatedAt.Format(time.DateTime),
		}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
