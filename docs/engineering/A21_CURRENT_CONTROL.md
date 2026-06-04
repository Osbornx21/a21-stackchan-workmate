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
- Current source HEAD before this workspace voice/wake-control cut:
  `b93d820 feat(gateway): add workspace roleplay controls`
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
- Current integration audit:
  `docs/engineering/A21_INTEGRATION_AUDIT_20260604.md`.

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

- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`

Transition:

- `T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001`

Target:

- Advance internal test 4 without regressing internal test 3: make `roleplay`
  and `professional` the two user-facing modes, keep `dialogue` as a
  compatibility alias, and define the A21 Cloud/Web/App plus V21 Knowledge
  Service/Adapter shape for public-only and personal+public professional query.

Current focused cut:

- Host-local workspace voice/wake-control plan:
  `docs/plans/2026-06-04-workspace-voice-chain-wake-control-surface.md`
- Transition:
  `T-WORKSPACE-VOICE-CHAIN-WAKE-CONTROL-SURFACE-001`
- Target:
  let `/workspace` configure provider voice-chain selection and wake-word
  intent through existing Gateway contracts while avoiding provider/V21
  execution, real indexing, firmware build, serial, NVS, or physical hardware
  action.

## Scoped Hardware Parity Transition

Transition:

- `T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001`

Plan:

- `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`

Status:

- Gap map frozen in
  `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`.
- This is a docs/state baseline only. It does not expose new Gateway controls,
  start Gateway, execute providers or V21, build firmware, flash firmware,
  touch serial, or write NVS.
- Low-risk MCP/status parity has also landed as a Gateway contract through
  `POST /v1/xiaozhi/mcp-control` for only
  `self.audio_speaker.set_volume`, `self.get_device_status`,
  `self.screen.set_brightness`, `self.screen.set_theme`, and
  `self.screen.get_info`.
- Status-display registry parity has landed as Gateway/protocol state:
  `/v1/devices` now carries normalized `display_state` metadata and
  trace/session linkage from stock Xiaozhi turn states or A21 device events.
  This is not physical screen acceptance; `display_state_physical_accepted`
  remains false until a separate hardware evidence report exists.
- Official avatar/action parity has landed as a host/Gateway mapping contract:
  `BuildOfficialActionPlan` maps state/face/motion to official
  `ControlAvatar`, `ControlMotion`, or `DanceSequence` packets plus redacted
  action metadata. `servo_x` remains candidate-only and RGB is
  `*_no_rgb_frame` metadata until a hardware evidence window proves more.
- Official StackChan/Xiaozhi capabilities remain reference material. A21
  product availability still requires A21 evidence and the promotion gates in
  the capability charter.

Current official-source reference:

- StackChan root:
  `da156e1fa0e1c2a5e00b78fbf69b1f7e7bca0483`, dirty working tree, read as
  working-tree reference.
- Xiaozhi sub-tree:
  `e77dedb1309153bb63fed285772962c920c97dd4`, detached clean `HEAD`, read as
  `HEAD` reference.

Next operator/control action:

- Dispatch `T-STACKCHAN-OFFICIAL-ACTION-PHYSICAL-EVIDENCE-001` only in an
  approved foreground hardware window for touch/barge-in/visible action
  evidence.
- Keep reboot, upgrade, camera/photo, screen snapshot, camera stream/video,
  NFC, infrared, app lifecycle, firmware, flash, serial, and NVS out of the
  next worker.

## Current Evidence Manifest

Current manifest:

- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`

Launch decisions must use the manifest plus the named reports. Do not infer
launch truth from raw report directory mtime.

## Active Workers

No active worker writer is currently authorized.

Recent thread control snapshot:

- A21 server mainline control thread:
  `019e7f81-e218-7df1-8743-1ed66e7ddd37`, status `idle`.
- A21 hardware control thread:
  `019e8873-52f0-7570-8efd-04b899db7d4e`, status `notLoaded`.
- Historical worktrees remain registered for evidence and branch history, but
  they are not the active write surface for this transition.
- Do not delete old worktrees, prune loose objects, or reinterpret old worker
  outputs as current evidence during the launch-critical path.

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
- No provider execute from Mac except the already scoped StepFun evidence
  recorded in the manifest.
- No V21 execute unless the adapter boundary is configured and this active
  professional validation plan's redaction/contract rules are followed.
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
- Integration cleanup/review is constrained by
  `docs/engineering/A21_INTEGRATION_AUDIT_20260604.md`: internal test 3 commits
  are confirmed ancestors of current `HEAD`; do not perform broad revert or
  branch cleanup that could erase accepted endpoint-side protocol progress.

## Latest Control-Tower Result - 2026-06-04 03:33 CST

The current mainline commits are now pushed to the remote tracking branch, but
the ECS runtime remains blocked by control-plane and public Gateway health.

