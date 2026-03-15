// Package selfupdate provides background update checking and self-update logic
// for plane-cli. It uses the GitHub Releases API to detect new versions and
// caches the last check timestamp to avoid excessive API calls.
package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

const (
	// GitHubRepo is the owner/repo slug for release detection.
	GitHubRepo = "mggarofalo/plane-cli"

	// CheckInterval is the minimum time between update checks.
	CheckInterval = 24 * time.Hour

	// cacheFileName is the name of the update-check cache file.
	cacheFileName = "update-check.json"

	// cacheDirName is the application cache directory name.
	cacheDirName = "plane-cli"
)

// CheckResult holds the result of an update check.
type CheckResult struct {
	// NewVersionAvailable is true if a newer version exists.
	NewVersionAvailable bool
	// LatestVersion is the latest version string (e.g., "0.2.0").
	LatestVersion string
}

// checkCache is the on-disk format for the update-check cache.
type checkCache struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version,omitempty"`
}

// cacheDir returns the cache directory, respecting XDG_CACHE_HOME.
func cacheDir() (string, error) {
	dir := os.Getenv("XDG_CACHE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, cacheDirName), nil
}

// cachePath returns the full path to the update-check cache file.
func cachePath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, cacheFileName), nil
}

// loadCache reads the cached check data from disk.
func loadCache() (*checkCache, error) {
	path, err := cachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c checkCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// saveCache writes check data to disk.
func saveCache(c *checkCache) error {
	path, err := cachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ShouldCheck returns true if enough time has passed since the last check.
func ShouldCheck() bool {
	c, err := loadCache()
	if err != nil {
		// No cache or corrupt — should check.
		return true
	}
	return time.Since(c.LastChecked) >= CheckInterval
}

// CheckForUpdate queries GitHub Releases for the latest version and compares
// it against currentVersion. It updates the on-disk cache regardless of result.
func CheckForUpdate(ctx context.Context, currentVersion string) (*CheckResult, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("creating GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return nil, fmt.Errorf("creating updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(GitHubRepo))
	if err != nil {
		return nil, fmt.Errorf("detecting latest version: %w", err)
	}

	result := &CheckResult{}

	if found && latest.Version() != "" {
		result.LatestVersion = latest.Version()
		result.NewVersionAvailable = latest.GreaterThan(currentVersion)
	}

	// Always update the cache timestamp.
	_ = saveCache(&checkCache{
		LastChecked:   time.Now(),
		LatestVersion: result.LatestVersion,
	})

	return result, nil
}

// CachedLatestVersion returns the latest version from the last check, if any.
// This is used for the startup notification without making an API call.
func CachedLatestVersion() string {
	c, err := loadCache()
	if err != nil {
		return ""
	}
	return c.LatestVersion
}
