# A21 Control Ledger

Status: active integration ledger.
Date: 2026-05-31.
Ledger branch: `codex/a21-integration-governance-slices`.
Last accepted integration commit before this ledger update: `598c0d9`.

This ledger is the control tower's current operating board. It records which
branch, worktree, thread role, and tool tier are authorized next. Update it
whenever the control tower changes branch ownership, resumes or pauses an
execution thread, promotes a new tool tier, or accepts a handoff.

This ledger does not replace `A21_PROJECT_CONTROL.md`. The project-control doc
defines policy; this ledger records the current queue and accepted state.

## Current Accepted State

- Control branch: `codex/a21-project-control`.
- Control HEAD before this integration update:
  `7bdfe9d docs(control): record PCM flash ADR handoff`.
- Integration branch: `codex/a21-integration-governance-slices`.
- Integration HEAD before this ledger update:
  `598c0d9 fix(providers): reject agent profiles as voice primary`.
- Main worktree: `/Users/jiyurun/Documents/New project`.
- Main worktree status at acceptance: clean.
- `a21 control-guard` is the active machine-readable tool-tier gate.
- `a21 promotion-readiness` is the active machine-readable integration
  promotion gate for separating local review readiness from external promotion
  readiness.
- StackChan hardware mainline diagnostic branch is committed at
  `1e38804 fix(hardware): enforce StackChan capability honesty`.
- Provider Spine DeepSeek text-stream readiness branch is committed at
  `6b7fdf0 fix(providers): measure first content from content deltas`.
- Professional V21 evidence adapter readiness branch is committed at
  `fc61793 fix(v21): wire configured adapter into gateway env`.
- Official PCM bridge app flash ADR docs-only branch is committed at
  `a031f3d docs(firmware): add PCM bridge flash ADR gate`.
- Integration branch merged all four accepted slices:
  `5674b08`, `3c3ae1f`, `6abf8d7`, and `acdd929`; ledger follow-ups are
  `1872ca9`, `54af08b`, `045147d`, `2339d4e`, `9f9d5e6`, `3ddcc45`,
  `e55b652`, `5e54eb9`, `4d356e7`, `3b9f05d`, `f9df726`, `94d87c3`, and
  `193a1f3`; Task 4 acceptance is `7af0259`, post-commit review tracking is
  `819fe8b`, the Task 4 trace-fidelity P2 fix is `22c90ae`, Provider Spine
  Task 5 verification tracking is `ac2ab42`, PRD next-slice audit tracking is
  `52c64d2`, provider latency slice-open tracking is `28171ad`, and provider
  latency bench scaffold acceptance is `4532df8`; provider latency handoff
  acceptance is `8bc41f6`, and provider latency post-review tracking is
  `37697b9`; the post-review P2 mode vocabulary fix is `d03365f`, provider
  latency post-review acceptance is `b8d46f8`, provider fixture schema
  slice-open tracking is `44d6c76`, provider fixture metadata contract
  acceptance is `f69c6b8`, provider fixture metadata ledger acceptance is
  `0e46761`, provider fixture metadata post-review tracking is `cfc1658`,
  provider fixture metadata post-review acceptance is `7eb3a8d`, provider
  fixture sidecar hardening tracking is `68e1cc8`, provider fixture sidecar
  hardening implementation is `a4e3a6f`, sidecar hardening acceptance is
  `447b917`, fixture sidecar post-review tracking is `0633b2b`, fixture
  sidecar post-review closure is `a1742ae`, AgentTaskProvider Bridge scaffold
  opening is `8974460`, AgentTaskProvider Bridge scaffold acceptance is
  `560df00`, AgentTaskProvider Bridge post-review tracking is `f15b4f2`, and
  AgentTaskProvider Bridge post-review P2 fix is `598c0d9`.
- Read-only integration review found no P0/P1/P2 issues against the merged
  governance baseline at `1872ca9`.
- Control tower has selected the single combined integration branch as the
  current local promotion candidate. Topic branches remain frozen evidence and
  rollback/review handles, not the preferred next PR shape.
- The local repository has no configured remote and no `main` or `master`
  branch. Remote PR creation or local mainline merge therefore requires a
  later explicit target decision outside this ledger update.
- Current integration HEAD after AgentTaskProvider Bridge post-review fix:
  `598c0d9 fix(providers): reject agent profiles as voice primary`.
- Current PRD Phase 5 AgentTaskProvider Bridge state is T1/T2 scaffold only:
  external agents remain an explicit Agent I/O Layer, not an A21 router,
  second brain, backend orchestrator, or realtime first-response owner. Real
  Hermes/MiMo/OpenClaw runtime, provider execution, V21 execution, Gateway
  runtime, and hardware/device-control paths remain unauthorized future
  windows.
- Official StackChan PCM bridge NVS-only lane is accepted as closed for the
  current M3 governance slice.
- Official StackChan PCM bridge app flash-execute remains T8 blocked. The new
  ADR document is draft gate evidence only; it is not execute authorization.
- No current PRD slice authorizes raw `pio upload`, `idf.py flash`, copied
  `esptool write_flash`, provider/V21 execute, or background hardware writes.

## Thread Ledger

| Thread | Role | Worktree | Status | Max tier | Write authority |
| --- | --- | --- | --- | --- | --- |
| `019e7b6f-dedb-73c1-aee6-2c438858da03` | Control tower | `/Users/jiyurun/Documents/New project` | active | T1 by default; higher only after declaration | yes |
| `019e7c57-bd71-72d0-9cf8-ec9674f41bf0` | AgentTaskProvider Bridge post-commit review | `/Users/jiyurun/.codex/worktrees/7077/New project` | completed; two P2 findings fixed by control at `598c0d9` | T0/T1/T2 | no |
| `019e7c48-9fc7-7ff0-8563-965fe9da9f72` | AgentTaskProvider Bridge scaffold implementation | `/Users/jiyurun/.codex/worktrees/04c0/New project` | completed; accepted into integration branch at `560df00` | T1/T2 | no |
| `019e7c41-1842-7a30-a185-aafc73ba2d73` | Provider fixture sidecar post-commit review | `/Users/jiyurun/.codex/worktrees/8922/New project` | completed; no P0/P1/P2 findings | T0/T1/T2 | no |
| `019e7c36-4617-73a1-aba0-1d35acc26efe` | Provider fixture sidecar hardening implementation | `/Users/jiyurun/.codex/worktrees/38a4/New project` | accepted into integration branch at `a4e3a6f`; post-review closed at `a1742ae` | T1/T2 | no |
| `019e7c30-7329-7a32-996e-0566c9746e5d` | Provider fixture metadata post-commit review | `/Users/jiyurun/.codex/worktrees/1181/New project` | completed; no P0/P1/P2 findings | T0/T1/T2 | no |
| `019e7c25-ec43-7591-9954-5227c7288e89` | Provider latency fixture schema implementation | `/Users/jiyurun/.codex/worktrees/6b8d/New project` | accepted into integration branch at `f69c6b8` | T1/T2 | no |
| `019e7c1d-4678-7eb2-8666-9c5c331585d0` | Provider latency bench post-commit review | `/Users/jiyurun/.codex/worktrees/0a27/New project` | completed; P2 mode vocabulary finding fixed by control at `d03365f` | T0/T1/T2 | no |
| `019e7c0d-f7a0-7323-8413-e3e2aac53a94` | Provider latency bench scaffold implementation | `/Users/jiyurun/.codex/worktrees/acc7/New project` | completed; handoff accepted into integration branch at `4532df8` | T1/T2 | no |
| `019e7c09-98d6-75d0-85a4-f0bf63cd4e3b` | PRD next-slice audit | `/Users/jiyurun/.codex/worktrees/ed15/New project` | completed; recommended provider-latency-bench scaffold | T0/T1 | no |
| `019e7c00-ff6c-7f72-9851-a6e3ce637baf` | Fast Companion Hybrid post-commit review | `/Users/jiyurun/.codex/worktrees/de55/New project` | completed; P2 trace-fidelity finding fixed by control | T0/T1/T2 | no |
| `019e7bed-4e1e-7512-8f21-1647b2357c00` | Fast Companion Hybrid Gateway boundary implementation | `/Users/jiyurun/.codex/worktrees/80c8/New project` | completed; handoff accepted into integration branch | T1/T2 | no |
| `019e7be6-bca3-71f2-9770-857b9da48b67` | Provider Spine Fast Companion Hybrid boundary audit | `/Users/jiyurun/.codex/worktrees/0d72/New project` | completed; no P0/P1 regression; Task 4 Gateway gap confirmed; no diff | T1/T2 | no |
| `019e7bdf-a187-7f02-9cce-0f9d605ac9c9` | Provider Spine text-stream parser coverage audit | `/Users/jiyurun/.codex/worktrees/9136/New project` | completed; no P0/P1 implementation gaps; no diff | T1/T2 | no |
| `019e7bce-bacf-76e3-98f3-1e53fffe1377` | Promotion-readiness gate review | `/Users/jiyurun/.codex/worktrees/9f7f/New project` | completed; no P0/P1/P2 findings on `3ddcc45` | T0/T1 | no |
| `019e7bba-71ca-71d0-84cc-78424d4d07ab` | Integration review / governance slices | `/Users/jiyurun/.codex/worktrees/47a6/New project` | completed; no P0/P1/P2 findings on `1872ca9` | T0/T1 | no |
| `019e7bb0-bf95-74f3-a935-1e89644bd417` | PCM bridge app flash ADR docs-only | `/Users/jiyurun/.codex/worktrees/42b1/New project` | completed; committed `a031f3d` | T0/T1 | no |
| `019e7ba8-2bec-7f12-83ce-8b0fd1cc06c9` | Professional V21 evidence adapter readiness | `/Users/jiyurun/.codex/worktrees/ab7a/New project` | completed; committed `fc61793` | T1/T2 | no |
| `019e7ba1-d81e-74c3-bd2e-a6191344085a` | Provider Spine / DeepSeek text-stream readiness | `/Users/jiyurun/.codex/worktrees/84d6/New project` | completed; committed `6b7fdf0` | T1/T2 | no |
| `019e7b99-141e-70a3-b0fd-c5dd38b5cab5` | StackChan hardware mainline diagnostic consolidation | `/Users/jiyurun/.codex/worktrees/ddec/New project` | completed; committed `1e38804` | T1/T2 | no |
| `019e7b91-537b-7233-b682-276f77e1b871` | Mainline NVS closure review | `/Users/jiyurun/.codex/worktrees/88f4/New project` | completed; no diff | T1 | no |
| `019e7b80-25e2-73e3-9e6c-05112ebbf82f` | Governance review | `/Users/jiyurun/.codex/worktrees/0caf/New project` | completed | T0 | no |
| `019e7b80-25e2-73e3-9e6c-05001307bfdf` | Mainline recovery planning | `/Users/jiyurun/.codex/worktrees/68a6/New project` | completed | T0 | no |
| `019e7b80-25e5-72f3-8bef-13a9ceb44a9e` | Hardware-window ADR prep | `/Users/jiyurun/.codex/worktrees/49ef/New project` | completed | T0 | no |
| `019e740c-bec6-79c0-a717-9f6191e4a750` | Earlier M3 execution | `/Users/jiyurun/Documents/New project` | paused | report-only until resumed | no |
| `019e797b-6a67-7b31-93c8-7eb2ad8610b2` | Baseline read-only review | `/Users/jiyurun/Documents/New project` | completed | T0 | no |
| `019e7837-4584-7410-917c-97d2c530f7ef` | API procurement/research | `/Users/jiyurun/Documents/New project` | idle | T0 | no |

