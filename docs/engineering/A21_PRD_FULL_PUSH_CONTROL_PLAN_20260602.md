# A21 PRD Full Push Control Plan

Status: active control-tower plan  
Date: 2026-06-02  
Owner: A21 control tower  
Base branch: `codex/a21-integration-runtime-readiness-20260601`  
Current baseline evidence: `ef8bc2b fix(readiness): accept host local xiaozhi product chain`

## 0. Control Rule

This plan exists to prevent context compression, duplicated subthreads, and giant monolithic coding sessions from derailing A21. The target is still full PRD launch, not demo acceptance and not candidate-only green.

Control-tower rules:

- Main thread owns burn-down, branch routing, worktree ownership, merges, and launch acceptance.
- Implementation runs in bounded worker threads or worktrees by feature slice.
- Read-only review threads stay read-only and must not become implementation lanes.
- A worker gets one clear write set, one acceptance command set, and one handoff.
- Subthreads use the default inherited model/settings unless the user explicitly overrides them; do not invent custom model settings.
- Provider execution belongs on 5080lab unless the user explicitly approves Mac execution.
- The Mac must not play audio; audio output evidence should use simulator files, redacted reports, or user/StackChan observation.
- Lack of CoreS3 only blocks physical acceptance, not server/provider/protocol/simulator implementation.
- No fake green: candidate/host-only evidence must never be promoted to PRD accepted by missing fields or zero defaults.

## 1. Current PRD Burn-Down Baseline

| Slice | Current status | Current evidence | Full-launch gap |
| --- | --- | --- | --- |
| Phase 0 Go-first foundation | Done | CLI, host gate, doctor, provider default mock, namespace guard | Keep green while integrating |
| Provider spine and text stream | Partial | Built-in profiles, text stream parser, smoke/repeat/redaction | Real selected provider execute reports and route eligibility closure |
| Fast companion hybrid lane | Partial | Xiaozhi product chain, ASR/Text/TTS adapters, Opus downlink, pacing, turn cancel | Real provider plus local ASR/TTS chain, three accepted rounds, physical playback evidence |
| Realtime voice lane | Partial | Provider-neutral realtime fixture, explicit arm concept, professional boundary | Live provider lane remains opt-in and must not enter professional mode |
| V21 professional mode | Contract done, launch partial | V21 adapter contract, checking feedback, evidence/cards/follow-ups | Executed adapter evidence and professional bench in launch rollup |
| Agent task bridge | Contract done | AgentTask interface, Hermes/MiMo profiles, safety mapper | Keep out of first-audio path; later UX polish only |
| Physical StackChan | Partial | Capability charts, evidence commands, playback ack/debug profile, firmware guards | CoreS3 physical audio/mic/touch/screen/servo/RGB/wake word acceptance |
| Personality and playbooks | Missing asset tree | PRD section 8 and scattered copy strings | Canonical `docs/personality` tree and eventual runtime composition |

Important correction: `8bfed60 fix(audio): improve xiaozhi tts downlink clarity` is already merged into the current baseline. The previous audio-risk review is not a live gap. Treat it as a regression guard only.

## 2. Branch And Worktree Strategy

Use the current integration branch only for reviewed merges and acceptance reports:

- `codex/a21-integration-runtime-readiness-20260601`: integration candidate and launch rollup.
- `codex/a21-control-prd-full-push-plan-20260602`: this control plan only.
- `codex/a21-mainline-voice-product-chain-*`: server-side voice chain implementation.
- `codex/a21-provider-selected-*`: provider execute/report/hot-plug closure.
- `codex/a21-mainline-professional-v21-*`: V21 professional closure.
- `codex/a21-mainline-personality-*`: docs/personality assets and prompt-composition boundary.
- `codex/a21-mainline-wake-word-*`: custom wake-word config/build planning and guarded firmware prep.
- `codex/a21-firmware-stackchan-*`: build/package/probe work without physical writes.
- `codex/a21-hardware-window-YYYYMMDD-*`: foreground-only CoreS3 physical acceptance.

