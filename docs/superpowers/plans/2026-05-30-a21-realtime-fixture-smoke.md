# A21 Realtime Fixture Smoke Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an explicit offline realtime fixture smoke so A21 can verify provider adapter event flow before real network smoke exists.

**Architecture:** Reuse existing realtime provider wrappers and inject a local fake `RealtimeDialer`. Keep real provider connectivity out of this command.

**Tech Stack:** Go, existing `ProviderSmokeReport`, existing OpenAI/Doubao realtime provider wrappers.

---

### Task 1: Red Tests

**Files:**
- Create: `internal/providers/realtime_fixture_smoke_test.go`
- Modify: `internal/app/app_test.go`

- [x] Test Doubao TTS fixture execution passes without leaking secrets.
- [x] Test OpenAI realtime fixture execution passes without leaking secrets.
- [x] Test fixture smoke requires `--execute` to run the fake connection.
- [x] Test legacy provider names are redacted.
- [x] Test CLI command output and exit codes.
- [x] Run targeted tests and verify missing command/function fails.

### Task 2: Provider Fixture Smoke

**Files:**
- Create: `internal/providers/realtime_fixture_smoke.go`
- Modify: `internal/app/app.go`

- [x] Add `RealtimeFixtureSmokeFromEnv`.
- [x] Reuse `RealtimeWebSocketPlanFromEnv` for readiness and redacted endpoint host reporting.
- [x] Execute OpenAI realtime wrapper with a fake dialer.
- [x] Execute Doubao realtime TTS wrapper with a fake dialer.
- [x] Preserve no-network behavior.
- [x] Add `provider-realtime-fixture` CLI command.

### Task 3: Docs and Verification

**Files:**
- Modify: `Makefile`
- Create: `docs/engineering/PHASE4L_REALTIME_FIXTURE_SMOKE.md`

- [x] Add `make provider-realtime-fixture`.
- [x] Document what fixture smoke proves and does not prove.
- [x] Run targeted tests, `make verify`, `make release-check`, and firmware guard checks.
