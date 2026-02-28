---
title: Installation
weight: 1
---

# Installation

## Homebrew (macOS / Linux)

```bash
brew install amr-athena/tap/athena
```

## Download a release

Pre-built binaries for macOS and Linux (amd64 / arm64) are available on the
[GitHub Releases](https://github.com/amr-athena/athena/releases) page.

```bash
# Example: macOS arm64
curl -sL https://github.com/amr-athena/athena/releases/latest/download/athena_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv athena /usr/local/bin/
```

## Build from source

Requires Go 1.25+:

```bash
go install github.com/amr-athena/athena/cmd/athena@latest
```

Or clone and build:

```bash
git clone https://github.com/amr-athena/athena.git
cd athena
go build -o athena ./cmd/athena
sudo mv athena /usr/local/bin/
```

## Verify installation

```bash
athena version
athena capabilities --format json
```

## Shell completion

Generate shell completion scripts:

```bash
# Bash
athena completion bash > /etc/bash_completion.d/athena

# Zsh
athena completion zsh > "${fpath[1]}/_athena"

# Fish
athena completion fish > ~/.config/fish/completions/athena.fish
```

## Next steps

- [Integration Guide]({{< relref "integration-guide" >}}) — set up Athena in your repository
- [Agent Preflight]({{< relref "agent-preflight" >}}) — configure AI agent workflows
