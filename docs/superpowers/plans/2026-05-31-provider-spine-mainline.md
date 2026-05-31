# Provider Spine Mainline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align A21 implementation with PRD v0.4 by making Provider Spine the next active software mainline while preserving StackChan firmware discipline.

**Architecture:** Keep the current Go-first repository. Add provider profiles and streaming text lanes behind `internal/providers`, keep Gateway mock by default, and route physical StackChan provider use only through existing guarded runtime controls. Firmware is not touched unless a task explicitly targets device acceptance.

**Tech Stack:** Go, `cmd/a21`, `internal/providers`, `internal/gateway`, existing reports under `reports/`, Markdown engineering docs.

**Control note, 2026-05-31:** Task 2 and Task 3 are already present in the
current integration baseline through the accepted
`codex/a21-provider-spine-deepseek-textstream` slice and the
`codex/a21-integration-governance-slices` branch. Do not duplicate parser or
stream-smoke implementation from this plan. Future workers should audit the
existing code first, then continue from Task 4 or a new control-approved slice.

---

### Task 1: Provider Profile Registry

**Files:**
- Modify: `internal/providers/catalog.go`
- Modify: `internal/providers/catalog_test.go`
- Modify: `docs/engineering/A21_DEVELOPMENT_MAINLINE.md`

- [x] **Step 1: Write failing tests for provider families and built-ins**

Add tests asserting profiles exist for `siliconflow`, `deepseek`, `stepfun`, `bailian_dashscope`, `moonshot`, `volcengine_ark`, `local_ollama`, `local_vllm`, `openai_realtime`, `doubao_realtime`, `doubao_tts_realtime`, `hermes_agent`, and `mimo_agent`. Also assert Baidu/Huawei primary names are blocked and redacted.

- [x] **Step 2: Run focused tests**

Run: `go test ./internal/providers -run 'ProviderCatalog|ProviderProfile'`

- [x] **Step 3: Add `ProviderFamily` and `ProviderProfile` fields**

Represent family, protocol, env names, capability labels, route eligibility, and default host without storing any key values.

- [x] **Step 4: Re-run focused tests**

Run: `go test ./internal/providers -run 'ProviderCatalog|ProviderProfile'`

- [x] **Step 5: Commit**

Run: `git commit -m "feat(providers): add provider reference profiles"`

### Task 2: Text Stream Parser

**Files:**
- Present: `internal/providers/textstream.go`
- Present: `internal/providers/textstream_client.go`
- Present: `internal/providers/textstream_test.go`

- [x] **Step 1: Write parser tests**

Cover OpenAI-style `delta.content`, StepFun-style `delta.reasoning`, `[DONE]`, non-2xx redacted error bodies, first-byte and first-content timing fields.

- [x] **Step 2: Run parser tests**

Run: `go test ./internal/providers -run TextStream`

- [x] **Step 3: Implement parser and event types**

Accepted implementation uses `TextStreamEvent`, `TextStreamParseResult`,
`TextStreamCompletionOptions`, and `TextStreamCompletionResult`; the earlier
draft type names `TextStreamRequest`, `TextMessage`, `ProviderUsage`, and
`ProviderTiming` were not adopted because no current caller needs that wider
API surface.

- [x] **Step 4: Re-run tests**

Run: `go test ./internal/providers -run TextStream`

- [x] **Step 5: Commit**

Covered by accepted Provider Spine commits in the current integration baseline;
do not create a duplicate parser commit from this stale task text.

### Task 3: Streaming Provider Smoke

**Files:**
- Modify: `internal/providers/smoke.go`
- Modify: `internal/providers/smoke_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `docs/engineering/PHASE4C_PROVIDER_SMOKE.md`

- [x] **Step 1: Write CLI tests for `--stream --repeat`**

Use `httptest` to simulate streaming chunks. Assert reports include provider, protocol, first_byte_ms, first_content_ms, total_duration_ms, repeat count, p50/p95 summary, and no key/model/prompt/proxy values.

- [x] **Step 2: Run CLI tests**

Run: `go test ./internal/app ./internal/providers -run 'ProviderSmoke|TextStream'`

- [x] **Step 3: Implement stream smoke options**

Add `--stream`, `--repeat`, and redacted timing output without executing any provider unless `--execute` is set.

- [x] **Step 4: Re-run tests**

Run: `go test ./internal/app ./internal/providers -run 'ProviderSmoke|TextStream'`

- [x] **Step 5: Commit**

Covered by accepted Provider Spine commits in the current integration baseline;
do not create a duplicate stream-smoke commit from this stale task text.

### Task 4: Fast Companion Hybrid Boundary

**Control note, 2026-05-31:** Thread
`019e7be6-bca3-71f2-9770-857b9da48b67` audited this task against the current
integration baseline. Existing app-level receipts partially cover the vertical
lane, but the Gateway-level routing boundary, unified Gateway trace waterfall,
and `PHASE7H_FAST_COMPANION_HYBRID.md` are still missing. Continue this task as
a new implementation slice; do not mark it complete from `local-voice-loopback`
or `stackchan-fast-companion-turn` alone.

**Control acceptance, 2026-05-31:** Thread
`019e7bed-4e1e-7512-8f21-1647b2357c00` implemented the Gateway-level Task 4
boundary, then the control tower integrated and tightened it on
`codex/a21-integration-governance-slices`. The accepted route is
`POST /v1/fast-companion/turn`; it is provider-neutral, mock by default, limited
to `companion`/`workmate`, requires an explicit local audio front-end identity,
and does not execute provider, V21, Gateway runtime, firmware, or device paths.

**Files:**
- Modify: `internal/gateway/server.go`
- Modify: `internal/gateway/server_test.go`
- Modify: `docs/engineering/PHASE7A_AUDIO_INGRESS.md`
- Create: `docs/engineering/PHASE7H_FAST_COMPANION_HYBRID.md`

- [x] **Step 1: Write Gateway routing tests**

Assert companion mode can route local audio front-end results to a text stream provider boundary, while professional mode refuses opaque realtime and keeps the V21 path.

- [x] **Step 2: Run Gateway tests**

Run: `go test ./internal/gateway -run 'FastCompanion|Professional|Realtime'`

- [x] **Step 3: Implement only the boundary**

Add trace markers for ASR first partial, provider first byte, provider first content, TTS first audio, downlink first frame, and playback start placeholders. Keep mock default.

- [x] **Step 4: Re-run Gateway tests**

Run: `go test ./internal/gateway -run 'FastCompanion|Professional|Realtime'`

- [x] **Step 5: Commit**

Run: `git commit -m "feat(gateway): add fast companion hybrid boundary"`

### Task 5: Verification

**Control acceptance, 2026-05-31:** The control tower completed Provider Spine
verification after the Fast Companion Hybrid trace-fidelity follow-up at
`22c90ae`. Provider smoke checks were dry-run only and did not execute provider
network calls because the required provider env vars are absent. Firmware was
not touched.

**Files:**
- Modify only docs needed by the previous tasks.

- [x] **Step 1: Run full verification**

Run:

```bash
make verify
go run ./cmd/a21 preflight
go run ./cmd/a21 doctor
go run ./cmd/a21 provider-smoke --provider deepseek
go run ./cmd/a21 provider-smoke --provider bailian_dashscope
```

- [x] **Step 2: Check namespace**

Run: `make namespace-audit`

- [x] **Step 3: Confirm no firmware artifacts changed unless a firmware task ran**

Run: `git status --short`

- [x] **Step 4: Commit verification doc updates**

Run: `git commit -m "docs(control): record provider spine verification"`
