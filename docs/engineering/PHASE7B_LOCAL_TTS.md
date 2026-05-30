# A21 Phase 7B Local And API ASR/TTS Boundary

## Purpose

Phase 7B adds the first local TTS adapter boundary for the fast companion hybrid lane and establishes the comparison discipline for local/API ASR and TTS. It is intentionally small: use mature local tooling to prove that A21 can synthesize speech without provider keys, proxy state, firmware changes, or external network calls, then compare API ASR/TTS candidates through Provider Spine or isolated mainland-lab probes before promotion.

This is not the final A21 voice or voice-clone solution. It is the replaceable ASR/TTS seam that later Piper, sherpa-onnx, CosyVoice, Doubao TTS, DashScope/Bailian, or another accepted local/API engine can implement behind the same reporting and redaction discipline.

The current selected real local TTS candidate for M2 is `sherpa-onnx`. macOS `say` is only a diagnostic fallback. It must not be counted as the PRD-grade local TTS lane.

Local ASR/TTS is a controlled baseline, not a permanent product rule. A21 may promote API ASR, API TTS, realtime TTS, or hybrid lanes when mainland-network evidence beats the local baseline on latency, long-tail stability, voice quality, barge-in behavior, privacy, failure handling, and cost. API bakeoffs must stay behind Provider Spine or an explicit isolated lab probe; provider keys must never enter Git, reports, logs, screenshots, firmware, or device messages.

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
go run ./cmd/a21 local-asr-smoke --output-dir reports
make local-asr-smoke
go run ./cmd/a21 local-voice-loopback --repeat 3 --output-dir reports
go run ./cmd/a21 local-voice-loopback --asr-provider sherpa_onnx --engine sherpa_onnx --repeat 3 --output-dir reports
go run ./cmd/a21 local-voice-loopback --text-provider deepseek --execute-text-provider --repeat 3 --output-dir reports
make local-voice-loopback
go run ./cmd/a21 stackchan-local-tts-playback --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --output-dir reports
make stackchan-local-tts-playback
go run ./cmd/a21 stackchan-fast-companion-turn --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --repeat 3 --output-dir reports
make stackchan-fast-companion-turn
```

Optional:

```bash
go run ./cmd/a21 local-tts-smoke --engine sherpa_onnx --speaker-id 21 --text "A21 本地语音链路测试。" --output-dir reports
go run ./cmd/a21 local-tts-smoke --engine macos_say --voice Tingting --text "A21 本地语音链路测试。" --output-dir reports
go run ./cmd/a21 local-asr-smoke --family paraformer --model-dir .a21-tools/sherpa-onnx-asr-models/sherpa-onnx-paraformer-zh-small-2024-03-09 --wav .a21-tools/sherpa-onnx-asr-models/sherpa-onnx-paraformer-zh-small-2024-03-09/test_wavs/0.wav --output-dir reports
```

Environment variables:

- `A21_LOCAL_TTS_ENGINE`: `sherpa_onnx` by default; `macos_say` for diagnostic fallback only.
- `A21_LOCAL_TTS_VOICE`: macOS fallback voice name.
- `A21_SHERPA_ONNX_MODEL_DIR`: optional override for the local sherpa model directory.
- `A21_SHERPA_ONNX_SPEAKER_ID`: optional VITS speaker id, default `21`.
- `A21_LOCAL_ASR_ENGINE`: `sherpa_onnx` by default for `local-asr-smoke`.
- `A21_LOCAL_ASR_PROVIDER`: `mock_asr` by default for `local-voice-loopback`; set `sherpa_onnx` for real local ASR.
- `A21_SHERPA_ONNX_ASR_MODEL_DIR`: optional override for the local sherpa ASR model directory.
- `A21_SHERPA_ONNX_ASR_FAMILY`: optional ASR family, one of `paraformer`, `sense_voice`, or `streaming_zipformer`.
- `A21_SHERPA_ONNX_ASR_WAV`: optional local ASR WAV fixture.
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
  -> mock ASR boundary or explicit sherpa-onnx ASR fixture
  -> mock OpenAI-compatible text_stream parser or explicit DeepSeek text_stream execution
  -> local acknowledgement TTS receipt
  -> selected local TTS
  -> existing Gateway mock barge-in latency bench
```

This command is not a fake product demo. It is a deterministic host loopback receipt that proves A21's current software seams can carry redacted timing evidence across VAD, ASR, text-stream parsing, local acknowledgement TTS, local answer TTS, and barge-in stop measurement before physical StackChan and real provider execution are allowed into the lane. The default stays `mock_asr` for stable local development; `--asr-provider sherpa_onnx` runs the tracked local ASR runner and passes the transcript internally to the text-stream step without writing it to stdout or reports.

By default it uses the deterministic mock text stream. `--text-provider deepseek --execute-text-provider` is the explicit paid-network path. It requires `A21_LAB_DEEPSEEK_API_KEY`, uses the P0 DeepSeek profile, sends only the current ASR transcript to the provider, uses the streamed provider content as the TTS input, and still redacts user input, ASR transcript, provider content, reasoning content, API key, model value, proxy value, and full URL from stdout and the saved report.

The command writes:

```text
reports/a21-local-voice-loopback-YYYYMMDD-HHMMSS.json
```

