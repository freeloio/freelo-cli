package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/freeloapp/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewTasksCmd creates the 'tasks' command group.
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
			page, _ := cmd.Flags().GetInt("page")

			// Build query path
			var path string
			if tasklistID != 0 && projectID != 0 {
				path = fmt.Sprintf("/project/%d/tasklist/%d/tasks", projectID, tasklistID)
			} else {
				// Use /all-tasks with filters
				path = "/all-tasks"
				params := []string{}
				if search != "" {
					params = append(params, "search_query="+search)
				}
				if projectID != 0 {
					params = append(params, fmt.Sprintf("projects_ids[]=%d", projectID))
				}
				if workerID != 0 {
					params = append(params, fmt.Sprintf("worker_id=%d", workerID))
				}
				if state != "" {
					params = append(params, "state_id="+state)
				}
				if page > 0 {
					params = append(params, fmt.Sprintf("p=%d", page))
				}
				if len(params) > 0 {
					path += "?" + strings.Join(params, "&")
				}
			}

			result, err := app.Client.Get(path)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			tasks, paginated := parsePaginatedItems(result)

			// Simplify for display
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
	cmd.Flags().Int("worker", 0, "Filter by worker (user) ID")
	cmd.Flags().String("state", "", "Filter by state")
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
			taskID := args[0]

			result, err := app.Client.Get("/task/" + taskID)
			if err != nil {
				out.Err(err, "not_found", "Check task ID with 'freelo tasks list'")
				return err
			}

			var task map[string]any
			_ = json.Unmarshal(result, &task)

			name := ""
			if n, ok := task["name"].(string); ok {
				name = n
			}

			out.OK(task, name, []output.Breadcrumb{
				{Action: "edit", Cmd: fmt.Sprintf("freelo tasks edit %s --name <name>", taskID), Description: "Edit task"},
				{Action: "finish", Cmd: fmt.Sprintf("freelo tasks finish %s", taskID), Description: "Complete task"},
				{Action: "comment", Cmd: fmt.Sprintf("freelo comments create --task %s --content <text>", taskID), Description: "Add comment"},
				{Action: "description", Cmd: fmt.Sprintf("freelo tasks description %s", taskID), Description: "View description"},
				{Action: "track", Cmd: fmt.Sprintf("freelo tracking start --task %s", taskID), Description: "Start time tracking"},
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
			dueDate, _ := cmd.Flags().GetString("due-date")
			workerID, _ := cmd.Flags().GetInt("worker")
			priority, _ := cmd.Flags().GetString("priority")
			comment, _ := cmd.Flags().GetString("comment")

			if projectID == 0 || tasklistID == 0 || name == "" {
				return fmt.Errorf("--project, --tasklist, and --name are required")
			}

			body := map[string]any{
				"name": name,
			}
			if dueDate != "" {
				body["due_date"] = dueDate
			}
			if workerID != 0 {
				body["worker"] = map[string]any{"id": workerID}
			}
			if priority != "" {
				body["priority"] = priority
			}
			if comment != "" {
				body["comment"] = map[string]any{"content": comment}
			}

			path := fmt.Sprintf("/project/%d/tasklist/%d/tasks", projectID, tasklistID)
			result, err := app.Client.Post(path, body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			var task map[string]any
			_ = json.Unmarshal(result, &task)

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
			taskID := args[0]

			body := map[string]any{}
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				body["name"] = name
			}
			if dueDate, _ := cmd.Flags().GetString("due-date"); dueDate != "" {
				body["due_date"] = dueDate
			}
			if workerID, _ := cmd.Flags().GetInt("worker"); workerID != 0 {
				body["worker"] = map[string]any{"id": workerID}
			}
			if priority, _ := cmd.Flags().GetString("priority"); priority != "" {
				body["priority_enum"] = priority
			}

			if len(body) == 0 {
				return fmt.Errorf("at least one field to edit is required (--name, --due-date, --worker, --priority)")
			}

			result, err := app.Client.Post("/task/"+taskID, body)
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}

			var task map[string]any
			_ = json.Unmarshal(result, &task)

			out.OK(task, fmt.Sprintf("Task %s updated", taskID), nil)
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
			taskID := args[0]

			_, err := app.Client.Post("/task/"+taskID+"/finish", nil)
			if err != nil {
				out.Err(err, "finish_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(taskID), "finished": true}, fmt.Sprintf("Task %s finished", taskID), nil)
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
			taskID := args[0]

			_, err := app.Client.Post("/task/"+taskID+"/activate", nil)
			if err != nil {
				out.Err(err, "activate_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(taskID), "activated": true}, fmt.Sprintf("Task %s activated", taskID), nil)
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
			taskID := args[0]
			tasklistID, _ := cmd.Flags().GetInt("tasklist")

			if tasklistID == 0 {
				return fmt.Errorf("--tasklist is required")
			}

			path := fmt.Sprintf("/task/%s/move/%d", taskID, tasklistID)
			_, err := app.Client.Post(path, nil)
			if err != nil {
				out.Err(err, "move_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(taskID), "moved_to_tasklist": tasklistID}, fmt.Sprintf("Task %s moved to tasklist %d", taskID, tasklistID), nil)
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
			taskID := args[0]
			content, _ := cmd.Flags().GetString("set")

			if content != "" {
				// Set description
				_, err := app.Client.Post("/task/"+taskID+"/description", map[string]any{
					"content": content,
				})
				if err != nil {
					out.Err(err, "set_description_failed", "")
					return err
				}
				out.OK(map[string]any{"id": mustInt(taskID), "description_set": true}, "Description updated", nil)
				return nil
			}

			// Get description
			result, err := app.Client.Get("/task/" + taskID + "/description")
			if err != nil {
				out.Err(err, "get_description_failed", "")
				return err
			}

			var desc map[string]any
			_ = json.Unmarshal(result, &desc)
			out.OK(desc, "", nil)
			return nil
		},
	}
	cmd.Flags().String("set", "", "Set description content (HTML)")
	return cmd
}
