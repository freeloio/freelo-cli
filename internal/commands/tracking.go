package commands

import (
	"encoding/json"
	"fmt"

	"github.com/freeloapp/freelo-cli/internal/output"
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

			body := map[string]any{
				"task_id": taskID,
			}
			if note != "" {
				body["note"] = note
			}

			result, err := app.Client.Post("/timetracking/start", body)
			if err != nil {
				out.Err(err, "start_failed", "")
				return err
			}

			var tracking map[string]any
			_ = json.Unmarshal(result, &tracking)

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

			result, err := app.Client.Post("/timetracking/stop", nil)
			if err != nil {
				out.Err(err, "stop_failed", "Is there an active timer? Check with 'freelo tracking status'")
				return err
			}

			var report map[string]any
			_ = json.Unmarshal(result, &report)

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

			result, err := app.Client.Get("/timetracking/status")
			if err != nil {
				out.Err(err, "status_failed", "")
				return err
			}

			var status map[string]any
			_ = json.Unmarshal(result, &status)

			summary := "No active tracking"
			if taskID, ok := status["task_id"]; ok && taskID != nil {
				summary = fmt.Sprintf("Tracking task %v", taskID)
			}

			out.OK(status, summary, []output.Breadcrumb{
				{Action: "stop", Cmd: "freelo tracking stop", Description: "Stop tracking"},
			})
			return nil
		},
	}
}
