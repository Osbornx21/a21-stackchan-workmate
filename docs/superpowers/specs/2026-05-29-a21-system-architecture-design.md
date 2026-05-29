# A21 System Architecture Design

Date: 2026-05-29
Status: approved direction from user, pending user review of written spec
Workspace: `/Users/jiyurun/Documents/New project`

## 1. Mandate

A21 is a new StackChan-centered voice companion platform for the Shanghai office. It must be engineered as a clean new product, not as an X21/V21 continuation. The system must deliver an excellent end-user experience while staying disciplined enough that development does not spiral out of control.

The permanent requirements are:

- Elegant, simple code.
- Observable and repairable engineering.
- Rigorous, pragmatic architecture with no tolerance for hidden state, accidental proxy inheritance, or project identity leaks.
- Extreme end-user experience: low latency, full duplex, voice cloning, knowledge-backed roleplay, expressive StackChan screen/motion/RGB/touch behavior, and NOMI-class emotional coherence.
- Use mature components and proven architectures. Avoid self-built patches where a good wheel exists.
- Keep A21 independent from X21 and V21 in processes, ports, data, branches, names, runtime assumptions, and development workflow.

## 2. Top-Level Decision

A21 will use a Go-first product backbone with Python AI sidecars, ESP-IDF firmware, and a TypeScript operator console.

This is the chosen route:

- **A21 Core, Go:** session orchestration, device gateway, provider contracts, environment guardrails, config, auth, observability, and API.
- **A21 AI Sidecars, Python:** local or experimental ASR/TTS/model adapters where Python has the best ecosystem.
- **A21 Firmware, ESP-IDF C/C++:** StackChan OEM runtime on CoreS3/ESP32-S3.
- **A21 Console, TypeScript/React:** operator UI, device status, provider switching, traces, latency dashboards, and office setup.

The core reason: Go gives the project a stable spine and strong concurrency model; Python remains available where model tooling is strongest; firmware stays close to the device; TypeScript is used only where it naturally belongs.

## 3. Non-Goals

A21 will not:

- Reuse X21 default ports, environment variables, provider defaults, process names, or `.env` files.
- Call V21 services by default.
- Treat local proxy state as invisible infrastructure.
- Build its own vector database, WebRTC stack, Opus codec, tracing framework, or model serving system unless a benchmark proves the existing options fail.
- Optimize for a benchmark that was taken through a polluted network state.
- Ship device firmware changes without a rollback path and real-device evidence.

## 4. Project Namespace

A21 owns its own namespace:

- Environment variables: `A21_*`
- Process names: `a21-*`
- Docker Compose project: `a21`
- Docker volumes: `a21_*`
- Logs: `.a21/logs` or `logs/a21-*`
- Runtime state: `.a21/run`
- Metrics service names: `a21-core`, `a21-voice-sidecar`, `a21-knowledge`, `a21-console`, `a21-stackchan`
- Local port block: `21000-21499`

Initial reserved ports:

- `21080`: A21 Core HTTP API and OTA endpoint
- `21081`: A21 StackChan realtime WebSocket if split from `21080`
- `21073`: A21 Console
- `21086`: A21 local observability UI
- `21095`: A21 local ASR sidecar
- `21114`: A21 local LLM/model adapter
- `21434`: A21 Ollama-compatible adapter if needed

The startup preflight must fail if a reserved port is occupied by a non-A21 process.

## 5. System Decomposition

### 5.1 A21 Core

A21 Core is a Go modular monolith. It owns the authoritative runtime contracts but does not own heavy model inference.

Responsibilities:

- Device OTA endpoint and device session lifecycle.
- StackChan realtime gateway.
- Voice turn orchestration.
- Provider registry and provider contract enforcement.
- Persona and roleplay routing.
- Knowledge query coordination.
- Trace, metric, structured log emission.
- Runtime preflight and environment fingerprinting.
- Operator API for the console.

Internal packages should stay small:

- `runtimeguard`: detects ports, env leaks, proxy mode, cwd leaks, and toolchain state.
- `device`: OTA, device auth, capabilities, heartbeats, and StackChan protocol.
- `voice`: turn state machine, interruption, audio segment routing, latency accounting.
- `providers`: ASR, LLM, TTS, realtime, embedding, reranking, and knowledge provider interfaces.
- `knowledge`: retrieval orchestration and evidence formatting.
- `persona`: role memory, system prompts, safety/personality boundaries, and style policies.
- `observability`: OpenTelemetry, metrics, structured logging, trace IDs.
- `ops`: health checks, readiness, diagnostics, and local office audit helpers.

The core must be understandable without reading sidecar internals.

### 5.2 A21 Voice Pipeline

The default voice architecture is a cascaded streaming pipeline:

`StackChan audio input -> VAD/turn detector -> ASR partials -> brain/router -> LLM stream -> TTS stream -> StackChan playback and semantic events`

This is the practical first path because it allows independent benchmarking of every stage. Native speech-to-speech providers may be added later as provider adapters, but they cannot replace the baseline until they beat it in measured Shanghai office conditions.

Each voice turn gets:

- `session_id`
- `turn_id`
- `audio_chunk_id`
- `asr_partial_id`
- `llm_request_id`
- `tts_segment_id`
- `device_event_id`

Required latency metrics:

- Audio capture to gateway receive.
- Gateway receive to first ASR partial.
- Final/usable ASR to first LLM token.
- First LLM token to first TTS audio.
- First TTS audio to device playback start.
- User interruption to playback stop.
- User interruption to visible StackChan feedback.
- Total time to first audio.

### 5.3 A21 Device Protocol

StackChan starts with WebSocket plus Opus for the embedded device path. Browser/operator surfaces can use WebRTC later through LiveKit or equivalent infrastructure.

Protocol principles:

- WebSocket carries audio, control, state, and device telemetry.
- Audio uses Opus by default, with PCM only for diagnostics.
- Device messages are framed and versioned.
- Every message carries monotonic sequence number, device timestamp when available, server receive timestamp, and trace correlation IDs.
- Device events are semantic, not register-level.
- Protocol supports explicit cancellation and barge-in.
- Protocol supports backpressure so slow TTS or bad network does not create stale speech.

Core message families:

- `hello`: device identity, firmware version, capabilities, hardware profile.
- `audio.input`: microphone audio frames.
- `audio.output`: server audio frames to play.
- `turn.state`: listening, thinking, speaking, interrupted, error.
- `device.face`: expression and intensity.
- `device.display`: subtitle, status, evidence hint, notification.
- `device.motion`: nod, look, idle motion, dance cue, servo-safe movement.
- `device.rgb`: mood color and animation.
- `device.touch`: screen touch, top/body touch, pet feedback, barge-in.
- `health`: battery, Wi-Fi, RSSI, heap, task lag, audio buffer state.

### 5.4 A21 Firmware/OEM Runtime

The firmware is a product surface, not a thin network shim.

Core tasks:

- Audio capture and playback with ring buffers.
- Realtime network session.
- Screen expression runtime.
- Touch/motion/RGB semantic event handling.
- Local idle life and presence behavior.
- Device health telemetry.
- Safe OTA, rollback, and NVS preservation.

Firmware must expose a smooth user experience even when cloud providers are slow:

- Immediate listening animation.
- Immediate interruption animation and playback stop.
- Thinking state that does not feel frozen.
- Speaking state synchronized with audio chunks.
- Error states that feel intentional.
- Local idle and attention shifts so StackChan feels alive.

No firmware provisioning flow may wipe servo calibration or unrelated NVS keys.

### 5.5 A21 Provider Layer

Providers are replaceable adapters. Paid APIs are allowed but never hard-coded into the product logic.

Provider interfaces:

- `ASRProvider`: streaming audio input, partial/final transcript output, timestamps, confidence, cancellation.
- `LLMProvider`: streaming text, tool calls, role/context input, cancellation, token usage.
- `TTSProvider`: streaming text input, audio chunk output, voice clone controls, cancellation, first-audio metrics.
- `RealtimeProvider`: optional native speech-to-speech provider.
- `EmbeddingProvider`: text to dense/sparse embeddings.
- `RerankProvider`: candidate evidence reranking.
- `KnowledgeProvider`: query to evidence set with source metadata.

Initial provider catalog:

- DeepSeek API: LLM candidate, OpenAI-compatible, streaming capable.
- Bailian/DashScope Qwen: LLM candidate, OpenAI-compatible China endpoint.
- DashScope CosyVoice: realtime/streaming TTS and voice clone candidate.
- Local FunASR or SenseVoice-family services: local ASR candidates, isolated as A21 sidecars rather than reusing X21 services.
- OpenAI Realtime: optional reference/native realtime provider, not default in mainland deployment until measured.
- Local Ollama-compatible adapters: offline/diagnostic LLM fallback, not production default.

Provider adapters must record:

- Region and endpoint.
- Network mode.
- First token/audio latency.
- Timeout and retry behavior.
- Whether ambient proxy was used.
- Cost and usage.
- Degraded-mode behavior.

Provider calls to localhost/LAN must never use ambient HTTP/SOCKS proxy settings. Cloud provider calls must declare whether proxy/TUN is allowed.

### 5.6 A21 Knowledge and Roleplay

A21 needs fast but deep roleplay knowledge. The baseline is hybrid retrieval plus reranking, not vector-only retrieval.

Data stores:

- PostgreSQL for durable app metadata, device registry, persona state, evaluation runs, and configuration.
- Qdrant for A21-owned vector/hybrid retrieval.
- Object storage or filesystem-backed blob storage for audio references, voice samples, screenshots, and artifacts.

Retrieval flow:

`query -> rewrite/classify -> metadata filters -> dense retrieval -> sparse/keyword retrieval -> fusion -> rerank -> evidence pack -> LLM prompt`

Roleplay flow:

`turn context -> persona memory -> knowledge evidence -> emotional state -> response plan -> LLM stream -> TTS -> device events`

Rules:

- Evidence-backed knowledge and roleplay style are separate layers.
- The system must know whether it is answering from knowledge, persona memory, or general model ability.
- Evidence shown on StackChan screen must be compact and must not trample subtitles.
- Persona memory must be inspectable and editable from the console.

### 5.7 A21 Console

The console is an operator tool, not a marketing page.

Core views:

- Environment preflight and active network mode.
- Device registry and live StackChan status.
- Voice session timeline.
- Provider status and latency.
- Knowledge corpus and retrieval evaluation.
- Persona/memory editor.
- Firmware build/flash/rollback checklist.
- Office audit checklist.
- Trace and error viewer.

The console must never hide failures behind friendly copy. It should make the system easier to repair.

### 5.8 Observability

OpenTelemetry is the default telemetry model.

Telemetry requirements:

- One trace per voice turn.
- Structured logs with `session_id`, `turn_id`, provider, device ID, network mode, and trace ID.
- Metrics for every latency stage.
- Device telemetry for Wi-Fi RSSI, heap, audio queue depth, playback queue, battery, firmware version, and task lag.
- Provider telemetry for status, latency percentiles, error rate, timeout rate, cancellation rate, and cost.

The first dashboard should answer:

- Why was this turn slow?
- Which component added the delay?
- Did proxy/TUN affect this provider call?
- Did device playback lag behind TTS?
- Did interruption stop audio quickly enough?
- Did StackChan show the right emotion/state?

### 5.9 Runtime Guardrails

A21 must protect itself from X21/V21 and from ambient machine state.

Phase 1 local startup preflight fails on:

- Legacy env vars: `X21_*`, `V21_*`, `ROLEPLAY_*`, `VOICE_KNOWLEDGE_*`.
- Legacy ports when configured as A21 ports.
- Process cwd pointing to `/Users/jiyurun/Documents/小马暴力` or `/Users/jiyurun/Documents/v21-knowledge-platform` for an A21 runtime.
- Provider endpoints pointing to known X21/V21 ports unless running an explicit migration/audit command.
- Missing runtime fingerprint.

Container deployment preflight, before any Docker/Compose-based A21 runtime is introduced, must additionally fail on:

- Docker compose project not named `a21`.
- Docker volume not prefixed `a21_`.

Real-device acceptance preflight must additionally fail on:

- Unknown network mode for real-device acceptance.

The preflight reports but does not stop unrelated legacy projects unless the user explicitly asks.

### 5.10 Development Workflow

Repository rules:

- `main` remains clean and shippable.
- Feature branches use `codex/a21-*`.
- Every change has a verification command.
- No cross-project edits from the A21 workspace.
- New modules require contract tests.
- Device claims require simulator and hardware evidence when hardware is involved.

