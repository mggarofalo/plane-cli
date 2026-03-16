package cmdgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/output"
)

func TestExtractRelationParams(t *testing.T) {
	t.Run("extracts module and cycle from body", func(t *testing.T) {
		body := map[string]any{
			"name":   "Test Issue",
			"module": "module-uuid-123",
			"cycle":  "cycle-uuid-456",
			"state":  "state-uuid",
		}

		relations := ExtractRelationParams(body)

		if relations["module"] != "module-uuid-123" {
			t.Errorf("expected module=module-uuid-123, got %q", relations["module"])
		}
		if relations["cycle"] != "cycle-uuid-456" {
			t.Errorf("expected cycle=cycle-uuid-456, got %q", relations["cycle"])
		}

		// Verify they were removed from body
		if _, ok := body["module"]; ok {
			t.Error("module should have been removed from body")
		}
		if _, ok := body["cycle"]; ok {
			t.Error("cycle should have been removed from body")
		}

		// Verify other fields are untouched
		if body["name"] != "Test Issue" {
			t.Error("name should be untouched")
		}
		if body["state"] != "state-uuid" {
			t.Error("state should be untouched")
		}
	})

	t.Run("returns nil when no relations present", func(t *testing.T) {
		body := map[string]any{
			"name":  "Test Issue",
			"state": "state-uuid",
		}

		relations := ExtractRelationParams(body)

		if relations != nil {
			t.Errorf("expected nil, got %v", relations)
		}
	})

	t.Run("handles nil body", func(t *testing.T) {
		relations := ExtractRelationParams(nil)

		if relations != nil {
			t.Errorf("expected nil, got %v", relations)
		}
	})

	t.Run("ignores non-string values", func(t *testing.T) {
		body := map[string]any{
			"name":   "Test Issue",
			"module": 12345,
		}

		relations := ExtractRelationParams(body)

		if relations != nil {
			t.Errorf("expected nil, got %v", relations)
		}

		// Non-string module should remain in body (not extracted)
		if _, ok := body["module"]; !ok {
			t.Error("non-string module should remain in body")
		}
	})

	t.Run("ignores empty string values", func(t *testing.T) {
		body := map[string]any{
			"name":   "Test Issue",
			"module": "",
		}

		relations := ExtractRelationParams(body)

		if relations != nil {
			t.Errorf("expected nil, got %v", relations)
		}
	})

	t.Run("extracts only module when cycle absent", func(t *testing.T) {
		body := map[string]any{
			"name":   "Test Issue",
			"module": "module-uuid-123",
		}

		relations := ExtractRelationParams(body)

		if relations["module"] != "module-uuid-123" {
			t.Errorf("expected module=module-uuid-123, got %q", relations["module"])
		}
		if _, ok := relations["cycle"]; ok {
			t.Error("cycle should not be present")
		}
	})
}

