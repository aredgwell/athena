// Package config implements athena.toml parsing, defaults, and precedence resolution.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// SchemaVersion is the current manifest schema version.
const SchemaVersion = 2

// PolicyLevel represents an Athena policy enforcement level.
type PolicyLevel string

const (
	PolicyStrict   PolicyLevel = "strict"
	PolicyStandard PolicyLevel = "standard"
	PolicyLenient  PolicyLevel = "lenient"
)

// ExecutionMode represents the default execution mode.
type ExecutionMode string

const (
	ExecutionDirect    ExecutionMode = "direct"
	ExecutionPlanFirst ExecutionMode = "plan-first"
)

// Config is the top-level athena.toml manifest.
type Config struct {
	Version            int                 `toml:"version"`
	Features           Features            `toml:"features"`
	Templates          map[string]string   `toml:"templates"`
	Scopes             map[string]string   `toml:"scopes"`
	GC                 GCConfig            `toml:"gc"`
	Tools              ToolsConfig         `toml:"tools"`
	Telemetry          TelemetryConfig     `toml:"telemetry"`
	Policy             PolicyConfig        `toml:"policy"`
	PolicyGates        PolicyGatesConfig   `toml:"policy_gates"`
	Lock               LockConfig          `toml:"lock"`
	Execution          ExecutionConfig     `toml:"execution"`
	Context            ContextConfig       `toml:"context"`
	Security           SecurityConfig      `toml:"security"`
	Changelog          ChangelogConfig     `toml:"changelog"`
	ConventionalCommits ConventionalConfig `toml:"conventional_commits"`
	Hooks              HooksConfig         `toml:"hooks"`
	Optimize           OptimizeConfig      `toml:"optimize"`
}

// Features controls which files are scaffolded and managed.
type Features struct {
	AIMemory         bool   `toml:"ai-memory"`
	Contributing     bool   `toml:"contributing"`
	Editorconfig     bool   `toml:"editorconfig"`
	AgentsMD         bool   `toml:"agents-md"`
	AgentTooling     bool   `toml:"agent-tooling"`
	AgentMetrics     bool   `toml:"agent-metrics"`
	ClaudeShim       bool   `toml:"claude-shim"`
	CursorShim       bool   `toml:"cursor-shim"`
	CopilotShim      bool   `toml:"copilot-shim"`
	RepomixConfig    bool   `toml:"repomix-config"`
	CIWorkflow       string `toml:"ci-workflow"`
	SecurityBaseline bool   `toml:"security-baseline"`
	PreCommitHooks   bool   `toml:"pre-commit-hooks"`
	ChangelogFeature bool   `toml:"changelog"`
}

// GCConfig controls garbage collection behavior.
type GCConfig struct {
	Days int `toml:"days"`
}

// ToolsConfig lists required and recommended CLI tools.
type ToolsConfig struct {
	Required    []string `toml:"required"`
	Recommended []string `toml:"recommended"`
}

// TelemetryConfig controls local telemetry behavior.
type TelemetryConfig struct {
	Enabled              bool   `toml:"enabled"`
	Path                 string `toml:"path"`
	RequireRunIDForAgents bool  `toml:"require_run_id_for_agents"`
	CaptureTokenUsage    bool   `toml:"capture_token_usage"`
}

// PolicyConfig sets the default policy level.
type PolicyConfig struct {
	Default PolicyLevel `toml:"default"`
}

// PolicyGatesConfig controls PR/revision policy gating.
type PolicyGatesConfig struct {
	Enabled        bool     `toml:"enabled"`
	ReportPath     string   `toml:"report_path"`
	RequiredChecks []string `toml:"required_checks"`
}

// LockConfig controls mutation lock behavior.
type LockConfig struct {
	TTL            string `toml:"ttl"`
	AllowForceReap bool   `toml:"allow_force_reap"`
}

// TTLDuration parses the TTL string into a time.Duration.
func (l LockConfig) TTLDuration() (time.Duration, error) {
	if l.TTL == "" {
		return 15 * time.Minute, nil
	}
	return time.ParseDuration(l.TTL)
}

