# Start Here (Agent)

Follow these steps exactly before implementing features.

## 1. Read Order

1. `AGENTS.md`
2. `docs/system-context.md`
3. `config/README.md`
4. `config/athena/AGENTS.md`
5. `ATHENA.md`
6. `docs/implementation/SEQUENTIAL_FEATURE_PLAN.md`

## 2. Preflight Commands

```bash
athena doctor
athena check
```

## 3. Execution Model

- Implement feature-by-feature in `docs/implementation/SEQUENTIAL_FEATURE_PLAN.md`.
- Do not skip dependencies.
- For each feature:
  - update `.ai/memory/plan.md`
  - run feature acceptance tests
  - run `go test ./...`

## 4. Completion Handoff

Write `.athena/feature-handoffs/<feature-id>.json` after each feature.
