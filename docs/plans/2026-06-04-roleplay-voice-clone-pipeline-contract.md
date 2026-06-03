# 2026-06-04 - Roleplay Voice Clone Pipeline Contract

Status: completed host-local runtime cut.
Owner: A21 control tower.
Transition: `T-ROLEPLAY-VOICE-CLONE-PIPELINE-CONTRACT-001`.
Created: 2026-06-04 CST.

## Goal

Close the internal test 4 roleplay gap where the selected
`voice_clone_profile` updated the voice-chain selector but was not carried as
per-turn evidence into the provider-neutral voice pipeline and TTS request.

## Boundary

- Gateway and provider-neutral voice pipeline only.
- No provider execution, no V21 execution, no Gateway service start, no
  firmware build, no flash, no serial, no NVS, and no physical hardware action.
- Do not store prompt text, transcripts, provider output, raw/base64 audio,
  reference audio paths, reference text, voice samples, URLs, local paths, or
  credentials.

## Transition

Current state:

- `/v1/roleplay-profile` can select `voice_clone_profile`.
- Selecting a clone profile maps the effective TTS profile to
  `voice_clone_cli`.
- Roleplay prompt input reaches the text-stream request, but the selected A21
  voice profile is not explicit in `VoicePipelineRequest`, `TTSAdapterRequest`,
  or `VoicePipelineReport`.

Target state:

- `VoicePipelineRequest` carries a runtime-only `VoiceCloneProfile`.
- `TTSAdapterRequest` receives the same safe A21 voice profile ID.
- Local TTS adapters pass the safe profile ID through `LocalTTSOptions.Voice`
  so `voice_clone_cli` can bind the current turn to the selected voice identity
  without exposing sample configuration.
- `VoicePipelineReport.Input` exposes only the safe profile ID and
  `voice_clone_sample_not_recorded`.
- Fast-companion and stock `/v1/xiaozhi` roleplay turns both trace
  `roleplay.voice_clone_profile.used` when a clone profile is selected.

Acceptance condition:

- Focused provider and Gateway tests prove the selected clone profile reaches
  the TTS boundary and stock Xiaozhi voice-pipeline request.
- Reports/traces do not leak prompt text, memory text, transcripts, provider
  output, reference audio/text, local paths, URLs, credentials, voice samples,
  or raw/base64 audio.
- Internal test 3 Xiaozhi audio protocol behavior is not changed.

Rollback path:

- Revert only this scoped runtime-contract commit. Keep the earlier roleplay
  profile, memory, prompt, voice-chain, V21 adapter, and hardware parity
  records intact.

## Verification

- `go test ./internal/providers ./internal/gateway -run 'VoicePipelineRunnerPassesVoiceCloneProfile|VoicePipelineAdaptersFromEnvSelectsVoiceCloneCLI|FastCompanionHybridRunsVoicePipelineWhenFramesProvided|XiaozhiWebSocketListenStopRunsVoicePipelineAndSendsPacedOpus|RoleplayProfileEndpointPersistsScenarioVoiceClone' -count=1`

Broader verification is recorded in the handoff entry for this transition.
