package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/mggarofalo/plane-cli/internal/cmdgen"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/mggarofalo/plane-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	flagDryRun    bool
	flagField     string
	flagFields    string
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
	pf.StringVarP(&flagOutput, "output", "o", "", "Output format: json, table (default: table for TTY, json otherwise)")
	pf.StringVar(&flagAPIURL, "api-url", "", "Base API URL")
	pf.StringVar(&flagAPIKey, "api-key", "", "API key (prefer keyring or env var)")
	pf.BoolVar(&flagVerbose, "verbose", false, "Debug HTTP logging (tokens redacted)")
	pf.IntVar(&flagPerPage, "per-page", 100, "Items per page (max 100)")
	pf.StringVar(&flagCursor, "cursor", "", "Pagination cursor")
	pf.BoolVar(&flagAll, "all", false, "Auto-paginate and return all results")
	pf.BoolVarP(&flagDryRun, "dry-run", "n", false, "Print request details without executing")
	pf.StringVar(&flagField, "field", "", "Extract a single field from JSON response (supports dotted paths, e.g. state_detail.name)")
	pf.StringVar(&flagFields, "fields", "", "Extract multiple fields as TSV (comma-separated, e.g. id,name,state_detail.name)")
	rootCmd.MarkFlagsMutuallyExclusive("field", "fields")
}

// Execute runs the root command and returns an exit code.
func Execute() int {
	registerDynamicCommands()

	err := rootCmd.Execute()

	waitForPendingUpdates(5 * time.Second)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return api.ExitCodeFromError(err)
	}
	return api.ExitSuccess
}

// registerDynamicCommands creates Cobra commands for each API resource topic.
// Uses DefaultTopics for command structure (since they contain granular CRUD entries),
// and per-command spec cache for endpoint metadata.
func registerDynamicCommands() {
	cfg, err := auth.LoadConfig()
	if err != nil {
		return
	}

	profile := cfg.ActiveProfile
	resolver := &auth.Resolver{Config: cfg}
	docsURL := resolver.ResolveDocsURL(flagDocsURL)
	if docsURL == "" {
		docsURL = docs.DefaultBaseURL
	}

	deps := &cmdgen.Deps{
		NewClient:        NewClient,
		RequireWorkspace: RequireWorkspace,
		RequireProject:   RequireProject,
		PaginationParams: PaginationParams,
		Formatter:        Formatter,
		IsUUID:           IsUUID,
		FlagAll:          &flagAll,
		FlagPerPage:      &flagPerPage,
		FlagDryRun:       &flagDryRun,
		FlagField:        &flagField,
		FlagFields:       &flagFields,
		Profile:          profile,
		BaseURL:          docsURL,
	}

	// Use DefaultTopics which have granular CRUD entries per resource.
	// The remote llms.txt only has overview-level entries, not individual endpoints.
	for _, topic := range docs.DefaultTopics {
		if !cmdgen.FilteredTopicName(topic.Name) {
			continue
		}
		if !cmdgen.TopicHasExecutableEntries(&topic) {
			continue
		}

		// Load cached specs for this topic
		cachedSpecs, _ := docs.LoadTopicSpecs(profile, topic.Name)

		topicCmd := cmdgen.BuildTopicCommand(topic.Name, &topic, cachedSpecs, deps)
		rootCmd.AddCommand(topicCmd)
	}
}

// waitForPendingUpdates gives background spec refresh goroutines time to finish.
func waitForPendingUpdates(timeout time.Duration) {
	timer := time.After(timeout)
	done := make(chan struct{})
	go func() {
		cmdgen.PendingUpdates.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-timer:
	}
}

// NewClient creates an API client from resolved flags, env, and config.
// Exported for use by cmdgen package.
func NewClient() (*api.Client, error) {
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

// RequireWorkspace returns the resolved workspace, or an error.
// Exported for use by cmdgen package.
func RequireWorkspace(client *api.Client) error {
	if client.Workspace == "" {
		return fmt.Errorf("workspace is required. Use --workspace flag, set %s, or configure via 'plane auth login'", auth.EnvWorkspace)
	}
	return nil
}

// RequireProject returns the project UUID. If --project is a short identifier
// (e.g. "PLANECLI"), it resolves it to a UUID via the API.
// Exported for use by cmdgen package.
func RequireProject() (string, error) {
	if flagProject == "" {
		return "", fmt.Errorf("project is required. Use --project flag")
	}
	if len(flagProject) == 36 && strings.Count(flagProject, "-") == 4 {
		return flagProject, nil
	}
	return resolveProjectIdentifier(flagProject)
}

// resolvedProjects caches identifier→UUID lookups within a session.
var resolvedProjects = map[string]string{}

func resolveProjectIdentifier(identifier string) (string, error) {
	upper := strings.ToUpper(identifier)
	if id, ok := resolvedProjects[upper]; ok {
		return id, nil
	}

	client, err := NewClient()
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

// PaginationParams returns pagination params from flags.
// Exported for use by cmdgen package.
func PaginationParams() api.PaginationParams {
	perPage := flagPerPage
	if perPage < 1 {
		perPage = 1
	} else if perPage > 100 {
		perPage = 100
	}
	return api.PaginationParams{
		PerPage: perPage,
		Cursor:  flagCursor,
	}
}

// Formatter returns the output formatter based on the --output flag.
// When no explicit format is given, defaults to table for TTY and json otherwise.
// Exported for use by cmdgen package.
func Formatter() output.Formatter {
	format := flagOutput
	if format == "" {
		if term.IsTerminal(int(os.Stdout.Fd())) {
			format = "table"
		} else {
			format = "json"
		}
	}
	return output.New(format)
}

// NewSessionClient creates an API client using session cookie auth.
// Returns nil if no session cookie is stored.
func NewSessionClient() *api.Client {
	cfg, err := auth.LoadConfig()
	if err != nil {
		return nil
	}

	store, err := auth.NewKeyringStore("")
	if err != nil {
		return nil
	}

	profile := cfg.ActiveProfile
	sessionCookie, err := store.Get(profile + "/session-token")
	if err != nil || sessionCookie == "" {
		return nil
	}

	resolver := &auth.Resolver{Config: cfg}
	apiURL := resolver.ResolveAPIURL(flagAPIURL)
	if apiURL == "" {
		return nil
	}

	workspace := resolver.ResolveWorkspace(flagWorkspace)

	var debugWriter *os.File
	if flagVerbose {
		debugWriter = os.Stderr
	}

	return api.NewSessionClient(apiURL, sessionCookie, workspace, flagVerbose, debugWriter)
}

// IsUUID checks if a string looks like a UUID (contains dashes and is ~36 chars).
// Exported for use by cmdgen package.
func IsUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		}
	}
	return true
}
