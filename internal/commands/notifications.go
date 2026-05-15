package commands

import (
	"fmt"

	freelo "github.com/freeloio/freelo-go/freeloapi"
	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewNotificationsCmd creates the 'notifications' command group.
//
// Phase 3 migration: uses app.FreeloClient.
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

			params := &freelo.GetAllNotificationsParams{}
			if unreadOnly {
				t := true
				params.OnlyUnread = &t
			}
			if err := setPageFilter(cmd, &params.P); err != nil {
				return err
			}

			body, err := consumeAPIBody(app.FreeloClient.GetAllNotifications(cmd.Context(), params))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			notifications, _ := parsePaginatedItems(body)

			out.OK(notifications, fmt.Sprintf("%d notifications", len(notifications)), []output.Breadcrumb{
				{Action: "read", Cmd: "freelo notifications read <id>", Description: "Mark as read"},
			})
			return nil
		},
	}
	cmd.Flags().Bool("unread", false, "Show only unread notifications")
	cmd.Flags().Int("page", 0, "Page number (>= 1; omit for first page)")
	return cmd
}

func newNotificationsReadCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "read <notification-id>",
		Short: "Mark notification as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			notifID, err := parseIntArg(args[0], "notification-id")
			if err != nil {
				return err
			}

			if _, err := consumeAPIObject(app.FreeloClient.MarkNotificationAsRead(cmd.Context(), notifID)); err != nil {
				out.Err(err, "read_failed", "")
				return err
			}

			out.OK(map[string]any{"id": notifID, "read": true}, "Marked as read", nil)
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
			notifID, err := parseIntArg(args[0], "notification-id")
			if err != nil {
				return err
			}

			if _, err := consumeAPIObject(app.FreeloClient.MarkNotificationAsUnread(cmd.Context(), notifID)); err != nil {
				out.Err(err, "unread_failed", "")
				return err
			}

			out.OK(map[string]any{"id": notifID, "unread": true}, "Marked as unread", nil)
			return nil
		},
	}
}
