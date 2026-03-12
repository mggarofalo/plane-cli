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
	rootCmd.AddCommand(projectCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectGetCmd)
	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectUpdateCmd)
	projectCmd.AddCommand(projectDeleteCmd)
	projectCmd.AddCommand(projectArchiveCmd)
	projectCmd.AddCommand(projectUnarchiveCmd)

	// Get uses the global --project flag

	// Create flags
	projectCreateCmd.Flags().String("name", "", "Project name (required)")
	projectCreateCmd.Flags().String("identifier", "", "Project identifier (e.g., PROJ)")
	projectCreateCmd.Flags().String("description", "", "Project description")
	projectCreateCmd.Flags().Int("network", 2, "Network visibility (0=secret, 2=public)")
	projectCreateCmd.MarkFlagRequired("name")

	// Update flags
	projectUpdateCmd.Flags().String("name", "", "Project name")
	projectUpdateCmd.Flags().String("description", "", "Project description")
	projectUpdateCmd.Flags().Int("network", -1, "Network visibility")
}

var projectCmd = &cobra.Command{
	Use:     "project",
	Aliases: []string{"proj"},
	Short:   "Manage projects",
	Long: `Manage projects in the workspace.

API docs: https://developers.plane.so/api-reference/project/overview
CLI docs: plane docs project`,
}

var projectListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List projects in the workspace",
	Example: `  plane project list -w dev
  plane project list -w dev --output table`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}

		svc := api.NewProjectsService(client)
		resp, err := svc.List(context.Background(), paginationParams())
		if err != nil {
			return err
		}

		out := models.ProjectList{Results: resp.Results, TotalCount: resp.TotalCount}
		return formatter().Format(os.Stdout, out)
	},
}

var projectGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a project by ID",
	Example: `  plane project get -p 014260e0-...
  plane project get -p PLANECLI`,
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

		svc := api.NewProjectsService(client)
		project, err := svc.Get(context.Background(), projectID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, project)
	},
}

var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new project",
	Example: `  plane project create -w dev --name "My Project" --identifier MP
  plane project create -w dev --name "Backend" --description "Backend services"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		identifier, _ := cmd.Flags().GetString("identifier")
		description, _ := cmd.Flags().GetString("description")
		network, _ := cmd.Flags().GetInt("network")

		input := models.ProjectCreate{
			Name:        name,
			Identifier:  identifier,
			Description: description,
			Network:     &network,
		}

		svc := api.NewProjectsService(client)
		project, err := svc.Create(context.Background(), input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, project)
	},
}

var projectUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a project",
	Example: `  plane project update -p PLANECLI --name "New Name"`,
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

		input := models.ProjectUpdate{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			input.Name = &v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
		}
		if cmd.Flags().Changed("network") {
			v, _ := cmd.Flags().GetInt("network")
			input.Network = &v
		}

		svc := api.NewProjectsService(client)
		project, err := svc.Update(context.Background(), projectID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, project)
	},
}

var projectDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a project",
	Example: `  plane project delete -p <project-id>`,
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

		svc := api.NewProjectsService(client)
		if err := svc.Delete(context.Background(), projectID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Project deleted.")
		return nil
	},
}

var projectArchiveCmd = &cobra.Command{
	Use:     "archive",
	Short:   "Archive a project",
	Example: `  plane project archive -p <project-id>`,
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

		svc := api.NewProjectsService(client)
		if err := svc.Archive(context.Background(), projectID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Project archived.")
		return nil
	},
}

var projectUnarchiveCmd = &cobra.Command{
	Use:     "unarchive",
	Short:   "Unarchive a project",
	Example: `  plane project unarchive -p <project-id>`,
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

		svc := api.NewProjectsService(client)
		if err := svc.Unarchive(context.Background(), projectID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Project unarchived.")
		return nil
	},
}
