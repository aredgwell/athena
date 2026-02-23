package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validNote = `---
id: context-20260220-auth-flow
title: Auth Flow Analysis
type: context
status: active
created: "2026-02-20"
updated: "2026-02-20"
schema_version: 1
---

# Auth Flow Analysis

Body content here.
`

func TestNoteNewCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ai")

	note, err := NewNote(dir, "context", "test-slug", "Test Title")
	if err != nil {
		t.Fatalf("new note: %v", err)
	}

	if note.Frontmatter.Type != "context" {
		t.Errorf("type: got %s, want context", note.Frontmatter.Type)
	}
	if note.Frontmatter.Status != "active" {
		t.Errorf("status: got %s, want active", note.Frontmatter.Status)
	}
	if note.Frontmatter.Title != "Test Title" {
		t.Errorf("title: got %s", note.Frontmatter.Title)
	}
	if !strings.Contains(note.Frontmatter.ID, "context-") {
		t.Errorf("id should contain type prefix: %s", note.Frontmatter.ID)
	}
	if !strings.Contains(note.Frontmatter.ID, "test-slug") {
		t.Errorf("id should contain slug: %s", note.Frontmatter.ID)
	}

	// Verify file exists and is parseable
	_, err = ParseNote(note.Path)
	if err != nil {
		t.Fatalf("parse created note: %v", err)
	}
}

func TestNoteCloseCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ai")
	note, _ := NewNote(dir, "investigation", "test", "Test")

	if err := CloseNote(note.Path, "closed"); err != nil {
		t.Fatal(err)
	}

	updated, _ := ParseNote(note.Path)
	if updated.Frontmatter.Status != "closed" {
		t.Errorf("status: got %s, want closed", updated.Frontmatter.Status)
	}
}

func TestNotePromoteCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ai")
	note, _ := NewNote(dir, "improvement", "auth", "Auth Improvement")

	if err := PromoteNote(note.Path, "docs/architecture/auth.md"); err != nil {
		t.Fatal(err)
	}

	updated, _ := ParseNote(note.Path)
	if updated.Frontmatter.Status != "promoted" {
		t.Errorf("status: got %s, want promoted", updated.Frontmatter.Status)
	}
	if updated.Frontmatter.PromotionTarget != "docs/architecture/auth.md" {
		t.Errorf("promotion_target: got %s", updated.Frontmatter.PromotionTarget)
	}
}

func TestNoteListCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ai")
	NewNote(dir, "context", "a", "Note A")
	NewNote(dir, "investigation", "b", "Note B")
	note3, _ := NewNote(dir, "context", "c", "Note C")
	CloseNote(note3.Path, "closed")

	t.Run("list all", func(t *testing.T) {
		all, err := ListNotes(dir, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != 3 {
			t.Errorf("expected 3 notes, got %d", len(all))
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		ctx, _ := ListNotes(dir, "", "context")
		if len(ctx) != 2 {
			t.Errorf("expected 2 context notes, got %d", len(ctx))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		active, _ := ListNotes(dir, "active", "")
		if len(active) != 2 {
			t.Errorf("expected 2 active notes, got %d", len(active))
		}
	})
}

func TestParseNote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.md")
	os.WriteFile(path, []byte(validNote), 0o644)

	note, err := ParseNote(path)
	if err != nil {
		t.Fatal(err)
	}

	if note.Frontmatter.ID != "context-20260220-auth-flow" {
		t.Errorf("id: got %s", note.Frontmatter.ID)
	}
	if note.Frontmatter.SchemaVersion != 1 {
		t.Errorf("schema_version: got %d", note.Frontmatter.SchemaVersion)
	}
	if !strings.Contains(note.Body, "Body content") {
		t.Error("body should contain content after frontmatter")
	}
}

func TestParseNoteInvalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"no frontmatter", "# Just a title\n"},
		{"unterminated", "---\nid: test\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "test.md")
			os.WriteFile(path, []byte(tt.content), 0o644)

			_, err := ParseNote(path)
			if err == nil {
				t.Error("expected parse error")
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID("context", "auth-flow")
	if !strings.HasPrefix(id, "context-") {
		t.Errorf("expected prefix context-, got %s", id)
	}
	if !strings.HasSuffix(id, "-auth-flow") {
		t.Errorf("expected suffix -auth-flow, got %s", id)
	}
}
