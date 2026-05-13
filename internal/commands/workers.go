package commands

import (
	"fmt"
	"strings"

	freelo "github.com/freeloio/freelo-go/freeloapi"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/spf13/cobra"
)

func NewWorkersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workers",
		Aliases: []string{"members"},
		Short:   "Manage project workers/members",
	}

	cmd.AddCommand(
		newWorkersListCmd(app),
		newWorkersInviteCmd(app),
		newWorkersRemoveCmd(app),
	)

	return cmd
}

func newWorkersListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workers in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			if projectID == 0 {
				return fmt.Errorf("--project is required")
			}

			body, err := consumeAPIBody(app.FreeloClient.GetProjectWorkers(cmd.Context(), projectID, &freelo.GetProjectWorkersParams{}))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			workers, _ := parsePaginatedItems(body)

			simplified := make([]map[string]any, 0, len(workers))
			for _, w := range workers {
				simplified = append(simplified, map[string]any{
					"id":       w["id"],
					"fullname": w["fullname"],
					"email":    w["email"],
				})
			}

			out.OK(simplified, fmt.Sprintf("%d workers", len(simplified)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	return cmd
}

// parseEmails splits a comma-separated list and wraps each into the typed
// openapi_types.Email. Empty tokens are skipped.
func parseEmails(s string) []openapi_types.Email {
	parts := strings.Split(s, ",")
	out := make([]openapi_types.Email, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, openapi_types.Email(p))
		}
	}
	return out
}

func newWorkersInviteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invite",
		Short: "Invite users to projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			emailsStr, _ := cmd.Flags().GetString("emails")
			projectsStr, _ := cmd.Flags().GetString("projects")

			if emailsStr == "" || projectsStr == "" {
				return fmt.Errorf("--emails and --projects are required")
			}

			emails := parseEmails(emailsStr)

			projectList := strings.Split(projectsStr, ",")
			projectInts := make([]int, 0, len(projectList))
			for _, p := range projectList {
				token := strings.TrimSpace(p)
				if token == "" {
					continue
				}
				v, perr := parseIntArg(token, "--projects entry")
				if perr != nil {
					return perr
				}
				projectInts = append(projectInts, v)
			}

			body := freelo.InviteUsersToProjectsJSONRequestBody{
				Emails:      &emails,
				ProjectsIds: projectInts,
			}

			resp, err := consumeAPIObject(app.FreeloClient.InviteUsersToProjects(cmd.Context(), body))
			if err != nil {
				out.Err(err, "invite_failed", "")
				return err
			}
			out.OK(resp, fmt.Sprintf("Invited %d users to %d projects", len(emails), len(projectInts)), nil)
			return nil
		},
	}
	cmd.Flags().String("emails", "", "Comma-separated emails (required)")
	cmd.Flags().String("projects", "", "Comma-separated project IDs (required)")
	return cmd
}

func newWorkersRemoveCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove workers from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			emailsStr, _ := cmd.Flags().GetString("emails")

			if projectID == 0 || emailsStr == "" {
				return fmt.Errorf("--project and --emails are required")
			}

			body := freelo.RemoveProjectWorkersByEmailsJSONRequestBody{
				UsersEmails: parseEmails(emailsStr),
			}

			resp, err := consumeAPIObject(app.FreeloClient.RemoveProjectWorkersByEmails(cmd.Context(), projectID, body))
			if err != nil {
				out.Err(err, "remove_failed", "")
				return err
			}
			out.OK(resp, fmt.Sprintf("Workers removed from project %d", projectID), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("emails", "", "Comma-separated emails (required)")
	return cmd
}
