package main

import (
	"fmt"
	"os"

	"github.com/freeloio/freelo-cli/internal/cli"
)

func main() {
	// SilenceErrors=true on the root cobra.Command suppresses Cobra's own
	// error rendering, which we want so successful flows can render their
	// own JSON envelopes via internal/output. The trade-off is that any
	// error that escapes Execute() (validation failures from RunE, missing
	// positional args, unknown subcommands, etc.) needs to be surfaced
	// here — otherwise the user sees exit 1 with no message and no clue
	// what went wrong.
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
