# 2026-06-05 - Power Lifecycle Cold Boot Evidence Guard

Status: implemented and locally verified in current working tree, pending
commit and deployment.
Transition: `T-POWER-LIFECYCLE-COLD-BOOT-EVIDENCE-GUARD-001`.

## Problem

Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` correctly treated
StackChan power/lifecycle as a product blocker. The current Gateway already has
`/v1/power-lifecycle`, and the product overlay already carries the AXP2101
power-key register profile, but the acceptance endpoint could still be called
with only broad boolean fields. That makes it too easy to mistake an already
online Xiaozhi socket for a real no-USB cold boot from the physical power key.

## Target State

- Keep the WDT-safe direct Xiaozhi product firmware path.
- Keep the official-compatible product flash lane unchanged.
- Keep product physical acceptance false until foreground cold boot is observed.
- Require the power-lifecycle acceptance payload to identify the StackChan
  battery power-key cold-boot source, USB-disconnected boot condition, power-key
  press window, PMIC profile, and boot observation timestamp.

## Implementation

- `POST /v1/power-lifecycle-acceptance` now requires:
  - `boot_source=battery_power_key_cold_boot`
  - `usb_connected_during_boot=false`
  - `power_key_hold_ms` in `250..12000`
  - `pmic_power_key_profile=a21_stackchan_axp2101_pwrkey_v1`
  - `boot_observed_at_ms>0`
- Successful acceptance records the boot source, PMIC profile, power-key hold
  window, and USB-disconnected condition into redacted device capabilities.
- `GET /v1/power-lifecycle` now includes a `pmic_power_key_profile` item so the
  state machine distinguishes PMIC profile acceptance from generic online
  status.

## Acceptance

- Focused Gateway tests cover accepted cold-boot evidence, missing PMIC/boot
  evidence rejection, and the existing online-socket requirement.
- Broad verification must pass before commit:
  - `git diff --check`
  - `GOMAXPROCS=2 go test ./internal/gateway -run 'TestPowerLifecycle|TestHardwareAcceptance' -count=1`
  - `GOMAXPROCS=2 go test -race ./internal/gateway -run 'PowerLifecycle|OfficialStackChan|Xiaozhi|WorkspaceConsolePageServed' -count=1`
  - `GOMAXPROCS=2 make verify`
  - `GOMAXPROCS=2 make preflight`
  - `GOMAXPROCS=2 make doctor`

## Rollback

Revert the Gateway/protocol/doc changes. This only changes acceptance strictness;
it does not alter firmware, flash, NVS, provider execution, or live device
control.

## Forbidden

- No firmware flash or NVS write in this transition.
- No generic Xiaozhi product flash lane.
- No provider/V21 execution.
- No Git prune/gc.
- Do not mark physical power-key acceptance without a foreground no-USB boot.
