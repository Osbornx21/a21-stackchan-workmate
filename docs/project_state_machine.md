# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-02.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-HW-FLASHED-OFFICIAL-RUNTIME-NETWORK-BLOCKED`

A21 has a Go-first Gateway/Core foundation, stock-compatible Xiaozhi transport,
official StackChan avatar/action relay, provider/V21 boundaries, a repo-carried
control workflow, and a freshly rebuilt official StackChan Xiaozhi-compatible
firmware candidate. The official-compatible candidate has now been flashed in a
foreground hardware window, and the latest firmware enters the official
Xiaozhi runtime directly. It is not yet fully PRD accepted because the device is
currently blocked at Wi-Fi/relay provisioning and physical evidence collection.

Current control branch:

- `codex/a21-hardware-window-20260602-stackchan-prd`

Current notable baseline:

- `4613946 fix(firmware): enter official xiaozhi runtime directly`
- `e7e9b03 feat(firmware): autostart official xiaozhi candidate`
- `37ef8f3 feat(firmware): add official xiaozhi nvs connection config`
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
| Firmware candidate | `S4-FOREGROUND-FLASHED-DIRECT-RUNTIME` | Commit `4613946`; flash report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`; app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d` | `S5-GATEWAY-CONNECTED` |
| Device connection/NVS | `S2-NVS-WRITTEN-RELAY-STALE-OR-WIFI-MISSING` | NVS report `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-204515-1780404315388792000.json`; latest serial log shows `No AP found` then AP `Xiaozhi-6A61` | `S3-WIFI-OR-RELAY-CONNECTED` |
| Provider hot-plug | `S2-HOST-READY` | Provider profiles and redacted smoke/evidence contracts exist | `S3-REAL-PROVIDER-ROTATION` |
| V21 adapter | `S2-HOST-READY` | Adapter contract exists; no firmware key or V21 internals should leak into A21 | `S3-PROFESSIONAL-EVIDENCE-RUN` |
| Memory/personality | `S1-IMPLEMENTED-HOST` | Host-side memory/personality work exists but needs current PRD burn-down refresh | `S2-READINESS-REVIEWED` |
| Physical StackChan acceptance | `S1-FLASHED-NOT-ONLINE` | Official-compatible candidate is flashed, but no current Gateway connection or PRD physical evidence is recorded after the direct-runtime fix | `S2-CONNECTED-AND-OBSERVED` |

## Active Transition

### T-HW-002: Recover Network/Relay And Collect Physical Evidence

Current state:

- `S-HW-FLASHED-OFFICIAL-RUNTIME-NETWORK-BLOCKED`

Target state:

- `S3-PHYSICAL-VOICE-EVIDENCE`

Trigger:

- The official Xiaozhi-compatible candidate was flashed through the T7
  foreground guard.
- The latest firmware no longer enters the setup/QR or watchdog failure path.
- Serial evidence now shows Wi-Fi scan failure and AP `Xiaozhi-6A61`.

Actions:

- Confirm current branch, HEAD, dirty state, serial port, and relay freshness.
- Choose either operator-visible Wi-Fi configuration through `Xiaozhi-6A61` or
  a guarded NVS connection update for a current relay.
- Reconnect the device to A21 Gateway using stock-compatible Xiaozhi protocol.
- Collect physical audio, barge-in, avatar/action, wake, provider, and
  readiness evidence.

Acceptance conditions:

- Device connects to A21 Gateway using stock-compatible Xiaozhi protocol.
- Real audio output, microphone input, barge-in stop, official avatar/action,
  wake behavior, and provider rotation evidence are recorded without leaking
  keys or debug-only protocol fields.
- Host/mock/candidate evidence remains labeled separately from physical
  acceptance.

Failure state:

- `F-HW-002-STALE-RELAY` if the recorded temporary relay no longer resolves or
  forwards OTA/WS.
- `F-HW-002-WIFI-NOT-CONFIGURED` if the device remains in AP config mode.
- `F-HW-002-UNCONFIRMED-WRITE` if any worker attempts background NVS/flash
  writes.
- `F-HW-002-NO-PHYSICAL-EVIDENCE` if the device connects but evidence is not
  recorded.

Rollback path:

- Preserve boot/NVS evidence before changing connection settings.
- Restore the previous known-good official StackChan package through a guarded
  foreground flash path if the A21 candidate must be reverted.

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
| T-FW-005: Add official compatible NVS connection config | Completed | Commit `37ef8f3`; T7 NVS write report `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-204515-1780404315388792000.json` preserved servo calibration and Wi-Fi credentials while updating OTA/WS route. |
| T-FW-006: Autostart official Xiaozhi candidate | Superseded | Commit `e7e9b03`; initial autostart removed setup gate but hit a setup-uninstall watchdog path during field testing. |
| T-FW-007: Enter official Xiaozhi runtime directly | Completed | Commit `4613946`; latest app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`; flashed through report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`. |

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-001b: Full StackChan physical acceptance after flash | Flash executed, but Gateway connection and physical PRD evidence remain incomplete | Complete `T-HW-002`, then collect accepted physical audio, barge-in, avatar/action, wake, provider, and readiness reports. |
| T-HW-002: Network/relay recovery after direct runtime flash | Device is in Wi-Fi config AP after `No AP found`; relay host may be temporary/stale | Operator-visible Wi-Fi configuration or foreground guarded NVS relay update. |
| T-PRD-001: Declare full PRD physical acceptance | Missing flashed-device evidence | Physical audio, barge-in, action/screen, wake, provider, and readiness reports. |
| T-FW-003: Custom wake-word product acceptance | Needs guarded flash and physical proof | Wake package review, false-wake rejection, operator wake proof. |

## Next Candidate Transitions

1. `T-HW-002: Recover Network/Relay And Collect Physical Evidence`
   - Confirm whether to configure AP `Xiaozhi-6A61` or write a fresh guarded
     NVS relay.
   - Reconnect the flashed candidate to A21 Gateway.
   - Collect physical audio, barge-in, avatar/action, wake, provider, and
     readiness evidence.

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
