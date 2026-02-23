package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func writeNote(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckCommand(t *testing.T) {
	t.Run("valid notes", func(t *testing.T) {
		dir := t.TempDir()
		writeNote(t, dir, "ctx.md", `---
id: context-20260220-test
title: Test Note
type: context
status: active
created: "2026-02-20"
updated: "2026-02-20"
schema_version: 1
---

Content here.
`)

		results, summary, err := Check(CheckOptions{Dir: dir})
		if err != nil {
			t.Fatal(err)
		}

		if summary.FilesScanned != 1 {
			t.Errorf("scanned: got %d, want 1", summary.FilesScanned)
		}
		if summary.Valid != 1 {
			t.Errorf("valid: got %d, want 1", summary.Valid)
		}
		if summary.Invalid != 0 {
			t.Errorf("invalid: got %d, want 0", summary.Invalid)
		}
		if !results[0].Valid {
			t.Errorf("result should be valid, errors: %v", results[0].Errors)
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		dir := t.TempDir()
		writeNote(t, dir, "bad.md", `---
id: ""
title: ""
type: ""
status: ""
created: ""
updated: ""
---

No fields.
`)

		results, summary, err := Check(CheckOptions{Dir: dir})
		if err != nil {
			t.Fatal(err)
		}

		if summary.Invalid != 1 {
			t.Errorf("invalid: got %d, want 1", summary.Invalid)
		}
		if len(results[0].Errors) < 6 {
			t.Errorf("expected at least 6 errors, got %d: %v", len(results[0].Errors), results[0].Errors)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		dir := t.TempDir()
		writeNote(t, dir, "bad-type.md", `---
id: test-20260220-x
title: Test
type: invalid_type
status: active
created: "2026-02-20"
updated: "2026-02-20"
---

Content.
`)

		results, _, _ := Check(CheckOptions{Dir: dir})
		if results[0].Valid {
			t.Error("expected invalid for bad type")
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		dir := t.TempDir()
		writeNote(t, dir, "bad-status.md", `---
id: test-20260220-x
title: Test
type: context
status: unknown_status
created: "2026-02-20"
updated: "2026-02-20"
---

Content.
`)

		results, _, _ := Check(CheckOptions{Dir: dir})
		if results[0].Valid {
			t.Error("expected invalid for bad status")
		}
	})

	t.Run("parse error", func(t *testing.T) {
		dir := t.TempDir()
		writeNote(t, dir, "nofm.md", "# No frontmatter\n")

		results, summary, _ := Check(CheckOptions{Dir: dir})
		if summary.Invalid != 1 {
			t.Errorf("invalid: got %d, want 1", summary.Invalid)
		}
		if len(results[0].Errors) == 0 {
			t.Error("expected parse error")
		}
	})

	t.Run("multiple notes", func(t *testing.T) {
		dir := t.TempDir()
		validContent := `---
id: context-20260220-a
title: Note A
type: context
status: active
created: "2026-02-20"
updated: "2026-02-20"
schema_version: 1
---

A.
`
		writeNote(t, dir, "a.md", validContent)
		writeNote(t, dir, "b.md", validContent)
		writeNote(t, dir, "sub/c.md", validContent)

		_, summary, err := Check(CheckOptions{Dir: dir})
		if err != nil {
			t.Fatal(err)
		}
		if summary.FilesScanned != 3 {
			t.Errorf("scanned: got %d, want 3", summary.FilesScanned)
		}
		if summary.Valid != 3 {
			t.Errorf("valid: got %d, want 3", summary.Valid)
		}
	})

	t.Run("strict schema passes for current version", func(t *testing.T) {
		dir := t.TempDir()
		writeNote(t, dir, "current.md", `---
id: test-20260220-x
title: Test
type: context
status: active
created: "2026-02-20"
updated: "2026-02-20"
schema_version: 1
---

Content.
`)

		results, _, _ := Check(CheckOptions{Dir: dir, StrictSchema: true})
		if !results[0].Valid {
			t.Errorf("strict schema should pass for current version, errors: %v", results[0].Errors)
		}
	})

	t.Run("strict schema fails for absent version", func(t *testing.T) {
		dir := t.TempDir()
		// schema_version absent => raw value 0, strict mode checks raw value
		writeNote(t, dir, "noschema.md", `---
id: test-20260220-x
title: Test
type: context
status: active
created: "2026-02-20"
updated: "2026-02-20"
---

Content.
`)

		results, _, _ := Check(CheckOptions{Dir: dir, StrictSchema: true})
		if results[0].Valid {
			t.Error("strict schema should fail when schema_version is absent")
		}
	})
}

func TestCheckFixCommand(t *testing.T) {
	t.Run("fix does not modify valid notes", func(t *testing.T) {
		dir := t.TempDir()
		writeNote(t, dir, "ok.md", `---
id: context-20260220-ok
title: OK Note
type: context
status: active
created: "2026-02-20"
updated: "2026-02-20"
schema_version: 1
---

Good.
`)

		_, summary, _ := Check(CheckOptions{Dir: dir, Fix: true, BackupDir: filepath.Join(dir, "backups")})
		if summary.Fixed != 0 {
			t.Errorf("fixed: got %d, want 0 (nothing to fix)", summary.Fixed)
		}
	})
}
