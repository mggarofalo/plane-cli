package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/models"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueGetCmd)
	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueUpdateCmd)
	issueCmd.AddCommand(issueDeleteCmd)
	issueCmd.AddCommand(issueSearchCmd)

	// Get
	issueGetCmd.Flags().String("issue", "", "Issue ID or sequence identifier, e.g. PLANECLI-123 (required)")
	issueGetCmd.MarkFlagRequired("issue")

	// Create
	issueCreateCmd.Flags().String("name", "", "Issue name (required)")
	issueCreateCmd.Flags().String("description", "", "Issue description")
	issueCreateCmd.Flags().String("priority", "none", "Priority: urgent, high, medium, low, none")
	issueCreateCmd.Flags().String("state-id", "", "State ID (UUID)")
	issueCreateCmd.Flags().String("state", "", "State name (e.g. Backlog, Todo, \"In Progress\", Done)")
	issueCreateCmd.Flags().String("parent-id", "", "Parent issue ID")
	issueCreateCmd.Flags().StringSlice("assignees", nil, "Assignee IDs")
	issueCreateCmd.Flags().StringSlice("labels", nil, "Label IDs")
	issueCreateCmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	issueCreateCmd.Flags().String("target-date", "", "Target date (YYYY-MM-DD)")
	issueCreateCmd.MarkFlagRequired("name")

	// Update
	issueUpdateCmd.Flags().String("issue", "", "Issue ID (required)")
	issueUpdateCmd.Flags().String("name", "", "Issue name")
	issueUpdateCmd.Flags().String("description", "", "Issue description")
	issueUpdateCmd.Flags().String("priority", "", "Priority: urgent, high, medium, low, none")
	issueUpdateCmd.Flags().String("state-id", "", "State ID (UUID)")
	issueUpdateCmd.Flags().String("state", "", "State name (e.g. Backlog, Todo, \"In Progress\", Done)")
	issueUpdateCmd.Flags().String("parent-id", "", "Parent issue ID")
	issueUpdateCmd.Flags().StringSlice("assignees", nil, "Assignee IDs")
	issueUpdateCmd.Flags().StringSlice("labels", nil, "Label IDs")
	issueUpdateCmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	issueUpdateCmd.Flags().String("target-date", "", "Target date (YYYY-MM-DD)")
	issueUpdateCmd.MarkFlagRequired("issue")

	// Delete
	issueDeleteCmd.Flags().String("issue", "", "Issue ID (required)")
	issueDeleteCmd.MarkFlagRequired("issue")

	// Search
	issueSearchCmd.Flags().String("query", "", "Search query (required)")
	issueSearchCmd.MarkFlagRequired("query")
}

var issueCmd = &cobra.Command{
	Use:     "issue",
	Aliases: []string{"i"},
	Short:   "Manage issues (work items)",
	Long: `Manage issues (work items) in a project.

API docs: https://developers.plane.so/api-reference/issue/overview
CLI docs: plane docs issue`,
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues in a project",
	Example: `  plane issue list -p PLANECLI
  plane issue list -p PLANECLI --all
  plane issue list -p PLANECLI --output table`,
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

		svc := api.NewIssuesService(client)

		if flagAll {
			issues, err := svc.ListAll(context.Background(), projectID, flagPerPage)
			if err != nil {
				return err
			}
			out := models.IssueList{Results: issues, TotalCount: len(issues)}
			return formatter().Format(os.Stdout, out)
		}

		resp, err := svc.List(context.Background(), projectID, paginationParams())
		if err != nil {
			return err
		}

		out := models.IssueList{
			Results:         resp.Results,
			TotalCount:      resp.TotalCount,
			NextCursor:      resp.NextCursor,
			PrevCursor:      resp.PrevCursor,
			NextPageResults: resp.NextPageResults,
		}
		return formatter().Format(os.Stdout, out)
	},
}

var issueGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get an issue by ID or sequence identifier",
	Example: `  plane issue get --issue abc123-def456 -p PLANECLI
  plane issue get --issue PLANECLI-1 -p PLANECLI`,
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

		issueRef, _ := cmd.Flags().GetString("issue")
		svc := api.NewIssuesService(client)

		var issue *models.Issue
		if strings.Contains(issueRef, "-") && !isUUID(issueRef) {
			issue, err = svc.GetBySequence(context.Background(), projectID, issueRef)
		} else {
			issue, err = svc.Get(context.Background(), projectID, issueRef)
		}
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, issue)
	},
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Example: `  plane issue create -p PLANECLI --name "Fix login bug" --priority high
  plane issue create -p PLANECLI --name "Add feature" --state-id <id> --assignees user1,user2`,
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
		priority, _ := cmd.Flags().GetString("priority")
		stateID, _ := cmd.Flags().GetString("state-id")
		stateName, _ := cmd.Flags().GetString("state")
		parentID, _ := cmd.Flags().GetString("parent-id")
		assignees, _ := cmd.Flags().GetStringSlice("assignees")
		labels, _ := cmd.Flags().GetStringSlice("labels")
		startDate, _ := cmd.Flags().GetString("start-date")
		targetDate, _ := cmd.Flags().GetString("target-date")

		// Resolve --state name to ID if provided
		if stateName != "" && stateID == "" {
			resolved, err := resolveStateName(projectID, stateName)
			if err != nil {
				return err
			}
			stateID = resolved
		}

		input := models.IssueCreate{
			Name:        name,
			Description: description,
			Priority:    priority,
			StateID:     stateID,
			ParentID:    parentID,
			AssigneeIDs: assignees,
			LabelIDs:    labels,
			StartDate:   startDate,
			TargetDate:  targetDate,
		}

		svc := api.NewIssuesService(client)
		issue, err := svc.Create(context.Background(), projectID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, issue)
	},
}

var issueUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an issue",
	Example: `  plane issue update --issue <id> -p PLANECLI --priority high --state Done`,
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

		issueID, _ := cmd.Flags().GetString("issue")
		input := models.IssueUpdate{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			input.Name = &v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
		}
		if cmd.Flags().Changed("priority") {
			v, _ := cmd.Flags().GetString("priority")
			input.Priority = &v
		}
		if cmd.Flags().Changed("state") {
			stateName, _ := cmd.Flags().GetString("state")
			resolved, err := resolveStateName(projectID, stateName)
			if err != nil {
				return err
			}
			input.StateID = &resolved
		} else if cmd.Flags().Changed("state-id") {
			v, _ := cmd.Flags().GetString("state-id")
			input.StateID = &v
		}
		if cmd.Flags().Changed("parent-id") {
			v, _ := cmd.Flags().GetString("parent-id")
			input.ParentID = &v
		}
		if cmd.Flags().Changed("assignees") {
			v, _ := cmd.Flags().GetStringSlice("assignees")
			input.AssigneeIDs = v
		}
		if cmd.Flags().Changed("labels") {
			v, _ := cmd.Flags().GetStringSlice("labels")
			input.LabelIDs = v
		}
		if cmd.Flags().Changed("start-date") {
			v, _ := cmd.Flags().GetString("start-date")
			input.StartDate = &v
		}
		if cmd.Flags().Changed("target-date") {
			v, _ := cmd.Flags().GetString("target-date")
			input.TargetDate = &v
		}

		svc := api.NewIssuesService(client)
		issue, err := svc.Update(context.Background(), projectID, issueID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, issue)
	},
}

var issueDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete an issue",
	Example: `  plane issue delete --issue <id> -p PLANECLI`,
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

		issueID, _ := cmd.Flags().GetString("issue")
		svc := api.NewIssuesService(client)
		if err := svc.Delete(context.Background(), projectID, issueID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Issue deleted.")
		return nil
	},
}

var issueSearchCmd = &cobra.Command{
	Use:     "search",
	Short:   "Search for issues",
	Example: `  plane issue search -p PLANECLI --query "login bug"`,
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

		query, _ := cmd.Flags().GetString("query")
		svc := api.NewIssuesService(client)
		results, err := svc.Search(context.Background(), projectID, query)
		if err != nil {
			return err
		}

		out := models.IssueList{Results: results, TotalCount: len(results)}
		return formatter().Format(os.Stdout, out)
	},
}

// isUUID checks if a string looks like a UUID (contains dashes and is ~36 chars).
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		}
	}
	return true
}
