# A21 PRD Full Push Control Plan

Status: active control-tower plan  
Date: 2026-06-02  
Owner: A21 control tower  
Base branch: `codex/a21-integration-runtime-readiness-20260601`  
Current integration checkpoint: `3f0162d chore(control): record v21 adapter evidence refresh`
Current post-worker checkpoint: this document revision

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
- HEAD before this post-worker checkpoint:
  `3f0162d chore(control): record v21 adapter evidence refresh`
- Main worktree dirty state: only untracked `tools/__pycache__/`
- Current `product-readiness --use-latest-reports`: `status=mock_demo_ready`,
  `launch_ready=false`, `demo_ready=true`,
  canonical missing real evidence is `real_provider_smoke`,
  `physical_stackchan_online`, `physical_stackchan_prd_acceptance`,
  `wake_word_product_ready`
- Current server-side missing evidence is `provider_smoke`, `wake_word`.
- Current positive no-hardware/server evidence: `server_side.v21_professional_evidence_ready=true`
  and `server_side.host_voice_loopback_ready=true`.
- Current continuous voice state: `voice.continuous_voice_ready=true`; xiaozhi
  host product-chain evidence now requires `repeat >= 3`, at least three answer
  turns, at least three barge-in turns, zero failures, answer first-audio p95
  under 1500 ms, and barge-in stop p95 under 300 ms before readiness can ingest
  it. Local voice loopback evidence now also requires `repeat >= 3` before it
  can close continuous voice or host voice loopback readiness.
- Current provider state: `provider.real_provider_ready=false`; the 5080lab
  operator packet exists, but no real returned bundle has been imported yet.
- Current wake-word state: `wake_word.product_ready=false` and
  `wake_word.firmware_package_available=true`; a current reviewed custom
  MultiNet package exists at
  `a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json`
  for desired phrase `小阿二一` / `xiao a er yi`. It is below activation:
  `flash_allowed=false`, `flash_executed=false`, and physical wake proof is
  still required before `wake_word_product_ready` may close.
- Latest full verification:
  `env NO_PROXY='localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16' no_proxy='localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16' make verify`
  passed after the provider runbook, reviewed-build receipt guard, voice
  readiness, launch false-green guard, wake package help-contract, Gateway
  half-duplex playback arm hardening, explicit voice-mode selector, and
  personality runtime prompt merges.
- Latest targeted verification:
  `go test ./internal/gateway -run 'MockPlayback|AudioProbe|RealtimeOnNext|DeviceControl|HalfDuplex|Playback' -count=1`
  passed, and
  `go test ./internal/app -run 'StackChanSpeaker|StackChanTouch|ProductReadiness' -count=1`
  passed.
- Latest wake/readiness targeted verification:
  `go test ./internal/app -run 'WakeWordFirmware|ProductReadiness' -count=1`
  passed on current mainline after ingesting the custom wake package evidence.
- Latest voice-mode verification:
  `go test ./internal/gateway -run 'VoiceMode|FastCompanion|DeviceControl|Simulator' -count=1`
  passed,
  `go test ./internal/app -run 'VoiceMode|ProductReadiness' -count=1`
  passed, and a current-HEAD temporary Gateway on `127.0.0.1:21081` exposed
  `voiceMode`, `voiceModeReadout`, and `/v1/voice-modes`. Selecting
  `pure_cloud` persisted the visible choice and blocked
  `/v1/fast-companion/turn` instead of silently routing planned mode.
- Latest personality runtime verification:
  `go test ./internal/personality ./internal/app -run 'Personality|FastCompanion|LocalVoiceLoopback' -count=1`
  passed,
  `go test ./internal/app -run 'LocalVoiceLoopback.*TextStream|FastCompanion' -count=1`
  passed, `git diff --check` passed, and
  `product-readiness --use-latest-reports` stayed `launch_ready=false` with the
  same real-evidence gaps.
- Latest control-tower refresh:
  `go run ./cmd/a21 product-readiness --use-latest-reports --output-dir reports`
  wrote `a21-product-readiness-20260602-075224.json` with
  `status=mock_demo_ready`, `launch_ready=false`, `demo_ready=true`,
  `wake_word.firmware_package_available=true`,
  server-side `missing_evidence=["provider_smoke","wake_word"]`, and canonical
  `missing_real_evidence=["real_provider_smoke","physical_stackchan_online","physical_stackchan_prd_acceptance","wake_word_product_ready"]`.