// ExecutionConfig controls plan/apply behavior.
type ExecutionConfig struct {
	PlanDir             string        `toml:"plan_dir"`
	JournalPath         string        `toml:"journal_path"`
	DefaultMode         ExecutionMode `toml:"default_mode"`
	EnforceIdempotency  bool          `toml:"enforce_idempotency"`
}

// ContextConfig controls repomix integration.
type ContextConfig struct {
	Provider       string                    `toml:"provider"`
	DefaultProfile string                    `toml:"default_profile"`
	OutputPath     string                    `toml:"output_path"`
	Compress       bool                      `toml:"compress"`
	SecurityCheck  bool                      `toml:"security_check"`
	Include        []string                  `toml:"include"`
	Ignore         []string                  `toml:"ignore"`
	Profiles       map[string]ContextProfile `toml:"profiles"`
}

// ContextProfile defines a named context packing profile.
type ContextProfile struct {
	Style     string `toml:"style"`
	Compress  bool   `toml:"compress"`
	StripDiff bool   `toml:"strip_diff"`
}

// SecurityConfig controls security scan behavior.
type SecurityConfig struct {
	EnableSecretsScan  bool   `toml:"enable_secrets_scan"`
	EnableWorkflowLint bool   `toml:"enable_workflow_lint"`
	ReportDir          string `toml:"report_dir"`
}

// ChangelogConfig controls changelog generation.
type ChangelogConfig struct {
	Enabled           bool   `toml:"enabled"`
	Path              string `toml:"path"`
	UnreleasedHeading string `toml:"unreleased_heading"`
}

// ConventionalConfig controls conventional commit enforcement.
type ConventionalConfig struct {
	Enforce      bool     `toml:"enforce"`
	RequireScope bool     `toml:"require_scope"`
	Types        []string `toml:"types"`
}

// HooksConfig controls pre-commit hook behavior.
type HooksConfig struct {
	PreCommit bool `toml:"pre_commit"`
}

// OptimizeConfig controls optimization recommendations.
type OptimizeConfig struct {
	Enabled      bool   `toml:"enabled"`
	WindowDays   int    `toml:"window_days"`
	MinSamples   int    `toml:"min_samples"`
	ProposalPath string `toml:"proposal_path"`
	AutoApply    bool   `toml:"auto_apply"`
}

// Preset names for athena init.
const (
	PresetMinimal  = "minimal"
	PresetStandard = "standard"
	PresetFull     = "full"
)

// Default returns a Config with built-in default values (standard preset).
func Default() Config {
	return ConfigForPreset(PresetStandard)
}

