package commands

import (
	"fmt"

	"github.com/freeloio/freelo-cli/internal/output"
	freelo "github.com/freeloio/freelo-go/freeloapi"
	"github.com/spf13/cobra"
)

// NewTasklistsCmd creates the 'tasklists' command group.
//
// Phase 3 migration: all subcommands use app.FreeloClient via the raw
// *http.Response methods.
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

			simplified := make([]map[string]any, 0)
			crumbs := []output.Breadcrumb{
				{Action: "view", Cmd: "freelo tasklists show <id>", Description: "View tasklist detail"},
			}

			if projectID != 0 {
				// Use /project/{id} rather than /all-tasklists?projects_ids[]=
				// because onboarding/demo projects (e.g. 580898) don't appear
				// in the /all-* listings — same quirk documented in the skill.
				// /project/{id} returns them reliably via its `tasklists` field.
				project, err := consumeAPIObject(app.FreeloClient.GetProject(cmd.Context(), projectID))
				if err != nil {
					out.Err(err, "api_error", "")
					return err
				}
				// The server has historically returned `tasklists` as a JSON
				// array. If a future schema swap (e.g. to a paginated object)
				// makes this assertion fail, we'd silently return zero items
				// — surface that as a clear error instead of an empty success.
				if raw, present := project["tasklists"]; present {
					tls, ok := raw.([]any)
					if !ok {
						out.Err(fmt.Errorf("unexpected shape for project.tasklists: %T (server may have changed schema)", raw), "api_error", "")
						return fmt.Errorf("unexpected tasklists shape: %T", raw)
					}
					for _, tl := range tls {
						if m, ok := tl.(map[string]any); ok {
							simplified = append(simplified, map[string]any{
								"id":   m["id"],
								"name": m["name"],
							})
						}
					}
				}
				crumbs = append(crumbs, output.Breadcrumb{
					Action:      "tasks",
					Cmd:         fmt.Sprintf("freelo tasks list --project %d --tasklist <id>", projectID),
					Description: "List tasks in tasklist",
				})
			} else {
				body, err := consumeAPIBody(app.FreeloClient.GetAllTasklists(cmd.Context(), &freelo.GetAllTasklistsParams{}))
				if err != nil {
					out.Err(err, "api_error", "")
					return err
				}
				tasklists, _ := parsePaginatedItems(body)
				for _, tl := range tasklists {
					simplified = append(simplified, map[string]any{
						"id":   tl["id"],
						"name": tl["name"],
					})
				}
			}

			out.OK(simplified, fmt.Sprintf("%d tasklists", len(simplified)), crumbs)
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
			tasklistID, err := parseIntArg(args[0], "tasklist-id")
			if err != nil {
				return err
			}

			tasklist, err := consumeAPIObject(app.FreeloClient.GetTasklist(cmd.Context(), tasklistID))
			if err != nil {
				out.Err(err, "not_found", "")
				return err
			}

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

			body := freelo.CreateTasklistJSONRequestBody{Name: name}
			if budget, _ := cmd.Flags().GetString("budget"); budget != "" {
				body.Budget = &budget
			}

			tasklist, err := consumeAPIObject(app.FreeloClient.CreateTasklist(cmd.Context(), projectID, body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			out.OK(tasklist, fmt.Sprintf("Tasklist '%s' created", name), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("name", "", "Tasklist name (required)")
	cmd.Flags().String("budget", "", "Budget amount (2 decimal places, no separator)")
	return cmd
}
