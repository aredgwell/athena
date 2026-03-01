package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aredgwell/athena/internal/notes"
)

func setupNotes(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".ai")
	notes.NewNote(dir, "context", "a", "Note A")
	notes.NewNote(dir, "investigation", "b", "Note B")
	n3, _ := notes.NewNote(dir, "context", "c", "Note C")
	notes.CloseNote(n3.Path, "closed")
	return dir
}

func TestIndexCommand(t *testing.T) {
	dir := setupNotes(t)

	idx, err := Build(dir)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	if idx.Version != 1 {
		t.Errorf("version: got %d, want 1", idx.Version)
	}
	if idx.Counts.Total != 3 {
		t.Errorf("total: got %d, want 3", idx.Counts.Total)
	}
	if idx.Counts.Active != 2 {
		t.Errorf("active: got %d, want 2", idx.Counts.Active)
	}
	if idx.Counts.Closed != 1 {
		t.Errorf("closed: got %d, want 1", idx.Counts.Closed)
	}
	if len(idx.Entries) != 3 {
		t.Errorf("entries: got %d, want 3", len(idx.Entries))
	}
}

func TestIndexWrite(t *testing.T) {
	dir := setupNotes(t)
	idx, _ := Build(dir)

	outPath := filepath.Join(t.TempDir(), ".ai", "index.yaml")
	if err := Write(idx, outPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("index file should not be empty")
	}
}

func TestBuildSearch(t *testing.T) {
	dir := setupNotes(t)
	idx, err := BuildSearch(dir)
	if err != nil {
		t.Fatalf("BuildSearch: %v", err)
	}
	if idx.DocCount != 3 {
		t.Errorf("doc count: got %d, want 3", idx.DocCount)
	}
	if idx.Version != 1 {
		t.Errorf("version: got %d, want 1", idx.Version)
	}
	if len(idx.Documents) != 3 {
		t.Errorf("documents: got %d, want 3", len(idx.Documents))
	}
}

func TestBuildSearchEmpty(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ai")
	os.MkdirAll(dir, 0o755)
	idx, err := BuildSearch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if idx.DocCount != 0 {
		t.Errorf("doc count: got %d, want 0", idx.DocCount)
	}
}

func TestIndexEmpty(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ai")
	os.MkdirAll(dir, 0o755)

	idx, err := Build(dir)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Counts.Total != 0 {
		t.Errorf("total: got %d, want 0", idx.Counts.Total)
	}
}
