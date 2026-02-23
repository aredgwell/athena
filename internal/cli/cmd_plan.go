package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/amr-athena/athena/internal/execution"
	"github.com/spf13/cobra"
)

func runPlan(cmd *cobra.Command, args []string) error {
	start := time.Now()
	env := NewEnvelope("plan", time.Since(start)).WithData(map[string]any{
		"message": "Use 'athena <command> --dry-run' to preview mutations.",
	})
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintln(w, "Use 'athena <command> --dry-run' to preview mutations.")
	})
	return nil
}

func runApply(cmd *cobra.Command, args []string) error {
	start := time.Now()
	planID, _ := cmd.Flags().GetString("plan-id")
	if planID == "" {
		return fmt.Errorf("--plan-id is required")
	}

	planDir := rc.cfg.Execution.PlanDir
	if planDir == "" {
		planDir = filepath.Join(athenaDir(), "plans")
	}
	journalPath := rc.cfg.Execution.JournalPath
	if journalPath == "" {
		journalPath = filepath.Join(athenaDir(), "ops-journal.jsonl")
	}

	store := execution.NewPlanStore(planDir)
	journal := execution.NewJournal(journalPath)
	engine := execution.NewEngine(journal)

	plan, err := store.Load(planID)
	if err != nil {
		return err
	}

	txID := fmt.Sprintf("tx_%s_%d", planID, time.Now().Unix())
	err = engine.Apply(plan, txID, func(step execution.PlanStep) (string, error) {
		if err := os.MkdirAll(filepath.Dir(step.TargetPath), 0o755); err != nil {
			return "", err
		}
		return "", os.WriteFile(step.TargetPath, []byte(step.Detail), 0o644)
	}, func(entry execution.JournalEntry) error {
		if entry.BackupPath != "" {
			data, readErr := os.ReadFile(entry.BackupPath)
			if readErr != nil {
				return readErr
			}
			// Restore to the original target by reading Detail (which held the target path)
			return os.WriteFile(entry.Detail, data, 0o644)
		}
		return nil
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("apply", time.Since(start)).WithData(map[string]any{
		"plan_id": planID,
		"tx_id":   txID,
	})
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Applied plan %s (tx: %s)\n", planID, txID)
	})
	return nil
}

func runRollback(cmd *cobra.Command, args []string) error {
	start := time.Now()
	txID, _ := cmd.Flags().GetString("tx")
	if txID == "" {
		return fmt.Errorf("--tx is required")
	}

	journalPath := rc.cfg.Execution.JournalPath
	if journalPath == "" {
		journalPath = filepath.Join(athenaDir(), "ops-journal.jsonl")
	}

	journal := execution.NewJournal(journalPath)
	engine := execution.NewEngine(journal)

	err := engine.Rollback(txID, func(entry execution.JournalEntry) error {
		if entry.BackupPath != "" {
			data, readErr := os.ReadFile(entry.BackupPath)
			if readErr != nil {
				return readErr
			}
			return os.WriteFile(entry.Detail, data, 0o644)
		}
		return nil
	})
	if err != nil {
		return err
	}

	env := NewEnvelope("rollback", time.Since(start)).WithData(map[string]any{
		"tx_id": txID,
	})
	writeOutput(cmd, env, func(w io.Writer) {
		fmt.Fprintf(w, "Rolled back transaction %s\n", txID)
	})
	return nil
}
