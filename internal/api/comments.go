package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type CommentsService struct {
	Client *Client
}

func NewCommentsService(client *Client) *CommentsService {
	return &CommentsService{Client: client}
}

func (s *CommentsService) baseURL(projectID, issueID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "work-items", issueID, "comments")
}

func (s *CommentsService) List(ctx context.Context, projectID, issueID string) ([]models.Comment, error) {
	url := s.baseURL(projectID, issueID)
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("listing comments: %w", err)
	}

	var resp models.PaginatedResponse[models.Comment]
	if err := json.Unmarshal(body, &resp); err != nil {
		var comments []models.Comment
		if err2 := json.Unmarshal(body, &comments); err2 != nil {
			return nil, fmt.Errorf("parsing comments response: %w", err)
		}
		return comments, nil
	}
	return resp.Results, nil
}

func (s *CommentsService) Get(ctx context.Context, projectID, issueID, commentID string) (*models.Comment, error) {
	url := s.baseURL(projectID, issueID) + commentID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting comment: %w", err)
	}

	var comment models.Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, fmt.Errorf("parsing comment response: %w", err)
	}
	return &comment, nil
}

func (s *CommentsService) Create(ctx context.Context, projectID, issueID string, input models.CommentCreate) (*models.Comment, error) {
	body, err := s.Client.Post(ctx, s.baseURL(projectID, issueID), input)
	if err != nil {
		return nil, fmt.Errorf("creating comment: %w", err)
	}

	var comment models.Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, fmt.Errorf("parsing comment response: %w", err)
	}
	return &comment, nil
}

func (s *CommentsService) Update(ctx context.Context, projectID, issueID, commentID string, input models.CommentUpdate) (*models.Comment, error) {
	url := s.baseURL(projectID, issueID) + commentID + "/"
	body, err := s.Client.Patch(ctx, url, input)
	if err != nil {
		return nil, fmt.Errorf("updating comment: %w", err)
	}

	var comment models.Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, fmt.Errorf("parsing comment response: %w", err)
	}
	return &comment, nil
}

func (s *CommentsService) Delete(ctx context.Context, projectID, issueID, commentID string) error {
	url := s.baseURL(projectID, issueID) + commentID + "/"
	return s.Client.Delete(ctx, url)
}
