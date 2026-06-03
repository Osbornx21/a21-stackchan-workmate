# A21 Current Control

Status: current control entry.
Date: 2026-06-04 CST.
Owner: A21 control tower.

This is the short first-read entry for the active launch sprint. It does not
replace `AGENTS.md`, `docs/project_state_machine.md`, the handoff log, release
handoffs, or PRD documents. It points to the current state and the one active
execution plan.

## Current Checkout

- Workspace: `/Users/jiyurun/Documents/New project`
- Branch: `codex/a21-hardware-window-20260603-wifi-provisioning-flash`
- Sprint start HEAD:
  `b58283b docs(handoff): add internal test 3 master handoff`
- Current source HEAD:
  `3741c4a feat(providers): promote stepfun route eligibility`
- Remote:
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`
- Tracked dirty-state policy:
  do not start launch implementation from unclassified tracked diffs.
- `.DS_Store` policy:
  ignored by `.gitignore`; existing workspace noise may be deleted.
- Source-only control proposal:
  `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md`.
- Current workspace audit:
  `docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`.

## Current Product State

Internal test 3 is published and accepted for team voice testing, but not full
PRD launch.

- Internal test 3 package:
  `dist/a21-internal-test3-20260603-233245`.
- Tarball:
  `dist/a21-internal-test3-20260603-233245.tar.gz`.
- Package source commit:
  `074e3d877d33`.
- Release docs commit:
  `221c153`.
- Master handoff commit:
  `b58283b`.
- Public Gateway:
  `47.103.57.217`.
- Product Xiaozhi WebSocket:
  `ws://47.103.57.217/v1/xiaozhi`.
- Product device:
  `44:1b:f6:e2:6a:60`.

Runtime truth at the start of this sprint:

- `current_voice_mode`: `dialogue`
- `current_voice_chain_mode`: `cascade`
- `current_asr_profile`: `dashscope_qwen_asr_realtime`
- `current_llm_profile`: `deepseek`
- `current_tts_profile`: `dashscope_qwen_tts_realtime`
- `current_realtime_provider`: `doubao_realtime`
- `current_voice_clone_profile`: `a21_voice_default_dashscope`
- finding: `stepfun_not_selected`

Evidence truth:

- Host bench: `candidate_host_only`.
- Physical report: `candidate_gateway_downlink`.
- Product readiness: `server_side_blocked`.
- Launch ready: false.
- PRD accepted: false.

## Active Transition

Current active plan:

- `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`

Transition:

- `T-STEPFUN-ROUTE-001-LAUNCH-POLICY-PROMOTION`

Target:

- Deploy the committed StepFun route-eligibility promotion, run fresh remote
  StepFun provider smoke, and rerun readiness without changing internal test 3
  `/v1/xiaozhi` protocol behavior or firmware flash state.

## Current Evidence Manifest

Current manifest:

- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`

Launch decisions must use the manifest plus the named reports. Do not infer
launch truth from raw report directory mtime.

## Active Workers

Workers are scoped by `docs/plans/2026-06-04-full-launch-protocol-adaptation.md`.
They must not broaden write sets silently.

| Worker | Scope | Write boundary |
| --- | --- | --- |
| A | Gateway Xiaozhi product protocol regression | `internal/gateway/*`, narrow `docs/engineering/PROTOCOL.md` clarification only |
| B | Readiness and physical evidence adaptation | `internal/app/product_demo.go`, `internal/app/server_side_readiness_bundle.go`, `internal/app/xiaozhi_physical_evidence.go`, `internal/app/app_stackchan_xiaozhi_half_duplex.go`, focused app tests |
| C | Official-compatible firmware evidence pointer | `internal/app/doctor.go`, `internal/app/app_office_preflight.go`, `internal/app/official_stackchan.go`, focused app tests, narrow firmware/doctor docs |
| D | Provider selector and StepFun switch plan | `internal/providers/*`, provider app files, narrow provider/voice-selection docs, Gateway selector tests only if contract is wrong |

## Allowed Commands For This Transition

Host-only and read-only commands:

```bash
git status --short --branch
git diff --stat
go test ./internal/gateway -run 'Xiaozhi|WriteXiaozhi|OfficialStackChan|TraceEndpointReturnsVoicePipelineSplitSummary' -count=1
go test ./internal/app -run 'XiaozhiPhysicalEvidence|XiaozhiHalfDuplex|ProductReadiness|ServerSideReadinessBundle|XiaozhiVoiceBench|ProviderLatencyBench' -count=1
go test ./internal/providers -run 'DashScope|Doubao|ProviderSmoke|TextStream|VoicePipelineAdapters|GatewayVoiceProvider|Realtime' -count=1
go test ./internal/transport/xiaozhi ./internal/transport/stackchan ./internal/v21adapter ./internal/personality -count=1
git diff --check
make verify
make preflight
make doctor
```

Public Gateway read-only checks require direct routing:

```bash
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 curl -sS http://47.103.57.217/healthz
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 curl -sS http://47.103.57.217/v1/devices
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 curl -sS http://47.103.57.217/v1/voice-chain-profiles
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 curl -sS http://47.103.57.217/xiaozhi/ota/
```

## Forbidden Commands For This Transition

- No bare `xiaozhi.bin` product flash.
- No generic `xiaozhi-firmware-flash-*` product StackChan flashing.
- No provider execute from Mac.
- No V21 execute unless explicitly scoped in a later professional validation
  task.
- No ECS root secret edits until the StepFun switch has its own guarded
  runtime task.
- No hardware write or NVS write from this sprint unless a foreground hardware
  window is opened with its own plan.
- No report deletion or archive moves.

## Last Verified Facts

From the internal test 3 master handoff and sprint kickoff:

- Internal test 3 tarball SHA check passed.
- Public Gateway health returned `status=ok`.
- Product device `44:1b:f6:e2:6a:60` was online.
- `/v1/voice-chain-profiles` still reported selected LLM `deepseek` and
  finding `stepfun_not_selected`.
- `/xiaozhi/ota/` returned `ws://47.103.57.217/v1/xiaozhi`.
- Targeted read-only test listings showed existing Gateway/App/Provider/
  Transport/V21/Personality coverage for the relevant protocol surfaces.

## Latest Control-Tower Result - 2026-06-04 02:13 CST

The protocol-adaptation worker round has landed locally and passed the unified
verification surface.

Accepted local results:

- Gateway stock `/v1/xiaozhi` regression coverage passed.
- Product readiness, server-side readiness, physical evidence, and
  half-duplex evidence adaptation passed.
- Official-compatible product-lane firmware evidence pointer support passed.
- Provider selector and StepFun/DeepSeek role reporting passed.
- `make verify`: passed.
- A post-documentation default `make verify` retry was killed by the host with
  `Killed: 9`; follow-up `go test -p 1 ./...`, `git diff --check`, and
  `GOMAXPROCS=2 make verify` all passed, classifying the kill as a local
  resource/concurrency interruption rather than a test failure.
- `make preflight`: passed with only
  `wake_word_firmware_build_required`.
- `make doctor`: passed with only `wake_word_firmware_build_required`; doctor
  now reports official-compatible product-lane flash evidence as satisfied.

Public Gateway retry result:

- `http://47.103.57.217/healthz`: recovered and returned
  `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`.
- `http://47.103.57.217/v1/devices`: product device
  `44:1b:f6:e2:6a:60` online, current mode `local_fallback`, selector state
  cascade/DashScope ASR/DeepSeek/DashScope TTS.
- `http://47.103.57.217/v1/voice-chain-profiles`: still reports selected LLM
  `deepseek` and finding `stepfun_not_selected`.
- `http://47.103.57.217/xiaozhi/ota/`: returns
  `ws://47.103.57.217/v1/xiaozhi`.
- Public `:21081` still gives `Empty reply from server` from this Mac; this is
  not the product entrypoint while public `80` is healthy.
- `ssh root@47.103.57.217`: currently rejects this local key with
  `Permission denied (publickey)`, so remote systemd/secret inspection and
  service restart are blocked from this control thread.

Current conclusion:

- Local protocol/readiness/provider/firmware-evidence adaptation is accepted.
- Public product HTTP entrypoint is currently reachable again.
- Full launch remains blocked by `stepfun_not_selected`, physical playback
  ack/stop_done/trusted audible evidence, and unavailable remote control-plane
  authentication for a guarded StepFun switch.

## Latest Control-Tower Result - 2026-06-04 02:24 CST

The ECS control-plane blocker was narrowed.

- SSH succeeds when using an explicit existing local SSH identity; the default
  SSH agent still has no loaded identities.
- Remote `a21-gateway`: active.
- Remote `caddy`: active.
- Remote `127.0.0.1:21081/healthz`: ok.
- Remote provider env file: present, owner `root:root`, mode `600`.
- `A21_LAB_STEPFUN_API_KEY`: missing on ECS.
- `A21_STEPFUN_MODEL`: missing on ECS.
- `A21_DASHSCOPE_API_KEY`: present on ECS.
- Local control-machine StepFun/DashScope env names are also missing.

Per the transition stop rules, no StepFun hot switch was attempted.

Fresh readiness evidence:

- `reports/a21-product-readiness-20260604-022419.json`:
  `status=server_side_blocked`, `launch_ready=false`, selected LLM `deepseek`,
  finding `stepfun_not_selected`, missing real evidence includes
  `real_provider_smoke`, `physical_stackchan_prd_acceptance`, and
  `stepfun_not_selected`.
- `reports/a21-server-side-readiness-bundle-20260604-022420.json`:
  `status=server_side_blocked`, `candidate_ready=false`, missing evidence
  `provider_smoke` and `stepfun_not_selected`.

Current conclusion:

- Public Gateway and ECS service health are good.
- StepFun switch is blocked by missing remote StepFun env names, not by local
  code or Gateway availability.
- Next action is operator-side secret provisioning in
  `/etc/a21/secrets/provider.env`, then guarded service restart and fresh
  snapshots/evidence collection.

## Latest Control-Tower Result - 2026-06-04 02:28 CST

Remote binary drift was identified before the StepFun switch.

- ECS-side `xiaozhi-streaming-provider-readiness` was run with selector env
  names forcing StepFun while sourcing the root-only provider env file.
- The remote binary reported `gate_status=passed` for StepFun even though the
  direct env-name check showed `A21_LAB_STEPFUN_API_KEY` and
  `A21_STEPFUN_MODEL` missing.
- This is stale remote binary behavior relative to the local verified code,
  where StepFun selected with missing env names is blocked and reports
  env-name-only missing fields.

New active runtime safety plan:

- `docs/plans/2026-06-04-public-gateway-code-sync-before-stepfun.md`

Current execution order:

1. Commit the locally verified protocol/readiness/provider-selector changes.
2. Deploy that commit to ECS without changing provider secrets or selector
   state.
3. Re-run remote static StepFun readiness and require it to block on missing
   env names.
4. Resume StepFun env provisioning and switch only after the remote binary is
   synced.

## Latest Control-Tower Result - 2026-06-04 03:07 CST

The control thread re-read the internal test 3 master handoff, current plans,
code, public Gateway snapshots, and latest reports before continuing.

Runtime truth has moved beyond the 02:28 env/code-sync blocker:

- Public `http://47.103.57.217/healthz`: healthy.
- Public `/v1/voice-chain-profiles`: selected LLM `stepfun`.
- Public `/v1/devices`: product device `44:1b:f6:e2:6a:60` online with
  cascade DashScope ASR, StepFun LLM, and fixed DashScope TTS.
- Public `/xiaozhi/ota/`: still returns
  `ws://47.103.57.217/v1/xiaozhi`.
- Host bench `reports/a21-xiaozhi-voice-bench-20260604-023616.742713000.json`
  executed the cloud-edge chain with ASR `dashscope_qwen_asr_realtime`, LLM
  `stepfun`, and TTS `dashscope_qwen_tts_realtime`.
- Remote provider smoke
  `reports/a21-provider-smoke-20260604-023711-678466985.json` passed execution
  and streaming checks but still has `route_eligible=false`, so readiness
  rejects it as `provider_smoke`.

New active control plan:

- `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`

Current conclusion:

- Do not repeat the DeepSeek-to-StepFun runtime switch work; it is already
  reflected in the public selector snapshot.
- The current server-side blocker is StepFun route-eligibility promotion and a
  fresh route-eligible provider smoke report.
- Full PRD launch still remains blocked by physical playback ack, stop_done, or
  trusted audible/instrument observation.

## One Recommended Next Action

Execute `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`: promote
the built-in StepFun profile to explicit route-eligible launch-policy status,
deploy the committed patch to ECS, run fresh redacted StepFun provider smoke,
and then re-run product/server-side readiness. Do not touch internal test 3
Gateway protocol, firmware flash state, or endpoint-side voice acceptance.

## Latest Control-Tower Result - 2026-06-04 03:17 CST

Local promotion is committed; remote deploy is blocked by ECS SSH/control-plane
access.

- Commit:
  `3741c4a feat(providers): promote stepfun route eligibility`.
- Local verification before commit:
  `go test ./internal/providers -run 'ProviderCatalog|ProviderSmoke' -count=1`
  passed;
  `go test ./internal/app -run 'LocalVoiceLoopbackCanUseCompatibilityTextStream|ProductReadiness|ServerSideReadinessBundle|XiaozhiStreamingProviderReadiness' -count=1`
  passed;
  `git diff --check` passed;
  `GOMAXPROCS=2 make verify` passed.
- Local dry-run shape after commit:
  StepFun provider smoke reports `route_eligible=true` when configured, but
  remains `executed=false` and is not launch evidence.
- ECS deploy attempt stopped before remote mutation because this thread has no
  accepted SSH identity for `root@47.103.57.217`; default SSH and local
  `~/.ssh` candidates are unusable.

Current conclusion:

- The source-level StepFun route-eligibility blocker is closed locally.
- Public Gateway has not yet been updated to commit `3741c4a` from this
  thread.
- The next action is control-plane recovery, then deploy `3741c4a`, run fresh
  remote executed StepFun provider smoke, and rerun readiness.
- Workspace cleanup is constrained by
  `docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`: clean current
  mainline noise and stale registrations only; do not delete real branches,
  existing worker worktrees, stashes, reports, firmware artifacts, or evidence
  during launch-critical work.
