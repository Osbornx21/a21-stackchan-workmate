# 2026-06-03 - Xiaozhi Streaming Provider Readiness

Status: active.
Owner: A21 control tower.
Transition: `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`.

## Background And Problem

The device side now speaks the stock Xiaozhi WebSocket shape: JSON control and
binary Opus frames share the long `/v1/xiaozhi` socket. A21 also has a Gateway
seam that can start a streaming ASR session on Xiaozhi `listen.start` and feed
decoded PCM frames as Opus arrives.

The remaining gap is provider reality. Current recorded evidence shows real
Opus ingress, mock streaming ASR seam tests, route-eligible text-stream provider
evidence, and better Iflytek TTS playback. It does not yet prove the full
Xiaozhi-style chain:

`local wake/VAD -> Opus frames -> streaming ASR -> streaming LLM -> streaming TTS -> paced Opus downlink`

The risk is that `streaming_zipformer` model names, `/v1/xiaozhi/say`, mock
streaming tests, or WAV-based Iflytek TTS are mistaken for realtime voice parity.

## Current System State

- Gateway `/v1/xiaozhi` decodes incoming stock Opus frames and can append PCM
  frames to a `providers.StreamingASRSession` when the selected ASR adapter
  implements `providers.StreamingASRAdapter`.
- The existing Sherpa adapter writes accumulated PCM to a temporary WAV and
  runs the Python smoke runner. Even when the model family is
  `streaming_zipformer`, this is still a file/batch boundary.
- The existing Iflytek TTS code uses a WebSocket provider, but the A21 adapter
  collects provider PCM into a WAV report and only then re-reads it into 24 kHz
  chunks. That is not Xiaozhi-style immediate TTS downlink.
- The realtime parity CLI already rejects `/say`, fast-companion, and local
  fallback traces, but it needs a provider-readiness companion gate so a future
  model cannot claim provider readiness without true streaming adapters.

## Target State

- Add a static, non-executing readiness gate that reports whether the currently
  selected ASR/LLM/TTS profiles satisfy Xiaozhi realtime provider requirements.
- The gate must block current batch/WAV/mock paths and explain exactly why.
- The gate must be redaction-safe: no transcript, raw audio, provider output,
  credential values, full URLs, or local absolute paths in reports.
- The gate must feed the state machine and handoff log so future workers know
  the next required implementation step.

## Non-Goals

- Do not execute providers or use credentials in this transition.
- Do not flash firmware, restart Gateway, or play audio.
- Do not claim product readiness or wake acceptance.
- Do not implement the full Iflytek IAT or streaming TTS adapter in this same
  step.

## Impact Scope

- `internal/app/app.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Execution Steps

1. Add CLI `a21 xiaozhi-streaming-provider-readiness`.
2. Inspect `providers.VoicePipelineSelectionFromEnv` and static profile names.
3. Instantiate adapters only to check local interface support, not to execute.
4. Classify:
   - mock ASR as fixture-only;
   - Sherpa ASR as batch/WAV, even for `streaming_zipformer`;
   - Iflytek/cloud ASR profiles as missing adapter until implemented;
   - text-stream LLM as stream-capable only when a configured text-stream
     profile exists;
   - local/Iflytek/voice-clone TTS as file/WAV boundary until a streaming TTS
     adapter emits chunks as provider audio arrives.
5. Return non-zero unless ASR, LLM, and TTS all satisfy the strict streaming
   provider contract.
6. Add tests for default mock, Sherpa+StepFun+Iflytek, and a future fully
   streaming fixture profile.
7. Update state/handoff docs.

## Acceptance Criteria

- Focused app tests pass.
- `git diff --check` passes.
- The default environment reports `blocked`.
- `A21_ASR_LOCAL_PROFILE=sherpa_onnx` plus `A21_TTS_FAST_PROFILE=iflytek_tts`
  still reports `blocked` with explicit WAV/batch findings.
- A test-only fixture profile can prove the gate turns green only when all
  three stages are marked true streaming.

## Rollback

- Revert the new CLI/report and docs only. It is static/read-only and does not
  alter runtime behavior.

## Risks

- Static gates can drift as adapters evolve. Keep the gate tied to explicit
  adapter/profile contracts and update tests when a real streaming adapter
  lands.
- This transition produces a truthful blocker, not a usable voice improvement
  by itself. The next implementation transition must wire a real streaming ASR
  provider or long-lived local streaming runner.

## Manual Confirmation Points

- None for this static gate. Physical confirmation comes after a real provider
  adapter and `xiaozhi-realtime-parity` trace prove the chain.
