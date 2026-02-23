# Linear Import Notes

## Files

- Issue import CSV: `docs/implementation/linear-import.csv`
- Dependency map: `docs/implementation/linear-dependencies.csv`

## Import Process

1. Create project: `Athena CLI Buildout`.
2. Import `linear-import.csv` using Linear CSV import.
3. Ensure issue titles retain the `F00`..`F17` prefix.
4. Apply dependencies from `linear-dependencies.csv`:
   - `Issue` depends on `DependsOn`.

## MCP-Friendly Linking Strategy

When linking via MCP/API, resolve by prefix in title:

- `F12 - Policy Gate`
- `F07 - Notes + Validation + Schema Evolution`

Then create dependency edge: `F12` blocked by `F07`.

## Recommended Workflow States

- `Backlog` -> `Todo` -> `In Progress` -> `In Review` -> `Done`

## Suggested Labels

- `athena`
- `agent-executable`
- `phase-sequential`