Rules:

- Only the control tower may write the main worktree until it explicitly hands
  off a single implementation slice.
- Completed detached Codex worktrees remain read-only evidence unless the
  control tower creates a new branch for implementation.
- A paused execution thread may report status only. It must not resume edits
  from stale branch context.
- Hardware-write authority is never implied by a thread title. It requires a
  foreground `codex/a21-hardware-window-*` branch, clean tree, explicit port,
  explicit confirmation token, and a passing `control_guard` receipt.
- Provider execution authority is never implied by provider-thread ownership.
  It requires a separate T4 declaration, local env confirmation, redacted
  receipt path, and no key or prompt/output text in saved reports.

## Accepted Handoffs

### PRD Next-Slice Audit Thread

Opened by the control tower after accepting Provider Spine Task 5 verification
at `ac2ab42`.

Evidence:

- Audit thread: `019e7c09-98d6-75d0-85a4-f0bf63cd4e3b`.
- Audit worktree: `/Users/jiyurun/.codex/worktrees/ed15/New project`.
- Starting branch: `codex/a21-integration-governance-slices`.
- Starting HEAD:
  `ac2ab42 docs(control): record provider spine verification`.
- Scope: compare `docs/prd/A21_PRD.md`, current code, and engineering docs
  after Provider Spine completion, then recommend the next PRD-aligned slice.
- Required output: current branch/HEAD/dirty state, P0/P1/P2 PRD gap matrix,
  recommended next slice, file scope, verification commands, maximum tool tier,
  forbidden actions, and whether a new implementation thread is needed.
- Maximum tier: T0/T1.
- Forbidden: file edits, commits, pushes, Gateway runtime or service startup,
  provider/V21 execute, durable provider reports with payloads, firmware/NVS/
  flash/raw upload/serial writes, `/v1/devices/control`, and physical device
  paths.

Decision:

- Accept the audit result: Provider Spine Task 1 through Task 5 are complete in
  the current integration baseline, and the next PRD pressure is no longer the
  Fast Companion Gateway placeholder boundary.
- The next implementation slice is `Provider Latency Bench Scaffold / Fast
  Companion Candidate Report`.
- The slice must remain T1/T2 and produce a redacted candidate-chain report
  shape only. It must not execute provider/V21 calls, start Gateway/runtime,
  write durable provider payload reports, touch firmware/NVS/flash/serial
  paths, call `/v1/devices/control`, or use physical device paths.
- Real provider execution, real V21 execution, and StackChan physical
  acceptance remain separate T4/T6 windows requiring explicit authorization.

Audit handoff evidence:

- Audit branch/head: detached HEAD at
  `ac2ab42 docs(control): record provider spine verification`, decorated by
  `codex/a21-integration-governance-slices`.
- Audit dirty files: none.
- P0 gap: A21 has a Gateway Fast Companion placeholder trace, but lacks a
  unified provider-latency candidate report/bench proving ASR/provider/TTS/
  downlink/playback timing across mock/fixture/host-only chains.
- P0 gap: current provider smoke, local voice loopback, and StackChan fast
  companion receipts are partial evidence; they do not yet converge into the
  PRD benchmark contract with p50/p95/p99, redacted network/proxy metadata,
  fallback/failure counts, and `promotion_gate=not_production`.
- P1 gap: physical StackChan acceptance remains future T6 evidence and should
  not be opened before the latency evidence scaffold exists.
- P1 gap: barge-in is covered at mock/Gateway level, but not yet production
  VAD/AEC/full-duplex/physical speaker proof.
- P2 gap: professional mode and V21 evidence path are sufficiently guarded for
  this decision; real V21 execute is still T4.
- Audit verification passed:
  `go test ./internal/gateway -run 'FastCompanion|RealtimeSessionStartRejectsProfessional|ProfessionalModeUsesV21|AudioWSBargeIn' -count=1`.
- Audit verification passed:
  `go test ./internal/providers -run 'ProviderCatalog|ProviderProfile|TextStream|ProviderSmoke' -count=1`.
- Audit verification passed:
  `go test ./internal/app -run 'LocalVoiceLoopback|StackChanFastCompanion|RunProviderSmoke|GatewayServerFromEnv|RunV21AdapterSmoke|RunDoctor' -count=1`.
- Audit verification passed: `go run ./cmd/a21 namespace-audit`.
- Audit verification passed: `git diff --check`.
- Audit did not change files, commit, push, start Gateway/runtime, execute
  provider/V21 calls, generate durable payload reports, touch firmware/NVS/
  flash/serial paths, call `/v1/devices/control`, or use physical device paths.

### Provider Latency Bench Scaffold Implementation Thread

Opened by the control tower after accepting the PRD next-slice audit.

Evidence:

- Implementation thread: `019e7c0d-f7a0-7323-8413-e3e2aac53a94`.
- Implementation worktree: `/Users/jiyurun/.codex/worktrees/acc7/New project`.
- Starting integration HEAD:
  `52c64d2 docs(control): track prd next-slice audit`.
- Target branch: `codex/a21-provider-latency-bench-scaffold`.
- Scope: TDD implementation of `provider-latency-bench` mock/fixture scaffold
  for Fast Companion candidate-chain reports.
- Expected report fields: `trace_id`, `session_id`, `device_id`, execution
  mode, redacted network/proxy metadata, provider profile/family labels, ASR
  first partial, provider first byte, provider first content, TTS first audio,
  downlink first frame, device playback start, barge-in stop/cancel
  placeholders, p50/p95/p99 summary, fallback/failure counts, and
  `promotion_gate=not_production`.
- Expected files:
  `internal/app/provider_latency_bench*.go`, `internal/app/app.go` and tests,
  `docs/engineering/A21_PROVIDER_BENCHMARKS.md`,
  `docs/engineering/LATENCY_BUDGET.md`, `docs/engineering/DOCTOR.md`, and
  optionally `Makefile`.
- Maximum tier: T1/T2.
- Forbidden: provider `--execute`, V21 execute, Gateway runtime/service
  startup, durable provider reports with payloads, firmware/NVS/flash/raw
  upload/serial writes, `/v1/devices/control`, physical device paths,
  production dependency additions, secrets, prompt/transcript/provider output/
  reasoning text, full URLs, proxy URLs, or local paths in reports.

Decision:

- Accept the implementation handoff into the integration branch at
  `4532df8 feat(app): add provider latency bench scaffold`.
- The accepted command is a T1/T2 mock/fixture-only report scaffold. It rejects
  `--execute`, sets `promotion_gate=not_production`, marks
  `provider_executed=false`, `v21_executed=false`, and
  `hardware_executed=false`, and stores no prompt, transcript, provider output,
  reasoning text, credential values, full provider URLs, proxy URLs, raw
  payloads, or full local fixture paths.
- Do not open T4 provider/V21 or T6/T7/T8 hardware windows from this
  implementation slice.

Acceptance evidence:

- Implementation thread handoff branch:
  `codex/a21-provider-latency-bench-scaffold`.
- Implementation handoff dirty files: `Makefile`,
  `docs/engineering/A21_PROVIDER_BENCHMARKS.md`,
  `docs/engineering/DOCTOR.md`, `docs/engineering/LATENCY_BUDGET.md`,
  `internal/app/app.go`, `internal/app/app_test.go`, and
  `internal/app/provider_latency_bench.go`.
- Control tower reapplied the handoff diff onto
  `codex/a21-integration-governance-slices` after
  `28171ad docs(control): open provider latency bench slice`.
- Verification passed:
  `go test ./internal/app -run 'ProviderLatencyBench|LatencyBench|ProviderSmoke|LocalVoiceLoopback' -count=1`.
- Verification passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke|ProviderCatalog' -count=1`.
- Verification passed: `go run ./cmd/a21 namespace-audit`.
- Verification passed: `git diff --check`.
- Verification passed:
  `go run ./cmd/a21 provider-latency-bench --provider deepseek --iterations 2`.
- Verification passed: `make verify`.
- Post-commit default gate passed: `go run ./cmd/a21 preflight`.
- Post-commit default gate passed with a non-blocking firmware artifact warning:
  `go run ./cmd/a21 doctor`. The warning was
  `firmware_current_artifact_missing` for commit `4532df8b3780`; this slice did
  not build or promote firmware artifacts.
- Post-commit promotion gate result:
  `go run ./cmd/a21 promotion-readiness` reported `review_ready=true` and
  `dirty_file_count=0`, while external promotion remains blocked because the
  repository has no configured remote, target remote, or target branch.
- No provider execute, V21 execute, Gateway runtime/service startup,
  firmware/NVS/flash/raw upload/serial write, `/v1/devices/control`, or
  physical device path was used.

Residual gaps:

- The scaffold is report-shape evidence only. ASR, real provider, TTS, Gateway
  runtime, LAN, physical StackChan playback, and barge-in timing remain
  unmeasured until a separately authorized T4/T6 window.

### Provider Latency Bench Post-Commit Review

Accepted from thread `019e7c1d-4678-7eb2-8666-9c5c331585d0`.

Evidence:

- Review worktree: `/Users/jiyurun/.codex/worktrees/0a27/New project`.
- Review HEAD: `8bc41f6 docs(control): accept provider latency bench scaffold`.
- Review dirty files: none.
- Review result: no P0 findings and no P1 findings.
- Review P2 finding: `provider-latency-bench` used `host_baseline` while
  `A21_PROVIDER_BENCHMARKS.md` defines the canonical execution mode vocabulary
  as `mock`, `fixture`, `host_loopback`, `physical_stackchan`, or
  `external_lab`.
- Control tower fixed the P2 at
  `d03365f fix(app): align provider latency mode vocabulary`.
- The fix changes CLI help, validation, and report output to canonical
  `host_loopback`, rejects deprecated `host_baseline`, and adds tests for both
  behaviors.

Review verification evidence:

- Review thread passed:
  `git diff --check 28171ad..8bc41f6`.
- Review thread passed:
  `go test ./internal/app -run 'ProviderLatencyBench|LatencyBench|ProviderSmoke|LocalVoiceLoopback' -count=1`.
- Review thread passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke|ProviderCatalog|Network' -count=1`.
- Review thread passed:
  `go run ./cmd/a21 provider-latency-bench --provider deepseek --iterations 2`.
- Review thread confirmed synthetic secret/path redaction had no matches.
- Review thread confirmed `go run ./cmd/a21 provider-latency-bench --execute`
  was rejected.
- Review thread passed: `go run ./cmd/a21 namespace-audit`.
- Review thread passed: `make verify`.
- Review thread passed: `go run ./cmd/a21 preflight`.

Control verification after the P2 fix:

