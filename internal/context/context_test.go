package context

import (
	"fmt"
	"testing"

	"github.com/amr-athena/athena/internal/config"
)

// mockRunner simulates external command execution.
type mockRunner struct {
	available  bool
	runOutput  string
	runErr     error
}

func (m *mockRunner) Run(name string, args ...string) ([]byte, error) {
	if m.runErr != nil {
		return nil, m.runErr
	}
	return []byte(m.runOutput), nil
}

func (m *mockRunner) LookPath(name string) (string, error) {
	if !m.available {
		return "", fmt.Errorf("not found: %s", name)
	}
	return "/usr/local/bin/" + name, nil
}

func defaultCfg() config.ContextConfig {
	return config.ContextConfig{
		Provider:       "repomix",
		DefaultProfile: "review",
		OutputPath:     ".athena/context/pack.xml",
		Compress:       true,
		SecurityCheck:  true,
		Include:        []string{"**/*.go", "**/*.md"},
		Ignore:         []string{".git/**", "node_modules/**"},
		Profiles: map[string]config.ContextProfile{
			"review":  {Style: "xml", Compress: true, StripDiff: true},
			"handoff": {Style: "markdown", Compress: true, StripDiff: false},
			"release": {Style: "plain", Compress: false, StripDiff: false},
		},
	}
}

func TestCheckAvailability(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{available: true})
		if err := svc.CheckAvailability(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{available: false})
		err := svc.CheckAvailability()
		if err == nil {
			t.Fatal("expected error for missing tool")
		}
		if err.Error() != "[ATHENA-TOOL-001] repomix is not installed or not in PATH" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestContextPackCommand(t *testing.T) {
	t.Run("default profile", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "packed context output",
		})

		result, err := svc.Pack(PackOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if result.Profile != "review" {
			t.Errorf("profile: got %s, want review", result.Profile)
		}
		if result.OutputPath != ".athena/context/pack.xml" {
			t.Errorf("output: got %s, want .athena/context/pack.xml", result.OutputPath)
		}
	})

	t.Run("explicit profile", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "handoff output",
		})

		result, err := svc.Pack(PackOptions{Profile: "handoff"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Profile != "handoff" {
			t.Errorf("profile: got %s, want handoff", result.Profile)
		}
	})

	t.Run("unknown profile", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{available: true})

		_, err := svc.Pack(PackOptions{Profile: "nonexistent"})
		if err == nil {
			t.Fatal("expected error for unknown profile")
		}
	})

	t.Run("dry run", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{available: true})

		result, err := svc.Pack(PackOptions{DryRun: true})
		if err != nil {
			t.Fatal(err)
		}
		if !result.DryRun {
			t.Error("dry_run should be true")
		}
		if result.Output != "" {
			t.Error("dry run should not produce output")
		}
	})

	t.Run("stdout mode", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "stdout content",
		})

		result, err := svc.Pack(PackOptions{Stdout: true})
		if err != nil {
			t.Fatal(err)
		}
		if result.OutputPath != "" {
			t.Errorf("stdout mode should have empty output_path, got %s", result.OutputPath)
		}
	})

	t.Run("custom output path", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "custom output",
		})

		result, err := svc.Pack(PackOptions{OutputPath: "/tmp/custom.xml"})
		if err != nil {
			t.Fatal(err)
		}
		if result.OutputPath != "/tmp/custom.xml" {
			t.Errorf("output: got %s, want /tmp/custom.xml", result.OutputPath)
		}
	})

	t.Run("changed files mode", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "changed output",
		})

		result, err := svc.Pack(PackOptions{Changed: true, DryRun: true})
		if err != nil {
			t.Fatal(err)
		}
		hasChanged := false
		for _, a := range result.Args {
			if a == "--changed" {
				hasChanged = true
			}
		}
		if !hasChanged {
			t.Errorf("args should contain --changed: %v", result.Args)
		}
	})

	t.Run("passthrough args", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "output",
		})

		result, err := svc.Pack(PackOptions{
			DryRun:          true,
			PassthroughArgs: []string{"--extra", "flag"},
		})
		if err != nil {
			t.Fatal(err)
		}
		hasExtra := false
		for _, a := range result.Args {
			if a == "--extra" {
				hasExtra = true
			}
		}
		if !hasExtra {
			t.Errorf("args should contain passthrough --extra: %v", result.Args)
		}
	})

	t.Run("repomix failure", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runErr:    fmt.Errorf("exit status 1"),
		})

		_, err := svc.Pack(PackOptions{})
		if err == nil {
			t.Fatal("expected error on repomix failure")
		}
	})

	t.Run("graceful degradation when missing", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{available: false})

		_, err := svc.Pack(PackOptions{})
		if err == nil {
			t.Fatal("expected error when repomix missing")
		}
		errMsg := err.Error()
		if errMsg != "[ATHENA-TOOL-001] repomix is not installed or not in PATH" {
			t.Errorf("unexpected error: %s", errMsg)
		}
	})
}

