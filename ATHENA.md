# Athena CLI Specification

Repo-local AI working memory scaffolding, lifecycle management, and governance
checks. A single Go binary that manages the `.ai/` directory tree — structured
notes with YAML frontmatter, machine-readable indexes, context retrieval for
agent onboarding, and policy gates for CI integration.

## Agent Constraints

- Go 1.23+ standard library except for dependencies listed in this spec.
- Do not modify `go.mod` or `go.sum` without explicit permission.
- 100% test coverage for YAML frontmatter parsing and schema migration logic.
- Do not create packages outside `cmd/athena` and `internal/`.
- Do not change path contracts (`athena.toml`, `.athena/checksums.json`,
  `.athena/backups/`, `.athena/telemetry.jsonl`, `.athena/reports/`,
  `.athena/locks/`) without migration tests.
- External integrations (`repomix`, `gitleaks`, `actionlint`, `pre-commit`) are
  optional and must degrade gracefully when missing.

## CLI Surface

```text
athena [--verbose|--debug|--quiet] [--format text|json] [--policy strict|standard|lenient] [--lock-timeout DURATION] [--actor CLIENT] <command> [flags]

# Core — working memory lifecycle
athena init [--force] [--dry-run] [--preset minimal|standard|full] [--with-pre-commit]
athena upgrade [--dry-run]
athena check [--fix] [--strict-schema]
athena index
athena gc [--days N] [--dry-run]
athena doctor
athena capabilities
athena note new   --type TYPE [--slug SLUG] [--title TITLE] [--component COMPONENT]
athena note close --status STATUS PATH
athena note promote --target TARGET PATH
athena note list  [--status STATUS] [--type TYPE]
athena context query [--component COMPONENT] [--type TYPE] [--status STATUS] [--format json|text]
athena context timeline --component COMPONENT [--format json|text]
athena context search "QUERY" [--limit N]

# Governance — policy and commit hygiene
athena policy gate [--pr REF]
athena commit lint [--from REF] [--to REF]
athena security scan [--secrets] [--workflows] [--report-format json|sarif]

# Orchestration — release and optimization
athena changelog [--since TAG] [--next VERSION] [--dry-run]
athena release propose [--since TAG] [--next VERSION]
athena release approve --proposal-id ID
athena hooks install [--pre-commit]

# Diagnostics — telemetry-derived insights
athena report
athena optimize recommend [--window 30d]

# MCP — Model Context Protocol server
athena mcp

# Utility
athena version
athena completion {bash|zsh|fish|powershell}
```

### Global Flags

| Flag             | Description                                                  |
| ---------------- | ------------------------------------------------------------ |
| `--verbose`      | User-facing progress detail                                  |
| `--debug`        | Structured internals (template/hash/file/lock/external-tool) |
| `--quiet`        | Errors and final summary only                                |
| `--format`       | `text` (default) or `json` for machine-readable output       |
| `--policy`       | `strict`, `standard` (default), or `lenient`                 |
| `--lock-timeout` | Maximum wait for mutation lock acquisition                   |
| `--actor`        | Client identity for telemetry (e.g. `cursor`, `claude-code`) |

### Core Commands

These commands form the primary workflow: scaffold the `.ai/` structure, manage
notes through their lifecycle, query context for agent onboarding, and validate
the working memory.

| Command            | Description                                                    |
| ------------------ | -------------------------------------------------------------- |
| `init`             | Scaffold framework files into current repo                     |
| `upgrade`          | Safe upgrade: skip user-modified files, update unmodified ones |
| `check`            | Validate frontmatter; `--fix` applies schema migrations        |
| `index`            | Rebuild `.ai/index.yaml`                                       |
| `gc`               | Mark notes as `stale` after inactivity threshold (default 45d) |
| `doctor`           | Diagnose config drift, toolchain readiness, path health        |
| `capabilities`     | Machine-readable command/feature/contract inventory            |
| `note new`         | Create note from template with frontmatter                     |
| `note close`       | Transition note to a terminal status                           |
| `note promote`     | Mark note as promoted with target doc path                     |
| `note list`        | List notes with optional status/type filters                   |
| `context query`    | Filtered note retrieval by component, type, status             |
| `context timeline` | Chronological view of notes for a component                    |
| `context search`   | BM25 search with Porter stemming, fuzzy matching, and snippets |

