package commands

import (
	"fmt"
	"time"

	"github.com/freeloio/freelo-cli/internal/api/freelo"
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
			status, err := consumeAPIObject(app.FreeloClient.GetOutOfOffice(cmd.Context(), userID))
			if err != nil {
				out.Err(err, "api_error", "")
				return err
			}
			out.OK(status, "", nil)
			return nil
		},
	}
	cmd.Flags().Int("user", 0, "User ID (required)")
	return cmd
}

// parseOOODate accepts either "YYYY-MM-DD" (treated as start of day UTC) or
// "YYYY-MM-DD HH:MM:SS" (also UTC). The legacy CLI advertised the second
// form in --help; we accept both.
func parseOOODate(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("want YYYY-MM-DD [HH:MM:SS] (UTC), got %q", s)
}

func newOOOEnableCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable out-of-office",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			userID, _ := cmd.Flags().GetInt("user")
			fromStr, _ := cmd.Flags().GetString("from")
			toStr, _ := cmd.Flags().GetString("to")

			if userID == 0 || fromStr == "" || toStr == "" {
				return fmt.Errorf("--user, --from, and --to are required")
			}

			from, err := parseOOODate(fromStr)
			if err != nil {
				return fmt.Errorf("--from: %v", err)
			}
			to, err := parseOOODate(toStr)
			if err != nil {
				return fmt.Errorf("--to: %v", err)
			}

			body := freelo.EnableOutOfOfficeJSONRequestBody{}
			body.OutOfOffice.DateFrom = from
			body.OutOfOffice.DateTo = to

			resp, err := consumeAPIObject(app.FreeloClient.EnableOutOfOffice(cmd.Context(), userID, body))
			if err != nil {
				out.Err(err, "enable_failed", "")
				return err
			}
			out.OK(resp, fmt.Sprintf("Out-of-office enabled %s to %s", fromStr, toStr), nil)
			return nil
		},
	}
	cmd.Flags().Int("user", 0, "User ID (required)")
	cmd.Flags().String("from", "", "Start date UTC (YYYY-MM-DD [HH:MM:SS])")
	cmd.Flags().String("to", "", "End date UTC (YYYY-MM-DD [HH:MM:SS])")
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
			if _, err := consumeAPIObject(app.FreeloClient.DisableOutOfOffice(cmd.Context(), userID)); err != nil {
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