- Verification passed:
  `go test ./internal/app -run 'ProviderLatencyBench|LatencyBench|ProviderSmoke|LocalVoiceLoopback' -count=1`.
- Verification passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke|ProviderCatalog|Network' -count=1`.
- Verification passed:
  `go run ./cmd/a21 provider-latency-bench --provider mock --mode host_loopback --iterations 1`.
- Verification passed:
  `go run ./cmd/a21 provider-latency-bench --mode host_baseline` rejected the
  deprecated spelling with `--mode must be mock, fixture, or host_loopback`.
- Verification passed: `go run ./cmd/a21 namespace-audit`.
- Verification passed: `git diff --check`.
- Verification passed: `make verify`.
- Post-fix default gate passed: `go run ./cmd/a21 doctor`. It still reports the
  non-blocking `firmware_current_artifact_missing` warning for commit
  `d03365f98c8f`; this slice did not build or promote firmware artifacts.
- Post-fix promotion gate result:
  `go run ./cmd/a21 promotion-readiness` reported `review_ready=true` and
  `dirty_file_count=0`, while external promotion remains blocked because the
  repository has no configured remote, target remote, or target branch.
- A transient `reserved_port_in_use` doctor result on port `127.0.0.1:21080`
  was not reproducible; `lsof` found no listener on `21080`, `21081`, or
  `21073`, and the subsequent doctor run passed.
- No provider execute, V21 execute, Gateway runtime/service startup,
  firmware/NVS/flash/raw upload/serial write, `/v1/devices/control`, or
  physical device path was used.

### Provider Latency Fixture Schema Implementation

Accepted from thread `019e7c25-ec43-7591-9954-5227c7288e89`.

Evidence:

- Implementation branch: `codex/a21-provider-latency-fixture-schema`.
- Implementation worktree:
  `/Users/jiyurun/.codex/worktrees/6b8d/New project`.
- Starting integration HEAD:
  `b8d46f8 docs(control): record provider latency post review`.
- Control tower opened the slice at
  `44d6c76 docs(control): open provider fixture schema slice`.
- Scope: add no-execute fixture metadata/report contract coverage for fixture
  identity, audio format, sample rate, channels, duration/sample/window
  metadata, and redacted structured findings for invalid fixture metadata.
- Expected files: `internal/app/provider_latency_bench.go`,
  `internal/app/app_test.go`, `docs/engineering/A21_PROVIDER_BENCHMARKS.md`,
  `docs/engineering/LATENCY_BUDGET.md`, and `docs/engineering/DOCTOR.md`.
- Maximum tier: T1/T2.
- Forbidden: provider `--execute`, V21 execute, Gateway runtime/service
  startup, durable provider reports with payloads, firmware/NVS/flash/raw
  upload/serial writes, `/v1/devices/control`, physical device paths,
  production dependency additions, secrets, raw PCM, base64 audio, prompt,
  transcript, provider output, reasoning text, full URLs, proxy URLs, or full
  local paths in reports.

Decision:

- Accept the fixture schema handoff into the integration branch at
  `f69c6b8 feat(app): add provider fixture metadata contract`.
- The accepted report contract still stores only fixture basenames. Redacted
  JSON sidecars may contribute `a21.provider_latency_fixture.v1` metadata:
  fixture identity, audio format, sample rate, channel count, duration, sample
  count, window length, and window count.
- Invalid, missing, oversized, unknown-field, payload-bearing, unsafe, or
  trailing-content sidecars produce a fixed structured finding
  `fixture_sidecar_invalid` and increment `failure_count` without panicking or
  echoing raw file contents, full paths, full URLs, proxy URLs, credentials, or
  payload text.
- The control tower added an extra trailing-payload regression test before
  committing the handoff, so a valid JSON object followed by a second prompt or
  payload object is also rejected.
- This remains report-shape evidence only. It does not authorize real
  provider/V21/Gateway/hardware measurement.

Acceptance evidence:

- Control verification passed:
  `go test ./internal/app -run 'ProviderLatencyBench|LatencyBench|ProviderSmoke|LocalVoiceLoopback' -count=1`.
- Control verification passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke|ProviderCatalog|Network' -count=1`.
- Control verification passed: `go run ./cmd/a21 namespace-audit`.
- Control verification passed: `git diff --check`.
- Control verification passed: `make verify`.
- Control dry run passed without provider/V21/hardware execution:
  `go run ./cmd/a21 provider-latency-bench --provider mock --fixture /tmp/a21-secret-path-fixture.json --iterations 1`.
  The report kept only `a21-secret-path-fixture.json`, returned
  `failure_count=1`, and emitted the fixed `fixture_sidecar_invalid` finding.
- No provider execute, V21 execute, Gateway runtime/service startup,
  firmware/NVS/flash/raw upload/serial write, `/v1/devices/control`, physical
  device path, durable provider payload report, production dependency, secret,
  prompt/transcript/provider output/reasoning payload, full URL, proxy URL, or
  full local path was used or saved.

Residual gaps:

- Real ASR/provider/TTS/downlink/playback latency, LAN behavior, physical
  StackChan acceptance, and barge-in timing remain unmeasured until a separately
  authorized T4/T6 window.

### Provider Fixture Metadata Post-Commit Review

Accepted from thread `019e7c30-7329-7a32-996e-0566c9746e5d`.

Evidence:

- Review worktree:
  `/Users/jiyurun/.codex/worktrees/1181/New project`.
- Review HEAD:
  `0e46761 docs(control): accept provider fixture metadata slice`.
- Review worktree state: detached HEAD, clean.
- Reviewed commits:
  `f69c6b8 feat(app): add provider fixture metadata contract` and
  `0e46761 docs(control): accept provider fixture metadata slice`.
- Review result: no P0 findings, no P1 findings, and no P2 findings.

Review verification evidence:

- Review passed: `git diff --check f69c6b8^..0e46761`.
- Review passed:
  `go test ./internal/app -run 'ProviderLatencyBench|LatencyBench|ProviderSmoke|LocalVoiceLoopback' -count=1`.
- Review passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke|ProviderCatalog|Network' -count=1`.
- Review passed: `go run ./cmd/a21 namespace-audit`.
- Review passed: `make verify`.
- Review passed: `go run ./cmd/a21 preflight`.
- Review confirmed `provider-latency-bench --execute` is rejected.
- Review confirmed `--fixture` stores only the basename and invalid JSON
  sidecars report only fixed `fixture_sidecar_invalid` findings with
  `failure_count=1` and execution flags false.
- Review worktree `go run ./cmd/a21 doctor` returned nonzero for isolated
  environment reasons: reserved port `127.0.0.1:21080` plus missing local
  PlatformIO/tooling artifact state in the detached review worktree. Control
  tower rechecked the main worktree and found no listener on `21080`, `21081`,
  or `21073`; `go run ./cmd/a21 doctor` passed on the integration branch with
  only the expected non-blocking `firmware_current_artifact_missing` warning.
- Review worktree `go run ./cmd/a21 promotion-readiness` returned nonzero
  because the worktree was detached at `0e46761`, so `review_ready=false` in
  that isolated environment. Control tower separately verified the integration
  branch at `cfc1658` reports `review_ready=true`, `dirty_file_count=0`, and
  external promotion blocked only by missing remote, target remote, and target
  branch.

Decision:

- Accept the review as closure for the fixture metadata contract slice.
- No immediate control-tower code or docs fix is required for `f69c6b8` or
  `0e46761`.
- Treat detached-worktree doctor and promotion-readiness differences as review
  environment evidence, not product or contract regressions.
- `promotion-readiness` assertions that require `review_ready=true` must be
  run from the integration branch's main worktree or another worktree with the
  candidate branch checked out. A detached review worktree may record branch,
  ancestor, toolchain, or environment differences, but a detached-HEAD
  `review_ready=false` result is not evidence of a provider fixture contract
  regression by itself.

Next queue:

- Opened implementation thread `019e7c36-4617-73a1-aba0-1d35acc26efe` for the
  narrow T1/T2 Provider Fixture Sidecar Hardening slice.
- Implementation worktree:
  `/Users/jiyurun/.codex/worktrees/38a4/New project`.
- Scope: dedicated tests around oversized fixture sidecars and unknown-field
  sidecars, plus documentation clarifying that promotion-readiness
  `review_ready=true` should be asserted on the integration branch, not on
  detached review worktrees.
- Expected files: `internal/app/app_test.go`, and optionally
  `docs/engineering/A21_CONTROL_LEDGER.md` or `docs/engineering/DOCTOR.md`.
- Forbidden: production code changes unless tests prove a gap, provider
  `--execute`, V21 execute, Gateway runtime/service startup, durable provider
  reports with payloads, firmware/NVS/flash/raw upload/serial write,
  `/v1/devices/control`, physical device paths, production dependency
  additions, secrets, or provider payloads.
- Keep real provider/V21/Gateway/hardware latency measurement reserved for a
  separately authorized T4/T6 window.

### Provider Fixture Sidecar Hardening Acceptance

Accepted from thread `019e7c36-4617-73a1-aba0-1d35acc26efe`.

Evidence:

- Implementation branch: `codex/a21-provider-fixture-sidecar-hardening`.
- Implementation worktree:
  `/Users/jiyurun/.codex/worktrees/38a4/New project`.
- Implementation handoff state: branch dirty only with the intended
  `internal/app/app_test.go` and `docs/engineering/A21_CONTROL_LEDGER.md`
  changes; no production code changes.
- Accepted integration commit:
  `a4e3a6f test(app): harden provider fixture sidecar rejection`.
- Added dedicated tests for oversized fixture sidecars and unknown-field
  sidecars. Both assert fixed `fixture_sidecar_invalid` findings,
  `failure_count=1`, basename-only fixture identity, nil invalid metadata,
  false provider/V21/hardware execution flags, and no payload, path,
  prompt/transcript/raw PCM/base64/full URL/proxy/credential leakage.
- Clarified that `promotion-readiness review_ready=true` assertions must be
  taken from the integration branch main worktree or a checked-out candidate
  branch, not from detached review worktrees.

Control-tower verification on the integration branch:

- Passed:
  `go test ./internal/app -run 'ProviderLatencyBench|LatencyBench|ProviderSmoke|LocalVoiceLoopback' -count=1`.
- Passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke|ProviderCatalog|Network' -count=1`.
- Passed: `go run ./cmd/a21 namespace-audit`.
- Passed: `git diff --check`.
- Passed: `make verify`.
- Passed after commit `a4e3a6f`: `go run ./cmd/a21 preflight`.
- Passed after commit `a4e3a6f`: `go run ./cmd/a21 doctor`, with only the
  expected non-blocking `firmware_current_artifact_missing` warning.
- `go run ./cmd/a21 promotion-readiness` after commit `a4e3a6f` reported
  `review_ready=true`, `dirty_file_count=0`, and external promotion blocked
  only by missing remote, target remote, and target branch.

Decision:

- Accept the sidecar hardening slice as T1/T2 closure.
- Keep real provider/V21/Gateway/hardware latency evidence outside this slice
  until a separately authorized T4/T6 window.

Next queue:

- Opened post-commit review thread
  `019e7c41-1842-7a30-a185-aafc73ba2d73` for `a4e3a6f^..447b917`.
- Review thread title:
  `A21 Provider Fixture Sidecar：Post-Commit Review`.
- Review worktree:
  `/Users/jiyurun/.codex/worktrees/8922/New project`.
- Scope: read-only P0/P1/P2 review of sidecar hardening tests, ledger
  accuracy, namespace safety, redaction coverage, and no production/provider/
  V21/Gateway/hardware execution regression.
- Forbidden: edits, commits, provider `--execute`, V21 execute, Gateway runtime
  or service startup, durable provider reports with payloads, firmware/NVS/
  flash/raw upload/serial writes, real `/v1/devices/control`, physical device
  paths, production dependency additions, secrets, or provider payloads.

Review result:

- Thread `019e7c41-1842-7a30-a185-aafc73ba2d73` reported no P0, P1, or P2
  findings for `a4e3a6f^..447b917`.
- Review confirmed the sidecar hardening tests exercise the
  `provider-latency-bench --fixture` JSON sidecar path. Oversized sidecars hit
  the existing `providerLatencyFixtureSidecarMaxBytes` guard, and unknown-field
  sidecars hit the existing `DisallowUnknownFields` decoder path.
- Review confirmed the shared assertion covers fixed `fixture_sidecar_invalid`,
  `failure_count=1`, basename-only fixture ID, nil invalid metadata, false
  provider/V21/hardware execution flags, false redaction storage flags, and no
  full path, payload, prompt/transcript/raw PCM/base64/full URL/proxy/
  credential-like leakage.
- Review confirmed the slice changed no production Go code, added no
  dependency, introduced no namespace pollution, and did not restore
  `host_baseline`.

Review verification evidence:

- Passed: `git diff --check a4e3a6f^..447b917`.
- Passed:
  `go test ./internal/app -run 'ProviderLatencyBench|LatencyBench|ProviderSmoke|LocalVoiceLoopback' -count=1`.
- Passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke|ProviderCatalog|Network' -count=1`.
- Passed: `go run ./cmd/a21 namespace-audit`.
- Passed: `make verify`.
- Passed: `go run ./cmd/a21 doctor`.
- Passed: `go run ./cmd/a21 preflight` on review rerun.
- First review `preflight` attempt saw transient `127.0.0.1:21080` occupancy;
  immediate review `lsof` found no listener, and control tower separately
  verified no main-worktree listener plus a passing `preflight`.
- Review `promotion-readiness` returned `review_ready=false` only because the
  review worktree was detached at `447b917`; this matches the documented
  detached-worktree caveat and is not a sidecar hardening regression.

Post-review decision:

- Keep the sidecar hardening slice accepted.
- Do not block control-tower progress on this slice.
- Keep provider/V21/Gateway/hardware latency evidence reserved for a separately
  authorized T4/T6 window.

### AgentTaskProvider Bridge Scaffold Implementation Thread

Opened by the control tower after provider latency and fixture evidence
hardening closed.

Evidence:

- Implementation thread: `019e7c48-9fc7-7ff0-8563-965fe9da9f72`.
- Implementation thread title:
  `A21 AgentTask Bridge：Scaffold Implementation`.
- Implementation worktree:
  `/Users/jiyurun/.codex/worktrees/04c0/New project`.
- Starting branch: `codex/a21-integration-governance-slices`.
- Starting control HEAD:
  `a1742ae docs(control): record fixture sidecar post review`.
- Target branch: `codex/a21-agent-task-bridge-scaffold`.
- Scope: PRD Phase 5 T1/T2 scaffold for `AgentTaskProvider`,
  `AgentTaskRequest`, `AgentTaskEvent`, fake/smoke event consumption, and
  A21-owned semantic event mapping.
- User architecture boundary: external agents are an explicit Agent I/O Layer
  selected by frontend/config; A21 does not route automatically, does not let
  external agents become the realtime first-response path, and does not hand
  them firmware, provider-env, V21-internal, Gateway-runtime, or physical
  device control.
- Expected files: `internal/providers/contracts.go`, optional
  `internal/providers/agent_task*.go`, provider tests, optional
  `internal/app` smoke CLI/tests if the implementation keeps CLI parity, and
  optional `docs/engineering/PHASE5_AGENT_TASK_BRIDGE.md`, `DOCTOR.md`, or
  `OBSERVABILITY.md`.
- Forbidden: real Hermes/MiMo/OpenClaw or other agent runtime startup,
  provider `--execute`, V21 execute, Gateway runtime or service startup,
  durable reports with agent payloads, firmware/NVS/flash/raw upload/serial
  writes, real `/v1/devices/control`, physical device paths, production
  dependency additions, secrets, provider payloads, or agent payload text.

Acceptance gates:

- TDD red/green evidence for the new AgentTask contract and event mapper.
- `go test ./internal/providers -run 'AgentTask|ProviderCatalog|ProviderProfile' -count=1`.
- If CLI is touched:
  `go test ./internal/app -run 'AgentTask|ProviderSmoke|RunDoctor' -count=1`.
- `go run ./cmd/a21 namespace-audit`.
- `git diff --check`.
- `make verify` if the slice touches shared app CLI or docs.

Decision:

- Accept the implementation handoff into the integration branch at
  `560df00 feat(providers): add agent task bridge scaffold`.
- The accepted scaffold is provider-package T1/T2 only. It defines the
  AgentTask contract, fake `sse`/`http`/`stdio` event streams, an A21-owned
  semantic report shape, redaction markers, and forbidden request findings.
- The control tower added a stricter catalog guard before acceptance:
  `A21_PROVIDER_PRIMARY=hermes_agent` or `mimo_agent` now produces
  `provider_agent_task_primary` and does not mark the agent profile selected.
  Agent-task readiness remains controlled only by `A21_AGENT_PROVIDER_PRIMARY`.
- Do not open real Hermes/MiMo/OpenClaw runtime, provider execution, V21
  execution, Gateway runtime, durable agent-payload reports, firmware/NVS/
  flash/raw upload/serial writes, `/v1/devices/control`, or physical device
  paths from this scaffold.

Acceptance evidence:

- Implementation branch: `codex/a21-agent-task-bridge-scaffold`.
- Implementation handoff dirty files:
  `docs/engineering/DOCTOR.md`, `docs/engineering/OBSERVABILITY.md`,
  `docs/engineering/PROTOCOL.md`,
  `docs/engineering/PHASE5_AGENT_TASK_BRIDGE.md`,
  `internal/providers/agent_task.go`,
  `internal/providers/agent_task_test.go`,
  `internal/providers/catalog.go`,
  `internal/providers/catalog_test.go`, and
  `internal/providers/smoke.go`.
- Implementation thread reported TDD red on missing `AgentTask*` APIs, then
  green after the minimal scaffold.
- Implementation thread caught and fixed an over-broad provider-env safety
  scan so ordinary A21 semantic context such as `a21_mode` is not rejected.
- Control tower independently verified before acceptance:
  `go test ./internal/providers -run 'AgentTask|ProviderCatalog|ProviderProfile' -count=1`.
- Control tower independently verified: `go test ./internal/providers -count=1`.
- Control tower independently verified: `go run ./cmd/a21 namespace-audit`.
- Control tower independently verified: `git diff --check`.
- Control tower independently verified: `make verify`.
- Semantic reports store event kind, final marker, trace/session identity,
  text length, and redaction markers only. They do not store external agent
  text, tool payloads, credentials, full URLs, local paths, provider env
  values, or raw external-agent control payloads.
- No real Hermes/MiMo/OpenClaw runtime, provider execution, V21 execution,
  Gateway runtime/service startup, durable agent-payload report, firmware/NVS/
  flash/raw upload/serial write, `/v1/devices/control`, physical device path,
  production dependency, secret, provider payload, or agent payload was added
  or run.

Remaining gap:

- Real adapter/runtime work remains a future T4/T6 window with explicit
  authorization, redacted receipts, and runtime gates.

### AgentTaskProvider Bridge Post-Commit Review Thread

Opened by the control tower after accepting the AgentTaskProvider Bridge
scaffold.

Evidence:

- Review thread: `019e7c57-bd71-72d0-9cf8-ec9674f41bf0`.
- Review thread title: `A21 AgentTask Bridge：Post-Commit Review`.
- Review worktree: `/Users/jiyurun/.codex/worktrees/7077/New project`.
- Starting branch: `codex/a21-integration-governance-slices`.
- Starting HEAD:
  `aaeda0e docs(control): accept agent task bridge scaffold`.
- Scope: read-only P0/P1/P2 review of implementation commit `560df00` and
  ledger acceptance commit `aaeda0e`.
- Required focus: no runtime/provider/V21/Gateway/hardware escalation,
  no namespace pollution, no raw agent/provider/tool payload leakage, no
  `A21_PROVIDER_PRIMARY` selection of agent-task profiles, accurate ledger
  evidence, and adequate tests for the T1/T2 scaffold.
- Forbidden: file edits, commits, pushes, provider `--execute`, V21 execute,
  Gateway runtime/service startup, durable reports with payloads, firmware/NVS/
  flash/raw upload/serial writes, real `/v1/devices/control`, physical device
  paths, production dependency additions, secrets, provider payloads, or agent
  payloads.

Post-acceptance control-tower gates:

- Passed: `go run ./cmd/a21 preflight`.
- Passed with expected warning: `go run ./cmd/a21 doctor`. Warning:
  `firmware_current_artifact_missing` for current commit `aaeda0e`; this docs/
  provider scaffold slice did not build or promote firmware artifacts.
- `go run ./cmd/a21 promotion-readiness` returned nonzero with
  `review_ready=true`, `dirty_file_count=0`, `external_promotion_ready=false`,
  and blockers `promotion_remote_missing`, `promotion_target_remote_missing`,
  and `promotion_target_branch_missing`.

Review result:

- Thread `019e7c57-bd71-72d0-9cf8-ec9674f41bf0` reported no P0 or P1
  findings for `560df00^..aaeda0e`.
- P2 finding: adding `hermes_agent` and `mimo_agent` to the shared provider
  catalog let the voice-provider factory surface them as known unavailable
  voice providers when `A21_PROVIDER_PRIMARY` targeted an agent profile,
  despite the catalog already rejecting them as voice primary choices.
- P2 finding: `docs/engineering/OBSERVABILITY.md` described AgentTask reports
  as package-level T1/T2 only, but also listed `agent_task.*` markers under
  current mock trace events, which overstated current runtime observability.
- Review verification passed:
  `go test ./internal/providers -run 'AgentTask|ProviderCatalog|ProviderProfile' -count=1`.
- Review verification passed: `go test ./internal/providers -count=1`.
- Review verification passed: `go run ./cmd/a21 namespace-audit`.
- Review verification passed: `git diff --check 560df00^..aaeda0e`.
- Review verification passed: `make verify`.
- Review worktree stayed clean and did not edit files, commit, push, execute
  providers or V21, start Gateway runtime, create durable payload reports,
  touch firmware/NVS/flash/serial paths, call `/v1/devices/control`, or use
  physical device paths.

