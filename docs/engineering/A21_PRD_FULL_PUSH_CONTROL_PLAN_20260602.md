# A21 PRD Full Push Control Plan

Status: active control-tower plan  
Date: 2026-06-02  
Owner: A21 control tower  
Base branch: `codex/a21-integration-runtime-readiness-20260601`  
Current baseline evidence: `f288741 feat(wake-word): add build receipt command`

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
- Local verification that touches doctor/network checks must preserve A21 direct
  routing for localhost/LAN/private ranges. If `make verify` fails on
  `proxy_direct_bypass_missing`, rerun with the documented A21 `NO_PROXY`
  ranges; do not weaken doctor or tests to hide a proxy misconfiguration.

## 0.1 Current Control Checkpoint

This section is the compression-safe handoff point. Update it before launching new
implementation waves.

Current integration branch:

- Branch: `codex/a21-integration-runtime-readiness-20260601`
- HEAD: `f288741 feat(wake-word): add build receipt command`
- Main worktree dirty state: only untracked `tools/__pycache__/`
- Current `product-readiness --use-latest-reports`: `status=mock_demo_ready`,
  `launch_ready=false`, `demo_ready=true`,
  canonical missing real evidence is `real_provider_smoke`,
  `physical_stackchan_online`, `physical_stackchan_prd_acceptance`,
  `wake_word_product_ready`
- Latest full verification:
  `env NO_PROXY='localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16' no_proxy='localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16' make verify`
  passed after the wake build receipt merge.
- Latest targeted verification:
  `go test ./internal/app -run 'WakeWord|FirmwareBuildReceipt|FirmwarePackage|ProductReadiness' -count=1`
  passed.
- Latest no-hardware CLI smoke:
  `wake-word-firmware-build-receipt` produced
  `a21.wake_word_firmware_build.v1` with `status=built`, and
  `wake-word-firmware-package --build-receipt` consumed it and returned
  `status=packaged`, `product_ready=false`, with the expected
  `wake_word_firmware_package_below_activation` finding.

Already merged into the current baseline and must not be rediscovered as active
gaps:

- Audio clarity: `8bfed60 fix(audio): improve xiaozhi tts downlink clarity` and
  `960db85 test(audio): guard xiaozhi downlink clarity`
- Canonical launch rollup: `9431423 feat(readiness): add canonical launch rollup`
- Personality assets: `425c009 docs(personality): add A21 personality assets`
- Host voice/V21 evidence surfaces: `3909204 feat(readiness): close host voice and V21 evidence gates`
- Provider p99/evidence import: `3a8f526 feat(provider): add 5080lab selected provider evidence package`,
  `bfb56a6 feat(provider): import 5080lab evidence bundles`,
  `401242b feat(provider): package evidence bundles`
- Wake-word UI/config/pending status and firmware package command:
  `f74bd6e feat(wake-word): expose custom pending firmware status`,
  `c3637ff feat(wake-word): package custom firmware artifact`,
  `9a9a68c feat(app): ingest wake word firmware package readiness`
- Mode/privacy red line closure:
  `2d102c4 fix(agentplan): block private professional routing`,
  `61d121d docs(protocol): clarify private mode routing red line`
- Realtime evidence visibility closure:
  `66798f8 feat(provider): expose realtime fixture evidence`,
  `9526976 docs(provider): document realtime fixture evidence`
- Wake-word no-hardware diagnostic closure:
  `2f01049 feat(wake-word): report package input diagnostics`
- Provider selected-evidence closure:
  `f57cbda feat(provider): enforce selected smoke evidence`
- Wake-word build receipt closure:
  `f288741 feat(wake-word): add build receipt command`

Active workers that the control tower must poll before duplicating work:

| Worker | Thread | Worktree | Branch | Owned slice | Current status |
| --- | --- | --- | --- | --- | --- |
| None | - | - | - | - | No active implementation worker at this checkpoint |

Recently completed workers:

| Worker | Thread | Worktree | Branch | Owned slice | Current status |
| --- | --- | --- | --- | --- | --- |
| Realtime evidence closure | `019e851c-1c6b-7d53-8437-6cbe4b57692c` | `/Users/jiyurun/.codex/worktrees/5434/New project` | `codex/a21-realtime-evidence-closure-20260602` | Provider realtime fixture report/output-dir and product-readiness visibility | Merged via `66798f8` and `9526976`; do not duplicate |
| Mode/privacy closure | `019e851d-1917-75c1-b4cb-f4a26c7851ca` | `/Users/jiyurun/.codex/worktrees/7232/New project` | `codex/a21-mode-privacy-closure-20260602` | Public/private/focus/professional mode red lines and visible state | Merged via `2d102c4` and `61d121d`; do not duplicate |
| Wake-word no-hardware diagnostic closure | `019e8528-fd53-77b3-a17e-676eed19ed9e` | `/Users/jiyurun/.codex/worktrees/7a3c/New project` | `codex/a21-wake-word-package-readiness-20260602` | Missing build-dir/receipt diagnostic package reports and product-readiness next-action guard | Merged via `2f01049`; do not duplicate |
| Provider selected-evidence closure | `019e8529-78c3-7fa3-9a23-a776180e5009` | `/Users/jiyurun/.codex/worktrees/fc26/New project` | `codex/a21-provider-5080lab-evidence-closure-20260602` | Selected-provider matching for latest smoke evidence, package/import, and 5080lab repeat-3 runbook | Merged via `f57cbda`; do not duplicate |
| Wake build receipt closure | `019e8537-d0bf-71b1-b72a-b5e3cf90c793` | `/Users/jiyurun/.codex/worktrees/c7e2/New project` | `codex/a21-wake-build-receipt-20260602` | Build receipt command and explicit receipt-to-package bridge for reviewed xiaozhi/ESP-SR MultiNet output | Merged via `f288741`; do not duplicate |
| No-hardware PRD gap audit | `019e8537-d17e-7320-a13d-6a122cab9dd0` | `/Users/jiyurun/.codex/worktrees/efbe/New project` | detached at `9939bda` | Read-only audit of current no-hardware gaps | Completed read-only; no merge needed |

