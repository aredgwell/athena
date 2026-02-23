# Athena Stack Guide

## Scope

This guide applies to implementation under:

- `cmd/athena`
- `internal/*`

## Working Rules

- Implement against `ATHENA.md` command and schema contracts.
- Keep command behavior machine-readable first (`--format json`).
- Ensure idempotency and testability for mutating operations.
- Add package-level tests for all new behavior.

## Validation Baseline

- `go test ./...`
- command/package tests listed in `ATHENA.md` command reference
