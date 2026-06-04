# 2026-06-05 - No-Cable Boot Power Lifecycle Recovery

Status: active.
Transition: `T-NO-CABLE-BOOT-POWER-LIFECYCLE-001`.

## Problem

The product device can boot and reconnect when USB or a flashing reset brings
it up, but the user reports that the physical power button still does not start
the prototype when used as a standalone product. Earlier control records
correctly left `no-cable boot/power` open; the latest product flash did not
close that acceptance.

## Current Evidence

- Public Gateway shows product device `44:1b:f6:e2:6a:60` online while powered.
- Product app flash on commit `1c9dece` passed, but only proved guarded flash,
  reboot reconnect, and post-flash body MCP delivery.
- The official-compatible overlay directly called `GetHAL().startXiaozhi()`
  inside `app_main` and parked in a custom feed/delay loop. This bypassed the
  official `requestXiaozhiStart -> main loop break -> uninstall apps ->
  DestroyMooncake -> startXiaozhi` lifecycle.

## Target State

- Keep the no-welcome product behavior.
- Keep official-compatible product lane and artifact name.
- Request Xiaozhi start through the official lifecycle instead of directly
  starting Xiaozhi from `app_main`.
- Build and flash only through the guarded
  `a21-stackchan-official-xiaozhi-compatible` product lane.
- Treat physical power-button/cold no-cable boot as pending until the operator
  confirms the foreground behavior.

## Acceptance

- Focused overlay tests prove the A21 product overlay requests Xiaozhi through
  the official lifecycle and no longer adds a direct `startXiaozhi` bypass.
- Official-compatible product build passes.
- Guarded no-write flash plan passes.
- Guarded flash execute passes on `/dev/cu.usbmodem1101`.
- After flash, product device reconnects to the public Gateway and can receive
  `mode_ritual` and `full_check` commands.
- Operator confirms whether physical power button cold boot succeeds without
  USB. Until then, `physical_accepted=false` remains honest.

## Rollback

- Revert the overlay/test/docs transition and reflash the previous
  official-compatible product artifact if the device fails to boot or connect
  after the guarded flash.

## Forbidden

- No generic `xiaozhi.bin` product flash.
- No NVS write unless a separate foreground NVS transition is opened.
- No provider/V21 execution.
- No Git prune/gc.
