package commands

import (
	"encoding/json"
	"fmt"

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

			result, err := app.Client.Get(fmt.Sprintf("/project/%d/pinned-items", projectID))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			var items []map[string]any
			_ = json.Unmarshal(result, &items)
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

			body := map[string]any{"link": link}
			if title != "" {
				body["title"] = title
			}

			result, err := app.Client.Post(fmt.Sprintf("/project/%d/pinned-items", projectID), body)
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}
			var item map[string]any
			_ = json.Unmarshal(result, &item)
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
			_, err := app.Client.Delete("/pinned-item/" + args[0])
			if err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}
			out.OK(map[string]any{"id": mustInt(args[0]), "deleted": true}, "Pinned item removed", nil)
			return nil
		},
	}
}
