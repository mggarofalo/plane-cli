package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(docsCmd)

	docsCmd.Flags().Bool("list", false, "List all topics")
	docsCmd.Flags().String("url", "", "Fetch a specific URL directly")
}

var docsCmd = &cobra.Command{
	Use:   "docs [topic] [action]",
	Short: "Browse the Plane API documentation",
	Long: `Browse and retrieve Plane API documentation from developers.plane.so.

With no arguments, lists all available documentation topics.
With a topic, shows the available pages for that topic.
With a topic and action, fetches and displays the specific doc page.`,
	Example: `  plane docs                     # List all topics
  plane docs issue               # List issue-related docs
  plane docs issue create        # Show the "Create Work Item" docs
  plane docs introduction        # Show the API introduction
  plane docs --url https://developers.plane.so/api-reference/issue/overview`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		directURL, _ := cmd.Flags().GetString("url")
		if directURL != "" {
			return fetchAndPrint(directURL)
		}

		if len(args) == 0 {
			return listTopics()
		}

		topicName := args[0]
		topic := docs.Lookup(topicName)
		if topic == nil {
			return fmt.Errorf("unknown topic %q. Run 'plane docs' to see available topics", topicName)
		}

		if len(args) == 1 {
			// If topic has only 1 entry, fetch it directly
			if len(topic.Entries) == 1 {
				return fetchAndPrint(topic.Entries[0].URL())
			}
			return showTopic(topic)
		}

		// Find matching entry by action keyword
		action := args[1]
		entry := docs.LookupEntry(topicName, action)
		if entry == nil {
			fmt.Fprintf(os.Stderr, "No match for %q in topic %q. Available pages:\n\n", action, topicName)
			return showTopic(topic)
		}

		return fetchAndPrint(entry.URL())
	},
}

func listTopics() error {
	fmt.Println("Plane API Documentation Topics")
	fmt.Println("==============================")
	fmt.Println()
	fmt.Println("Usage: plane docs <topic> [action]")
	fmt.Println()

	for _, topic := range docs.Registry {
		fmt.Printf("  %-16s  %d pages\n", topic.Name, len(topic.Entries))
	}

	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  plane docs issue            # list issue endpoint docs")
	fmt.Println("  plane docs issue create     # fetch 'Create Work Item' page")
	fmt.Println("  plane docs introduction     # API auth, pagination, rate limits")
	fmt.Println()
	fmt.Printf("Full docs: %s\n", docs.BaseURL)
	return nil
}

func showTopic(topic *docs.Topic) error {
	fmt.Printf("Topic: %s\n", topic.Name)
	fmt.Printf("%s\n\n", strings.Repeat("=", len(topic.Name)+7))

	for _, entry := range topic.Entries {
		fmt.Printf("  %-30s  %s\n", entry.Title, entry.URL())
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
