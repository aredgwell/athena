package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

func testFiles() []ManagedFile {
	return []ManagedFile{
		{Path: "AGENTS.md", Content: []byte("# AGENTS\n"), Feature: "agents-md"},
		{Path: ".editorconfig", Content: []byte("[*]\nindent_style = space\n"), Feature: "editorconfig"},
	}
}

func TestInitCommand(t *testing.T) {
	t.Run("fresh init", func(t *testing.T) {
		root := t.TempDir()
		files := testFiles()

		summary, err := Init(files, InitOptions{RepoRoot: root, Version: "0.1.0"})
		if err != nil {
			t.Fatalf("init: %v", err)
		}

		if summary.Written != 2 {
			t.Errorf("written: got %d, want 2", summary.Written)
		}
		if summary.Skipped != 0 {
			t.Errorf("skipped: got %d, want 0", summary.Skipped)
		}

		// Verify files exist
		for _, mf := range files {
			data, err := os.ReadFile(filepath.Join(root, mf.Path))
			if err != nil {
				t.Errorf("file %s not created: %v", mf.Path, err)
			}
			if string(data) != string(mf.Content) {
				t.Errorf("file %s content mismatch", mf.Path)
			}
		}

		// Verify checksums written
		cs, err := LoadChecksums(root)
		if err != nil {
			t.Fatal(err)
		}
		if cs == nil {
			t.Fatal("checksums not written")
		}
		if len(cs.Files) != 2 {
			t.Errorf("checksum entries: got %d, want 2", len(cs.Files))
		}
	})

	t.Run("init with collision skip", func(t *testing.T) {
		root := t.TempDir()
		// Pre-create a file
		os.MkdirAll(root, 0o755)
		os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("existing content"), 0o644)

		summary, err := Init(testFiles(), InitOptions{RepoRoot: root, Version: "0.1.0"})
		if err != nil {
			t.Fatal(err)
		}

		if summary.Written != 1 {
			t.Errorf("written: got %d, want 1", summary.Written)
		}
		if summary.Skipped != 1 {
			t.Errorf("skipped: got %d, want 1", summary.Skipped)
		}

		// Existing file should be unchanged
		data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if string(data) != "existing content" {
			t.Error("existing file should not be overwritten")
		}
	})

	t.Run("init with force", func(t *testing.T) {
		root := t.TempDir()
		os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("old"), 0o644)

		summary, err := Init(testFiles(), InitOptions{
			RepoRoot: root, Version: "0.1.0", Force: true,
		})
		if err != nil {
			t.Fatal(err)
		}

		if summary.BackedUp != 1 {
			t.Errorf("backed_up: got %d, want 1", summary.BackedUp)
		}

		// File should be overwritten
		data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if string(data) != "# AGENTS\n" {
			t.Error("file should be overwritten with force")
		}

		// Backup should exist
		backups, _ := filepath.Glob(filepath.Join(root, ".athena", "backups", "AGENTS.md.*.bak"))
		if len(backups) == 0 {
			t.Error("expected backup file")
		}
	})

	t.Run("init dry run", func(t *testing.T) {
		root := t.TempDir()

		summary, err := Init(testFiles(), InitOptions{
			RepoRoot: root, Version: "0.1.0", DryRun: true,
		})
		if err != nil {
			t.Fatal(err)
		}

		if summary.Written != 2 {
			t.Errorf("written: got %d, want 2", summary.Written)
		}

		// No files should actually exist
		for _, mf := range testFiles() {
			if _, err := os.Stat(filepath.Join(root, mf.Path)); err == nil {
				t.Errorf("dry-run should not create %s", mf.Path)
			}
		}

		// No checksums written
		cs, _ := LoadChecksums(root)
		if cs != nil {
			t.Error("dry-run should not write checksums")
		}
	})

	t.Run("init with custom resolver", func(t *testing.T) {
		root := t.TempDir()
		os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("old"), 0o644)

		resolver := func(path string) ConflictResolution {
			return ResolutionOverwrite
		}

		summary, err := Init(testFiles(), InitOptions{
			RepoRoot: root, Version: "0.1.0", IsTTY: true,
			ResolveConflict: resolver,
		})
		if err != nil {
			t.Fatal(err)
		}

		if summary.Overwritten != 1 {
			t.Errorf("overwritten: got %d, want 1", summary.Overwritten)
		}
	})
}

