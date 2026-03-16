package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mggarofalo/plane-cli/internal/api"
)

// --- FindByName tests ---

func TestFindByName_Hit(t *testing.T) {
	cr := &CachedResource{
		FetchedAt: time.Now(),
		Entries: []Entry{
			{ID: "aaa", Name: "Backlog"},
			{ID: "bbb", Name: "In Progress"},
			{ID: "ccc", Name: "Done"},
		},
	}

	id := cr.FindByName("In Progress")
	if id != "bbb" {
		t.Errorf("expected bbb, got %s", id)
	}
}

func TestFindByName_Miss(t *testing.T) {
	cr := &CachedResource{
		FetchedAt: time.Now(),
		Entries: []Entry{
			{ID: "aaa", Name: "Backlog"},
		},
	}

	id := cr.FindByName("nonexistent")
	if id != "" {
		t.Errorf("expected empty string, got %s", id)
	}
}

func TestFindByName_CaseInsensitive(t *testing.T) {
	cr := &CachedResource{
		FetchedAt: time.Now(),
		Entries: []Entry{
			{ID: "aaa", Name: "Backlog"},
		},
	}

	id := cr.FindByName("backlog")
	if id != "aaa" {
		t.Errorf("expected aaa, got %s", id)
	}

	id = cr.FindByName("BACKLOG")
	if id != "aaa" {
		t.Errorf("expected aaa, got %s", id)
	}
}

func TestFindByName_DisplayName(t *testing.T) {
	cr := &CachedResource{
		FetchedAt: time.Now(),
		Entries: []Entry{
			{ID: "u1", DisplayName: "John Smith"},
		},
	}

	id := cr.FindByName("john smith")
	if id != "u1" {
		t.Errorf("expected u1, got %s", id)
	}
}

func TestFindByName_Identifier(t *testing.T) {
	cr := &CachedResource{
		FetchedAt: time.Now(),
		Entries: []Entry{
			{ID: "p1", Identifier: "PLANECLI"},
		},
	}

	id := cr.FindByName("planecli")
	if id != "p1" {
		t.Errorf("expected p1, got %s", id)
	}
}

// --- IsStale / IsExpired tests ---

func TestIsStale_Fresh(t *testing.T) {
	cr := &CachedResource{FetchedAt: time.Now()}
	if cr.IsStale() {
		t.Error("expected fresh cache to not be stale")
	}
}

func TestIsStale_Old(t *testing.T) {
	cr := &CachedResource{FetchedAt: time.Now().Add(-2 * time.Hour)}
	if !cr.IsStale() {
		t.Error("expected 2-hour-old cache to be stale")
	}
}

func TestIsStale_Boundary(t *testing.T) {
	// Just under 1 hour — should not be stale
	cr := &CachedResource{FetchedAt: time.Now().Add(-59 * time.Minute)}
	if cr.IsStale() {
		t.Error("expected 59-min-old cache to not be stale")
	}
}

func TestIsExpired_Fresh(t *testing.T) {
	cr := &CachedResource{FetchedAt: time.Now()}
	if cr.IsExpired() {
		t.Error("expected fresh cache to not be expired")
	}
}

func TestIsExpired_Old(t *testing.T) {
	cr := &CachedResource{FetchedAt: time.Now().Add(-8 * 24 * time.Hour)}
	if !cr.IsExpired() {
		t.Error("expected 8-day-old cache to be expired")
	}
}

func TestIsExpired_Boundary(t *testing.T) {
	// 6 days — should not be expired
	cr := &CachedResource{FetchedAt: time.Now().Add(-6 * 24 * time.Hour)}
	if cr.IsExpired() {
		t.Error("expected 6-day-old cache to not be expired")
	}
}

// --- Store tests ---

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	return NewStore("test")
}

func TestStore_SaveAndLoad(t *testing.T) {
	store := testStore(t)

	original := &CachedResource{
		FetchedAt: time.Now().Truncate(time.Millisecond),
		Entries: []Entry{
			{ID: "aaa", Name: "Alpha"},
			{ID: "bbb", Name: "Beta"},
		},
	}

	err := store.Save("ws1", "proj1", KindStates, original)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load("ws1", "proj1", KindStates)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil loaded resource")
	}

	if len(loaded.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].ID != "aaa" {
		t.Errorf("expected aaa, got %s", loaded.Entries[0].ID)
	}
}

func TestStore_LoadMissing(t *testing.T) {
	store := testStore(t)

	loaded, err := store.Load("ws1", "proj1", KindStates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for missing file")
	}
}

