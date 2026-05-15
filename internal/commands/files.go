package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	freelo "github.com/freeloio/freelo-go/freeloapi"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const maxUploadSize = 100 * 1024 * 1024

func NewFilesCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "files",
		Aliases: []string{"file"},
		Short:   "Manage files",
	}

	cmd.AddCommand(
		newFilesListCmd(app),
		newFilesDownloadCmd(app),
		newFilesUploadCmd(app),
	)

	return cmd
}

func newFilesListCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all docs and files",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			projectID, _ := cmd.Flags().GetInt("project")
			fileType, _ := cmd.Flags().GetString("type")

			params := &freelo.GetAllDocsAndFilesParams{}
			setProjectsFilter(&params.ProjectsIds, projectID)
			if fileType != "" {
				t := freelo.GetAllDocsAndFilesParamsType(fileType)
				params.Type = &t
			}
			if err := setPageFilter(cmd, &params.P); err != nil {
				return err
			}

			body, err := consumeAPIBody(app.FreeloClient.GetAllDocsAndFiles(cmd.Context(), params))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			items, _ := parsePaginatedItems(body)
			out.OK(items, fmt.Sprintf("%d items", len(items)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().String("type", "", "Filter: directory, link, file, document")
	cmd.Flags().Int("page", 0, "Page number (>= 1; omit for first page)")
	return cmd
}

func newFilesDownloadCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <file-uuid>",
		Short: "Download a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			fileUUIDStr := args[0]
			outputPath, _ := cmd.Flags().GetString("output")

			fileUUID, err := uuid.Parse(fileUUIDStr)
			if err != nil {
				return fmt.Errorf("invalid file UUID: %w", err)
			}

			resp, err := app.FreeloClient.DownloadFile(cmd.Context(), fileUUID)
			if err != nil {
				out.Err(err, "download_failed", "")
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
			}

			if outputPath == "" {
				// Derive from Content-Disposition when available; fall back
				// to the UUID if the header is missing or malformed.
				cd := resp.Header.Get("Content-Disposition")
				if idx := strings.Index(cd, "filename="); idx >= 0 {
					// SECURITY: use filepath.Base to strip any traversal.
					outputPath = filepath.Base(strings.Trim(cd[idx+len("filename="):], `" `))
				}
				if outputPath == "" || outputPath == "." || outputPath == ".." {
					outputPath = fileUUIDStr
				}
			}
			// SECURITY: only allow relative paths to be written to the
			// current directory — never honor a relative path the server
			// might propose with parent components.
			if !filepath.IsAbs(outputPath) {
				outputPath = filepath.Base(outputPath)
			}

			// Write to a .partial sibling and rename on success — this
			// way a network failure or signal mid-transfer can never
			// leave the caller with a truncated file under the final
			// name. Cleanup removes the partial on any error.
			partialPath := outputPath + ".partial"

			// #nosec G304 -- partialPath derives from outputPath, which is
			// either an absolute path the user explicitly passed via
			// --output or filepath.Base()-stripped data from the server's
			// Content-Disposition / the file UUID, so it is rooted in the
			// working directory.
			file, err := os.OpenFile(partialPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			written, copyErr := io.Copy(file, resp.Body)
			closeErr := file.Close()

			switch {
			case copyErr != nil:
				_ = os.Remove(partialPath)
				return fmt.Errorf("failed to write file: %w", copyErr)
			case closeErr != nil:
				_ = os.Remove(partialPath)
				return fmt.Errorf("failed to close file: %w", closeErr)
			}

			if err := os.Rename(partialPath, outputPath); err != nil {
				_ = os.Remove(partialPath)
				return fmt.Errorf("failed to rename partial download: %w", err)
			}

			out.OK(map[string]any{
				"uuid":  fileUUIDStr,
				"path":  outputPath,
				"bytes": written,
			}, fmt.Sprintf("Downloaded to %s (%d bytes)", outputPath, written), nil)
			return nil
		},
	}
	cmd.Flags().StringP("output", "o", "", "Output file path")
	return cmd
}

func newFilesUploadCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload <file-path>",
		Short: "Upload a file (max 100MB)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			filePath := args[0]

			// #nosec G304 -- filePath is the user's own positional CLI arg
			// pointing at a file on their machine; the user IS the trust
			// boundary for an upload command.
			info, err := os.Stat(filePath)
			if err != nil {
				return fmt.Errorf("failed to stat file: %w", err)
			}
			if info.Size() > maxUploadSize {
				return fmt.Errorf("file exceeds 100MB limit")
			}

			// #nosec G304 -- same: user-provided upload source path.
			f, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()

			filename := filepath.Base(filePath)

			// Stream the multipart body through an io.Pipe. The
			// goroutine writes the multipart envelope + file bytes
			// into the pipe; the HTTP client reads from the pipe and
			// pushes to the network. RAM usage stays at the size of
			// the multipart writer's internal buffer (~32KB), not the
			// full file — important for the 100MB ceiling.
			pr, pw := io.Pipe()
			writer := multipart.NewWriter(pw)

			go func() {
				// Any error from this goroutine is propagated to the
				// reader side via pw.CloseWithError, so the HTTP client
				// surfaces it instead of seeing a truncated request.
				defer pw.Close()
				part, perr := writer.CreateFormFile("file", filename)
				if perr != nil {
					_ = pw.CloseWithError(fmt.Errorf("create multipart part: %w", perr))
					return
				}
				if _, cerr := io.Copy(part, f); cerr != nil {
					_ = pw.CloseWithError(fmt.Errorf("stream file body: %w", cerr))
					return
				}
				if werr := writer.Close(); werr != nil {
					_ = pw.CloseWithError(fmt.Errorf("finalize multipart: %w", werr))
					return
				}
			}()

			resp, err := app.FreeloClient.UploadFileWithBody(cmd.Context(), writer.FormDataContentType(), pr)
			if err != nil {
				// If the goroutine failed mid-write, the SDK returns
				// the pipe error wrapped — make sure we don't double-
				// report and we don't leak the response body.
				out.Err(err, "upload_failed", "")
				return err
			}

			result, err := consumeAPIObject(resp, nil)
			if err != nil {
				// Distinguish a context cancellation from a server-
				// side rejection — the message is otherwise opaque.
				if errors.Is(err, context.Canceled) {
					out.Err(err, "upload_failed", "cancelled")
				} else {
					out.Err(err, "upload_failed", "")
				}
				return err
			}

			out.OK(result, fmt.Sprintf("File '%s' uploaded", filename), nil)
			return nil
		},
	}
	return cmd
}
