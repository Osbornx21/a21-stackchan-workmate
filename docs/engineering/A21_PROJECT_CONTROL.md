# A21 Project Control

Status: active control document.
Date: 2026-05-31.

This document exists because A21 has moved from a clean foundation into
high-speed hardware, provider, and real-device work. The codebase is not
uncontrolled by design, but the execution surface can become uncontrolled when
multiple threads edit the same worktree or when hardware-write commands appear
faster than the governance docs can catch up.

## Current Control State

- Control branch: `codex/a21-project-control`.
- Previous branch name `codex/a21-phase1-clean-skeleton` is stale for current
  M3, StackChan hardware, provider, NVS, and official bridge work.
- Main execution thread is paused until the control thread explicitly resumes
  it.
- The official PCM bridge NVS-only lane may be reviewed and stabilized.
- The official PCM bridge app flash-execute lane is blocked until an ADR,
  reviewed guard, and fresh verification approve it.
- `go run ./cmd/a21 control-guard` is now the machine-readable control gate.
  T7 execute paths consume it before hardware writes and write its branch,
  commit, worktree, dirty-state, tier, and command evidence into execution
  receipts.
- `go run ./cmd/a21 promotion-readiness` is the machine-readable integration
  promotion gate. It separates local review readiness from external promotion
  readiness, and it must stay host-only: no provider execution, V21 execution,
  Gateway runtime, or hardware/device side effect.
- `docs/engineering/A21_CONTROL_LEDGER.md` is the current control tower queue:
  it records accepted handoffs, active/paused thread roles, worktree ownership,
  and the next PRD-authorized implementation slice.
- No raw `pio upload`, `idf.py flash`, copied esptool command, or generic
  firmware path is allowed.

## Branch Model

A21 branches must name the lane they control. A branch name is part of the
safety system, not cosmetic metadata.

| Branch pattern | Purpose | Allowed work |
| --- | --- | --- |
| `codex/a21-project-control` | Control tower | governance docs, red build fixes, thread/branch/tool triage |
| `codex/a21-integration-<bundle>` | Integration candidate | merge already-accepted slices, resolve conflicts, run T1/T2 verification, record review outcome |
| `codex/a21-mainline-<milestone>` | Product implementation | one milestone slice after control approval |
| `codex/a21-provider-<provider-or-spine>` | Provider work | provider contracts, smoke tests, redacted reports |
| `codex/a21-firmware-<capability>` | Firmware build/probe lane | build/test/report work without physical writes |
| `codex/a21-hardware-window-<date>-<capability>` | Foreground hardware window | one physical-device write or acceptance window |
| `codex/a21-docs-<topic>` | Documentation-only | docs, diagrams, ADR drafts, no runtime edits |

Rules:

- Do not continue feature work on a stale branch name.
- Integration branches do not create new product scope. They may combine only
  accepted slice branches plus ledger/review evidence, and they must keep T4
  provider/V21 execution and T6/T7/T8 hardware windows closed.
- Do not mix provider, firmware write, V21 adapter, and product UX work in one
  branch unless the control thread explicitly declares it a release branch.
- Hardware-write branches are single-thread and foreground-only.
- Provider branches cannot touch firmware write paths.
- Firmware branches cannot add provider keys, provider URLs, or V21 internals.
- A branch that touches `Makefile`, `internal/app`, or firmware flash guards
  must run `make verify` before handoff.
- If the tree is dirty, new work must first classify every dirty file as
  keep, finish, quarantine, or discard-by-user-approval.

## Tool Tree

Commands are grouped by blast radius. A task must declare the highest tier it
will touch before execution.

| Tier | Label | Examples | Rules |
| --- | --- | --- | --- |
| T0 | Read-only inspection | `rg`, `git status`, `git diff`, `go list` | Always allowed in control/review threads |
| T1 | Host-only verification | `go test ./...`, `git diff --check`, `go run ./cmd/a21 namespace-audit`, `go run ./cmd/a21 promotion-readiness`, `make verify` | Allowed when no hardware/service side effect is expected |
| T2 | Local reports and dry runs | `preflight`, `doctor`, `latency-bench`, `provider-smoke` without `--execute`, `v21-adapter-smoke` without `--execute`, `provider-realtime-fixture --execute` because it is an offline fixture | Reports must stay redacted |
| T3 | Local runtime/service | `gateway`, simulator, loopback, local ASR/TTS smoke | Must declare ports and stop processes after the window |
| T4 | External/provider execution | `provider-smoke --execute`, `v21-adapter-smoke --execute`, `local-voice-loopback --execute-text-provider`, `stackchan-fast-companion-turn --execute-text-provider` | Requires explicit env, redaction, no key in command output |
| T5 | Firmware build/package | `firmware-tools`, `firmware-test`, `firmware-build`, `firmware-package`, official StackChan build lanes | No physical writes; package requires clean worktree |
| T6 | Physical validation commands | `stackchan-*acceptance`, `/v1/devices/control` probes | Must use explicit device ID, trace/report path, and final idle check |
| T7 | Physical writes | `stackchan-official-pcm-bridge-nvs-execute`, `stackchan-official-pcm-bridge-flash-execute`, `stackchan-official-audio-smoke-flash-execute`, `firmware-*-flash-execute` | One foreground thread only, exact confirmation token, explicit port, `control-guard` receipt |
| T8 | Blocked until ADR | raw `pio run -t upload`, raw `idf.py flash`, copied `esptool write_flash` | Not allowed from normal threads |