func TestUpgradeCommand(t *testing.T) {
	t.Run("upgrade unmodified files", func(t *testing.T) {
		root := t.TempDir()

		// Init first
		origFiles := testFiles()
		Init(origFiles, InitOptions{RepoRoot: root, Version: "0.1.0"})

		// Upgrade with new content
		newFiles := []ManagedFile{
			{Path: "AGENTS.md", Content: []byte("# AGENTS v2\n"), Feature: "agents-md"},
			{Path: ".editorconfig", Content: []byte("[*]\nindent_style = tab\n"), Feature: "editorconfig"},
		}

		summary, err := Upgrade(newFiles, UpgradeOptions{RepoRoot: root, Version: "0.2.0"})
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}

		if summary.Overwritten != 2 {
			t.Errorf("overwritten: got %d, want 2", summary.Overwritten)
		}
		if summary.BackedUp != 2 {
			t.Errorf("backed_up: got %d, want 2", summary.BackedUp)
		}

		// Verify new content
		data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if string(data) != "# AGENTS v2\n" {
			t.Errorf("file should be upgraded, got: %s", string(data))
		}
	})

	t.Run("upgrade skips user-modified files", func(t *testing.T) {
		root := t.TempDir()

		// Init
		Init(testFiles(), InitOptions{RepoRoot: root, Version: "0.1.0"})

		// Modify AGENTS.md by the user
		os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("user edits"), 0o644)

		newFiles := []ManagedFile{
			{Path: "AGENTS.md", Content: []byte("# AGENTS v2\n"), Feature: "agents-md"},
			{Path: ".editorconfig", Content: []byte("[*]\nindent_style = tab\n"), Feature: "editorconfig"},
		}

		summary, err := Upgrade(newFiles, UpgradeOptions{RepoRoot: root, Version: "0.2.0"})
		if err != nil {
			t.Fatal(err)
		}

		if summary.Skipped != 1 {
			t.Errorf("skipped: got %d, want 1", summary.Skipped)
		}
		if summary.Overwritten != 1 {
			t.Errorf("overwritten: got %d, want 1", summary.Overwritten)
		}

		// User-modified file should be untouched
		data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if string(data) != "user edits" {
			t.Error("user-modified file should not be overwritten")
		}
	})

	t.Run("upgrade writes new files", func(t *testing.T) {
		root := t.TempDir()

		Init(testFiles(), InitOptions{RepoRoot: root, Version: "0.1.0"})

		// Upgrade adds a new file
		newFiles := []ManagedFile{
			{Path: "AGENTS.md", Content: []byte("# AGENTS\n"), Feature: "agents-md"},
			{Path: "NEW_FILE.md", Content: []byte("# NEW\n"), Feature: "new-feature"},
		}

		summary, err := Upgrade(newFiles, UpgradeOptions{RepoRoot: root, Version: "0.2.0"})
		if err != nil {
			t.Fatal(err)
		}

		if summary.Written != 1 {
			t.Errorf("written: got %d, want 1", summary.Written)
		}
	})

	t.Run("upgrade without checksums fails", func(t *testing.T) {
		root := t.TempDir()
		_, err := Upgrade(testFiles(), UpgradeOptions{RepoRoot: root})
		if err == nil {
			t.Fatal("expected error without checksums.json")
		}
	})

	t.Run("upgrade dry run", func(t *testing.T) {
		root := t.TempDir()
		Init(testFiles(), InitOptions{RepoRoot: root, Version: "0.1.0"})

		newFiles := []ManagedFile{
			{Path: "AGENTS.md", Content: []byte("# v2\n"), Feature: "agents-md"},
		}

		summary, err := Upgrade(newFiles, UpgradeOptions{RepoRoot: root, DryRun: true})
		if err != nil {
			t.Fatal(err)
		}

		if summary.Overwritten != 1 {
			t.Errorf("overwritten: got %d, want 1", summary.Overwritten)
		}

		// Original content should be unchanged
		data, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
		if string(data) != "# AGENTS\n" {
			t.Error("dry-run should not modify files")
		}
	})

	t.Run("idempotent repeated init", func(t *testing.T) {
		root := t.TempDir()
		files := testFiles()

		// First init
		Init(files, InitOptions{RepoRoot: root, Version: "0.1.0"})

		// Second init — files already exist, should skip without force
		summary, err := Init(files, InitOptions{RepoRoot: root, Version: "0.1.0"})
		if err != nil {
			t.Fatal(err)
		}

		if summary.Skipped != 2 {
			t.Errorf("expected 2 skipped on re-init, got %d", summary.Skipped)
		}
	})
}

func TestHashContent(t *testing.T) {
	h1 := HashContent([]byte("hello"))
	h2 := HashContent([]byte("hello"))
	h3 := HashContent([]byte("world"))

	if h1 != h2 {
		t.Error("same content should produce same hash")
	}
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
	if len(h1) < 10 {
		t.Errorf("hash too short: %s", h1)
	}
}
