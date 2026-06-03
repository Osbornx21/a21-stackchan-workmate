# 2026-06-03 - Xiaozhi Provider State Machine Rebuild

Status: active control plan.
Owner: A21 control tower.
Transition: `T-XIAOZHI-PROVIDER-STATE-MACHINE-REBUILD-001`.

## Goal

Rebuild the A21 public `/v1/xiaozhi` voice chain around the official Xiaozhi
device state machine and provider realtime event machines so one physical
StackChan turn can become low-latency, clear, interruptible, and wake-capable
without treating host-loopback or static readiness as PRD green.

## Current Root Cause

The transport/audio work from the Mac Gateway was not wasted. A21 still has
stock-shaped Xiaozhi WebSocket, Opus ingress/downlink, queued ingress, ASR
partial reuse, nonblocking ASR commit, and barge-in cancellation surfaces.

The public Gateway break is lower in the provider bridge:

- DashScope ASR can produce partial/final in public bench.
- LLM can produce first content in traces.
- DashScope TTS currently behaves like a synchronous RPC: it writes
  `session.update`, `input_text_buffer.append`, `input_text_buffer.commit`,
  `session.finish`, and only then starts reading.
- Official Qwen-TTS Realtime is an event machine: connection returns
  `session.created`; client updates session; client appends text and commits;
  server returns `response.created`, `response.audio.delta`, then
  `response.audio.done`/`response.done`; `session.finish` is the cleanup phase,
  not the trigger that should precede reading audio.
- Doubao-style realtime TTS follows the same stable pattern:
  `tts_session.update` plus `input_text.append`/`input_text.done` while a
  receive loop drains `response.audio.delta` until `response.audio.done`.

This means the next work must rebuild provider state handling, not keep
patching fallback strings.

## Official State Machines To Preserve

### Xiaozhi Device / Gateway State

`Idle -> Connecting -> Listening -> Speaking -> Idle`

- Device opens WebSocket and sends `hello`.
- Server replies `hello`.
- Device sends `listen/start`; microphone Opus frames stream as binary.
- Server sends `stt` once text is available.
- Server sends `tts/start`; device enters Speaking and stops mic streaming.
- Server sends one or more `tts/sentence_start` messages and binary Opus audio.
- Server sends `tts/stop`; device returns to Idle or configured auto-listen.
- Device abort/new wake/touch during Listening or Speaking must cancel current
  turn, stop provider/downlink work, and return to a stable state.

### Qwen-ASR Realtime

- Open WebSocket with model query and bearer auth.
- Send `session.update` with `turn_detection: null` for manual push-to-talk
  mode.
- Start the read loop immediately after connect/session update.
- Append small PCM chunks through `input_audio_buffer.append`.
- On speech end/listen stop, send `input_audio_buffer.commit`.
- Send `session.finish` only after audio is complete and the session should
  close.
- Consume partial/final transcription events until
  `conversation.item.input_audio_transcription.completed` or `session.finished`.

### Qwen-TTS Realtime

- Open WebSocket with model query and bearer auth.
- Start read loop before or immediately after `session.update`; do not wait
  until all writes finish.
- Send `session.update` and wait for `session.updated` when practical.
- In commit mode, send `input_text_buffer.append`, then
  `input_text_buffer.commit` to trigger synthesis.
- Treat `response.audio.delta` as first-class streaming audio and forward each
  chunk through the 60 ms PCM-to-Opus downlink path as soon as it arrives.
- Treat `response.audio.done` or `response.done` as audio completion.
- Send `session.finish` for cleanup after the response is done, or close on
  abort. Do not send it before starting the read loop.

### Doubao / Volcengine Realtime TTS Shape

- Send `tts_session.update`.
- Send one or more `input_text.append` events.
- Send `input_text.done`.
- Receive `tts_session.updated`, then `response.audio.delta` events until
  `response.audio.done`.
- The mature pattern uses concurrent send and receive, not a blocking
  request/response wait.

## Target A21 State Machine

For a cascade turn:

1. Xiaozhi `listen/start` creates a cancellable turn context.
2. Opus ingress queue decodes and appends frames to streaming ASR while the
   WebSocket control loop remains free for `abort`.
3. ASR partial starts LLM once, if configured.
4. LLM deltas are sentence/phrase segmented.
5. Each segment starts a provider TTS session with read-first lifecycle.
6. First provider audio delta immediately creates `tts.first_audio`, starts
   `sentence_start` if not already sent for that segment, and pushes Opus
   downlink frames.
7. Any `abort`, new wake, touch barge-in, socket close, or provider error
   cancels ASR/LLM/TTS/downlink work and always writes a stable `tts.stop`
   unless the socket is already dead.
8. Trace markers distinguish ASR, LLM, TTS, downlink, provider failure, and
   cancellation stages without leaking transcript, audio, URL, token, or key.

For an end-to-end realtime mode:

- Keep the frontend selector/API surfaces, but do not route physical product
  acceptance through the end-to-end path until a real provider session and
  Xiaozhi device trace are proven.

