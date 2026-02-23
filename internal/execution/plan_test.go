package execution

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestPlanCommand(t *testing.T) {
	t.Run("create and save plan", func(t *testing.T) {
		dir := t.TempDir()
		store := NewPlanStore(dir)

		steps := []PlanStep{
			{Index: 0, Action: ActionWrite, TargetPath: "AGENTS.md", Detail: "write AGENTS.md"},
			{Index: 1, Action: ActionWrite, TargetPath: ".editorconfig", Detail: "write .editorconfig"},
		}
		plan := NewPlan("plan_test_001", "init", nil, steps, nil)

		if err := store.Save(plan); err != nil {
			t.Fatalf("save: %v", err)
		}

		loaded, err := store.Load("plan_test_001")
		if err != nil {
			t.Fatalf("load: %v", err)
		}

		if loaded.PlanID != "plan_test_001" {
			t.Errorf("plan_id: got %s", loaded.PlanID)
		}
		if loaded.Command != "init" {
			t.Errorf("command: got %s", loaded.Command)
		}
		if len(loaded.Steps) != 2 {
			t.Errorf("steps: got %d", len(loaded.Steps))
		}
		if loaded.IdempotencyKey == "" {
			t.Error("expected non-empty idempotency_key")
		}
	})

	t.Run("idempotency key stability", func(t *testing.T) {
		steps := []PlanStep{
			{Index: 0, Action: ActionWrite, TargetPath: "a.md", Detail: "write a"},
		}
		key1 := ComputeIdempotencyKey("init", nil, steps)
		key2 := ComputeIdempotencyKey("init", nil, steps)
		if key1 != key2 {
			t.Error("idempotency keys should be deterministic")
		}

		key3 := ComputeIdempotencyKey("upgrade", nil, steps)
		if key1 == key3 {
			t.Error("different commands should produce different keys")
		}
	})

	t.Run("load missing plan", func(t *testing.T) {
		store := NewPlanStore(t.TempDir())
		_, err := store.Load("nonexistent")
		if err == nil {
			t.Error("expected error for missing plan")
		}
	})
}

func TestApplyCommand(t *testing.T) {
	t.Run("successful apply", func(t *testing.T) {
		journalPath := filepath.Join(t.TempDir(), "journal.jsonl")
		journal := NewJournal(journalPath)
		engine := NewEngine(journal)

		steps := []PlanStep{
			{Index: 0, Action: ActionWrite, TargetPath: "file1.md", Detail: "write file1"},
			{Index: 1, Action: ActionWrite, TargetPath: "file2.md", Detail: "write file2"},
		}
		plan := NewPlan("plan_apply_ok", "init", nil, steps, nil)

		executed := []int{}
		stepFn := func(step PlanStep) (string, error) {
			executed = append(executed, step.Index)
			return "", nil
		}
		rollbackFn := func(entry JournalEntry) error { return nil }

		if err := engine.Apply(plan, "tx_apply_ok", stepFn, rollbackFn); err != nil {
			t.Fatalf("apply: %v", err)
		}

		if len(executed) != 2 {
			t.Errorf("expected 2 steps executed, got %d", len(executed))
		}

		entries, _ := journal.EntriesForTx("tx_apply_ok")
		// Expect: tx_started, step_applied(0), step_applied(1), tx_committed
		if len(entries) != 4 {
			t.Fatalf("expected 4 journal entries, got %d", len(entries))
		}
		if entries[0].EventType != EventTxStarted {
			t.Errorf("entry[0]: got %s, want tx_started", entries[0].EventType)
		}
		if entries[3].EventType != EventTxCommitted {
			t.Errorf("entry[3]: got %s, want tx_committed", entries[3].EventType)
		}
	})

	t.Run("apply with failure triggers rollback", func(t *testing.T) {
		journalPath := filepath.Join(t.TempDir(), "journal.jsonl")
		journal := NewJournal(journalPath)
		engine := NewEngine(journal)

		steps := []PlanStep{
			{Index: 0, Action: ActionWrite, TargetPath: "ok.md", Detail: "write ok"},
			{Index: 1, Action: ActionWrite, TargetPath: "fail.md", Detail: "write fail"},
			{Index: 2, Action: ActionWrite, TargetPath: "never.md", Detail: "write never"},
		}
		plan := NewPlan("plan_fail", "init", nil, steps, nil)

		stepFn := func(step PlanStep) (string, error) {
			if step.Index == 1 {
				return "", fmt.Errorf("disk full")
			}
			return "/backup/" + step.TargetPath, nil
		}

		rolledBack := []int{}
		rollbackFn := func(entry JournalEntry) error {
			rolledBack = append(rolledBack, entry.StepIndex)
			return nil
		}

		err := engine.Apply(plan, "tx_fail", stepFn, rollbackFn)
		if err == nil {
			t.Fatal("expected error from failed step")
		}

		// Only step 0 should be rolled back (step 1 failed, step 2 never ran)
		if len(rolledBack) != 1 || rolledBack[0] != 0 {
			t.Errorf("expected rollback of step 0, got %v", rolledBack)
		}

		entries, _ := journal.EntriesForTx("tx_fail")
		// Should contain: tx_started, step_applied(0), tx_failed(1), step_rolled_back(0), tx_rolled_back
		hasRolledBack := false
		hasFailed := false
		for _, e := range entries {
			if e.EventType == EventStepRolledBack {
				hasRolledBack = true
			}
			if e.EventType == EventTxFailed {
				hasFailed = true
			}
		}
		if !hasFailed {
			t.Error("expected tx_failed event")
		}
		if !hasRolledBack {
			t.Error("expected step_rolled_back event")
		}
	})
}

