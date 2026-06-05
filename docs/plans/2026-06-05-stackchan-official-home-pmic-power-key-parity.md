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

After the first parity flash still did not prove physical no-USB cold boot, the
active hypothesis was narrowed further:

- NVS does not contain a `xiaozhi/idle_sec` or `xiaozhi/ext_pwr` value that
  would explain immediate battery-mode shutdown.
- The symptom still looks like an AXP2101 battery cold-boot hold problem or a
  rail/threshold problem before A21 Gateway can observe the device.
- AXP2101 `REG24` controls the battery-voltage power-off threshold. The next
  product candidate lowers it to the 2.6V setting to tolerate cold-boot inrush,
  while leaving DCDC over/under-voltage protection enabled.
- The product heartbeat now carries a read-only PMIC raw register snapshot as
  `pmic_power_status` so future Gateway evidence can distinguish battery
  absent/low/charging, power-off causes, and applied register values.

2026-06-05 16:32 CST update:

- A stock-official no-overlay baseline was T7-flashed and captured over USB. It
  boots stock `stack-chan` 1.4.1 to the official Launcher and initializes the
  expected hardware stack under USB power.
- The operator then reported the same no-USB PWRKEY symptom on the stock A/B:
  screen/red LED flash, then no usable boot. That shifts the unresolved boundary
  below A21 Gateway, voice, and AI.AGENT runtime.
- The A21 overlay PMIC write hunk for `REG10`, `REG22`, and `REG24` did not
  close the symptom and diverges from stock official. It has been removed from
  the product candidate. Only read-only PMIC diagnostics remain.
- Future PMIC changes require new evidence from serial PMIC snapshots,
  battery/rail inspection, or official hardware documentation. Do not restore
  speculative AXP2101 write churn as a default product fix.

## Target State

- Keep official StackChan Home, setup, mobile surfaces, and `AI.AGENT` mode
  entry intact.
- Restore only the conservative AXP2101 PMIC power-key parity:
  - `REG10 |= 0x04` for the hardware PWRON shutdown fallback.
  - `REG22 = 0b110` so PWRON and OFFLEVEL can request PMIC power-off.
  - `REG24 = 0x00` for the 2.6V battery-voltage power-off threshold.
  - `REG27 = 0x00` to preserve official ON/OFF timing.
- Expose read-only PMIC raw state in the heartbeat without adding any remote
  power-control surface.
- Product flash uses only the guarded official-compatible lane and artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`.

## Actions

1. Update the official-compatible overlay with the narrow PMIC parity hunk.
2. Update overlay tests so the official front-end remains guarded while this
   exact PMIC parity hunk is required.
3. Add PMIC cold-boot threshold hardening and raw snapshot heartbeat evidence.
4. Verify the overlay applies to the clean official StackChan source.
5. Run focused tests and `make verify`.
6. Build and flash the product artifact through the guarded product lane.
7. Collect Gateway/device evidence, then request physical no-USB boot/shutdown
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

## Build Evidence

- 2026-06-05 15:10 CST: focused overlay tests passed.
- 2026-06-05 15:10 CST: `git diff --check` passed.
- 2026-06-05 15:10 CST: `GOMAXPROCS=2 make verify` passed.
- 2026-06-05 15:10 CST: guarded product build passed.
- App artifact:
  `a21-stackchan-official-xiaozhi-compatible.bin`; SHA-256
  `31615f23fb2dc5d8e746fc6b6ed3186b65a020f29e9c72f114b89b39b4aaf36e`.
- Build report:
  `reports/a21-stackchan-official-baseline-20260605-150958-1780643398410587000.json`.

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
