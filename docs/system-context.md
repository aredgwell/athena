# System Context

## Purpose

This repository implements the Athena CLI — a Go-based scaffolder and lifecycle
manager for AI-native repository workflows.

## Architecture

- Language: Go 1.23+
- CLI framework: Cobra + Viper
- Config file: `athena.toml`
- Implementation: `cmd/athena`, `internal/*`

## Constraints

- Deterministic and idempotent behavior.
- Data safety for upgrades/migrations via backups + checksums.
- Machine-readable contracts for agent workflows.

## Sources of Truth

1. `ATHENA.md` — product specification
2. `AGENTS.md` — agent execution protocol
3. `docs/implementation/SEQUENTIAL_FEATURE_PLAN.md` — implementation plan
