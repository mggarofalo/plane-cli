package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type UsersService struct {
	Client *Client
}

func NewUsersService(client *Client) *UsersService {
	return &UsersService{Client: client}
}

// Me returns the authenticated user.
func (s *UsersService) Me(ctx context.Context) (*models.User, error) {
	url := s.Client.URL("v1", "users", "me")
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting current user: %w", err)
	}

	var user models.User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing user response: %w", err)
	}
	return &user, nil
}