## Implementation Tasks

### Task 1: Freeze Half-Finished DashScope Patch

Files:

- Modify: `internal/providers/dashscope_realtime.go`
- Test: `internal/providers/dashscope_realtime_test.go`

Steps:

1. Keep the useful redacted finding names from the current dirty diff.
2. Reformat the read loop cleanly.
3. Do not commit this as a standalone fix; it must land together with the
   read-first lifecycle tests below.

Acceptance:

- `git diff --check` passes.
- TTS no-audio/error findings remain redacted and stage-specific.

### Task 2: Add Failing DashScope TTS State-Machine Tests

Files:

- Modify: `internal/providers/dashscope_realtime_test.go`

Tests to add before implementation:

- `TestDashScopeRealtimeTTSWaitsForSessionUpdatedBeforeText`
  - Fake server sends `session.created`, then `session.updated`, then audio
    lifecycle events.
  - Assert client event order is `session.update`,
    `input_text_buffer.append`, `input_text_buffer.commit`,
    `session.finish`.
  - Assert append/commit are not sent before `session.updated`.

- `TestDashScopeRealtimeTTSDoesNotFinishBeforeAudioDone`
  - Fake server sends `session.updated`, `response.created`,
    `response.audio.delta`, `response.audio.done`, `response.done`,
    `session.finished`.
  - Assert first audio chunk is emitted before any client `session.finish`.
  - Assert `session.finish` is sent only after `response.audio.done` or
    `response.done`.

- `TestDashScopeRealtimeTTSAbortClosesWithoutSessionFinish`
  - Cancel the request context after session update.
  - Assert the adapter closes the session and does not send cleanup
    `session.finish` after cancellation.

Expected RED:

- Current code sends `session.finish` before reading provider audio and does
  not wait for `session.updated`.

### Task 3: Rebuild DashScope TTS Adapter

Files:

- Modify: `internal/providers/dashscope_realtime.go`

Implementation:

- Introduce a small TTS event loop state:
  `created -> updating -> ready -> text_committed -> audio_streaming ->
  audio_done -> finishing -> finished`.
- Start reading immediately after the WebSocket connects.
- Write `session.update`.
- Wait for `session.updated` or tolerate `session.created` as informational.
- Write append and commit only after update acknowledgement or a bounded
  timeout if the provider skips acknowledgement.
- Emit audio chunks on every `response.audio.delta`.
- On `response.audio.done` or `response.done`, flush the chunker, then send
  `session.finish` if the context is still active.
- Treat `session.finished` as terminal success if audio has been seen.
- On provider `error`, invalid delta, no audio, or read failure, emit redacted
  `VoiceAudioChunk{Finding, Err}`.
- On context cancellation, close without sending `session.finish`.

Acceptance:

- DashScope TTS tests prove read-first and finish-after-audio lifecycle.
- Existing ASR tests still pass.

### Task 4: Gateway Turn Cleanup Contract

Files:

- Modify if needed: `internal/gateway/server.go`
- Modify if needed: `internal/gateway/server_test.go`

Tests:

- A provider TTS chunk error during answer path records the specific TTS
  failure marker and still writes `tts.stop`.
- Abort during answer TTS cancels provider work and writes `tts.stop` once if
  the socket remains open.
- Fast-ack downlink failure does not poison the answer chain unless the turn is
  actually aborted or the connection write failed.

Acceptance:

- Xiaozhi state always exits Speaking via `tts.stop` or a closed socket.
- No duplicate stop storms.

### Task 5: Verification And Deploy

Commands, serial only:

```bash
go test ./internal/providers -run 'DashScopeRealtimeTTS|DashScopeRealtimeASR' -count=1
go test ./internal/gateway ./internal/app ./internal/providers -count=1
git diff --check
make verify
```

Then deploy the committed build to `47.103.57.217`, run:

```bash
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 \
  go run ./cmd/a21 xiaozhi-voice-bench \
  --gateway-url http://47.103.57.217 \
  --repeat 1 \
  --timeout-ms 20000 \
  --require-product-chain \
  --output-dir reports
```

Expected host/public evidence before physical:

- ASR partial/final observed.
- LLM first content observed.
- TTS first audio observed.
- `audio.downlink.first_frame` observed.
- Binary Opus downlink frames observed.
- Barge-in path writes/cancels cleanly.

Physical evidence still required:

- StackChan on `ws://47.103.57.217/v1/xiaozhi`.
- Real mic/Opus ingress.
- Audible playback.
- Touch/new-wake barge-in.
- Wake from idle with the configured phrase.

## Rollback

- Revert the DashScope adapter commit and restart the public Gateway with the
  previous known binary if the provider session deadlocks.
- Keep Mac local Gateway selectable.
- Keep product firmware lane unchanged:
  `a21-stackchan-official-xiaozhi-compatible.bin`.

## Acceptance Boundary

Passing unit tests and public host bench can unblock the main edge voice chain
candidate. It cannot declare full PRD green until physical StackChan wake,
mic-driven dialogue, audible playback, and barge-in evidence pass.
