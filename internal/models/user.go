package models

import "time"

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
