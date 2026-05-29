# A21 Provider Realtime Plan CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an explicit no-network CLI for realtime provider readiness plans.

**Architecture:** Reuse `providers.RealtimeWebSocketPlanFromEnv` rather than creating a parallel planner. The CLI emits the same redacted JSON shape used by doctor, rejects `--execute`, and returns nonzero only for failed plans or usage errors.

**Tech Stack:** Go, `internal/app`, existing provider plan types, Makefile, Markdown docs.

---

### Task 1: Red Tests

**Files:**
- Modify: `internal/app/app_test.go`

- [x] Add a test for `provider-realtime-plan --provider doubao_tts_realtime` with Doubao TTS env.
- [x] Assert status is `ready`, endpoint host is present, and secrets/model/voice/header values are absent.
- [x] Add a test that legacy provider names are redacted and return exit code 1.
- [x] Add a test that `--execute` is rejected because this command is dry-run only.
- [x] Run targeted tests and verify the command is currently unknown.

### Task 2: CLI Implementation

**Files:**
- Modify: `internal/app/app.go`
- Modify: `Makefile`

- [x] Add `provider-realtime-plan` to `Run`.
- [x] Parse optional `--provider`.
- [x] Reject `--execute` with a usage error.
- [x] Encode `providers.RealtimeWebSocketPlanFromEnv(os.Environ(), provider)`.
- [x] Return exit code 1 only when report status is `failed`.
- [x] Add `make provider-realtime-plan` with optional `A21_PROVIDER`.

### Task 3: Docs and Verification

**Files:**
- Create: `docs/engineering/PHASE4I_PROVIDER_REALTIME_PLAN_CLI.md`
- Modify: `docs/engineering/DOCTOR.md`
- Modify: `docs/engineering/PHASE4H_DOUBAO_REALTIME_TTS_PROVIDER.md`

- [x] Document command usage and no-network behavior.
- [x] Document that this does not replace provider smoke execution.
- [x] Run targeted tests, `make verify`, `make release-check`, and firmware guard checks.
