# A21 Phase 4B Provider Cascade

## Purpose

Phase 4B adds a small provider cascade so A21 can compose primary and fallback voice providers without embedding provider-specific behavior in the gateway.

This is architecture plumbing, not a real provider integration.

## Current Behavior

`CascadeVoiceProvider`:

- calls providers in configured order
- returns the first successful `StartTurn` stream
- returns the first successful `Cancel` stream
- preserves cancellation reason and stream ID through child provider cancel acknowledgements
- reports `healthy` when the first available child provider is healthy
- reports `unavailable` with `ErrVoiceProviderUnavailable` when no child provider is healthy
- returns `ErrVoiceProviderUnavailable` when no provider can handle the request
- closes all child providers and joins close errors

## Why This Exists

A21 will likely need a China-mainland-first provider path plus fallbacks. The cascade boundary lets future adapters for Doubao realtime, OpenAI realtime, Bailian/DashScope, and local sidecars plug in without changing gateway protocol handling.

`internal/providers` also owns the provider HTTP network policy used by future adapters. The current policy has two modes:

- `direct`: default; provider HTTP clients use an explicit transport with no environment proxy function.
- `explicit_proxy`: enabled only by `A21_PROVIDER_PROXY_URL`; provider HTTP clients use that proxy URL and report only the env variable name.

This policy is separate from runtime `NO_PROXY` coverage. It prevents real provider adapters from accidentally inheriting Codex, shell, Dragon Cat Lite, or other ambient proxy settings while still allowing an explicit provider egress proxy when the Shanghai network requires one.

`internal/providers.ProviderCatalogFromEnv` is the current readiness registry. It does not create providers yet. It lets doctor show which primary provider is intended and which env names are still missing before a real adapter can be built or smoke-tested.

`internal/providers.ProviderSmokeFromEnv` is the first smoke boundary. It is intentionally separate from provider construction: doctor reports only a dry-run smoke plan, and `a21 provider-smoke --execute` is required before A21 makes a real provider request.

## Governance

Real provider adapters should use official SDKs or mature protocol clients where available. Provider SDK types must remain behind `internal/providers` and must not leak into:

- gateway device protocol
- simulator UI
- StackChan firmware
- product mode state
- v21 adapter contracts

## Boundaries

Phase 4B does not implement:

- retry backoff
- cost-based routing
- credentials
- real realtime audio
- provider-specific cancellation protocol
- provider-specific HTTP/WebSocket clients
- registry-driven provider construction

Phase 4C adds provider smoke checks. See `PHASE4C_PROVIDER_SMOKE.md`.