Worktree rule:

- One write-capable worker per branch.
- Disjoint write sets between concurrent workers.
- Control tower may spawn read-only explorers for independent questions, but their output must be reconciled against current branch state before entering the plan.
- If a worker returns stale findings that contradict merged commits, the control tower marks them obsolete instead of rediscovering old work.

## 3. Execution Slices

### Slice A: Launch Rollup Aggregator

Goal: make one canonical report decide full PRD readiness without duplicated candidate paperwork.

Owner branch: `codex/a21-mainline-launch-rollup-20260602`  
Write set:

- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- related tests in `internal/app/*test.go`
- docs updates under `docs/engineering/OBSERVABILITY.md` only if fields change

Acceptance:

- `go test ./internal/app -run 'TestRunProductReadiness|TestRunServerSideReadinessBundle|TestProductReadiness'`
- `go run ./cmd/a21 server-side-readiness-bundle --output-dir reports`
- `go run ./cmd/a21 product-readiness --use-latest-reports --output-dir reports`
- `go run ./cmd/a21 product-readiness --use-latest-reports --require-real --output-dir reports` must fail honestly until real evidence exists

Done means:

- Provider, V21, voice, physical, wake-word, audio-quality, and redaction statuses compose into one machine-readable burn-down.
- Host-only evidence can reduce a gap but cannot set PRD accepted.
- Missing report fields are `missing`, not false success.

### Slice B: Selected Provider Closure

Goal: close hot-plug provider execution for at least one route-eligible text provider without Gateway business-logic edits.

Owner branch: `codex/a21-provider-selected-mainland-20260602`  
Write set:

- `internal/providers/*`
- `internal/app/provider_latency_bench.go`
- provider docs and redaction tests

Acceptance:

- `go test ./internal/providers ./internal/app`
- `go run ./cmd/a21 provider-smoke --provider <selected> --stream --repeat 3 --output-dir reports`
- On 5080lab only: `go run ./cmd/a21 provider-smoke --provider <selected> --execute --stream --repeat 3 --output-dir reports`
- On 5080lab only: provider latency report with p50/p95/p99 and redaction pass

Done means:

- One selected provider has real execute evidence.
- Fallback is recorded as fallback, not primary success.
- Reports contain env names and host/status/timing only; no key, prompt, provider output, model value, proxy URL, or full URL.

### Slice C: Continuous Voice Product Chain

Goal: server-side `OPUS -> PCM -> ASR -> text stream -> TTS -> OPUS paced downlink` becomes one continuous product path with clean abort.

Owner branch: `codex/a21-mainline-voice-product-chain-20260602`  
Write set:

- `internal/gateway/server.go`
- `internal/providers/voice_pipeline*.go`
- `internal/audio/*`
- `internal/app/xiaozhi_voice_bench.go`
- related tests

Acceptance:

- `go test ./internal/gateway ./internal/providers ./internal/audio ./internal/app`
- `go run ./cmd/a21 xiaozhi-voice-bench --repeat 3 --output-dir reports`
- `go run ./cmd/a21 product-readiness --use-latest-reports --output-dir reports`

Done means:

- Three host/simulator rounds report answer first-audio p95 under 1500ms.
- Barge-in stop/cancel remains under 300ms in host evidence.
- `8bfed60` audio clarity behavior remains covered by regression tests.
- No Mac speaker playback is used for validation.

### Slice D: V21 Professional Closure

Goal: professional mode is launch-grade on contract, timing, evidence/cards, and fallback behavior.

Owner branch: `codex/a21-mainline-professional-v21-20260602`  
Write set:

- `internal/v21adapter/*`
- `internal/gateway/server.go`
- `internal/app/xiaozhi_professional_bench.go`
- `internal/app/professional_adapter_bridge.go`
- tests and V21 docs

Acceptance:

- `go test ./internal/v21adapter ./internal/gateway ./internal/app`
- `go run ./cmd/a21 xiaozhi-professional-bench --output-dir reports`
- With adapter configured: `go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports`

