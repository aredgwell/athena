# Athena CLI

Repo-local AI working memory scaffolding, lifecycle management, and governance checks.

## Install

```bash
# Homebrew
brew install amr-athena/tap/athena

# Or from source (Go 1.25+)
go install github.com/amr-athena/athena/cmd/athena@latest
```

## Quick Start

```bash
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

- [Full docs](https://amr-athena.github.io/athena/) — specification, guides, and reference
- [ATHENA.md](ATHENA.md) — product specification
- [AGENTS.md](AGENTS.md) — agent execution protocol
- [CONTRIBUTING.md](CONTRIBUTING.md) — contribution guidelines

## Test

```bash
go test ./...        # unit + package tests
go test ./e2e/...    # end-to-end binary tests
```
