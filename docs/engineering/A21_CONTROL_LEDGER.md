# A21 Control Ledger

Status: active control ledger.
Date: 2026-05-31.
Ledger branch: `codex/a21-project-control`.
Last accepted control commit before this ledger update: `2df39b984558`.

This ledger is the control tower's current operating board. It records which
branch, worktree, thread role, and tool tier are authorized next. Update it
whenever the control tower changes branch ownership, resumes or pauses an
execution thread, promotes a new tool tier, or accepts a handoff.

This ledger does not replace `A21_PROJECT_CONTROL.md`. The project-control doc
defines policy; this ledger records the current queue and accepted state.

## Current Accepted State

- Control branch: `codex/a21-project-control`.
- Control HEAD before this ledger update: `2df39b984558 docs(control): add A21 control ledger`.
- Main worktree: `/Users/jiyurun/Documents/New project`.
- Main worktree status at acceptance: clean.
- `a21 control-guard` is the active machine-readable tool-tier gate.
- StackChan hardware mainline diagnostic branch is committed at
  `1e38804 fix(hardware): enforce StackChan capability honesty`.
- Official StackChan PCM bridge NVS-only lane is accepted as closed for the
  current M3 governance slice.
- Official StackChan PCM bridge app flash-execute remains T8 blocked until an
  ADR, reviewed execute guard, and fresh verification approve it.
- No current PRD slice authorizes raw `pio upload`, `idf.py flash`, copied
  `esptool write_flash`, provider/V21 execute, or background hardware writes.

## Thread Ledger

| Thread | Role | Worktree | Status | Max tier | Write authority |
| --- | --- | --- | --- | --- | --- |
| `019e7b6f-dedb-73c1-aee6-2c438858da03` | Control tower | `/Users/jiyurun/Documents/New project` | active | T1 by default; higher only after declaration | yes |
| `019e7ba1-d81e-74c3-bd2e-a6191344085a` | Provider Spine / DeepSeek text-stream readiness | `/Users/jiyurun/.codex/worktrees/84d6/New project` | active | T1/T2; T4 only after explicit provider-execute window | no |
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

1. Provider Spine / text-stream hot plug.
   - Thread: `019e7ba1-d81e-74c3-bd2e-a6191344085a`.
   - Worktree: `/Users/jiyurun/.codex/worktrees/84d6/New project`.
   - Branch: `codex/a21-provider-spine-deepseek-textstream`.
   - Gate: active read/plan/minimal T1/T2 readiness work only.
   - Max tier: T1/T2 by default. T4 only with explicit provider execution
     declaration, redaction check, and local env confirmation.
   - Forbidden: provider keys in docs/reports/logs, provider URLs in firmware,
     hidden proxy inheritance, Baidu/Huawei expansion, provider execute without
     control approval, or turning A21 into an agent router.

2. StackChan hardware mainline diagnostic consolidation.
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

3. Professional V21 evidence lane.
   - Branch: `codex/a21-mainline-professional-v21-contract`.
   - Gate: start only after Provider Spine queue state is explicit.
   - Max tier: T1/T2 by default. V21 execute is T4 and requires explicit
     adapter URL, redacted reports, and no query/answer/evidence text in saved
     output.

4. Official PCM bridge app flash ADR.
   - Branch: `codex/a21-docs-pcm-bridge-flash-adr`.
   - Max tier: T0/T1.
   - Purpose: draft ADR and reviewed execute-guard design only.
   - Forbidden: implementing or running app flash-execute before ADR acceptance.

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
