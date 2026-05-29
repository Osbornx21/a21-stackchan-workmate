# A21 Firmware Flash Plan Guard Implementation Plan

**Goal:** Add a no-flash `firmware-flash-plan` guard that combines the existing artifact, upload-port, and Gateway device identity checks into one receipt. This is still not a flashing command.

**Architecture:** Reuse `internal/firmwarecheck.ValidateUploadCandidate` and `ValidateDeviceIdentity`. The new plan only composes existing guards and checks that port/device/artifact/commit agree; it must not call PlatformIO upload or any serial write path.

**Safety:** The result must always include `flash_allowed:false`, `dry_run:true`, and an A21 guard ID. X21/V21 identity rejection remains delegated to the existing guards.

## Steps

- [x] Add failing firmwarecheck tests for a successful no-flash plan and busy-port rejection.
- [x] Add failing CLI test for `a21 firmware-flash-plan`.
- [x] Implement `BuildFlashPlan` as guard composition, not a new flash path.
- [x] Wire the CLI and Makefile target.
- [x] Update firmware release discipline docs.
- [ ] Run full verification, release-check, artifact-check, upload dry-run, and commit.
