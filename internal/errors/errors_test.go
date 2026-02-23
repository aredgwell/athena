package errors

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAthenaErrorInterface(t *testing.T) {
	err := New(ConfManifestParse, "failed to parse athena.toml", "Check TOML syntax.")
	var e error = err
	if e.Error() != "[ATHENA-CONF-002] failed to parse athena.toml" {
		t.Errorf("unexpected error string: %s", e.Error())
	}
}

func TestAthenaErrorJSON(t *testing.T) {
	err := New(ValFrontmatterInvalid, "missing required field 'title'", "Add 'title' to frontmatter.")

	data, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatal(jsonErr)
	}

	var decoded AthenaError
	if jsonErr := json.Unmarshal(data, &decoded); jsonErr != nil {
		t.Fatal(jsonErr)
	}

	if decoded.Code != ValFrontmatterInvalid {
		t.Errorf("code: got %s, want %s", decoded.Code, ValFrontmatterInvalid)
	}
	if decoded.Message != "missing required field 'title'" {
		t.Errorf("message mismatch: %s", decoded.Message)
	}
	if decoded.ActionableFix != "Add 'title' to frontmatter." {
		t.Errorf("fix mismatch: %s", decoded.ActionableFix)
	}
	if decoded.PolicyID != "" {
		t.Errorf("expected empty policy_id, got %s", decoded.PolicyID)
	}

	// Verify policy_id is omitted when empty
	if strings.Contains(string(data), "policy_id") {
		t.Error("expected policy_id to be omitted from JSON when empty")
	}
}

func TestWithPolicy(t *testing.T) {
	base := New(PolGateFailed, "gate failed", "Fix the issue.")
	withPol := base.WithPolicy(PolSchemaMismatch)

	if withPol.PolicyID != PolSchemaMismatch {
		t.Errorf("policy_id: got %s, want %s", withPol.PolicyID, PolSchemaMismatch)
	}

	// Original should be unmodified
	if base.PolicyID != "" {
		t.Error("WithPolicy should not modify the original error")
	}

	// JSON should include policy_id
	data, err := json.Marshal(withPol)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "policy_id") {
		t.Error("expected policy_id in JSON output")
	}
}

func TestErrorCodeNamespaces(t *testing.T) {
	tests := []struct {
		code   string
		prefix string
	}{
		{ConfManifestNotFound, NSConfig},
		{ConfManifestParse, NSConfig},
		{ConfVersionUnknown, NSConfig},
		{PolGateFailed, NSPolicy},
		{PolStrictViolation, NSPolicy},
		{ValFrontmatterMissing, NSValidate},
		{ValFrontmatterInvalid, NSValidate},
		{ExecPlanNotFound, NSExecution},
		{ExecPlanStale, NSExecution},
		{ExecRollbackFailed, NSExecution},
		{ExecIdempotencyFailed, NSExecution},
		{ExecPlanRequired, NSExecution},
		{ToolMissing, NSTool},
		{ToolExecFailed, NSTool},
		{ToolUnavailable, NSTool},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if !strings.HasPrefix(tt.code, tt.prefix) {
				t.Errorf("code %s does not have prefix %s", tt.code, tt.prefix)
			}
		})
	}
}
