// Package changelog implements changelog generation from commit history.
package changelog

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/amr-athena/athena/internal/commitlint"
)

// Entry is a single changelog item.
type Entry struct {
	Type        string `json:"type"`
	Scope       string `json:"scope,omitempty"`
	Description string `json:"description"`
	Breaking    bool   `json:"breaking,omitempty"`
}

// Section groups entries under a category heading.
type Section struct {
	Category string  `json:"category"`
	Entries  []Entry `json:"entries"`
}

// ChangelogResult holds the generated changelog content.
type ChangelogResult struct {
	Version     string    `json:"version,omitempty"`
	Sections    []Section `json:"sections"`
	Markdown    string    `json:"markdown"`
	DryRun      bool      `json:"dry_run"`
	EntryCount  int       `json:"entry_count"`
}

// Options controls changelog generation.
type Options struct {
	Commits     []string // raw commit messages
	NextVersion string   // e.g. "1.4.0"
	DryRun      bool
	OutputPath  string // e.g. "CHANGELOG.md"
}

// categoryMap maps commit types to changelog sections.
var categoryMap = map[string]string{
	"feat":     "Features",
	"fix":      "Bug Fixes",
	"perf":     "Performance",
	"docs":     "Documentation",
	"refactor": "Refactoring",
	"test":     "Tests",
	"build":    "Build",
	"ci":       "CI",
	"chore":    "Chores",
	"style":    "Styles",
	"revert":   "Reverts",
}

// Generate creates a changelog from parsed commits.
func Generate(opts Options) (*ChangelogResult, error) {
	entries := parseCommits(opts.Commits)

	sections := groupByCategory(entries)

	version := opts.NextVersion
	if version == "" {
		version = "Unreleased"
	}

	md := renderMarkdown(version, sections)

	result := &ChangelogResult{
		Version:    version,
		Sections:   sections,
		Markdown:   md,
		DryRun:     opts.DryRun,
		EntryCount: len(entries),
	}

	if !opts.DryRun && opts.OutputPath != "" {
		if err := writeChangelog(opts.OutputPath, md); err != nil {
			return result, err
		}
	}

	return result, nil
}

func parseCommits(messages []string) []Entry {
	var entries []Entry
	for _, msg := range messages {
		parsed, err := commitlint.Parse(msg)
		if err != nil {
			continue // skip non-conventional commits
		}
		entries = append(entries, Entry{
			Type:        parsed.Type,
			Scope:       parsed.Scope,
			Description: parsed.Description,
			Breaking:    parsed.Breaking,
		})
	}
	return entries
}

func groupByCategory(entries []Entry) []Section {
	groups := make(map[string][]Entry)
	var breakingEntries []Entry

	for _, e := range entries {
		if e.Breaking {
			breakingEntries = append(breakingEntries, e)
		}
		cat := categoryMap[e.Type]
		if cat == "" {
			cat = "Other"
		}
		groups[cat] = append(groups[cat], e)
	}

	var sections []Section

	// Breaking changes first
	if len(breakingEntries) > 0 {
		sections = append(sections, Section{
			Category: "BREAKING CHANGES",
			Entries:  breakingEntries,
		})
	}

	// Sort categories for deterministic output
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, cat := range keys {
		sections = append(sections, Section{
			Category: cat,
			Entries:  groups[cat],
		})
	}

	return sections
}

func renderMarkdown(version string, sections []Section) string {
	var b strings.Builder

	date := time.Now().Format("2006-01-02")
	if version == "Unreleased" {
		b.WriteString("## Unreleased\n\n")
	} else {
		b.WriteString(fmt.Sprintf("## %s (%s)\n\n", version, date))
	}

	for _, sec := range sections {
		b.WriteString(fmt.Sprintf("### %s\n\n", sec.Category))
		for _, e := range sec.Entries {
			scope := ""
			if e.Scope != "" {
				scope = fmt.Sprintf("**%s:** ", e.Scope)
			}
			b.WriteString(fmt.Sprintf("- %s%s\n", scope, e.Description))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func writeChangelog(path, content string) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// If file exists, insert new content after the first line (# Changelog header)
	if len(existing) > 0 {
		lines := strings.SplitN(string(existing), "\n", 2)
		header := lines[0]
		rest := ""
		if len(lines) > 1 {
			rest = lines[1]
		}
		content = header + "\n\n" + content + rest
	} else {
		content = "# Changelog\n\n" + content
	}

	return os.WriteFile(path, []byte(content), 0o644)
}
