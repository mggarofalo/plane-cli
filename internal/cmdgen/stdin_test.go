package cmdgen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/output"
)

// readJSONBody is a test helper that reads and decodes a JSON request body.
func readJSONBody(r *http.Request) (map[string]any, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	return body, nil
}

func TestReadStdinJSON(t *testing.T) {
	t.Run("reads valid JSON object", func(t *testing.T) {
		old := stdinReader
		defer func() { stdinReader = old }()
		stdinReader = strings.NewReader(`{"name": "Test Issue", "priority": 2}`)

		body, err := ReadStdinJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if body["name"] != "Test Issue" {
			t.Errorf("expected name=Test Issue, got %v", body["name"])
		}
		// JSON numbers unmarshal as float64
		if body["priority"] != float64(2) {
			t.Errorf("expected priority=2, got %v", body["priority"])
		}
	})

	t.Run("returns error for empty stdin", func(t *testing.T) {
		old := stdinReader
		defer func() { stdinReader = old }()
		stdinReader = strings.NewReader("")

		_, err := ReadStdinJSON()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "stdin is empty") {
			t.Errorf("expected 'stdin is empty' error, got: %v", err)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		old := stdinReader
		defer func() { stdinReader = old }()
		stdinReader = strings.NewReader("not json at all")

		_, err := ReadStdinJSON()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid JSON") {
			t.Errorf("expected 'invalid JSON' error, got: %v", err)
		}
	})

	t.Run("returns error for JSON array (not object)", func(t *testing.T) {
		old := stdinReader
		defer func() { stdinReader = old }()
		stdinReader = strings.NewReader(`["a", "b"]`)

		_, err := ReadStdinJSON()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid JSON") {
			t.Errorf("expected 'invalid JSON' error for array, got: %v", err)
		}
	})

	t.Run("returns error for JSON string (not object)", func(t *testing.T) {
		old := stdinReader
		defer func() { stdinReader = old }()
		stdinReader = strings.NewReader(`"just a string"`)

		_, err := ReadStdinJSON()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid JSON") {
			t.Errorf("expected 'invalid JSON' error for string, got: %v", err)
		}
	})

	t.Run("handles deeply nested JSON", func(t *testing.T) {
		old := stdinReader
		defer func() { stdinReader = old }()
		stdinReader = strings.NewReader(`{"name": "Test", "meta": {"key": "value"}}`)

		body, err := ReadStdinJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		meta, ok := body["meta"].(map[string]any)
		if !ok {
			t.Fatal("expected meta to be a map")
		}
		if meta["key"] != "value" {
			t.Errorf("expected meta.key=value, got %v", meta["key"])
		}
	})
}

func TestMergeStdinWithFlags(t *testing.T) {
	t.Run("flags override stdin values", func(t *testing.T) {
		stdinBody := map[string]any{
			"name":  "stdin name",
			"state": "stdin-state-uuid",
		}
		flagBody := map[string]any{
			"name": "flag name",
		}

		merged := MergeStdinWithFlags(stdinBody, flagBody)

		if merged["name"] != "flag name" {
			t.Errorf("expected flag value to win, got %v", merged["name"])
		}
		if merged["state"] != "stdin-state-uuid" {
			t.Errorf("expected stdin value preserved, got %v", merged["state"])
		}
	})

	t.Run("returns nil when both are nil", func(t *testing.T) {
		merged := MergeStdinWithFlags(nil, nil)
		if merged != nil {
			t.Errorf("expected nil, got %v", merged)
		}
	})

	t.Run("returns stdin body when flags are nil", func(t *testing.T) {
		stdinBody := map[string]any{"name": "test"}

		merged := MergeStdinWithFlags(stdinBody, nil)

		if merged["name"] != "test" {
			t.Errorf("expected name=test, got %v", merged["name"])
		}
	})

	t.Run("returns flag body when stdin is nil", func(t *testing.T) {
		flagBody := map[string]any{"name": "test"}

		merged := MergeStdinWithFlags(nil, flagBody)

		if merged["name"] != "test" {
			t.Errorf("expected name=test, got %v", merged["name"])
		}
	})

	t.Run("merges disjoint keys", func(t *testing.T) {
		stdinBody := map[string]any{"description": "from stdin"}
		flagBody := map[string]any{"name": "from flags"}

		merged := MergeStdinWithFlags(stdinBody, flagBody)

		if merged["description"] != "from stdin" {
			t.Errorf("expected description=from stdin, got %v", merged["description"])
		}
		if merged["name"] != "from flags" {
			t.Errorf("expected name=from flags, got %v", merged["name"])
		}
	})
}

