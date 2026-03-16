package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	selfupdate2 "github.com/mggarofalo/plane-cli/internal/selfupdate"
	"github.com/spf13/cobra"
)

var flagCheckOnly bool

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&flagCheckOnly, "check", false, "Check for updates without installing")
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update plane to the latest version",
	Long: `Check for and install the latest version of plane.

Downloads the latest release from GitHub, verifies the checksum, and replaces
the current binary in-place. Use --check to only check without installing.

Development builds (version "dev") skip the update check.`,
	Example: `  plane update          # Update to the latest version
  plane update --check  # Check without updating`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if version == "dev" {
			fmt.Fprintln(os.Stderr, "Skipping update: development build (version \"dev\").")
			return nil
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
		defer cancel()

		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Current version: %s\n", version)
			fmt.Fprintln(os.Stderr, "Checking for updates...")
		}

		result, err := selfupdate2.CheckForUpdate(ctx, version)
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}

		if !result.NewVersionAvailable {
			fmt.Fprintf(os.Stderr, "Already up to date (v%s).\n", version)
			return nil
		}

		fmt.Fprintf(os.Stderr, "New version available: v%s\n", result.LatestVersion)

		if flagCheckOnly {
			fmt.Fprintf(os.Stderr, "Run \"plane update\" to install.\n")
			return nil
		}

		if !flagQuiet {
			fmt.Fprintln(os.Stderr, "Downloading and installing...")
		}

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locating current executable: %w", err)
		}

		if err := selfupdate2.DownloadAndApply(ctx, result.Release, exe); err != nil {
			return fmt.Errorf("updating: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Updated to v%s.\n", result.LatestVersion)
		return nil
	},
}