- Pushed to
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`:
  `765ed41`, `3741c4a`, `8396261`, and `5dba606`.
- `git branch -r --contains` confirms those commits are now present on the
  remote tracking branch.
- Public TCP ports `22`, `80`, `443`, and `21081` on `47.103.57.217` accept
  connections.
- Public HTTP paths `/healthz`, `/v1/devices`, `/v1/voice-chain-profiles`, and
  `/xiaozhi/ota/` currently return empty HTTP replies from this control Mac.
- Default SSH remains unusable for `root@47.103.57.217`; no ECS files,
  services, secrets, selectors, firmware, flash state, or NVS were mutated.

Current conclusion:

- Repository integration is now recovered on the remote branch.
- ECS deploy of `3741c4a` or newer is still pending.
- Public Gateway health is now an active runtime blocker that requires ECS
  control-plane access or an approved operator on the host.
- Do not rerun or reinterpret older provider/readiness reports as launch
  evidence while the public Gateway is returning empty replies.

## Latest Control-Tower Result - 2026-06-04 04:02 CST

The ECS runtime blocker is resolved for the A21 launch path, and the StepFun
route-eligibility transition has moved from provider blocker to V21/physical
blocker.

- Deployed to ECS through the approved jump path and `/opt/a21.next` safe swap:
  `d9362a7 feat(readiness): accept cloud-edge xiaozhi evidence`.
- Remote focused tests before swap:
  `go test ./internal/app -run 'ProductReadiness|ServerSideReadinessBundle|XiaozhiVoiceBench|ProviderLatency' -count=1`
  passed;
  `go test ./internal/providers -run 'ProviderCatalog|ProviderSmoke' -count=1`
  passed.
- Remote `a21-gateway`: active.
- Remote Caddy: active.
- Remote `127.0.0.1:21081/healthz`: ok.
- Remote selector after restart: cascade DashScope ASR, StepFun LLM, fixed
  DashScope TTS, `findings=null`.
- ECS provider env file remains root-owned and mode `600`; only A21 profile
  selector IDs were changed, not provider secret values.
- Fresh StepFun provider smoke:
  `reports/a21-provider-smoke-20260604-035200-977132343.json`, `passed`,
  `executed=true`, `stream=true`, `route_eligible=true`, no fallback.
- Fresh static Xiaozhi provider readiness:
  `reports/a21-xiaozhi-streaming-provider-readiness-20260604-035317-1780516397705439206.json`,
  `gate_status=passed`, `stepfun_selected`.
- Fresh cloud-edge Xiaozhi host bench:
  `reports/a21-xiaozhi-voice-bench-20260604-035502.222374820.json`,
  `candidate_host_only`, `cloud_edge`, `failure_count=0`,
  answer first-audio p95 `809 ms`, barge-in stop p95 `0 ms`.
- Fresh product readiness:
  `reports/a21-product-readiness-20260604-040153.json`,
  `server_side_blocked`, with provider evidence and host voice loopback ready.
- Fresh server-side readiness bundle:
  `reports/a21-server-side-readiness-bundle-20260604-040207.json`,
  `server_side_blocked`, missing `v21_professional_smoke`.

Current conclusion:

- StepFun route eligibility and cloud-edge host voice evidence are now accepted
  by readiness.
- Full launch remains blocked by `v21_professional_execution` and
  `physical_stackchan_prd_acceptance`.
- Direct curls from this control Mac to `47.103.57.217` still return empty
  replies, while ECS loopback and 5080lab public HTTP are healthy. Treat this as
  a source-path/network issue to inspect in Aliyun/network tooling, not as an
  A21 Gateway runtime blocker.

## Latest Control-Tower Result - 2026-06-04 04:14 CST

The StepFun cloud-edge server-side slice is accepted at source and ECS runtime,
and the active server-side gap is now V21 professional execution.

Current control facts:

- Current branch is clean and synced to
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash` at
  `20d11a0`.
- A21 thread inspection shows no active worker writer.
- Registered historical worktrees are not being cleaned or pruned.
- Fresh StepFun provider and cloud-edge host voice reports remain the current
  server-side evidence.
- `A21_V21_ADAPTER_URL`, `A21_V21_BACKEND_URL`, and
  `A21_V21_ADAPTER_TOKEN` are missing in the current shell.
- Local `127.0.0.1:21121` is not listening.
- Existing V21 reports are old 2026-06-01/02 evidence and must not be used to
  close the 2026-06-04 V21 gate without a fresh adapter run.
- Fresh V21 dry-run report:
  `reports/a21-v21-adapter-smoke-20260604-041657.json`, `skipped`,
  `configured=false`, `executed=false`, `redaction_ok=true`.
- Fresh local collector reports
  `reports/a21-product-readiness-20260604-041710.json` and
  `reports/a21-server-side-readiness-bundle-20260604-041710.json` are
  control-shell blocker evidence only; they do not supersede the 04:02 ECS
  StepFun/cloud-edge canonical evidence because the local shell has no live
  Gateway/provider selector context.

Active transition:

- `docs/plans/2026-06-04-v21-professional-execution-validation.md`
- `T-V21-PROFESSIONAL-EXECUTION-001`

Current conclusion:

- Server-side readiness is ready for V21 adapter evidence, but V21 is blocked
  on a missing adapter/backend boundary from this control shell.

## Latest Control-Tower Result - 2026-06-04 04:36 CST

The V21 professional execution gate is closed for local adapter-boundary
evidence.

What happened:

- Read V21 control thread
  `codex://threads/019e68bc-4fb6-7ce0-ad67-5b1dd0de478f`.
