package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// NewUsersCmd creates the 'users' command group.
func NewUsersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "users",
		Aliases: []string{"user"},
		Short:   "Manage users",
	}

	cmd.AddCommand(
		newUsersMeCmd(app),
		newUsersListCmd(app),
	)

	return cmd
}

func newUsersMeCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show current user info",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			result, err := app.Client.Get("/users/me")
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			var user map[string]any
			_ = json.Unmarshal(result, &user)

			fullname := ""
			if fn, ok := user["fullname"].(string); ok {
				fullname = fn
			}

			out.OK(user, fullname, nil)
			return nil
		},
	}
}

func newUsersListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all coworker users",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()

			result, err := app.Client.Get("/users")
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}

			users, _ := parsePaginatedItems(result)

			simplified := make([]map[string]any, 0, len(users))
			for _, u := range users {
				simplified = append(simplified, map[string]any{
					"id":       u["id"],
					"fullname": u["fullname"],
					"email":    u["email"],
				})
			}

			out.OK(simplified, fmt.Sprintf("%d users", len(simplified)), nil)
			return nil
		},
	}
}
