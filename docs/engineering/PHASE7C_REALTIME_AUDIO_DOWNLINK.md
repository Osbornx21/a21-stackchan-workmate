# A21 Phase 7C Realtime Audio Downlink Boundary

## Purpose

Phase 7C connects provider realtime output events back to the A21 audio WebSocket without exposing StackChan firmware to provider-specific event formats.

## Current Implementation

The provider realtime session interface now includes:

- `SendAudio`
- `CommitAndCreateResponse`
- `Cancel`
- `Events`
- `Close`

Gateway starts a downlink pump when it creates a realtime provider session for `/ws/audio`. The pump consumes provider-neutral `VoiceEvent` values from `Events()` and emits ordinary A21 envelopes:

- `control.event` for semantic state such as `speaking`
- `audio.playback.chunk` when the event carries a `VoiceAudioChunk`

The pump preserves `trace_id`, `session_id`, and `device_id`, records the active playback `stream_id`, and serializes WebSocket writes so the audio ingress loop and provider downlink loop cannot write concurrently to the same socket.

OpenAI realtime sessions now read provider WebSocket messages, map `response.output_audio.delta` into A21 `VoiceEventSpeaking` events, and leave all provider-specific parsing inside `internal/providers`.

## Observability

Trace markers:

- `provider.audio.downlink`
- `audio.playback.chunk.sent`
- `control.speaking.sent`

Metrics:

- `a21_realtime_audio_downlink_events_total`
- `a21_audio_playback_chunk_total`

## Boundaries

This phase proves the provider-output plumbing with fake WebSocket connections and Gateway WebSocket integration tests. It does not claim real provider first-audio latency, real StackChan speaker playback, microphone capture, AEC, or physical full-duplex.

No firmware code receives OpenAI, Doubao, or any provider-native event shape. Firmware continues to receive only A21 `control.event` and `audio.playback.chunk`.
