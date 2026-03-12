package models

import (
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

type Comment struct {
	ID          string    `json:"id"`
	Body        string    `json:"comment_stripped,omitempty"`
	BodyHTML    string    `json:"comment_html,omitempty"`
	ActorDetail *User     `json:"actor_detail,omitempty"`
	IssueID     string    `json:"issue,omitempty"`
	ProjectID   string    `json:"project"`
	WorkspaceID string    `json:"workspace"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedBy   string    `json:"created_by"`
}

type CommentCreate struct {
	CommentHTML string `json:"comment_html"`
}

type CommentUpdate struct {
	CommentHTML *string `json:"comment_html,omitempty"`
}

type CommentList struct {
	Results []Comment `json:"results"`
}

func (cl CommentList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Author", "Body", "Created"}
	rows := make([][]string, len(cl.Results))
	for i, c := range cl.Results {
		author := c.CreatedBy
		if c.ActorDetail != nil {
			author = c.ActorDetail.DisplayName
		}
		rows[i] = []string{c.ID, author, truncate(c.Body, 60), c.CreatedAt.Format(time.DateOnly)}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
