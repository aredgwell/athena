// Package e2e runs end-to-end tests against the built athena binary.
// These tests exercise the real process boundary — the binary is built once
// via TestMain and invoked via os/exec for each test case.
package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary once for all tests.
	tmp, err := os.MkdirTemp("", "athena-e2e-*")
	if err != nil {
		panic(err)
	}
	binaryPath = filepath.Join(tmp, "athena")

	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/athena")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// athena runs the binary with args in the given working directory.
func athena(t *testing.T, dir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = dir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

// setupRepo creates an athena-managed repo in a temp directory.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".athena"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".ai"), 0o755)
	os.WriteFile(filepath.Join(dir, "athena.toml"), []byte(`version = 2
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
`), 0o644)
	return dir
}

// parseJSON parses JSON output into a generic map.
func parseJSON(t *testing.T, output string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(output), &m); err != nil {
		t.Fatalf("failed to parse JSON: %v\nraw: %s", err, output)
	}
	return m
}

// --- Binary Lifecycle ---

func TestBinary_Version(t *testing.T) {
	dir := t.TempDir()
	out, _, code := athena(t, dir, "version")
	if code != 0 {
		t.Fatalf("exit code: %d", code)
	}
	if !strings.Contains(out, "athena version") {
		t.Errorf("expected version output, got: %s", out)
	}
}

func TestBinary_Help(t *testing.T) {
	dir := t.TempDir()
	out, _, code := athena(t, dir, "--help")
	if code != 0 {
		t.Fatalf("exit code: %d", code)
	}
	if !strings.Contains(out, "athena") {
		t.Error("help should mention athena")
	}
	if !strings.Contains(out, "Available Commands") {
		t.Error("help should list available commands")
	}
}

func TestBinary_Capabilities_JSON(t *testing.T) {
	dir := setupRepo(t)
	out, _, code := athena(t, dir, "capabilities", "--format", "json")
	if code != 0 {
		t.Fatalf("exit code: %d", code)
	}
	env := parseJSON(t, out)
	if env["command"] != "capabilities" {
		t.Errorf("command: %v", env["command"])
	}
	if env["ok"] != true {
		t.Error("ok should be true")
	}
	data := env["data"].(map[string]any)
	commands := data["commands"].([]any)
	if len(commands) < 30 {
		t.Errorf("commands: got %d, want >= 30", len(commands))
	}
}

func TestBinary_Capabilities_Text(t *testing.T) {
	dir := setupRepo(t)
	out, _, code := athena(t, dir, "capabilities", "--format", "text")
	if code != 0 {
		t.Fatalf("exit code: %d", code)
	}
	if !strings.Contains(out, "Commands") {
		t.Error("text output should contain Commands")
	}
}

// --- Init Workflow ---

func TestBinary_Init(t *testing.T) {
	dir := t.TempDir()
	// init should create .athena/ and .ai/ structure
	out, _, code := athena(t, dir, "init", "--format", "json")
	if code != 0 {
		t.Fatalf("exit code: %d", code)
	}
	env := parseJSON(t, out)
	if env["ok"] != true {
		t.Error("init should succeed")
	}

	// Verify scaffold was created
	if _, err := os.Stat(filepath.Join(dir, "athena.toml")); err != nil {
		t.Error("athena.toml should exist after init")
	}
	if _, err := os.Stat(filepath.Join(dir, ".ai")); err != nil {
		t.Error(".ai should exist after init")
	}
}

func TestBinary_InitDryRun(t *testing.T) {
	dir := t.TempDir()
	out, _, code := athena(t, dir, "init", "--dry-run", "--format", "json")
	if code != 0 {
		t.Fatalf("exit code: %d", code)
	}
	env := parseJSON(t, out)
	data := env["data"].(map[string]any)
	actions := data["actions"].([]any)
	if len(actions) == 0 {
		t.Error("dry-run should list planned actions")
	}

	// Verify nothing was created
	if _, err := os.Stat(filepath.Join(dir, "athena.toml")); err == nil {
		t.Error("dry-run should not create athena.toml")
	}
}

