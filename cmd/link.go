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
	rootCmd.AddCommand(linkCmd)
	linkCmd.AddCommand(linkListCmd)
	linkCmd.AddCommand(linkGetCmd)
	linkCmd.AddCommand(linkCreateCmd)
	linkCmd.AddCommand(linkUpdateCmd)
	linkCmd.AddCommand(linkDeleteCmd)

	linkCmd.PersistentFlags().String("issue", "", "Issue ID (required)")
	linkCmd.MarkPersistentFlagRequired("issue")

	linkGetCmd.Flags().String("link", "", "Link ID (required)")
	linkGetCmd.MarkFlagRequired("link")

	linkCreateCmd.Flags().String("url", "", "Link URL (required)")
	linkCreateCmd.Flags().String("title", "", "Link title")
	linkCreateCmd.MarkFlagRequired("url")

	linkUpdateCmd.Flags().String("link", "", "Link ID (required)")
	linkUpdateCmd.Flags().String("url", "", "Link URL")
	linkUpdateCmd.Flags().String("title", "", "Link title")
	linkUpdateCmd.MarkFlagRequired("link")

	linkDeleteCmd.Flags().String("link", "", "Link ID (required)")
	linkDeleteCmd.MarkFlagRequired("link")
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Manage issue links",
	Long: `Manage links on issues.

API docs: https://developers.plane.so/api-reference/link/overview
CLI docs: plane docs link`,
}

var linkListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List links on an issue",
	Example: `  plane link list --issue <issue-id> -p PLANECLI`,
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

		svc := api.NewLinksService(client)
		links, err := svc.List(context.Background(), projectID, issueID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.LinkList{Results: links})
	},
}

var linkGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a link by ID",
	Example: `  plane link get --link <link-id> --issue <issue-id> -p PLANECLI`,
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
		linkID, _ := cmd.Flags().GetString("link")

		svc := api.NewLinksService(client)
		link, err := svc.Get(context.Background(), projectID, issueID, linkID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, link)
	},
}

var linkCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a link on an issue",
	Example: `  plane link create --issue <issue-id> --url "https://example.com" --title "Docs" -p PLANECLI`,
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
		linkURL, _ := cmd.Flags().GetString("url")
		title, _ := cmd.Flags().GetString("title")

		input := models.LinkCreate{URL: linkURL, Title: title}

		svc := api.NewLinksService(client)
		link, err := svc.Create(context.Background(), projectID, issueID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, link)
	},
}

var linkUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a link",
	Example: `  plane link update --link <link-id> --issue <issue-id> --title "Updated" -p PLANECLI`,
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
		linkID, _ := cmd.Flags().GetString("link")

		input := models.LinkUpdate{}
		if cmd.Flags().Changed("url") {
			v, _ := cmd.Flags().GetString("url")
			input.URL = &v
		}
		if cmd.Flags().Changed("title") {
			v, _ := cmd.Flags().GetString("title")
			input.Title = &v
		}

		svc := api.NewLinksService(client)
		link, err := svc.Update(context.Background(), projectID, issueID, linkID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, link)
	},
}

var linkDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a link",
	Example: `  plane link delete --link <link-id> --issue <issue-id> -p PLANECLI`,
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
		linkID, _ := cmd.Flags().GetString("link")

		svc := api.NewLinksService(client)
		if err := svc.Delete(context.Background(), projectID, issueID, linkID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Link deleted.")
		return nil
	},
}
