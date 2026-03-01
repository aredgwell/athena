package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/aredgwell/athena/internal/gc"
	"github.com/aredgwell/athena/internal/notes"
	"github.com/aredgwell/athena/internal/validate"
	"github.com/spf13/cobra"
)

func runReviewPromotions(cmd *cobra.Command, args []string) error {
	start := time.Now()

	allNotes, err := notes.ListNotes(aiDir(), "active", "")
	if err != nil {
		return err
	}

	type candidate struct {
		Path  string `json:"path"`
		ID    string `json:"id"`
		Type  string `json:"type"`
		Title string `json:"title"`
	}
	var candidates []candidate
	for _, n := range allNotes {
		if n.Frontmatter.Type == "improvement" || n.Frontmatter.Type == "investigation" {
			candidates = append(candidates, candidate{
				Path:  n.Path,
				ID:    n.Frontmatter.ID,
				Type:  n.Frontmatter.Type,
				Title: n.Frontmatter.Title,
			})
		}
	}

	env := NewEnvelope("review promotions", time.Since(start)).WithData(map[string]any{
		"count":      len(candidates),
		"candidates": candidates,
	})
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "%d promotion candidates\n", len(candidates))
		for _, c := range candidates {
			fmt.Fprintf(w, "  [%s] %s: %s\n", c.Type, c.ID, c.Title)
		}
	})
	return nil
}

func runReviewWeekly(cmd *cobra.Command, args []string) error {
	start := time.Now()
	days, _ := cmd.Flags().GetInt("days")

	dir := aiDir()

	// Step 1: GC
	gcResult, _ := gc.Run(dir, days, false)

	// Step 2: Check
	_, checkSummary, _ := validate.Check(validate.CheckOptions{Dir: dir})

	// Step 3: Promotions
	allNotes, _ := notes.ListNotes(dir, "active", "")
	var promotable int
	for _, n := range allNotes {
		if n.Frontmatter.Type == "improvement" || n.Frontmatter.Type == "investigation" {
			promotable++
		}
	}

	result := map[string]any{
		"gc_marked":            gcResult.Marked,
		"check_scanned":        checkSummary.FilesScanned,
		"check_invalid":        checkSummary.Invalid,
		"promotion_candidates": promotable,
	}

	env := NewEnvelope("review weekly", time.Since(start)).WithData(result)
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Weekly review (%d-day window):\n", days)
		fmt.Fprintf(w, "  GC: %d notes marked stale\n", gcResult.Marked)
		fmt.Fprintf(w, "  Check: %d scanned, %d invalid\n", checkSummary.FilesScanned, checkSummary.Invalid)
		fmt.Fprintf(w, "  Promotions: %d candidates\n", promotable)
	})
	return nil
}