- Confirmed V21 is currently operated as a local/LAN Docker Compose service.
- Started Docker Desktop and brought up the V21 LAN demo backend from
  `/Users/jiyurun/Documents/v21-knowledge-platform`.
- Verified V21 health:
  - `127.0.0.1:18081/api/v1/healthz`: ok.
  - `192.168.1.20:18081/api/v1/healthz`: ok.
- Verified V21 runtime is configured for retrieval and LLM.
- Verified V21 active collection discovery has an active release.
- Temporarily started A21 adapter bridge:
  `127.0.0.1:21121 -> 127.0.0.1:18081`.
- Ran fresh executed adapter smoke:
  `reports/a21-v21-adapter-smoke-20260604-043456.json`, `passed`,
  `executed=true`, `redaction_ok=true`.
- Ran fresh readiness collectors:
  `reports/a21-product-readiness-20260604-043528.json` and
  `reports/a21-server-side-readiness-bundle-20260604-043528.json`.

Current evidence truth:

- Provider evidence: ready from
  `reports/a21-provider-smoke-20260604-035200-977132343.json`.
- V21 professional evidence: ready from
  `reports/a21-v21-adapter-smoke-20260604-043456.json`.
- Host voice evidence: ready as cloud-edge candidate from
  `reports/a21-xiaozhi-voice-bench-20260604-035502.222374820.json`.
- The local collector remains `server_side_blocked` because this shell has no
  live A21 Gateway, wake-word status, or voice-chain selector context.

Current conclusion:

- V21 is no longer the active server-side evidence blocker.
- This is local adapter-boundary execution evidence, not a permanent ECS
  Gateway V21 topology.
- Full PRD remains blocked by physical StackChan acceptance and current live
  Gateway/wake/selector evidence refresh.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Start

The active product direction has moved from internal test 3 evidence closure to
internal test 4 PRD completion.

Current internal test 4 decisions:

- User-facing modes are now `roleplay` and `professional`.
- `roleplay` is the default embodied mode for personality, memory hints,
  role-play playbooks, voice-clone selection, low-latency speech, and Xiaozhi
  playback.
- `professional` remains the only mode allowed to call the A21/V21 adapter.
- `dialogue`, `workmate`, `companion`, and `co_creation` are compatibility
  aliases or lower-level playbook labels under `roleplay`.
- Mode switching must be user-initiated by voice, touch, app, or web control.
- A21 Cloud/Web/App is the target workspace surface for upload, indexing,
  public-only query, personal-only query, personal+public query, and device
  binding.
- V21 should evolve into a Knowledge Service/Adapter boundary with scoped
  `workspace_id`, `user_id`, and `query_scope` fields; A21 must not read V21
  internals or store uploaded documents on StackChan.

First code/doc cut in progress:

- Plan:
  `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`.
- PRD v0.5 mode/workspace update.
- Protocol mode contract update.
- Gateway `/v1/voice-modes` catalog defaults to `roleplay`, lists
  `roleplay`/`professional`, and accepts old `dialogue` input as a
  compatibility alias.
- `/v1/voice-modes` now carries `selected_ritual` and per-mode `ritual`
  metadata. Selecting `professional` returns `screen_label=PRO`, the shared
  evidence-first checking cue, `expression=professional`, trace marker
  `professional.checking_feedback.sent`, `workspace_policy=professional_only`,
  `v21_allowed=true`, and `physical_accepted=false`.
- Fast companion accepts `roleplay` while still rejecting selected
  `professional` before provider or V21 execution.

Guardrails:

- Do not regress internal test 3 Xiaozhi audio state machine, StepFun/DashScope
  selector evidence, V21 adapter evidence, or product firmware lane.
- Do not claim internal test 4 cloud workspace readiness until upload/index
  scope, adapter v2, and hardware professional consult evidence are separately
  proven.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Roleplay Runtime Slice

The first internal test 4 runtime cut now moves beyond mode naming into a
Gateway-owned roleplay profile contract.

Current implementation state:

- Gateway exposes `GET/POST/PUT /v1/roleplay-profile` with schema
  `a21.gateway.roleplay_profile.v1`.
- The runtime selector carries `roleplay_profile`, scenario/playbook, and
  `voice_clone_profile`.
- Selecting a roleplay voice clone also updates the existing voice-chain
  selector, so the roleplay path reaches the selected TTS/voice-clone boundary
  without adding a second provider control plane.
- Fast companion roleplay turns return a redacted roleplay runtime summary and
  trace `roleplay.profile.ready`; memory readiness adds
  `roleplay.memory.ready`.
- The runtime summary reports prompt composition readiness and memory hint
  counts, but keeps prompt text, memory text, transcripts, provider output,
  voice-clone samples, and V21 evidence out of responses and traces.

Current conclusion:

- `roleplay` is now the concrete default embodied/personality lane for internal
  test 4, with scenario and voice-clone selection exposed to the simulator.
- `professional` remains separate and still cannot execute through
  fast-companion/dialogue endpoints.
- Cloud upload/index/query-scope and hardware professional consult remain
  planned work, not readiness claims.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Roleplay Memory Control

The roleplay runtime now has a product-visible memory control surface without
turning memory into persisted logs or V21 context.

Current implementation state:

