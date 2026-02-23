package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangelogCommand(t *testing.T) {
	t.Run("basic generation", func(t *testing.T) {
		commits := []string{
			"feat(auth): add login endpoint",
			"fix(db): handle null pointer",
			"docs: update README",
		}

		result, err := Generate(Options{
			Commits:     commits,
			NextVersion: "1.0.0",
			DryRun:      true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.EntryCount != 3 {
			t.Errorf("entries: got %d, want 3", result.EntryCount)
		}
		if !strings.Contains(result.Markdown, "## 1.0.0") {
			t.Error("markdown should contain version header")
		}
		if !strings.Contains(result.Markdown, "### Features") {
			t.Error("markdown should contain Features section")
		}
		if !strings.Contains(result.Markdown, "### Bug Fixes") {
			t.Error("markdown should contain Bug Fixes section")
		}
	})

	t.Run("unreleased", func(t *testing.T) {
		result, _ := Generate(Options{
			Commits: []string{"feat: new feature"},
			DryRun:  true,
		})
		if !strings.Contains(result.Markdown, "## Unreleased") {
			t.Error("should use Unreleased when no version")
		}
	})

	t.Run("breaking changes", func(t *testing.T) {
		result, _ := Generate(Options{
			Commits: []string{"feat(api)!: remove old endpoint"},
			DryRun:  true,
		})
		if !strings.Contains(result.Markdown, "### BREAKING CHANGES") {
			t.Error("should include BREAKING CHANGES section")
		}
	})

	t.Run("skip non-conventional", func(t *testing.T) {
		result, _ := Generate(Options{
			Commits: []string{
				"feat: valid commit",
				"random message not conventional",
			},
			DryRun: true,
		})
		if result.EntryCount != 1 {
			t.Errorf("entries: got %d, want 1 (non-conventional skipped)", result.EntryCount)
		}
	})

	t.Run("scope in output", func(t *testing.T) {
		result, _ := Generate(Options{
			Commits: []string{"fix(auth): token expiry"},
			DryRun:  true,
		})
		if !strings.Contains(result.Markdown, "**auth:**") {
			t.Error("markdown should include scope")
		}
	})

	t.Run("deterministic ordering", func(t *testing.T) {
		result1, _ := Generate(Options{
			Commits: []string{
				"fix: bug a",
				"feat: feature b",
				"docs: update c",
			},
			DryRun: true,
		})
		result2, _ := Generate(Options{
			Commits: []string{
				"fix: bug a",
				"feat: feature b",
				"docs: update c",
			},
			DryRun: true,
		})
		if result1.Markdown != result2.Markdown {
			t.Error("changelog should be deterministic")
		}
	})
}

func TestWriteChangelog(t *testing.T) {
	t.Run("new file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "CHANGELOG.md")
		_, err := Generate(Options{
			Commits:     []string{"feat: initial"},
			NextVersion: "0.1.0",
			OutputPath:  path,
		})
		if err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		content := string(data)
		if !strings.Contains(content, "# Changelog") {
			t.Error("should have Changelog header")
		}
		if !strings.Contains(content, "## 0.1.0") {
			t.Error("should have version section")
		}
	})

	t.Run("existing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "CHANGELOG.md")
		os.WriteFile(path, []byte("# Changelog\n\n## 0.1.0\n\nOld stuff.\n"), 0o644)

		_, err := Generate(Options{
			Commits:     []string{"feat: new feature"},
			NextVersion: "0.2.0",
			OutputPath:  path,
		})
		if err != nil {
			t.Fatal(err)
		}

		data, _ := os.ReadFile(path)
		content := string(data)
		if !strings.Contains(content, "## 0.2.0") {
			t.Error("should contain new version")
		}
		if !strings.Contains(content, "## 0.1.0") {
			t.Error("should preserve old version")
		}
	})

	t.Run("dry run does not write", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "CHANGELOG.md")
		_, err := Generate(Options{
			Commits:     []string{"feat: new"},
			NextVersion: "1.0.0",
			OutputPath:  path,
			DryRun:      true,
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = os.Stat(path)
		if !os.IsNotExist(err) {
			t.Error("dry run should not create file")
		}
	})
}

func TestSections(t *testing.T) {
	result, _ := Generate(Options{
		Commits: []string{
			"feat: feature 1",
			"feat: feature 2",
			"fix: bug 1",
			"perf: speed up",
		},
		DryRun: true,
	})

	if len(result.Sections) < 3 {
		t.Errorf("sections: got %d, want >= 3", len(result.Sections))
	}
}
