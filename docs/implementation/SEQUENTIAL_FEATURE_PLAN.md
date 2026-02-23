# Athena CLI Sequential Implementation Plan

This plan serializes `ATHENA.md` into implementable features that can be executed sequentially by an AI agent from start to finish.

## Execution Rules

- Implement features strictly in order.
- Do not start a feature until dependencies are completed and verified.
- Each feature must end with:
  - code changes committed (or patch staged)
  - acceptance tests passing
  - updated docs/changelogs for scope
  - a short machine-readable handoff note under `.athena/feature-handoffs/<feature-id>.json`

## Global Gate

Before each feature:

1. Run `go test ./...` baseline and record failures (starting after F00).
2. Run only impacted package tests while implementing.
3. Re-run full `go test ./...` before closing the feature.

## Feature Backlog

## F00 - Repository Bootstrap

- Goal: Create a minimal Go project skeleton so sequential features can execute.
- Dependencies: none
- Scope:
  - `go.mod`, `go.sum`
  - `cmd/athena/main.go` (minimal root command)
  - initial package directories under `internal/`
  - baseline test harness smoke test
- Acceptance:
  - `go test ./...`
  - `go run ./cmd/athena --help`
- DoD:
  - repository is buildable/testable for downstream feature implementation

## F01 - Core Config and Contracts

- Goal: Implement config loading, defaults, and schema structures for `athena.toml`.
- Dependencies: F00
- Scope:
  - `internal/config`
  - config structs for `[features]`, `[policy]`, `[policy_gates]`, `[execution]`, `[lock]`, `[telemetry]`, `[context]`, `[security]`, `[optimize]`
  - precedence helper functions matching `ATHENA.md` matrix
- Acceptance:
  - `go test ./internal/config -v`
  - table-driven tests for defaults + precedence resolution
- DoD:
  - deterministic merge order and validation errors surfaced with stable codes

## F02 - Error Taxonomy + JSON Envelope

- Goal: Implement shared response envelope and error model.
- Dependencies: F01
- Scope:
  - `internal/errors`
  - `error_code`, `actionable_fix`, optional `policy_id`
  - common JSON output envelope support in CLI helpers
- Acceptance:
  - `go test ./internal/errors -v`
  - `go test ./internal/cli -run TestJSONEnvelope -v`
- DoD:
  - all non-zero JSON responses include at least one structured error

## F03 - Capabilities Command

- Goal: Implement machine-readable capability negotiation.
- Dependencies: F01, F02
- Scope:
  - `internal/capabilities`
  - `athena capabilities --format json`
  - schema version and command inventory output
- Acceptance:
  - `go test ./internal/capabilities -v`
  - snapshot test for capabilities JSON
- DoD:
  - stable output contract + versioned capability payload

## F04 - Locking + Journaling Foundation

- Goal: Implement lock manager and operation journal primitives.
- Dependencies: F01, F02
- Scope:
  - `internal/lock`
  - `internal/execution` journal writer/reader
  - lock TTL + stale lock behavior
- Acceptance:
  - `go test ./internal/lock -v`
  - `go test ./internal/execution -run TestJournalPrimitives -v`
- DoD:
  - atomic lock acquisition and append-only journal semantics

## F05 - Plan / Apply / Rollback Engine

- Goal: Implement two-phase execution model and transaction handling.
- Dependencies: F04
- Scope:
  - `athena plan`, `athena apply`, `athena rollback`
  - plan persistence and precondition checks
  - tx lifecycle events: started/applied/failed/rolled_back/committed
- Acceptance:
  - `go test ./internal/execution -v`
  - integration test: plan -> apply -> forced failure -> rollback
- DoD:
  - `plan-first` mode enforced from config

## F06 - Scaffold Init + Upgrade

- Goal: Implement managed file lifecycle with checksums/backups/conflict handling.
- Dependencies: F01, F04, F05
- Scope:
  - `internal/scaffold`
  - `init` conflict resolution rules
  - `upgrade` checksum comparison + backups
  - dry-run behavior
- Acceptance:
  - `go test ./internal/scaffold -v`
- DoD:
  - idempotent repeated `init`/`upgrade` behavior

## F07 - Notes + Validation + Schema Evolution

- Goal: Implement note frontmatter contract and migration engine.
- Dependencies: F01, F02, F05
- Scope:
  - `internal/notes`
  - `internal/validate`
  - `check` read-only and `check --fix` mutating behavior
- Acceptance:
  - `go test ./internal/validate -v`
  - `go test ./internal/notes -v`
- DoD:
  - migration + strict-schema behavior implemented and tested

## F08 - Index + GC + Report Baseline

- Goal: Implement index generation, garbage collection, and metric computation.
- Dependencies: F07
- Scope:
  - `internal/index`, `internal/gc`, `internal/report`
