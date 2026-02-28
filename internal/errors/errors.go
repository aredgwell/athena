// Package errors implements the Athena error taxonomy and actionable fix mapping.
package errors

import "fmt"

// Error code namespace prefixes.
const (
	NSConfig    = "ATHENA-CONF"
	NSPolicy    = "ATHENA-POL"
	NSValidate  = "ATHENA-VAL"
	NSExecution = "ATHENA-EXEC"
	NSTool      = "ATHENA-TOOL"
)

// Well-known error codes.
const (
	// Config errors
	ConfManifestNotFound = "ATHENA-CONF-001"
	ConfManifestParse    = "ATHENA-CONF-002"
	ConfVersionUnknown   = "ATHENA-CONF-003"
	ConfMissingRequired  = "ATHENA-CONF-004"

	// Policy errors
	PolGateFailed      = "ATHENA-POL-001"
	PolStrictViolation = "ATHENA-POL-002"
	PolSchemaMismatch  = "ATHENA-POL-003"

	// Validation errors
	ValFrontmatterMissing = "ATHENA-VAL-001"
	ValFrontmatterInvalid = "ATHENA-VAL-002"
	ValSchemaMigration    = "ATHENA-VAL-003"

	// Execution errors
	ExecPlanNotFound      = "ATHENA-EXEC-001"
	ExecPlanStale         = "ATHENA-EXEC-002"
	ExecRollbackFailed    = "ATHENA-EXEC-003"
	ExecIdempotencyFailed = "ATHENA-EXEC-004"
	ExecPlanRequired      = "ATHENA-EXEC-005"

	// Tool errors
	ToolMissing     = "ATHENA-TOOL-001"
	ToolExecFailed  = "ATHENA-TOOL-002"
	ToolUnavailable = "ATHENA-TOOL-003"
)

// AthenaError is a structured, machine-readable error with actionable remediation.
type AthenaError struct {
	Code          string `json:"error_code"`
	Message       string `json:"message"`
	ActionableFix string `json:"actionable_fix"`
	PolicyID      string `json:"policy_id,omitempty"`
}

// Error implements the error interface.
func (e *AthenaError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// New creates a new AthenaError.
func New(code, message, fix string) *AthenaError {
	return &AthenaError{
		Code:          code,
		Message:       message,
		ActionableFix: fix,
	}
}

// WithPolicy returns a copy of the error with PolicyID set.
func (e *AthenaError) WithPolicy(policyID string) *AthenaError {
	return &AthenaError{
		Code:          e.Code,
		Message:       e.Message,
		ActionableFix: e.ActionableFix,
		PolicyID:      policyID,
	}
}