- `POST`/`PUT /v1/roleplay-profile` accepts `memory_hints` and
  `clear_memory`.
- Runtime hints are sanitized through the existing personality memory policy:
  unsafe URLs, local paths, and credential-looking strings are rejected; long
  hints are bounded before prompt use.
- Gateway stores only sanitized runtime hint text in memory and returns only
  readiness, counts, and finding codes.
- Fast companion roleplay turns can use the configured hint count while still
  reporting `prompt_stored=false`, `memory_text_stored=false`,
  `professional_route_allowed=false`, and `v21_executed=false`.
- Simulator exposes a one-hint roleplay memory control and clear button, but
  displays only memory status/count.

Current conclusion:

- Roleplay now covers the PRD's first user-visible persona/memory/voice-clone
  control loop.
- This is not durable long-term memory, document upload, personal workspace
  indexing, provider transcript storage, or V21 professional retrieval.

## Latest Control-Tower Result - 2026-06-04 Roleplay Prompt Enters Voice Pipeline

The roleplay lane now carries the selected personality/scenario/memory prompt
into the low-latency text-stream provider boundary instead of stopping at
configuration readiness.

Current implementation state:

- `VoicePipelineRequest` has a redacted runtime-only `TextPrompt` field.
- Voice pipeline text-stream adapters receive `TextPrompt` when present;
  reports continue to record ASR/transcript counts separately and do not store
  prompt text.
- `VoicePipelineReport` exposes only `prompt_input_ready` and
  `prompt_input_not_recorded`.
- Fast-companion roleplay turns with PCM frames compose the selected roleplay
  prompt and pass it into the voice pipeline.
- Stock `/v1/xiaozhi` roleplay turns also compose the selected roleplay prompt
  before the text-stream provider boundary.
- Gateway traces only `roleplay.prompt_input.used`; prompt bodies, memory text,
  transcripts, and provider output remain unrecorded.

Current conclusion:

- Roleplay persona and memory can now affect actual voice answers on the
  low-latency provider path, not only simulator/profile summaries.
- This is still host/runtime evidence. Physical StackChan roleplay acceptance
  requires an operator-triggered stock `/v1/xiaozhi` turn and audible/provider
  evidence.

## Latest Control-Tower Result - 2026-06-04 Roleplay Voice Clone Enters TTS Boundary

The roleplay lane now carries the selected safe voice-clone profile into the
provider-neutral voice pipeline and TTS adapter request instead of stopping at
the selector/catalog layer.

Current implementation state:

- `VoicePipelineRequest` has a redacted runtime-only `VoiceCloneProfile`.
- `TTSAdapterRequest` carries the same safe A21 voice profile ID.
- Local TTS adapters pass the safe profile through `LocalTTSOptions.Voice`, so
  `voice_clone_cli` can bind the turn to the selected voice identity without
  exposing reference audio/text paths.
- `VoicePipelineReport.input.voice_clone_profile` exposes only safe A21
  profile IDs, and report redaction includes
  `voice_clone_sample_not_recorded`.
- Fast-companion roleplay turns with PCM frames and stock `/v1/xiaozhi`
  roleplay turns both pass the selected profile to the voice pipeline.
- Gateway traces only `roleplay.voice_clone_profile.used` when a clone profile
  is selected; it does not trace samples, prompt bodies, memory text,
  transcripts, provider output, local paths, URLs, or credentials.

Current conclusion:

- Roleplay voice selection now has runtime evidence at the TTS boundary:
  selected profile -> voice pipeline request -> TTS request -> redacted report.
- This still does not execute a real clone provider or prove physical
  StackChan roleplay audio. Provider execution and physical acceptance remain
  separate gates.

## Latest Control-Tower Result - 2026-06-04 Professional Mode Ritual Contract

Professional mode selection now has a first-class switching ritual contract
instead of only changing an enum.

Current implementation state:

- `GET/POST /v1/voice-modes` returns `selected_ritual` and per-mode `ritual`
  metadata.
- The professional ritual uses screen label `PRO`, cue
  `我在查，先把证据和置信度拉出来。`, expression `professional`, trace marker
  `professional.checking_feedback.sent`, workspace policy
  `professional_only`, `v21_allowed=true`, and `physical_accepted=false`.
- The simulator displays the selected mode cue through `modeRitualReadout`.
- Fast companion remains blocked when the selected voice mode is
  `professional`; no provider or V21 execution occurs from that endpoint.

Current conclusion:

- Web/app/hardware controls now have a shared redacted ritual payload for
  entering professional mode. Actual professional answers and evidence still
  require the existing professional/V21 path, and physical StackChan
  acceptance remains separate.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Professional Workspace Contract

The next internal test 4 cut gives professional mode an explicit workspace and
query-scope contract without implementing cloud upload/index execution yet.

Current implementation state:

- Gateway exposes `GET/POST/PUT /v1/professional-workspace` with schema
  `a21.gateway.professional_workspace.v1`.
- The selected professional context carries redacted `user_id`,
  `workspace_id`, and `query_scope`.
- Supported contract scopes are `public_only`, `personal_only`, and
  `personal_plus_public`.
- Professional turns now send `device_id`, `user_id`, `workspace_id`, and
  `query_scope` to the A21/V21 adapter contract v2.