- Latest V21 adapter refresh:
  `go run ./cmd/a21 v21-adapter-smoke --adapter-url http://127.0.0.1:21121 --execute --output-dir reports`
  passed against the explicit A21 adapter bridge and wrote
  `a21-v21-adapter-smoke-20260602-073834.json`. The follow-up
  `A21_V21_ADAPTER_URL=http://127.0.0.1:21121 product-readiness --use-latest-reports`
  wrote `a21-product-readiness-20260602-074129.json`, with
  `v21.configured=true`, `v21.healthy=true`,
  `v21.v21_professional_execution.valid=true`, source
  `a21-v21-adapter-smoke-20260602-073834.json`, and the same remaining
  server-side gaps: `provider_smoke`, `wake_word`. The V21 configure/start
  next-action is gone when the explicit adapter URL is supplied.
- Latest server-side bundle refresh:
  `A21_V21_ADAPTER_URL=http://127.0.0.1:21121 server-side-readiness-bundle --use-latest-reports`
  wrote `a21-server-side-readiness-bundle-20260602-075224.json`, with
  `v21.ready=true`, `host_voice.ready=true`, `provider.ready=false`,
  `wake_word.ready=false`, next actions only for provider and wake-word, and no
  PRD fake-green.
- Latest provider operator safety smoke:
  `make provider-5080lab-runbook A21_PROVIDER=mock` failed with exit 2,
  `make provider-5080lab-runbook A21_PROVIDER=selected_provider` printed a
  command bundle only, and
  `make provider-5080lab-runbook A21_PROVIDER=selected_provider A21_PROVIDER_5080LAB_OUTPUT_DIR=/tmp/a21`
  failed with exit 2.
- Latest provider operator packet refresh:
  `make provider-5080lab-runbook A21_PROVIDER=deepseek` printed the current
  print-only 5080lab command sequence. The next provider burn-down event is the
  returned `a21-5080lab-provider-evidence-*.tgz` import, not any Mac-side
  provider `--execute`.
- Latest no-hardware CLI smoke:
  `wake-word-firmware-build-receipt` without `--review-report` failed without
  writing a receipt; the same command with matching
  `a21.wake_word_firmware_build_review.v1` evidence produced
  `a21.wake_word_firmware_build.v1` with `status=built` and
  `review_report=a21-wake-word-build-review.json`; then
  `wake-word-firmware-package --build-receipt` consumed it and returned
  `status=packaged`, `product_ready=false`,
  `build_review=a21-wake-word-build-review.json`, and the expected
  `wake_word_firmware_package_below_activation` finding.
- Latest real build-dir wake-word diagnostic:
  `/Users/jiyurun/Documents/小马暴力/sources/xiaozhi-esp32/build-m5stack-core-s3/xiaozhi.bin`
  exists, but its source `sdkconfig` keeps `CONFIG_USE_AFE_WAKE_WORD=y` and
  `CONFIG_USE_CUSTOM_WAKE_WORD` disabled, with logs packaging
  `wn9_nihaoxiaozhi_tts`. This is stock `你好小智`, not the current A21
  custom `小阿二一` MultiNet build. Running
  `wake-word-firmware-package --plan reports/a21-wake-word-firmware-plan-20260602-073252-1780356772002781000.json --build-dir /Users/jiyurun/Documents/小马暴力/sources/xiaozhi-esp32/build-m5stack-core-s3 --commit x21-stock-check --output-dir reports`
  correctly rejected the input and wrote
  `a21-wake-word-firmware-package-20260602-073950-1780357190953924000.json`.
  Do not use this stock build to close `wake_word_product_ready`.
