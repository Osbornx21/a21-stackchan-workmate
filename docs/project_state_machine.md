# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-02.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-HW-PHYSICAL-XIAOZHI-GATEWAY-DOWNLINK-CANDIDATE`

Active child transition: `T-AUDIO-001`.

A21 has a Go-first Gateway/Core foundation, stock-compatible Xiaozhi transport,
official StackChan avatar/action relay, provider/V21 boundaries, a repo-carried
control workflow, and a freshly rebuilt official StackChan Xiaozhi-compatible
firmware candidate. The official-compatible candidate has been flashed in a
foreground hardware window, the latest firmware enters the official Xiaozhi
runtime directly, and the device has now connected to an A21 Gateway over the
stock Xiaozhi WebSocket path. A physical wake/turn produced Gateway uplink,
downlink, and barge-in candidate evidence. The user reports the audible sound
is still wrong and likely TTS-related, so `T-AUDIO-001` is active to isolate
TTS generation from Opus/downlink/device playback before any PRD acceptance.
Phase 1 host downlink isolation is now integrated in `xiaozhi-voice-bench` via
decoded Opus `downlink_audio_quality`; Phase 2 physical A/B remains pending. It
is not yet full PRD accepted because audible playback observation or trusted
device playback ack, real provider smoke, and custom wake proof remain missing.

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
| Xiaozhi audio/protocol | `S2-STOCK-OPUS-A21-GATEWAY-COMPAT-WARNING` | Physical path uses stock Xiaozhi profile and Opus uplink/downlink, but serial log `reports/a21-stackchan-physical-wake-serial-20260602-2130.log` shows repeated stock-firmware `Unknown message type: listen` warnings | `S3-STOCK-CLEAN-AUDIO-ISOLATED` |
| TTS/audio quality | `S1A-HOST-POST-OPUS-METRICS-ADDED-PHYSICAL-BAD-SOUND` | Audio-quality guard and downlink clarity code landed; `xiaozhi-voice-bench` now reports decoded Opus `downlink_audio_quality`; host bench selects `sherpa_onnx_tts`; current physical evidence lacks operator/instrument audible observation | `S2-TTS-VS-DOWNLINK-ROOT-CAUSE-ISOLATED` |

## Active Transition

### T-AUDIO-001: Isolate Xiaozhi TTS Quality From Opus/Device Playback

Current state:

- `S1-HOST-PCM-GUARDED-PHYSICAL-BAD-SOUND`

Target state:

- `S2-TTS-VS-DOWNLINK-ROOT-CAUSE-ISOLATED`

Trigger:

- Physical Gateway downlink is now proven as a candidate path, but the user
  reports that the audible sound is still wrong and likely TTS-related.
- Previous audio/TTS optimization commits are present, but they only prove
  host-side PCM/downlink guardrails, not physical audible quality.

Actions:

- Use `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md`.
- Keep the main thread as control tower.
- Dispatch a worker for host downlink objective isolation.
- Integrate the worker result so `xiaozhi-voice-bench` can report decoded
  Opus/downlink PCM quality without storing raw audio.
- Avoid hardware writes, NVS writes, provider/V21 execution, Mac audio
  playback, and PRD overclaiming.
- After worker return, run a foreground physical A/B only if the operator
  approves any Gateway/TTS profile change.

Acceptance conditions:

- The project can classify the bad sound as TTS/model, Opus/downlink, firmware
  playback, or stock-control compatibility with concrete evidence.
- Host-only and physical evidence remain labeled separately.
- Any follow-up fix has focused tests and does not reintroduce the old PCM
  bridge as a product path.

Failure state:

- `F-AUDIO-001-UNISOLATED` if reports still cannot distinguish TTS generation
  from post-Opus/downlink quality.
- `F-AUDIO-001-PHYSICAL-OVERCLAIM` if host-only evidence is promoted to PRD
  acceptance.
- `F-AUDIO-001-SCOPE-DRIFT` if a worker touches firmware, NVS, provider/V21,
  or Mac audio outside the plan.

Rollback path:

- Keep the current `sherpa_onnx_tts` host-local route as baseline.
- Revert any host-only report additions if they destabilize tests.
- Do not mutate firmware/NVS in this transition, so hardware rollback should
  not be needed.

Next state:

- `S2-TTS-VS-DOWNLINK-ROOT-CAUSE-ISOLATED`

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
| T-AUDIO-000: Read-only Xiaozhi audio/protocol audit | Completed | Worker thread `019e888d-f57d-7922-8e48-24b00255a122` found audio optimization landed, the physical audio path is stock-profile Xiaozhi Opus through A21 Gateway rather than old PCM bridge, and the most likely bad-sound boundary is TTS generation before Opus/downlink. |
| T-AUDIO-001a: Host downlink objective isolation | Completed | Worker thread `019e8895-43a6-7e23-a4f3-601f0451ab50`; `xiaozhi-voice-bench` now decodes captured binary downlink Opus frames and reports redacted aggregate `downlink_audio_quality` while preserving host-only candidate semantics. |

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-002b: Full StackChan physical acceptance after Gateway downlink | Gateway downlink exists, but device playback ack or operator/instrument audible observation is missing | Collect accepted audible playback evidence, device playback timing, and barge-in playback stop_done, then regenerate `xiaozhi-physical-evidence`. |
| T-AUDIO-002: Declare product voice quality acceptable | Physical sound is reported wrong; host post-Opus metrics exist, but foreground physical A/B and audible observation are still missing | Complete `T-AUDIO-001` Phase 2, run foreground physical A/B, then record operator/instrument observation. |
| T-PROVIDER-001: Real provider smoke on hardware path | Latest readiness still selects `mock`; `real_provider_smoke` missing | Run approved host-side provider smoke/rotation with keys outside firmware and redacted reports. |
| T-PRD-001: Declare full PRD physical acceptance | Candidate physical Gateway evidence exists but PRD accepted remains false | Close `T-HW-002b`, `T-PROVIDER-001`, and custom wake proof, then rerun product readiness. |
| T-FW-003: Custom wake-word product acceptance | Needs guarded flash and physical proof | Wake package review, false-wake rejection, operator wake proof. |

## Next Candidate Transitions

1. `T-AUDIO-001: Isolate Xiaozhi TTS Quality From Opus/Device Playback`
   - Phase 1 host downlink objective isolation is integrated.
   - Control thread next runs foreground physical A/B only after operator
     approval for any Gateway/TTS profile swap.
   - Output decides whether the next fix is TTS profile/model, Opus/downlink,
     firmware playback, or protocol cleanup.

2. `T-PROTOCOL-001: Clean Stock Xiaozhi Control Compatibility`
   - Review whether stock-device `listen` ack replies should be suppressed,
     gated, or changed while preserving host bench semantics.
   - Acceptance requires no more stock-firmware `Unknown message type: listen`
     warning in a fresh physical turn, without breaking listen/abort tests.

3. `T-PROVIDER-001: Real Provider Rotation Evidence On Hardware Path`
   - Run local ASR + cloud LLM + local TTS, cloud ASR + cloud LLM + local TTS,
     and cloud ASR + cloud LLM + cloud TTS through the selected Gateway profile.
   - Keep provider keys host-side only and reports redacted.
   - Record latency and quality evidence without changing firmware provider
     storage.
