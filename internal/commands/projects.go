package commands

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/freeloapp/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewProjectsCmd creates the 'projects' command group.
func NewProjectsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project", "p"},
		Short:   "Manage projects",
	}

	cmd.AddCommand(
		newProjectsListCmd(app),
		newProjectsShowCmd(app),
		newProjectsCreateCmd(app),
		newProjectsArchiveCmd(app),
		newProjectsActivateCmd(app),
		newProjectsDeleteCmd(app),
	)

	return cmd
}

func newProjectsListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all active projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			result, err := app.Client.Get("/projects")
			if err != nil {
				out.Err(err, "api_error", "Check your authentication with 'freelo auth status'")
				return err
			}

			var projects []map[string]any
			if err := json.Unmarshal(result, &projects); err != nil {
				return err
			}

			// Simplify for display
			simplified := make([]map[string]any, 0, len(projects))
			for _, p := range projects {
				item := map[string]any{
					"id":   p["id"],
					"name": p["name"],
				}
				if v, ok := p["state"]; ok {
					item["state"] = v
				}
				if v, ok := p["currency_iso"]; ok {
					item["currency"] = v
				}
				if v, ok := p["date_add"]; ok {
					item["created"] = v
				}
				simplified = append(simplified, item)
			}

			out.OK(simplified, fmt.Sprintf("%d projects", len(simplified)), []output.Breadcrumb{
				{Action: "view", Cmd: "freelo projects show <id>", Description: "View project detail"},
				{Action: "create", Cmd: "freelo projects create --name <name>", Description: "Create new project"},
			})
			return nil
		},
	}
}

func newProjectsShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <project-id>",
		Short: "Show project detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID := args[0]

			result, err := app.Client.Get("/project/" + projectID)
			if err != nil {
				out.Err(err, "not_found", "Check the project ID with 'freelo projects list'")
				return err
			}

			var project map[string]any
			_ = json.Unmarshal(result, &project)

			id := projectID
			name := ""
			if n, ok := project["name"].(string); ok {
				name = n
			}

			out.OK(project, name, []output.Breadcrumb{
				{Action: "tasks", Cmd: fmt.Sprintf("freelo tasks list --project %s", id), Description: "List tasks in project"},
				{Action: "tasklists", Cmd: fmt.Sprintf("freelo tasklists list --project %s", id), Description: "List tasklists"},
			})
			return nil
		},
	}
}

func newProjectsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			name, _ := cmd.Flags().GetString("name")
			currency, _ := cmd.Flags().GetString("currency")

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			body := map[string]any{
				"name": name,
			}
			if currency != "" {
				body["currency_iso"] = currency
			}

			result, err := app.Client.Post("/projects", body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			var project map[string]any
			_ = json.Unmarshal(result, &project)

			id := ""
			if v, ok := project["id"]; ok {
				id = fmt.Sprintf("%v", v)
			}

			out.OK(project, fmt.Sprintf("Project '%s' created", name), []output.Breadcrumb{
				{Action: "view", Cmd: fmt.Sprintf("freelo projects show %s", id), Description: "View project"},
				{Action: "tasklist", Cmd: fmt.Sprintf("freelo tasklists create --project %s --name <name>", id), Description: "Create tasklist"},
			})
			return nil
		},
	}
	cmd.Flags().String("name", "", "Project name (required)")
	cmd.Flags().String("currency", "", "Currency ISO code (CZK, EUR, USD)")
	return cmd
}

func newProjectsArchiveCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "archive <project-id>",
		Short: "Archive a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID := args[0]

			_, err := app.Client.Post("/project/"+projectID+"/archive", nil)
			if err != nil {
				out.Err(err, "archive_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(projectID), "archived": true}, fmt.Sprintf("Project %s archived", projectID), nil)
			return nil
		},
	}
}

func newProjectsActivateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "activate <project-id>",
		Short: "Activate an archived project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID := args[0]

			_, err := app.Client.Post("/project/"+projectID+"/activate", nil)
			if err != nil {
				out.Err(err, "activate_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(projectID), "activated": true}, fmt.Sprintf("Project %s activated", projectID), nil)
			return nil
		},
	}
}

func newProjectsDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID := args[0]

			_, err := app.Client.Delete("/project/" + projectID)
			if err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(projectID), "deleted": true}, fmt.Sprintf("Project %s deleted", projectID), nil)
			return nil
		},
	}
}

func mustInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
