# System Context

## Purpose

This repository defines and implements the Athena CLI, a Go-based replacement
for shell-script AI memory tooling.

## Current State

- Canonical product specification: `ATHENA.md`
- Sequential implementation plan: `docs/implementation/SEQUENTIAL_FEATURE_PLAN.md`
- Agent execution protocol: `AGENTS.md`
- Legacy shell tooling (reference behavior): `scripts/ai/*`

## Architecture Direction

- Language: Go 1.23+
- CLI framework: Cobra + Viper
- Config file: `athena.toml`
- Primary implementation root: `cmd/athena`, `internal/*`

## Constraints

- Keep behavior deterministic and idempotent.
- Preserve data safety for upgrades/migrations through backups + checksums.
- Prefer machine-readable contracts for agent workflows.

## Sources of Truth

1. `ATHENA.md`
2. `AGENTS.md`
3. `docs/implementation/SEQUENTIAL_FEATURE_PLAN.md`