func TestExtractCreatedID(t *testing.T) {
	t.Run("extracts id from valid response", func(t *testing.T) {
		resp := []byte(`{"id": "abc-123", "name": "Test Issue"}`)

		id, err := ExtractCreatedID(resp)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "abc-123" {
			t.Errorf("expected id=abc-123, got %q", id)
		}
	})

	t.Run("returns error for missing id", func(t *testing.T) {
		resp := []byte(`{"name": "Test Issue"}`)

		_, err := ExtractCreatedID(resp)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		resp := []byte(`not json`)

		_, err := ExtractCreatedID(resp)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty id", func(t *testing.T) {
		resp := []byte(`{"id": ""}`)

		_, err := ExtractCreatedID(resp)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for non-string id", func(t *testing.T) {
		resp := []byte(`{"id": 12345}`)

		_, err := ExtractCreatedID(resp)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestResolveSliceIfNeeded(t *testing.T) {
	uuid1 := "550e8400-e29b-41d4-a716-446655440000"
	uuid2 := "660e8400-e29b-41d4-a716-446655440000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/workspaces/test-ws/work-items/PROJ-1/":
			fmt.Fprintf(w, `{"id": "%s"}`, uuid1)
		case "/api/v1/workspaces/test-ws/work-items/PROJ-2/":
			fmt.Fprintf(w, `{"id": "%s"}`, uuid2)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"detail": "not found"}`)
		}
	}))
	defer srv.Close()

	isUUID := func(s string) bool { return len(s) == 36 && s[8] == '-' }

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		IsUUID:           isUUID,
	}

	t.Run("resolves sequence IDs in slice", func(t *testing.T) {
		input := []string{"PROJ-1", "PROJ-2"}
		result, err := resolveSliceIfNeeded(context.Background(), input, "issues", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result[0] != uuid1 {
			t.Errorf("expected %s, got %s", uuid1, result[0])
		}
		if result[1] != uuid2 {
			t.Errorf("expected %s, got %s", uuid2, result[1])
		}
	})

	t.Run("passes through UUIDs unchanged", func(t *testing.T) {
		input := []string{uuid1}
		result, err := resolveSliceIfNeeded(context.Background(), input, "issues", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result[0] != uuid1 {
			t.Errorf("expected %s, got %s", uuid1, result[0])
		}
	})

	t.Run("leaves unresolvable values unchanged", func(t *testing.T) {
		input := []string{"PROJ-999"}
		result, err := resolveSliceIfNeeded(context.Background(), input, "issues", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result[0] != "PROJ-999" {
			t.Errorf("expected PROJ-999, got %s", result[0])
		}
	})
}

func TestResolveIfNeeded_SequenceID(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/workspaces/test-ws/work-items/PROJ-42/" {
			fmt.Fprintf(w, `{"id": "%s"}`, uuid)
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
		IsUUID:           isUUID,
	}

	t.Run("resolves sequence ID for work_item_id", func(t *testing.T) {
		result, err := resolveIfNeeded(context.Background(), "PROJ-42", "work_item_id", nil, "", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != uuid {
			t.Errorf("expected %s, got %s", uuid, result)
		}
	})

	t.Run("resolves sequence ID for parent", func(t *testing.T) {
		result, err := resolveIfNeeded(context.Background(), "PROJ-42", "parent", nil, "", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != uuid {
			t.Errorf("expected %s, got %s", uuid, result)
		}
	})

	t.Run("does not resolve sequence ID for non-issue params", func(t *testing.T) {
		result, err := resolveIfNeeded(context.Background(), "PROJ-42", "state", nil, "", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should not resolve — "state" is not an issue ref param, and PROJ-42 is not a state name
		if result == uuid {
			t.Error("should not have resolved sequence ID for state param")
		}
	})

	t.Run("passes through UUIDs", func(t *testing.T) {
		result, err := resolveIfNeeded(context.Background(), uuid, "work_item_id", nil, "", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != uuid {
			t.Errorf("expected %s, got %s", uuid, result)
		}
	})
}

func TestIsIDOnly(t *testing.T) {
	t.Run("returns false when nil", func(t *testing.T) {
		deps := &Deps{}
		if isIDOnly(deps) {
			t.Error("expected false for nil FlagIDOnly")
		}
	})

	t.Run("returns false when false", func(t *testing.T) {
		f := false
		deps := &Deps{FlagIDOnly: &f}
		if isIDOnly(deps) {
			t.Error("expected false")
		}
	})

	t.Run("returns true when true", func(t *testing.T) {
		f := true
		deps := &Deps{FlagIDOnly: &f}
		if !isIDOnly(deps) {
			t.Error("expected true")
		}
	})

	t.Run("returns false when deps is nil", func(t *testing.T) {
		if isIDOnly(nil) {
			t.Error("expected false for nil deps")
		}
	})
}

func TestIsDryRun(t *testing.T) {
	t.Run("returns false when nil", func(t *testing.T) {
		deps := &Deps{}
		if isDryRun(deps) {
			t.Error("expected false for nil FlagDryRun")
		}
	})

	t.Run("returns false when false", func(t *testing.T) {
		f := false
		deps := &Deps{FlagDryRun: &f}
		if isDryRun(deps) {
			t.Error("expected false")
		}
	})

	t.Run("returns true when true", func(t *testing.T) {
		f := true
		deps := &Deps{FlagDryRun: &f}
		if !isDryRun(deps) {
			t.Error("expected true")
		}
	})
}

func TestSnapshotBody(t *testing.T) {
	t.Run("returns nil for nil body", func(t *testing.T) {
		if snapshotBody(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("snapshot is independent of original", func(t *testing.T) {
		original := map[string]any{
			"name":   "Test",
			"module": "mod-uuid",
		}
		snap := snapshotBody(original)

		// Mutate original
		delete(original, "module")

		if _, ok := snap["module"]; !ok {
			t.Error("snapshot should still have 'module' after original was mutated")
		}
	})
}

func TestDryRun_POST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not be made in dry-run mode")
	}))
	defer srv.Close()

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
		FlagDryRun:       &dryRun,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"name": "Test Issue"},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDryRun_POST_WithRelations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not be made in dry-run mode")
	}))
	defer srv.Close()

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
		FlagDryRun:       &dryRun,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
			{Name: "module", Location: docs.ParamBody, Type: "string"},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"name": "Test Issue", "module": "550e8400-e29b-41d4-a716-446655440000"},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDryRun_DELETE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not be made in dry-run mode")
	}))
	defer srv.Close()

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
		FlagDryRun:       &dryRun,
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
}

func TestDryRun_GET_StillExecutes(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"results": [], "total_pages": 1}`)
	}))
	defer srv.Close()

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
		FlagDryRun:       &dryRun,
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
	if !called {
		t.Error("GET request should still be made even with dry-run flag")
	}
}

