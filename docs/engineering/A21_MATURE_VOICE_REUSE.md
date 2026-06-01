# A21 Mature Voice Reuse Contract

Status: active mainline guardrail.

## Purpose

A21 must stop treating hand-written audio and voice plumbing as a virtue. The project can keep its Go-first spine, A21 protocol, StackChan product semantics, V21 boundary, and provider redaction rules, but established media, voice, transport, tracing, parsing, and firmware tooling should come from mature protocols, libraries, SDKs, or framework patterns whenever they fit.

X21 is also not an authority here. X21 is useful because it contains real StackChan failure evidence and latency lessons, but much of its runtime was hand-built under pressure. A21 may borrow measured parameters and failure taxonomies from X21 only after translating them into A21-owned, provider-neutral contracts. A21 must not copy X21's architecture, Python monolith, flag soup, provider coupling, protocol identity, ports, env names, or firmware upload habits.

## Reuse Ladder

When adding or changing A21 voice/audio/protocol behavior, choose in this order:

1. Official standard or mature protocol: WebRTC, Opus, OpenTelemetry, Prometheus, JSON Schema-style validation, MCP where it fits device tools.
2. Official SDK or maintained library: provider SDKs/protocol clients, WebRTC APM, ESP-SR, ESP-ADF/esp audio codec, M5Unified, StackChan-BSP, M5Stack-Avatar, sherpa-onnx, Silero VAD, Qdrant or equivalent retrieval components.
3. Mature framework pattern wrapped by A21: LiveKit/Pipecat style frame pipelines, turn detection, interruption, ordered system/control/data frame lanes, STT-LLM-TTS streaming graph, lifecycle hooks.
4. Small A21 adapter around a mature component, with A21 trace/session/device IDs, redaction, proxy policy, and tests.
5. Custom implementation only when an ADR explains why the mature option fails A21's latency, firmware, network, privacy, licensing, packaging, or product constraints.

## Current Hand-Rolled Code Classification

The following A21 code is allowed only as a baseline, harness, or compatibility bridge until mature replacements are evaluated:

| Current A21 piece | Allowed role | Mature direction |
| --- | --- | --- |
| `a21-rms-vad` | deterministic tests, trace shape, mock barge-in | WebRTC APM, ESP-SR AFE/AEC/VAD, provider-side turn detection, or measured neural VAD |
| JSON/base64 `pcm_s16le` audio frames | safe early Gateway/firmware loop and diagnostics | A21-owned binary Opus profile for real device media, with PCM kept for debug fixtures |
| firmware base64 PCM playback parser | hardware-free buffer/cancel proof | mature codec/playback path using ESP audio codec or M5/ESP audio components |
| manual speaker/mic switching | diagnostic guard while CoreS3 mic path is unstable | ESP-SR/ESP-ADF/M5Unified path with explicit AEC/full-duplex evidence |
| custom provider websocket parsing | fixture smoke and first wrapper skeletons | official provider SDKs or stable protocol clients hidden behind `internal/providers` |
| ad hoc latency commands | local evidence while the bench contract grows | shared `provider-latency-bench` family following `A21_PROVIDER_BENCHMARKS.md` |

None of these baselines may be promoted to production by inertia.

## StackChan Speaker Baseline

The previous A21 hand-written speaker path produced audible "telegraph" artifacts on real StackChan hardware. That path is frozen as a diagnostic failure case and must not be used as the speaker acceptance baseline.

The accepted speaker baseline is now the official StackChan/CoreS3 codec path:

- official StackChan source is exported from clean Git `HEAD`, even if the local source worktree is dirty;
- CoreS3 output goes through the official `AudioCodec` abstraction and `OutputData`, not a custom A21 `playRaw` loop;
- the codec evidence includes `esp_codec_dev_open`, `esp_codec_dev_write`, `CreateDuplexChannels`, StackChan `audio.cpp`, CoreS3 board config, and Xiaozhi `AudioService` output-task usage;
- the A21 official audio-smoke overlay builds `a21-stackchan-official-audio-smoke.bin` with ESP-IDF and writes only through the guarded `stackchan-official-audio-smoke-flash-*` commands;
- the A21 official PCM bridge overlay builds `a21-stackchan-official-pcm-bridge.bin` with ESP-IDF and uses official `AudioCodec::OutputData` plus a Xiaozhi-style output queue for Gateway-driven PCM playback experiments;
- real-device acceptance requires a continuous two-tone pattern to be heard clearly without the earlier "telegraph" artifact.

Any future real speech downlink work must migrate toward this official codec lane or an A21 adapter over the same mature codec/HAL boundary. The official PCM bridge has a build lane and no-flash plan lane, but remains M3-prep only until a separate execute guard and NVS provisioning receipt exist. The old A21 speaker parser may remain only as a host-side protocol fixture until removed.

## Mature Inputs To Prefer

### Transport And Media

