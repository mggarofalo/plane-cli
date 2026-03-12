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
	rootCmd.AddCommand(labelCmd)
	labelCmd.AddCommand(labelListCmd)
	labelCmd.AddCommand(labelGetCmd)
	labelCmd.AddCommand(labelCreateCmd)
	labelCmd.AddCommand(labelUpdateCmd)
	labelCmd.AddCommand(labelDeleteCmd)

	labelGetCmd.Flags().String("label", "", "Label ID (required)")
	labelGetCmd.MarkFlagRequired("label")

	labelCreateCmd.Flags().String("name", "", "Label name (required)")
	labelCreateCmd.Flags().String("color", "#999999", "Label color (hex)")
	labelCreateCmd.Flags().String("description", "", "Label description")
	labelCreateCmd.Flags().String("parent-id", "", "Parent label ID")
	labelCreateCmd.MarkFlagRequired("name")

	labelUpdateCmd.Flags().String("label", "", "Label ID (required)")
	labelUpdateCmd.Flags().String("name", "", "Label name")
	labelUpdateCmd.Flags().String("color", "", "Label color (hex)")
	labelUpdateCmd.Flags().String("description", "", "Label description")
	labelUpdateCmd.Flags().String("parent-id", "", "Parent label ID")
	labelUpdateCmd.MarkFlagRequired("label")

	labelDeleteCmd.Flags().String("label", "", "Label ID (required)")
	labelDeleteCmd.MarkFlagRequired("label")
}

var labelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage labels",
	Long: `Manage labels in a project.

API docs: https://developers.plane.so/api-reference/label/overview
CLI docs: plane docs label`,
}

var labelListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List labels in a project",
	Example: `  plane label list -p PLANECLI`,
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

		svc := api.NewLabelsService(client)
		labels, err := svc.List(context.Background(), projectID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, models.LabelList{Results: labels})
	},
}

var labelGetCmd = &cobra.Command{
	Use:     "get",
	Short:   "Get a label by ID",
	Example: `  plane label get --label <label-id> -p PLANECLI`,
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

		labelID, _ := cmd.Flags().GetString("label")
		svc := api.NewLabelsService(client)
		label, err := svc.Get(context.Background(), projectID, labelID)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, label)
	},
}

var labelCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new label",
	Example: `  plane label create -p PLANECLI --name "bug" --color "#FF0000"`,
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
		description, _ := cmd.Flags().GetString("description")
		parentID, _ := cmd.Flags().GetString("parent-id")

		input := models.LabelCreate{
			Name:        name,
			Color:       color,
			Description: description,
			ParentID:    parentID,
		}

		svc := api.NewLabelsService(client)
		label, err := svc.Create(context.Background(), projectID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, label)
	},
}

var labelUpdateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update a label",
	Example: `  plane label update --label <label-id> -p PLANECLI --color "#00FF00"`,
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

		labelID, _ := cmd.Flags().GetString("label")
		input := models.LabelUpdate{}
		if cmd.Flags().Changed("name") {
			v, _ := cmd.Flags().GetString("name")
			input.Name = &v
		}
		if cmd.Flags().Changed("color") {
			v, _ := cmd.Flags().GetString("color")
			input.Color = &v
		}
		if cmd.Flags().Changed("description") {
			v, _ := cmd.Flags().GetString("description")
			input.Description = &v
		}
		if cmd.Flags().Changed("parent-id") {
			v, _ := cmd.Flags().GetString("parent-id")
			input.ParentID = &v
		}

		svc := api.NewLabelsService(client)
		label, err := svc.Update(context.Background(), projectID, labelID, input)
		if err != nil {
			return err
		}

		return formatter().Format(os.Stdout, label)
	},
}

var labelDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Delete a label",
	Example: `  plane label delete --label <label-id> -p PLANECLI`,
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

		labelID, _ := cmd.Flags().GetString("label")
		svc := api.NewLabelsService(client)
		if err := svc.Delete(context.Background(), projectID, labelID); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Label deleted.")
		return nil
	},
}
