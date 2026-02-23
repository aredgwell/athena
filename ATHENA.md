# Athena CLI Specification

Product specification for the `athena` CLI tool - a portable, schema-driven
scaffolder and lifecycle manager for AI-native repository workflows.

## Instructions for AI Coding Agents

This section governs autonomous or semi-autonomous implementations of this
specification.

### Agent Constraints

- Strictly adhere to Go 1.23+ standard library usage except for dependencies
  explicitly listed in this spec.
- Do not modify `go.mod` or `go.sum` without explicit permission from the
  repository owner, except during approved bootstrap work (for example, feature
  `F00` in the sequential implementation plan).
- Ensure 100% test coverage for YAML frontmatter parsing and schema migration
  logic.
- Do not create new packages outside `cmd/athena` and the defined `internal/`
  tree without permission.
- Do not change path contracts for `athena.toml`, `.athena/checksums.json`,
  `.athena/backups/`, `.athena/telemetry.jsonl`, `.athena/reports/`, or
  `.athena/locks/` without adding migration tests.
- Use `--debug` logging to expose checksum comparisons, template rendering
  decisions, external tool invocations, lock acquisition, and planned file
  mutations.
- External integrations (`repomix`, `gitleaks`, `actionlint`, `pre-commit`) are
  optional dependencies and must degrade gracefully when missing.
- AI agents should use plan/apply flow for mutating operations unless direct
  mode is explicitly permitted by repository policy.
- AI agents should attach a shared `run_id` to all Athena invocations in a task
  for telemetry correlation.

### Parallel Implementation Phases

Implement in isolated Git worktrees to maximize parallelism and minimize merge
risk:

| Phase              | Worktree             | Scope                                                                                                                                                                                   | Exit Criteria                                                                                                               |
| ------------------ | -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| A                  | `wt-config-scaffold` | `internal/config`, `internal/scaffold`, `internal/lock`                                                                                                                                 | Manifest defaults, init/upgrade planning, conflict resolution, lock behavior tested                                         |
| B                  | `wt-templates`       | `internal/templates`, changelog/conventional templates                                                                                                                                  | Embedded templates render deterministically and pass golden tests                                                           |
| C                  | `wt-validate`        | `internal/validate`, `internal/commitlint`                                                                                                                                              | Frontmatter + schema migration + conventional-commit validation pass with coverage targets                                  |
| D                  | `wt-index-report`    | `internal/index`, `internal/report`, `internal/telemetry`                                                                                                                               | Deterministic index/report output and telemetry parsing pass                                                                |
| E                  | `wt-integrations`    | `internal/context`, `internal/security`, `internal/hooks`, `internal/doctor`, `internal/policy`, `internal/capabilities`, `internal/execution`, `internal/release`, `internal/optimize` | Integration wrappers, policy gates, plan/apply flow, release gates, and optimization recommendations pass integration tests |
| Final (sequential) | `wt-cli-main`        | `internal/cli`, `cmd/athena/main.go`                                                                                                                                                    | Wire all commands, global flags, and policy behavior after A-E merge cleanly                                                |

### Executable Acceptance Contract

Every CLI command in the reference table below has an explicit verification
command that agents must run before claiming completion.

## Problem

The Athena Framework is currently distributed as a shell-script-based template
(`install.sh` + 9 Bash scripts + Taskfile glue). This works but has
limitations:

- No upgrade path - re-running `install.sh --force` overwrites user edits
- No feature selection - all-or-nothing installation
- Shell scripts are fragile for YAML frontmatter manipulation
- No cross-platform story (Windows, non-Bash environments)
- No versioning contract between the tool and installed artefacts

## Solution

A single Go binary (`athena`) that replaces `install.sh`, the shell scripts,
and the `.task/ai.yml` Taskfile. It embeds templates at compile time, supports
schema-driven feature selection, safe upgrades, and optional integrations with
local CLI tools for context packing, security checks, and hook automation.

## CLI Surface

```text
athena [--verbose|--debug|--quiet] [--format text|json] [--policy strict|standard|lenient] [--lock-timeout DURATION] <command> [flags]
athena init [--force] [--dry-run] [--preset minimal|standard|full] [--with-pre-commit]
athena upgrade [--dry-run]
athena check [--fix] [--strict-schema] [--secrets] [--workflows]
athena index
athena gc [--days N] [--dry-run]
athena tools [--strict]
athena doctor
athena capabilities
athena policy gate [--pr REF]
athena plan <mutating-command> [args...]
athena apply --plan-id PLAN_ID
athena rollback --tx TX_ID [--to-step N]
athena security scan [--secrets] [--workflows] [--report-format json|sarif]
athena context pack [--profile review|handoff|release] [--changed] [--stdout] [--output PATH] [--dry-run] [-- <repomix_args...>]
athena context mcp [--stdio]
athena context budget [--profile PROFILE] [--max-tokens N]
athena note new   --type TYPE [--slug SLUG] [--title TITLE]
athena note close --status STATUS PATH
athena note promote --target TARGET PATH
athena note list  [--status STATUS] [--type TYPE]
athena review promotions
athena review weekly [--days N]
athena report
athena commit lint [--from REF] [--to REF]
athena changelog [--since TAG] [--next VERSION] [--dry-run]
athena release propose [--since TAG] [--next VERSION]
athena release approve --proposal-id ID
athena hooks install [--pre-commit]
athena optimize recommend [--window 30d]
athena version
athena completion {bash|zsh|fish|powershell}
```

Global behavior:

- `--verbose`: user-facing progress detail.
- `--debug`: structured internals (template/hash/file/lock/external-tool
  traces).
- `--quiet`: errors and final summary only.
- `--format`: machine-readable output (`json`) for automation, default `text`.
- `--format` applies to primary command output only. Security report artifacts
  use command-specific `--report-format`.
