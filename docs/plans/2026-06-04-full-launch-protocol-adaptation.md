# 2026-06-04 - A21 Full Launch Protocol Adaptation Sprint

Status: current control plan.
Owner: A21 control tower.
Transition: `T-FULL-LAUNCH-001-PROTOCOL-ADAPTATION-SPRINT`.
Created: 2026-06-04 CST.

## Goal

Move from internal test 3 voice-main-chain release to a full-launch sprint by
closing the protocol-change adaptation gaps introduced by the stock
`/v1/xiaozhi` Opus product path, while preserving the A21 launch discipline:
no fake PRD green, no provider/key leaks, no bare `xiaozhi.bin` product flash,
and no accidental X21/V21 runtime identity.

## Current State

- Current branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Current HEAD:
  `b58283b docs(handoff): add internal test 3 master handoff`.
- Internal test 3 package:
  `dist/a21-internal-test3-20260603-233245`.
- Main public Gateway:
  `47.103.57.217`.
- Product StackChan WebSocket:
  `ws://47.103.57.217/v1/xiaozhi`.
- Live runtime truth from the internal test 3 handoff:
  cascade chain with ASR `dashscope_qwen_asr_realtime`, LLM `deepseek`,
  fixed TTS `dashscope_qwen_tts_realtime`, and finding
  `stepfun_not_selected`.
- Evidence level:
  host bench `candidate_host_only`, physical report
  `candidate_gateway_downlink`, product readiness `server_side_blocked`.

Internal test 3 is accepted for team voice testing. It is not full PRD launch
green.

## Target State

- Gateway stock `/v1/xiaozhi` product protocol has regression tests for the
  complete message order and post-TTS suppressed-listen drain behavior.
- Product readiness and server-side readiness explicitly ingest selected
  voice-chain state and block on `stepfun_not_selected` when the launch profile
  requires StepFun.
- Xiaozhi physical evidence is matched to the requested `device_id`,
  `trace_id`, and `session_id` before readiness attaches it.
- Trusted physical audible/playback sidecar semantics are clear and do not
  over-credit stock downlink as PRD playback acceptance.
- Doctor/preflight can recognize official-compatible product-lane flash/build
  evidence without confusing it with generic PlatformIO current artifacts.
- Provider/selector surfaces are re-audited after the protocol pivot, including
  StepFun remote switch commands, redacted evidence, and hot-switch runtime
  boundaries.
- All changes are covered by focused tests and then `make verify`.

## Stop Rules

- Do not flash product StackChan with bare `xiaozhi.bin`.
- Do not add provider keys, Wi-Fi credentials, proxy values, transcripts,
  prompt text, raw/base64 audio, or local secret paths to repo, reports, logs,
  traces, docs, or stdout.
- Do not claim `launch_ready=true` from host bench, half-duplex candidate,
  static provider readiness, or `candidate_gateway_downlink`.
- Do not merge stale worktrees or old rescue branches directly.
- Do not run provider execute, V21 execute, or hardware write commands from a
  worker unless the worker task explicitly authorizes that tier.

## Worker Dispatch Matrix

Each worker must assume other workers may be active. Workers must not revert
unrelated edits and must keep to the assigned write set. If a worker needs to
touch a file outside its write set, it must stop and report the reason.

### Worker A: Gateway Xiaozhi Product Protocol Regression

Worker execution task:

- Add focused regression coverage for the internal test 3 stock Xiaozhi product
  protocol.

Write set:

- `internal/gateway/server_test.go`
- `internal/gateway/server.go` only if a regression test exposes a real bug
- `docs/engineering/PROTOCOL.md` only for narrowly clarifying product
  `/v1/xiaozhi` versus legacy/dev `/ws/audio` wording

Boundary conditions:

- No app/readiness/provider/firmware changes.
- No runtime deploy.
- No provider execute or hardware commands.

Acceptance commands:

```bash
go test ./internal/gateway -run 'Xiaozhi|WriteXiaozhi|OfficialStackChan|TraceEndpointReturnsVoicePipelineSplitSummary' -count=1
go test ./internal/transport/xiaozhi ./internal/transport/stackchan -count=1
git diff --check
```

Required cases:

- Successful answer followed by immediate stock physical `listen.start`, Opus
  tail frames, and `listen.stop` is suppressed/drained with no new answer
  downlink.
- Full product message order is pinned: no outbound speech before
  `listen.stop`; then `stt`, `tts.start`, sentence/binary Opus, and `tts.stop`.
- Protocol-version 2 Gateway lifecycle is covered or explicitly declared
  transport-only in docs/tests.
- Batch/non-streaming ASR either has `stt -> tts.start` coverage or is
  explicitly below product acceptance.

### Worker B: Readiness And Physical Evidence Adaptation

Worker execution task:

- Adapt app readiness/reporting to the internal test 3 evidence hierarchy and
  voice-chain selector truth.

Write set:

- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- `internal/app/xiaozhi_physical_evidence.go`
- `internal/app/app_stackchan_xiaozhi_half_duplex.go`
- focused tests in `internal/app/*test.go`

Boundary conditions:

- No Gateway protocol changes.
- No provider adapters.
- No firmware commands.
- No report deletion.

Acceptance commands:

```bash
go test ./internal/app -run 'XiaozhiPhysicalEvidence|XiaozhiHalfDuplex|ProductReadiness|ServerSideReadinessBundle|XiaozhiVoiceBench|ProviderLatencyBench' -count=1
git diff --check
```

Required cases:

- Product readiness/server-side readiness surfaces selected voice-chain state
  and reports `stepfun_not_selected` as a blocker when launch policy expects
  StepFun.
