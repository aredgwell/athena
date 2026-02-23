# Config Ownership Boundary

This repository currently has one primary ownership boundary for implementation:

- `cmd/athena` and `internal/*`: Athena CLI product code

Supporting boundaries:

- `docs/*`: specifications, runbooks, implementation plans
- `scripts/ai/*`: legacy shell behavior reference only

## Rule

New runtime product logic should be implemented in Go under `cmd/` and
`internal/` unless explicitly scoped as legacy compatibility script behavior.