// ConfigForPreset returns a Config with feature flags set for the given preset.
func ConfigForPreset(preset string) Config {
	cfg := Config{
		Version: SchemaVersion,
		Features: standardFeatures(),
		Scopes: map[string]string{
			"app":   "Application code",
			"api":   "API layer",
			"infra": "Infrastructure configuration",
			"docs":  "Documentation",
			"ci":    "CI/CD pipelines and workflows",
			"meta":  "Repository-level config",
		},
		GC: GCConfig{Days: 45},
		Tools: ToolsConfig{
			Required:    []string{"git", "rg", "jq", "yq", "task"},
			Recommended: []string{"repomix", "gitleaks", "actionlint", "pre-commit", "difft", "fzf", "shfmt", "shellcheck"},
		},
		Telemetry: TelemetryConfig{
			Enabled:              true,
			Path:                 ".athena/telemetry.jsonl",
			RequireRunIDForAgents: true,
			CaptureTokenUsage:    true,
		},
		Policy:      PolicyConfig{Default: PolicyStandard},
		PolicyGates: PolicyGatesConfig{
			Enabled:        true,
			ReportPath:     ".athena/reports/policy-gate.json",
			RequiredChecks: []string{"check", "security_scan", "commit_lint"},
		},
		Lock: LockConfig{TTL: "15m", AllowForceReap: false},
		Execution: ExecutionConfig{
			PlanDir:            ".athena/plans",
			JournalPath:        ".athena/ops-journal.jsonl",
			DefaultMode:        ExecutionDirect,
			EnforceIdempotency: true,
		},
		Context: ContextConfig{
			Provider:       "repomix",
			DefaultProfile: "review",
			OutputPath:     ".athena/context/pack.xml",
			Compress:       true,
			SecurityCheck:  true,
			Include:        []string{"**/*.go", "**/*.md", "athena.toml"},
			Ignore:         []string{".git/**", "node_modules/**", "dist/**"},
			Profiles: map[string]ContextProfile{
				"review":  {Style: "xml", Compress: true, StripDiff: true},
				"handoff": {Style: "markdown", Compress: true, StripDiff: false},
				"release": {Style: "plain", Compress: false, StripDiff: false},
			},
		},
		Security: SecurityConfig{
			EnableSecretsScan:  true,
			EnableWorkflowLint: true,
			ReportDir:          ".athena/reports",
		},
		Changelog: ChangelogConfig{
			Enabled:           true,
			Path:              "CHANGELOG.md",
			UnreleasedHeading: "## Unreleased",
		},
		ConventionalCommits: ConventionalConfig{
			Enforce:      true,
			RequireScope: false,
			Types:        []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"},
		},
		Hooks:    HooksConfig{PreCommit: true},
		Optimize: OptimizeConfig{
			Enabled:      true,
			WindowDays:   30,
			MinSamples:   50,
			ProposalPath: ".athena/optimization/proposals",
			AutoApply:    false,
		},
	}

	switch preset {
	case PresetMinimal:
		cfg.Features = minimalFeatures()
	case PresetFull:
		cfg.Features = fullFeatures()
	// standard is the default, already set above
	}

	return cfg
}

func minimalFeatures() Features {
	return Features{
		AgentsMD:   true,
		AIMemory:   true,
		ClaudeShim: true,
		CIWorkflow: "none",
	}
}

func standardFeatures() Features {
	return Features{
		AIMemory:         true,
		Contributing:     true,
		Editorconfig:     true,
		AgentsMD:         true,
		AgentTooling:     true,
		AgentMetrics:     false,
		ClaudeShim:       true,
		CursorShim:       true,
		CopilotShim:      true,
		RepomixConfig:    true,
		CIWorkflow:       "github",
		SecurityBaseline: true,
		PreCommitHooks:   true,
		ChangelogFeature: true,
	}
}

func fullFeatures() Features {
	f := standardFeatures()
	f.AgentMetrics = true
	return f
}

// Load reads and parses an athena.toml file, applying defaults for missing fields.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}
	return Parse(data)
}

// Parse decodes TOML data into a Config, applying defaults for missing fields.
func Parse(data []byte) (Config, error) {
	cfg := Default()

	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	// Validate schema version
	if cfg.Version < 1 || cfg.Version > SchemaVersion {
		return Config{}, fmt.Errorf("unsupported manifest version %d (supported: 1-%d)", cfg.Version, SchemaVersion)
	}

	// Migrate v1 to v2 in memory
	if cfg.Version == 1 {
		cfg.Version = SchemaVersion
	}

	_ = md // metadata available for future undecoded-key warnings
	return cfg, nil
}

// ResolvePolicy returns the effective policy level given a CLI override and config default.
// CLI flag takes precedence over config.
func ResolvePolicy(cliFlag string, cfgDefault PolicyLevel) PolicyLevel {
	if cliFlag != "" {
		return PolicyLevel(cliFlag)
	}
	return cfgDefault
}

// ResolveExecutionMode returns the effective execution mode given a CLI override and config default.
func ResolveExecutionMode(cliFlag string, cfgDefault ExecutionMode) ExecutionMode {
	if cliFlag != "" {
		return ExecutionMode(cliFlag)
	}
	return cfgDefault
}

// ResolveFormat returns the effective output format given a CLI override and a default.
func ResolveFormat(cliFlag string, defaultFmt string) string {
	if cliFlag != "" {
		return cliFlag
	}
	if defaultFmt != "" {
		return defaultFmt
	}
	return "text"
}
