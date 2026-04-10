package commands

import (
	"encoding/json"
	"fmt"
	"strings"

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

			result, err := app.Client.Get(fmt.Sprintf("/project/%d/workers", projectID))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			workers, _ := parsePaginatedItems(result)

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

func newWorkersInviteCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invite",
		Short: "Invite users to projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			emails, _ := cmd.Flags().GetString("emails")
			projectIDs, _ := cmd.Flags().GetString("projects")

			if emails == "" || projectIDs == "" {
				return fmt.Errorf("--emails and --projects are required")
			}

			emailList := strings.Split(emails, ",")
			for i := range emailList {
				emailList[i] = strings.TrimSpace(emailList[i])
			}

			projectList := strings.Split(projectIDs, ",")
			projectInts := make([]int, 0, len(projectList))
			for _, p := range projectList {
				projectInts = append(projectInts, mustInt(strings.TrimSpace(p)))
			}

			body := map[string]any{
				"emails":       emailList,
				"projects_ids": projectInts,
			}

			result, err := app.Client.Post("/users/manage-workers", body)
			if err != nil {
				out.Err(err, "invite_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, fmt.Sprintf("Invited %d users to %d projects", len(emailList), len(projectInts)), nil)
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
			emails, _ := cmd.Flags().GetString("emails")

			if projectID == 0 || emails == "" {
				return fmt.Errorf("--project and --emails are required")
			}

			emailList := strings.Split(emails, ",")
			for i := range emailList {
				emailList[i] = strings.TrimSpace(emailList[i])
			}

			result, err := app.Client.Post(fmt.Sprintf("/project/%d/remove-workers/by-emails", projectID), map[string]any{
				"emails": emailList,
			})
			if err != nil {
				out.Err(err, "remove_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, fmt.Sprintf("Workers removed from project %d", projectID), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Project ID (required)")
	cmd.Flags().String("emails", "", "Comma-separated emails (required)")
	return cmd
}
