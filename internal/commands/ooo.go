package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func NewOutOfOfficeCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "out-of-office",
		Aliases: []string{"ooo"},
		Short:   "Manage out-of-office status",
	}

	cmd.AddCommand(
		newOOOStatusCmd(app),
		newOOOEnableCmd(app),
		newOOODisableCmd(app),
	)

	return cmd
}

func newOOOStatusCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check out-of-office status",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			userID, _ := cmd.Flags().GetInt("user")
			if userID == 0 {
				return fmt.Errorf("--user is required")
			}

			result, err := app.Client.Get(fmt.Sprintf("/user/%d/out-of-office", userID))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			var status map[string]any
			_ = json.Unmarshal(result, &status)
			out.OK(status, "", nil)
			return nil
		},
	}
	cmd.Flags().Int("user", 0, "User ID (required)")
	return cmd
}

func newOOOEnableCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable out-of-office",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			userID, _ := cmd.Flags().GetInt("user")
			dateFrom, _ := cmd.Flags().GetString("from")
			dateTo, _ := cmd.Flags().GetString("to")

			if userID == 0 || dateFrom == "" || dateTo == "" {
				return fmt.Errorf("--user, --from, and --to are required")
			}

			result, err := app.Client.Post(fmt.Sprintf("/user/%d/out-of-office", userID), map[string]any{
				"date_from": dateFrom,
				"date_to":   dateTo,
			})
			if err != nil {
				out.Err(err, "enable_failed", "")
				return err
			}
			var resp any
			_ = json.Unmarshal(result, &resp)
			out.OK(resp, fmt.Sprintf("Out-of-office enabled %s to %s", dateFrom, dateTo), nil)
			return nil
		},
	}
	cmd.Flags().Int("user", 0, "User ID (required)")
	cmd.Flags().String("from", "", "Start date UTC (YYYY-MM-DD HH:MM:SS)")
	cmd.Flags().String("to", "", "End date UTC (YYYY-MM-DD HH:MM:SS)")
	return cmd
}

func newOOODisableCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable out-of-office",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			userID, _ := cmd.Flags().GetInt("user")
			if userID == 0 {
				return fmt.Errorf("--user is required")
			}

			_, err := app.Client.Delete(fmt.Sprintf("/user/%d/out-of-office", userID))
			if err != nil {
				out.Err(err, "disable_failed", "")
				return err
			}
			out.OK(map[string]any{"user_id": userID, "out_of_office": false}, "Out-of-office disabled", nil)
			return nil
		},
	}
	cmd.Flags().Int("user", 0, "User ID (required)")
	return cmd
}