- Latest current custom wake package:
  a fresh bounded worker built an external xiaozhi/ESP-SR MultiNet CoreS3
  artifact in scratch space, with `CONFIG_USE_CUSTOM_WAKE_WORD=y`,
  `CONFIG_CUSTOM_WAKE_WORD="xiao a er yi"`,
  `CONFIG_CUSTOM_WAKE_WORD_DISPLAY="小阿二一"`, and threshold `35`. The control
  tower then regenerated mainline evidence from the reviewed receipt:
  `a21-wake-word-firmware-plan-20260602-075100-1780357860846684000.json`,
  `.a21-run/wake-word/reviewed-receipt-20260602-075100/a21-wake-word-build.json`,
  and
  `a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json`.
  The packaged artifact is
  `a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-a2d3dc882b42-20260602-075112.bin`
  with sha256
  `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`.
  This closes the no-hardware package gap, not the guarded flash or physical
  custom wake acceptance gap.

Already merged into the current baseline and must not be rediscovered as active
gaps:

- Audio clarity: `8bfed60 fix(audio): improve xiaozhi tts downlink clarity` and
  `960db85 test(audio): guard xiaozhi downlink clarity`
- Canonical launch rollup: `9431423 feat(readiness): add canonical launch rollup`
- Personality assets: `425c009 docs(personality): add A21 personality assets`
- Host voice/V21 evidence surfaces: `3909204 feat(readiness): close host voice and V21 evidence gates`
- Latest contextual readiness evidence selection:
  `fd3fd14 fix(readiness): match latest contextual evidence reports`
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
- Provider 5080lab operator packet:
  `6e83e72 chore(provider): add 5080lab runbook wrapper`
- Wake-word reviewed-build authenticity guard:
  `93cba07 fix(wake-word): require reviewed build evidence`
- Xiaozhi host product-chain evidence guard:
  `f1b1510 fix(readiness): require xiaozhi voice bench rounds`
- Local voice loopback evidence guard:
  `35e62aa fix(readiness): require repeated local voice loopback evidence`
- Wake package operator help contract:
  `1878fc8 fix(wake-word): document build receipt package option`
- Gateway half-duplex playback arm hardening:
  `2120ee6 fix(gateway): harden half duplex playback arm`
- Explicit operator voice-mode selector:
  `4f4b767 feat(gateway): add explicit voice mode selector`,
  merged by `de64e50 merge: voice mode selector slice`
- Personality runtime prompt composer:
  `5babfdb feat(personality): compose fast companion runtime prompt`,
  merged by `8e221de merge: personality runtime prompt slice`

Active workers that the control tower must poll before duplicating work:

| Worker | Thread | Worktree | Branch | Owned slice | Current status |
| --- | --- | --- | --- | --- | --- |
| None | - | - | - | - | No active write worker as of `bcaa61c`; start a fresh bounded worker before any new implementation slice |

Worktree hygiene checkpoint:

- Main worktree dirty state is still only untracked `tools/__pycache__/`.
- Completed worker worktrees may remain for audit/history. Do not merge or reset
  them directly after context compression.
- Three old worktrees still show local dirty entries, but their useful ideas are
  already represented in current mainline or must be re-applied from a fresh
  current-HEAD branch if needed:
  `/Users/jiyurun/.codex/worktrees/5434/New project`,
  `/Users/jiyurun/.codex/worktrees/7232/New project`, and
  `/Users/jiyurun/.codex/worktrees/de71/New project`.
  Treat them as frozen reference material, not active implementation lanes.

Recently completed workers:

