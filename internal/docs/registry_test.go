package docs

import (
	"context"
	"strings"
	"testing"
)

func TestLoad_FallbackUsesCustomBaseURL(t *testing.T) {
	// Use a non-existent profile so cache miss is guaranteed,
	// and a base URL that will fail the remote fetch, forcing
	// fallback to RebaseTopics.
	custom := "https://custom.example.com"
	r := NewRegistry("__test_no_cache__", custom)
	r.Quiet = true

	if err := r.Load(context.Background()); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	topics := r.Topics()
	if len(topics) == 0 {
		t.Fatal("Load() returned no topics")
	}

	for _, topic := range topics {
		for _, entry := range topic.Entries {
			if strings.Contains(entry.URL, DefaultBaseURL) {
				t.Errorf("entry URL still points to DefaultBaseURL: %s", entry.URL)
			}
			if !strings.HasPrefix(entry.URL, custom) {
				t.Errorf("entry URL doesn't start with custom base %q: %s", custom, entry.URL)
			}
		}
	}
}

func TestLoad_FallbackDefaultBaseURL(t *testing.T) {
	// When using DefaultBaseURL, fallback should return topics with
	// DefaultBaseURL URLs (i.e., unmodified DefaultTopics).
	r := NewRegistry("__test_no_cache__", DefaultBaseURL)
	r.Quiet = true

	if err := r.Load(context.Background()); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	topics := r.Topics()
	if len(topics) == 0 {
		t.Fatal("Load() returned no topics")
	}

	for _, topic := range topics {
		for _, entry := range topic.Entries {
			if !strings.HasPrefix(entry.URL, DefaultBaseURL) {
				t.Errorf("entry URL doesn't start with DefaultBaseURL: %s", entry.URL)
			}
		}
	}
}
