# A21 Roleplay Soul Profile Contract

Date: 2026-06-04

Status: implemented and verified.

Transition: `T-INTERNAL-TEST4-ROLEPLAY-SOUL-PROFILE-001`

## Current State

- A21 already exposes `roleplay` / `professional` as user-facing modes.
- Gateway exposes `/v1/roleplay-profile` and can select scenario, memory hints,
  and voice-clone profile.
- The selected scenario and memory hints already enter the low-latency voice
  pipeline prompt input, and the selected safe voice-clone profile reaches the
  TTS boundary.
- `roleplay_profile` currently behaves like a single default identity slot, so
  the simulator cannot switch among distinct role soul/personality profiles
  even though the PRD and user direction require role immersion.

## Target State

- `roleplay_profile` becomes a safe role-soul selector with multiple A21-owned
  soul profiles.
- The selected soul profile is composed into roleplay prompt input before mode,
  scenario, and memory overlays.
- `/v1/roleplay-profile` returns safe soul/profile metadata, prompt-part
  readiness, memory counts, and selected voice-clone profile without returning
  prompt text, memory text, transcripts, provider output, sample paths, URLs,
  credentials, V21 evidence, or audio.
- The simulator can select the role soul, scenario, voice clone, and bounded
  memory hint together, and the saved selection affects subsequent
  fast-companion and stock Xiaozhi roleplay voice-pipeline turns.

## Boundary Conditions

- No real provider execution, V21 execution, cloud storage, Gateway service
  start, firmware build, flash, serial, NVS, ECS change, or physical hardware
  action.
- No new production dependency.
- No X21 naming in A21 implementation. V21 appears only in professional
  no-execute boundary text.
- Do not store or return composed prompt bodies.
- Keep `dialogue` as a compatibility alias under `roleplay`.

## Implementation Steps

1. Add role-soul personality assets under `docs/personality/role_souls/` and
   document the runtime composition order.
2. Extend the personality composer with an optional `RoleSoul` layer.
3. Expand Gateway `roleplay_profile` validation/options to include multiple
   A21 soul profiles and safe prompt-part metadata.
4. Ensure roleplay runtime summaries and voice-pipeline prompt composition use
   the selected soul profile.
5. Add simulator controls/readouts for role soul selection.
6. Add focused tests proving selected soul reaches prompt input while API
   responses/traces remain redacted.
7. Update protocol, control, state-machine, and handoff docs.

## Acceptance

- `POST /v1/roleplay-profile` can select a non-default role soul together with
  scenario and voice-clone profile.
- Fast-companion voice-pipeline request captures prompt input containing the
  selected soul asset and selected scenario/memory structure.
- `/v1/roleplay-profile`, fast-companion responses, traces, and simulator
  readouts do not leak prompt bodies or private memory text.
- The simulator page includes `roleplayProfile`, `roleplayScenario`,
  `voiceCloneProfile`, and memory controls.
- Focused Gateway/personality tests, `git diff --check`, and `make verify`
  pass.

## Failure State

- Role soul selection does not affect voice-pipeline prompt input.
- Prompt text, memory text, reference audio paths, credentials, provider output,
  or V21 evidence leak into API responses, traces, docs intended as reports, or
  simulator readouts.
- Professional/V21 execution occurs from a roleplay turn.
- Internal test 3 Xiaozhi protocol behavior changes.

## Rollback

- Remove the new role-soul assets and composer option.
- Revert Gateway roleplay profile options to the default profile only.
- Keep existing scenario, memory, and voice-clone controls intact.

## Next State

- Roleplay can be evaluated as a configurable persona/voice experience rather
  than only a single prompt lane. A separate physical evidence window can then
  test selected soul + selected voice clone on StackChan audio.

## Implementation Result

- Added `docs/personality/role_souls/default.md`,
  `docs/personality/role_souls/wry_peer.md`, and
  `docs/personality/role_souls/calm_anchor.md`.
- Extended `personality.Compose` with optional `RoleSoul` prompt layering.
- Expanded `/v1/roleplay-profile` so `roleplay_profile` can select
  `a21_roleplay_default`, `a21_roleplay_wry_peer`, or
  `a21_roleplay_calm_anchor`.
- Runtime summaries now include `soul_prompt_input_ready` and safe
  `prompt_parts`; prompt bodies remain unreturned and untraced.
- Fast-companion and stock Xiaozhi roleplay prompt composition use the
  selected role soul before calling the provider-neutral voice pipeline.
- Simulator now exposes a role soul selector/readout beside scenario,
  voice-clone, and memory controls.

## Verification

- `go test ./internal/personality ./internal/gateway -run 'TestPersonalityComposeRoleSoulAddsOnlySelectedSoul|TestRoleplayProfileEndpointSelectsSoulProfileAndReturnsSafeCatalog|TestFastCompanionHybridRunsVoicePipelineWhenFramesProvided|TestSimulatorPageServed' -count=1`:
  passed.
- `go test ./internal/personality ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