func TestPostCreateActions_GracefulWarnings(t *testing.T) {
	issueResp := []byte(`{"id": "issue-uuid-1", "name": "Test"}`)

	t.Run("warns on module attach failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "server error"}`)
		}))
		defer srv.Close()

		client := api.NewClient(srv.URL, "test-token", "test-ws", false, nil)
		relations := map[string]string{"module": "mod-uuid"}

		// Should not panic; warnings go to stderr
		PostCreateActions(context.Background(), relations, issueResp, client, "proj-uuid", &Deps{})
	})

	t.Run("warns on cycle attach failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error": "server error"}`)
		}))
		defer srv.Close()

		client := api.NewClient(srv.URL, "test-token", "test-ws", false, nil)
		relations := map[string]string{"cycle": "cyc-uuid"}

		PostCreateActions(context.Background(), relations, issueResp, client, "proj-uuid", &Deps{})
	})

	t.Run("warns on extractCreatedID failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("should not make API calls when ID extraction fails")
		}))
		defer srv.Close()

		client := api.NewClient(srv.URL, "test-token", "test-ws", false, nil)
		relations := map[string]string{"module": "mod-uuid"}

		PostCreateActions(context.Background(), relations, []byte(`not json`), client, "proj-uuid", &Deps{})
	})

	t.Run("succeeds when endpoints return 200", func(t *testing.T) {
		var moduleCalled, cycleCalled bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/modules/mod-uuid/module-issues/" {
				moduleCalled = true
			}
			if r.URL.Path == "/api/v1/workspaces/test-ws/projects/proj-uuid/cycles/cyc-uuid/cycle-issues/" {
				cycleCalled = true
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		}))
		defer srv.Close()

		client := api.NewClient(srv.URL, "test-token", "test-ws", false, nil)
		relations := map[string]string{"module": "mod-uuid", "cycle": "cyc-uuid"}

		PostCreateActions(context.Background(), relations, issueResp, client, "proj-uuid", &Deps{})

		if !moduleCalled {
			t.Error("expected module endpoint to be called")
		}
		if !cycleCalled {
			t.Error("expected cycle endpoint to be called")
		}
	})
}

func TestIsQuiet(t *testing.T) {
	t.Run("returns false when deps is nil", func(t *testing.T) {
		if IsQuiet(nil) {
			t.Error("expected false for nil deps")
		}
	})

	t.Run("returns false when FlagQuiet is nil", func(t *testing.T) {
		deps := &Deps{}
		if IsQuiet(deps) {
			t.Error("expected false for nil FlagQuiet")
		}
	})

	t.Run("returns false when false", func(t *testing.T) {
		f := false
		deps := &Deps{FlagQuiet: &f}
		if IsQuiet(deps) {
			t.Error("expected false")
		}
	})

	t.Run("returns true when true", func(t *testing.T) {
		f := true
		deps := &Deps{FlagQuiet: &f}
		if !IsQuiet(deps) {
			t.Error("expected true")
		}
	})
}

func TestInfof_Quiet(t *testing.T) {
	t.Run("suppresses output when quiet", func(t *testing.T) {
		q := true
		deps := &Deps{FlagQuiet: &q}
		// Infof writes to os.Stderr; in quiet mode it should be a no-op.
		// If it panics or errors, the test fails. We can't easily capture
		// stderr in a unit test, but we verify it doesn't crash.
		Infof(deps, "should not appear: %s\n", "test")
	})

	t.Run("does not suppress when not quiet", func(t *testing.T) {
		q := false
		deps := &Deps{FlagQuiet: &q}
		Infof(deps, "should appear: %s\n", "test")
	})

	t.Run("does not suppress when nil", func(t *testing.T) {
		deps := &Deps{}
		Infof(deps, "should appear: %s\n", "test")
	})
}

func TestQuiet_DELETE_SuppressesDeleted(t *testing.T) {
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

	quiet := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagQuiet:        &quiet,
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

func TestIsStrict(t *testing.T) {
	t.Run("returns false when deps is nil", func(t *testing.T) {
		if IsStrict(nil) {
			t.Error("expected false for nil deps")
		}
	})

	t.Run("returns false when FlagStrict is nil", func(t *testing.T) {
		deps := &Deps{}
		if IsStrict(deps) {
			t.Error("expected false for nil FlagStrict")
		}
	})

	t.Run("returns false when false", func(t *testing.T) {
		f := false
		deps := &Deps{FlagStrict: &f}
		if IsStrict(deps) {
			t.Error("expected false")
		}
	})

	t.Run("returns true when true", func(t *testing.T) {
		f := true
		deps := &Deps{FlagStrict: &f}
		if !IsStrict(deps) {
			t.Error("expected true")
		}
	})
}

func TestResolutionError_ExitCode(t *testing.T) {
	err := &ResolutionError{Msg: "could not resolve \"In Progrss\" for state"}
	if err.ExitCode() != api.ExitValidation {
		t.Errorf("expected exit code %d, got %d", api.ExitValidation, err.ExitCode())
	}
	if err.Error() != "could not resolve \"In Progrss\" for state" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestWarnOrFailResolution_NoStrict(t *testing.T) {
	// Without strict: should return the literal value and no error
	deps := &Deps{}
	val, err := warnOrFailResolution("In Progrss", "state", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "In Progrss" {
		t.Errorf("expected literal value, got %q", val)
	}
}

func TestWarnOrFailResolution_Strict(t *testing.T) {
	strict := true
	deps := &Deps{FlagStrict: &strict}
	_, err := warnOrFailResolution("In Progrss", "state", deps)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	resErr, ok := err.(*ResolutionError)
	if !ok {
		t.Fatalf("expected *ResolutionError, got %T", err)
	}
	if resErr.ExitCode() != api.ExitValidation {
		t.Errorf("expected exit code %d, got %d", api.ExitValidation, resErr.ExitCode())
	}
}

func TestResolveIfNeeded_WarnOnNameFailure(t *testing.T) {
	// Server returns no matching state
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a list with states that don't match "In Progrss"
		fmt.Fprint(w, `{"results": [{"id": "aaa", "name": "In Progress"}, {"id": "bbb", "name": "Done"}]}`)
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

	// Without strict: returns literal value
	result, err := resolveIfNeeded(context.Background(), "In Progrss", "state", nil, "", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "In Progrss" {
		t.Errorf("expected literal value, got %q", result)
	}
}

func TestResolveIfNeeded_StrictFailsOnNameMismatch(t *testing.T) {
	// Server returns no matching state
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"results": [{"id": "aaa", "name": "In Progress"}, {"id": "bbb", "name": "Done"}]}`)
	}))
	defer srv.Close()

	isUUID := func(s string) bool { return len(s) == 36 && s[8] == '-' }
	strict := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		IsUUID:           isUUID,
		FlagStrict:       &strict,
	}

	// With strict: returns error
	_, err := resolveIfNeeded(context.Background(), "In Progrss", "state", nil, "", deps)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*ResolutionError); !ok {
		t.Fatalf("expected *ResolutionError, got %T", err)
	}
}

