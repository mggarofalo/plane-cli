package weburl

import (
	"encoding/json"
	"testing"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		name         string
		apiBaseURL   string
		workspace    string
		projectID    string
		pathTemplate string
		resourceID   string
		expected     string
	}{
		{
			name:         "issue detail",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
			resourceID:   "issue-uuid",
			expected:     "https://plane.example.com/my-ws/projects/proj-uuid/issues/issue-uuid",
		},
		{
			name:         "cycle detail",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/cycles/{cycle_id}/",
			resourceID:   "cycle-uuid",
			expected:     "https://plane.example.com/my-ws/projects/proj-uuid/cycles/cycle-uuid",
		},
		{
			name:         "module detail",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/modules/{module_id}/",
			resourceID:   "mod-uuid",
			expected:     "https://plane.example.com/my-ws/projects/proj-uuid/modules/mod-uuid",
		},
		{
			name:         "page detail",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/pages/{page_id}/",
			resourceID:   "page-uuid",
			expected:     "https://plane.example.com/my-ws/projects/proj-uuid/pages/page-uuid",
		},
		{
			name:         "project detail",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/",
			resourceID:   "proj-uuid",
			expected:     "https://plane.example.com/my-ws/projects/proj-uuid/issues/",
		},
		{
			name:         "initiative detail",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/initiatives/{initiative_id}/",
			resourceID:   "init-uuid",
			expected:     "https://plane.example.com/my-ws/initiatives/init-uuid",
		},
		{
			name:         "cloud plane api.plane.so",
			apiBaseURL:   "https://api.plane.so",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
			resourceID:   "issue-uuid",
			expected:     "https://app.plane.so/my-ws/projects/proj-uuid/issues/issue-uuid",
		},
		{
			name:         "list endpoint also builds URL for create responses",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
			resourceID:   "issue-uuid",
			expected:     "https://plane.example.com/my-ws/projects/proj-uuid/issues/issue-uuid",
		},
		{
			name:         "empty workspace returns empty",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
			resourceID:   "issue-uuid",
			expected:     "",
		},
		{
			name:         "empty resourceID returns empty",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
			resourceID:   "",
			expected:     "",
		},
		{
			name:         "empty apiBaseURL returns empty",
			apiBaseURL:   "",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
			resourceID:   "issue-uuid",
			expected:     "",
		},
		{
			name:         "project-scoped resource without projectID returns empty",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
			resourceID:   "issue-uuid",
			expected:     "",
		},
		{
			name:         "unknown resource segment returns empty",
			apiBaseURL:   "https://plane.example.com",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/unknown-things/{thing_id}/",
			resourceID:   "thing-uuid",
			expected:     "",
		},
		{
			name:         "trailing slash on apiBaseURL",
			apiBaseURL:   "https://plane.example.com/",
			workspace:    "my-ws",
			projectID:    "proj-uuid",
			pathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/",
			resourceID:   "issue-uuid",
			expected:     "https://plane.example.com/my-ws/projects/proj-uuid/issues/issue-uuid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Build(tt.apiBaseURL, tt.workspace, tt.projectID, tt.pathTemplate, tt.resourceID)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestInject(t *testing.T) {
	t.Run("injects web_url into single object", func(t *testing.T) {
		resp := []byte(`{"id":"issue-uuid","name":"Test Issue"}`)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/")

		var obj map[string]any
		if err := json.Unmarshal(result, &obj); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		webURL, ok := obj["web_url"].(string)
		if !ok {
			t.Fatal("web_url not found in result")
		}
		expected := "https://plane.example.com/my-ws/projects/proj-uuid/issues/issue-uuid"
		if webURL != expected {
			t.Errorf("expected %q, got %q", expected, webURL)
		}
	})

	t.Run("skips when web_url already present", func(t *testing.T) {
		resp := []byte(`{"id":"issue-uuid","name":"Test","web_url":"https://existing.url"}`)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/")

		var obj map[string]any
		if err := json.Unmarshal(result, &obj); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if obj["web_url"] != "https://existing.url" {
			t.Errorf("web_url should not have been overwritten, got %v", obj["web_url"])
		}
	})

	t.Run("skips paginated envelope", func(t *testing.T) {
		resp := []byte(`{"results":[{"id":"a1"}],"total_count":1}`)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/")

		var obj map[string]any
		if err := json.Unmarshal(result, &obj); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if _, ok := obj["web_url"]; ok {
			t.Error("web_url should not be injected into paginated envelope")
		}
	})

	t.Run("skips array response", func(t *testing.T) {
		resp := []byte(`[{"id":"a1"},{"id":"a2"}]`)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/")

		// Should be returned unchanged
		if string(result) != string(resp) {
			t.Errorf("array response should be returned unchanged")
		}
	})

	t.Run("skips when no id field", func(t *testing.T) {
		resp := []byte(`{"name":"Test Issue"}`)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/")

		var obj map[string]any
		if err := json.Unmarshal(result, &obj); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if _, ok := obj["web_url"]; ok {
			t.Error("web_url should not be injected when id is missing")
		}
	})

	t.Run("skips invalid JSON", func(t *testing.T) {
		resp := []byte(`not json`)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/")

		if string(result) != string(resp) {
			t.Error("invalid JSON should be returned unchanged")
		}
	})

	t.Run("skips empty response", func(t *testing.T) {
		resp := []byte(``)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/")

		if string(result) != string(resp) {
			t.Error("empty response should be returned unchanged")
		}
	})

	t.Run("injects for list endpoint path template (create response)", func(t *testing.T) {
		resp := []byte(`{"id":"issue-uuid","name":"Test Issue"}`)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/")

		var obj map[string]any
		if err := json.Unmarshal(result, &obj); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		webURL, ok := obj["web_url"].(string)
		if !ok {
			t.Fatal("web_url should be injected for single object from list path")
		}
		expected := "https://plane.example.com/my-ws/projects/proj-uuid/issues/issue-uuid"
		if webURL != expected {
			t.Errorf("expected %q, got %q", expected, webURL)
		}
	})

	t.Run("preserves all existing fields", func(t *testing.T) {
		resp := []byte(`{"id":"issue-uuid","name":"Test Issue","priority":2,"archived":false}`)
		result := Inject(resp, "https://plane.example.com", "my-ws", "proj-uuid",
			"/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/")

		var obj map[string]any
		if err := json.Unmarshal(result, &obj); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}
		if obj["name"] != "Test Issue" {
			t.Error("name field should be preserved")
		}
		if obj["priority"] != float64(2) {
			t.Error("priority field should be preserved")
		}
		if obj["archived"] != false {
			t.Error("archived field should be preserved")
		}
	})
}

func TestResourceSegment(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"issue detail", "/api/v1/workspaces/{ws}/projects/{proj}/work-items/{id}/", "work-items"},
		{"issue list", "/api/v1/workspaces/{ws}/projects/{proj}/work-items/", "work-items"},
		{"cycle detail", "/api/v1/workspaces/{ws}/projects/{proj}/cycles/{id}/", "cycles"},
		{"project detail", "/api/v1/workspaces/{ws}/projects/{id}/", "projects"},
		{"initiative detail", "/api/v1/workspaces/{ws}/initiatives/{id}/", "initiatives"},
		{"nested sub-resource", "/api/v1/workspaces/{ws}/projects/{proj}/cycles/{cycle_id}/cycle-issues/", "cycle-issues"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resourceSegment(tt.path)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestToWebBase(t *testing.T) {
	tests := []struct {
		name     string
		apiURL   string
		expected string
	}{
		{"self-hosted", "https://plane.example.com", "https://plane.example.com"},
		{"cloud api", "https://api.plane.so", "https://app.plane.so"},
		{"trailing slash", "https://plane.example.com/", "https://plane.example.com"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toWebBase(tt.apiURL)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