func TestStdinKeys(t *testing.T) {
	t.Run("returns stdin keys not overridden by flags", func(t *testing.T) {
		stdinBody := map[string]any{
			"name":  "stdin name",
			"state": "In Progress",
			"label": "Bug",
		}
		flagBody := map[string]any{
			"name": "flag name",
		}

		keys := StdinKeys(stdinBody, flagBody)

		if !keys["state"] {
			t.Error("expected 'state' to be a stdin key")
		}
		if !keys["label"] {
			t.Error("expected 'label' to be a stdin key")
		}
		if keys["name"] {
			t.Error("'name' should be overridden by flag, not a stdin key")
		}
	})

	t.Run("handles nil flag body", func(t *testing.T) {
		stdinBody := map[string]any{"name": "test"}

		keys := StdinKeys(stdinBody, nil)

		if !keys["name"] {
			t.Error("expected 'name' to be a stdin key")
		}
	})
}

func TestIsStdin(t *testing.T) {
	t.Run("returns false when nil deps", func(t *testing.T) {
		if isStdin(nil) {
			t.Error("expected false for nil deps")
		}
	})

	t.Run("returns false when FlagStdin is nil", func(t *testing.T) {
		deps := &Deps{}
		if isStdin(deps) {
			t.Error("expected false for nil FlagStdin")
		}
	})

	t.Run("returns false when false", func(t *testing.T) {
		f := false
		deps := &Deps{FlagStdin: &f}
		if isStdin(deps) {
			t.Error("expected false")
		}
	})

	t.Run("returns true when true", func(t *testing.T) {
		f := true
		deps := &Deps{FlagStdin: &f}
		if !isStdin(deps) {
			t.Error("expected true")
		}
	})
}

