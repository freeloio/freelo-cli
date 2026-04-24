package commands

import (
	"runtime"

	"github.com/spf13/cobra"
)

// NewVersionCmd creates the 'version' command.
func NewVersionCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := app.Output()
			out.OK(map[string]any{
				"version": app.Version,
				"go":      runtime.Version(),
				"os":      runtime.GOOS,
				"arch":    runtime.GOARCH,
			}, "freelo-cli "+app.Version, nil)
			return nil
		},
	}
}
