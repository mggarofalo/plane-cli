package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const specStaleDuration = 24 * time.Hour

// CachedSpec is the on-disk representation of a single endpoint spec.
type CachedSpec struct {
	FetchedAt time.Time    `json:"fetched_at"`
	BaseURL   string       `json:"base_url"`
	Spec      EndpointSpec `json:"spec"`
}

// IsStale returns true if the spec is older than 24 hours.
func (c *CachedSpec) IsStale() bool {
	return time.Since(c.FetchedAt) > specStaleDuration
}

// SpecCacheDir returns the directory for per-command spec caches.
func SpecCacheDir(profile string) (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "specs", profile), nil
}

// LoadSpec reads a single cached spec file.
func LoadSpec(profile, topicName, cmdName string) (*CachedSpec, error) {
	dir, err := SpecCacheDir(profile)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, topicName, cmdName+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cached CachedSpec
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("parsing spec cache %s: %w", path, err)
	}

	return &cached, nil
}

// WriteSpec writes a single spec to its cache file.
func WriteSpec(profile, baseURL string, spec *EndpointSpec) error {
	dir, err := SpecCacheDir(profile)
	if err != nil {
		return err
	}

	cmdName := SpecFileName(spec.EntryTitle)
	topicDir := filepath.Join(dir, spec.TopicName)
	if err := os.MkdirAll(topicDir, 0700); err != nil {
		return fmt.Errorf("creating spec cache dir: %w", err)
	}

	cached := CachedSpec{
		FetchedAt: time.Now(),
		BaseURL:   baseURL,
		Spec:      *spec,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling spec: %w", err)
	}

	path := filepath.Join(topicDir, cmdName+".json")
	return os.WriteFile(path, data, 0600)
}

// LoadTopicSpecs reads all cached specs for a given topic.
func LoadTopicSpecs(profile, topicName string) ([]CachedSpec, error) {
	dir, err := SpecCacheDir(profile)
	if err != nil {
		return nil, err
	}

	topicDir := filepath.Join(dir, topicName)
	entries, err := os.ReadDir(topicDir)
	if err != nil {
		return nil, err
	}

	var specs []CachedSpec
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(topicDir, e.Name()))
		if err != nil {
			continue
		}

		var cached CachedSpec
		if err := json.Unmarshal(data, &cached); err != nil {
			continue
		}
		specs = append(specs, cached)
	}

	return specs, nil
}

// ListCachedTopics returns the names of topic directories in the spec cache.
func ListCachedTopics(profile string) ([]string, error) {
	dir, err := SpecCacheDir(profile)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var topics []string
	for _, e := range entries {
		if e.IsDir() {
			topics = append(topics, e.Name())
		}
	}
	return topics, nil
}

// SpecFileInfo holds lightweight metadata about a single cached spec file.
type SpecFileInfo struct {
	FileName  string    `json:"file_name"`
	Size      int64     `json:"size"`
	FetchedAt time.Time `json:"fetched_at"`
}

// ListTopicSpecFiles returns file-level metadata for all cached specs in a topic.
// Unlike LoadTopicSpecs, it only reads enough of each file to extract the
// fetched_at timestamp and uses os.Stat for the file size.
func ListTopicSpecFiles(profile, topicName string) ([]SpecFileInfo, error) {
	dir, err := SpecCacheDir(profile)
	if err != nil {
		return nil, err
	}

	topicDir := filepath.Join(dir, topicName)
	entries, err := os.ReadDir(topicDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []SpecFileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		// Read just enough to extract fetched_at
		data, err := os.ReadFile(filepath.Join(topicDir, e.Name()))
		if err != nil {
			continue
		}

		var envelope struct {
			FetchedAt time.Time `json:"fetched_at"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}

		files = append(files, SpecFileInfo{
			FileName:  e.Name(),
			Size:      info.Size(),
			FetchedAt: envelope.FetchedAt,
		})
	}

	return files, nil
}

// SpecFileName derives a cache file name from an entry title.
// Exported so callers can look up specs by entry title.
func SpecFileName(entryTitle string) string {
	lower := strings.ToLower(entryTitle)
	lower = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, lower)
	// Collapse multiple dashes
	for strings.Contains(lower, "--") {
		lower = strings.ReplaceAll(lower, "--", "-")
	}
	return strings.Trim(lower, "-")
}
