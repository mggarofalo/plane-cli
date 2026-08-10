package docs

import (
	"strings"
	"testing"
)

func TestRebaseTopics_DefaultBaseURL(t *testing.T) {
	// When baseURL equals DefaultBaseURL, should return DefaultTopics directly.
	got := RebaseTopics(DefaultBaseURL)
	if &got[0] != &DefaultTopics[0] {
		t.Error("expected same slice when baseURL == DefaultBaseURL")
	}
}

func TestRebaseTopics_EmptyBaseURL(t *testing.T) {
	got := RebaseTopics("")
	if &got[0] != &DefaultTopics[0] {
		t.Error("expected same slice when baseURL is empty")
	}
}

func TestRebaseTopics_CustomBaseURL(t *testing.T) {
	custom := "https://docs.example.com"
	got := RebaseTopics(custom)

	// Must be a different slice (deep copy)
	if len(got) != len(DefaultTopics) {
		t.Fatalf("topic count mismatch: got %d, want %d", len(got), len(DefaultTopics))
	}

	for i, topic := range got {
		if topic.Name != DefaultTopics[i].Name {
			t.Errorf("topic[%d] name mismatch: got %q, want %q", i, topic.Name, DefaultTopics[i].Name)
		}
		if len(topic.Entries) != len(DefaultTopics[i].Entries) {
			t.Errorf("topic[%d] entry count mismatch: got %d, want %d", i, len(topic.Entries), len(DefaultTopics[i].Entries))
			continue
		}
		for j, entry := range topic.Entries {
			orig := DefaultTopics[i].Entries[j]
			if entry.Title != orig.Title {
				t.Errorf("topic[%d].entry[%d] title mismatch: got %q, want %q", i, j, entry.Title, orig.Title)
			}
			if strings.Contains(entry.URL, DefaultBaseURL) {
				t.Errorf("topic[%d].entry[%d] URL still contains DefaultBaseURL: %s", i, j, entry.URL)
			}
			if !strings.HasPrefix(entry.URL, custom) {
				t.Errorf("topic[%d].entry[%d] URL doesn't start with custom base: %s", i, j, entry.URL)
			}
			// Path should be preserved
			expectedPath := strings.TrimPrefix(orig.URL, DefaultBaseURL)
			if !strings.HasSuffix(entry.URL, expectedPath) {
				t.Errorf("topic[%d].entry[%d] URL path changed: got %s, want suffix %s", i, j, entry.URL, expectedPath)
			}
		}
	}
}

func TestRebaseTopics_TrailingSlash(t *testing.T) {
	// A trailing slash on the custom base URL must not produce double-slash URLs.
	custom := "https://docs.example.com/"
	got := RebaseTopics(custom)

	for i, topic := range got {
		for j, entry := range topic.Entries {
			if strings.Contains(entry.URL, "//api") {
				t.Errorf("topic[%d].entry[%d] has double-slash URL: %s", i, j, entry.URL)
			}
			if !strings.HasPrefix(entry.URL, "https://docs.example.com/") {
				t.Errorf("topic[%d].entry[%d] URL doesn't start with expected base: %s", i, j, entry.URL)
			}
		}
	}
}

func TestRebaseTopics_DefaultBaseURLWithTrailingSlash(t *testing.T) {
	// DefaultBaseURL with a trailing slash should still return DefaultTopics directly.
	got := RebaseTopics(DefaultBaseURL + "/")
	if &got[0] != &DefaultTopics[0] {
		t.Error("expected same slice when baseURL == DefaultBaseURL with trailing slash")
	}
}

func TestDefaultTopics_Relation(t *testing.T) {
	var relation *Topic
	for i := range DefaultTopics {
		if DefaultTopics[i].Name == "relation" {
			relation = &DefaultTopics[i]
			break
		}
	}
	if relation == nil {
		t.Fatal("DefaultTopics has no \"relation\" topic")
	}

	// Overview plus the three executable endpoints.
	want := map[string]string{
		"Overview":        "/api-reference/work-item-relations/overview",
		"Create Relation": "/api-reference/work-item-relations/create-work-item-relation",
		"List Relations":  "/api-reference/work-item-relations/list-work-item-relations",
		"Remove Relation": "/api-reference/work-item-relations/remove-work-item-relation",
	}
	if len(relation.Entries) != len(want) {
		t.Fatalf("relation entry count = %d, want %d", len(relation.Entries), len(want))
	}

	for _, entry := range relation.Entries {
		path, ok := want[entry.Title]
		if !ok {
			t.Errorf("unexpected relation entry title %q", entry.Title)
			continue
		}
		if entry.URL != DefaultBaseURL+path {
			t.Errorf("entry %q URL = %q, want %q", entry.Title, entry.URL, DefaultBaseURL+path)
		}
		delete(want, entry.Title)
	}
	for title := range want {
		t.Errorf("missing relation entry %q", title)
	}
}

func TestRebaseTopics_DoesNotMutateOriginal(t *testing.T) {
	// Save a sample URL before rebasing
	origURL := DefaultTopics[0].Entries[0].URL

	custom := "https://custom.example.com"
	_ = RebaseTopics(custom)

	// Original must be unchanged
	if DefaultTopics[0].Entries[0].URL != origURL {
		t.Errorf("DefaultTopics was mutated: got %q, want %q", DefaultTopics[0].Entries[0].URL, origURL)
	}
}
