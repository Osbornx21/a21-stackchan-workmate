# A21 Current Evidence Manifest

Status: current evidence manifest.
Date: 2026-06-05 CST.
Owner: A21 control tower.
Schema intent: `a21.current_evidence_manifest.v1` human-readable first cut.

This manifest names the evidence that may be used for current launch routing.
It does not replace raw reports. It prevents raw report mtime or optimistic
child reports from becoming the launch decision source.

## Canonical Decision

- `launch_ready`: false
- `prd_accepted`: false
- `demo_ready`: true for internal test 3 voice-main-chain testing
- `current_status`: `internal_test4_mode_contract_cut_in_progress_physical_pending`
- `current_release_level`: internal test 3 accepted; internal test 4 active build
- `host_only_evidence_use`: gap reduction only
- `candidate_physical_evidence_use`: Gateway/downlink diagnosis only
- `requires_physical_acceptance`: true

## Current Package

Accepted internal-test package:

- `dist/a21-internal-test3-20260603-233245`

Accepted tarball:

- `dist/a21-internal-test3-20260603-233245.tar.gz`

SHA-256:

- `62d2e62fcab0dee9cbf1e4ae5c49ec5941ce162ec65c0bf2c873b3e70ffda93f`

Package manifest:

- `dist/a21-internal-test3-20260603-233245/a21-internal-test3-manifest.json`

Package README:

- `dist/a21-internal-test3-20260603-233245/README_INTERNAL_TEST3.txt`

Package source commit:

- `074e3d877d33`

Release documentation commit:

- `221c153`

Master handoff commit:

- `b58283b`

## Accepted Reports

These are accepted for their stated scope only.

| Scope | Report | Status | Launch Use |
| --- | --- | --- | --- |
| Public Gateway health snapshot | `dist/a21-internal-test3-20260603-233245/reports/a21-public-gateway-healthz.json` | ok | Server reachability only |
| Public Gateway devices snapshot | `dist/a21-internal-test3-20260603-233245/reports/a21-public-gateway-devices.json` | product device online at snapshot | Device registry snapshot only |
| Public Gateway profile snapshot | `dist/a21-internal-test3-20260603-233245/reports/a21-public-gateway-profiles.json` | available | Transport profile snapshot only |
| Public Gateway voice-chain snapshot | `dist/a21-internal-test3-20260603-233245/reports/a21-public-gateway-voice-chain-profiles.json` | `stepfun_not_selected` | Runtime selector truth; blocker until switched if StepFun is launch policy |
| Public OTA snapshot | `dist/a21-internal-test3-20260603-233245/reports/a21-public-gateway-ota.json` | returns public Xiaozhi WS | OTA routing only |
| Host voice bench | `dist/a21-internal-test3-20260603-233245/reports/a21-xiaozhi-voice-bench-20260603-233014.342186000.json` | `candidate_host_only` | Host loopback candidate, not physical PRD |
| Physical Xiaozhi evidence | `dist/a21-internal-test3-20260603-233245/reports/a21-xiaozhi-physical-evidence-20260603-232946.250456000.json` | `candidate_gateway_downlink` | Physical Gateway/downlink candidate, not playback acceptance |
| Product readiness | `dist/a21-internal-test3-20260603-233245/reports/a21-product-readiness-20260603-233027.json` | `server_side_blocked` | Canonical readiness for package |
| Server-side readiness bundle | `dist/a21-internal-test3-20260603-233245/reports/a21-server-side-readiness-bundle-20260603-233027.json` | `server_side_blocked` | Server-side bundle only |
| Streaming provider readiness | `dist/a21-internal-test3-20260603-233245/reports/a21-xiaozhi-streaming-provider-readiness-20260603-233026-1780500626486079000.json` | local no-execute blocked | Local static gate only; remote runtime snapshot is separate |
| Official-compatible product flash | `dist/a21-internal-test3-20260603-233245/reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-170812-1780477692092580000.json` | executed product-lane flash | Firmware lane evidence, not wake/playback PRD acceptance |
| Packaged binary host gate | `dist/a21-internal-test3-20260603-233245/reports/a21-packaged-binary-gate-host.json` | passed with warnings | Packaged binary host gate only |

