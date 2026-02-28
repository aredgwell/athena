# Agent Preflight

Run before implementation:

```bash
athena doctor
athena check
```

If `athena check` fails due to missing baseline `.ai` notes, create/update:

- `.ai/README.md`
- `.ai/context/session_state.md`
- `.ai/memory/plan.md`
