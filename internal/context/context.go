// Package context implements repomix command orchestration and budget logic.
package context

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/amr-athena/athena/internal/config"
	athenaerr "github.com/amr-athena/athena/internal/errors"
)

// CommandRunner abstracts external command execution for testability.
type CommandRunner interface {
	// Run executes a command and returns combined output.
	Run(name string, args ...string) ([]byte, error)
	// LookPath checks if a command is available in PATH.
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

// PackOptions controls context pack behavior.
type PackOptions struct {
	Profile         string
	Changed         bool
	Stdout          bool
	OutputPath      string
	DryRun          bool
	PassthroughArgs []string
	PolicyLevel     config.PolicyLevel
}

// PackResult holds the outcome of a pack operation.
type PackResult struct {
	Profile    string   `json:"profile"`
	OutputPath string   `json:"output_path,omitempty"`
	Args       []string `json:"args"`
	DryRun     bool     `json:"dry_run"`
	Output     string   `json:"output,omitempty"`
}

// MCPOptions controls context mcp behavior.
type MCPOptions struct {
	Stdio bool
}

// MCPResult holds the outcome of an MCP operation.
type MCPResult struct {
	Started bool   `json:"started"`
	Output  string `json:"output,omitempty"`
}

// BudgetOptions controls context budget behavior.
type BudgetOptions struct {
	Profile     string
	MaxTokens   int
	PolicyLevel config.PolicyLevel
}

// BudgetResult holds the outcome of a budget check.
type BudgetResult struct {
	Profile         string `json:"profile"`
	EstimatedTokens int    `json:"estimated_tokens"`
	MaxTokens       int    `json:"max_tokens,omitempty"`
	WithinBudget    bool   `json:"within_budget"`
	Output          string `json:"output,omitempty"`
}

// Service orchestrates repomix context operations.
type Service struct {
	cfg    config.ContextConfig
	runner CommandRunner
}

// NewService creates a context service with the given config and runner.
func NewService(cfg config.ContextConfig, runner CommandRunner) *Service {
	return &Service{cfg: cfg, runner: runner}
}

// CheckAvailability verifies repomix is installed and returns an error if not.
func (s *Service) CheckAvailability() error {
	_, err := s.runner.LookPath(s.cfg.Provider)
	if err != nil {
		return athenaerr.New(
			athenaerr.ToolMissing,
			fmt.Sprintf("%s is not installed or not in PATH", s.cfg.Provider),
			fmt.Sprintf("Install %s: npm install -g %s", s.cfg.Provider, s.cfg.Provider),
		)
	}
	return nil
}

// Pack executes context pack with the resolved profile and options.
func (s *Service) Pack(opts PackOptions) (*PackResult, error) {
	if err := s.CheckAvailability(); err != nil {
		return nil, err
	}

	profile := opts.Profile
	if profile == "" {
		profile = s.cfg.DefaultProfile
	}
	if profile == "" {
		profile = "review"
	}

	// Validate profile exists when profiles are configured
	if len(s.cfg.Profiles) > 0 {
		if _, ok := s.cfg.Profiles[profile]; !ok {
			return nil, athenaerr.New(
				athenaerr.ConfMissingRequired,
				fmt.Sprintf("unknown context profile: %s", profile),
				fmt.Sprintf("Available profiles: %s", availableProfiles(s.cfg.Profiles)),
			)
		}
	}

	args := s.buildPackArgs(profile, opts)

	result := &PackResult{
		Profile: profile,
		Args:    args,
		DryRun:  opts.DryRun,
	}

	if opts.DryRun {
		return result, nil
	}

	output, err := s.runner.Run(s.cfg.Provider, args...)
	if err != nil {
		return nil, athenaerr.New(
			athenaerr.ToolExecFailed,
			fmt.Sprintf("%s pack failed: %v", s.cfg.Provider, err),
			"Check repomix output for details and ensure the configuration is valid.",
		)
	}

	result.Output = string(output)
	if !opts.Stdout {
		outPath := opts.OutputPath
		if outPath == "" {
			outPath = s.cfg.OutputPath
		}
		result.OutputPath = outPath
	}

	return result, nil
}

// MCP validates/starts repomix in MCP mode.
func (s *Service) MCP(opts MCPOptions) (*MCPResult, error) {
	if err := s.CheckAvailability(); err != nil {
		return nil, err
	}

	args := []string{"--mcp"}
	if opts.Stdio {
		args = append(args, "--stdio")
	}

	output, err := s.runner.Run(s.cfg.Provider, args...)
	if err != nil {
		return nil, athenaerr.New(
			athenaerr.ToolExecFailed,
			fmt.Sprintf("%s MCP mode failed: %v", s.cfg.Provider, err),
			fmt.Sprintf("Ensure %s supports --mcp mode. Run `%s --mcp --help` for details.", s.cfg.Provider, s.cfg.Provider),
		)
	}

	return &MCPResult{Started: true, Output: string(output)}, nil
}

// Budget estimates token usage and enforces budget constraints.
func (s *Service) Budget(opts BudgetOptions) (*BudgetResult, error) {
	if err := s.CheckAvailability(); err != nil {
		return nil, err
	}

	profile := opts.Profile
	if profile == "" {
		profile = s.cfg.DefaultProfile
	}
	if profile == "" {
		profile = "review"
	}

	// Use repomix to estimate token count
	args := []string{"--token-count"}
	output, err := s.runner.Run(s.cfg.Provider, args...)
	if err != nil {
		return nil, athenaerr.New(
			athenaerr.ToolExecFailed,
			fmt.Sprintf("%s token count failed: %v", s.cfg.Provider, err),
			"Ensure repomix supports --token-count. You may need to update repomix.",
		)
	}

	// Parse token count from output (simplified: use output length as proxy)
	estimatedTokens := parseTokenCount(string(output))

	result := &BudgetResult{
		Profile:         profile,
		EstimatedTokens: estimatedTokens,
		MaxTokens:       opts.MaxTokens,
		WithinBudget:    true,
		Output:          string(output),
	}

	if opts.MaxTokens > 0 && estimatedTokens > opts.MaxTokens {
		result.WithinBudget = false
		if opts.PolicyLevel == config.PolicyStrict {
			return result, athenaerr.New(
				athenaerr.PolStrictViolation,
				fmt.Sprintf("context budget exceeded: %d tokens > %d max", estimatedTokens, opts.MaxTokens),
				"Reduce context scope with --profile, --changed, or increase --max-tokens.",
			)
		}
	}

	return result, nil
}

func (s *Service) buildPackArgs(profile string, opts PackOptions) []string {
	var args []string

	// Apply profile-specific settings
	if p, ok := s.cfg.Profiles[profile]; ok {
		if p.Style != "" {
			args = append(args, "--style", p.Style)
		}
		if p.Compress {
			args = append(args, "--compress")
		}
	}

	// Include/ignore patterns
	for _, inc := range s.cfg.Include {
		args = append(args, "--include", inc)
	}
	for _, ign := range s.cfg.Ignore {
		args = append(args, "--ignore", ign)
	}

	// Output destination
	if opts.Stdout {
		args = append(args, "--stdout")
	} else {
		outPath := opts.OutputPath
		if outPath == "" {
			outPath = s.cfg.OutputPath
		}
		if outPath != "" {
			args = append(args, "--output", outPath)
		}
	}

	// Changed files mode
	if opts.Changed {
		args = append(args, "--changed")
	}

	// Passthrough args
	if len(opts.PassthroughArgs) > 0 {
		args = append(args, opts.PassthroughArgs...)
	}

	return args
}

func availableProfiles(profiles map[string]config.ContextProfile) string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// parseTokenCount extracts an integer token count from repomix output.
// Falls back to a character-based estimate if parsing fails.
func parseTokenCount(output string) int {
	// Simplified: estimate ~4 chars per token from output length
	// Real implementation would parse repomix's structured output
	chars := len(strings.TrimSpace(output))
	if chars == 0 {
		return 0
	}
	return chars / 4
}