Post-review decision:

- Accept the review.
- Close both P2 findings in control at
  `598c0d9 fix(providers): reject agent profiles as voice primary`.
- The fix rejects `A21_PROVIDER_PRIMARY=hermes_agent` or `mimo_agent` in both
  `NewVoiceProviderFromEnv` and Gateway selected voice-provider mode as
  `invalid_agent_task_primary`, without echoing the raw agent profile name in
  health JSON or error text.
- The fix moves AgentTask marker names from current mock runtime trace events
  into reserved future semantic marker vocabulary.
- Verification passed:
  `go test ./internal/providers -run 'AgentTask|ProviderCatalog|ProviderProfile|VoiceProviderFromEnv|GatewayVoiceProviderFromEnv' -count=1`.
- Verification passed: `go test ./internal/providers -count=1`.
- Verification passed: `go run ./cmd/a21 namespace-audit`.
- Verification passed: `git diff --check`.
- Verification passed: `make verify`.
- Continue to keep real Hermes/MiMo/OpenClaw runtime, provider execution, V21
  execution, Gateway runtime, and hardware/device-control paths behind future
  explicit windows.

### Fast Companion Hybrid Boundary Audit

Accepted from thread `019e7be6-bca3-71f2-9770-857b9da48b67`.

Evidence:

- Audit branch: `codex/a21-fast-companion-hybrid-boundary-audit`.
- Audit worktree: `/Users/jiyurun/.codex/worktrees/0d72/New project`.
- Audit HEAD: `3b9f05d08ad0`.
- Audit dirty, staged, and untracked files: none.
- Audit result: no P0/P1 safety regression.
- Existing `internal/app` coverage includes `local-voice-loopback` and
  `stackchan-fast-companion-turn`, which stitch local/front-end evidence,
  `text_stream` provider boundaries, local TTS, and StackChan downlink receipts.
- Professional/realtime boundary is covered: `/v1/realtime/session` rejects
  `professional`, and `/v1/mock-turn` professional mode uses V21 adapter
  evidence fields with honest fallback.
- Gateway default mock safety is covered; provider execution remains opt-in.
- Task 4 is not fully covered: there is no Gateway-level Fast Companion Hybrid
  routing boundary, no `FastCompanion`/`text_stream` Gateway route/test, no
  unified Gateway waterfall for `asr.first_partial`, provider first byte,
  provider first content, TTS first audio, downlink first frame, and playback
  start placeholders, and no `docs/engineering/PHASE7H_FAST_COMPANION_HYBRID.md`.
- Audit verification passed:
  `go test ./internal/gateway -run 'FastCompanion|Professional|Realtime' -count=1`.
- Audit verification passed:
  `go test ./internal/app -run 'LocalVoiceLoopback|StackChanFastCompanion|GatewayServerFromEnv|ProviderSmoke|LocalTTS|LocalASR' -count=1`.
- Audit verification passed: `git diff --check`.
- Audit verification passed: `go run ./cmd/a21 namespace-audit`.
- Audit verification passed: `make verify`.
- Audit verification passed: `go run ./cmd/a21 preflight`.
- Audit verification passed: `go run ./cmd/a21 doctor`, with expected isolated
  worktree firmware warnings.
- No provider execute, V21 execute, Gateway runtime, durable provider report,
  firmware change, NVS execute, flash execute, raw upload, serial write,
  `/v1/devices/control`, or physical device path was touched.

Decision:

- Accept the audit as proof that Task 4 needs a follow-up implementation slice.
- Keep existing app-level fast companion receipts as evidence, not as proof of
  Gateway-level Task 4 completion.
- Open a separate implementation thread from a clean integration HEAD. Keep it
  T1/T2 only and forbid provider/V21 execute, Gateway runtime, firmware/NVS/
  flash/serial writes, and physical device control.

### Fast Companion Hybrid Gateway Boundary Implementation Thread

Opened by the control tower after accepting the Fast Companion Hybrid Boundary
Audit.

Evidence:

- Implementation thread: `019e7bed-4e1e-7512-8f21-1647b2357c00`.
- Implementation worktree: `/Users/jiyurun/.codex/worktrees/80c8/New project`.
- Starting integration HEAD:
  `94d87c3 docs(control): accept fast companion audit`.
- Target branch: `codex/a21-fast-companion-hybrid-boundary`.
- Scope: Task 4 of
  `docs/superpowers/plans/2026-05-31-provider-spine-mainline.md`.
- Expected files:
  `internal/gateway/server.go`, `internal/gateway/server_test.go`,
  `docs/engineering/PHASE7A_AUDIO_INGRESS.md`, and
  `docs/engineering/PHASE7H_FAST_COMPANION_HYBRID.md`.
- The implementation thread must use TDD and keep the diff narrow around the
  Gateway-level boundary and trace shape.
- Maximum tier: T1/T2.
- Forbidden: provider/V21 execute, Gateway runtime or service startup, durable
  provider reports with payloads, firmware changes, NVS execute, flash execute,
  raw upload, serial writes, `/v1/devices/control`, and physical device paths.
- Handoff branch: `codex/a21-fast-companion-hybrid-boundary`.
- Handoff HEAD: `94d87c393e2a`.
- Handoff dirty files:
  `internal/gateway/server.go`, `internal/gateway/server_test.go`,
  `docs/engineering/PHASE7A_AUDIO_INGRESS.md`,
  `docs/engineering/OBSERVABILITY.md`, and
  `docs/engineering/PHASE7H_FAST_COMPANION_HYBRID.md`.
- Handoff verification passed:
  `go test ./internal/gateway -run 'FastCompanionHybrid|RealtimeSessionStartRejectsProfessional|ProfessionalModeUsesV21' -count=1`.
- Handoff verification passed: `go test ./internal/gateway -count=1`.
- Handoff verification passed: `git diff --check`.
- Handoff verification passed: `go run ./cmd/a21 namespace-audit`.
- Handoff verification passed: `go run ./cmd/a21 preflight`.
- Handoff verification passed: `go run ./cmd/a21 doctor`, with existing
  firmware warnings only: no repo-local PlatformIO venv/core and no
  release-ledger artifact for the current commit.
- Handoff verification passed: `make verify`.
- Control tower integration tightened the route after handoff: only
  `companion` and `workmate` may enter, `local_audio.asr_provider` is required,
  and local audio counters must be non-negative.
- Control tower integration also corrected the trace catalog marker from stale
  `v21.first_result` wording to the actual `v21.query.first_result`.
- Control tower verification passed:
  `go test ./internal/gateway -run 'FastCompanionHybrid|RealtimeSessionStartRejectsProfessional|ProfessionalModeUsesV21' -count=1`.
- Control tower verification passed: `go test ./internal/gateway -count=1`.
- Control tower verification passed: `go test ./internal/app -count=1`.
- Control tower verification passed: `make verify`.
- Control tower verification passed: `go run ./cmd/a21 namespace-audit`.
- Control tower dry-run provider smoke passed without execution:
  `go run ./cmd/a21 provider-smoke --provider deepseek` returned `skipped`,
  `configured=false`, and `executed=false` because
  `A21_LAB_DEEPSEEK_API_KEY` is absent.
- Control tower dry-run provider smoke passed without execution:
  `go run ./cmd/a21 provider-smoke --provider bailian_dashscope` returned
  `skipped`, `configured=false`, and `executed=false` because
  `A21_DASHSCOPE_API_KEY` and `A21_DASHSCOPE_MODEL` are absent.
- Control tower preflight re-run passed after a transient reserved-port
  finding cleared; no Gateway runtime was started and no process was killed.
- Control tower doctor passed with the expected firmware-artifact warning for
  the current pre-commit HEAD.

Decision:

- Accept the implementation as Provider Spine Task 4's Gateway-level boundary
  slice.
- Keep it T1/T2-only: this acceptance does not open provider/V21 execute,
  Gateway runtime startup, firmware/NVS/flash/serial writes,
  `/v1/devices/control`, or physical device paths.
- Keep `PHASE7H_FAST_COMPANION_HYBRID.md` as the source of truth for this
  placeholder trace contract until a later ADR promotes real provider, TTS,
  downlink, or device playback timings.

### Fast Companion Hybrid Post-Commit Review Thread

Opened by the control tower after committing
`7af0259 feat(gateway): add fast companion hybrid boundary`.

Evidence:

- Review thread: `019e7c00-ff6c-7f72-9851-a6e3ce637baf`.
- Review worktree: `/Users/jiyurun/.codex/worktrees/de55/New project`.
- Review target: `193a1f3..7af0259`.
- Scope: read-only review of the Task 4 Gateway route, mode guard,
  professional/V21 boundary, mock default safety, trace markers, and docs.
- Allowed tier: T0/T1/T2 only.
- Forbidden: file edits, commits, pushes, Gateway runtime startup,
  provider/V21 execute, durable provider reports with payloads, firmware/NVS/
  flash/raw upload/serial writes, `/v1/devices/control`, and physical device
  paths.
- Review result: no P0 or P1 findings.
- Review P2 finding: `asr.first_partial` had the marker shape but did not use
  `local_audio.first_partial_ms`, so the local ASR timing evidence was not yet
  meaningful for latency comparison.
- Review verification passed:
  `go test ./internal/gateway -run 'FastCompanionHybrid|RealtimeSessionStartRejectsProfessional|ProfessionalModeUsesV21' -count=1`.
- Review verification passed: `git diff --check`.
- Review verification passed: `go run ./cmd/a21 namespace-audit`.
- Control tower follow-up fixed the P2 before opening Task 5: the Gateway now
  records `asr.first_partial` at `received_at + local_audio.first_partial_ms`,
  tests assert the trace offset, and `PHASE7H_FAST_COMPANION_HYBRID.md`
  distinguishes local ASR timing from later provider/TTS/downlink placeholders.

Decision:

- Accept the post-commit review and close its P2 finding in the control branch.
- Do not open a Task 5 execution window from the review thread; Task 5 remains
  under control-tower authorization.

### Provider Spine Task 5 Verification

Accepted by the control tower after closing the Fast Companion Hybrid
trace-fidelity P2 at `22c90ae`.

Evidence:

- Verification branch: `codex/a21-integration-governance-slices`.
- Verification HEAD: `22c90ae fix(gateway): use local ASR timing in fast
  companion trace`.
- Worktree status before doc update: clean.
- Verification passed: `make verify`.
- Verification passed: `go run ./cmd/a21 preflight`.
- Verification passed: `go run ./cmd/a21 doctor`, with the expected
  `firmware_current_artifact_missing` warning for commit `22c90aecc381`.
- Verification passed without provider execution:
  `go run ./cmd/a21 provider-smoke --provider deepseek` returned `skipped`,
  `configured=false`, `executed=false`, and missing
  `A21_LAB_DEEPSEEK_API_KEY`.
- Verification passed without provider execution:
  `go run ./cmd/a21 provider-smoke --provider bailian_dashscope` returned
  `skipped`, `configured=false`, `executed=false`, and missing
  `A21_DASHSCOPE_API_KEY` plus `A21_DASHSCOPE_MODEL`.
