package models

import (
	"io"
	"time"

	"github.com/mggarofalo/plane-cli/internal/output"
)

// User represents a Plane user.
type User struct {
	ID          string    `json:"id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Email       string    `json:"email"`
	Avatar      string    `json:"avatar"`
	DisplayName string    `json:"display_name"`
	IsActive    bool      `json:"is_active"`
	IsBot       bool      `json:"is_bot"`
	Role        int       `json:"role"`
	DateJoined  time.Time `json:"date_joined"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (u *User) RenderTable(w io.Writer) error {
	headers := []string{"Field", "Value"}
	rows := [][]string{
		{"ID", u.ID},
		{"Display Name", u.DisplayName},
		{"Email", u.Email},
		{"First Name", u.FirstName},
		{"Last Name", u.LastName},
		{"Active", boolStr(u.IsActive)},
		{"Date Joined", u.DateJoined.Format("2006-01-02")},
	}
	output.WriteTable(w, headers, rows)
	return nil
}

func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
