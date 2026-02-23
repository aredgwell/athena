package gc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amr-athena/athena/internal/notes"
)

func TestGCCommand(t *testing.T) {
	t.Run("marks old active notes as stale", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), ".ai")
		notes.NewNote(dir, "context", "recent", "Recent Note")

		// Create an old note by writing it directly with an old date
		oldDir := filepath.Join(dir, "context")
		os.MkdirAll(oldDir, 0o755)
		oldNote := `---
id: context-20250101-old
title: Old Note
type: context
status: active
created: "2025-01-01"
updated: "2025-01-01"
schema_version: 1
---

Old content.
`
		os.WriteFile(filepath.Join(oldDir, "context-20250101-old.md"), []byte(oldNote), 0o644)

		result, err := Run(dir, 45, false)
		if err != nil {
			t.Fatal(err)
		}

		if result.Scanned != 2 {
			t.Errorf("scanned: got %d, want 2", result.Scanned)
		}
		if result.Marked != 1 {
			t.Errorf("marked: got %d, want 1", result.Marked)
		}

		// Verify the old note is now stale
		n, _ := notes.ParseNote(filepath.Join(oldDir, "context-20250101-old.md"))
		if n.Frontmatter.Status != "stale" {
			t.Errorf("status: got %s, want stale", n.Frontmatter.Status)
		}
	})

	t.Run("dry run does not modify", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), ".ai")
		oldDir := filepath.Join(dir, "context")
		os.MkdirAll(oldDir, 0o755)
		oldNote := `---
id: context-20250101-old
title: Old Note
type: context
status: active
created: "2025-01-01"
updated: "2025-01-01"
schema_version: 1
---

Old.
`
		path := filepath.Join(oldDir, "context-20250101-old.md")
		os.WriteFile(path, []byte(oldNote), 0o644)

		result, err := Run(dir, 45, true)
		if err != nil {
			t.Fatal(err)
		}
		if result.Marked != 1 {
			t.Errorf("marked: got %d, want 1", result.Marked)
		}

		// File should be unchanged
		n, _ := notes.ParseNote(path)
		if n.Frontmatter.Status != "active" {
			t.Error("dry-run should not modify files")
		}
	})

	t.Run("skips non-active notes", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), ".ai")
		oldDir := filepath.Join(dir, "context")
		os.MkdirAll(oldDir, 0o755)
		closedNote := `---
id: context-20250101-closed
title: Closed Note
type: context
status: closed
created: "2025-01-01"
updated: "2025-01-01"
schema_version: 1
---

Closed.
`
		os.WriteFile(filepath.Join(oldDir, "context-20250101-closed.md"), []byte(closedNote), 0o644)

		result, _ := Run(dir, 45, false)
		if result.Marked != 0 {
			t.Errorf("marked: got %d, want 0 (closed notes should be skipped)", result.Marked)
		}
	})
}
