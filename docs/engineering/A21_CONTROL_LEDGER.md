# A21 Control Ledger

Status: active integration ledger.
Date: 2026-05-31.
Ledger branch: `codex/a21-integration-governance-slices`.
Last accepted integration commit before this ledger update: `94d87c3`.

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
  `94d87c3 docs(control): accept fast companion audit`.
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
  `e55b652`, `5e54eb9`, `4d356e7`, `3b9f05d`, `f9df726`, and `94d87c3`.
- Read-only integration review found no P0/P1/P2 issues against the merged
  governance baseline at `1872ca9`.
- Control tower has selected the single combined integration branch as the
  current local promotion candidate. Topic branches remain frozen evidence and
  rollback/review handles, not the preferred next PR shape.
- The local repository has no configured remote and no `main` or `master`
  branch. Remote PR creation or local mainline merge therefore requires a
  later explicit target decision outside this ledger update.
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
| `019e7bed-4e1e-7512-8f21-1647b2357c00` | Fast Companion Hybrid Gateway boundary implementation | `/Users/jiyurun/.codex/worktrees/80c8/New project` | active; Task 4 implementation slice | T1/T2 | yes, in its isolated worktree only |
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

Decision:

- This is the active Provider Spine Task 4 implementation slice.
- Control tower will not accept completion until the implementation thread
  hands back dirty files, tests, and verification evidence.

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
