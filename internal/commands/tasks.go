package commands

import (
	"fmt"
	"strconv"
	"time"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewTasksCmd creates the 'tasks' command group.
//
// Phase 3 migration: all subcommands call the oapi-codegen-generated typed
// client via app.FreeloClient. We deliberately use the raw *http.Response
// methods (not the *WithResponse typed ones) and decode the body ourselves
// — see readRawBody in helpers.go for the rationale.
func NewTasksCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tasks",
		Aliases: []string{"task", "t"},
		Short:   "Manage tasks",
	}

	cmd.AddCommand(
		newTasksListCmd(app),
		newTasksShowCmd(app),
		newTasksCreateCmd(app),
		newTasksEditCmd(app),
		newTasksFinishCmd(app),
		newTasksActivateCmd(app),
		newTasksMoveCmd(app),
		newTasksDescriptionCmd(app),
	)

	return cmd
}

// parseDueDate turns a "YYYY-MM-DD" flag value into a *time.Time. Returns
// nil if the input is empty. Any parse error is surfaced so the user learns
// immediately rather than after the request fails server-side.
func parseDueDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid --due-date %q, want YYYY-MM-DD", s)
	}
	return &t, nil
}

func newTasksListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks (all or filtered by project/tasklist)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			projectID, _ := cmd.Flags().GetInt("project")
			tasklistID, _ := cmd.Flags().GetInt("tasklist")
			search, _ := cmd.Flags().GetString("search")
			workerID, _ := cmd.Flags().GetInt("worker")
			state, _ := cmd.Flags().GetString("state")

			ctx := cmd.Context()
			var body []byte
			var err error

			if tasklistID != 0 && projectID != 0 {
				body, err = consumeAPIBody(app.FreeloClient.GetTasksInTasklist(ctx, projectID, tasklistID, nil))
			} else {
				params := &freelo.GetAllTasksParams{}
				if search != "" {
					params.SearchQuery = &search
				}
				if projectID != 0 {
					ids := []int{projectID}
					params.ProjectsIds = &ids
				}
				// /all-tasks does not expose a worker filter in the OpenAPI spec.
				// The handwritten client used to pass worker_id= and the server
				// silently ignored it; we stay silent here too.
				_ = workerID
				if state != "" {
					if n, perr := strconv.Atoi(state); perr == nil {
						params.StateId = &n
					}
				}
				body, err = consumeAPIBody(app.FreeloClient.GetAllTasks(ctx, params))
			}

			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			tasks, paginated := parsePaginatedItems(body)

			simplified := make([]map[string]any, 0, len(tasks))
			for _, t := range tasks {
				item := map[string]any{
					"id":   t["id"],
					"name": t["name"],
				}
				if v, ok := t["state"]; ok {
					item["state"] = v
				}
				if v, ok := t["priority"]; ok && v != nil {
					item["priority"] = v
				}
				if v, ok := t["due_date"]; ok && v != nil {
					item["due_date"] = v
				}
				if worker, ok := t["worker"].(map[string]any); ok {
					if fn, ok := worker["fullname"].(string); ok {
						item["worker"] = fn
					}
				}
				simplified = append(simplified, item)
			}

			summary := fmt.Sprintf("%d tasks", len(simplified))
			if paginated != nil && paginated.Total > 0 {
				summary = fmt.Sprintf("%d tasks (total: %d)", len(simplified), paginated.Total)
			}

			out.OK(simplified, summary, []output.Breadcrumb{
				{Action: "view", Cmd: "freelo tasks show <id>", Description: "View task detail"},
				{Action: "create", Cmd: "freelo tasks create --project <id> --tasklist <id> --name <name>", Description: "Create task"},
				{Action: "finish", Cmd: "freelo tasks finish <id>", Description: "Complete task"},
			})
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("tasklist", 0, "Filter by tasklist ID (requires --project)")
	cmd.Flags().StringP("search", "s", "", "Search query")
	cmd.Flags().Int("worker", 0, "(unsupported by /all-tasks — see note in code) Filter by worker ID")
	cmd.Flags().String("state", "", "Filter by state ID (numeric)")
	cmd.Flags().Int("page", 0, "Page number (0-indexed)")
	return cmd
}

func newTasksShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show task detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID := mustInt(args[0])

			task, err := consumeAPIObject(app.FreeloClient.GetTask(cmd.Context(), taskID))
			if err != nil {
				out.Err(err, "not_found", "Check task ID with 'freelo tasks list'")
				return err
			}

			name := ""
			if n, ok := task["name"].(string); ok {
				name = n
			}

			out.OK(task, name, []output.Breadcrumb{
				{Action: "edit", Cmd: fmt.Sprintf("freelo tasks edit %d --name <name>", taskID), Description: "Edit task"},
				{Action: "finish", Cmd: fmt.Sprintf("freelo tasks finish %d", taskID), Description: "Complete task"},
				{Action: "comment", Cmd: fmt.Sprintf("freelo comments create --task %d --content <text>", taskID), Description: "Add comment"},
				{Action: "description", Cmd: fmt.Sprintf("freelo tasks description %d", taskID), Description: "View description"},
				{Action: "track", Cmd: fmt.Sprintf("freelo tracking start --task %d", taskID), Description: "Start time tracking"},
			})
			return nil
		},
	}
}

func newTasksCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			projectID, _ := cmd.Flags().GetInt("project")
			tasklistID, _ := cmd.Flags().GetInt("tasklist")
			name, _ := cmd.Flags().GetString("name")
			dueDateStr, _ := cmd.Flags().GetString("due-date")
			workerID, _ := cmd.Flags().GetInt("worker")
			priority, _ := cmd.Flags().GetString("priority")
			comment, _ := cmd.Flags().GetString("comment")

			if projectID == 0 || tasklistID == 0 || name == "" {
				return fmt.Errorf("--project, --tasklist, and --name are required")
			}

			dueDate, err := parseDueDate(dueDateStr)
			if err != nil {
				return err
			}

			body := freelo.TaskCreate{Name: name}
			if dueDate != nil {
				body.DueDate = dueDate
			}
			if workerID != 0 {
				body.Worker = &workerID
			}
			if priority != "" {
				p := freelo.TaskCreatePriorityEnum(priority)
				body.PriorityEnum = &p
			}
			if comment != "" {
				body.Comment = &struct {
					Content *string `json:"content,omitempty"`
				}{Content: &comment}
			}

			task, err := consumeAPIObject(app.FreeloClient.CreateTask(cmd.Context(), projectID, tasklistID, body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			id := ""
			if v, ok := task["id"]; ok {
				id = fmt.Sprintf("%v", v)
			}

			out.OK(task, fmt.Sprintf("Task '%s' created", name), []output.Breadcrumb{
				{Action: "view", Cmd: fmt.Sprintf("freelo tasks show %s", id), Description: "View task"},
			})
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().Int("tasklist", 0, "Tasklist ID (required)")
	cmd.Flags().String("name", "", "Task name (required)")
	cmd.Flags().String("due-date", "", "Due date (YYYY-MM-DD)")
	cmd.Flags().Int("worker", 0, "Assign to worker (user ID)")
	cmd.Flags().String("priority", "", "Priority: h (high), m (medium), l (low)")
	cmd.Flags().String("comment", "", "Initial comment text")
	return cmd
}

func newTasksEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <task-id>",
		Short: "Edit an existing task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID := mustInt(args[0])

			body := freelo.EditTaskJSONBody{}
			setAny := false
			if v, _ := cmd.Flags().GetString("name"); v != "" {
				body.Name = &v
				setAny = true
			}
			if v, _ := cmd.Flags().GetString("due-date"); v != "" {
				t, err := parseDueDate(v)
				if err != nil {
					return err
				}
				body.DueDate = t
				setAny = true
			}
			if v, _ := cmd.Flags().GetInt("worker"); v != 0 {
				body.Worker = &v
				setAny = true
			}
			if v, _ := cmd.Flags().GetString("priority"); v != "" {
				p := freelo.EditTaskJSONBodyPriorityEnum(v)
				body.PriorityEnum = &p
				setAny = true
			}
			if !setAny {
				return fmt.Errorf("at least one field to edit is required (--name, --due-date, --worker, --priority)")
			}

			task, err := consumeAPIObject(app.FreeloClient.EditTask(cmd.Context(), taskID, freelo.EditTaskJSONRequestBody(body)))
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}

			out.OK(task, fmt.Sprintf("Task %d updated", taskID), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "New task name")
	cmd.Flags().String("due-date", "", "New due date (YYYY-MM-DD)")
	cmd.Flags().Int("worker", 0, "Assign to worker (user ID)")
	cmd.Flags().String("priority", "", "Priority: h, m, l")
	return cmd
}

func newTasksFinishCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "finish <task-id>",
		Short: "Mark task as finished",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID := mustInt(args[0])

			if _, err := consumeAPIObject(app.FreeloClient.FinishTask(cmd.Context(), taskID)); err != nil {
				out.Err(err, "finish_failed", "")
				return err
			}

			out.OK(map[string]any{"id": taskID, "finished": true}, fmt.Sprintf("Task %d finished", taskID), nil)
			return nil
		},
	}
}

func newTasksActivateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "activate <task-id>",
		Short: "Reopen a finished task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID := mustInt(args[0])

			if _, err := consumeAPIObject(app.FreeloClient.ActivateTask(cmd.Context(), taskID)); err != nil {
				out.Err(err, "activate_failed", "")
				return err
			}

			out.OK(map[string]any{"id": taskID, "activated": true}, fmt.Sprintf("Task %d activated", taskID), nil)
			return nil
		},
	}
}

func newTasksMoveCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <task-id>",
		Short: "Move task to another tasklist",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID := mustInt(args[0])
			tasklistID, _ := cmd.Flags().GetInt("tasklist")

			if tasklistID == 0 {
				return fmt.Errorf("--tasklist is required")
			}

			_, err := consumeAPIObject(app.FreeloClient.MoveTask(cmd.Context(), taskID, tasklistID, freelo.MoveTaskJSONRequestBody{}))
			if err != nil {
				out.Err(err, "move_failed", "")
				return err
			}

			out.OK(map[string]any{"id": taskID, "moved_to_tasklist": tasklistID}, fmt.Sprintf("Task %d moved to tasklist %d", taskID, tasklistID), nil)
			return nil
		},
	}
	cmd.Flags().Int("tasklist", 0, "Target tasklist ID (required)")
	return cmd
}

func newTasksDescriptionCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "description <task-id>",
		Short: "Get or set task description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			taskID := mustInt(args[0])
			content, _ := cmd.Flags().GetString("set")

			if content != "" {
				body := freelo.EditTaskDescriptionJSONRequestBody{Content: content}
				if _, err := consumeAPIObject(app.FreeloClient.EditTaskDescription(cmd.Context(), taskID, body)); err != nil {
					out.Err(err, "set_description_failed", "")
					return err
				}
				out.OK(map[string]any{"id": taskID, "description_set": true}, "Description updated", nil)
				return nil
			}

			desc, err := consumeAPIObject(app.FreeloClient.GetTaskDescription(cmd.Context(), taskID))
			if err != nil {
				out.Err(err, "get_description_failed", "")
				return err
			}
			out.OK(desc, "", nil)
			return nil
		},
	}
	cmd.Flags().String("set", "", "Set description content (HTML)")
	return cmd
}
