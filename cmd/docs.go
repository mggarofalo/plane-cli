package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/mggarofalo/plane-cli/internal/cmdgen"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/spf13/cobra"
)

var flagDocsURL string

// specTopicSummary holds aggregated info about cached specs for a single topic.
type specTopicSummary struct {
	Name    string              `json:"name"`
	Files   []docs.SpecFileInfo `json:"files"`
	Oldest  time.Time           `json:"oldest"`
	Newest  time.Time           `json:"newest"`
	IsStale bool                `json:"is_stale"`
}

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.AddCommand(docsUpdateCmd)
	docsCmd.AddCommand(docsUpdateSpecsCmd)
	docsCmd.AddCommand(docsListSpecsCmd)

	docsCmd.PersistentFlags().StringVar(&flagDocsURL, "docs-url", "", "Docs base URL (default: profile or "+docs.DefaultBaseURL+")")
	docsCmd.Flags().Bool("list", false, "List all topics")
	docsCmd.Flags().String("url", "", "Fetch a specific URL directly")
}

var docsCmd = &cobra.Command{
	Use:   "docs [topic] [action]",
	Short: "Browse the Plane API documentation",
	Long: `Browse and retrieve Plane API documentation.

With no arguments, lists all available documentation topics.
With a topic, shows the available pages for that topic.
With a topic and action, fetches and displays the specific doc page.

The docs index is fetched from {docs-url}/llms.txt and cached locally.
Run 'plane docs update' to refresh the cache.`,
	Example: `  plane docs                     # List all topics
  plane docs issue               # List issue-related docs
  plane docs issue create        # Show the "Create Work Item" docs
  plane docs api-reference       # List all API reference entries
  plane docs update              # Refresh docs index from remote
  plane docs update-specs        # Pre-warm endpoint spec cache
  plane docs list-specs          # Show cached endpoint specs
  plane docs --url https://developers.plane.so/api-reference/issue/overview`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		directURL, _ := cmd.Flags().GetString("url")
		if directURL != "" {
			return fetchAndPrint(directURL)
		}

		registry, err := buildRegistry(cmd.Context())
		if err != nil {
			return err
		}

		if len(args) == 0 {
			return listTopics(registry)
		}

		topicName := args[0]
		topic := registry.Lookup(topicName)
		if topic == nil {
			return fmt.Errorf("unknown topic %q. Run 'plane docs' to see available topics", topicName)
		}

		if len(args) == 1 {
			if len(topic.Entries) == 1 {
				return fetchAndPrint(topic.Entries[0].URL)
			}
			return showTopic(topic)
		}

		action := args[1]
		entry := registry.LookupEntry(topicName, action)
		if entry == nil {
			if !flagQuiet {
				fmt.Fprintf(os.Stderr, "No match for %q in topic %q. Available pages:\n\n", action, topicName)
			}
			return showTopic(topic)
		}

		// When --output json is explicitly set, emit structured endpoint spec
		// instead of raw markdown. Only check the flag value (not auto-detect
		// from TTY) so piping docs to less/grep still gets markdown.
		if flagOutput == "json" {
			return fetchAndPrintSpec(cmd.Context(), topicName, *entry)
		}

		return fetchAndPrint(entry.URL)
	},
}

var docsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh the docs index from the remote llms.txt",
	RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := newRegistryNoLoad()
		if err != nil {
			return err
		}

		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Fetching docs index from %s/llms.txt ...\n", registry.BaseURL)
		}
		if err := registry.Update(cmd.Context()); err != nil {
			return fmt.Errorf("updating docs: %w", err)
		}

		total := 0
		for _, t := range registry.Topics() {
			total += len(t.Entries)
		}
		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Updated: %d topics, %d entries.\n", len(registry.Topics()), total)
		}
		return nil
	},
}