- Gateway traces only `professional.workspace.ready` and
  `professional.query_scope.<scope>`; it does not trace user/workspace labels
  or utterance text.
- V21 adapter responses and smoke reports can carry redacted
  `source_scope_counts` and `workspace_status`.

Current conclusion:

- A21 now has the no-execute professional workspace/query-scope API contract
  needed for cloud/app and V21-side workers.
- Upload/import/index job APIs, durable account binding, and personal corpus
  enforcement remain separate unshipped slices.
- Full PRD readiness remains blocked by cloud workspace execution, hardware
  professional consult evidence, and physical StackChan acceptance.

## Latest Control-Tower Result - 2026-06-04 V21 Native Scope Worker Completed

The V21 scoped worker completed the first native A21 v2 voice-query contract
cut in the V21 repository.

Current implementation state:

- V21 worker branch:
  `origin/codex/a21-v2-workspace-query-scope-native-contract`.
- V21 worker commit:
  `ad61246 feat(voice-query): accept A21 workspace scope metadata`.
- V21 `/internal/v1/knowledge/voice-query` now natively accepts
  `device_id`, `user_id`, `workspace_id`, and `query_scope`.
- Valid V21 `query_scope` values match A21 internal test 4:
  `public_only`, `personal_only`, and `personal_plus_public`.
- V21 responses now include safe metadata fields `source_scope_counts` and
  `workspace_status`.
- Because current V21 schema cannot yet prove personal/public ACL enforcement,
  the worker truthfully returns `workspace_status=scope_contract_ready_acl_pending`
  and zero classified `source_scope_counts` instead of claiming personal corpus
  search is enforced.

Current conclusion:

- The A21/V21 adapter contract is now aligned on request/response shape at the
  V21 worker branch level.
- V21 merge/release and an ACL/schema ADR remain required before A21 can claim
  personal/public corpus enforcement or nonzero personal/public scope counts.
- The worker did not touch A21 code, A21 Gateway runtime, StackChan hardware,
  firmware, provider APIs, or the dirty V21 LAN/desktop connector work.

## Latest Control-Tower Result - 2026-06-04 V21 Scope Retrieval Guard Completed

The next V21 scoped worker moved the native A21 v2 bridge from metadata-only
acceptance to executable source-scope filtering.

Current implementation state:

- V21 worker branch:
  `origin/codex/a21-v2-workspace-scope-retrieval-guard`.
- V21 worker commit:
  `fccd0ac feat(voice-query): enforce A21 workspace source scope`.
- V21 retrieval requests now carry safe A21 v2 metadata:
  `device_id`, `user_id`, `workspace_id`, and `query_scope`.
- V21 retrieval results and voice-query evidence now carry safe
  `source_scope=public|personal` labels.
- `public_only`, `personal_only`, and `personal_plus_public` are filtered
  before answer generation; unclassified evidence is not promoted into scoped
  A21 responses.
- V21 HTTP retrieval adapter and Qdrant sidecar both pass through
  `source_scope`; the Postgres retrieval writer can derive scope from chunk
  metadata, source-unit locator metadata, or `object_refs.access_class`.
- Scoped responses can report `workspace_status=searchable` with classified
  `source_scope_counts`; legacy requests without `query_scope` still keep
  `scope_contract_ready_acl_pending` and zero claimed scope counts.

Verification:

- V21 `go test ./...` from `backend/`: passed.
- V21 sidecar `python3 -m unittest retrieval_service_test.py`: passed.
- V21 retrieval eval
  `python3 -m unittest evals/retrieval/test_qdrant_retrieval_sidecar.py`:
  passed.
- V21 `git diff --check`: passed.
- V21 `make compose-config`: passed.
- V21 `make test` and `make check`: blocked at `check-toolchain` because the
  Makefile expects Node `v24.15.0` and this shell has Node `v25.8.0`.

Current conclusion:

- A21 can now treat the V21 worker branch as source-scope guard evidence, not
  merely shape evidence.
- This still does not complete durable tenant/account ACL, real personal
  upload indexing, cloud storage, V21 merge/release, or physical StackChan
  professional consult acceptance.
- The worker did not start V21/A21 services, execute providers, ingest private
  documents, touch ECS, firmware, serial, NVS, or the dirty V21 LAN/desktop
  connector worktree.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Workspace Job Skeleton

The workspace surface now has a no-execute upload/import/index job contract.

Current implementation state:

- Gateway exposes `GET/POST/PUT /v1/workspace-upload-jobs` with schema
  `a21.gateway.workspace_upload_jobs.v1`.
- `POST` creates a redacted metadata job for `upload` or `import` source kinds.
- `GET` polls all jobs or a single `job_id`.
- `PUT` supports `mark_failed`, `retry`, and `delete`.
- Jobs store only redacted labels, source scope, content type, size, status,
  attempt count, and trace/session/device IDs.
- Jobs explicitly report `accepted_no_execute`,
  `not_started_no_execute`, and redaction flags; no document text, bytes,
  base64 payload, import URL, local path, credential, or provider output is
  accepted.
- Simulator exposes a `Workspace Job` button that creates a no-execute metadata
  job from the current professional query-scope context.

Current conclusion:

