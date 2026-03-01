---
title: Athena CLI
type: docs
---

# Athena CLI

AI agents lose context between sessions. Athena gives them structured, persistent working memory inside your repository — a `.ai/` directory of Markdown notes with YAML frontmatter, lifecycle management, BM25 search, and governance gates.

{{< button relref="/docs/guides/integration-guide" >}}Integration Guide{{< /button >}}
{{< button relref="/docs/guides/mcp-setup" >}}MCP Setup{{< /button >}}
{{< button relref="/docs/specification" >}}Specification{{< /button >}}

## Quick Start

```sh
# Install (Go 1.25+)
go install github.com/amr-athena/athena/cmd/athena@latest

# Bootstrap a repository
athena init --preset standard

# Verify
athena doctor
athena check
```

## MCP Server

Connect your IDE agent directly to Athena via [Model Context Protocol](https://modelcontextprotocol.io/). See the [MCP Setup Guide]({{< relref "/docs/guides/mcp-setup" >}}) for configuration.

## Documentation

Browse the sidebar to explore guides, reference material, and the full specification.
