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
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(stateListCmd)
	stateCmd.AddCommand(stateGetCmd)
	stateCmd.AddCommand(stateCreateCmd)
	stateCmd.AddCommand(stateUpdateCmd)
	stateCmd.AddCommand(stateDeleteCmd)

	stateGetCmd.Flags().String("state", "", "State ID (required)")
	stateGetCmd.MarkFlagRequired("state")

	stateCreateCmd.Flags().String("name", "", "State name (required)")
	stateCreateCmd.Flags().String("color", "#999999", "State color (hex)")
	stateCreateCmd.Flags().String("group", "backlog", "State group: backlog, unstarted, started, completed, cancelled")
	stateCreateCmd.Flags().String("description", "", "State description")
	stateCreateCmd.MarkFlagRequired("name")

	stateUpdateCmd.Flags().String("state", "", "State ID (required)")
	stateUpdateCmd.Flags().String("name", "", "State name")
	stateUpdateCmd.Flags().String("color", "", "State color (hex)")
	stateUpdateCmd.Flags().String("group", "", "State group")
	stateUpdateCmd.Flags().String("description", "", "State description")
	stateUpdateCmd.MarkFlagRequired("state")

	stateDeleteCmd.Flags().String("state", "", "State ID (required)")
	stateDeleteCmd.MarkFlagRequired("state")
}

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage workflow states",
	Long: `Manage workflow states in a project.

API docs: https://developers.plane.so/api-reference/state/overview
CLI docs: plane docs state`,
}

var stateListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List states in a project",
	Example: `  plane state list -p PLANECLI`,
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

		svc := api.NewStatesService(client)
		states, err := svc.List(context.Background(), projectID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.StateList{Results: states})
	},
}

var stateGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a state by ID",
	Example: `  plane state get --state <state-id> -p PLANECLI`,
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

		stateID, _ := cmd.Flags().GetString("state")
		svc := api.NewStatesService(client)
		state, err := svc.Get(context.Background(), projectID, stateID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, state)
	},
}

var stateCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new state",
	Example: `  plane state create -p PLANECLI --name "In Review" --group started --color "#FFA500"`,
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
		color, _ := cmd.Flags().GetString("color")
		group, _ := cmd.Flags().GetString("group")
		description, _ := cmd.Flags().GetString("description")

		input := models.StateCreate{
			Name:        name,
			Color:       color,
			Group:       group,
			Description: description,
		}

		svc := api.NewStatesService(client)
		state, err := svc.Create(context.Background(), projectID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, state)
	},
}

var stateUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a state",
	Example: `  plane state update --state <state-id> -p PLANECLI --name "Code Review"`,
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

		stateID, _ := cmd.Flags().GetString("state")
		input := models.StateUpdate{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			input.Name = &v
		}
		if cmd.Flags().Changed("color") {
			v, _ := cmd.Flags().GetString("color")
			input.Color = &v
		}
		if cmd.Flags().Changed("group") {
			v, _ := cmd.Flags().GetString("group")
			input.Group = &v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
		}

		svc := api.NewStatesService(client)
		state, err := svc.Update(context.Background(), projectID, stateID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, state)
	},
}

var stateDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a state",
	Example: `  plane state delete --state <state-id> -p PLANECLI`,
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

		stateID, _ := cmd.Flags().GetString("state")
		svc := api.NewStatesService(client)
		if err := svc.Delete(context.Background(), projectID, stateID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "State deleted.")
		return nil
	},
}
