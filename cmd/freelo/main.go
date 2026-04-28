package main

import (
	"fmt"
	"os"

	"github.com/freeloio/freelo-cli/internal/cli"
	"github.com/freeloio/freelo-cli/internal/output"
)

func main() {
	// SilenceErrors=true on the root cobra.Command suppresses Cobra's own
	// error rendering, which we want so successful flows can render their
	// own JSON envelopes via internal/output. The trade-off is that any
	// error that escapes Execute() (validation failures from RunE, missing
	// positional args, unknown subcommands, etc.) would otherwise show up
	// as exit 1 with no message. Surface it here as a fallback — but only
	// when no Writer.Err call has already rendered the same information.
	if err := cli.Execute(); err != nil {
		if !output.ErrorWasRendered() {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}
