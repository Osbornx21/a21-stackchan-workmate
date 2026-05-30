# A21 Phase 7G Audio Front-End Evaluation

## Purpose

Phase 7G turns "use mature wheels" into a concrete A21 gate for VAD, AEC, noise suppression, and turn detection. A21 should not grow a custom production speech front-end unless a benchmark proves mature options fail the Shanghai office and StackChan constraints.

## Command

```bash
go run ./cmd/a21 audio-front-end-plan
```

The command is plan-only. It performs no network calls, loads no native audio libraries, and changes no runtime behavior. It emits the current A21-owned candidate list, guardrails, and evidence required before any candidate can be promoted.

## Current Candidates

- `webrtc_apm`: first evaluation target for Gateway/desktop-side AEC, noise suppression, AGC, and classic speech front-end behavior.
- `provider_side_vad`: evaluated per provider behind the provider adapter only; useful when realtime providers expose reliable turn detection but too opaque to own the whole A21 speech boundary.
- `silero_vad`: neural server-side VAD candidate; useful for speech/non-speech quality, but it does not solve acoustic echo cancellation.
- `a21_rms_vad`: deterministic development baseline only; kept for tests and trace shape, not production.

## Promotion Evidence

Before any candidate becomes default, A21 needs:

- mock benchmark preservation
- recorded Shanghai-office noise benchmark
- physical StackChan speaker-to-mic echo report
- barge-in stop timing
- first-audio waterfall impact
- CPU and memory profile on the target Gateway machines
- `a21_vad_detector_decisions_total{detector,result}` continuity

## Source Baseline

- WebRTC Audio Processing Module documents AEC, noise suppression, AGC, standalone use, and native pipeline use: https://webrtc.googlesource.com/src/+/refs/heads/main/modules/audio_processing/g3doc/audio_processing_module.md
- WebRTC keeps VAD implementation sources under its audio processing tree: https://webrtc.googlesource.com/src/+/main/modules/audio_processing/vad/
- Silero VAD is an open-source neural VAD candidate: https://github.com/snakers4/silero-vad

## Non-Goals

This phase does not add a production VAD, AEC, or neural model runtime. It only prevents accidental promotion of the RMS detector and creates a repeatable decision surface for the next adapter spike.
