# 2026-06-03 - Xiaozhi Nonblocking ASR Commit

Status: completed host-local transition.
Owner: A21 control tower.
Transition: `T-XIAOZHI-NONBLOCKING-ASR-COMMIT-001`.

## Problem

The stock `/v1/xiaozhi` path now starts streaming ASR from incoming Opus frames
and can use ASR partials to begin the answer before `listen.stop`. The final
ASR commit path is still too handler-local: `listen.stop` and VAD auto-stop call
`commitXiaozhiStreamingASR` synchronously, and that function can spend up to
the commit timeout plus final polling in the WebSocket read loop.

That shape is not close enough to Xiaozhi's realtime state machine. A device
may send abort/barge-in or other control messages while final ASR is settling;
the server must keep reading the socket instead of making the WebSocket loop
wait for ASR finalization.

## Target State

- `listen.stop` and VAD auto-stop transition the turn into an async
  `asr.stream.commit` state without blocking the WebSocket read loop.
- Abort/barge-in can be read and processed while ASR commit is still pending.
- If a streaming ASR final arrives and no partial-driven answer already started,
  the normal voice-pipeline task starts from the streaming final transcript.
- If streaming ASR fails or times out, the trace records a truthful blocker
  rather than silently treating batch/WAV fallback as realtime parity.

## Scope

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

## Non-Goals

- No provider execution.
- No V21 execution.
- No Gateway start/stop.
- No `/v1/xiaozhi/say`, host-loopback runtime, firmware build/flash, NVS,
  serial, hardware action, or audio playback.
- No MQTT+UDP implementation in this transition.

## Acceptance

- Red test first proves current `listen.stop` blocks abort processing while
  streaming ASR commit is pending.
- Gateway test proves abort is recorded while commit remains blocked.
- Gateway test proves a streaming ASR commit/final path can start the voice
  pipeline from streaming final without calling batch `Transcribe`.
- Existing Xiaozhi streaming ASR, partial bridge, voice-pipeline, and parity
  tests continue to pass.
- `git diff --check`, focused tests, and `make verify` pass.

## Failure States

- `F-XIAOZHI-ASR-COMMIT-BLOCKS-WS` if the WebSocket read loop cannot process
  abort/barge-in during ASR commit.
- `F-XIAOZHI-ASR-COMMIT-BATCH-REGRESSION` if a streaming ASR session falls back
  to batch WAV transcription for realtime evidence.
- `F-XIAOZHI-ASR-COMMIT-DOUBLE-TURN` if partial-driven and final-driven paths
  start two voice-pipeline tasks.

## Rollback

Revert the async commit transition, tests, and state/log entries. Existing
stock `/v1/xiaozhi` transport, ASR partial bridge, real-profile parity gate,
Sherpa smoke, and TTS adapter seam remain intact.
