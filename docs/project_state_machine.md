# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-02.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-INT-FLASH-PLAN-READY`

A21 has a Go-first Gateway/Core foundation, stock-compatible Xiaozhi transport,
official StackChan avatar/action relay, provider/V21 boundaries, a repo-carried
control workflow, and a freshly rebuilt official StackChan Xiaozhi-compatible
firmware candidate with a dedicated no-write flash plan. It is not yet fully
PRD accepted because the physical StackChan flash and hardware evidence are
still pending.

Current control branch:

- `codex/a21-hardware-window-20260602-stackchan-prd`

Current notable baseline:

- `6f34091 feat(firmware): add official xiaozhi compatible flash plan`
- `59f30f4 docs(control): record official flash seam worker dispatch`
- `69c4bbe docs(control): add handoff and state machine workflow`
- `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`

## Module States

| Module | State | Evidence | Next state |
| --- | --- | --- | --- |
| Control workflow | `S1-REPO-CARRIED-CONTROL` | Commit `69c4bbe`; `docs/agent_handoff_log.md`, `docs/project_state_machine.md`, and `docs/plans/` exist | `S2-WORKER-TRANSITION-OPERATING` |
| Gateway `/v1/xiaozhi` | `S2-HOST-READY` | Stock-compatible hello/listen/abort, binary unwrap, Opus, turn/cancel, pacing, and downlink tests exist | `S3-PHYSICAL-VOICE-EVIDENCE` |
| Official StackChan avatar/action relay | `S2-HOST-READY` | Gateway/transport mapping exists for official StackChan packets | `S3-FLASHED-OFFICIAL-CANDIDATE` |
| Firmware candidate | `S3-FLASH-PLAN-APPROVED` | Commit `6f34091`; no-write report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-195247-1780401167369729000.json`; app SHA-256 `053d3ba0d0c8690898967337a02bce8d3fd957899ebdabe4d9f4ae1e2b28c80d` | `S4-FOREGROUND-FLASHED` |
| Provider hot-plug | `S2-HOST-READY` | Provider profiles and redacted smoke/evidence contracts exist | `S3-REAL-PROVIDER-ROTATION` |
| V21 adapter | `S2-HOST-READY` | Adapter contract exists; no firmware key or V21 internals should leak into A21 | `S3-PROFESSIONAL-EVIDENCE-RUN` |
| Memory/personality | `S1-IMPLEMENTED-HOST` | Host-side memory/personality work exists but needs current PRD burn-down refresh | `S2-READINESS-REVIEWED` |
| Physical StackChan acceptance | `S0-PENDING-HARDWARE` | No final flashed official-candidate evidence yet in this state document | `S1-FLASHED-AND-OBSERVED` |

## Active Transition

### T-HW-001: Flash Official Xiaozhi-Compatible Candidate And Collect Evidence

Current state:

- `S3-FLASH-PLAN-APPROVED`

Target state:

- `S3-PHYSICAL-VOICE-EVIDENCE`

Trigger:

- Integrated host verification passed and a fresh official Xiaozhi-compatible
  firmware candidate has been rebuilt from the control branch.
- Dedicated no-write plan/execute seams exist for the official
  Xiaozhi-compatible A21 product candidate.

Actions:

- Re-run the no-write flash plan if the port, build directory, or artifact
  changed.
- Execute flash only in a foreground operator window with explicit confirmation.
- Collect physical audio, barge-in, avatar/action, wake, provider, and readiness
  evidence.

Acceptance conditions:

- Official candidate flashes successfully to the intended StackChan hardware.
- Device connects to A21 Gateway using stock-compatible Xiaozhi protocol.
- Real audio output, microphone input, barge-in stop, official avatar/action,
  wake behavior, and provider rotation evidence are recorded without leaking
  keys or debug-only protocol fields.
- Host/mock/candidate evidence remains labeled separately from physical
  acceptance.

Failure state:

- `F-HW-001-WRONG-ARTIFACT` if the selected artifact is not the official
  Xiaozhi-compatible A21 candidate.
- `F-HW-001-UNCONFIRMED-WRITE` if a worker attempts background flash/NVS writes
  without explicit foreground confirmation.
- `F-HW-001-NO-PHYSICAL-EVIDENCE` if the device is flashed but evidence is not
  recorded.

Rollback path:

- Restore the previous known-good official StackChan package using the guarded
  flash path and record the rollback evidence.

Next state:

- `S3-PHYSICAL-VOICE-EVIDENCE`

## Completed Transitions

| Transition | Result | Notes |
| --- | --- | --- |
| T-FW-001: Freeze external X21 Xiaozhi builds | Completed | Commit `f7c95f0`; protects A21 from consuming external X21 Xiaozhi build dirs. |
| T-GW-001: Sync Xiaozhi turns to official StackChan | Completed | Commit `a36206f`; supports official StackChan turn synchronization. |
| T-GW-002: Relay official StackChan avatar actions | Completed | Commit `ea51c67`; maps Gateway state/action to official StackChan relay. |
| T-TR-001: Map A21 events to official StackChan frames | Completed | Commit `cdabe89`; keeps screen/action relay on official StackChan packet shapes. |
| T-FW-002: Add official Xiaozhi-compatible StackChan build | Completed host/build candidate | Commit `987bbb0`; candidate build lane exists, physical flash still pending. |
| T-GOV-001: Establish repo-carried workflow state | Completed | Commit `69c4bbe`; adds handoff log, state machine, and plan discipline. |
| T-VERIFY-001: Integrated host verification after governance merge | Completed | `make verify` passed; mainline official candidate rebuild passed with app SHA-256 `053d3ba0d0c8690898967337a02bce8d3fd957899ebdabe4d9f4ae1e2b28c80d`. |
| T-FW-004: Add official compatible candidate flash seam | Completed | Commit `6f34091`; no-write plan passed for `/dev/cu.usbmodem1101` with `dry_run=true`, `flash_allowed=false`, and app offset `0x20000`. |

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-001: Physical flash and full StackChan acceptance | Hardware/foreground operator window required | Explicit execute confirmation, operator presence, Gateway profile, no background worker writes. |
| T-PRD-001: Declare full PRD physical acceptance | Missing flashed-device evidence | Physical audio, barge-in, action/screen, wake, provider, and readiness reports. |
| T-FW-003: Custom wake-word product acceptance | Needs guarded flash and physical proof | Wake package review, false-wake rejection, operator wake proof. |

## Next Candidate Transitions

1. `T-HW-001: Flash Official Xiaozhi-Compatible Candidate And Collect Evidence`
   - Create a foreground hardware-window branch.
   - Re-run no-write flash plan if port or artifact changed.
   - Execute flash only with explicit confirmation and operator presence.

2. `T-PRD-002: Refresh PRD Burn-Down With Repo-Carried Evidence`
   - Re-read `docs/prd/A21_PRD.md` and latest reports.
   - Classify each requirement as implemented, host-ready, simulated,
     candidate, blocked by hardware, or missing.
   - Keep host/mock/candidate evidence separate from physical acceptance.

3. `T-PROVIDER-001: Real Provider Rotation Evidence On Hardware Path`
   - Run local ASR + cloud LLM + local TTS, cloud ASR + cloud LLM + local TTS,
     and cloud ASR + cloud LLM + cloud TTS through the selected Gateway profile.
   - Keep provider keys host-side only and reports redacted.
   - Record latency and quality evidence without changing firmware provider
     storage.
