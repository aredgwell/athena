# Agent Preflight

Run before implementation:

```bash
task ai:tools:check
task ai:check
```

If `task ai:check` fails due to missing baseline `.ai` notes, create/update:

- `.ai/README.md`
- `.ai/context/session_state.md`
- `.ai/memory/plan.md`
