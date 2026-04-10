package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewReportsCmd creates the 'reports' command group (work reports).
func NewReportsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reports",
		Aliases: []string{"report", "wr"},
		Short:   "Manage work reports",
	}

	cmd.AddCommand(
		newReportsListCmd(app),
		newReportsCreateCmd(app),
		newReportsEditCmd(app),
		newReportsDeleteCmd(app),
	)

	return cmd
}

func newReportsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List work reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			userID, _ := cmd.Flags().GetInt("user")
			page, _ := cmd.Flags().GetInt("page")

			path := "/work-reports"
			params := []string{}
			if projectID != 0 {
				params = append(params, fmt.Sprintf("projects_ids[]=%d", projectID))
			}
			if userID != 0 {
				params = append(params, fmt.Sprintf("users_ids[]=%d", userID))
			}
			if page > 0 {
				params = append(params, fmt.Sprintf("p=%d", page))
			}
			if len(params) > 0 {
				path += "?" + strings.Join(params, "&")
			}

			result, err := app.Client.Get(path)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			reports, _ := parsePaginatedItems(result)

			simplified := make([]map[string]any, 0, len(reports))
			for _, r := range reports {
				item := map[string]any{
					"id":      r["id"],
					"minutes": r["minutes"],
					"note":    r["note"],
				}
				if v, ok := r["date_reported"]; ok {
					item["date"] = v
				}
				if worker, ok := r["worker"].(map[string]any); ok {
					item["worker"] = worker["fullname"]
				}
				simplified = append(simplified, item)
			}

			out.OK(simplified, fmt.Sprintf("%d work reports", len(simplified)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("user", 0, "Filter by user ID")
	cmd.Flags().Int("page", 0, "Page number")
	return cmd
}

func newReportsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a work report",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			taskID, _ := cmd.Flags().GetInt("task")
			minutes, _ := cmd.Flags().GetInt("minutes")
			date, _ := cmd.Flags().GetString("date")
			note, _ := cmd.Flags().GetString("note")
			workerID, _ := cmd.Flags().GetInt("worker")

			if taskID == 0 || minutes == 0 {
				return fmt.Errorf("--task and --minutes are required")
			}

			body := map[string]any{
				"minutes": minutes,
			}
			if date != "" {
				body["date_reported"] = date
			}
			if note != "" {
				body["note"] = note
			}
			if workerID != 0 {
				body["worker_id"] = workerID
			}

			path := fmt.Sprintf("/task/%d/work-reports", taskID)
			result, err := app.Client.Post(path, body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			var report map[string]any
			_ = json.Unmarshal(result, &report)

			out.OK(report, fmt.Sprintf("Work report created: %d minutes", minutes), nil)
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().Int("minutes", 0, "Duration in minutes (required)")
	cmd.Flags().String("date", "", "Date reported (YYYY-MM-DD)")
	cmd.Flags().String("note", "", "Note")
	cmd.Flags().Int("worker", 0, "Worker (user) ID")
	return cmd
}

func newReportsEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <report-id>",
		Short: "Edit a work report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			reportID := args[0]

			body := map[string]any{}
			if minutes, _ := cmd.Flags().GetInt("minutes"); minutes != 0 {
				body["minutes"] = minutes
			}
			if date, _ := cmd.Flags().GetString("date"); date != "" {
				body["date_reported"] = date
			}
			if note, _ := cmd.Flags().GetString("note"); note != "" {
				body["note"] = note
			}

			if len(body) == 0 {
				return fmt.Errorf("at least one field to edit is required")
			}

			result, err := app.Client.Post("/work-reports/"+reportID, body)
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}

			var report map[string]any
			_ = json.Unmarshal(result, &report)

			out.OK(report, fmt.Sprintf("Report %s updated", reportID), nil)
			return nil
		},
	}
	cmd.Flags().Int("minutes", 0, "Duration in minutes")
	cmd.Flags().String("date", "", "Date reported (YYYY-MM-DD)")
	cmd.Flags().String("note", "", "Note")
	return cmd
}

func newReportsDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <report-id>",
		Short: "Delete a work report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			reportID := args[0]

			_, err := app.Client.Delete("/work-reports/" + reportID)
			if err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(reportID), "deleted": true}, fmt.Sprintf("Report %s deleted", reportID), nil)
			return nil
		},
	}
}
