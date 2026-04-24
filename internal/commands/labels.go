package commands

import (
	"encoding/json"
	"fmt"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// NewLabelsCmd creates the 'labels' command group.
//
// Phase 3 migration: the two task-label mutation endpoints use oneOf
// request schemas (TaskLabelAddInput / TaskLabelRemoveInput), so the body
// is built via the From* helpers on the generated union types. Project-
// label operations are straightforward typed bodies.
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
			body, err := consumeAPIBody(app.FreeloClient.FindAvailableProjectLabels(cmd.Context()))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			// Freelo returns JSON null when the workspace has no labels
			// yet — normalize to an empty slice so the envelope stays []-
			// typed instead of leaking null through to consumers.
			labels := make([]map[string]any, 0)
			_ = json.Unmarshal(body, &labels)
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

			label := struct {
				Color *string `json:"color,omitempty"`
				Name  *string `json:"name,omitempty"`
			}{Name: &name}
			if color != "" {
				label.Color = &color
			}
			labels := []struct {
				Color *string `json:"color,omitempty"`
				Name  *string `json:"name,omitempty"`
			}{label}
			body := freelo.CreateTaskLabelsJSONRequestBody{Labels: &labels}

			resp, err := consumeAPIObject(app.FreeloClient.CreateTaskLabels(cmd.Context(), body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			out.OK(resp, fmt.Sprintf("Label '%s' created", name), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "Label name (required)")
	cmd.Flags().String("color", "", "Color hex code (e.g. #ff0000)")
	return cmd
}

// buildTaskLabelAdd returns a TaskLabelAddInput built from either a UUID
// (union variant 0) or a name + optional color (union variant 1).
func buildTaskLabelAdd(uuidStr, name, color string) (freelo.TaskLabelAddInput, error) {
	var in freelo.TaskLabelAddInput
	if uuidStr != "" {
		u, err := uuid.Parse(uuidStr)
		if err != nil {
			return in, fmt.Errorf("invalid --uuid: %w", err)
		}
		if err := in.FromTaskLabelAddInput0(freelo.TaskLabelAddInput0{Uuid: u}); err != nil {
			return in, err
		}
		return in, nil
	}
	v := freelo.TaskLabelAddInput1{Name: name}
	if color != "" {
		v.Color = &color
	}
	if err := in.FromTaskLabelAddInput1(v); err != nil {
		return in, err
	}
	return in, nil
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
			color, _ := cmd.Flags().GetString("color")

			if taskID == 0 {
				return fmt.Errorf("--task is required")
			}
			if uuid == "" && name == "" {
				return fmt.Errorf("--name or --uuid is required")
			}

			in, err := buildTaskLabelAdd(uuid, name, color)
			if err != nil {
				return err
			}
			body := freelo.AddTaskLabelsToTaskJSONRequestBody{Labels: []freelo.TaskLabelAddInput{in}}

			resp, err := consumeAPIObject(app.FreeloClient.AddTaskLabelsToTask(cmd.Context(), taskID, body))
			if err != nil {
				out.Err(err, "add_failed", "")
				return err
			}
			out.OK(resp, fmt.Sprintf("Label added to task %d", taskID), nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("name", "", "Label name (use --uuid to reference an existing label instead)")
	cmd.Flags().String("uuid", "", "Label UUID")
	cmd.Flags().String("color", "", "Label color hex (optional, only when --name)")
	return cmd
}

func newLabelsRemoveFromTaskCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-from-task",
		Short: "Remove labels from a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID, _ := cmd.Flags().GetInt("task")
			uuidStr, _ := cmd.Flags().GetString("uuid")
			name, _ := cmd.Flags().GetString("name")
			color, _ := cmd.Flags().GetString("color")

			if taskID == 0 {
				return fmt.Errorf("--task is required")
			}
			if uuidStr == "" && name == "" {
				return fmt.Errorf("--uuid or --name is required")
			}

			var in freelo.TaskLabelRemoveInput
			switch {
			case uuidStr != "":
				u, err := uuid.Parse(uuidStr)
				if err != nil {
					return fmt.Errorf("invalid --uuid: %w", err)
				}
				if err := in.FromTaskLabelRemoveInput0(freelo.TaskLabelRemoveInput0{Uuid: u}); err != nil {
					return err
				}
			case color != "":
				if err := in.FromTaskLabelRemoveInput2(freelo.TaskLabelRemoveInput2{Name: name, Color: color}); err != nil {
					return err
				}
			default:
				if err := in.FromTaskLabelRemoveInput1(freelo.TaskLabelRemoveInput1{Name: name}); err != nil {
					return err
				}
			}
			body := freelo.RemoveTaskLabelsFromTaskJSONRequestBody{Labels: []freelo.TaskLabelRemoveInput{in}}

			resp, err := consumeAPIObject(app.FreeloClient.RemoveTaskLabelsFromTask(cmd.Context(), taskID, body))
			if err != nil {
				out.Err(err, "remove_failed", "")
				return err
			}
			out.OK(resp, fmt.Sprintf("Label removed from task %d", taskID), nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("uuid", "", "Label UUID (preferred)")
	cmd.Flags().String("name", "", "Label name (removes all colors unless --color is given)")
	cmd.Flags().String("color", "", "Narrow by color hex (only when --name)")
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

			body := freelo.AddProjectLabelToProjectJSONRequestBody{Name: &name}
			if color != "" {
				body.Color = &color
			}

			resp, err := consumeAPIObject(app.FreeloClient.AddProjectLabelToProject(cmd.Context(), projectID, body))
			if err != nil {
				out.Err(err, "add_failed", "")
				return err
			}
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
			labelID, _ := cmd.Flags().GetInt("label-id")

			if projectID == 0 || labelID == 0 {
				return fmt.Errorf("--project and --label-id are required")
			}

			body := freelo.RemoveProjectLabelFromProjectJSONRequestBody{Id: &labelID}

			resp, err := consumeAPIObject(app.FreeloClient.RemoveProjectLabelFromProject(cmd.Context(), projectID, body))
			if err != nil {
				out.Err(err, "remove_failed", "")
				return err
			}
			out.OK(resp, "Label removed from project", nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().Int("label-id", 0, "Label ID (required)")
	return cmd
}

func newLabelsEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <label-id>",
		Short: "Edit a project label",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			labelID := mustInt(args[0])

			body := freelo.EditProjectLabelJSONRequestBody{}
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				body.Name = &name
			}
			if color, _ := cmd.Flags().GetString("color"); color != "" {
				body.Color = &color
			}
			if body.Name == nil && body.Color == nil {
				return fmt.Errorf("at least --name or --color is required")
			}

			resp, err := consumeAPIObject(app.FreeloClient.EditProjectLabel(cmd.Context(), labelID, body))
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}
			out.OK(resp, fmt.Sprintf("Label %d updated", labelID), nil)
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
			labelID := mustInt(args[0])
			if _, err := consumeAPIObject(app.FreeloClient.DeleteProjectLabel(cmd.Context(), labelID)); err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"id": labelID, "deleted": true}, fmt.Sprintf("Label %d deleted", labelID), nil)
			return nil
		},
	}
}
