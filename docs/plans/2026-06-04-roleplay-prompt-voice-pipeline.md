# A21 Roleplay Prompt Voice Pipeline Cut

Date: 2026-06-04

Status: implemented and verified.

Transition: `T-ROLEPLAY-PROMPT-VOICE-PIPELINE-001`

## Current State

- A21 exposes roleplay profile, scenario, voice-clone profile, and bounded
  runtime memory-hint controls.
- Gateway composes roleplay prompt input as readiness proof, but the selected
  personality/scenario/memory prompt is not yet guaranteed to reach the actual
  low-latency voice pipeline text provider.
- The stock Xiaozhi device-side state machine, Opus path, and internal test 3
  voice behavior must not be changed.

## Target State

- Roleplay voice turns pass a composed A21 personality prompt to the text
  stream provider while keeping raw prompt, memory text, transcript, provider
  output, URLs, paths, credentials, and audio payloads out of reports/traces.
- Fast companion and stock `/v1/xiaozhi` roleplay paths both use the same
  provider-neutral prompt-input mechanism.
- Reports expose only readiness/count metadata, not prompt bodies.

## Not Doing

- No durable long-term memory, account sync, V21 retrieval, document upload,
  provider execution, Gateway runtime start, firmware build, flash, serial,
  NVS, or physical hardware action.
- No new production dependency or stack rewrite.
- No change to professional mode/V21 routing.

## Impact Scope

- `internal/providers/voice_pipeline.go`
- `internal/gateway/server.go`
- Focused Gateway/provider tests.
- Protocol/current-control/state/handoff docs.

## Acceptance

- A fake text-stream adapter can prove it receives a prompt containing the
  roleplay scenario/memory-instruction structure rather than only a raw
  transcript.
- Fast-companion responses and traces still do not leak raw prompt or memory
  text.
- Voice-pipeline reports mark prompt input as present and not recorded.
- Existing roleplay, MCP, and voice-pipeline tests still pass.

## Failure State

- Prompt or memory text appears in any API response, trace, report, handoff, or
  test fixture output that is intended to be redacted.
- Professional/V21 execution occurs from a roleplay turn.
- Internal test 3 Xiaozhi protocol/voice behavior regresses.

## Rollback

- Remove the prompt-input field from `VoicePipelineRequest`.
- Revert Gateway prompt injection while keeping roleplay profile and memory
  control endpoints intact.

## Next State

- Persona-quality and physical roleplay evidence can be tested against actual
  voice turns instead of only configuration summaries.

## Implementation Result

- `VoicePipelineRequest.TextPrompt` carries runtime-only prompt input.
- The voice pipeline sends `TextPrompt` to the text-stream adapter when present.
- `VoicePipelineReport` records only `prompt_input_ready` and
  `prompt_input_not_recorded`.
- Fast-companion roleplay turns with PCM frames inject the selected
  personality/scenario/memory prompt before provider text streaming.
- Stock `/v1/xiaozhi` roleplay turns inject the same prompt before provider
  text streaming.
- Responses/traces continue to avoid prompt bodies, memory text, transcripts,
  provider output, URLs, paths, credentials, and raw/base64 audio.

## Verification

- `go test ./internal/providers ./internal/gateway -run 'VoicePipelineRunnerUsesPromptInput|VoicePipelineRunnerProducesDownlinkReady|FastCompanionHybridRunsVoicePipelineWhenFramesProvided|XiaozhiWebSocketListenStopRunsVoicePipelineAndSendsPacedOpus|RoleplayProfileEndpointSetsRuntimeMemoryHints|RoleplayProfileEndpointPersistsScenarioVoiceClone' -count=1`:
  passed.
- `go test ./internal/providers ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