### Governance Commands

Policy enforcement and commit hygiene for CI integration.

| Command         | Description                                           |
| --------------- | ----------------------------------------------------- |
| `policy gate`   | Evaluate PR/revision against configured policy gates  |
| `commit lint`   | Validate commit history against Conventional Commits  |
| `security scan` | Run `gitleaks`/`actionlint`, emit local reports       |

### Orchestration Commands

Compose core and governance commands into release workflows.

| Command           | Description                                     |
| ----------------- | ----------------------------------------------- |
| `changelog`       | Update `CHANGELOG.md` from conventional commits |
| `release propose` | Generate release proposal with approval gates   |
| `release approve` | Execute a release proposal after gate check     |
| `hooks install`   | Install/update local pre-commit hooks           |

### Diagnostic Commands

Telemetry-derived insights. Require `.athena/telemetry.jsonl` data to be useful.

| Command             | Description                                        |
| ------------------- | -------------------------------------------------- |
| `report`            | Compute working memory effectiveness metrics       |
| `optimize recommend`| Propose config tuning from telemetry (no auto-apply)|

### MCP Server

`athena mcp` starts a Model Context Protocol server over stdio, exposing
Athena commands as MCP tools and resources. Eliminates shell-spawning
overhead for IDE agents (Claude Code, Cursor, Windsurf).

**Resources** (read-only, URI-addressable):
`athena://capabilities`, `athena://config`, `athena://notes`,
`athena://index`, `athena://report`

**Tools** (callable actions):
`note_new`, `note_close`, `note_promote`, `note_read`, `note_list`,
`check`, `check_fix`, `index_rebuild`, `gc_scan`, `doctor`, `report`,
`context_search`

Transport: stdio only. Semantics are identical to CLI invocation.

## Manifest Schema

Each repository may contain an `athena.toml` (or fall back to built-in defaults).
Feature flags in `[features]` control which files `init`/`upgrade` manage.
Config sections (`[gc]`, `[policy]`, etc.) control runtime behavior independently.

```toml
version = 2

[features]
ai-memory         = true
contributing      = true
editorconfig      = true
agents-md         = true
agent-tooling     = true
agent-metrics     = false
claude-shim       = true
cursor-shim       = true
copilot-shim      = true
repomix-config    = true
ci-workflow       = "github"
security-baseline = true
pre-commit-hooks  = true
changelog         = true

[templates]
# Override built-in templates with repo-local paths
# note = ".athena/templates/note.md"

[scopes]
app   = "Application code"
api   = "API layer"
infra = "Infrastructure configuration"
docs  = "Documentation"
ci    = "CI/CD pipelines and workflows"
meta  = "Repository-level config"

[gc]
days = 45

[tools]
required    = ["git", "rg", "jq", "yq"]
recommended = ["repomix", "gitleaks", "actionlint", "pre-commit", "difft", "fzf"]

[telemetry]
enabled = true
path    = ".athena/telemetry.jsonl"

[policy]
default = "standard"

[policy_gates]
enabled = true
report_path = ".athena/reports/policy-gate.json"
required_checks = ["check", "security_scan", "commit_lint"]

[lock]
ttl = "15m"
allow_force_reap = false

[security]
enable_secrets_scan  = true
enable_workflow_lint = true
report_dir           = ".athena/reports"

[changelog]
enabled             = true
path                = "CHANGELOG.md"
unreleased_heading  = "## Unreleased"

[conventional_commits]
enforce       = true
require_scope = false
types         = ["feat", "fix", "docs", "style", "refactor", "perf", "test", "build", "ci", "chore", "revert"]

[hooks]
pre_commit = true

[optimize]
enabled = true
window_days = 30
min_samples = 50
proposal_path = ".athena/optimization/proposals"
auto_apply = false
```

