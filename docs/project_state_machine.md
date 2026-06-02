# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-02.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-HW-PHYSICAL-XIAOZHI-GATEWAY-DOWNLINK-CANDIDATE`

A21 has a Go-first Gateway/Core foundation, stock-compatible Xiaozhi transport,
official StackChan avatar/action relay, provider/V21 boundaries, a repo-carried
control workflow, and a freshly rebuilt official StackChan Xiaozhi-compatible
firmware candidate. The official-compatible candidate has been flashed in a
foreground hardware window, the latest firmware enters the official Xiaozhi
runtime directly, and the device has now connected to an A21 Gateway over the
stock Xiaozhi WebSocket path. A physical wake/turn produced Gateway uplink,
downlink, and barge-in candidate evidence. It is not yet full PRD accepted
because audible playback observation or trusted device playback ack, real
provider smoke, and custom wake proof remain missing.

Current control branch:

- `codex/a21-hardware-window-20260602-stackchan-prd`

Current notable baseline:

- `4613946 fix(firmware): enter official xiaozhi runtime directly`
- `eeacbd3 docs(control): recover hardware network state`
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
| Gateway `/v1/xiaozhi` | `S3-PHYSICAL-DEVICE-CONNECTED` | Gateway on `127.0.0.1:21081` / LAN port `21081` accepted the physical device via stock Xiaozhi WebSocket; trace `a21-trace-44-1b-f6-e2-6a-60` has Opus uplink, VAD, ASR final, provider first content, TTS first audio, and Opus downlink | `S4-AUDIBLE-PLAYBACK-ACCEPTED` |
| Official StackChan avatar/action relay | `S2-HOST-READY` | Gateway/transport mapping exists for official StackChan packets | `S3-FLASHED-OFFICIAL-CANDIDATE` |
| Firmware candidate | `S5-GATEWAY-CONNECTED` | Commit `4613946`; flash report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`; app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d` | `S6-PRD-PHYSICAL-ACCEPTED` |
| Device connection/NVS | `S3-LAN-GATEWAY-CONNECTED` | Latest guarded NVS execution `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-212542-1780406742553040000.json` pointed OTA/WS to the LAN-bound A21 Gateway; reset serial log shows OTA connection to `21081` and activation | `S4-STABLE-RECONNECT-EVIDENCE` |
| Provider hot-plug | `S2-HOST-READY` | Provider profiles and redacted smoke/evidence contracts exist | `S3-REAL-PROVIDER-ROTATION` |
| V21 adapter | `S2-HOST-READY` | Adapter contract exists; no firmware key or V21 internals should leak into A21 | `S3-PROFESSIONAL-EVIDENCE-RUN` |
| Memory/personality | `S1-IMPLEMENTED-HOST` | Host-side memory/personality work exists but needs current PRD burn-down refresh | `S2-READINESS-REVIEWED` |
| Physical StackChan acceptance | `S2-CANDIDATE-GATEWAY-DOWNLINK` | `reports/a21-xiaozhi-physical-evidence-20260602-213147.097784000.json` reports physical device online, stock profile, mic delivery ratio 1, answer first downlink 555 ms, and barge-in metrics; PRD accepted remains false | `S3-AUDIBLE-PLAYBACK-AND-PRD-ACCEPTED` |

## Active Transition

### T-HW-002: Recover Network/Relay And Collect Physical Evidence

Current state:

- `S-HW-PHYSICAL-XIAOZHI-GATEWAY-DOWNLINK-CANDIDATE`

Target state:

- `S3-PHYSICAL-VOICE-EVIDENCE`

Trigger:

- The official Xiaozhi-compatible candidate was flashed through the T7
  foreground guard.
- The latest firmware no longer enters the setup/QR or watchdog failure path.
- The old temporary relay returned `503`; a LAN-bound A21 Gateway on port
  `21081` was verified by `/healthz` and `/xiaozhi/ota/`.
- A foreground guarded NVS update pointed OTA/WS to the LAN Gateway and the
  device connected through stock Xiaozhi WebSocket after wake.

Actions:

- Complete audible playback or trusted device playback ack evidence for the
  physical Xiaozhi turn.
- Preserve the physical Gateway trace and serial evidence without storing raw
  audio or transcript bodies.
- Close real provider smoke on the host side, then rerun readiness.
- Keep custom wake as blocked until guarded wake firmware flash and physical
  custom wake proof are recorded.

Acceptance conditions:

- Device connects to A21 Gateway using stock-compatible Xiaozhi protocol.
- Real microphone input, Gateway downlink, barge-in stop, official
  avatar/action, wake behavior, and provider rotation evidence are recorded
  without leaking keys or debug-only protocol fields.
- Audible playback observation or trusted device playback ack is present.
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
- `F-HW-002-AUDIBLE-ACK-MISSING` if Gateway downlink exists but physical
  audible playback or trusted device playback ack remains unproven.

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
| T-GOV-002: Recover hardware network state docs | Completed | Commit `eeacbd3`; reconciled collapsed control thread, active plan, state machine, and handoff log before foreground NVS execution. |
| T-HW-002a: Refresh connection route and prove physical Gateway downlink | Completed candidate | Latest NVS execution `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-212542-1780406742553040000.json`; serial logs `reports/a21-stackchan-direct-xiaozhi-serial-reset-20260602-2128.log` and `reports/a21-stackchan-physical-wake-serial-20260602-2130.log`; physical evidence report `reports/a21-xiaozhi-physical-evidence-20260602-213147.097784000.json`; readiness report `reports/a21-product-readiness-20260602-213204.json`. |

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-002b: Full StackChan physical acceptance after Gateway downlink | Gateway downlink exists, but device playback ack or operator/instrument audible observation is missing | Collect accepted audible playback evidence, device playback timing, and barge-in playback stop_done, then regenerate `xiaozhi-physical-evidence`. |
| T-PROVIDER-001: Real provider smoke on hardware path | Latest readiness still selects `mock`; `real_provider_smoke` missing | Run approved host-side provider smoke/rotation with keys outside firmware and redacted reports. |
| T-PRD-001: Declare full PRD physical acceptance | Candidate physical Gateway evidence exists but PRD accepted remains false | Close `T-HW-002b`, `T-PROVIDER-001`, and custom wake proof, then rerun product readiness. |
| T-FW-003: Custom wake-word product acceptance | Needs guarded flash and physical proof | Wake package review, false-wake rejection, operator wake proof. |

## Next Candidate Transitions

1. `T-HW-003: Close Physical Audible Playback And PRD Evidence`
   - Collect operator or instrument audible playback observation matched to
     trace `a21-trace-44-1b-f6-e2-6a-60` or a fresh trace.
   - Capture trusted device playback ack/timing and barge-in stop_done if
     available.
   - Rerun `xiaozhi-physical-evidence` and product readiness.

2. `T-PROVIDER-001: Real Provider Rotation Evidence On Hardware Path`
   - Run local ASR + cloud LLM + local TTS, cloud ASR + cloud LLM + local TTS,
     and cloud ASR + cloud LLM + cloud TTS through the selected Gateway profile.
   - Keep provider keys host-side only and reports redacted.
   - Record latency and quality evidence without changing firmware provider
     storage.

3. `T-WAKE-001: Guarded Custom Wake Physical Proof`
   - Use the existing wake firmware package only through a guarded hardware
     window.
   - Prove the custom wake phrase, false-wake rejection, stock-wake rejection,
     and product readiness without fake-greening the current stock wake.
