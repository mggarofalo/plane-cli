package cmd

import (
	"testing"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

func TestCollectSpecEntries_UsesDefaultTopics(t *testing.T) {
	entries := collectSpecEntries(docs.DefaultTopics)

	// DefaultTopics has ~109 individual endpoint entries (after filtering
	// out overview pages and overridden topics like introduction/user).
	// The llms.txt cache would only produce ~18 entries. Assert we get a
	// count consistent with DefaultTopics, not the cache.
	if len(entries) < 80 {
		t.Errorf("collectSpecEntries returned %d entries, want >= 80 (DefaultTopics has ~109 endpoints)", len(entries))
	}

	// Verify every collected entry has the required fields populated.
	for i, e := range entries {
		if e.topicName == "" {
			t.Errorf("entry %d: topicName is empty", i)
		}
		if e.entry.Title == "" {
			t.Errorf("entry %d (topic %s): entry.Title is empty", i, e.topicName)
		}
		if e.entry.URL == "" {
			t.Errorf("entry %d (topic %s): entry.URL is empty", i, e.topicName)
		}
	}
}

func TestCollectSpecEntries_FiltersOverriddenTopics(t *testing.T) {
	entries := collectSpecEntries(docs.DefaultTopics)

	for _, e := range entries {
		if e.topicName == "introduction" {
			t.Error("collectSpecEntries included 'introduction' topic, which should be filtered")
		}
		if e.topicName == "user" {
			t.Error("collectSpecEntries included 'user' topic, which should be filtered")
		}
	}
}

func TestCollectSpecEntries_SkipsOverviewEntries(t *testing.T) {
	entries := collectSpecEntries(docs.DefaultTopics)

	for _, e := range entries {
		if e.entry.Title == "Overview" {
			t.Errorf("collectSpecEntries included Overview entry for topic %q", e.topicName)
		}
		if e.entry.Title == "API Introduction" {
			t.Errorf("collectSpecEntries included API Introduction entry for topic %q", e.topicName)
		}
	}
}

func TestCollectSpecEntries_EmptyInput(t *testing.T) {
	entries := collectSpecEntries(nil)
	if len(entries) != 0 {
		t.Errorf("collectSpecEntries(nil) returned %d entries, want 0", len(entries))
	}

	entries = collectSpecEntries([]docs.Topic{})
	if len(entries) != 0 {
		t.Errorf("collectSpecEntries([]) returned %d entries, want 0", len(entries))
	}
}

func TestCollectSpecEntries_MatchesSkillgenSource(t *testing.T) {
	// The key invariant: collectSpecEntries must iterate the same source
	// (DefaultTopics) and apply the same filtering as skillgen.Generate.
	// Count the entries that skillgen would process.
	var skillgenCount int
	for _, topic := range docs.DefaultTopics {
		for _, entry := range topic.Entries {
			if entry.Title == "Overview" || entry.Title == "API Introduction" {
				continue
			}
			// skillgen also uses IsAPIReferenceURL but all DefaultTopics
			// entries have api-reference URLs, so just count non-overview.
			skillgenCount++
		}
	}

	// collectSpecEntries applies the same filters plus FilteredTopicName
	// (which excludes introduction and user topics).
	entries := collectSpecEntries(docs.DefaultTopics)

	// The count from collectSpecEntries should equal skillgenCount minus
	// entries from filtered-out topics (introduction has 0 after Overview
	// filter, user has 1 "Get Current User").
	// This is a sanity check — exact numbers may change as DefaultTopics
	// is updated, but they should be close.
	if len(entries) < skillgenCount-5 {
		t.Errorf("collectSpecEntries returned %d entries, but skillgen-equivalent count is %d; they should be close", len(entries), skillgenCount)
	}
}
