# A21 Lean Carve Log

This file records irreversible carve actions before they happen. It is not a
run diary and must stay short.

## Initial State - 2026-06-05

- Branch `main-lean` created from frozen HEAD `090eca6`.
- Current local branch count is above the lean target; branch deletion has not
  started.
- No tracked report/artifact removal has started.
- No product behavior rewrite has started.

## Planned Deletion Classes

- Local/runtime artifacts to ignore or move out of the repository:
  `reports/`, `.a21-run/`, `.a21-tmp/`, `.a21-tools/`, `dist/`, and
  `.playwright-cli/`.
- Branches with no content unique from `main-lean` may be deleted only after
  `git log main-lean..<branch> --oneline` is reviewed.
- Branches with unique useful commits must be cherry-picked individually before
  deletion; the cherry-picked commit or "no unique content" decision must be
  recorded below.

## Deletions

- Planned mechanical replacement before file split: remove the original
  oversized Go files after generating same-package split files from their
  top-level declarations. Reason: enforce reviewable file sizes without
  changing runtime behavior.
  - `internal/app/app.go`
  - `internal/app/app_test.go`
  - `internal/app/product_demo.go`
  - `internal/app/official_stackchan.go`
  - `internal/app/official_stackchan_test.go`
  - `internal/app/provider_latency_bench.go`
  - `internal/app/xiaozhi_physical_evidence.go`
  - `internal/app/xiaozhi_physical_evidence_test.go`
  - `internal/app/xiaozhi_voice_bench.go`
  - `internal/app/xiaozhi_professional_bench.go`
  - `internal/app/fast_companion_turn.go`
  - `internal/gateway/server.go`
  - `internal/gateway/server_test.go`
  - `internal/gateway/workspace_console.go`
  - `internal/providers/voice_pipeline.go`
  - `internal/providers/voice_pipeline_adapters.go`
  - `internal/providers/voice_pipeline_real_adapters_test.go`
  - `internal/audio/local_tts.go`
- Planned tracked report removal before `git rm`: remove stale generated report
  artifacts from Git while keeping `reports/` ignored. Reason: reports are
  runtime evidence artifacts, not product source.
  - `reports/a21-stackchan-official-baseline-20260604-234848-1780588128571683000.json`
  - `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-235129-1780588289127583000.json`
  - `reports/a21-stackchan-state-reaction-evidence-20260604-2316.json`
  - `reports/a21-stackchan-touch-reaction-evidence-20260604-224756.json`
  - `reports/a21-xiaozhi-half-duplex-acceptance-20260604-235541.288482000.json`
  - `reports/a21-xiaozhi-physical-evidence-20260604-235541.527765000.json`

## Latency Lock

- 2026-06-05 current carve attempt: `A21_PROVIDER_PRIMARY=stepfun A21_GATEWAY_URL=http://47.103.57.217 A21_DEVICE_ID=44:1b:f6:e2:6a:60 A21_LOCAL_ASR_PROVIDER=mock_asr make stackchan-fast-companion-turn` failed before provider/audio timing because Gateway `GET /v1/devices` returned EOF. Report path: `reports/a21-stackchan-fast-companion-turn-20260605-211656.json`; status `failed`; finding `gateway device report failed`. No p95/barge-in success recorded.
- 2026-06-05 continuation: the Gateway EOF was reclassified as a false
  negative from the local TUN route. Source-bound checks with
  `192.168.1.27` returned HTTP 200 for `/healthz`, `/v1/devices`, and
  `/xiaozhi/ota/`; `/v1/voice-chain-profiles` reports selected LLM
  `stepfun`. `A21_PROVIDER_PRIMARY=stepfun A21_GATEWAY_URL=http://47.103.57.217 A21_DIRECT_SOURCE_IP=192.168.1.27 A21_DEVICE_ID=44:1b:f6:e2:6a:60 A21_LOCAL_ASR_PROVIDER=sherpa_onnx A21_FAST_COMPANION_LISTEN_SOURCE=stackchan_mic make stackchan-fast-companion-turn` now fails with report path
  `reports/a21-stackchan-fast-companion-turn-20260605-213250.json`, status
  `failed`, and finding `device missing from Gateway`. No p95/barge-in success
  recorded.
