package cmdgen

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
)

// sequenceIDRe matches identifiers like "PROJ-123" or "RECEIPTS-351".
var sequenceIDRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*-\d+$`)

// IsSequenceID returns true if s looks like a Plane sequence ID (e.g. "PROJ-42").
func IsSequenceID(s string) bool {
	return sequenceIDRe.MatchString(s)
}

// ResolveSequenceID resolves a sequence ID like "PROJ-42" to a full UUID
// by calling the work-items lookup endpoint.
func ResolveSequenceID(ctx context.Context, sequenceID string, deps *Deps) (string, error) {
	client, err := deps.NewClient()
	if err != nil {
		return "", err
	}
	if err := deps.RequireWorkspace(client); err != nil {
		return "", err
	}

	lookupURL := fmt.Sprintf("%s/api/v1/workspaces/%s/work-items/%s/",
		client.BaseURL, client.Workspace, sequenceID)

	respBody, err := client.Get(ctx, lookupURL)
	if err != nil {
		return "", fmt.Errorf("resolving sequence ID %q: %w", sequenceID, err)
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing sequence ID response for %q: %w", sequenceID, err)
	}

	id, ok := result["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("sequence ID %q: response missing 'id' field", sequenceID)
	}
	return id, nil
}

// ResolveNameToUUID resolves a human-readable name to a UUID by finding the
// corresponding resource's list endpoint and searching for a match.
func ResolveNameToUUID(ctx context.Context, name, paramName string, deps *Deps) (string, error) {
	// Extract resource name from param: e.g., "state_id" → "state"
	resourceName := strings.TrimSuffix(paramName, "_id")

	client, err := deps.NewClient()
	if err != nil {
		return "", err
	}
	if err := deps.RequireWorkspace(client); err != nil {
		return "", err
	}

	projectID := ""
	if deps.RequireProject != nil {
		projectID, _ = deps.RequireProject()
	}

	// Try to find the list endpoint for this resource
	listURL := buildListURL(client, resourceName, projectID)
	if listURL == "" {
		return "", fmt.Errorf("cannot resolve %q: no list endpoint known for %q", name, resourceName)
	}

	respBody, err := client.Get(ctx, listURL)
	if err != nil {
		return "", fmt.Errorf("resolving %q for %s: %w", name, resourceName, err)
	}

	return findByName(respBody, name)
}

// buildListURL constructs the list endpoint URL for a given resource type.
func buildListURL(client *api.Client, resourceName, projectID string) string {
	base := fmt.Sprintf("%s/api/v1/workspaces/%s", client.BaseURL, client.Workspace)

	// Map resource names to their list endpoint paths
	switch resourceName {
	case "state":
		if projectID != "" {
			return fmt.Sprintf("%s/projects/%s/states/", base, projectID)
		}
	case "label":
		if projectID != "" {
			return fmt.Sprintf("%s/projects/%s/labels/", base, projectID)
		}
	case "cycle":
		if projectID != "" {
			return fmt.Sprintf("%s/projects/%s/cycles/", base, projectID)
		}
	case "module":
		if projectID != "" {
			return fmt.Sprintf("%s/projects/%s/modules/", base, projectID)
		}
	case "member":
		return fmt.Sprintf("%s/members/", base)
	case "project":
		return fmt.Sprintf("%s/projects/", base)
	}

	return ""
}

// findByName searches a JSON response for an object whose "name" (or "display_name"
// or "identifier") field matches the given name (case-insensitive).
func findByName(respBody []byte, name string) (string, error) {
	lowerName := strings.ToLower(name)

	// Try as paginated response first
	var paginated struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(respBody, &paginated); err == nil && len(paginated.Results) > 0 {
		return searchItems(paginated.Results, lowerName)
	}

	// Try as plain array
	var items []json.RawMessage
	if err := json.Unmarshal(respBody, &items); err == nil && len(items) > 0 {
		return searchItems(items, lowerName)
	}

	return "", fmt.Errorf("name %q not found", name)
}

func searchItems(items []json.RawMessage, lowerName string) (string, error) {
	for _, raw := range items {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		id, _ := obj["id"].(string)
		if id == "" {
			continue
		}

		// Check various name fields
		for _, field := range []string{"name", "display_name", "identifier"} {
			if val, ok := obj[field].(string); ok {
				if strings.ToLower(val) == lowerName {
					return id, nil
				}
			}
		}
	}
	return "", fmt.Errorf("name %q not found", lowerName)
}
