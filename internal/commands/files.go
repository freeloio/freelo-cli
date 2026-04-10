package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

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

			path := "/all-docs-and-files"
			params := []string{}
			if projectID != 0 {
				params = append(params, fmt.Sprintf("projects_ids[]=%d", projectID))
			}
			if fileType != "" {
				params = append(params, "type="+fileType)
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

			items, _ := parsePaginatedItems(result)

			out.OK(items, fmt.Sprintf("%d items", len(items)), nil)
			return nil
		},
	}
	cmd.Flags().IntP("project", "p", 0, "Filter by project ID")
	cmd.Flags().String("type", "", "Filter: directory, link, file, document")
	cmd.Flags().Int("page", 0, "Page number")
	return cmd
}

func newFilesDownloadCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <file-uuid>",
		Short: "Download a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			fileUUID := args[0]
			outputPath, _ := cmd.Flags().GetString("output")

			email, secret, err := app.Auth.GetCredentials()
			if err != nil {
				return err
			}

			url := app.Config.BaseURL + "/file/" + fileUUID
			req, _ := http.NewRequest("GET", url, nil)
			req.SetBasicAuth(email, secret)
			req.Header.Set("User-Agent", "FreeloCLI")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				out.Err(err, "download_failed", "")
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
			}

			if outputPath == "" {
				// Try to get filename from Content-Disposition
				cd := resp.Header.Get("Content-Disposition")
				if cd != "" && strings.Contains(cd, "filename=") {
					parts := strings.Split(cd, "filename=")
					if len(parts) > 1 {
						// SECURITY: Use filepath.Base to prevent path traversal
						// (e.g., server returning filename="../../.bashrc")
						outputPath = filepath.Base(strings.Trim(parts[1], "\" "))
					}
				}
				if outputPath == "" || outputPath == "." || outputPath == ".." {
					outputPath = fileUUID
				}
			}
			// SECURITY: Always strip directory components from output path
			if !filepath.IsAbs(outputPath) {
				outputPath = filepath.Base(outputPath)
			}

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
				"uuid":  fileUUID,
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

			// Read file
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			if len(data) > 100*1024*1024 {
				return fmt.Errorf("file exceeds 100MB limit")
			}

			email, secret, err := app.Auth.GetCredentials()
			if err != nil {
				return err
			}

			// Create multipart request using mime/multipart (secure random boundary)
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
			writer.Close()

			url := app.Config.BaseURL + "/file/upload"
			req, _ := http.NewRequest("POST", url, &buf)
			req.SetBasicAuth(email, secret)
			req.Header.Set("User-Agent", "FreeloCLI")
			req.Header.Set("Content-Type", writer.FormDataContentType())

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				out.Err(err, "upload_failed", "")
				return err
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 400 {
				return fmt.Errorf("upload failed: HTTP %d: %s", resp.StatusCode, string(respBody))
			}

			var result map[string]any
			_ = json.Unmarshal(respBody, &result)

			out.OK(result, fmt.Sprintf("File '%s' uploaded", filename), nil)
			return nil
		},
	}
	return cmd
}