- Read-only recovery precheck with the same direct-source path wrote
  `reports/a21-stackchan-product-recovery-20260605-213040.json` with status
  `product_offline_serial_missing`; official relay is disconnected,
  `/dev/cu.usbmodem1101` is absent, and the next actions are
  `connect_product_usb_or_power` and `recheck_product_recovery_status`.
- 2026-06-05 22:16 CST continuation: direct-source Gateway health is OK and
  product device `44:1b:f6:e2:6a:60` is listed, but it is stale and
  `xiaozhi_ws_disconnected`. The direct-source true-provider-intent
  `make stackchan-fast-companion-turn` run wrote
  `reports/a21-stackchan-fast-companion-turn-20260605-221543.json`, status
  `failed`, finding `device is not online`; read-only recovery wrote
  `reports/a21-stackchan-product-recovery-20260605-221605.json`, status
  `product_offline_serial_missing`. Local `stepfun` provider smoke is also not
  closed because required provider env is missing locally; 5080lab remains an
  allowed provider-evidence path, not a substitute for physical StackChan
  p95/barge-in acceptance.
- 2026-06-05 22:27 CST continuation after user reported 5080lab available:
  5080lab responds to LAN ping at `192.168.1.6`, but TCP/22 and SSH time out,
  so Codex cannot run the 5080 provider closure remotely yet. No fresh
  `reports/a21-5080lab-provider-evidence-*.tgz` is present on the control
  machine. Local dry-run StepFun provider smoke wrote
  `reports/5080lab-provider/a21-provider-smoke-20260605-222655-624183000.json`
  with `status=ready`, `configured=true`, `route_eligible=true`,
  `stream=true`, `executed=false`; no provider network execution was run on
  the Mac. Direct-source Gateway health remains OK and device
  `44:1b:f6:e2:6a:60` remains `xiaozhi_ws_disconnected`. The direct-source
  true-provider-intent `make stackchan-fast-companion-turn` run wrote
  `reports/a21-stackchan-fast-companion-turn-20260605-222723.json`, status
  `failed`, finding `device is not online`; read-only recovery wrote
  `reports/a21-stackchan-product-recovery-20260605-222723.json`, status
  `product_offline_serial_missing`. No p95/barge-in success recorded.

## Lean Gate Results - 2026-06-05

- Branch fan-in completed to exactly three local branches:
  `main-lean`, `codex/a21-stabilization-after-internal-test4-20260605`, and
  `codex/a21-hardware-window-20260605-product-flash-9f4532a`.
- Tracked generated report artifacts were removed from Git; `reports/` remains
  ignored for future runtime evidence artifacts.
- Product/lab CLI split completed: product `cmd/a21` no longer dispatches
  demo/bench/evidence/professional lab commands, and lab execution moves behind
  `cmd/a21-lab`.
- Oversized Go files were mechanically split without changing package
  ownership. Current maximum product implementation file is 773 lines; current
  maximum test file is 896 lines.
- Product dependency check passed:
  `go list -deps ./cmd/a21 | grep -iE 'bench|demo|evidence|professional'`
  produced no matches.
- Product binary symbol check passed: temporary `cmd/a21` binary had no `lab`
  or `simulator` symbol matches under `go tool nm`.
- Verification passed: `go test ./...`, `make verify`, `make preflight`
  sequential rerun, `make doctor`, and `bash scripts/lean-gate.sh`.
- CI verification passed on `main-lean` at commit `3502a57042c7`: GitHub
  Actions run `27019468711` passed both `Run A21 lean gate` and
  `Run A21 release check`.