| Worker | Thread | Worktree | Branch | Owned slice | Current status |
| --- | --- | --- | --- | --- | --- |
| Personality runtime prompt loader | `019e8581-3a1d-7d62-960e-66d729b14644` | `/Users/jiyurun/.codex/worktrees/3d2d/New project` | `codex/a21-mainline-personality-runtime-20260602` | PRD Phase 2/section 8 runtime composer for fast-companion prompts | Merged via `5babfdb`/`8e221de`; no provider execute, no Mac audio, no hardware; do not duplicate |
| Wake custom build/package closure | `019e858f-b619-7f52-991d-39827af8301b` | `/Users/jiyurun/.codex/worktrees/259a/New project` | worker scratch from current mainline | No-hardware xiaozhi/ESP-SR custom MultiNet build and reviewed A21 package evidence | Completed clean; evidence regenerated in mainline reports; no product-code merge needed; do not duplicate |
| Stale rescue branch audit | `019e8566-5485-76f0-ae0d-9df00f88671c` | `/Users/jiyurun/.codex/worktrees/1900/New project` | detached at `8165328` | Read-only audit of old rescue/pivot branches for PRD-useful patches | Completed read-only; its useful `voice_mode` follow-up is now merged from fresh mainline, not stale branch merge |
| Explicit voice-mode selector | `019e8573-d562-75d2-b126-7bfb72d6af1f` | `/Users/jiyurun/.codex/worktrees/7142/New project` | `codex/a21-mainline-voice-mode-selector-20260602` | Explicit `voice_mode` catalog/status/selection without hidden routing | Initial worker stopped cleanly with no implementation; control tower reclaimed, implemented, verified, and merged via `4f4b767`/`de64e50`; do not duplicate |
| Gateway half-duplex arm hardening | `019e8569-b007-75a1-ae6a-34b3d2b7fd4b` | `/Users/jiyurun/.codex/worktrees/de71/New project` | `codex/a21-gateway-half-duplex-arm-hardening-20260602` | `mock_playback_on_next_audio_frame` multi-chunk arm and failed-delivery rollback | Worker stopped with partial tests; control tower reclaimed and merged current implementation via `2120ee6`; do not merge worker branch |
| Realtime evidence closure | `019e851c-1c6b-7d53-8437-6cbe4b57692c` | `/Users/jiyurun/.codex/worktrees/5434/New project` | `codex/a21-realtime-evidence-closure-20260602` | Provider realtime fixture report/output-dir and product-readiness visibility | Merged via `66798f8` and `9526976`; do not duplicate |
| Mode/privacy closure | `019e851d-1917-75c1-b4cb-f4a26c7851ca` | `/Users/jiyurun/.codex/worktrees/7232/New project` | `codex/a21-mode-privacy-closure-20260602` | Public/private/focus/professional mode red lines and visible state | Merged via `2d102c4` and `61d121d`; do not duplicate |
| Wake-word no-hardware diagnostic closure | `019e8528-fd53-77b3-a17e-676eed19ed9e` | `/Users/jiyurun/.codex/worktrees/7a3c/New project` | `codex/a21-wake-word-package-readiness-20260602` | Missing build-dir/receipt diagnostic package reports and product-readiness next-action guard | Merged via `2f01049`; do not duplicate |
| Provider selected-evidence closure | `019e8529-78c3-7fa3-9a23-a776180e5009` | `/Users/jiyurun/.codex/worktrees/fc26/New project` | `codex/a21-provider-5080lab-evidence-closure-20260602` | Selected-provider matching for latest smoke evidence, package/import, and 5080lab repeat-3 runbook | Merged via `f57cbda`; do not duplicate |
| Wake build receipt closure | `019e8537-d0bf-71b1-b72a-b5e3cf90c793` | `/Users/jiyurun/.codex/worktrees/c7e2/New project` | `codex/a21-wake-build-receipt-20260602` | Build receipt command and explicit receipt-to-package bridge for reviewed xiaozhi/ESP-SR MultiNet output | Merged via `f288741`; do not duplicate |
| No-hardware PRD gap audit | `019e8537-d17e-7320-a13d-6a122cab9dd0` | `/Users/jiyurun/.codex/worktrees/efbe/New project` | detached at `9939bda` | Read-only audit of current no-hardware gaps | Completed read-only; no merge needed |
| Provider 5080lab operator packet | `019e8544-788b-7e21-924c-0af318e6d8fa` | `/Users/jiyurun/.codex/worktrees/fe41/New project` | `codex/a21-provider-5080lab-closure-20260602` | Print-only selected-provider 5080lab runbook with mock/unsafe path rejection | Merged via `6e83e72`; do not duplicate |
| Wake build authenticity guard | `019e8544-42e4-7b82-816c-ecea282af127` | `/Users/jiyurun/.codex/worktrees/8e44/New project` | `codex/a21-wake-build-authenticity-guard-20260602` | Require explicit matching reviewed-build evidence before producing `a21-wake-word-build.json` | Merged via `93cba07`; do not duplicate |
| Voice product-chain readiness | `019e8550-ea8d-7bc2-b248-115a4574cacb` | `/Users/jiyurun/.codex/worktrees/95ac/New project` | `codex/a21-voice-product-chain-readiness-20260602` | Require xiaozhi host product-chain reports to prove at least three answer and barge-in rounds before closing continuous voice readiness | Merged via `f1b1510`; do not duplicate |
| Launch-rollup false-green review | `019e8559-7305-75c3-a3b6-e929e0a3bb57` | `/Users/jiyurun/.codex/worktrees/06f6/New project` | `codex/a21-launch-rollup-falsegreen-review-20260602` | Require local voice loopback reports to prove repeated evidence before closing continuous voice or host loopback readiness | Merged via `35e62aa`; do not duplicate |

