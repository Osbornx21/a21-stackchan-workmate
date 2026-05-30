# A21 Phase 7B Local TTS Boundary

## Purpose

Phase 7B adds the first local TTS adapter boundary for the fast companion hybrid lane. It is intentionally small: use mature local OS speech tooling to prove that A21 can synthesize speech without provider keys, proxy state, firmware changes, or external network calls.

This is not the final A21 voice or voice-clone solution. It is the replaceable local TTS seam that later Piper, sherpa-onnx, CosyVoice, Doubao TTS, or another accepted voice engine can implement behind the same reporting and redaction discipline.

## Current Implementation

`internal/audio.SynthesizeMacOSSay` uses:

- macOS `say`
- macOS `afconvert`
- default voice `Tingting`
- output format `wav_pcm_s16le_16000_mono`

The CLI command is:

```bash
go run ./cmd/a21 local-tts-smoke --output-dir reports
make local-tts-smoke
go run ./cmd/a21 local-voice-loopback --output-dir reports
make local-voice-loopback
```

Optional:

```bash
go run ./cmd/a21 local-tts-smoke --voice Tingting --text "A21 本地语音链路测试。" --output-dir reports
```

## Report Contract

The command writes:

```text
reports/a21-local-tts-YYYYMMDD-HHMMSS.wav
reports/a21-local-tts-smoke-YYYYMMDD-HHMMSS.json
```

The report includes provider, voice name, output format, output path, output bytes, text byte count, total duration, and `tts_first_audio_ms`.

The report does not store input text, provider credentials, proxy values, auth headers, generated transcript text, or model values.

## Local Voice Loopback

`local-voice-loopback` stitches the current local pieces into one host-side evidence report:

```text
mock VAD fixture
  -> mock ASR boundary
  -> mock OpenAI-compatible text_stream parser
  -> macOS local TTS
  -> existing Gateway mock barge-in latency bench
```

This command is not a fake product demo. It is a deterministic host loopback receipt that proves A21's current software seams can carry redacted timing evidence across VAD, ASR placeholder, text-stream parsing, local TTS, and barge-in stop measurement before physical StackChan and real provider execution are allowed into the lane.

The command writes:

```text
reports/a21-local-voice-loopback-YYYYMMDD-HHMMSS.json
```

The report records byte counts and timings only; it does not record the input utterance, mock ASR transcript, mock provider output text, or reasoning text.

## Boundary

This phase proves only local synthesis and WAV conversion. It does not prove:

- A21 final character voice quality
- streaming TTS first chunk latency
- voice cloning
- physical StackChan audibility
- speaker volume or mouth sync
- full-duplex AEC

Physical speaker acceptance remains under the guarded StackChan hardware path.