func TestResolveIfNeeded_StrictFailsOnSequenceID(t *testing.T) {
	// Server returns 404 for the sequence ID
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"detail": "not found"}`)
	}))
	defer srv.Close()

	isUUID := func(s string) bool { return len(s) == 36 && s[8] == '-' }
	strict := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		IsUUID:           isUUID,
		FlagStrict:       &strict,
	}

	// With strict: returns error for unresolvable sequence ID
	_, err := resolveIfNeeded(context.Background(), "PROJ-999", "work_item_id", nil, "", deps)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*ResolutionError); !ok {
		t.Fatalf("expected *ResolutionError, got %T", err)
	}
}

func TestResolveIfNeeded_WarnOnSequenceIDFailure(t *testing.T) {
	// Server returns 404
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"detail": "not found"}`)
	}))
	defer srv.Close()

	isUUID := func(s string) bool { return len(s) == 36 && s[8] == '-' }
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		IsUUID:           isUUID,
	}

	// Without strict: returns literal value (with warning to stderr)
	result, err := resolveIfNeeded(context.Background(), "PROJ-999", "work_item_id", nil, "", deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "PROJ-999" {
		t.Errorf("expected PROJ-999, got %s", result)
	}
}

func TestResolveSliceIfNeeded_StrictFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"detail": "not found"}`)
	}))
	defer srv.Close()

	isUUID := func(s string) bool { return len(s) == 36 && s[8] == '-' }
	strict := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		IsUUID:           isUUID,
		FlagStrict:       &strict,
	}

	// With strict: should error on unresolvable sequence ID in slice
	_, err := resolveSliceIfNeeded(context.Background(), []string{"PROJ-999"}, "issues", deps)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*ResolutionError); !ok {
		t.Fatalf("expected *ResolutionError, got %T", err)
	}
}

func TestStrict_POST_FailsOnResolutionError(t *testing.T) {
	// Server returns empty list of states (so resolution fails)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"results": []}`)
	}))
	defer srv.Close()

	strict := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagStrict:       &strict,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
			{Name: "state", Location: docs.ParamBody, Type: "string"},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"name": "Test Issue", "state": "In Progrss"},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := err.(*ResolutionError); !ok {
		t.Fatalf("expected *ResolutionError, got %T: %v", err, err)
	}
}

