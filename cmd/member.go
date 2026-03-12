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
	rootCmd.AddCommand(memberCmd)
	memberCmd.AddCommand(memberListCmd)
	memberCmd.AddCommand(memberListProjectCmd)
	memberCmd.AddCommand(memberAddCmd)
	memberCmd.AddCommand(memberRemoveCmd)

	memberAddCmd.Flags().String("member-id", "", "User ID to add (required)")
	memberAddCmd.Flags().Int("role", 15, "Role: 5=Guest, 10=Viewer, 15=Member, 20=Admin")
	memberAddCmd.MarkFlagRequired("member-id")

	memberRemoveCmd.Flags().String("member-id", "", "User ID to remove (required)")
	memberRemoveCmd.MarkFlagRequired("member-id")
}

var memberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage workspace and project members",
	Long: `Manage workspace and project members.

API docs: https://developers.plane.so/api-reference/members/overview
CLI docs: plane docs member`,
}

var memberListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List workspace members",
	Example: `  plane member list -w dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClient()
		if err != nil {
			return err
		}
		if err := requireWorkspace(client); err != nil {
			return err
		}

		svc := api.NewMembersService(client)
		members, err := svc.ListWorkspace(context.Background())
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.MemberList{Results: members})
	},
}

var memberListProjectCmd = &cobra.Command{
	Use:     "list-project",
	Short:   "List project members",
	Example: `  plane member list-project -p PLANECLI`,
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

		svc := api.NewMembersService(client)
		members, err := svc.ListProject(context.Background(), projectID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.MemberList{Results: members})
	},
}

var memberAddCmd = &cobra.Command{
	Use:     "add",
	Short:   "Add a member to a project",
	Example: `  plane member add --member-id <user-id> --role 15 -p PLANECLI`,
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

		memberID, _ := cmd.Flags().GetString("member-id")
		role, _ := cmd.Flags().GetInt("role")

		svc := api.NewMembersService(client)
		if err := svc.Add(context.Background(), projectID, memberID, role); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Member added.")
		return nil
	},
}

var memberRemoveCmd = &cobra.Command{
	Use:     "remove",
	Short:   "Remove a member from a project",
	Example: `  plane member remove --member-id <user-id> -p PLANECLI`,
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

		memberID, _ := cmd.Flags().GetString("member-id")
		svc := api.NewMembersService(client)
		if err := svc.Remove(context.Background(), projectID, memberID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Member removed.")
		return nil
	},
}
