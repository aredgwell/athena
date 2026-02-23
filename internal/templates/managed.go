package templates

import (
	"fmt"

	"github.com/amr-athena/athena/internal/config"
	"github.com/amr-athena/athena/internal/scaffold"
)

// ManagedFiles returns the set of files to scaffold based on config feature flags.
func ManagedFiles(cfg config.Config) []scaffold.ManagedFile {
	var files []scaffold.ManagedFile

	// Core athena config
	files = append(files, scaffold.ManagedFile{
		Path:    "athena.toml",
		Content: renderAthenaToml(cfg),
	})

	// .athena directory
	files = append(files, scaffold.ManagedFile{
		Path:    ".athena/checksums.json",
		Content: []byte(`{"version":1,"installed_version":"","files":{}}`),
	})

	// .ai directory
	if cfg.Features.AIMemory {
		files = append(files, scaffold.ManagedFile{
			Path:    ".ai/.gitkeep",
			Content: []byte{},
			Feature: "ai-memory",
		})
	}

	if cfg.Features.AgentsMD {
		files = append(files, scaffold.ManagedFile{
			Path:    "AGENTS.md",
			Content: []byte("# Agent Protocol\n\nSee athena.toml for configuration.\n"),
			Feature: "agents-md",
		})
	}

	if cfg.Features.ClaudeShim {
		files = append(files, scaffold.ManagedFile{
			Path:    "CLAUDE.md",
			Content: []byte("# Claude Tool Shim\n\nFollow `AGENTS.md` as the canonical execution protocol for this repository.\nIf any instructions conflict, `AGENTS.md` takes precedence.\n"),
			Feature: "claude-shim",
		})
	}

	if cfg.Features.Editorconfig {
		files = append(files, scaffold.ManagedFile{
			Path:    ".editorconfig",
			Content: renderEditorconfig(),
			Feature: "editorconfig",
		})
	}

	return files
}

func renderAthenaToml(cfg config.Config) []byte {
	return []byte(fmt.Sprintf(`# Athena CLI Configuration
version = %d

[features]
ai-memory = %t
agents-md = %t
claude-shim = %t
editorconfig = %t

[gc]
days = %d

[policy]
default = %q

[telemetry]
enabled = %t
path = %q
`, cfg.Version,
		cfg.Features.AIMemory,
		cfg.Features.AgentsMD,
		cfg.Features.ClaudeShim,
		cfg.Features.Editorconfig,
		cfg.GC.Days,
		cfg.Policy.Default,
		cfg.Telemetry.Enabled,
		cfg.Telemetry.Path,
	))
}

func renderEditorconfig() []byte {
	return []byte(`root = true

[*]
end_of_line = lf
insert_final_newline = true
charset = utf-8
indent_style = space
indent_size = 2

[*.go]
indent_style = tab

[Makefile]
indent_style = tab
`)
}
