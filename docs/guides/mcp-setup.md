---
title: MCP Setup
weight: 3
---

# MCP Server Setup

Athena includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that exposes working memory tools and resources over stdio. This lets IDE agents (Claude Code, Cursor, Windsurf, Zed) interact with Athena directly — no shell spawning needed.

## Prerequisites

1. Install the `athena` binary ([Installation]({{< relref "installation" >}}))
2. Initialise your repository: `athena init --preset standard`

## Configuration

### Claude Code

Add to `.claude/settings.json` in your project root (or `~/.claude/settings.json` for global):

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

### Cursor

Add to `.cursor/mcp.json` in your project root:

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

### Windsurf

Add to `~/.codeium/windsurf/mcp_config.json`:

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

### Generic MCP client

Any MCP-compatible client that supports stdio transport can connect:

```json
{
  "command": "athena",
  "args": ["mcp"],
  "transport": "stdio"
}
```

## What the MCP server exposes

### Resources (read-only)

| URI | Description |
|-----|-------------|
| `athena://capabilities` | Command inventory and schema versions |
| `athena://config` | Current `athena.toml` configuration |
| `athena://notes` | All notes with frontmatter metadata |
| `athena://index` | Note index (`.ai/index.yaml`) |
| `athena://report` | Effectiveness metrics |

### Tools

#### Notes & memory

| Tool | Description | Mutates? |
|------|-------------|----------|
| `note_new` | Create a note from template | Yes |
| `note_close` | Transition note to terminal status | Yes |
| `note_promote` | Promote note to canonical docs | Yes |
| `note_read` | Read a note's full content | No |
| `note_list` | List notes with status/type filters | No |
| `context_search` | BM25 full-text search over notes | No |

#### Validation & diagnostics

| Tool | Description | Mutates? |
|------|-------------|----------|
| `check` | Validate frontmatter and schema | No |
| `check_fix` | Validate and auto-fix schema issues | Yes |
| `index_rebuild` | Rebuild note index and search index | Yes |
| `gc_scan` | Identify stale notes past inactivity threshold | No |
| `doctor` | Run repository diagnostics | No |
| `report` | Compute effectiveness metrics | No |

#### Governance

| Tool | Description | Mutates? |
|------|-------------|----------|
| `policy_gate` | Run policy gate checks with per-check pass/fail | No |
| `commit_lint` | Validate a commit message against conventional commit rules | No |
| `security_scan` | Run secret detection and workflow lint checks | No |

#### Context management

| Tool | Description | Mutates? |
|------|-------------|----------|
| `context_pack` | Generate a context bundle using a configured repomix profile | No |
| `context_budget` | Estimate token count and check against a budget threshold | No |

## Verify it works

After configuring, restart your IDE agent and check that Athena tools appear in the MCP tool list. You can also test from the command line:

```bash
# The MCP server starts and communicates over stdio.
# This is primarily for IDE integration — you won't see
# output when running interactively.
athena mcp
```

## Next steps

- [Integration Guide]({{< relref "integration-guide" >}}) — full workflow reference
- [Agent Preflight]({{< relref "agent-preflight" >}}) — session startup checklist
