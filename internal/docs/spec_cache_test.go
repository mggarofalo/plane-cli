package docs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListTopicSpecFiles(t *testing.T) {
	// Use a temp dir as XDG_CACHE_HOME so SpecCacheDir resolves there.
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	profile := "test-profile"
	topicName := "issue"

	// Create the topic directory: <tmp>/plane-cli/specs/<profile>/<topic>
	topicDir := filepath.Join(tmp, "plane-cli", "specs", profile, topicName)
	if err := os.MkdirAll(topicDir, 0700); err != nil {
		t.Fatal(err)
	}

	now := time.Now().Truncate(time.Second)
	earlier := now.Add(-2 * time.Hour)

	// Write two spec files
	specs := []struct {
		name      string
		fetchedAt time.Time
	}{
		{"create-work-item.json", now},
		{"list-work-items.json", earlier},
	}

	for _, s := range specs {
		cached := CachedSpec{
			FetchedAt: s.fetchedAt,
			BaseURL:   "https://example.com",
			Spec: EndpointSpec{
				TopicName:  topicName,
				EntryTitle: s.name,
				Method:     "GET",
			},
		}
		data, err := json.MarshalIndent(cached, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(topicDir, s.name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Also create a non-JSON file that should be skipped
	if err := os.WriteFile(filepath.Join(topicDir, "README.txt"), []byte("ignore me"), 0600); err != nil {
		t.Fatal(err)
	}

	// Also create a subdirectory that should be skipped
	if err := os.MkdirAll(filepath.Join(topicDir, "subdir"), 0700); err != nil {
		t.Fatal(err)
	}

	files, err := ListTopicSpecFiles(profile, topicName)
	if err != nil {
		t.Fatalf("ListTopicSpecFiles: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Check that we got the right file names (they come in ReadDir order)
	fileNames := map[string]bool{}
	for _, f := range files {
		fileNames[f.FileName] = true
		if f.Size <= 0 {
			t.Errorf("expected positive size for %s, got %d", f.FileName, f.Size)
		}
		if f.FetchedAt.IsZero() {
			t.Errorf("expected non-zero FetchedAt for %s", f.FileName)
		}
	}

	if !fileNames["create-work-item.json"] {
		t.Error("missing create-work-item.json in results")
	}
	if !fileNames["list-work-items.json"] {
		t.Error("missing list-work-items.json in results")
	}
}

func TestListTopicSpecFiles_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	profile := "test-profile"
	topicName := "empty-topic"

	topicDir := filepath.Join(tmp, "plane-cli", "specs", profile, topicName)
	if err := os.MkdirAll(topicDir, 0700); err != nil {
		t.Fatal(err)
	}

	files, err := ListTopicSpecFiles(profile, topicName)
	if err != nil {
		t.Fatalf("ListTopicSpecFiles: %v", err)
	}

	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestListTopicSpecFiles_NonExistent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmp)

	files, err := ListTopicSpecFiles("no-such-profile", "no-such-topic")
	if err != nil {
		t.Fatalf("expected nil error for non-existent dir, got: %v", err)
	}
	if files != nil {
		t.Fatalf("expected nil files for non-existent dir, got %d", len(files))
	}
}

func TestSpecFileName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Create Work Item", "create-work-item"},
		{"List Work Items", "list-work-items"},
		{"Get a Module", "get-a-module"},
		{"  Trim  Spaces  ", "trim-spaces"},
		{"Special!@#Chars", "specialchars"},
	}

	for _, tt := range tests {
		got := SpecFileName(tt.input)
		if got != tt.want {
			t.Errorf("SpecFileName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
