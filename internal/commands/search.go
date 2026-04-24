package commands

import (
	"fmt"
	"strings"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewSearchCmd creates the 'search' command.
//
// Phase 3 migration: POST /search via app.FreeloClient.
func NewSearchCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search across Freelo (tasks, projects, comments, files)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			query := strings.Join(args, " ")
			entityType, _ := cmd.Flags().GetString("type")
			projectID, _ := cmd.Flags().GetInt("project")
			page, _ := cmd.Flags().GetInt("page")

			body := freelo.SearchJSONRequestBody{SearchQuery: query}
			if entityType != "" {
				et := freelo.SearchJSONBodyEntityType(entityType)
				body.EntityType = &et
			}
			if projectID != 0 {
				ids := []int{projectID}
				body.ProjectsIds = &ids
			}
			if page > 0 {
				body.Page = &page
			}

			respBody, err := consumeAPIBody(app.FreeloClient.Search(cmd.Context(), body))
			if err != nil {
				out.Err(err, "search_failed", "")
				return err
			}

			items, paginated := parsePaginatedItems(respBody)

			simplified := make([]map[string]any, 0, len(items))
			for _, m := range items {
				item := map[string]any{
					"id":   m["id"],
					"name": m["name"],
					"type": m["type"],
				}
				if v, ok := m["state"]; ok {
					item["state"] = v
				}
				simplified = append(simplified, item)
			}

			total := ""
			if paginated != nil && paginated.Total > 0 {
				total = fmt.Sprintf(" (total: %d)", paginated.Total)
			}

			out.OK(simplified, fmt.Sprintf("%d results for '%s'%s", len(simplified), query, total), []output.Breadcrumb{
				{Action: "view", Cmd: "freelo tasks show <id>", Description: "View task result"},
				{Action: "view", Cmd: "freelo projects show <id>", Description: "View project result"},
			})
			return nil
		},
	}
	cmd.Flags().StringP("type", "t", "", "Filter by type: task, subtask, project, tasklist, file, comment")
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("page", 0, "Page number (0-indexed)")
	return cmd
}
