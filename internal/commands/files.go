package commands

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
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
			page, _ := cmd.Flags().GetInt("page")

			params := &freelo.GetAllDocsAndFilesParams{}
			if projectID != 0 {
				ids := []int{projectID}
				params.ProjectsIds = &ids
			}
			if fileType != "" {
				t := freelo.GetAllDocsAndFilesParamsType(fileType)
				params.Type = &t
			}
			if page > 0 {
				p := freelo.PageParam(page)
				params.P = &p
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
	cmd.Flags().Int("page", 0, "Page number (0-indexed)")
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

			// #nosec G304 -- outputPath is either an absolute path the user
			// explicitly passed via --output, or filepath.Base()-stripped
			// data from the server's Content-Disposition / the file UUID,
			// so it is rooted in the working directory.
			file, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			written, err := io.Copy(file, resp.Body)
			if err != nil {
				return fmt.Errorf("failed to write file: %w", err)
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
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			filename := filepath.Base(filePath)
			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)
			part, err := writer.CreateFormFile("file", filename)
			if err != nil {
				return fmt.Errorf("failed to create form file: %w", err)
			}
			if _, err := part.Write(data); err != nil {
				return fmt.Errorf("failed to write file data: %w", err)
			}
			// Close finalizes the multipart boundary in the buffer; the
			// underlying writer is bytes.Buffer (no I/O can fail).
			_ = writer.Close()

			resp, err := app.FreeloClient.UploadFileWithBody(cmd.Context(), writer.FormDataContentType(), &buf)
			if err != nil {
				out.Err(err, "upload_failed", "")
				return err
			}

			result, err := consumeAPIObject(resp, nil)
			if err != nil {
				out.Err(err, "upload_failed", "")
				return err
			}

			out.OK(result, fmt.Sprintf("File '%s' uploaded", filename), nil)
			return nil
		},
	}
	return cmd
}
