package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type StickiesService struct {
	Client *Client
}

func NewStickiesService(client *Client) *StickiesService {
	return &StickiesService{Client: client}
}

func (s *StickiesService) baseURL() string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "stickies")
}

func (s *StickiesService) List(ctx context.Context, params PaginationParams) (*models.PaginatedResponse[models.Sticky], error) {
	body, err := s.Client.GetPaginated(ctx, s.baseURL(), params)
	if err != nil {
		return nil, fmt.Errorf("listing stickies: %w", err)
	}

	var resp models.PaginatedResponse[models.Sticky]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing stickies response: %w", err)
	}
	return &resp, nil
}

func (s *StickiesService) Get(ctx context.Context, stickyID string) (*models.Sticky, error) {
	url := s.baseURL() + stickyID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting sticky: %w", err)
	}

	var sticky models.Sticky
	if err := json.Unmarshal(body, &sticky); err != nil {
		return nil, fmt.Errorf("parsing sticky response: %w", err)
	}
	return &sticky, nil
}

func (s *StickiesService) Create(ctx context.Context, input models.StickyCreate) (*models.Sticky, error) {
	body, err := s.Client.Post(ctx, s.baseURL(), input)
	if err != nil {
		return nil, fmt.Errorf("creating sticky: %w", err)
	}

	var sticky models.Sticky
	if err := json.Unmarshal(body, &sticky); err != nil {
		return nil, fmt.Errorf("parsing sticky response: %w", err)
	}
	return &sticky, nil
}

func (s *StickiesService) Update(ctx context.Context, stickyID string, input models.StickyUpdate) (*models.Sticky, error) {
	url := s.baseURL() + stickyID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating sticky: %w", err)
	}

	var sticky models.Sticky
	if err := json.Unmarshal(body, &sticky); err != nil {
		return nil, fmt.Errorf("parsing sticky response: %w", err)
	}
	return &sticky, nil
}

func (s *StickiesService) Delete(ctx context.Context, stickyID string) error {
	url := s.baseURL() + stickyID + "/"
	return s.Client.Delete(ctx, url)
}
