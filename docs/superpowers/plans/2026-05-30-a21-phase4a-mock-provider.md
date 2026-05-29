# A21 Phase 4A Mock Provider Implementation Plan

**Goal:** Move gateway mock voice behavior behind a provider adapter boundary.

## Scope

Phase 4A implements:

- realtime voice provider session/request/event/cancel types
- deterministic `MockVoiceProvider`
- injectable gateway voice provider through `ServerOptions`
- gateway mapping from provider events to A21 control events
- architecture governance reinforcement: mature wheels and official SDKs first

Phase 4A does not implement:

- real provider network calls
- credentials
- realtime audio deltas
- cascade fallback
- provider health checks

## Tasks

- [x] Add failing mock provider event stream tests.
- [x] Implement `VoiceProvider` and `MockVoiceProvider`.
- [x] Add failing gateway provider injection test.
- [x] Connect gateway mock turn and cancel to the provider boundary.
- [x] Update governance and provider docs.
- [x] Run full verification.
- [x] Commit as `feat: add a21 mock voice provider`.
