package cmd

import (
	"encoding/json"
	"testing"
)

func TestExtractNames_PaginatedResponse(t *testing.T) {
	response := `{
		"results": [
			{"id": "aaa", "name": "Backlog"},
			{"id": "bbb", "name": "In Progress"},
			{"id": "ccc", "name": "Done"}
		]
	}`

	names, err := extractNames([]byte(response))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"Backlog", "In Progress", "Done"}
	if len(names) != len(want) {
		t.Fatalf("got %d names, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestExtractNames_PlainArray(t *testing.T) {
	response := `[
		{"id": "1", "name": "Bug"},
		{"id": "2", "name": "Feature"}
	]`

	names, err := extractNames([]byte(response))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"Bug", "Feature"}
	if len(names) != len(want) {
		t.Fatalf("got %d names, want %d", len(names), len(want))
	}
	for i, n := range names {
		if n != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, n, want[i])
		}
	}
}

func TestExtractNames_EmptyResults(t *testing.T) {
	response := `{"results": []}`

	names, err := extractNames([]byte(response))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty names, got %v", names)
	}

	// Verify JSON serialization produces [] not null (BUG-001 fix)
	data, err := json.Marshal(names)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("expected JSON [], got %s", string(data))
	}
}

func TestExtractNames_SkipsMissingName(t *testing.T) {
	response := `[
		{"id": "1", "name": "Valid"},
		{"id": "2"},
		{"id": "3", "name": ""}
	]`

	names, err := extractNames([]byte(response))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(names) != 1 || names[0] != "Valid" {
		t.Errorf("expected [Valid], got %v", names)
	}
}

func TestOutputJSON(t *testing.T) {
	// Just verify it produces valid JSON
	values := []string{"urgent", "high", "medium", "low", "none"}
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var roundtrip []string
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(roundtrip) != len(values) {
		t.Fatalf("got %d values, want %d", len(roundtrip), len(values))
	}
	for i, v := range roundtrip {
		if v != values[i] {
			t.Errorf("values[%d] = %q, want %q", i, v, values[i])
		}
	}
}

func TestRunComplete_Priority(t *testing.T) {
	// Priority is a static list — does not require API calls.
	// We just test that runComplete doesn't error with a nil context for priority.
	err := runComplete(nil, "priority")
	if err != nil {
		t.Fatalf("unexpected error for priority: %v", err)
	}
}

func TestRunComplete_UnsupportedParam(t *testing.T) {
	err := runComplete(nil, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unsupported param")
	}
	expected := `unsupported parameter "nonexistent"`
	if got := err.Error(); got[:len(expected)] != expected {
		t.Errorf("error = %q, want prefix %q", got, expected)
	}
}

func TestRunComplete_CaseInsensitive(t *testing.T) {
	// Priority should work regardless of case
	for _, param := range []string{"PRIORITY", "Priority", "priority"} {
		err := runComplete(nil, param)
		if err != nil {
			t.Errorf("unexpected error for param %q: %v", param, err)
		}
	}
}

func TestRunComplete_LabelAliases(t *testing.T) {
	// "label" and "labels" should both be recognized (they'll fail on API
	// call since we have no client, but the param dispatch should succeed).
	for _, param := range []string{"label", "labels"} {
		err := runComplete(nil, param)
		if err == nil {
			// Without auth config, we expect an error from NewClient, not from
			// "unsupported parameter".
			t.Logf("expected API/auth error for %q (no client configured)", param)
			continue
		}
		if err.Error() == `unsupported parameter "`+param+`"; supported: state, priority, label(s)` {
			t.Errorf("param %q was treated as unsupported", param)
		}
	}
}
