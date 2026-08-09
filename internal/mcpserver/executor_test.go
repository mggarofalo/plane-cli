package mcpserver

import (
	"context"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

// specWithArrayParam builds a minimal POST spec carrying a single string[]
// body param, which is the shape shared by relation create, module
// add-work-items and cycle add-work-items.
func specWithArrayParam(paramName string) *docs.EndpointSpec {
	return &docs.EndpointSpec{
		Method:       "POST",
		PathTemplate: "/api/v1/workspaces/{workspace_slug}/projects/{project_id}/work-items/{work_item_id}/relations/",
		Params: []docs.ParamSpec{
			{Name: "work_item_id", Type: "string", Required: true, Location: docs.ParamPath},
			{Name: paramName, Type: "string[]", Required: true, Location: docs.ParamBody},
		},
	}
}

func bodyStrings(t *testing.T, body map[string]any, key string) []string {
	t.Helper()
	raw, ok := body[key]
	if !ok {
		t.Fatalf("body has no %q key: %#v", key, body)
	}
	slice, ok := raw.([]string)
	if !ok {
		t.Fatalf("body[%q] is %T, want []string", key, raw)
	}
	return slice
}

func TestCollectBodyFromMap_IssueRefArrayPassesUUIDsThrough(t *testing.T) {
	// UUIDs short-circuit inside resolveValue before any client is built, so
	// this stays hermetic while still exercising the issue-ref array branch.
	uuids := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}
	args := map[string]any{"issues": []any{uuids[0], uuids[1]}}

	body, err := collectBodyFromMap(context.Background(), specWithArrayParam("issues"), args, "test-ws", "proj-uuid", &Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := bodyStrings(t, body, "issues")
	if len(got) != len(uuids) {
		t.Fatalf("issues length = %d, want %d", len(got), len(uuids))
	}
	for i, want := range uuids {
		if got[i] != want {
			t.Errorf("issues[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestCollectBodyFromMap_NonIssueRefArrayIsNotResolved(t *testing.T) {
	// Only issue-reference arrays get per-element resolution. Anything else —
	// assignees, labels — must reach the body verbatim, even when a value
	// happens to look like a sequence ID.
	args := map[string]any{"assignees": []any{"PROJ-42", "not-a-uuid"}}

	body, err := collectBodyFromMap(context.Background(), specWithArrayParam("assignees"), args, "test-ws", "proj-uuid", &Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := bodyStrings(t, body, "assignees")
	want := []string{"PROJ-42", "not-a-uuid"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("assignees[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsIssueRefParam_DelegatesToCmdgen(t *testing.T) {
	// The MCP server must not keep its own copy of this set; related_issue was
	// added for relation remove and has to be visible here too.
	tests := []struct {
		param string
		want  bool
	}{
		{"work_item_id", true},
		{"parent", true},
		{"issues", true},
		{"related_issue", true},
		{"assignees", false},
		{"labels", false},
		{"name", false},
	}

	for _, tt := range tests {
		t.Run(tt.param, func(t *testing.T) {
			if got := isIssueRefParam(tt.param); got != tt.want {
				t.Errorf("isIssueRefParam(%q) = %v, want %v", tt.param, got, tt.want)
			}
		})
	}
}
