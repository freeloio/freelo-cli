package commands

import (
	"encoding/json"
	"fmt"

	freelo "github.com/freeloio/freelo-go/freeloapi"
	"github.com/spf13/cobra"
)

func NewPinnedCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pinned",
		Aliases: []string{"pins"},
		Short:   "Manage pinned items in projects",
	}

	cmd.AddCommand(
		newPinnedListCmd(app),
		newPinnedCreateCmd(app),
		newPinnedDeleteCmd(app),
	)

	return cmd
}

func newPinnedListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pinned items in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			if projectID == 0 {
				return fmt.Errorf("--project is required")
			}
			body, err := consumeAPIBody(app.FreeloClient.GetPinnedItems(cmd.Context(), projectID))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			items := make([]map[string]any, 0)
			if err := json.Unmarshal(body, &items); err != nil {
				out.Err(err, "api_error", "")
				return fmt.Errorf("decode response: %w", err)
			}
			out.OK(items, fmt.Sprintf("%d pinned items", len(items)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	return cmd
}

func newPinnedCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Pin an item to a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			link, _ := cmd.Flags().GetString("link")
			title, _ := cmd.Flags().GetString("title")

			if projectID == 0 || link == "" {
				return fmt.Errorf("--project and --link are required")
			}

			body := freelo.PinItemToProjectJSONRequestBody{Link: link}
			if title != "" {
				body.Title = &title
			}

			item, err := consumeAPIObject(app.FreeloClient.PinItemToProject(cmd.Context(), projectID, body))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			out.OK(item, "Item pinned", nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("link", "", "URL to pin (required)")
	cmd.Flags().String("title", "", "Display title")
	return cmd
}

func newPinnedDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <pinned-item-id>",
		Short: "Remove a pinned item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			id, err := parseIntArg(args[0], "pinned-item-id")
			if err != nil {
				return err
			}
			if _, err := consumeAPIObject(app.FreeloClient.DeletePinnedItem(cmd.Context(), id)); err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"id": id, "deleted": true}, "Pinned item removed", nil)
			return nil
		},
	}
}
