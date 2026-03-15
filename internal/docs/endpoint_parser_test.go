package docs

import (
	"encoding/json"
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

func TestParseEndpointPage_InlineParamFormat(t *testing.T) {
	markdown := "# Create a work item\n" +
		"POST/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/\n" +
		"Creates a new work item.\n\n" +
		"### Path Parameters\n\n" +
		"`workspace_slug`:requiredstring\nThe workspace slug.\n\n" +
		"`project_id`:requiredstring\nThe project ID.\n\n" +
		"### Body Parameters\n\n" +
		"`name`:requiredstring\nName of the work item.\n\n" +
		"`description_html`:optionalstring\nHTML description.\n\n" +
		"`priority`:optionalstring\nPriority level.\n\n" +
		"`assignees`:optionalstring[]\nArray of user IDs.\n\n" +
		"`labels`:optionalstring[]\nArray of label IDs.\n\n" +
		"### Scopes\n\n" +
		"`projects.work_items:write`\n"

	entry := Entry{Title: "Create Work Item", URL: "https://developers.plane.so/api-reference/issue/add-issue"}
	spec := ParseEndpointPage(markdown, "issue", entry)

	if spec.Method != "POST" {
		t.Errorf("expected POST, got %s", spec.Method)
	}
	if spec.PathTemplate != "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/" {
		t.Errorf("unexpected path: %s", spec.PathTemplate)
	}

	// Should have body params but NOT path params (workspace_slug, project_id)
	bodyParams := 0
	for _, p := range spec.Params {
		if p.Location == ParamBody {
			bodyParams++
		}
		if p.Name == "workspace_slug" || p.Name == "project_id" {
			t.Errorf("should not include global path param %s", p.Name)
		}
	}
	if bodyParams < 3 {
		t.Errorf("expected at least 3 body params, got %d", bodyParams)
	}

	// Check specific params
	var foundName, foundAssignees bool
	for _, p := range spec.Params {
		switch p.Name {
		case "name":
			foundName = true
			if !p.Required {
				t.Error("name should be required")
			}
		case "assignees":
			foundAssignees = true
			if p.Type != "string[]" {
				t.Errorf("assignees should be string[], got %s", p.Type)
			}
		}
	}
	if !foundName {
		t.Error("missing name param")
	}
	if !foundAssignees {
		t.Error("missing assignees param")
	}
}

func TestExtractEnum(t *testing.T) {
	tests := []struct {
		desc     string
		expected []string
	}{
		{"Priority: urgent, high, medium, low, none", []string{"urgent", "high", "medium", "low", "none"}},
		{"one of: active, paused, completed", []string{"active", "paused", "completed"}},
		{"Values: open, closed", []string{"open", "closed"}},
		{"options: draft, published, archived", []string{"draft", "published", "archived"}},
		{"The work item name", nil},                             // no enum
		{"A single value", nil},                                 // no enum
		{"enum: started, stopped, pending", []string{"started", "stopped", "pending"}},
	}

	for _, tt := range tests {
		got := extractEnum(tt.desc)
		if tt.expected == nil {
			if got != nil {
				t.Errorf("extractEnum(%q) = %v, want nil", tt.desc, got)
			}
			continue
		}
		if len(got) != len(tt.expected) {
			t.Errorf("extractEnum(%q) = %v (len %d), want %v (len %d)", tt.desc, got, len(got), tt.expected, len(tt.expected))
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("extractEnum(%q)[%d] = %q, want %q", tt.desc, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestParseEndpointPage_EnumExtraction(t *testing.T) {
	markdown := `
# Create Work Item

POST /api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/

## Request Body

| Name | Type | Required | Description |
|------|------|----------|-------------|
| name | string | Yes | The work item name |
| priority | string | No | Priority: urgent, high, medium, low, none |
| state_id | uuid | No | State ID |

Response (201)
`

	entry := Entry{Title: "Create Work Item", URL: "https://developers.plane.so/api-reference/issue/add-issue"}
	spec := ParseEndpointPage(markdown, "issue", entry)

	for _, p := range spec.Params {
		if p.Name == "priority" {
			if len(p.Enum) != 5 {
				t.Errorf("expected 5 enum values for priority, got %d: %v", len(p.Enum), p.Enum)
			}
			expected := []string{"urgent", "high", "medium", "low", "none"}
			for i, v := range expected {
				if i >= len(p.Enum) || p.Enum[i] != v {
					t.Errorf("priority enum[%d] = %q, want %q", i, p.Enum[i], v)
				}
			}
			return
		}
	}
	t.Error("missing 'priority' param")
}

func TestParseEndpointPage_InlineEnumExtraction(t *testing.T) {
	markdown := "# Create a work item\n" +
		"POST /api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/\n" +
		"### Body Parameters\n\n" +
		"`name`:requiredstring\nName of the work item.\n\n" +
		"`priority`:optionalstring\nPriority: urgent, high, medium, low, none\n\n" +
		"### Scopes\n\n"

	entry := Entry{Title: "Create Work Item", URL: "https://example.com"}
	spec := ParseEndpointPage(markdown, "issue", entry)

	for _, p := range spec.Params {
		if p.Name == "priority" {
			if len(p.Enum) != 5 {
				t.Errorf("expected 5 enum values for priority, got %d: %v", len(p.Enum), p.Enum)
			}
			return
		}
	}
	t.Error("missing 'priority' param")
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

func TestEndpointSpec_JSONSchema(t *testing.T) {
	// Verify that EndpointSpec serializes to the expected JSON structure
	// that agents will consume via `plane docs <topic> <cmd> -o json`.
	spec := &EndpointSpec{
		TopicName:    "issue",
		EntryTitle:   "Create Work Item",
		SourceURL:    "https://developers.plane.so/api-reference/issue/add-issue",
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/",
		StatusCode:   201,
		Params: []ParamSpec{
			{Name: "name", Type: "string", Required: true, Location: ParamBody, Description: "The work item name"},
			{Name: "priority", Type: "string", Required: false, Location: ParamBody, Description: "Priority: urgent, high, medium, low, none", Enum: []string{"urgent", "high", "medium", "low", "none"}},
			{Name: "state_id", Type: "string", Required: false, Location: ParamBody, Description: "State ID"},
			{Name: "work_item_id", Type: "string", Required: true, Location: ParamPath},
		},
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal spec: %v", err)
	}

	// Unmarshal back and verify round-trip
	var decoded EndpointSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal spec: %v", err)
	}

	if decoded.Method != "POST" {
		t.Errorf("method = %q, want POST", decoded.Method)
	}
	if decoded.TopicName != "issue" {
		t.Errorf("topic_name = %q, want issue", decoded.TopicName)
	}
	if decoded.StatusCode != 201 {
		t.Errorf("status_code = %d, want 201", decoded.StatusCode)
	}
	if len(decoded.Params) != 4 {
		t.Fatalf("params count = %d, want 4", len(decoded.Params))
	}

	// Verify each param has the required fields
	for _, p := range decoded.Params {
		if p.Name == "" {
			t.Error("param name must not be empty")
		}
		if p.Type == "" {
			t.Error("param type must not be empty")
		}
		if p.Location == "" {
			t.Error("param location must not be empty")
		}
	}

	// Verify enum field round-trips correctly
	priorityParam := decoded.Params[1]
	if priorityParam.Name != "priority" {
		t.Fatalf("expected second param to be priority, got %s", priorityParam.Name)
	}
	if len(priorityParam.Enum) != 5 {
		t.Errorf("priority enum count = %d, want 5", len(priorityParam.Enum))
	}

	// Verify enum is omitted when empty
	stateParam := decoded.Params[2]
	if stateParam.Name != "state_id" {
		t.Fatalf("expected third param to be state_id, got %s", stateParam.Name)
	}
	if stateParam.Enum != nil {
		t.Errorf("state_id should have nil enum, got %v", stateParam.Enum)
	}

	// Verify omitempty on enum: check raw JSON doesn't contain "enum" for state_id
	_ = string(data)
	// The state_id param section should not contain "enum"
	// Check by re-marshaling just the state_id param
	stateData, _ := json.Marshal(spec.Params[2])
	stateJSON := string(stateData)
	if contains(stateJSON, `"enum"`) {
		t.Error("state_id JSON should not contain enum field (omitempty)")
	}

	// But priority should contain enum
	priorityData, _ := json.Marshal(spec.Params[1])
	priorityJSON := string(priorityData)
	if !contains(priorityJSON, `"enum"`) {
		t.Error("priority JSON should contain enum field")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
