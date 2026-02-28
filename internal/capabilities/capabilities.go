// Package capabilities implements machine-readable capability negotiation.
package capabilities

// SchemaVersions reports the current schema versions for all contracts.
type SchemaVersions struct {
	Manifest    int `json:"manifest"`
	Frontmatter int `json:"frontmatter"`
	Telemetry   int `json:"telemetry"`
}

// Payload is the capabilities JSON output.
type Payload struct {
	Commands         []string       `json:"commands"`
	CommandsComplete bool           `json:"commands_complete"`
	ExecutionModes   []string       `json:"execution_modes"`
	OutputFormats    []string       `json:"output_formats"`
	ReportFormats    []string       `json:"report_formats"`
	SchemaVersions   SchemaVersions `json:"schema_versions"`
}

// AllCommands is the complete list of Athena CLI commands.
var AllCommands = []string{
	"init",
	"upgrade",
	"check",
	"index",
	"gc",
	"tools",
	"doctor",
	"capabilities",
	"policy gate",
	"plan",
	"apply",
	"rollback",
	"security scan",
	"context pack",
	"context mcp",
	"context budget",
	"note new",
	"note close",
	"note promote",
	"note list",
	"review promotions",
	"review weekly",
	"report",
	"commit lint",
	"changelog",
	"release propose",
	"release approve",
	"hooks install",
	"optimize recommend",
	"mcp",
	"version",
	"completion",
}

// Get returns the current capabilities payload.
func Get() Payload {
	return Payload{
		Commands:         AllCommands,
		CommandsComplete: true,
		ExecutionModes:   []string{"direct", "plan-first"},
		OutputFormats:    []string{"text", "json"},
		ReportFormats:    []string{"json", "sarif"},
		SchemaVersions: SchemaVersions{
			Manifest:    2,
			Frontmatter: 1,
			Telemetry:   1,
		},
	}
}
