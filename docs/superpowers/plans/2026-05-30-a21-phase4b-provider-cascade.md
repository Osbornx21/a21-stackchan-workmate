# A21 Phase 4B Provider Cascade Implementation Plan

**Goal:** Add a small provider cascade so real provider adapters can be composed without leaking provider-specific logic into the gateway.

## Scope

Phase 4B implements:

- `CascadeVoiceProvider`
- primary-to-fallback `StartTurn`
- cancel fanout with first successful event stream
- tests for fallback and no-provider failure
- docs for cascade behavior

Phase 4B does not implement:

- real provider adapters
- provider credentials
- retry backoff
- cost routing
- provider health probes

## Tasks

- [x] Add failing cascade tests.
- [x] Implement cascade provider.
- [x] Document cascade boundary.
- [x] Run full verification.
- [x] Commit as `feat: add a21 provider cascade`.