func TestRollbackCommand(t *testing.T) {
	journalPath := filepath.Join(t.TempDir(), "journal.jsonl")
	journal := NewJournal(journalPath)
	engine := NewEngine(journal)

	// Simulate a successful apply first
	steps := []PlanStep{
		{Index: 0, Action: ActionWrite, TargetPath: "a.md", Detail: "write a"},
		{Index: 1, Action: ActionWrite, TargetPath: "b.md", Detail: "write b"},
		{Index: 2, Action: ActionWrite, TargetPath: "c.md", Detail: "write c"},
	}
	plan := NewPlan("plan_rb", "init", nil, steps, nil)

	stepFn := func(step PlanStep) (string, error) {
		return "/backup/" + step.TargetPath, nil
	}
	rollbackFn := func(entry JournalEntry) error { return nil }

	engine.Apply(plan, "tx_rb", stepFn, rollbackFn)

	// Now manually rollback
	rolledBack := []int{}
	manualRollback := func(entry JournalEntry) error {
		rolledBack = append(rolledBack, entry.StepIndex)
		return nil
	}

	if err := engine.Rollback("tx_rb", manualRollback); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// Should be rolled back in reverse order: 2, 1, 0
	if len(rolledBack) != 3 {
		t.Fatalf("expected 3 rollbacks, got %d", len(rolledBack))
	}
	if rolledBack[0] != 2 || rolledBack[1] != 1 || rolledBack[2] != 0 {
		t.Errorf("expected reverse order [2,1,0], got %v", rolledBack)
	}
}

func TestPlanApplyFailureRollbackIntegration(t *testing.T) {
	dir := t.TempDir()
	journalPath := filepath.Join(dir, "journal.jsonl")
	planDir := filepath.Join(dir, "plans")

	journal := NewJournal(journalPath)
	engine := NewEngine(journal)
	store := NewPlanStore(planDir)

	// 1. Create plan
	steps := []PlanStep{
		{Index: 0, Action: ActionWrite, TargetPath: "AGENTS.md", Detail: "scaffold AGENTS.md"},
		{Index: 1, Action: ActionWrite, TargetPath: ".editorconfig", Detail: "scaffold .editorconfig"},
		{Index: 2, Action: ActionExec, Detail: "run post-init hook"},
	}
	plan := NewPlan("plan_integration", "init", []string{"--preset", "standard"}, steps, []string{"no athena.toml"})

	if err := store.Save(plan); err != nil {
		t.Fatalf("save plan: %v", err)
	}

	// 2. Load plan (simulate plan-first flow)
	loaded, err := store.Load("plan_integration")
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}

	// 3. Apply with failure on step 2
	appliedSteps := []int{}
	stepFn := func(step PlanStep) (string, error) {
		if step.Index == 2 {
			return "", fmt.Errorf("hook execution failed")
		}
		appliedSteps = append(appliedSteps, step.Index)
		return fmt.Sprintf(".athena/backups/%s.bak", step.TargetPath), nil
	}

	rolledBackSteps := []int{}
	rollbackFn := func(entry JournalEntry) error {
		rolledBackSteps = append(rolledBackSteps, entry.StepIndex)
		return nil
	}

	err = engine.Apply(loaded, "tx_integration", stepFn, rollbackFn)
	if err == nil {
		t.Fatal("expected apply to fail")
	}

	// 4. Verify: steps 0,1 applied, then rolled back
	if len(appliedSteps) != 2 {
		t.Errorf("expected 2 steps applied, got %d", len(appliedSteps))
	}
	if len(rolledBackSteps) != 2 {
		t.Errorf("expected 2 steps rolled back, got %d", len(rolledBackSteps))
	}
	// Rollback should be in reverse: step 1 first, then step 0
	if len(rolledBackSteps) == 2 && (rolledBackSteps[0] != 1 || rolledBackSteps[1] != 0) {
		t.Errorf("expected reverse rollback [1,0], got %v", rolledBackSteps)
	}

	// 5. Verify journal
	entries, _ := journal.EntriesForTx("tx_integration")
	eventTypes := make([]EventType, len(entries))
	for i, e := range entries {
		eventTypes[i] = e.EventType
	}

	// Should see: tx_started, step_applied(0), step_applied(1), tx_failed(2),
	// step_rolled_back(1), step_rolled_back(0), tx_rolled_back
	if len(entries) < 5 {
		t.Errorf("expected at least 5 journal entries, got %d: %v", len(entries), eventTypes)
	}
}