func TestIDOnly_POST(t *testing.T) {
	issueUUID := "550e8400-e29b-41d4-a716-446655440000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": "%s", "name": "Test Issue", "state": "todo"}`, issueUUID)
	}))
	defer srv.Close()

	idOnly := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagIDOnly:       &idOnly,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"name": "Test Issue"},
		Slices: map[string][]string{},
	}

	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if got != issueUUID {
		t.Errorf("expected %q, got %q", issueUUID, got)
	}

	// Verify no trailing newline
	if len(got) > 0 && got[len(got)-1] == '\n' {
		t.Error("output should not end with a newline")
	}
}

func TestIDOnly_GET(t *testing.T) {
	issueUUID := "660e8400-e29b-41d4-a716-446655440000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id": "%s", "name": "Existing Issue", "state": "done"}`, issueUUID)
	}))
	defer srv.Close()

	idOnly := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagIDOnly:       &idOnly,
	}

	spec := &docs.EndpointSpec{
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
		EntryTitle:   "Get Work Item Detail",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Location: docs.ParamPath, Type: "string", Required: true},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"work-item-id": issueUUID},
		Slices: map[string][]string{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	if got != issueUUID {
		t.Errorf("expected %q, got %q", issueUUID, got)
	}
}

func TestExitCodeFromError_ResolutionError(t *testing.T) {
	err := &ResolutionError{Msg: "test"}
	code := api.ExitCodeFromError(err)
	if code != api.ExitValidation {
		t.Errorf("expected exit code %d, got %d", api.ExitValidation, code)
	}
}

func TestIsNoResolve(t *testing.T) {
	t.Run("returns false when deps is nil", func(t *testing.T) {
		if IsNoResolve(nil) {
			t.Error("expected false for nil deps")
		}
	})

	t.Run("returns false when FlagNoResolve is nil", func(t *testing.T) {
		deps := &Deps{}
		if IsNoResolve(deps) {
			t.Error("expected false for nil FlagNoResolve")
		}
	})

	t.Run("returns false when false", func(t *testing.T) {
		f := false
		deps := &Deps{FlagNoResolve: &f}
		if IsNoResolve(deps) {
			t.Error("expected false")
		}
	})

	t.Run("returns true when true", func(t *testing.T) {
		f := true
		deps := &Deps{FlagNoResolve: &f}
		if !IsNoResolve(deps) {
			t.Error("expected true")
		}
	})
}

