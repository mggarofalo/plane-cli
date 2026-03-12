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
	rootCmd.AddCommand(intakeCmd)
	intakeCmd.AddCommand(intakeListCmd)
	intakeCmd.AddCommand(intakeGetCmd)
	intakeCmd.AddCommand(intakeCreateCmd)
	intakeCmd.AddCommand(intakeUpdateCmd)
	intakeCmd.AddCommand(intakeDeleteCmd)

	intakeGetCmd.Flags().String("intake", "", "Intake issue ID (required)")
	intakeGetCmd.MarkFlagRequired("intake")

	intakeCreateCmd.Flags().String("name", "", "Intake issue name (required)")
	intakeCreateCmd.Flags().String("description", "", "Description")
	intakeCreateCmd.Flags().String("source", "", "Source")
	intakeCreateCmd.MarkFlagRequired("name")

	intakeUpdateCmd.Flags().String("intake", "", "Intake issue ID (required)")
	intakeUpdateCmd.Flags().String("name", "", "Intake issue name")
	intakeUpdateCmd.Flags().String("description", "", "Description")
	intakeUpdateCmd.Flags().String("status", "", "Status: pending, accepted, declined, snoozed, duplicate")
	intakeUpdateCmd.MarkFlagRequired("intake")

	intakeDeleteCmd.Flags().String("intake", "", "Intake issue ID (required)")
	intakeDeleteCmd.MarkFlagRequired("intake")
}

var intakeCmd = &cobra.Command{
	Use:   "intake",
	Short: "Manage intake issues",
	Long: `Manage intake issues from external sources.

API docs: https://developers.plane.so/api-reference/intake-issue/overview
CLI docs: plane docs intake`,
}

var intakeListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List intake issues",
	Example: `  plane intake list -p PLANECLI`,
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

		svc := api.NewIntakeService(client)
		resp, err := svc.List(context.Background(), projectID, paginationParams())
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.IntakeIssueList{Results: resp.Results})
	},
}

var intakeGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get an intake issue by ID",
	Example: `  plane intake get --intake <id> -p PLANECLI`,
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

		intakeID, _ := cmd.Flags().GetString("intake")
		svc := api.NewIntakeService(client)
		issue, err := svc.Get(context.Background(), projectID, intakeID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, issue)
	},
}

var intakeCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create an intake issue",
	Example: `  plane intake create --name "Bug report from user" --source "email" -p PLANECLI`,
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
		source, _ := cmd.Flags().GetString("source")

		input := models.IntakeIssueCreate{Name: name, Description: description, Source: source}

		svc := api.NewIntakeService(client)
		issue, err := svc.Create(context.Background(), projectID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, issue)
	},
}

var intakeUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update an intake issue",
	Example: `  plane intake update --intake <id> --status accepted -p PLANECLI`,
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

		intakeID, _ := cmd.Flags().GetString("intake")
		input := models.IntakeIssueUpdate{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			input.Name = &v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
		}
		if cmd.Flags().Changed("status") {
			v, _ := cmd.Flags().GetString("status")
			input.Status = &v
		}

		svc := api.NewIntakeService(client)
		issue, err := svc.Update(context.Background(), projectID, intakeID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, issue)
	},
}

var intakeDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete an intake issue",
	Example: `  plane intake delete --intake <id> -p PLANECLI`,
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

		intakeID, _ := cmd.Flags().GetString("intake")
		svc := api.NewIntakeService(client)
		if err := svc.Delete(context.Background(), projectID, intakeID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Intake issue deleted.")
		return nil
	},
}
