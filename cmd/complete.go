package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/spf13/cobra"
)

var flagCompleteParam string

func init() {
	rootCmd.AddCommand(completeCmd)
	completeCmd.Flags().StringVar(&flagCompleteParam, "param", "", "Parameter name to complete (state, priority, label/labels)")
	completeCmd.MarkFlagRequired("param")
}

var completeCmd = &cobra.Command{
	Use:   "complete",
	Short: "List valid values for enum-like parameters",
	Long: `Returns a JSON array of valid values for a given parameter name.

Designed for agent discovery: instead of trial-and-error or reading docs,
agents can query which values a parameter accepts.

Supported parameters:
  state     - project state names (requires --project)
  priority  - valid priority values (static list)
  label(s)  - project label names (requires --project)`,
	Example: `  plane complete --param state -p MYPROJ
  plane complete --param priority
  plane complete --param labels -p MYPROJ`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runComplete(cmd.Context(), flagCompleteParam)
	},
}

// runComplete resolves valid values for the given parameter and outputs them
// as a JSON array to stdout.
func runComplete(ctx context.Context, param string) error {
	normalized := strings.ToLower(strings.TrimSpace(param))

	switch normalized {
	case "priority":
		return outputJSON([]string{"urgent", "high", "medium", "low", "none"})

	case "state":
		return completeFromAPI(ctx, "state")

	case "label", "labels":
		return completeFromAPI(ctx, "label")

	default:
		return fmt.Errorf("unsupported parameter %q; supported: state, priority, label(s)", param)
	}
}

// completeFromAPI queries the Plane API for a list resource and extracts names.
func completeFromAPI(ctx context.Context, resource string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
	if err := RequireWorkspace(client); err != nil {
		return err
	}

	projectID, err := RequireProject()
	if err != nil {
		return err
	}

	listURL := completeListURL(client, resource, projectID)
	if listURL == "" {
		return fmt.Errorf("no list endpoint for %q", resource)
	}

	respBody, err := client.GetPaginated(ctx, listURL, api.PaginationParams{PerPage: 100})
	if err != nil {
		return fmt.Errorf("fetching %s list: %w", resource, err)
	}

	names, err := extractNames(respBody)
	if err != nil {
		return fmt.Errorf("parsing %s response: %w", resource, err)
	}

	return outputJSON(names)
}

// completeListURL returns the list endpoint URL for a completable resource.
func completeListURL(client *api.Client, resource, projectID string) string {
	base := fmt.Sprintf("%s/api/v1/workspaces/%s", client.BaseURL, client.Workspace)
	switch resource {
	case "state":
		return fmt.Sprintf("%s/projects/%s/states/", base, projectID)
	case "label":
		return fmt.Sprintf("%s/projects/%s/labels/", base, projectID)
	default:
		return ""
	}
}

// extractNames pulls the "name" field from each object in a paginated or array response.
func extractNames(respBody []byte) ([]string, error) {
	// Try paginated response first
	var paginated struct {
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(respBody, &paginated); err == nil && paginated.Results != nil {
		var items []json.RawMessage
		if err := json.Unmarshal(paginated.Results, &items); err == nil {
			return namesFromItems(items)
		}
	}

	// Try plain array
	var items []json.RawMessage
	if err := json.Unmarshal(respBody, &items); err == nil {
		return namesFromItems(items)
	}

	return nil, fmt.Errorf("unexpected response format")
}

// namesFromItems extracts the "name" field from each JSON object.
// Always returns a non-nil slice so json.Marshal produces [] not null.
func namesFromItems(items []json.RawMessage) ([]string, error) {
	names := make([]string, 0, len(items))
	for _, raw := range items {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}
		if name, ok := obj["name"].(string); ok && name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// outputJSON marshals the values as a JSON array and writes to stdout.
func outputJSON(values []string) error {
	data, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshaling output: %w", err)
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}
