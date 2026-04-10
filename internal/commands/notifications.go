package commands

import (
	"fmt"
	"strings"

	"github.com/freeloapp/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewNotificationsCmd creates the 'notifications' command group.
func NewNotificationsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "notifications",
		Aliases: []string{"notif", "n"},
		Short:   "Manage notifications",
	}

	cmd.AddCommand(
		newNotificationsListCmd(app),
		newNotificationsReadCmd(app),
		newNotificationsUnreadCmd(app),
	)

	return cmd
}

func newNotificationsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			unreadOnly, _ := cmd.Flags().GetBool("unread")
			page, _ := cmd.Flags().GetInt("page")

			path := "/all-notifications"
			params := []string{}
			if unreadOnly {
				params = append(params, "only_unread=1")
			}
			if page > 0 {
				params = append(params, fmt.Sprintf("p=%d", page))
			}
			if len(params) > 0 {
				path += "?" + strings.Join(params, "&")
			}

			result, err := app.Client.Get(path)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			notifications, _ := parsePaginatedItems(result)

			out.OK(notifications, fmt.Sprintf("%d notifications", len(notifications)), []output.Breadcrumb{
				{Action: "read", Cmd: "freelo notifications read <id>", Description: "Mark as read"},
			})
			return nil
		},
	}
	cmd.Flags().Bool("unread", false, "Show only unread notifications")
	cmd.Flags().Int("page", 0, "Page number")
	return cmd
}

func newNotificationsReadCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "read <notification-id>",
		Short: "Mark notification as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			notifID := args[0]

			_, err := app.Client.Post("/notification/"+notifID+"/mark-as-read", nil)
			if err != nil {
				out.Err(err, "mark_read_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(notifID), "read": true}, "Marked as read", nil)
			return nil
		},
	}
}

func newNotificationsUnreadCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "unread <notification-id>",
		Short: "Mark notification as unread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			notifID := args[0]

			_, err := app.Client.Post("/notification/"+notifID+"/mark-as-unread", nil)
			if err != nil {
				out.Err(err, "mark_unread_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(notifID), "unread": true}, "Marked as unread", nil)
			return nil
		},
	}
}
