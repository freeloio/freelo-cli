package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// NewAPICmd creates the 'api' command for raw API access.
func NewAPICmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Raw API access (get, post, put, delete)",
		Long:  `Make direct API calls to the Freelo API. Useful for endpoints not yet covered by specific commands.`,
	}

	cmd.AddCommand(
		newAPIGetCmd(app),
		newAPIPostCmd(app),
		newAPIPutCmd(app),
		newAPIDeleteCmd(app),
	)

	return cmd
}

func newAPIGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <path>",
		Short: "GET request to API",
		Long:  `Example: freelo api get /projects`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			path := ensureSlashPrefix(args[0])

			result, err := app.Client.Get(path)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			var data any
			_ = json.Unmarshal(result, &data)
			out.OK(data, "", nil)
			return nil
		},
	}
}

func newAPIPostCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post <path>",
		Short: "POST request to API",
		Long:  `Example: freelo api post /search --data '{"query":"test"}'`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			path := ensureSlashPrefix(args[0])
			dataStr, _ := cmd.Flags().GetString("data")

			var body any
			if dataStr != "" {
				if err := json.Unmarshal([]byte(dataStr), &body); err != nil {
					return fmt.Errorf("invalid JSON data: %w", err)
				}
			}

			result, err := app.Client.Post(path, body)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			var data any
			_ = json.Unmarshal(result, &data)
			out.OK(data, "", nil)
			return nil
		},
	}
	cmd.Flags().StringP("data", "d", "", "JSON request body")
	return cmd
}

func newAPIPutCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put <path>",
		Short: "PUT request to API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			path := ensureSlashPrefix(args[0])
			dataStr, _ := cmd.Flags().GetString("data")

			var body any
			if dataStr != "" {
				if err := json.Unmarshal([]byte(dataStr), &body); err != nil {
					return fmt.Errorf("invalid JSON data: %w", err)
				}
			}

			result, err := app.Client.Put(path, body)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			var data any
			_ = json.Unmarshal(result, &data)
			out.OK(data, "", nil)
			return nil
		},
	}
	cmd.Flags().StringP("data", "d", "", "JSON request body")
	return cmd
}

func newAPIDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <path>",
		Short: "DELETE request to API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			path := ensureSlashPrefix(args[0])

			result, err := app.Client.Delete(path)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			var data any
			_ = json.Unmarshal(result, &data)
			out.OK(data, "", nil)
			return nil
		},
	}
}

func ensureSlashPrefix(path string) string {
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}
