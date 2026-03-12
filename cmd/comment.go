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
	rootCmd.AddCommand(commentCmd)
	commentCmd.AddCommand(commentListCmd)
	commentCmd.AddCommand(commentGetCmd)
	commentCmd.AddCommand(commentCreateCmd)
	commentCmd.AddCommand(commentUpdateCmd)
	commentCmd.AddCommand(commentDeleteCmd)

	commentCmd.PersistentFlags().String("issue", "", "Issue ID (required)")
	commentCmd.MarkPersistentFlagRequired("issue")

	commentGetCmd.Flags().String("comment", "", "Comment ID (required)")
	commentGetCmd.MarkFlagRequired("comment")

	commentCreateCmd.Flags().String("body", "", "Comment body as HTML (required)")
	commentCreateCmd.MarkFlagRequired("body")

	commentUpdateCmd.Flags().String("comment", "", "Comment ID (required)")
	commentUpdateCmd.Flags().String("body", "", "Updated comment body as HTML")
	commentUpdateCmd.MarkFlagRequired("comment")

	commentDeleteCmd.Flags().String("comment", "", "Comment ID (required)")
	commentDeleteCmd.MarkFlagRequired("comment")
}

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage issue comments",
	Long: `Manage comments on issues.

API docs: https://developers.plane.so/api-reference/issue-comment/overview
CLI docs: plane docs comment`,
}

var commentListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List comments on an issue",
	Example: `  plane comment list --issue <issue-id> -p PLANECLI`,
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

		svc := api.NewCommentsService(client)
		comments, err := svc.List(context.Background(), projectID, issueID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.CommentList{Results: comments})
	},
}

var commentGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a comment by ID",
	Example: `  plane comment get --comment <id> --issue <issue-id> -p PLANECLI`,
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
		commentID, _ := cmd.Flags().GetString("comment")

		svc := api.NewCommentsService(client)
		comment, err := svc.Get(context.Background(), projectID, issueID, commentID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, comment)
	},
}

var commentCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a comment on an issue",
	Example: `  plane comment create --issue <issue-id> --body "<p>Looks good</p>" -p PLANECLI`,
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
		body, _ := cmd.Flags().GetString("body")

		input := models.CommentCreate{CommentHTML: body}

		svc := api.NewCommentsService(client)
		comment, err := svc.Create(context.Background(), projectID, issueID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, comment)
	},
}

var commentUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a comment",
	Example: `  plane comment update --comment <id> --issue <issue-id> --body "<p>Updated</p>" -p PLANECLI`,
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
		commentID, _ := cmd.Flags().GetString("comment")

		input := models.CommentUpdate{}
		if cmd.Flags().Changed("body") {
			v, _ := cmd.Flags().GetString("body")
			input.CommentHTML = &v
		}

		svc := api.NewCommentsService(client)
		comment, err := svc.Update(context.Background(), projectID, issueID, commentID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, comment)
	},
}

var commentDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a comment",
	Example: `  plane comment delete --comment <id> --issue <issue-id> -p PLANECLI`,
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
		commentID, _ := cmd.Flags().GetString("comment")

		svc := api.NewCommentsService(client)
		if err := svc.Delete(context.Background(), projectID, issueID, commentID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Comment deleted.")
		return nil
	},
}
