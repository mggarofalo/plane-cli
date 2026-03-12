package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type PagesService struct {
	Client *Client
}

func NewPagesService(client *Client) *PagesService {
	return &PagesService{Client: client}
}

// Pages use internal API path: /api/workspaces/{slug}/projects/{id}/pages/
func (s *PagesService) baseURL(projectID string) string {
	return fmt.Sprintf("%s/api/workspaces/%s/projects/%s/pages/", s.Client.BaseURL, s.Client.Workspace, projectID)
}

func (s *PagesService) List(ctx context.Context, projectID string) ([]models.Page, error) {
	body, err := s.Client.Get(ctx, s.baseURL(projectID))
	if err != nil {
		return nil, fmt.Errorf("listing pages: %w (pages require session auth)", err)
	}

	var pages []models.Page
	if err := json.Unmarshal(body, &pages); err != nil {
		return nil, fmt.Errorf("parsing pages response: %w", err)
	}
	return pages, nil
}

func (s *PagesService) Get(ctx context.Context, projectID, pageID string) (*models.Page, error) {
	url := s.baseURL(projectID) + pageID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting page: %w", err)
	}

	var page models.Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing page response: %w", err)
	}
	return &page, nil
}

func (s *PagesService) Create(ctx context.Context, projectID string, input models.PageCreate) (*models.Page, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID), input)
	if err != nil {
		return nil, fmt.Errorf("creating page: %w", err)
	}

	var page models.Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing page response: %w", err)
	}
	return &page, nil
}

func (s *PagesService) Update(ctx context.Context, projectID, pageID string, input models.PageUpdate) (*models.Page, error) {
	url := s.baseURL(projectID) + pageID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating page: %w", err)
	}

	var page models.Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing page response: %w", err)
	}
	return &page, nil
}

func (s *PagesService) Delete(ctx context.Context, projectID, pageID string) error {
	url := s.baseURL(projectID) + pageID + "/"
	return s.Client.Delete(ctx, url)
}
