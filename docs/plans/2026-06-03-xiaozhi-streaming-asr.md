# T-XIAOZHI-STREAMING-ASR-001 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move stock `/v1/xiaozhi` from turn-buffered ASR toward Xiaozhi-style
streaming ASR by pre-opening an ASR session on listen start, feeding decoded PCM
frames as they arrive, and using VAD/listen stop only to commit/finalize.

**Architecture:** Add an optional streaming ASR interface beside the existing
batch `ASRAdapter`. Gateway starts a streaming session per Xiaozhi listen turn
when the selected ASR adapter supports it; decoded Opus frames continue to be
recorded as `voicePipelineFrames` for fallback, but are also fed to the session.
If streaming ASR is unavailable or fails before finalization, the current
turn-buffered `VoicePipelineRunner` remains the fallback.

**Tech Stack:** Go, existing A21 Gateway `/v1/xiaozhi`, existing Opus decoder,
existing `audioIngress` VAD, existing `providers.VoicePipeline*` types.

---

## Background and Problem Definition

Current A21 product path uses a stock-shaped Xiaozhi WebSocket and binary Opus
transport, but host ASR still starts after `listen.stop`, VAD speech end, or max
duration. The decoded PCM frames are accumulated in `session.voicePipelineFrames`
and then passed to `ASRAdapter.Transcribe`.

That is not the Xiaozhi target. The target is:

`mic/i2s -> local wake/VAD -> Opus frames -> WebSocket -> streaming ASR session
-> streaming text -> streaming/segmented TTS -> paced Opus downlink`.

## Current System State

- `internal/gateway/server.go` decodes each binary Opus frame in
  `handleXiaozhiBinary`, then calls `observeXiaozhiDecodedIngress`.
- `observeXiaozhiDecodedIngress` appends a `providers.VoicePipelinePCMFrame`
  to `session.voicePipelineFrames` and records VAD markers.
- `listen.stop` and `maybeAutoStopXiaozhiTurnOnIngress` call
  `newXiaozhiTurnTask` and `startXiaozhiTurnTask`.
- `writeXiaozhiVoicePipelineTTS` then calls the voice pipeline runner with all
  accumulated frames.
- `providers.ASRAdapter` is batch-only:
  `Transcribe(ctx, ASRAdapterRequest{Frames: ...})`.
- `xiaozhi-realtime-parity` can now distinguish transport-only,
  turn-buffered-candidate, and streaming-candidate traces.

## Target State

Phase 1 target for this plan:

- A new optional `providers.StreamingASRAdapter` and `StreamingASRSession`
  interface exists.
- A fake/mock streaming ASR adapter can emit `asr.first_partial` before
  `listen.stop` or `vad.speech.end` in `/v1/xiaozhi` Gateway tests.
- Gateway records new trace markers:
  - `asr.stream.start`;
  - `asr.audio.append`;
  - `asr.first_partial`;
  - `asr.final`;
  - `asr.stream.cancelled` or `asr.stream.unavailable` when applicable.
- Batch ASR fallback still works when no streaming adapter is available.
- `xiaozhi-realtime-parity` can classify the fake streaming trace as
  `xiaozhi_realtime_candidate`.

Later phases:

- Replace the fake session with real streaming ASR provider adapters.
- Add real TTS streaming evidence; local WAV-based TTS remains
  `file_chunked`, not true streaming.

## Non-Goals

- Do not remove the current batch ASR path.
- Do not claim real cloud/local streaming ASR until a real provider session is
  implemented and credentialed smoke passes.
- Do not use `/v1/xiaozhi/say`, `xiaozhi-voice-bench`, or `/ws/audio` as a
  substitute for stock `/v1/xiaozhi` evidence.
- Do not flash firmware in this transition.
- Do not change TTS gain, speaker volume, wake phrase, or setup/no-welcome
  behavior.
- Do not store raw transcripts, prompts, provider output, secrets, or raw audio
  in reports.

## Impact Range

- `internal/providers/voice_pipeline.go`
  - Add optional streaming ASR interfaces and mock implementation.
- `internal/providers/voice_pipeline_test.go`
  - Test streaming ASR session behavior if provider-only helpers are added.