- `--policy`:
  - `strict`: fail on warnings and missing recommended checks.
  - `standard`: fail on hard errors only (default).
  - `lenient`: best-effort execution with warning-heavy output.
- `--lock-timeout`: maximum wait for mutation lock acquisition.
- Two-phase execution for mutating operations is supported through `athena plan`
  then `athena apply --plan-id`.

Human-first fallback for `athena note new`:

- If `stdin` is a TTY and `--slug` and/or `--title` are omitted, prompt
  interactively.
- If `stdin` is not a TTY, missing required values remain a configuration error.

### Command Reference

| Command              | Replaces                                               | Description                                                                                     | Verification Command                                               |
| -------------------- | ------------------------------------------------------ | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `init`               | `install.sh`                                           | Scaffold framework files into current repo                                                      | `go test ./internal/scaffold -run TestInitCommand -v`              |
| `upgrade`            | `install.sh --force` (unsafe)                          | Safe upgrade: skip user-modified files, update unmodified ones                                  | `go test ./internal/scaffold -run TestUpgradeCommand -v`           |
| `check`              | `validate-frontmatter.sh` (+ `reindex.sh` via `--fix`) | Validate frontmatter in read-only mode; `--fix` applies safe remediations and migrations        | `go test ./internal/validate -run TestCheckCommand -v`             |
| `index`              | `reindex.sh`                                           | Rebuild `.ai/index.yaml`                                                                        | `go test ./internal/index -run TestIndexCommand -v`                |
| `gc`                 | `gc.sh`                                                | Mark stale notes (default 45 days)                                                              | `go test ./internal/gc -run TestGCCommand -v`                      |
| `tools`              | `check-tools.sh`                                       | Report/enforce local CLI tool availability                                                      | `go test ./internal/tools -run TestToolsCommand -v`                |
| `doctor`             | (new)                                                  | Diagnose config drift, toolchain readiness, lock and path health                                | `go test ./internal/doctor -run TestDoctorCommand -v`              |
| `capabilities`       | (new)                                                  | Expose machine-readable command/features/contracts for safe agent adaptation                    | `go test ./internal/capabilities -run TestCapabilitiesCommand -v`  |
| `policy gate`        | (new)                                                  | Evaluate PR/revision against policy gates and emit machine-readable failures                    | `go test ./internal/policy -run TestPolicyGateCommand -v`          |
| `plan`               | (new)                                                  | Create immutable execution plans for mutating operations                                        | `go test ./internal/execution -run TestPlanCommand -v`             |
| `apply`              | (new)                                                  | Apply an approved plan by `plan_id` with transaction journaling                                 | `go test ./internal/execution -run TestApplyCommand -v`            |
| `rollback`           | (new)                                                  | Roll back a transaction using operation journal + backups                                       | `go test ./internal/execution -run TestRollbackCommand -v`         |
| `security scan`      | (new)                                                  | Run `gitleaks` and/or `actionlint`, emit local reports (`--report-format json or sarif`)        | `go test ./internal/security -run TestSecurityScanCommand -v`      |
| `context pack`       | (new)                                                  | Wrapper around `repomix` profiles, optional changed-files mode                                  | `go test ./internal/context -run TestContextPackCommand -v`        |
| `context mcp`        | (new)                                                  | Start or validate repomix MCP mode for assistant clients                                        | `go test ./internal/context -run TestContextMCPCommand -v`         |
| `context budget`     | (new)                                                  | Estimate context token budget and enforce max token caps                                        | `go test ./internal/context -run TestContextBudgetCommand -v`      |
| `note new`           | `new-note.sh`                                          | Create note from template with frontmatter (interactive fallback for missing title/slug on TTY) | `go test ./internal/notes -run TestNoteNewCommand -v`              |
| `note close`         | `close-note.sh`                                        | Transition note status                                                                          | `go test ./internal/notes -run TestNoteCloseCommand -v`            |
| `note promote`       | `promote-note.sh`                                      | Mark note as promoted with target doc path                                                      | `go test ./internal/notes -run TestNotePromoteCommand -v`          |
| `note list`          | (new)                                                  | List notes with optional status/type filters                                                    | `go test ./internal/notes -run TestNoteListCommand -v`             |
| `review promotions`  | `promotion-candidates.sh`                              | List promotion-ready notes                                                                      | `go test ./internal/cli -run TestReviewPromotionsCommand -v`       |
| `review weekly`      | Taskfile `ai:review:weekly`                            | Run gc + promotions + `check` (read-only) (+ optional security under strict policy)             | `go test ./internal/cli -run TestReviewWeeklyCommand -v`           |
| `report`             | (new)                                                  | Compute local memory effectiveness metrics from notes + telemetry                               | `go test ./internal/report -run TestReportCommand -v`              |
| `commit lint`        | (new)                                                  | Validate commit history against Conventional Commits rules                                      | `go test ./internal/commitlint -run TestCommitLintCommand -v`      |
| `changelog`          | (new)                                                  | Update `CHANGELOG.md` from conventional commits since last tag                                  | `go test ./internal/changelog -run TestChangelogCommand -v`        |
| `release propose`    | (new)                                                  | Generate release proposal (`lint + check + changelog + tag`) with approval gates                | `go test ./internal/release -run TestReleaseProposeCommand -v`     |
| `release approve`    | (new)                                                  | Approve and execute a release proposal with required gates                                      | `go test ./internal/release -run TestReleaseApproveCommand -v`     |
| `hooks install`      | (new)                                                  | Install/update local pre-commit hooks and generated config                                      | `go test ./internal/hooks -run TestHooksInstallCommand -v`         |
| `optimize recommend` | (new)                                                  | Propose bounded tuning changes from telemetry and outcomes (no auto-apply)                      | `go test ./internal/optimize -run TestOptimizeRecommendCommand -v` |
| `version`            | (new)                                                  | Print binary and schema version metadata                                                        | `go test ./internal/cli -run TestVersionCommand -v`                |
| `completion`         | (new)                                                  | Generate shell completions                                                                      | `go test ./internal/cli -run TestCompletionCommand -v`             |

