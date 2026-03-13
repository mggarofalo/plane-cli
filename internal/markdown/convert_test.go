package markdown

import (
	"strings"
	"testing"
)

func TestToHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"bold", "**bold**", "<strong>bold</strong>"},
		{"italic", "_italic_", "<em>italic</em>"},
		{"paragraph", "hello world", "<p>hello world</p>"},
		{"heading", "# Title", "<h1>Title</h1>"},
		{"link", "[text](https://example.com)", `<a href="https://example.com">text</a>`},
		{"list", "- item1\n- item2", "<li>item1</li>"},
		{"code", "`code`", "<code>code</code>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToHTML(tt.input)
			if err != nil {
				t.Fatalf("ToHTML(%q) error: %v", tt.input, err)
			}
			if !strings.Contains(got, tt.contains) {
				t.Errorf("ToHTML(%q) = %q, want substring %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"bold", "<p><strong>bold</strong></p>", "**bold**"},
		{"italic", "<p><em>italic</em></p>", "*italic*"},
		{"paragraph", "<p>hello world</p>", "hello world"},
		{"heading", "<h1>Title</h1>", "# Title"},
		{"link", `<a href="https://example.com">text</a>`, "[text](https://example.com)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromHTML(tt.input)
			if err != nil {
				t.Fatalf("FromHTML(%q) error: %v", tt.input, err)
			}
			if !strings.Contains(got, tt.contains) {
				t.Errorf("FromHTML(%q) = %q, want substring %q", tt.input, got, tt.contains)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	md := "**bold** and _italic_ text"
	html, err := ToHTML(md)
	if err != nil {
		t.Fatalf("ToHTML error: %v", err)
	}
	back, err := FromHTML(html)
	if err != nil {
		t.Fatalf("FromHTML error: %v", err)
	}
	if !strings.Contains(back, "**bold**") {
		t.Errorf("round-trip lost bold: got %q", back)
	}
	if !strings.Contains(back, "italic") {
		t.Errorf("round-trip lost italic: got %q", back)
	}
}

func TestToHTMLEmpty(t *testing.T) {
	got, err := ToHTML("")
	if err != nil {
		t.Fatalf("ToHTML empty error: %v", err)
	}
	if got != "" {
		t.Errorf("ToHTML empty = %q, want empty", got)
	}
}

func TestFromHTMLEmpty(t *testing.T) {
	got, err := FromHTML("")
	if err != nil {
		t.Fatalf("FromHTML empty error: %v", err)
	}
	if got != "" {
		t.Errorf("FromHTML empty = %q, want empty", got)
	}
}