- Namespace verification passed: `make namespace-audit`.
- Firmware status check: `git status --short --branch` showed a clean
  `codex/a21-integration-governance-slices` worktree before this docs update;
  no firmware artifacts were created or modified by Task 5.
- No provider/V21 execute, Gateway runtime startup, firmware/NVS/flash/serial
  writes, `/v1/devices/control`, durable provider report, or physical device
  path was used.

Decision:

- Mark Provider Spine Task 5 complete for the dry-run verification scope
  authorized by the current plan.
- Keep external promotion closed until a git remote and explicit target branch
  exist.
- Keep real provider/V21 execution as a separate T4 window requiring explicit
  env, redaction, and control-guard authorization.

### Provider Spine Plan Reconciliation

Accepted by the control tower as a governance correction after opening the
Text Stream Parser follow-up thread.

Evidence:

- Current branch: `codex/a21-integration-governance-slices`.
- Current HEAD before this ledger update:
  `4d356e7 feat(providers): add provider reference profiles`.
- Thread `019e7bdf-a187-7f02-9cce-0f9d605ac9c9` was created for Task 2 of
  `docs/superpowers/plans/2026-05-31-provider-spine-mainline.md`, then
  corrected by the control tower after current-state inspection showed the
  parser and streaming smoke already exist in the integration baseline.
- Audit handoff branch: `codex/a21-provider-textstream-parser`.
- Audit handoff HEAD: `4d356e7ca1ba`.
- Audit handoff dirty files: none.
- Audit handoff diff files: none.
- Audit result: no P0/P1 implementation gaps for Task 2 or Task 3.
- Existing files present in the current baseline:
  `internal/providers/textstream.go`,
  `internal/providers/textstream_client.go`,
  `internal/providers/textstream_test.go`,
  `internal/providers/smoke.go`, `internal/providers/smoke_test.go`, and
  `internal/app/app.go`.
- Current parser tests cover OpenAI-compatible `delta.content`,
  `delta.reasoning`, `delta.reasoning_content`, `[DONE]`, redacted HTTP
  errors, first-byte timing, and first-content timing that excludes reasoning
  deltas.
- Current provider-smoke tests cover `--stream`, `--repeat`, repeated
  first-byte/first-content/total timing, fallback trace/metrics, report
  redaction, and no provider execution unless `--execute` is set.
- Control-tower verification passed:
  `go test ./internal/providers -run TextStream -count=1`.
- Control-tower verification passed:
  `go test ./internal/app ./internal/providers -run 'ProviderSmoke|TextStream' -count=1`.
- Control-tower verification passed: `make verify`.
- Control-tower preflight passed: `go run ./cmd/a21 preflight`.
- Control-tower doctor passed: `go run ./cmd/a21 doctor`, with the expected
  warning that no release-ledger-validated A21 firmware artifact matches commit
  `4d356e7ca1ba`.
- Audit thread verification passed:
  `go test ./internal/providers -run TextStream -count=1`.
- Audit thread verification passed:
  `go test ./internal/providers -run ProviderSmoke -count=1`.
- Audit thread verification passed:
  `go test ./internal/app -run RunProviderSmokeAcceptsStreamRepeatFlags -count=1`.
- Audit thread verification passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke' -count=1`.
- Audit thread verification passed:
  `go test ./internal/app -run 'RunProviderSmoke|ProviderSmoke' -count=1`.
- Audit thread verification passed: `go run ./cmd/a21 namespace-audit`.
- Audit thread verification passed: `go run ./cmd/a21 preflight`.
- Audit thread verification passed: `make verify`.
- Audit thread `doctor` reported local-environment blockers in the isolated
  worktree, including `reserved_port_in_use` for `127.0.0.1:21080`, while the
  control-tower worktree doctor passed on the same integration commit.
- The plan was reconciled so Task 2 and Task 3 are not treated as fresh
  implementation work.
- No provider execute, V21 execute, Gateway runtime, durable provider report,
  NVS execute, flash execute, raw upload, serial write, app partition write,
  `/v1/devices/control`, or physical device path was touched.

Decision:

- Keep thread `019e7bdf-a187-7f02-9cce-0f9d605ac9c9` closed as a
  coverage-audit thread, not a duplicate parser implementation thread.
- No T4 provider execution window is needed for Task 2 or Task 3 coverage.
- Continue Provider Spine from Task 4 or a new control-approved slice.

### Provider Spine Reference Profile Registry

Accepted by the control tower as the next PRD-aligned Provider Spine slice.

Evidence:

- Current branch: `codex/a21-integration-governance-slices`.
- Current HEAD before this ledger update:
  `5e54eb9 docs(control): accept promotion readiness review`.
- Added provider-family vocabulary for `text_stream`, `voice_realtime`,
  `voice_hybrid`, `agent_task`, and `local_audio` while preserving the
  existing `mock` family.
- The built-in `ProviderProfile` catalog now covers PRD reference profiles:
  `siliconflow`, `deepseek`, `stepfun`, `bailian_dashscope`, `moonshot`,
  `volcengine_ark`, `local_ollama`, `local_vllm`, `openai_realtime`,
  `doubao_realtime`, `doubao_tts_realtime`, `hermes_agent`, and
  `mimo_agent`.
- P0 route eligibility remains deliberately narrow: only `mock` and
  `deepseek` are route-eligible.
- Doctor/provider reports now treat realtime references such as
  `doubao_realtime` and `doubao_tts_realtime` as known profiles instead of
  `unknown_provider`, while `provider-smoke` still refuses to execute realtime
  protocols.
- Baidu/Huawei and legacy provider names remain redacted and blocked.
- Focused provider tests passed:
  `go test ./internal/providers -run 'ProviderCatalog|ProviderProfile|ProviderSmoke' -count=1`.
- App/provider package tests passed:
  `go test ./internal/app ./internal/providers -count=1`.
- Project verification passed: `make verify`.
- Namespace audit passed: `go run ./cmd/a21 namespace-audit`.
- Provider dry-runs passed without execution:
  `go run ./cmd/a21 provider-smoke --provider deepseek`,
  `go run ./cmd/a21 provider-smoke --provider bailian_dashscope`, and
  `go run ./cmd/a21 provider-smoke --provider openai_realtime` all reported
  `executed=false`.
- Preflight passed: `go run ./cmd/a21 preflight`.
- Doctor passed: `go run ./cmd/a21 doctor`, with the expected warning that no
  release-ledger-validated A21 firmware artifact matches commit
  `5e54eb9a33f5`.
- No provider execute, V21 execute, Gateway runtime, durable report, NVS
  execute, flash execute, raw upload, serial write, app partition write,
  `/v1/devices/control`, or physical device path was touched.

Decision:

- Keep the Provider Spine catalog broad enough for PRD readiness and doctor
  visibility.
- Keep execution narrow until a future provider-latency or lane-promotion
  slice supplies redacted evidence and explicit authorization.
- Continue next Provider Spine work from the existing plan with Text Stream
  Parser or Streaming Provider Smoke only after this registry slice is
  committed and reviewed as needed.

### Machine-Readable Promotion Readiness Gate

Accepted by the control tower as the next local governance slice for the
selected integration promotion candidate.

Evidence:

- Current branch: `codex/a21-integration-governance-slices`.
- Current HEAD before this ledger update:
  `9f9d5e6 docs(control): record promotion gate refresh`.
- Added `go run ./cmd/a21 promotion-readiness` and `make
  promotion-readiness`.
- The report schema is `a21.promotion_readiness.v1`.
- The gate checks integration branch naming, detached HEAD, dirty worktree,
  accepted slice ancestry for `a031f3d`, `6b7fdf0`, `fc61793`, and
  `1e38804`, local `main`/`master` presence, configured git remotes, and an
  explicit target remote/branch.
- Targeted promotion-readiness tests passed:
  `go test ./internal/app -run 'TestRunPromotionReadiness' -count=1`.
- Package verification passed:
  `go test ./internal/app ./internal/runtimeguard -count=1`.
- Project verification passed: `make verify`.
- Namespace audit passed: `go run ./cmd/a21 namespace-audit`.
- Preflight passed: `go run ./cmd/a21 preflight`.
- Doctor passed: `go run ./cmd/a21 doctor`, with the expected warning that no
  release-ledger-validated A21 firmware artifact matches commit
  `9f9d5e6bdb1c`.
- A pre-commit `promotion-readiness` dry run returned the expected non-zero
  result while the worktree was dirty and no remote/target branch was
  configured.
- No provider execute, V21 execute, Gateway runtime, durable report, NVS
  execute, flash execute, raw upload, serial write, app partition write,
  `/v1/devices/control`, or physical device path was touched.

Decision:

- Use `promotion-readiness` for future integration handoffs and PR/mainline
  promotion decisions.
- Treat `review_ready=true` and `external_promotion_ready=false` as the
  expected clean local state until a target remote and target branch are
  explicitly configured.
- External promotion remains blocked on the same target-selection decision:
  choose/configure a git remote and target branch, or explicitly approve a
  local-only mainline branch.

### Promotion Readiness Gate Review

Accepted from thread `019e7bce-bacf-76e3-98f3-1e53fffe1377`.

Evidence:

- Review worktree: `/Users/jiyurun/.codex/worktrees/9f7f/New project`.
- Reviewed implementation commit:
  `3ddcc45 feat(control): add promotion readiness gate`.
- Control tower HEAD at review close:
  `e55b652 docs(control): track promotion readiness review thread`.
- Review result: no P0, P1, or P2 findings.
- Review conclusion: `promotion-readiness` is acceptable as the current
  integration branch's local promotion gate.
- Review confirmed the implementation is host-only and only reads cwd plus
  git metadata; it does not enter provider execution, V21 execution, Gateway
  runtime, NVS/flash/serial/write paths, hardware control, or
  `/v1/devices/control`.
- Review confirmed the gate distinguishes `review_ready` from
  `external_promotion_ready` and keeps external promotion blocked without a
  configured remote and explicit target branch.
- Review verification passed:
  `go test ./internal/app -run 'TestRunPromotionReadiness' -count=1`.
- Review verification passed: `git diff --check`.
- Review ran `go run ./cmd/a21 promotion-readiness` in the detached review
  worktree and received the expected non-zero result while confirming all four
  accepted slice commits are ancestors.
- Review did not change files, commit, start services, run provider/V21
  execute, run hardware write paths, or touch physical device control.

Decision:

- Keep the gate as accepted local promotion infrastructure.
- The only remaining manual promotion decision is still target selection:
  configure/choose a remote and target branch, or explicitly approve a
  local-only mainline branch.

### Promotion Candidate Full Gate Refresh

Accepted by the control tower after full local verification of the selected
promotion candidate.

Evidence:

- Current branch: `codex/a21-integration-governance-slices`.
- Current HEAD before this ledger update:
  `2339d4e docs(control): choose integration promotion path`.
- Worktree: `/Users/jiyurun/Documents/New project`.
- Dirty state before this ledger update: clean.
- Project verification passed: `make verify`.
- Preflight passed: `go run ./cmd/a21 preflight`.
- Doctor passed: `go run ./cmd/a21 doctor`, with one warning that no
  release-ledger-validated A21 firmware artifact matches commit
  `2339d4ee06c9`.
