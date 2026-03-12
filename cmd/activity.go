package cmd

import (
	"context"
	"os"

	"github.com/mggarofalo/plane-cli/internal/api"
	"github.com/mggarofalo/plane-cli/internal/models"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(activityCmd)
	activityCmd.AddCommand(activityListCmd)
	activityCmd.AddCommand(activityGetCmd)

	activityCmd.PersistentFlags().String("issue", "", "Issue ID (required)")
	activityCmd.MarkPersistentFlagRequired("issue")

	activityGetCmd.Flags().String("activity", "", "Activity ID (required)")
	activityGetCmd.MarkFlagRequired("activity")
}

var activityCmd = &cobra.Command{
	Use:   "activity",
	Short: "View issue activities",
	Long: `View activities on issues.

API docs: https://developers.plane.so/api-reference/issue-activity/overview
CLI docs: plane docs activity`,
}

var activityListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List activities on an issue",
	Example: `  plane activity list --issue <issue-id> -p PLANECLI`,
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
		issueID, err := resolveIssueRef(projectID, issueRef)
		if err != nil {
			return err
		}

		svc := api.NewActivitiesService(client)
		activities, err := svc.List(context.Background(), projectID, issueID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.ActivityList{Results: activities})
	},
}

var activityGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get an activity by ID",
	Example: `  plane activity get --activity <id> --issue <issue-id> -p PLANECLI`,
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
		issueID, err := resolveIssueRef(projectID, issueRef)
		if err != nil {
			return err
		}
		activityID, _ := cmd.Flags().GetString("activity")

		svc := api.NewActivitiesService(client)
		activity, err := svc.Get(context.Background(), projectID, issueID, activityID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, activity)
	},
}
