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
	rootCmd.AddCommand(cycleCmd)
	cycleCmd.AddCommand(cycleListCmd)
	cycleCmd.AddCommand(cycleGetCmd)
	cycleCmd.AddCommand(cycleCreateCmd)
	cycleCmd.AddCommand(cycleUpdateCmd)
	cycleCmd.AddCommand(cycleDeleteCmd)
	cycleCmd.AddCommand(cycleArchiveCmd)
	cycleCmd.AddCommand(cycleUnarchiveCmd)
	cycleCmd.AddCommand(cycleAddIssuesCmd)
	cycleCmd.AddCommand(cycleListIssuesCmd)
	cycleCmd.AddCommand(cycleRemoveIssueCmd)
	cycleCmd.AddCommand(cycleTransferCmd)

	cycleGetCmd.Flags().String("cycle", "", "Cycle ID (required)")
	cycleGetCmd.MarkFlagRequired("cycle")

	cycleCreateCmd.Flags().String("name", "", "Cycle name (required)")
	cycleCreateCmd.Flags().String("description", "", "Cycle description")
	cycleCreateCmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	cycleCreateCmd.Flags().String("end-date", "", "End date (YYYY-MM-DD)")
	cycleCreateCmd.MarkFlagRequired("name")

	cycleUpdateCmd.Flags().String("cycle", "", "Cycle ID (required)")
	cycleUpdateCmd.Flags().String("name", "", "Cycle name")
	cycleUpdateCmd.Flags().String("description", "", "Cycle description")
	cycleUpdateCmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	cycleUpdateCmd.Flags().String("end-date", "", "End date (YYYY-MM-DD)")
	cycleUpdateCmd.MarkFlagRequired("cycle")

	cycleDeleteCmd.Flags().String("cycle", "", "Cycle ID (required)")
	cycleDeleteCmd.MarkFlagRequired("cycle")

	cycleArchiveCmd.Flags().String("cycle", "", "Cycle ID (required)")
	cycleArchiveCmd.MarkFlagRequired("cycle")

	cycleUnarchiveCmd.Flags().String("cycle", "", "Cycle ID (required)")
	cycleUnarchiveCmd.MarkFlagRequired("cycle")

	cycleAddIssuesCmd.Flags().String("cycle", "", "Cycle ID (required)")
	cycleAddIssuesCmd.Flags().StringSlice("issue-ids", nil, "Issue IDs to add (required)")
	cycleAddIssuesCmd.MarkFlagRequired("cycle")
	cycleAddIssuesCmd.MarkFlagRequired("issue-ids")

	cycleListIssuesCmd.Flags().String("cycle", "", "Cycle ID (required)")
	cycleListIssuesCmd.MarkFlagRequired("cycle")

	cycleRemoveIssueCmd.Flags().String("cycle", "", "Cycle ID (required)")
	cycleRemoveIssueCmd.Flags().String("issue", "", "Issue ID to remove (required)")
	cycleRemoveIssueCmd.MarkFlagRequired("cycle")
	cycleRemoveIssueCmd.MarkFlagRequired("issue")

	cycleTransferCmd.Flags().String("cycle", "", "Source cycle ID (required)")
	cycleTransferCmd.Flags().String("to", "", "Target cycle ID (required)")
	cycleTransferCmd.MarkFlagRequired("cycle")
	cycleTransferCmd.MarkFlagRequired("to")
}

var cycleCmd = &cobra.Command{
	Use:   "cycle",
	Short: "Manage cycles (sprints)",
	Long: `Manage cycles (sprints) in a project.

API docs: https://developers.plane.so/api-reference/cycle/overview
CLI docs: plane docs cycle`,
}

var cycleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cycles in a project",
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

		svc := api.NewCyclesService(client)
		cycles, err := svc.List(context.Background(), projectID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.CycleList{Results: cycles})
	},
}

var cycleGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a cycle by ID",
	Example: `  plane cycle get --cycle <cycle-id> -p PLANECLI`,
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		svc := api.NewCyclesService(client)
		cycle, err := svc.Get(context.Background(), projectID, cycleID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, cycle)
	},
}

var cycleCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new cycle",
	Example: `  plane cycle create -p PLANECLI --name "Sprint 1" --start-date 2026-03-11 --end-date 2026-03-25`,
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
		endDate, _ := cmd.Flags().GetString("end-date")

		input := models.CycleCreate{
			Name:        name,
			Description: description,
			StartDate:   startDate,
			EndDate:     endDate,
			ProjectID:   projectID,
		}

		svc := api.NewCyclesService(client)
		cycle, err := svc.Create(context.Background(), projectID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, cycle)
	},
}

var cycleUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a cycle",
	Example: `  plane cycle update --cycle <cycle-id> -p PLANECLI --name "Sprint 1 (extended)"`,
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		input := models.CycleUpdate{}
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
		if cmd.Flags().Changed("end-date") {
			v, _ := cmd.Flags().GetString("end-date")
			input.EndDate = &v
		}

		svc := api.NewCyclesService(client)
		cycle, err := svc.Update(context.Background(), projectID, cycleID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, cycle)
	},
}

var cycleDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a cycle",
	Example: `  plane cycle delete --cycle <cycle-id> -p PLANECLI`,
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		svc := api.NewCyclesService(client)
		if err := svc.Delete(context.Background(), projectID, cycleID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Cycle deleted.")
		return nil
	},
}

var cycleArchiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive a cycle",
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		svc := api.NewCyclesService(client)
		return svc.Archive(context.Background(), projectID, cycleID)
	},
}

var cycleUnarchiveCmd = &cobra.Command{
	Use:   "unarchive",
	Short: "Unarchive a cycle",
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		svc := api.NewCyclesService(client)
		return svc.Unarchive(context.Background(), projectID, cycleID)
	},
}

var cycleAddIssuesCmd = &cobra.Command{
	Use:     "add-issues",
	Short:   "Add issues to a cycle",
	Example: `  plane cycle add-issues --cycle <cycle-id> --issue-ids id1,id2 -p PLANECLI`,
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		issueIDs, _ := cmd.Flags().GetStringSlice("issue-ids")
		svc := api.NewCyclesService(client)
		return svc.AddIssues(context.Background(), projectID, cycleID, issueIDs)
	},
}

var cycleListIssuesCmd = &cobra.Command{
	Use:     "list-issues",
	Short:   "List issues in a cycle",
	Example: `  plane cycle list-issues --cycle <cycle-id> -p PLANECLI`,
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		svc := api.NewCyclesService(client)
		resp, err := svc.ListIssues(context.Background(), projectID, cycleID, paginationParams())
		if err != nil {
			return err
		}

		out := models.IssueList{Results: resp.Results, TotalCount: resp.TotalCount}
		return formatter().Format(os.Stdout, out)
	},
}

var cycleRemoveIssueCmd = &cobra.Command{
	Use:     "remove-issue",
	Short:   "Remove an issue from a cycle",
	Example: `  plane cycle remove-issue --cycle <cycle-id> --issue <issue-id> -p PLANECLI`,
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		issueID, _ := cmd.Flags().GetString("issue")
		svc := api.NewCyclesService(client)
		return svc.RemoveIssue(context.Background(), projectID, cycleID, issueID)
	},
}

var cycleTransferCmd = &cobra.Command{
	Use:     "transfer",
	Short:   "Transfer incomplete issues to another cycle",
	Example: `  plane cycle transfer --cycle <source-id> --to <target-id> -p PLANECLI`,
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

		cycleID, _ := cmd.Flags().GetString("cycle")
		toCycle, _ := cmd.Flags().GetString("to")
		svc := api.NewCyclesService(client)
		return svc.Transfer(context.Background(), projectID, cycleID, toCycle)
	},
}
