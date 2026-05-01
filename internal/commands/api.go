package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

// NewAPICmd creates the 'api' command for raw API access.
//
// This is the CLI's escape hatch for endpoints not yet covered by
// purpose-built subcommands. After Phase 3 the typed Freelo client covers
// the whole OpenAPI spec, but `api get/post/put/delete` is still useful
// for calling internal / undocumented endpoints or for quick debugging.
//
// All four subcommands route through the SDK's Do, so they get the same
// auth + User-Agent + rate-limit + retry as the typed commands.
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

// doRaw performs an arbitrary HTTP call through the SDK pipeline and
// decodes the JSON response (or returns the raw bytes on parse failure).
func doRaw(cmd *cobra.Command, app *App, method, path string, body any) (any, error) {
	if app.SDK == nil {
		return nil, fmt.Errorf("internal: SDK client not initialized")
	}

	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	url := app.SDK.BaseURL()
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	url += path

	var req *http.Request
	var err error
	if reader != nil {
		req, err = http.NewRequestWithContext(cmd.Context(), method, url, reader)
	} else {
		req, err = http.NewRequestWithContext(cmd.Context(), method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.SDK.Do(req)
	if err != nil {
		return nil, err
	}
	return consumeAPIAny(resp)
}

// consumeAPIAny is a small variant of consumeAPIObject that tolerates
// non-object JSON (arrays, primitives) — the `api` passthrough can hit
// endpoints that return bare arrays.
func consumeAPIAny(resp *http.Response) (any, error) {
	raw, err := readRawBody(resp)
	if err != nil {
		return nil, err
	}
	if err := checkAPIStatus(raw, resp); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	var v any
	_ = json.Unmarshal(raw, &v)
	return v, nil
}

func newAPIGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <path>",
		Short: "GET request to API",
		Long:  `Example: freelo api get /projects`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			data, err := doRaw(cmd, app, "GET", ensureSlashPrefix(args[0]), nil)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			out.OK(data, "", nil)
			return nil
		},
	}
}

func newAPIPostCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post <path>",
		Short: "POST request to API",
		Long:  `Example: freelo api post /search --data '{"search_query":"test"}'`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			dataStr, _ := cmd.Flags().GetString("data")

			var body any
			if dataStr != "" {
				if err := json.Unmarshal([]byte(dataStr), &body); err != nil {
					return fmt.Errorf("invalid JSON data: %w", err)
				}
			}

			data, err := doRaw(cmd, app, "POST", ensureSlashPrefix(args[0]), body)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
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
			dataStr, _ := cmd.Flags().GetString("data")

			var body any
			if dataStr != "" {
				if err := json.Unmarshal([]byte(dataStr), &body); err != nil {
					return fmt.Errorf("invalid JSON data: %w", err)
				}
			}

			data, err := doRaw(cmd, app, "PUT", ensureSlashPrefix(args[0]), body)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
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
			data, err := doRaw(cmd, app, "DELETE", ensureSlashPrefix(args[0]), nil)
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
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