- North-star runtime evidence is not locked: the live public Gateway at
  `http://47.103.57.217` returned EOF for `/v1/devices`, so the true-provider
  fast companion p95 and barge-in stop metrics were not collected in this
  carve.
- Remote recovery probe is blocked from this workspace: SSH to
  `root@47.103.57.217` with the recorded A21 operations key closed the
  connection on port 22, and direct no-proxy `curl` to public `/healthz` and
  `/v1/devices` returned `Empty reply from server`.

## Branch Fan-In Deletion Plan - 2026-06-05

- Keep exactly three local branches: `main-lean` (active lean carve), `codex/a21-stabilization-after-internal-test4-20260605` (frozen internal-test4 baseline at `090eca6`), and `codex/a21-hardware-window-20260605-product-flash-9f4532a` (latest product-flash lane kept as the one flashing branch).
- Branches below are not cherry-picked: the user-approved frozen baseline is `090eca6`, current lean work starts from it, and these branch heads are either ancestors or historical side branches with no unique product value for this carve.
- Clean linked worktrees to remove before branch deletion:
  - `/private/tmp/a21-flash-clean-0680d42` for `codex/a21-hardware-window-20260605-product-flash-0680d42`: clean worktree for a branch deleted below.
  - `/private/tmp/a21-restore-0680d42` for `codex/a21-hardware-window-20260605-restore-0680d42`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-official-config-fallback-20260605/New project` for `codex/a21-official-config-fallback-20260605`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-official-pcm-bridge-5080-rescue/New project` for `codex/a21-p0-official-pcm-bridge-5080-rescue-20260601`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-official-xiaozhi-nvs-20260602/New project` for `codex/a21-official-xiaozhi-nvs-20260602`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-p0-voice-quality-integration/New project` for `codex/a21-p0-voice-quality-integration-20260601`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-physical-prd-review-acceptance/New project` for `codex/a21-physical-prd-review-acceptance-20260602`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-provider-compat-matrix/New project` for `codex/a21-provider-compat-matrix-20260602`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-pure-cloud-voice-matrix` for `codex/a21-pure-cloud-voice-matrix`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-sherpa-streaming-asr-runtime-manual/New project` for `codex/a21-sherpa-streaming-asr-runtime-manual-20260603`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-slice-a-launch-rollup-aggregator/New project` for `codex/a21-slice-a-launch-rollup-aggregator-20260602`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-slice-b-provider-execution-package/New project` for `codex/a21-slice-b-provider-execution-package-20260602`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-slice-e-personality-assets/New project` for `codex/a21-slice-e-personality-assets-20260602`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-slice-f-custom-wake-word/New project` for `codex/a21-slice-f-custom-wake-word-20260602`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/a21-xiaozhi-pivot-freeze/New project` for `codex/a21-xiaozhi-protocol-pivot-freeze-20260601`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/fc26/New project` for `codex/a21-provider-voiceclone-compat-matrix-20260602`: clean worktree for a branch deleted below.
  - `/Users/jiyurun/.codex/worktrees/m3-summary-rescue/New project` for `codex/a21-fast-companion-m3-readiness-summary-rescue`: clean worktree for a branch deleted below.
