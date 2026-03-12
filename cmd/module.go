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
	rootCmd.AddCommand(moduleCmd)
	moduleCmd.AddCommand(moduleListCmd)
	moduleCmd.AddCommand(moduleGetCmd)
	moduleCmd.AddCommand(moduleCreateCmd)
	moduleCmd.AddCommand(moduleUpdateCmd)
	moduleCmd.AddCommand(moduleDeleteCmd)
	moduleCmd.AddCommand(moduleArchiveCmd)
	moduleCmd.AddCommand(moduleUnarchiveCmd)
	moduleCmd.AddCommand(moduleAddIssuesCmd)
	moduleCmd.AddCommand(moduleListIssuesCmd)
	moduleCmd.AddCommand(moduleRemoveIssueCmd)

	moduleGetCmd.Flags().String("module", "", "Module ID (required)")
	moduleGetCmd.MarkFlagRequired("module")

	moduleCreateCmd.Flags().String("name", "", "Module name (required)")
	moduleCreateCmd.Flags().String("description", "", "Module description")
	moduleCreateCmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	moduleCreateCmd.Flags().String("target-date", "", "Target date (YYYY-MM-DD)")
	moduleCreateCmd.Flags().String("status", "", "Module status")
	moduleCreateCmd.Flags().String("lead-id", "", "Lead user ID")
	moduleCreateCmd.Flags().StringSlice("member-ids", nil, "Member IDs")
	moduleCreateCmd.MarkFlagRequired("name")

	moduleUpdateCmd.Flags().String("module", "", "Module ID (required)")
	moduleUpdateCmd.Flags().String("name", "", "Module name")
	moduleUpdateCmd.Flags().String("description", "", "Module description")
	moduleUpdateCmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	moduleUpdateCmd.Flags().String("target-date", "", "Target date (YYYY-MM-DD)")
	moduleUpdateCmd.Flags().String("status", "", "Module status")
	moduleUpdateCmd.Flags().String("lead-id", "", "Lead user ID")
	moduleUpdateCmd.Flags().StringSlice("member-ids", nil, "Member IDs")
	moduleUpdateCmd.MarkFlagRequired("module")

	moduleDeleteCmd.Flags().String("module", "", "Module ID (required)")
	moduleDeleteCmd.MarkFlagRequired("module")

	moduleArchiveCmd.Flags().String("module", "", "Module ID (required)")
	moduleArchiveCmd.MarkFlagRequired("module")

	moduleUnarchiveCmd.Flags().String("module", "", "Module ID (required)")
	moduleUnarchiveCmd.MarkFlagRequired("module")

	moduleAddIssuesCmd.Flags().String("module", "", "Module ID (required)")
	moduleAddIssuesCmd.Flags().StringSlice("issue-ids", nil, "Issue IDs to add (required)")
	moduleAddIssuesCmd.MarkFlagRequired("module")
	moduleAddIssuesCmd.MarkFlagRequired("issue-ids")

	moduleListIssuesCmd.Flags().String("module", "", "Module ID (required)")
	moduleListIssuesCmd.MarkFlagRequired("module")

	moduleRemoveIssueCmd.Flags().String("module", "", "Module ID (required)")
	moduleRemoveIssueCmd.Flags().String("issue", "", "Issue ID to remove (required)")
	moduleRemoveIssueCmd.MarkFlagRequired("module")
	moduleRemoveIssueCmd.MarkFlagRequired("issue")
}

var moduleCmd = &cobra.Command{
	Use:     "module",
	Aliases: []string{"mod"},
	Short:   "Manage modules",
	Long: `Manage modules in a project.

API docs: https://developers.plane.so/api-reference/module/overview
CLI docs: plane docs module`,
}

var moduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List modules in a project",
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

		svc := api.NewModulesService(client)
		modules, err := svc.List(context.Background(), projectID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.ModuleList{Results: modules})
	},
}

var moduleGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a module by ID",
	Example: `  plane module get --module <module-id> -p PLANECLI`,
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

		moduleID, _ := cmd.Flags().GetString("module")
		svc := api.NewModulesService(client)
		mod, err := svc.Get(context.Background(), projectID, moduleID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, mod)
	},
}

var moduleCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new module",
	Example: `  plane module create -p PLANECLI --name "Foundation" --description "Core infrastructure"`,
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
		startDate, _ := cmd.Flags().GetString("start-date")
		targetDate, _ := cmd.Flags().GetString("target-date")
		status, _ := cmd.Flags().GetString("status")
		leadID, _ := cmd.Flags().GetString("lead-id")
		memberIDs, _ := cmd.Flags().GetStringSlice("member-ids")

		input := models.ModuleCreate{
			Name:        name,
			Description: description,
			StartDate:   startDate,
			TargetDate:  targetDate,
			Status:      status,
			LeadID:      leadID,
			MemberIDs:   memberIDs,
			ProjectID:   projectID,
		}

		svc := api.NewModulesService(client)
		mod, err := svc.Create(context.Background(), projectID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, mod)
	},
}

var moduleUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a module",
	Example: `  plane module update --module <module-id> -p PLANECLI --name "Foundation v2"`,
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

		moduleID, _ := cmd.Flags().GetString("module")
		input := models.ModuleUpdate{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			input.Name = &v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
		}
		if cmd.Flags().Changed("start-date") {
			v, _ := cmd.Flags().GetString("start-date")
			input.StartDate = &v
		}
		if cmd.Flags().Changed("target-date") {
			v, _ := cmd.Flags().GetString("target-date")
			input.TargetDate = &v
		}
		if cmd.Flags().Changed("status") {
			v, _ := cmd.Flags().GetString("status")
			input.Status = &v
		}
		if cmd.Flags().Changed("lead-id") {
			v, _ := cmd.Flags().GetString("lead-id")
			input.LeadID = &v
		}
		if cmd.Flags().Changed("member-ids") {
			v, _ := cmd.Flags().GetStringSlice("member-ids")
			input.MemberIDs = v
		}

		svc := api.NewModulesService(client)
		mod, err := svc.Update(context.Background(), projectID, moduleID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, mod)
	},
}

var moduleDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a module",
	Example: `  plane module delete --module <module-id> -p PLANECLI`,
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

		moduleID, _ := cmd.Flags().GetString("module")
		svc := api.NewModulesService(client)
		if err := svc.Delete(context.Background(), projectID, moduleID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Module deleted.")
		return nil
	},
}

var moduleArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive a module",
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

		moduleID, _ := cmd.Flags().GetString("module")
		svc := api.NewModulesService(client)
		return svc.Archive(context.Background(), projectID, moduleID)
	},
}

var moduleUnarchiveCmd = &cobra.Command{
	Use:   "unarchive",
	Short: "Unarchive a module",
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

		moduleID, _ := cmd.Flags().GetString("module")
		svc := api.NewModulesService(client)
		return svc.Unarchive(context.Background(), projectID, moduleID)
	},
}

var moduleAddIssuesCmd = &cobra.Command{
	Use:     "add-issues",
	Short:   "Add issues to a module",
	Example: `  plane module add-issues --module <module-id> --issue-ids id1,id2 -p PLANECLI`,
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

		moduleID, _ := cmd.Flags().GetString("module")
		issueIDs, _ := cmd.Flags().GetStringSlice("issue-ids")
		svc := api.NewModulesService(client)
		return svc.AddIssues(context.Background(), projectID, moduleID, issueIDs)
	},
}

var moduleListIssuesCmd = &cobra.Command{
	Use:     "list-issues",
	Short:   "List issues in a module",
	Example: `  plane module list-issues --module <module-id> -p PLANECLI`,
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

		moduleID, _ := cmd.Flags().GetString("module")
		svc := api.NewModulesService(client)
		resp, err := svc.ListIssues(context.Background(), projectID, moduleID, paginationParams())
		if err != nil {
			return err
		}

		out := models.IssueList{Results: resp.Results, TotalCount: resp.TotalCount}
		return formatter().Format(os.Stdout, out)
	},
}

var moduleRemoveIssueCmd = &cobra.Command{
	Use:     "remove-issue",
	Short:   "Remove an issue from a module",
	Example: `  plane module remove-issue --module <module-id> --issue <issue-id> -p PLANECLI`,
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

		moduleID, _ := cmd.Flags().GetString("module")
		issueID, _ := cmd.Flags().GetString("issue")
		svc := api.NewModulesService(client)
		return svc.RemoveIssue(context.Background(), projectID, moduleID, issueID)
	},
}
