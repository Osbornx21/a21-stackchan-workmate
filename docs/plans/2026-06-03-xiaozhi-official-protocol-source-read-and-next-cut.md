# Xiaozhi Official Protocol Source Read And Next Cut

Status: completed host-local cut.
Created: 2026-06-03 09:24 CST.

## Transition

`T-XIAOZHI-OFFICIAL-PROTOCOL-SOURCE-READ-AND-NEXT-CUT-001`

## Current State

A21 has a stock-compatible `/v1/xiaozhi` WebSocket path, binary Opus uplink
parsing/decoding, streaming ASR partial hooks, nonblocking ASR commit, and
host-side realtime profile truth gates. Full Xiaozhi realtime PRD acceptance is
still false because the path does not yet prove end-to-end real streaming
ASR/LLM/TTS/Opus playback on a physical stock device.

## Target State

Advance one protocol-faithful host-side cut toward the user-requested chain:

`mic/i2s -> wake/VAD -> Opus frame queue -> WebSocket or UDP streaming ->
streaming ASR -> streaming LLM -> streaming TTS -> Opus frame playback`

The next cut must align with upstream Xiaozhi behavior rather than widening
A21-only semantics.

## Trigger

The user re-reviewed Xiaozhi and reiterated that A21 must match Xiaozhi's
protocol and audio handling. They asked for multiple subthreads to carefully
read Xiaozhi source and A21 implementation before continuing.

## Main-Thread Action

- Reconfirm current branch, dirty state, latest plan/handoff, and Gateway
  `/v1/xiaozhi` implementation.
- Dispatch read-only explorers for distinct Xiaozhi/A21 source questions.
- While explorers run, inspect the local implementation for a narrow
  test-first transition that does not require provider execution, Gateway
  lifecycle changes, firmware build/flash, hardware, or audio playback.
- Implement only after an intentional red test.
- Update state machine and handoff log before returning.

## Worker Execution Tasks

Explorer A:

- Read upstream `xiaozhi-esp32` WebSocket and MQTT+UDP protocol docs plus
  protocol implementation.
- Return exact stock message/audio/state-machine expectations that A21 must not
  violate.

Explorer B:

- Read upstream `xiaozhi-esp32` audio hot path: local wake/VAD, Opus
  encode/decode, audio queues, listening/speaking transitions, and playback
  interruption.
- Return the most important A21 parity gaps and file references.

Explorer C:

- Read A21 `/v1/xiaozhi`, transport parser/builder, voice pipeline, and
  realtime parity code.
- Return the narrowest next implementation cut that improves stock Xiaozhi
  protocol/audio fidelity without live provider or hardware execution.

## Worker Boundary Conditions

- Read-only source review unless explicitly assigned a later worker patch.
- Do not run providers, V21, Gateway, firmware builds, flashes, NVS/serial,
  hardware, or audio playback.
- Do not introduce X21 runtime identity or A21-only protocol fields into stock
  Xiaozhi behavior.
- Treat upstream Xiaozhi as reference material and the user's A21 direction as
  authority.

## Worker Summary Format

- `answer`: concise finding.
- `stock evidence`: upstream file paths and relevant functions/docs.
- `A21 evidence`: A21 file paths if inspected.
- `gap/risk`: concrete mismatch or no blocking finding.
- `recommended next cut`: one scoped action.
- `commands run`: commands and whether they were read-only.

## Acceptance

- At least two independent source-read summaries are collected or the main
  thread records why they were unavailable.
- The implemented cut has a failing test first and then passes focused tests.
- `git diff --check` passes.
- `make verify` passes if Go code or diff-check-covered docs change.
- Handoff log and state machine record the completed work and remaining PRD
  blockers.

## Completed Work

- Explorer A completed a read-only stock protocol pass. It confirmed Xiaozhi is
  device-driven for `listen`; the server should send `hello`, `stt`, `llm`,
  `tts`, `mcp`, and related system messages, not rely on server-to-device
  `listen` for stock physical devices. The existing stock physical
  `listen`-reply suppression test already covers this invariant.
- Explorer B completed a read-only audio hot-path pass. It identified the
  largest host-side mismatch: upstream Xiaozhi can send wake-word pre-roll Opus
  before/around `listen/detect`, while A21 previously discarded binary Opus
  when `session.listening=false`.
- Explorer C could not be spawned because the thread limit was reached; the
  main thread performed the A21 implementation-gap read locally.
- Added stock `stt` emission before `tts start` when a streaming ASR partial or
  final transcript is available, matching upstream device display semantics
  without storing transcript text in traces.
- Added bounded wake pre-roll buffering for up to five decoded Opus frames in a
  truly idle, no-cooldown Xiaozhi session. On the next stock `listen/start`, the
  pre-roll is attached to the turn, counted in audio ingress evidence, and fed
  to streaming ASR if a streaming ASR session is active.
- Preserved existing no-speech/host-say cooldown behavior: Opus frames during
  cooldown or while a current turn is active remain `ignored_not_listening`
  rather than being mistaken for wake pre-roll.

## Verification

- Red test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketStreamingASRFinalSendsStockSTTBeforeTTS -count=1`
  failed because the first post-ASR message was `tts/start` instead of stock
  `stt`.
- Red test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketWakePrerollOpusFeedsNextTurn -count=1`
  failed because the voice pipeline did not receive the pre-listen Opus frame.
- Focused Gateway tests passed:
  `go test ./internal/gateway -run 'TestXiaozhiWebSocket(WakePrerollOpusFeedsNextTurn|StreamingASRFinalSendsStockSTTBeforeTTS|StreamingASRFinalStartsPipelineWithoutBatchFallback|ListenStopRunsVoicePipelineAndSendsPacedOpus|StreamingASRStartsBeforeListenStop|ASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal)' -count=1`.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Focused app parity/readiness tests passed:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity|TestXiaozhiStreamingProviderReadiness' -count=1`.

## Boundaries Observed

- No provider execution.
- No V21 execution.
- No Gateway start/stop.
- No `/v1/xiaozhi/say`.
- No host-loopback runtime acceptance.
- No firmware build, flash, NVS, serial, hardware action, or audio playback.

## Failure State

- Explorer findings show the selected cut would widen non-stock protocol
  behavior.
- The cut requires provider/V21/hardware execution that is outside this host
  slice.
- Focused tests or `make verify` fail.

## Rollback

Revert only this transition's files after preserving the failed red/green
evidence in the handoff log.

## Next State

If this cut passes, continue with the next strict Xiaozhi parity transition:
real provider streaming TTS execution only when explicitly authorized and fully
configured, or physical stock `/v1/xiaozhi` trace collection in an approved
hardware window.
