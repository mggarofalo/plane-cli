package cmd

import (
	"context"
	"os"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(meCmd)
}

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show the authenticated user",
	Example: `  plane me
  plane me --output table`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := NewClient()
		if err != nil {
			return err
		}

		users := api.NewUsersService(client)
		user, err := users.Me(context.Background())
		if err != nil {
			return err
		}

		return Formatter().Format(os.Stdout, user)
	},
}
