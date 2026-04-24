package commands

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewProjectsCmd creates the 'projects' command group.
//
// Phase 3 migration: subcommands use app.FreeloClient. See tasks.go for
// the overall approach (raw *http.Response methods, not *WithResponse).
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

			body, err := consumeAPIBody(app.FreeloClient.GetProjects(cmd.Context(), nil))
			if err != nil {
				out.Err(err, "api_error", "Check your authentication with 'freelo auth status'")
				return err
			}

			var projects []map[string]any
			if err := json.Unmarshal(body, &projects); err != nil {
				return err
			}

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
			projectID := mustInt(args[0])

			project, err := consumeAPIObject(app.FreeloClient.GetProject(cmd.Context(), projectID))
			if err != nil {
				out.Err(err, "not_found", "Check the project ID with 'freelo projects list'")
				return err
			}

			name := ""
			if n, ok := project["name"].(string); ok {
				name = n
			}

			out.OK(project, name, []output.Breadcrumb{
				{Action: "tasks", Cmd: fmt.Sprintf("freelo tasks list --project %d", projectID), Description: "List tasks in project"},
				{Action: "tasklists", Cmd: fmt.Sprintf("freelo tasklists list --project %d", projectID), Description: "List tasklists"},
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
			if currency == "" {
				// The OpenAPI spec marks currency_iso as required. Freelo's
				// default for Czech users is CZK; document the default rather
				// than send an empty string the server would reject.
				currency = "CZK"
			}

			body := freelo.CreateProjectJSONRequestBody{
				Name:        name,
				CurrencyIso: freelo.CreateProjectJSONBodyCurrencyIso(currency),
			}

			project, err := consumeAPIObject(app.FreeloClient.CreateProject(cmd.Context(), body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

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
	cmd.Flags().String("currency", "", "Currency ISO code (defaults to CZK if not set)")
	return cmd
}

func newProjectsArchiveCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "archive <project-id>",
		Short: "Archive a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID := mustInt(args[0])

			if _, err := consumeAPIObject(app.FreeloClient.ArchiveProject(cmd.Context(), projectID)); err != nil {
				out.Err(err, "archive_failed", "")
				return err
			}

			out.OK(map[string]any{"id": projectID, "archived": true}, fmt.Sprintf("Project %d archived", projectID), nil)
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
			projectID := mustInt(args[0])

			if _, err := consumeAPIObject(app.FreeloClient.ActivateProject(cmd.Context(), projectID)); err != nil {
				out.Err(err, "activate_failed", "")
				return err
			}

			out.OK(map[string]any{"id": projectID, "activated": true}, fmt.Sprintf("Project %d activated", projectID), nil)
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
			projectID := mustInt(args[0])

			if _, err := consumeAPIObject(app.FreeloClient.DeleteProject(cmd.Context(), projectID)); err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}

			out.OK(map[string]any{"id": projectID, "deleted": true}, fmt.Sprintf("Project %d deleted", projectID), nil)
			return nil
		},
	}
}

// mustInt is shared with other command groups; kept here because this is
// where it was originally defined.
func mustInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