The report records byte counts, ASR provider/model basename/WAV basename, ASR decode timing, ASR RTF, provider family, execution flag, endpoint host, delta counts, and timings only; it does not record the input utterance, ASR transcript, provider output text, local acknowledgement text, reasoning text, provider credentials, model value, proxy value, full URL, full model path, or full WAV path. It separately records `local_ack_*` timing for perceived acknowledgement and `answer_first_audio_total_*` timing for the provider-backed answer. With `--repeat N`, it records `tts_first_audio_p50_ms`, `tts_first_audio_p95_ms`, `first_audio_total_p50_ms`, and `first_audio_total_p95_ms`; the legacy `first_audio_total_*` fields remain answer-total aliases until downstream report consumers migrate.

## Local ASR Smoke

`local-asr-smoke` is the tracked A21 entry point for local sherpa-onnx ASR evaluation. It uses the isolated `.a21-tools/sherpa-onnx-venv` runtime and the repository script `scripts/a21_sherpa_onnx_asr_smoke.py`, not the sherpa CLI. This avoids coupling A21 to optional CLI packaging while still using the mature sherpa-onnx recognizer APIs.

The command writes:

```text
reports/a21-local-asr-smoke-YYYYMMDD-HHMMSS-NANO.json
```

The report records decode time, input duration, real-time factor, transcript character count, model directory basename, and WAV basename. It does not record transcript text, full model path, full WAV path, provider credentials, proxy values, auth headers, or raw audio.

## StackChan Local TTS Playback

`stackchan-local-tts-playback` is the first real local-TTS-to-device downlink bridge. It synthesizes a local TTS WAV, parses only 16 kHz / 16-bit / mono PCM, splits it into 20 ms chunks, and sends those chunks through Gateway `/v1/devices/control` as `audio.playback.chunk` envelopes. Reports include byte counts, TTS timing, playback chunk count, trace/session/stream ids, and the generated WAV path. They do not include input text or full model paths.

This command proves that selected local TTS audio can enter the same StackChan playback path as Gateway/provider audio. It still does not claim human-audible quality unless an operator separately records physical sound observation.

## StackChan Fast Companion Turn

`stackchan-fast-companion-turn` is the M3-prep vertical receipt. It composes the current host-side `local-voice-loopback` evidence with real StackChan downlink delivery:

```text
host fixture or future StackChan mic evidence
  -> selected ASR boundary
  -> mock or explicit DeepSeek text stream
  -> local acknowledgement TTS
  -> selected answer TTS
  -> Gateway /v1/devices/control
  -> StackChan audio.playback.chunk downlink
```

The default `listen_source` is `host_fixture`, so the report is not an M3 pass even when all chunks are delivered. It sets `m3_candidate=false` and records a finding explaining that the listen source is not physical StackChan microphone evidence. `--listen-source stackchan_mic` is intentionally rejected until the command consumes real physical mic turn evidence; it is reserved for the future true mic-driven run after the device path can prove capture, VAD, turn start, acknowledgement playback, answer playback, and barge-in timing in the same traced window.

Reports include the redacted timing and delivery receipt:

```text
reports/a21-stackchan-fast-companion-turn-YYYYMMDD-HHMMSS.json
```

The report records local acknowledgement first-audio timing, answer first-audio timing, playback chunk/batch counts, playback clear status, transport, trace/session ids, device identity, M3-candidate status, and findings. It does not record input text, ASR transcript, provider output text, local acknowledgement text, provider credentials, auth headers, proxy values, full provider URLs, full model paths, or full WAV fixture paths.

If Gateway or the device report is unavailable, the command still writes a failed redacted report so the run leaves a diagnosable artifact instead of disappearing into stderr.

## Boundary

This phase proves only local synthesis and WAV conversion. It does not prove:

- A21 final character voice quality
- streaming TTS first chunk latency
- voice cloning
- physical StackChan audibility
- speaker volume or mouth sync
- full-duplex AEC

Physical speaker acceptance remains under the guarded StackChan hardware path.

## Local Bakeoff Hook

The local workstation may have an ignored model-cache hook at:

```text
.a21-tools/a21-local-model-hook/
```

This hook is not part of the tracked A21 mainline and must not be treated as a product dependency. It is a time-saving pointer for future local ASR/TTS adapter work. Before promoting any candidate, rerun the hook locally and convert the result into a tracked A21 report, adapter plan, or ADR as appropriate.

Useful commands:

```bash
.a21-tools/a21-local-model-hook/a21-local-model-bakeoff.sh summary
.a21-tools/a21-local-model-hook/a21-local-model-bakeoff.sh tts
.a21-tools/a21-local-model-hook/a21-local-model-bakeoff.sh asr
```

The last local bakeoff kept `vits-icefall-zh-aishell3` as the default local TTS candidate, kept `vits-melo-tts-zh_en` and `kokoro-multi-lang-v1_1` as future voice-quality candidates, rejected `matcha-icefall-zh-baker` for now, and identified Paraformer small, SenseVoice, and streaming Zipformer as local ASR candidates worth formal adapter evaluation.

## Current Status

`sherpa-onnx` is installed in the isolated `.a21-tools/sherpa-onnx-venv` toolchain. The selected Chinese VITS model download/extraction must complete before M2 can be promoted from software-ready to evidence-passed. The macOS fallback loopback may continue to validate host-side timing and report redaction, but it is not promoted to the selected local TTS implementation.