### Feature Flags

Each feature maps to one or more files. `init` only writes files for enabled
features. `upgrade` only touches files belonging to enabled features.

| Feature             | Files                                          |
| ------------------- | ---------------------------------------------- |
| `ai-memory`         | `.ai/**`                                       |
| `contributing`      | `CONTRIBUTING.md`                              |
| `editorconfig`      | `.editorconfig`                                |
| `agents-md`         | `AGENTS.md`                                    |
| `agent-tooling`     | `AGENT_TOOLING.md`                             |
| `agent-metrics`     | `AGENT_METRICS.md`                             |
| `claude-shim`       | `CLAUDE.md`                                    |
| `cursor-shim`       | `.cursorrules`                                 |
| `copilot-shim`      | `.github/copilot-instructions.md`              |
| `repomix-config`    | `repomix.config.json`                          |
| `ci-workflow`       | `.github/workflows/athena-framework-check.yml` |
| `security-baseline` | `.gitleaks.toml`, `.github/actionlint.yaml`    |
| `pre-commit-hooks`  | `.pre-commit-config.yaml`                      |
| `changelog`         | `CHANGELOG.md`                                 |

### Presets

| Preset     | Enabled Features                    |
| ---------- | ----------------------------------- |
| `minimal`  | `agents-md`, `ai-memory`, `claude-shim` |
| `standard` | All features except `agent-metrics` |
| `full`     | All features                        |

### Manifest Evolution

- Current schema version is `2`.
- Version `1` manifests are auto-migrated to `2` in memory.
- `upgrade` writes back `version = 2` when any manifest rewrite occurs.

## Note Frontmatter Contract

All notes in `.ai/` must contain YAML frontmatter:

| Key                | Type    | Required | Description                                                              |
| ------------------ | ------- | -------- | ------------------------------------------------------------------------ |
| `id`               | string  | yes      | `<type>-YYYYMMDD-<slug>`                                                |
| `title`            | string  | yes      | Human-readable title                                                     |
| `type`             | string  | yes      | `context`, `investigation`, `troubleshooting`, `wip`, `improvement`, `session`, `memory` |
| `status`           | string  | yes      | `active`, `closed`, `stale`, `superseded`, `promoted`                    |
| `created`          | date    | yes      | ISO 8601 date                                                            |
| `updated`          | date    | yes      | ISO 8601 date                                                            |
| `schema_version`   | integer | no       | Defaults to `1`                                                          |
| `related`          | list    | no       | Paths to related notes                                                   |
| `promotion_target` | string  | no       | Target canonical doc path                                                |
| `supersedes`       | list    | no       | IDs of superseded notes                                                  |
| `tags`             | list    | no       | Freeform tags                                                            |
| `component`        | string  | no       | Component/service (used by `context query`)                              |

### Note Status Lifecycle

Valid status values and their semantics:

| Status       | Meaning                                           | Typical Transition From |
| ------------ | ------------------------------------------------- | ----------------------- |
| `active`     | Note is current and relevant                      | (initial)               |
| `closed`     | Resolved, no further action needed                | `active`                |
| `stale`      | Inactive past GC threshold, awaiting triage       | `active` (via `gc`)     |
| `superseded` | Replaced by a newer note (set `supersedes` field) | `active`                |
| `promoted`   | Content moved to canonical docs                   | `active`                |

`note close --status` accepts any valid status. `gc` sets status to `stale`
for active notes older than `[gc].days`. `note promote` sets status to
`promoted` and records `promotion_target`. A `stale` note can be returned to
`active` via `note close --status active`.

### What `check --fix` Mutates

`check` is read-only by default. With `--fix`:

- Applies schema version migration (bumps `schema_version` to latest)
- Writes backups to `.athena/backups/` before modifying any note

`check --fix` does **not**: add missing required fields, reformat dates,
delete invalid notes, or change note content beyond schema migration.

### What `upgrade` Considers User-Modified

