# A21 Latency Budget

Status: PRD v0.4 acceptance companion.
Date: 2026-06-01.

This file records the machine-readable latency targets used by the server
mainline. It does not promote host fixtures or candidate reports to physical
acceptance.

## Product Gates

| Gate | Target | Required markers |
| --- | --- | --- |
| First audible response | P50 < 900 ms, P95 < 1500 ms | ASR first partial/final, provider first byte/content, TTS first audio, downlink first frame, device playback start |
| Barge-in | stop/cancel P95 < 300 ms | barge-in detected, provider cancel, playback stop, speaking state stop |
| Professional checking feedback | <= 1200 ms | professional trigger, checking feedback emitted, V21 query start |
| Professional answer | no fixed tail target yet | conclusion, evidence, confidence, follow-up, V21 first result/error/timeout |
| Provider hot plug | no Gateway business edit | provider profile/env/smoke timing and redacted report |

## Promotion Rules

- P50/P95 values must come from an A21-owned report with `trace_id`,
  `session_id`, and `device_id`.
- Physical StackChan claims require device-side evidence for playback start,
  stop, and visible/speaking state. Host fixtures stay candidate evidence.
- Provider reports must use the shared contract in
  `docs/engineering/A21_PROVIDER_BENCHMARKS.md`.
- Reports must not store prompt text, transcripts, provider output, reasoning,
  raw audio, encoded audio payloads, credentials, full URLs, proxy URLs, or
  full local paths.

## Current Status

No PRD latency gate is accepted on `codex/a21-server-mainline` yet. The next
valid step is WS-1 protocol transport evidence, followed by streaming pipeline
and physical-device evidence.

`go run ./cmd/a21 xiaozhi-voice-bench` is the current host-loopback Xiaozhi
candidate gate. It connects to an already-running Gateway `/v1/xiaozhi`, sends
synthetic Opus uplink or a 16 kHz mono speech WAV supplied through `--input-wav`
and encoded into 60 ms Opus frames through the stock profile, measures answer
first-audio and abort-to-stop timing across repeated turns, and writes a
redacted `a21.xiaozhi_voice_bench.v1` report. A passing run uses
`acceptance_status=candidate_host_only` and always keeps `prd_accepted=false`
because it does not prove physical StackChan microphone capture, device
playback start, visible/speaking state, multi-frame TTS cancellation, or
real-device barge-in stop.

`go run ./cmd/a21 xiaozhi-professional-bench` is the host-only Xiaozhi
professional-mode runtime report. Its default mode starts an in-process Gateway
with mock ASR, mock TTS, and a fake V21 client, drives `/v1/xiaozhi` with
`listen/start mode=professional`, Opus uplink, and `listen/stop`, then records
only redacted timing/status/count fields. With `--gateway-url`, the same bench
drives an already-running A21 Gateway and can promote the professional report
to `acceptance_status=external_gateway_ready` when Gateway traces prove
`v21.query.start`, `v21.query.first_result`, checking feedback within 1200 ms,
evidence/cards/follow-ups, abort stop, stale result suppression, and redaction.
Both modes always keep `prd_accepted=false` because they still do not prove
physical StackChan microphone capture, audible playback start, visible state, or
real-device barge-in stop.
