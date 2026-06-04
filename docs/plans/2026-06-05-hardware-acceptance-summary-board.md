# Hardware Acceptance Summary Board

Status: deployed; foreground physical confirmation pending.
Date: 2026-06-05 CST.

## Goal

Give the operator a single product-surface view of the prototype body
acceptance state so mode ritual and full-check evidence are not confused with
final physical acceptance.

## Transition

- Current state: mode ritual and `full_check` have delivery endpoints and
  physical acceptance endpoints, but their current status is split across
  panels and raw registry fields.
- Target state: `GET /v1/hardware-acceptance` summarizes machine-delivered and
  physical-accepted state for `mode_ritual` and `full_check`; `/workspace`
  exposes an `Acceptance Board` that shows the next operator action.
- Acceptance: focused tests prove the endpoint reads redacted registry
  evidence only, reports `physical_pending` when delivery exists without
  operator acceptance, and the workspace exposes the board.

## Boundaries

- Read-only summary endpoint; no hardware command is sent by the summary.
- No automatic physical acceptance.
- No firmware build or flash.
- No NVS write.
- No provider or V21 execution.
- No camera/NFC/IR/reboot/OTA/app-lifecycle exposure.
- No internal-test3 voice/protocol rollback.

## Deployment Evidence

- Commit `5d786ef feat(gateway): summarize hardware acceptance` added
  `GET /v1/hardware-acceptance` and the `/workspace` `Acceptance Board`.
- Local red/green evidence: before implementation,
  `TestWorkspaceConsolePageServed` missed `/v1/hardware-acceptance`, and the
  endpoint returned HTTP 404. After implementation, focused Gateway tests and
  `GOMAXPROCS=2 make verify` passed.
- ECS `47.103.57.217` is deployed at `5d786ef`; remote focused Gateway tests,
  remote Go build, systemd restart, loopback `/healthz`, and public direct
  `/healthz` passed.
- Public `/workspace` smoke found `Acceptance Board`, `Refresh Acceptance`,
  `hardwareAcceptanceStatus`, `hardwareAcceptanceItems`, and
  `/v1/hardware-acceptance`.
- After deployment, the first summary correctly returned
  `overall_status=machine_evidence_pending` because the in-memory registry had
  restarted.
- Live roleplay ritual trace
  `a21-trace-mode-ritual-summary-ready-5d786ef-202606050405` was delivered
  with `step_delay_ms=180` and `total_planned_delay_ms=540`.
- Live `full_check` trace
  `a21-trace-full-check-summary-ready-5d786ef-202606050405` was delivered
  with `step_delay_ms=180`, `total_planned_delay_ms=2700`, and trace
  `summary.last_offset_ms=2713`.
- Final public `/v1/hardware-acceptance?device_id=44:1b:f6:e2:6a:60` returned
  `overall_status=physical_pending` with both `mode_ritual` and `full_check`
  marked `delivery_status=delivered`, `physical_accepted=false`, and next
  actions `accept_visible_mode_ritual` /
  `accept_visible_full_check`.

The board is a control and recovery surface. It does not claim physical
acceptance until the operator or an instrument records visible evidence.
