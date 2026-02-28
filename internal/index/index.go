// Package index implements .ai/index.yaml generation.
package index

import (
	"os"
	"path/filepath"
	"time"

	"github.com/amr-athena/athena/internal/notes"
	"github.com/amr-athena/athena/internal/search"
	"gopkg.in/yaml.v3"
)

// Index is the .ai/index.yaml structure.
type Index struct {
	Version   int     `yaml:"version"`
	Generated string  `yaml:"generated"`
	Counts    Counts  `yaml:"counts"`
	Entries   []Entry `yaml:"entries"`
}

// Counts summarizes note statuses.
type Counts struct {
	Total  int `yaml:"total"`
	Active int `yaml:"active"`
	Stale  int `yaml:"stale"`
	Closed int `yaml:"closed"`
}

// Entry is a single note in the index.
type Entry struct {
	Path    string `yaml:"path"`
	Type    string `yaml:"type"`
	Status  string `yaml:"status"`
	Updated string `yaml:"updated"`
	Title   string `yaml:"title"`
}

// Build generates an index from notes in the given directory.
func Build(notesDir string) (*Index, error) {
	allNotes, err := notes.ListNotes(notesDir, "", "")
	if err != nil {
		return nil, err
	}

	idx := &Index{
		Version:   1,
		Generated: time.Now().UTC().Format(time.RFC3339),
	}

	for _, n := range allNotes {
		relPath, _ := filepath.Rel(filepath.Dir(notesDir), n.Path)
		if relPath == "" {
			relPath = n.Path
		}

		entry := Entry{
			Path:    relPath,
			Type:    n.Frontmatter.Type,
			Status:  n.Frontmatter.Status,
			Updated: n.Frontmatter.Updated,
			Title:   n.Frontmatter.Title,
		}
		idx.Entries = append(idx.Entries, entry)

		idx.Counts.Total++
		switch n.Frontmatter.Status {
		case "active":
			idx.Counts.Active++
		case "stale":
			idx.Counts.Stale++
		case "closed":
			idx.Counts.Closed++
		}
	}

	return idx, nil
}

// BuildSearch generates a BM25 search index from notes in the given directory.
func BuildSearch(notesDir string) (*search.SearchIndex, error) {
	allNotes, err := notes.ListNotes(notesDir, "", "")
	if err != nil {
		return nil, err
	}

	docs := make([]search.IndexableDoc, 0, len(allNotes))
	for _, n := range allNotes {
		relPath, _ := filepath.Rel(filepath.Dir(notesDir), n.Path)
		if relPath == "" {
			relPath = n.Path
		}
		docs = append(docs, search.IndexableDoc{
			Path:   relPath,
			Title:  n.Frontmatter.Title,
			Type:   n.Frontmatter.Type,
			Status: n.Frontmatter.Status,
			Tags:   n.Frontmatter.Tags,
			Body:   n.Body,
		})
	}

	return search.BuildIndex(docs), nil
}

// Write writes the index as YAML to the given path.
func Write(idx *Index, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(idx)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
