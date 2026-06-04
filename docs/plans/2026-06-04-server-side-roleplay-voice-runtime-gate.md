# 2026-06-04 - Server-Side Roleplay Voice Runtime Gate

Status: completed.
Owner: A21 control tower.
Transition: `T-SERVER-SIDE-ROLEPLAY-VOICE-RUNTIME-GATE-001`.
Created: 2026-06-04 CST.

## Goal

Prevent server-side candidate readiness from bypassing roleplay voice immersion.
`server-side-readiness-bundle` must require and collect roleplay voice runtime
evidence in addition to provider, V21, host voice, wake, and voice-chain
selector evidence.

## Current State

- Source HEAD at plan start:
  `050723e feat(readiness): generate roleplay voice probe evidence`.
- `a21 roleplay-voice-probe` can already generate
  `a21-roleplay-voice-probe-*.json`.
- `product-readiness` already surfaces `roleplay_voice_runtime` in canonical
  missing real evidence, but `server_side.candidate_ready` does not yet require
  it.

## Result

- `product-readiness.server_side` now includes
  `roleplay_voice_runtime_ready` and `roleplay_voice_source_report`.
- `server_side.missing_evidence` includes `roleplay_voice_runtime` until a
  matched ready roleplay voice probe report is present.
- `server-side-readiness-bundle` exposes a `roleplay_voice` evidence block.
- `server-side-readiness-bundle --collect-missing` now runs
  `a21 roleplay-voice-probe --require-ready` and absorbs the generated report
  only when product readiness accepts it.

## Boundaries

- No ECS/root-secret change.
- No V21 execution beyond existing explicitly authorized bundle behavior.
- No firmware build, flash, serial, NVS, or physical hardware action.
- No report deletion, prune/gc, or destructive Git cleanup.
- The gate is host/Gateway runtime readiness. Physical wake, audible playback,
  voice-clone quality, hardware expression, and full PRD acceptance remain
  separate gates.

## Acceptance

- Focused app tests pass for server-side bundle roleplay collection and
  candidate gating.
- `git diff --check` passes.
- `GOMAXPROCS=2 make verify` passes.

## Rollback

Revert only this gate and bundle collection change. Do not revert internal test
3 voice/protocol commits, roleplay runtime evidence ingestion, or the
roleplay voice probe generator.