- Provider dry-run passed without execution:
  `go run ./cmd/a21 provider-smoke --provider deepseek --stream --repeat 3`
  reported `executed=false`, `configured=false`, and missing
  `A21_LAB_DEEPSEEK_API_KEY`.
- V21 adapter dry-run passed without execution:
  `go run ./cmd/a21 v21-adapter-smoke` reported `executed=false`,
  `configured=false`, and missing adapter URL.
- `go run ./cmd/a21 control-guard --command 'stackchan-official-pcm-bridge-flash-execute'`
  returned the expected non-zero T8 blocked result.
- `go run ./cmd/a21 control-guard --command 'stackchan-official-pcm-bridge-nvs-execute'`
  returned the expected non-zero T7 hardware-window branch requirement.
- No provider execute, V21 execute, Gateway runtime, durable report, NVS
  execute, flash execute, raw upload, serial write, app partition write,
  `/v1/devices/control`, or physical device path was touched.

Decision:

- Keep `codex/a21-integration-governance-slices` as the current verified local
  promotion candidate.
- The only remaining promotion blocker is target selection: choose/configure a
  git remote and target branch, or explicitly approve a local-only mainline
  branch.

### Integration Promotion Shape

Accepted by the control tower after branch topology review.

Evidence:

- Current branch: `codex/a21-integration-governance-slices`.
- Current HEAD before this ledger update:
  `045147d docs(control): accept integration review`.
- Local branches present:
  `codex/a21-integration-governance-slices`,
  `codex/a21-project-control`,
  `codex/a21-phase1-clean-skeleton`,
  `codex/a21-docs-pcm-bridge-flash-adr`,
  `codex/a21-provider-spine-deepseek-textstream`,
  `codex/a21-mainline-professional-v21-contract`,
  `codex/a21-mainline-stackchan-hardware-diagnostic`.
- No local `main` or `master` branch exists.
- No git remote is configured.
- `docs/engineering/A21_PROJECT_CONTROL.md` now includes the
  `codex/a21-integration-<bundle>` branch pattern and limits it to accepted
  slice merges plus T1/T2 verification/review evidence.
- The combined branch preserves the four topic branch commits and has already
  passed integration review with no P0/P1/P2 findings.

Decision:

- Use `codex/a21-integration-governance-slices` as the current local promotion
  candidate for this governance slice bundle.
- Keep the four topic branches available for review, bisect, or rollback, but
  do not make four separate topic PRs the preferred path unless a remote/mainline
  reviewer asks for split review.
- Do not merge into `codex/a21-phase1-clean-skeleton`; that branch is stale for
  current M3, StackChan hardware, provider, NVS, and official bridge work.
- Do not merge into an invented `main` or `master` branch.
- Next external integration action is blocked on choosing or configuring a git
  remote and naming the target branch, or explicitly approving a local-only
  mainline branch.

### Read-Only Integration Review

Accepted from thread `019e7bba-71ca-71d0-84cc-78424d4d07ab`.

Evidence:

- Review worktree: `/Users/jiyurun/.codex/worktrees/47a6/New project`.
- Reviewed commit: `1872ca9 docs(control): record integrated governance baseline`.
- Review result: no P0, P1, or P2 findings.
- Review conclusion: `codex/a21-integration-governance-slices` is
  review-ready.
- Review confirmed the four accepted slice commits are ancestors of the
  integration baseline: `a031f3d`, `6b7fdf0`, `fc61793`, and `1e38804`.
- Review confirmed Provider first-content timing is set only from content
  deltas, not reasoning deltas.
- Review confirmed V21 adapter use remains explicit, direct no-ambient-proxy,
  and does not fall back to mock V21 evidence on invalid configured adapter
  URLs.
- Review confirmed StackChan hardware mainline keeps `microphone=available`
  blocked for release reports and blocks planned hardware from being declared
  falsely available.
- Review confirmed the official PCM bridge app flash ADR remains Draft/T8
  blocked and does not authorize app flash.
- Review confirmed the control ledger accurately reflects branches, threads,
  verification, and forbidden T4/T6/T7/T8 actions.
- Review verification passed: `git diff --check`.
- Review verification passed:
  `go test ./internal/app ./internal/providers ./internal/v21adapter -count=1`.
- Review verification passed: `go run ./cmd/a21 namespace-audit`.
- Review verification ran
  `go run ./cmd/a21 control-guard --command 'stackchan-official-pcm-bridge-flash-execute'`
  and received the expected T8 blocked result.
- Review did not change files, commit, start services, run provider/V21 execute,
  run hardware write paths, or touch physical device control.

Control-tower follow-up evidence:

- Latest integration HEAD: `54af08b docs(control): track integration review thread`.
- Control-tower verification passed on latest HEAD: `make verify`.
- Control-tower targeted non-cached tests passed:
  `go test ./internal/app ./internal/providers ./internal/v21adapter ./internal/runtimeguard -count=1`.

Decision:

- Treat `codex/a21-integration-governance-slices` as review-ready for the next
  PR/mainline integration decision.
- Keep all T4 provider/V21 execute and T6/T7/T8 hardware operations closed
  until an explicit future window is opened by the control tower.

### Integrated Governance Slices Baseline

Accepted on the control tower main worktree after merging the four accepted
slices.

Evidence:

- Branch: `codex/a21-integration-governance-slices`.
- Worktree: `/Users/jiyurun/Documents/New project`.
- Base control commit:
  `7bdfe9d docs(control): record PCM flash ADR handoff`.
- Integration HEAD before this ledger update:
  `acdd929 merge: integrate StackChan hardware diagnostic honesty`.
- Merged branches:
  `codex/a21-docs-pcm-bridge-flash-adr`,
  `codex/a21-provider-spine-deepseek-textstream`,
  `codex/a21-mainline-professional-v21-contract`,
  `codex/a21-mainline-stackchan-hardware-diagnostic`.
- Merge commits: `5674b08`, `3c3ae1f`, `6abf8d7`, `acdd929`.
- Merge conflict state: no manual conflict resolution required. Git `ort`
  auto-merged the shared `internal/app` changes.
- Dirty state after merges and before this ledger update: clean.
- Targeted integration tests passed:
  `go test ./internal/app ./internal/providers ./internal/v21adapter -count=1`.
- Project verification passed: `make verify`.
- Whitespace verification passed: `git diff --check`.
- Namespace verification passed: `go run ./cmd/a21 namespace-audit`.
- Preflight passed: `go run ./cmd/a21 preflight`.
- Doctor passed: `go run ./cmd/a21 doctor`, with a warning that no
  release-ledger-validated firmware artifact matches integration commit
  `acdd9291835a`.
- Provider dry-run passed without execution:
  `go run ./cmd/a21 provider-smoke --provider deepseek --stream --repeat 3`
  reported `executed=false`, `configured=false`, and missing
  `A21_LAB_DEEPSEEK_API_KEY`.
- V21 adapter dry-run passed without execution:
  `go run ./cmd/a21 v21-adapter-smoke` reported `executed=false`,
  `configured=false`, and missing adapter URL.
- `go run ./cmd/a21 control-guard --command 'stackchan-official-pcm-bridge-flash-execute'`
  rejected the command as T8 blocked until ADR plus reviewed execute guard.
- `go run ./cmd/a21 control-guard --command 'stackchan-official-pcm-bridge-nvs-execute'`
  rejected the command because the integration branch is not a
  `codex/a21-hardware-window-*` branch.
- No provider execute, V21 execute, Gateway runtime, durable report, NVS
  execute, flash execute, raw upload, serial write, app partition write,
  `/v1/devices/control`, or physical device path was touched.

Decision:

- Accept `codex/a21-integration-governance-slices` as the current combined
  verification baseline for these four PRD/governance slices.
- Keep the original slice branches available as reviewable units of work.
- Do not treat this integration branch as permission for T4 provider/V21
  execution or T6/T7/T8 hardware operations.
- Next control decision is whether to use this integration branch as the PR
  branch, or split review into the four already-accepted topic branches.

### Official PCM Bridge App Flash ADR Docs-Only Gate

Accepted from thread `019e7bb0-bf95-74f3-a935-1e89644bd417`.

Evidence:

- Branch: `codex/a21-docs-pcm-bridge-flash-adr`.
- Worktree: `/Users/jiyurun/.codex/worktrees/42b1/New project`.
- Committed HEAD: `a031f3d docs(firmware): add PCM bridge flash ADR gate`.
- Dirty state after commit: clean.
- Changed files:
  `docs/engineering/adr/0005-official-pcm-bridge-app-flash.md`,
  `docs/engineering/FIRMWARE_RELEASE_DISCIPLINE.md`.
- New ADR status is Draft, not accepted.
- The ADR records why `stackchan-official-pcm-bridge-flash-execute` remains T8
  blocked and lists the evidence required before any future downgrade to T7 or
  T6.
- `FIRMWARE_RELEASE_DISCIPLINE.md` now points operators at the draft ADR gate
  from the official PCM bridge lane.
- Control-tower verification passed: `git diff --check`.
- Control-tower review confirmed the text does not authorize implementation,
  app flashing, NVS writes, provider execution, V21 execution, Gateway runtime,
  serial writes, raw uploads, `/v1/devices/control`, or a hardware window.
- No build, Gateway runtime, provider execute, V21 execute, NVS execute, flash
  execute, raw upload, serial write, durable report, or physical device path
  was touched.

Decision:

- Accept the docs-only ADR gate slice as governance evidence.
- Keep `stackchan-official-pcm-bridge-flash-execute` classified T8 blocked.
- A future downgrade requires a separate accepted ADR, reviewed execute guard,
  fresh verification, rollback package, NVS receipt, operator token, explicit
  USB target, and foreground hardware-window branch.

### Professional V21 Evidence Adapter Readiness

Accepted from thread `019e7ba8-2bec-7f12-83ce-8b0fd1cc06c9`, with control
tower verification and commit.

Evidence:

- Branch: `codex/a21-mainline-professional-v21-contract`.
- Worktree: `/Users/jiyurun/.codex/worktrees/ab7a/New project`.
- Committed HEAD: `fc61793 fix(v21): wire configured adapter into gateway env`.
- Dirty state after commit: clean.
- Changed files: `internal/app/app.go`, `internal/app/app_test.go`,
  `internal/v21adapter/client.go`, `internal/v21adapter/client_test.go`.
- `newGatewayServerFromEnv` now honors explicit `A21_V21_ADAPTER_URL` for
  professional Gateway turns.
- Invalid adapter configuration fails honestly through the professional path
  instead of silently falling back to mock V21 evidence.
- V21 adapter HTTP client now uses direct networking rather than inheriting
  ambient proxy variables.
- Added tests for configured adapter routing, invalid-adapter honest failure,
  and no ambient proxy inheritance.
- Control-tower verification passed:
  `go test ./internal/app ./internal/v21adapter -run 'GatewayServerFromEnv|HTTPClientDoesNotUseAmbientProxy|V21|Professional' -count=1`.