Document hierarchy:

- `docs/a21/00-project-charter-and-home-baseline.md`: baseline and principles.
- `docs/a21/01-architecture-research-and-options.md`: research and options.
- This file: approved system architecture design.
- Future implementation plans: one plan per phase.

## 6. Validation Strategy

### 6.1 Test Layers

- Unit tests for provider contracts, protocol frames, runtime guards, and state machines.
- Integration tests with fake ASR/LLM/TTS providers.
- Latency tests with deterministic audio fixtures.
- StackChan simulator tests for protocol and semantic events.
- Firmware QEMU/simulation where practical.
- Real StackChan acceptance for audio, screen, touch, motion, RGB, Wi-Fi, OTA, and rollback.
- Office network acceptance after the Shanghai environment is available.

### 6.2 Acceptance Metrics

Initial targets are engineering gates, not final promises:

- Startup preflight explains all blocking conditions.
- Local fake-provider turn produces first audio under 300 ms.
- Cloud-provider benchmark records P50/P95 per stage before any provider is selected as default.
- Interruption stops playback and updates screen within a measured, visible budget.
- No A21 runtime connects to X21/V21 by accident.
- Every real-device test records serial port, firmware version, network mode, provider set, trace IDs, and logs.

### 6.3 Office Audit

When at the Shanghai office, A21 must measure:

- Office authenticated Wi-Fi behavior.
- `wang301` Wi-Fi latency, jitter, client isolation, multicast/mDNS, and throughput.
- Wired LAN path among high-performance PCs, MacBook Pros, and StackChan.
- Whether StackChan can join each network directly.
- Whether Dragon Cat Lite/proxy must be disabled, bypassed, or split-tunneled.
- Which machine should host A21 Core and sidecars.

## 7. Implementation Phases

### Phase 0: Architecture and Guardrails

Output:

- Approved architecture spec.
- A21 namespace.
- Runtime preflight design.
- Provider contract design.
- Office audit checklist.

### Phase 1: Clean Skeleton

Output:

- Go module with `a21-core`.
- Runtime preflight CLI.
- Reserved port checker.
- Environment fingerprint.
- Fake provider interfaces.
- Minimal protocol types.
- CI-style local verification.

### Phase 2: Simulated Voice Turn

Output:

- Fake StackChan client.
- Streaming turn state machine.
- Fake ASR/LLM/TTS adapters.
- Trace per turn.
- Console or CLI timeline for latency.

### Phase 3: Provider Benchmarks

Output:

- DeepSeek LLM adapter benchmark.
- Bailian/Qwen LLM adapter benchmark.
- DashScope CosyVoice TTS benchmark.
- Local ASR sidecar benchmark.
- Provider decision based on latency, quality, reliability, and cost.

### Phase 4: StackChan Device Path

Output:

- A21 firmware protocol implementation.
- Audio input/output path.
- Semantic face/display/motion/RGB/touch events.
- OTA and rollback.
- Real-device trace evidence.

### Phase 5: Shanghai Office Deployment

Output:

- Office network report.
- Host selection.
- LAN/proxy policy.
- Device acceptance report.
- Operator runbook.

## 8. Reference Architecture Sources

- LiveKit Agents: https://docs.livekit.io/agents/
- Pipecat transports: https://docs.pipecat.ai/guides/learn/transports
- W3C WebRTC Recommendation: https://www.w3.org/TR/webrtc/
- IETF RFC 6716 Opus: https://www.rfc-editor.org/rfc/rfc6716
- OpenAI voice agents and Realtime API: https://platform.openai.com/docs/guides/voice-agents
- DeepSeek API docs: https://api-docs.deepseek.com/
- Alibaba Bailian OpenAI-compatible docs: https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope
- Alibaba CosyVoice WebSocket docs: https://help.aliyun.com/zh/model-studio/developer-reference/cosyvoice-websocket-api
- Qdrant hybrid search and reranking: https://qdrant.tech/documentation/advanced-tutorials/reranking-hybrid-search/
- OpenTelemetry docs: https://opentelemetry.io/docs/
- M5Stack StackChan docs: https://docs.m5stack.com/en/StackChan
- M5Stack StackChan repository: https://github.com/m5stack/StackChan
