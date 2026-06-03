# A21 Current Evidence Manifest

Status: current evidence manifest.
Date: 2026-06-04 CST.
Owner: A21 control tower.
Schema intent: `a21.current_evidence_manifest.v1` human-readable first cut.

This manifest names the evidence that may be used for current launch routing.
It does not replace raw reports. It prevents raw report mtime or optimistic
child reports from becoming the launch decision source.

## Canonical Decision

- `launch_ready`: false
- `prd_accepted`: false
- `demo_ready`: true for internal test 3 voice-main-chain testing
- `current_status`: `server_side_blocked`
- `current_release_level`: internal test 3
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

- Live runtime still selects LLM `deepseek`; `stepfun` is only recommended.
- Voice-chain snapshot includes `stepfun_not_selected`.
- Product readiness remains `server_side_blocked`.
- Machine-readable physical evidence remains `candidate_gateway_downlink`.
- Stock firmware does not expose `device.playback.ack` or
  `device.playback.stop_done`.
- Trusted operator/instrument audible observation is not yet encoded as launch
  evidence.
- Local `make preflight` and `make doctor` still warn about firmware/wake
  surfaces; those warnings must be classified, not hidden.

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
- ECS has not yet been updated to `3741c4a` from this thread because SSH
  control-plane access is unavailable.
- The next launch-routing evidence must be a fresh remote executed StepFun
  provider-smoke report produced after ECS is running `3741c4a` or newer.

## Missing Real Evidence

Current full-launch gaps:

- Fresh route-eligible executed StepFun provider-smoke report.
- Fresh product/server-side readiness reports after that provider-smoke report.
- Fresh physical Xiaozhi evidence after the current selected chain.
- Trusted physical audible/playback observation or debug playback
  acknowledgement.
- `device.playback.stop_done` or trusted stop observation for barge-in.
- Wake-word product proof for the selected wake phrase/profile.
- Final `product-readiness --require-real` style report that remains honest
  until all physical gates are complete.

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
