package security

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aredgwell/athena/internal/config"
)

type mockRunner struct {
	available map[string]bool
	outputs   map[string]string
	errors    map[string]error
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		available: make(map[string]bool),
		outputs:   make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockRunner) Run(name string, args ...string) ([]byte, error) {
	if err, ok := m.errors[name]; ok && err != nil {
		out := ""
		if o, ok := m.outputs[name]; ok {
			out = o
		}
		return []byte(out), err
	}
	out := m.outputs[name]
	return []byte(out), nil
}

func (m *mockRunner) LookPath(name string) (string, error) {
	if avail, ok := m.available[name]; ok && avail {
		return "/usr/local/bin/" + name, nil
	}
	return "", fmt.Errorf("not found: %s", name)
}

func defaultCfg() config.SecurityConfig {
	return config.SecurityConfig{
		EnableSecretsScan:  true,
		EnableWorkflowLint: true,
		ReportDir:          "",
	}
}

func TestScanBothPass(t *testing.T) {
	runner := newMockRunner()
	runner.available["gitleaks"] = true
	runner.available["actionlint"] = true

	svc := NewService(defaultCfg(), runner)
	result, err := svc.Scan(ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Error("expected OK when both pass")
	}
	if result.Tools["gitleaks"].Status != "pass" {
		t.Errorf("gitleaks: got %s, want pass", result.Tools["gitleaks"].Status)
	}
	if result.Tools["actionlint"].Status != "pass" {
		t.Errorf("actionlint: got %s, want pass", result.Tools["actionlint"].Status)
	}
}

func TestScanGitleaksFindings(t *testing.T) {
	runner := newMockRunner()
	runner.available["gitleaks"] = true
	runner.available["actionlint"] = true
	runner.outputs["gitleaks"] = `[{"match":"secret1"},{"match":"secret2"}]`
	runner.errors["gitleaks"] = fmt.Errorf("exit status 1")

	svc := NewService(defaultCfg(), runner)
	result, err := svc.Scan(ScanOptions{Secrets: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Error("expected not OK when gitleaks finds secrets")
	}
	if result.Tools["gitleaks"].Findings != 2 {
		t.Errorf("findings: got %d, want 2", result.Tools["gitleaks"].Findings)
	}
}

func TestScanSecretsOnly(t *testing.T) {
	runner := newMockRunner()
	runner.available["gitleaks"] = true

	svc := NewService(defaultCfg(), runner)
	result, err := svc.Scan(ScanOptions{Secrets: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Tools["actionlint"]; ok {
		t.Error("actionlint should not run when only --secrets")
	}
}

func TestScanWorkflowsOnly(t *testing.T) {
	runner := newMockRunner()
	runner.available["actionlint"] = true

	svc := NewService(defaultCfg(), runner)
	result, err := svc.Scan(ScanOptions{Workflows: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Tools["gitleaks"]; ok {
		t.Error("gitleaks should not run when only --workflows")
	}
}

func TestScanMissingToolStrict(t *testing.T) {
	runner := newMockRunner()
	// Neither tool available

	svc := NewService(defaultCfg(), runner)
	result, err := svc.Scan(ScanOptions{PolicyLevel: config.PolicyStrict})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Error("expected not OK under strict when tools missing")
	}
	if result.Tools["gitleaks"].Status != "fail" {
		t.Errorf("gitleaks: got %s, want fail", result.Tools["gitleaks"].Status)
	}
}

func TestScanMissingToolStandard(t *testing.T) {
	runner := newMockRunner()
	// Neither tool available

	svc := NewService(defaultCfg(), runner)
	result, err := svc.Scan(ScanOptions{PolicyLevel: config.PolicyStandard})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Error("expected OK under standard when tools missing (warn only)")
	}
	if result.Tools["gitleaks"].Status != "skip" {
		t.Errorf("gitleaks: got %s, want skip", result.Tools["gitleaks"].Status)
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warnings for missing tools")
	}
}

func TestScanLenientWithFindings(t *testing.T) {
	runner := newMockRunner()
	runner.available["actionlint"] = true
	runner.outputs["actionlint"] = `[{"error":"some issue"}]`
	runner.errors["actionlint"] = fmt.Errorf("exit status 1")

	svc := NewService(defaultCfg(), runner)
	result, err := svc.Scan(ScanOptions{
		Workflows:   true,
		PolicyLevel: config.PolicyLenient,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Error("expected OK under lenient even with findings")
	}
}

func TestScanWriteReportJSON(t *testing.T) {
	runner := newMockRunner()
	runner.available["gitleaks"] = true
	reportDir := filepath.Join(t.TempDir(), "reports")

	svc := NewService(defaultCfg(), runner)
	_, err := svc.Scan(ScanOptions{
		Secrets:      true,
		ReportFormat: "json",
		ReportDir:    reportDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(reportDir, "security-scan.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("report file should not be empty")
	}
}

func TestScanWriteReportSARIF(t *testing.T) {
	runner := newMockRunner()
	runner.available["gitleaks"] = true
	reportDir := filepath.Join(t.TempDir(), "reports")

	svc := NewService(defaultCfg(), runner)
	_, err := svc.Scan(ScanOptions{
		Secrets:      true,
		ReportFormat: "sarif",
		ReportDir:    reportDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(filepath.Join(reportDir, "security-scan.sarif.json"))
	if err != nil {
		t.Fatalf("SARIF report not found: %v", err)
	}
}

func TestCheckTool(t *testing.T) {
	runner := newMockRunner()
	runner.available["gitleaks"] = true

	if err := CheckTool(runner, "gitleaks"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err := CheckTool(runner, "missing-tool")
	if err == nil {
		t.Error("expected error for missing tool")
	}
}

func TestCountJSONFindings(t *testing.T) {
	if got := countJSONFindings([]byte(`[{"a":1},{"b":2}]`)); got != 2 {
		t.Errorf("got %d, want 2", got)
	}
	if got := countJSONFindings([]byte(`not json`)); got != 1 {
		t.Errorf("got %d, want 1 (non-empty non-JSON)", got)
	}
	if got := countJSONFindings([]byte("")); got != 0 {
		t.Errorf("got %d, want 0 (empty)", got)
	}
}
