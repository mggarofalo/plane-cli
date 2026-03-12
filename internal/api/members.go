package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type MembersService struct {
	Client *Client
}

func NewMembersService(client *Client) *MembersService {
	return &MembersService{Client: client}
}

func (s *MembersService) ListWorkspace(ctx context.Context) ([]models.Member, error) {
	url := s.Client.URL("v1", "workspaces", s.Client.Workspace, "members")
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("listing workspace members: %w", err)
	}

	var members []models.Member
	if err := json.Unmarshal(body, &members); err != nil {
		return nil, fmt.Errorf("parsing members response: %w", err)
	}
	return members, nil
}

func (s *MembersService) ListProject(ctx context.Context, projectID string) ([]models.Member, error) {
	url := s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "members")
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("listing project members: %w", err)
	}

	var members []models.Member
	if err := json.Unmarshal(body, &members); err != nil {
		return nil, fmt.Errorf("parsing members response: %w", err)
	}
	return members, nil
}

func (s *MembersService) Add(ctx context.Context, projectID string, memberID string, role int) error {
	url := s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "members")
	payload := map[string]any{"member_id": memberID, "role": role}
	_, err := s.Client.Post(ctx, url, payload)
	return err
}

func (s *MembersService) Remove(ctx context.Context, projectID, memberID string) error {
	url := s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "members", memberID)
	return s.Client.Delete(ctx, url)
}
