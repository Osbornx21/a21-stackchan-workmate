# 2026-06-04 - Roleplay Official Expression Plan

Status: completed and verified.
Owner: A21 control tower.
Transition: `T-INTERNAL-TEST4-ROLEPLAY-OFFICIAL-EXPRESSION-PLAN-001`.

## Goal

Expose a safe, no-send roleplay expression plan that maps the selected A21 role
soul, scenario, memory readiness, and voice-clone profile into existing
official StackChan avatar/action semantics.

## Starting State

- Roleplay soul, scenario, memory, and voice-clone profile already enter the
  voice-pipeline prompt/TTS boundary.
- `/v1/devices` reflects safe roleplay device state.
- `internal/transport/stackchan` already maps individual A21 semantic
  `state`/`face`/`motion` events to official `ControlAvatar`, `ControlMotion`,
  and `DanceSequence` packets with `official_action_physical_accepted=false`.
- `/v1/roleplay-profile` does not yet expose which official StackChan action
  semantics represent the selected roleplay identity.

## Boundary

Allowed:

- Gateway roleplay profile response shape.
- Host-side official action plan metadata using the existing StackChan adapter.
- Simulator readout for safe expression-plan metadata.
- Host-local tests and control docs.

Forbidden:

- Do not write to `/stackChan/ws`, `/v1/stackchan/official/control`, serial,
  NVS, firmware, or ECS.
- Do not execute provider, V21, voice-clone CLI, or audio playback.
- Do not expose prompt text, memory text, transcripts, provider output, audio,
  voice-clone reference paths, or voice samples.
- Do not claim physical screen, servo, RGB, or roleplay acceptance.
- Do not add a custom face engine outside the official StackChan adapter.

## Implementation Steps

1. Add safe `RoleplayExpressionPlan` response types to Gateway.
2. Build the plan from selected roleplay state using existing
   `stackchantransport.BuildOfficialActionPlan` calls.
3. Include actions for baseline posture, selected role soul, scenario emphasis,
   and memory cue when memory is ready.
4. Aggregate packet count and semantic surfaces while keeping
   `physical_accepted=false`.
5. Add the expression plan to `/v1/roleplay-profile` and simulator readout.
6. Add tests proving deterministic plan content, no prompt/memory/sample leak,
   and no physical acceptance overclaim.
7. Update control docs, project state, and handoff log after verification.

## Acceptance

- `GET/POST /v1/roleplay-profile` includes `expression_plan`.
- The plan uses existing official StackChan action metadata only.
- Selected `a21_roleplay_wry_peer` + `engineer_pushback` + memory-ready state
  produces role-soul/scenario/memory actions with packet counts and surfaces.
- The response does not include raw prompt, memory hint text, provider output,
  audio, or voice-clone sample data.
- `physical_accepted` remains false for every action and the aggregate plan.
- Focused Gateway tests, `git diff --check`, and `GOMAXPROCS=2 make verify`
  pass.

## Execution Update - 2026-06-04

- Gateway `RoleplayProfileResponse` now includes `expression_plan`.
- The plan uses `stackchantransport.BuildOfficialActionPlan` for all phases and
  aggregates packet count plus phase-prefixed official semantic surfaces.
- Default phases include baseline posture, role soul expression, and scenario
  emphasis; memory-ready state adds a memory cue.
- Selected `a21_roleplay_wry_peer` maps to the official happy face metadata;
  `engineer_pushback` maps to thinking state metadata; memory-ready state maps
  to the official nod motion metadata.
- The simulator roleplay panel shows the no-send policy, action count, and
  packet count.
- The response redaction policy states that prompt text, memory text,
  transcripts, provider output, audio, and voice-clone samples are not stored.
- `physical_accepted` remains false for every action and the aggregate plan.
- No `/stackChan/ws`, `/v1/stackchan/official/control`, serial, NVS, firmware,
  ECS, provider, V21, voice-clone CLI, or audio playback action occurred.

## Handoff Format

- Transition.
- What changed.
- Files changed.
- Tests run and results.
- Runtime or physical evidence.
- Deviations from plan.
- Remaining issues.
- Next suggested action.
- Forbidden actions avoided.
