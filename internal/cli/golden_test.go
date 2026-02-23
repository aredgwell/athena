package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/amr-athena/athena/internal/capabilities"
)

func TestCapabilitiesGolden(t *testing.T) {
	payload := capabilities.Get()

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	output := string(data)

	// Verify structure stability
	if !strings.Contains(output, "\"commands\"") {
		t.Error("missing commands field")
	}
	if !strings.Contains(output, "\"schema_versions\"") {
		t.Error("missing schema_versions field")
	}
	if !strings.Contains(output, "\"execution_modes\"") {
		t.Error("missing execution_modes field")
	}
	if !strings.Contains(output, "\"output_formats\"") {
		t.Error("missing output_formats field")
	}
	if !strings.Contains(output, "\"report_formats\"") {
		t.Error("missing report_formats field")
	}

	// Verify key schema versions
	if payload.SchemaVersions.Manifest != 2 {
		t.Errorf("manifest schema: got %d, want 2", payload.SchemaVersions.Manifest)
	}
	if payload.SchemaVersions.Frontmatter != 1 {
		t.Errorf("frontmatter schema: got %d, want 1", payload.SchemaVersions.Frontmatter)
	}
	if payload.SchemaVersions.Telemetry != 1 {
		t.Errorf("telemetry schema: got %d, want 1", payload.SchemaVersions.Telemetry)
	}

	// Verify command count matches expected
	if len(payload.Commands) < 30 {
		t.Errorf("commands: got %d, want >= 30", len(payload.Commands))
	}

	// Verify key commands present
	required := []string{"init", "upgrade", "check", "gc", "doctor", "capabilities",
		"policy gate", "plan", "apply", "rollback", "security scan",
		"context pack", "note new", "commit lint", "changelog",
		"release propose", "release approve", "hooks install",
		"optimize recommend", "report", "version"}
	for _, cmd := range required {
		found := false
		for _, c := range payload.Commands {
			if c == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required command: %s", cmd)
		}
	}
}

func TestEnvelopeJSONContract(t *testing.T) {
	t.Run("success envelope contract", func(t *testing.T) {
		env := NewEnvelope("check", 0)
		env.WithData(map[string]int{"files_scanned": 12, "valid": 12})

		var raw map[string]interface{}
		data, _ := json.Marshal(env)
		json.Unmarshal(data, &raw)

		// Required fields
		for _, field := range []string{"command", "ok", "duration_ms", "warnings", "errors"} {
			if _, ok := raw[field]; !ok {
				t.Errorf("missing required field: %s", field)
			}
		}

		if raw["ok"] != true {
			t.Error("ok should be true for success envelope")
		}
	})

	t.Run("error envelope contract", func(t *testing.T) {
		env := NewEnvelope("check", 0)
		env.AddError(nil) // Would normally add real error

		var raw map[string]interface{}
		data, _ := json.Marshal(env)
		json.Unmarshal(data, &raw)

		if raw["ok"] != false {
			t.Error("ok should be false for error envelope")
		}
	})
}