Current canonical PRD gaps from the latest product-readiness run:

- `real_provider_smoke`
- `physical_stackchan_online`
- `physical_stackchan_prd_acceptance`
- `wake_word_product_ready`

Current server-side gaps from the latest product-readiness run:

- `provider_smoke`
- `wake_word`

Current next moves:

1. Provider execute closure for 5080lab: prepare/run/import a selected real
   route-eligible text provider smoke bundle from 5080lab, then rerun
   product-readiness to reduce `real_provider_smoke`.
2. Wake-word package closure: run `wake-word-firmware-build-receipt` against a
   reviewed xiaozhi/ESP-SR MultiNet build directory, then
   `wake-word-firmware-package --build-receipt` to produce the current matching
   package report. This may close `firmware_package_available`, but
   `product_ready` must remain false until guarded flash and physical wake proof.
3. Hardware window when CoreS3 returns: collect physical online/audio/playback
   stop/custom wake evidence only after the server/provider/wake package seams
   are ready.

## 1. Current PRD Burn-Down Baseline

| Slice | Current status | Current evidence | Full-launch gap |
| --- | --- | --- | --- |
| Phase 0 Go-first foundation | Done | CLI, host gate, doctor, provider default mock, namespace guard | Keep green while integrating |
| Provider spine and text stream | Server contract done for selected evidence intake, real evidence pending | Built-in profiles, text stream parser, smoke/repeat/redaction, p99, 5080lab runbook, evidence package/import, selected-provider mismatch rejection | 5080lab real non-mock executed smoke bundle |
| Fast companion hybrid lane | Host/simulator chain done, physical pending | Xiaozhi product chain, ASR/Text/TTS adapters, Opus downlink, pacing, turn cancel, audio-clarity regression, host p95 evidence | Real provider evidence plus physical StackChan audible playback and barge-in acceptance |
| Realtime voice lane | Offline evidence/reporting done; live provider still pending | Provider-neutral fixture, explicit arm concept, `provider-realtime-fixture --output-dir`, product-readiness `provider.realtime_*` fields | Live provider lane smoke and physical playback acceptance |
| V21 professional mode | No-hardware evidence ready, launch physical/user acceptance pending | V21 adapter contract, checking feedback, evidence/cards/follow-ups, `v21_professional_execution` rollup | Keep real adapter evidence current; physical/public-mode acceptance still required |
| Agent task bridge | Contract done | AgentTask interface, Hermes/MiMo profiles, safety mapper | Keep out of first-audio path; later UX polish only |
| Physical StackChan | Partial | Capability charts, evidence commands, playback ack/debug profile, firmware guards, candidate downlink reports | CoreS3 physical online, mic/audio/playback stop, touch/screen/servo/RGB/wake word acceptance |
| Wake word | No-hardware config/plan/build-receipt/diagnostic package path done; current reviewed build package/product-ready still pending | Frontend/Gateway desired phrase persistence, pending firmware status, guarded plan/package commands, package report ingestion, missing build-dir/receipt diagnostic reports, explicit build receipt bridge | Current matching reviewed build package report, guarded flash, and physical custom wake proof |
| Personality and playbooks | Asset tree done | `docs/personality` assets merged | Runtime composition and scenario UX wiring only when needed |

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

1. Produce or import a real 5080lab provider evidence bundle. The Mac control
   tower can package/import returned redacted reports, but executed provider
   smoke belongs on 5080lab.
2. Dispatch a new no-hardware realtime voice lane worker. Scope: evidence/report
   closure for the already-present fixture/live boundary, one-shot arm proof,
   professional-mode rejection, and product-readiness visibility; no provider
   execute on Mac.
3. Dispatch a new no-hardware AgentTask bridge worker after realtime evidence
   scope is stable. Scope: disabled-by-default smoke, progress/result/error
   semantics, no first-audio-path integration.
4. Dispatch a public/private/focus and professional-output privacy worker for
   UX safety once runtime mode surfaces are stable. Scope: copy/state/tests, not
   provider execution.
5. When CoreS3 returns, open exactly one Slice G foreground hardware window for
   physical online, microphone, speaker, barge-in stop, screen/touch/servo/RGB,
   and custom wake-word acceptance.

## 5. Reporting Template

Every control-tower report to the user must answer:

- 本轮完成了哪个功能模块或组合切片。
- 改了哪些文件或合并了哪个分支。
- 怎么验收，命令和结果是什么。
- PRD 燃尽减少了哪一项。
- 下一步推进哪个切片。
- 仍未完成的只列真实剩余项，不重复列已经完成或已合入的旧风险。

## 6. Current Next Move

The current next implementation move is controlled realtime voice evidence
closure in a fresh worktree. Do not reimplement realtime transport from scratch:
the repo already has provider-neutral session tests, one-shot physical arming,
unarmed physical suppression, professional-mode rejection, and provider audio
uplink/downlink mapping. The next slice should make those surfaces visible in
the same product-readiness/reporting language used by provider, V21, wake-word,
and physical evidence.

Voice clarity is not an open implementation gap; it is a regression check.