`upgrade` compares each managed file's current SHA-256 hash against the hash
stored in `.athena/checksums.json` at install time. If they differ, the file
is considered user-modified and is **skipped** (not overwritten). Unmodified
files are backed up to `.athena/backups/` then overwritten with the new version.

## Context Query and Timeline

`context query` filters notes by `--component`, `--type`, and `--status`.
Returns matching notes with frontmatter metadata and file paths.

`context timeline --component <x>` returns notes in chronological order
by `updated` date. Useful for understanding decision history for a component.

Both support `--format json`.

## Policy Gating

`policy gate` evaluates a PR or revision against configured gates from
`[policy_gates].required_checks`.

Canonical check IDs and what they execute:

| Check ID        | Runs                        |
| --------------- | --------------------------- |
| `check`         | `athena check`              |
| `security_scan` | `athena security scan`      |
| `commit_lint`   | `athena commit lint`        |
| `changelog`     | `athena changelog --dry-run`|
| `doctor`        | `athena doctor`             |

Output includes machine-readable failure reasons. JSON report written to
`[policy_gates].report_path`.

### `policy gate --format json`

```json
{
  "command": "policy gate",
  "ok": false,
  "gates": [
    { "id": "check", "passed": true },
    { "id": "commit_lint", "passed": false, "error": "ATHENA-POL-003" }
  ],
  "failures": [
    {
      "policy_id": "ATHENA-POL-003",
      "severity": "error",
      "summary": "Conventional commit validation failed",
      "fix_hint": "Run `athena commit lint` to see details."
    }
  ]
}
```

## Release Workflow

`release propose` composes core and governance checks into a gated proposal:

1. `athena release propose --since <last-tag> --next <X.Y.Z>`
   - Internally runs: `commit lint`, `check`, `security scan`, `changelog`
   - Produces a proposal artifact with gate results and a SHA-based gate hash
2. Review proposal artifact.
3. `athena release approve --proposal-id <ID>`
   - Re-runs all gates and compares results against the proposal hash
   - **Rejects** if any gate result has changed since proposal time (drift detection)
   - On success, marks the proposal as approved

### `release propose --format json`

```json
{
  "command": "release propose",
  "ok": true,
  "proposal_id": "relprop_20260223_01",
  "next_version": "1.4.0",
  "gate_hash": "sha256:abc123...",
  "gates": [
    { "id": "commit_lint", "passed": true },
    { "id": "check", "passed": true },
    { "id": "security_scan", "passed": true }
  ]
}
```

## Machine-Readable Output Contracts

When `--format json` is used, all commands emit a stable envelope:

```json
{
  "command": "<command-name>",
  "ok": true,
  "policy": "standard",
  "duration_ms": 31,
  "warnings": [],
  "errors": []
}
```

Errors include `error_code`, `message`, `actionable_fix`, and optional
`policy_id`.

### Key Command Outputs

**`capabilities --format json`** — agents use this to adapt without hardcoding
command availability:

```json
{
  "command": "capabilities",
  "ok": true,
  "commands": ["init", "upgrade", "check", "index", "gc", "doctor", "..."],
  "schema_versions": { "manifest": 2, "frontmatter": 1, "telemetry": 1 },
  "output_formats": ["text", "json"],
  "policy_levels": ["strict", "standard", "lenient"]
}
```

**`report --format json`** — note-derived metrics are always available;
telemetry-derived metrics require `.athena/telemetry.jsonl`:

```json
{
  "command": "report",
  "ok": true,
  "telemetry_coverage": 0.76,
  "metrics": {
    "staleness_ratio": 0.12,
    "promotion_rate": 0.67,
    "orphan_rate": 0.25,
    "autonomous_execution": 0.44,
    "context_efficiency": 0.8,
    "security_hygiene": 1.0
  }
}
```

`telemetry_coverage` indicates confidence for telemetry-derived metrics
(`autonomous_execution`, `context_efficiency`, `security_hygiene`).
Note-derived metrics (`staleness_ratio`, `promotion_rate`, `orphan_rate`)
are always computed from `.ai/**`.

