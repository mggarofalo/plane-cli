package cache

import (
	"strings"
	"time"
)

const (
	// SoftTTL is how long a cached resource is fresh. After this, a background
	// refresh is triggered but the cached value is still returned.
	SoftTTL = 1 * time.Hour

	// HardTTL is the maximum age of a cached resource. After this, the cache
	// entry is treated as expired and a synchronous refresh is required.
	HardTTL = 7 * 24 * time.Hour
)

// ResourceKind identifies a type of Plane resource that can be cached.
type ResourceKind string

const (
	KindStates  ResourceKind = "states"
	KindLabels  ResourceKind = "labels"
	KindCycles  ResourceKind = "cycles"
	KindModules ResourceKind = "modules"
	KindMembers ResourceKind = "members"
	KindProjects ResourceKind = "projects"
)

// AllKinds lists every resource kind for iteration.
var AllKinds = []ResourceKind{
	KindStates, KindLabels, KindCycles, KindModules, KindMembers, KindProjects,
}

// ResourceKindFromParam maps a parameter name (e.g. "state", "state_id") to
// a ResourceKind. Returns empty string if unknown.
func ResourceKindFromParam(paramName string) ResourceKind {
	name := strings.TrimSuffix(paramName, "_id")
	switch name {
	case "state":
		return KindStates
	case "label":
		return KindLabels
	case "cycle":
		return KindCycles
	case "module":
		return KindModules
	case "member":
		return KindMembers
	case "project":
		return KindProjects
	}
	return ""
}

// Entry represents a single cached resource (state, label, etc.).
type Entry struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Identifier  string `json:"identifier,omitempty"`
}

// CachedResource holds a timestamped list of entries for a resource kind.
type CachedResource struct {
	FetchedAt time.Time `json:"fetched_at"`
	Entries   []Entry   `json:"entries"`
}

// IsStale returns true if the cache is older than SoftTTL. A stale cache
// triggers a background refresh but can still be used.
func (c *CachedResource) IsStale() bool {
	return time.Since(c.FetchedAt) > SoftTTL
}

// IsExpired returns true if the cache is older than HardTTL. An expired cache
// must not be used; a synchronous refresh is required.
func (c *CachedResource) IsExpired() bool {
	return time.Since(c.FetchedAt) > HardTTL
}

// FindByName searches entries for a case-insensitive match across name,
// display_name, and identifier fields. Returns the matching entry's ID,
// or empty string if not found.
func (c *CachedResource) FindByName(name string) string {
	lower := strings.ToLower(name)
	for _, e := range c.Entries {
		if strings.ToLower(e.Name) == lower {
			return e.ID
		}
		if e.DisplayName != "" && strings.ToLower(e.DisplayName) == lower {
			return e.ID
		}
		if e.Identifier != "" && strings.ToLower(e.Identifier) == lower {
			return e.ID
		}
	}
	return ""
}
