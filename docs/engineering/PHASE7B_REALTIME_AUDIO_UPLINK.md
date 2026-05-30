# A21 Phase 7B Realtime Audio Uplink Boundary

## Purpose

Phase 7B connects the existing Gateway `/ws/audio` ingress boundary to provider realtime sessions without changing the StackChan protocol and without giving firmware any provider-specific responsibility.

## Current Implementation

Gateway now detects whether the selected voice provider implements the A21 realtime session interface:

- `StartRealtimeSession`
- `SendAudio`
- `CommitAndCreateResponse`
- `Cancel`
- `Close`
- `Events` is consumed by the follow-on downlink slice

When that interface exists, Gateway `/ws/audio`:

- starts or reuses a provider realtime session on `vad.speech.start`
- forwards detected speech frames as A21 `protocol.AudioChunk`
- skips pure silence hangover frames
- commits and asks the provider to create a response on `vad.speech.end`
- returns semantic A21 control events (`listening`, then `thinking`)
- closes any provider realtime session owned by the audio WebSocket when that socket disconnects

The default mock provider path is unchanged: non-realtime providers still receive the deterministic mock playback path used by simulator and firmware buffer tests.

## Observability

Trace markers:

- `provider.realtime_session.start`
- `provider.realtime_session.error`
- `provider.audio.append`
- `provider.audio.append.error`
- `provider.audio.commit`
- `provider.audio.commit.error`

Metrics:

- `a21_realtime_audio_uplink_frames_total`
- `a21_realtime_audio_commit_total`

## Boundaries

Phase 7C now reads provider output events through the same A21 realtime session interface. This Phase 7B document remains the uplink contract: it establishes provider-neutral audio append and commit. It still does not claim real first-audio latency by itself.

This phase does not prove real StackChan microphone capture, acoustic echo cancellation, speaker playback, or hardware full-duplex. It only establishes the provider-neutral uplink control point that those future slices must use.
