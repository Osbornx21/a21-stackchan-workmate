# A21 Phase 7F VAD Detector Observability

## Purpose

Phase 7F makes VAD decisions visible at the detector boundary. A21 must be able to compare RMS, WebRTC VAD/AEC, provider-side turn detection, or a future neural VAD without changing the firmware protocol, audio WebSocket, or barge-in state machine.

## Current Signal

Gateway increments:

- `a21_vad_detector_decisions_total{detector="a21-rms-vad",result="speech"}`
- `a21_vad_detector_decisions_total{detector="a21-rms-vad",result="silence"}`

The label set is intentionally small:

- `detector`: internal detector adapter identity
- `result`: `speech` or `silence`

`a21_vad_speech_start_total` and `a21_vad_speech_end_total` still track state transitions. The detector decision metric tracks per-frame input decisions, so it can be used to compare sensitivity, false starts, and silence handling before a detector is promoted.

## Guardrails

This metric does not make the RMS detector production-ready. It only makes the current development detector observable.

Future adapters must keep detector names stable, low-cardinality, and A21-owned. Do not put device IDs, trace IDs, provider session IDs, raw file names, or user text into metric labels.

`go run ./cmd/a21 audio-front-end-plan` lists the current candidate set and the metrics each candidate must preserve before it can replace the deterministic RMS baseline.
