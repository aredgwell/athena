package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amr-athena/athena/internal/notes"
)

// setupTestRepo creates a temp directory with athena.toml, .athena/, and .ai/ structure,
// and sets testRepoRoot so all commands resolve to this directory.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Write athena.toml
	toml := `version = 2
[features]
ai-memory = true
agents-md = true
[gc]
days = 45
[policy]
default = "standard"
[telemetry]
enabled = true
path = ".athena/telemetry.jsonl"
[conventional_commits]
types = ["feat", "fix", "docs", "chore", "test", "refactor"]
`
	os.MkdirAll(filepath.Join(dir, ".athena"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".ai"), 0o755)
	os.WriteFile(filepath.Join(dir, "athena.toml"), []byte(toml), 0o644)

	testRepoRoot = dir
	t.Cleanup(func() { testRepoRoot = "" })
	return dir
}

// setupTestRepoWithNotes adds sample notes to a test repo.
func setupTestRepoWithNotes(t *testing.T) string {
	t.Helper()
	dir := setupTestRepo(t)
	aiDir := filepath.Join(dir, ".ai")

	notes.NewNote(aiDir, "context", "auth", "Auth Analysis")
	notes.NewNote(aiDir, "investigation", "perf", "Performance Review")
	notes.NewNote(aiDir, "wip", "refactor", "Refactoring Plan")

	return dir
}

// parseEnvelope parses JSON output into a generic map for field validation.
func parseEnvelope(t *testing.T, output string) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal([]byte(output), &env); err != nil {
		t.Fatalf("failed to parse JSON envelope: %v\nraw output: %s", err, output)
	}
	return env
}

// --- JSON Contract Tests ---

func TestJSONContract_Capabilities(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("capabilities", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)

	if env["command"] != "capabilities" {
		t.Errorf("command: got %v, want capabilities", env["command"])
	}
	if env["ok"] != true {
		t.Error("ok should be true")
	}

	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data should be an object")
	}
	commands, ok := data["commands"].([]any)
	if !ok {
		t.Fatal("data.commands should be an array")
	}
	if len(commands) < 30 {
		t.Errorf("commands count: got %d, want >= 30", len(commands))
	}
}

