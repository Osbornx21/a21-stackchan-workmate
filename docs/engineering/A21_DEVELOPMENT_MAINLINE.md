# A21 Development Mainline

Status: v0.4, aligned with `docs/prd/A21_PRD.md`.

## Source Of Truth

`docs/prd/A21_PRD.md` is the current product-level directive for A21. It supersedes older roadmap ordering where it conflicts with product priority, but it does not repeal verified local engineering facts:

- Go-first A21 Core/Gateway remains the approved foundation.
- StackChan remains a thin device for sensing and expression.
- Branch, tool-tier, and Codex-thread control now live in `docs/engineering/A21_PROJECT_CONTROL.md`; hardware-write work must follow that control document before any implementation thread resumes.
- Firmware build, package, artifact, and guarded flashing discipline remain mandatory.
- A21/X21/V21 naming, process, port, env, log, report, and artifact isolation remains mandatory.
- Provider keys never enter firmware, docs output, reports, traces, logs, screenshots, Git, or device messages.
- External latency labs and procurement experiments stay outside the A21 mainline; only redacted conclusions and profile/env names return.
- Mature protocol, library, SDK, and framework-pattern reuse is mandatory for established voice/audio infrastructure. `docs/engineering/A21_MATURE_VOICE_REUSE.md` is the active guardrail for avoiding X21-style hand-rolled media, VAD/AEC, provider, transport, and benchmarking drift.
- X21 is now a read-only, one-way reference source for prior real-device voice lessons. Borrowed parameters, algorithms, or state-machine rules must be rewritten through A21 provider-neutral packages and named in the commit body according to `docs/engineering/A21_LEGACY_ONE_WAY_REFERENCE.md`.

## Current Mainline

The active product-engineering mainline is now:

```text
StackChan embodied device foundation
  + Provider Spine / text streaming hot plug
  + Local audio front-end boundaries
  -> Fast Companion Hybrid Lane
  -> Professional V21 evidence lane
  -> Controlled realtime voice lane
  -> AgentTask bridge
  -> Product polish and office scenarios
```

This means Provider Spine and mainland-latency-controlled fast companion behavior remain the next software pressure, while the already-built StackChan firmware/link discipline remains a non-regression base.

Current active slice: finish the StackChan hardware mainline before moving deeper into Provider Spine. The slice is deliberately non-destructive: use `stackchan-hardware-mainline` to read Gateway device capability declarations, preserve the no-flash discipline, and lock the next hardware tracks into an ordered diagnostic path.

## Priority Order

### P0. Provider Spine / Text Stream Hot Plug

Goal: make provider selection profile-driven without changing Gateway business logic, but keep the current P0 execution path narrow until the vertical StackChan experience has evidence.

Required shape:

- provider family vocabulary remains `text_stream`, `voice_realtime`, `voice_hybrid`, `agent_task`, and `local_audio`;
- the current P0 executable registry is deliberately narrow: `mock`, `deepseek`, and `local_ollama`;
- Baidu and Huawei are blocked provider names, not candidates;
- local overrides through `A21_PROVIDER_PROFILES_PATH` are deferred until the single DeepSeek route has clean smoke and latency evidence;
- provider smoke and doctor output show env names, host, status, timing, and redacted findings only.

### P0. Fast Companion Hybrid Lane

Goal: achieve low perceived latency through the best measured combination of local audio front-end, cloud/local ASR, streaming text provider, and cloud/local/streaming TTS. Local ASR/TTS are the current controlled baseline, not a permanent rule.

Mature-reuse rule: new work in this lane must check `docs/engineering/A21_MATURE_VOICE_REUSE.md` before adding hand-written transport, VAD, AEC, codec, provider websocket, TTS pacing, or benchmark logic. A temporary harness is allowed only when it is labelled as such and has a mature replacement path.

Default lane:

```text
StackChan mic
  -> Gateway VAD/barge-in
  -> measured ASR lane: local or cloud API
  -> OpenAI-compatible streaming text provider
  -> measured TTS lane: local, cloud API, or streaming TTS
  -> StackChan speaker/screen/servo/RGB/touch state
```

Acceptance target:

- first audible response P50 < 900 ms and P95 < 1500 ms for the candidate chain;
- barge-in local stop/cancel P95 < 300 ms;
- first byte, first content, TTS first audio, downlink first frame, and device playback start are recorded.
- ASR/TTS selection is evidence-led: compare local and API candidates on mainland-network first response, P95 tail, subjective voice quality, interruption behavior, privacy boundary, failure behavior, cost, and implementation risk before promoting a lane.
- Provider comparison must reuse the shared benchmark contract in `docs/engineering/A21_PROVIDER_BENCHMARKS.md` instead of creating provider-specific gates. External benchmarks can supply measurement vocabulary and fixture ideas, but accepted A21 evidence must preserve A21 trace/session/device IDs, redacted network/proxy metadata, per-stage waterfall timings, and physical StackChan markers when hardware latency is claimed.
- Voice/audio implementation must prefer mature inputs: WebRTC APM for Gateway/desktop AEC/NS/AGC evaluation, ESP-SR or ESP-ADF for firmware-side AFE/AEC/codec evaluation, Opus for real device media, official provider SDKs or stable protocol clients inside `internal/providers`, and LiveKit/Pipecat-style frame/interrupt patterns when shaping the pipeline.
- Experiments that need a mainland-network external lab must be packaged as isolated probes with redacted JSON reports and no provider keys in Git, reports, logs, screenshots, or firmware.

