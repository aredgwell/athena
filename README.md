# Athena CLI

AI agents lose context between sessions. Athena gives them structured, persistent working memory inside your repository.

Athena manages a `.ai/` directory of Markdown notes with YAML frontmatter — creating, validating, indexing, searching, and retiring knowledge so agents can pick up where they left off. It also provides governance (policy gates, commit linting, security scans) and a built-in MCP server for direct IDE integration.

## Install

```bash
# From source (Go 1.25+)
go install github.com/amr-athena/athena/cmd/athena@latest
```

Pre-built binaries are also available on the [Releases](https://github.com/aredgwell/athena/releases) page.

## Quick Start

```bash
athena init --preset standard   # scaffold .ai/ and config files
athena doctor                    # verify toolchain health
athena check                     # validate working memory
```

## What It Does

- **Scaffolds** a `.ai/` directory tree — persistent working memory that lives in your repo alongside your code
- **Manages** note lifecycle: create, close, promote, mark stale
- **Searches** notes with BM25 full-text search (Porter stemming, fuzzy matching, snippets)
- **Validates** YAML frontmatter with schema migration support
- **Queries** notes by component, type, and status for agent onboarding
- **Gates** PRs against configurable policy checks
- **Exposes** an MCP server (`athena mcp`) for IDE agents — Claude Code, Cursor, Windsurf
- **Emits** machine-readable JSON for CI and agent automation

## MCP Server

Athena includes a [Model Context Protocol](https://modelcontextprotocol.io/) server for IDE agents. Add to your `.claude/settings.json` (Claude Code) or `.cursor/mcp.json` (Cursor):

```json
{
  "mcpServers": {
    "athena": {
      "command": "athena",
      "args": ["mcp"]
    }
  }
}
```

See the [MCP Setup Guide](docs/guides/mcp-setup.md) for Windsurf and other clients.

## Documentation

- [Full docs](https://aredgwell.github.io/athena/) — specification, guides, and reference
- [MCP Setup](docs/guides/mcp-setup.md) — configure IDE agent integration
- [Integration Guide](docs/guides/integration-guide.md) — full workflow reference
- [ATHENA.md](ATHENA.md) — product specification
- [AGENTS.md](AGENTS.md) — agent execution protocol
- [CONTRIBUTING.md](CONTRIBUTING.md) — contribution guidelines

## Test

```bash
go test ./...        # unit + package tests
go test ./e2e/...    # end-to-end binary tests
```
