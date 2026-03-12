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
	rootCmd.AddCommand(pageCmd)
	pageCmd.AddCommand(pageListCmd)
	pageCmd.AddCommand(pageGetCmd)
	pageCmd.AddCommand(pageCreateCmd)
	pageCmd.AddCommand(pageUpdateCmd)
	pageCmd.AddCommand(pageDeleteCmd)

	pageGetCmd.Flags().String("page", "", "Page ID (required)")
	pageGetCmd.MarkFlagRequired("page")

	pageCreateCmd.Flags().String("name", "", "Page name (required)")
	pageCreateCmd.Flags().String("description", "", "Page body as HTML")
	pageCreateCmd.Flags().Int("access", 0, "Access: 0=public, 1=private")
	pageCreateCmd.Flags().String("color", "", "Page color")
	pageCreateCmd.MarkFlagRequired("name")

	pageUpdateCmd.Flags().String("page", "", "Page ID (required)")
	pageUpdateCmd.Flags().String("name", "", "Page name")
	pageUpdateCmd.Flags().String("description", "", "Page body as HTML")
	pageUpdateCmd.Flags().Int("access", -1, "Access: 0=public, 1=private")
	pageUpdateCmd.Flags().String("color", "", "Page color")
	pageUpdateCmd.MarkFlagRequired("page")

	pageDeleteCmd.Flags().String("page", "", "Page ID (required)")
	pageDeleteCmd.MarkFlagRequired("page")
}

var pageCmd = &cobra.Command{
	Use:   "page",
	Short: "Manage pages (uses internal API)",
	Long: `Manage project pages. Note: pages use Plane's internal API and may require session authentication.

API docs: https://developers.plane.so/api-reference/page/overview
CLI docs: plane docs page`,
}

var pageListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List pages in a project",
	Example: `  plane page list -p PLANECLI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}
		projectID, err := requireProject()
		if err != nil {
			return err
		}

		svc := api.NewPagesService(client)
		pages, err := svc.List(context.Background(), projectID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.PageList{Results: pages})
	},
}

var pageGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a page by ID",
	Example: `  plane page get --page <page-id> -p PLANECLI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}
		projectID, err := requireProject()
		if err != nil {
			return err
		}

		pageID, _ := cmd.Flags().GetString("page")
		svc := api.NewPagesService(client)
		page, err := svc.Get(context.Background(), projectID, pageID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, page)
	},
}

var pageCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new page",
	Example: `  plane page create --name "Architecture" --description "<p>Design doc</p>" -p PLANECLI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}
		projectID, err := requireProject()
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		access, _ := cmd.Flags().GetInt("access")
		color, _ := cmd.Flags().GetString("color")

		input := models.PageCreate{
			Name:        name,
			Description: description,
			Access:      &access,
			Color:       color,
		}

		svc := api.NewPagesService(client)
		page, err := svc.Create(context.Background(), projectID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, page)
	},
}

var pageUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a page",
	Example: `  plane page update --page <page-id> --name "Updated Title" -p PLANECLI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}
		projectID, err := requireProject()
		if err != nil {
			return err
		}

		pageID, _ := cmd.Flags().GetString("page")
		input := models.PageUpdate{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			input.Name = &v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
		}
		if cmd.Flags().Changed("access") {
			v, _ := cmd.Flags().GetInt("access")
			input.Access = &v
		}
		if cmd.Flags().Changed("color") {
			v, _ := cmd.Flags().GetString("color")
			input.Color = &v
		}

		svc := api.NewPagesService(client)
		page, err := svc.Update(context.Background(), projectID, pageID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, page)
	},
}

var pageDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a page",
	Example: `  plane page delete --page <page-id> -p PLANECLI`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}
		projectID, err := requireProject()
		if err != nil {
			return err
		}

		pageID, _ := cmd.Flags().GetString("page")
		svc := api.NewPagesService(client)
		if err := svc.Delete(context.Background(), projectID, pageID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Page deleted.")
		return nil
	},
}
