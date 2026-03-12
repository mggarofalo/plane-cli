package models

import (
	"io"

	"github.com/mggarofalo/plane-cli/internal/output"
)

// Member represents a workspace or project member.
// The API returns a flat user-like object with a role field.
type Member struct {
	ID          string `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Avatar      string `json:"avatar"`
	DisplayName string `json:"display_name"`
	Role        int    `json:"role"`
}

type MemberList struct {
	Results []Member `json:"results"`
}

func RoleName(role int) string {
	switch role {
	case 5:
		return "Guest"
	case 10:
		return "Viewer"
	case 15:
		return "Member"
	case 20:
		return "Admin"
	default:
		return ""
	}
}

func (ml MemberList) RenderTable(w io.Writer) error {
	headers := []string{"ID", "Name", "Email", "Role"}
	rows := make([][]string, len(ml.Results))
	for i, m := range ml.Results {
		rows[i] = []string{m.ID, m.DisplayName, m.Email, RoleName(m.Role)}
	}
	output.WriteTable(w, headers, rows)
	return nil
}
