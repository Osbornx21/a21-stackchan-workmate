# A21 Roleplay Memory Control Surface

Date: 2026-06-04

## Transition

`T-ROLEPLAY-MEMORY-CONTROL-SURFACE-001`

## Current State

- A21 already exposes `roleplay` and `professional` as the two user-facing
  product modes.
- `/v1/roleplay-profile` selects the roleplay profile, scenario/playbook, and
  voice-clone profile.
- Roleplay memory hints are composed in memory through the personality package,
  but the active runtime mainly receives them from `A21_MEMORY_*` environment
  variables.
- Responses and traces already report memory readiness/counts and avoid raw
  prompt, memory, transcript, provider output, voice sample, and V21 leakage.

## Target State

- Operators can set and clear bounded roleplay session memory hints through the
  existing roleplay profile control surface.
- Memory hints remain prompt-input-only runtime material. Gateway responses,
  traces, docs, and simulator readouts report only counts/status/findings.
- Voice-clone selection continues to bridge to `voice_chain_profile` without
  changing professional/V21 scope.

## Trigger

- Internal test 4 needs the Xiaozhi-style roleplay/persona/memory lane to be a
  product-visible surface, paired with voice-clone profile selection, while
  preserving internal test 3 device-side voice behavior.

## Actions

1. Extend the roleplay profile request contract with bounded memory hint input
   and clear semantics.
2. Reuse the existing personality memory policy for filtering, truncation,
   limits, and redacted state reporting.
3. Keep hints in Gateway runtime memory only; do not persist them to reports,
   traces, provider payload logs, V21, or firmware.
4. Add simulator controls for setting one session hint and clearing runtime
   hints.
5. Update protocol, control state, state machine, handoff log, and tests.

## Acceptance

- `POST /v1/roleplay-profile` can set safe session memory hints and clear them.
- The response reports memory readiness and hint counts without echoing raw
  hints.
- Unsafe hint material such as URLs, local paths, and credential-looking strings
  is blocked or reported as redacted findings.
- Fast companion roleplay turns use the configured hint count, record only
  readiness trace markers, and keep `v21_executed=false`.
- Focused Gateway/personality tests and `make verify` pass.

## Failure State

- Raw hint text appears in a response, trace, report, log, or docs example.
- Roleplay memory setup silently routes to V21/professional.
- Voice-clone selection regresses or stops updating the voice-chain profile.
- Internal test 3 fast companion / Xiaozhi protocol behavior is changed.

## Rollback Path

- Revert the Gateway request/runtime-memory additions and simulator controls.
- Keep the existing environment-variable memory path and roleplay profile
  selection intact.
- Do not revert internal test 3 protocol changes, voice-chain profile support,
  professional workspace contracts, or hardware parity docs.

## Next State

- After this transition, schedule roleplay persona quality/evidence review:
  prompt assets, scenario examples, voice clone profile readiness, wake/touch
  entry points, and physical turn evidence.