### P0. StackChan Physical Acceptance

Goal: keep the embodied device foundation real while Provider Spine advances.

Already preserved:

- A21 firmware identity and guarded flash discipline;
- screen expression, RGB, Y-axis servo, touch semantics, speaker buffer, audio frame uplink, runtime echo, and full hardware capability declaration.

Next accepted work must improve or protect actual product behavior. Planned hardware such as camera, IMU, ambient light, proximity, battery, NFC, infrared, and second servo axis stays `planned_*` until implementation and evidence prove it is not worse than original StackChan behavior.

Active hardware order:

1. IMU read-only diagnostic probe.
2. Ambient light read-only diagnostic probe.
3. Proximity read-only diagnostic probe.
4. Battery read-only diagnostic probe.
5. Second servo axis safety spike.
6. Camera privacy-safe vision spike.
7. NFC explicit opt-in interaction spike.
8. Infrared explicit opt-in interaction spike.

Acceptance entry:

```bash
go run ./cmd/a21 stackchan-hardware-mainline --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --output-dir reports
```

The command is a planning and diagnostic gate only. It must not flash firmware, delete artifacts, or promote a planned capability to `available`.

Current progress: the IMU track now has a guarded native runtime boundary, isolated `a21_stackchan_cores3_imu_probe` build lane, raw-upload blocker, guarded flash plan/execute lane, and `stackchan-imu-probe-acceptance` report gate. It remains diagnostic-only until physical evidence proves posture and motion telemetry improve A21's embodied behavior. The sensor track now has a guarded `a21_stackchan_cores3_sensor_probe` build lane using the mature M5CoreS3 LTR553 path for ambient/proximity telemetry and StackChan-BSP INA226 for battery voltage/current telemetry, raw-upload blocker, guarded flash plan/execute lane requiring all three diagnostic markers, and `stackchan-sensor-probe-acceptance` report gate. Release firmware still declares those capabilities as planned until physical evidence proves product value.

Provider spine progress: `ProviderProfile` is now the single registry shape, and the built-in catalog covers the PRD reference profiles `siliconflow`, `deepseek`, `stepfun`, `bailian_dashscope`, `moonshot`, `volcengine_ark`, `local_ollama`, `local_vllm`, `openai_realtime`, `doubao_realtime`, `doubao_tts_realtime`, `hermes_agent`, and `mimo_agent`. This is readiness visibility, not broad execution authorization: the P0 route-eligible set remains intentionally narrow at `mock`, `deepseek`, and `local_ollama`. Baidu/Huawei candidates remain blocked and redacted. DeepSeek is the P0 OpenAI-compatible cloud text-stream profile with a default `deepseek-chat` model and the lab key name `A21_LAB_DEEPSEEK_API_KEY`; `local_ollama` is the P0 local-fallback text-stream lane through Ollama `/api/chat`, with explicit local base URL and model env requirements. `provider-smoke --provider deepseek --stream --repeat N` and `provider-smoke --provider local_ollama --stream --repeat N` record redacted first-byte, first-content, total-duration, fallback, trace, and metric evidence while preserving the existing no-network-without-`--execute` rule. Other provider families and profiles stay as known non-route-eligible readiness or plan boundaries until a follow-up slice promotes them with evidence.

Local audio progress: `sherpa-onnx` is now the selected M2 local TTS baseline, with macOS `say + afconvert` retained only as a diagnostic fallback. Local is not a permanent ASR/TTS rule; API ASR/TTS candidates must be compared through Provider Spine or isolated mainland-lab probes before promotion. `local-tts-smoke` produces redacted evidence and a 16 kHz mono PCM WAV without provider keys, global proxy dependence, firmware changes, or final-voice claims. `local-asr-smoke` now verifies the isolated sherpa-onnx ASR runtime against local Paraformer/SenseVoice/Zipformer-compatible model directories through A21's tracked runner; reports keep decode timing, RTF, transcript length, model basename, and WAV basename, but never transcript text or full local paths. Product readiness treats the default repo-local sherpa ASR cache as an available host-local ASR candidate when no explicit ASR env is set; this is static readiness only and does not execute ASR or change Gateway's default mock runtime. `local-voice-loopback --asr-provider sherpa_onnx` now passes the local ASR transcript internally into the text-stream step while keeping the saved report redacted, and still supports mock ASR as the default stable path. The loopback stitches mock VAD, selected ASR, local acknowledgement TTS, mock OpenAI-compatible text-stream parsing or explicit DeepSeek text-stream execution, selected answer TTS, and existing Gateway barge-in bench into one host-side redacted timing receipt with separate `local_ack_*` perceived-response timing and `answer_first_audio_total_*` provider-backed answer timing. The explicit DeepSeek path requires `--text-provider deepseek --execute-text-provider`, consumes `A21_LAB_DEEPSEEK_API_KEY`, feeds provider content to answer TTS, and still keeps input/ASR transcript/provider output/reasoning/local acknowledgement text/model/secret values out of reports. `stackchan-local-tts-playback` sends selected local TTS PCM chunks through Gateway to the real StackChan playback path. `stackchan-fast-companion-turn` is now the M3-prep vertical receipt: it delivers both local acknowledgement audio and answer audio through the real StackChan downlink path, but defaults to `listen_source=host_fixture` and therefore marks `m3_candidate=false` until future `stackchan_mic` evidence proves the same turn was driven by physical microphone capture. The isolated `.a21-tools` sherpa runtime, selected Chinese VITS model, and first local ASR fixtures are installed; physical microphone capture remains blocked by the current firmware mic stop-crash guard and real DeepSeek timing evidence requires the lab key to be exported in the shell.

