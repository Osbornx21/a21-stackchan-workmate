# A21 Phase 2C Device Simulator Implementation Plan

**Goal:** Serve a built-in browser simulator from the Go gateway so A21 can be exercised without physical StackChan hardware.

## Scope

Phase 2C implements:

- `GET /simulator`
- a single-page simulator UI served by the gateway
- control WebSocket connect/disconnect
- mock turn and interrupt buttons
- mock audio-frame button
- visible face state, mode, trace ID, session ID, and event log

Phase 2C does not implement:

- microphone capture
- audio playback
- React/Vite dev console
- real firmware rendering
- provider or v21 integration

## Tasks

- [ ] Add failing gateway test that `/simulator` returns HTML with A21 simulator markers.
- [ ] Implement `/simulator` using a small embedded HTML string.
- [ ] Verify the page with Go tests.
- [ ] Run the gateway locally and validate the UI through the in-app browser.
- [ ] Document the simulator boundary.
- [ ] Run full verification and commit as `feat: add a21 device simulator`.
