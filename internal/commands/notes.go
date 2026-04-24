package commands

import (
	"encoding/json"
	"fmt"

	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewNotesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "notes",
		Aliases: []string{"note"},
		Short:   "Manage project notes",
	}

	cmd.AddCommand(
		newNotesCreateCmd(app),
		newNotesShowCmd(app),
		newNotesEditCmd(app),
		newNotesDeleteCmd(app),
	)

	return cmd
}

func newNotesCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a note in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			name, _ := cmd.Flags().GetString("name")
			content, _ := cmd.Flags().GetString("content")

			if projectID == 0 || name == "" {
				return fmt.Errorf("--project and --name are required")
			}

			body := map[string]any{"name": name}
			if content != "" {
				body["content"] = content
			}

			result, err := app.Client.Post(fmt.Sprintf("/project/%d/note", projectID), body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			var note map[string]any
			_ = json.Unmarshal(result, &note)

			id := ""
			if v, ok := note["id"]; ok {
				id = fmt.Sprintf("%v", v)
			}

			out.OK(note, fmt.Sprintf("Note '%s' created", name), []output.Breadcrumb{
				{Action: "view", Cmd: fmt.Sprintf("freelo notes show %s", id), Description: "View note"},
			})
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("name", "", "Note title (required)")
	cmd.Flags().String("content", "", "Note content (HTML)")
	return cmd
}

func newNotesShowCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <note-id>",
		Short: "Show note detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			result, err := app.Client.Get("/note/" + args[0])
			if err != nil {
				out.Err(err, "not_found", "")
				return err
			}
			var note map[string]any
			_ = json.Unmarshal(result, &note)

			name := ""
			if n, ok := note["name"].(string); ok {
				name = n
			}
			out.OK(note, name, nil)
			return nil
		},
	}
}

func newNotesEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <note-id>",
		Short: "Edit a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			noteID := args[0]

			body := map[string]any{}
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				body["name"] = name
			}
			if content, _ := cmd.Flags().GetString("content"); content != "" {
				body["content"] = content
			}
			if len(body) == 0 {
				return fmt.Errorf("at least --name or --content is required")
			}

			result, err := app.Client.Post("/note/"+noteID, body)
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}
			var note map[string]any
			_ = json.Unmarshal(result, &note)
			out.OK(note, fmt.Sprintf("Note %s updated", noteID), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "New title")
	cmd.Flags().String("content", "", "New content (HTML)")
	return cmd
}

func newNotesDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <note-id>",
		Short: "Delete a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			_, err := app.Client.Delete("/note/" + args[0])
			if err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"id": mustInt(args[0]), "deleted": true}, fmt.Sprintf("Note %s deleted", args[0]), nil)
			return nil
		},
	}
}