Current canonical PRD gaps from the latest product-readiness run:

- `real_provider_smoke`
- `physical_stackchan_online`
- `physical_stackchan_prd_acceptance`
- `wake_word_product_ready`

Current server-side gaps from the latest product-readiness run:

- `provider_smoke`
- `wake_word`

Current next moves:

1. Provider execute closure for 5080lab: run
   `make provider-5080lab-runbook A21_PROVIDER=deepseek`
   to print the lab packet, execute the printed provider commands on 5080lab,
   import the returned bundle on the control machine, then rerun
   product-readiness to reduce `real_provider_smoke`. Do not run provider
   `--execute` on this Mac.
2. V21 adapter evidence refresh is current as of
   `a21-v21-adapter-smoke-20260602-073834.json`. Keep it as evidence and do not
   open more V21 implementation work unless the adapter contract regresses or a
   newer backend change requires a refresh. Do not call V21 internals from A21
   code.
3. Wake-word package closure: no-hardware custom package evidence now exists for
   the current `小阿二一` / `xiao a er yi` MultiNet build. Do not rebuild this
   unless the desired phrase/threshold or upstream xiaozhi commit changes.
   `product_ready` must remain false until a guarded flash and physical wake
   proof are collected. The previously found X21 stock build directory remains
   rejected diagnostic evidence only.
4. Optional no-hardware polish should be dispatched only if it burns a current
   PRD gap or fixes a regression. Do not reopen audio clarity, host voice
   loopback, voice-mode selector, personality runtime, selected-provider
   import, or launch false-green guards unless a new regression appears.
5. Hardware window when CoreS3 returns: collect physical online/audio/playback
   stop/custom wake evidence only after the server/provider/wake package seams
   are ready.

## 1. Current PRD Burn-Down Baseline

| Slice | Current status | Current evidence | Full-launch gap |
| --- | --- | --- | --- |
| Phase 0 Go-first foundation | Done | CLI, host gate, doctor, provider default mock, namespace guard | Keep green while integrating |
| Provider spine and text stream | Server contract done for selected evidence intake, real evidence pending | Built-in profiles, text stream parser, smoke/repeat/redaction, p99, 5080lab runbook, evidence package/import, selected-provider mismatch rejection | 5080lab real non-mock executed smoke bundle |
| Fast companion hybrid lane | Host/simulator chain done, physical pending | Xiaozhi product chain, ASR/Text/TTS adapters, Opus downlink, pacing, turn cancel, audio-clarity regression, half-duplex arm hardening, host p95 evidence | Real provider evidence plus physical StackChan audible playback and barge-in acceptance |
| Realtime voice lane | Offline evidence/reporting done; live provider still pending | Provider-neutral fixture, explicit arm concept, `provider-realtime-fixture --output-dir`, product-readiness `provider.realtime_*` fields | Live provider lane smoke and physical playback acceptance |
| V21 professional mode | No-hardware evidence ready, launch physical/user acceptance pending | V21 adapter contract, checking feedback, evidence/cards/follow-ups, `v21_professional_execution` rollup | Keep real adapter evidence current; physical/public-mode acceptance still required |
| Agent task bridge | Contract done | AgentTask interface, Hermes/MiMo profiles, safety mapper | Keep out of first-audio path; later UX polish only |
| Physical StackChan | Partial | Capability charts, evidence commands, playback ack/debug profile, firmware guards, candidate downlink reports | CoreS3 physical online, mic/audio/playback stop, touch/screen/servo/RGB/wake word acceptance |
| Wake word | No-hardware config/plan/build-receipt/custom package path done; product-ready still pending | Frontend/Gateway desired phrase persistence, pending firmware status, guarded plan/package commands, package report ingestion, missing build-dir/receipt diagnostic reports, explicit build receipt bridge, current reviewed `小阿二一` MultiNet package artifact and sha | Guarded flash and physical custom wake proof |
| Personality and playbooks | Runtime workmate composer done, scenario UX later only when needed | `docs/personality` assets merged; `internal/personality` composes core + tone + one mode + optional scenario/failure overlay for fast companion prompt | Scenario selection/product UX polish only when a real flow needs it |

