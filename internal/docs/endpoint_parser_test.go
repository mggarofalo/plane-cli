package docs

import (
	"testing"
)

func TestParseEndpointPage_CreateWorkItem(t *testing.T) {
	markdown := `
# Create Work Item

POST /api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/

## Request Body

| Name | Type | Required | Description |
|------|------|----------|-------------|
| name | string | Yes | The work item name |
| description_html | string | No | HTML description |
| priority | string | No | Priority: urgent, high, medium, low, none |
| state_id | uuid | No | State ID |
| assignee_ids | array | No | List of assignee UUIDs |
| label_ids | array | No | List of label UUIDs |
| start_date | date | No | Start date |
| target_date | date | No | Target date |

Response (201)
`

	entry := Entry{Title: "Create Work Item", URL: "https://developers.plane.so/api-reference/issue/add-issue"}
	spec := ParseEndpointPage(markdown, "issue", entry)

	if spec.Method != "POST" {
		t.Errorf("expected method POST, got %s", spec.Method)
	}
	if spec.PathTemplate != "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/" {
		t.Errorf("unexpected path: %s", spec.PathTemplate)
	}
	if spec.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", spec.StatusCode)
	}
	if !spec.RequiresWorkspace() {
		t.Error("expected RequiresWorkspace to be true")
	}
	if !spec.RequiresProject() {
		t.Error("expected RequiresProject to be true")
	}

	// Check we got body params (not workspace_slug or project_id)
	var foundName, foundState, foundAssignees bool
	for _, p := range spec.Params {
		switch p.Name {
		case "name":
			foundName = true
			if !p.Required {
				t.Error("expected 'name' to be required")
			}
			if p.Location != ParamBody {
				t.Errorf("expected 'name' location body, got %s", p.Location)
			}
		case "state_id":
			foundState = true
		case "assignee_ids":
			foundAssignees = true
			if p.Type != "string[]" {
				t.Errorf("expected assignee_ids type string[], got %s", p.Type)
			}
		}
	}
	if !foundName {
		t.Error("missing 'name' param")
	}
	if !foundState {
		t.Error("missing 'state_id' param")
	}
	if !foundAssignees {
		t.Error("missing 'assignee_ids' param")
	}
}

func TestParseEndpointPage_ListIssues(t *testing.T) {
	markdown := `
# List Work Items

GET /api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/

Returns a paginated list of work items.

Response (200)
`

	entry := Entry{Title: "List Work Items", URL: "https://developers.plane.so/api-reference/issue/list-issues"}
	spec := ParseEndpointPage(markdown, "issue", entry)

	if spec.Method != "GET" {
		t.Errorf("expected method GET, got %s", spec.Method)
	}
	if spec.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", spec.StatusCode)
	}
}

func TestParseEndpointPage_DeleteIssue(t *testing.T) {
	markdown := `
# Delete Work Item

DELETE /api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/

204 No Content
`

	entry := Entry{Title: "Delete Work Item", URL: "https://developers.plane.so/api-reference/issue/delete-issue"}
	spec := ParseEndpointPage(markdown, "issue", entry)

	if spec.Method != "DELETE" {
		t.Errorf("expected method DELETE, got %s", spec.Method)
	}
	if spec.StatusCode != 204 {
		t.Errorf("expected status 204, got %d", spec.StatusCode)
	}

	// Should have work_item_id as path param
	var found bool
	for _, p := range spec.Params {
		if p.Name == "work_item_id" {
			found = true
			if p.Location != ParamPath {
				t.Errorf("expected path location, got %s", p.Location)
			}
			if !p.Required {
				t.Error("expected work_item_id to be required")
			}
		}
	}
	if !found {
		t.Error("missing 'work_item_id' path param")
	}
}

func TestParseEndpointPage_FallbackMethodFromTitle(t *testing.T) {
	markdown := `Some doc page with no method+path pattern.`

	tests := []struct {
		title    string
		expected string
	}{
		{"Create Cycle", "POST"},
		{"Add Comment", "POST"},
		{"List Modules", "GET"},
		{"Get Module Detail", "GET"},
		{"Update Label", "PATCH"},
		{"Delete State", "DELETE"},
		{"Remove Work Item", "DELETE"},
		{"Search Work Items", "GET"},
	}

	for _, tt := range tests {
		entry := Entry{Title: tt.title, URL: "https://example.com"}
		spec := ParseEndpointPage(markdown, "test", entry)
		if spec.Method != tt.expected {
			t.Errorf("title %q: expected method %s, got %s", tt.title, tt.expected, spec.Method)
		}
	}
}

func TestInferStatusCode(t *testing.T) {
	if inferStatusCode("POST") != 201 {
		t.Error("POST should default to 201")
	}
	if inferStatusCode("DELETE") != 204 {
		t.Error("DELETE should default to 204")
	}
	if inferStatusCode("GET") != 200 {
		t.Error("GET should default to 200")
	}
	if inferStatusCode("PATCH") != 200 {
		t.Error("PATCH should default to 200")
	}
}

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"string", "string"},
		{"String", "string"},
		{"uuid", "string"},
		{"integer", "number"},
		{"int", "number"},
		{"number", "number"},
		{"boolean", "boolean"},
		{"bool", "boolean"},
		{"array", "string[]"},
		{"string[]", "string[]"},
		{"list", "string[]"},
		{"date", "string"},
	}

	for _, tt := range tests {
		got := normalizeType(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeType(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
