# 2026-06-04 - Roleplay Device State Reflection

Status: completed host-local registry/simulator cut.
Owner: A21 control tower.
Transition: `T-INTERNAL-TEST4-ROLEPLAY-DEVICE-STATE-REFLECTION-001`.

## Goal

Make the current A21 roleplay identity visible as safe runtime/device state so
the hardware and simulator can reflect the selected role soul, scenario,
bounded memory readiness, and voice-clone binding without exposing prompt
bodies, memory text, transcripts, provider output, or voice-clone samples.

## Starting State

- Role soul, scenario, memory hints, and voice-clone profile already enter the
  fast-companion and stock `/v1/xiaozhi` voice-pipeline prompt/TTS boundary.
- `/v1/roleplay-profile` returns a safe runtime summary.
- `/v1/devices` currently reflects voice mode, chain, ASR/LLM/TTS, realtime
  provider, and voice-clone profile, but it does not explicitly reflect the
  active roleplay profile/scenario or memory readiness.
- The simulator top roleplay panel shows soul and memory from
  `/v1/roleplay-profile`, while the device registry panel does not show the
  device-side roleplay identity state.

## Boundary

Allowed:

- Gateway device registry fields and runtime echo metadata.
- Simulator readout for safe roleplay registry fields.
- Host-local Gateway tests and control docs.

Forbidden:

- Do not start Gateway as a runtime service.
- Do not execute providers, V21, or voice-clone CLI.
- Do not store or display prompt text, memory text, transcripts, provider
  output, document text, audio, voice-clone reference paths, or samples.
- Do not claim physical StackChan roleplay acceptance.
- Do not touch firmware, serial, NVS, ECS, Docker, or prune/gc.

## Implementation Steps

1. Extend `DeviceRecord` with safe roleplay fields:
   `current_roleplay_profile`, `current_roleplay_scenario`,
   `roleplay_soul_ready`, `roleplay_memory_ready`,
   `roleplay_memory_hint_count`, and `roleplay_physical_accepted`.
2. Add a lightweight roleplay device-state snapshot derived from the existing
   roleplay selection and memory redaction policy.
3. Populate the new fields in live control updates and in `/v1/devices`
   snapshots. Merge safe roleplay metadata into `runtime_echo` using only IDs,
   booleans, and counts.
4. Update the simulator registry panel to show role soul, scenario, memory, and
   the existing voice-clone profile as a coherent roleplay device state.
5. Add tests proving profile/scenario/memory reflection, no memory text leak,
   and no physical acceptance overclaim.
6. Update control docs, project state, and handoff log after verification.

## Acceptance

- `/v1/devices` exposes the selected roleplay profile and scenario after a
  device/control turn.
- `/v1/devices` exposes memory readiness/count but not memory text.
- `/v1/devices` keeps `roleplay_physical_accepted=false`.
- Simulator registry contains and updates the safe roleplay readouts.
- Focused Gateway tests, `git diff --check`, and `GOMAXPROCS=2 make verify`
  pass.

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
