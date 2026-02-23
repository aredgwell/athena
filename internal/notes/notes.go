// Package notes implements note lifecycle: new, close, promote, list.
package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Valid note types.
var ValidTypes = []string{"context", "investigation", "troubleshooting", "wip", "improvement", "session", "memory"}

// Valid note statuses.
var ValidStatuses = []string{"active", "closed", "stale", "superseded", "promoted"}

// CurrentFrontmatterVersion is the latest frontmatter schema version.
const CurrentFrontmatterVersion = 1

// Frontmatter is the YAML frontmatter contract for all .ai/ notes.
type Frontmatter struct {
	ID              string   `yaml:"id"`
	Title           string   `yaml:"title"`
	Type            string   `yaml:"type"`
	Status          string   `yaml:"status"`
	Created         string   `yaml:"created"`
	Updated         string   `yaml:"updated"`
	SchemaVersion   int      `yaml:"schema_version,omitempty"`
	Related         []string `yaml:"related,omitempty"`
	PromotionTarget string   `yaml:"promotion_target,omitempty"`
	Supersedes      []string `yaml:"supersedes,omitempty"`
	Tags            []string `yaml:"tags,omitempty"`
}

// Note represents a parsed note file.
type Note struct {
	Path        string
	Frontmatter Frontmatter
	Body        string
}

// ParseNote reads and parses a note file.
func ParseNote(path string) (*Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseNoteContent(path, data)
}

// ParseNoteContent parses note content from bytes.
func ParseNoteContent(path string, data []byte) (*Note, error) {
	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("missing frontmatter delimiter in %s", path)
	}

	endIdx := strings.Index(content[4:], "\n---")
	if endIdx == -1 {
		return nil, fmt.Errorf("unterminated frontmatter in %s", path)
	}

	fmRaw := content[4 : 4+endIdx]
	body := content[4+endIdx+4:] // skip closing ---\n

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}

	return &Note{Path: path, Frontmatter: fm, Body: body}, nil
}

// GenerateID creates a note ID from type, date, and slug.
func GenerateID(noteType, slug string) string {
	date := time.Now().Format("20060102")
	return fmt.Sprintf("%s-%s-%s", noteType, date, slug)
}

// RenderNote creates markdown content with frontmatter.
func RenderNote(fm Frontmatter, body string) []byte {
	fmBytes, _ := yaml.Marshal(fm)
	return []byte(fmt.Sprintf("---\n%s---\n%s", string(fmBytes), body))
}

// NewNote creates a new note file.
func NewNote(dir, noteType, slug, title string) (*Note, error) {
	id := GenerateID(noteType, slug)
	now := time.Now().Format("2006-01-02")

	fm := Frontmatter{
		ID:            id,
		Title:         title,
		Type:          noteType,
		Status:        "active",
		Created:       now,
		Updated:       now,
		SchemaVersion: CurrentFrontmatterVersion,
	}

	// Determine subdirectory from type
	subDir := filepath.Join(dir, noteType)
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%s.md", id)
	path := filepath.Join(subDir, filename)

	body := fmt.Sprintf("\n# %s\n\n", title)
	content := RenderNote(fm, body)

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return nil, err
	}

	return &Note{Path: path, Frontmatter: fm, Body: body}, nil
}

// CloseNote transitions a note's status.
func CloseNote(path, status string) error {
	note, err := ParseNote(path)
	if err != nil {
		return err
	}

	note.Frontmatter.Status = status
	note.Frontmatter.Updated = time.Now().Format("2006-01-02")

	content := RenderNote(note.Frontmatter, note.Body)
	return os.WriteFile(path, content, 0o644)
}

// PromoteNote marks a note as promoted with a target path.
func PromoteNote(path, target string) error {
	note, err := ParseNote(path)
	if err != nil {
		return err
	}

	note.Frontmatter.Status = "promoted"
	note.Frontmatter.PromotionTarget = target
	note.Frontmatter.Updated = time.Now().Format("2006-01-02")

	content := RenderNote(note.Frontmatter, note.Body)
	return os.WriteFile(path, content, 0o644)
}

// ListNotes finds notes in a directory with optional filters.
func ListNotes(dir string, statusFilter, typeFilter string) ([]*Note, error) {
	var notes []*Note

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		note, parseErr := ParseNote(path)
		if parseErr != nil {
			return nil // skip unparseable files
		}

		if statusFilter != "" && note.Frontmatter.Status != statusFilter {
			return nil
		}
		if typeFilter != "" && note.Frontmatter.Type != typeFilter {
			return nil
		}

		notes = append(notes, note)
		return nil
	})

	return notes, err
}

// Contains checks if a string is in a slice.
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
