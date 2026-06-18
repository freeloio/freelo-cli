package commands

import (
	"fmt"

	freelo "github.com/freeloio/freelo-go/freeloapi"
	"github.com/spf13/cobra"
)

// NewTaskchecksCmd creates the 'taskchecks' command group.
//
// Taskchecks are checklist-style items on a task. The Freelo API exposes
// only mutation endpoints under /taskcheck/{id} — there is no list/create
// endpoint (taskchecks are created via the subtasks API, subtask type
// "taskcheck"). So this group is edit/activate/finish/delete only.
func NewTaskchecksCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "taskchecks",
		Aliases: []string{"taskcheck", "tc"},
		Short:   "Manage taskchecks (edit/activate/finish/delete — created via subtasks)",
	}

	cmd.AddCommand(
		newTaskchecksEditCmd(app),
		newTaskchecksFinishCmd(app),
		newTaskchecksActivateCmd(app),
		newTaskchecksDeleteCmd(app),
	)

	return cmd
}

func newTaskchecksEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <taskcheck-id>",
		Short: "Edit a taskcheck (name and/or assigned worker)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			id, err := parseIntArg(args[0], "taskcheck-id")
			if err != nil {
				return err
			}

			body := freelo.EditTaskcheckJSONRequestBody{}
			if cmd.Flags().Changed("name") {
				name, _ := cmd.Flags().GetString("name")
				body.Name = &name
			}
			if cmd.Flags().Changed("worker") {
				worker, _ := cmd.Flags().GetInt("worker")
				body.Worker = &worker
			}
			if body.Name == nil && body.Worker == nil {
				return fmt.Errorf("nothing to edit: pass --name and/or --worker")
			}

			obj, err := consumeAPIObject(app.FreeloClient.EditTaskcheck(cmd.Context(), id, body))
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}
			out.OK(obj, fmt.Sprintf("Taskcheck %d updated", id), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "New taskcheck name")
	cmd.Flags().Int("worker", 0, "Assign to worker (user ID)")
	return cmd
}

func newTaskchecksFinishCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "finish <taskcheck-id>",
		Short: "Mark a taskcheck as finished",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			id, err := parseIntArg(args[0], "taskcheck-id")
			if err != nil {
				return err
			}
			if _, err := consumeAPIObject(app.FreeloClient.FinishTaskcheck(cmd.Context(), id)); err != nil {
				out.Err(err, "finish_failed", "")
				return err
			}
			out.OK(map[string]any{"id": id, "finished": true}, fmt.Sprintf("Taskcheck %d finished", id), nil)
			return nil
		},
	}
}

func newTaskchecksActivateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "activate <taskcheck-id>",
		Short: "Reactivate a finished taskcheck",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			id, err := parseIntArg(args[0], "taskcheck-id")
			if err != nil {
				return err
			}
			if _, err := consumeAPIObject(app.FreeloClient.ActivateTaskcheck(cmd.Context(), id)); err != nil {
				out.Err(err, "activate_failed", "")
				return err
			}
			out.OK(map[string]any{"id": id, "active": true}, fmt.Sprintf("Taskcheck %d activated", id), nil)
			return nil
		},
	}
}

func newTaskchecksDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <taskcheck-id>",
		Short: "Delete a taskcheck",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			id, err := parseIntArg(args[0], "taskcheck-id")
			if err != nil {
				return err
			}
			if _, err := consumeAPIObject(app.FreeloClient.DeleteTaskcheck(cmd.Context(), id)); err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"id": id, "deleted": true}, fmt.Sprintf("Taskcheck %d deleted", id), nil)
			return nil
		},
	}
}