- Acceptance:
  - `go test ./internal/index -v`
  - `go test ./internal/gc -v`
  - `go test ./internal/report -v`
- DoD:
  - deterministic index output + report metric math validated

## F09 - Telemetry with Token Correlation

- Goal: Implement telemetry schema and run/token diagnostics linkage.
- Dependencies: F08, F02
- Scope:
  - `internal/telemetry`
  - fields: `run_id`, `task_id`, `agent_name`, `model`, token/cost fields
  - telemetry_coverage metric support
- Acceptance:
  - `go test ./internal/telemetry -v`
  - report tests with partial telemetry coverage
- DoD:
  - linked telemetry works for optimize and report inputs

## F10 - Context Integration (Repomix)

- Goal: Implement context pack/mcp/budget wrappers.
- Dependencies: F01, F02, F09
- Scope:
  - `internal/context`
  - profile resolution + passthrough args
  - budget enforcement in strict policy
- Acceptance:
  - `go test ./internal/context -v`
- DoD:
  - graceful degradation when repomix missing

## F11 - Security Scan + Doctor

- Goal: Implement tool orchestration diagnostics.
- Dependencies: F01, F02, F09
- Scope:
  - `internal/security`
  - `internal/doctor`
  - report artifacts for `json|sarif`
- Acceptance:
  - `go test ./internal/security -v`
  - `go test ./internal/doctor -v`
- DoD:
  - policy-aware severity handling

## F12 - Policy Gate

- Goal: Implement machine-readable PR/revision policy gating.
- Dependencies: F07, F10, F11
- Scope:
  - `internal/policy`
  - output failures with `policy_id`, `severity`, `summary`, `fix_hint`
- Acceptance:
  - `go test ./internal/policy -v`
- DoD:
  - gate results align with configured required checks

## F13 - Commit Lint + Changelog + Hooks

- Goal: Implement conventional commits tooling and changelog generation.
- Dependencies: F01, F02
- Scope:
  - `internal/commitlint`, `internal/changelog`, `internal/hooks`
- Acceptance:
  - `go test ./internal/commitlint -v`
  - `go test ./internal/changelog -v`
  - `go test ./internal/hooks -v`
- DoD:
  - stable changelog update behavior and hook scaffolding

## F14 - Release Propose / Approve with Gates

- Goal: Implement autonomous release assistant flow with approval gates.
- Dependencies: F12, F13, F05
- Scope:
  - `internal/release`
  - proposal artifact with gate statuses
  - approve execution with staleness check against proposal state
- Acceptance:
  - `go test ./internal/release -v`
- DoD:
  - gated release flow blocks when checks drift/fail

## F15 - Optimize Recommend

- Goal: Implement bounded optimization proposals from telemetry outcomes.
- Dependencies: F09, F10, F11
- Scope:
  - `internal/optimize`
  - recommendation generation only (no auto-apply)
- Acceptance:
  - `go test ./internal/optimize -v`
- DoD:
  - proposals include confidence + projected impact + sample count

## F16 - CLI Wiring and End-to-End Contracts

- Goal: Wire all commands/flags and finalize JSON contracts.
- Dependencies: F01-F15
- Scope:
  - `internal/cli`, `cmd/athena/main.go`
  - completion + global flags + policy propagation
- Acceptance:
  - `go test ./internal/cli -v`
  - `go test ./...`
- DoD:
  - all command reference entries executable with expected outputs

## F17 - Hardening + Golden + Integration Suite

- Goal: Finalize deterministic behavior and regression protection.
- Dependencies: F16
- Scope:
  - golden tests for templates + JSON contracts
  - integration tests across plan/apply/rollback and release gating
- Acceptance:
  - `go test ./...`
  - CI pipeline green with linting
- DoD:
  - stable baseline for iterative evolution

## Recommended Agent Loop Per Feature

1. Read feature spec section(s) and dependencies.
2. Implement minimal viable code for acceptance tests.
3. Add/adjust tests first where behavior is ambiguous.
4. Run package tests, then full test suite.
5. Emit handoff JSON:

```json
{
  "feature_id": "F05",
  "status": "completed",
  "acceptance": ["go test ./internal/execution -v", "go test ./..."],
  "open_risks": []
}
```

## Optional Linear Mapping

If using Linear MCP, create one project `Athena CLI Buildout` and one issue per
feature (`F00`..`F17`) with dependency links matching this document.

Minimum fields:

- title: `F05 - Plan / Apply / Rollback Engine`
- description: copy feature section
- labels: `athena`, `agent-executable`, `phase-sequential`
- parent: `Athena CLI Buildout`
- dependencies: prior feature IDs
