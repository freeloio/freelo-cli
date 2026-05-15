package commands

import (
	"fmt"

	freelo "github.com/freeloio/freelo-go/freeloapi"
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

			body := freelo.CreateNoteJSONRequestBody{Name: name}
			if content != "" {
				body.Content = &content
			}

			note, err := consumeAPIObject(app.FreeloClient.CreateNote(cmd.Context(), projectID, body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

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
			noteID, err := parseIntArg(args[0], "note-id")
			if err != nil {
				return err
			}
			note, err := consumeAPIObject(app.FreeloClient.GetNote(cmd.Context(), noteID))
			if err != nil {
				out.Err(err, "not_found", "")
				return err
			}
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
		Short: "Edit a note (--name is required; EditNote spec demands a full name)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			noteID, err := parseIntArg(args[0], "note-id")
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")
			content, _ := cmd.Flags().GetString("content")

			if name == "" {
				// Spec marks `name` as required (non-pointer) on EditNoteJSONBody.
				// Old CLI sent partial bodies, which the server would have
				// rejected with a 400 anyway. Fail-fast with a clear CLI
				// message rather than a remote error round-trip.
				return fmt.Errorf("--name is required (Freelo EditNote expects the full title)")
			}

			body := freelo.EditNoteJSONRequestBody{Name: name}
			if content != "" {
				body.Content = &content
			}

			note, err := consumeAPIObject(app.FreeloClient.EditNote(cmd.Context(), noteID, body))
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}
			out.OK(note, fmt.Sprintf("Note %d updated", noteID), nil)
			return nil
		},
	}
	cmd.Flags().String("name", "", "New title (required)")
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
			noteID, err := parseIntArg(args[0], "note-id")
			if err != nil {
				return err
			}
			if _, err := consumeAPIObject(app.FreeloClient.DeleteNote(cmd.Context(), noteID)); err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"id": noteID, "deleted": true}, fmt.Sprintf("Note %d deleted", noteID), nil)
			return nil
		},
	}
}
