package commands

import (
	"github.com/freeloio/freelo-cli/internal/api"
	"github.com/freeloio/freelo-cli/internal/api/freelo"
	"github.com/freeloio/freelo-cli/internal/auth"
	"github.com/freeloio/freelo-cli/internal/config"
	"github.com/freeloio/freelo-cli/internal/output"
	"github.com/spf13/cobra"
)

// App is the shared dependency container passed to all commands.
//
// During the Phase 3 migration both HTTP clients coexist: `Client` is the
// legacy handwritten client that unmigrated commands still call, and
// `FreeloClient` is the oapi-codegen-generated typed client used by commands
// that have been migrated. Once every command is migrated, `Client` and
// `internal/api/client.go` get deleted.
type App struct {
	Config       *config.Config
	Auth         auth.Provider
	Client       *api.Client
	FreeloClient *freelo.ClientWithResponses
	Output       func() *output.Writer
	Version      string
}

// AppGetter is a function that returns the App (resolved lazily after PersistentPreRun).
type AppGetter func() *App

// Lazy command constructors — these wrap the original constructors to work with
// deferred App initialization (App is created in PersistentPreRun based on --dev flag).

func NewAuthCmdLazy(getApp AppGetter) *cobra.Command      { return wrapLazy(getApp, NewAuthCmd) }
func NewProjectsCmdLazy(getApp AppGetter) *cobra.Command  { return wrapLazy(getApp, NewProjectsCmd) }
func NewTasksCmdLazy(getApp AppGetter) *cobra.Command     { return wrapLazy(getApp, NewTasksCmd) }
func NewTasklistsCmdLazy(getApp AppGetter) *cobra.Command { return wrapLazy(getApp, NewTasklistsCmd) }
func NewSubtasksCmdLazy(getApp AppGetter) *cobra.Command  { return wrapLazy(getApp, NewSubtasksCmd) }
func NewSearchCmdLazy(getApp AppGetter) *cobra.Command    { return wrapLazy(getApp, NewSearchCmd) }
func NewCommentsCmdLazy(getApp AppGetter) *cobra.Command  { return wrapLazy(getApp, NewCommentsCmd) }
func NewNotificationsCmdLazy(getApp AppGetter) *cobra.Command {
	return wrapLazy(getApp, NewNotificationsCmd)
}
func NewTrackingCmdLazy(getApp AppGetter) *cobra.Command { return wrapLazy(getApp, NewTrackingCmd) }
func NewReportsCmdLazy(getApp AppGetter) *cobra.Command  { return wrapLazy(getApp, NewReportsCmd) }
func NewLabelsCmdLazy(getApp AppGetter) *cobra.Command   { return wrapLazy(getApp, NewLabelsCmd) }
func NewNotesCmdLazy(getApp AppGetter) *cobra.Command    { return wrapLazy(getApp, NewNotesCmd) }
func NewFilesCmdLazy(getApp AppGetter) *cobra.Command    { return wrapLazy(getApp, NewFilesCmd) }
func NewCustomFieldsCmdLazy(getApp AppGetter) *cobra.Command {
	return wrapLazy(getApp, NewCustomFieldsCmd)
}
func NewPinnedCmdLazy(getApp AppGetter) *cobra.Command    { return wrapLazy(getApp, NewPinnedCmd) }
func NewTemplatesCmdLazy(getApp AppGetter) *cobra.Command { return wrapLazy(getApp, NewTemplatesCmd) }
func NewUsersCmdLazy(getApp AppGetter) *cobra.Command     { return wrapLazy(getApp, NewUsersCmd) }
func NewWorkersCmdLazy(getApp AppGetter) *cobra.Command   { return wrapLazy(getApp, NewWorkersCmd) }
func NewOutOfOfficeCmdLazy(getApp AppGetter) *cobra.Command {
	return wrapLazy(getApp, NewOutOfOfficeCmd)
}
func NewInvoicesCmdLazy(getApp AppGetter) *cobra.Command { return wrapLazy(getApp, NewInvoicesCmd) }
func NewEventsCmdLazy(getApp AppGetter) *cobra.Command   { return wrapLazy(getApp, NewEventsCmd) }
func NewSkillCmdLazy(getApp AppGetter) *cobra.Command    { return wrapLazy(getApp, NewSkillCmd) }
func NewVersionCmdLazy(getApp AppGetter) *cobra.Command  { return wrapLazy(getApp, NewVersionCmd) }
func NewAPICmdLazy(getApp AppGetter) *cobra.Command      { return wrapLazy(getApp, NewAPICmd) }

// wrapLazy creates a command using a placeholder App, then replaces its RunE
// functions at execution time with the real App from PersistentPreRun.
// This approach: build the command tree at init time (for help/completion),
// but defer the actual App creation until the --dev flag is resolved.
func wrapLazy(getApp AppGetter, builder func(*App) *cobra.Command) *cobra.Command {
	// Build with a placeholder to get the command structure (Use, Short, flags, subcommands)
	placeholder := &App{}
	cmd := builder(placeholder)

	// Override all RunE functions to use the real app
	patchRunE(cmd, getApp, builder)
	return cmd
}

func patchRunE(cmd *cobra.Command, getApp AppGetter, builder func(*App) *cobra.Command) {
	// For leaf commands with RunE, replace with lazy version
	if cmd.RunE != nil {
		originalUse := cmd.Use
		cmd.RunE = func(c *cobra.Command, args []string) error {
			app := getApp()
			if app == nil {
				return nil
			}
			// Rebuild the real command and find the matching subcommand
			realCmd := builder(app)
			target := findSubCmd(realCmd, originalUse)
			if target != nil && target.RunE != nil {
				return target.RunE(c, args)
			}
			return nil
		}
	}

	// Recurse into subcommands
	for _, sub := range cmd.Commands() {
		patchRunE(sub, getApp, builder)
	}
}

func findSubCmd(cmd *cobra.Command, use string) *cobra.Command {
	if cmd.Use == use {
		return cmd
	}
	for _, sub := range cmd.Commands() {
		if found := findSubCmd(sub, use); found != nil {
			return found
		}
	}
	return nil
}
