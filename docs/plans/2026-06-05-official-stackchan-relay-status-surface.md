# Official StackChan Relay Status Surface

Date: 2026-06-05 CST.
Owner: A21 control tower.

## Transition

- ID: `T-STACKCHAN-OFFICIAL-RELAY-STATUS-SURFACE-001`
- Current state:
  official `/stackChan/ws` control and auto fanout can deliver frames when the
  official socket is connected, but operators cannot inspect exact/default
  socket connection, latest packet metadata, or next acceptance action without
  reading registry internals.
- Target state:
  Gateway exposes a read-only official relay status surface and `/workspace`
  shows the relay state separately from MCP fallback and physical acceptance.

## Trigger

Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` kept identifying the
official StackChan avatar/action relay as a P1 product gap. Current software
remediation has closed race/preflight/wake gates, so the remaining no-hardware
step is to make the official relay state visible and testable before another
physical flash/acceptance window.

## Actions

- Add `GET /v1/stackchan/official/status?device_id=...`.
- Report schema, requested device, exact/default official relay socket,
  connected age, latest trace/session/event, packet count, semantic surfaces,
  physical acceptance, fallback availability, and next action.
- Add `/workspace` relay status refresh, visible connected/next-action
  metrics, and exported `official_relay_*` metadata.
- Document the contract in `docs/engineering/PROTOCOL.md`.
- Keep endpoint observational only; no official frame send, no MCP fallback
  trigger, no physical acceptance promotion.

## Acceptance

- `TestOfficialStackChanStatusReportsDisconnectedAndNextAction` passes.
- `TestOfficialStackChanStatusReportsConnectedFallbackSocket` passes.
- `TestOfficialStackChanStatusReportsDeliveryMetadata` passes.
- `TestWorkspaceConsolePageServed` contains
  `/v1/stackchan/official/status`.
- Gateway review race subset remains green.
- Full `make verify`, `make preflight`, and `make doctor` pass sequentially.

## Failure And Rollback

- Failure state:
  status endpoint returns misleading physical acceptance, hides disconnected
  official socket state, or causes official action delivery regressions.
- Rollback path:
  remove the status handler/route, workspace status controls, and protocol
  text while preserving existing `/v1/stackchan/official/control`.

## Next State

`S-OFFICIAL-STACKCHAN-RELAY-STATUS-SURFACE-READY`, followed by physical product
recovery, `/stackChan/ws` online evidence, official control delivery to product
MAC, and operator/instrument physical acceptance.
