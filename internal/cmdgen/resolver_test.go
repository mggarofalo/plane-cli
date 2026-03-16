package cmdgen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/cache"
)

func TestFindByName_PaginatedResponse(t *testing.T) {
	response := `{
		"results": [
			{"id": "aaa-bbb-ccc", "name": "Backlog"},
			{"id": "ddd-eee-fff", "name": "In Progress"},
			{"id": "ggg-hhh-iii", "name": "Done"}
		]
	}`

	id, err := findByName([]byte(response), "In Progress")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "ddd-eee-fff" {
		t.Errorf("expected ddd-eee-fff, got %s", id)
	}
}

func TestFindByName_PlainArray(t *testing.T) {
	response := `[
		{"id": "aaa", "name": "Alpha"},
		{"id": "bbb", "name": "Beta"}
	]`

	id, err := findByName([]byte(response), "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "bbb" {
		t.Errorf("expected bbb, got %s", id)
	}
}

func TestFindByName_CaseInsensitive(t *testing.T) {
	response := `{"results": [{"id": "123", "name": "Backlog"}]}`
	id, err := findByName([]byte(response), "backlog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "123" {
		t.Errorf("expected 123, got %s", id)
	}
}

func TestFindByName_DisplayName(t *testing.T) {
	response := `[{"id": "u1", "display_name": "John Smith"}]`
	id, err := findByName([]byte(response), "John Smith")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "u1" {
		t.Errorf("expected u1, got %s", id)
	}
}

func TestFindByName_NotFound(t *testing.T) {
	response := `{"results": [{"id": "123", "name": "Backlog"}]}`
	_, err := findByName([]byte(response), "nonexistent")
	if err == nil {
		t.Error("expected error for not found name")
	}
}

func TestSearchItems(t *testing.T) {
	items := []json.RawMessage{
		json.RawMessage(`{"id": "1", "name": "Alpha"}`),
		json.RawMessage(`{"id": "2", "identifier": "PROJ"}`),
	}

	id, err := searchItems(items, "proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "2" {
		t.Errorf("expected 2, got %s", id)
	}
}

func TestIsSequenceID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"PROJ-42", true},
		{"RECEIPTS-351", true},
		{"A-1", true},
		{"abc-99", true},
		{"P2-100", true},
		{"proj-0", true},
		{"-42", false},
		{"42-PROJ", false},
		{"PROJ", false},
		{"PROJ-", false},
		{"-1", false},
		{"", false},
		{"550e8400-e29b-41d4-a716-446655440000", false},
		{"some name", false},
		{"PROJ-42-extra", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsSequenceID(tt.input)
			if got != tt.want {
				t.Errorf("IsSequenceID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveSequenceID(t *testing.T) {
	t.Run("resolves valid sequence ID", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/workspaces/test-ws/work-items/PROJ-42/" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			fmt.Fprint(w, `{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "Test Issue"}`)
		}))
		defer srv.Close()

		deps := &Deps{
			NewClient: func() (*api.Client, error) {
				return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
			},
			RequireWorkspace: func(c *api.Client) error { return nil },
		}

		id, err := ResolveSequenceID(context.Background(), "PROJ-42", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "550e8400-e29b-41d4-a716-446655440000" {
			t.Errorf("expected UUID, got %s", id)
		}
	})

	t.Run("returns error on API failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"detail": "not found"}`)
		}))
		defer srv.Close()

		deps := &Deps{
			NewClient: func() (*api.Client, error) {
				return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
			},
			RequireWorkspace: func(c *api.Client) error { return nil },
		}

		_, err := ResolveSequenceID(context.Background(), "PROJ-999", deps)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on missing id field", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `{"name": "Test Issue"}`)
		}))
		defer srv.Close()

		deps := &Deps{
			NewClient: func() (*api.Client, error) {
				return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
			},
			RequireWorkspace: func(c *api.Client) error { return nil },
		}

		_, err := ResolveSequenceID(context.Background(), "PROJ-42", deps)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func testCacheStore(t *testing.T) *cache.Store {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	return cache.NewStore("test")
}

