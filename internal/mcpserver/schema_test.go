package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

func TestBuildInputSchema_BasicTypes(t *testing.T) {
	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/",
		Params: []docs.ParamSpec{
			{Name: "workspace_slug", Type: "string", Location: docs.ParamPath, Required: true},
			{Name: "project_id", Type: "string", Location: docs.ParamPath, Required: true},
			{Name: "name", Type: "string", Location: docs.ParamBody, Required: true, Description: "Issue name"},
			{Name: "priority", Type: "number", Location: docs.ParamBody, Description: "Priority level"},
			{Name: "is_draft", Type: "boolean", Location: docs.ParamBody, Description: "Draft status"},
			{Name: "labels", Type: "string[]", Location: docs.ParamBody, Description: "Label IDs"},
		},
	}

	raw := BuildInputSchema(spec)
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	// Should be object type
	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be a map")
	}

	// workspace_slug and project_id should be hidden
	if _, exists := props["workspace_slug"]; exists {
		t.Error("workspace_slug should be hidden from schema")
	}
	if _, exists := props["project_id"]; exists {
		t.Error("project_id should be hidden from schema")
	}

	// But workspace/project override params should be present
	if _, exists := props["workspace"]; !exists {
		t.Error("workspace override param should be present")
	}
	if _, exists := props["project"]; !exists {
		t.Error("project override param should be present")
	}

	// Check name param
	nameProp, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatal("expected name property")
	}
	if nameProp["type"] != "string" {
		t.Errorf("expected name type 'string', got %v", nameProp["type"])
	}

	// Check priority is number
	priorityProp := props["priority"].(map[string]any)
	if priorityProp["type"] != "number" {
		t.Errorf("expected priority type 'number', got %v", priorityProp["type"])
	}

	// Check is_draft is boolean
	draftProp := props["is_draft"].(map[string]any)
	if draftProp["type"] != "boolean" {
		t.Errorf("expected is_draft type 'boolean', got %v", draftProp["type"])
	}

	// Check labels is array
	labelsProp := props["labels"].(map[string]any)
	if labelsProp["type"] != "array" {
		t.Errorf("expected labels type 'array', got %v", labelsProp["type"])
	}

	// Check required includes "name" but not hidden params
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatal("expected required to be an array")
	}
	found := false
	for _, r := range required {
		if r == "name" {
			found = true
		}
		if r == "workspace_slug" || r == "project_id" {
			t.Errorf("hidden param %v should not be in required", r)
		}
	}
	if !found {
		t.Error("expected 'name' in required array")
	}
}

func TestBuildInputSchema_GETPagination(t *testing.T) {
	spec := &docs.EndpointSpec{
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/",
		Params: []docs.ParamSpec{
			{Name: "workspace_slug", Type: "string", Location: docs.ParamPath},
			{Name: "project_id", Type: "string", Location: docs.ParamPath},
		},
	}

	raw := BuildInputSchema(spec)
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	props := schema["properties"].(map[string]any)

	// Should have pagination params
	for _, name := range []string{"page_size", "cursor", "all"} {
		if _, exists := props[name]; !exists {
			t.Errorf("expected pagination param %q in GET schema", name)
		}
	}
}

func TestBuildInputSchema_NoPaginationForPOST(t *testing.T) {
	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/",
		Params: []docs.ParamSpec{
			{Name: "workspace_slug", Type: "string", Location: docs.ParamPath},
			{Name: "project_id", Type: "string", Location: docs.ParamPath},
		},
	}

	raw := BuildInputSchema(spec)
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	props := schema["properties"].(map[string]any)

	for _, name := range []string{"page_size", "cursor", "all"} {
		if _, exists := props[name]; exists {
			t.Errorf("pagination param %q should not be in POST schema", name)
		}
	}
}

func TestBuildInputSchema_HTMLParam(t *testing.T) {
	spec := &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/test/",
		Params: []docs.ParamSpec{
			{Name: "description_html", Type: "string", Location: docs.ParamBody, Description: "Description content"},
		},
	}

	raw := BuildInputSchema(spec)
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	props := schema["properties"].(map[string]any)

	// Should expose as "description" (markdown), not "description_html"
	if _, exists := props["description_html"]; exists {
		t.Error("description_html should be exposed as 'description'")
	}
	if _, exists := props["description"]; !exists {
		t.Error("expected 'description' (markdown equivalent) in schema")
	}
}

func TestBuildInputSchema_NoWorkspaceProject(t *testing.T) {
	spec := &docs.EndpointSpec{
		Method:       "GET",
		PathTemplate: "/api/v1/users/me/",
		Params:       []docs.ParamSpec{},
	}

	raw := BuildInputSchema(spec)
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	props := schema["properties"].(map[string]any)

	if _, exists := props["workspace"]; exists {
		t.Error("workspace override should not be present for non-workspace endpoint")
	}
	if _, exists := props["project"]; exists {
		t.Error("project override should not be present for non-project endpoint")
	}
}
