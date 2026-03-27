package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Set via ldflags at build time.
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func init() {
	// Enable --version flag on the root command (prints short version string).
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("plane {{.Version}}\n")

	// Keep the "version" subcommand for detailed output, but mark it deprecated.
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:        "version",
	Short:      "Print version information",
	Deprecated: "use --version instead",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("plane %s (commit: %s, built: %s)\n", version, commit, date)
	},
}
