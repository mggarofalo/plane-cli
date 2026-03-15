package selfupdate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheDir(t *testing.T) {
	// Test with XDG_CACHE_HOME set.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	dir, err := cacheDir()
	if err != nil {
		t.Fatalf("cacheDir() error: %v", err)
	}
	want := filepath.Join(tmpDir, cacheDirName)
	if dir != want {
		t.Errorf("cacheDir() = %q, want %q", dir, want)
	}
}

func TestCacheDirDefault(t *testing.T) {
	// Test without XDG_CACHE_HOME (uses ~/.cache).
	t.Setenv("XDG_CACHE_HOME", "")

	dir, err := cacheDir()
	if err != nil {
		t.Fatalf("cacheDir() error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".cache", cacheDirName)
	if dir != want {
		t.Errorf("cacheDir() = %q, want %q", dir, want)
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	now := time.Now().Truncate(time.Second)
	c := &checkCache{
		LastChecked:   now,
		LatestVersion: "1.2.3",
	}

	if err := saveCache(c); err != nil {
		t.Fatalf("saveCache() error: %v", err)
	}

	loaded, err := loadCache()
	if err != nil {
		t.Fatalf("loadCache() error: %v", err)
	}

	if !loaded.LastChecked.Truncate(time.Second).Equal(now) {
		t.Errorf("LastChecked = %v, want %v", loaded.LastChecked, now)
	}
	if loaded.LatestVersion != "1.2.3" {
		t.Errorf("LatestVersion = %q, want %q", loaded.LatestVersion, "1.2.3")
	}
}

func TestShouldCheckNoCache(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// No cache file exists, should check.
	if !ShouldCheck() {
		t.Error("ShouldCheck() = false with no cache, want true")
	}
}

func TestShouldCheckRecent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Write a recent cache entry.
	c := &checkCache{
		LastChecked:   time.Now(),
		LatestVersion: "1.0.0",
	}
	if err := saveCache(c); err != nil {
		t.Fatalf("saveCache() error: %v", err)
	}

	if ShouldCheck() {
		t.Error("ShouldCheck() = true with recent cache, want false")
	}
}

func TestShouldCheckStale(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Write a stale cache entry (25 hours ago).
	c := &checkCache{
		LastChecked:   time.Now().Add(-25 * time.Hour),
		LatestVersion: "1.0.0",
	}
	if err := saveCache(c); err != nil {
		t.Fatalf("saveCache() error: %v", err)
	}

	if !ShouldCheck() {
		t.Error("ShouldCheck() = false with stale cache, want true")
	}
}

func TestCachedLatestVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// No cache.
	if v := CachedLatestVersion(); v != "" {
		t.Errorf("CachedLatestVersion() = %q with no cache, want empty", v)
	}

	// With cache.
	c := &checkCache{
		LastChecked:   time.Now(),
		LatestVersion: "2.0.0",
	}
	if err := saveCache(c); err != nil {
		t.Fatalf("saveCache() error: %v", err)
	}

	if v := CachedLatestVersion(); v != "2.0.0" {
		t.Errorf("CachedLatestVersion() = %q, want %q", v, "2.0.0")
	}
}

func TestLoadCacheCorrupt(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", tmpDir)

	// Write corrupt data.
	path := filepath.Join(tmpDir, cacheDirName, cacheFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("mkdir error: %v", err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Should return error.
	_, err := loadCache()
	if err == nil {
		t.Error("loadCache() with corrupt data should return error")
	}

	// ShouldCheck should return true when cache is corrupt.
	if !ShouldCheck() {
		t.Error("ShouldCheck() = false with corrupt cache, want true")
	}
}
