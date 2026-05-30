# A21 Codex Masterplan

Status: v0.3, aligned with the current Go-first A21 Phase 1 foundation.

## Purpose

This document distills the external A21 master directive into rules that fit the current repository. The source directive is valuable for product soul, network paranoia, voice UX, StackChan semantics, V21 boundary thinking, and observability. Its TypeScript/pnpm monorepo proposal is not adopted as the current structure because A21 Phase 1 has already established a small Go core with runtime guardrails.

The authoritative A21 architecture is the user's own design plus the verified A21 foundation already committed in this repository. External agent outputs and public project patterns are reference material. They can strengthen A21, but they must not replace the user's architecture by drift.

## Product Constitution

A21 is a StackChan-centered desk workmate for high-pressure intelligent-cockpit product work. It is allowed to be warm, funny, and slightly sharp, but it must not become clingy, romantic, childish, or therapeutic. It is a colleague-like presence that can listen, help reframe office frustration, co-create language, and switch into professional evidence mode.

A21 is not:

- a generic voice assistant
- a startup coach
- a therapy bot
- a customer-service bot
- a pure search UI
- a toy that only performs cuteness
- a V21 voice shell

## Architecture Direction

Current approved direction:

- Go owns the A21 core, CLI, runtime guardrails, protocol contracts, provider contracts, and future gateway spine.
- Python may be introduced later for ASR/TTS/model sidecars where the ecosystem justifies it.
- TypeScript/React may be introduced later for simulator/dev console surfaces.
- Firmware remains a separate StackChan lane and must keep provider secrets, proxy logic, and V21 access out of the device.

StackChan is responsible for sensing and expression. A21 Core/Gateway is responsible for thinking, providers, proxy policy, V21 bridging, session state, and observability.

## Architecture Governance

A21 must stay disciplined at the architecture and software-engineering level. Mature, proven libraries and official SDKs are preferred for established infrastructure such as WebSocket transport, metrics, parsing, audio codecs, provider APIs, firmware build systems, and observability export.

Custom A21 code should focus on:

- A21 protocol and mode semantics
- StackChan expression and device contracts
- provider adapter boundaries
- proxy and China-mainland network guardrails
- X21/V21 contamination defense
- product experience orchestration
- observability correlation

Do not hand-roll mature infrastructure just to keep code "pure". If A21 rejects a mature library or SDK, write an ADR explaining the tradeoff, maintenance cost, and replacement path.

## Voice Strategy

A21 must support two voice paths:

1. Companion fast path: optimized for natural low-latency, interruption, emotion, and workmate presence.
2. Professional path: STT/text-visible, V21 retrieval, evidence handling, summarization, and TTS with observable intermediate states.

The professional path must not depend on an opaque end-to-end speech model as the only source of truth. It needs auditable query, retrieval, evidence, confidence, and response blocks.

## Runtime Guardrails

Phase 1 local preflight already enforces:

- legacy env prefixes are blocked
- A21 endpoint env vars pointing at known X21/V21 ports are blocked
- legacy working directories are blocked
- A21 reserved port conflicts are blocked
- minimum network/DNS fingerprint is required

Future container preflight must add:

- compose project identity checks
- Docker volume namespace checks
- service name and container label checks

Future real-device acceptance must add:

- audio device checks
- serial identity checks
- StackChan reachability
- LAN/proxy bypass checks
- first-audio and barge-in timing evidence

## Provider Policy

DeepSeek, Bailian/DashScope, Doubao, OpenAI, local ASR/TTS, and future providers are resources, not architecture. They must enter through provider adapters with:

- explicit network/proxy mode
- health checks
- latency probes
- cancellation semantics
- cost/usage accounting
- structured error codes

The core protocol and business logic must stay provider-neutral.

## V21 Boundary

V21 is a professional knowledge system, not an A21 internal module. A21 must call V21 only through a clear adapter. It must not directly read V21 databases, assume V21 chunk schema, inherit V21 ports, or reuse V21 service names.

Professional mode can send a user-confirmed query to V21. Companion/private/roleplay content must not automatically become V21 retrieval context.

## StackChan Experience Rules

The screen is a face, not a small phone. Servos express attention, not decoration. RGB indicates state, not visual noise. Touch, screen, audio, mouth, motion, and state must be semantically coordinated.

First firmware acceptance must prove:

- connected state
- listening/thinking/speaking/interrupted/error expressions
- safe servo clamp
- local fallback when gateway is unavailable
- no provider secrets in firmware

## Observability Rules

Any voice delay must be traceable. A turn should eventually expose:

- capture start/end
- uplink first frame
- VAD start/end
- ASR first partial
- LLM first token
- V21 query start/first result/error/timeout
- V21 query latency histogram
- TTS first chunk
- downlink first frame
- device playback start
- barge-in detection
- provider cancel
- playback stop
- fallback and error code

Phase 1 has the first runtime fingerprint. Current Gateway observability includes in-memory trace markers and Prometheus metrics for mock turns, barge-in, audio frames, mock playback chunk sends, invalid firmware identity, WebSocket connections, and professional V21 query latency. Later phases must add OpenTelemetry spans, structured logs, and latency benchmark reports.