Provider benchmark progress: `docs/engineering/A21_PROVIDER_BENCHMARKS.md` is now the mainline contract for importing public STT/TTS/LLM/S2S benchmark methodology without adopting external architecture. Future real-provider latency work should extend the existing A21 report family toward `provider-latency-bench` semantics instead of adding one-off gates per provider.

Mature voice reuse progress: `docs/engineering/A21_MATURE_VOICE_REUSE.md` is now the mainline contract for turning "do not reinvent the wheel" into executable review pressure. Current A21 hand-written pieces such as RMS VAD, JSON/base64 PCM frames, manual firmware mic/speaker switching, and wrapper-level provider websocket parsing are explicitly classified as baselines or harnesses until mature alternatives are evaluated.

StackChan speaker correction: real-device testing proved the previous hand-written A21 speaker path produced "telegraph" artifacts and is not acceptable as a product or hardware baseline. A21 now has an official StackChan/CoreS3 codec smoke lane that exports official StackChan from Git `HEAD`, verifies mature codec evidence, applies the narrow `a21-official-audio-smoke` overlay, builds with ESP-IDF, records all flash parts from `flash_args`, and flashes only through the guarded `stackchan-official-audio-smoke-flash --execute` command. User acceptance passed on the physical device, so future audio downlink work must migrate toward this official codec/HAL boundary rather than tune the old `playRaw` path. The current M3-prep bridge is guarded in three layers: `stackchan-official-pcm-bridge-build` produces `a21-stackchan-official-pcm-bridge.bin` from official StackChan Git `HEAD`, uses official `AudioCodec::OutputData` with a Xiaozhi-style output queue, and reads `a21/device_id` plus `a21/audio_ws_url` from NVS; `stackchan-official-pcm-bridge-nvs --execute` backs up and rewrites only the NVS partition at `0x9000/0x4000`, preserving existing entries while mutating only those two A21 namespace keys; `stackchan-official-pcm-bridge-flash --execute` validates and flashes the bridge app only through the foreground T7 hardware guard and exact app-flash confirmation token.

### P0. Professional V21 Evidence Lane

Goal: keep professional mode auditable and separate from opaque realtime voice.

Rules:

- professional mode calls only the A21 V21 adapter contract;
- companion/private/roleplay content does not automatically become V21 retrieval context;
- public mode does not externalize sensitive evidence;
- every answer keeps `trace_id`, source evidence, confidence, screen cards, and fallback behavior.

### P1. Controlled Realtime Voice Lane

Goal: keep speech-to-speech providers as controlled optional capability, not default architecture.

Rules:

- provider events are converted to A21 provider-neutral events before Gateway or firmware sees them;
- physical StackChan requires one-shot `realtime_on_next_speech` arming before paid realtime provider startup;
- professional mode is explicitly rejected by the realtime lane;
- fixture smoke comes before real network execution.

### P1. AgentTask Bridge

Goal: use mature agent frameworks for long tasks without letting them own A21.

Rules:

- `hermes_agent` and `mimo_agent` are `agent_task` profiles only;
- agent output maps back into A21 semantic events;
- agents cannot write firmware commands, provider env, V21 internals, or Gateway runtime state directly;
- the lane is disabled until explicit `A21_AGENT_PROVIDER_PRIMARY` and smoke pass.

### P2. Product Polish

Goal: turn the working system into a desk workmate people want to talk to.

Work includes:

- personality files instead of one giant prompt;
- office scenario playbooks;
- public/private/focus transitions;
- roleplay and co-creation flows;
- failure copy that preserves A21's presence.

## Non-Regression Rule

No PRD-driven task may regress:

- guarded firmware package/flash flow;
- StackChan capability reporting;
- LAN/proxy directness checks;
- provider report redaction;
- A21 namespace audit;
- V21 adapter boundary;
- Gateway default mock safety.

If a new PRD slice requires changing one of these, it needs an ADR before code.
