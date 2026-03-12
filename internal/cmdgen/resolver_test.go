package cmdgen

import (
	"encoding/json"
	"testing"
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
