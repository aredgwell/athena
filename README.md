# Athena CLI

Repo-local AI working memory scaffolding, lifecycle management, and governance checks.

## Quick Start

```bash
go build -o athena ./cmd/athena
athena init --preset standard
athena doctor
athena check
```

## What It Does

- **Scaffolds** the `.ai/` directory tree with structured note templates
- **Manages** note lifecycle: create, close, promote, mark stale
- **Validates** YAML frontmatter with schema migration support
- **Queries** notes by component, type, and status for agent onboarding
- **Gates** PRs against configurable policy checks
- **Emits** machine-readable JSON for CI and agent automation

## Documentation

- [ATHENA.md](ATHENA.md) — full product specification
- [AGENTS.md](AGENTS.md) — agent execution protocol
- [CONTRIBUTING.md](CONTRIBUTING.md) — contribution guidelines

## Test

```bash
go test ./...
```
