package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.Version != SchemaVersion {
		t.Errorf("expected version %d, got %d", SchemaVersion, cfg.Version)
	}
	if cfg.GC.Days != 45 {
		t.Errorf("expected gc.days=45, got %d", cfg.GC.Days)
	}
	if cfg.Policy.Default != PolicyStandard {
		t.Errorf("expected policy=standard, got %s", cfg.Policy.Default)
	}
	if cfg.Execution.DefaultMode != ExecutionDirect {
		t.Errorf("expected execution mode=direct, got %s", cfg.Execution.DefaultMode)
	}
	if !cfg.Execution.EnforceIdempotency {
		t.Error("expected enforce_idempotency=true")
	}
	if !cfg.Features.AIMemory {
		t.Error("expected ai-memory=true in standard preset")
	}
	if cfg.Features.AgentMetrics {
		t.Error("expected agent-metrics=false in standard preset")
	}
}

func TestPresets(t *testing.T) {
	tests := []struct {
		preset       string
		agentsMD     bool
		aiMemory     bool
		claudeShim   bool
		agentMetrics bool
		contributing bool
	}{
		{"minimal", true, true, true, false, false},
		{"standard", true, true, true, false, true},
		{"full", true, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			cfg := ConfigForPreset(tt.preset)
			if cfg.Features.AgentsMD != tt.agentsMD {
				t.Errorf("agents-md: got %v, want %v", cfg.Features.AgentsMD, tt.agentsMD)
			}
			if cfg.Features.AIMemory != tt.aiMemory {
				t.Errorf("ai-memory: got %v, want %v", cfg.Features.AIMemory, tt.aiMemory)
			}
			if cfg.Features.ClaudeShim != tt.claudeShim {
				t.Errorf("claude-shim: got %v, want %v", cfg.Features.ClaudeShim, tt.claudeShim)
			}
			if cfg.Features.AgentMetrics != tt.agentMetrics {
				t.Errorf("agent-metrics: got %v, want %v", cfg.Features.AgentMetrics, tt.agentMetrics)
			}
			if cfg.Features.Contributing != tt.contributing {
				t.Errorf("contributing: got %v, want %v", cfg.Features.Contributing, tt.contributing)
			}
		})
	}
}

func TestParseFullManifest(t *testing.T) {
	tomlData := `
version = 2

[features]
ai-memory = true
contributing = true
editorconfig = true
agents-md = true
agent-tooling = true
agent-metrics = false
claude-shim = true
cursor-shim = true
copilot-shim = true
repomix-config = true
ci-workflow = "github"
security-baseline = true
pre-commit-hooks = true
changelog = true

[scopes]
app = "Application code"
api = "API layer"

[gc]
days = 30

[tools]
required = ["git", "rg"]
recommended = ["repomix"]

[telemetry]
enabled = true
path = ".athena/telemetry.jsonl"
require_run_id_for_agents = true
capture_token_usage = true

[policy]
default = "strict"

[policy_gates]
enabled = true
report_path = ".athena/reports/policy-gate.json"
required_checks = ["check", "security_scan"]

[lock]
ttl = "10m"
allow_force_reap = true

[execution]
plan_dir = ".athena/plans"
journal_path = ".athena/ops-journal.jsonl"
default_mode = "plan-first"
enforce_idempotency = true

[context]
provider = "repomix"
default_profile = "handoff"
output_path = ".athena/context/pack.xml"
compress = false
security_check = true
include = ["**/*.go"]
ignore = [".git/**"]

[context.profiles.review]
style = "xml"
compress = true
strip_diff = true

[security]
enable_secrets_scan = true
enable_workflow_lint = false
report_dir = ".athena/reports"

[changelog]
enabled = true
path = "CHANGELOG.md"
unreleased_heading = "## Unreleased"

[conventional_commits]
enforce = true
require_scope = true
types = ["feat", "fix"]

[hooks]
pre_commit = true

[optimize]
enabled = false
window_days = 14
min_samples = 100
proposal_path = ".athena/optimization/proposals"
auto_apply = false
`
	cfg, err := Parse([]byte(tomlData))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if cfg.Version != SchemaVersion {
		t.Errorf("version: got %d, want %d", cfg.Version, SchemaVersion)
	}
	if cfg.GC.Days != 30 {
		t.Errorf("gc.days: got %d, want 30", cfg.GC.Days)
	}
	if cfg.Policy.Default != PolicyStrict {
		t.Errorf("policy: got %s, want strict", cfg.Policy.Default)
	}
	if cfg.Execution.DefaultMode != ExecutionPlanFirst {
		t.Errorf("execution mode: got %s, want plan-first", cfg.Execution.DefaultMode)
	}
	if !cfg.Lock.AllowForceReap {
		t.Error("expected allow_force_reap=true")
	}
	if cfg.Context.DefaultProfile != "handoff" {
		t.Errorf("context profile: got %s, want handoff", cfg.Context.DefaultProfile)
	}
	if !cfg.ConventionalCommits.RequireScope {
		t.Error("expected require_scope=true")
	}
	if len(cfg.ConventionalCommits.Types) != 2 {
		t.Errorf("conventional types: got %d, want 2", len(cfg.ConventionalCommits.Types))
	}
	if cfg.Optimize.Enabled {
		t.Error("expected optimize.enabled=false")
	}
	if cfg.Optimize.WindowDays != 14 {
		t.Errorf("optimize.window_days: got %d, want 14", cfg.Optimize.WindowDays)
	}
}

