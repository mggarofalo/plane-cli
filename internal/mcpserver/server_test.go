package mcpserver

import (
	"strings"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

func TestBuildTool_BasicEndpoint(t *testing.T) {
	spec := &docs.EndpointSpec{
		TopicName:    "issue",
		EntryTitle:   "Create Work Item",
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/",
		Params: []docs.ParamSpec{
			{Name: "workspace_slug", Type: "string", Location: docs.ParamPath},
			{Name: "project_id", Type: "string", Location: docs.ParamPath},
			{Name: "name", Type: "string", Location: docs.ParamBody, Required: true},
		},
	}

	cfg := &Config{Workspace: "test-ws", Project: "proj-uuid"}
	entry := BuildTool("issue", spec, cfg)

	if entry == nil {
		t.Fatal("expected non-nil ToolEntry")
	}

	if entry.Tool.Name != "issue_create" {
		t.Errorf("expected tool name 'issue_create', got %q", entry.Tool.Name)
	}

	if entry.Tool.Description == "" {
		t.Error("expected non-empty description")
	}

	if entry.Tool.Annotations == nil {
		t.Fatal("expected non-nil annotations")
	}

	// POST should not be read-only
	if entry.Tool.Annotations.ReadOnlyHint {
		t.Error("POST tool should not be read-only")
	}

	if entry.Handler == nil {
		t.Error("expected non-nil handler")
	}

	// issue_create must remind agents that modules are attached separately
	if !strings.Contains(entry.Tool.Description, "module_add_work_items") {
		t.Errorf("issue_create description should mention module_add_work_items follow-up, got %q", entry.Tool.Description)
	}
}

func TestBuildTool_ModuleHintOnlyForIssueCreate(t *testing.T) {
	// A non-issue_create POST should NOT get the module hint.
	spec := &docs.EndpointSpec{
		TopicName:    "label",
		EntryTitle:   "Create Label",
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/labels/",
		Params: []docs.ParamSpec{
			{Name: "name", Type: "string", Location: docs.ParamBody, Required: true},
		},
	}

	cfg := &Config{Workspace: "test-ws", Project: "proj-uuid"}
	entry := BuildTool("label", spec, cfg)

	if entry == nil {
		t.Fatal("expected non-nil ToolEntry")
	}
	if strings.Contains(entry.Tool.Description, "module_add_work_items") {
		t.Errorf("only issue_create should mention module_add_work_items, got %q", entry.Tool.Description)
	}
}

func TestBuildTool_GETEndpoint(t *testing.T) {
	spec := &docs.EndpointSpec{
		TopicName:    "issue",
		EntryTitle:   "List Work Items",
		Method:       "GET",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/",
	}

	cfg := &Config{Workspace: "test-ws"}
	entry := BuildTool("issue", spec, cfg)

	if entry == nil {
		t.Fatal("expected non-nil ToolEntry")
	}

	if entry.Tool.Name != "issue_list" {
		t.Errorf("expected tool name 'issue_list', got %q", entry.Tool.Name)
	}

	if !entry.Tool.Annotations.ReadOnlyHint {
		t.Error("GET tool should be read-only")
	}
}

func TestBuildTool_DELETEEndpoint(t *testing.T) {
	spec := &docs.EndpointSpec{
		TopicName:    "issue",
		EntryTitle:   "Delete Work Item",
		Method:       "DELETE",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/issues/{work_item_id}/",
	}

	cfg := &Config{Workspace: "test-ws"}
	entry := BuildTool("issue", spec, cfg)

	if entry == nil {
		t.Fatal("expected non-nil ToolEntry")
	}

	if entry.Tool.Name != "issue_delete" {
		t.Errorf("expected tool name 'issue_delete', got %q", entry.Tool.Name)
	}

	if entry.Tool.Annotations.DestructiveHint == nil || !*entry.Tool.Annotations.DestructiveHint {
		t.Error("DELETE tool should be destructive")
	}
}

func TestBuildTool_SkipsOverview(t *testing.T) {
	spec := &docs.EndpointSpec{
		TopicName:  "issue",
		EntryTitle: "Overview",
		Method:     "GET",
	}

	cfg := &Config{Workspace: "test-ws"}
	entry := BuildTool("issue", spec, cfg)

	if entry != nil {
		t.Error("expected nil for Overview entry")
	}
}

func TestDeriveActionName(t *testing.T) {
	tests := []struct {
		title     string
		topic     string
		expected  string
	}{
		{"Create Work Item", "issue", "create"},
		{"List Work Items", "issue", "list"},
		{"Get Work Item Detail", "issue", "get"},
		{"Delete Work Item", "issue", "delete"},
		{"Add Cycle Work Items", "cycle", "add_work_items"},
		{"List Archived Cycles", "cycle", "list_archived"},
		{"Get by Sequence ID", "issue", "get_by_sequence_id"},
		{"Transfer Work Items", "cycle", "transfer_work_items"},
		{"Overview", "issue", ""},
		{"API Introduction", "introduction", ""},
		{"Create State", "state", "create"},
		{"List States", "state", "list"},
		{"Get State Detail", "state", "get"},
		{"Add Link", "link", "add"},
		{"List Links", "link", "list"},
		{"Get Current User", "user", "get"},
		{"Add Intake Issue", "intake", "add"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := deriveActionName(tt.title, tt.topic)
			if result != tt.expected {
				t.Errorf("deriveActionName(%q, %q) = %q, want %q", tt.title, tt.topic, result, tt.expected)
			}
		})
	}
}

func TestAnnotationsFromMethod(t *testing.T) {
	tests := []struct {
		method      string
		readOnly    bool
		destructive bool
		idempotent  bool
	}{
		{"GET", true, false, true},
		{"POST", false, false, false},
		{"PUT", false, false, true},
		{"PATCH", false, false, false},
		{"DELETE", false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			ann := AnnotationsFromMethod(tt.method)
			if ann.ReadOnlyHint != tt.readOnly {
				t.Errorf("ReadOnlyHint: got %v, want %v", ann.ReadOnlyHint, tt.readOnly)
			}
			if ann.DestructiveHint == nil {
				t.Fatal("DestructiveHint should not be nil")
			}
			if *ann.DestructiveHint != tt.destructive {
				t.Errorf("DestructiveHint: got %v, want %v", *ann.DestructiveHint, tt.destructive)
			}
			if ann.IdempotentHint != tt.idempotent {
				t.Errorf("IdempotentHint: got %v, want %v", ann.IdempotentHint, tt.idempotent)
			}
		})
	}
}

func TestErrorResult(t *testing.T) {
	result := errorResult("something went wrong")
	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"8b1ccfc2-4b91-484f-b7c5-a9e9304ac13b", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"ABCDEF01-2345-6789-abcd-ef0123456789", true},
		// Too short
		{"550e8400-e29b-41d4-a716-44665544000", false},
		// Too long
		{"550e8400-e29b-41d4-a716-4466554400001", false},
		// Missing hyphens
		{"550e8400xe29b-41d4-a716-446655440000", false},
		// Non-hex characters (BUG-004 regression)
		{"550e8400-e29b-41d4-a716-44665544000g", false},
		{"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz", false},
		{"hello-wo-rld!-this-is-n-ot-a-valid-uu", false},
		// Empty
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isUUID(tt.input)
			if result != tt.expected {
				t.Errorf("isUUID(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
