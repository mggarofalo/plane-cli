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
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("plane %s (commit: %s, built: %s)\n", version, commit, date)
	},
}
