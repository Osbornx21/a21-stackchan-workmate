# Xiaozhi Stale Opus Ingress Suppression

Status: in verification.
Created: 2026-06-03 11:00 CST.

## Transition

`T-XIAOZHI-STALE-OPUS-INGRESS-SUPPRESSION-001`

## Current State

A21 has a stock-compatible `/v1/xiaozhi` WebSocket path with streaming ASR,
stock `stt` ordering, bounded wake pre-roll, nonblocking ASR commit, and a
bounded Opus ingress queue. The queue keeps JSON control responsive while audio
append is slow.

The remaining host-side queue risk is stale frame ownership: an Opus queue item
created under an old turn can reach processing after that turn is cancelled by
`abort`, wake-as-abort, or `listen/start` barge-in. If processing only consults
session-global state, old audio can be decoded into the next listening turn.

## Target State

Cancelled-turn Opus ingress items must be suppressed before decode, audio
ingress, VAD, or streaming ASR append. This keeps the Xiaozhi state machine
clean when Speaking is interrupted and Listening resumes.

## Scope

- Host-local Gateway only.
- Add a failing Gateway test first.
- Preserve stock protocol fields and current barge-in/abort behavior.
- Preserve raw audio and transcript privacy.

## Non-Scope

- No provider execution.
- No V21 execution.
- No Gateway start/stop.
- No `/v1/xiaozhi/say`.
- No firmware build, flash, NVS, serial, hardware action, or audio playback.
- No claim of physical PRD acceptance.

## Action

- Test that `processXiaozhiOpusIngressFrame` with a cancelled item context
  records stale suppression and does not produce `audio.ingress.buffered`.
- Add a guard before decode/ingress processing when the item context is already
  cancelled.

## Acceptance

- Red test fails because the stale frame currently reaches `audio.ingress`.
- Red test turns green after the guard.
- Focused Xiaozhi Gateway tests still pass.
- `go test ./internal/gateway -count=1` passes.
- `git diff --check` passes.
- `make verify` passes.

## Remaining PRD Gap

This is host-local queue correctness only. Full Xiaozhi realtime PRD acceptance
still requires real streaming ASR/LLM/TTS profile execution, physical stock
`/v1/xiaozhi` trace, wake/tap trigger evidence, audible playback, touch/barge-in
proof, playback stop completion, and idle recovery.
