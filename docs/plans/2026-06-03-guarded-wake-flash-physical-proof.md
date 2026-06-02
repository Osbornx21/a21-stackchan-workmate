# T-FW-003 Guarded Custom Wake Flash And Physical Proof Plan

Status: active plan
Date: 2026-06-03
Owner: A21 control tower
Transition: `T-FW-003-GUARDED-CUSTOM-WAKE-FLASH-AND-PHYSICAL-PROOF`

## Background And Problem Definition

A21 has custom wake-word package evidence, but the live physical firmware still
uses stock Xiaozhi WakeNet/touch activation. The operator reports wake is weak
or absent. Product readiness must not claim `wake_word.product_ready=true` until
a guarded custom wake package is flashed and physically observed.

## Current System State

- Current baseline: `b5a405e`.
- Package report:
  `reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json`.
- Artifact:
  `reports/a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-a2d3dc882b42-20260602-075112.bin`.
- Package state is below activation: `product_ready=false`,
  `flash_allowed=false`, `flash_executed=false`.
- Current physical firmware has not flashed this package.

## Target State

`S3-GUARDED-WAKE-FLASH-AND-PHYSICAL-PROOF`

- A no-write flash/runbook proves exact package/artifact/port/identity.
- Only after explicit operator confirmation, a guarded foreground flash may run.
- Physical proof report confirms custom wake phrase matched, false wakes
  rejected, and stock wake rejected.

## Non-Goals

- Do not flash automatically from this plan.
- Do not alter provider/TTS/half-duplex code.
- Do not claim custom wake readiness from Gateway `/v1/wake-word` config alone.
- Do not overwrite Wi-Fi/NVS calibration accidentally.

## Impact Scope

- Wake-word package/report validation.
- Guarded flash plan/report.
- Physical proof and wake-word physical acceptance reports.
- Product-readiness wake-word evidence.

## Phased Execution

### Phase 1: Package Integrity Review

Actions:

- Confirm package report exists and names the expected artifact, manifest, and
  SHA-256.
- Confirm artifact exists and checksum matches.
- Confirm package is below activation and not already product-ready.

Acceptance:

- Integrity review summary lists package report, artifact basename, SHA-256,
  and current blocked state.

### Phase 2: No-Write Flash Gate

Actions:

- Prepare or locate the guarded flash command for the exact package/artifact.
- Run only no-write/dry-run validation unless the operator supplies explicit
  write confirmation in the foreground.

Acceptance:

- Dry-run report is `ready` or records exact blocker.
- `flash_allowed=false` and `flash_executed=false` until operator confirmation.

### Phase 3: Foreground Flash Window

Actions:

- Only after operator confirms port and write token, run the guarded flash
  command.
- Verify Gateway reconnect and device identity after reboot.

Acceptance:

- Guarded flash report is `status=passed`, `flash_allowed=true`,
  `flash_executed=true`.
- Device reconnects to A21 Gateway.

### Phase 4: Physical Wake Proof

Actions:

- Operator says the custom wake phrase multiple times.
- Record whether custom wake activates, random/false wake is rejected, and
  stock wake no longer triggers if that is part of the package intent.
- Run `wake-word-physical-proof` and `wake-word-physical-acceptance`.

Acceptance:

- `wake_word_physical_acceptance.v1` has `product_ready=true`.
- Product readiness closes only wake-word server-side gap; full launch may
  still be blocked by provider/half-duplex.

## Rollback

- Reflash last accepted official-compatible app artifact if custom wake flash
  fails or device behavior regresses.
- Preserve NVS connection settings unless the rollback plan explicitly requires
  reprovisioning.

## Risks

- Wrong flash artifact can regress the accepted audio path.
- Wake package may not include the latest accepted audio/Gateway behavior.
- Physical proof depends on environment noise and operator phrase consistency.

## Human Confirmation Points

- Upload port and write confirmation token before any flash.
- Operator listening/wake observation after reboot.
- Whether to roll back if stock wake behavior changes unexpectedly.

## Worker Execution Task

Worker owns only package integrity and no-write gate prep unless operator
explicitly provides guarded flash confirmation.

Allowed:

- Read package/artifact/manifest reports.
- Run no-write validation/report commands.
- Prepare exact foreground flash command for operator.

Forbidden:

- Flashing without explicit confirmation.
- NVS writes.
- Provider/TTS/half-duplex changes.
- Claiming product-ready from package-only evidence.

Return summary format:

- Package/artifact integrity.
- Dry-run/no-write report path.
- Exact guarded flash command if ready.
- Blockers.
- Required operator action.
