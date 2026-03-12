package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type ProjectsService struct {
	Client *Client
}

func NewProjectsService(client *Client) *ProjectsService {
	return &ProjectsService{Client: client}
}

func (s *ProjectsService) baseURL() string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects")
}

func (s *ProjectsService) List(ctx context.Context, params PaginationParams) (*models.PaginatedResponse[models.Project], error) {
	body, err := s.Client.GetPaginated(ctx, s.baseURL(), params)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	var resp models.PaginatedResponse[models.Project]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing projects response: %w", err)
	}
	return &resp, nil
}

func (s *ProjectsService) Get(ctx context.Context, projectID string) (*models.Project, error) {
	url := s.baseURL() + projectID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}

	var project models.Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("parsing project response: %w", err)
	}
	return &project, nil
}

func (s *ProjectsService) Create(ctx context.Context, input models.ProjectCreate) (*models.Project, error) {
	body, err := s.Client.Post(ctx, s.baseURL(), input)
	if err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	var project models.Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("parsing project response: %w", err)
	}
	return &project, nil
}

func (s *ProjectsService) Update(ctx context.Context, projectID string, input models.ProjectUpdate) (*models.Project, error) {
	url := s.baseURL() + projectID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating project: %w", err)
	}

	var project models.Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("parsing project response: %w", err)
	}
	return &project, nil
}

func (s *ProjectsService) Delete(ctx context.Context, projectID string) error {
	url := s.baseURL() + projectID + "/"
	return s.Client.Delete(ctx, url)
}

func (s *ProjectsService) Archive(ctx context.Context, projectID string) error {
	url := s.baseURL() + projectID + "/archive/"
	_, err := s.Client.Post(ctx, url, nil)
	return err
}

func (s *ProjectsService) Unarchive(ctx context.Context, projectID string) error {
	url := s.baseURL() + projectID + "/archive/"
	return s.Client.Delete(ctx, url)
}
