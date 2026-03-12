package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type LinksService struct {
	Client *Client
}

func NewLinksService(client *Client) *LinksService {
	return &LinksService{Client: client}
}

func (s *LinksService) baseURL(projectID, issueID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "work-items", issueID, "links")
}

func (s *LinksService) List(ctx context.Context, projectID, issueID string) ([]models.Link, error) {
	body, err := s.Client.Get(ctx, s.baseURL(projectID, issueID))
	if err != nil {
		return nil, fmt.Errorf("listing links: %w", err)
	}

	var resp models.PaginatedResponse[models.Link]
	if err := json.Unmarshal(body, &resp); err != nil {
		var links []models.Link
		if err2 := json.Unmarshal(body, &links); err2 != nil {
			return nil, fmt.Errorf("parsing links response: %w", err)
		}
		return links, nil
	}
	return resp.Results, nil
}

func (s *LinksService) Get(ctx context.Context, projectID, issueID, linkID string) (*models.Link, error) {
	url := s.baseURL(projectID, issueID) + linkID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting link: %w", err)
	}

	var link models.Link
	if err := json.Unmarshal(body, &link); err != nil {
		return nil, fmt.Errorf("parsing link response: %w", err)
	}
	return &link, nil
}

func (s *LinksService) Create(ctx context.Context, projectID, issueID string, input models.LinkCreate) (*models.Link, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID, issueID), input)
	if err != nil {
		return nil, fmt.Errorf("creating link: %w", err)
	}

	var link models.Link
	if err := json.Unmarshal(body, &link); err != nil {
		return nil, fmt.Errorf("parsing link response: %w", err)
	}
	return &link, nil
}

func (s *LinksService) Update(ctx context.Context, projectID, issueID, linkID string, input models.LinkUpdate) (*models.Link, error) {
	url := s.baseURL(projectID, issueID) + linkID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating link: %w", err)
	}

	var link models.Link
	if err := json.Unmarshal(body, &link); err != nil {
		return nil, fmt.Errorf("parsing link response: %w", err)
	}
	return &link, nil
}

func (s *LinksService) Delete(ctx context.Context, projectID, issueID, linkID string) error {
	url := s.baseURL(projectID, issueID) + linkID + "/"
	return s.Client.Delete(ctx, url)
}
