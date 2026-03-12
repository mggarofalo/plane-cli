package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type LabelsService struct {
	Client *Client
}

func NewLabelsService(client *Client) *LabelsService {
	return &LabelsService{Client: client}
}

func (s *LabelsService) baseURL(projectID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "labels")
}

func (s *LabelsService) List(ctx context.Context, projectID string) ([]models.Label, error) {
	body, err := s.Client.Get(ctx, s.baseURL(projectID))
	if err != nil {
		return nil, fmt.Errorf("listing labels: %w", err)
	}

	var resp models.PaginatedResponse[models.Label]
	if err := json.Unmarshal(body, &resp); err != nil {
		var labels []models.Label
		if err2 := json.Unmarshal(body, &labels); err2 != nil {
			return nil, fmt.Errorf("parsing labels response: %w", err)
		}
		return labels, nil
	}
	return resp.Results, nil
}

func (s *LabelsService) Get(ctx context.Context, projectID, labelID string) (*models.Label, error) {
	url := s.baseURL(projectID) + labelID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting label: %w", err)
	}

	var label models.Label
	if err := json.Unmarshal(body, &label); err != nil {
		return nil, fmt.Errorf("parsing label response: %w", err)
	}
	return &label, nil
}

func (s *LabelsService) Create(ctx context.Context, projectID string, input models.LabelCreate) (*models.Label, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID), input)
	if err != nil {
		return nil, fmt.Errorf("creating label: %w", err)
	}

	var label models.Label
	if err := json.Unmarshal(body, &label); err != nil {
		return nil, fmt.Errorf("parsing label response: %w", err)
	}
	return &label, nil
}

func (s *LabelsService) Update(ctx context.Context, projectID, labelID string, input models.LabelUpdate) (*models.Label, error) {
	url := s.baseURL(projectID) + labelID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating label: %w", err)
	}

	var label models.Label
	if err := json.Unmarshal(body, &label); err != nil {
		return nil, fmt.Errorf("parsing label response: %w", err)
	}
	return &label, nil
}

func (s *LabelsService) Delete(ctx context.Context, projectID, labelID string) error {
	url := s.baseURL(projectID) + labelID + "/"
	return s.Client.Delete(ctx, url)
}
