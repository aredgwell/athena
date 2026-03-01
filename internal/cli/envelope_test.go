package cli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	atherr "github.com/aredgwell/athena/internal/errors"
)

func TestJSONEnvelope(t *testing.T) {
	t.Run("success envelope", func(t *testing.T) {
		env := NewEnvelope("check", 31*time.Millisecond)
		env.WithPolicy("standard")

		if !env.OK {
			t.Error("expected ok=true")
		}
		if env.Command != "check" {
			t.Errorf("command: got %s, want check", env.Command)
		}
		if env.DurationMS != 31 {
			t.Errorf("duration_ms: got %d, want 31", env.DurationMS)
		}

		buf := new(bytes.Buffer)
		if err := env.WriteJSON(buf); err != nil {
			t.Fatal(err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}

		if decoded["ok"] != true {
			t.Errorf("JSON ok: got %v", decoded["ok"])
		}
		if decoded["command"] != "check" {
			t.Errorf("JSON command: got %v", decoded["command"])
		}
		if decoded["policy"] != "standard" {
			t.Errorf("JSON policy: got %v", decoded["policy"])
		}

		// Warnings and errors should be empty arrays, not null
		warnings, ok := decoded["warnings"].([]any)
		if !ok {
			t.Fatal("warnings should be an array")
		}
		if len(warnings) != 0 {
			t.Errorf("expected empty warnings, got %d", len(warnings))
		}

		errors, ok := decoded["errors"].([]any)
		if !ok {
			t.Fatal("errors should be an array")
		}
		if len(errors) != 0 {
			t.Errorf("expected empty errors, got %d", len(errors))
		}
	})

	t.Run("error envelope", func(t *testing.T) {
		env := NewEnvelope("check", 15*time.Millisecond)
		env.AddError(atherr.New(
			atherr.ValFrontmatterInvalid,
			"Frontmatter schema mismatch",
			"Run `athena check --fix`.",
		).WithPolicy(atherr.PolSchemaMismatch))

		if env.OK {
			t.Error("expected ok=false after AddError")
		}

		buf := new(bytes.Buffer)
		if err := env.WriteJSON(buf); err != nil {
			t.Fatal(err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}

		if decoded["ok"] != false {
			t.Error("JSON ok should be false")
		}

		errs, ok := decoded["errors"].([]any)
		if !ok || len(errs) != 1 {
			t.Fatalf("expected 1 error, got %v", decoded["errors"])
		}

		errObj, ok := errs[0].(map[string]any)
		if !ok {
			t.Fatal("error should be an object")
		}
		if errObj["error_code"] != atherr.ValFrontmatterInvalid {
			t.Errorf("error_code: got %v", errObj["error_code"])
		}
		if errObj["actionable_fix"] != "Run `athena check --fix`." {
			t.Errorf("actionable_fix: got %v", errObj["actionable_fix"])
		}
		if errObj["policy_id"] != atherr.PolSchemaMismatch {
			t.Errorf("policy_id: got %v", errObj["policy_id"])
		}
	})

	t.Run("warnings", func(t *testing.T) {
		env := NewEnvelope("doctor", 5*time.Millisecond)
		env.AddWarning("repomix not found")
		env.AddWarning("gitleaks not found")

		if !env.OK {
			t.Error("warnings should not set ok=false")
		}
		if len(env.Warnings) != 2 {
			t.Errorf("expected 2 warnings, got %d", len(env.Warnings))
		}
	})

	t.Run("with data", func(t *testing.T) {
		env := NewEnvelope("report", 100*time.Millisecond)
		env.WithData(map[string]float64{"staleness_ratio": 0.12})

		buf := new(bytes.Buffer)
		if err := env.WriteJSON(buf); err != nil {
			t.Fatal(err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatal(err)
		}

		data, ok := decoded["data"].(map[string]any)
		if !ok {
			t.Fatal("expected data field")
		}
		if data["staleness_ratio"] != 0.12 {
			t.Errorf("staleness_ratio: got %v", data["staleness_ratio"])
		}
	})
}
