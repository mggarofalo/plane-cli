package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CachedRegistry is the on-disk representation of a fetched docs registry.
type CachedRegistry struct {
	FetchedAt time.Time `json:"fetched_at"`
	BaseURL   string    `json:"base_url"`
	Topics    []Topic   `json:"topics"`
}

// CacheDir returns the directory used for caching, respecting XDG_CACHE_HOME.
func CacheDir() (string, error) {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "plane-cli"), nil
}

// CachePath returns the path to the cache file for a given profile.
func CachePath(profile string) (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "docs-"+profile+".json"), nil
}

// loadCache reads the cached registry from disk. Returns an error if the cache
// doesn't exist, can't be read, or the baseURL doesn't match expectedBaseURL.
func loadCache(profile, expectedBaseURL string) ([]Topic, error) {
	path, err := CachePath(profile)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading cache: %w", err)
	}

	var cached CachedRegistry
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("parsing cache: %w", err)
	}

	if cached.BaseURL != expectedBaseURL {
		return nil, fmt.Errorf("cache baseURL %q does not match expected %q", cached.BaseURL, expectedBaseURL)
	}

	return cached.Topics, nil
}

// writeCache writes topics to disk as a JSON cache file.
func writeCache(profile, baseURL string, topics []Topic) error {
	path, err := CachePath(profile)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	cached := CachedRegistry{
		FetchedAt: time.Now(),
		BaseURL:   baseURL,
		Topics:    topics,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}

	return nil
}