- Control-tower verification passed: `git diff --check`.
- Control-tower verification passed: `make verify`.
- Control-tower verification passed: `go run ./cmd/a21 namespace-audit`.
- Control-tower verification passed: `go run ./cmd/a21 preflight`.
- Control-tower verification passed: `go run ./cmd/a21 doctor`, with only
  expected isolated worktree firmware tool/artifact warnings.
- Control-tower verification passed: `go run ./cmd/a21 v21-adapter-smoke`
  dry-run, without `--execute`.
- No real V21 execute, provider execute, key use, Gateway runtime, firmware,
  hardware, NVS, flash, serial write, or `/v1/devices/control` path was touched.

Decision:

- Accept the V21 professional adapter readiness code slice.
- Do not open T4 automatically. Real V21 adapter execution requires explicit
  adapter URL, redacted report path, and no saved query/answer/evidence bodies.
- Keep A21 as the professional-mode caller and presentation owner, not V21's
  voice skin.

### Provider Spine / DeepSeek Text-Stream Readiness

Accepted from thread `019e7ba1-d81e-74c3-bd2e-a6191344085a`.

Evidence:

- Branch: `codex/a21-provider-spine-deepseek-textstream`.
- Worktree: `/Users/jiyurun/.codex/worktrees/84d6/New project`.
- Committed HEAD: `6b7fdf0 fix(providers): measure first content from content deltas`.
- Dirty state after commit: clean.
- Changed files: `internal/providers/smoke.go`,
  `internal/providers/smoke_test.go`,
  `internal/providers/textstream_client.go`,
  `internal/providers/textstream_test.go`.
- `first_content_ms` now starts only on `content` deltas, not on reasoning
  deltas, so DeepSeek reasoning tokens do not understate first visible content
  latency.
- Added reasoning-only SSE tests for provider smoke and text-stream completion.
- Redaction checks cover synthetic key, model name, and reasoning text.
- Control-tower verification passed:
  `go test ./internal/providers -run 'TextStream|ProviderSmoke' -count=1`.
- Control-tower verification passed: `git diff --check`.
- Control-tower verification passed: `make verify`.
- Thread verification also passed `go test ./...`, `go run ./cmd/a21 namespace-audit`,
  `go run ./cmd/a21 preflight`, `go run ./cmd/a21 doctor`, and redacted
  `provider-smoke` dry-run checks.
- No real provider execute, key use, external provider call, durable report,
  firmware change, Gateway runtime, V21 execute, or hardware path was touched.

Decision:

- Accept the Provider Spine readiness code slice.
- Do not open T4 automatically. Real DeepSeek execution is a separate explicit
  provider window only if the control tower wants live first-byte/first-content
  evidence.
- Continue to keep A21 as provider-neutral input/output, not an agent router.

### StackChan Hardware Mainline Diagnostic Consolidation

Accepted from thread `019e7b99-141e-70a3-b0fd-c5dd38b5cab5`, with a control
tower follow-up tightening the microphone release invariant.

Evidence:

- Branch: `codex/a21-mainline-stackchan-hardware-diagnostic`.
- Worktree: `/Users/jiyurun/.codex/worktrees/ddec/New project`.
- Committed HEAD: `1e38804 fix(hardware): enforce StackChan capability honesty`.
- Dirty state after commit: clean.
- Changed files: `internal/app/app.go`, `internal/app/app_test.go`.
- Added `capability_invariants` to `stackchan-hardware-mainline` reports.
- Blocked false `available` promotion for planned hardware such as IMU.
- Blocked release `microphone=available`; release microphone remains
  `disabled_*` or diagnostic-only until a separate production evidence gate.
- Aligned ordered hardware target statuses with the current hardware charter:
  `planned_ambient_light_sensor`, `planned_proximity_sensor`,
  `planned_550mah_battery`, `planned_continuous_rotation_axis`,
  `planned_core_s3_camera`, `planned_nfc`, and `planned_infrared_tx_rx`.
- TDD red evidence: `go test ./internal/app -run TestRunStackChanHardwareMainlineBlocksReleaseMicrophonePromotion -count=1`
  failed before the microphone invariant was tightened.
- Targeted tests passed:
  `go test ./internal/app -run 'TestRunStackChanHardwareMainline' -count=1`.
- `make verify` passed.
- `go run ./cmd/a21 namespace-audit` passed.
- `go run ./cmd/a21 preflight` passed.
- `go run ./cmd/a21 doctor` passed on rerun with only expected isolated
  worktree firmware tool/artifact warnings. The first run reported transient
  `127.0.0.1:21080` usage, but `lsof` found no listener and the immediate
  rerun passed.
- No durable report was generated.
- No hardware, Gateway runtime, provider execute, V21 execute, NVS execute,
  flash execute, raw upload, or `/v1/devices/control` call was run.

Decision:

- Accept the diagnostic consolidation branch as the current hardware honesty
  code slice.
- Do not claim physical acceptance for microphone, IMU, sensors, second servo
  axis, camera, NFC, or infrared from this report-only gate.
- Keep future physical validation behind an explicit T6 foreground window.
- Leave branch integration or PR creation to the control tower's next merge
  decision; Provider Spine readiness may start in its own thread without T4.

### M3 Official PCM Bridge NVS Closure

Accepted from thread `019e7b91-537b-7233-b682-276f77e1b871`.

Evidence:

- Worktree was detached at `1c6de691437a`.
- No dirty files and no code changes were produced.
- `go test ./internal/app ./internal/runtimeguard` passed.
- `git diff --check` passed.
- `go run ./cmd/a21 namespace-audit` passed.
- `make verify` passed.
- `go run ./cmd/a21 preflight` passed.
- `go run ./cmd/a21 doctor` passed with only expected environment/artifact
  warnings in the detached worktree.
- `go run ./cmd/a21 control-guard --command 'stackchan-official-pcm-bridge-nvs-execute'`
  rejected T7 execution in the detached/background worktree.
- `go run ./cmd/a21 control-guard --command 'stackchan-official-pcm-bridge-flash-execute'`
  rejected T8 app flashing until ADR.
- No persistent report was generated.
- No `.a21-run` directory was created.
- No hardware, Gateway, provider, V21, NVS execute, flash execute, or raw upload
  command was run.

Decision:

- Accept the NVS-only lane as control-closed for this slice.
- Do not create `codex/a21-mainline-m3-pcm-bridge-nvs-closure`; there is no
  implementation diff to carry.
- Keep app flash-execute blocked.

## Authorized Next Queue

1. PR / mainline integration decision.
   - Combined verification branch:
     `codex/a21-integration-governance-slices`.
   - Read-only review thread:
     `019e7bba-71ca-71d0-84cc-78424d4d07ab`, completed with no P0/P1/P2
     findings.
   - Topic branches remain available:
     `codex/a21-mainline-stackchan-hardware-diagnostic`,
     `codex/a21-provider-spine-deepseek-textstream`,
     `codex/a21-mainline-professional-v21-contract`,
     `codex/a21-docs-pcm-bridge-flash-adr`.
   - Max tier: T1/T2.
   - Purpose: promote the verified combined branch once a remote/target branch
     is selected, or once a local-only mainline branch is explicitly approved.
   - Current local repository has no configured git remote and no `main` or
     `master`; push/PR or local mainline merge requires adding/selecting a
     remote and target branch outside this ledger change.
   - Forbidden: provider/V21 execute, Gateway runtime, hardware writes,
     background flash, NVS execute, raw upload, or changing PRD scope while
     integrating.

2. Professional V21 evidence lane.
   - Branch: `codex/a21-mainline-professional-v21-contract`.
   - Status: accepted code slice at
     `fc61793 fix(v21): wire configured adapter into gateway env`.
   - Max tier: T1/T2 by default. V21 execute is T4 and requires explicit
     adapter URL, redacted reports, and no query/answer/evidence text in saved
     output.
   - Forbidden: treating A21 as V21's voice skin, copying V21 internals,
     storing evidence bodies in reports, Gateway runtime, provider execute, or
     V21 execute without control approval.

3. Provider Spine / text-stream hot plug.
   - Accepted implementation branch:
     `codex/a21-provider-spine-deepseek-textstream`.
   - Accepted implementation status:
     `6b7fdf0 fix(providers): measure first content from content deltas`.
   - Current integration registry extension:
     `4d356e7 feat(providers): add provider reference profiles`.
   - Completed audit thread:
     `019e7bdf-a187-7f02-9cce-0f9d605ac9c9`.
   - Completed audit worktree:
     `/Users/jiyurun/.codex/worktrees/9136/New project`.
   - Active next audit thread:
     `019e7be6-bca3-71f2-9770-857b9da48b67`.
   - Active next audit worktree:
     `/Users/jiyurun/.codex/worktrees/0d72/New project`.
   - Active implementation thread:
     `019e7bed-4e1e-7512-8f21-1647b2357c00`.
   - Active implementation worktree:
     `/Users/jiyurun/.codex/worktrees/80c8/New project`.
   - Active implementation branch:
     `codex/a21-fast-companion-hybrid-boundary`.
   - Max tier: T1/T2 by default. T4 only with explicit provider execution
     declaration, redaction check, and local env confirmation.
   - Purpose: implement the Gateway-level Fast Companion Hybrid boundary and
     unified trace placeholders after the audit confirmed this is a real Task 4
     gap.
   - Forbidden: provider keys in docs/reports/logs, provider URLs in firmware,
     hidden proxy inheritance, Baidu/Huawei expansion, provider execute without
     control approval, or turning A21 into an agent router.

4. StackChan hardware mainline diagnostic consolidation.
   - Branch: `codex/a21-mainline-stackchan-hardware-diagnostic`.
   - Status: accepted code slice at
     `1e38804 fix(hardware): enforce StackChan capability honesty`.
   - Goal: protect the embodied hardware foundation while Provider Spine
     advances.
   - Max tier: T1/T2 unless the control tower explicitly opens a T6 foreground
     physical-validation window.
   - Allowed: docs/tests around `stackchan-hardware-mainline`, capability
     declaration invariants, no-flash evidence gates, and planned-vs-available
     capability honesty.
   - Forbidden: firmware writes, app flash, NVS execute, provider/V21 execute,
     Gateway background runtime left running, or claiming physical acceptance
     without fresh physical evidence.

5. Official PCM bridge app flash execute.
   - Status: T8 blocked.
   - Branch: none authorized.
   - Required before any downgrade: accepted ADR, reviewed execute guard, NVS
     receipt, rollback package, fresh verification, explicit USB target,
     operator token, and foreground `codex/a21-hardware-window-*` branch.
   - Forbidden: adding or running `stackchan-official-pcm-bridge-flash-execute`,
     raw `pio upload`, raw `idf.py flash`, copied `esptool write_flash`, or any
     serial/app partition write from normal Codex worktrees.

## Ledger Update Checklist

Before a control handoff, update this ledger with:

- branch and HEAD;
- worktree path and dirty state;
- thread id, role, status, max tier, and write authority;
- accepted handoff evidence;
- next authorized branch and forbidden actions;
- verification commands and results;
- whether any report, service, provider, V21 adapter, Gateway, NVS, app
  partition, or physical device was touched.
