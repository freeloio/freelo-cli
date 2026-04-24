package commands

import (
	"fmt"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/spf13/cobra"
)

// NewEventsCmd creates the 'events' command group.
//
// Phase 3 migration: single read-only listing endpoint (/events) via
// app.FreeloClient.GetAllEvents with typed filter params.
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

			params := &freelo.GetAllEventsParams{}
			if projectID != 0 {
				ids := []int{projectID}
				params.ProjectsIds = &ids
			}
			if userID != 0 {
				ids := []int{userID}
				params.UsersIds = &ids
			}
			if page > 0 {
				p := freelo.PageParam(page)
				params.P = &p
			}

			body, err := consumeAPIBody(app.FreeloClient.GetAllEvents(cmd.Context(), params))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			events, _ := parsePaginatedItems(body)

			out.OK(events, fmt.Sprintf("%d events", len(events)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("user", 0, "Filter by user ID")
	cmd.Flags().Int("page", 0, "Page number (0-indexed)")
	return cmd
}
