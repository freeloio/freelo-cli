package commands

import (
	freelosdk "github.com/freeloio/freelo-go"
	"github.com/freeloio/freelo-go/freeloapi"
	"github.com/freeloio/freelo-cli/internal/config"
	"github.com/freeloio/freelo-cli/internal/credstore"
	"github.com/freeloio/freelo-cli/internal/output"
)

// App is the shared dependency container passed to every command.
//
// Cobra resolves the root persistent flags (--dev, --json, etc.) before
// dispatching to a leaf RunE; only at that point do we know which Freelo
// instance / credential store / output format to wire up. So root.go
// constructs a single empty *App at startup, hands the same pointer to
// every command builder, and mutates the struct in place from
// PersistentPreRunE. Leaf RunE callbacks then read app.* live — the
// pointer they captured at build time is the same pointer root.go just
// populated.
//
// This deliberately keeps the dependency wiring boring: no lazy
// wrappers, no per-call command-tree rebuild, no Use-string-based
// lookups. A single *App, mutated once per process invocation, observed
// by every leaf.
type App struct {
	Config       *config.Config
	Auth         *credstore.Store
	FreeloClient *freeloapi.ClientWithResponses
	SDK          *freelosdk.Client
	Output       func() *output.Writer
	Version      string
}
