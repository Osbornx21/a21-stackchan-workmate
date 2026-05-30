# A21 Phase 7G Audio Front-End Evaluation

## Purpose

Phase 7G turns "use mature wheels" into a concrete A21 gate for VAD, AEC, noise suppression, and turn detection. A21 should not grow a custom production speech front-end unless a benchmark proves mature options fail the Shanghai office and StackChan constraints.

## Command

```bash
go run ./cmd/a21 audio-front-end-plan
go run ./cmd/a21 audio-front-end-eval --mock
go run ./cmd/a21 audio-front-end-eval --fixture reports/a21-audio-fixture.json
```

`audio-front-end-plan` is plan-only. It performs no network calls, loads no native audio libraries, and changes no runtime behavior. It emits the current A21-owned candidate list, guardrails, and evidence required before any candidate can be promoted.

`audio-front-end-eval --mock` runs the deterministic A21 RMS baseline over `a21_mock_vad_fixture_v1`. It reports frame counts, true/false positives, true/false negatives, precision, recall, start/end events, required future metrics, and `promotion_gate: not_production`.

`audio-front-end-eval --fixture <path>` runs the same report shape over an A21 labelled PCM fixture. `--mock` and `--fixture` are mutually exclusive. There is no implicit real evaluation mode, so nobody can mistake a synthetic or labelled-frame report for provider, AEC, full-duplex, or physical StackChan acceptance.

## Fixture Contract

Fixture files use JSON:

```json
{
  "schema_version": "a21.audio.frontend_fixture.v1",
  "dataset": "a21_shanghai_office_noise_sample_001",
  "detector": "a21-rms-vad",
  "sample_rate_hz": 16000,
  "channels": 1,
  "duration_ms": 20,
  "frames": [
    {
      "seq": 1,
      "expected_speech": false,
      "pcm_s16le_base64": "..."
    }
  ]
}
```

Current constraints:

- `detector` must be `a21-rms-vad` until a mature adapter is actually implemented.
- `sample_rate_hz` must be `16000`.
- `channels` must be `1`.
- `duration_ms` must be `20`.
- Each `pcm_s16le_base64` frame must decode to 640 bytes.
- `dataset` and `detector` must not contain X21/V21 legacy identity.

This fixture is an evaluation intermediate, not the preferred long-term audio asset format. Future recorders/converters can use mature WAV/Opus tooling and emit this labelled frame format for A21 evaluation.

## Current Candidates

- `webrtc_apm`: first evaluation target for Gateway/desktop-side AEC, noise suppression, AGC, and classic speech front-end behavior.
- `provider_side_vad`: evaluated per provider behind the provider adapter only; useful when realtime providers expose reliable turn detection but too opaque to own the whole A21 speech boundary.
- `silero_vad`: neural server-side VAD candidate; useful for speech/non-speech quality, but it does not solve acoustic echo cancellation.
- `a21_rms_vad`: deterministic development baseline only; kept for tests and trace shape, not production.

## Promotion Evidence

Before any candidate becomes default, A21 needs:

- mock benchmark preservation
- `audio-front-end-eval --mock` report preservation
- labelled fixture preservation through `audio-front-end-eval --fixture`
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