func TestContextMCPCommand(t *testing.T) {
	t.Run("start mcp", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "mcp started",
		})

		result, err := svc.MCP(MCPOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Started {
			t.Error("MCP should be started")
		}
	})

	t.Run("stdio mode", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "stdio mcp",
		})

		result, err := svc.MCP(MCPOptions{Stdio: true})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Started {
			t.Error("MCP should be started")
		}
	})

	t.Run("missing tool", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{available: false})

		_, err := svc.MCP(MCPOptions{})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestContextBudgetCommand(t *testing.T) {
	t.Run("within budget", func(t *testing.T) {
		// 80 chars of output / 4 = 20 tokens
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "01234567890123456789012345678901234567890123456789012345678901234567890123456789",
		})

		result, err := svc.Budget(BudgetOptions{MaxTokens: 100})
		if err != nil {
			t.Fatal(err)
		}
		if !result.WithinBudget {
			t.Error("should be within budget")
		}
		if result.EstimatedTokens != 20 {
			t.Errorf("tokens: got %d, want 20", result.EstimatedTokens)
		}
	})

	t.Run("exceeds budget lenient", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "01234567890123456789012345678901234567890123456789012345678901234567890123456789",
		})

		result, err := svc.Budget(BudgetOptions{
			MaxTokens:   5,
			PolicyLevel: config.PolicyLenient,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.WithinBudget {
			t.Error("should be over budget")
		}
	})

	t.Run("exceeds budget strict fails", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "01234567890123456789012345678901234567890123456789012345678901234567890123456789",
		})

		_, err := svc.Budget(BudgetOptions{
			MaxTokens:   5,
			PolicyLevel: config.PolicyStrict,
		})
		if err == nil {
			t.Fatal("expected error under strict policy when budget exceeded")
		}
		if err.Error() != "[ATHENA-POL-002] context budget exceeded: 20 tokens > 5 max" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no max tokens", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "some output",
		})

		result, err := svc.Budget(BudgetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !result.WithinBudget {
			t.Error("no max tokens should always be within budget")
		}
	})

	t.Run("default profile resolution", func(t *testing.T) {
		svc := NewService(defaultCfg(), &mockRunner{
			available: true,
			runOutput: "output",
		})

		result, err := svc.Budget(BudgetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if result.Profile != "review" {
			t.Errorf("profile: got %s, want review", result.Profile)
		}
	})
}

func TestBuildPackArgs(t *testing.T) {
	svc := NewService(defaultCfg(), &mockRunner{available: true})

	args := svc.buildPackArgs("review", PackOptions{
		Stdout:  false,
		Changed: true,
	})

	expected := map[string]bool{
		"--style":    false,
		"--compress": false,
		"--include":  false,
		"--ignore":   false,
		"--output":   false,
		"--changed":  false,
	}

	for _, a := range args {
		if _, ok := expected[a]; ok {
			expected[a] = true
		}
	}

	for flag, found := range expected {
		if !found {
			t.Errorf("missing expected flag: %s in args %v", flag, args)
		}
	}
}

func TestParseTokenCount(t *testing.T) {
	if got := parseTokenCount(""); got != 0 {
		t.Errorf("empty: got %d, want 0", got)
	}

	// 40 chars / 4 = 10 tokens
	if got := parseTokenCount("1234567890123456789012345678901234567890"); got != 10 {
		t.Errorf("40 chars: got %d, want 10", got)
	}
}
