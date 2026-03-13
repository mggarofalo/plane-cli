package cmdgen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/api"
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
