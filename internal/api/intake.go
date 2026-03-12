package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type IntakeService struct {
	Client *Client
}

func NewIntakeService(client *Client) *IntakeService {
	return &IntakeService{Client: client}
}

func (s *IntakeService) baseURL(projectID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "intake-issues")
}

func (s *IntakeService) List(ctx context.Context, projectID string, params PaginationParams) (*models.PaginatedResponse[models.IntakeIssue], error) {
	body, err := s.Client.GetPaginated(ctx, s.baseURL(projectID), params)
	if err != nil {
		return nil, fmt.Errorf("listing intake issues: %w", err)
	}

	var resp models.PaginatedResponse[models.IntakeIssue]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing intake issues response: %w", err)
	}
	return &resp, nil
}

func (s *IntakeService) Get(ctx context.Context, projectID, intakeID string) (*models.IntakeIssue, error) {
	url := s.baseURL(projectID) + intakeID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting intake issue: %w", err)
	}

	var issue models.IntakeIssue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing intake issue response: %w", err)
	}
	return &issue, nil
}

func (s *IntakeService) Create(ctx context.Context, projectID string, input models.IntakeIssueCreate) (*models.IntakeIssue, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID), input)
	if err != nil {
		return nil, fmt.Errorf("creating intake issue: %w", err)
	}

	var issue models.IntakeIssue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing intake issue response: %w", err)
	}
	return &issue, nil
}

func (s *IntakeService) Update(ctx context.Context, projectID, intakeID string, input models.IntakeIssueUpdate) (*models.IntakeIssue, error) {
	url := s.baseURL(projectID) + intakeID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating intake issue: %w", err)
	}

	var issue models.IntakeIssue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing intake issue response: %w", err)
	}
	return &issue, nil
}

func (s *IntakeService) Delete(ctx context.Context, projectID, intakeID string) error {
	url := s.baseURL(projectID) + intakeID + "/"
	return s.Client.Delete(ctx, url)
}