Important correction: `8bfed60 fix(audio): improve xiaozhi tts downlink clarity` is already merged into the current baseline. The previous audio-risk review is not a live gap. Treat it as a regression guard only.

## 2. Branch And Worktree Strategy

Use the current integration branch only for reviewed merges and acceptance reports:

- `codex/a21-integration-runtime-readiness-20260601`: integration candidate and launch rollup.
- `codex/a21-control-prd-full-push-plan-20260602`: this control plan only.
- `codex/a21-mainline-voice-product-chain-*`: server-side voice chain implementation.
- `codex/a21-mainline-voice-mode-selector-*`: explicit front-end/operator voice-mode selection without hidden routing.
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

### Slice H: Explicit Voice Mode Selector

Goal: make the operator/front-end voice-mode choice explicit and visible without
turning A21 into a hidden router or weakening the existing
public/private/professional red lines.

Owner branch: `codex/a21-mainline-voice-mode-selector-20260602`
Write set:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `docs/engineering/PROTOCOL.md`
- new `docs/engineering/VOICE_MODE_SELECTION.md` if the operator contract needs
  its own page

Acceptance:

- `go test ./internal/gateway -run 'VoiceMode|FastCompanion|DeviceControl|Simulator' -count=1`
- `go test ./internal/app -run 'VoiceMode|ProductReadiness' -count=1`
- `git diff --check`
- `env NO_PROXY='localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16' no_proxy='localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16' make verify`

Done means:

- A front-end/operator can list and select a `voice_mode` explicitly.
- `voice_mode` is separate from product `mode`; professional/private routing
  rules stay governed by existing mode/privacy guards.
- The selected state is visible in Gateway/simulator status and reports.
- No provider execute, V21 execute, Mac audio, firmware, or hardware path is
  opened by this slice.

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

1. Control-tower no-hardware evidence refresh: done for V21 adapter at
   `127.0.0.1:21121`, using explicit A21 adapter smoke only. Do not repeat this
   unless the adapter/backend changes.
2. External/provider lane: produce or import a real 5080lab provider evidence
   bundle using the generated runbook. The Mac control tower can package/import
   returned redacted reports, but executed provider smoke belongs on 5080lab.
3. Wake/package lane: no-hardware package evidence is current. Keep it below
   activation and explicitly not product-ready until the hardware window can
   flash and prove physical custom wake.
4. Dispatch a fresh no-hardware worker only for a concrete new failing command
   or report/import gap after Slice H starts. Do not widen Slice H if provider
   or wake evidence returns unrelated failures.
5. Dispatch AgentTask bridge or mode/privacy UX polish only after provider/wake
   evidence import is not blocking server-side readiness; these must not enter
   the first-audio path.
6. When CoreS3 returns, open exactly one Slice G foreground hardware window for
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

Do not start Slice H again; explicit `voice_mode` is already merged. The current
next move is to burn down server-side evidence:

1. Wait for or trigger the approved 5080lab selected-provider execution bundle,
   then import it with `provider-evidence-import`.
2. Hold the current custom wake package as the accepted no-hardware artifact:
   `a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-a2d3dc882b42-20260602-075112.bin`.
   The next wake action is guarded flash plus physical custom wake proof when
   CoreS3 returns. The found X21 stock `build-m5stack-core-s3` path remains
   rejected because it lacks the matching custom wake receipt.
3. If neither external evidence lane is immediately available, open one fresh
   no-hardware worker only for a concrete report-ingestion bug or operator
   helper that reduces `provider_smoke` or `wake_word`.

Provider `--execute` still does not run on this Mac. Firmware package evidence is
not physical wake acceptance, and no CoreS3 absence may block simulator,
protocol, provider packaging, V21 boundary, or report-ingestion work.

Voice clarity is not an open implementation gap; it is a regression check.
Old detached rescue branches are source material only; no stale branch should
be merged into current mainline.