func TestStore_LoadCorruptJSON(t *testing.T) {
	store := testStore(t)

	// Write corrupt data directly
	base, _ := store.baseDir()
	dir := filepath.Join(base, "ws1", "proj1")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "states.json"), []byte("{corrupt"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loaded, err := store.Load("ws1", "proj1", KindStates)
	if err != nil {
		t.Fatalf("unexpected error on corrupt file: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for corrupt file")
	}
}

func TestStore_Clear(t *testing.T) {
	store := testStore(t)

	cr := &CachedResource{FetchedAt: time.Now(), Entries: []Entry{{ID: "a", Name: "A"}}}
	if err := store.Save("ws1", "proj1", KindStates, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("ws2", "proj2", KindLabels, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Clear all
	if err := store.Clear(""); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	loaded, _ := store.Load("ws1", "proj1", KindStates)
	if loaded != nil {
		t.Error("expected nil after clear")
	}
	loaded, _ = store.Load("ws2", "proj2", KindLabels)
	if loaded != nil {
		t.Error("expected nil after clear")
	}
}

func TestStore_ClearWorkspace(t *testing.T) {
	store := testStore(t)

	cr := &CachedResource{FetchedAt: time.Now(), Entries: []Entry{{ID: "a", Name: "A"}}}
	if err := store.Save("ws1", "proj1", KindStates, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("ws2", "proj2", KindLabels, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Clear only ws1
	if err := store.Clear("ws1"); err != nil {
		t.Fatalf("Clear ws1 failed: %v", err)
	}

	loaded, _ := store.Load("ws1", "proj1", KindStates)
	if loaded != nil {
		t.Error("expected nil after clear ws1")
	}
	loaded, _ = store.Load("ws2", "proj2", KindLabels)
	if loaded == nil {
		t.Error("expected ws2 to survive ws1 clear")
	}
}

func TestStore_ClearKind(t *testing.T) {
	store := testStore(t)

	cr := &CachedResource{FetchedAt: time.Now(), Entries: []Entry{{ID: "a", Name: "A"}}}
	if err := store.Save("ws1", "proj1", KindStates, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("ws1", "proj1", KindLabels, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.ClearKind("ws1", "proj1", KindStates); err != nil {
		t.Fatalf("ClearKind failed: %v", err)
	}

	loaded, _ := store.Load("ws1", "proj1", KindStates)
	if loaded != nil {
		t.Error("expected states to be cleared")
	}
	loaded, _ = store.Load("ws1", "proj1", KindLabels)
	if loaded == nil {
		t.Error("expected labels to survive")
	}
}

func TestStore_ListAll(t *testing.T) {
	store := testStore(t)

	cr := &CachedResource{FetchedAt: time.Now(), Entries: []Entry{{ID: "a", Name: "A"}}}
	if err := store.Save("ws1", "proj1", KindStates, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("ws1", "", KindMembers, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("ws2", "proj2", KindLabels, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	items, err := store.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

// --- ParseEntries tests ---

func TestParseEntries_Paginated(t *testing.T) {
	body := `{
		"results": [
			{"id": "aaa", "name": "Backlog"},
			{"id": "bbb", "name": "Done", "display_name": "Finished"}
		]
	}`

	entries, err := ParseEntries([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != "aaa" || entries[0].Name != "Backlog" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].DisplayName != "Finished" {
		t.Errorf("expected display_name Finished, got %s", entries[1].DisplayName)
	}
}

func TestParseEntries_PlainArray(t *testing.T) {
	body := `[
		{"id": "aaa", "name": "Alpha"},
		{"id": "bbb", "identifier": "PROJ"}
	]`

	entries, err := ParseEntries([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[1].Identifier != "PROJ" {
		t.Errorf("expected identifier PROJ, got %s", entries[1].Identifier)
	}
}

func TestParseEntries_SkipsMissingID(t *testing.T) {
	body := `[{"name": "No ID"}, {"id": "aaa", "name": "Has ID"}]`
	entries, err := ParseEntries([]byte(body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestParseEntries_EmptyResponse(t *testing.T) {
	entries, err := ParseEntries([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}
}

// --- PopulateFromAPI test ---

func TestPopulateFromAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results": [
			{"id": "s1", "name": "Backlog"},
			{"id": "s2", "name": "In Progress"},
			{"id": "s3", "name": "Done"}
		]}`)
	}))
	defer srv.Close()

	store := testStore(t)
	client := api.NewClient(srv.URL, "test-token", "test-ws", false, nil)

	fetch := BuildFetchFunc(client, KindStates, "proj1")
	cached, err := PopulateFromAPI(context.Background(), store, "test-ws", "proj1", KindStates, fetch)
	if err != nil {
		t.Fatalf("PopulateFromAPI failed: %v", err)
	}

	if len(cached.Entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(cached.Entries))
	}

	// Verify it was persisted
	loaded, err := store.Load("test-ws", "proj1", KindStates)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected persisted cache")
	}
	if len(loaded.Entries) != 3 {
		t.Errorf("expected 3 persisted entries, got %d", len(loaded.Entries))
	}

	// Verify FindByName works on the loaded cache
	id := loaded.FindByName("Done")
	if id != "s3" {
		t.Errorf("expected s3, got %s", id)
	}
}

// --- ResourceKindFromParam tests ---

func TestResourceKindFromParam(t *testing.T) {
	tests := []struct {
		param string
		want  ResourceKind
	}{
		{"state_id", KindStates},
		{"state", KindStates},
		{"label_id", KindLabels},
		{"label", KindLabels},
		{"cycle_id", KindCycles},
		{"cycle", KindCycles},
		{"module_id", KindModules},
		{"module", KindModules},
		{"member_id", KindMembers},
		{"member", KindMembers},
		{"project_id", KindProjects},
		{"project", KindProjects},
		{"unknown_id", ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			got := ResourceKindFromParam(tt.param)
			if got != tt.want {
				t.Errorf("ResourceKindFromParam(%q) = %q, want %q", tt.param, got, tt.want)
			}
		})
	}
}

// --- RefreshInBackground test ---

func TestRefreshInBackground(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		fmt.Fprint(w, `[{"id": "x", "name": "Refreshed"}]`)
	}))
	defer srv.Close()

	store := testStore(t)
	client := api.NewClient(srv.URL, "test-token", "test-ws", false, nil)
	fetch := BuildFetchFunc(client, KindMembers, "")

	RefreshInBackground(store, "test-ws", "", KindMembers, fetch)
	PendingRefreshes.Wait()

	if callCount != 1 {
		t.Errorf("expected 1 API call, got %d", callCount)
	}

	loaded, _ := store.Load("test-ws", "", KindMembers)
	if loaded == nil {
		t.Fatal("expected cache to be populated after background refresh")
	}
	if len(loaded.Entries) != 1 || loaded.Entries[0].Name != "Refreshed" {
		t.Errorf("unexpected entries: %+v", loaded.Entries)
	}
}

// --- BuildFetchFunc URL construction tests ---

func TestBuildFetchFunc_URLs(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.Path
		fmt.Fprint(w, `[]`)
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL, "tok", "my-ws", false, nil)

	tests := []struct {
		kind      ResourceKind
		projectID string
		wantPath  string
	}{
		{KindStates, "p1", "/api/v1/workspaces/my-ws/projects/p1/states/"},
		{KindLabels, "p1", "/api/v1/workspaces/my-ws/projects/p1/labels/"},
		{KindCycles, "p1", "/api/v1/workspaces/my-ws/projects/p1/cycles/"},
		{KindModules, "p1", "/api/v1/workspaces/my-ws/projects/p1/modules/"},
		{KindMembers, "", "/api/v1/workspaces/my-ws/members/"},
		{KindProjects, "", "/api/v1/workspaces/my-ws/projects/"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			fetch := BuildFetchFunc(client, tt.kind, tt.projectID)
			_, err := fetch(context.Background())
			if err != nil {
				t.Fatalf("fetch failed: %v", err)
			}
			if capturedURL != tt.wantPath {
				t.Errorf("expected path %s, got %s", tt.wantPath, capturedURL)
			}
		})
	}
}

// --- Marshaling round-trip ---

func TestCachedResource_JSONRoundTrip(t *testing.T) {
	original := &CachedResource{
		FetchedAt: time.Now().Truncate(time.Millisecond),
		Entries: []Entry{
			{ID: "a", Name: "Alpha", DisplayName: "A Display", Identifier: "ALPHA"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded CachedResource
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(decoded.Entries))
	}
	e := decoded.Entries[0]
	if e.ID != "a" || e.Name != "Alpha" || e.DisplayName != "A Display" || e.Identifier != "ALPHA" {
		t.Errorf("unexpected entry: %+v", e)
	}
}
