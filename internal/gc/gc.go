// Package gc implements garbage collection for stale notes.
package gc

import (
	"time"

	"github.com/amr-athena/athena/internal/notes"
)

// Result records the outcome of a GC run.
type Result struct {
	Scanned int      `json:"scanned"`
	Marked  int      `json:"marked"`
	Paths   []string `json:"paths,omitempty"`
}

// Run scans notes in a directory and marks active notes older than
// the threshold as stale. In dry-run mode it reports without modifying.
func Run(dir string, days int, dryRun bool) (*Result, error) {
	allNotes, err := notes.ListNotes(dir, "", "")
	if err != nil {
		return nil, err
	}

	threshold := time.Now().AddDate(0, 0, -days)
	result := &Result{Scanned: len(allNotes)}

	for _, n := range allNotes {
		if n.Frontmatter.Status != "active" {
			continue
		}

		updated, err := time.Parse("2006-01-02", n.Frontmatter.Updated)
		if err != nil {
			continue
		}

		if updated.Before(threshold) {
			result.Marked++
			result.Paths = append(result.Paths, n.Path)

			if !dryRun {
				notes.CloseNote(n.Path, "stale")
			}
		}
	}

	return result, nil
}
