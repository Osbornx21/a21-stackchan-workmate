# A21 Architecture Research and Options

Date: 2026-05-29
Status: research/options, not the approved implementation spec.

## Scope

This note collects architecture references for A21 before implementation starts. It focuses on the first irreversible choices: realtime transport, voice pipeline shape, language/runtime split, provider abstraction, knowledge retrieval, observability, and isolation from X21/V21.

## Source-Backed Findings

### Realtime Voice Transport

- WebRTC is the mature standard path for browser/mobile realtime audio. W3C defines the browser API and IETF standardizes the protocol family. OpenAI and LiveKit both recommend WebRTC for low-latency client-side voice use cases.
- WebSocket remains practical for server-to-server and embedded device links. Pipecat frames WebRTC as recommended for client applications and WebSocket as good for server-to-server paths.
- StackChan/CoreS3 is an ESP32-S3 device. For the device link, WebSocket plus Opus is a lower-risk embedded path than trying to force full WebRTC into firmware on day one.
- Opus is the right audio codec default for interactive speech. RFC 6716 defines Opus as an interactive speech/audio codec, and Xiph notes frame sizes from 2.5 ms to 60 ms.

Implication for A21: use dual transport boundaries. Browser/operator surfaces can use WebRTC; StackChan should start with an A21-owned WebSocket/Opus protocol and a semantic device-event channel.

### Voice Pipeline Shape

- LiveKit Agents explicitly handles streaming audio through STT -> LLM -> TTS, turn detection, interruptions, and orchestration.
- Pipecat uses a pipeline of processors for realtime audio/text/video frames and supports modular transports.
- A 2026 realtime voice-agent paper reports that the practical industry pattern is still a cascaded streaming pipeline, with realtime behavior coming from pipelining across STT, LLM, and TTS rather than one magic model. It reports native speech-to-speech can be too slow in some cases and gives a measured P50 time-to-first-audio around 947 ms for a tuned cascaded pipeline.
- OpenAI Realtime is a strong reference for native speech-to-speech plus tool calling, but provider availability and China network constraints mean A21 should treat this as an optional provider path, not the only architecture.

Implication for A21: first architecture should be a cascaded streaming pipeline with clean provider adapters and enough abstraction to later test native speech-to-speech providers.

### China Provider Resources

- DeepSeek's official API is OpenAI-compatible and supports streaming. Current official docs list V4 model names and note older aliases are scheduled for deprecation on 2026-07-24.
- Alibaba Bailian/DashScope exposes OpenAI-compatible chat endpoints using `https://dashscope.aliyuncs.com/compatible-mode/v1` for Beijing, with separate endpoints for Singapore and US.
- DashScope CosyVoice exposes realtime TTS over WebSocket at `wss://dashscope.aliyuncs.com/api-ws/v1/inference` for China mainland service scope.
- CosyVoice has official/open research and code lineage for multilingual zero-shot and streaming TTS. It is relevant to voice cloning, but A21 should benchmark it rather than assume it wins.

Implication for A21: define provider contracts around streaming behavior, first-byte/first-audio latency, cancellation, cost, and region/network mode. Do not let API availability choose the architecture.

### Knowledge Retrieval

- Qdrant supports dense, sparse, hybrid search, and reranking flows. Its docs emphasize hybrid retrieval plus reranking as a way to keep latency low while improving relevance.
- For A21 roleplay and knowledge, vector-only retrieval is too brittle. Product experience needs exact names, facts, dates, and style memory, so hybrid search plus reranking is the safer baseline.

Implication for A21: use a separate A21 knowledge service with Qdrant or equivalent hybrid retrieval, explicit reranking, and deterministic metadata filters. Do not connect to V21 containers by default.

### Observability

- OpenTelemetry is vendor-neutral and covers traces, metrics, and logs. Its observability primer frames traces as the path of a request across services.
- Voice agents need finer granularity than generic HTTP spans. A21 should trace `session_id`, `turn_id`, `audio_chunk_id`, `asr_partial_id`, `llm_request_id`, `tts_segment_id`, and `device_event_id`.

Implication for A21: observability is a first-class product feature. The system should make latency visible before anyone tries to optimize it.

### StackChan Device Reality

- M5Stack's official StackChan repository describes CoreS3 hardware with ESP32-S3, 16 MB flash, 8 MB PSRAM, capacitive touch display, dual microphones, speaker, camera, servos, RGB LEDs, IR, NFC, and touch panel.
- This is not a tiny ESP32-C3 class terminal. A21 should expose high-level semantic events for face, display, motion, touch, RGB, and companion state.

