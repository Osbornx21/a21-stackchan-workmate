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