func TestResolveIfNeeded_NoResolve(t *testing.T) {
	// Server should never be called when --no-resolve is active
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not be made when --no-resolve is active")
	}))
	defer srv.Close()

	noResolve := true
	isUUID := func(s string) bool { return len(s) == 36 && s[8] == '-' }
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		IsUUID:           isUUID,
		FlagNoResolve:    &noResolve,
	}

	t.Run("passes through non-UUID for _id param", func(t *testing.T) {
		result, err := resolveIfNeeded(context.Background(), "my-state-name", "state_id", nil, "", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "my-state-name" {
			t.Errorf("expected literal value, got %q", result)
		}
	})

	t.Run("passes through sequence ID without resolving", func(t *testing.T) {
		result, err := resolveIfNeeded(context.Background(), "PROJ-42", "work_item_id", nil, "", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "PROJ-42" {
			t.Errorf("expected PROJ-42, got %q", result)
		}
	})

	t.Run("passes through resolvable param name", func(t *testing.T) {
		result, err := resolveIfNeeded(context.Background(), "In Progress", "state", nil, "", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "In Progress" {
			t.Errorf("expected literal value, got %q", result)
		}
	})
}

func TestResolveSliceIfNeeded_NoResolve(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("HTTP request should not be made when --no-resolve is active")
	}))
	defer srv.Close()

	noResolve := true
	isUUID := func(s string) bool { return len(s) == 36 && s[8] == '-' }
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		IsUUID:           isUUID,
		FlagNoResolve:    &noResolve,
	}

	t.Run("passes through sequence IDs without resolving", func(t *testing.T) {
		input := []string{"PROJ-1", "PROJ-2"}
		result, err := resolveSliceIfNeeded(context.Background(), input, "issues", deps)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result[0] != "PROJ-1" || result[1] != "PROJ-2" {
			t.Errorf("expected literal values, got %v", result)
		}
	})
}

func TestNoResolve_POST_SkipsResolution(t *testing.T) {
	requestReceived := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": "new-uuid", "name": "Test"}`)
	}))
	defer srv.Close()

	noResolve := true
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagNoResolve:    &noResolve,
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
			{Name: "state", Location: docs.ParamBody, Type: "string"},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"name": "Test Issue", "state": "In Progress"},
		Slices: map[string][]string{},
	}

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !requestReceived {
		t.Error("expected POST request to be made")
	}
}

func TestWebURL_InjectedInGET(t *testing.T) {
	issueUUID := "550e8400-e29b-41d4-a716-446655440000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id": "%s", "name": "Test Issue", "state": "done"}`, issueUUID)
	}))
	defer srv.Close()

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
	}

	spec := &docs.EndpointSpec{
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
		EntryTitle:   "Get Work Item Detail",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Location: docs.ParamPath, Type: "string", Required: true},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"work-item-id": issueUUID},
		Slices: map[string][]string{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Parse the JSON output to check for web_url
	var result map[string]any
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, got)
	}

	webURL, ok := result["web_url"].(string)
	if !ok {
		t.Fatalf("web_url not found in output. Got: %s", got)
	}

	expected := srv.URL + "/test-ws/projects/proj-uuid/issues/" + issueUUID
	if webURL != expected {
		t.Errorf("expected web_url=%q, got %q", expected, webURL)
	}
}

func TestWebURL_InjectedInPOST(t *testing.T) {
	issueUUID := "660e8400-e29b-41d4-a716-446655440000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id": "%s", "name": "New Issue"}`, issueUUID)
	}))
	defer srv.Close()

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
	}

	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		EntryTitle:   "Create Work Item",
		Params: []docs.ParamSpec{
			{Name: "name", Location: docs.ParamBody, Type: "string"},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"name": "New Issue"},
		Slices: map[string][]string{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// POST to a list URL returns a single created resource; web_url should be injected
	var result map[string]any
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, got)
	}

	webURL, ok := result["web_url"].(string)
	if !ok {
		t.Fatalf("web_url should be injected for POST create response. Got: %s", got)
	}

	expected := srv.URL + "/test-ws/projects/proj-uuid/issues/" + issueUUID
	if webURL != expected {
		t.Errorf("expected web_url=%q, got %q", expected, webURL)
	}
}

