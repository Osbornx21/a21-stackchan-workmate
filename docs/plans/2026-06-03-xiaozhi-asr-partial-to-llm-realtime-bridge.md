# 2026-06-03 - Xiaozhi ASR Partial To LLM Realtime Bridge

Status: planned worker transition.
Owner: scoped worker branch, reviewed by A21 control tower.
Transition: `T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001`.

## Problem

A21 now has stock-shaped `/v1/xiaozhi` WebSocket/Opus transport, decoded PCM
ingress, a streaming ASR session seam, LLM text streaming, streaming TTS seam,
and paced Opus downlink. The chain is still not Xiaozhi realtime because the
answer side begins after ASR final/listen stop. ASR partials are observed as
trace/evidence markers, but they do not yet drive a downstream LLM/TTS turn.

This transition is the narrow host-side bridge that proves the next real link:
an ASR partial from the stock `/v1/xiaozhi` media path can start LLM streaming
before ASR final, and TTS can consume first text/audio before the whole turn is
complete.

## Current State

- Main branch at planning time: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Observed HEAD: `188b341`.
- `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` is a truthful blocker:
  `model_dir_missing`.
- `T-STREAMING-TTS-RUNTIME-PROOF-001` is a truthful blocker:
  `execute_flag_required` unless operator authorizes provider execution.
- `/v1/xiaozhi/say`, host loopback, mock ASR/TTS, static readiness, and WAV/file
  paths remain excluded from acceptance.

## Target State

`S-XIAOZHI-CONTINUOUS-TURN-BRIDGE-CANDIDATE`

- A stock `/v1/xiaozhi` turn can record ordered trace markers proving an ASR
  partial is available before listen stop or ASR final.
- LLM first content can be emitted from a partial-driven turn before ASR final.
- TTS first audio can be emitted before full answer completion.
- The existing batch/final transcript path remains available as rollback and
  fallback.
- Evidence gates reject traces that only prove static seams, `/say`, fake
  providers, host loopback, or full-response/WAV boundaries.

## Scope

Likely files:

- `internal/gateway/server.go`
- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/gateway/server_test.go`
- `internal/providers/voice_pipeline_test.go`
- `internal/app/xiaozhi_realtime_parity.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Allowed:

- Add additive interfaces/events for partial transcript delivery.
- Add a fake/test streaming ASR session that emits partial/final ordering.
- Add trace markers and tests for ordering.
- Preserve the current final-transcript path.

Forbidden:

- No firmware build, flash, NVS, serial, hardware, or audio playback.
- No Gateway service start/stop in the worker.
- No provider or V21 execution unless the operator explicitly reopens scope.
- No `/v1/xiaozhi/say` as acceptance evidence.
- No bare `xiaozhi.bin`.
- No raw transcripts, raw/base64 audio, credentials, full URLs, proxy values, or
  absolute local paths in reports.

## Acceptance

- A focused unit or Gateway test proves `asr.first_partial` occurs before
  `vad.speech.end` or `listen.stop` in the stock `/v1/xiaozhi` path.
- A provider/pipeline test proves LLM first content is emitted before ASR final
  for the partial-driven path.
- A streaming TTS test proves first 60 ms downlink-ready PCM chunk before
  pipeline completion.
- `xiaozhi-realtime-parity` accepts only ordered stock traces and rejects
  `/say`, mock, fast-companion, host-loopback, and turn-buffered paths.
- Existing batch ASR/TTS fallback tests continue to pass.
- Verification: focused Go tests, `git diff --check`, and `make verify`.

## Rollback

Revert the bridge plan, Gateway lifecycle changes, provider/pipeline changes,
parity-gate changes, and tests. Existing streaming ASR seam, Sherpa helper
smoke, Doubao realtime TTS adapter, paced Opus downlink, and batch fallback
paths remain intact.

## Completion Boundary

This transition may produce a host-side continuous-turn bridge candidate. It
does not by itself complete Xiaozhi realtime PRD acceptance. Full acceptance
still needs real ASR model/runtime, real streaming TTS provider execution if
selected, product firmware wake or labeled tap trigger, physical stock
`/v1/xiaozhi` trace, device playback, touch/barge-in, and idle recovery.