- The A21 Cloud/Web/App workspace API skeleton is now present for upload/import
  job lifecycle UX and worker integration.
- Real upload storage, indexing execution, delete propagation, source ACLs,
  durable user/device binding, and V21 personal/public corpus enforcement
  remain separate unshipped slices.

## Latest Control-Tower Result - 2026-06-04 Professional Workspace Read Records

Professional evidence turns now leave a safe read ledger for upload/read-record
discipline without storing query text or retrieved content.

Current implementation state:

- Gateway exposes `GET /v1/professional-read-records` with schema
  `a21.gateway.professional_read_records.v1`.
- Mock professional turns and stock Xiaozhi professional turns start a
  memory-only read record before V21 query and complete or fail it with safe
  status/scope/count/timing metadata.
- Records store trace/session/device IDs, redacted user/workspace labels,
  query scope, privacy scope, latency profile, answer style, utterance length
  bucket, source-scope counts, workspace status, and redaction flags only.
- Traces add `professional.read_record.started`,
  `professional.read_record.completed`, and
  `professional.read_record.failed`.

Current conclusion:

- Internal test 4 now has an auditable professional-read metadata surface.
- The ledger is memory-only; durable audit storage and real cloud document
  indexing remain separate work.

## Latest Control-Tower Result - 2026-06-04 Simulator Workspace Audit Surface

The simulator now exposes the workspace/read-record metadata operators need for
internal test 4 checks.

Current implementation state:

- The simulator has a `Workspace Audit` section with a no-execute workspace job
  action and a `Read Records` refresh action.
- The surface shows last workspace job status plus read-record count, status,
  failure code, query scope, utterance bucket, source-scope counts, workspace
  status, and privacy scope.
- Professional evidence responses refresh read records for the current trace.
- The simulator displays metadata only, not raw queries, retrieved text,
  evidence bodies, provider output, document text, URLs, paths, credentials,
  voice transcripts, or audio.

Current conclusion:

- The internal-test operator surface can inspect upload/read discipline without
  leaving the safe metadata boundary.
- Browser screenshot tooling was unavailable in that round; local HTTP smoke
  covered the served page and endpoint presence.

## Latest Control-Tower Result - 2026-06-04 Professional Voice Trigger Route

Professional mode can now be entered from speech by explicit user trigger
phrases instead of only API/app selection.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-professional-voice-trigger-route.md`.
- `/v1/mock-turn` routes default/roleplay/workmate/companion trigger text such
  as "认真查一下" into `professionalTurnResponse`.
- Stock `/v1/xiaozhi` default `realtime`/workmate turns route to
  professional when the streaming ASR final contains an explicit trigger such
  as "给我证据".
- Xiaozhi triggered professional turns reuse the streaming ASR final and avoid
  a duplicate batch ASR pass.
- Negated phrases such as "不要进专业检索", "不用专业模式", and
  "别查 V21" do not trigger V21.
- Trace markers are redacted:
  `professional.voice_trigger.detected` and
  `xiaozhi.professional_route.voice_trigger`.

Current conclusion:

- The PRD's user-initiated mode switch now has a Gateway route on both the mock
  control path and the stock Xiaozhi socket.
- This is host-local Gateway evidence only. It does not execute real provider
  or real V21, start Gateway as a runtime service, build/flash firmware, write
  serial/NVS, or prove physical StackChan professional consult acceptance.

## Latest Control-Tower Result - 2026-06-04 Workspace Source Readiness Registry

The workspace surface now has a memory-only source/readiness registry layered
on top of the no-execute upload/import job contract.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-source-readiness-registry.md`.
- Gateway exposes `GET /v1/workspace-sources` with schema
  `a21.gateway.workspace_sources.v1`.
- Creating a workspace upload/import job now creates a linked redacted
  `source_id` with `readiness=metadata_only` and
  `index_status=not_started_no_execute`.
- `PUT /v1/workspace-upload-jobs` accepts
  `mark_searchable` / `mark_indexed_metadata_only` and marks the source as
  `searchable_metadata_only` without real indexing.
- `mark_failed`, `retry`, and `delete` keep source readiness in sync; deletion
  leaves only a redacted tombstone.
- `/v1/professional-workspace` runtime now includes source-scope counts,
  searchable source-scope counts, and query-scope readiness while keeping
  `v21_execution_allowed=false`.
- The simulator Workspace Audit surface can refresh source readiness and show
  source count plus readiness summary.
- Trace markers remain metadata-only:
  `workspace.source.created_metadata_only` and
  `workspace.index_job.searchable_metadata_only`.

Current conclusion:

- Internal test 4 can now show "uploaded/imported source exists" and
  "searchable metadata candidate" per public/personal scope, which is closer to
  the cloud workspace PRD shape.
- This still does not store documents, run indexing, enforce V21 ACLs, execute
  providers/V21, start Gateway, deploy ECS, or prove physical hardware
  professional consult.

## Latest Control-Tower Result - 2026-06-04 Workspace Document Upload Intake

