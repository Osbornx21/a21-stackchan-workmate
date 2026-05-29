# A21 Doubao Realtime TTS Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a safe Doubao realtime TTS provider boundary without confusing it with Doubao end-to-end speech-to-speech.

**Architecture:** Reuse the existing realtime WebSocket adapter and provider catalog. Add a TTS-specific event mapper and redacted dry-run plan for `doubao_tts_realtime`; do not add real network execution or Gateway traffic switching.

**Tech Stack:** Go, `internal/providers`, fake WebSocket connections, Markdown docs.

---

### Task 1: Red Tests

**Files:**
- Modify: `internal/providers/realtime_test.go`
- Modify: `internal/providers/catalog_test.go`
- Create: `internal/providers/doubao_realtime_tts_test.go`

- [x] Write failing tests for Doubao TTS realtime plan readiness and missing voice env.
- [x] Write failing tests for provider catalog readiness and secret redaction.
- [x] Write failing tests for TTS session/text/audio event mapping and fake-session writes.
- [x] Run targeted tests and observe missing symbols / unsupported provider.

### Task 2: Minimal Implementation

**Files:**
- Create: `internal/providers/doubao_realtime_tts.go`
- Modify: `internal/providers/realtime.go`
- Modify: `internal/providers/catalog.go`
- Modify: `internal/providers/smoke.go`

- [x] Add `doubao_tts_realtime` catalog entry.
- [x] Add redacted Doubao TTS realtime plan support.
- [x] Add event mapping for `tts_session.update`, `input_text.append`, `input_text.done`, and `response.audio.delta`.
- [x] Keep provider-smoke execution unsupported for realtime WebSocket providers.
- [x] Run targeted provider tests.

### Task 3: Docs and Verification

**Files:**
- Create: `docs/engineering/PHASE4G_DOUBAO_REALTIME_TTS.md`
- Modify: `docs/engineering/DOCTOR.md`
- Modify: `docs/engineering/PHASE4F_OPENAI_REALTIME_PROVIDER_WRAPPER.md`

- [x] Document that this is TTS-only, not S2S.
- [x] Document required env names and no-network safety.
- [ ] Run full verification, release-check, firmware guard checks, then commit.
