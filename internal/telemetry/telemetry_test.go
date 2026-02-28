package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndReadAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry.jsonl")
	store := NewStore(path)

	now := time.Now().UTC().Truncate(time.Second)
	rec := Record{
		Timestamp:       now,
		Command:         "note promote",
		RunID:           "run_001",
		TaskID:          "task_auth",
		AgentName:       "codex",
		Model:           "gpt-5-codex",
		ExecutionTimeMS: 42,
		PromptTokens:    812,
		CompletionTokens: 221,
		TotalTokens:     1033,
		CostUSD:         0.0184,
		IsTTY:           false,
		ExitCode:        0,
		ErrorCode:       nil,
		PolicyLevel:     "standard",
		IsDryRun:        false,
		Context:         map[string]string{"note_id": "improvement-20260220-auth-flow"},
	}

	if err := store.Append(rec); err != nil {
		t.Fatalf("append: %v", err)
	}

	records, err := store.ReadAll()
	if err != nil {
		t.Fatalf("readall: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records: got %d, want 1", len(records))
	}

	got := records[0]
	if got.Command != "note promote" {
		t.Errorf("command: got %s, want note promote", got.Command)
	}
	if got.RunID != "run_001" {
		t.Errorf("run_id: got %s, want run_001", got.RunID)
	}
	if got.TotalTokens != 1033 {
		t.Errorf("total_tokens: got %d, want 1033", got.TotalTokens)
	}
	if got.CostUSD != 0.0184 {
		t.Errorf("cost_usd: got %f, want 0.0184", got.CostUSD)
	}
	if got.ErrorCode != nil {
		t.Errorf("error_code: got %v, want nil", got.ErrorCode)
	}
	if got.Context["note_id"] != "improvement-20260220-auth-flow" {
		t.Errorf("context[note_id]: got %s", got.Context["note_id"])
	}
}

func TestAppendMultiple(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry.jsonl")
	store := NewStore(path)

	for i := 0; i < 3; i++ {
		err := store.Append(Record{
			Timestamp: time.Now().UTC(),
			Command:   "gc",
			RunID:     "run_multi",
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	records, err := store.ReadAll()
	if err != nil {
		t.Fatalf("readall: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("records: got %d, want 3", len(records))
	}
}

func TestReadAllMissingFile(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nonexistent.jsonl"))

	records, err := store.ReadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Errorf("records: got %v, want nil", records)
	}
}

func TestErrorCodeJSON(t *testing.T) {
	t.Run("null error_code", func(t *testing.T) {
		rec := Record{Command: "init", ErrorCode: nil}
		data, _ := json.Marshal(rec)
		var m map[string]interface{}
		json.Unmarshal(data, &m)

		if m["error_code"] != nil {
			t.Errorf("error_code should be null, got %v", m["error_code"])
		}
	})

	t.Run("non-null error_code", func(t *testing.T) {
		code := "ATHENA-CONF-001"
		rec := Record{Command: "init", ErrorCode: &code}
		data, _ := json.Marshal(rec)
		var m map[string]interface{}
		json.Unmarshal(data, &m)

		if m["error_code"] != "ATHENA-CONF-001" {
			t.Errorf("error_code: got %v, want ATHENA-CONF-001", m["error_code"])
		}
	})
}

func TestCoverage(t *testing.T) {
	t.Run("zero commands", func(t *testing.T) {
		if got := Coverage(nil, 0); got != 0 {
			t.Errorf("coverage: got %f, want 0", got)
		}
	})

	t.Run("partial coverage", func(t *testing.T) {
		records := make([]Record, 3)
		got := Coverage(records, 4)
		if got != 0.75 {
			t.Errorf("coverage: got %f, want 0.75", got)
		}
	})

	t.Run("full coverage", func(t *testing.T) {
		records := make([]Record, 5)
		got := Coverage(records, 5)
		if got != 1.0 {
			t.Errorf("coverage: got %f, want 1.0", got)
		}
	})
}

func TestFilterByRunID(t *testing.T) {
	records := []Record{
		{Command: "init", RunID: "run_a"},
		{Command: "gc", RunID: "run_b"},
		{Command: "note new", RunID: "run_a"},
		{Command: "check", RunID: ""},
	}

	filtered := FilterByRunID(records, "run_a")
	if len(filtered) != 2 {
		t.Fatalf("filtered: got %d, want 2", len(filtered))
	}
	if filtered[0].Command != "init" {
		t.Errorf("filtered[0]: got %s, want init", filtered[0].Command)
	}
	if filtered[1].Command != "note new" {
		t.Errorf("filtered[1]: got %s, want note new", filtered[1].Command)
	}
}

func TestFilterByRunIDNoMatch(t *testing.T) {
	records := []Record{{Command: "gc", RunID: "run_x"}}
	filtered := FilterByRunID(records, "run_missing")
	if len(filtered) != 0 {
		t.Errorf("filtered: got %d, want 0", len(filtered))
	}
}

func TestJSONLinesFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry.jsonl")
	store := NewStore(path)

	store.Append(Record{Command: "init", Timestamp: time.Now().UTC()})
	store.Append(Record{Command: "gc", Timestamp: time.Now().UTC()})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Each record should be a single line ending with \n
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("lines: got %d, want 2 (JSONLines format)", lines)
	}
}

func TestTokenCorrelation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "telemetry.jsonl")
	store := NewStore(path)

	// Simulate agent task with multiple commands sharing a run_id
	runID := "run_20260223_corr"
	commands := []struct {
		cmd    string
		tokens int
		cost   float64
	}{
		{"context pack", 500, 0.01},
		{"note new", 200, 0.004},
		{"check", 100, 0.002},
	}

	for _, c := range commands {
		store.Append(Record{
			Timestamp:   time.Now().UTC(),
			Command:     c.cmd,
			RunID:       runID,
			AgentName:   "claude",
			TotalTokens: c.tokens,
			CostUSD:     c.cost,
		})
	}

	// Add unrelated record
	store.Append(Record{
		Timestamp: time.Now().UTC(),
		Command:   "gc",
		RunID:     "run_other",
	})

	records, _ := store.ReadAll()
	if len(records) != 4 {
		t.Fatalf("total records: got %d, want 4", len(records))
	}

	correlated := FilterByRunID(records, runID)
	if len(correlated) != 3 {
		t.Fatalf("correlated: got %d, want 3", len(correlated))
	}

	// Verify token aggregation capability
	var totalTokens int
	var totalCost float64
	for _, r := range correlated {
		totalTokens += r.TotalTokens
		totalCost += r.CostUSD
	}
	if totalTokens != 800 {
		t.Errorf("total tokens for run: got %d, want 800", totalTokens)
	}
	if totalCost < 0.015 || totalCost > 0.017 {
		t.Errorf("total cost for run: got %f, want ~0.016", totalCost)
	}
}

func TestActorField(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "telemetry.jsonl"))

	rec := Record{
		Command: "check",
		Actor:   "claude-code",
	}

	if err := s.Append(rec); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	records, _ := s.ReadAll()
	if records[0].Actor != "claude-code" {
		t.Errorf("expected actor claude-code, got %s", records[0].Actor)
	}
}
