# 2026-06-05 - StackChan Native Motion And Power Diagnostics

Status: deployed and product-flashed; physical acceptance pending operator.
Transition: `T-STACKCHAN-NATIVE-MOTION-POWER-DIAGNOSTICS-001`.

## Problem

Foreground testing still reports two product regressions:

- No-USB physical power-key boot/shutdown is not accepted.
- Touch and body actions are visibly smaller than the official StackChan
  experience.

The previous PMIC parity flash was necessary but not sufficient. The next
transition must stop blind power-register churn and expose power source
telemetry while restoring StackChan motion amplitudes to official-scale servo
semantics.

## Scope

- Increase official `/stackChan/ws` motion sequences and Gateway MCP body
  fallback amplitudes while staying inside official StackChan head limits.
- Increase firmware-local screen/top touch RGB, vibration, and head feedback.
- Add product heartbeat battery/power-source runtime echo so Gateway can tell
  whether no-cable failure is battery/supply path or power-key state machine.
- Build, flash, and deploy only through the A21 official-compatible product
  lane.

## Out Of Scope

- No rollback of internal-test3 voice protocol changes.
- No generic `xiaozhi.bin` product flash.
- No provider key changes.
- No Mooncake teardown lifecycle retry in this transition.

## Acceptance

- Focused transport/Gateway/app overlay tests pass.
- `GOMAXPROCS=2 make verify` passes.
- Product official-compatible firmware build passes.
- Guarded product flash uses `a21-stackchan-official-xiaozhi-compatible.bin`.
- Public Gateway receives a heartbeat with battery/power-source runtime echo.
- Operator performs foreground confirmation for no-USB power-key behavior and
  visible body/touch amplitude; physical acceptance remains false until then.

## 2026-06-05 12:54 CST Progress

- Gateway, transport, and firmware overlay implementation is complete for this
  transition.
- Overlay apply was fixed for `protocol.cc` by replacing the fragile one-line
  include hunk with stable context.
- Focused app/Gateway/transport tests passed earlier in this transition.
- `GOMAXPROCS=2 make verify` passed.
- Product build passed through `a21-stackchan-official-xiaozhi-compatible-build`
  with app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  SHA-256 `357211684819d915106f8b72acdfe3bbf7d4b2f63d4547970599d05efdcd9c51`.
- First guarded flash attempt was correctly blocked by the A21 control guard
  because the implementation worktree was dirty. Next action is commit, ECS
  deploy, then guarded product flash on `/dev/cu.usbmodem1101`.

## 2026-06-05 13:01 CST Progress

- Commit `2356745a8586` was deployed to ECS `47.103.57.217` through
  `/opt/a21.next` safe swap.
- Remote focused Gateway/App tests passed, remote Go build passed, and
  `a21-gateway` restarted active with loopback and public `/healthz` OK.
- Guarded product flash passed on `/dev/cu.usbmodem1101` through
  `a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- Flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-125915-1780635555099789000.json`.
- Post-flash public Gateway evidence:
  product device online, official relay connected through
  `stackchan_official_ws`, runtime echo includes battery/power keys
  `battery_level`, `battery_charging`, `battery_discharging`,
  `external_power`, `power_source`, and `pmic_power_key_profile`.
- Post-flash official motion control trace
  `a21-trace-post-native-motion-flash` returned `status=delivered`,
  `delivered_transport=stackchan_official_ws`, and
  `motion=official_dance_sequence`.
- `/v1/power-lifecycle` now reports `battery_telemetry=diagnostic_runtime_echo`
  while no-cable cold boot, physical power button, and PMIC profile physical
  acceptance remain explicitly missing until foreground operator evidence is
  submitted.
