# A21 Provider Selected Health Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `doctor` voice health reflect `A21_PROVIDER_PRIMARY` instead of always reporting the mock provider.

**Architecture:** Add a provider factory in `internal/providers` that returns a configured local provider object without dialing external services. Use it only in doctor health probing for now; Gateway defaults remain mock until a later explicit runtime switch is designed.

**Tech Stack:** Go, `internal/providers`, `internal/app`, existing provider adapters.

---

### Task 1: Red Tests

**Files:**
- Create: `internal/providers/factory_test.go`
- Modify: `internal/app/app_test.go`

- [x] Test default provider factory returns mock.
- [x] Test Doubao realtime TTS selected provider health is healthy and redacted.
- [x] Test OpenAI realtime selected provider health is healthy.
- [x] Test legacy provider primary becomes unavailable without echoing the raw value.
- [x] Test doctor voice health follows Doubao TTS instead of always showing mock.
- [x] Run targeted tests and verify missing factory / mock-only doctor behavior.

### Task 2: Provider Factory

**Files:**
- Create: `internal/providers/factory.go`

- [x] Add `NewVoiceProviderFromEnv(env []string) VoiceProvider`.
- [x] Return mock for empty or `mock`.
- [x] Return OpenAI realtime provider for `openai_realtime`.
- [x] Return Doubao realtime TTS provider for `doubao_tts_realtime`.
- [x] Return redacted unavailable provider for legacy, unknown, and not-yet-wrapped providers.

### Task 3: Doctor Integration and Docs

**Files:**
- Modify: `internal/app/doctor.go`
- Modify: `docs/engineering/DOCTOR.md`
- Create: `docs/engineering/PHASE4J_PROVIDER_SELECTED_HEALTH.md`

- [x] Make the default doctor health probe use `NewVoiceProviderFromEnv(os.Environ())`.
- [x] Document that doctor selected health is no-network and does not switch Gateway runtime.
- [x] Run targeted tests, `make verify`, `make release-check`, and firmware guard checks.
