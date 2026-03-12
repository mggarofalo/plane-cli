package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type IssuesService struct {
	Client *Client
}

func NewIssuesService(client *Client) *IssuesService {
	return &IssuesService{Client: client}
}

func (s *IssuesService) baseURL(projectID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "work-items")
}

func (s *IssuesService) List(ctx context.Context, projectID string, params PaginationParams) (*models.PaginatedResponse[models.Issue], error) {
	body, err := s.Client.GetPaginated(ctx, s.baseURL(projectID), params)
	if err != nil {
		return nil, fmt.Errorf("listing issues: %w", err)
	}

	var resp models.PaginatedResponse[models.Issue]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing issues response: %w", err)
	}
	return &resp, nil
}

func (s *IssuesService) ListAll(ctx context.Context, projectID string, perPage int) ([]models.Issue, error) {
	if perPage <= 0 {
		perPage = 100
	}
	return AutoPaginate[models.Issue](ctx, s.Client, func(cursor string) (string, error) {
		u, err := url.Parse(s.baseURL(projectID))
		if err != nil {
			return "", err
		}
		p := PaginationParams{PerPage: perPage, Cursor: cursor}
		p.Apply(u)
		return u.String(), nil
	})
}

func (s *IssuesService) Get(ctx context.Context, projectID, issueID string) (*models.Issue, error) {
	reqURL := s.baseURL(projectID) + issueID + "/"
	body, err := s.Client.Get(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}

	var issue models.Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue response: %w", err)
	}
	return &issue, nil
}

// GetBySequence retrieves an issue by its sequence identifier (e.g., "PLANECLI-123").
func (s *IssuesService) GetBySequence(ctx context.Context, projectID, identifier string) (*models.Issue, error) {
	// Parse "PROJ-123" format
	parts := strings.SplitN(identifier, "-", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid sequence identifier %q: expected format PROJ-123", identifier)
	}

	reqURL := s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "work-items") + "?sequence_id=" + parts[1]
	body, err := s.Client.Get(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("getting issue by sequence: %w", err)
	}

	var resp models.PaginatedResponse[models.Issue]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing issue response: %w", err)
	}

	if len(resp.Results) == 0 {
		return nil, &APIError{StatusCode: 404, Status: "Not Found", URL: reqURL, Body: fmt.Sprintf("issue %s not found", identifier)}
	}
	return &resp.Results[0], nil
}

func (s *IssuesService) Create(ctx context.Context, projectID string, input models.IssueCreate) (*models.Issue, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID), input)
	if err != nil {
		return nil, fmt.Errorf("creating issue: %w", err)
	}

	var issue models.Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue response: %w", err)
	}
	return &issue, nil
}

func (s *IssuesService) Update(ctx context.Context, projectID, issueID string, input models.IssueUpdate) (*models.Issue, error) {
	reqURL := s.baseURL(projectID) + issueID + "/"
	body, err := s.Client.Patch(ctx, reqURL, input)
	if err != nil {
		return nil, fmt.Errorf("updating issue: %w", err)
	}

	var issue models.Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing issue response: %w", err)
	}
	return &issue, nil
}

func (s *IssuesService) Delete(ctx context.Context, projectID, issueID string) error {
	reqURL := s.baseURL(projectID) + issueID + "/"
	return s.Client.Delete(ctx, reqURL)
}

// Search searches for issues by query string using the list endpoint's search parameter.
func (s *IssuesService) Search(ctx context.Context, projectID, query string) ([]models.Issue, error) {
	reqURL := s.baseURL(projectID) + "?search=" + url.QueryEscape(query)
	body, err := s.Client.Get(ctx, reqURL)
	if err != nil {
		return nil, fmt.Errorf("searching issues: %w", err)
	}

	var resp models.PaginatedResponse[models.Issue]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}
	return resp.Results, nil
}