func TestWebURL_NotInjectedForList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"results": [{"id": "a1"}, {"id": "a2"}], "total_count": 2}`)
	}))
	defer srv.Close()

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
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

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	// Should NOT have web_url because this is a paginated list
	var result map[string]any
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, got)
	}

	if _, ok := result["web_url"]; ok {
		t.Error("web_url should not be injected into paginated list responses")
	}
}

func TestWebURL_NotInjectedWhenAlreadyPresent(t *testing.T) {
	issueUUID := "550e8400-e29b-41d4-a716-446655440000"
	existingURL := "https://existing.plane.so/my-ws/projects/proj/issues/" + issueUUID
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id": "%s", "name": "Test", "web_url": "%s"}`, issueUUID, existingURL)
	}))
	defer srv.Close()

	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
	}

	spec := &docs.EndpointSpec{
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
		EntryTitle:   "Get Work Item Detail",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Location: docs.ParamPath, Type: "string", Required: true},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"work-item-id": issueUUID},
		Slices: map[string][]string{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := buf.String()

	var result map[string]any
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, got)
	}

	webURL, ok := result["web_url"].(string)
	if !ok {
		t.Fatal("web_url should be present")
	}

	// Should preserve the existing URL, not replace it
	if webURL != existingURL {
		t.Errorf("expected existing web_url=%q, got %q", existingURL, webURL)
	}
}

