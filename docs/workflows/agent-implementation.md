# Agent Implementation Workflow

## Start

1. Read `AGENTS.md`
2. Read `docs/system-context.md`
3. Read `config/README.md`
4. Read `config/athena/AGENTS.md`
5. Read `ATHENA.md`
6. Read `docs/implementation/SEQUENTIAL_FEATURE_PLAN.md`

## Per Feature

1. Pick next uncompleted feature (`F00`..`F17`)
2. Update `.ai/memory/plan.md`
3. Implement minimal passing behavior
4. Run feature acceptance tests
5. Run `go test ./...`
6. Write handoff to `.athena/feature-handoffs/<feature-id>.json`
