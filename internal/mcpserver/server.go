package mcpserver

import (
	"context"
	"fmt"
	"os"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/auth"
	"github.com/mggarofalo/plane-cli/internal/cache"
	"github.com/mggarofalo/plane-cli/internal/cmdgen"
	"github.com/mggarofalo/plane-cli/internal/docs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config holds the server configuration, analogous to cmdgen.Deps but
// adapted for MCP server use (no Cobra dependency).
type Config struct {
	Workspace  string
	Project    string
	Profile    string
	BaseURL    string
	APIURL     string
	APIKey     string
	Quiet      bool
	cacheStore *cache.Store // singleton, shared across all BuildDepsFor calls
}

// NewClient creates an API client with the given workspace override.
// It resolves credentials from the same chain as the CLI: flag → env → keyring.
func (c *Config) NewClient(workspace string) (*api.Client, error) {
	cfg, err := auth.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	resolver := &auth.Resolver{
		FlagToken: c.APIKey,
		Env:       &auth.EnvSource{Quiet: c.Quiet},
		Config:    cfg,
	}

	store, err := auth.NewKeyringStore("")
	if err == nil {
		resolver.Store = store
	}

	resolved, err := resolver.Resolve()
	if err != nil {
		return nil, err
	}
	defer resolved.Credential.Clear()

	apiURL := resolver.ResolveAPIURL(c.APIURL)
	if apiURL == "" {
		return nil, fmt.Errorf("no API URL configured; run 'plane auth login' or set %s", auth.EnvURL)
	}

	if workspace == "" {
		workspace = resolver.ResolveWorkspace("")
	}

	return api.NewClient(apiURL, resolved.Credential.Token(), workspace, false, nil), nil
}

// BuildDepsFor creates a cmdgen.Deps struct for name resolution operations
// using the given workspace and project values (which may be per-call overrides).
// This allows the MCP executor to reuse the CLI's resolution logic with the
// correct context for each tool call.
func (c *Config) BuildDepsFor(workspace, project string) *cmdgen.Deps {
	return &cmdgen.Deps{
		NewClient: func() (*api.Client, error) {
			return c.NewClient(workspace)
		},
		RequireWorkspace: func(client *api.Client) error {
			if client.Workspace == "" {
				return fmt.Errorf("workspace is required")
			}
			return nil
		},
		RequireProject: func() (string, error) {
			if project == "" {
				return "", fmt.Errorf("project is required")
			}
			return project, nil
		},
		IsUUID:     func(s string) bool { return isUUID(s) },
		CacheStore: c.cacheStore,
	}
}

// Run creates the MCP server, registers tools from cached specs, and runs
// the stdio transport. It blocks until the client disconnects or the context
// is cancelled.
func Run(ctx context.Context, cfg *Config) error {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "plane-cli",
			Version: version(),
		},
		&mcp.ServerOptions{
			Instructions: "Plane project management API server. " +
				"Each tool corresponds to a Plane API endpoint. " +
				"Use workspace and project params to override server defaults per-call.",
		},
	)

	// Load auth config
	authCfg, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("loading auth config: %w", err)
	}

	profile := authCfg.ActiveProfile
	if cfg.Profile != "" {
		profile = cfg.Profile
	}
	cfg.Profile = profile
	cfg.cacheStore = cache.NewStore(profile)

	// Resolve workspace from config if not provided
	if cfg.Workspace == "" {
		resolver := &auth.Resolver{Config: authCfg}
		cfg.Workspace = resolver.ResolveWorkspace("")
	}

	// Resolve docs URL
	resolver := &auth.Resolver{Config: authCfg}
	docsURL := resolver.ResolveDocsURL("")
	if docsURL == "" {
		docsURL = docs.DefaultBaseURL
	}
	cfg.BaseURL = docsURL

	// Register tools from cached specs
	toolCount := 0
	for _, topic := range docs.DefaultTopics {
		if !cmdgen.FilteredTopicName(topic.Name) {
			continue
		}
		if !cmdgen.TopicHasExecutableEntries(&topic) {
			continue
		}

		cachedSpecs, err := docs.LoadTopicSpecs(profile, topic.Name)
		if err != nil {
			continue
		}

		for _, cached := range cachedSpecs {
			entry := BuildTool(topic.Name, &cached.Spec, cfg)
			if entry == nil {
				continue
			}
			server.AddTool(entry.Tool, entry.Handler)
			toolCount++
		}
	}

	if toolCount == 0 {
		return fmt.Errorf("no endpoint specs cached; run 'plane docs update-specs' first to populate the spec cache")
	}

	if !cfg.Quiet {
		fmt.Fprintf(os.Stderr, "plane MCP server: %d tools registered, listening on stdio\n", toolCount)
	}

	// Run on stdio transport
	return server.Run(ctx, &mcp.StdioTransport{})
}

// version returns the CLI version for the MCP server implementation field.
func version() string {
	// This is set at build time via ldflags in the CLI; we use a fallback here.
	return "dev"
}