func TestJSONContract_Report(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("report", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)

	if env["command"] != "report" {
		t.Errorf("command: got %v", env["command"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data should be an object")
	}
	if _, ok := data["staleness_ratio"]; !ok {
		t.Error("missing staleness_ratio in data")
	}
	if _, ok := data["promotion_rate"]; !ok {
		t.Error("missing promotion_rate in data")
	}
}

func TestJSONContract_Check(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("check", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)

	if env["command"] != "check" {
		t.Errorf("command: got %v", env["command"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data should be an object")
	}
	summary, ok := data["summary"].(map[string]any)
	if !ok {
		t.Fatal("data.summary should be an object")
	}
	if summary["files_scanned"].(float64) != 3 {
		t.Errorf("files_scanned: got %v, want 3", summary["files_scanned"])
	}
}

func TestJSONContract_GC(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("gc", "--dry-run", "--days", "45", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)

	if env["command"] != "gc" {
		t.Errorf("command: got %v", env["command"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data should be an object")
	}
	if _, ok := data["scanned"]; !ok {
		t.Error("missing scanned in data")
	}
}

func TestJSONContract_Doctor(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("doctor", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)

	if env["command"] != "doctor" {
		t.Errorf("command: got %v", env["command"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data should be an object")
	}
	if _, ok := data["checks"]; !ok {
		t.Error("missing checks in data")
	}
}

func TestJSONContract_Index(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("index", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)

	if env["command"] != "index" {
		t.Errorf("command: got %v", env["command"])
	}
}

func TestJSONContract_PolicyGate(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("policy", "gate", "--pr", "refs/pull/1/head", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)

	if env["command"] != "policy gate" {
		t.Errorf("command: got %v", env["command"])
	}
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatal("data should be an object")
	}
	if data["target_ref"] != "refs/pull/1/head" {
		t.Errorf("target_ref: got %v", data["target_ref"])
	}
}

// --- Smoke Tests ---

func TestSmoke_Capabilities(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("capabilities", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Commands") {
		t.Errorf("expected Commands in output, got: %s", out)
	}
}

func TestSmoke_Report(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("report", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Staleness ratio") {
		t.Errorf("expected Staleness ratio in output, got: %s", out)
	}
}

func TestSmoke_Doctor(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("doctor", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Doctor:") {
		t.Errorf("expected Doctor: in output, got: %s", out)
	}
}

func TestSmoke_Check(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("check", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Scanned") {
		t.Errorf("expected Scanned in output, got: %s", out)
	}
}

func TestSmoke_GC(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("gc", "--dry-run", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Scanned") {
		t.Errorf("expected Scanned in output, got: %s", out)
	}
}

func TestSmoke_Index(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("index", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Index built") {
		t.Errorf("expected Index built in output, got: %s", out)
	}
}

func TestSmoke_Tools(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("tools", "--format", "text")
	if err != nil {
		// tools may fail if required tools are missing — that's ok
		_ = out
		return
	}
	// Should produce some output about tool availability
	if len(out) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestSmoke_Plan(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("plan", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "--dry-run") {
		t.Errorf("expected --dry-run suggestion, got: %s", out)
	}
}

func TestSmoke_NoteList(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("note", "list", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "3 notes") {
		t.Errorf("expected 3 notes, got: %s", out)
	}
}

func TestSmoke_NoteNew(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("note", "new", "--type", "context", "--slug", "test", "--title", "Test Note", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Created") {
		t.Errorf("expected Created in output, got: %s", out)
	}
}

func TestSmoke_ReviewPromotions(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("review", "promotions", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "promotion candidates") {
		t.Errorf("expected promotion candidates in output, got: %s", out)
	}
}

func TestSmoke_OptimizeRecommend(t *testing.T) {
	setupTestRepo(t)
	out, err := executeCommand("optimize", "recommend", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Window:") {
		t.Errorf("expected Window: in output, got: %s", out)
	}
}

// --- Flag Tests ---

func TestFlags_GCDays(t *testing.T) {
	setupTestRepoWithNotes(t)
	out, err := executeCommand("gc", "--dry-run", "--days", "1", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)
	data := env["data"].(map[string]any)
	// With days=1, all recently created notes should still be active
	if data["marked"].(float64) != 0 {
		t.Errorf("no notes should be marked stale with days=1, got %v", data["marked"])
	}
}

func TestFlags_CheckStrictSchema(t *testing.T) {
	dir := setupTestRepo(t)
	// Create a note without explicit schema_version
	noteDir := filepath.Join(dir, ".ai", "context")
	os.MkdirAll(noteDir, 0o755)
	content := `---
id: context-20250101-test
title: Test Note
type: context
status: active
created: "2025-01-01"
updated: "2025-01-01"
---

Content.
`
	os.WriteFile(filepath.Join(noteDir, "context-20250101-test.md"), []byte(content), 0o644)

	out, err := executeCommand("check", "--strict-schema", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)
	// strict-schema should fail for notes missing schema_version
	if env["ok"] != false {
		t.Error("strict-schema check should fail for note without schema_version")
	}
}

func TestFlags_NoteListFilter(t *testing.T) {
	dir := setupTestRepoWithNotes(t)
	aiDir := filepath.Join(dir, ".ai")

	// Close one note
	allNotes, _ := notes.ListNotes(aiDir, "", "")
	if len(allNotes) > 0 {
		notes.CloseNote(allNotes[0].Path, "closed")
	}

	// Filter by status=active
	out, err := executeCommand("note", "list", "--status", "active", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)
	data := env["data"].(map[string]any)
	count := data["count"].(float64)
	if count != 2 {
		t.Errorf("active notes: got %v, want 2", count)
	}
}

// --- Dry-Run Tests ---

func TestDryRun_Init(t *testing.T) {
	setupTestRepo(t)
	// Note: explicitly set --format json to parse envelope
	out, err := executeCommand("init", "--dry-run", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	env := parseEnvelope(t, out)
	data := env["data"].(map[string]any)
	// Dry run should produce actions but not modify filesystem
	actions := data["actions"].([]any)
	if len(actions) == 0 {
		t.Error("init --dry-run should list actions")
	}
}

func TestDryRun_GCNoSideEffects(t *testing.T) {
	dir := setupTestRepo(t)
	aiDir := filepath.Join(dir, ".ai")

	// Create an old note that would be marked stale
	noteDir := filepath.Join(aiDir, "context")
	os.MkdirAll(noteDir, 0o755)
	content := `---
id: context-20240101-old
title: Old Note
type: context
status: active
created: "2024-01-01"
updated: "2024-01-01"
schema_version: 1
---

Old content.
`
	notePath := filepath.Join(noteDir, "context-20240101-old.md")
	os.WriteFile(notePath, []byte(content), 0o644)

	// Run GC dry-run
	_, err := executeCommand("gc", "--dry-run", "--days", "45", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}

	// Verify note was NOT modified
	note, _ := notes.ParseNote(notePath)
	if note.Frontmatter.Status != "active" {
		t.Errorf("dry-run should not modify note status, got %s", note.Frontmatter.Status)
	}
}

// --- Error Path Tests ---

func TestError_ApplyMissingPlanID(t *testing.T) {
	setupTestRepo(t)
	_, err := executeCommand("apply")
	if err == nil {
		t.Error("apply without --plan-id should fail")
	}
}

func TestError_RollbackMissingTx(t *testing.T) {
	setupTestRepo(t)
	_, err := executeCommand("rollback")
	if err == nil {
		t.Error("rollback without --tx should fail")
	}
}

func TestError_NoteCloseMissingArg(t *testing.T) {
	setupTestRepo(t)
	_, err := executeCommand("note", "close")
	if err == nil {
		t.Error("note close without path arg should fail")
	}
}

func TestError_NotePromoteMissingArg(t *testing.T) {
	setupTestRepo(t)
	_, err := executeCommand("note", "promote")
	if err == nil {
		t.Error("note promote without path arg should fail")
	}
}

// --- Envelope Field Tests ---

func TestEnvelopeFields_AllCommands(t *testing.T) {
	setupTestRepoWithNotes(t)

	// Override gitLogFn to avoid real git calls
	origGitLog := gitLogFn
	gitLogFn = func(from, to string) ([]string, error) {
		return []string{"feat: test feature", "fix: test fix"}, nil
	}
	t.Cleanup(func() { gitLogFn = origGitLog })

	commands := [][]string{
		{"capabilities", "--format", "json"},
		{"report", "--format", "json"},
		{"check", "--format", "json"},
		{"gc", "--dry-run", "--format", "json"},
		{"doctor", "--format", "json"},
		{"index", "--format", "json"},
		{"plan", "--format", "json"},
		{"note", "list", "--format", "json"},
		{"review", "promotions", "--format", "json"},
		{"optimize", "recommend", "--format", "json"},
		{"policy", "gate", "--format", "json"},
		{"commit", "lint", "--format", "json"},
		{"changelog", "--dry-run", "--format", "json"},
	}

	for _, args := range commands {
		t.Run(strings.Join(args[:len(args)-2], " "), func(t *testing.T) {
			out, err := executeCommand(args...)
			if err != nil {
				t.Fatalf("command %v: %v", args, err)
			}

			env := parseEnvelope(t, out)

			// Every envelope must have these fields
			required := []string{"command", "ok", "duration_ms", "warnings", "errors"}
			for _, field := range required {
				if _, ok := env[field]; !ok {
					t.Errorf("missing required envelope field: %s", field)
				}
			}
		})
	}
}
