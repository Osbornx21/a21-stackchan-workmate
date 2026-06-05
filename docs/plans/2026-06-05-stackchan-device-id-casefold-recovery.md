# 2026-06-05 - StackChan Device ID Casefold Recovery

Status: implemented locally, verified, pending commit and ECS deployment.
Transition: `T-STACKCHAN-DEVICE-ID-CASEFOLD-RECOVERY-001`.

## Problem

After the product firmware recovery, serial evidence showed the official avatar
relay was connected and receiving heartbeat pings, but Gateway HTTP status could
still report the relay disconnected when queried with lowercase product MAC
`44:1b:f6:e2:6a:60`.

The live Gateway had split one physical StackChan into two records:

- Uppercase MAC `44:1B:F6:E2:6A:60`: stale official relay record, last event
  `stackchan.official_ws.connected`.
- Lowercase MAC `44:1b:f6:e2:6a:60`: online Xiaozhi product socket, last event
  `device.heartbeat`.

The recovery CLI also stopped at the first case-insensitive match, so it could
select the uppercase stale record before seeing the lowercase online record.

## Target State

- Hardware MAC device IDs are normalized for Gateway socket/status lookup.
- Official relay status and control work regardless of MAC case.
- Product recovery prefers an online/latest record when duplicate
  case-variant MAC records are present.
- No firmware flash, NVS write, provider/V21 execution, or product protocol
  rollback is needed for this server-side fix.

## Implementation

- Added a Gateway `deviceIDLookupKey` helper that lowercases MAC-shaped device
  IDs and leaves non-MAC device IDs unchanged.
- Applied the lookup key to official StackChan socket register/unregister,
  official control lookup, status lookup, Xiaozhi-to-official fanout lookup,
  and official relay connection registry writes.
- Updated product recovery device selection to prefer online records, then
  newer `last_seen_ms`, then lower `device_age_ms`.
- Added regression tests for uppercase official relay plus lowercase product
  status/control, and for recovery choosing lowercase online over uppercase
  stale.

## Acceptance

- Focused Gateway official relay casefold tests pass.
- Focused product recovery casefold tests pass.
- Broader Gateway/App `OfficialStackChan|ProductRecovery|Xiaozhi|PowerLifecycle`
  test subset passes.
- `GOMAXPROCS=2 make verify` passes.
- ECS deployment and public post-deploy smoke must show lowercase
  `/v1/stackchan/official/status` connected and product recovery no longer
  classifies this as ROM/download recovery while the device is online.

## Forbidden

- No generic `xiaozhi.bin` product flash.
- No NVS write.
- No provider/V21 execution.
- No Git prune/gc.
- Do not mark physical power-key or full PRD acceptance from this server-side
  fix alone.
