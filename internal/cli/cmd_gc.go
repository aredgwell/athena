package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/aredgwell/athena/internal/gc"
	"github.com/spf13/cobra"
)

func runGC(cmd *cobra.Command, args []string) error {
	start := time.Now()
	days, _ := cmd.Flags().GetInt("days")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	result, err := gc.Run(aiDir(), days, dryRun)
	if err != nil {
		return err
	}

	env := NewEnvelope("gc", time.Since(start)).WithData(result)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Scanned %d notes, marked %d stale", result.Scanned, result.Marked)
		if dryRun {
			fmt.Fprint(w, " (dry-run)")
		}
		fmt.Fprintln(w)
		for _, p := range result.Paths {
			fmt.Fprintf(w, "  %s\n", p)
		}
	})
	return nil
}
