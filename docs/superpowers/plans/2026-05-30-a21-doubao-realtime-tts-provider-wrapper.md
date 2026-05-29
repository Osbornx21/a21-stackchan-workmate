# A21 Doubao Realtime TTS Provider Wrapper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wrap the Doubao realtime TTS event mapper in a health-checkable provider boundary with explicit session entry only.

**Architecture:** Use the existing `RealtimeDialer` test seam but do not reuse OpenAI-style `RealtimeWebSocketAdapter.Connect`, because Doubao TTS initializes with `tts_session.update`. The wrapper keeps `StartTurn` disabled to prevent accidental provider calls from Gateway text paths.

**Tech Stack:** Go, `internal/providers`, fake WebSocket connections.

---

### Task 1: Red Tests

**Files:**
- Create: `internal/providers/doubao_realtime_tts_provider_test.go`

- [x] Write failing health redaction test.
- [x] Write failing missing env test.
- [x] Write failing explicit session event-order test.
- [x] Write failing `StartTurn` no-dial test.
- [x] Run targeted tests and observe missing provider symbols.

### Task 2: Provider Wrapper

**Files:**
- Create: `internal/providers/doubao_realtime_tts_provider.go`

- [x] Add `DoubaoRealtimeTTSProviderConfig`.
- [x] Add env constructor and health method.
- [x] Add explicit `StartRealtimeTTSSession` using `RealtimeDialer`.
- [x] Send `tts_session.update` as the first provider event.
- [x] Keep `StartTurn` disabled.
- [x] Run targeted provider tests.

### Task 3: Docs and Verification

**Files:**
- Create: `docs/engineering/PHASE4H_DOUBAO_REALTIME_TTS_PROVIDER.md`
- Modify: `docs/engineering/PHASE4G_DOUBAO_REALTIME_TTS.md`

- [x] Document explicit-session-only safety.
- [x] Document TTS-only scope and no Gateway switch.
- [ ] Run full verification, release-check, firmware guard checks, then commit.
