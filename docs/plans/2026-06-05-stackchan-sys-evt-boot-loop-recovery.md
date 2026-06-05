# 2026-06-05 - StackChan sys_evt Boot Loop Recovery

Status: implemented, committed, rebuilt, product-flashed, and serial-verified.
Transition: `T-STACKCHAN-SYS-EVT-BOOT-LOOP-RECOVERY-001`.

## Problem

The product StackChan is not hard-bricked: macOS enumerates the ESP32-S3 USB
JTAG/serial device as `/dev/cu.usbmodem1101`, serial number
`44:1B:F6:E2:6A:60`. Serial boot capture shows the A21 official-compatible app
initializes AXP2101, display, camera, touch, IMU, servos, and audio, then
reboots after Wi-Fi finds `ChinaNet-N6e3` with:

```text
***ERROR*** A stack overflow in task sys_evt has been detected.
```

The visible infinite white flashing screen is therefore an application boot
loop, not a complete ROM/USB brick.

## Target State

- Preserve the A21 official-compatible product lane and artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Keep direct Xiaozhi start and official avatar relay support.
- Do not start a second Wi-Fi/network path from the A21 avatar relay task.
- Increase the ESP system event task stack for the heavier StackChan product
  path.
- Rebuild, guarded-flash, and collect serial evidence showing no `sys_evt`
  stack overflow across Wi-Fi scan/connect.

## Implementation

- `Hal::startA21WebSocketAvatarRuntime` now waits for the Xiaozhi network path
  instead of calling `startNetwork(onStartLog)` again from the avatar relay
  task.
- `firmware/sdkconfig.defaults` now sets
  `CONFIG_ESP_SYSTEM_EVENT_TASK_STACK_SIZE=8192`.
- Overlay tests reject reintroducing the second `startNetwork(onStartLog)` call.

## Acceptance

- Focused official-compatible overlay tests pass.
- Broader app firmware/product recovery tests pass.
- `make a21-stackchan-official-xiaozhi-compatible-build` passes and produces
  the product app artifact.
- Guarded product flash through
  `a21-stackchan-official-xiaozhi-compatible-flash-execute` passes from a clean
  control worktree.
- Post-flash serial capture reaches Wi-Fi connect without `sys_evt` stack
  overflow or reset loop.

## Additional Root Cause

After the boot loop fix, the official avatar relay still failed with
`Failed to get host by name`. The upstream official `WebSocket::Connect`
implementation parses the first `:` after `ws://` as a host/port separator.
The A21 relay URL included the product MAC address in `device_id` with literal
colons, so the parser could misread the query string as part of the host/port
field.

## Additional Implementation

- The A21 official avatar relay URL now percent-encodes MAC-address colons as
  `%3A` in the `device_id` query parameter.
- Overlay tests require the escaped device-id path so this parser compatibility
  issue is not silently reintroduced.

## Result

- Commit `cef0e7b fix(firmware): prevent stackchan sys_evt boot loop`.
- Commit `a95bb86 fix(firmware): encode stackchan relay device id`.
- Product build passed:
  `GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`.
- Final guarded product flash passed:
  `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- Final app artifact SHA-256:
  `c68e9557f065eb40a184ea88d8f111fbc4aa87ca6729c38899cc8555edc7acaf`.
- Final flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-095236-1780624356871525000.json`.
- Post-flash serial summary:
  no `stack overflow in task sys_evt`, no `Rebooting...`, no `rst:`, no
  `Guru Meditation`, no `abort()`, Xiaozhi session
  `a21-session-44-1b-f6-e2-6a-60` observed, quiet control websocket active,
  official avatar relay connected, heartbeat pings observed, and no DNS
  failure.

## Forbidden

- No generic `xiaozhi.bin` product flash.
- No NVS credential write in this transition.
- No provider/V21 execution.
- No Git prune/gc.
- Do not mark physical power-key or PRD acceptance until foreground evidence is
  collected.
