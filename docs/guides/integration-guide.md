# Athena Integration Guide

How to use Athena to maximise AI-agent effectiveness in a source code repository.

---

## 1. Install and Bootstrap

### Install the binary

```sh
# From source (Go 1.25+)
go install github.com/aredgwell/athena/cmd/athena@latest
```

### Bootstrap a repository

```sh
cd /path/to/your-repo
athena init --preset standard
```

This creates:

| Path | Purpose |
|------|---------|
| `athena.toml` | Feature flags, policy level, tool lists |
| `.athena/checksums.json` | Integrity hashes of every managed file |
| `.ai/` + `.ai/index.yaml` | AI working-memory root and note index |
| `AGENTS.md` | Canonical agent execution protocol |
| `CLAUDE.md` | Claude-specific shim pointing to `AGENTS.md` |
| `.editorconfig` | Consistent formatting across editors |

Available presets:

| Preset | Scope |
|--------|-------|
| `minimal` | `agents-md`, `ai-memory`, `claude-shim` only |
| `standard` | Everything except `agent-metrics` (recommended) |
| `full` | All features including `agent-metrics` |

After init, verify the installation:

```sh
athena doctor
athena tools
```

---

## 2. Core Concept: AI Working Memory

Athena manages a structured `.ai/` directory that serves as **persistent working memory** for AI agents across sessions. Notes are Markdown files with YAML frontmatter that follow a strict lifecycle.

### Note types

| Type | When to create | Example |
|------|---------------|---------|
| `context` | Recording architectural decisions, domain knowledge, patterns | `context-20260224-auth-flow.md` |
| `investigation` | Deep-diving into a problem or exploring options | `investigation-20260224-perf-bottleneck.md` |
| `troubleshooting` | Debugging issues, recording symptoms and fixes | `troubleshooting-20260224-ci-timeout.md` |
| `wip` | Tracking in-progress work across sessions | `wip-20260224-api-migration.md` |
| `improvement` | Proposing changes to the codebase or process | `improvement-20260224-error-handling.md` |
| `session` | Session log summarising what was done | `session-20260224-sprint-review.md` |
| `memory` | Long-lived reference knowledge | `memory-20260224-deployment-runbook.md` |

### Note lifecycle

```
active → closed       (work complete, archived for reference)
active → stale        (auto-marked by gc after N days of inactivity)
active → promoted     (knowledge graduated to canonical docs)
active → superseded   (replaced by a newer note)
```

### Commands for note management

```sh
# Create a new note
athena note new --type investigation --slug perf-bottleneck --title "API latency investigation"

# List active notes
athena note list --status active

# List by type
athena note list --type investigation

# Close a completed note
athena note close --status closed .ai/investigations/investigation-20260224-perf-bottleneck.md

# Promote knowledge to canonical docs
athena note promote --target docs/architecture/api-latency.md .ai/improvements/improvement-20260224-error-handling.md

# Rebuild the index after manual edits
athena index
```

---

## 3. Agent Session Workflow

This is the recommended workflow for an AI agent operating in an Athena-managed repository.

### Session start (preflight)

```sh
# 1. Discover capabilities (machine-readable)
athena capabilities --format json

# 2. Run diagnostics
athena doctor --format json

# 3. Check tool availability
athena tools --format json

# 4. Review active notes for context
athena note list --status active --format json

# 5. Validate current state
athena check --format json
```

An agent should parse `capabilities` output to discover available commands rather than hardcoding them. The `--format json` flag returns a stable envelope that agents can reliably parse:

```json
{
  "command": "capabilities",
  "ok": true,
  "duration_ms": 2,
  "warnings": [],
  "errors": [],
  "data": {
    "commands": ["init", "upgrade", "check", ...],
    "schema_versions": { "manifest": 2, "frontmatter": 1, "telemetry": 1 },
    "execution_modes": ["direct", "plan-first"],
    "output_formats": ["text", "json"],
    "report_formats": ["json", "sarif"]
  }
}
```

