package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/freeloapp/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewCommentsCmd creates the 'comments' command group.
func NewCommentsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "comments",
		Aliases: []string{"comment"},
		Short:   "Manage comments",
	}

	cmd.AddCommand(
		newCommentsListCmd(app),
		newCommentsCreateCmd(app),
		newCommentsEditCmd(app),
		newCommentsDeleteCmd(app),
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

			path := "/all-comments"
			params := []string{}
			if projectID != 0 {
				params = append(params, fmt.Sprintf("projects_ids[]=%d", projectID))
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

			comments, _ := parsePaginatedItems(result)

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
	cmd.Flags().Int("page", 0, "Page number")
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

			path := fmt.Sprintf("/task/%d/comments", taskID)
			result, err := app.Client.Post(path, map[string]any{
				"content": content,
			})
			if err != nil {
				out.Err(err, "create_failed", "")
				return err
			}

			var comment map[string]any
			_ = json.Unmarshal(result, &comment)

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
			commentID := args[0]
			content, _ := cmd.Flags().GetString("content")

			if content == "" {
				return fmt.Errorf("--content is required")
			}

			result, err := app.Client.Post("/comment/"+commentID, map[string]any{
				"content": content,
			})
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}

			var comment map[string]any
			_ = json.Unmarshal(result, &comment)

			out.OK(comment, fmt.Sprintf("Comment %s updated", commentID), nil)
			return nil
		},
	}
	cmd.Flags().String("content", "", "New content (required)")
	return cmd
}

func newCommentsDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <comment-id>",
		Short: "Delete a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			commentID := args[0]

			_, err := app.Client.Delete("/comment/" + commentID)
			if err != nil {
				out.Err(err, "delete_failed", "")
				return err
			}

			out.OK(map[string]any{"id": mustInt(commentID), "deleted": true}, fmt.Sprintf("Comment %s deleted", commentID), nil)
			return nil
		},
	}
}
