package docs

import (
	"context"
	"fmt"
	"os"
)

// Entry represents a single documentation page.
type Entry struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Topic groups related doc entries under a heading.
type Topic struct {
	Name    string  `json:"name"`
	Entries []Entry `json:"entries"`
}

// DocsRegistry manages documentation topics, loading them from cache, remote, or defaults.
type DocsRegistry struct {
	Profile string
	BaseURL string
	Quiet   bool
	topics  []Topic
}

// NewRegistry creates a new DocsRegistry for the given profile and base URL.
func NewRegistry(profile, baseURL string) *DocsRegistry {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &DocsRegistry{
		Profile: profile,
		BaseURL: baseURL,
	}
}

// Load populates the registry following the resolution chain:
// 1. Try cache (if baseURL matches)
// 2. Try fetch {baseURL}/llms.txt, parse, write cache
// 3. Fall back to DefaultTopics with a stderr hint
func (r *DocsRegistry) Load(ctx context.Context) error {
	// 1. Try cache
	if topics, err := loadCache(r.Profile, r.BaseURL); err == nil && len(topics) > 0 {
		r.topics = topics
		return nil
	}

	// 2. Try remote fetch
	if topics, err := r.fetchAndCache(ctx); err == nil {
		r.topics = topics
		return nil
	}

	// 3. Fall back to defaults (rebased to the configured base URL)
	r.topics = RebaseTopics(r.BaseURL)
	if !r.Quiet {
		fmt.Fprintf(os.Stderr, "hint: using built-in defaults. Run 'plane docs update' to fetch latest docs index.\n")
	}
	return nil
}

// Update forces a remote refresh of the docs registry, bypassing the cache.
func (r *DocsRegistry) Update(ctx context.Context) error {
	topics, err := r.fetchAndCache(ctx)
	if err != nil {
		return err
	}
	r.topics = topics
	return nil
}

// Topics returns the loaded topics.
func (r *DocsRegistry) Topics() []Topic {
	return r.topics
}

// Lookup finds a topic by name (case-insensitive). If no topic matches,
// it searches all entry titles and URLs for a substring match and returns
// a synthetic single-entry topic if found.
func (r *DocsRegistry) Lookup(name string) *Topic {
	for i := range r.topics {
		if equalsIgnoreCase(r.topics[i].Name, name) {
			return &r.topics[i]
		}
	}

	// Backward compatibility: search across all entries for a substring match
	var matched []Entry
	for _, topic := range r.topics {
		for _, entry := range topic.Entries {
			if containsIgnoreCase(entry.Title, name) || containsIgnoreCase(entry.URL, name) {
				matched = append(matched, entry)
			}
		}
	}
	if len(matched) > 0 {
		return &Topic{
			Name:    name,
			Entries: matched,
		}
	}

	return nil
}

// LookupEntry finds a specific entry within a topic by matching action keywords.
func (r *DocsRegistry) LookupEntry(topicName, action string) *Entry {
	topic := r.Lookup(topicName)
	if topic == nil {
		return nil
	}
	for i := range topic.Entries {
		if containsIgnoreCase(topic.Entries[i].Title, action) {
			return &topic.Entries[i]
		}
	}
	return nil
}

func (r *DocsRegistry) fetchAndCache(ctx context.Context) ([]Topic, error) {
	raw, err := FetchLLMSTxt(ctx, r.BaseURL)
	if err != nil {
		return nil, err
	}

	topics, err := ParseLLMSTxt(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing llms.txt: %w", err)
	}
	if len(topics) == 0 {
		return nil, fmt.Errorf("llms.txt contained no topics")
	}

	// Best-effort cache write
	_ = writeCache(r.Profile, r.BaseURL, topics)

	return topics, nil
}

func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func containsIgnoreCase(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := range needle {
			ch, cn := haystack[i+j], needle[j]
			if ch >= 'A' && ch <= 'Z' {
				ch += 32
			}
			if cn >= 'A' && cn <= 'Z' {
				cn += 32
			}
			if ch != cn {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
