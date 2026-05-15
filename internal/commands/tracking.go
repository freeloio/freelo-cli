package commands

import (
	"fmt"

	"github.com/freeloio/freelo-cli/internal/output"
	freelo "github.com/freeloio/freelo-go/freeloapi"
	"github.com/spf13/cobra"
)

// NewTrackingCmd creates the 'tracking' command group.
func NewTrackingCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tracking",
		Aliases: []string{"track", "timer"},
		Short:   "Time tracking",
	}

	cmd.AddCommand(
		newTrackingStartCmd(app),
		newTrackingStopCmd(app),
		newTrackingStatusCmd(app),
	)

	return cmd
}

func newTrackingStartCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start time tracking on a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			taskID, _ := cmd.Flags().GetInt("task")
			note, _ := cmd.Flags().GetString("note")

			if taskID == 0 {
				return fmt.Errorf("--task is required")
			}

			body := freelo.StartTimeTrackingJSONRequestBody{TaskId: &taskID}
			if note != "" {
				body.Note = &note
			}

			tracking, err := consumeAPIObject(app.FreeloClient.StartTimeTracking(cmd.Context(), body))
			if err != nil {
				out.Err(err, "start_failed", "")
				return err
			}

			out.OK(tracking, fmt.Sprintf("Time tracking started on task %d", taskID), []output.Breadcrumb{
				{Action: "stop", Cmd: "freelo tracking stop", Description: "Stop tracking"},
				{Action: "status", Cmd: "freelo tracking status", Description: "Check tracking status"},
			})
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("note", "", "Optional note")
	return cmd
}

func newTrackingStopCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop active time tracking",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			report, err := consumeAPIObject(app.FreeloClient.StopTimeTracking(cmd.Context()))
			if err != nil {
				out.Err(err, "stop_failed", "Is there an active timer? Check with 'freelo tracking status'")
				return err
			}

			out.OK(report, "Time tracking stopped", nil)
			return nil
		},
	}
}

func newTrackingStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active time tracking status",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			status, err := consumeAPIObject(app.FreeloClient.GetTimeTrackingStatus(cmd.Context()))
			if err != nil {
				out.Err(err, "status_failed", "")
				return err
			}

			// Server contract: returns bare JSON `null` when nothing is
			// being tracked, otherwise an object with `task_id`. We
			// normalize to a stable envelope `{"active": bool, "server": <orig>}`
			// so consumers (jq, agents) always see the same top-level
			// shape and a clear bool. Prior versions injected an
			// "active" key directly into the server's map; that worked
			// but lied about what the server returned. The wrapper is
			// explicit about what's ours and what's the server's.
			summary := "No active tracking"
			active := false
			if status != nil {
				if taskID, ok := status["task_id"]; ok && taskID != nil {
					active = true
					summary = fmt.Sprintf("Tracking task %v", taskID)
				}
			}
			envelope := map[string]any{
				"active": active,
				"server": status, // nil when nothing is tracked
			}
			status = envelope

			out.OK(status, summary, []output.Breadcrumb{
				{Action: "stop", Cmd: "freelo tracking stop", Description: "Stop tracking"},
			})
			return nil
		},
	}
}
