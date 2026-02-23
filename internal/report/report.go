// Package report implements local memory effectiveness metrics computation.
package report

import (
	"github.com/amr-athena/athena/internal/notes"
)

// Metrics holds computed memory effectiveness metrics.
type Metrics struct {
	StalenessRatio float64 `json:"staleness_ratio"`
	PromotionRate  float64 `json:"promotion_rate"`
	OrphanRate     float64 `json:"orphan_rate"`
}

// Compute calculates metrics from notes in the given directory.
func Compute(dir string) (*Metrics, error) {
	allNotes, err := notes.ListNotes(dir, "", "")
	if err != nil {
		return nil, err
	}

	if len(allNotes) == 0 {
		return &Metrics{}, nil
	}

	var staleCount, promotedCount, orphanCount int
	var promotableCount int // improvement + investigation types

	for _, n := range allNotes {
		fm := n.Frontmatter

		if fm.Status == "stale" {
			staleCount++
		}
		if fm.Status == "promoted" {
			promotedCount++
		}

		if fm.Type == "improvement" || fm.Type == "investigation" {
			promotableCount++
		}

		if len(fm.Related) == 0 {
			orphanCount++
		}
	}

	total := float64(len(allNotes))

	metrics := &Metrics{
		StalenessRatio: float64(staleCount) / total,
		OrphanRate:     float64(orphanCount) / total,
	}

	if promotableCount > 0 {
		metrics.PromotionRate = float64(promotedCount) / float64(promotableCount)
	}

	return metrics, nil
}