## Current Runtime Blockers

- Live ECS runtime selects LLM `stepfun` in the cascade chain.
- Fresh route-eligible StepFun provider smoke has passed.
- Fresh cloud-edge Xiaozhi host bench has passed as `candidate_host_only`.
- Product readiness remains `server_side_blocked` because V21 professional
  execution and physical PRD acceptance are still missing.
- Machine-readable physical evidence remains below PRD acceptance.
- Stock firmware does not expose `device.playback.ack` or
  `device.playback.stop_done`.
- Trusted operator/instrument audible observation is not yet encoded as launch
  evidence.
- Local `make preflight` and `make doctor` still warn about firmware/wake
  surfaces; those warnings must be classified, not hidden.

## 2026-06-05 Hardware Body Machine Evidence

This evidence is accepted for product-socket delivery and reconnect routing. It
does not by itself prove physical PRD acceptance.

| Scope | Evidence | Status | Launch Use |
| --- | --- | --- | --- |
| Product reconnect flash plan | `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-020621-1780596381380115000.json` | `status=ready`, product artifact SHA `3eef974929aed78cdd77232897485aaac25bce8aa98daa4d8d78b3d96662b7ac` | Product-lane flash guard evidence |
| Product reconnect flash execute | `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-020726-1780596446779669000.json` | `status=passed`, `flash_allowed=true`, `flash_executed=true` | Product-lane flash execution evidence |
| Public device reconnect | `/v1/devices`, device `44:1b:f6:e2:6a:60` | moved from disconnected to online with heartbeat after flash | Product socket recovery evidence |
| Product body scene | trace `a21-trace-hardware-showtime-flash-b9c0baa-202606050208` | HTTP 200 `status=delivered`, 8 body-scene steps, 16 trace markers | Machine-readable screen/RGB/servo delivery evidence |

Current decision after this evidence:

- `launch_ready`: false
- `prd_accepted`: false
- `hardware_body_machine_evidence`: ready for showtime scene
- `hardware_body_physical_acceptance`: pending operator/instrument
  confirmation

## 2026-06-04 Protocol Adaptation Verification

The full-launch protocol-adaptation worker round is accepted as local
gap-reduction evidence, not as launch evidence.

Accepted local verification:

- `go test ./internal/gateway -run 'Xiaozhi|WriteXiaozhi|OfficialStackChan|TraceEndpointReturnsVoicePipelineSplitSummary' -count=1`:
  passed.
- `go test ./internal/app -run 'XiaozhiPhysicalEvidence|XiaozhiHalfDuplex|ProductReadiness|ServerSideReadinessBundle|XiaozhiVoiceBench|ProviderLatencyBench' -count=1`:
  passed.
- `go test ./internal/app -run 'Doctor|OfficePreflight|StackChanOfficialXiaozhiCompatible|XiaozhiFirmwareFlash|FirmwareCurrentArtifact' -count=1`:
  passed.
- `go test ./internal/app -run 'XiaozhiStreamingProviderReadiness|ProviderSmoke|ProviderCompat|ProviderLatency' -count=1`:
  passed.
- `go test ./internal/providers -run 'DashScope|Doubao|ProviderSmoke|TextStream|VoicePipelineAdapters|GatewayVoiceProvider|Realtime' -count=1`:
  passed.
- `go test ./internal/transport/xiaozhi ./internal/transport/stackchan -count=1`:
  passed.
- `go test ./internal/gateway -run 'VoiceChainProfiles|GatewayProfiles' -count=1`:
  passed.
- `make verify`: passed.
- A later default-concurrency `make verify` retry after documentation updates
  was killed by the host with `Killed: 9`; `go test -p 1 ./...`,
  `git diff --check`, and `GOMAXPROCS=2 make verify` passed immediately after.
- `make preflight`: passed with warning
  `wake_word_firmware_build_required`.
- `make doctor`: passed with warning `wake_word_firmware_build_required` and
  `firmware.product_lane_artifact_evidence.status=satisfied` from the executed
  official-compatible product-lane flash report.

Live retry snapshot at 2026-06-04 02:13 CST:

