# Xiaozhi Opus Ingress Queue

Status: completed host-local cut.
Created: 2026-06-03 10:36 CST.

## Transition

`T-XIAOZHI-OPUS-INGRESS-QUEUE-001`

## Current State

A21's stock `/v1/xiaozhi` path parses binary Opus frames, decodes them to PCM,
pushes VAD/audio-ingress evidence, and appends frames to streaming ASR. The
path now also preserves stock `stt` ordering and bounded wake pre-roll. The
remaining host-side mismatch is that listening Opus frame processing still
happens inline on the WebSocket read loop.

## Target State

Move listening Opus frame decode/VAD/streaming-ASR append behind a bounded
per-session ingress queue so JSON control frames such as `abort` can still be
read while audio processing or ASR append is slow. This matches Xiaozhi's
realtime device shape more closely: audio frames flow through bounded queues,
while control state remains responsive.

## Trigger

The full realtime objective requires:

`mic/i2s -> wake/VAD -> Opus frame queue -> WebSocket or UDP streaming ->
streaming ASR -> streaming LLM -> streaming TTS -> Opus frame playback`

The current A21 path has WebSocket streaming and streaming ASR hooks, but lacks
the explicit Opus frame queue boundary on the host ingress side.

## Scope

- Host-local Gateway code only.
- Add a failing Gateway test first.
- Preserve stock physical behavior and cooldown/wake pre-roll semantics.
- Keep raw audio and transcript privacy unchanged.

## Non-Scope

- No provider execution.
- No V21 execution.
- No Gateway start/stop.
- No `/v1/xiaozhi/say`.
- No host-loopback runtime acceptance.
- No firmware build, flash, NVS, serial, hardware action, or audio playback.
- No UDP/MQTT implementation in this cut.

## Proposed Action

- Add a bounded per-session Opus ingress queue for listening frames.
- `handleXiaozhiBinary` should parse/adopt the frame, record enqueue evidence,
  and return without waiting for Opus decode, VAD, or streaming ASR append.
- A queue worker should preserve frame order, decode and observe ingress using
  the existing logic, append to streaming ASR, and record drops if the queue is
  full.
- Stop/abort/socket close should cancel or drain the worker without blocking
  the WebSocket read loop.

## Acceptance

- Red test proves a blocking streaming ASR `AppendFrame` prevents `abort` from
  being processed before the queue change.
- Focused Gateway test proves `abort` is recorded while the audio queue worker
  is blocked on ASR append.
- Existing stock STT, wake pre-roll, streaming ASR, partial bridge, nonblocking
  commit, and paced Opus downlink tests still pass.
- `go test ./internal/gateway -count=1` passes.
- `git diff --check` passes.
- `make verify` passes.

## Completed Work

- Added a bounded per-session Opus ingress queue for listening `/v1/xiaozhi`
  frames.
- `handleXiaozhiBinary` now parses/adopts incoming Opus, records receipt,
  enqueues the frame, and returns to the WebSocket read loop without waiting
  for Opus decode, VAD/audio ingress, or streaming ASR append.
- Added a queue worker that preserves per-session frame order, decodes Opus,
  pushes audio ingress/VAD evidence, appends frames to streaming ASR, and
  records explicit queue drops when the bounded queue is full.
- `listen.stop` now starts an async stop-finalize path: it waits briefly for
  queued ingress to catch up, then commits streaming ASR or starts the voice
  pipeline. The control WebSocket read loop remains free to process `abort`.
- Queue context is canceled on socket close and replaced on new turns.

## Verification

- Red Gateway test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketOpusAppendDoesNotBlockAbortControlFrame -count=1`
  failed because `xiaozhi.abort.received` was not recorded while streaming ASR
  `AppendFrame` was blocked.
- Focused Xiaozhi Gateway tests passed:
  `go test ./internal/gateway -run 'TestXiaozhiWebSocket(OpusAppendDoesNotBlockAbortControlFrame|WakePrerollOpusFeedsNextTurn|StreamingASRFinalSendsStockSTTBeforeTTS|StreamingASRFinalStartsPipelineWithoutBatchFallback|ListenStopDoesNotBlockAbortWhileStreamingASRCommitPending|ListenStopRunsVoicePipelineAndSendsPacedOpus|StreamingASRStartsBeforeListenStop|ASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal)' -count=1`.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Focused app parity/readiness tests passed:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity|TestXiaozhiStreamingProviderReadiness' -count=1`.
- `git diff --check` passed.
- `make verify` passed.

## Boundaries Observed

- No provider execution.
- No V21 execution.
- No Gateway start/stop.
- No `/v1/xiaozhi/say`.
- No host-loopback runtime acceptance.
- No firmware build, flash, NVS, serial, hardware action, or audio playback.

## Failure State

- The queue drops or reorders frames without trace evidence.
- `listen.stop`/`abort` can race with frame processing and start stale pipeline
  work after the turn is cancelled.
- Existing wake pre-roll or cooldown suppression regresses.

## Rollback

Revert only this transition's queue code, tests, plan, state-machine entry, and
handoff entry. Prior stock STT, wake pre-roll, nonblocking ASR commit, partial
bridge, real-profile gate, Sherpa smoke, and realtime TTS seam remain intact.

## Next State

If this cut passes, continue with either physical stock `/v1/xiaozhi` evidence
collection in an approved runtime/hardware window or an authorized real
streaming TTS provider runtime proof. Full PRD acceptance remains false until
real streaming ASR/LLM/TTS profile markers, physical playback, wake/tap,
barge-in/touch, and idle recovery are proven.
