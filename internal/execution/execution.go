// Package execution implements plan/apply orchestration, idempotency, and rollback.
package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// EventType represents a journal event type.
type EventType string

const (
	EventTxStarted    EventType = "tx_started"
	EventStepApplied  EventType = "step_applied"
	EventStepRolledBack EventType = "step_rolled_back"
	EventTxCommitted  EventType = "tx_committed"
	EventTxFailed     EventType = "tx_failed"
	EventTxRolledBack EventType = "tx_rolled_back"
)

// JournalEntry is a single entry in the append-only operations journal.
type JournalEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	TxID         string    `json:"tx_id"`
	PlanID       string    `json:"plan_id,omitempty"`
	EventType    EventType `json:"event_type"`
	StepIndex    int       `json:"step_index,omitempty"`
	Command      string    `json:"command,omitempty"`
	BackupPath   string    `json:"backup_path,omitempty"`
	Detail       string    `json:"detail,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// Journal provides append-only journal operations.
type Journal struct {
	path string
}

// NewJournal creates a journal at the given path.
func NewJournal(path string) *Journal {
	return &Journal{path: path}
}

// Append writes a journal entry as a JSON line.
func (j *Journal) Append(entry JournalEntry) error {
	f, err := os.OpenFile(j.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening journal: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// ReadAll reads all journal entries from the file.
func (j *Journal) ReadAll() ([]JournalEntry, error) {
	data, err := os.ReadFile(j.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []JournalEntry
	dec := json.NewDecoder(
		newLineReader(data),
	)
	for dec.More() {
		var entry JournalEntry
		if err := dec.Decode(&entry); err != nil {
			return entries, fmt.Errorf("decoding journal entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// EntriesForTx returns all journal entries for a given transaction ID.
func (j *Journal) EntriesForTx(txID string) ([]JournalEntry, error) {
	all, err := j.ReadAll()
	if err != nil {
		return nil, err
	}
	var filtered []JournalEntry
	for _, e := range all {
		if e.TxID == txID {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

// lineReader wraps raw bytes to be consumed by json.Decoder line-by-line.
type lineReader struct {
	data []byte
	pos  int
}

func newLineReader(data []byte) *lineReader {
	return &lineReader{data: data}
}

func (r *lineReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
