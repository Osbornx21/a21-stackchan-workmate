# A21 Phase 3B Doctor Reports Implementation Plan

**Goal:** Make `a21 doctor` persist a timestamped JSON report so network/proxy/port failures can be compared across home and Shanghai office environments.

## Scope

Phase 3B implements:

- `a21 doctor --output-dir <dir>`
- default `reports/` output directory
- `a21-doctor-YYYYMMDD-HHMMSS.json` file naming
- tests for file creation and invalid flags
- doctor docs update

Phase 3B does not implement:

- human-readable table output
- exit code 2 for warnings-only
- provider connectivity checks
- StackChan ping or mDNS checks
- v21 adapter health checks

## Tasks

- [x] Add failing doctor report tests.
- [x] Implement doctor flag parsing and JSON file writing.
- [x] Update docs.
- [x] Run full verification.
- [x] Commit as `feat: persist a21 doctor reports`.