### Error Taxonomy

| Namespace        | Scope                     |
| ---------------- | ------------------------- |
| `ATHENA-CONF-*`  | Configuration and manifest|
| `ATHENA-POL-*`   | Policy gate failures      |
| `ATHENA-VAL-*`   | Validation/schema errors  |
| `ATHENA-EXEC-*`  | Execution/plan failures   |
| `ATHENA-TOOL-*`  | External tool integration |

Every non-zero JSON response includes at least one `error_code` with an
`actionable_fix` hint.

## Exit Codes

| Code | Meaning                                          |
| ---- | ------------------------------------------------ |
| 0    | Success                                          |
| 1    | Validation or policy failure                     |
| 2    | Configuration/runtime contract error             |
| 3    | Lock acquisition timeout or concurrency conflict |
| 4    | Plan/apply contract failure                      |

## Implementation

- **Language**: Go 1.24+
- **CLI**: `cobra` + `viper`
- **Template embedding**: `go:embed`
- **YAML**: `gopkg.in/yaml.v3`
- **TOML**: `github.com/BurntSushi/toml`
- **MCP**: `github.com/modelcontextprotocol/go-sdk`
- **TTY**: `golang.org/x/term`
- **External CLIs** (optional): `repomix`, `gitleaks`, `actionlint`, `pre-commit`

### Project Structure

```text
cmd/athena/main.go
internal/
  cli/           # cobra commands and root flags
  config/        # athena.toml parsing and defaults
  lock/          # lock acquisition/release
  scaffold/      # init + upgrade logic
  notes/         # note lifecycle
  validate/      # frontmatter validation + migration
  index/         # .ai/index.yaml generation
  gc/            # staleness marking
  report/        # effectiveness metrics
  capabilities/  # capability negotiation
  policy/        # policy gate evaluation
  context/       # note query/timeline
  security/      # gitleaks/actionlint wrappers
  doctor/        # repository diagnostics
  release/       # release proposal and approval
  optimize/      # telemetry-driven recommendations
  hooks/         # pre-commit install
  commitlint/    # conventional commit linting
  changelog/     # changelog generation
  telemetry/     # telemetry append/read
  mcp/           # MCP server (tools + resources over stdio)
  search/        # BM25 search (Porter stemmer, fuzzy matching, snippets)
  templates/     # go:embed template tree
```

## Telemetry

State-mutating and operational commands append records to `.athena/telemetry.jsonl`.
Strictly local — no remote collection.

```json
{
  "timestamp": "2026-02-23T21:14:21Z",
  "command": "note promote",
  "actor": "claude-code",
  "run_id": "run_20260223_4f2a",
  "execution_time_ms": 42,
  "is_tty": false,
  "exit_code": 0,
  "error_code": null,
  "policy_level": "standard"
}
```

`actor` identifies the client (e.g. `cursor`, `claude-code`, `aider`).
`is_tty: false` indicates autonomous agent execution.

## Source Control Guidance

`.ai/index.yaml` is deterministic (sorted by path) but will conflict when
multiple branches add different notes. Recommended approach:

- Add `.ai/index.yaml` to `.gitignore`
- Rebuild locally via `athena index` (or a post-checkout git hook)
- The index is cheap to regenerate — it reads frontmatter from `.ai/**/*.md`

`.athena/telemetry.jsonl` is append-only and merges cleanly with concatenation.
If conflicts occur, concatenate both sides and deduplicate by timestamp +
command.

## Scope Boundaries

Athena owns:
- `.ai/` directory structure and working memory lifecycle
- Note scaffolding, validation, indexing, and staleness marking
- Context retrieval for agent onboarding
- Policy gates and governance checks
- Local diagnostics and effectiveness reporting
- Conventional commit linting and changelog generation
- Release orchestration composing the above

Athena does **not** own:
- Repository-specific business validation
- Git history mutation (rebases, force pushes)
- Remote telemetry or cloud analytics
- Autonomous code self-modification
