package commands

import (
	"fmt"
	"time"

	freelo "github.com/freeloio/freelo-go/freeloapi"
	openapi_types "github.com/oapi-codegen/runtime/types"
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

// parseOpenAPIDate turns "YYYY-MM-DD" into the generated *openapi_types.Date,
// or nil on empty. Returns an error on malformed input.
func parseOpenAPIDate(s, flag string) (*openapi_types.Date, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid --%s %q, want YYYY-MM-DD", flag, s)
	}
	return &openapi_types.Date{Time: t}, nil
}

func newReportsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List work reports",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			userID, _ := cmd.Flags().GetInt("user")
			fromStr, _ := cmd.Flags().GetString("from")
			toStr, _ := cmd.Flags().GetString("to")

			params := &freelo.GetWorkReportsParams{}
			setProjectsFilter(&params.ProjectsIds, projectID)
			setUsersFilter(&params.UsersIds, userID)
			from, err := parseOpenAPIDate(fromStr, "from")
			if err != nil {
				return err
			}
			params.DateReportedRangeDateFrom = from
			to, err := parseOpenAPIDate(toStr, "to")
			if err != nil {
				return err
			}
			params.DateReportedRangeDateTo = to
			if err := setPageFilter(cmd, &params.P); err != nil {
				return err
			}

			body, err := consumeAPIBody(app.FreeloClient.GetWorkReports(cmd.Context(), params))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			reports, _ := parsePaginatedItems(body)

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
				if project, ok := r["project"].(map[string]any); ok {
					item["project_id"] = project["id"]
					item["project_name"] = project["name"]
				}
				if task, ok := r["task"].(map[string]any); ok {
					item["task_id"] = task["id"]
					item["task_name"] = task["name"]
				}
				simplified = append(simplified, item)
			}

			out.OK(simplified, fmt.Sprintf("%d work reports", len(simplified)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("user", 0, "Filter by user ID")
	cmd.Flags().String("from", "", "Filter by date_reported start (YYYY-MM-DD)")
	cmd.Flags().String("to", "", "Filter by date_reported end (YYYY-MM-DD)")
	cmd.Flags().Int("page", 0, "Page number (>= 1; omit for first page)")
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
			dateStr, _ := cmd.Flags().GetString("date")
			note, _ := cmd.Flags().GetString("note")
			workerID, _ := cmd.Flags().GetInt("worker")

			if taskID == 0 || minutes == 0 {
				return fmt.Errorf("--task and --minutes are required")
			}

			date, err := parseOpenAPIDate(dateStr, "date")
			if err != nil {
				return err
			}

			body := freelo.CreateWorkReportJSONRequestBody{Minutes: minutes}
			if date != nil {
				body.DateReported = date
			}
			if note != "" {
				body.Note = &note
			}
			if workerID != 0 {
				body.WorkerId = &workerID
			}

			report, err := consumeAPIObject(app.FreeloClient.CreateWorkReport(cmd.Context(), taskID, body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

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
			reportID, err := parseIntArg(args[0], "report-id")
			if err != nil {
				return err
			}

			body := freelo.EditWorkReportJSONRequestBody{}
			setAny := false
			if minutes, _ := cmd.Flags().GetInt("minutes"); minutes != 0 {
				body.Minutes = &minutes
				setAny = true
			}
			if dateStr, _ := cmd.Flags().GetString("date"); dateStr != "" {
				d, err := parseOpenAPIDate(dateStr, "date")
				if err != nil {
					return err
				}
				body.DateReported = d
				setAny = true
			}
			if note, _ := cmd.Flags().GetString("note"); note != "" {
				body.Note = &note
				setAny = true
			}
			if !setAny {
				return fmt.Errorf("at least one field to edit is required")
			}

			report, err := consumeAPIObject(app.FreeloClient.EditWorkReport(cmd.Context(), reportID, body))
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}

			out.OK(report, fmt.Sprintf("Report %d updated", reportID), nil)
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
			reportID, err := parseIntArg(args[0], "report-id")
			if err != nil {
				return err
			}

			if _, err := consumeAPIObject(app.FreeloClient.DeleteWorkReport(cmd.Context(), reportID)); err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}

			out.OK(map[string]any{"id": reportID, "deleted": true}, fmt.Sprintf("Report %d deleted", reportID), nil)
			return nil
		},
	}
}
