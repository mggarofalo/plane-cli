package output

import (
	"bytes"
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
	}{
		{"string field", "id", "abc-123\n"},
		{"string field name", "name", "Test Issue\n"},
		{"number field", "priority", "2\n"},
		{"missing field", "nonexistent", "\n"},
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

func TestExtractField_DottedPath(t *testing.T) {
	data := []byte(`{"id": "abc-123", "state_detail": {"id": "state-1", "name": "In Progress"}}`)

	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{"nested field", "state_detail.name", "In Progress\n"},
		{"nested id", "state_detail.id", "state-1\n"},
		{"missing nested", "state_detail.color", "\n"},
		{"missing parent", "nonexistent.name", "\n"},
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
	}{
		{"top-level", "id", "abc"},
		{"nested one level", "state_detail.name", "Done"},
		{"nested two levels", "state_detail.group_detail.name", "completed"},
		{"missing top-level", "nonexistent", nil},
		{"missing nested", "state_detail.nonexistent", nil},
		{"missing deep", "state_detail.group_detail.nonexistent", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := traversePath(item, tt.path)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
