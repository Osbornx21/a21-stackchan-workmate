# A21 Phase 7D Realtime First-Audio Waterfall

## Purpose

Phase 7D measures the first provider audio event after A21 commits a realtime input turn. This gives A21 a hard signal for the low-latency fast path before any real-provider or physical-device claims are made.

## Current Implementation

When Gateway `/ws/audio` detects `vad.speech.end`, it calls `CommitAndCreateResponse` on the realtime provider session and records the commit time for that trace/session/device key.

When the provider downlink pump sees the first `VoiceEvent` that carries audio, Gateway:

- records `provider.audio.first_downlink`
- observes `a21_realtime_first_audio_ms`
- keeps emitting normal A21 `control.event` and `audio.playback.chunk` envelopes

The first-audio state is cleared when the realtime audio session closes, so later sessions cannot inherit stale timing state.

## Observability

Trace markers:

- `provider.audio.commit`
- `provider.audio.downlink`
- `provider.audio.first_downlink`
- `audio.playback.chunk.sent`

`GET /v1/traces?trace_id=<trace_id>` also exposes `provider_commit_to_first_audio_ms` in its summary when both commit and first-downlink markers are present.

Metric:

- `a21_realtime_first_audio_ms`

## Boundaries

This measures Gateway/provider-adapter timing only. It does not include device microphone capture, LAN jitter, StackChan speaker buffer latency, acoustic echo cancellation, or physical playback start. Those must be added as separate spans before A21 claims end-to-end first-audio latency.
