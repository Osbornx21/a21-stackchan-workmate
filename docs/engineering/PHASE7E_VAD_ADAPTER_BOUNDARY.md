# A21 Phase 7E VAD Adapter Boundary

## Purpose

Phase 7E separates A21 audio ingress buffering from voice activity detection. This keeps the current deterministic RMS detector useful for tests while preventing it from becoming the long-term production VAD by accident.

## Current Implementation

`internal/audio.Ingress` now owns:

- bounded frame buffering
- per stream speech-active state
- silence hangover
- `vad.speech.start` and `vad.speech.end` event generation

`internal/audio.VADDetector` owns:

- per-frame speech decision
- detector score
- detector identity

The default detector is `a21-rms-vad`, which preserves the existing mock behavior and keeps tests deterministic. Tests can inject a scripted detector to prove Gateway-facing ingress behavior does not depend on RMS internals.

## Boundaries

This is not a production VAD claim. The RMS detector is a development baseline only.

Future mature adapters should plug into `VADDetector` without changing:

- StackChan firmware protocol
- Gateway audio WebSocket contract
- provider realtime session contract
- barge-in state machine
- trace/session/device identity propagation

## Next Candidates

Future work should evaluate mature VAD/AEC options behind this boundary, likely including:

- WebRTC VAD/AEC for classic low-latency speech front-end behavior
- provider-side VAD when the realtime provider has reliable server-side turn detection
- local neural VAD only if latency, deployment size, and maintenance cost are justified

Any chosen adapter must be benchmarked with A21 latency reports before it is treated as the production path.
