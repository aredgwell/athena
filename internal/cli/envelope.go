package cli

import (
	"encoding/json"
	"io"
	"time"

	atherr "github.com/aredgwell/athena/internal/errors"
)

// Envelope is the common JSON output wrapper for all Athena commands.
type Envelope struct {
	Command    string                `json:"command"`
	OK         bool                  `json:"ok"`
	Policy     string                `json:"policy,omitempty"`
	DurationMS int64                 `json:"duration_ms"`
	Warnings   []string              `json:"warnings"`
	Errors     []*atherr.AthenaError `json:"errors"`
	Data       any                   `json:"data,omitempty"`
}

// NewEnvelope creates a success envelope for a command.
func NewEnvelope(command string, duration time.Duration) *Envelope {
	return &Envelope{
		Command:    command,
		OK:         true,
		DurationMS: duration.Milliseconds(),
		Warnings:   []string{},
		Errors:     []*atherr.AthenaError{},
	}
}

// AddWarning appends a warning message.
func (e *Envelope) AddWarning(msg string) {
	e.Warnings = append(e.Warnings, msg)
}

// AddError appends a structured error and marks the envelope as failed.
func (e *Envelope) AddError(err *atherr.AthenaError) {
	e.OK = false
	e.Errors = append(e.Errors, err)
}

// WithPolicy sets the policy level on the envelope.
func (e *Envelope) WithPolicy(policy string) *Envelope {
	e.Policy = policy
	return e
}

// WithData sets the command-specific payload.
func (e *Envelope) WithData(data any) *Envelope {
	e.Data = data
	return e
}

// WriteJSON serializes the envelope as JSON to the writer.
func (e *Envelope) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(e)
}
