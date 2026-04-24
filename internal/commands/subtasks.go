package commands

import (
	"encoding/json"
	"fmt"

	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewSubtasksCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subtasks",
		Aliases: []string{"subtask", "st"},
		Short:   "Manage subtasks",
	}

	cmd.AddCommand(
		newSubtasksListCmd(app),
		newSubtasksCreateCmd(app),
		newSubtasksShowCmd(app),
		newSubtasksFinishCmd(app),
		newSubtasksActivateCmd(app),
		newSubtasksDeleteCmd(app),
	)

	return cmd
}

func newSubtasksListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List subtasks of a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID, _ := cmd.Flags().GetInt("task")
			if taskID == 0 {
				return fmt.Errorf("--task is required")
			}

			result, err := app.Client.Get(fmt.Sprintf("/task/%d/subtasks", taskID))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			subtasks, _ := parsePaginatedItems(result)

			simplified := make([]map[string]any, 0, len(subtasks))
			for _, s := range subtasks {
				item := map[string]any{"id": s["id"], "name": s["name"]}
				if v, ok := s["state"]; ok {
					item["state"] = v
				}
				if v, ok := s["due_date"]; ok && v != nil {
					item["due_date"] = v
				}
				if w, ok := s["worker"].(map[string]any); ok {
					item["worker"] = w["fullname"]
				}
				simplified = append(simplified, item)
			}

			out.OK(simplified, fmt.Sprintf("%d subtasks", len(simplified)), []output.Breadcrumb{
				{Action: "create", Cmd: fmt.Sprintf("freelo subtasks create --task %d --name <name>", taskID), Description: "Create subtask"},
			})
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Parent task ID (required)")
	return cmd
}

func newSubtasksCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a subtask",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID, _ := cmd.Flags().GetInt("task")
			name, _ := cmd.Flags().GetString("name")
			dueDate, _ := cmd.Flags().GetString("due-date")
			workerID, _ := cmd.Flags().GetInt("worker")
			priority, _ := cmd.Flags().GetString("priority")

			if taskID == 0 || name == "" {
				return fmt.Errorf("--task and --name are required")
			}

			body := map[string]any{"name": name}
			if dueDate != "" {
				body["due_date"] = dueDate
			}
			if workerID != 0 {
				body["worker"] = map[string]any{"id": workerID}
			}
			if priority != "" {
				body["priority"] = priority
			}

			result, err := app.Client.Post(fmt.Sprintf("/task/%d/subtasks", taskID), body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			var subtask map[string]any
			_ = json.Unmarshal(result, &subtask)
			out.OK(subtask, fmt.Sprintf("Subtask '%s' created", name), nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Parent task ID (required)")
	cmd.Flags().String("name", "", "Subtask name (required)")
	cmd.Flags().String("due-date", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().Int("worker", 0, "Assign to worker (user ID)")
	cmd.Flags().String("priority", "", "Priority: h, m, l")
	return cmd
}

func newSubtasksShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <subtask-id>",
		Short: "Show subtask detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			result, err := app.Client.Get("/subtask/" + args[0])
			if err != nil {
				out.Err(err, "not_found", "")
				return err
			}
			var subtask map[string]any
			_ = json.Unmarshal(result, &subtask)
			out.OK(subtask, "", nil)
			return nil
		},
	}
}

func newSubtasksFinishCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "finish <subtask-id>",
		Short: "Mark subtask as finished",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			_, err := app.Client.Post("/subtask/"+args[0]+"/finish", nil)
			if err != nil {
				out.Err(err, "finish_failed", "")
				return err
			}
			out.OK(map[string]any{"id": mustInt(args[0]), "finished": true}, fmt.Sprintf("Subtask %s finished", args[0]), nil)
			return nil
		},
	}
}

func newSubtasksActivateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "activate <subtask-id>",
		Short: "Reopen a finished subtask",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			_, err := app.Client.Post("/subtask/"+args[0]+"/activate", nil)
			if err != nil {
				out.Err(err, "activate_failed", "")
				return err
			}
			out.OK(map[string]any{"id": mustInt(args[0]), "activated": true}, fmt.Sprintf("Subtask %s activated", args[0]), nil)
			return nil
		},
	}
}

func newSubtasksDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <subtask-id>",
		Short: "Delete a subtask",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			_, err := app.Client.Delete("/subtask/" + args[0])
			if err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"id": mustInt(args[0]), "deleted": true}, fmt.Sprintf("Subtask %s deleted", args[0]), nil)
			return nil
		},
	}
}
