# Athena CLI Sequential Implementation Plan

Features F00-F17, F19, and F20 are implemented and passing (`go test ./...`). F18 is deferred.

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
| F19 | MCP Server Mode | `internal/mcp` |
| F20 | Content Search | `internal/search`, `internal/index`, `internal/cli`, `internal/mcp` |

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

## F19 - MCP Server Mode (Implemented)

- Uses official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk` v1.4.0)
- 5 resources (capabilities, config, notes, index, report)
- 12 tools (note_new, note_close, note_promote, note_read, note_list, check, check_fix, index_rebuild, gc_scan, doctor, report, context_search)
- Transport: stdio only

## F20 - Content Search (Implemented)

- BM25 lexical search over note contents (`internal/search` package)
- Search index built during `athena index`, stored at `.ai/search-index.json`
- `athena context search "query" [--limit N]` CLI command
- `context_search` MCP tool
- Pure Go, no external dependencies
- Title tokens weighted 3x, tag tokens 2x for relevance boosting
