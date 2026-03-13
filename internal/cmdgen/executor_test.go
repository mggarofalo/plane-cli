package cmdgen

import (
	"testing"
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
