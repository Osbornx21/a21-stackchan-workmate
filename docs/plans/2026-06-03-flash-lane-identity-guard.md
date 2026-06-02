# T-FLASH-GUARD-001 Product Flash Lane Identity Guard

Status: completed
Date: 2026-06-03
Owner: A21 control tower
Transition: `T-FLASH-GUARD-001-PRODUCT-FLASH-LANE-IDENTITY-GUARD`

## Background And Problem Definition

A foreground hardware window accidentally used the generic
`xiaozhi-firmware-flash-execute` lane to flash a custom wake artifact whose app
part was `xiaozhi.bin`. The command passed generic T7 controls but reverted the
physical product device to the plain Xiaozhi UI. The correct StackChan product
lane is `a21-stackchan-official-xiaozhi-compatible-flash-execute`, whose app
part is `a21-stackchan-official-xiaozhi-compatible.bin`.

## Current System State

- Bad incident evidence:
  `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json`.
- Corrective recovery evidence:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`.
- Gateway/device recovery evidence:
  `a21-trace-recovery-stackchan-volume-1780424386012` and
  `a21-trace-recovery-stackchan-relay-wav-1780424386012`.

## Target State

`S-FLASH-LANE-GUARD-INTEGRATED`

- Generic bare Xiaozhi flash cannot be mistaken for a StackChan product flash.
- Product StackChan flash reports and runbooks must use the official-compatible
  schema, command, confirm token, and app filename.

## Completion Evidence

- Generic bare Xiaozhi no-write plan against the incident build now fails
  before any hardware write with:
  `xiaozhi-firmware-flash is not a product StackChan flash lane for
  xiaozhi.bin at 0x20000`.
- Correct official-compatible no-write plan still passes:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-023008-1780425008542994000.json`.
- Focused tests passed:
  `go test ./internal/app -run 'TestRunXiaozhiFirmwareFlash|TestRunStackChanOfficialXiaozhiCompatibleFlash|TestCollectOfficialStackChanBuildArtifactsFindsXiaozhiCompatibleAppFromFlashArgs|TestRunWakeWordFirmware(BuildReceipt|Package)' -count=1`.
- `make verify` passed.

## Non-Goals

- Do not block non-product package integrity or no-write wake evidence.
- Do not run hardware flash or NVS writes from this worker transition.
- Do not change provider, TTS, Gateway audio, or wake implementation logic.
- Do not remove the future custom wake integration path; only guard the wrong
  product flash lane.

## Impact Scope

- Flash command validation and focused tests only.
- Documentation/state handoff for the incident.

## Phased Execution

### Phase 1: Guard Behavior

Actions:

- Add the smallest validation that rejects or clearly fails a product
  StackChan flow when the app part is `xiaozhi.bin` or the command/schema is
  the generic bare Xiaozhi flash lane.
- Error text must direct operators to
  `a21-stackchan-official-xiaozhi-compatible-flash-execute` for the product
  StackChan app.

Acceptance:

- Focused test proves the bad lane is rejected for product use.
- Existing official-compatible flash plan/execute tests remain valid.

### Phase 2: Verification

Actions:

- Run focused app tests for flash command validation.
- Run `git diff --check`.

Acceptance:

- Focused tests pass.
- No hardware write, NVS write, service start/stop, provider call, or V21
  execution occurred.

## Rollback

- Revert only the guard patch if it blocks legitimate no-write evidence
  generation.
- Keep the recovered StackChan-compatible firmware on the device.

## Risks

- Over-broad rejection could block future custom wake development. Keep the
  guard scoped to product StackChan flash identity, not package/readiness
  evidence.
- Error messaging must be explicit enough for future workers under time
  pressure.

## Worker Execution Task

Worker owns only `T-FLASH-GUARD-001`.

Allowed:

- Edit focused flash validation code and tests.
- Run focused Go tests and `git diff --check`.

Forbidden:

- Hardware flash or NVS write.
- Provider/V21 execution.
- Service start/stop.
- Broad firmware, Gateway, provider, or audio refactors.

Return summary format:

- Guard behavior.
- Changed files.
- Tests run and results.
- Any legitimate workflow that might need a bypass or follow-up.
