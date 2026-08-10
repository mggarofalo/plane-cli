package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestExtractID_SingleObject(t *testing.T) {
	data := []byte(`{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "Test Issue"}`)

	var buf bytes.Buffer
	err := ExtractID(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "550e8400-e29b-41d4-a716-446655440000"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractID_NoTrailingNewline(t *testing.T) {
	data := []byte(`{"id": "abc-123", "name": "Test"}`)

	var buf bytes.Buffer
	err := ExtractID(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if got != "abc-123" {
		t.Errorf("expected %q, got %q", "abc-123", got)
	}
	if got[len(got)-1] == '\n' {
		t.Error("output should not end with a newline")
	}
}

func TestExtractID_Array(t *testing.T) {
	data := []byte(`[{"id": "a1", "name": "First"}, {"id": "a2", "name": "Second"}]`)

	var buf bytes.Buffer
	err := ExtractID(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "a1\na2"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractID_PaginatedEnvelope(t *testing.T) {
	data := []byte(`{"results": [{"id": "p1"}, {"id": "p2"}], "total_count": 2}`)

	var buf bytes.Buffer
	err := ExtractID(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "p1\np2"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractID_MissingID(t *testing.T) {
	data := []byte(`{"name": "No ID here"}`)

	var buf bytes.Buffer
	err := ExtractID(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Missing id should produce empty string (no newline)
	if buf.String() != "" {
		t.Errorf("expected empty string, got %q", buf.String())
	}
}

func TestExtractID_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)

	var buf bytes.Buffer
	err := ExtractID(&buf, data)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestExtractField_SingleObject(t *testing.T) {
	data := []byte(`{"id": "abc-123", "name": "Test Issue", "priority": 2}`)

	tests := []struct {
		name     string
		field    string
		expected string
		wantErr  bool
	}{
		{"string field", "id", "abc-123\n", false},
		{"string field name", "name", "Test Issue\n", false},
		{"number field", "priority", "2\n", false},
		// A missing field used to print a blank line and exit 0, which reads
		// as "the field is empty" rather than "there is no such field".
		{"missing field", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExtractField(&buf, data, tt.field)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got output %q", buf.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestExtractField_DottedPath(t *testing.T) {
	data := []byte(`{"id": "abc-123", "state_detail": {"id": "state-1", "name": "In Progress"}}`)

	tests := []struct {
		name     string
		field    string
		expected string
		wantErr  bool
	}{
		{"nested field", "state_detail.name", "In Progress\n", false},
		{"nested id", "state_detail.id", "state-1\n", false},
		{"missing nested", "state_detail.color", "", true},
		{"missing parent", "nonexistent.name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExtractField(&buf, data, tt.field)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got output %q", buf.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestExtractField_Array(t *testing.T) {
	data := []byte(`[{"id": "a1", "name": "First"}, {"id": "a2", "name": "Second"}]`)

	var buf bytes.Buffer
	err := ExtractField(&buf, data, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "a1\na2\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractField_PaginatedEnvelope(t *testing.T) {
	data := []byte(`{"results": [{"id": "p1"}, {"id": "p2"}], "total_count": 2}`)

	var buf bytes.Buffer
	err := ExtractField(&buf, data, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "p1\np2\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractField_BooleanAndNull(t *testing.T) {
	data := []byte(`{"active": true, "archived": false, "deleted": null}`)

	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{"true bool", "active", "true\n"},
		{"false bool", "archived", "false\n"},
		{"null field", "deleted", "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExtractField(&buf, data, tt.field)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestExtractField_FloatNumber(t *testing.T) {
	data := []byte(`{"sort_order": 95535.5, "count": 42}`)

	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{"float number", "sort_order", "95535.5\n"},
		{"integer number", "count", "42\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExtractField(&buf, data, tt.field)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, buf.String())
			}
		})
	}
}

func TestExtractField_NestedObject(t *testing.T) {
	// When the field resolves to a nested object (not fully traversed), output compact JSON
	data := []byte(`{"id": "abc", "state_detail": {"id": "s1", "name": "Done"}}`)

	var buf bytes.Buffer
	err := ExtractField(&buf, data, "state_detail")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should output compact JSON for the nested object
	got := buf.String()
	if got != `{"id":"s1","name":"Done"}`+"\n" {
		t.Errorf("expected compact JSON, got %q", got)
	}
}

func TestExtractField_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)

	var buf bytes.Buffer
	err := ExtractField(&buf, data, "id")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestExtractFields_SingleObject(t *testing.T) {
	data := []byte(`{"id": "abc-123", "name": "Test Issue", "priority": 2}`)

	var buf bytes.Buffer
	err := ExtractFields(&buf, data, []string{"id", "name", "priority"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "id\tname\tpriority\nabc-123\tTest Issue\t2\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractFields_Array(t *testing.T) {
	data := []byte(`[{"id": "a1", "name": "First"}, {"id": "a2", "name": "Second"}]`)

	var buf bytes.Buffer
	err := ExtractFields(&buf, data, []string{"id", "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "id\tname\na1\tFirst\na2\tSecond\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractFields_DottedPath(t *testing.T) {
	data := []byte(`{"id": "abc", "state_detail": {"name": "Done"}}`)

	var buf bytes.Buffer
	err := ExtractFields(&buf, data, []string{"id", "state_detail.name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "id\tstate_detail.name\nabc\tDone\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractFields_MissingFields(t *testing.T) {
	data := []byte(`{"id": "abc"}`)

	var buf bytes.Buffer
	err := ExtractFields(&buf, data, []string{"id", "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "id\tnonexistent\nabc\t\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestExtractFields_PaginatedEnvelope(t *testing.T) {
	data := []byte(`{"results": [{"id": "p1", "name": "A"}, {"id": "p2", "name": "B"}]}`)

	var buf bytes.Buffer
	err := ExtractFields(&buf, data, []string{"id", "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "id\tname\np1\tA\np2\tB\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestFormatRawValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"integer float", float64(42), "42"},
		{"decimal float", float64(3.14), "3.14"},
		{"true bool", true, "true"},
		{"false bool", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRawValue(tt.input)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestTraversePath(t *testing.T) {
	item := map[string]any{
		"id":   "abc",
		"name": "Test",
		"state_detail": map[string]any{
			"id":   "s1",
			"name": "Done",
			"group_detail": map[string]any{
				"name": "completed",
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		expected any
		found    bool
	}{
		{"top-level", "id", "abc", true},
		{"nested one level", "state_detail.name", "Done", true},
		{"nested two levels", "state_detail.group_detail.name", "completed", true},
		{"missing top-level", "nonexistent", nil, false},
		{"missing nested", "state_detail.nonexistent", nil, false},
		{"missing deep", "state_detail.group_detail.nonexistent", nil, false},
		{"through a scalar", "id.name", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := traversePath(item, tt.path)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
			if found != tt.found {
				t.Errorf("expected found=%v, got %v", tt.found, found)
			}
		})
	}
}

func TestTraverseValue_ArrayIndices(t *testing.T) {
	doc := map[string]any{
		"results": []any{
			map[string]any{"name": "first", "tags": []any{"a", "b"}},
			map[string]any{"name": "second"},
		},
		"blocked_by": []any{
			map[string]any{"issue_id": "i1", "project_id": "p1"},
		},
		"empty":  []any{},
		"scalar": "plain",
	}

	tests := []struct {
		name     string
		path     string
		expected any
		found    bool
	}{
		{"index into envelope rows", "results.0.name", "first", true},
		{"second row", "results.1.name", "second", true},
		{"nested array of scalars", "results.0.tags.1", "b", true},
		{"relation shape", "blocked_by.0.issue_id", "i1", true},
		{"index out of range", "results.5.name", nil, false},
		{"negative index", "results.-1.name", nil, false},
		{"non-numeric segment on an array", "results.name", nil, false},
		{"index into an empty array", "empty.0", nil, false},
		{"index into a scalar", "scalar.0", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := traverseValue(doc, tt.path)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
			if found != tt.found {
				t.Errorf("expected found=%v, got %v", tt.found, found)
			}
		})
	}
}

func TestTraverseValue_PresentButNull(t *testing.T) {
	doc := map[string]any{"parent": nil}

	got, found := traverseValue(doc, "parent")
	if got != nil {
		t.Errorf("expected nil value, got %v", got)
	}
	if !found {
		t.Error("a present-but-null field must count as found; otherwise it is indistinguishable from a typo")
	}
}

func TestExtractField_EnvelopeTopLevelKey(t *testing.T) {
	// Used to look total_count up on each row, miss, and print a blank line
	// per row.
	data := []byte(`{"results":[{"name":"a"},{"name":"b"}],"total_count":2}`)

	var buf bytes.Buffer
	if err := ExtractField(&buf, data, "total_count"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := buf.String(); got != "2\n" {
		t.Errorf("got %q, want %q", got, "2\n")
	}
}

func TestExtractField_PerRowStillWorks(t *testing.T) {
	// A key that exists on rows but not on the envelope must still produce one
	// line per row.
	data := []byte(`{"results":[{"name":"a"},{"name":"b"}],"total_count":2}`)

	var buf bytes.Buffer
	if err := ExtractField(&buf, data, "name"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := buf.String(); got != "a\nb\n" {
		t.Errorf("got %q, want %q", got, "a\nb\n")
	}
}

func TestExtractField_IndexedPath(t *testing.T) {
	data := []byte(`{"results":[{"name":"a"},{"name":"b"}],"total_count":2}`)

	var buf bytes.Buffer
	if err := ExtractField(&buf, data, "results.1.name"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := buf.String(); got != "b\n" {
		t.Errorf("got %q, want %q", got, "b\n")
	}
}

func TestExtractField_GroupedObject(t *testing.T) {
	// Work-item relations: an object keyed by relation type, not an envelope.
	data := []byte(`{"blocking":[],"blocked_by":[{"issue_id":"i1","project_id":"p1"}]}`)

	var buf bytes.Buffer
	if err := ExtractField(&buf, data, "blocked_by.0.issue_id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := buf.String(); got != "i1\n" {
		t.Errorf("got %q, want %q", got, "i1\n")
	}
}

func TestExtractField_UnresolvablePathErrors(t *testing.T) {
	cases := map[string][]byte{
		"envelope":      []byte(`{"results":[{"name":"a"}],"total_count":1}`),
		"single object": []byte(`{"id":"abc","name":"Test"}`),
		"plain array":   []byte(`[{"name":"a"},{"name":"b"}]`),
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExtractField(&buf, data, "no_such_field")
			if err == nil {
				t.Fatalf("expected an error, got output %q", buf.String())
			}
			if !strings.Contains(err.Error(), "no_such_field") {
				t.Errorf("error should name the path, got: %v", err)
			}
			if buf.Len() != 0 {
				t.Errorf("expected no output on error, got %q", buf.String())
			}
		})
	}
}

func TestExtractField_EmptyResultsIsNotAMissingField(t *testing.T) {
	// An empty project or a filter that matched nothing must not fail a
	// script. Zero rows is zero lines, not "no such field".
	cases := map[string][]byte{
		"empty envelope": []byte(`{"results":[],"total_count":0}`),
		"empty array":    []byte(`[]`),
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := ExtractField(&buf, data, "name"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if buf.Len() != 0 {
				t.Errorf("expected no output, got %q", buf.String())
			}
		})
	}
}

func TestExtractField_PresentButNullPrintsEmpty(t *testing.T) {
	data := []byte(`{"id":"abc","parent":null}`)

	var buf bytes.Buffer
	if err := ExtractField(&buf, data, "parent"); err != nil {
		t.Fatalf("a present-but-null field must not error: %v", err)
	}
	if got := buf.String(); got != "\n" {
		t.Errorf("got %q, want %q", got, "\n")
	}
}

func TestExtractFields_MissingColumnStaysBlank(t *testing.T) {
	// The TSV form is addressed by column position, so an absent column has to
	// keep its slot rather than error.
	data := []byte(`{"results":[{"name":"a"},{"name":"b"}],"total_count":2}`)

	var buf bytes.Buffer
	if err := ExtractFields(&buf, data, []string{"name", "no_such_field"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "name\tno_such_field\na\t\nb\t\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
