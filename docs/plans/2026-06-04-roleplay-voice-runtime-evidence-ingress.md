# 2026-06-04 - Roleplay Voice Runtime Evidence Ingress

Status: completed.
Owner: A21 control tower.
Transition: `T-ROLEPLAY-VOICE-RUNTIME-EVIDENCE-001`.
Created: 2026-06-04 CST.

## Goal

Move roleplay acceptance from static profile readiness toward runtime evidence:
`product-readiness` should be able to ingest a redacted roleplay voice probe
report proving the selected role soul, scenario, memory prompt, voice-clone
profile, and prompt input reached a roleplay voice turn and produced an audio
downlink boundary.

## Current State

- Branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Source HEAD at plan start:
  `bd7c5dd feat(readiness): surface roleplay immersion readiness`.
- Working tree at plan start:
  clean and synced with origin.
- Product readiness already fetches `GET /v1/roleplay-profile`.
- The workspace Voice Probe already exercises `/v1/fast-companion/turn`, but
  product readiness does not yet ingest roleplay voice runtime evidence.

## Target State

- `a21 product-readiness` accepts `--roleplay-voice-report <report.json>`.
- `--use-latest-reports` can select the latest safe
  `a21-roleplay-voice-probe-*.json` report.
- The top-level `roleplay` readiness object shows whether runtime voice evidence
  is available, matched to the current Gateway profile, and ready.
- Accepted evidence surfaces only safe IDs, booleans, counts, route/status,
  execution mode, and source basename.
- Unsafe reports become safe findings and do not leak prompt text, memory text,
  ASR text, provider output, audio, voice samples, URLs, paths, or credentials.

## Boundaries

- No provider execution.
- No V21 execution.
- No ECS deployment or root secret changes.
- No Gateway protocol route changes.
- No firmware build, flash, serial, NVS, or physical hardware action.
- No prune/gc or destructive Git cleanup.
- This cut must not claim physical StackChan roleplay acceptance or full PRD
  launch readiness.

## Acceptance

- Focused `internal/app` tests cover:
  - ready roleplay voice runtime report ingestion;
  - unsafe report rejection and redaction;
  - latest-report auto-selection.
- `git diff --check` passed.
- `GOMAXPROCS=2 make verify` passed.

## Rollback

Revert only this transition's app/tests/docs changes. Do not revert internal
test 3 endpoint voice/protocol commits, internal test 4 workspace/roleplay
surface commits, or the completed roleplay immersion readiness cut.
