// Package security implements gitleaks/actionlint wrappers and report persistence.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aredgwell/athena/internal/config"
	athenaerr "github.com/aredgwell/athena/internal/errors"
)

// CommandRunner abstracts external command execution for testability.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
	LookPath(name string) (string, error)
}

// ExecRunner is the real implementation using os/exec.
type ExecRunner struct{}

func (ExecRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func (ExecRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// ToolResult holds the outcome of a single tool scan.
type ToolResult struct {
	Status   string `json:"status"`
	Findings int    `json:"findings"`
	Detail   string `json:"detail,omitempty"`
}

// ScanOptions controls security scan behavior.
type ScanOptions struct {
	Secrets      bool
	Workflows    bool
	ReportFormat string // "json" or "sarif"
	ReportDir    string
	PolicyLevel  config.PolicyLevel
}

// ScanResult holds the aggregate scan outcome.
type ScanResult struct {
	OK       bool                  `json:"ok"`
	Tools    map[string]ToolResult `json:"tools"`
	Warnings []string              `json:"warnings,omitempty"`
	Errors   []string              `json:"errors,omitempty"`
}

// Service orchestrates security scanning operations.
type Service struct {
	cfg    config.SecurityConfig
	runner CommandRunner
}

// NewService creates a security service with the given config and runner.
func NewService(cfg config.SecurityConfig, runner CommandRunner) *Service {
	return &Service{cfg: cfg, runner: runner}
}

// Scan runs configured security scans with policy-aware severity handling.
func (s *Service) Scan(opts ScanOptions) (*ScanResult, error) {
	result := &ScanResult{
		OK:    true,
		Tools: make(map[string]ToolResult),
	}

	// If neither flag is supplied, run both when enabled in config
	runSecrets := opts.Secrets || (!opts.Secrets && !opts.Workflows && s.cfg.EnableSecretsScan)
	runWorkflows := opts.Workflows || (!opts.Secrets && !opts.Workflows && s.cfg.EnableWorkflowLint)

	if runSecrets {
		s.runGitleaks(result, opts)
	}

	if runWorkflows {
		s.runActionlint(result, opts)
	}

	// Write report artifacts
	if opts.ReportDir != "" && opts.ReportFormat != "" {
		if err := s.writeReport(result, opts); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (s *Service) runGitleaks(result *ScanResult, opts ScanOptions) {
	_, err := s.runner.LookPath("gitleaks")
	if err != nil {
		s.handleMissingTool(result, "gitleaks", opts.PolicyLevel)
		return
	}

	output, err := s.runner.Run("gitleaks", "detect", "--no-git", "--report-format", "json")
	if err != nil {
		// gitleaks returns non-zero when findings exist
		findings := countJSONFindings(output)
		result.Tools["gitleaks"] = ToolResult{Status: "fail", Findings: findings}
		result.OK = false
		result.Errors = append(result.Errors, "gitleaks found blocking secrets")
		return
	}

	result.Tools["gitleaks"] = ToolResult{Status: "pass", Findings: 0}
}

func (s *Service) runActionlint(result *ScanResult, opts ScanOptions) {
	_, err := s.runner.LookPath("actionlint")
	if err != nil {
		s.handleMissingTool(result, "actionlint", opts.PolicyLevel)
		return
	}

	output, err := s.runner.Run("actionlint", "-format", "json")
	if err != nil {
		findings := countJSONFindings(output)
		result.Tools["actionlint"] = ToolResult{Status: "fail", Findings: findings}
		if opts.PolicyLevel != config.PolicyLenient {
			result.OK = false
			result.Errors = append(result.Errors, "actionlint found workflow issues")
		}
		return
	}

	result.Tools["actionlint"] = ToolResult{Status: "pass", Findings: 0}
}

func (s *Service) handleMissingTool(result *ScanResult, tool string, policy config.PolicyLevel) {
	detail := fmt.Sprintf("%s not installed", tool)
	switch policy {
	case config.PolicyStrict:
		result.Tools[tool] = ToolResult{Status: "fail", Detail: detail}
		result.OK = false
		result.Errors = append(result.Errors, fmt.Sprintf("missing required tool: %s", tool))
	default:
		result.Tools[tool] = ToolResult{Status: "skip", Detail: detail}
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s not available, skipping", tool))
	}
}

func (s *Service) writeReport(result *ScanResult, opts ScanOptions) error {
	if err := os.MkdirAll(opts.ReportDir, 0o755); err != nil {
		return fmt.Errorf("creating report dir: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	var filename string
	switch opts.ReportFormat {
	case "sarif":
		filename = "security-scan.sarif.json"
	default:
		filename = "security-scan.json"
	}

	path := filepath.Join(opts.ReportDir, filename)
	return os.WriteFile(path, data, 0o644)
}

// countJSONFindings attempts to parse JSON array output and return length.
func countJSONFindings(data []byte) int {
	var items []json.RawMessage
	if json.Unmarshal(data, &items) == nil {
		return len(items)
	}
	// If it's not a JSON array, count as 1 finding if there's any output
	if len(data) > 0 {
		return 1
	}
	return 0
}

// CheckTool verifies a specific security tool is available and returns a structured error if not.
func CheckTool(runner CommandRunner, tool string) error {
	_, err := runner.LookPath(tool)
	if err != nil {
		return athenaerr.New(
			athenaerr.ToolMissing,
			fmt.Sprintf("%s is not installed or not in PATH", tool),
			fmt.Sprintf("Install %s and ensure it is in your PATH.", tool),
		)
	}
	return nil
}
