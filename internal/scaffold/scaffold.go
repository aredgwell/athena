// Package scaffold implements init and upgrade logic with checksums, backups, and conflict handling.
package scaffold

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ConflictResolution describes how to handle a file collision during init.
type ConflictResolution string

const (
	ResolutionOverwrite       ConflictResolution = "overwrite"
	ResolutionSkip            ConflictResolution = "skip"
	ResolutionBackupOverwrite ConflictResolution = "backup-and-overwrite"
)

// ChecksumFile is the .athena/checksums.json format.
type ChecksumFile struct {
	Version          int               `json:"version"`
	InstalledVersion string            `json:"installed_version"`
	Files            map[string]string `json:"files"`
}

// ManagedFile represents a file to be scaffolded.
type ManagedFile struct {
	Path    string // relative path in the repo
	Content []byte // rendered content
	Feature string // feature flag that gates this file
}

// ActionType describes what happened to a file during init/upgrade.
type ActionType string

const (
	ActionWritten     ActionType = "written"
	ActionOverwritten ActionType = "overwritten"
	ActionSkipped     ActionType = "skipped"
	ActionBackedUp    ActionType = "backed_up"
)

// FileAction records the outcome for a single file.
type FileAction struct {
	Path   string     `json:"path"`
	Action ActionType `json:"action"`
	Reason string     `json:"reason,omitempty"`
}

// Summary records the totals from an init or upgrade operation.
type Summary struct {
	Written     int          `json:"written"`
	Overwritten int          `json:"overwritten"`
	Skipped     int          `json:"skipped"`
	BackedUp    int          `json:"backed_up"`
	Actions     []FileAction `json:"actions"`
}

// HashContent returns "sha256:<hex>" for the given data.
func HashContent(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h[:])
}

// LoadChecksums reads the checksums file from a repo root.
func LoadChecksums(repoRoot string) (*ChecksumFile, error) {
	path := filepath.Join(repoRoot, ".athena", "checksums.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cf ChecksumFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, err
	}
	return &cf, nil
}

// SaveChecksums writes the checksums file to a repo root.
func SaveChecksums(repoRoot string, cf *ChecksumFile) error {
	dir := filepath.Join(repoRoot, ".athena")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "checksums.json"), data, 0o644)
}