- `http://47.103.57.217/healthz`: ok.
- `http://47.103.57.217/v1/devices`: product device
  `44:1b:f6:e2:6a:60` online, current mode `local_fallback`.
- `http://47.103.57.217/v1/voice-chain-profiles`: selected LLM `deepseek`,
  finding `stepfun_not_selected`.
- `http://47.103.57.217/xiaozhi/ota/`: returns
  `ws://47.103.57.217/v1/xiaozhi`.
- `ssh root@47.103.57.217`: unavailable from this control thread due
  `Permission denied (publickey)`.

Manifest decision after this verification remains unchanged:

- `launch_ready`: false
- `prd_accepted`: false
- `current_status`: `server_side_blocked`
- `next_runtime_transition`:
  `docs/plans/2026-06-04-ecs-control-plane-and-stepfun-switch.md`

## 2026-06-04 ECS Env Blocker Verification

The ECS control-plane check succeeded with an explicit local SSH identity, but
the StepFun switch remains blocked by missing remote env names.

Remote ECS env-name status:

- Provider env file: present, root-owned, mode `600`.
- `A21_LAB_STEPFUN_API_KEY`: missing.
- `A21_STEPFUN_MODEL`: missing.
- `A21_DASHSCOPE_API_KEY`: present.
- `A21_BAILIAN_QWEN_TTS_MODEL`: missing.

Local control-machine env-name status:

- `A21_LAB_STEPFUN_API_KEY`: missing.
- `A21_STEPFUN_MODEL`: missing.
- `A21_DASHSCOPE_API_KEY`: missing.
- `A21_BAILIAN_QWEN_TTS_MODEL`: missing.

Fresh reports:

| Scope | Report | Status | Launch Use |
| --- | --- | --- | --- |
| Product readiness retry | `reports/a21-product-readiness-20260604-022419.json` | `server_side_blocked`, selected LLM `deepseek`, `stepfun_not_selected` | Current blocker evidence only |
| Server-side readiness retry | `reports/a21-server-side-readiness-bundle-20260604-022420.json` | `server_side_blocked`, missing `provider_smoke` and `stepfun_not_selected` | Current blocker evidence only |

Manifest decision remains unchanged:

- `launch_ready`: false
- `prd_accepted`: false
- `current_status`: `server_side_blocked`
- `runtime_switch_blocker`: `stepfun_env_missing`

## 2026-06-04 StepFun Runtime And Route Eligibility Update

The public Gateway runtime has moved beyond the earlier env/code-sync blocker.
Use the following as the current routing truth until superseded by newer named
reports:

- Public `/v1/voice-chain-profiles`: selected LLM `stepfun`.
- Public `/v1/devices`: product device `44:1b:f6:e2:6a:60` online with
  cascade DashScope ASR, StepFun LLM, and fixed DashScope TTS.
- Public `/xiaozhi/ota/`: `ws://47.103.57.217/v1/xiaozhi`.
- Host bench:
  `reports/a21-xiaozhi-voice-bench-20260604-023616.742713000.json` executed
  the cloud-edge chain with StepFun selected, but remains host-loopback
  evidence only.
- Provider smoke:
  `reports/a21-provider-smoke-20260604-023711-678466985.json` passed execution
  and streaming checks, but records `route_eligible=false` from the pre-promotion
  catalog and cannot close provider readiness.

New route decision:

- Built-in `stepfun` is being promoted to explicit route-eligible
  launch-policy LLM status by
  `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`.
- After that commit is deployed, readiness must use a fresh executed StepFun
  provider-smoke report with `route_eligible=true`; do not rewrite or
  reinterpret older reports.

Promotion source status:

- Commit `3741c4a feat(providers): promote stepfun route eligibility` promotes
  the built-in StepFun catalog entry locally.
- Local dry-run provider smoke now reports `route_eligible=true` for configured
  StepFun, but `executed=false`; this is a schema/shape check only.
- The local commit chain through
  `5dba606 docs(control): record integration audit` has now been pushed to
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- ECS has not yet been updated to `3741c4a` or newer from this thread because
  SSH control-plane access is unavailable.
- Public Gateway health is currently not usable as launch evidence:
  `/healthz`, `/v1/devices`, `/v1/voice-chain-profiles`, and `/xiaozhi/ota/`
  return empty HTTP replies from the control Mac.
