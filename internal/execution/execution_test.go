package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJournalPrimitives(t *testing.T) {
	t.Run("append and read", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ops-journal.jsonl")
		j := NewJournal(path)

		entry1 := JournalEntry{
			Timestamp: time.Now().UTC(),
			TxID:      "tx_001",
			EventType: EventTxStarted,
			Command:   "init",
		}
		entry2 := JournalEntry{
			Timestamp: time.Now().UTC(),
			TxID:      "tx_001",
			EventType: EventStepApplied,
			StepIndex: 1,
			Detail:    "wrote AGENTS.md",
		}
		entry3 := JournalEntry{
			Timestamp: time.Now().UTC(),
			TxID:      "tx_001",
			EventType: EventTxCommitted,
		}

		for _, e := range []JournalEntry{entry1, entry2, entry3} {
			if err := j.Append(e); err != nil {
				t.Fatalf("append: %v", err)
			}
		}

		entries, err := j.ReadAll()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if len(entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(entries))
		}

		if entries[0].EventType != EventTxStarted {
			t.Errorf("entry[0] type: got %s, want tx_started", entries[0].EventType)
		}
		if entries[1].StepIndex != 1 {
			t.Errorf("entry[1] step_index: got %d, want 1", entries[1].StepIndex)
		}
		if entries[2].EventType != EventTxCommitted {
			t.Errorf("entry[2] type: got %s, want tx_committed", entries[2].EventType)
		}
	})

	t.Run("entries for tx", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ops-journal.jsonl")
		j := NewJournal(path)

		entries := []JournalEntry{
			{TxID: "tx_a", EventType: EventTxStarted, Timestamp: time.Now().UTC()},
			{TxID: "tx_b", EventType: EventTxStarted, Timestamp: time.Now().UTC()},
			{TxID: "tx_a", EventType: EventStepApplied, StepIndex: 1, Timestamp: time.Now().UTC()},
			{TxID: "tx_a", EventType: EventTxCommitted, Timestamp: time.Now().UTC()},
			{TxID: "tx_b", EventType: EventTxFailed, Timestamp: time.Now().UTC()},
		}

		for _, e := range entries {
			if err := j.Append(e); err != nil {
				t.Fatal(err)
			}
		}

		txA, err := j.EntriesForTx("tx_a")
		if err != nil {
			t.Fatal(err)
		}
		if len(txA) != 3 {
			t.Errorf("expected 3 entries for tx_a, got %d", len(txA))
		}

		txB, err := j.EntriesForTx("tx_b")
		if err != nil {
			t.Fatal(err)
		}
		if len(txB) != 2 {
			t.Errorf("expected 2 entries for tx_b, got %d", len(txB))
		}
	})

	t.Run("read empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ops-journal.jsonl")
		j := NewJournal(path)

		entries, err := j.ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if entries != nil {
			t.Errorf("expected nil entries for nonexistent file, got %d", len(entries))
		}
	})

	t.Run("event types", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ops-journal.jsonl")
		j := NewJournal(path)

		eventTypes := []EventType{
			EventTxStarted, EventStepApplied, EventStepRolledBack,
			EventTxCommitted, EventTxFailed, EventTxRolledBack,
		}

		for i, et := range eventTypes {
			if err := j.Append(JournalEntry{
				TxID:      "tx_types",
				EventType: et,
				StepIndex: i,
				Timestamp: time.Now().UTC(),
			}); err != nil {
				t.Fatal(err)
			}
		}

		entries, err := j.ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != len(eventTypes) {
			t.Fatalf("expected %d entries, got %d", len(eventTypes), len(entries))
		}
		for i, e := range entries {
			if e.EventType != eventTypes[i] {
				t.Errorf("entry[%d]: got %s, want %s", i, e.EventType, eventTypes[i])
			}
		}
	})

	t.Run("backup path and plan id", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ops-journal.jsonl")
		j := NewJournal(path)

		entry := JournalEntry{
			TxID:       "tx_backup",
			PlanID:     "plan_20260223_abc",
			EventType:  EventStepApplied,
			StepIndex:  1,
			BackupPath: ".athena/backups/AGENTS.md.20260223.bak",
			Timestamp:  time.Now().UTC(),
		}

		if err := j.Append(entry); err != nil {
			t.Fatal(err)
		}

		entries, err := j.ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if entries[0].PlanID != "plan_20260223_abc" {
			t.Errorf("plan_id: got %s", entries[0].PlanID)
		}
		if entries[0].BackupPath != ".athena/backups/AGENTS.md.20260223.bak" {
			t.Errorf("backup_path: got %s", entries[0].BackupPath)
		}
	})

	t.Run("append only", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ops-journal.jsonl")
		j := NewJournal(path)

		j.Append(JournalEntry{TxID: "tx_1", EventType: EventTxStarted, Timestamp: time.Now().UTC()})

		info1, _ := os.Stat(path)
		size1 := info1.Size()

		j.Append(JournalEntry{TxID: "tx_2", EventType: EventTxStarted, Timestamp: time.Now().UTC()})

		info2, _ := os.Stat(path)
		size2 := info2.Size()

		if size2 <= size1 {
			t.Error("journal should be append-only (size should grow)")
		}
	})
}
