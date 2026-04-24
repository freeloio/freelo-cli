package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/freeloio/freelo-cli/internal/api"
	"github.com/spf13/cobra"
)

// NewAPICmd creates the 'api' command for raw API access.
//
// This is the CLI's generic escape hatch for endpoints not yet covered by
// purpose-built subcommands. After Phase 3 the typed Freelo client covers
// the whole OpenAPI spec, but `api get/post/put/delete` is still useful
// for calling internal / undocumented endpoints or for quick debugging.
//
// All four subcommands go through the same retrying/rate-limited doer and
// auth/UA editors as the typed commands — we reuse the underlying
// *freelo.Client rather than spawning http.DefaultClient.
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

// doRaw performs an arbitrary HTTP call against the Freelo API using the
// wrapper's doer + editors. Returns the decoded JSON body (or raw on
// parse failure).
func doRaw(app *App, method, path string, body any) (any, error) {
	raw, ok := api.RawClientFromResponses(app.FreeloClient)
	if !ok {
		return nil, fmt.Errorf("internal: FreeloClient is not a *freelo.Client")
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	url := raw.Server + path
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Run the shared editors (Basic Auth + User-Agent) before the call.
	for _, edit := range raw.RequestEditors {
		if err := edit(req.Context(), req); err != nil {
			return nil, err
		}
	}

	resp, err := raw.Client.Do(req)
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
			data, err := doRaw(app, "GET", ensureSlashPrefix(args[0]), nil)
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

			data, err := doRaw(app, "POST", ensureSlashPrefix(args[0]), body)
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

			data, err := doRaw(app, "PUT", ensureSlashPrefix(args[0]), body)
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
			data, err := doRaw(app, "DELETE", ensureSlashPrefix(args[0]), nil)
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