// BackupFile creates a backup of a file under .athena/backups/.
func BackupFile(repoRoot, relPath string) (string, error) {
	src := filepath.Join(repoRoot, relPath)
	backupDir := filepath.Join(repoRoot, ".athena", "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}
	ts := time.Now().UTC().Format("20060102T150405")
	backupName := fmt.Sprintf("%s.%s.bak", filepath.Base(relPath), ts)
	backupPath := filepath.Join(backupDir, backupName)

	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	return backupPath, os.WriteFile(backupPath, data, 0o644)
}

// InitOptions controls init behavior.
type InitOptions struct {
	RepoRoot string
	DryRun   bool
	Force    bool
	IsTTY    bool
	Version  string
	// ResolveConflict is called for each collision in TTY mode.
	// If nil and TTY, defaults to skip.
	ResolveConflict func(path string) ConflictResolution
}

// Init scaffolds managed files into a repository.
func Init(files []ManagedFile, opts InitOptions) (*Summary, error) {
	summary := &Summary{}

	// Load existing checksums (may be nil)
	existing, _ := LoadChecksums(opts.RepoRoot)
	checksums := &ChecksumFile{
		Version:          1,
		InstalledVersion: opts.Version,
		Files:            make(map[string]string),
	}
	if existing != nil {
		checksums = existing
	}

	for _, mf := range files {
		targetPath := filepath.Join(opts.RepoRoot, mf.Path)
		_, fileExists := statFile(targetPath)

		if !fileExists {
			// No collision — write the file
			if !opts.DryRun {
				if err := writeFile(targetPath, mf.Content); err != nil {
					return summary, fmt.Errorf("writing %s: %w", mf.Path, err)
				}
				checksums.Files[mf.Path] = HashContent(mf.Content)
			}
			summary.Written++
			summary.Actions = append(summary.Actions, FileAction{Path: mf.Path, Action: ActionWritten})
			continue
		}

		// Collision exists
		resolution := resolveCollision(mf.Path, opts)

		switch resolution {
		case ResolutionOverwrite:
			if !opts.DryRun {
				if err := writeFile(targetPath, mf.Content); err != nil {
					return summary, err
				}
				checksums.Files[mf.Path] = HashContent(mf.Content)
			}
			summary.Overwritten++
			summary.Actions = append(summary.Actions, FileAction{Path: mf.Path, Action: ActionOverwritten})

		case ResolutionBackupOverwrite:
			if !opts.DryRun {
				if _, err := BackupFile(opts.RepoRoot, mf.Path); err != nil {
					return summary, fmt.Errorf("backing up %s: %w", mf.Path, err)
				}
				if err := writeFile(targetPath, mf.Content); err != nil {
					return summary, err
				}
				checksums.Files[mf.Path] = HashContent(mf.Content)
			}
			summary.BackedUp++
			summary.Overwritten++
			summary.Actions = append(summary.Actions, FileAction{Path: mf.Path, Action: ActionBackedUp})

		case ResolutionSkip:
			summary.Skipped++
			summary.Actions = append(summary.Actions, FileAction{Path: mf.Path, Action: ActionSkipped, Reason: "collision"})
		}
	}

	if !opts.DryRun {
		if err := SaveChecksums(opts.RepoRoot, checksums); err != nil {
			return summary, fmt.Errorf("saving checksums: %w", err)
		}
	}

	return summary, nil
}

// UpgradeOptions controls upgrade behavior.
type UpgradeOptions struct {
	RepoRoot string
	DryRun   bool
	Version  string
}

// Upgrade updates managed files, skipping user-modified ones.
func Upgrade(files []ManagedFile, opts UpgradeOptions) (*Summary, error) {
	summary := &Summary{}

	checksums, err := LoadChecksums(opts.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("loading checksums: %w", err)
	}
	if checksums == nil {
		return nil, fmt.Errorf("checksums.json missing: run athena init first")
	}

	for _, mf := range files {
		targetPath := filepath.Join(opts.RepoRoot, mf.Path)
		_, fileExists := statFile(targetPath)

		if !fileExists {
			// File doesn't exist — write it
			if !opts.DryRun {
				if err := writeFile(targetPath, mf.Content); err != nil {
					return summary, err
				}
				checksums.Files[mf.Path] = HashContent(mf.Content)
			}
			summary.Written++
			summary.Actions = append(summary.Actions, FileAction{Path: mf.Path, Action: ActionWritten})
			continue
		}

		// File exists — check if user-modified
		storedHash, hasHash := checksums.Files[mf.Path]
		if !hasHash {
			// Not tracked — skip
			summary.Skipped++
			summary.Actions = append(summary.Actions, FileAction{Path: mf.Path, Action: ActionSkipped, Reason: "not tracked"})
			continue
		}

		currentData, err := os.ReadFile(targetPath)
		if err != nil {
			return summary, err
		}
		currentHash := HashContent(currentData)

		if currentHash != storedHash {
			// User-modified — skip
			summary.Skipped++
			summary.Actions = append(summary.Actions, FileAction{Path: mf.Path, Action: ActionSkipped, Reason: "user-modified"})
			continue
		}

		// Unmodified — backup and overwrite
		if !opts.DryRun {
			if _, err := BackupFile(opts.RepoRoot, mf.Path); err != nil {
				return summary, err
			}
			if err := writeFile(targetPath, mf.Content); err != nil {
				return summary, err
			}
			checksums.Files[mf.Path] = HashContent(mf.Content)
		}
		summary.BackedUp++
		summary.Overwritten++
		summary.Actions = append(summary.Actions, FileAction{Path: mf.Path, Action: ActionOverwritten})
	}

	if !opts.DryRun {
		checksums.InstalledVersion = opts.Version
		if err := SaveChecksums(opts.RepoRoot, checksums); err != nil {
			return summary, err
		}
	}

	return summary, nil
}

func resolveCollision(path string, opts InitOptions) ConflictResolution {
	if opts.IsTTY && opts.ResolveConflict != nil {
		return opts.ResolveConflict(path)
	}
	if opts.Force {
		return ResolutionBackupOverwrite
	}
	return ResolutionSkip
}

func statFile(path string) (os.FileInfo, bool) {
	info, err := os.Stat(path)
	return info, err == nil
}

func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
