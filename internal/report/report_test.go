package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aredgwell/athena/internal/notes"
)

func TestReportCommand(t *testing.T) {
	t.Run("basic metrics", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), ".ai")

		// Create notes with various statuses
		notes.NewNote(dir, "context", "a", "Active Context")
		notes.NewNote(dir, "context", "b", "Active Context 2")

		n3, _ := notes.NewNote(dir, "improvement", "c", "Improvement")
		notes.PromoteNote(n3.Path, "docs/x.md")

		notes.NewNote(dir, "investigation", "d", "Investigation")

		// Create a stale note
		staleDir := filepath.Join(dir, "context")
		os.MkdirAll(staleDir, 0o755)
		staleNote := `---
id: context-20250101-stale
title: Stale Note
type: context
status: stale
created: "2025-01-01"
updated: "2025-01-01"
schema_version: 1
---

Stale.
`
		os.WriteFile(filepath.Join(staleDir, "context-20250101-stale.md"), []byte(staleNote), 0o644)

		metrics, err := Compute(dir)
		if err != nil {
			t.Fatal(err)
		}

		if metrics.StalenessRatio != 1.0/5.0 {
			t.Errorf("staleness_ratio: got %f, want 0.2", metrics.StalenessRatio)
		}

		// 1 promoted out of 2 promotable (improvement + investigation)
		if metrics.PromotionRate != 0.5 {
			t.Errorf("promotion_rate: got %f, want 0.5", metrics.PromotionRate)
		}

		// All notes have empty related lists = all orphans
		if metrics.OrphanRate != 1.0 {
			t.Errorf("orphan_rate: got %f, want 1.0", metrics.OrphanRate)
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), ".ai")
		os.MkdirAll(dir, 0o755)

		metrics, err := Compute(dir)
		if err != nil {
			t.Fatal(err)
		}
		if metrics.StalenessRatio != 0 {
			t.Errorf("staleness_ratio: got %f, want 0", metrics.StalenessRatio)
		}
	})

	t.Run("no promotable types", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), ".ai")
		notes.NewNote(dir, "context", "a", "Context Only")

		metrics, _ := Compute(dir)
		if metrics.PromotionRate != 0 {
			t.Errorf("promotion_rate: got %f, want 0 (no promotable types)", metrics.PromotionRate)
		}
	})
}
