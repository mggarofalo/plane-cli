package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/mggarofalo/plane-cli/internal/cmdgen"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/spf13/cobra"
)

var flagDocsURL string

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.AddCommand(docsUpdateCmd)
	docsCmd.AddCommand(docsUpdateSpecsCmd)

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
	cmdName := cmdgen.DeriveSubcommandName(entry.Title, topicName)

	// Try the spec cache first
	if cmdName != "" {
		if cached, err := docs.LoadSpec(profile, topicName, cmdName); err == nil && cached != nil {
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
