package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type ModulesService struct {
	Client *Client
}

func NewModulesService(client *Client) *ModulesService {
	return &ModulesService{Client: client}
}

func (s *ModulesService) baseURL(projectID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "modules")
}

func (s *ModulesService) List(ctx context.Context, projectID string) ([]models.Module, error) {
	body, err := s.Client.Get(ctx, s.baseURL(projectID))
	if err != nil {
		return nil, fmt.Errorf("listing modules: %w", err)
	}

	var resp models.PaginatedResponse[models.Module]
	if err := json.Unmarshal(body, &resp); err != nil {
		var modules []models.Module
		if err2 := json.Unmarshal(body, &modules); err2 != nil {
			return nil, fmt.Errorf("parsing modules response: %w", err)
		}
		return modules, nil
	}
	return resp.Results, nil
}

func (s *ModulesService) Get(ctx context.Context, projectID, moduleID string) (*models.Module, error) {
	url := s.baseURL(projectID) + moduleID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting module: %w", err)
	}

	var mod models.Module
	if err := json.Unmarshal(body, &mod); err != nil {
		return nil, fmt.Errorf("parsing module response: %w", err)
	}
	return &mod, nil
}

func (s *ModulesService) Create(ctx context.Context, projectID string, input models.ModuleCreate) (*models.Module, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID), input)
	if err != nil {
		return nil, fmt.Errorf("creating module: %w", err)
	}

	var mod models.Module
	if err := json.Unmarshal(body, &mod); err != nil {
		return nil, fmt.Errorf("parsing module response: %w", err)
	}
	return &mod, nil
}

func (s *ModulesService) Update(ctx context.Context, projectID, moduleID string, input models.ModuleUpdate) (*models.Module, error) {
	url := s.baseURL(projectID) + moduleID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating module: %w", err)
	}

	var mod models.Module
	if err := json.Unmarshal(body, &mod); err != nil {
		return nil, fmt.Errorf("parsing module response: %w", err)
	}
	return &mod, nil
}

func (s *ModulesService) Delete(ctx context.Context, projectID, moduleID string) error {
	url := s.baseURL(projectID) + moduleID + "/"
	return s.Client.Delete(ctx, url)
}

func (s *ModulesService) Archive(ctx context.Context, projectID, moduleID string) error {
	url := s.baseURL(projectID) + moduleID + "/archive/"
	_, err := s.Client.Post(ctx, url, nil)
	return err
}

func (s *ModulesService) Unarchive(ctx context.Context, projectID, moduleID string) error {
	url := s.baseURL(projectID) + moduleID + "/archive/"
	return s.Client.Delete(ctx, url)
}

func (s *ModulesService) AddIssues(ctx context.Context, projectID, moduleID string, issueIDs []string) error {
	url := s.baseURL(projectID) + moduleID + "/module-issues/"
	payload := map[string][]string{"issues": issueIDs}
	_, err := s.Client.Post(ctx, url, payload)
	return err
}

func (s *ModulesService) ListIssues(ctx context.Context, projectID, moduleID string, params PaginationParams) (*models.PaginatedResponse[models.Issue], error) {
	url := s.baseURL(projectID) + moduleID + "/module-issues/"
	body, err := s.Client.GetPaginated(ctx, url, params)
	if err != nil {
		return nil, fmt.Errorf("listing module issues: %w", err)
	}

	var resp models.PaginatedResponse[models.Issue]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing module issues response: %w", err)
	}
	return &resp, nil
}

func (s *ModulesService) RemoveIssue(ctx context.Context, projectID, moduleID, issueID string) error {
	url := s.baseURL(projectID) + moduleID + "/module-issues/" + issueID + "/"
	return s.Client.Delete(ctx, url)
}
