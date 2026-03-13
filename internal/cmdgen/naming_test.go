package cmdgen

import (
	"testing"
)

func TestDeriveSubcommandName(t *testing.T) {
	tests := []struct {
		title     string
		topicName string
		expected  string
	}{
		{"Create Work Item", "issue", "create"},
		{"List Work Items", "issue", "list"},
		{"Get Work Item", "issue", "get"},
		{"Get Work Item Detail", "issue", "get"},
		{"Update Work Item Detail", "issue", "update"},
		{"Delete Work Item", "issue", "delete"},
		{"Search Work Items", "issue", "search"},
		{"Get by Sequence ID", "issue", "get-by-sequence-id"},
		{"Create Cycle", "cycle", "create"},
		{"List Cycles", "cycle", "list"},
		{"Add Work Items", "cycle", "add-work-items"},
		{"List Cycle Work Items", "cycle", "list-work-items"},
		{"List Archived Cycles", "cycle", "list-archived"},
		{"Transfer Work Items", "cycle", "transfer-work-items"},
		{"Archive Cycle", "cycle", "archive"},
		{"Unarchive Cycle", "cycle", "unarchive"},
		{"Remove Work Item", "cycle", "remove-work-item"},
		{"Delete Cycle", "cycle", "delete"},
		{"Overview", "issue", ""},
		{"API Introduction", "introduction", ""},
		{"Add Comment", "comment", "add"},
		{"List Comments", "comment", "list"},
		{"Add Intake Issue", "intake", "add"},
		{"List Intake Issues", "intake", "list"},
		{"Get Intake Issue", "intake", "get"},
		{"Get Workspace Members", "member", "get-workspace"},
		{"Get Project Members", "member", "get-project"},
		{"Add Workspace Page", "page", "add-workspace"},
		{"Get Workspace Page", "page", "get-workspace"},
		{"Add Project Page", "page", "add-project"},
		{"Get Project Page", "page", "get-project"},
		{"Add Sticky", "sticky", "add"},
		{"List Stickies", "sticky", "list"},
		{"Link Work Items", "customer", "link-work-items"},
		{"Unlink Work Item", "customer", "unlink-work-item"},
		{"List Customer Work Items", "customer", "list-work-items"},
		{"Get Worklogs for Issue", "worklog", "get-for-issue"},
		{"Get Total Time", "worklog", "get-total-time"},
	}

	for _, tt := range tests {
		got := DeriveSubcommandName(tt.title, tt.topicName)
		if got != tt.expected {
			t.Errorf("DeriveSubcommandName(%q, %q) = %q, want %q", tt.title, tt.topicName, got, tt.expected)
		}
	}
}

func TestParamToFlagName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"name", "name"},
		{"state_id", "state-id"},
		{"description_html", "description-html"},
		{"workspace_slug", "workspace-slug"},
	}

	for _, tt := range tests {
		got := ParamToFlagName(tt.input)
		if got != tt.expected {
			t.Errorf("ParamToFlagName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsHTMLParam(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"description_html", true},
		{"content_html", true},
		{"name", false},
		{"state_id", false},
		{"html", false},
	}

	for _, tt := range tests {
		got := IsHTMLParam(tt.input)
		if got != tt.expected {
			t.Errorf("IsHTMLParam(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestMarkdownFlagName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"description_html", "description"},
		{"content_html", "content"},
		{"long_description_html", "long-description"},
	}

	for _, tt := range tests {
		got := MarkdownFlagName(tt.input)
		if got != tt.expected {
			t.Errorf("MarkdownFlagName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestIsAPIReferenceURL(t *testing.T) {
	if !IsAPIReferenceURL("https://developers.plane.so/api-reference/issue/add-issue") {
		t.Error("expected true for API reference URL")
	}
	if IsAPIReferenceURL("https://developers.plane.so/self-hosting/overview") {
		t.Error("expected false for non-API reference URL")
	}
}
