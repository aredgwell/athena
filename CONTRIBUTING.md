# Contributing

Contribution standards for this repository. These conventions apply to both human
contributors and AI agents. For the canonical AI agent execution protocol, see
`AGENTS.md`.

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

```text
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | When to use |
| --- | --- |
| `feat` | New capability or resource |
| `fix` | Bug fix or correction |
| `chore` | Maintenance (dependencies, tooling, CI) |
| `refactor` | Code restructure with no behaviour change |
| `docs` | Documentation only |
| `test` | Adding or updating tests |

### Scopes

Use the ownership boundary or component name. Define scopes that match your
repository structure. Examples:

| Scope | Area |
| --- | --- |
| `app` | Application code |
| `api` | API layer |
| `infra` | Infrastructure configuration |
| `docs` | Documentation |
| `ci` | CI/CD pipelines and workflows |
| `scripts` | Helper scripts |
| `meta` | Repository-level config (AGENTS.md, Taskfile, .editorconfig, etc.) |

### Examples

```text
feat(app): add user authentication flow
fix(api): correct pagination offset calculation
chore(ci): update GitHub Actions runner version
docs: add contributing guidelines
refactor(app): extract validation into shared module
```

### Rules

- Use imperative mood ("add", not "added" or "adds").
- Keep the subject line under 72 characters.
- Reference issue tracker IDs in the footer when applicable: `Refs: PROJECT-123`.
- Breaking changes: add `BREAKING CHANGE:` in the footer or `!` after the type.

## Branch Naming

```text
<type>/<scope>-<short-description>
```

Examples:

```text
feat/app-user-authentication
fix/api-pagination-offset
chore/ci-runner-update
docs/contributing-guidelines
```

## What Must Pass Before Merge

Run the validation checks that match the areas you touched. All must pass.
Refer to the **Validation Matrix** in `AGENTS.md` for the definitive list of
check commands per stack.

## Pull Request Description

Use this structure for PR descriptions:

```markdown
## Summary

Brief description of what changed and why.

## Changes

- Bullet list of specific changes

## Validation

- [ ] Relevant validation commands pass
- [ ] Lint/format checks pass
- [ ] Documentation updated (if behaviour changed)
- [ ] CHANGELOG.md updated (if user-facing change)
```

## Changelog

Maintain `CHANGELOG.md` using [Keep a Changelog](https://keepachangelog.com/)
format. Update the `[Unreleased]` section when making user-facing changes.

Use these categories:

- **Added** — new features or resources
- **Changed** — changes to existing behaviour
- **Fixed** — bug fixes
- **Removed** — removed features or resources

Do not add entries for internal refactors, CI changes, or dependency bumps unless
they affect user-facing behaviour.

## Release Process

1. Review the `[Unreleased]` section in `CHANGELOG.md`.
2. Choose a version number following [Semantic Versioning](https://semver.org/):
   - **Major** (`X.0.0`): breaking changes to public contracts
   - **Minor** (`0.X.0`): new capabilities or features
   - **Patch** (`0.0.X`): fixes, documentation corrections
3. Replace `[Unreleased]` with `[X.Y.Z] - YYYY-MM-DD` and add a fresh
   `[Unreleased]` heading above it.
4. Commit: `chore(meta): release vX.Y.Z`.
5. Tag: `git tag vX.Y.Z`.
6. Push: `git push && git push --tags`.
