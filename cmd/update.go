package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/creativeprojects/go-selfupdate"
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

		source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
		if err != nil {
			return fmt.Errorf("creating update source: %w", err)
		}

		updater, err := selfupdate.NewUpdater(selfupdate.Config{
			Source:    source,
			Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
		})
		if err != nil {
			return fmt.Errorf("creating updater: %w", err)
		}

		latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(selfupdate2.GitHubRepo))
		if err != nil {
			return fmt.Errorf("detecting latest release: %w", err)
		}
		if !found {
			return fmt.Errorf("no release found")
		}

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locating current executable: %w", err)
		}

		if err := updater.UpdateTo(ctx, latest, exe); err != nil {
			return fmt.Errorf("applying update: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Updated to v%s.\n", latest.Version())
		return nil
	},
}
