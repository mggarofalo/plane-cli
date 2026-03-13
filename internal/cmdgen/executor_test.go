package cmdgen

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/api"
)

func TestExtractRelationParams(t *testing.T) {
	t.Run("extracts module and cycle from body", func(t *testing.T) {
		body := map[string]any{
			"name":   "Test Issue",
			"module": "module-uuid-123",
			"cycle":  "cycle-uuid-456",
			"state":  "state-uuid",
		}

		relations := extractRelationParams(body)

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

		relations := extractRelationParams(body)

		if relations != nil {
			t.Errorf("expected nil, got %v", relations)
		}
	})

	t.Run("handles nil body", func(t *testing.T) {
		relations := extractRelationParams(nil)

		if relations != nil {
			t.Errorf("expected nil, got %v", relations)
		}
	})

	t.Run("ignores non-string values", func(t *testing.T) {
		body := map[string]any{
			"name":   "Test Issue",
			"module": 12345,
		}

		relations := extractRelationParams(body)

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

		relations := extractRelationParams(body)

		if relations != nil {
			t.Errorf("expected nil, got %v", relations)
		}
	})

	t.Run("extracts only module when cycle absent", func(t *testing.T) {
		body := map[string]any{
			"name":   "Test Issue",
			"module": "module-uuid-123",
		}

		relations := extractRelationParams(body)

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

		id, err := extractCreatedID(resp)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "abc-123" {
			t.Errorf("expected id=abc-123, got %q", id)
		}
	})

	t.Run("returns error for missing id", func(t *testing.T) {
		resp := []byte(`{"name": "Test Issue"}`)

		_, err := extractCreatedID(resp)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		resp := []byte(`not json`)

		_, err := extractCreatedID(resp)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty id", func(t *testing.T) {
		resp := []byte(`{"id": ""}`)

		_, err := extractCreatedID(resp)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for non-string id", func(t *testing.T) {
		resp := []byte(`{"id": 12345}`)

		_, err := extractCreatedID(resp)

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
		result := resolveSliceIfNeeded(context.Background(), input, "issues", deps)
		if result[0] != uuid1 {
			t.Errorf("expected %s, got %s", uuid1, result[0])
		}
		if result[1] != uuid2 {
			t.Errorf("expected %s, got %s", uuid2, result[1])
		}
	})

	t.Run("passes through UUIDs unchanged", func(t *testing.T) {
		input := []string{uuid1}
		result := resolveSliceIfNeeded(context.Background(), input, "issues", deps)
		if result[0] != uuid1 {
			t.Errorf("expected %s, got %s", uuid1, result[0])
		}
	})

	t.Run("leaves unresolvable values unchanged", func(t *testing.T) {
		input := []string{"PROJ-999"}
		result := resolveSliceIfNeeded(context.Background(), input, "issues", deps)
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
		result := resolveIfNeeded(context.Background(), "PROJ-42", "work_item_id", nil, "", deps)
		if result != uuid {
			t.Errorf("expected %s, got %s", uuid, result)
		}
	})

	t.Run("resolves sequence ID for parent", func(t *testing.T) {
		result := resolveIfNeeded(context.Background(), "PROJ-42", "parent", nil, "", deps)
		if result != uuid {
			t.Errorf("expected %s, got %s", uuid, result)
		}
	})

	t.Run("does not resolve sequence ID for non-issue params", func(t *testing.T) {
		result := resolveIfNeeded(context.Background(), "PROJ-42", "state", nil, "", deps)
		// Should not resolve — "state" is not an issue ref param, and PROJ-42 is not a state name
		if result == uuid {
			t.Error("should not have resolved sequence ID for state param")
		}
	})

	t.Run("passes through UUIDs", func(t *testing.T) {
		result := resolveIfNeeded(context.Background(), uuid, "work_item_id", nil, "", deps)
		if result != uuid {
			t.Errorf("expected %s, got %s", uuid, result)
		}
	})
}
