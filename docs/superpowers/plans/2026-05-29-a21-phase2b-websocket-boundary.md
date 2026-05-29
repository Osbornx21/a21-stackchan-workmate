# A21 Phase 2B WebSocket Boundary Implementation Plan

**Goal:** Add the first real-time transport boundary for A21 without connecting real audio providers or firmware.

**Decision:** Use `github.com/coder/websocket` for Go WebSocket transport. It is documented as minimal, idiomatic, context-aware, zero-dependency, and includes JSON helpers. This is a targeted dependency, not a framework adoption.

## Scope

Phase 2B implements:

- protocol payloads for device events and mock audio chunks
- `/ws/control` for semantic device events
- `/ws/audio` for mock audio-frame acceptance
- trace/session propagation over WebSocket messages
- docs/ADR for the WebSocket dependency

Phase 2B does not implement:

- real PCM streaming
- Opus
- browser simulator UI
- provider adapters
- v21 professional mode

## Tasks

- [ ] Add protocol tests for `DeviceEventPayload` and `AudioChunk`.
- [ ] Implement the protocol payload structs and constants.
- [ ] Add failing gateway WebSocket tests for `/ws/control` mock turn and interrupt.
- [ ] Add the WebSocket dependency and implement control WebSocket handling.
- [ ] Add failing gateway WebSocket test for `/ws/audio` mock audio ack.
- [ ] Implement audio WebSocket handling.
- [ ] Add ADR and update Phase 2A/protocol docs with the new boundary.
- [ ] Run full verification and commit as `feat: add a21 websocket transport boundary`.
