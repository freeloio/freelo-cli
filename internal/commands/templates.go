package commands

import (
	"encoding/json"
	"fmt"

	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewTemplatesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "templates",
		Aliases: []string{"template"},
		Short:   "Manage templates",
	}

	cmd.AddCommand(
		newTemplatesListCmd(app),
		newTemplatesCreateProjectCmd(app),
		newTemplatesCreateTasklistCmd(app),
		newTemplatesCreateTaskCmd(app),
	)

	return cmd
}

func newTemplatesListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List template projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			result, err := app.Client.Get("/template-projects")
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			templates, _ := parsePaginatedItems(result)

			simplified := make([]map[string]any, 0, len(templates))
			for _, t := range templates {
				simplified = append(simplified, map[string]any{
					"id":   t["id"],
					"name": t["name"],
				})
			}

			out.OK(simplified, fmt.Sprintf("%d templates", len(simplified)), []output.Breadcrumb{
				{Action: "create", Cmd: "freelo templates create-project --template <id> --name <name>", Description: "Create project from template"},
			})
			return nil
		},
	}
}

func newTemplatesCreateProjectCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-project",
		Short: "Create a project from a template",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			templateID, _ := cmd.Flags().GetInt("template")
			name, _ := cmd.Flags().GetString("name")

			if templateID == 0 || name == "" {
				return fmt.Errorf("--template and --name are required")
			}

			body := map[string]any{"name": name}

			result, err := app.Client.Post(fmt.Sprintf("/project/create-from-template/%d", templateID), body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			var project map[string]any
			_ = json.Unmarshal(result, &project)
			out.OK(project, fmt.Sprintf("Project '%s' created from template", name), nil)
			return nil
		},
	}
	cmd.Flags().Int("template", 0, "Template project ID (required)")
	cmd.Flags().String("name", "", "New project name (required)")
	return cmd
}

func newTemplatesCreateTasklistCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-tasklist",
		Short: "Create a tasklist from a template",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			templateID, _ := cmd.Flags().GetInt("template")

			if templateID == 0 {
				return fmt.Errorf("--template is required")
			}

			result, err := app.Client.Post(fmt.Sprintf("/tasklist/create-from-template/%d", templateID), nil)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			var tasklist map[string]any
			_ = json.Unmarshal(result, &tasklist)
			out.OK(tasklist, "Tasklist created from template", nil)
			return nil
		},
	}
	cmd.Flags().Int("template", 0, "Template tasklist ID (required)")
	return cmd
}

func newTemplatesCreateTaskCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-task",
		Short: "Create a task from a template",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			templateID, _ := cmd.Flags().GetInt("template")

			if templateID == 0 {
				return fmt.Errorf("--template is required")
			}

			result, err := app.Client.Post(fmt.Sprintf("/task/create-from-template/%d", templateID), nil)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			var task map[string]any
			_ = json.Unmarshal(result, &task)
			out.OK(task, "Task created from template", nil)
			return nil
		},
	}
	cmd.Flags().Int("template", 0, "Template task ID (required)")
	return cmd
}