// --- Note Lifecycle ---

func TestBinary_NoteLifecycle(t *testing.T) {
	dir := setupRepo(t)

	// Create a note
	out, _, code := athena(t, dir, "note", "new", "--type", "context", "--slug", "auth", "--title", "Auth Analysis", "--format", "json")
	if code != 0 {
		t.Fatalf("note new exit code: %d", code)
	}
	env := parseJSON(t, out)
	if env["ok"] != true {
		t.Error("note new should succeed")
	}

	// List notes
	out, _, code = athena(t, dir, "note", "list", "--format", "json")
	if code != 0 {
		t.Fatalf("note list exit code: %d", code)
	}
	env = parseJSON(t, out)
	data := env["data"].(map[string]any)
	if data["count"].(float64) != 1 {
		t.Errorf("expected 1 note, got %v", data["count"])
	}

	// Find the note file for close/promote
	notes := data["notes"].([]any)
	note := notes[0].(map[string]any)
	notePath := note["path"].(string)

	// Close the note
	out, _, code = athena(t, dir, "note", "close", notePath, "--format", "json")
	if code != 0 {
		t.Fatalf("note close exit code: %d", code)
	}
	env = parseJSON(t, out)
	if env["ok"] != true {
		t.Error("note close should succeed")
	}

	// Verify status changed
	out, _, code = athena(t, dir, "note", "list", "--status", "closed", "--format", "json")
	if code != 0 {
		t.Fatalf("note list --status closed exit code: %d", code)
	}
	env = parseJSON(t, out)
	data = env["data"].(map[string]any)
	if data["count"].(float64) != 1 {
		t.Errorf("expected 1 closed note, got %v", data["count"])
	}
}

// --- Check + Index + Search Pipeline ---

func TestBinary_CheckIndexSearch(t *testing.T) {
	dir := setupRepo(t)

	// Create some notes
	athena(t, dir, "note", "new", "--type", "context", "--slug", "auth", "--title", "Authentication middleware")
	athena(t, dir, "note", "new", "--type", "investigation", "--slug", "perf", "--title", "Performance bottleneck")

	// Check
	out, _, code := athena(t, dir, "check", "--format", "json")
	if code != 0 {
		t.Fatalf("check exit code: %d", code)
	}
	env := parseJSON(t, out)
	if env["ok"] != true {
		t.Error("check should pass for valid notes")
	}
	data := env["data"].(map[string]any)
	summary := data["summary"].(map[string]any)
	if summary["files_scanned"].(float64) != 2 {
		t.Errorf("files scanned: %v", summary["files_scanned"])
	}

	// Index (builds both metadata and search index)
	out, _, code = athena(t, dir, "index", "--format", "json")
	if code != 0 {
		t.Fatalf("index exit code: %d", code)
	}
	env = parseJSON(t, out)
	if env["ok"] != true {
		t.Error("index should succeed")
	}

	// Verify search index was created
	searchIdxPath := filepath.Join(dir, ".ai", "search-index.json")
	if _, err := os.Stat(searchIdxPath); err != nil {
		t.Error("search-index.json should exist after index")
	}

	// Search
	out, _, code = athena(t, dir, "context", "search", "authentication", "--format", "json")
	if code != 0 {
		t.Fatalf("context search exit code: %d", code)
	}
	env = parseJSON(t, out)
	if env["ok"] != true {
		t.Error("search should succeed")
	}
	data = env["data"].(map[string]any)
	results := data["results"].([]any)
	if len(results) == 0 {
		t.Error("search for 'authentication' should return results")
	}
}

// --- GC Dry-Run ---

func TestBinary_GCDryRun(t *testing.T) {
	dir := setupRepo(t)

	// Create a note
	athena(t, dir, "note", "new", "--type", "context", "--slug", "test", "--title", "Test")

	out, _, code := athena(t, dir, "gc", "--dry-run", "--format", "json")
	if code != 0 {
		t.Fatalf("gc exit code: %d", code)
	}
	env := parseJSON(t, out)
	if env["ok"] != true {
		t.Error("gc dry-run should succeed")
	}
}

