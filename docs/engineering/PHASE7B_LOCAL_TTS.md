# A21 Phase 7B Local TTS Boundary

## Purpose

Phase 7B adds the first local TTS adapter boundary for the fast companion hybrid lane. It is intentionally small: use mature local tooling to prove that A21 can synthesize speech without provider keys, proxy state, firmware changes, or external network calls.

This is not the final A21 voice or voice-clone solution. It is the replaceable local TTS seam that later Piper, sherpa-onnx, CosyVoice, Doubao TTS, or another accepted voice engine can implement behind the same reporting and redaction discipline.

The current selected real local TTS candidate for M2 is `sherpa-onnx`. macOS `say` is only a diagnostic fallback. It must not be counted as the PRD-grade local TTS lane.

## Current Implementation

`internal/audio.SynthesizeSherpaONNX` is the selected M2 local TTS lane. It uses:

- isolated Python under `.a21-tools/sherpa-onnx-venv`
- official `sherpa-onnx` VITS runtime
- model directory `.a21-tools/sherpa-onnx-models/vits-icefall-zh-aishell3`
- repository script `scripts/a21_sherpa_onnx_tts.py`
- macOS `afconvert` only to normalize output to 16 kHz mono PCM WAV

`internal/audio.SynthesizeMacOSSay` remains as a diagnostic fallback. It uses:

- macOS `say`
- macOS `afconvert`
- default voice `Tingting`
- output format `wav_pcm_s16le_16000_mono`

The CLI command is:

```bash
go run ./cmd/a21 local-tts-smoke --output-dir reports
make local-tts-smoke
go run ./cmd/a21 local-voice-loopback --repeat 3 --output-dir reports
make local-voice-loopback
go run ./cmd/a21 stackchan-local-tts-playback --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --output-dir reports
make stackchan-local-tts-playback
```

Optional:

```bash
go run ./cmd/a21 local-tts-smoke --engine sherpa_onnx --speaker-id 21 --text "A21 本地语音链路测试。" --output-dir reports
go run ./cmd/a21 local-tts-smoke --engine macos_say --voice Tingting --text "A21 本地语音链路测试。" --output-dir reports
```

Environment variables:

- `A21_LOCAL_TTS_ENGINE`: `sherpa_onnx` by default; `macos_say` for diagnostic fallback only.
- `A21_LOCAL_TTS_VOICE`: macOS fallback voice name.
- `A21_SHERPA_ONNX_MODEL_DIR`: optional override for the local sherpa model directory.
- `A21_SHERPA_ONNX_SPEAKER_ID`: optional VITS speaker id, default `21`.
- `A21_LOCAL_VOICE_LOOPBACK_REPEAT`: optional repeat count for loopback P50/P95 timing evidence.

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
  -> selected local TTS
  -> existing Gateway mock barge-in latency bench
```

This command is not a fake product demo. It is a deterministic host loopback receipt that proves A21's current software seams can carry redacted timing evidence across VAD, ASR placeholder, text-stream parsing, local TTS, and barge-in stop measurement before physical StackChan and real provider execution are allowed into the lane.

The command writes:

```text
reports/a21-local-voice-loopback-YYYYMMDD-HHMMSS.json
```

The report records byte counts and timings only; it does not record the input utterance, mock ASR transcript, mock provider output text, reasoning text, provider credentials, or full model path. With `--repeat N`, it records `tts_first_audio_p50_ms`, `tts_first_audio_p95_ms`, `first_audio_total_p50_ms`, and `first_audio_total_p95_ms`.

## StackChan Local TTS Playback

`stackchan-local-tts-playback` is the first real local-TTS-to-device downlink bridge. It synthesizes a local TTS WAV, parses only 16 kHz / 16-bit / mono PCM, splits it into 20 ms chunks, and sends those chunks through Gateway `/v1/devices/control` as `audio.playback.chunk` envelopes. Reports include byte counts, TTS timing, playback chunk count, trace/session/stream ids, and the generated WAV path. They do not include input text or full model paths.

This command proves that selected local TTS audio can enter the same StackChan playback path as Gateway/provider audio. It still does not claim human-audible quality unless an operator separately records physical sound observation.

## Boundary

This phase proves only local synthesis and WAV conversion. It does not prove:

- A21 final character voice quality
- streaming TTS first chunk latency
- voice cloning
- physical StackChan audibility
- speaker volume or mouth sync
- full-duplex AEC

Physical speaker acceptance remains under the guarded StackChan hardware path.

## Current Status

`sherpa-onnx` is installed in the isolated `.a21-tools/sherpa-onnx-venv` toolchain. The selected Chinese VITS model download/extraction must complete before M2 can be promoted from software-ready to evidence-passed. The macOS fallback loopback may continue to validate host-side timing and report redaction, but it is not promoted to the selected local TTS implementation.
