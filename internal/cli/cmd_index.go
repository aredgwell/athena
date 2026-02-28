package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/amr-athena/athena/internal/index"
	"github.com/amr-athena/athena/internal/search"
	"github.com/spf13/cobra"
)

func runIndex(cmd *cobra.Command, args []string) error {
	start := time.Now()
	dir := aiDir()

	idx, err := index.Build(dir)
	if err != nil {
		return err
	}

	outPath := filepath.Join(dir, "index.yaml")
	if err := index.Write(idx, outPath); err != nil {
		return err
	}

	// Build search index alongside metadata index.
	searchIdx, err := index.BuildSearch(dir)
	if err != nil {
		return err
	}
	searchPath := filepath.Join(dir, "search-index.json")
	if err := search.WriteIndex(searchIdx, searchPath); err != nil {
		return err
	}

	env := NewEnvelope("index", time.Since(start)).WithData(idx)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Index built: %d notes (%d active, %d stale, %d closed)\n",
			idx.Counts.Total, idx.Counts.Active, idx.Counts.Stale, idx.Counts.Closed)
		fmt.Fprintf(w, "Search index: %d documents\n", searchIdx.DocCount)
		fmt.Fprintf(w, "Written to %s\n", outPath)
	})
	return nil
}
