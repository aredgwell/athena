package capabilities

import (
	"encoding/json"
	"testing"
)

func TestCapabilitiesCommand(t *testing.T) {
	payload := Get()

	if !payload.CommandsComplete {
		t.Error("expected commands_complete=true")
	}
	if len(payload.Commands) != 31 {
		t.Errorf("expected 31 commands, got %d", len(payload.Commands))
	}

	// Verify key commands are present
	required := []string{"init", "upgrade", "check", "plan", "apply", "rollback", "capabilities", "version"}
	cmdSet := make(map[string]bool)
	for _, c := range payload.Commands {
		cmdSet[c] = true
	}
	for _, r := range required {
		if !cmdSet[r] {
			t.Errorf("missing required command: %s", r)
		}
	}
}

func TestCapabilitiesSchemaVersions(t *testing.T) {
	payload := Get()

	if payload.SchemaVersions.Manifest != 2 {
		t.Errorf("manifest version: got %d, want 2", payload.SchemaVersions.Manifest)
	}
	if payload.SchemaVersions.Frontmatter != 1 {
		t.Errorf("frontmatter version: got %d, want 1", payload.SchemaVersions.Frontmatter)
	}
	if payload.SchemaVersions.Telemetry != 1 {
		t.Errorf("telemetry version: got %d, want 1", payload.SchemaVersions.Telemetry)
	}
}

func TestCapabilitiesExecutionModes(t *testing.T) {
	payload := Get()

	if len(payload.ExecutionModes) != 2 {
		t.Fatalf("expected 2 execution modes, got %d", len(payload.ExecutionModes))
	}
	if payload.ExecutionModes[0] != "direct" || payload.ExecutionModes[1] != "plan-first" {
		t.Errorf("unexpected execution modes: %v", payload.ExecutionModes)
	}
}

func TestCapabilitiesOutputFormats(t *testing.T) {
	payload := Get()

	if len(payload.OutputFormats) != 2 {
		t.Fatalf("expected 2 output formats, got %d", len(payload.OutputFormats))
	}
	if payload.OutputFormats[0] != "text" || payload.OutputFormats[1] != "json" {
		t.Errorf("unexpected output formats: %v", payload.OutputFormats)
	}
}

func TestCapabilitiesJSONSnapshot(t *testing.T) {
	payload := Get()

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// Verify round-trip stability
	var decoded Payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.CommandsComplete != payload.CommandsComplete {
		t.Error("round-trip: commands_complete mismatch")
	}
	if len(decoded.Commands) != len(payload.Commands) {
		t.Errorf("round-trip: commands count %d != %d", len(decoded.Commands), len(payload.Commands))
	}
	if decoded.SchemaVersions != payload.SchemaVersions {
		t.Error("round-trip: schema_versions mismatch")
	}
}

func TestCapabilitiesReportFormats(t *testing.T) {
	payload := Get()

	if len(payload.ReportFormats) != 2 {
		t.Fatalf("expected 2 report formats, got %d", len(payload.ReportFormats))
	}
	formats := make(map[string]bool)
	for _, f := range payload.ReportFormats {
		formats[f] = true
	}
	if !formats["json"] || !formats["sarif"] {
		t.Errorf("expected json and sarif report formats, got %v", payload.ReportFormats)
	}
}