Done means:

- 1200ms checking feedback is present in trace and report.
- Final response preserves `fast_answer`, `confidence`, `evidence`, `speech_blocks`, `screen_cards`, and `follow_ups`.
- Public mode does not emit sensitive evidence content.

### Slice E: Personality Assets

Goal: satisfy PRD section 8 without one giant prompt.

Owner branch: `codex/a21-mainline-personality-assets-20260602`  
Write set:

- `docs/personality/core_identity.md`
- `docs/personality/tone_rules.md`
- `docs/personality/mode_prompts/*.md`
- `docs/personality/scenario_playbooks/*.md`
- optional tests only if runtime loader is introduced

Acceptance:

- `git diff --check`
- `make verify` if Go/docs checks require it

Done means:

- Core identity, tone rules, nine mode prompts, and seven scenario playbooks exist.
- Runtime composition rule is documented: core + tone + one mode + optional one scenario + failure overlay.
- No prompt-bloat loader is introduced unless a later slice explicitly needs it.

### Slice F: Custom Wake Word

Goal: front-end/simple Gateway config is already started; finish the guarded build/firmware path without pretending runtime hot-swap is possible.

Owner branch: `codex/a21-mainline-wake-word-20260602`  
Write set:

- `internal/gateway/wake_word.go`
- simulator UI in `internal/gateway/simulator.go`
- `internal/app/wake_word_firmware.go`
- firmware build docs/tests only

Acceptance:

- `go test ./internal/gateway ./internal/app -run 'WakeWord|Simulator'`
- `go run ./cmd/a21 gate --scope host`
- No hardware write command in this slice

Done means:

- Front-end config persists desired phrase/pinyin/threshold.
- Gateway reports built-in active vs custom pending firmware honestly.
- Build/flash remains guarded and separate.

### Slice G: Physical StackChan Acceptance

Goal: when CoreS3 returns, collect real evidence in one foreground hardware window.

Owner branch: `codex/a21-hardware-window-YYYYMMDD-stackchan-prd`  
Write set:

- Reports only unless a pre-approved firmware/hardware fix is required

Acceptance:

- `go run ./cmd/a21 gate --scope hardware --command "<exact command>" --tier T7`
- Official audio/PCM bridge proof if needed
- Mic uplink, speaker playback, touch, screen, servo, RGB, playback ack, stop_done
- `go run ./cmd/a21 product-readiness --use-latest-reports --require-real --output-dir reports`

Done means:

- Three consecutive physical rounds meet first-audio p95 under 1500ms.
- Barge-in playback/mouth/speaking stop p95 under 300ms.
- Device ends idle/safe.
- No firmware/provider key leak.

## 4. Immediate Dispatch Queue

Run in this order:

1. Control tower lands this plan and keeps the integration branch stable.
2. Worker 1 implements Slice A launch rollup aggregator.
3. Worker 2 implements Slice E personality assets in parallel because it is docs-only and disjoint.
4. Worker 3 audits Slice C voice-chain tests specifically to ensure `8bfed60` cannot regress.
5. After Slice A lands, dispatch Slice B to 5080lab for execute evidence.
6. After Slice A lands, dispatch Slice D for V21 professional execution evidence.
7. Keep Slice F ready for no-hardware implementation; do not flash until hardware window.
8. When CoreS3 returns, open exactly one Slice G foreground hardware window.

## 5. Reporting Template

Every control-tower report to the user must answer:

- 本轮完成了哪个功能模块或组合切片。
- 改了哪些文件或合并了哪个分支。
- 怎么验收，命令和结果是什么。
- PRD 燃尽减少了哪一项。
- 下一步推进哪个切片。
- 仍未完成的只列真实剩余项，不重复列已经完成或已合入的旧风险。

## 6. Current Next Move

The next implementation move is Slice A. It is the highest-leverage server-side change because it removes duplicated candidate gates while preserving full PRD acceptance. Slice E can run in parallel as a low-risk docs/product asset lane. Voice quality is not a new implementation gap; it stays as a regression check.