- The next launch-routing evidence must be a fresh remote executed StepFun
  provider-smoke report produced after ECS is running `3741c4a` or newer.

## Missing Real Evidence

Current full-launch gaps:

- V21 professional execution evidence through the explicit A21/V21 adapter
  boundary.
- Fresh physical Xiaozhi evidence after the current selected chain.
- Trusted physical audible/playback observation or debug playback
  acknowledgement.
- `device.playback.stop_done` or trusted stop observation for barge-in.
- Wake-word product proof for the selected wake phrase/profile.
- Final `product-readiness --require-real` style report that remains honest
  until all physical gates are complete.

## 2026-06-04 ECS StepFun Cloud-Edge Evidence Update

Current runtime evidence after deploying
`d9362a7 feat(readiness): accept cloud-edge xiaozhi evidence` to ECS:

| Scope | Report | Status | Launch Use |
| --- | --- | --- | --- |
| StepFun provider smoke | `reports/a21-provider-smoke-20260604-035200-977132343.json` | `passed`, `executed=true`, `stream=true`, `route_eligible=true` | Real provider evidence |
| Xiaozhi streaming provider readiness | `reports/a21-xiaozhi-streaming-provider-readiness-20260604-035317-1780516397705439206.json` | `passed`, `stepfun_selected` | Static provider-chain gate |
| Xiaozhi cloud-edge host bench | `reports/a21-xiaozhi-voice-bench-20260604-035502.222374820.json` | `candidate_host_only`, `failure_count=0`, `cloud_edge` | Server-side host voice evidence only |
| Product readiness | `reports/a21-product-readiness-20260604-040153.json` | `server_side_blocked`; provider and host voice ready | Canonical readiness |
| Server-side readiness bundle | `reports/a21-server-side-readiness-bundle-20260604-040207.json` | `server_side_blocked`; missing `v21_professional_smoke` | Server-side bundle |

Current decision:

- `launch_ready`: false
- `prd_accepted`: false
- `current_status`: `server_side_blocked`
- `provider_smoke`: closed for StepFun route eligibility
- `host_voice_loopback`: closed as cloud-edge candidate evidence
- remaining blockers: `v21_professional_execution` and
  `physical_stackchan_prd_acceptance`

Network note:

- ECS loopback, Caddy, 5080lab public checks, and Gateway selector are healthy.

## 2026-06-04 V21 Professional Evidence Transition

The next server-side transition is
`docs/plans/2026-06-04-v21-professional-execution-validation.md`.

Current V21 adapter state from the control shell:

- `A21_V21_ADAPTER_URL`: missing.
- `A21_V21_BACKEND_URL`: missing.
- `A21_V21_ADAPTER_TOKEN`: missing.
- Local adapter boundary `127.0.0.1:21121`: not listening.
- Historical V21 reports under `reports/` are from 2026-06-01/02 and are not
  current launch evidence for this 2026-06-04 sprint.

Evidence decision:

- Do not close `v21_professional_execution` from old reports or adapter health
  alone.
- Required evidence is a fresh redacted `v21-adapter-smoke --execute` report
  through the explicit A21/V21 adapter boundary.
- If no adapter/backend boundary is available, readiness remains
  `server_side_blocked` with a precise operator ask rather than an inferred
  failure of the A21 runtime.

Fresh local control-shell reports:

| Scope | Report | Status | Launch Use |
| --- | --- | --- | --- |
| V21 adapter dry-run | `reports/a21-v21-adapter-smoke-20260604-041657.json` | `skipped`, `configured=false`, `executed=false`, `redaction_ok=true` | Operator ask / boundary-missing evidence only |
| Local product readiness collector | `reports/a21-product-readiness-20260604-041710.json` | `server_side_blocked` from local shell missing Gateway/provider/selector/V21 | Not canonical launch evidence; do not supersede ECS 04:02 reports |
| Local server-side readiness collector | `reports/a21-server-side-readiness-bundle-20260604-041710.json` | `server_side_blocked` from local shell missing Gateway/provider/selector/V21 | Not canonical launch evidence; do not supersede ECS 04:02 reports |