The workspace surface now accepts real local document upload bytes without
promoting indexing or V21 execution.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-document-upload-intake.md`.
- Gateway exposes `POST /v1/workspace-documents` with schema
  `a21.gateway.workspace_documents.v1`.
- The endpoint accepts `multipart/form-data` only, stores bytes in an A21
  Gateway runtime directory, computes a safe `sha256:` document hash, and links
  the document to a workspace job/source pair.
- Linked jobs and sources report `stored_local_pending_index`,
  `storage_status=stored_local`, and `index_status=not_started_no_execute`.
- The local store defaults to `.a21-run/gateway/workspace-documents` and can
  be overridden by `A21_WORKSPACE_DOCUMENT_STORE_DIR`; a conservative per-file
  limit can be overridden by `A21_WORKSPACE_DOCUMENT_MAX_BYTES`.
- `/v1/professional-workspace` now shows `stored_local_pending_index` readiness
  when the selected scope has local stored uploads but no searchable source.
- Simulator Workspace Audit now includes a file picker and upload control, then
  displays only document/job/source metadata.
- Existing `/v1/workspace-upload-jobs` direct creation remains metadata-only
  and still rejects raw document text, bytes, base64 payloads, import URLs,
  local paths, credentials, and provider output.

Current conclusion:

- Internal test 4 has a real local upload intake layer: user file bytes can
  enter A21-controlled local storage and become visible as a safe pending-index
  source.
- This still does not parse, chunk, embed, index, upload to cloud storage,
  enforce durable auth/ACLs, execute provider/V21, start Gateway as a service,
  deploy ECS, or prove physical StackChan professional consult acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Soul Profile Contract

The roleplay profile selector now controls a real A21-owned role-soul layer
instead of a single default placeholder.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-soul-profile-contract.md`.
- Personality composition now supports an optional `role_souls/*.md` layer
  between `tone_rules` and `mode_prompts`.
- A21 roleplay profiles now include `a21_roleplay_default`,
  `a21_roleplay_wry_peer`, and `a21_roleplay_calm_anchor`.
- `/v1/roleplay-profile` accepts the selected `roleplay_profile` together with
  scenario, voice-clone profile, and bounded memory hints.
- Runtime summaries expose safe prompt-part identifiers such as
  `role_soul:a21_roleplay_wry_peer`, `soul_prompt_input_ready`, memory counts,
  and redaction flags, but not prompt bodies or memory text.
- Fast-companion and stock Xiaozhi roleplay prompt composition use the selected
  role soul before passing prompt input to the provider-neutral voice pipeline.
- Simulator controls now include a role soul selector and readout alongside
  scenario, voice clone, and memory controls.

Current conclusion:

- Internal test 4 roleplay now has a configurable persona/soul surface that can
  affect actual voice-pipeline prompt input, not only metadata.
- This is host-local contract evidence only. It does not execute provider/V21,
  start Gateway as a runtime service, deploy ECS, build/flash firmware, write
  serial/NVS, or prove physical StackChan roleplay audio acceptance.

## Latest Control-Tower Result - 2026-06-04 MCP Speaker Volume Freeze

The already-used official speaker volume MCP tool is now part of the unified
low-risk MCP control surface.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-stackchan-official-mcp-speaker-volume-freeze.md`.
- `/v1/xiaozhi/mcp-control` now allows `self.audio_speaker.set_volume` with
  bounded `volume=0..100`.
- The existing `/v1/xiaozhi/speaker-volume` compatibility/product shortcut is
  preserved.
- Unified MCP control still requires valid A21 `device_id`, an online Xiaozhi
  socket, `hello.features.mcp=true`, and trace/session/device identity.
- The endpoint records only the redacted send marker
  `xiaozhi.mcp.speaker_volume.sent` and safe device activity metadata such as
  bounded `speaker_volume`.
- Missing, out-of-range, or mixed screen/volume arguments are rejected before
  websocket write.

Current conclusion:

- Low-risk MCP parity now has one consistent Gateway control surface for
  speaker volume, device status, brightness, theme, and screen info.
- This is delivery-contract evidence only. Physical loudness/playback
  acceptance still requires fresh device evidence; no provider/V21, Gateway
  runtime service start, ECS, firmware, serial, NVS, or hardware action
  occurred.

## Latest Control-Tower Result - 2026-06-04 Workspace Index Request Ledger

Internal test 4 workspace upload readiness now has a no-execute indexing
request ledger.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-index-request-ledger.md`.
- Gateway now exposes `GET/POST /v1/workspace-index-jobs` with schema
  `a21.gateway.workspace_index_jobs.v1`.
- `POST /v1/workspace-index-jobs` accepts only safe document/job/source IDs
  and optional trace/session/device IDs. It verifies the stored local file is
  present, but does not read, parse, chunk, embed, OCR, upload, or send it to
  V21.
- Linked document, upload job, and source readiness now promote to
  `indexing_requested_no_execute`; the source summary exposes
  `indexing_requested_source_scope_counts`, and professional workspace
  readiness can report `indexing_api_ready=true` after a request is recorded.
- The simulator Workspace Audit surface now has an `Index Request` control and
  index-job readout using only safe IDs/status metadata.

Current conclusion:

- Local uploads now have a truthful adapter-ready pre-index state instead of a
  dead-end pending flag.
- This is not searchable readiness, real indexing, parsing, embedding, cloud
  storage, provider/V21 execution, durable auth/ACL, Gateway service startup,
  ECS deployment, firmware, serial, NVS, or physical StackChan professional
  consult acceptance.

