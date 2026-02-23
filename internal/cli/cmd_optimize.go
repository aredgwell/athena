package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/amr-athena/athena/internal/optimize"
	"github.com/amr-athena/athena/internal/telemetry"
	"github.com/spf13/cobra"
)

func runOptimizeRecommend(cmd *cobra.Command, args []string) error {
	start := time.Now()
	window, _ := cmd.Flags().GetString("window")

	// Parse "30d" format into days
	windowDays := 30
	if window != "" {
		if len(window) > 1 && window[len(window)-1] == 'd' {
			if n, err := strconv.Atoi(window[:len(window)-1]); err == nil {
				windowDays = n
			}
		}
	}

	// Load telemetry records
	storePath := rc.cfg.Telemetry.Path
	if storePath == "" {
		storePath = filepath.Join(athenaDir(), "telemetry.jsonl")
	}
	store := telemetry.NewStore(storePath)
	records, err := store.ReadAll()
	if err != nil {
		// No telemetry yet is not an error
		records = nil
	}

	result, err := optimize.Recommend(records, rc.cfg.Optimize, optimize.RecommendOptions{
		WindowDays: windowDays,
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("optimize recommend", time.Since(start)).WithData(result)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Window: %d days, %d proposals\n", result.WindowDays, len(result.Proposals))
		for _, p := range result.Proposals {
			fmt.Fprintf(w, "  [%s] %v → %v (%.0f%% token reduction, confidence %.2f)\n",
				p.Target, p.Current, p.Recommended,
				p.ProjectedTokenReduction*100, p.Confidence)
		}
	})
	return nil
}
