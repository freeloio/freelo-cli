package cli

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	freelosdk "github.com/freeloio/freelo-go"

	"github.com/freeloio/freelo-cli/internal/commands"
	"github.com/freeloio/freelo-cli/internal/config"
	"github.com/freeloio/freelo-cli/internal/credstore"
	"github.com/freeloio/freelo-cli/internal/output"
)

// Version is set at build time via ldflags. The default suffixes -dev so
// `go install` users without ldflags see "this isn't a release build".
var Version = "v1.2.1-dev"

// resolveVersion returns the version string the CLI should report.
//
// Priority: ldflags > module build info > the -dev fallback.
//
// `make build` and goreleaser inject the real tag via ldflags, so Version
// no longer ends in -dev and we trust it. But `go install <module>@vX.Y.Z`
// (and @latest) cannot pass ldflags — there Version stays the -dev fallback,
// yet the Go toolchain records the resolved module version in the binary's
// build info. Recover it from there so `freelo --version` reports the tag a
// user actually installed instead of a misleading -dev string.
//
// A plain `go build` / `go run` of a local checkout reports "(devel)" for the
// main module version; in that case we keep the -dev fallback.
func resolveVersion() string {
	if !strings.HasSuffix(Version, "-dev") {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; strings.HasPrefix(v, "v") {
			return v
		}
	}
	return Version
}

// Execute is the main entry point for the CLI.
func Execute() error {
	// Lock in the reported version once, up front, so the App, the
	// User-Agent, `freelo version`, and `freelo --version` all agree.
	Version = resolveVersion()

	// Single *App instance shared with every command builder. It starts
	// empty; PersistentPreRunE populates the fields before any leaf RunE
	// fires. See internal/commands/app.go for the design rationale.
	app := &commands.App{Version: Version}

	rootCmd := &cobra.Command{
		Use:   "freelo",
		Short: "Freelo CLI — manage projects, tasks, and more from your terminal",
		Long: `Full access to Freelo from your terminal.

freelo is the official command-line interface for Freelo.
Manage projects, tasks, time tracking, and more from your terminal
or through AI agents.

freelo works with any AI agent that can run shell commands.`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return setupApp(cmd, app)
		},
	}

	// `freelo --version` / `-v`. Cobra registers the flag automatically once
	// Version is set; override the template to read "freelo-cli <version>"
	// (matches the `freelo version` subcommand's human-mode string).
	rootCmd.SetVersionTemplate("freelo-cli {{.Version}}\n")

	// Global flags
	rootCmd.PersistentFlags().Bool("dev", false, "Use development environment (requires FREELO_DEV_URL)")
	rootCmd.PersistentFlags().BoolP("json", "j", false, "Output as JSON with envelope")
	rootCmd.PersistentFlags().Bool("agent", false, "Agent mode: raw JSON output (--json + --quiet)")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Minimal output")
	rootCmd.PersistentFlags().Bool("ids-only", false, "Output only IDs")
	rootCmd.PersistentFlags().Bool("count", false, "Output only count")
	rootCmd.PersistentFlags().IntP("project", "p", 0, "Project ID context")

	// Register all commands. Each builder captures the shared `app`
	// pointer — its fields get populated by setupApp before any RunE
	// runs.
	rootCmd.AddCommand(
		// Core
		commands.NewAuthCmd(app),
		commands.NewProjectsCmd(app),
		commands.NewTasksCmd(app),
		commands.NewTasklistsCmd(app),
		commands.NewSubtasksCmd(app),
		commands.NewTaskchecksCmd(app),
		commands.NewSearchCmd(app),
		// Communication
		commands.NewCommentsCmd(app),
		commands.NewNotificationsCmd(app),
		// Time & reports
		commands.NewTrackingCmd(app),
		commands.NewReportsCmd(app),
		// Organization
		commands.NewLabelsCmd(app),
		commands.NewNotesCmd(app),
		commands.NewFilesCmd(app),
		commands.NewCustomFieldsCmd(app),
		commands.NewPinnedCmd(app),
		commands.NewTemplatesCmd(app),
		// People
		commands.NewUsersCmd(app),
		commands.NewWorkersCmd(app),
		commands.NewOutOfOfficeCmd(app),
		// Finance
		commands.NewInvoicesCmd(app),
		// Activity
		commands.NewEventsCmd(app),
		// Utility
		commands.NewSkillCmd(app),
		commands.NewVersionCmd(app),
		commands.NewAPICmd(app),
	)

	return rootCmd.Execute()
}

// setupApp resolves global flags, loads config + credentials, builds the
// Freelo SDK client, and writes everything into the shared *App.
func setupApp(cmd *cobra.Command, app *commands.App) error {
	devMode, _ := cmd.Flags().GetBool("dev")

	cfg, err := config.Load(devMode)
	if err != nil {
		return err
	}
	store := credstore.New(devMode)

	sdk, err := freelosdk.New(
		freelosdk.WithBaseURL(cfg.BaseURL),
		freelosdk.WithAuth(store.AsProvider()),
		freelosdk.WithUserAgent("FreeloCLI/"+Version),
	)
	if err != nil {
		return fmt.Errorf("build Freelo client: %w", err)
	}

	// Resolve output format from the mutually-exclusive set of flags.
	// Order of precedence (first match wins): agent > json > quiet > ids
	// > count > auto. Users typically pass only one of these; the order
	// shouldn't surprise anyone who's mixing them.
	var format output.Format
	agent, _ := cmd.Flags().GetBool("agent")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")
	idsOnly, _ := cmd.Flags().GetBool("ids-only")
	count, _ := cmd.Flags().GetBool("count")

	switch {
	case agent:
		format = output.FormatAgent
	case jsonFlag:
		format = output.FormatJSON
	case quiet:
		format = output.FormatQuiet
	case idsOnly:
		format = output.FormatIDs
	case count:
		format = output.FormatCount
	default:
		format = output.FormatAuto
	}

	// Mutate in place — every command builder captured this same *App
	// pointer at registration time.
	app.Config = cfg
	app.Auth = store
	app.FreeloClient = sdk.API
	app.SDK = sdk
	app.Output = func() *output.Writer { return output.NewWriter(format) }

	if devMode && format != output.FormatAgent && format != output.FormatJSON {
		fmt.Fprintf(cmd.ErrOrStderr(), "[DEV] Using %s\n", cfg.BaseURL)
	}
	return nil
}
