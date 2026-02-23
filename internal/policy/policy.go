// Package policy implements PR/revision policy gate evaluation.
package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/amr-athena/athena/internal/config"
)

// CheckFunc is a function that runs a single gate check.
// Returns nil if the check passes, or a Failure describing what went wrong.
type CheckFunc func() *Failure

// Failure describes a single policy gate failure.
type Failure struct {
	PolicyID string `json:"policy_id"`
	Severity string `json:"severity"` // "error" or "warning"
	Summary  string `json:"summary"`
	FixHint  string `json:"fix_hint"`
}

// GateOptions controls policy gate behavior.
type GateOptions struct {
	TargetRef      string
	RequiredChecks []string
	ReportPath     string
	PolicyLevel    config.PolicyLevel
}

// GateResult holds the aggregate gate evaluation outcome.
type GateResult struct {
	OK        bool      `json:"ok"`
	TargetRef string    `json:"target_ref"`
	Passed    []string  `json:"passed,omitempty"`
	Failures  []Failure `json:"failures,omitempty"`
}

// Gate evaluates a PR/revision against configured policy gates.
type Gate struct {
	cfg    config.PolicyGatesConfig
	checks map[string]CheckFunc
}

// NewGate creates a policy gate with the given config and check registry.
func NewGate(cfg config.PolicyGatesConfig, checks map[string]CheckFunc) *Gate {
	return &Gate{cfg: cfg, checks: checks}
}

// Evaluate runs all required checks and returns the gate result.
func (g *Gate) Evaluate(opts GateOptions) (*GateResult, error) {
	ref := opts.TargetRef
	if ref == "" {
		ref = "HEAD"
	}

	requiredChecks := opts.RequiredChecks
	if len(requiredChecks) == 0 {
		requiredChecks = g.cfg.RequiredChecks
	}

	result := &GateResult{
		OK:        true,
		TargetRef: ref,
	}

	for _, checkID := range requiredChecks {
		fn, ok := g.checks[checkID]
		if !ok {
			result.Failures = append(result.Failures, Failure{
				PolicyID: "ATHENA-POL-001",
				Severity: "error",
				Summary:  fmt.Sprintf("unknown gate check: %s", checkID),
				FixHint:  "Verify [policy_gates].required_checks in athena.toml.",
			})
			result.OK = false
			continue
		}

		failure := fn()
		if failure != nil {
			result.Failures = append(result.Failures, *failure)
			if failure.Severity == "error" {
				result.OK = false
			}
		} else {
			result.Passed = append(result.Passed, checkID)
		}
	}

	// Write report if configured
	reportPath := opts.ReportPath
	if reportPath == "" {
		reportPath = g.cfg.ReportPath
	}
	if reportPath != "" {
		if err := writeGateReport(result, reportPath); err != nil {
			return result, err
		}
	}

	return result, nil
}

func writeGateReport(result *GateResult, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating report dir: %w", err)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// CanonicalChecks returns the mapping of canonical check IDs to commands.
func CanonicalChecks() map[string]string {
	return map[string]string{
		"check":         "athena check",
		"security_scan": "athena security scan",
		"commit_lint":   "athena commit lint",
		"changelog":     "athena changelog --dry-run",
		"doctor":        "athena doctor",
	}
}
