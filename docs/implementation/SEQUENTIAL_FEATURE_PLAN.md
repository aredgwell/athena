# Athena CLI Sequential Implementation Plan

Features F00-F17 are implemented and passing (`go test ./...`). F18 is deferred.

## Completed Features

| Feature | Description | Package(s) |
|---------|-------------|------------|
| F00 | Repository Bootstrap | `cmd/athena`, `internal/` skeleton |
| F01 | Core Config and Contracts | `internal/config` |
| F02 | Error Taxonomy + JSON Envelope | `internal/errors` |
| F03 | Capabilities Command | `internal/capabilities` |
| F04 | Locking + Journaling Foundation | `internal/lock`, `internal/execution` |
| F05 | Plan / Apply / Rollback Engine | `internal/execution` |
| F06 | Scaffold Init + Upgrade | `internal/scaffold` |
| F07 | Notes + Validation + Schema Evolution | `internal/notes`, `internal/validate` |
| F08 | Index + GC + Report Baseline | `internal/index`, `internal/gc`, `internal/report` |
| F09 | Telemetry with Token Correlation | `internal/telemetry` |
| F10 | Context Integration (Repomix + Query) | `internal/context` |
| F11 | Security Scan + Doctor | `internal/security`, `internal/doctor` |
| F12 | Policy Gate | `internal/policy` |
| F13 | Commit Lint + Changelog + Hooks | `internal/commitlint`, `internal/changelog`, `internal/hooks` |
| F14 | Release Propose / Approve with Gates | `internal/release` |
| F15 | Optimize Recommend | `internal/optimize` |
| F16 | CLI Wiring and End-to-End Contracts | `internal/cli`, `cmd/athena/main.go` |
| F17 | Hardening + Golden + Integration Suite | `internal/cli` (integration tests) |

## F18 - Linear Integration (Future)

- Goal: Outbound issue creation from policy gate failures and report outputs.
- Dependencies: F12, F08
- Scope:
  - `internal/integrations/linear`
  - Policy gate failures → create/update Linear issues with failure details and fix hints
  - Report outputs (stale notes, promotion candidates) → create Linear backlog issues
  - Configuration via `[integrations.linear]` in `athena.toml` (API key via env var)
- Status: **Deferred**
- Design notes:
  - Outbound only (Athena → Linear); Linear remains source of truth for work tracking
  - Athena is a knowledge layer, not an issue tracker

## F19 - MCP Server Mode (Future)

- Goal: Expose Athena commands as Model Context Protocol tools/resources over stdio.
- Dependencies: F16
- Scope:
  - `athena mcp` command that starts a local MCP server over stdio
  - Read-only commands (`context query`, `context timeline`, `capabilities`, `note list`, `index`) → MCP Resources
  - Mutating commands (`note new`, `note close`, `note promote`, `check --fix`) → MCP Tools
  - Evaluate Go MCP libraries (e.g. `github.com/mark3labs/mcp-go`)
- Status: **Deferred**
- Design notes:
  - Eliminates shell-spawning overhead for IDE agents (Cursor, Windsurf)
  - Transport: stdio only (no HTTP server)
  - Must preserve identical semantics to CLI invocation

## F20 - Content Search (Future)

- Goal: Lightweight lexical search over note contents for agent onboarding.
- Dependencies: F08
- Scope:
  - `athena context search "query" [--limit N]`
  - BM25 or TF-IDF index built during `athena index`
  - Index artifact stored alongside `.ai/index.yaml`
  - Pure Go implementation (no external dependencies)
- Status: **Deferred**
- Design notes:
  - Fills the gap between metadata query (`context query --component X`) and full-text grep
  - Agents often have a concept ("auth middleware") but not the exact component name
  - Returns ranked results with title, path, and relevance score
