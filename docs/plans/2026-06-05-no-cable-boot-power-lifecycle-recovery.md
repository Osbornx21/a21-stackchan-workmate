# 2026-06-05 - No-Cable Boot Power Lifecycle Recovery

Status: active, lifecycle request path rejected by serial evidence.
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

## Runtime Finding

- Commit `fabffd4` changed the overlay to request Xiaozhi through the official
  lifecycle. The product build and guarded flash passed, but post-flash MCP
  body commands returned HTTP 409 `xiaozhi websocket is not connected`.
- Serial evidence on `/dev/cu.usbmodem1101` showed repeated task watchdog
  triggers with CPU0 running `main`.
- The decoded backtrace pointed at `GetMooncake().uninstallAllApps()` from
  `app_main`, specifically AppSetup/AppLauncher LVGL object teardown. This
  makes the official teardown path unsafe as an immediate product fix.
- The product lane must therefore preserve the WDT-safe direct
  `GetHAL().startXiaozhi()` path until a separate lifecycle-teardown transition
  can prove a non-regressing official app shutdown.

## Target State

- Keep the no-welcome product behavior and avoid the WDT-triggering Mooncake
  teardown path.
- Keep official-compatible product lane and artifact name.
- Start Xiaozhi directly after official apps preload and keep feeding the
  watchdog so the product runtime reconnects to the Gateway.
- Build and flash only through the guarded
  `a21-stackchan-official-xiaozhi-compatible` product lane.
- Treat physical power-button/cold no-cable boot as pending until the operator
  confirms the foreground behavior.

## Acceptance

- Focused overlay tests prove the A21 product overlay starts Xiaozhi directly
  before Mooncake teardown and does not request the unsafe lifecycle path.
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
