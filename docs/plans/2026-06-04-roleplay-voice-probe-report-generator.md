# 2026-06-04 - Roleplay Voice Probe Report Generator

Status: completed.
Owner: A21 control tower.
Transition: `T-ROLEPLAY-VOICE-PROBE-REPORT-GENERATOR-001`.
Created: 2026-06-04 CST.

## Goal

Close the next roleplay evidence gap after
`T-ROLEPLAY-VOICE-RUNTIME-EVIDENCE-001`: add a host-side CLI that generates a
safe `a21-roleplay-voice-probe-*.json` report from the live Gateway
`/v1/fast-companion/turn` roleplay voice path so `product-readiness
--use-latest-reports` no longer depends on a hand-authored fixture.

## Current State

- Branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Source HEAD at plan start:
  `fb5d40b feat(readiness): ingest roleplay voice runtime evidence`.
- Working tree at plan start:
  clean and synced with origin.
- `a21 product-readiness` and `a21 server-side-readiness-bundle` can already
  ingest safe `a21.roleplay_voice_probe.v1` reports.
- No CLI currently generates that report from Gateway runtime evidence.

## Target State

- `a21 roleplay-voice-probe` sends a short redacted local-audio probe to
  `POST /v1/fast-companion/turn` in `roleplay` mode.
- The command fetches `GET /v1/traces?trace_id=...`, filters trace markers to
  the existing allowed roleplay voice marker set, and writes a safe
  `a21-roleplay-voice-probe-*.json` report.
- A complete Gateway voice-pipeline turn records `status=passed`; incomplete or
  unavailable runtime evidence records `status=blocked` without overclaiming.
- The report omits prompt text, memory text, ASR text, provider output, audio
  payloads, voice samples, full URLs, local paths, and credential values.
- `--require-ready` turns an incomplete report into a non-zero CLI exit while
  still preserving the generated blocked report.
- Focused tests prove a generated ready report is accepted by
  `product-readiness --use-latest-reports` and incomplete runtime evidence does
  not pass.

## Boundaries

- No ECS deployment, remote command, or root secret edit.
- No V21 execution.
- No firmware build, flash, serial, NVS, or physical hardware action.
- No provider call is made directly by the CLI; any provider/runtime execution
  happens only through the already configured Gateway voice path.
- No report deletion, archive move, prune/gc, destructive Git cleanup, or
  large-scale rollback.
- This cut is host/Gateway runtime evidence only. It does not claim audible
  voice-clone quality, hardware expression delivery, physical wake acceptance,
  or full PRD launch readiness.

## Acceptance

- Focused app tests cover ready report generation, latest-report ingestion,
  blocked report generation, and `--require-ready`.
- `git diff --check` passes.
- `GOMAXPROCS=2 make verify` passes.

## Result

- Added `a21 roleplay-voice-probe`.
- Ready Gateway voice-pipeline evidence writes `status=passed` and can be
  selected by `a21 product-readiness --use-latest-reports`.
- Incomplete Gateway evidence writes `status=blocked`; `--require-ready`
  returns non-zero after preserving the report.
- Reports use the existing `a21.roleplay_voice_probe.v1` schema and omit
  prompt bodies, memory text, ASR text, provider output, audio payloads, voice
  samples, full URLs, local paths, credentials, physical acceptance, and PRD
  acceptance claims.

## Rollback

Revert only this transition's CLI/tests/docs changes. Do not revert internal
test 3 endpoint voice/protocol commits or the completed roleplay readiness and
roleplay runtime evidence ingress cuts.
