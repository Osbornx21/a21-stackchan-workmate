# A21 Phase 4B Provider Cascade

## Purpose

Phase 4B adds a small provider cascade so A21 can compose primary and fallback voice providers without embedding provider-specific behavior in the gateway.

This is architecture plumbing, not a real provider integration.

## Current Behavior

`CascadeVoiceProvider`:

- calls providers in configured order
- returns the first successful `StartTurn` stream
- returns the first successful `Cancel` stream
- returns `ErrVoiceProviderUnavailable` when no provider can handle the request
- closes all child providers and joins close errors

## Why This Exists

A21 will likely need a China-mainland-first provider path plus fallbacks. The cascade boundary lets future adapters for Doubao realtime, OpenAI realtime, Bailian/DashScope, and local sidecars plug in without changing gateway protocol handling.

## Governance

Real provider adapters should use official SDKs or mature protocol clients where available. Provider SDK types must remain behind `internal/providers` and must not leak into:

- gateway device protocol
- simulator UI
- StackChan firmware
- product mode state
- v21 adapter contracts

## Boundaries

Phase 4B does not implement:

- provider health probes
- retry backoff
- cost-based routing
- credentials
- real realtime audio
- provider-specific cancellation protocol
