# Provider Spine Mainline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align A21 implementation with PRD v0.4 by making Provider Spine the next active software mainline while preserving StackChan firmware discipline.

**Architecture:** Keep the current Go-first repository. Add provider profiles and streaming text lanes behind `internal/providers`, keep Gateway mock by default, and route physical StackChan provider use only through existing guarded runtime controls. Firmware is not touched unless a task explicitly targets device acceptance.

**Tech Stack:** Go, `cmd/a21`, `internal/providers`, `internal/gateway`, existing reports under `reports/`, Markdown engineering docs.

---

### Task 1: Provider Profile Registry

**Files:**
- Modify: `internal/providers/catalog.go`
- Modify: `internal/providers/catalog_test.go`
- Modify: `docs/engineering/A21_DEVELOPMENT_MAINLINE.md`

- [ ] **Step 1: Write failing tests for provider families and built-ins**

Add tests asserting profiles exist for `siliconflow`, `deepseek`, `stepfun`, `bailian_dashscope`, `moonshot`, `volcengine_ark`, `local_ollama`, `local_vllm`, `openai_realtime`, `doubao_realtime`, `doubao_tts_realtime`, `hermes_agent`, and `mimo_agent`. Also assert Baidu/Huawei primary names are blocked and redacted.

- [ ] **Step 2: Run focused tests**

Run: `go test ./internal/providers -run 'ProviderCatalog|ProviderProfile'`

- [ ] **Step 3: Add `ProviderFamily` and `ProviderProfile` fields**

Represent family, protocol, env names, capability labels, route eligibility, and default host without storing any key values.

- [ ] **Step 4: Re-run focused tests**

Run: `go test ./internal/providers -run 'ProviderCatalog|ProviderProfile'`

- [ ] **Step 5: Commit**

Run: `git commit -m "feat: add provider spine profile registry"`

### Task 2: Text Stream Parser

**Files:**
- Create: `internal/providers/textstream.go`
- Create: `internal/providers/textstream_test.go`

- [ ] **Step 1: Write parser tests**

Cover OpenAI-style `delta.content`, StepFun-style `delta.reasoning`, `[DONE]`, non-2xx redacted error bodies, first-byte and first-content timing fields.

- [ ] **Step 2: Run parser tests**

Run: `go test ./internal/providers -run TextStream`

- [ ] **Step 3: Implement parser and event types**

Add `TextStreamRequest`, `TextMessage`, `TextStreamEvent`, `ProviderUsage`, and `ProviderTiming` with redaction-safe errors.

- [ ] **Step 4: Re-run tests**

Run: `go test ./internal/providers -run TextStream`

- [ ] **Step 5: Commit**

Run: `git commit -m "feat: add text stream provider parser"`

### Task 3: Streaming Provider Smoke

**Files:**
- Modify: `internal/providers/smoke.go`
- Modify: `internal/providers/smoke_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `docs/engineering/PHASE4C_PROVIDER_SMOKE.md`

- [ ] **Step 1: Write CLI tests for `--stream --repeat`**

Use `httptest` to simulate streaming chunks. Assert reports include provider, protocol, first_byte_ms, first_content_ms, total_duration_ms, repeat count, p50/p95 summary, and no key/model/prompt/proxy values.

- [ ] **Step 2: Run CLI tests**

Run: `go test ./internal/app ./internal/providers -run 'ProviderSmoke|TextStream'`

- [ ] **Step 3: Implement stream smoke options**

Add `--stream`, `--repeat`, and redacted timing output without executing any provider unless `--execute` is set.

- [ ] **Step 4: Re-run tests**

Run: `go test ./internal/app ./internal/providers -run 'ProviderSmoke|TextStream'`

- [ ] **Step 5: Commit**

Run: `git commit -m "feat: add streaming provider smoke timing"`

### Task 4: Fast Companion Hybrid Boundary

**Files:**
- Modify: `internal/gateway/server.go`
- Modify: `internal/gateway/server_test.go`
- Modify: `docs/engineering/PHASE7A_AUDIO_INGRESS.md`
- Create: `docs/engineering/PHASE7H_FAST_COMPANION_HYBRID.md`

- [ ] **Step 1: Write Gateway routing tests**

Assert companion mode can route local audio front-end results to a text stream provider boundary, while professional mode refuses opaque realtime and keeps the V21 path.

- [ ] **Step 2: Run Gateway tests**

Run: `go test ./internal/gateway -run 'FastCompanion|Professional|Realtime'`

- [ ] **Step 3: Implement only the boundary**

Add trace markers for ASR first partial, provider first byte, provider first content, TTS first audio, downlink first frame, and playback start placeholders. Keep mock default.

- [ ] **Step 4: Re-run Gateway tests**

Run: `go test ./internal/gateway -run 'FastCompanion|Professional|Realtime'`

- [ ] **Step 5: Commit**

Run: `git commit -m "feat: add fast companion hybrid routing boundary"`

### Task 5: Verification

**Files:**
- Modify only docs needed by the previous tasks.

- [ ] **Step 1: Run full verification**

Run:

```bash
make verify
go run ./cmd/a21 preflight
go run ./cmd/a21 doctor
go run ./cmd/a21 provider-smoke --provider deepseek
go run ./cmd/a21 provider-smoke --provider bailian_dashscope
```

- [ ] **Step 2: Check namespace**

Run: `make namespace-audit`

- [ ] **Step 3: Confirm no firmware artifacts changed unless a firmware task ran**

Run: `git status --short`

- [ ] **Step 4: Commit verification doc updates**

Run: `git commit -m "docs: record provider spine verification"`
