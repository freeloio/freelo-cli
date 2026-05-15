package commands

import (
	"bytes"
	"encoding/json"
	"fmt"

	freelo "github.com/freeloio/freelo-go/freeloapi"
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

			params := &freelo.GetAllCommentsParams{}
			setProjectsFilter(&params.ProjectsIds, projectID)
			if err := setPageFilter(cmd, &params.P); err != nil {
				return err
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
	cmd.Flags().Int("page", 0, "Page number (>= 1; omit for first page)")
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
			fileUUIDs, _ := cmd.Flags().GetStringArray("file")

			if taskID == 0 || content == "" {
				return fmt.Errorf("--task and --content are required")
			}

			comment, err := postCommentWithFiles(cmd, app, taskID, 0, content, fileUUIDs)
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
	cmd.Flags().StringArray("file", nil, "Attach an uploaded file by UUID (repeat for multiple: --file <uuid> --file <uuid>)")
	return cmd
}

func newCommentsEditCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <comment-id>",
		Short: "Edit a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			commentID, err := parseIntArg(args[0], "comment-id")
			if err != nil {
				return err
			}
			content, _ := cmd.Flags().GetString("content")
			fileUUIDs, _ := cmd.Flags().GetStringArray("file")

			if content == "" {
				return fmt.Errorf("--content is required")
			}

			comment, err := postCommentWithFiles(cmd, app, 0, commentID, content, fileUUIDs)
			if err != nil {
				out.Err(err, "edit_failed", "")
				return err
			}

			out.OK(comment, fmt.Sprintf("Comment %d updated", commentID), nil)
			return nil
		},
	}
	cmd.Flags().String("content", "", "New content (required)")
	cmd.Flags().StringArray("file", nil, "Attach an uploaded file by UUID (repeat for multiple). Files passed here REPLACE any previous attachments — pass all the file UUIDs you want to keep.")
	return cmd
}

// postCommentWithFiles routes a comment create or edit through the typed
// client when no files are attached, and through the *WithBody escape hatch
// (with a hand-crafted JSON body) when there are. Reason: the OpenAPI spec
// models attachments as FileUpload{download_url, filename}, but the live
// server only honors the undocumented {"uuid": "<uuid>"} shape for files
// already uploaded via /file/upload. The typed client can't emit that
// shape because the spec doesn't describe it.
//
// Pass taskID > 0 for create; pass commentID > 0 for edit. Exactly one of
// the two should be set.
func postCommentWithFiles(cmd *cobra.Command, app *App, taskID, commentID int, content string, fileUUIDs []string) (map[string]any, error) {
	ctx := cmd.Context()

	if len(fileUUIDs) == 0 {
		// Typed path — preserves spec coverage when no files involved.
		if taskID != 0 {
			return consumeAPIObject(app.FreeloClient.CreateComment(ctx, taskID, freelo.CreateCommentJSONRequestBody{Content: content}))
		}
		return consumeAPIObject(app.FreeloClient.EditComment(ctx, commentID, freelo.EditCommentJSONRequestBody{Content: content}))
	}

	files, err := validateAndWrapFileUUIDs(fileUUIDs)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		"content": content,
		"files":   files,
	})
	if err != nil {
		return nil, err
	}

	if taskID != 0 {
		return consumeAPIObject(app.FreeloClient.CreateCommentWithBody(ctx, taskID, "application/json", bytes.NewReader(payload)))
	}
	return consumeAPIObject(app.FreeloClient.EditCommentWithBody(ctx, commentID, "application/json", bytes.NewReader(payload)))
}
