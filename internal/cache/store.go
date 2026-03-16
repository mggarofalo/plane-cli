package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mggarofalo/plane-cli/internal/docs"
)

// Store manages on-disk resource caches scoped to a profile.
type Store struct {
	profile string
	mu      sync.Mutex
}

// NewStore creates a Store for the given auth profile.
func NewStore(profile string) *Store {
	return &Store{profile: profile}
}

// baseDir returns the root directory for resource caches.
func (s *Store) baseDir() (string, error) {
	dir, err := docs.CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "resources", s.profile), nil
}

// kindPath returns the path to a cache file for a specific resource kind.
// Project-scoped resources include the projectID in the path; workspace-level
// resources (members, projects) use "_" as a placeholder.
func (s *Store) kindPath(workspace, projectID string, kind ResourceKind) (string, error) {
	base, err := s.baseDir()
	if err != nil {
		return "", err
	}
	proj := projectID
	if proj == "" {
		proj = "_"
	}
	return filepath.Join(base, workspace, proj, string(kind)+".json"), nil
}

// Load reads a cached resource from disk. Returns nil, nil if the file does
// not exist.
func (s *Store) Load(workspace, projectID string, kind ResourceKind) (*CachedResource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.kindPath(workspace, projectID, kind)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache %s: %w", path, err)
	}

	var cached CachedResource
	if err := json.Unmarshal(data, &cached); err != nil {
		// Corrupt file — treat as missing
		return nil, nil
	}

	return &cached, nil
}

// Save writes a cached resource to disk.
func (s *Store) Save(workspace, projectID string, kind ResourceKind, resource *CachedResource) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.kindPath(workspace, projectID, kind)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.MarshalIndent(resource, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// Clear removes all cached resources for the store's profile.
// If workspace is non-empty, only that workspace's caches are removed.
func (s *Store) Clear(workspace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	base, err := s.baseDir()
	if err != nil {
		return err
	}

	target := base
	if workspace != "" {
		target = filepath.Join(base, workspace)
	}

	if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clearing cache: %w", err)
	}
	return nil
}

// ClearKind removes a single resource kind's cache file.
func (s *Store) ClearKind(workspace, projectID string, kind ResourceKind) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.kindPath(workspace, projectID, kind)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clearing cache kind: %w", err)
	}
	return nil
}

// CacheFileInfo holds summary information about a single cache file.
type CacheFileInfo struct {
	Workspace string
	ProjectID string
	Kind      ResourceKind
	Resource  *CachedResource
}

// ListAll walks the cache directory and returns info for every cached file.
func (s *Store) ListAll() ([]CacheFileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	base, err := s.baseDir()
	if err != nil {
		return nil, err
	}

	var results []CacheFileInfo

	// Walk: base/{workspace}/{project}/{kind}.json
	workspaces, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, ws := range workspaces {
		if !ws.IsDir() {
			continue
		}
		wsPath := filepath.Join(base, ws.Name())
		projects, err := os.ReadDir(wsPath)
		if err != nil {
			continue
		}
		for _, proj := range projects {
			if !proj.IsDir() {
				continue
			}
			projPath := filepath.Join(wsPath, proj.Name())
			files, err := os.ReadDir(projPath)
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				ext := filepath.Ext(f.Name())
				if ext != ".json" {
					continue
				}
				kindName := f.Name()[:len(f.Name())-len(ext)]

				data, err := os.ReadFile(filepath.Join(projPath, f.Name()))
				if err != nil {
					continue
				}
				var cached CachedResource
				if err := json.Unmarshal(data, &cached); err != nil {
					continue
				}

				projectID := proj.Name()
				if projectID == "_" {
					projectID = ""
				}

				results = append(results, CacheFileInfo{
					Workspace: ws.Name(),
					ProjectID: projectID,
					Kind:      ResourceKind(kindName),
					Resource:  &cached,
				})
			}
		}
	}

	return results, nil
}
