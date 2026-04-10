package commands

import (
	"encoding/json"
	"fmt"

	"github.com/freeloapp/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewTasklistsCmd creates the 'tasklists' command group.
func NewTasklistsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tasklists",
		Aliases: []string{"tasklist", "tl"},
		Short:   "Manage tasklists",
	}

	cmd.AddCommand(
		newTasklistsListCmd(app),
		newTasklistsShowCmd(app),
		newTasklistsCreateCmd(app),
	)

	return cmd
}

func newTasklistsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasklists",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")

			var path string
			if projectID != 0 {
				path = fmt.Sprintf("/project/%d", projectID)
				// Project detail includes tasklists
				result, err := app.Client.Get(path)
				if err != nil {
					out.Err(err, "api_error", "")
					return err
				}

				var project map[string]any
				_ = json.Unmarshal(result, &project)

				tasklists := []map[string]any{}
				if tls, ok := project["tasklists"].([]any); ok {
					for _, tl := range tls {
						if tlMap, ok := tl.(map[string]any); ok {
							tasklists = append(tasklists, map[string]any{
								"id":   tlMap["id"],
								"name": tlMap["name"],
							})
						}
					}
				}

				out.OK(tasklists, fmt.Sprintf("%d tasklists", len(tasklists)), []output.Breadcrumb{
					{Action: "view", Cmd: "freelo tasklists show <id>", Description: "View tasklist detail"},
					{Action: "tasks", Cmd: fmt.Sprintf("freelo tasks list --project %d --tasklist <id>", projectID), Description: "List tasks in tasklist"},
				})
				return nil
			}

			// All tasklists
			result, err := app.Client.Get("/all-tasklists")
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			tasklists, _ := parsePaginatedItems(result)

			simplified := make([]map[string]any, 0, len(tasklists))
			for _, tl := range tasklists {
				simplified = append(simplified, map[string]any{
					"id":   tl["id"],
					"name": tl["name"],
				})
			}

			out.OK(simplified, fmt.Sprintf("%d tasklists", len(simplified)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	return cmd
}

func newTasklistsShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <tasklist-id>",
		Short: "Show tasklist detail with tasks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			tasklistID := args[0]

			result, err := app.Client.Get("/tasklist/" + tasklistID)
			if err != nil {
				out.Err(err, "not_found", "")
				return err
			}

			var tasklist map[string]any
			_ = json.Unmarshal(result, &tasklist)

			name := ""
			if n, ok := tasklist["name"].(string); ok {
				name = n
			}

			out.OK(tasklist, name, nil)
			return nil
		},
	}
}

func newTasklistsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new tasklist in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			projectID, _ := cmd.Flags().GetInt("project")
			name, _ := cmd.Flags().GetString("name")

			if projectID == 0 || name == "" {
				return fmt.Errorf("--project and --name are required")
			}

			body := map[string]any{"name": name}

			if budget, _ := cmd.Flags().GetString("budget"); budget != "" {
				body["budget"] = budget
			}

			path := fmt.Sprintf("/project/%d/tasklists", projectID)
			result, err := app.Client.Post(path, body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			var tasklist map[string]any
			_ = json.Unmarshal(result, &tasklist)

			out.OK(tasklist, fmt.Sprintf("Tasklist '%s' created", name), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("name", "", "Tasklist name (required)")
	cmd.Flags().String("budget", "", "Budget amount")
	return cmd
}