T7/T8 rules:

- No background Codex thread may run hardware-write commands.
- No hardware write may run while another A21 thread is active in any A21
  worktree for the same repository.
- T7 commands must run from a clean `codex/a21-hardware-window-*` branch.
- T7 commands must not run from detached HEAD or from `.codex/worktrees`
  background worktrees.
- The command must identify the artifact, branch, commit, port, device ID, and
  expected report path before execution.
- The execution report must include a `control_guard` object proving command,
  tier, branch, commit, worktree path, detached/dirty state, and guard result.
- The operator must preserve NVS calibration and report whether servo
  calibration was present.
- After execution, the device must be returned to an honest idle or safe state,
  or the report must mark the failure plainly.

## Thread Roles

Codex threads are treated as project actors.

| Role | Thread | Rules |
| --- | --- | --- |
| Control tower | `A21 control tower` | Owns branch names, stop/resume decisions, governance docs, and release gates |
| Main execution | `A21 mainline execution` | One implementation slice at a time; paused when control detects risk |
| Read-only review | `A21 read-only review` | May inspect and report; no file writes, no build/service/hardware side effects |
| Procurement/research | `A21 API procurement` | No repo edits; no secrets in files or reports |
| Hardware window | created per session | Foreground only; one device action; no parallel thread |

Rules:

- Only one write-capable thread may edit any A21 worktree for this repository
  at a time.
- If two active threads target the same worktree, the control tower pauses the
  non-control thread before editing.
- A thread must state its branch, dirty files, intended files, highest tool
  tier, and forbidden actions before making changes.
- A paused thread may only report status until the control tower resumes it.
- Read-only threads must not turn into implementation threads; open a new
  execution branch/thread instead.
- Procurement threads must not import external recommendations directly into
  A21 architecture. They create proposals only.

## Handoff Contract

Every implementation handoff must include:

- branch name and HEAD commit;
- dirty files, staged files, and untracked files;
- exact commands run and their result;
- for integration branches, the `promotion-readiness` result and whether any
  non-zero exit was the expected external-target blocker;
- report paths for generated evidence;
- hardware/device state when StackChan was involved;
- next action, blocked reason, or required ADR.

Every hardware handoff must additionally include:

- USB serial port;
- device ID;
- firmware identity and commit reported by the device;
- whether Gateway was left running;
- whether the device ended in `idle`, `listening`, `speaking`, `error`, or
  bootloader state;
- whether any NVS, app partition, or release artifact was written.

## Immediate Stop Rules

Stop and return to the control tower if any of these happen:

- `go test ./...` stops compiling.
- A branch adds a flash-execute command not documented in this file and
  `FIRMWARE_RELEASE_DISCIPLINE.md`.
- A report or stdout includes a provider key, full proxy URL, Wi-Fi password,
  full audio websocket URL, or transcript text that should be redacted.
- A thread wants to mix T7 hardware writes with provider execution.
- StackChan overheats, white-screens, remains speaking after playback, or fails
  to return to a safe state.
- X21 or V21 runtime naming appears outside guardrails, tests, docs, or the
  explicit V21 adapter boundary.

## Resume Rules

To resume the paused main execution thread:

1. The control branch must be green on `go test ./...`, `git diff --check`,
   `go run ./cmd/a21 namespace-audit`, `make verify`,
   `go run ./cmd/a21 preflight`, and `go run ./cmd/a21 doctor`.
2. Dirty files must be classified and either committed, intentionally left as a
   narrow working set, or moved to a new branch.
3. The resumed thread receives a single-slice prompt with forbidden actions.
4. If hardware writes are needed, create a hardware-window branch/thread and do
   not run it in the background.
