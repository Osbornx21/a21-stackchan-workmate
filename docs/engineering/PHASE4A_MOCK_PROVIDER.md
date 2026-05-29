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
- `VoiceProvider`
- `MockVoiceProvider`

The default gateway uses `MockVoiceProvider`. Tests can inject a scripted provider through `NewServerWithOptions`.

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
- provider health checks
- provider latency histograms
- cascade fallback
