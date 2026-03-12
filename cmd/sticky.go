package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/models"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stickyCmd)
	stickyCmd.AddCommand(stickyListCmd)
	stickyCmd.AddCommand(stickyGetCmd)
	stickyCmd.AddCommand(stickyCreateCmd)
	stickyCmd.AddCommand(stickyUpdateCmd)
	stickyCmd.AddCommand(stickyDeleteCmd)

	stickyGetCmd.Flags().String("sticky", "", "Sticky ID (required)")
	stickyGetCmd.MarkFlagRequired("sticky")

	stickyCreateCmd.Flags().String("name", "", "Sticky name (required)")
	stickyCreateCmd.Flags().String("description", "", "Sticky description")
	stickyCreateCmd.Flags().String("color", "", "Sticky color")
	stickyCreateCmd.MarkFlagRequired("name")

	stickyUpdateCmd.Flags().String("sticky", "", "Sticky ID (required)")
	stickyUpdateCmd.Flags().String("name", "", "Sticky name")
	stickyUpdateCmd.Flags().String("description", "", "Sticky description")
	stickyUpdateCmd.Flags().String("color", "", "Sticky color")
	stickyUpdateCmd.MarkFlagRequired("sticky")

	stickyDeleteCmd.Flags().String("sticky", "", "Sticky ID (required)")
	stickyDeleteCmd.MarkFlagRequired("sticky")
}

var stickyCmd = &cobra.Command{
	Use:   "sticky",
	Short: "Manage stickies (notes)",
	Long: `Manage stickies (quick notes).

API docs: https://developers.plane.so/api-reference/sticky/overview
CLI docs: plane docs sticky`,
}

var stickyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stickies",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}

		svc := api.NewStickiesService(client)
		resp, err := svc.List(context.Background(), paginationParams())
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.StickyList{Results: resp.Results})
	},
}

var stickyGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a sticky by ID",
	Example: `  plane sticky get --sticky <sticky-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}

		stickyID, _ := cmd.Flags().GetString("sticky")
		svc := api.NewStickiesService(client)
		sticky, err := svc.Get(context.Background(), stickyID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, sticky)
	},
}

var stickyCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new sticky",
	Example: `  plane sticky create --name "Remember this" --color "#FFFF00"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		color, _ := cmd.Flags().GetString("color")

		input := models.StickyCreate{Name: name, Description: description, Color: color}

		svc := api.NewStickiesService(client)
		sticky, err := svc.Create(context.Background(), input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, sticky)
	},
}

var stickyUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a sticky",
	Example: `  plane sticky update --sticky <sticky-id> --name "Updated note"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}

		stickyID, _ := cmd.Flags().GetString("sticky")
		input := models.StickyUpdate{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			input.Name = &v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
		}
		if cmd.Flags().Changed("color") {
			v, _ := cmd.Flags().GetString("color")
			input.Color = &v
		}

		svc := api.NewStickiesService(client)
		sticky, err := svc.Update(context.Background(), stickyID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, sticky)
	},
}

var stickyDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a sticky",
	Example: `  plane sticky delete --sticky <sticky-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}

		stickyID, _ := cmd.Flags().GetString("sticky")
		svc := api.NewStickiesService(client)
		if err := svc.Delete(context.Background(), stickyID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Sticky deleted.")
		return nil
	},
}
