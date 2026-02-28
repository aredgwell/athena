package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/amr-athena/athena/internal/search"
	"github.com/spf13/cobra"
)

func runContextSearch(cmd *cobra.Command, args []string) error {
	start := time.Now()
	query := args[0]
	limit, _ := cmd.Flags().GetInt("limit")

	searchPath := filepath.Join(aiDir(), "search-index.json")
	idx, err := search.ReadIndex(searchPath)
	if err != nil {
		return fmt.Errorf("search index not found: run 'athena index' first: %w", err)
	}

	results := idx.Query(query, limit)

	env := NewEnvelope("context search", time.Since(start)).WithData(map[string]any{
		"query":   query,
		"limit":   limit,
		"total":   len(results),
		"results": results,
	})

	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Search: %q (%d results)\n\n", query, len(results))
		for i, r := range results {
			fmt.Fprintf(w, "%d. [%.3f] %s\n", i+1, r.Score, r.Title)
			fmt.Fprintf(w, "   Path: %s  Type: %s  Status: %s\n", r.Path, r.Type, r.Status)
			if r.Snippet != "" {
				fmt.Fprintf(w, "   %s\n", r.Snippet)
			}
		}
		if len(results) == 0 {
			fmt.Fprintln(w, "No results found.")
		}
	})
	return nil
}
