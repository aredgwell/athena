# Config Ownership Boundary

Primary ownership boundary:

- `cmd/athena` and `internal/*`: Athena CLI product code

Supporting boundaries:

- `docs/*`: specifications, runbooks, implementation plans

## Rule

All runtime product logic lives in Go under `cmd/` and `internal/`.