- Physical evidence ingestion rejects mismatched or stale `device_id`,
  `trace_id`, or `session_id` when a target device/trace/session is provided.
- Trusted operator/instrument audible evidence remains distinct from Gateway
  downlink and stock debug playback echoes.
- Half-duplex candidate reports cannot be interpreted as PRD green when
  `device.playback.ack`, `device.playback.stop_done`, or trusted observation
  are missing.

### Worker C: Official-Compatible Firmware Evidence Pointer

Worker execution task:

- Make local doctor/preflight understand official-compatible product-lane flash
  or build evidence without weakening firmware safety guards.

Write set:

- `internal/app/doctor.go`
- `internal/app/app_office_preflight.go`
- `internal/app/official_stackchan.go`
- focused tests in `internal/app/*test.go`
- docs only if an env/report pointer contract is added:
  `docs/engineering/FIRMWARE_RELEASE_DISCIPLINE.md`,
  `docs/engineering/DOCTOR.md`

Boundary conditions:

- No flash execution.
- No generic Xiaozhi product flash path.
- No firmware source edits unless tests prove a metadata-only pointer requires
  it; stop before any build or upload action.

Acceptance commands:

```bash
go test ./internal/app -run 'Doctor|OfficePreflight|StackChanOfficialXiaozhiCompatible|XiaozhiFirmwareFlash|FirmwareCurrentArtifact' -count=1
git diff --check
```

Required cases:

- The latest executed official-compatible flash report can satisfy or clearly
  annotate product-lane artifact evidence.
- Newer dry-run flash plans must not override older executed flash evidence
  just because of mtime.
- Generic `xiaozhi.bin` product flash remains blocked.
- `wake_word_firmware_build_required` stays honest unless custom wake is
  actually accepted by the guarded wake evidence path.

### Worker D: Provider Selector And StepFun Runtime Switch Plan

Worker execution task:

- Audit and, if low-risk, adapt provider/voice-chain selector surfaces so the
  StepFun remote switch and post-switch evidence are explicit and redacted.

Write set:

- `internal/providers/*`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/provider_*.go`
- Gateway voice-chain selector tests only if the selector contract itself is
  wrong: `internal/gateway/server_test.go`
- docs:
  `docs/engineering/A21_PROVIDER_BENCHMARKS.md`,
  `docs/engineering/VOICE_MODE_SELECTION.md`,
  or a new narrow provider switch runbook under `docs/engineering/`

Boundary conditions:

- No provider key values.
- No remote ECS secret writes.
- No provider execute from Mac.
- No firmware/hardware edits.

Acceptance commands:

```bash
go test ./internal/providers -run 'DashScope|Doubao|ProviderSmoke|TextStream|VoicePipelineAdapters|GatewayVoiceProvider|Realtime' -count=1
go test ./internal/app -run 'XiaozhiStreamingProviderReadiness|ProviderSmoke|ProviderCompat|ProviderLatency' -count=1
go test ./internal/gateway -run 'VoiceChainProfiles|GatewayProfiles' -count=1
git diff --check
```

Required cases:

- StepFun selected state is distinguishable from DeepSeek fallback in reports.
- Remote switch runbook uses root-only `/etc/a21/secrets/provider.env`, service
  restart, and fresh `/v1/voice-chain-profiles`, host bench, physical evidence,
  and readiness reruns.
- Reports expose env names and profile IDs only, not key/model values.

## Local Control-Tower Work While Workers Run

The control tower should not duplicate worker implementation. It may do
non-overlapping docs/control work:

- Create a short current-control entry if approved by this plan.
- Create a current evidence manifest that pins internal test 3 reports and
  live Gateway snapshots.
- Update `docs/agent_handoff_log.md` after the worker round with actual
  changed files, tests, risks, and next action.
- Update `docs/project_state_machine.md` only after accepted changes land.

## Summary Format Required From Workers

Workers must return:

```text
What changed:
Files changed:
Tests run and results:
Deviations from plan:
Remaining risks/blockers:
Next suggested action:
```

If blocked, include the exact file/function/command where the block occurred
and do not broaden scope silently.

## Integration Order

1. Integrate Worker A first if it only adds tests/docs or exposes a real
   Gateway bug; Gateway protocol regressions are the base for all evidence.
2. Integrate Worker B second; readiness semantics depend on the Gateway event
   vocabulary.
3. Integrate Worker C third; firmware evidence pointer must not mask physical
   readiness truth.
4. Integrate Worker D once selector/report semantics are clear.
5. Run:

```bash
go test ./internal/gateway ./internal/app ./internal/providers ./internal/transport/xiaozhi ./internal/transport/stackchan ./internal/v21adapter ./internal/personality -count=1
git diff --check
make verify
make preflight
make doctor
```

Warnings from `make preflight` or `make doctor` must be classified in the
handoff log instead of hidden.

## Rollback

- Documentation-only or test-only changes can be reverted normally if they
  misclassify evidence.
- Runtime code changes must be reverted as a single worker commit if they
  weaken protocol ordering, redaction, product flash guards, or PRD readiness
  blockers.
- Public Gateway deployment must keep the existing `/opt/a21.prev` rollback
  pattern from the internal test 3 master handoff.

## Done Means

- The protocol-change adaptation matrix is covered by focused tests.
- Readiness reports tell the same truth as the live internal test 3 runtime:
  DeepSeek fallback is still selected until StepFun is actually switched and
  revalidated.
- Physical evidence cannot pass PRD acceptance without playback acknowledgement
  or trusted operator/instrument observation.
- Official-compatible firmware flash evidence is preserved without opening
  generic product flash lanes.
- The next launch-critical action is recorded in the handoff log and state
  machine.
