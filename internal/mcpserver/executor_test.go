package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mggarofalo/plane-cli/internal/api"
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

// autoPaginate runs the MCP executeAutoPageinate against a stub server
// returning the given body.
func autoPaginate(t *testing.T, body string) map[string]any {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL, "test-token", "test-ws", false, nil)

	got, err := executeAutoPageinate(context.Background(), client, srv.URL+"/relations/", map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(got, &result); err != nil {
		t.Fatalf("failed to parse output: %v\nOutput: %s", err, got)
	}
	return result
}

func TestAutoPaginate_NonEnvelopeResponsePassesThrough(t *testing.T) {
	// relation_list returns an object keyed by relation type, and every
	// single-resource GET tool returns a plain object. Neither is a pagination
	// envelope, so wrapping them would discard the payload entirely.
	result := autoPaginate(t, `{"blocking":[],"blocked_by":[{"project_id":"p1","issue_id":"i1"}]}`)

	if _, wrapped := result["results"]; wrapped {
		t.Fatalf("non-envelope response was wrapped in a pagination envelope: %#v", result)
	}
	if blockedBy, ok := result["blocked_by"].([]any); !ok || len(blockedBy) != 1 {
		t.Fatalf("blocked_by relation was dropped: %#v", result)
	}
}

func TestAutoPaginate_SingleResourceGetPassesThrough(t *testing.T) {
	result := autoPaginate(t, `{"id":"550e8400-e29b-41d4-a716-446655440000","name":"Fix login"}`)

	if _, wrapped := result["results"]; wrapped {
		t.Fatalf("single-resource GET was wrapped in a pagination envelope: %#v", result)
	}
	if result["name"] != "Fix login" {
		t.Errorf("name = %v, want %q", result["name"], "Fix login")
	}
}

func TestAutoPaginate_EmptyEnvelopeStillWrapped(t *testing.T) {
	result := autoPaginate(t, `{"results":[],"total_count":0,"next_page_results":false}`)

	results, ok := result["results"].([]any)
	if !ok {
		t.Fatalf("expected a results array, got: %#v", result)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
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