// --- Doctor ---

func TestBinary_Doctor(t *testing.T) {
	dir := setupRepo(t)
	out, _, code := athena(t, dir, "doctor", "--format", "json")
	if code != 0 {
		t.Fatalf("doctor exit code: %d", code)
	}
	env := parseJSON(t, out)
	if env["command"] != "doctor" {
		t.Errorf("command: %v", env["command"])
	}
	data := env["data"].(map[string]any)
	if _, ok := data["checks"]; !ok {
		t.Error("doctor should return checks data")
	}
}

// --- Report ---

func TestBinary_Report(t *testing.T) {
	dir := setupRepo(t)
	athena(t, dir, "note", "new", "--type", "context", "--slug", "test", "--title", "Test")

	out, _, code := athena(t, dir, "report", "--format", "json")
	if code != 0 {
		t.Fatalf("report exit code: %d", code)
	}
	env := parseJSON(t, out)
	if env["ok"] != true {
		t.Error("report should succeed")
	}
	data := env["data"].(map[string]any)
	if _, ok := data["staleness_ratio"]; !ok {
		t.Error("report should contain staleness_ratio")
	}
}

// --- Policy Gate ---

func TestBinary_PolicyGate(t *testing.T) {
	dir := setupRepo(t)
	out, _, code := athena(t, dir, "policy", "gate", "--pr", "refs/pull/1/head", "--format", "json")
	if code != 0 {
		t.Fatalf("policy gate exit code: %d", code)
	}
	env := parseJSON(t, out)
	if env["command"] != "policy gate" {
		t.Errorf("command: %v", env["command"])
	}
}

// --- Envelope Contract ---

func TestBinary_EnvelopeContract(t *testing.T) {
	dir := setupRepo(t)
	athena(t, dir, "note", "new", "--type", "context", "--slug", "test", "--title", "Test")

	commands := [][]string{
		{"capabilities", "--format", "json"},
		{"check", "--format", "json"},
		{"report", "--format", "json"},
		{"doctor", "--format", "json"},
		{"index", "--format", "json"},
		{"gc", "--dry-run", "--format", "json"},
		{"note", "list", "--format", "json"},
		{"review", "promotions", "--format", "json"},
		{"optimize", "recommend", "--format", "json"},
		{"policy", "gate", "--format", "json"},
	}

	required := []string{"command", "ok", "duration_ms", "warnings", "errors"}

	for _, args := range commands {
		name := strings.Join(args[:len(args)-2], " ")
		t.Run(name, func(t *testing.T) {
			out, _, code := athena(t, dir, args...)
			if code != 0 {
				t.Fatalf("exit code %d for %v", code, args)
			}
			env := parseJSON(t, out)
			for _, field := range required {
				if _, ok := env[field]; !ok {
					t.Errorf("missing required envelope field: %s", field)
				}
			}
		})
	}
}

// --- Error Paths ---

func TestBinary_ErrorMissingRepo(t *testing.T) {
	dir := t.TempDir()
	// Running check in a dir with no athena structure should still work
	// (it falls back to cwd)
	_, _, code := athena(t, dir, "check", "--format", "json")
	// May fail but should not crash
	if code < 0 {
		t.Error("should not crash")
	}
}

func TestBinary_ErrorNoteCloseMissingArg(t *testing.T) {
	dir := setupRepo(t)
	_, _, code := athena(t, dir, "note", "close")
	if code == 0 {
		t.Error("note close without path should fail")
	}
}

func TestBinary_ErrorApplyMissingPlanID(t *testing.T) {
	dir := setupRepo(t)
	_, _, code := athena(t, dir, "apply")
	if code == 0 {
		t.Error("apply without --plan-id should fail")
	}
}

// --- Completion ---

func TestBinary_Completion(t *testing.T) {
	dir := t.TempDir()
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			out, _, code := athena(t, dir, "completion", shell)
			if code != 0 {
				t.Fatalf("completion %s exit code: %d", shell, code)
			}
			if len(out) == 0 {
				t.Error("completion output should not be empty")
			}
		})
	}
}
