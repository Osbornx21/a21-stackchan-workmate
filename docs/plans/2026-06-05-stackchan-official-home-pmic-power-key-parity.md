# StackChan Official Home PMIC Power-Key Parity

Date: 2026-06-05

## Transition

`T-STACKCHAN-OFFICIAL-HOME-PMIC-POWER-KEY-PARITY-001`

## Current State

- The product device can enter the restored official StackChan Home after the
  guarded NVS provision marks `app_config/is_configed=1`.
- Gateway sees the product device online while USB/external power is present.
- Physical no-USB power-key boot is not accepted: pressing the upper-left power
  key flashes screen/red LED, then stops before the official front-end becomes
  usable.
- The current product overlay preserves the official launcher/setup/Home entry,
  but no longer carries the earlier AXP2101 PMIC power-key parity hunk.

## Root-Cause Hypothesis

The official front-end recovery fixed the UI/setup state machine but removed the
PMIC parity that previously enabled the AXP2101 PWRON/OFFLEVEL power-off path
and 16-second hardware shutdown fallback. The remaining failure is before
Gateway, provider, voice, or A21 mode entry can participate.

## Target State

- Keep official StackChan Home, setup, mobile surfaces, and `AI.AGENT` mode
  entry intact.
- Restore only the conservative AXP2101 PMIC power-key parity:
  - `REG10 |= 0x04` for the hardware PWRON shutdown fallback.
  - `REG22 = 0b110` so PWRON and OFFLEVEL can request PMIC power-off.
  - `REG27 = 0x00` to preserve official ON/OFF timing.
- Product flash uses only the guarded official-compatible lane and artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`.

## Actions

1. Update the official-compatible overlay with the narrow PMIC parity hunk.
2. Update overlay tests so the official front-end remains guarded while this
   exact PMIC parity hunk is required.
3. Verify the overlay applies to the clean official StackChan source.
4. Run focused tests and `make verify`.
5. Build and flash the product artifact through the guarded product lane.
6. Collect Gateway/device evidence, then request physical no-USB boot/shutdown
   acceptance.

## Acceptance

- `go test ./internal/app -run 'OfficialXiaozhiCompatibleOverlay|StackChanOfficialCandidateContract' -count=1` passes.
- `git diff --check` passes.
- Clean official source accepts
  `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`.
- `GOMAXPROCS=2 make verify` passes.
- Guarded product build and flash reports are recorded.
- Operator confirms no-USB power key can cold boot to official Home and shut
  down/restart without USB.

## Failure State

- If the same physical symptom remains after this flash, the failure is no
  longer attributable to the official Home gate or the firmware PMIC parity
  hunk. Escalate to battery/PMIC rail/latch hardware inspection and a stock
  official firmware A/B power test.

## Rollback Path

- Revert only this transition's overlay/test/docs commits.
- Do not revert internal-test3 voice/protocol changes, official Home NVS
  provision, Gateway protocol changes, or ECS deployments.

## Forbidden Actions

- No generic `xiaozhi.bin` product flash.
- No provider keys in firmware.
- No direct boot into Xiaozhi before official Home/Setup.
- No BLE/setup/main launcher rewrites in this transition.
- No Git prune/gc cleanup.
