package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/freeloio/freelo-cli/internal/api"
	"github.com/freeloio/freelo-cli/internal/auth"
	"github.com/freeloio/freelo-cli/internal/commands"
	"github.com/freeloio/freelo-cli/internal/config"
	"github.com/freeloio/freelo-cli/internal/output"
)

// Version is set at build time via ldflags.
var Version = "v1.0.0-dev"

// Execute is the main entry point for the CLI.
func Execute() error {
	// Determine output format from flags (set in PersistentPreRun)
	var outputFormat output.Format
	var app *commands.App

	rootCmd := &cobra.Command{
		Use:   "freelo",
		Short: "Freelo CLI — manage projects, tasks, and more from your terminal",
		Long: `Full access to Freelo from your terminal.

freelo is the official command-line interface for Freelo.
Manage projects, tasks, time tracking, and more from your terminal
or through AI agents.

freelo works with any AI agent that can run shell commands.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Resolve --dev flag
			devMode, _ := cmd.Flags().GetBool("dev")

			// Load config and auth based on environment
			cfg := config.Load(devMode)
			credsFile := config.CredentialsFilename(devMode)
			authProvider := auth.NewBasicAuth(credsFile)
			client := api.NewClient(cfg, authProvider)

			// Generated + wrapped Freelo client (used by Phase-3-migrated commands).
			// Build is infallible in practice — NewClientWithResponses only fails
			// on a malformed base URL, which config.Load has already validated.
			freeloClient, err := api.NewFreeloClient(cfg, authProvider, "FreeloCLI/"+Version)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "failed to build Freelo client: %v\n", err)
				return
			}

			// Resolve output format
			agent, _ := cmd.Flags().GetBool("agent")
			jsonFlag, _ := cmd.Flags().GetBool("json")
			quiet, _ := cmd.Flags().GetBool("quiet")
			idsOnly, _ := cmd.Flags().GetBool("ids-only")
			count, _ := cmd.Flags().GetBool("count")

			switch {
			case agent:
				outputFormat = output.FormatAgent
			case jsonFlag:
				outputFormat = output.FormatJSON
			case quiet:
				outputFormat = output.FormatQuiet
			case idsOnly:
				outputFormat = output.FormatIDs
			case count:
				outputFormat = output.FormatCount
			default:
				outputFormat = output.FormatAuto
			}

			// Build app context
			app = &commands.App{
				Config:       cfg,
				Auth:         authProvider,
				Client:       client,
				FreeloClient: freeloClient,
				Output: func() *output.Writer {
					return output.NewWriter(outputFormat)
				},
				Version: Version,
			}

			// Show environment indicator for dev mode
			if devMode && outputFormat != output.FormatAgent && outputFormat != output.FormatJSON {
				fmt.Fprintf(cmd.ErrOrStderr(), "[DEV] Using %s\n", cfg.BaseURL)
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().Bool("dev", false, "Use development environment (requires FREELO_DEV_URL)")
	rootCmd.PersistentFlags().BoolP("json", "j", false, "Output as JSON with envelope")
	rootCmd.PersistentFlags().Bool("agent", false, "Agent mode: raw JSON output (--json + --quiet)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Minimal output")
	rootCmd.PersistentFlags().Bool("ids-only", false, "Output only IDs")
	rootCmd.PersistentFlags().Bool("count", false, "Output only count")
	rootCmd.PersistentFlags().IntP("project", "p", 0, "Project ID context")

	// Lazy app accessor for commands (resolved in PersistentPreRun)
	getApp := func() *commands.App { return app }

	// Register all commands
	rootCmd.AddCommand(
		// Core
		commands.NewAuthCmdLazy(getApp),
		commands.NewProjectsCmdLazy(getApp),
		commands.NewTasksCmdLazy(getApp),
		commands.NewTasklistsCmdLazy(getApp),
		commands.NewSubtasksCmdLazy(getApp),
		commands.NewSearchCmdLazy(getApp),
		// Communication
		commands.NewCommentsCmdLazy(getApp),
		commands.NewNotificationsCmdLazy(getApp),
		// Time & reports
		commands.NewTrackingCmdLazy(getApp),
		commands.NewReportsCmdLazy(getApp),
		// Organization
		commands.NewLabelsCmdLazy(getApp),
		commands.NewNotesCmdLazy(getApp),
		commands.NewFilesCmdLazy(getApp),
		commands.NewCustomFieldsCmdLazy(getApp),
		commands.NewPinnedCmdLazy(getApp),
		commands.NewTemplatesCmdLazy(getApp),
		// People
		commands.NewUsersCmdLazy(getApp),
		commands.NewWorkersCmdLazy(getApp),
		commands.NewOutOfOfficeCmdLazy(getApp),
		// Finance
		commands.NewInvoicesCmdLazy(getApp),
		// Activity
		commands.NewEventsCmdLazy(getApp),
		// Utility
		commands.NewSkillCmdLazy(getApp),
		commands.NewVersionCmdLazy(getApp),
		commands.NewAPICmdLazy(getApp),
	)

	return rootCmd.Execute()
}