## Latest Control-Tower Result - 2026-06-04 A21 Native V21 Voice-Query Bridge

The local A21 `v21-adapter-bridge` now consumes V21's native A21 v2
`/internal/v1/knowledge/voice-query` contract as its primary professional
query path.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-a21-v21-native-voice-query-bridge.md`.
- Bridge requests now pass safe A21 v2 fields into V21 native voice-query:
  `device_id`, `user_id`, `workspace_id`, `query_scope`, trace/session IDs,
  and the active collection ID.
- Native voice-query success mirrors V21-returned `source_scope_counts` and
  `workspace_status`; A21 no longer fabricates counts from `query_scope` on the
  green path.
- Direct retrieval remains only as a controlled no-evidence expansion fallback,
  and fallback counts are derived only from V21 result `source_scope` labels.
- Focused httptest coverage proves the bridge calls native voice-query on the
  green path, avoids direct retrieval there, preserves explicit
  `personal_plus_public` workspace scope metadata, and still retries the
  child-lock ASR fragment through the legacy retrieval fallback after a
  native `no_evidence` response.

Current conclusion:

- A21 is now wired to the V21 source-scope guard worker contract instead of
  silently bypassing it through direct retrieval for normal professional
  queries.
- This is adapter-boundary contract evidence only. It is not V21 merge/release,
  real personal upload indexing, durable tenant/account ACL, cloud storage,
  provider execution, Gateway service startup, ECS deployment, firmware,
  serial, NVS, or physical StackChan professional consult acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Device State Reflection

The selected roleplay identity is now visible as safe device/registry state,
not only as hidden prompt input.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-device-state-reflection.md`.
- `/v1/devices` now reflects safe roleplay state:
  `current_roleplay_profile`, `current_roleplay_scenario`,
  `roleplay_soul_ready`, `roleplay_memory_ready`,
  `roleplay_memory_hint_count`, and `roleplay_physical_accepted=false`.
- Device `runtime_echo` now receives only safe roleplay IDs, booleans, counts,
  and explicit non-storage flags for prompt text, memory text, and voice
  samples.
- The simulator Device Registry panel now shows role soul, scenario, and role
  memory alongside the existing voice-clone profile.
- Focused tests prove selected `a21_roleplay_wry_peer` +
  `engineer_pushback` + one safe memory hint reaches the device registry while
  unsafe URLs, memory text, and control text stay out of `/v1/devices`.

Current conclusion:

- Roleplay is now better reflected as an embodied runtime state: the device
  registry can tell which persona/scenario/memory posture is active without
  exposing the role prompt or memory content.
- This is host-local registry/simulator evidence only. It does not execute
  providers, V21, or voice-clone CLI; it does not start Gateway as a service,
  deploy ECS, build/flash firmware, write serial/NVS, or prove physical
  StackChan roleplay audio/visual acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Official Expression Plan

Roleplay identity now has a safe host-side expression plan instead of stopping
at hidden prompt/device-state metadata.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-official-expression-plan.md`.
- `/v1/roleplay-profile` now returns `expression_plan` with schema
  `a21.roleplay_expression_plan.v1`.
- The plan maps baseline posture, selected role soul, scenario emphasis, and
  memory cue phases to existing official StackChan action-plan metadata.
- The simulator shows the expression delivery policy plus action and packet
  counts.
- The plan records only safe profile/scenario/voice-clone IDs, memory
  readiness/count, event kinds, event values, packet counts, official semantic
  surfaces, and redaction flags.

Current conclusion:

- Roleplay is one step closer to embodied expression because the Gateway can
  now describe which official StackChan semantics should represent the selected
  role state.
- This is still a no-send contract with
  `delivery_policy=no_send_plan_only` and `physical_accepted=false`. It does
  not execute providers, V21, voice-clone CLI, Gateway runtime delivery,
  `/stackChan/ws`, firmware, serial, NVS, ECS, or physical hardware action.

## Latest Control-Tower Result - 2026-06-04 Workspace Console Product Surface

Internal test 4 workspace progress is now visible as a user-facing web console,
not only as simulator/debug controls.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-console-product-surface.md`.
- Gateway serves `GET /workspace` with a compact app shell for workspace setup,
  upload/index, readiness, and mode-boundary state.
- The console calls existing safe APIs only:
  `/v1/professional-workspace`, `/v1/workspace-documents`,
  `/v1/workspace-index-jobs`, `/v1/workspace-sources`,
  `/v1/professional-read-records`, `/v1/roleplay-profile`, and
  `/v1/voice-modes`.
- Playwright rendered the page at desktop `1270x900` and mobile `390x900`
  with no overflow findings.
- A dummy local upload through the page produced `stored_local`, and the page's
  index request produced `indexing_requested_no_execute`, source count `1`,
  and `searchable=false`.

Current conclusion:

- This is visible product movement for the cloud/web/app workspace shape:
  users can now operate the safe upload/index/readiness contract from a web
  surface.
- This is not real parsing, embedding, indexing, cloud storage, auth/device
  binding, provider execution, V21 execution, Gateway deployment, ECS,
  firmware, serial, NVS, or physical StackChan professional consult acceptance.
