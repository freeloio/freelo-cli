package commands

import (
	"fmt"
	"time"

	freelo "github.com/freeloio/freelo-go/freeloapi"
	"github.com/freeloio/freelo-go/freelotime"
	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewSubtasksCmd creates the 'subtasks' command group.
//
// The Freelo API only supports listing and creating subtasks (endpoints
// /task/{id}/subtasks). The pre-v1.0.0 CLI exposed `show`, `finish`,
// `activate`, and `delete` subcommands calling /subtask/{id}{/finish,...},
// but every one of those paths returns 404 from the server — even with a
// real subtask ID. They never worked. Dropped in Phase 3 to match the
// actual API surface documented in the OpenAPI spec.
func NewSubtasksCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subtasks",
		Aliases: []string{"subtask", "st"},
		Short:   "Manage subtasks (list + create only — see comment in subtasks.go)",
	}

	cmd.AddCommand(
		newSubtasksListCmd(app),
		newSubtasksCreateCmd(app),
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

			body, err := consumeAPIBody(app.FreeloClient.GetSubtasksInTask(cmd.Context(), taskID, &freelo.GetSubtasksInTaskParams{}))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			subtasks, _ := parsePaginatedItems(body)

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
			dueDateStr, _ := cmd.Flags().GetString("due-date")
			workerID, _ := cmd.Flags().GetInt("worker")
			priority, _ := cmd.Flags().GetString("priority")

			if taskID == 0 || name == "" {
				return fmt.Errorf("--task and --name are required")
			}

			body := freelo.SubtaskCreate{Name: name}
			if dueDateStr != "" {
				t, err := time.Parse("2006-01-02", dueDateStr)
				if err != nil {
					return fmt.Errorf("invalid --due-date %q, want YYYY-MM-DD", dueDateStr)
				}
				body.DueDate = &freelotime.Time{Time: t}
			}
			if workerID != 0 {
				body.Worker = &workerID
			}
			if priority != "" {
				p := freelo.SubtaskCreatePriorityEnum(priority)
				body.PriorityEnum = &p
			}

			subtask, err := consumeAPIObject(app.FreeloClient.CreateSubtask(cmd.Context(), taskID, body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
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