func TestParseVersion1Migration(t *testing.T) {
	tomlData := `version = 1`
	cfg, err := Parse([]byte(tomlData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != SchemaVersion {
		t.Errorf("expected migrated version %d, got %d", SchemaVersion, cfg.Version)
	}
}

func TestParseUnsupportedVersion(t *testing.T) {
	tomlData := `version = 99`
	_, err := Parse([]byte(tomlData))
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestParseInvalidTOML(t *testing.T) {
	_, err := Parse([]byte(`[invalid`))
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestParseDefaultsApplied(t *testing.T) {
	// Minimal TOML should get all defaults
	tomlData := `version = 2`
	cfg, err := Parse([]byte(tomlData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Defaults from standard preset should be applied
	if cfg.GC.Days != 45 {
		t.Errorf("expected default gc.days=45, got %d", cfg.GC.Days)
	}
	if cfg.Policy.Default != PolicyStandard {
		t.Errorf("expected default policy=standard, got %s", cfg.Policy.Default)
	}
	if len(cfg.Tools.Required) == 0 {
		t.Error("expected default required tools to be populated")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "athena.toml")

	content := `version = 2
[gc]
days = 60
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GC.Days != 60 {
		t.Errorf("gc.days: got %d, want 60", cfg.GC.Days)
	}
}

func TestLoadFileMissing(t *testing.T) {
	_, err := Load("/nonexistent/athena.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLockTTLDuration(t *testing.T) {
	tests := []struct {
		ttl      string
		expected time.Duration
	}{
		{"15m", 15 * time.Minute},
		{"30s", 30 * time.Second},
		{"1h", time.Hour},
		{"", 15 * time.Minute}, // default
	}

	for _, tt := range tests {
		t.Run(tt.ttl, func(t *testing.T) {
			lc := LockConfig{TTL: tt.ttl}
			d, err := lc.TTLDuration()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d != tt.expected {
				t.Errorf("got %v, want %v", d, tt.expected)
			}
		})
	}
}

func TestLockTTLDurationInvalid(t *testing.T) {
	lc := LockConfig{TTL: "invalid"}
	_, err := lc.TTLDuration()
	if err == nil {
		t.Fatal("expected error for invalid TTL")
	}
}

func TestResolvePolicyPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		cliFlag  string
		cfgDef   PolicyLevel
		expected PolicyLevel
	}{
		{"cli override", "strict", PolicyStandard, PolicyStrict},
		{"config default", "", PolicyLenient, PolicyLenient},
		{"cli takes precedence", "lenient", PolicyStrict, PolicyLenient},
		{"empty cli uses config", "", PolicyStandard, PolicyStandard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvePolicy(tt.cliFlag, tt.cfgDef)
			if got != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestResolveExecutionModePrecedence(t *testing.T) {
	tests := []struct {
		name     string
		cliFlag  string
		cfgDef   ExecutionMode
		expected ExecutionMode
	}{
		{"cli override", "plan-first", ExecutionDirect, ExecutionPlanFirst},
		{"config default", "", ExecutionPlanFirst, ExecutionPlanFirst},
		{"cli direct", "direct", ExecutionPlanFirst, ExecutionDirect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveExecutionMode(tt.cliFlag, tt.cfgDef)
			if got != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestResolveFormatPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		cliFlag    string
		defaultFmt string
		expected   string
	}{
		{"cli json", "json", "text", "json"},
		{"default text", "", "text", "text"},
		{"fallback", "", "", "text"},
		{"cli text", "text", "json", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveFormat(tt.cliFlag, tt.defaultFmt)
			if got != tt.expected {
				t.Errorf("got %s, want %s", got, tt.expected)
			}
		})
	}
}
