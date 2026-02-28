# Agent Execution Protocol

This is the canonical AI-agent entrypoint for this repository.
If guidance in tool-specific helper files conflicts with this file, follow `AGENTS.md`.

## 1. Start Path (Deterministic)

Before modifying code, follow this path in order:

1. Read `docs/system-context.md` for architecture constraints and operating model.
2. Read `config/README.md` (or equivalent) to choose the ownership boundary.
3. Read stack-specific guides for touched areas (for example `config/<stack>/AGENTS.md`).
4. Read canonical reference docs before changing declared facts.
5. Read `AGENT_TOOLING.md` for preferred local binaries and command patterns.
6. If improving Athena Framework itself, read `AGENT_METRICS.md`.
7. If task is procedural, read the relevant `docs/workflows/` or `docs/guides/` file.
8. If integrating Athena into a repository, read `docs/guides/integration-guide.md`.
9. If implementing Athena CLI features, follow `docs/implementation/START_HERE.md`.

## 2. Development Loop

For each scoped task:

1. Plan: create/update `.ai/memory/plan.md` with touched files and validation commands.
2. Execute: apply minimal edits in the correct ownership boundary.
3. Validate: run stack-appropriate checks before concluding.
4. Document: update canonical docs if declared behavior changed.

## 3. Validation Matrix

Define and maintain task entrypoints per stack and keep them current.
Examples:

- App/unit/integration tests
- Lint/format/static analysis
- Docs integrity checks
- IaC checks (Terraform/Ansible/Nix, etc.)
- AI tooling awareness (`athena doctor`)
- AI memory hygiene (`athena check`)

## 4. Constraints

- Never commit plaintext secrets.
- Respect ownership boundaries between stacks.
- Prefer deterministic and idempotent automation.
- Commits and PRs: follow `CONTRIBUTING.md` for message format, branch naming, and PR conventions.

## 5. Definition of Done

- [ ] Code is idempotent where applicable.
- [ ] Ownership boundaries were respected.
- [ ] Required validation commands were run.
- [ ] Documentation reflects the new declared state.
- [ ] Any manual Day 2 steps are clearly noted.

## 6. Agent Tooling Awareness

- Run `athena doctor` when session tooling availability is unknown.
- Prefer structured CLI tools (`rg`, `jq`, `yq`, `htmlq`, `difft`) over ad-hoc parsing.
- Use fallbacks only when preferred tools are unavailable.