- `internal/gateway/server.go`
  - Track one streaming ASR session per `xiaozhiSession`.
  - Start/feed/commit/cancel the session at existing listen/ingress/stop/abort
    points.
- `internal/gateway/server_test.go`
  - Add stock `/v1/xiaozhi` tests proving ASR partial before stop/VAD end.
- `internal/app/xiaozhi_realtime_parity.go`
  - Add any missing marker awareness only if required by tests.
- `docs/project_state_machine.md`
  - Add active transition state.
- `docs/agent_handoff_log.md`
  - Record result and next handoff.

## Task 1: Provider Streaming ASR Interfaces

**Files:**

- Modify: `internal/providers/voice_pipeline.go`
- Test: `internal/providers/voice_pipeline_test.go`

- [ ] **Step 1: Add interface types**

Add beside `ASRAdapter`:

```go
type StreamingASRAdapter interface {
	Name() string
	StartStreamingASR(ctx context.Context, req StreamingASRStartRequest) (StreamingASRSession, error)
}

type StreamingASRStartRequest struct {
	Session VoiceSession
	Mode    string
}

type StreamingASRSession interface {
	AppendFrame(ctx context.Context, frame VoicePipelinePCMFrame) error
	Events() <-chan ASRAdapterEvent
	Commit(ctx context.Context) error
	Cancel(error)
}
```

- [ ] **Step 2: Add mock streaming adapter**

Add a constructor used by tests:

```go
func NewMockStreamingASRAdapter(name string) StreamingASRAdapter
```

Behavior:

- `StartStreamingASR` opens an event channel.
- First `AppendFrame` with `RMS > 0` emits one partial
  `ASRAdapterEvent{Text: "fixture transcript should never be stored"}`.
- `Commit` emits final
  `ASRAdapterEvent{Text: "fixture transcript should never be stored", Final: true}`
  and closes the channel.
- `Cancel` closes the channel without final.

- [ ] **Step 3: Test mock session ordering**

Run:

```bash
go test ./internal/providers -run TestMockStreamingASRAdapter -count=1
```

Expected: pass, proving partial emits after `AppendFrame` and final emits only
after `Commit`.

## Task 2: Gateway Session Lifecycle

**Files:**

- Modify: `internal/gateway/server.go`
- Test: `internal/gateway/server_test.go`

- [ ] **Step 1: Extend `xiaozhiSession`**

Add fields:

```go
streamingASRSession providers.StreamingASRSession
streamingASRFinalText string
streamingASRHasPartial bool
streamingASRHasFinal bool
streamingASRUnavailable bool
```

Reset those fields in `resetXiaozhiOpusIngress`.

- [ ] **Step 2: Start session on listen start**

After `session.resetXiaozhiOpusIngress()` and turn start on listen start, call a
new helper:

```go
s.startXiaozhiStreamingASR(ctx, session, mode)
```

The helper:

- gets the current voice pipeline runner;
- if its ASR adapter implements `providers.StreamingASRAdapter`, starts a
  streaming session;
- records `asr.stream.start`;
- launches a goroutine reading session events and records `asr.first_partial`
  and `asr.final`;
- stores final text on the `xiaozhiSession` under lock.

If unavailable, record nothing or `asr.stream.unavailable` only in debug tests;
do not spam stock traces.

- [ ] **Step 3: Feed decoded frames**

In `observeXiaozhiDecodedIngress`, after building the
`VoicePipelinePCMFrame`, call:

```go
s.appendXiaozhiStreamingASRFrame(ctx, session, pipelineFrame)
```

The helper records `asr.audio.append` only when a streaming session exists.

- [ ] **Step 4: Commit on VAD/listen stop**

Before starting the existing turn task on `listen.stop` and
`maybeAutoStopXiaozhiTurnOnIngress`, call:

```go
s.commitXiaozhiStreamingASR(ctx, session)
```

It must be idempotent and must not block indefinitely.

- [ ] **Step 5: Cancel on abort/barge-in/socket close**

When a turn is aborted or the connection closes, cancel the streaming ASR
session so goroutines do not leak.

## Task 3: Use Streaming ASR Final In Voice Pipeline

**Files:**

- Modify: `internal/gateway/server.go`
- Modify: `internal/providers/voice_pipeline.go` only if the request needs an
  explicit transcript field.
