# 2026-06-05 - StackChan sys_evt Boot Loop Recovery

Status: implemented in working tree, pending commit, product flash, and physical
evidence.
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

## Forbidden

- No generic `xiaozhi.bin` product flash.
- No NVS credential write in this transition.
- No provider/V21 execution.
- No Git prune/gc.
- Do not mark physical power-key or PRD acceptance until foreground evidence is
  collected.
