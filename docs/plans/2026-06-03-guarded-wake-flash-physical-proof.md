# T-WAKE-INTEGRATE-001 Custom Wake In StackChan-Compatible App Plan

Status: active plan, revised after bare-flash incident
Date: 2026-06-03
Owner: A21 control tower
Transition: `T-WAKE-INTEGRATE-001-CUSTOM-WAKE-IN-STACKCHAN-COMPATIBLE-APP`

## Background And Problem Definition

A21 has custom wake-word package evidence, but the live physical firmware still
uses stock Xiaozhi WakeNet/touch activation. The operator reports wake is weak
or absent. Product readiness must not claim `wake_word.product_ready=true` until
custom wake is integrated into the StackChan-compatible product app, flashed
through the correct product lane, and physically observed.

On 2026-06-03 a foreground hardware window flashed the restored custom wake
artifact through the generic `xiaozhi-firmware-flash-execute` lane. The flash
technically passed, but the app part was `xiaozhi.bin`; the physical device UI
regressed to plain Xiaozhi instead of preserving the StackChan avatar/product
experience. The device was immediately recovered by flashing
`a21-stackchan-official-xiaozhi-compatible.bin` through the correct lane. This
plan now treats bare `xiaozhi.bin` as invalid for product StackChan flashes.

## Current System State

- Current baseline: `b5a405e`.
- Package report:
  `reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json`.
- Artifact:
  `reports/a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-a2d3dc882b42-20260602-075112.bin`.
- Package state is below activation: `product_ready=false`,
  `flash_allowed=false`, `flash_executed=false`.
- Bare custom wake flash report:
  `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json`.
  This is incident evidence only, not product acceptance.
- Corrective StackChan-compatible flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`.
  This restored the accepted product app SHA-256
  `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.
- Current physical firmware is back on the StackChan-compatible app and has not
  integrated custom wake.

## Target State

`S3-CUSTOM-WAKE-IN-STACKCHAN-COMPATIBLE-NO-WRITE`

- Custom wake assets are rebuilt or ported into
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- A no-write flash/runbook proves exact product artifact, port, identity, and
  app lane.
- Only after explicit operator confirmation, a guarded foreground flash may run.
- Physical proof report confirms custom wake phrase matched, false wakes
  rejected, and stock wake rejected.

## Non-Goals

- Do not flash automatically from this plan.
- Do not use `xiaozhi-firmware-flash-execute` or app file `xiaozhi.bin` on the
  product StackChan device.
- Do not alter provider/TTS/half-duplex code.
- Do not claim custom wake readiness from Gateway `/v1/wake-word` config alone.
- Do not overwrite Wi-Fi/NVS calibration accidentally.

## Impact Scope

- Wake-word package/report validation.
- StackChan-compatible product app build/report.
- Guarded official-compatible flash plan/report.
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

### Phase 2: StackChan-Compatible Wake Integration

Actions:

- Port or rebuild the custom wake Kconfig/assets into the official-compatible
  StackChan app lane.
- Verify the build report still shows official StackChan behavior preserved.

Acceptance:

- App file is `a21-stackchan-official-xiaozhi-compatible.bin`, not `xiaozhi.bin`.
- Build evidence preserves `AppAvatar`, `AppAiAgent`, and
  `GetHAL().startXiaozhi()` behavior.

### Phase 3: Official-Compatible No-Write Flash Gate

Actions:

- Run only `a21-stackchan-official-xiaozhi-compatible-flash-plan`.
- Confirm app part at `0x20000` is
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Confirm no-write report is ready before requesting foreground execution.

Acceptance:

- Dry-run report is `ready` or records exact blocker.
- `flash_allowed=false` and `flash_executed=false` until operator confirmation.
- No runbook references `xiaozhi-firmware-flash-execute` for the product
  StackChan device.

### Phase 4: Foreground Flash Window

Actions:

- Only after operator confirms port and write token, run the guarded flash
  command.
- Verify Gateway reconnect and device identity after reboot.

Acceptance:

- Guarded flash report is `status=passed`, `flash_allowed=true`,
  `flash_executed=true`.
- Device reconnects to A21 Gateway.

### Phase 5: Physical Wake Proof

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
  fails or device behavior regresses. Current rollback report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`.
- Preserve NVS connection settings unless the rollback plan explicitly requires
  reprovisioning.

## Risks

- Wrong flash artifact can regress the accepted audio path and avatar surface;
  the 2026-06-03 `xiaozhi.bin` incident already proved this risk.
- Wake package may not include the latest accepted audio/Gateway behavior.
- Physical proof depends on environment noise and operator phrase consistency.

## Human Confirmation Points

- Upload port and write confirmation token before any flash.
- Operator listening/wake observation after reboot.
- Whether to roll back if stock wake behavior changes unexpectedly.

## Worker Execution Task

Worker owns only StackChan-compatible wake integration prep and no-write gate
prep unless operator explicitly provides guarded flash confirmation.

Allowed:

- Read package/artifact/manifest reports.
- Run no-write validation/report commands.
- Prepare exact foreground flash command for operator.

Forbidden:

- Flashing without explicit confirmation.
- Using `xiaozhi-firmware-flash-execute` or `xiaozhi.bin` as the product
  StackChan flash artifact.
- NVS writes.
- Provider/TTS/half-duplex changes.
- Claiming product-ready from package-only evidence.

Return summary format:

- Package/artifact integrity.
- StackChan-compatible build report and app SHA.
- Dry-run/no-write official-compatible flash report path.
- Exact guarded flash command if ready.
- Blockers.
- Required operator action.