func TestResolveStdinBody(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return states list for resolution
		if strings.Contains(r.URL.Path, "/states/") {
			fmt.Fprintf(w, `{"results": [{"id": "%s", "name": "In Progress"}]}`, uuid)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	isUUID := func(s string) bool { return len(s) == 36 && s[8] == '-' }
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		IsUUID:           isUUID,
	}

	t.Run("resolves state name to UUID", func(t *testing.T) {
		body := map[string]any{
			"name":     "Test Issue",
			"state_id": "In Progress",
		}
		stdinKeys := map[string]bool{"state_id": true}

		err := ResolveStdinBody(context.Background(), body, stdinKeys, deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if body["state_id"] != uuid {
			t.Errorf("expected state_id=%s, got %v", uuid, body["state_id"])
		}
	})

	t.Run("skips non-stdin keys", func(t *testing.T) {
		body := map[string]any{
			"name":     "Test Issue",
			"state_id": "In Progress",
		}
		// state_id not in stdinKeys means it came from flags (already resolved)
		stdinKeys := map[string]bool{}

		err := ResolveStdinBody(context.Background(), body, stdinKeys, deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should NOT be resolved since it's not a stdin key
		if body["state_id"] != "In Progress" {
			t.Errorf("expected state_id to remain 'In Progress', got %v", body["state_id"])
		}
	})

	t.Run("handles nil body", func(t *testing.T) {
		err := ResolveStdinBody(context.Background(), nil, map[string]bool{"x": true}, deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles nil deps", func(t *testing.T) {
		body := map[string]any{"state_id": "In Progress"}
		err := ResolveStdinBody(context.Background(), body, map[string]bool{"state_id": true}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestStdin_ExecuteSpecFromArgs_POST(t *testing.T) {
	var receivedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := readJSONBody(r)
			if err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			receivedBody = body
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id": "new-uuid", "name": "Test"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// Set up stdin with JSON
	old := stdinReader
	defer func() { stdinReader = old }()
	stdinReader = strings.NewReader(`{"name": "Stdin Issue", "priority": 2}`)

	stdinFlag := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStdin:        &stdinFlag,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
	}

	parsed := &ParsedArgs{
		Values: map[string]string{},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["name"] != "Stdin Issue" {
		t.Errorf("expected name=Stdin Issue, got %v", receivedBody["name"])
	}
}

func TestStdin_FlagsOverrideStdin(t *testing.T) {
	var receivedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, err := readJSONBody(r)
			if err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			receivedBody = body
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id": "new-uuid", "name": "Flag Name"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// stdin provides name and priority
	old := stdinReader
	defer func() { stdinReader = old }()
	stdinReader = strings.NewReader(`{"name": "Stdin Name", "priority": 2}`)

	stdinFlag := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStdin:        &stdinFlag,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
		},
	}

	// Flag provides name="Flag Name" which should override stdin name
	parsed := &ParsedArgs{
		Values: map[string]string{"name": "Flag Name"},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["name"] != "Flag Name" {
		t.Errorf("expected flag name to override stdin, got %v", receivedBody["name"])
	}
	// priority from stdin should still be present
	if receivedBody["priority"] != float64(2) {
		t.Errorf("expected priority=2 from stdin, got %v", receivedBody["priority"])
	}
}

func TestStdin_IgnoredForGET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"results": [], "total_pages": 1}`)
	}))
	defer srv.Close()

	// stdin has data, but it should be ignored for GET
	old := stdinReader
	defer func() { stdinReader = old }()
	stdinReader = strings.NewReader(`{"name": "Should Be Ignored"}`)

	stdinFlag := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStdin:        &stdinFlag,
	}

	spec := &docs.EndpointSpec{
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "List Work Items",
	}

	parsed := &ParsedArgs{
		Values: map[string]string{},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// If we got here without error, the stdin was correctly ignored for GET
}

func TestStdin_IgnoredForDELETE(t *testing.T) {
	deleteCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	old := stdinReader
	defer func() { stdinReader = old }()
	stdinReader = strings.NewReader(`{"name": "Should Be Ignored"}`)

	stdinFlag := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStdin:        &stdinFlag,
	}

	spec := &docs.EndpointSpec{
		Method:       "DELETE",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
		EntryTitle:   "Delete Work Item",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Location: docs.ParamPath, Type: "string", Required: true},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"work-item-id": "550e8400-e29b-41d4-a716-446655440000"},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected DELETE request to be made")
	}
}

func TestStdin_PATCH(t *testing.T) {
	var receivedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PATCH" {
			body, err := readJSONBody(r)
			if err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			receivedBody = body
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"id": "uuid-1", "name": "Updated"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	old := stdinReader
	defer func() { stdinReader = old }()
	stdinReader = strings.NewReader(`{"name": "Updated Name", "description": "updated via stdin"}`)

	stdinFlag := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStdin:        &stdinFlag,
	}

	spec := &docs.EndpointSpec{
		Method:       "PATCH",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
		EntryTitle:   "Update Work Item",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Location: docs.ParamPath, Type: "string", Required: true},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"work-item-id": "550e8400-e29b-41d4-a716-446655440000"},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["name"] != "Updated Name" {
		t.Errorf("expected name=Updated Name, got %v", receivedBody["name"])
	}
	if receivedBody["description"] != "updated via stdin" {
		t.Errorf("expected description=updated via stdin, got %v", receivedBody["description"])
	}
}

func TestStdin_DryRunShowsStdinBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not be made in dry-run mode")
	}))
	defer srv.Close()

	old := stdinReader
	defer func() { stdinReader = old }()
	stdinReader = strings.NewReader(`{"name": "Stdin Name", "priority": 1}`)

	stdinFlag := true
	dryRun := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStdin:        &stdinFlag,
		FlagDryRun:       &dryRun,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
	}

	parsed := &ParsedArgs{
		Values: map[string]string{},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No HTTP call should have been made (dry-run), test passes if we get here
}

func TestStdin_InvalidJSON_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not be made with invalid stdin JSON")
	}))
	defer srv.Close()

	old := stdinReader
	defer func() { stdinReader = old }()
	stdinReader = strings.NewReader(`{invalid json}`)

	stdinFlag := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStdin:        &stdinFlag,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
	}

	parsed := &ParsedArgs{
		Values: map[string]string{},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' error, got: %v", err)
	}
}

func TestStdin_NameResolution(t *testing.T) {
	stateUUID := "550e8400-e29b-41d4-a716-446655440000"

	var receivedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// State resolution endpoint
		if strings.Contains(r.URL.Path, "/states/") && r.Method == "GET" {
			fmt.Fprintf(w, `{"results": [{"id": "%s", "name": "In Progress"}]}`, stateUUID)
			return
		}
		// Create endpoint
		if r.Method == "POST" {
			body, err := readJSONBody(r)
			if err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			receivedBody = body
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id": "new-uuid"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// stdin provides state by name (not UUID)
	old := stdinReader
	defer func() { stdinReader = old }()
	stdinReader = strings.NewReader(`{"name": "Test Issue", "state_id": "In Progress"}`)

	stdinFlag := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStdin:        &stdinFlag,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
			{Name: "state_id", Location: docs.ParamBody, Type: "string"},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// state_id should have been resolved from "In Progress" to UUID
	if receivedBody["state_id"] != stateUUID {
		t.Errorf("expected state_id to be resolved to %s, got %v", stateUUID, receivedBody["state_id"])
	}
}
