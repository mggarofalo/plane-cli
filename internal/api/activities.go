package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mggarofalo/plane-cli/internal/models"
)

type ActivitiesService struct {
	Client *Client
}

func NewActivitiesService(client *Client) *ActivitiesService {
	return &ActivitiesService{Client: client}
}

func (s *ActivitiesService) baseURL(projectID, issueID string) string {
	return s.Client.URL("v1", "workspaces", s.Client.Workspace, "projects", projectID, "work-items", issueID, "activities")
}

func (s *ActivitiesService) List(ctx context.Context, projectID, issueID string) ([]models.Activity, error) {
	body, err := s.Client.Get(ctx, s.baseURL(projectID, issueID))
	if err != nil {
		return nil, fmt.Errorf("listing activities: %w", err)
	}

	var resp models.PaginatedResponse[models.Activity]
	if err := json.Unmarshal(body, &resp); err != nil {
		var activities []models.Activity
		if err2 := json.Unmarshal(body, &activities); err2 != nil {
			return nil, fmt.Errorf("parsing activities response: %w", err)
		}
		return activities, nil
	}
	return resp.Results, nil
}

func (s *ActivitiesService) Get(ctx context.Context, projectID, issueID, activityID string) (*models.Activity, error) {
	url := s.baseURL(projectID, issueID) + activityID + "/"
	body, err := s.Client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting activity: %w", err)
	}

	var activity models.Activity
	if err := json.Unmarshal(body, &activity); err != nil {
		return nil, fmt.Errorf("parsing activity response: %w", err)
	}
	return &activity, nil
}
