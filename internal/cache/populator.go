package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mggarofalo/plane-cli/internal/api"
)

// PendingRefreshes tracks in-flight background cache refreshes.
// Callers should call PendingRefreshes.Wait() before process exit to avoid
// losing writes.
var PendingRefreshes sync.WaitGroup

// FetchFunc fetches raw JSON from a list endpoint.
type FetchFunc func(ctx context.Context) ([]byte, error)

// BuildFetchFunc creates a FetchFunc that GETs a resource list endpoint.
func BuildFetchFunc(client *api.Client, kind ResourceKind, projectID string) FetchFunc {
	return func(ctx context.Context) ([]byte, error) {
		url := buildResourceURL(client, kind, projectID)
		if url == "" {
			return nil, fmt.Errorf("no list endpoint for kind %q", kind)
		}
		return client.Get(ctx, url)
	}
}

// buildResourceURL constructs the list endpoint URL for a resource kind.
func buildResourceURL(client *api.Client, kind ResourceKind, projectID string) string {
	base := fmt.Sprintf("%s/api/v1/workspaces/%s", client.BaseURL, client.Workspace)

	switch kind {
	case KindStates:
		if projectID != "" {
			return fmt.Sprintf("%s/projects/%s/states/", base, projectID)
		}
	case KindLabels:
		if projectID != "" {
			return fmt.Sprintf("%s/projects/%s/labels/", base, projectID)
		}
	case KindCycles:
		if projectID != "" {
			return fmt.Sprintf("%s/projects/%s/cycles/", base, projectID)
		}
	case KindModules:
		if projectID != "" {
			return fmt.Sprintf("%s/projects/%s/modules/", base, projectID)
		}
	case KindMembers:
		return fmt.Sprintf("%s/members/", base)
	case KindProjects:
		return fmt.Sprintf("%s/projects/", base)
	}
	return ""
}

// ParseEntries extracts Entry objects from a JSON response body. Handles both
// paginated responses (with "results" key) and plain arrays.
func ParseEntries(respBody []byte) ([]Entry, error) {
	// Try paginated response first
	var paginated struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(respBody, &paginated); err == nil && len(paginated.Results) > 0 {
		return extractEntries(paginated.Results)
	}

	// Try plain array
	var items []json.RawMessage
	if err := json.Unmarshal(respBody, &items); err == nil && len(items) > 0 {
		return extractEntries(items)
	}

	return nil, nil
}

func extractEntries(items []json.RawMessage) ([]Entry, error) {
	entries := make([]Entry, 0, len(items))
	for _, raw := range items {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		id, _ := obj["id"].(string)
		if id == "" {
			continue
		}

		e := Entry{ID: id}
		if v, ok := obj["name"].(string); ok {
			e.Name = v
		}
		if v, ok := obj["display_name"].(string); ok {
			e.DisplayName = v
		}
		if v, ok := obj["identifier"].(string); ok {
			e.Identifier = v
		}

		entries = append(entries, e)
	}
	return entries, nil
}

// PopulateFromAPI fetches resources from the API and saves them to the store.
func PopulateFromAPI(ctx context.Context, store *Store, workspace, projectID string, kind ResourceKind, fetch FetchFunc) (*CachedResource, error) {
	respBody, err := fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", kind, err)
	}

	entries, err := ParseEntries(respBody)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", kind, err)
	}

	// Don't overwrite a valid stale cache with empty entries from a
	// transient API error or unrecognized response format.
	if entries == nil {
		return nil, nil
	}

	cached := &CachedResource{
		FetchedAt: time.Now(),
		Entries:   entries,
	}

	if err := store.Save(workspace, projectID, kind, cached); err != nil {
		// Non-fatal: we have the data in memory even if disk write fails
		return cached, nil
	}

	return cached, nil
}

// RefreshInBackground starts a goroutine that refreshes a cache entry.
// The caller should wait on PendingRefreshes before process exit.
func RefreshInBackground(store *Store, workspace, projectID string, kind ResourceKind, fetch FetchFunc) {
	PendingRefreshes.Add(1)
	go func() {
		defer PendingRefreshes.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// Best-effort; errors are silently ignored since the stale cache is still valid
		_, _ = PopulateFromAPI(ctx, store, workspace, projectID, kind, fetch)
	}()
}
