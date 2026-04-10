package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func NewLabelsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "labels",
		Aliases: []string{"label"},
		Short:   "Manage task and project labels",
	}

	cmd.AddCommand(
		newLabelsListCmd(app),
		newLabelsCreateCmd(app),
		newLabelsAddToTaskCmd(app),
		newLabelsRemoveFromTaskCmd(app),
		newLabelsAddToProjectCmd(app),
		newLabelsRemoveFromProjectCmd(app),
		newLabelsEditCmd(app),
		newLabelsDeleteCmd(app),
	)

	return cmd
}

func newLabelsListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			result, err := app.Client.Get("/project-labels/find-available")
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			var labels []map[string]any
			_ = json.Unmarshal(result, &labels)
			out.OK(labels, fmt.Sprintf("%d labels", len(labels)), nil)
			return nil
		},
	}
}

func newLabelsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create task labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			name, _ := cmd.Flags().GetString("name")
			color, _ := cmd.Flags().GetString("color")

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			body := map[string]any{"name": name}
			if color != "" {
				body["color"] = color
			}

			result, err := app.Client.Post("/task-labels", body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			var label map[string]any
			_ = json.Unmarshal(result, &label)
			out.OK(label, fmt.Sprintf("Label '%s' created", name), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "Label name (required)")
	cmd.Flags().String("color", "", "Color hex code (e.g. #ff0000)")
	return cmd
}

func newLabelsAddToTaskCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-to-task",
		Short: "Add labels to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID, _ := cmd.Flags().GetInt("task")
			name, _ := cmd.Flags().GetString("name")
			uuid, _ := cmd.Flags().GetString("uuid")

			if taskID == 0 {
				return fmt.Errorf("--task is required")
			}

			body := map[string]any{}
			if uuid != "" {
				body["labels"] = []map[string]any{{"uuid": uuid}}
			} else if name != "" {
				body["labels"] = []map[string]any{{"name": name}}
			} else {
				return fmt.Errorf("--name or --uuid is required")
			}

			result, err := app.Client.Post(fmt.Sprintf("/task-labels/add-to-task/%d", taskID), body)
			if err != nil {
				out.Err(err, "add_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, fmt.Sprintf("Label added to task %d", taskID), nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("name", "", "Label name")
	cmd.Flags().String("uuid", "", "Label UUID")
	return cmd
}

func newLabelsRemoveFromTaskCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-from-task",
		Short: "Remove labels from a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID, _ := cmd.Flags().GetInt("task")
			uuid, _ := cmd.Flags().GetString("uuid")

			if taskID == 0 || uuid == "" {
				return fmt.Errorf("--task and --uuid are required")
			}

			body := map[string]any{
				"labels": []map[string]any{{"uuid": uuid}},
			}

			result, err := app.Client.Post(fmt.Sprintf("/task-labels/remove-from-task/%d", taskID), body)
			if err != nil {
				out.Err(err, "remove_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, fmt.Sprintf("Label removed from task %d", taskID), nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("uuid", "", "Label UUID (required)")
	return cmd
}

func newLabelsAddToProjectCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-to-project",
		Short: "Add a label to a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			name, _ := cmd.Flags().GetString("name")
			color, _ := cmd.Flags().GetString("color")

			if projectID == 0 || name == "" {
				return fmt.Errorf("--project and --name are required")
			}

			body := map[string]any{"name": name}
			if color != "" {
				body["color"] = color
			}

			result, err := app.Client.Post(fmt.Sprintf("/project-labels/add-to-project/%d", projectID), body)
			if err != nil {
				out.Err(err, "add_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, fmt.Sprintf("Label added to project %d", projectID), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("name", "", "Label name (required)")
	cmd.Flags().String("color", "", "Color hex code")
	return cmd
}

func newLabelsRemoveFromProjectCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-from-project",
		Short: "Remove a label from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			labelID, _ := cmd.Flags().GetString("label-id")

			if projectID == 0 || labelID == "" {
				return fmt.Errorf("--project and --label-id are required")
			}

			result, err := app.Client.Post(fmt.Sprintf("/project-labels/remove-from-project/%d", projectID), map[string]any{
				"label_id": labelID,
			})
			if err != nil {
				out.Err(err, "remove_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, "Label removed from project", nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("label-id", "", "Label ID (required)")
	return cmd
}

func newLabelsEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <label-id>",
		Short: "Edit a project label",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			labelID := args[0]

			body := map[string]any{}
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				body["name"] = name
			}
			if color, _ := cmd.Flags().GetString("color"); color != "" {
				body["color"] = color
			}

			if len(body) == 0 {
				return fmt.Errorf("at least --name or --color is required")
			}

			result, err := app.Client.Post("/project-labels/"+labelID, body)
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}
			var label map[string]any
			_ = json.Unmarshal(result, &label)
			out.OK(label, fmt.Sprintf("Label %s updated", labelID), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "New name")
	cmd.Flags().String("color", "", "New color hex code")
	return cmd
}

func newLabelsDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <label-id>",
		Short: "Delete a project label",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			_, err := app.Client.Delete("/project-labels/" + args[0])
			if err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"id": args[0], "deleted": true}, fmt.Sprintf("Label %s deleted", args[0]), nil)
			return nil
		},
	}
}
