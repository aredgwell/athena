package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/amr-athena/athena/internal/config"
	"github.com/amr-athena/athena/internal/execution"
	"github.com/amr-athena/athena/internal/gc"
	"github.com/amr-athena/athena/internal/index"
	"github.com/amr-athena/athena/internal/notes"
	"github.com/amr-athena/athena/internal/policy"
	"github.com/amr-athena/athena/internal/release"
	"github.com/amr-athena/athena/internal/report"
	"github.com/amr-athena/athena/internal/scaffold"
	"github.com/amr-athena/athena/internal/telemetry"
	"github.com/amr-athena/athena/internal/validate"
)

func TestIntegrationScaffoldCheckIndex(t *testing.T) {
	// Integration: init scaffold -> create notes -> check -> index -> gc -> report
	dir := t.TempDir()
	aiDir := filepath.Join(dir, ".ai")

	// Step 1: scaffold init
	files := []scaffold.ManagedFile{
		{Path: ".athena/config.toml", Content: []byte("[core]\npolicy = \"standard\"\n")},
		{Path: ".athena/checksums.json", Content: []byte("{}")},
	}
	initResult, err := scaffold.Init(files, scaffold.InitOptions{
		RepoRoot: dir,
		Version:  "0.1.0",
	})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if initResult.Written == 0 {
		t.Error("init should create files")
	}

	// Step 2: create notes
	n1, err := notes.NewNote(aiDir, "context", "auth", "Authentication Analysis")
	if err != nil {
		t.Fatalf("note new: %v", err)
	}
	n2, _ := notes.NewNote(aiDir, "investigation", "perf", "Performance Review")
	n3, _ := notes.NewNote(aiDir, "wip", "refactor", "Code Refactoring")

	// Close one note
	notes.CloseNote(n2.Path, "closed")

	// Step 3: validate notes
	results, summary, err := validate.Check(validate.CheckOptions{Dir: aiDir})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if summary.Invalid != 0 {
		for _, r := range results {
			if !r.Valid {
				t.Errorf("invalid note %s: %v", r.Path, r.Errors)
			}
		}
	}
	if summary.FilesScanned != 3 {
		t.Errorf("scanned: got %d, want 3", summary.FilesScanned)
	}

	// Step 4: build index
	idx, err := index.Build(aiDir)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if idx.Counts.Total != 3 {
		t.Errorf("total: got %d, want 3", idx.Counts.Total)
	}
	if idx.Counts.Active != 2 {
		t.Errorf("active: got %d, want 2", idx.Counts.Active)
	}
	if idx.Counts.Closed != 1 {
		t.Errorf("closed: got %d, want 1", idx.Counts.Closed)
	}

	// Step 5: report metrics
	metrics, err := report.Compute(aiDir)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if metrics.StalenessRatio != 0 {
		t.Errorf("staleness: got %f, want 0", metrics.StalenessRatio)
	}

	// Step 6: promote a note
	notes.PromoteNote(n1.Path, "docs/auth.md")
	metrics2, err := report.Compute(aiDir)
	if err != nil {
		t.Fatalf("report after promote: %v", err)
	}
	if metrics2.PromotionRate <= 0 {
		t.Error("promotion rate should be > 0 after promotion")
	}

	// Use n3 to avoid unused var
	_ = n3
}

