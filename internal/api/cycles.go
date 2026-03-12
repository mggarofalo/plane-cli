package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type CyclesService struct {
	Client *Client
}

func NewCyclesService(client *Client) *CyclesService {
	return &CyclesService{Client: client}
}

func (s *CyclesService) baseURL(projectID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "cycles")
}

func (s *CyclesService) List(ctx context.Context, projectID string) ([]models.Cycle, error) {
	body, err := s.Client.Get(ctx, s.baseURL(projectID))
	if err != nil {
		return nil, fmt.Errorf("listing cycles: %w", err)
	}

	var resp models.PaginatedResponse[models.Cycle]
	if err := json.Unmarshal(body, &resp); err != nil {
		var cycles []models.Cycle
		if err2 := json.Unmarshal(body, &cycles); err2 != nil {
			return nil, fmt.Errorf("parsing cycles response: %w", err)
		}
		return cycles, nil
	}
	return resp.Results, nil
}

func (s *CyclesService) Get(ctx context.Context, projectID, cycleID string) (*models.Cycle, error) {
	url := s.baseURL(projectID) + cycleID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting cycle: %w", err)
	}

	var cycle models.Cycle
	if err := json.Unmarshal(body, &cycle); err != nil {
		return nil, fmt.Errorf("parsing cycle response: %w", err)
	}
	return &cycle, nil
}

func (s *CyclesService) Create(ctx context.Context, projectID string, input models.CycleCreate) (*models.Cycle, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID), input)
	if err != nil {
		return nil, fmt.Errorf("creating cycle: %w", err)
	}

	var cycle models.Cycle
	if err := json.Unmarshal(body, &cycle); err != nil {
		return nil, fmt.Errorf("parsing cycle response: %w", err)
	}
	return &cycle, nil
}

func (s *CyclesService) Update(ctx context.Context, projectID, cycleID string, input models.CycleUpdate) (*models.Cycle, error) {
	url := s.baseURL(projectID) + cycleID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating cycle: %w", err)
	}

	var cycle models.Cycle
	if err := json.Unmarshal(body, &cycle); err != nil {
		return nil, fmt.Errorf("parsing cycle response: %w", err)
	}
	return &cycle, nil
}

func (s *CyclesService) Delete(ctx context.Context, projectID, cycleID string) error {
	url := s.baseURL(projectID) + cycleID + "/"
	return s.Client.Delete(ctx, url)
}

func (s *CyclesService) Archive(ctx context.Context, projectID, cycleID string) error {
	url := s.baseURL(projectID) + cycleID + "/archive/"
	_, err := s.Client.Post(ctx, url, nil)
	return err
}

func (s *CyclesService) Unarchive(ctx context.Context, projectID, cycleID string) error {
	url := s.baseURL(projectID) + cycleID + "/archive/"
	return s.Client.Delete(ctx, url)
}

func (s *CyclesService) AddIssues(ctx context.Context, projectID, cycleID string, issueIDs []string) error {
	url := s.baseURL(projectID) + cycleID + "/cycle-issues/"
	payload := map[string][]string{"issues": issueIDs}
	_, err := s.Client.Post(ctx, url, payload)
	return err
}

func (s *CyclesService) ListIssues(ctx context.Context, projectID, cycleID string, params PaginationParams) (*models.PaginatedResponse[models.Issue], error) {
	url := s.baseURL(projectID) + cycleID + "/cycle-issues/"
	body, err := s.Client.GetPaginated(ctx, url, params)
	if err != nil {
		return nil, fmt.Errorf("listing cycle issues: %w", err)
	}

	var resp models.PaginatedResponse[models.Issue]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing cycle issues response: %w", err)
	}
	return &resp, nil
}

func (s *CyclesService) RemoveIssue(ctx context.Context, projectID, cycleID, issueID string) error {
	url := s.baseURL(projectID) + cycleID + "/cycle-issues/" + issueID + "/"
	return s.Client.Delete(ctx, url)
}

func (s *CyclesService) Transfer(ctx context.Context, projectID, cycleID, newCycleID string) error {
	url := s.baseURL(projectID) + cycleID + "/transfer-issues/"
	payload := map[string]string{"new_cycle_id": newCycleID}
	_, err := s.Client.Post(ctx, url, payload)
	return err
}