var docsUpdateSpecsCmd = &cobra.Command{
	Use:   "update-specs",
	Short: "Fetch and cache endpoint specs for all API topics",
	Long: `Pre-warms the per-command spec cache by fetching all API doc pages,
parsing endpoint metadata, and writing individual cache files.

Useful for:
- Pre-warming the cache for all commands
- Refreshing after Plane releases new API features
- CI/Docker image builds`,
	RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := buildRegistry(cmd.Context())
		if err != nil {
			return err
		}

		cfg, err := auth.LoadConfig()
		if err != nil {
			return err
		}
		profile := cfg.ActiveProfile
		baseURL := registry.BaseURL

		// Collect all entries to fetch
		type work struct {
			topicName string
			entry     docs.Entry
		}
		var entries []work
		for _, topic := range registry.Topics() {
			if !cmdgen.FilteredTopicName(topic.Name) {
				continue
			}
			for _, entry := range topic.Entries {
				if !cmdgen.IsAPIReferenceURL(entry.URL) {
					continue
				}
				cmdName := cmdgen.DeriveSubcommandName(entry.Title, topic.Name)
				if cmdName == "" {
					continue
				}
				entries = append(entries, work{topicName: topic.Name, entry: entry})
			}
		}

		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Fetching specs for %d endpoints...\n", len(entries))
		}

		// Bounded concurrent fetching
		const workers = 8
		var wg sync.WaitGroup
		ch := make(chan work, len(entries))
		var successCount int64
		var failCount int64

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for w := range ch {
					markdown, err := docs.Fetch(cmd.Context(), w.entry.URL)
					if err != nil {
						atomic.AddInt64(&failCount, 1)
						continue
					}
					spec := docs.ParseEndpointPage(markdown, w.topicName, w.entry)
					if spec == nil {
						atomic.AddInt64(&failCount, 1)
						continue
					}
					if err := docs.WriteSpec(profile, baseURL, spec); err != nil {
						atomic.AddInt64(&failCount, 1)
						continue
					}
					atomic.AddInt64(&successCount, 1)
				}
			}()
		}

		for _, w := range entries {
			ch <- w
		}
		close(ch)
		wg.Wait()

		success := atomic.LoadInt64(&successCount)
		fail := atomic.LoadInt64(&failCount)
		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Updated: %d endpoint specs cached", success)
			if fail > 0 {
				fmt.Fprintf(os.Stderr, " (%d failed)", fail)
			}
			fmt.Fprintln(os.Stderr, ".")
		}
		return nil
	},
}

