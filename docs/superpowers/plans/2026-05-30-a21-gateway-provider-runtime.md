# A21 Gateway Provider Runtime Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make real provider entry into Gateway runtime explicit, safe, observable, and still default-mock.

**Architecture:** Keep provider construction inside `internal/providers`. Gateway receives a provider object through existing `ServerOptions`. CLI startup chooses the runtime provider through a dedicated env guard instead of letting `A21_PROVIDER_PRIMARY` silently affect Gateway.

**Tech Stack:** Go, existing `VoiceProvider` interface, existing Gateway server injection, existing doctor voice report.

---

### Task 1: Red Tests

**Files:**
- Create: `internal/providers/runtime_factory_test.go`
- Modify: `internal/app/app_test.go`

- [x] Test Gateway provider runtime defaults to mock even when `A21_PROVIDER_PRIMARY` is real.
- [x] Test explicit selected runtime uses the selected provider object.
- [x] Test legacy runtime mode is redacted and unavailable.
- [x] Test Gateway server health remains mock by default.
- [x] Test Gateway server health uses selected provider only with explicit runtime opt-in.
- [x] Test current realtime providers keep ordinary text turns guarded without leaking secrets.
- [x] Test doctor reports selected health separately from Gateway runtime provider.
- [x] Run targeted tests and verify missing runtime factory/reporting fails.

### Task 2: Runtime Factory and CLI Wiring

**Files:**
- Create: `internal/providers/runtime_factory.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/doctor.go`

- [x] Add `NewGatewayVoiceProviderFromEnv(env []string)`.
- [x] Default to `a21-mock-voice`.
- [x] Use `NewVoiceProviderFromEnv` only when `A21_GATEWAY_VOICE_PROVIDER=selected`.
- [x] Redact invalid or legacy runtime modes.
- [x] Start Gateway with the runtime provider factory.
- [x] Add `voice.gateway_provider` to doctor output.

### Task 3: Docs and Verification

**Files:**
- Modify: `docs/engineering/DOCTOR.md`
- Modify: `docs/engineering/PHASE4J_PROVIDER_SELECTED_HEALTH.md`
- Create: `docs/engineering/PHASE4K_GATEWAY_PROVIDER_RUNTIME.md`

- [x] Document `A21_GATEWAY_VOICE_PROVIDER`.
- [x] Document readiness provider versus Gateway runtime provider.
- [x] Document that current realtime wrappers still require explicit realtime session paths.
- [x] Run targeted tests, `make verify`, `make release-check`, and firmware guard checks.
