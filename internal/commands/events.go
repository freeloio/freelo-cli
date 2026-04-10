package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func NewEventsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "events",
		Aliases: []string{"activity"},
		Short:   "View activity events",
	}

	cmd.AddCommand(newEventsListCmd(app))
	return cmd
}

func newEventsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activity events",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			userID, _ := cmd.Flags().GetInt("user")
			page, _ := cmd.Flags().GetInt("page")

			path := "/events"
			params := []string{}
			if projectID != 0 {
				params = append(params, fmt.Sprintf("projects_ids[]=%d", projectID))
			}
			if userID != 0 {
				params = append(params, fmt.Sprintf("users_ids[]=%d", userID))
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

			events, _ := parsePaginatedItems(result)

			out.OK(events, fmt.Sprintf("%d events", len(events)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("user", 0, "Filter by user ID")
	cmd.Flags().Int("page", 0, "Page number")
	return cmd
}
