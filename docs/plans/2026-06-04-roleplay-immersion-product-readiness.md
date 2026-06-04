# 2026-06-04 - Roleplay Immersion Product Readiness

Status: completed.
Owner: A21 control tower.
Transition: `T-ROLEPLAY-IMMERSION-READINESS-001`.
Created: 2026-06-04 CST.

## Goal

Make product readiness expose whether the current roleplay/immersion contract
is actually ready: selected role soul, scenario, voice-clone profile, bounded
memory prompt readiness, safe prompt composition, and official StackChan
expression planning.

This is a reporting/acceptance visibility cut. The Gateway already exposes
`/v1/roleplay-profile` and voice turns already carry roleplay prompt/voice
metadata into the voice pipeline. The missing piece is that `a21
product-readiness` does not yet show this as an explicit PRD readiness surface.

## Current State

- Branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Source HEAD at plan start:
  `bc35339 feat(readiness): ingest selected voice-chain capability evidence`.
- Working tree at plan start:
  clean and synced with origin.
- Internal test 3 voice/protocol changes are preserved.
- Workspace roleplay controls and voice probe are already complete; do not
  redo that UI work in this transition.

## Target State

- `a21 product-readiness` fetches `GET /v1/roleplay-profile` from the Gateway.
- The product report includes a top-level `roleplay` readiness object with:
  selected roleplay profile, scenario, voice-clone profile, soul readiness,
  prompt-composed status, memory configured/readiness/count, expression plan
  availability, action/packet counts, redaction flags, and physical acceptance
  truth.
- Invalid, unsafe, or unavailable roleplay profile responses become safe
  findings instead of silent readiness.
- This cut does not execute providers, V21, voice-clone CLI, firmware, ECS, or
  physical hardware. It must not claim physical roleplay acceptance.

## Boundaries

- No new Gateway route.
- No frontend rewrite.
- No provider execution.
- No V21 execution.
- No ECS/runtime deployment.
- No firmware build, flash, serial, NVS, or physical hardware action.
- No prune/gc or destructive Git cleanup.

## Acceptance

- Focused `internal/app` tests cover ready roleplay ingestion and invalid/unsafe
  roleplay profile handling.
- `git diff --check` passed.
- `GOMAXPROCS=2 make verify` passed.
- Product readiness JSON does not leak prompt text, memory text, transcript,
  provider output, audio, voice samples, URLs, local paths, or credentials.
- `roleplay.physical_accepted` remains false until a separate physical hardware
  evidence report exists.

## Rollback

Revert only this transition's app/tests/docs changes. Do not revert internal
test 3 protocol/audio commits or internal test 4 workspace/roleplay/professional
surface commits.
