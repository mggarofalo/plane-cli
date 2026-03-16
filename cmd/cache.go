package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/mggarofalo/plane-cli/internal/cache"
	"github.com/mggarofalo/plane-cli/internal/output"
	"github.com/spf13/cobra"
)

var flagCacheWorkspace string

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheShowCmd)

	cacheClearCmd.Flags().StringVarP(&flagCacheWorkspace, "workspace", "w", "", "Only clear caches for this workspace")
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the local resource cache",
	Long: `Manage the local cache of Plane resources (states, labels, cycles, etc.).

The CLI caches name-to-UUID lookup results to avoid repeated API calls.
Cached entries are valid for 1 hour (soft TTL) and expire after 7 days (hard TTL).
Stale entries trigger a background refresh while still returning the cached value.`,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear cached resources",
	Long: `Remove all cached resource data for the active profile.
Use --workspace to limit clearing to a specific workspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := auth.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		store := cache.NewStore(cfg.ActiveProfile)
		if err := store.Clear(flagCacheWorkspace); err != nil {
			return fmt.Errorf("clearing cache: %w", err)
		}

		if !flagQuiet {
			if flagCacheWorkspace != "" {
				fmt.Fprintf(os.Stderr, "Cleared resource cache for workspace %q.\n", flagCacheWorkspace)
			} else {
				fmt.Fprintf(os.Stderr, "Cleared all resource caches.\n")
			}
		}
		return nil
	},
}

var cacheShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show cached resource summary",
	Long: `Display a summary of all cached resource files for the active profile.
Shows workspace, project, resource kind, entry count, fetch time, and staleness.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := auth.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		store := cache.NewStore(cfg.ActiveProfile)
		items, err := store.ListAll()
		if err != nil {
			return fmt.Errorf("listing cache: %w", err)
		}

		if len(items) == 0 {
			if !flagQuiet {
				fmt.Fprintln(os.Stderr, "No cached resources.")
			}
			return nil
		}

		f := Formatter()
		if _, ok := f.(*output.TableFormatter); ok {
			return showCacheTable(items)
		}
		return showCacheJSON(items)
	},
}

func showCacheTable(items []cache.CacheFileInfo) error {
	fmt.Printf("%-20s %-38s %-10s %5s  %-20s  %s\n",
		"WORKSPACE", "PROJECT", "KIND", "COUNT", "FETCHED", "STATUS")
	for _, item := range items {
		proj := item.ProjectID
		if proj == "" {
			proj = "(workspace)"
		}
		status := "fresh"
		if item.Resource.IsExpired() {
			status = "expired"
		} else if item.Resource.IsStale() {
			status = "stale"
		}
		fmt.Printf("%-20s %-38s %-10s %5d  %-20s  %s\n",
			item.Workspace,
			proj,
			item.Kind,
			len(item.Resource.Entries),
			item.Resource.FetchedAt.Local().Format("2006-01-02 15:04:05"),
			status,
		)
	}
	return nil
}

type cacheShowEntry struct {
	Workspace string `json:"workspace"`
	ProjectID string `json:"project_id,omitempty"`
	Kind      string `json:"kind"`
	Count     int    `json:"count"`
	FetchedAt string `json:"fetched_at"`
	Stale     bool   `json:"stale"`
	Expired   bool   `json:"expired"`
}

func showCacheJSON(items []cache.CacheFileInfo) error {
	entries := make([]cacheShowEntry, 0, len(items))
	for _, item := range items {
		entries = append(entries, cacheShowEntry{
			Workspace: item.Workspace,
			ProjectID: item.ProjectID,
			Kind:      string(item.Kind),
			Count:     len(item.Resource.Entries),
			FetchedAt: item.Resource.FetchedAt.Format("2006-01-02T15:04:05Z07:00"),
			Stale:     item.Resource.IsStale(),
			Expired:   item.Resource.IsExpired(),
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}