var docsListSpecsCmd = &cobra.Command{
	Use:   "list-specs",
	Short: "List cached endpoint specs with freshness info",
	Long: `Shows which endpoint specs are currently cached, grouped by topic.

Displays file count, staleness, and total coverage so you can verify
cache state when debugging update-specs or skills generate issues.

Use --output json for machine-readable output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := auth.LoadConfig()
		if err != nil {
			return err
		}
		profile := cfg.ActiveProfile

		topics, err := docs.ListCachedTopics(profile)
		if err != nil {
			return fmt.Errorf("listing cached topics: %w", err)
		}

		if len(topics) == 0 {
			fmt.Fprintln(os.Stderr, "No cached specs found. Run 'plane docs update-specs' to populate.")
			return nil
		}

		sort.Strings(topics)

		// Build per-topic info
		topicInfos := []specTopicSummary{}
		var totalSpecs int

		for _, topicName := range topics {
			files, err := docs.ListTopicSpecFiles(profile, topicName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to list specs for topic %q: %v\n", topicName, err)
				continue
			}
			if len(files) == 0 {
				continue
			}

			sort.Slice(files, func(i, j int) bool {
				return files[i].FileName < files[j].FileName
			})

			oldest := files[0].FetchedAt
			newest := files[0].FetchedAt
			for _, f := range files[1:] {
				if f.FetchedAt.Before(oldest) {
					oldest = f.FetchedAt
				}
				if f.FetchedAt.After(newest) {
					newest = f.FetchedAt
				}
			}

			totalSpecs += len(files)
			topicInfos = append(topicInfos, specTopicSummary{
				Name:    topicName,
				Files:   files,
				Oldest:  oldest,
				Newest:  newest,
				IsStale: time.Since(oldest) > 24*time.Hour,
			})
		}

		// Count total possible endpoints from the registry (best-effort)
		var totalEndpoints int
		registry, regErr := buildRegistry(cmd.Context())
		if regErr == nil {
			for _, topic := range registry.Topics() {
				if !cmdgen.FilteredTopicName(topic.Name) {
					continue
				}
				for _, entry := range topic.Entries {
					if !cmdgen.IsAPIReferenceURL(entry.URL) {
						continue
					}
					if cmdgen.DeriveSubcommandName(entry.Title, topic.Name) != "" {
						totalEndpoints++
					}
				}
			}
		}

		if flagOutput == "json" {
			return printListSpecsJSON(topicInfos, totalSpecs, totalEndpoints)
		}

		return printListSpecsTable(profile, topicInfos, totalSpecs, totalEndpoints)
	},
}

func printListSpecsTable(profile string, topics []specTopicSummary, totalSpecs, totalEndpoints int) error {
	fmt.Printf("Cached endpoint specs (profile: %s)\n\n", profile)

	for _, t := range topics {
		fmt.Printf("  %-20s %d specs  (oldest: %s)\n", t.Name+"/", len(t.Files), formatAge(t.Oldest))
		for _, f := range t.Files {
			fmt.Printf("    %-36s %s  %s\n", f.FileName, f.FetchedAt.Format("2006-01-02"), formatSize(f.Size))
		}
	}

	fmt.Printf("\nTotal: %d specs across %d topics\n", totalSpecs, len(topics))
	if totalEndpoints > 0 {
		missing := totalEndpoints - totalSpecs
		if missing > 0 {
			fmt.Printf("Missing: %d endpoints (run 'plane docs update-specs' to populate)\n", missing)
		} else if missing < 0 {
			fmt.Printf("Note: %d extra cached specs (stale entries from removed endpoints?)\n", -missing)
		} else {
			fmt.Println("All endpoints cached.")
		}
	}

	return nil
}

func printListSpecsJSON(topics []specTopicSummary, totalSpecs, totalEndpoints int) error {
	result := struct {
		Topics         []specTopicSummary `json:"topics"`
		TotalSpecs     int         `json:"total_specs"`
		TotalEndpoints int         `json:"total_endpoints,omitempty"`
		MissingCount   int         `json:"missing_count,omitempty"`
	}{
		Topics:         topics,
		TotalSpecs:     totalSpecs,
		TotalEndpoints: totalEndpoints,
	}
	if totalEndpoints > 0 {
		missing := totalEndpoints - totalSpecs
		if missing > 0 {
			result.MissingCount = missing
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
}

func buildRegistry(ctx context.Context) (*docs.DocsRegistry, error) {
	registry, err := newRegistryNoLoad()
	if err != nil {
		return nil, err
	}
	if err := registry.Load(ctx); err != nil {
		return nil, fmt.Errorf("loading docs: %w", err)
	}
	return registry, nil
}

func newRegistryNoLoad() (*docs.DocsRegistry, error) {
	cfg, err := auth.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	resolver := &auth.Resolver{Config: cfg}
	profile := cfg.ActiveProfile
	docsURL := resolver.ResolveDocsURL(flagDocsURL)

	registry := docs.NewRegistry(profile, docsURL)
	registry.Quiet = flagQuiet
	return registry, nil
}

func listTopics(registry *docs.DocsRegistry) error {
	fmt.Println("Plane API Documentation Topics")
	fmt.Println("==============================")
	fmt.Println()
	fmt.Println("Usage: plane docs <topic> [action]")
	fmt.Println()

	for _, topic := range registry.Topics() {
		fmt.Printf("  %-24s  %d entries\n", topic.Name, len(topic.Entries))
	}

	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  plane docs issue            # list issue endpoint docs")
	fmt.Println("  plane docs issue create     # fetch 'Create Work Item' page")
	fmt.Println("  plane docs api-reference    # list all API reference entries")
	fmt.Println()
	fmt.Printf("Full docs: %s\n", registry.BaseURL)
	return nil
}

func showTopic(topic *docs.Topic) error {
	fmt.Printf("Topic: %s\n", topic.Name)
	fmt.Printf("%s\n\n", strings.Repeat("=", len(topic.Name)+7))

	for _, entry := range topic.Entries {
		fmt.Printf("  %-30s  %s\n", entry.Title, entry.URL)
	}

	fmt.Println()
	fmt.Printf("Fetch a page: plane docs %s <action>\n", topic.Name)
	fmt.Println("  action is matched as a substring against the page title")
	return nil
}

func fetchAndPrint(url string) error {
	if !flagQuiet {
		fmt.Fprintf(os.Stderr, "Fetching %s ...\n\n", url)
	}

	content, err := docs.Fetch(context.Background(), url)
	if err != nil {
		return err
	}

	fmt.Println(content)
	if !flagQuiet {
		fmt.Fprintf(os.Stderr, "\n---\nSource: %s\n", url)
	}
	return nil
}

// fetchAndPrintSpec resolves the endpoint spec for an entry (from cache or by
// fetching and parsing the doc page) and outputs it as structured JSON.
func fetchAndPrintSpec(ctx context.Context, topicName string, entry docs.Entry) error {
	cfg, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	profile := cfg.ActiveProfile

	// Try the spec cache first. Use SpecFileName (same key WriteSpec uses)
	// rather than DeriveSubcommandName (which strips topic aliases).
	cacheKey := docs.SpecFileName(entry.Title)
	if cacheKey != "" {
		if cached, err := docs.LoadSpec(profile, topicName, cacheKey); err == nil && cached != nil {
			return printSpecJSON(&cached.Spec)
		}
	}

	// Fetch the doc page and parse the spec
	if !flagQuiet {
		fmt.Fprintf(os.Stderr, "Fetching spec from %s ...\n", entry.URL)
	}
	markdown, err := docs.Fetch(ctx, entry.URL)
	if err != nil {
		return err
	}

	spec := docs.ParseEndpointPage(markdown, topicName, entry)
	if spec == nil {
		return fmt.Errorf("could not parse endpoint spec from %s", entry.URL)
	}

	// Best-effort cache write
	resolver := &auth.Resolver{Config: cfg}
	baseURL := resolver.ResolveDocsURL(flagDocsURL)
	_ = docs.WriteSpec(profile, baseURL, spec)

	return printSpecJSON(spec)
}

// printSpecJSON writes an EndpointSpec to stdout as indented JSON.
func printSpecJSON(spec *docs.EndpointSpec) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(spec)
}
