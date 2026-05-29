# A21 Phase 7A Audio Ingress Boundary

## Purpose

Phase 7A starts the low-latency voice-link foundation at the Gateway audio ingress point. It does not connect a real provider and does not claim production VAD. It adds the first testable place where StackChan/simulator audio frames become observable buffered media instead of disappearing into a single-frame mock response.

## Current Implementation

`internal/audio` owns:

- bounded frame buffering per session/trace/device key
- dropped-frame accounting when the buffer reaches its cap
- PCM16 little-endian RMS calculation from the `audio.frame` payload
- RMS-based mock VAD transitions:
  - `vad.speech.start`
  - `vad.speech.end`
- a short silence hangover before speech end

Gateway `/ws/audio` now records:

- `audio.frame.received`
- `audio.ingress.buffered`
- `audio.ingress.invalid`
- `vad.speech.start`
- `vad.speech.end`

Gateway metrics now include:

- `a21_audio_ingress_frames_total`
- `a21_audio_ingress_dropped_frames_total`
- `a21_audio_ingress_buffer_depth`
- `a21_vad_speech_start_total`
- `a21_vad_speech_end_total`

## Boundaries

The current VAD is deliberately simple RMS thresholding. It is useful for trace, buffer, and state-machine integration tests only. It is not a replacement for a mature VAD/AEC stack.

Future production work must keep this boundary but can replace the internals with a mature component or provider-side turn detection adapter. The replacement must preserve A21 trace names, metrics, and interruption semantics so simulator, firmware, and provider adapters do not become tightly coupled to a specific VAD library.

This phase still does not prove:

- real StackChan microphone capture
- acoustic echo cancellation
- full-duplex capture/playback
- production VAD quality
- LAN jitter performance
- real provider first-audio latency
