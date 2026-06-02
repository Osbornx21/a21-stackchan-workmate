# 2026-06-03 - Xiaozhi Streaming TTS Adapter

Status: active.
Owner: A21 control tower plus scoped workers.
Transition: `T-XIAOZHI-STREAMING-TTS-ADAPTER-001`.

## Background And Problem

The user's review is correct: Xiaozhi voice quality and latency come from a
realtime media pipeline, not from a normal request/response chatbot loop. A21
has moved closer to the Xiaozhi shape on transport and ASR seams, but current
TTS still contains a file boundary:

`LLM segment -> TTS provider/local synthesizer -> complete WAV/file -> read WAV
as 60 ms chunks -> Gateway Opus downlink`.

The `TTSAdapter` interface already returns a channel, and Gateway can stream
chunks from `VoicePipelineRunner.RunStream()` into stock Xiaozhi Opus downlink.
However, `localTTSAdapter.Synthesize()` blocks until a complete WAV exists
before returning chunks. This keeps the architecture below Xiaozhi parity.

## Current System State

- `internal/gateway/server.go` sends stock Xiaozhi `tts` lifecycle messages and
  binary Opus downlink on the long `/v1/xiaozhi` WebSocket.
- `VoicePipelineRunner.RunStream()` already emits audio chunks as soon as a TTS
  adapter yields them.
- `localTTSAdapter` still creates a complete WAV and reads it back into
  24 kHz mono 60 ms chunks.
- `DoubaoRealtimeTTSProvider` already has a realtime WebSocket session shape:
  session update, `input_text.append`, `input_text.done`, and
  `response.audio.delta` mapping.
- Doubao realtime audio deltas are provider-sized PCM chunks, not guaranteed to
  already match Xiaozhi's 60 ms chunk size, so A21 must frame them into 60 ms
  PCM chunks before Gateway Opus encoding.
- `xiaozhi-streaming-provider-readiness` currently blocks TTS profiles such as
  `iflytek_tts`, `voice_clone_cli`, and local Sherpa as WAV/file boundaries.

## Target State

- Add an explicit streaming TTS adapter path that does not require a complete
  WAV/file before yielding audio.
- Reuse existing realtime TTS provider/session primitives where possible.
- Add a small PCM16 mono chunker that can accumulate provider audio deltas and
  emit exact 60 ms downlink-ready PCM chunks.
- Add a selectable TTS profile such as `doubao_tts_realtime` that can be marked
  as streaming by the static provider gate only when required env is present.
- Preserve all current local/WAV TTS adapters as blocked by the strict Xiaozhi
  realtime gate.

## Non-Goals

- Do not flash firmware.
- Do not start, stop, or restart Gateway.
- Do not call real Doubao/Iflytek/5080 providers in this transition.
- Do not change TTS gain, codec volume, wake word, setup/no-welcome, or NVS.
- Do not claim physical PRD acceptance or full Xiaozhi realtime parity.
- Do not remove local TTS or voice clone capability; keep them as fallback or
  non-realtime candidates.

## Impact Scope

- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Execution Steps

1. Add a streaming TTS capability marker or interface in providers so tests and
   readiness code can distinguish true streaming adapters from channel-shaped
   WAV adapters.
2. Add a PCM16 mono delta chunker that accepts arbitrary base64 PCM deltas and
   emits exact `pcm_s16le`, 24 kHz or configured sample-rate, mono, 60 ms
   `VoiceAudioChunk` values.
3. Add `DoubaoRealtimeTTSTTSAdapter` around the existing
   `DoubaoRealtimeTTSProvider`.
4. The adapter must:
   - start a realtime TTS session;
   - send the text delta;
   - send text done;
   - read server events until EOF/context cancel;
   - map audio delta events to PCM chunks;
   - frame PCM into 60 ms chunks;
   - close the provider session;
   - return stable redacted errors.
5. Extend `VoicePipelineAdaptersFromEnv` so
   `A21_TTS_FAST_PROFILE=doubao_tts_realtime` selects this adapter.
6. Extend `xiaozhi-streaming-provider-readiness` so:
   - local/WAV profiles remain blocked with
     `tts_wav_file_boundary_not_xiaozhi_streaming`;
   - `doubao_tts_realtime` is streaming and implemented in Gateway;
   - `doubao_tts_realtime` is ready only when required env is configured;
   - reports store only env names and booleans, not secrets, provider output,
     transcript, audio payload, full URLs, or local paths.
7. Add focused tests using fake realtime conn/server messages. No provider
   network call is allowed.

## Acceptance Criteria

- Provider tests prove the realtime TTS adapter:
  - writes `tts_session.update`, `input_text.append`, and `input_text.done`;
  - reads provider `response.audio.delta` events from a fake conn;
  - yields downlink-ready PCM chunks before any WAV/file exists;
  - emits exact 60 ms chunks accepted by Gateway;
  - closes the realtime session;
  - does not leak text, secrets, URLs, or raw/base64 audio in reports/errors.
- Existing local TTS tests still prove local TTS is WAV-based and therefore not
  Xiaozhi realtime.
- Readiness tests prove:
  - `iflytek_tts` remains blocked as WAV/file boundary;
  - `doubao_tts_realtime` without env is blocked as missing config;
  - `doubao_tts_realtime` with required env can make the TTS stage ready;
  - total gate still blocks if ASR is mock or helper proof is missing.
- Commands pass:
  - `go test ./internal/providers -run 'TestDoubaoRealtimeTTS|TestVoicePipelineAdaptersFromEnv.*TTS|TestLocalTTSAdapter' -count=1`
  - `go test ./internal/app -run TestXiaozhiStreamingProviderReadiness -count=1`
  - `git diff --check`
  - `make verify`

## Rollback

- Revert the additive realtime TTS adapter/profile/readiness changes.
- Existing accepted 3x gain, no-welcome firmware, ASR seam, and local/voice
  clone fallback remain unaffected.

## Risks

- Provider audio deltas may not align to 60 ms; the chunker must buffer and pad
  only at final drain.
- A configured realtime TTS profile is still static/no-execute evidence until a
  future provider smoke and physical `/v1/xiaozhi` trace prove runtime behavior.
- Doubao realtime TTS may not be the final contest provider; the adapter shape
  should be provider-neutral enough to reuse for Iflytek or 5080 streaming
  later.
- If the adapter returns a channel but internally waits for EOF before yielding,
  the change would recreate the same false-streaming problem. Tests must assert
  first audio appears after the first provider delta, before final EOF.

## Manual Confirmation Points

- None for this no-provider, no-hardware transition.
- Physical listening acceptance happens only after ASR and TTS streaming are
  both runtime-proven and an operator-triggered stock `/v1/xiaozhi` trace is
  captured.

## Worker Return Format

Workers must return:

- This stage did what;
- Files changed;
- Tests run and results;
- Whether it deviated from plan;
- Remaining blockers;
- Recommended next step.
