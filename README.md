# Athena CLI

A portable, schema-driven scaffolder and lifecycle manager for AI-native
repository workflows.

## Prerequisites

- Go 1.25+
- [Task](https://taskfile.dev) (optional, for task runner commands)

## Quick Start

```sh
# Build
task build          # or: go build -o bin/athena ./cmd/athena

# Run tests
task test           # or: go test ./...

# Install to $GOPATH/bin
task install        # or: go install ./cmd/athena
```

## Task Runner Commands

| Command             | Description                                |
|---------------------|--------------------------------------------|
| `task build`        | Build the `athena` binary to `./bin/`      |
| `task test`         | Run all tests                              |
| `task test:verbose` | Run all tests with verbose output          |
| `task test:cover`   | Run tests and print coverage summary       |
| `task test:cover:html` | Open coverage report in browser         |
| `task lint`         | Run `go vet`                               |
| `task fmt`          | Format all Go source files                 |
| `task tidy`         | Tidy and verify `go.mod`                   |
| `task clean`        | Remove build artifacts and coverage files  |
| `task check`        | Run fmt + lint + tests (CI entrypoint)     |
| `task install`      | Install `athena` to `$GOPATH/bin`          |

AI-specific tasks are namespaced under `ai:` (e.g. `task ai:check`, `task ai:gc`).

## Project Structure

```
cmd/athena/          CLI entrypoint
internal/
  cli/               Cobra command wiring and JSON envelope
  capabilities/      Supported commands and schema versions
  changelog/         Conventional commit changelog generation
  commitlint/        Commit message parsing and linting
  config/            TOML configuration loading
  context/           Repomix context packing and MCP integration
  doctor/            Repository health diagnostics
  errors/            Structured error taxonomy
  execution/         Plan/apply/rollback with journaling
  gc/                Stale note garbage collection
  hooks/             Pre-commit hook installation
  index/             Note index builder
  lock/              Atomic file-based locking
  notes/             Note lifecycle (new, close, promote, list)
  optimize/          Telemetry-driven tuning recommendations
  policy/            PR/revision policy gate evaluation
  release/           Release proposal and approval workflow
  report/            Memory effectiveness metrics
  scaffold/          Init/upgrade with checksums and backups
  security/          Gitleaks/actionlint security scanning
  telemetry/         Usage telemetry with token correlation
  validate/          Frontmatter validation and schema migration
```

## Documentation

| Document | Purpose |
|----------|---------|
| [Integration Guide](docs/guides/integration-guide.md) | How to use Athena for maximum AI-agent effectiveness |
| [ATHENA.md](ATHENA.md) | Full product specification |
| [AGENTS.md](AGENTS.md) | Canonical agent execution protocol |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Commit style, branch naming, PR conventions |

## Configuration

Athena reads `athena.toml` from the repository root. See [ATHENA.md](ATHENA.md)
for the full specification.

## License

See [LICENSE](LICENSE).