## Manifest Schema

Each target repository may contain an `athena.toml` (or fall back to built-in
defaults). The manifest controls which features are scaffolded and how
templates are customised.

```toml
# athena.toml
version = 2

[features]
ai-memory         = true     # .ai/ directory tree and templates
contributing      = true     # CONTRIBUTING.md
editorconfig      = true     # .editorconfig
agents-md         = true     # AGENTS.md
agent-tooling     = true     # AGENT_TOOLING.md
agent-metrics     = false    # AGENT_METRICS.md (opt-out example)
claude-shim       = true     # CLAUDE.md
cursor-shim       = true     # .cursorrules
copilot-shim      = true     # .github/copilot-instructions.md
repomix-config    = true     # repomix.config.json
ci-workflow       = "github" # "github" | "none"
security-baseline = true     # .gitleaks.toml + actionlint config
pre-commit-hooks  = true     # .pre-commit-config.yaml
changelog         = true     # CHANGELOG.md scaffolding

[templates]
# Override built-in templates with repo-local paths
# note = ".athena/templates/note.md"
# improvement = ".athena/templates/improvement.md"
# changelog = ".athena/templates/changelog.md"

[scopes]
# Conventional commit scopes (used in CONTRIBUTING.md generation)
app    = "Application code"
api    = "API layer"
infra  = "Infrastructure configuration"
docs   = "Documentation"
ci     = "CI/CD pipelines and workflows"
meta   = "Repository-level config"

[gc]
days = 45

[tools]
# Override tool lists for `athena tools`
required    = ["git", "rg", "jq", "yq", "task"]
recommended = ["repomix", "gitleaks", "actionlint", "pre-commit", "difft", "fzf", "shfmt", "shellcheck"]

[telemetry]
enabled = true
path    = ".athena/telemetry.jsonl"
require_run_id_for_agents = true
capture_token_usage = true

[policy]
default = "standard" # strict | standard | lenient

[policy_gates]
enabled = true
report_path = ".athena/reports/policy-gate.json"
# Canonical IDs: check | security_scan | commit_lint | changelog | doctor
required_checks = ["check", "security_scan", "commit_lint"]

[lock]
ttl = "15m"
allow_force_reap = false

[execution]
plan_dir = ".athena/plans"
journal_path = ".athena/ops-journal.jsonl"
default_mode = "direct" # direct | plan-first
enforce_idempotency = true

[context]
provider         = "repomix"
default_profile  = "review"
output_path      = ".athena/context/pack.xml"
compress         = true
security_check   = true
include          = ["**/*.go", "**/*.md", "athena.toml"]
ignore           = [".git/**", "node_modules/**", "dist/**"]

[context.profiles.review]
style      = "xml"
compress   = true
strip_diff = true

[context.profiles.handoff]
style      = "markdown"
compress   = true
strip_diff = false

[context.profiles.release]
style      = "plain"
compress   = false
strip_diff = false

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

Each feature maps to one or more files. `athena init` only writes files for
enabled features. `athena upgrade` only touches files belonging to enabled
features.

| Feature             | Files                                                  |
| ------------------- | ------------------------------------------------------ |
| `ai-memory`         | `.ai/**`, `scripts/ai/**` (if keeping shell fallbacks) |
| `contributing`      | `CONTRIBUTING.md`                                      |
| `editorconfig`      | `.editorconfig`                                        |
| `agents-md`         | `AGENTS.md`                                            |
| `agent-tooling`     | `AGENT_TOOLING.md`                                     |
| `agent-metrics`     | `AGENT_METRICS.md`                                     |
| `claude-shim`       | `CLAUDE.md`                                            |
| `cursor-shim`       | `.cursorrules`                                         |
| `copilot-shim`      | `.github/copilot-instructions.md`                      |
| `repomix-config`    | `repomix.config.json`                                  |
| `ci-workflow`       | `.github/workflows/athena-framework-check.yml`         |
| `security-baseline` | `.gitleaks.toml`, `.github/actionlint.yaml`            |
| `pre-commit-hooks`  | `.pre-commit-config.yaml`                              |
| `changelog`         | `CHANGELOG.md`                                         |

### Presets

Presets provide sensible defaults without requiring a manifest file.

| Preset     | Enabled Features                            |
| ---------- | ------------------------------------------- |
| `minimal`  | `agents-md`, `ai-memory`, `claude-shim`     |
| `standard` | All features enabled except `agent-metrics` |
| `full`     | All features + `agent-metrics`              |

When no `athena.toml` exists, `athena init` uses the `standard` preset and
writes an `athena.toml` recording the selection.

### Config Precedence Matrix

When multiple knobs influence behavior, Athena resolves them in this order:

| Concern                     | Highest Precedence                         | Middle                                                   | Lowest            | Notes                                                                                          |
| --------------------------- | ------------------------------------------ | -------------------------------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------- |
| Command behavior            | CLI flags                                  | `[policy]`, subsystem config                             | Built-in defaults | Example: `--policy strict` overrides `[policy].default`.                                       |
| Scaffolding file creation   | `[features]` flags                         | Preset defaults                                          | Built-in defaults | Feature flags gate whether files are managed at all.                                           |
| Runtime subsystem execution | CLI command flags                          | subsystem section (`[security]`, `[context]`, `[hooks]`) | Built-in defaults | Example: `athena security scan --secrets` runs secrets scan even if workflow lint is disabled. |
| Execution mode              | explicit command (`plan`/`apply`)          | `[execution].default_mode`                               | Built-in defaults | `plan-first` mode requires approved plan execution for mutating operations.                    |
| Report output format        | command-specific flags (`--report-format`) | global `--format`                                        | Built-in defaults | Artifact/report formats are separate from primary command output.                              |

### Manifest Evolution

Athena must support manifest schema evolution explicitly.

- Current manifest schema version is `2`.
- Version `1` manifests are supported and auto-migrated to `2` in memory.
- `athena init` writes version `2` manifests.
- `athena upgrade`:
  - accepts `version = 1` and `version = 2`
  - writes back `version = 2` when any manifest rewrite occurs
  - fails for unknown future versions with `ATHENA-CONF-*` errors

## Embedded Templates

All template content is embedded into the binary at compile time using Go's
`embed` directive. Templates use Go `text/template` syntax for variable
interpolation (e.g. scopes in `CONTRIBUTING.md`, tool lists in
`AGENT_TOOLING.md`, commit types in pre-commit/changelog templates).

Static files (those without variables) are copied verbatim.

Template variables available:

| Variable                   | Source                              |
| -------------------------- | ----------------------------------- |
| `.Scopes`                  | `[scopes]` table from manifest      |
| `.Tools.Required`          | `[tools].required` from manifest    |
| `.Tools.Recommended`       | `[tools].recommended` from manifest |
| `.Features`                | Feature flags map                   |
| `.GCDays`                  | `[gc].days` from manifest           |
| `.PolicyLevel`             | `[policy].default` from manifest    |
| `.ConventionalCommitTypes` | `[conventional_commits].types`      |

## Context Integration (Repomix)

Athena does not implement a custom context packing engine. It orchestrates local
context packing through Repomix as an external CLI.

Behavior:

1. `athena context pack` resolves profile defaults from `[context]`.
2. Athena checks for `repomix` availability and prints actionable install hints
   when absent.
3. `--changed` mode computes changed files from git and passes the list to
   Repomix via stdin path mode.
4. `--stdout` streams packed context to stdout; otherwise output is written to
   `[context].output_path`.
5. `--` passthrough forwards raw args directly to Repomix.
6. `athena context mcp` validates/starts `repomix --mcp` for MCP-aware clients.
7. `athena context budget` computes token counts and fails under strict policy
   when the budget exceeds `--max-tokens`.

## Security and Workflow Diagnostics

Athena can orchestrate repository-local security and CI diagnostics without
sending data to remote services.

- `athena security scan --secrets` runs `gitleaks`.
- `athena security scan --workflows` runs `actionlint`.
- If neither flag is supplied, both scans run when enabled in `[security]`.
- Reports are written to `[security].report_dir` as JSON and optionally SARIF.
- `--report-format` selects the emitted report artifact format.
- `athena check --secrets --workflows` provides a unified one-command check
  path for CI.

Policy behavior:

- `strict`: missing optional tools becomes a failure.
- `standard`: missing optional tools is warning-only; found critical findings
  fail.
- `lenient`: findings are reported but command exits successfully unless parsing
  fails.

## Capability Negotiation

`athena capabilities --format json` returns a stable contract agents can use to
adapt safely without hardcoding command availability.

Capability payload includes:

- supported commands and flags
- schema versions (`manifest`, `frontmatter`, `telemetry`)
- output formats and report formats
- optional external tool support status
- policy levels and execution modes (`direct`, `plan-first`)

## Policy Gating for PRs

`athena policy gate` evaluates a PR or revision against configured gates.

- Default required gates come from `[policy_gates].required_checks`.
- Canonical check IDs are:
  - `check` -> `athena check`
  - `security_scan` -> `athena security scan`
  - `commit_lint` -> `athena commit lint`
  - `changelog` -> `athena changelog --dry-run`
  - `doctor` -> `athena doctor`
- Output includes machine-readable failure reasons with:
  - `policy_id`
  - `severity`
  - `summary`
  - `fix_hint`
- JSON report is written to `[policy_gates].report_path`.

Target resolution:

1. If `--pr REF` is provided, evaluate that ref directly (branch, tag, or full
   git ref).
2. If `--pr` is omitted, evaluate `HEAD`.
3. In CI, callers should pass explicit refs to avoid ambiguous detached-head
   behavior.
4. Athena does not require a remote provider API for gate evaluation.

Example failure item:

```json
{
  "policy_id": "ATHENA-POL-003",
  "severity": "error",
  "summary": "Frontmatter schema mismatch",
  "fix_hint": "Run `athena check --fix` and re-run policy gate."
}
```

## Execution Model (Plan and Apply)

Athena supports two-phase execution for mutating operations.

Every mutating command must have a plan representation and an apply path.

1. `athena plan <mutating-command> [args...]` computes a deterministic plan and
   stores it under `[execution].plan_dir`.
2. `athena apply --plan-id PLAN_ID` executes the exact plan contents.
3. Plans are immutable once created; applies fail if environment preconditions
   no longer match.

Rules:

- Mutating commands remain callable directly in `direct` mode.
- In `plan-first` mode (`[execution].default_mode`), direct mutation attempts
  fail with a policy error and require explicit plan/apply.
- `--dry-run` is independent and never writes.
- Direct-mode mutating commands execute within implicit transactions and are
  journaled using the same `tx_*` event model as `apply`.

## Idempotency Guarantees

When `[execution].enforce_idempotency = true`, each mutating operation must be
idempotent for identical inputs and repository state.

- Read-only commands must be deterministic for identical inputs and repository
  state.
- Re-running a completed plan must be a no-op with explicit `idempotent_noop`
  status.
- Operation identity uses command + normalized args + repository fingerprint.
- Non-idempotent divergence must fail with a stable `error_code`.

## Check and Fix Semantics

`athena check` is read-only by default.

- Without `--fix`, `athena check` must not write files.
- `athena check --fix` may:
  - apply safe frontmatter migrations
  - apply deterministic index rebuilds equivalent to `athena index`
  - write backup files before mutating existing note content
- `athena check --fix` prints a mutation summary: fixed, unchanged, skipped.

## Doctor Diagnostics

`athena doctor` reports repository readiness and config/tooling drift.

Checks:

1. Manifest parse and schema compatibility.
2. Managed path integrity (`.athena/`, `.ai/`, checksum files, lock dir).
3. Optional tool availability (`repomix`, `gitleaks`, `actionlint`,
   `pre-commit`) with policy-aware severity.
4. Lock health (active lock owner, stale lock eligibility by `[lock].ttl`).
5. Template checksum drift for managed files.

## Init Conflict Resolution

`athena init` must define behavior when managed files already exist but
`.athena/checksums.json` is missing.

Algorithm:

1. Acquire repository mutation lock.
2. Detect file collisions for the selected feature set.
3. If `--dry-run` is set, compute and print the planned actions without writing
   files, backups, or checksums, then exit successfully.
4. If no collisions exist, proceed and write `.athena/checksums.json`.
5. If collisions exist and `stdin` is a TTY, prompt per file with:
   `overwrite | skip | backup-and-overwrite`.
6. If collisions exist and `stdin` is not a TTY:
   - With `--force`: use `backup-and-overwrite`.
   - Without `--force`: default to `skip` and return exit code `2` if all files
     are skipped.
7. `backup-and-overwrite` writes `.athena/backups/<path>.<timestamp>.bak`
   before replacing the file.
8. Print summary totals: written, overwritten, skipped, backed up.

## Upgrade Strategy

`athena upgrade` must be safe to run repeatedly without data loss.

Algorithm:

1. Acquire repository mutation lock.
2. Read `athena.toml` (required for upgrade - refuse if missing).
3. Read `.athena/checksums.json` (required for safe overwrite decisions).
4. If `--dry-run` is set, compute and print the planned actions without writing
   files, backups, or checksums, then exit successfully.
5. For each managed file belonging to an enabled feature:
   - If file does not exist in target, write it.
   - If file exists and matches the previously-installed checksum, back it up to
     `.athena/backups/<path>.<timestamp>.bak` and overwrite with the new
     version (or report this plan in `--dry-run`).
   - If file exists and differs from stored checksum, skip and report
     `user-modified`.
6. Rewrite `.athena/checksums.json` for files written or overwritten.
7. Print summary: written, overwritten, backed up, skipped (user-modified),
   skipped (feature disabled).

To detect user modifications, `athena init` and `athena upgrade` write a
`.athena/checksums.json` file containing SHA-256 hashes of every installed file
at the time of installation. On upgrade, the current file hash is compared
against the stored hash. If they differ, the file has been user-modified and is
left untouched unless the user explicitly chooses overwrite during `init`
conflict resolution.

```json
{
  "version": 1,
  "installed_version": "0.3.0",
  "files": {
    "AGENTS.md": "sha256:abc123...",
    ".editorconfig": "sha256:def456..."
  }
}
```

## Note Frontmatter Contract

All notes in `.ai/` must contain YAML frontmatter with these required keys:

| Key                | Type    | Required | Description                                                                                      |
| ------------------ | ------- | -------- | ------------------------------------------------------------------------------------------------ |
| `id`               | string  | yes      | Unique identifier: `<type>-YYYYMMDD-<slug>`                                                      |
| `title`            | string  | yes      | Human-readable title                                                                             |
| `type`             | string  | yes      | One of: `context`, `investigation`, `troubleshooting`, `wip`, `improvement`, `session`, `memory` |
| `status`           | string  | yes      | One of: `active`, `closed`, `stale`, `superseded`, `promoted`                                    |
| `created`          | date    | yes      | ISO 8601 date                                                                                    |
| `updated`          | date    | yes      | ISO 8601 date                                                                                    |
| `schema_version`   | integer | no       | Frontmatter schema version; defaults to `1` when absent                                          |
| `related`          | list    | no       | Paths to related notes                                                                           |
| `promotion_target` | string  | no       | Target canonical doc path                                                                        |
| `supersedes`       | list    | no       | IDs of superseded notes                                                                          |
| `tags`             | list    | no       | Freeform tags                                                                                    |

`athena check` validates this contract and returns a non-zero exit code on
failure.

## Schema Evolution

Schema migration policy for future frontmatter versions (for example `v1` to
`v2`):

1. `athena check`:
   - validates current schema directly when `schema_version` equals latest
   - for older versions, attempts in-memory migration and validates the
     migrated structure
   - emits compatibility warnings for migrated notes
   - returns exit code `1` only when migration fails or required fields cannot
     be derived
2. `athena check --fix`:
   - applies migratable schema upgrades on disk
   - writes backups in `.athena/backups/` before rewriting migrated notes
   - updates `schema_version` to latest for rewritten notes
3. `athena upgrade`:
   - applies migratable schema upgrades on disk (unless `--dry-run`)
   - writes backups in `.athena/backups/` before rewriting migrated notes
   - updates `schema_version` to latest for rewritten notes
4. Optional strict mode (`athena check --strict-schema`) fails when any note is
   not already on the latest schema.

## Conventional Commits and Changelog

Athena uses Conventional Commits to standardize history and drive changelog
updates.

Conventional commit contract:

- Header format: `<type>(<optional-scope>): <description>`.
- `type` must be one of `[conventional_commits].types`.
- If scope is provided, it should match one of `[scopes]` keys.
- Breaking changes must use `!` in header or a `BREAKING CHANGE:` footer.

Commands:

- `athena commit lint` validates commits in a selectable git range.
- `athena changelog` parses conventional commits since a baseline tag and
  updates `CHANGELOG.md`.

Changelog behavior:

1. Ensure `CHANGELOG.md` exists; scaffold if absent.
2. Parse commits since `--since` tag (or last semver tag by default).
3. Group entries by category (`feat`, `fix`, `perf`, etc.).
4. Keep `## Unreleased` at top and add version section when `--next` is set.
5. Preserve manual notes outside generated markers.

## Release Workflow

Athena provides a deterministic release flow driven by commit history and
manifest policy.

Recommended release sequence:

1. `athena release propose --since <last-tag> --next <X.Y.Z>`
2. Review proposal artifact (gates, changelog diff, tag proposal).
3. `athena release approve --proposal-id <ID>`
4. Create release commit (for example: `chore(release): vX.Y.Z`).
5. Tag `vX.Y.Z` and publish via existing release pipeline.

Rules:

- `release propose` internally runs `commit lint`, `check`, `security scan`,
  and `changelog` in plan mode.
- `release approve` requires all approval gates to be satisfied and unchanged
  from proposal time.
- Release commits must pass commit lint and policy checks.
- `CHANGELOG.md` must contain the target version section before tagging.
- Under strict policy, blocking findings in check/security prevent release.

## Optimization Recommendations

`athena optimize recommend` analyzes local outcomes and telemetry to propose
bounded improvements.

Scope of recommendations:

- context profile defaults (include/ignore/compression)
- policy threshold tuning (warning severities, gate ordering)
- GC window and note lifecycle defaults
- check sequencing for faster successful completion

Guardrails:

- Recommendations are written as proposal files under `[optimize].proposal_path`.
- No autonomous self-modification of Athena code is performed.
- `auto_apply` remains `false` by default and requires explicit approval to
  change.
- Proposals include projected benefit and confidence scores with sample counts.

## Index Format

`athena index` generates `.ai/index.yaml`:

```yaml
version: 1
generated: "2026-02-23T12:00:00Z"
counts:
  total: 5
  active: 3
  stale: 1
  closed: 1
entries:
  - path: ".ai/context/ctx-20260220-auth-flow.md"
    type: context
    status: active
    updated: "2026-02-20"
    title: "Authentication Flow Analysis"
```

## Concurrency and Locking

All state-mutating commands must enforce single-writer behavior.

Mutating commands include:

- `init`, `upgrade`, `gc`
- `check --fix`
- `note new`, `note close`, `note promote`
- `hooks install`, `changelog`
- `apply`, `rollback`, `release approve`
- any command that rewrites `.ai/index.yaml` or files under `.athena/`

Lock contract:

1. Acquire `.athena/locks/repo.lock` using atomic file creation.
2. Lock metadata includes PID, hostname, command, and timestamp.
3. Wait up to `--lock-timeout` for lock release.
4. On timeout, exit with code `3` and print lock holder details.
5. Stale locks older than `[lock].ttl` can be force-reaped when
   `[lock].allow_force_reap = true` and user confirmation is available (or
   strict non-interactive refusal).

## Operation Journal and Transactions

Every mutating operation (direct mode and apply mode) is executed as a
transaction and logged in `[execution].journal_path` as append-only JSONLines.

Journal event types:

- `tx_started`
- `step_applied`
- `step_rolled_back`
- `tx_committed`
- `tx_failed`
- `tx_rolled_back`

Transaction contract:

1. Each transaction has a unique `tx_id` and references optional `plan_id`.
2. Each step records pre-image backup location (if applicable).
3. Failed transactions must attempt best-effort rollback automatically.
4. `athena rollback --tx TX_ID` performs manual rollback using journal + backups.

## Machine-Readable Output Contracts

When `--format json` is used, Athena emits stable JSON envelopes with command
specific payloads.

Common envelope:

```json
{
  "command": "check",
  "ok": true,
  "policy": "standard",
  "duration_ms": 31,
  "warnings": [],
  "errors": []
}
```

Error object contract:

```json
{
  "error_code": "ATHENA-VAL-001",
  "message": "Frontmatter schema mismatch",
  "actionable_fix": "Run `athena check --fix`.",
  "policy_id": "ATHENA-POL-003"
}
```

### `athena check --format json`

```json
{
  "command": "check",
  "ok": true,
  "mode": "read-only",
  "summary": {
    "files_scanned": 12,
    "valid": 12,
    "invalid": 0,
    "migratable": 1,
    "fixed": 0
  }
}
```

### `athena doctor --format json`

```json
{
  "command": "doctor",
  "ok": true,
  "checks": [
    { "name": "manifest", "status": "pass" },
    { "name": "tooling", "status": "warn", "detail": "repomix missing" }
  ]
}
```

### `athena capabilities --format json`

Illustrative example (abbreviated command list):

```json
{
  "command": "capabilities",
  "ok": true,
  "capabilities": {
    "commands": ["check", "policy gate", "plan", "apply", "optimize recommend"],
    "commands_complete": false,
    "execution_modes": ["direct", "plan-first"],
    "output_formats": ["text", "json"],
    "report_formats": ["json", "sarif"],
    "schema_versions": {
      "manifest": 2,
      "frontmatter": 1,
      "telemetry": 1
    }
  }
}
```

### `athena policy gate --format json`

```json
{
  "command": "policy gate",
  "ok": false,
  "target_ref": "refs/pull/42/head",
  "errors": [
    {
      "error_code": "ATHENA-POL-003",
      "message": "Frontmatter schema mismatch",
      "actionable_fix": "Run `athena check --fix` and re-run policy gate.",
      "policy_id": "ATHENA-POL-003"
    }
  ],
  "failures": [
    {
      "policy_id": "ATHENA-POL-003",
      "severity": "error",
      "summary": "Frontmatter schema mismatch",
      "fix_hint": "Run `athena check --fix` and re-run policy gate."
    }
  ]
}
```

### `athena report --format json`

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

### `athena plan --format json`

```json
{
  "command": "plan",
  "ok": true,
  "plan_id": "plan_20260223_abc123",
  "mutations": 4,
  "idempotency_key": "sha256:..."
}
```

### `athena release propose --format json`

```json
{
  "command": "release propose",
  "ok": true,
  "proposal_id": "relprop_20260223_01",
  "next_version": "1.4.0",
  "gates": [
    { "name": "commit_lint", "status": "pass" },
    { "name": "check", "status": "pass" },
    { "name": "security_scan", "status": "pass" }
  ]
}
```

### `athena optimize recommend --format json`

```json
{
  "command": "optimize recommend",
  "ok": true,
  "window_days": 30,
  "proposals": [
    {
      "proposal_id": "opt_20260223_aa1",
      "target": "context.profiles.review.compress",
      "current": true,
      "recommended": false,
      "projected_token_reduction": 0.18,
      "confidence": 0.74
    }
  ]
}
```

### `athena security scan --format json`

```json
{
  "command": "security scan",
  "ok": false,
  "errors": [
    {
      "error_code": "ATHENA-TOOL-002",
      "message": "gitleaks found blocking secrets",
      "actionable_fix": "Remove or rotate exposed secrets and rerun `athena security scan`."
    }
  ],
  "tools": {
    "gitleaks": { "status": "fail", "findings": 2 },
    "actionlint": { "status": "pass", "findings": 0 }
  }
}
```

Security artifact outputs:

- `athena security scan --report-format json` writes a JSON artifact to
  `[security].report_dir`.
- `athena security scan --report-format sarif` writes a SARIF artifact to
  `[security].report_dir`.

## Error Taxonomy and Remediation

Athena errors must be stable, machine-readable, and actionable for agents.

Error code namespaces:

- `ATHENA-CONF-*` configuration and manifest issues
- `ATHENA-POL-*` policy gate failures
- `ATHENA-VAL-*` validation/schema failures
- `ATHENA-EXEC-*` planning/apply/rollback/idempotency failures
- `ATHENA-TOOL-*` optional tool integration failures

Requirements:

- Every non-zero exit response in `--format json` includes at least one
  `error_code`.
- Every `error_code` includes an `actionable_fix` hint.
- Policy-related failures include `policy_id` when applicable.

## Exit Codes

| Code | Meaning                                                                                                               |
| ---- | --------------------------------------------------------------------------------------------------------------------- |
| 0    | Success                                                                                                               |
| 1    | Validation or policy failure (frontmatter, schema migration, commit lint, security findings in strict/standard modes) |
| 2    | Configuration/runtime contract error (bad manifest, missing args, unresolved init collisions)                         |
| 3    | Lock acquisition timeout or concurrency conflict                                                                      |
| 4    | Plan/apply contract failure (missing approval gate, stale plan preconditions, rollback failure)                       |

## Implementation Notes

### Language and Toolchain

- **Language**: Go (1.23+)
- **Template embedding**: `go:embed` directive
- **CLI framework**: `cobra` + `viper` for flag/config binding
- **YAML parsing**: `gopkg.in/yaml.v3`
- **TOML parsing**: `github.com/BurntSushi/toml`
- **TTY detection**: `golang.org/x/term`
- **Hashing**: `crypto/sha256` (stdlib)
- **External CLIs (optional)**: `repomix`, `gitleaks`, `actionlint`,
  `pre-commit`

### Build and Release

- **Build**: `goreleaser` for cross-compilation (linux/darwin/windows,
  amd64/arm64)
- **Release**: GitHub Releases with checksums and signed binaries
- **Distribution**:
  - Direct download from GitHub Releases
  - Homebrew tap (`amr-athena/tap/athena`)
  - Nix flake overlay (optional, for NixOS users)
- **Versioning**: Semantic versioning, tagged as `vX.Y.Z`

### Project Structure (indicative)

```text
cmd/
  athena/
    main.go
internal/
  cli/              # cobra command definitions and root flags
  config/           # athena.toml parsing and defaults
  lock/             # lock acquisition/release and stale lock handling
  scaffold/         # init + upgrade logic
  notes/            # note lifecycle (new, close, promote, list)
  validate/         # frontmatter validation + migration checks
  index/            # .ai/index.yaml generation
  gc/               # garbage collection
  report/           # local metrics calculations
  capabilities/     # capability negotiation and schema reporting
  policy/           # policy gate evaluation and PR contract reporting
  execution/        # plan/apply orchestration, idempotency, rollback
  context/          # repomix command orchestration and budget logic
  security/         # gitleaks/actionlint wrappers and report persistence
  doctor/           # repository diagnostics aggregation
  release/          # release proposal and approval gate orchestration
  optimize/         # bounded recommendation generation from telemetry
  hooks/            # pre-commit config generation/install
  commitlint/       # conventional commit parsing and linting
  changelog/        # changelog generation from commit history
  telemetry/        # telemetry append/read helpers
  errors/           # stable error taxonomy and actionable fix mapping
  tools/            # tool availability checking
  templates/        # go:embed template tree
    embed.go
    files/          # raw template files
embed/              # compiled-in template content
```

### Testing Strategy

- **Unit tests**:
  - frontmatter parsing + schema migration (100% coverage target)
  - config defaults and feature flag logic
  - checksum comparison and conflict resolution rules
  - capability payload stability and schema version reporting
  - policy gate classification (`policy_id`, severity, fix hint mapping)
  - idempotency key generation and duplicate plan detection
  - lock acquisition and stale lock behavior
  - transaction journal serialization and rollback step validation
  - conventional commit parser and changelog grouping
  - telemetry JSONLines parsing and report metrics math
- **Integration tests**:
  - `athena init` into a temp directory with collision prompts and dry-run paths
  - `athena upgrade` with user-modified files and backup writes
  - `athena check` against valid/invalid/migratable notes (read-only mode)
  - `athena check --fix` for deterministic repair and migration writes
  - `athena plan` then `athena apply` parity with direct execution
  - `athena rollback` recovery from partial mutation failures
  - `athena policy gate` over passing/failing fixture revisions
  - `athena report` over fixture notes + telemetry logs
  - `athena capabilities` stability snapshot tests
  - `athena context pack` wrapper behavior with mocked repomix executable
  - `athena security scan` wrapper behavior with mocked gitleaks/actionlint
  - `athena release propose` and `athena release approve` gate flow tests
  - `athena optimize recommend` proposal generation from fixture telemetry
  - `athena hooks install` end-to-end scaffold/update behavior
  - `athena changelog` update behavior with fixture git histories
  - JSON output contract tests for `check`, `doctor`, `report`,
    `security scan`, `capabilities`, `policy gate`, and `plan`
- **Golden file tests**: rendered templates compared against expected output
- **CI**: GitHub Actions with `go test ./...`, `golangci-lint`, GoReleaser
  snapshot builds

## Migration from Shell Scripts

The Go CLI replaces all shell scripts. For backward compatibility during
transition:

1. The Taskfile (`.task/ai.yml`) can be updated to call `athena` subcommands
   instead of shell scripts, preserving the `task ai:*` interface.
2. Once users have `athena` installed, the Taskfile and `scripts/ai/` directory
   become optional.
3. A future `athena init` will stop scaffolding `scripts/ai/` and `.task/ai.yml`
   by default (behind a `legacy-taskfile` feature flag during transition).

## Scope Boundaries

The `athena` CLI owns:

- Scaffolding and upgrading framework files in target repositories
- AI working memory lifecycle (notes, index, validation, gc)
- Local diagnostics, policy checks, and memory effectiveness reporting
- Wrapping local external tools for context packing, security checks, and hooks
- Conventional commit linting and changelog generation
- Plan/apply orchestration, transaction journaling, and rollback mechanics
- Release proposal and approval-gated release orchestration
  (including gated release commit/tag preparation flows)
- Bounded optimization recommendations from local telemetry

The `athena` CLI does **not** own:

- Repository-specific business validation commands beyond configured wrappers
- Git operations that mutate history (rebases, force pushes)
- Implementing a native context packer (delegate to `repomix`)
- Remote telemetry collection or cloud analytics
- Autonomous code self-modification or unapproved auto-tuning

## Metrics & Local Diagnostics

To ensure the `.ai/` directory functions as an effective episodic memory rather
than a graveyard of unresolved context, `athena` tracks local usage metrics.
This telemetry is strictly local (no cloud phoning home) and is used to
generate effectiveness reports for repository maintainers and agents.

### The `athena report` Command

A new command, `athena report`, parses local data to output a summary of
repository health:

- **Staleness Ratio:** Percentage of notes that reach the GC threshold (for
  example, 45 days) without transitioning to `closed` or `promoted`.
- **Promotion Rate:** Percentage of `improvement` or `investigation` notes that
  successfully reach the `promoted` status.
- **Orphan Rate:** Percentage of notes with an empty `related` list (indicating
  poor knowledge graph connectivity).
- **Autonomous Execution:** Ratio of commands executed by AI agents (headless)
  versus humans (TTY attached).
- **Context Efficiency:** Ratio of context-pack commands that stay within token
  budget.
- **Security Hygiene:** Ratio of scans with zero critical findings.

Report data-source policy:

- Note-state metrics (`staleness_ratio`, `promotion_rate`, `orphan_rate`) are
  computed from `.ai/**` and remain valid without telemetry.
- Execution metrics (`autonomous_execution`, `context_efficiency`,
  `security_hygiene`) are computed from telemetry and tool reports when
  available.
- Output includes `telemetry_coverage` to indicate confidence for telemetry-
  derived metrics.

Token/diagnostic correlation contract:

- Agent orchestrators should supply a shared `run_id` for all Athena command
  invocations in a task.
- When available, token/cost fields are attached to telemetry events and
  attributed to command outcomes.
- Athena uses this linkage for optimization recommendations, not automatic
  runtime mutation.

### Telemetry Log Format

State-mutating commands (`init`, `upgrade`, `check --fix`, `note new`,
`note close`, `note promote`, `gc`, `hooks install`, `changelog`, `apply`,
`rollback`, `release approve`) append a record to `.athena/telemetry.jsonl`.

Operational commands that feed report metrics (`context pack`,
`context budget`, `security scan`) also append telemetry records.

Read-only diagnostic commands may append opt-in telemetry records when
`[telemetry].enabled = true`.

`error_code` semantics:

- `null` means successful execution (`exit_code = 0`).
- a non-empty string means command failure with a stable Athena error code.

To differentiate human developers and autonomous AI agents, `athena` checks if
the process is attached to a TTY (`golang.org/x/term`).

**JSONLines schema:**

```json
{
  "timestamp": "2026-02-23T21:14:21Z",
  "command": "note promote",
  "run_id": "run_20260223_4f2a",
  "task_id": "task_auth_flow",
  "agent_name": "codex",
  "model": "gpt-5-codex",
  "execution_time_ms": 42,
  "prompt_tokens": 812,
  "completion_tokens": 221,
  "total_tokens": 1033,
  "cost_usd": 0.0184,
  "is_tty": false,
  "exit_code": 0,
  "error_code": null,
  "policy_level": "standard",
  "is_dry_run": false,
  "context": {
    "note_id": "improvement-20260220-auth-flow",
    "target_doc": "docs/architecture/auth.md"
  }
}
```

Note: `is_tty: false` indicates the command was likely run autonomously by an
AI coding agent.
