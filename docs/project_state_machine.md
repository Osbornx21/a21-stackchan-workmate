# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-02.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-INT-HARDWARE-CANDIDATE`

A21 has a Go-first Gateway/Core foundation, stock-compatible Xiaozhi transport,
official StackChan avatar/action relay, provider/V21 boundaries, and a newly
added official StackChan Xiaozhi-compatible firmware candidate. It is not yet
fully PRD accepted because physical StackChan flashing and hardware evidence are
still pending.

Current control branch:

- `codex/a21-hardware-window-20260602-stackchan-prd`

Current notable baseline:

- `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`

## Module States

| Module | State | Evidence | Next state |
| --- | --- | --- | --- |
| Control workflow | `S0-BOOTSTRAPPING` | `T-GOV-001` adds docs for handoff/state-machine execution | `S1-REPO-CARRIED-CONTROL` |
| Gateway `/v1/xiaozhi` | `S2-HOST-READY` | Stock-compatible hello/listen/abort, binary unwrap, Opus, turn/cancel, pacing, and downlink tests exist | `S3-PHYSICAL-VOICE-EVIDENCE` |
| Official StackChan avatar/action relay | `S2-HOST-READY` | Gateway/transport mapping exists for official StackChan packets | `S3-FLASHED-OFFICIAL-CANDIDATE` |
| Firmware candidate | `S2-BUILD-CANDIDATE` | `a21-stackchan-official-xiaozhi-compatible` preserves official avatar/action and Xiaozhi start in build report | `S3-FLASH-PLAN-APPROVED` |
| Provider hot-plug | `S2-HOST-READY` | Provider profiles and redacted smoke/evidence contracts exist | `S3-REAL-PROVIDER-ROTATION` |
| V21 adapter | `S2-HOST-READY` | Adapter contract exists; no firmware key or V21 internals should leak into A21 | `S3-PROFESSIONAL-EVIDENCE-RUN` |
| Memory/personality | `S1-IMPLEMENTED-HOST` | Host-side memory/personality work exists but needs current PRD burn-down refresh | `S2-READINESS-REVIEWED` |
| Physical StackChan acceptance | `S0-PENDING-HARDWARE` | No final flashed official-candidate evidence yet in this state document | `S1-FLASHED-AND-OBSERVED` |

## Active Transition

### T-GOV-001: Establish Repo-Carried Handoff And State Machine Workflow

Current state:

- `S0-THREAD-CARRIED-CONTROL`

Target state:

- `S1-REPO-CARRIED-CONTROL`

Trigger:

- User requested that A21 stop relying on long Codex context and use main
  control, scoped workers, handoff logs, plans, and state transitions.

Actions:

- Update `AGENTS.md` with concise workflow discipline.
- Add `docs/agent_handoff_log.md`.
- Add `docs/project_state_machine.md`.
- Add a first detailed plan under `docs/plans/`.

Acceptance conditions:

- Only governance/docs files changed.
- `git diff --check` passes.
- The transition is committed on a scoped docs branch.
- The main control thread receives branch, commit, changed files, verification
  results, and next candidate transitions.

Failure state:

- `F-GOV-001-DOCS-DRIFT` if the transition touches business code, firmware,
  provider, Gateway runtime, or hardware paths.
- `F-GOV-001-UNCOMMITTED` if work remains dirty without an intentional handoff.

Rollback path:

- Revert the docs commit or remove the three new docs plus the `AGENTS.md`
  workflow section.

Next state:

- `S1-REPO-CARRIED-CONTROL`

## Completed Transitions

| Transition | Result | Notes |
| --- | --- | --- |
| T-FW-001: Freeze external X21 Xiaozhi builds | Completed | Commit `f7c95f0`; protects A21 from consuming external X21 Xiaozhi build dirs. |
| T-GW-001: Sync Xiaozhi turns to official StackChan | Completed | Commit `a36206f`; supports official StackChan turn synchronization. |
| T-GW-002: Relay official StackChan avatar actions | Completed | Commit `ea51c67`; maps Gateway state/action to official StackChan relay. |
| T-TR-001: Map A21 events to official StackChan frames | Completed | Commit `cdabe89`; keeps screen/action relay on official StackChan packet shapes. |
| T-FW-002: Add official Xiaozhi-compatible StackChan build | Completed host/build candidate | Commit `987bbb0`; candidate build lane exists, physical flash still pending. |

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-001: Physical flash and full StackChan acceptance | Hardware/foreground operator window required | Approved flash plan, explicit port/device ID, Gateway profile, no background worker writes. |
| T-PRD-001: Declare full PRD physical acceptance | Missing flashed-device evidence | Physical audio, barge-in, action/screen, wake, provider, and readiness reports. |
| T-FW-003: Custom wake-word product acceptance | Needs guarded flash and physical proof | Wake package review, false-wake rejection, operator wake proof. |

## Next Candidate Transitions

1. `T-VERIFY-001: Integrated Host Verification After Governance Merge`
   - Run `make verify` on the integrated control branch.
   - Rebuild `a21-stackchan-official-xiaozhi-compatible` from mainline if a
     fresh integrated report/artifact is needed.

2. `T-HW-001: Flash Official Xiaozhi-Compatible Candidate And Collect Evidence`
   - Create a foreground hardware-window branch.
   - Run no-write flash plan first.
   - Execute flash only with explicit confirmation and operator presence.

3. `T-PRD-002: Refresh PRD Burn-Down With Repo-Carried Evidence`
   - Re-read `docs/prd/A21_PRD.md` and latest reports.
   - Classify each requirement as implemented, host-ready, simulated,
     candidate, blocked by hardware, or missing.
   - Keep host/mock/candidate evidence separate from physical acceptance.

