// Package commitlint implements conventional commit parsing and linting.
package commitlint

import (
	"fmt"
	"regexp"
	"strings"
)

// conventionalPattern matches: type(scope): description
// or: type: description
// or: type(scope)!: description (breaking change)
var conventionalPattern = regexp.MustCompile(
	`^(?P<type>\w+)(?:\((?P<scope>[^)]+)\))?(?P<breaking>!)?:\s+(?P<description>.+)$`,
)

// ParsedCommit holds the parsed components of a conventional commit.
type ParsedCommit struct {
	Type        string `json:"type"`
	Scope       string `json:"scope,omitempty"`
	Description string `json:"description"`
	Breaking    bool   `json:"breaking"`
	Body        string `json:"body,omitempty"`
	Footer      string `json:"footer,omitempty"`
}

// LintResult holds the outcome of linting a single commit message.
type LintResult struct {
	Message string   `json:"message"`
	Valid   bool     `json:"valid"`
	Parsed  *ParsedCommit `json:"parsed,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

// LintOptions controls commit lint behavior.
type LintOptions struct {
	ValidTypes  []string
	ValidScopes []string // empty means any scope allowed
	RequireScope bool
}

// LintSummary holds the aggregate outcome of linting multiple commits.
type LintSummary struct {
	Total   int          `json:"total"`
	Valid   int          `json:"valid"`
	Invalid int          `json:"invalid"`
	Results []LintResult `json:"results"`
}

// DefaultTypes returns the standard conventional commit types.
func DefaultTypes() []string {
	return []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"}
}

// Parse parses a single commit message into its conventional components.
func Parse(message string) (*ParsedCommit, error) {
	lines := strings.SplitN(message, "\n", 2)
	header := strings.TrimSpace(lines[0])

	match := conventionalPattern.FindStringSubmatch(header)
	if match == nil {
		return nil, fmt.Errorf("not a conventional commit: %s", header)
	}

	result := conventionalPattern.SubexpNames()
	parsed := &ParsedCommit{}
	for i, name := range result {
		switch name {
		case "type":
			parsed.Type = match[i]
		case "scope":
			parsed.Scope = match[i]
		case "breaking":
			parsed.Breaking = match[i] == "!"
		case "description":
			parsed.Description = match[i]
		}
	}

	// Parse body and footer
	if len(lines) > 1 {
		rest := strings.TrimSpace(lines[1])
		parts := strings.SplitN(rest, "\n\n", 2)
		parsed.Body = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			parsed.Footer = strings.TrimSpace(parts[1])
		}

		// Check for BREAKING CHANGE footer
		if strings.Contains(rest, "BREAKING CHANGE:") || strings.Contains(rest, "BREAKING-CHANGE:") {
			parsed.Breaking = true
		}
	}

	return parsed, nil
}

// Lint validates a single commit message against the given options.
func Lint(message string, opts LintOptions) LintResult {
	result := LintResult{
		Message: strings.TrimSpace(strings.SplitN(message, "\n", 2)[0]),
		Valid:   true,
	}

	parsed, err := Parse(message)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	result.Parsed = parsed

	// Validate type
	if len(opts.ValidTypes) > 0 {
		found := false
		for _, t := range opts.ValidTypes {
			if t == parsed.Type {
				found = true
				break
			}
		}
		if !found {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("invalid type %q, allowed: %s", parsed.Type, strings.Join(opts.ValidTypes, ", ")))
		}
	}

	// Validate scope requirement
	if opts.RequireScope && parsed.Scope == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "scope is required")
	}

	// Validate scope against allowed list
	if parsed.Scope != "" && len(opts.ValidScopes) > 0 {
		found := false
		for _, s := range opts.ValidScopes {
			if s == parsed.Scope {
				found = true
				break
			}
		}
		if !found {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("invalid scope %q, allowed: %s", parsed.Scope, strings.Join(opts.ValidScopes, ", ")))
		}
	}

	return result
}

// LintAll validates multiple commit messages.
func LintAll(messages []string, opts LintOptions) *LintSummary {
	summary := &LintSummary{Total: len(messages)}

	for _, msg := range messages {
		result := Lint(msg, opts)
		summary.Results = append(summary.Results, result)
		if result.Valid {
			summary.Valid++
		} else {
			summary.Invalid++
		}
	}

	return summary
}