- Test: `internal/gateway/server_test.go`

- [ ] **Step 1: Carry final transcript into turn task**

Add to `xiaozhiTurnTask`:

```go
streamingASRFinalText string
streamingASRUsed bool
```

Populate it in `newXiaozhiTurnTask`.

- [ ] **Step 2: Avoid batch ASR when streaming final exists**

The minimal acceptable first implementation can still call existing
`RunStream`, but it must not re-run batch ASR when a streaming final transcript
is available. Add one of:

- a `VoicePipelineRequest.Transcript` field used by `VoicePipelineRunner`; or
- a lightweight Gateway runner path that starts text streaming with the known
  transcript.

Acceptance: a test adapter can assert `ASRAdapter.Transcribe` is not called
when streaming final exists.

- [ ] **Step 3: Preserve fallback**

If streaming ASR is unavailable, errors, or produces no final, keep the current
`Frames` batch path unchanged.

## Task 4: Gateway Trace Contract Tests

**Files:**

- Modify: `internal/gateway/server_test.go`
- Modify: `internal/app/xiaozhi_realtime_parity_test.go` only if marker
  semantics change.

- [ ] **Step 1: Add `/v1/xiaozhi` streaming ASR ordering test**

Test name:

```go
TestXiaozhiWebSocketStreamingASRStartsBeforeListenStop
```

Scenario:

- configure a test runner/adapters with mock streaming ASR;
- connect to `/v1/xiaozhi`;
- send hello;
- send listen start;
- send one speech Opus frame;
- inspect trace before sending listen stop;
- require `asr.stream.start`, `asr.audio.append`, and `asr.first_partial`;
- require no `xiaozhi.voice_pipeline.start` before stop.

- [ ] **Step 2: Add VAD-end finalization test**

Use existing silence frames to trigger VAD speech end. Require:

- `asr.first_partial` occurs before `vad.speech.end`;
- `asr.final` occurs after commit;
- downlink still reaches `audio.downlink.first_frame`.

- [ ] **Step 3: Add fallback test**

With the existing non-streaming mock ASR, require the old batch path still
passes and trace classification remains `turn_buffered_xiaozhi_candidate`.

## Task 5: Evidence and State Update

**Files:**

- Modify: `docs/project_state_machine.md`
- Modify: `docs/agent_handoff_log.md`

- [ ] **Step 1: Record phase result**

State that Phase 1 proves the Gateway can run streaming-ASR semantics with a
fake adapter, not that real ASR provider parity is complete.

- [ ] **Step 2: Record next transition**

Add next transition:

`T-XIAOZHI-STREAMING-ASR-PROVIDER-001`

Goal: connect a real streaming ASR provider or a proven local streaming engine
without falling back to WAV/file boundaries.

## Verification Commands

Run after implementation:

```bash
go test ./internal/providers -run 'TestMockStreamingASRAdapter' -count=1
go test ./internal/gateway -run 'TestXiaozhiWebSocketStreamingASR|TestXiaozhiWebSocket(VADSpeechEndAutoStopsRealtimeTurn|ListenStopRunsVoicePipelineAndSendsPacedOpus)' -count=1
go test ./internal/app -run 'TestXiaozhiRealtimeParity' -count=1
make verify
git diff --check
```

## Rollback Plan

- If streaming ASR session lifecycle creates instability, revert the Gateway
  lifecycle changes and keep provider interfaces/tests if they remain unused and
  passing.
- If interfaces are wrong, remove the new interfaces and return to the
  evidence-only state `ae4f5da`.
- No firmware or NVS rollback is involved.

## Risks

- A fake streaming adapter can prove architecture but not real provider
  readiness.
- Blocking on `Commit` can stall the Xiaozhi socket; commits must be bounded or
  asynchronous.
- ASR event goroutines can leak if abort/socket close does not cancel sessions.
- Re-running batch ASR after streaming final would erase the latency benefit.
- Local Sherpa WAV ASR is not streaming and must remain a fallback.

## Human Confirmation Needed

- After fake streaming tests pass, the user must choose or authorize the real
  streaming ASR provider/local engine for `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`
  unless an already configured provider profile proves official streaming
  support.
