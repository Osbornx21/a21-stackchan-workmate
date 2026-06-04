# A21 Roleplay Voice Runtime Probe Closure

## Transition

`T-ROLEPLAY-VOICE-RUNTIME-PROBE-CLOSURE-001`

## Current State

With a local Gateway running, `server-side-readiness-bundle --collect-missing`
now closes Gateway, professional ritual, professional read-record, wake-word,
and voice-chain selector evidence. The remaining local software-side gap is
`roleplay_voice_runtime`: the live `roleplay-voice-probe` observes prompt input
and audio downlink, but the default selected voice profile is not traced as
used and the voice-pipeline branch does not record the host/simulator playback
start boundary.

Selecting `a21_voice_clone_default` proves the clone-profile marker but falls
back locally when the clone TTS runtime is unavailable. The default safe
`a21_voice_default_dashscope` profile can produce local audio and should still
prove that the selected voice profile was carried into the voice pipeline.

## Target State

- Fast Companion roleplay voice-pipeline traces record
  `roleplay.voice_clone_profile.used` when a safe selected voice profile is
  passed into the provider-neutral voice pipeline.
- The same branch records `device.playback.start` when the Gateway emits the
  first host/simulator playback chunk.
- `roleplay-voice-probe --require-ready` can produce a passed no-hardware
  runtime report against the local Gateway without claiming physical StackChan
  acceptance.

## Out Of Scope

- No firmware build, flash, NVS, serial, ECS deployment, physical hardware
  action, provider-key change, V21 repository change, or real clone-audio
  quality acceptance.
- No weakening of the physical PRD gate; reports must keep
  `physical_accepted=false` and `prd_accepted=false`.

## Acceptance

- Focused Gateway tests prove selected default voice profile trace and
  playback-start boundary in the voice-pipeline branch.
- Focused app tests prove `roleplay-voice-probe` and server-side collection can
  ingest the fresh live report.
- Runtime evidence from local Gateway shows server-side readiness missing only
  `provider_smoke` and physical acceptance.