Use the 04:02 ECS StepFun/cloud-edge reports as the current canonical
provider/host-voice evidence. Use the 04:16/04:17 local reports only to explain
why V21 execution cannot run from this shell yet.

## 2026-06-04 V21 Local Adapter Execution Evidence

V21 was confirmed through its own control thread as a local/LAN Docker Compose
service. The active local backend was brought up on `18081`, and A21 used a
temporary bridge on `21121` to preserve the explicit adapter boundary.

Accepted V21 evidence:

| Scope | Report | Status | Launch Use |
| --- | --- | --- | --- |
| V21 adapter execution | `reports/a21-v21-adapter-smoke-20260604-043456.json` | `passed`, `configured=true`, `executed=true`, `redaction_ok=true`, evidence `5`, speech `1`, card `1`, follow-up `1` | V21 professional adapter-boundary evidence |
| Product readiness with V21 evidence | `reports/a21-product-readiness-20260604-043528.json` | `server_side_blocked`; provider, V21, and host voice ready; missing local Gateway/wake/selector/physical | Gap-reduction only, not physical PRD |
| Server-side readiness with V21 evidence | `reports/a21-server-side-readiness-bundle-20260604-043528.json` | `server_side_blocked`; provider ready, V21 ready, host voice ready; missing `gateway`, `wake_word`, `voice_chain_selector` | Current server-side rollup for local-control context |

Evidence decision:

- `v21_professional_execution` is closed for the A21 adapter contract.
- Do not claim permanent ECS Gateway V21 topology from this local bridge run.
- Do not claim full PRD readiness; physical StackChan acceptance remains
  required.
- This control Mac still receives empty HTTP replies from direct public curls,
  and a later ECS tcpdump did not observe the Mac curl reaching the host. Treat
  this as a source-path/network issue, not as current A21 runtime health

## 2026-06-04 Internal Test 4 Mode And Workspace Evidence Direction

Internal test 4 is now the active build direction, but no internal test 4
release package exists yet.

Accepted as source/control evidence in this cut:

- PRD v0.5 names `roleplay` and `professional` as the two user-facing modes.
- `roleplay` is the default embodied mode for role/personality, memory hints,
  playbooks, voice-clone selection, and the existing low-latency Xiaozhi speech
  path.
- `professional` remains the user-confirmed V21 evidence path.
- `dialogue` is a backwards-compatible alias under `roleplay`, not a launch
  mode.
- The A21 Cloud/Web/App target shape is documented for upload, indexing,
  device binding, public-only query, personal-only query, and
  personal+public query.
- V21 adapter v2 direction is documented with `workspace_id`, `user_id`, and
  `query_scope` fields while preserving the A21/V21 boundary.

Evidence not yet claimed:

- No A21 Cloud/Web/App upload implementation exists in this repository.
- No V21 upload/index/query-scope implementation has been verified.
- No permanent public Gateway V21 topology is claimed from the local bridge
  smoke.
- No physical StackChan internal test 4 professional consult evidence is
  claimed.
- Internal test 3 package evidence remains the accepted release evidence until
  an internal test 4 package is built and verified.
  evidence.

## Source-Only Or Blocked Evidence

Do not use these as launch acceptance:

- Any host-only `xiaozhi-voice-bench` report without physical playback markers.
- `/v1/xiaozhi/say` playback as a substitute for normal mic-driven dialogue.
- `stackchan-accept --check xiaozhi-half-duplex` reports that are
  `physical_review_required` or missing playback ack/stop done.
- Static provider-shape reports without real execution.
- Generic `xiaozhi-firmware-flash-*` product StackChan evidence.
- Bare `xiaozhi.bin` product flash evidence.
- Historical `real_launch_ready` reports from older readiness schemas.

## Redaction Contract

Evidence in this manifest and current launch reports must not include:

- provider key values
- Wi-Fi credentials
- private keys
- proxy secret values
- transcripts
- prompt text
- provider output text
- raw audio or base64 audio payloads
- local secret paths
- full URLs with credentials or secrets

## Next Evidence Updates

The next manifest revision should be written after the protocol-adaptation
worker round lands and fresh verification is run. It should name the exact new
reports rather than relying on `--use-latest-reports` mtime.
