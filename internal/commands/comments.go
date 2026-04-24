package commands

import (
	"fmt"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewCommentsCmd creates the 'comments' command group.
//
// Same dead-command situation as subtasks: the pre-v1.0.0 CLI exposed
// `delete` calling DELETE /comment/{id}, but the server returns 404 for
// that verb — probed live and the spec only documents POST /comment/{id}
// (edit). The Freelo API has no comment deletion. Dropped before v1.0.0
// to avoid shipping a command that can never succeed.
func NewCommentsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "comments",
		Aliases: []string{"comment"},
		Short:   "Manage comments (no delete — Freelo API doesn't support it)",
	}

	cmd.AddCommand(
		newCommentsListCmd(app),
		newCommentsCreateCmd(app),
		newCommentsEditCmd(app),
	)

	return cmd
}

func newCommentsListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all comments (filterable)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			page, _ := cmd.Flags().GetInt("page")

			params := &freelo.GetAllCommentsParams{}
			if projectID != 0 {
				ids := []int{projectID}
				params.ProjectsIds = &ids
			}
			if page > 0 {
				p := freelo.PageParam(page)
				params.P = &p
			}

			body, err := consumeAPIBody(app.FreeloClient.GetAllComments(cmd.Context(), params))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			comments, _ := parsePaginatedItems(body)

			simplified := make([]map[string]any, 0, len(comments))
			for _, c := range comments {
				item := map[string]any{
					"id":      c["id"],
					"content": c["content"],
				}
				if author, ok := c["author"].(map[string]any); ok {
					item["author"] = author["fullname"]
				}
				if v, ok := c["date_add"]; ok {
					item["created"] = v
				}
				simplified = append(simplified, item)
			}

			out.OK(simplified, fmt.Sprintf("%d comments", len(simplified)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().Int("page", 0, "Page number (0-indexed)")
	return cmd
}

func newCommentsCreateCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Add a comment to a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			taskID, _ := cmd.Flags().GetInt("task")
			content, _ := cmd.Flags().GetString("content")

			if taskID == 0 || content == "" {
				return fmt.Errorf("--task and --content are required")
			}

			comment, err := consumeAPIObject(app.FreeloClient.CreateComment(cmd.Context(), taskID, freelo.CreateCommentJSONRequestBody{Content: content}))
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			out.OK(comment, "Comment added", []output.Breadcrumb{
				{Action: "view", Cmd: fmt.Sprintf("freelo tasks show %d", taskID), Description: "View task"},
			})
			return nil
		},
	}
	cmd.Flags().Int("task", 0, "Task ID (required)")
	cmd.Flags().String("content", "", "Comment content (required)")
	return cmd
}

func newCommentsEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <comment-id>",
		Short: "Edit a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			commentID := mustInt(args[0])
			content, _ := cmd.Flags().GetString("content")

			if content == "" {
				return fmt.Errorf("--content is required")
			}

			comment, err := consumeAPIObject(app.FreeloClient.EditComment(cmd.Context(), commentID, freelo.EditCommentJSONRequestBody{Content: content}))
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}

			out.OK(comment, fmt.Sprintf("Comment %d updated", commentID), nil)
			return nil
		},
	}
	cmd.Flags().String("content", "", "New content (required)")
	return cmd
}