func TestWebURL_ExtractableViaFieldFlag(t *testing.T) {
	issueUUID := "550e8400-e29b-41d4-a716-446655440000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id": "%s", "name": "Test Issue"}`, issueUUID)
	}))
	defer srv.Close()

	fieldFlag := "web_url"
	deps := &Deps{
		NewClient: func() (*api.Client, error) {
			return api.NewClient(srv.URL, "test-token", "test-ws", false, nil), nil
		},
		RequireWorkspace: func(c *api.Client) error { return nil },
		RequireProject:   func() (string, error) { return "proj-uuid", nil },
		PaginationParams: func() api.PaginationParams { return api.PaginationParams{PerPage: 100} },
		Formatter:        func() output.Formatter { return output.New("json") },
		IsUUID:           func(s string) bool { return len(s) == 36 && s[8] == '-' },
		FlagField:        &fieldFlag,
	}

	spec := &docs.EndpointSpec{
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
		EntryTitle:   "Get Work Item Detail",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Location: docs.ParamPath, Type: "string", Required: true},
		},
	}

	parsed := &ParsedArgs{
		Values: map[string]string{"work-item-id": issueUUID},
		Slices: map[string][]string{},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ExecuteSpecFromArgs(context.Background(), spec, parsed, deps)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	got := strings.TrimSpace(buf.String())

	expected := srv.URL + "/test-ws/projects/proj-uuid/issues/" + issueUUID
	if got != expected {
		t.Errorf("expected --field web_url to output %q, got %q", expected, got)
	}
}

func TestGenerateHelp_ResolvableAnnotations(t *testing.T) {
	spec := &docs.EndpointSpec{
		TopicName:    "issue",
		EntryTitle:   "Create Work Item",
		SourceURL:    "https://example.com",
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/",
		Params: []docs.ParamSpec{
			{Name: "name", Type: "string", Description: "Issue title"},
			{Name: "state_id", Type: "string", Description: "UUID of the state"},
			{Name: "label_id", Type: "string", Description: "UUID of the label"},
			{Name: "state", Type: "string", Description: "State of the issue"},
			{Name: "priority", Type: "string", Description: "Issue priority"},
		},
	}

	var buf bytes.Buffer
	GenerateHelp(&buf, "issue", "create", spec)
	output := buf.String()

	// Resolvable params should have "(accepts name or UUID)" annotation
	if !strings.Contains(output, "UUID of the state (accepts name or UUID)") {
		t.Errorf("expected state_id to have name-resolvable annotation, got:\n%s", output)
	}
	if !strings.Contains(output, "UUID of the label (accepts name or UUID)") {
		t.Errorf("expected label_id to have name-resolvable annotation, got:\n%s", output)
	}
	if !strings.Contains(output, "State of the issue (accepts name or UUID)") {
		t.Errorf("expected state (resolvableParams entry) to have annotation, got:\n%s", output)
	}

	// Non-resolvable params should NOT have the annotation
	if strings.Contains(output, "Issue title (accepts name or UUID)") {
		t.Error("name param should not have resolvable annotation")
	}
	if strings.Contains(output, "Issue priority (accepts name or UUID)") {
		t.Error("priority param should not have resolvable annotation")
	}

	// Name resolution summary section should be present
	if !strings.Contains(output, "Name resolution:") {
		t.Error("expected 'Name resolution:' summary section in help output")
	}
	if !strings.Contains(output, "Flags that accept names:") {
		t.Error("expected 'Flags that accept names:' in resolution summary")
	}
	if !strings.Contains(output, "plane resolution") {
		t.Error("expected reference to 'plane resolution' command")
	}
}

func TestGenerateHelp_SequenceIDAnnotations(t *testing.T) {
	spec := &docs.EndpointSpec{
		TopicName:    "issue",
		EntryTitle:   "Get Work Item Detail",
		SourceURL:    "https://example.com",
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/{work_item_id}/",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Type: "string", Required: true, Description: "Work item identifier", Location: "path"},
			{Name: "parent", Type: "string", Description: "Parent issue"},
			{Name: "issues", Type: "string[]", Description: "Related issues"},
		},
	}

	var buf bytes.Buffer
	GenerateHelp(&buf, "issue", "get", spec)
	output := buf.String()

	// issueRefParams should have sequence ID annotation
	if !strings.Contains(output, "(accepts UUID or sequence ID, e.g. PROJ-42)") {
		t.Errorf("expected sequence ID annotation for work_item_id, got:\n%s", output)
	}

	// Should appear for parent and issues too
	parentLine := false
	issuesLine := false
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "--parent") && strings.Contains(line, "sequence ID") {
			parentLine = true
		}
		if strings.Contains(line, "--issues") && strings.Contains(line, "sequence ID") {
			issuesLine = true
		}
	}
	if !parentLine {
		t.Error("expected sequence ID annotation for --parent flag")
	}
	if !issuesLine {
		t.Error("expected sequence ID annotation for --issues flag")
	}

	// Summary should mention sequence ID flags
	if !strings.Contains(output, "Flags that accept sequence IDs") {
		t.Error("expected sequence ID summary in Name resolution section")
	}
}

func TestGenerateHelp_NoResolutionSection_WhenNoResolvableParams(t *testing.T) {
	spec := &docs.EndpointSpec{
		TopicName:    "project",
		EntryTitle:   "List Projects",
		SourceURL:    "https://example.com",
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/",
		Params: []docs.ParamSpec{
			{Name: "name", Type: "string", Description: "Filter by name"},
			{Name: "sort_by", Type: "string", Description: "Sort order"},
		},
	}

	var buf bytes.Buffer
	GenerateHelp(&buf, "project", "list", spec)
	output := buf.String()

	if strings.Contains(output, "Name resolution:") {
		t.Error("should not show Name resolution section when no params are resolvable")
	}
}

func TestIsResolvableParam(t *testing.T) {
	tests := []struct {
		param string
		want  bool
	}{
		{"state", true},      // resolvableParams entry
		{"module", true},     // resolvableParams entry
		{"cycle", true},      // resolvableParams entry
		{"label", true},      // resolvableParams entry
		{"state_id", true},   // _id suffix, known kind
		{"label_id", true},   // _id suffix, known kind
		{"member_id", true},  // _id suffix, known kind
		{"name", false},      // not resolvable
		{"priority", false},  // not resolvable
		{"random_id", false}, // _id suffix but unknown kind
	}

	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			got := IsResolvableParam(tt.param)
			if got != tt.want {
				t.Errorf("IsResolvableParam(%q) = %v, want %v", tt.param, got, tt.want)
			}
		})
	}
}

func TestIsIssueRefParam(t *testing.T) {
	tests := []struct {
		param string
		want  bool
	}{
		{"work_item_id", true},
		{"parent", true},
		{"issues", true},
		{"state_id", false},
		{"name", false},
	}

	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			got := IsIssueRefParam(tt.param)
			if got != tt.want {
				t.Errorf("IsIssueRefParam(%q) = %v, want %v", tt.param, got, tt.want)
			}
		})
	}
}
