// Package validate implements frontmatter validation and schema migration checks.
package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aredgwell/athena/internal/notes"
)

// ValidationResult holds the outcome of validating a single note.
type ValidationResult struct {
	Path       string   `json:"path"`
	Valid      bool     `json:"valid"`
	Migratable bool     `json:"migratable"`
	Fixed      bool     `json:"fixed"`
	Errors     []string `json:"errors,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
}

// CheckSummary is the aggregate result of a check operation.
type CheckSummary struct {
	FilesScanned int `json:"files_scanned"`
	Valid        int `json:"valid"`
	Invalid      int `json:"invalid"`
	Migratable   int `json:"migratable"`
	Fixed        int `json:"fixed"`
}

// CheckOptions controls check behavior.
type CheckOptions struct {
	Dir          string
	Fix          bool
	StrictSchema bool
	BackupDir    string // for fix mode backups
}

// Check validates all notes in a directory.
func Check(opts CheckOptions) ([]ValidationResult, *CheckSummary, error) {
	var results []ValidationResult
	summary := &CheckSummary{}

	err := filepath.Walk(opts.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		result := validateNote(path, opts)
		results = append(results, result)
		summary.FilesScanned++

		if result.Valid {
			summary.Valid++
		} else {
			summary.Invalid++
		}
		if result.Migratable {
			summary.Migratable++
		}
		if result.Fixed {
			summary.Fixed++
		}

		return nil
	})

	return results, summary, err
}

func validateNote(path string, opts CheckOptions) ValidationResult {
	result := ValidationResult{Path: path, Valid: true}

	note, err := notes.ParseNote(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %s", err))
		return result
	}

	fm := note.Frontmatter

	// Required fields
	if fm.ID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing required field: id")
	}
	if fm.Title == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing required field: title")
	}
	if fm.Type == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing required field: type")
	} else if !notes.Contains(notes.ValidTypes, fm.Type) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid type: %s", fm.Type))
	}
	if fm.Status == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing required field: status")
	} else if !notes.Contains(notes.ValidStatuses, fm.Status) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid status: %s", fm.Status))
	}
	if fm.Created == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing required field: created")
	}
	if fm.Updated == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "missing required field: updated")
	}

	// Schema version migration check
	schemaVer := fm.SchemaVersion
	if schemaVer == 0 {
		schemaVer = 1 // default when absent
	}

	if schemaVer < notes.CurrentFrontmatterVersion {
		result.Migratable = true
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("schema_version %d can be migrated to %d", schemaVer, notes.CurrentFrontmatterVersion))

		if opts.Fix {
			if err := migrateNote(path, note, opts); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("migration failed: %s", err))
			} else {
				result.Fixed = true
			}
		}
	}

	if opts.StrictSchema && fm.SchemaVersion != notes.CurrentFrontmatterVersion {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("strict schema: version %d != latest %d", fm.SchemaVersion, notes.CurrentFrontmatterVersion))
	}

	return result
}

func migrateNote(path string, note *notes.Note, opts CheckOptions) error {
	// Backup before modifying
	if opts.BackupDir != "" {
		if err := os.MkdirAll(opts.BackupDir, 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ts := time.Now().UTC().Format("20060102T150405")
		backupPath := filepath.Join(opts.BackupDir, fmt.Sprintf("%s.%s.bak", filepath.Base(path), ts))
		if err := os.WriteFile(backupPath, data, 0o644); err != nil {
			return err
		}
	}

	// Apply migration: set schema_version to latest
	note.Frontmatter.SchemaVersion = notes.CurrentFrontmatterVersion
	note.Frontmatter.Updated = time.Now().Format("2006-01-02")

	content := notes.RenderNote(note.Frontmatter, note.Body)
	return os.WriteFile(path, content, 0o644)
}