func TestResolveNameToUUID_CacheHit(t *testing.T) {
	apiCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		fmt.Fprint(w, `[{"id": "s1", "name": "Done"}]`)
	}))
	defer srv.Close()

	store := testCacheStore(t)

	// Pre-populate cache
	cr := &cache.CachedResource{
		FetchedAt: time.Now(),
		Entries: []cache.Entry{
			{ID: "cached-uuid", Name: "Done"},
		},
	}
	store.Save("test-ws", "proj1", cache.KindStates, cr)

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "tok", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj1", nil },
		CacheStore:       store,
	}

	id, err := ResolveNameToUUID(context.Background(), "Done", "state_id", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cached-uuid" {
		t.Errorf("expected cached-uuid, got %s", id)
	}
	if apiCalls != 0 {
		t.Errorf("expected 0 API calls (cache hit), got %d", apiCalls)
	}
}

func TestResolveNameToUUID_CacheMiss(t *testing.T) {
	apiCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		fmt.Fprint(w, `[{"id": "api-uuid", "name": "Done"}]`)
	}))
	defer srv.Close()

	store := testCacheStore(t)

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "tok", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj1", nil },
		CacheStore:       store,
	}

	id, err := ResolveNameToUUID(context.Background(), "Done", "state_id", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "api-uuid" {
		t.Errorf("expected api-uuid, got %s", id)
	}
	if apiCalls != 1 {
		t.Errorf("expected 1 API call (cache miss), got %d", apiCalls)
	}

	// Verify cache was populated
	loaded, _ := store.Load("test-ws", "proj1", cache.KindStates)
	if loaded == nil {
		t.Fatal("expected cache to be populated after API call")
	}
	if loaded.FindByName("Done") != "api-uuid" {
		t.Error("expected populated cache to contain the resolved entry")
	}
}

func TestResolveNameToUUID_CacheStale(t *testing.T) {
	apiCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		fmt.Fprint(w, `[{"id": "fresh-uuid", "name": "Done"}]`)
	}))
	defer srv.Close()

	store := testCacheStore(t)

	// Pre-populate with stale cache (2 hours old)
	cr := &cache.CachedResource{
		FetchedAt: time.Now().Add(-2 * time.Hour),
		Entries: []cache.Entry{
			{ID: "stale-uuid", Name: "Done"},
		},
	}
	store.Save("test-ws", "proj1", cache.KindStates, cr)

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "tok", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj1", nil },
		CacheStore:       store,
	}

	id, err := ResolveNameToUUID(context.Background(), "Done", "state_id", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return stale cached value
	if id != "stale-uuid" {
		t.Errorf("expected stale-uuid (cached), got %s", id)
	}

	// Wait for background refresh
	cache.PendingRefreshes.Wait()

	// Background refresh should have made an API call
	if apiCalls != 1 {
		t.Errorf("expected 1 API call (background refresh), got %d", apiCalls)
	}
}

func TestResolveNameToUUID_CacheExpired(t *testing.T) {
	apiCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		fmt.Fprint(w, `[{"id": "fresh-uuid", "name": "Done"}]`)
	}))
	defer srv.Close()

	store := testCacheStore(t)

	// Pre-populate with expired cache (8 days old)
	cr := &cache.CachedResource{
		FetchedAt: time.Now().Add(-8 * 24 * time.Hour),
		Entries: []cache.Entry{
			{ID: "expired-uuid", Name: "Done"},
		},
	}
	store.Save("test-ws", "proj1", cache.KindStates, cr)

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "tok", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj1", nil },
		CacheStore:       store,
	}

	id, err := ResolveNameToUUID(context.Background(), "Done", "state_id", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall through to API since cache is expired
	if id != "fresh-uuid" {
		t.Errorf("expected fresh-uuid (from API), got %s", id)
	}
	if apiCalls != 1 {
		t.Errorf("expected 1 API call (expired cache), got %d", apiCalls)
	}
}

func TestResolveNameToUUID_NoCacheStore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id": "api-uuid", "name": "Done"}]`)
	}))
	defer srv.Close()

	// No CacheStore — should work like before
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "tok", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj1", nil },
		CacheStore:       nil,
	}

	id, err := ResolveNameToUUID(context.Background(), "Done", "state_id", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "api-uuid" {
		t.Errorf("expected api-uuid, got %s", id)
	}
}
