# A21 Phase 4A Mock Voice Provider

## Purpose

Phase 4A introduces the first realtime voice-provider boundary without connecting real Doubao, OpenAI, Bailian, or local model services.

The goal is architectural control: gateway behavior now flows through a provider adapter contract instead of embedding provider-like behavior directly in HTTP or WebSocket handlers.

## Current Contract

`internal/providers` now includes:

- `VoiceSession`
- `VoiceTurnRequest`
- `VoiceCancelRequest`
- `VoiceEvent`
- `VoiceProviderHealth`
- `VoiceProvider`
- `MockVoiceProvider`

The default gateway uses `MockVoiceProvider`. Tests can inject a scripted provider through `NewServerWithOptions`.

`VoiceCancelRequest` carries the cancellation reason and optional stream ID. `VoiceEvent` echoes cancellation reason and stream ID when a provider acknowledges cancel. This keeps barge-in semantics explicit before real provider-specific `cancel`, `truncate`, or `session reset` calls are added.

`VoiceProvider.Health(ctx)` returns provider name, status, configured state, realtime capability, active child provider when applicable, and a short detail field. The deterministic mock reports `healthy`, `configured=true`, and `realtime=true`.

Gateway exposes the current voice provider health at:

```text
GET /v1/providers/voice/health
```

Healthy or degraded providers return HTTP 200 with lower-case JSON fields. Unavailable providers return HTTP 503 with the same response shape. This endpoint is the first provider observability boundary for future Doubao/OpenAI/Bailian adapters; it must not expose provider credentials or SDK-specific internals.

`a21 doctor` also includes a `voice` section that uses the same provider health contract. Today it reports the deterministic mock provider; future real provider selection must update doctor and Gateway through the same construction path so Shanghai-office debugging sees the same adapter state from CLI and HTTP.

The same `voice` doctor section now includes a provider network report from `internal/providers.NetworkPolicyFromEnv`. This report deliberately records network mode and env variable names only. By default provider HTTP clients are `direct` and explicitly disable ambient `HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY` inheritance. `A21_PROVIDER_PROXY_URL` switches provider HTTP clients to `explicit_proxy` mode for future cloud egress, but that setting does not change LAN, StackChan, or V21-local routing.

The voice section also includes a provider registry report from `internal/providers.ProviderCatalogFromEnv`. It currently audits readiness for:

- `mock`
- `doubao_realtime`
- `openai_realtime`
- `bailian_dashscope`
- `deepseek`

The registry reports selected provider, capability labels, required env names, present env names, and missing env names. It never prints env values. `A21_PROVIDER_PRIMARY` selects the intended primary provider for readiness reporting. Unknown provider names are redacted to `unknown_provider`; legacy-looking provider names containing X21 or V21 are redacted to `invalid_legacy_provider` and become blocking doctor findings.

## Current Event Mapping

- provider `thinking` -> device `thinking`
- provider `speaking` -> device `speaking`
- provider `cancelled` -> device `interrupted`
- unknown/error event -> device `error`

The gateway still emits the initial `listening` event itself because it represents local device state before provider thinking begins.

## Architecture Rule

Real providers must enter through this boundary or a direct successor of it. A21 should use official or mature provider SDKs/protocol clients where available, wrapped by A21 adapters. Provider SDK types must not leak into protocol, simulator, firmware, or product mode logic.

## Boundaries

Phase 4A does not implement:

- real provider credentials
- Doubao realtime
- OpenAI realtime
- Bailian/DashScope
- streaming audio deltas
- provider latency histograms
- cascade fallback
- real provider connectivity checks
- real provider construction from the registry

Phase 4B adds cascade fallback. See `PHASE4B_PROVIDER_CASCADE.md`.

Phase 4C adds explicit dry-run/execute provider smoke checks for OpenAI-compatible text providers. See `PHASE4C_PROVIDER_SMOKE.md`.