- Browser/operator realtime surfaces should prefer WebRTC or a LiveKit-compatible media plane once A21 needs multi-client monitoring or browser audio.
- StackChan/CoreS3 device media should prefer A21 WebSocket plus Opus before full WebRTC-in-firmware. The A21 protocol stays A21-owned; xiaozhi's `hello/audio_params` and binary Opus behavior are reference material, not identity.
- Opus is the default mature codec target for interactive speech. A21 should evaluate 20 ms, 40 ms, and 60 ms frames with real StackChan CPU, Wi-Fi, and playback evidence instead of copying X21's 60 ms value blindly.
- Host WS-1 Opus encode/decode currently wraps the pinned pure-Go
  `github.com/thesyncim/gopus` library behind `internal/audio/opuscodec`. A21
  keeps third-party types out of Gateway/protocol surfaces; the local 16 kHz and
  24 kHz PCM inputs are normalized to 48 kHz only at this boundary so Opus TOC
  duration stays correct. The product local TTS adapter should prefer producing
  48 kHz PCM directly so this boundary does not need A21-owned upsampling for
  downlink speech. This is host codec evidence, not a product voice-chain
  acceptance result.
- PCM16 JSON/base64 remains acceptable for fixtures, loopback, and diagnostic visibility, not for the final low-latency physical media path.

The live protocol contract defines the frame/envelope, redaction,
direct-connect/proxy, fallback, metrics, and gate expectations for that
direction only. It does not approve a runtime Opus path, a native codec
dependency, WebRTC dependency, Gateway startup, provider execution, firmware
change, or hardware acceptance.

### Audio Front End

- WebRTC Audio Processing Module is the first Gateway/desktop-side candidate for AEC, noise suppression, automatic gain control, and classic voice frontend behavior.
- ESP-SR is the first firmware-side candidate for AFE, AEC, WakeNet, MultiNet, and VADNet diagnostics on ESP32-S3.
- ESP-ADF or Espressif audio codec components should be evaluated before A21 writes custom codec pipelines on firmware.
- Silero VAD and sherpa-onnx are valid local model candidates behind A21 adapters, but they do not replace AEC by themselves.

### Turn Taking And Pipeline Shape

- LiveKit and Pipecat are pattern sources for ordered frame processing, interrupt priority, STT/LLM/TTS streaming, turn detection, and cancel/truncate semantics.
- A21 should adopt the pattern, not necessarily the runtime owner. Provider, V21, privacy, and StackChan semantic events remain A21-owned.
- Interruption is a first-class system/control event. It must preempt playback, cancel provider work, update conversation state, clear or truncate unheard output, and emit trace markers.

### Hardware Expression

- M5Unified and StackChan-BSP stay the first choice for CoreS3 display, speaker, touch, sensors, RGB, and servo APIs.
- M5Stack-Avatar or the original StackChan expression concepts should be preferred before A21 invents a worse face, mouth, blink, gaze, or breath engine.
- Firmware effects that are worse than the original StackChan experience stay diagnostic or planned.

## Mandatory Design Check

Before implementing any new voice/audio/protocol/runtime path, the development thread must record:

- Which mature protocol/library/SDK/framework pattern was checked.
- Whether it is adopted, wrapped, deferred, or rejected.
- If rejected, the ADR or engineering note explaining the A21-specific reason.
- The A21 boundary that hides third-party types from firmware, protocol, product mode logic, and V21 adapter code.
- The trace markers, metrics, and redacted report fields that prove behavior.

For changes touching latency-sensitive behavior, the default answer "we can hand-roll a simple one" is not acceptable unless the result is explicitly labelled as a temporary harness.

## Immediate Mainline Consequences

1. Audio-front-end evaluation remains the promotion gate for VAD/AEC/noise suppression. Its candidate list should stay machine-readable, with availability and placeholder state for WebRTC APM, ESP-SR, provider-side VAD, Silero VAD, and the A21 RMS baseline. The contract should grow before the RMS detector grows features.
2. `docs/engineering/A21_PROVIDER_BENCHMARKS.md` is the shared gate for provider latency and quality. Do not create provider-specific one-off benchmarks unless the shared report shape cannot express the result.
3. The next real media slice should build from the A21 binary Opus transport plan, not more JSON/base64 tuning. That plan is contract-only; it keeps JSON/base64 PCM as the current runtime and fixture lane until future adapter and hardware gates pass.
4. The next full-duplex slice should evaluate WebRTC APM and/or ESP-SR AEC with a speaker-to-mic echo fixture before changing barge-in thresholds. The current report contract explicitly marks those native/runtime adapters unavailable; it is not approval to add native dependencies or execute hardware work.
5. X21-derived fixes must be rewritten through A21 packages and commit messages must cite the X21 source according to `A21_LEGACY_ONE_WAY_REFERENCE.md`.

## Source Pointers

- WebRTC APM: https://webrtc.googlesource.com/src/+/refs/heads/main/modules/audio_processing/g3doc/audio_processing_module.md
- ESP-SR ESP32-S3 guide: https://docs.espressif.com/projects/esp-sr/en/latest/esp32s3/index.html
- ESP-SR AEC: https://docs.espressif.com/projects/esp-sr/en/latest/esp32s3/acoustic_echo_cancellation/README.html
- ESP-ADF overview: https://docs.espressif.com/projects/esp-adf/en/latest/about.html
- ESP-ADF Opus decoder: https://docs.espressif.com/projects/esp-adf/en/latest/api-reference/codecs/opus_decoder.html
- Opus RFC 6716: https://www.rfc-editor.org/rfc/rfc6716
- LiveKit turn detection: https://docs.livekit.io/agents/build/turns/
- Pipecat pipeline and frame processing: https://docs.pipecat.ai/pipecat/learn/pipeline
