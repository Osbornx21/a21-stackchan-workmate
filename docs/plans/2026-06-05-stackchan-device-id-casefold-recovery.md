# 2026-06-05 - StackChan Device ID Casefold Recovery

Status: implemented, verified, committed, pushed, deployed to ECS, and public
post-deploy verified.
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

## Deployment Result

- Commit `7b32956 fix(gateway): normalize stackchan hardware mac ids` was
  pushed to
  `origin/codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
- Deployed directly over SSH using
  `/Users/jiyurun/.ssh/x21_aliyun_stackchan` to ECS
  `i-uf63f4ymqc2dxtljxz2n` / `47.103.57.217`.
- Source archive SHA-256:
  `97c78693a6dd1d97a242a497a198d38f79b3bc0e8cbd6e8b5a49b954a516f36b`.
- Remote focused tests passed in `/opt/a21.next`:
  `GOMAXPROCS=2 go test ./internal/gateway ./internal/app -run 'OfficialStackChan|ProductRecovery|Xiaozhi|PowerLifecycle' -count=1`.
- Remote build passed:
  `GOMAXPROCS=2 go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- `/opt/a21.next` was safe-swapped to `/opt/a21`, `a21-gateway` restarted
  active, and loopback `/healthz` passed.
- Post-deploy loopback and public checks show one normalized lowercase MAC
  device record, `connection_status=online`, and lowercase official status
  `connected=true`, `official_device_id=44:1b:f6:e2:6a:60`,
  `delivered_transport=stackchan_official_ws`.
- Local public recovery precheck wrote
  `reports/a21-stackchan-product-recovery-20260605-101947.json` with
  `status=product_online_official_relay_ready`,
  `device_count=1`, `device_online=true`, `official_relay.connected=true`,
  and `rom_download_required=false`.

## Forbidden

- No generic `xiaozhi.bin` product flash.
- No NVS write.
- No provider/V21 execution.
- No Git prune/gc.
- Do not mark physical power-key or full PRD acceptance from this server-side
  fix alone.
