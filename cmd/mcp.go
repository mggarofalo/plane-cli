package cmd

import (
	"github.com/mggarofalo/plane-cli/internal/mcpserver"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(mcpCmd)
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start a stdio MCP server exposing Plane API tools",
	Long: `Starts a Model Context Protocol (MCP) server on stdio (stdin/stdout JSON-RPC).

Each cached Plane API endpoint becomes an MCP tool that AI agents can discover
and call. Tool definitions are generated dynamically from the same endpoint spec
cache used by the CLI commands.

Prerequisites:
  plane docs update-specs    # populate the endpoint spec cache

The server resolves workspace and project from the same configuration as the CLI
(env vars, keyring, config file). Per-tool-call overrides are supported via
optional workspace/project parameters in each tool's input schema.

Usage with Claude Code:
  Add to .claude/settings.json:
  {
    "mcpServers": {
      "plane": {
        "command": "plane",
        "args": ["mcp", "--quiet"]
      }
    }
  }`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &mcpserver.Config{
			Workspace: flagWorkspace,
			Project:   flagProject,
			APIURL:    flagAPIURL,
			APIKey:    flagAPIKey,
			Quiet:     flagQuiet,
		}
		return mcpserver.Run(cmd.Context(), cfg)
	},
}