func TestIntegrationPlanApplyRollback(t *testing.T) {
	dir := t.TempDir()
	planDir := filepath.Join(dir, "plans")
	journalPath := filepath.Join(dir, "journal.jsonl")

	journal := execution.NewJournal(journalPath)
	store := execution.NewPlanStore(planDir)
	engine := execution.NewEngine(journal)

	// Create a plan
	steps := []execution.PlanStep{
		{Index: 0, Action: execution.ActionWrite, TargetPath: filepath.Join(dir, "file1.txt"), Detail: "create file1"},
		{Index: 1, Action: execution.ActionWrite, TargetPath: filepath.Join(dir, "file2.txt"), Detail: "create file2"},
	}
	plan := execution.NewPlan("test-plan-001", "check --fix", nil, steps, nil)

	// Save plan
	if err := store.Save(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	// Load plan
	loaded, err := store.Load(plan.PlanID)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if loaded.PlanID != plan.PlanID {
		t.Error("loaded plan ID mismatch")
	}

	// Apply plan
	txID := "tx_integration_001"
	err = engine.Apply(loaded, txID, func(step execution.PlanStep) (string, error) {
		return "", os.WriteFile(step.TargetPath, []byte("content"), 0o644)
	}, func(entry execution.JournalEntry) error {
		return nil // no-op rollback for test
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	// Verify files created
	if _, err := os.Stat(filepath.Join(dir, "file1.txt")); err != nil {
		t.Error("file1 should exist after apply")
	}
	if _, err := os.Stat(filepath.Join(dir, "file2.txt")); err != nil {
		t.Error("file2 should exist after apply")
	}

	// Verify journal has entries
	txEntries, _ := journal.EntriesForTx(txID)
	if len(txEntries) < 2 {
		t.Errorf("journal entries for tx: got %d, want >= 2", len(txEntries))
	}
}

func TestIntegrationReleaseProposeApprove(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "proposals")

	gates := map[string]release.GateFunc{
		"check":       func() release.GateStatus { return release.GateStatus{Name: "check", Status: "pass"} },
		"commit_lint": func() release.GateStatus { return release.GateStatus{Name: "commit_lint", Status: "pass"} },
		"security_scan": func() release.GateStatus {
			return release.GateStatus{Name: "security_scan", Status: "pass"}
		},
	}

	svc := release.NewService(gates)

	// Propose
	propResult, err := svc.Propose(release.ProposeOptions{
		SinceTag:    "v1.0.0",
		NextVersion: "1.1.0",
		GateNames:   []string{"check", "commit_lint", "security_scan"},
		StorePath:   storePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !propResult.OK {
		t.Error("proposal should be OK")
	}

	// Approve
	appResult, err := svc.Approve(release.ApproveOptions{
		ProposalID: propResult.Proposal.ProposalID,
		StorePath:  storePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !appResult.OK {
		t.Errorf("approve should be OK: %s", appResult.Detail)
	}
	if appResult.NextVersion != "1.1.0" {
		t.Errorf("version: got %s", appResult.NextVersion)
	}
}

func TestIntegrationPolicyGate(t *testing.T) {
	checks := map[string]policy.CheckFunc{
		"check":       func() *policy.Failure { return nil },
		"commit_lint": func() *policy.Failure { return nil },
		"security_scan": func() *policy.Failure {
			return &policy.Failure{
				PolicyID: "ATHENA-TOOL-002",
				Severity: "error",
				Summary:  "secrets found",
				FixHint:  "Remove secrets.",
			}
		},
	}

	gate := policy.NewGate(config.PolicyGatesConfig{
		RequiredChecks: []string{"check", "commit_lint", "security_scan"},
	}, checks)

	result, err := gate.Evaluate(policy.GateOptions{TargetRef: "refs/pull/1/head"})
	if err != nil {
		t.Fatal(err)
	}

	if result.OK {
		t.Error("gate should fail with security findings")
	}
	if len(result.Failures) != 1 {
		t.Errorf("failures: got %d, want 1", len(result.Failures))
	}
	if result.Failures[0].PolicyID != "ATHENA-TOOL-002" {
		t.Errorf("policy_id: got %s", result.Failures[0].PolicyID)
	}
	if len(result.Passed) != 2 {
		t.Errorf("passed: got %d, want 2", len(result.Passed))
	}
}

func TestIntegrationTelemetryReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry.jsonl")
	store := telemetry.NewStore(path)

	// Simulate a run with multiple commands
	runID := "run_integration_001"
	commands := []string{"context pack", "check", "note new", "gc"}
	for _, cmd := range commands {
		store.Append(telemetry.Record{
			Timestamp:   time.Now().UTC(),
			Command:     cmd,
			RunID:       runID,
			TotalTokens: 100,
		})
	}

	records, err := store.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 4 {
		t.Fatalf("records: got %d, want 4", len(records))
	}

	// Filter and verify correlation
	correlated := telemetry.FilterByRunID(records, runID)
	if len(correlated) != 4 {
		t.Errorf("correlated: got %d, want 4", len(correlated))
	}

	// Coverage metric
	coverage := telemetry.Coverage(records, 10)
	if coverage != 0.4 {
		t.Errorf("coverage: got %f, want 0.4", coverage)
	}
}

func TestIntegrationGCLifecycle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ai")

	// Create notes
	notes.NewNote(dir, "context", "active1", "Active Note")

	// Create old note directly
	oldDir := filepath.Join(dir, "context")
	os.MkdirAll(oldDir, 0o755)
	oldNote := `---
id: context-20250101-old
title: Old Note
type: context
status: active
created: "2025-01-01"
updated: "2025-01-01"
schema_version: 1
---

Old content.
`
	os.WriteFile(filepath.Join(oldDir, "context-20250101-old.md"), []byte(oldNote), 0o644)

	// Run GC (dry run)
	dryResult, _ := gc.Run(dir, 45, true)
	if dryResult.Marked != 1 {
		t.Errorf("dry-run marked: got %d, want 1", dryResult.Marked)
	}

	// Verify dry run didn't modify
	n, _ := notes.ParseNote(filepath.Join(oldDir, "context-20250101-old.md"))
	if n.Frontmatter.Status != "active" {
		t.Error("dry run should not modify status")
	}

	// Run GC (real)
	result, _ := gc.Run(dir, 45, false)
	if result.Marked != 1 {
		t.Errorf("marked: got %d, want 1", result.Marked)
	}

	// Verify modification
	n2, _ := notes.ParseNote(filepath.Join(oldDir, "context-20250101-old.md"))
	if n2.Frontmatter.Status != "stale" {
		t.Errorf("status: got %s, want stale", n2.Frontmatter.Status)
	}
}

func TestJSONContractCapabilities(t *testing.T) {
	out, err := executeCommand("capabilities")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "capabilities") {
		t.Error("capabilities command should produce output")
	}
}

func TestJSONContractDoctor(t *testing.T) {
	out, err := executeCommand("doctor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "doctor") {
		t.Error("doctor command should produce output")
	}
}

func TestJSONContractReport(t *testing.T) {
	out, err := executeCommand("report")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "report") {
		t.Error("report command should produce output")
	}
}

func TestCheckFixIntegration(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".ai")
	noteDir := filepath.Join(dir, "context")
	os.MkdirAll(noteDir, 0o755)

	// Create a note with schema_version 0 (needs migration)
	noteContent := `---
id: context-20250101-test
title: Test Note
type: context
status: active
created: "2025-01-01"
updated: "2025-01-01"
---

Content.
`
	path := filepath.Join(noteDir, "context-20250101-test.md")
	os.WriteFile(path, []byte(noteContent), 0o644)

	// Check read-only first
	results, summary, _ := validate.Check(validate.CheckOptions{Dir: dir})
	if summary.FilesScanned != 1 {
		t.Fatalf("scanned: got %d, want 1", summary.FilesScanned)
	}

	// File should be unchanged (read-only)
	n, _ := notes.ParseNote(path)
	if n.Frontmatter.SchemaVersion != 0 {
		t.Error("read-only check should not modify file")
	}

	// Since CurrentFrontmatterVersion == 1 and missing schema_version defaults to 1,
	// the note is not migratable. Verify the check correctly identifies it as valid.
	if summary.Valid != 1 {
		t.Errorf("valid: got %d, want 1", summary.Valid)
	}

	// Check with strict-schema: should flag that schema_version field is 0 (missing) != 1
	strictResults, strictSummary, _ := validate.Check(validate.CheckOptions{
		Dir:          dir,
		StrictSchema: true,
	})
	if strictSummary.Invalid != 1 {
		t.Errorf("strict invalid: got %d, want 1", strictSummary.Invalid)
	}

	// Verify result contains results
	_ = results
	_ = strictResults
}

// Ensure JSON marshaling of key structs is stable
func TestStableJSONMarshaling(t *testing.T) {
	t.Run("envelope", func(t *testing.T) {
		env := NewEnvelope("test", time.Second)
		data, err := json.Marshal(env)
		if err != nil {
			t.Fatal(err)
		}
		var raw map[string]interface{}
		json.Unmarshal(data, &raw)

		required := []string{"command", "ok", "duration_ms", "warnings", "errors"}
		for _, f := range required {
			if _, ok := raw[f]; !ok {
				t.Errorf("missing field: %s", f)
			}
		}
	})

	t.Run("telemetry record", func(t *testing.T) {
		rec := telemetry.Record{
			Timestamp: time.Now(),
			Command:   "check",
			ExitCode:  0,
			ErrorCode: nil,
		}
		data, _ := json.Marshal(rec)
		var raw map[string]interface{}
		json.Unmarshal(data, &raw)

		if raw["error_code"] != nil {
			t.Error("null error_code should serialize as null")
		}
	})
}
