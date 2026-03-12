package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type StatesService struct {
	Client *Client
}

func NewStatesService(client *Client) *StatesService {
	return &StatesService{Client: client}
}

func (s *StatesService) baseURL(projectID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "states")
}

func (s *StatesService) List(ctx context.Context, projectID string) ([]models.State, error) {
	body, err := s.Client.Get(ctx, s.baseURL(projectID))
	if err != nil {
		return nil, fmt.Errorf("listing states: %w", err)
	}

	var resp models.PaginatedResponse[models.State]
	if err := json.Unmarshal(body, &resp); err != nil {
		// States endpoint might return a flat array
		var states []models.State
		if err2 := json.Unmarshal(body, &states); err2 != nil {
			return nil, fmt.Errorf("parsing states response: %w", err)
		}
		return states, nil
	}
	return resp.Results, nil
}

func (s *StatesService) Get(ctx context.Context, projectID, stateID string) (*models.State, error) {
	url := s.baseURL(projectID) + stateID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting state: %w", err)
	}

	var state models.State
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, fmt.Errorf("parsing state response: %w", err)
	}
	return &state, nil
}

func (s *StatesService) Create(ctx context.Context, projectID string, input models.StateCreate) (*models.State, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID), input)
	if err != nil {
		return nil, fmt.Errorf("creating state: %w", err)
	}

	var state models.State
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, fmt.Errorf("parsing state response: %w", err)
	}
	return &state, nil
}

func (s *StatesService) Update(ctx context.Context, projectID, stateID string, input models.StateUpdate) (*models.State, error) {
	url := s.baseURL(projectID) + stateID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating state: %w", err)
	}

	var state models.State
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, fmt.Errorf("parsing state response: %w", err)
	}
	return &state, nil
}

func (s *StatesService) Delete(ctx context.Context, projectID, stateID string) error {
	url := s.baseURL(projectID) + stateID + "/"
	return s.Client.Delete(ctx, url)
}
