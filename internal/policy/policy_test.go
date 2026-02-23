package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amr-athena/athena/internal/config"
)

func passCheck() CheckFunc {
	return func() *Failure { return nil }
}

func failCheck(policyID, summary, hint string) CheckFunc {
	return func() *Failure {
		return &Failure{
			PolicyID: policyID,
			Severity: "error",
			Summary:  summary,
			FixHint:  hint,
		}
	}
}

func warnCheck(policyID, summary, hint string) CheckFunc {
	return func() *Failure {
		return &Failure{
			PolicyID: policyID,
			Severity: "warning",
			Summary:  summary,
			FixHint:  hint,
		}
	}
}

func TestGateAllPass(t *testing.T) {
	checks := map[string]CheckFunc{
		"check":         passCheck(),
		"security_scan": passCheck(),
		"commit_lint":   passCheck(),
	}

	gate := NewGate(config.PolicyGatesConfig{
		RequiredChecks: []string{"check", "security_scan", "commit_lint"},
	}, checks)

	result, err := gate.Evaluate(GateOptions{TargetRef: "refs/pull/42/head"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Error("expected OK when all checks pass")
	}
	if len(result.Passed) != 3 {
		t.Errorf("passed: got %d, want 3", len(result.Passed))
	}
	if result.TargetRef != "refs/pull/42/head" {
		t.Errorf("target_ref: got %s", result.TargetRef)
	}
}

func TestGateWithFailure(t *testing.T) {
	checks := map[string]CheckFunc{
		"check": passCheck(),
		"security_scan": failCheck(
			"ATHENA-POL-002",
			"gitleaks found secrets",
			"Remove secrets and rerun.",
		),
	}

	gate := NewGate(config.PolicyGatesConfig{
		RequiredChecks: []string{"check", "security_scan"},
	}, checks)

	result, err := gate.Evaluate(GateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Error("expected not OK when a check fails")
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures: got %d, want 1", len(result.Failures))
	}
	if result.Failures[0].PolicyID != "ATHENA-POL-002" {
		t.Errorf("policy_id: got %s", result.Failures[0].PolicyID)
	}
	if result.Failures[0].Severity != "error" {
		t.Errorf("severity: got %s, want error", result.Failures[0].Severity)
	}
}

func TestGateWarningDoesNotFail(t *testing.T) {
	checks := map[string]CheckFunc{
		"check": warnCheck(
			"ATHENA-POL-001",
			"minor issue",
			"Consider fixing.",
		),
	}

	gate := NewGate(config.PolicyGatesConfig{
		RequiredChecks: []string{"check"},
	}, checks)

	result, err := gate.Evaluate(GateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Error("warnings should not cause gate failure")
	}
	if len(result.Failures) != 1 {
		t.Errorf("failures: got %d, want 1 (warning)", len(result.Failures))
	}
}

func TestGateUnknownCheck(t *testing.T) {
	gate := NewGate(config.PolicyGatesConfig{
		RequiredChecks: []string{"nonexistent"},
	}, map[string]CheckFunc{})

	result, err := gate.Evaluate(GateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Error("expected not OK for unknown check")
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failures: got %d, want 1", len(result.Failures))
	}
	if result.Failures[0].PolicyID != "ATHENA-POL-001" {
		t.Errorf("policy_id: got %s", result.Failures[0].PolicyID)
	}
}

func TestGateDefaultRef(t *testing.T) {
	gate := NewGate(config.PolicyGatesConfig{}, map[string]CheckFunc{})

	result, err := gate.Evaluate(GateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.TargetRef != "HEAD" {
		t.Errorf("target_ref: got %s, want HEAD", result.TargetRef)
	}
}

func TestGateOverrideChecks(t *testing.T) {
	checks := map[string]CheckFunc{
		"check":       passCheck(),
		"commit_lint": passCheck(),
	}

	gate := NewGate(config.PolicyGatesConfig{
		RequiredChecks: []string{"check", "commit_lint"},
	}, checks)

	// Override to only run check
	result, err := gate.Evaluate(GateOptions{
		RequiredChecks: []string{"check"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Passed) != 1 {
		t.Errorf("passed: got %d, want 1", len(result.Passed))
	}
}

func TestGateWriteReport(t *testing.T) {
	checks := map[string]CheckFunc{
		"check": passCheck(),
	}

	reportPath := filepath.Join(t.TempDir(), "reports", "gate.json")

	gate := NewGate(config.PolicyGatesConfig{
		RequiredChecks: []string{"check"},
	}, checks)

	_, err := gate.Evaluate(GateOptions{
		ReportPath: reportPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("report should not be empty")
	}
}

func TestGateMultipleFailures(t *testing.T) {
	checks := map[string]CheckFunc{
		"check": failCheck("ATHENA-POL-003", "schema mismatch", "Run check --fix"),
		"security_scan": failCheck("ATHENA-TOOL-002", "secrets found", "Remove secrets"),
		"commit_lint": passCheck(),
	}

	gate := NewGate(config.PolicyGatesConfig{
		RequiredChecks: []string{"check", "security_scan", "commit_lint"},
	}, checks)

	result, err := gate.Evaluate(GateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Error("expected not OK")
	}
	if len(result.Failures) != 2 {
		t.Errorf("failures: got %d, want 2", len(result.Failures))
	}
	if len(result.Passed) != 1 {
		t.Errorf("passed: got %d, want 1", len(result.Passed))
	}
}

func TestCanonicalChecks(t *testing.T) {
	checks := CanonicalChecks()
	expected := []string{"check", "security_scan", "commit_lint", "changelog", "doctor"}
	for _, id := range expected {
		if _, ok := checks[id]; !ok {
			t.Errorf("missing canonical check: %s", id)
		}
	}
}
