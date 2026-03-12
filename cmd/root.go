package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/mggarofalo/plane-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagWorkspace string
	flagProject   string
	flagOutput    string
	flagAPIURL    string
	flagAPIKey    string
	flagVerbose   bool
	flagPerPage   int
	flagCursor    string
	flagAll       bool
)

var rootCmd = &cobra.Command{
	Use:   "plane",
	Short: "CLI for the Plane project management API",
	Long: `plane is a command-line tool for interacting with the Plane project management API (plane.so).
Designed for both AI agents and humans.

API docs: https://developers.plane.so/api-reference/introduction
Browse docs: plane docs`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&flagWorkspace, "workspace", "w", "", "Workspace slug")
	pf.StringVarP(&flagProject, "project", "p", "", "Project ID or identifier")
	pf.StringVarP(&flagOutput, "output", "o", "json", "Output format: json, table")
	pf.StringVar(&flagAPIURL, "api-url", "", "Base API URL")
	pf.StringVar(&flagAPIKey, "api-key", "", "API key (prefer keyring or env var)")
	pf.BoolVar(&flagVerbose, "verbose", false, "Debug HTTP logging (tokens redacted)")
	pf.IntVar(&flagPerPage, "per-page", 100, "Items per page (max 100)")
	pf.StringVar(&flagCursor, "cursor", "", "Pagination cursor")
	pf.BoolVar(&flagAll, "all", false, "Auto-paginate and return all results")
}

// Execute runs the root command and returns an exit code.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return api.ExitCodeFromError(err)
	}
	return api.ExitSuccess
}

// newClient creates an API client from resolved flags, env, and config.
func newClient() (*api.Client, error) {
	cfg, err := auth.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	resolver := &auth.Resolver{
		FlagToken: flagAPIKey,
		Env:       &auth.EnvSource{},
		Config:    cfg,
	}

	// Try keyring (best-effort — may fail in CI/headless)
	store, err := auth.NewKeyringStore("")
	if err == nil {
		resolver.Store = store
	}

	resolved, err := resolver.Resolve()
	if err != nil {
		return nil, err
	}
	defer resolved.Credential.Clear()

	apiURL := resolver.ResolveAPIURL(flagAPIURL)
	if apiURL == "" {
		return nil, fmt.Errorf("no API URL configured. Run 'plane auth login' or set %s", auth.EnvURL)
	}

	workspace := resolver.ResolveWorkspace(flagWorkspace)

	var debugWriter *os.File
	if flagVerbose {
		debugWriter = os.Stderr
	}

	return api.NewClient(apiURL, resolved.Credential.Token(), workspace, flagVerbose, debugWriter), nil
}

// requireWorkspace returns the resolved workspace, or an error.
func requireWorkspace(client *api.Client) error {
	if client.Workspace == "" {
		return fmt.Errorf("workspace is required. Use --workspace flag, set %s, or configure via 'plane auth login'", auth.EnvWorkspace)
	}
	return nil
}

// requireProject returns the project UUID. If --project is a short identifier
// (e.g. "PLANECLI"), it resolves it to a UUID via the API.
func requireProject() (string, error) {
	if flagProject == "" {
		return "", fmt.Errorf("project is required. Use --project flag")
	}
	// If it looks like a UUID, use it directly
	if len(flagProject) == 36 && strings.Count(flagProject, "-") == 4 {
		return flagProject, nil
	}
	// Otherwise treat it as an identifier and resolve
	return resolveProjectIdentifier(flagProject)
}

// resolvedProjects caches identifier→UUID lookups within a session.
var resolvedProjects = map[string]string{}

func resolveProjectIdentifier(identifier string) (string, error) {
	upper := strings.ToUpper(identifier)
	if id, ok := resolvedProjects[upper]; ok {
		return id, nil
	}

	client, err := newClient()
	if err != nil {
		return "", err
	}

	svc := api.NewProjectsService(client)
	resp, err := svc.List(context.Background(), api.PaginationParams{PerPage: 100})
	if err != nil {
		return "", fmt.Errorf("resolving project identifier %q: %w", identifier, err)
	}

	for _, p := range resp.Results {
		resolvedProjects[strings.ToUpper(p.Identifier)] = p.ID
	}

	if id, ok := resolvedProjects[upper]; ok {
		return id, nil
	}
	return "", fmt.Errorf("project with identifier %q not found", identifier)
}

// resolveStateName resolves a state name (e.g. "Backlog") to its UUID for a given project.
func resolveStateName(projectID, name string) (string, error) {
	client, err := newClient()
	if err != nil {
		return "", err
	}

	svc := api.NewStatesService(client)
	states, err := svc.List(context.Background(), projectID)
	if err != nil {
		return "", fmt.Errorf("resolving state %q: %w", name, err)
	}

	lower := strings.ToLower(name)
	for _, s := range states {
		if strings.ToLower(s.Name) == lower {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("state %q not found in project", name)
}

// resolveIssueRef resolves an issue reference (UUID or sequence identifier like "PLANECLI-2") to a UUID.
func resolveIssueRef(projectID, issueRef string) (string, error) {
	if isUUID(issueRef) {
		return issueRef, nil
	}
	// Sequence identifier — look up via API
	client, err := newClient()
	if err != nil {
		return "", err
	}
	svc := api.NewIssuesService(client)
	issue, err := svc.GetBySequence(context.Background(), projectID, issueRef)
	if err != nil {
		return "", err
	}
	return issue.ID, nil
}

// paginationParams returns pagination params from flags.
func paginationParams() api.PaginationParams {
	return api.PaginationParams{
		PerPage: flagPerPage,
		Cursor:  flagCursor,
	}
}

// formatter returns the output formatter based on the --output flag.
func formatter() output.Formatter {
	return output.New(flagOutput)
}