## Phase Roadmap

Phase 0: repository governance and engineering docs. Completed by this document set plus `AGENTS.md`, `PORTS.md`, `NETWORK.md`, `LATENCY_BUDGET.md`, `OBSERVABILITY.md`, `PROTOCOL.md`, and `V21_INTEGRATION.md`.

Phase 1: Go core foundation. Current baseline includes build identity, runtime guardrails, preflight/doctor, tracked-path namespace audit, protocol contracts, provider contracts, a root README entrypoint, GitHub Actions release-check CI, and verification.

Phase 2: gateway mock and simulator. Current baseline includes HTTP health, audio/control WebSocket boundaries, deterministic mock voice provider, session/trace propagation, state transitions, device registry, in-memory trace waterfall, trace latency summary, office visibility mode indicators, and a built-in browser simulator that can show Gateway audio downlink chunk counts, buffer depth, active stream ID, and key latency summary fields.

Phase 3: doctor and observability expansion. Current baseline includes richer reports, firmware/toolchain/artifact/serial diagnostics, Prometheus-compatible metrics, a mock trace waterfall with computed summary fields, optional V21 health checks, explicit multi-sample LAN TCP reachability/jitter probe receipts, and mock latency-bench reporting for mock turn, professional turn, barge-in, audio WebSocket downlink shape, current commit, network fingerprint, and redacted proxy-policy metadata. OpenTelemetry export and real provider/device latency benches remain future work.

Phase 4: provider adapters. Current baseline includes mock and cascade voice providers, provider-neutral health status, explicit cancel reason/stream acknowledgements, OpenAI/Doubao realtime adapter boundaries, selected-provider doctor health, redacted provider smoke evidence reports, redacted realtime plans for OpenAI realtime, Doubao realtime speech-to-speech, and Doubao realtime TTS, an explicit Gateway runtime provider guard, offline realtime fixture smoke through fake WebSocket dialers, provider-neutral Gateway realtime session start/cancel endpoints with trace markers and latency metrics, provider audio delta mapping into A21 `audio.playback.chunk` envelopes, and OpenAI realtime WebSocket output reads mapped back into A21 `VoiceEvent` downlink events. Gateway remains mock by default; `A21_GATEWAY_VOICE_PROVIDER=selected` is required before the provider selected by `A21_PROVIDER_PRIMARY` enters Gateway runtime. Professional mode is kept out of the realtime fast path and must use the V21 evidence path. Next slices should add real provider credential smoke and physical-device audio acceptance behind this boundary.

Phase 5: StackChan firmware MVP. Current baseline includes A21-only CoreS3 firmware identity, repository-local pinned PlatformIO bootstrap, Wi-Fi config/runtime guards, control/audio WebSocket transport probes, mock Gateway audio downlink chunks, bounded firmware playback buffering, reconnect/local fallback state, semantic motion runtime with Y-axis servo clamp safety, semantic RGB state runtime, semantic touch intent runtime, playback start/stop/clear state machine, build/package/upload/device-identity dry-run discipline, release index plus per-artifact manifest guards, latest same-commit upload guard, timestamped no-flash flash-plan receipts, PlatformIO build-provenance checks, and native tests. Real mic capture/playback samples, calibrated servo/RGB hardware output, calibrated screen/top-sensor touch hardware mapping, OTA, and physical flashing remain future work.

Phase 6: V21 professional mode. Current baseline includes a V21 adapter contract, HTTP/mock clients, Gateway professional mode routing, explicit evidence fields, confidence, speech blocks, failure fallback, trace markers, optional doctor health checks, simulator evidence-card rendering, and redacted `v21-adapter-smoke` readiness/execution reports for the adapter boundary. Real Shanghai endpoint execution, endpoint-specific permission checks, and timeout metrics remain future work.

Phase 7: low-latency and full-duplex optimization. Current baseline includes bounded audio ingress buffering, an explicit VAD detector adapter boundary with deterministic RMS default, detector-labelled VAD decision metrics, `audio-front-end-plan` for mature VAD/AEC candidate governance, `audio-front-end-eval --mock` and `audio-front-end-eval --fixture` for deterministic/labelled-frame baseline reporting with speech start/end lag, timestamped audio front-end eval report artifacts, audio-path barge-in cancellation, provider-neutral realtime audio uplink forwarding, provider-neutral realtime audio downlink pumping, and commit-to-first-provider-audio waterfall metrics for providers that expose an explicit realtime session interface. This still does not claim production VAD, AEC, hardware full-duplex, real provider latency, or physical-device latency acceptance. Next slices should add mature VAD/AEC adapter implementation spikes and real StackChan mic/speaker acceptance.

Phase 8: product polish. Add personality prompts, office scenario playbooks, public/private transitions, failure copy, and expression polish.

## Rule For External Directives

When another process proposes a broad architecture, use this filter:

- User-originated A21 decisions have priority over external agent output.
- Keep product soul, constraints, tests, and observability ideas.
- Reject stack rewrites that conflict with the approved A21 spine unless an ADR justifies them.
- Convert useful future ideas into phased docs instead of immediate code churn.
- Never let a long directive override verified local evidence.
