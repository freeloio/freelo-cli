package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewAuthCmd creates the 'auth' command group.
func NewAuthCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}

	cmd.AddCommand(
		newAuthLoginCmd(app),
		newAuthLogoutCmd(app),
		newAuthStatusCmd(app),
	)

	return cmd
}

func newAuthLoginCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Freelo (email + API key)",
		Long: `Store your Freelo credentials for CLI access.

You can find your API key at: https://app.freelo.io/profil/nastaveni

Alternatively, set FREELO_EMAIL and FREELO_API_KEY environment variables.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("Email: ")
			email, _ := reader.ReadString('\n')
			email = strings.TrimSpace(email)

			fmt.Print("API Key: ")
			apiKeyBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("failed to read API key: %w", err)
			}
			apiKey := strings.TrimSpace(string(apiKeyBytes))

			if email == "" || apiKey == "" {
				return fmt.Errorf("email and API key are required")
			}

			// Verify credentials by calling /users/me. Store first so the
			// FreeloClient's per-request auth editor picks them up.
			if err := app.Auth.Store(email, apiKey); err != nil {
				return err
			}

			resp, err := consumeAPIObject(app.FreeloClient.GetUsersMe(cmd.Context()))
			if err != nil {
				_ = app.Auth.Clear()
				return fmt.Errorf("authentication failed: %w", err)
			}

			// API returns {"result":"success","user":{"id":N}} or {"id":N,"fullname":"..."}
			user := resp
			if u, ok := resp["user"].(map[string]any); ok {
				user = u
			}

			fullname := ""
			if fn, ok := user["fullname"].(string); ok {
				fullname = fn
			}

			out.OK(map[string]any{
				"authenticated": true,
				"email":         email,
				"user_id":       user["id"],
				"fullname":      fullname,
			}, fmt.Sprintf("Authenticated as %s (%s)", fullname, email), []output.Breadcrumb{
				{Action: "list", Cmd: "freelo projects list", Description: "List your projects"},
			})
			return nil
		},
	}
}

func newAuthLogoutCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			if err := app.Auth.Clear(); err != nil {
				return err
			}
			out.OK(map[string]any{"authenticated": false}, "Logged out", nil)
			return nil
		},
	}
}

func newAuthStatusCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			if !app.Auth.IsAuthenticated() {
				out.OK(map[string]any{
					"authenticated": false,
				}, "Not authenticated", []output.Breadcrumb{
					{Action: "login", Cmd: "freelo auth login", Description: "Login to Freelo"},
				})
				return nil
			}

			resp, err := consumeAPIObject(app.FreeloClient.GetUsersMe(cmd.Context()))
			if err != nil {
				out.OK(map[string]any{
					"authenticated": false,
					"error":         err.Error(),
				}, "Credentials invalid", []output.Breadcrumb{
					{Action: "login", Cmd: "freelo auth login", Description: "Re-login to Freelo"},
				})
				return nil
			}

			user := resp
			if u, ok := resp["user"].(map[string]any); ok {
				user = u
			}

			email, _, _ := app.Auth.GetCredentials()
			fullname := ""
			if fn, ok := user["fullname"].(string); ok {
				fullname = fn
			}

			out.OK(map[string]any{
				"authenticated": true,
				"email":         email,
				"user_id":       user["id"],
				"fullname":      fullname,
			}, fmt.Sprintf("Authenticated as %s (%s)", fullname, email), nil)
			return nil
		},
	}
}
