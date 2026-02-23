package execution

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StepAction describes what a plan step does.
type StepAction string

const (
	ActionWrite    StepAction = "write"
	ActionOverwrite StepAction = "overwrite"
	ActionDelete   StepAction = "delete"
	ActionExec     StepAction = "exec"
)

// PlanStep is a single mutation in an execution plan.
type PlanStep struct {
	Index      int        `json:"index"`
	Action     StepAction `json:"action"`
	TargetPath string     `json:"target_path,omitempty"`
	BackupPath string     `json:"backup_path,omitempty"`
	Detail     string     `json:"detail"`
}

// Plan is an immutable execution plan for a mutating command.
type Plan struct {
	PlanID         string     `json:"plan_id"`
	Command        string     `json:"command"`
	Args           []string   `json:"args,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	Steps          []PlanStep `json:"steps"`
	IdempotencyKey string     `json:"idempotency_key"`
	Preconditions  []string   `json:"preconditions,omitempty"`
}

// ComputeIdempotencyKey derives a stable hash from command, args, and step details.
func ComputeIdempotencyKey(command string, args []string, steps []PlanStep) string {
	h := sha256.New()
	h.Write([]byte(command))
	for _, a := range args {
		h.Write([]byte(a))
	}
	for _, s := range steps {
		h.Write([]byte(fmt.Sprintf("%d:%s:%s:%s", s.Index, s.Action, s.TargetPath, s.Detail)))
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

// NewPlan creates a new immutable plan.
func NewPlan(planID, command string, args []string, steps []PlanStep, preconditions []string) *Plan {
	return &Plan{
		PlanID:         planID,
		Command:        command,
		Args:           args,
		CreatedAt:      time.Now().UTC(),
		Steps:          steps,
		IdempotencyKey: ComputeIdempotencyKey(command, args, steps),
		Preconditions:  preconditions,
	}
}

// PlanStore persists and loads plans from a directory.
type PlanStore struct {
	dir string
}

// NewPlanStore creates a plan store at the given directory.
func NewPlanStore(dir string) *PlanStore {
	return &PlanStore{dir: dir}
}

// Save writes a plan to disk as JSON.
func (s *PlanStore) Save(plan *Plan) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("creating plan directory: %w", err)
	}
	path := filepath.Join(s.dir, plan.PlanID+".json")
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads a plan from disk by plan ID.
func (s *PlanStore) Load(planID string) (*Plan, error) {
	path := filepath.Join(s.dir, planID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading plan %s: %w", planID, err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parsing plan: %w", err)
	}
	return &plan, nil
}

// StepFunc is a function that executes a single plan step.
// It returns an optional backup path for rollback purposes.
type StepFunc func(step PlanStep) (backupPath string, err error)

// RollbackFunc is a function that rolls back a single step using its journal entry.
type RollbackFunc func(entry JournalEntry) error

// Engine orchestrates plan execution with journaling.
type Engine struct {
	journal *Journal
}

// NewEngine creates an execution engine.
func NewEngine(journal *Journal) *Engine {
	return &Engine{journal: journal}
}

// Apply executes a plan's steps within a transaction, journaling each step.
// If a step fails, it attempts best-effort rollback of completed steps.
func (e *Engine) Apply(plan *Plan, txID string, execute StepFunc, rollback RollbackFunc) error {
	// Journal: tx_started
	if err := e.journal.Append(JournalEntry{
		Timestamp: time.Now().UTC(),
		TxID:      txID,
		PlanID:    plan.PlanID,
		EventType: EventTxStarted,
		Command:   plan.Command,
	}); err != nil {
		return fmt.Errorf("journaling tx_started: %w", err)
	}

	var appliedEntries []JournalEntry

	for _, step := range plan.Steps {
		backupPath, err := execute(step)

		entry := JournalEntry{
			Timestamp:  time.Now().UTC(),
			TxID:       txID,
			PlanID:     plan.PlanID,
			EventType:  EventStepApplied,
			StepIndex:  step.Index,
			BackupPath: backupPath,
			Detail:     step.Detail,
		}

		if err != nil {
			entry.EventType = EventTxFailed
			entry.ErrorMessage = err.Error()
			e.journal.Append(entry)

			// Best-effort rollback of previously applied steps
			e.rollbackSteps(txID, plan.PlanID, appliedEntries, rollback)
			return fmt.Errorf("step %d failed: %w", step.Index, err)
		}

		if jErr := e.journal.Append(entry); jErr != nil {
			return fmt.Errorf("journaling step %d: %w", step.Index, jErr)
		}
		appliedEntries = append(appliedEntries, entry)
	}

	// Journal: tx_committed
	return e.journal.Append(JournalEntry{
		Timestamp: time.Now().UTC(),
		TxID:      txID,
		PlanID:    plan.PlanID,
		EventType: EventTxCommitted,
	})
}

// Rollback rolls back a transaction using the journal and rollback function.
func (e *Engine) Rollback(txID string, rollback RollbackFunc) error {
	entries, err := e.journal.EntriesForTx(txID)
	if err != nil {
		return fmt.Errorf("reading journal for tx %s: %w", txID, err)
	}

	// Collect applied steps in reverse order
	var applied []JournalEntry
	for _, entry := range entries {
		if entry.EventType == EventStepApplied {
			applied = append(applied, entry)
		}
	}

	return e.rollbackSteps(txID, "", applied, rollback)
}

func (e *Engine) rollbackSteps(txID, planID string, applied []JournalEntry, rollback RollbackFunc) error {
	// Rollback in reverse order
	var lastErr error
	for i := len(applied) - 1; i >= 0; i-- {
		entry := applied[i]
		if err := rollback(entry); err != nil {
			lastErr = err
			e.journal.Append(JournalEntry{
				Timestamp:    time.Now().UTC(),
				TxID:         txID,
				PlanID:       planID,
				EventType:    EventTxFailed,
				StepIndex:    entry.StepIndex,
				ErrorMessage: fmt.Sprintf("rollback failed: %s", err),
			})
			continue
		}
		e.journal.Append(JournalEntry{
			Timestamp: time.Now().UTC(),
			TxID:      txID,
			PlanID:    planID,
			EventType: EventStepRolledBack,
			StepIndex: entry.StepIndex,
		})
	}

	if lastErr != nil {
		return lastErr
	}

	return e.journal.Append(JournalEntry{
		Timestamp: time.Now().UTC(),
		TxID:      txID,
		PlanID:    planID,
		EventType: EventTxRolledBack,
	})
}
