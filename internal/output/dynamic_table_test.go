package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"id", "Id"},
		{"name", "Name"},
		{"created_at", "Created At"},
		{"sequence_id", "Sequence Id"},
		{"is_archived", "Is Archived"},
		{"", ""},
	}
	for _, tt := range tests {
		got := toTitleCase(tt.in)
		if got != tt.want {
			t.Errorf("toTitleCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"eaf07920-649f-412f-adc7-b5490e5580a8", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"not-a-uuid", false},
		{"eaf07920649f412fadc7b5490e5580a8", false}, // no dashes
		{"", false},
	}
	for _, tt := range tests {
		got := looksLikeUUID(tt.in)
		if got != tt.want {
			t.Errorf("looksLikeUUID(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestFormatCell(t *testing.T) {
	tests := []struct {
		val, col, want string
	}{
		// UUID shortening
		{"eaf07920-649f-412f-adc7-b5490e5580a8", "id", "eaf07920"},
		// Non-UUID id stays as is
		{"short", "id", "short"},
		// Date column strips time
		{"2026-03-11T16:54:16.514442-04:00", "created_at", "2026-03-11"},
		{"2026-03-11T16:54:16.514442-04:00", "start_date", "2026-03-11"},
		// No T separator — keep as is
		{"2026-03-11", "created_at", "2026-03-11"},
		// Empty stays empty
		{"", "name", ""},
		// Regular value
		{"some value", "name", "some value"},
	}
	for _, tt := range tests {
		got := formatCell(tt.val, tt.col)
		if got != tt.want {
			t.Errorf("formatCell(%q, %q) = %q, want %q", tt.val, tt.col, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in     string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world this is long", 10, "hello w..."},
		{"short", 0, "short"}, // 0 means no limit
	}
	for _, tt := range tests {
		got := truncate(tt.in, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
		}
	}
}

func TestExtractItems(t *testing.T) {
	t.Run("paginated envelope", func(t *testing.T) {
		data := `{"results": [{"id": "1", "name": "foo"}], "total_count": 1}`
		items, err := extractItems([]byte(data))
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		if items[0]["name"] != "foo" {
			t.Errorf("got name %v, want foo", items[0]["name"])
		}
	})

	t.Run("plain array", func(t *testing.T) {
		data := `[{"id": "1"}, {"id": "2"}]`
		items, err := extractItems([]byte(data))
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 2 {
			t.Fatalf("got %d items, want 2", len(items))
		}
	})

	t.Run("single object", func(t *testing.T) {
		data := `{"id": "1", "name": "solo"}`
		items, err := extractItems([]byte(data))
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := extractItems([]byte("not json"))
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestSelectColumns(t *testing.T) {
	item := map[string]any{
		"id":         "abc",
		"name":       "test",
		"priority":   "high",
		"created_at": "2026-01-01",
		"labels":     []any{}, // non-simple, should be excluded
	}
	cols := selectColumns(item)
	if len(cols) == 0 {
		t.Fatal("expected at least one column")
	}
	// id, name, priority, created_at should all be present
	colSet := map[string]bool{}
	for _, c := range cols {
		colSet[c] = true
	}
	for _, expected := range []string{"id", "name", "priority", "created_at"} {
		if !colSet[expected] {
			t.Errorf("expected column %q not found in %v", expected, cols)
		}
	}
}

func TestFormatDynamicTable(t *testing.T) {
	data := map[string]any{
		"results": []map[string]any{
			{
				"id":         "eaf07920-649f-412f-adc7-b5490e5580a8",
				"name":       "Test Issue",
				"priority":   "high",
				"created_at": "2026-03-11T16:54:16.514442-04:00",
			},
		},
		"total_count": 1,
	}
	raw, _ := json.Marshal(data)

	var buf bytes.Buffer
	err := FormatDynamicTable(&buf, raw)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	// Check title case headers
	if !strings.Contains(out, "Name") {
		t.Error("expected title case header 'Name'")
	}
	// Check UUID is shortened
	if !strings.Contains(out, "eaf07920") {
		t.Error("expected shortened UUID")
	}
	if strings.Contains(out, "eaf07920-649f") {
		t.Error("UUID should be truncated, not full")
	}
	// Check date is stripped
	if !strings.Contains(out, "2026-03-11") {
		t.Error("expected date in output")
	}
	if strings.Contains(out, "T16:54") {
		t.Error("timestamp should be stripped to date only")
	}
}

func TestFormatDynamicTable_EmptyResults(t *testing.T) {
	data := `{"results": []}`
	var buf bytes.Buffer
	err := FormatDynamicTable(&buf, []byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Table format not available") {
		t.Error("expected fallback message for empty results")
	}
}
