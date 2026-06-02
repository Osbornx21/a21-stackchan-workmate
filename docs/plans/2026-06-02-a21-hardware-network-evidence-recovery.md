# A21 Hardware Network And Evidence Recovery Plan

Status: active plan.
Date: 2026-06-02.
Transition: `T-HW-002`.

## Background And Problem Definition

The previous control-tower thread collapsed during the foreground hardware
window after A21 had advanced beyond no-write planning. The official
Xiaozhi-compatible product candidate was built, NVS connection settings were
written, and the candidate was flashed. The latest firmware change replaced an
unsafe setup-uninstall autostart path with direct entry into the official
Xiaozhi runtime.

The current blocker is no longer the old setup/QR gate or the earlier watchdog
failure. The latest serial evidence shows the device entering Xiaozhi Wi-Fi
configuration mode and starting AP `Xiaozhi-6A61` after repeated `No AP found`
scans. A21 must now recover network/relay connectivity and collect physical
evidence without rewriting the architecture.

## Current System State

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Current HEAD: `4613946 fix(firmware): enter official xiaozhi runtime directly`.
- Working tree at recovery: clean.
- Latest official-compatible build report:
  `reports/a21-stackchan-official-baseline-20260602-204430-1780404270212940000.json`.
- Latest app artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Latest app SHA-256:
  `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`.
- Latest NVS execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-204515-1780404315388792000.json`.
- NVS evidence: write executed, servo calibration present, Wi-Fi credentials
  preserved, and OTA/WebSocket pointed at a temporary HTTP/WS relay host.
- Latest flash execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`.
- Flash evidence: write executed through T7 guard on `/dev/cu.usbmodem1101`,
  app offset `0x20000`, app SHA-256 above.
- Latest serial evidence:
  `reports/a21-stackchan-direct-xiaozhi-serial-20260602-2047.log`.
- Serial status: no setup/QR gate and no watchdog in the matching excerpt;
  device scanned Wi-Fi, found no AP, then entered Wi-Fi config AP
  `Xiaozhi-6A61`.

## Target State

- StackChan/CoreS3 joins a known available network without losing calibration.
- OTA and WebSocket route to a currently reachable A21 Gateway or approved
  relay.
- Device reaches A21 Gateway using stock-compatible Xiaozhi protocol.
- Physical evidence records microphone input, TTS downlink, audible playback,
  barge-in stop, official avatar/action behavior, wake behavior, provider
  routing, and product-readiness burn-down.
- Reports keep host/candidate evidence separate from physical acceptance.

## Non-Goals

- Do not re-open the old PCM bridge as the product path.
- Do not copy or execute X21 firmware/package inputs.
- Do not place provider keys, Wi-Fi passwords, proxy values, or V21 internals
  in firmware or reports.
- Do not play audio from the Mac; operator speech should trigger the device.
- Do not perform background flash/NVS writes from a worker.
- Do not claim full PRD launch until physical evidence and product-readiness
  agree.

## Impact Scope

Control conversation:

- Owns branch state, hardware-window authorization, final review, and updates to
  `docs/agent_handoff_log.md` and `docs/project_state_machine.md`.

Worker, if dispatched:

- May inspect reports, docs, Make targets, and current serial/network evidence.
- May prepare commands and checklists.
- Must not flash, write NVS, open serial for long capture, start Gateway
  long-running runtime, execute provider/V21, or play Mac audio unless the main
  foreground control explicitly reopens that tier.

## Phased Execution

### Phase 1 - Recovery Reconnaissance

Actions:

- Confirm branch, HEAD, and dirty status.
- Read `AGENTS.md`, `docs/agent_handoff_log.md`,
  `docs/project_state_machine.md`, this plan, and the latest relevant reports.
- Confirm whether `/dev/cu.usbmodem1101` or another intended port is present.
- Determine whether the relay host recorded in the latest NVS report is still
  live; if not, prepare a new approved relay or operator-provided network.

Acceptance:

- No file, service, provider, V21, serial, NVS, flash, or audio side effect is
  produced by reconnaissance.
- Recovery note states the current device blocker as network/relay
  provisioning, not setup/QR or Gateway protocol absence.

### Phase 2 - Network/Relay Decision

Actions:

- Choose one of two operator-visible routes:
  `device_wifi_config_ap` for configuring `Xiaozhi-6A61`, or
  `guarded_nvs_connection_update` for writing a new OTA/WS relay.
- Keep Wi-Fi credentials out of repo, reports, stdout summaries, and commits.
- If using NVS update, require foreground T7 guard and explicit confirmation.

Acceptance:

- Device can reach the chosen OTA endpoint.
- OTA returns a WebSocket URL reachable from the device network.
- Existing calibration and identity entries remain preserved.

### Phase 3 - Gateway Connection And Physical Voice

Actions:

- Start or verify the A21 Gateway profile only in the foreground control window.
- Trigger speech by operator voice, not Mac audio.
- Capture Gateway trace and redacted physical evidence.

Acceptance:

- Device connects to `/v1/xiaozhi`.
- Evidence shows real uplink microphone frames and TTS downlink frames.
- Audible playback is recorded through operator/instrument observation without
  storing raw audio or transcript bodies.

### Phase 4 - Barge-In, Avatar/Action, Wake, Provider

Actions:

- Test barge-in while TTS is active.
- Verify official StackChan avatar/action relay.
- Verify current wake behavior; custom wake remains blocked until guarded proof
  exists.
- Run provider rotation evidence only from approved host-side env and redacted
  reports.

Acceptance:

- Barge-in stop metric is present and under PRD threshold if the behavior passes.
- Avatar/action evidence confirms official StackChan behavior is preserved.
- Provider evidence contains no key, prompt, transcript, provider output, full
  URL, proxy, or local secret path.

### Phase 5 - Product Readiness And Handoff

Actions:

- Run product-readiness using the latest accepted reports.
- Update `docs/project_state_machine.md`.
- Append `docs/agent_handoff_log.md` with exact command results and blockers.

Acceptance:

- Final state is either `physical_accepted` with evidence, or a precise blocked
  transition with the failing stage, reason, rollback path, and next action.

## Rollback Plan

- If the official-compatible candidate boots but cannot connect, preserve boot
  and NVS evidence before changing anything.
- If the relay is stale, update only the connection route through a guarded path
  or operator-visible Wi-Fi configuration.
- If product behavior regresses, restore the previous known-good official
  Xiaozhi package through the guarded flash path and record rollback evidence.

## Risks

- Temporary relay domains expire quickly and can make a good firmware look
  broken.
- Wi-Fi AP availability can be the real blocker even when Gateway and firmware
  are correct.
- A background worker performing serial or NVS writes would break the hardware
  window discipline.
- Candidate flash success can be mistaken for PRD acceptance; it is only the
  entry point to physical evidence.

## Human Confirmation Points

- Confirm the network route: configure `Xiaozhi-6A61` manually or approve a new
  guarded NVS relay update.
- Confirm any Wi-Fi credentials out of band; do not paste them into reports.
- Confirm the intended serial/upload port before any NVS or flash write.
- Confirm when the operator is ready to speak to the StackChan for physical
  evidence.

## Worker Handoff Format

Worker must return:

- stage completed;
- files changed, if any;
- commands run and results;
- whether the result followed this plan;
- remaining blockers;
- exact next foreground action for the control thread.
