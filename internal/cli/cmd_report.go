package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/amr-athena/athena/internal/report"
	"github.com/spf13/cobra"
)

func runReport(cmd *cobra.Command, args []string) error {
	start := time.Now()
	metrics, err := report.Compute(aiDir())
	if err != nil {
		return err
	}
	env := NewEnvelope("report", time.Since(start)).WithData(metrics)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Staleness ratio: %.2f\n", metrics.StalenessRatio)
		fmt.Fprintf(w, "Promotion rate:  %.2f\n", metrics.PromotionRate)
		fmt.Fprintf(w, "Orphan rate:     %.2f\n", metrics.OrphanRate)
	})
	return nil
}