Implication for A21: firmware should be treated as an OEM product surface, not a thin transport shim.

## Architecture Options

### Option A: Go Core, Python AI Sidecars, ESP-IDF Firmware

Shape:

- Go modular monolith for A21 API, session orchestration, device gateway, provider contracts, auth/config, health checks, and observability.
- Python sidecars for local ASR/TTS/model tooling where the ecosystem is strongest.
- ESP-IDF C/C++ firmware for StackChan.
- TypeScript/React for operator console.

Pros:

- Strong control-plane reliability and concurrency.
- Clear provider isolation.
- Good long-term operations story with single-binary Go services.
- Python is contained to model/AI adapters instead of owning the whole system.

Cons:

- More initial adapter work.
- Some realtime voice frameworks are Python/TypeScript-first, so direct reuse may need wrapping.

### Option B: Python/Pipecat-Led Realtime Core

Shape:

- Python owns the realtime voice pipeline using Pipecat-style processors.
- FastAPI/WebSocket device gateway.
- Provider integrations stay mostly native to Python.
- ESP-IDF firmware and TypeScript console remain separate.

Pros:

- Fastest path to experimental voice pipelines.
- Best access to AI/ASR/TTS ecosystem.
- Easy to benchmark multiple providers.

Cons:

- Higher risk of runtime drift, dependency sprawl, and long-running process fragility.
- Harder to make the core feel like a disciplined product platform unless boundaries are extremely strict.

### Option C: LiveKit-Centered Media Plane

Shape:

- LiveKit handles realtime rooms/media for browser/mobile/operator surfaces.
- A21 gateway bridges StackChan WebSocket/Opus into the media/session layer.
- Agents run in LiveKit Agents or an adjacent service.

Pros:

- Strong existing solution for WebRTC, room/session media, interruption, and multi-client monitoring.
- Good fit if office operators, phones, or dashboards need to join sessions.

Cons:

- More moving parts early.
- StackChan still needs a custom embedded bridge.
- May be too much infrastructure before the device path is proven.

## Recommendation

Start with Option A as the product backbone, but borrow proven concepts from Pipecat and LiveKit:

- A21 Core in Go for discipline, isolation, session state, gateway, provider contracts, and observability.
- A21 Voice Pipeline as a streaming graph concept: transport input -> VAD/turn detector -> ASR partials -> brain/router -> LLM stream -> TTS stream -> device playback/control.
- Python sidecars only where model ecosystem requires them.
- LiveKit remains a candidate for browser/operator WebRTC once the StackChan path is stable.
- Pipecat remains a reference for pipeline semantics and benchmarking, not necessarily the runtime owner.

This gives A21 a clean spine without ignoring mature voice-agent work.

## Defensive Isolation Rules

- A21 runtime must fail if it sees legacy ports, env vars, compose projects, or cwd references unless explicitly running a migration/audit command.
- Provider adapters must default to no ambient proxy inheritance for localhost/LAN and must record network mode for cloud calls.
- Every local service must include `a21` in process name, logs, metrics service name, compose project, and volume prefix.
- A21 startup must emit an environment fingerprint before opening device connections.
- A21 must distinguish `dev`, `lab`, `office`, and `device-acceptance` modes.

## Research References

- LiveKit Agents introduction: https://docs.livekit.io/agents/
- Pipecat transports: https://docs.pipecat.ai/guides/learn/transports
- W3C WebRTC Recommendation: https://www.w3.org/TR/webrtc/
- IETF RFC 6716 Opus: https://www.rfc-editor.org/rfc/rfc6716
- Xiph Opus announcement: https://xiph.org/press/2012/rfc-6716/
- OpenAI voice agents: https://platform.openai.com/docs/guides/voice-agents
- OpenAI Realtime WebRTC: https://platform.openai.com/docs/guides/realtime-webrtc
- DeepSeek official API docs: https://api-docs.deepseek.com/
- Alibaba Bailian OpenAI-compatible docs: https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope
- Alibaba CosyVoice WebSocket docs: https://help.aliyun.com/zh/model-studio/developer-reference/cosyvoice-websocket-api
- CosyVoice repository: https://github.com/FunAudioLLM/CosyVoice
- Qdrant hybrid search and reranking: https://qdrant.tech/documentation/advanced-tutorials/reranking-hybrid-search/
- OpenTelemetry docs: https://opentelemetry.io/docs/
- M5Stack StackChan docs: https://docs.m5stack.com/en/StackChan
- M5Stack StackChan repository: https://github.com/m5stack/StackChan
