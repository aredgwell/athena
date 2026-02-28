// Package telemetry implements telemetry append/read helpers.
package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Record is a single telemetry event in JSONLines format.
type Record struct {
	Timestamp       time.Time         `json:"timestamp"`
	Command         string            `json:"command"`
	RunID           string            `json:"run_id,omitempty"`
	TaskID          string            `json:"task_id,omitempty"`
	AgentName       string            `json:"agent_name,omitempty"`
	Model           string            `json:"model,omitempty"`
	ExecutionTimeMS int64             `json:"execution_time_ms"`
	PromptTokens    int               `json:"prompt_tokens,omitempty"`
	CompletionTokens int              `json:"completion_tokens,omitempty"`
	TotalTokens     int               `json:"total_tokens,omitempty"`
	CostUSD         float64           `json:"cost_usd,omitempty"`
	IsTTY           bool              `json:"is_tty"`
	ExitCode        int               `json:"exit_code"`
	ErrorCode       *string           `json:"error_code"`
	PolicyLevel     string            `json:"policy_level,omitempty"`
	IsDryRun        bool              `json:"is_dry_run"`
	Actor           string            `json:"actor,omitempty"`
	Context         map[string]string `json:"context,omitempty"`
}

// Store provides telemetry read/write operations.
type Store struct {
	path string
}

// NewStore creates a telemetry store at the given path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Append writes a telemetry record as a JSON line.
func (s *Store) Append(rec Record) error {
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening telemetry: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// ReadAll reads all telemetry records from the file.
func (s *Store) ReadAll() ([]Record, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var records []Record
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var rec Record
		if err := dec.Decode(&rec); err != nil {
			return records, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// Coverage computes telemetry_coverage: ratio of commands with telemetry data.
func Coverage(records []Record, totalCommands int) float64 {
	if totalCommands == 0 {
		return 0
	}
	return float64(len(records)) / float64(totalCommands)
}

// FilterByRunID returns records matching a specific run_id.
func FilterByRunID(records []Record, runID string) []Record {
	var filtered []Record
	for _, r := range records {
		if r.RunID == runID {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
