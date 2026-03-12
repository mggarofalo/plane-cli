package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/spf13/cobra"
)

var flagDocsURL string

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.AddCommand(docsUpdateCmd)

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
			fmt.Fprintf(os.Stderr, "No match for %q in topic %q. Available pages:\n\n", action, topicName)
			return showTopic(topic)
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

		fmt.Fprintf(os.Stderr, "Fetching docs index from %s/llms.txt ...\n", registry.BaseURL)
		if err := registry.Update(cmd.Context()); err != nil {
			return fmt.Errorf("updating docs: %w", err)
		}

		total := 0
		for _, t := range registry.Topics() {
			total += len(t.Entries)
		}
		fmt.Fprintf(os.Stderr, "Updated: %d topics, %d entries.\n", len(registry.Topics()), total)
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

	return docs.NewRegistry(profile, docsURL), nil
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
	fmt.Fprintf(os.Stderr, "Fetching %s ...\n\n", url)

	content, err := docs.Fetch(context.Background(), url)
	if err != nil {
		return err
	}

	fmt.Println(content)
	fmt.Fprintf(os.Stderr, "\n---\nSource: %s\n", url)
	return nil
}