- Detached dirty worktree intentionally untouched because it does not count as an active branch: `/private/tmp/a21-firmware-clean-32c5286`.
- Branch deletion list:
  - `codex/a` (`0b4e878`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): open binary opus post review
  - `codex/a21-5080-indextts2-bridge` (`0a8e77d`, ahead=2): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(audio): use remote IndexTTS2 model root and pcm16 output
  - `codex/a21-agent-task-bridge-scaffold` (`a1742ae`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): record fixture sidecar post review
  - `codex/a21-audio-front-end-evidence-contract` (`2b4f2ad`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): open post-latency prd audit
  - `codex/a21-binary-opus-media-contract` (`cc27176`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): accept post-audio prd audit
  - `codex/a21-control-prd-full-push-plan-20260602` (`07f25e2`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: merge: add 5080lab provider evidence package
  - `codex/a21-correct-firmware-build-20260602` (`c7bec37`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(firmware): add official xiaozhi compatible stackchan build
  - `codex/a21-docs-pcm-bridge-flash-adr` (`a031f3d`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(firmware): add PCM bridge flash ADR gate
  - `codex/a21-failure-copy-recovery-contract` (`ac34c94`, ahead=9): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(product): add failure recovery copy contract
  - `codex/a21-fast-companion-hybrid-boundary` (`94d87c3`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): accept fast companion audit
  - `codex/a21-fast-companion-hybrid-boundary-audit` (`3b9f05d`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): reconcile provider spine parser plan
  - `codex/a21-fast-companion-m3-readiness-report` (`2ac6ece`, ahead=13): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): summarize fast companion m3 readiness
  - `codex/a21-fast-companion-m3-readiness-report-802c` (`80b4e43`, ahead=13): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): summarize fast companion m3 readiness
  - `codex/a21-fast-companion-m3-readiness-summary` (`0d1e2f1`, ahead=13): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): summarize fast companion m3 readiness
  - `codex/a21-fast-companion-m3-readiness-summary-rescue` (`90efcb0`, ahead=13): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): summarize fast companion m3 readiness
  - `codex/a21-fast-companion-runtime-20260601` (`d33b7eb`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(gateway): run fast companion voice pipeline
  - `codex/a21-fast-mainline` (`0b4e878`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): open binary opus post review
  - `codex/a21-freeze-x21-firmware-source-20260602` (`4cf7014`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(firmware): freeze external x21 xiaozhi builds
  - `codex/a21-frontend-xiaozhi-console-20260601` (`2f61df2`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(gateway): add xiaozhi operations console
  - `codex/a21-gateway-half-duplex-arm-hardening-20260602` (`1878fc8`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(wake-word): document build receipt package option
  - `codex/a21-governance-lean-gates` (`ccd3190`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(runtimeguard): allow verified gateway launch port
  - `codex/a21-half-duplex-speaker-evidence-fix` (`2dce170`, ahead=16): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(gateway): clear failed validation arms
  - `codex/a21-hardware-full-validation-20260531` (`ff3bed6`, ahead=14): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: merge: a21 fast companion m3 readiness summary
  - `codex/a21-hardware-progress-summary` (`722f2c9`, ahead=6): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): summarize stackchan hardware progress
  - `codex/a21-hardware-window-20260531-official-pcm-bridge` (`a215625`, ahead=4): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(providers): promote fast mainland text streams
  - `codex/a21-hardware-window-20260601-stackchan` (`c747227`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(app): ingest executed v21 adapter smoke
  - `codex/a21-hardware-window-20260601-xiaozhi-physical` (`e2999e0`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(readiness): ingest external xiaozhi professional proof
  - `codex/a21-hardware-window-20260601-xiaozhi-physical-evidence` (`a6b9889`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): add xiaozhi physical evidence report
  - `codex/a21-hardware-window-20260602-stackchan-prd` (`2f8a63f`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(xiaozhi): suppress stale opus ingress after abort
  - `codex/a21-hardware-window-20260603-flash-clean` (`4c4178a`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(stackchan): keep official dependency cache clean
  - `codex/a21-hardware-window-20260603-wake-override-flash` (`aa80523`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(firmware): allow wake on idle socket
  - `codex/a21-hardware-window-20260603-wifi-provisioning-flash` (`87579cf`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(readiness): record stepfun server candidate closure
  - `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs` (`1387d58`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: chore(release): prepare internal test 4 package
  - `codex/a21-hardware-window-20260605-product-flash-0680d42` (`0680d42`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(firmware): regenerate official xiaozhi overlay
  - `codex/a21-hardware-window-20260605-restore-0680d42` (`0680d42`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(firmware): regenerate official xiaozhi overlay
  - `codex/a21-hardware-window-bootstrap-20260531-2343` (`ff3bed6`, ahead=14): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: merge: a21 fast companion m3 readiness summary
  - `codex/a21-hardware-window-mic-probe-20260531` (`ff3bed6`, ahead=14): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: merge: a21 fast companion m3 readiness summary
  - `codex/a21-hardware-window-product-demo-20260531` (`8cb49fa`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(firmware): add bidirectional official pcm bridge
  - `codex/a21-hw-flash-plan-20260602` (`114f1e3`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): plan hardware flash evidence transition
  - `codex/a21-iflytek-lane-ledger` (`39c028a`, ahead=5): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: docs(control): register iflytek voice lane audit
  - `codex/a21-integration-governance-slices` (`5f8b792`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: docs(control): coordinate mainline acceleration
  - `codex/a21-integration-runtime-readiness-20260601` (`4fa66c8`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(audio): use remote IndexTTS2 model root and pcm16 output
  - `codex/a21-internal-test4-mainline-20260604` (`87579cf`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(readiness): record stepfun server candidate closure
  - `codex/a21-launch-rollup-falsegreen-review-20260602` (`5038787`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(readiness): require repeated local voice loopback evidence
  - `codex/a21-m3-readiness-gate-impl` (`14db2a9`, ahead=4): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): add m3 readiness gate report
  - `codex/a21-m3-readiness-gate-review` (`14db2a9`, ahead=4): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): add m3 readiness gate report
  - `codex/a21-mainline-acceleration` (`8c0600f`, ahead=17): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: merge: a21 half-duplex speaker evidence fix
  - `codex/a21-mainline-personality-runtime-20260602` (`8a210d3`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(personality): add bounded memory prompt state
  - `codex/a21-mainline-professional-v21-contract` (`fc61793`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(v21): wire configured adapter into gateway env
  - `codex/a21-mainline-runtime-readiness-20260601` (`bd6baa1`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(readiness): auto-ingest latest evidence reports
  - `codex/a21-mainline-stackchan-hardware-diagnostic` (`1e38804`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(hardware): enforce StackChan capability honesty
  - `codex/a21-mainline-voice-mode-selector-20260602` (`4f4b767`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(gateway): add explicit voice mode selector
  - `codex/a21-mode-privacy-closure-20260602` (`01788de`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(agentplan): block private professional routing
  - `codex/a21-mode-transition-contract` (`8d894aa`, ahead=6): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(product): add mode transition contract
  - `codex/a21-official-compatible-flash-plan-20260602` (`cbbd70e`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(firmware): add official xiaozhi compatible flash plan
  - `codex/a21-official-config-fallback-20260605` (`7aaf50a`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(stackchan): unlock official home via a21 nvs config
  - `codex/a21-official-deps-download-20260531` (`f5c0df1`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(firmware): fetch commit deps directly
  - `codex/a21-official-xiaozhi-nvs-20260602` (`dfe6952`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(firmware): add official xiaozhi nvs connection config
  - `codex/a21-p0-asr-recognition-20260601` (`a8619ab`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: chore(audio): archive local asr recognition spike
  - `codex/a21-p0-audio-current-cutoff-fix` (`dea0cd5`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: chore(firmware): archive current playback cutoff spike
  - `codex/a21-p0-official-pcm-bridge-5080-rescue-20260601` (`8c0600f`, ahead=17): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: merge: a21 half-duplex speaker evidence fix
  - `codex/a21-p0-official-pcm-bridge-root-20260601` (`8c0600f`, ahead=17): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: merge: a21 half-duplex speaker evidence fix
  - `codex/a21-p0-streaming-pipeline-bargein-20260601` (`8c0600f`, ahead=17): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: merge: a21 half-duplex speaker evidence fix
  - `codex/a21-p0-tts-current-cutoff-mature-fix-20260601` (`313e06c`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: chore(audio): archive tts cutoff transition spike
  - `codex/a21-p0-voice-quality-integration-20260601` (`c1cd21e`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: chore(audio): archive voice quality integration spike
  - `codex/a21-phase1-clean-skeleton` (`83e8715`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(firmware): plan official StackChan bridge flashing
  - `codex/a21-physical-instrument-evidence-77f5` (`2eb61c9`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): ingest xiaozhi instrument evidence
  - `codex/a21-physical-observation-cli-20260601` (`73615aa`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(readiness): add xiaozhi observation sidecar
  - `codex/a21-physical-prd-review-acceptance-20260602` (`61303d7`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: chore(control): record wake physical proof recorder
  - `codex/a21-prd-p0-dialogue-readiness-report-20260601` (`947e46d`, ahead=17): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(app): redact fast companion readiness source
  - `codex/a21-product-copy-scaffold` (`b13fea9`, ahead=2): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(product): add professional copy scaffold
  - `codex/a21-product-demo-launcher` (`32375c4`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(app): add product demo readiness launcher
  - `codex/a21-project-control` (`7bdfe9d`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): record PCM flash ADR handoff
  - `codex/a21-provider-5080lab-closure-20260602` (`bacd9da`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: chore(provider): add 5080lab runbook wrapper
  - `codex/a21-provider-5080lab-evidence-closure-20260602` (`897a950`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(provider): enforce selected smoke evidence
  - `codex/a21-provider-compat-matrix-20260602` (`1a18efb`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(provider): add compatibility matrix
  - `codex/a21-provider-evidence-package-20260602` (`b8f1a50`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(provider): package evidence bundles
  - `codex/a21-provider-fixture-sidecar-hardening` (`7eb3a8d`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): record provider fixture post review
  - `codex/a21-provider-hot-plug-profiles` (`f8413aa`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(providers): load hot-plug text stream profiles
  - `codex/a21-provider-latency-bench-scaffold` (`52c64d2`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): track prd next-slice audit
  - `codex/a21-provider-latency-fixture-schema` (`b8d46f8`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): record provider latency post review
  - `codex/a21-provider-latency-report-v2` (`2f34854`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(control): track post-agent prd audit
  - `codex/a21-provider-spine-deepseek-textstream` (`6b7fdf0`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(providers): measure first content from content deltas
  - `codex/a21-provider-textstream-parser` (`4d356e7`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(providers): add provider reference profiles
  - `codex/a21-provider-v21-nohardware-closure-20260602` (`d963935`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(app): select newest usable V21 readiness evidence
  - `codex/a21-provider-voiceclone-compat-matrix-20260602` (`b0672eb`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(provider): preserve voice clone matrix evidence
  - `codex/a21-pure-cloud-voice-matrix` (`bfb1389`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(cloud-voice): add no-execute profile selector
  - `codex/a21-realtime-evidence-closure-20260602` (`b94fce0`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(provider): expose realtime fixture evidence
  - `codex/a21-server-mainline` (`869c9c9`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(gateway): add xiaozhi ota and guarded flash
  - `codex/a21-sherpa-realmodel-no-audio-smoke-20260603` (`8dbd5b9`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(a21): add sherpa streaming asr smoke
  - `codex/a21-sherpa-streaming-asr-runtime-helper` (`199d796`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(providers): add sherpa streaming ASR helper
  - `codex/a21-sherpa-streaming-asr-runtime-manual-20260603` (`041ad69`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(providers): add sherpa streaming asr helper
  - `codex/a21-silero-vad-executable` (`48b5d0d`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(audio): add executable silero vad runner
  - `codex/a21-slice-a-launch-rollup-aggregator-20260602` (`9431423`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(readiness): add canonical launch rollup
  - `codex/a21-slice-b-provider-execution-package-20260602` (`3a8f526`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(provider): add 5080lab selected provider evidence package
  - `codex/a21-slice-e-personality-assets-20260602` (`425c009`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(personality): add A21 personality assets
  - `codex/a21-slice-f-custom-wake-word-20260602` (`f74bd6e`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(wake-word): expose custom pending firmware status
  - `codex/a21-streaming-tts-runtime-proof-001` (`37ffcca`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(a21): add streaming tts runtime smoke
  - `codex/a21-virtual-xiaozhi-harness` (`9fd6038`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(app): split provider latency benchmark stages
  - `codex/a21-voice-clone-wrapper` (`cfe1aae`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(audio): support remote voice clone wrapper
  - `codex/a21-voice-compat-safety-20260602` (`dfcd7e8`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(audio): add voice clone tts wrapper
  - `codex/a21-voice-mode-selector-contract` (`a27f9b1`, ahead=6): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(gateway): add voice mode selector contract
  - `codex/a21-voice-product-chain-readiness-20260602` (`f24c3d2`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(readiness): require xiaozhi voice bench rounds
  - `codex/a21-wake-build-authenticity-guard-20260602` (`c8530ce`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(wake-word): require reviewed build evidence
  - `codex/a21-wake-build-receipt-20260602` (`91a4f9e`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(wake-word): add build receipt command
  - `codex/a21-wake-physical-acceptance-20260602` (`40109e4`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(wake-word): add physical acceptance evidence seam
  - `codex/a21-wake-physical-proof-20260602` (`a5354bb`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(wake-word): add physical proof recorder
  - `codex/a21-wake-word-burndown-20260602` (`cb2d6bd`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(wake-word): add simulator reset export controls
  - `codex/a21-wake-word-nohardware-closure` (`57a47c9`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(readiness): select latest usable evidence reports
  - `codex/a21-wake-word-package-readiness-20260602` (`2feccd7`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(wake-word): report package input diagnostics
  - `codex/a21-wake-word-package-readiness-ingestion-20260602` (`f4531bf`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): ingest wake word firmware package readiness
  - `codex/a21-workflow-state-machine-20260602` (`8349dfe`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: docs(control): add handoff and state machine workflow
  - `codex/a21-workspace-local-index-20260604` (`bf20e56`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(hardware): record wifi provisioning attempt
  - `codex/a21-ws-v21-physical-asr-bridge` (`3dfc791`, ahead=2): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(v21): honor bridge error taxonomy bodies
  - `codex/a21-ws-xiaozhi-debug-playback-ack-firmware` (`bebc03a`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(firmware): add xiaozhi debug playback ack overlay
  - `codex/a21-ws2-fast-ack-downlink` (`422e191`, ahead=2): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(gateway): close xiaozhi fast ack failure lifecycle
  - `codex/a21-ws2-real-voice-adapters` (`9fd6038`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(app): split provider latency benchmark stages
  - `codex/a21-ws2b-streaming-downlink` (`497d7a5`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(gateway): stream xiaozhi answer downlink
  - `codex/a21-ws2h-voice-answer-length` (`2843d36`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(providers): cap voice pipeline text budget
  - `codex/a21-ws3-turn-cancel` (`4001582`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(gateway): harden xiaozhi turn cancel seam
  - `codex/a21-ws3-xiaozhi-barge-in` (`9fd6038`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(app): split provider latency benchmark stages
  - `codex/a21-ws4-professional-bridge-readiness` (`82b81f7`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(v21adapter): expose professional bridge readiness
  - `codex/a21-ws4-professional-readiness-report` (`1ab098a`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(v21adapter): add professional readiness report
  - `codex/a21-ws4-professional-v21-bridge` (`acf8530`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(gateway): wire xiaozhi voice pipeline adapters
  - `codex/a21-ws4c-professional-readiness-ingestion` (`26a548e`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): ingest professional readiness reports
  - `codex/a21-ws4d-xiaozhi-professional-voice-bridge` (`c00549c`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(gateway): bridge xiaozhi professional turns
  - `codex/a21-ws4e-xiaozhi-professional-utterance` (`200a85b`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(gateway): derive xiaozhi professional query from ASR
  - `codex/a21-ws4f-xiaozhi-professional-bench` (`4f7f99e`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): add xiaozhi professional bench
  - `codex/a21-ws5-silero-vad-adapter` (`acf8530`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat(gateway): wire xiaozhi voice pipeline adapters
  - `codex/a21-ws5-xiaozhi-device-mcp-schema` (`b937f4b`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(a21): add xiaozhi stackchan device extension schema
  - `codex/a21-ws6-provider-latency-ingest` (`b919878`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): ingest host loopback provider latency reports
  - `codex/a21-ws6-virtual-report-ingest` (`e64104a`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): ingest virtual xiaozhi latency reports
  - `codex/a21-ws6-virtual-xiaozhi-repeat` (`ec8fb96`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(tools): aggregate virtual xiaozhi repeats
  - `codex/a21-ws6-virtual-xiaozhi-report-ingestion` (`2bc37fc`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): ingest virtual xiaozhi latency reports
  - `codex/a21-ws6-xiaozhi-host-local-report-semantics` (`407bcef`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(a21): harden xiaozhi host-local bench semantics
  - `codex/a21-ws6b-product-readiness-host-evidence` (`d822dfc`, ahead=2): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(app): require complete xiaozhi readiness evidence
  - `codex/a21-ws7a-provider-hotplug-proof` (`a996260`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: test(providers): prove hotplug provider smoke route
  - `codex/a21-ws8a-physical-stackchan-evidence-contract` (`3234565`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: test(app): add physical StackChan evidence contract
  - `codex/a21-ws8b-product-readiness-physical-evidence` (`15749b4`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(app): ingest physical stackchan readiness evidence
  - `codex/a21-xiaozhi-audio-quality-20260601` (`8bfed60`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(audio): improve xiaozhi tts downlink clarity
  - `codex/a21-xiaozhi-bench-evidence-hardening` (`771c9fb`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(app): harden xiaozhi bench product-chain evidence
  - `codex/a21-xiaozhi-playback-ack-debug-profile` (`f534f2b`, ahead=2): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: fix(gateway): validate xiaozhi playback stream ids
  - `codex/a21-xiaozhi-protocol-pivot-20260601` (`0056132`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: docs(architecture): archive xiaozhi pivot planning
  - `codex/a21-xiaozhi-protocol-pivot-freeze-20260601` (`614759a`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: docs(architecture): pivot hardware path to xiaozhi protocol
  - `codex/a21-xiaozhi-protocol-pivot-rescue-20260601` (`9ccad4b`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: docs(architecture): archive xiaozhi pivot rescue notes
  - `codex/a21-xiaozhi-protocol-server-20260601` (`0bd0cc5`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(gateway): add xiaozhi websocket compatibility
  - `codex/a21-xiaozhi-provider-pipeline-20260601` (`eda35c2`, ahead=18): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: chore(providers): checkpoint xiaozhi pipeline contract
  - `codex/god` (`8c0600f`, ahead=17): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: merge: a21 half-duplex speaker evidence fix
  - `codex/network` (`d15a7b4`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: feat: promote public voice gateway and wifi provisioning
  - `codex/newworld` (`090eca6`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: test(providers): stabilize realtime fake concurrency
  - `codex/proxy` (`5d3b548`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: chore(network): default make targets to A21 direct no-proxy
  - `codex/stackchan-official-hw-parity-gap-map-001` (`d02a592`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: docs(stackchan): map official hardware parity gaps
  - `codex/stackchan-official-mcp-status-parity-001` (`3ec1540`, ahead=1): delete; no unique product value after frozen `090eca6` baseline audit; no cherry-pick. Last subject: feat(gateway): add xiaozhi mcp status parity
  - `codex/t-hw-volume-001-official-codec-volume` (`f49abde`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: docs(audio): plan stackchan volume control
  - `main` (`4fa66c8`, ahead=0): delete; no commits ahead of `main-lean`; no cherry-pick. Last subject: fix(audio): use remote IndexTTS2 model root and pcm16 output