### During work

```sh
# Create a WIP note tracking the current task
athena note new --type wip --slug api-migration --title "Migrating REST to gRPC"

# Pack context for an LLM prompt (changed files only)
athena context pack --profile review --changed --stdout

# Pack full repo context for a handoff
athena context pack --profile handoff --output .athena/context/handoff.xml

# Estimate token budget before packing
athena context budget --profile review --max-tokens 100000

# Validate after making changes
athena check --format json

# Lint commits before pushing
athena commit lint --from main --to HEAD
```

### Session end

```sh
# Close completed notes
athena note close --status closed .ai/wip/wip-20260224-api-migration.md

# Run validation
athena check

# Generate effectiveness report
athena report --format json
```

---

## 4. MCP Server for IDE Agents

`athena mcp` starts a [Model Context Protocol](https://modelcontextprotocol.io/) server over stdio, exposing Athena commands as MCP tools and resources. This eliminates shell-spawning overhead for IDE agents and gives them structured access to your repository's working memory.

The server exposes 17 tools across four categories (notes, validation, governance, context management) and 5 read-only resources. See the [MCP Setup Guide](mcp-setup.md) for per-client configuration (Claude Code, Cursor, Windsurf) and the complete tool/resource reference.

---

## 5. Context Packing for LLM Prompts

The `context` command group wraps [repomix](https://github.com/yamadashy/repomix) to generate token-optimised repository snapshots for LLM consumption.

### Profiles

| Profile | Use case | Style | Compressed |
|---------|----------|-------|------------|
| `review` | Code review, PR analysis | XML | Yes |
| `handoff` | Transferring context between agents | Markdown | Yes |
| `release` | Release notes, changelog context | Plain | No |

### Examples

```sh
# Pack only changed files for a review prompt
athena context pack --profile review --changed --stdout | pbcopy

# Full repo pack for agent handoff
athena context pack --profile handoff

# Check how much budget a pack would use
athena context budget --profile review --max-tokens 128000

# Start repomix MCP server for tool-use integration
athena context mcp --stdio
```

### Configuration in `athena.toml`

```toml
[context]
provider        = "repomix"
default_profile = "review"
output_path     = ".athena/context/pack.xml"
compress        = true
security_check  = true
include         = ["**/*.go", "**/*.md", "athena.toml"]
ignore          = [".git/**", "node_modules/**", "dist/**"]

[context.profiles.review]
style      = "xml"
compress   = true
strip_diff = true
```

---

## 6. Validation and Health Checks

### Continuous validation

```sh
# Full validation: frontmatter, schema, note integrity
athena check

# With auto-fix for simple issues
athena check --fix

# Include secrets and workflow linting
athena check --secrets --workflows

# Strict mode: enforce latest schema version on all notes
athena check --strict-schema
```

### Repository diagnostics

```sh
# Comprehensive health check
athena doctor

# Check external tool availability
athena tools
athena tools --strict  # treat missing recommended tools as errors
```

### Garbage collection

Notes go stale over time. GC identifies inactive notes:

```sh
# Preview what would be marked stale (45-day default)
athena gc --dry-run

# Mark stale notes with a 30-day threshold
athena gc --days 30

# Weekly review: combines gc + check + promotion candidates
athena review weekly --days 7
```

---

## 7. Policy Gates and Security

### Policy levels

Athena enforces three policy levels, set in `athena.toml` or via `--policy` flag:

| Level | Behaviour |
|-------|-----------|
| `strict` | All checks must pass. No warnings tolerated. |
| `standard` | Errors fail, warnings are reported but allowed. |
| `lenient` | Advisory only. Nothing blocks. |

### PR gate evaluation

```sh
# Evaluate current state against policy gates
athena policy gate --pr HEAD --format json
```

This runs the checks configured in `[policy_gates].required_checks` (default: `check`, `security_scan`, `commit_lint`) and produces a structured pass/fail report.

### Security scanning

```sh
# Full security scan (secrets + workflow lint)
athena security scan

# Secrets only (requires gitleaks)
athena security scan --secrets

# Workflow lint only (requires actionlint)
athena security scan --workflows

# SARIF output for CI integration
athena security scan --report-format sarif
```

---

## 8. Plan/Apply Execution Model

For high-stakes mutations, Athena supports a two-phase execution model.

### Direct mode (default)

Mutating commands execute immediately within an implicit transaction. Every step is journaled to `.athena/ops-journal.jsonl` for auditability.

### Plan-first mode

Enable in `athena.toml`:

```toml
[execution]
default_mode = "plan-first"
```

Then:

```sh
# Step 1: Compute a plan (no mutations)
athena plan init --preset standard
# → outputs plan_id: plan_20260224_abc123

# Step 2: Review the plan
cat .athena/plans/plan_20260224_abc123.json

# Step 3: Execute the plan
athena apply --plan-id plan_20260224_abc123

# If something goes wrong: rollback
athena rollback --tx TX_20260224_def456
athena rollback --tx TX_20260224_def456 --to-step 3  # partial rollback
```

### Idempotency

Re-running any mutating command with identical inputs and repo state is a no-op. The journal records `idempotent_noop` status so agents can safely retry without side effects.

---

## 9. Release Workflow

### Commit linting

```sh
# Lint commits on current branch
athena commit lint --from main --to HEAD
```

Enforces [Conventional Commits](https://www.conventionalcommits.org/) format. Types, scope requirements, and rules are configured in `[conventional_commits]`.

### Changelog generation

```sh
# Preview changelog entries
athena changelog --since v1.0.0 --next v1.1.0 --dry-run

# Generate and update CHANGELOG.md
athena changelog --since v1.0.0 --next v1.1.0
```

### Release proposals

```sh
# Generate a release proposal with gate checks
athena release propose --since v1.0.0 --next v1.1.0

# Approve and execute the release
athena release approve --proposal-id <id>
```

Release proposals run all policy gates before allowing approval.

---

## 10. Telemetry and Optimisation

### Local-only telemetry

Athena records command execution data to `.athena/telemetry.jsonl`. No data leaves the repository. Telemetry captures:

- Command name, execution time, exit code
- Token usage (prompt/completion/total) and cost when provided
- Agent name, model, run ID for correlation
- Policy level and dry-run status

### Effectiveness reporting

```sh
athena report --format json
```

Produces metrics like note count by status, promotion rate, staleness distribution, and coverage ratios.

### Automated tuning recommendations

```sh
athena optimize recommend --window 30d
```

Analyses telemetry over the specified window and proposes bounded configuration changes (e.g., adjusting GC threshold, context budget, or profile settings) with confidence scores and projected token reduction.

---

## 11. Configuration Reference

### Minimal `athena.toml`

```toml
version = 2

[features]
ai-memory    = true
agents-md    = true
claude-shim  = true

[gc]
days = 45

[policy]
default = "standard"

[telemetry]
enabled = true
```

### Recommended `athena.toml` for agent-heavy repositories

```toml
version = 2

[features]
ai-memory         = true
agents-md         = true
agent-tooling     = true
agent-metrics     = true
claude-shim       = true
editorconfig      = true
repomix-config    = true
security-baseline = true
pre-commit-hooks  = true
changelog         = true
contributing      = true

[gc]
days = 30

[tools]
required    = ["git", "rg", "jq", "yq", "task"]
recommended = ["repomix", "gitleaks", "actionlint", "pre-commit", "difft"]

[telemetry]
enabled                   = true
path                      = ".athena/telemetry.jsonl"
require_run_id_for_agents = true
capture_token_usage       = true

[policy]
default = "standard"

[policy_gates]
enabled         = true
required_checks = ["check", "security_scan", "commit_lint"]

[execution]
default_mode        = "direct"
enforce_idempotency = true

[context]
provider        = "repomix"
default_profile = "review"
compress        = true
security_check  = true

[conventional_commits]
enforce       = true
require_scope = false

[hooks]
pre_commit = true

[optimize]
enabled     = true
window_days = 30
min_samples = 50
auto_apply  = false
```

---

## 12. CI Integration

### GitHub Actions example

```yaml
name: Athena Framework Check
on: [pull_request]

jobs:
  athena-gate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - name: Install Athena
        run: go install github.com/aredgwell/athena@latest

      - name: Validate notes and schema
        run: athena check --strict-schema --format json

      - name: Lint commits
        run: athena commit lint --from ${{ github.event.pull_request.base.sha }} --to ${{ github.event.pull_request.head.sha }} --format json

      - name: Policy gate
        run: athena policy gate --pr HEAD --format json

      - name: Security scan
        run: athena security scan --report-format sarif
```

### Pre-commit hooks

```sh
athena hooks install --pre-commit
```

This generates a `.pre-commit-config.yaml` that runs `athena check` and `athena commit lint` on every commit.

---

## 13. Best Practices for Maximum Agent Effectiveness

### Structure knowledge as notes, not comments

Instead of leaving context in code comments or PR descriptions that get lost, create notes:

```sh
athena note new --type context --slug auth-flow --title "Authentication flow architecture"
```

Notes persist across sessions, are indexed, searchable, and follow a lifecycle.

### Use weekly reviews to prevent knowledge rot

```sh
athena review weekly
```

This combines garbage collection, validation, and promotion candidate identification in a single pass. Run it weekly (or on a cron) to keep the `.ai/` directory healthy.

### Promote mature knowledge to canonical docs

When an investigation or improvement note matures into stable knowledge, promote it:

```sh
athena note promote --target docs/architecture/auth-flow.md .ai/investigations/investigation-20260224-auth-flow.md
```

This prevents the `.ai/` directory from becoming a graveyard of stale notes while ensuring valuable knowledge reaches canonical documentation.

### Use context packing strategically

- **Start of session**: Pack changed files with `--profile review --changed` to give the agent focused context.
- **Handoff between agents**: Pack with `--profile handoff` for comprehensive context transfer.
- **Before release**: Pack with `--profile release` for changelog and release note generation.
- **Budget first**: Always run `athena context budget` before packing to avoid exceeding token limits.

### Let agents discover capabilities at runtime

Rather than hardcoding Athena commands in agent prompts, have agents call:

```sh
athena capabilities --format json
```

This returns the complete command list, schema versions, and supported execution modes. This makes agent integrations forward-compatible as Athena evolves.

### Use policy gates as agent guardrails

Set `[policy].default = "standard"` and configure `[policy_gates]` with required checks. Agents that run `athena policy gate` before pushing will catch:

- Invalid or missing note frontmatter
- Secret leaks in committed files
- Non-conventional commit messages
- Schema version mismatches

### Track agent effectiveness over time

Enable `[telemetry]` and `[optimize]` sections. Periodically run:

```sh
athena report
athena optimize recommend
```

This creates a feedback loop: telemetry feeds reports, reports identify inefficiencies, and optimise proposes bounded configuration changes.

---

## 14. Troubleshooting

| Symptom | Diagnostic | Fix |
|---------|-----------|-----|
| `athena check` reports schema errors | Notes have outdated frontmatter | `athena check --fix` |
| `athena upgrade` skips files | User modified a managed file | Expected behaviour; review diff and re-init with `--force` if intended |
| Lock timeout errors | Stale lock from crashed process | Check `.athena/locks/repo.lock` contents; lock auto-expires after TTL |
| `repomix` not found | Context packing requires repomix | `npm install -g repomix` or disable context features |
| `gitleaks` not found | Security scanning requires gitleaks | `brew install gitleaks` or `go install github.com/gitleaks/gitleaks/v8@latest` |
| No telemetry data for optimise | Too few command runs | Wait for `min_samples` (default 50) recorded events |
| `athena doctor` reports issues | Various | Follow the actionable fix in each error's `actionable_fix` field |
