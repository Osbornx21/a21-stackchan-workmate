# A21 Hardware Flash And Evidence Plan

Status: active plan.
Date: 2026-06-02.
Transition: `T-HW-001`.

## Background And Problem Definition

A21 now has host-ready Gateway protocol support and an integrated
`a21-stackchan-official-xiaozhi-compatible` firmware candidate that preserves
official StackChan avatar/action behavior and official Xiaozhi startup. Host
verification and a fresh mainline candidate rebuild have passed.

The remaining launch-critical gap is physical StackChan evidence: the candidate
must be flashed through a guarded foreground path, the device must connect to
A21 Gateway using the stock-compatible Xiaozhi protocol, and real audio,
barge-in, avatar/action, wake, provider, and readiness evidence must be
collected without false green.

## Current System State

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Current verified control HEAD before this plan: `9241998 docs(control):
  record integrated verification transition`.
- Correct product firmware candidate:
  `a21-stackchan-official-xiaozhi-compatible`.
- Fresh mainline build report:
  `reports/a21-stackchan-official-baseline-20260602-193126-1780399886135502000.json`.
- Fresh mainline app artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Fresh mainline app SHA-256:
  `053d3ba0d0c8690898967337a02bce8d3fd957899ebdabe4d9f4ae1e2b28c80d`.
- `make verify` has passed after the governance merge.
- No flash, NVS write, provider execution, V21 execution, Gateway runtime, or
  Mac audio playback was performed by the verification transition.

## Target State

- The exact flash artifact, board, upload port, firmware identity, and rollback
  path are known before any write.
- A no-write flash plan is generated and reviewed.
- Flash/NVS writes only occur in the main foreground hardware window after
  explicit operator confirmation.
- Physical evidence is recorded for:
  - StackChan online and connected to A21 Gateway;
  - stock-compatible Xiaozhi audio uplink/downlink;
  - first-audio latency and barge-in stop;
  - official avatar/action screen and motion behavior;
  - wake/custom wake status;
  - provider rotation through host-side keys only;
  - final product-readiness burn-down.

## Non-Goals

- Do not use X21 firmware packages or X21 build directories as A21 flash input.
- Do not resurrect the old PCM bridge as a product candidate.
- Do not store provider keys, Wi-Fi credentials, V21 internals, or proxy policy
  in firmware.
- Do not run background hardware writes from a worker.
- Do not claim full PRD acceptance from host, mock, simulated, or build-candidate
  evidence.
- Do not play audio from the Mac during this transition.

## Impact Scope

Expected worker scope:

- Read current plans, state machine, handoff log, Makefile targets, firmware
  reports, and hardware runbooks.
- Discover current serial devices and existing guarded flash commands using
  read-only commands only.
- Produce the exact no-write flash plan command, expected artifact identity,
  evidence command sequence, rollback command sequence, and operator checklist.
- Do not modify business code unless a blocking docs/report typo prevents
  handoff clarity.

Expected foreground control scope:

- Review the worker handoff.
- Run the no-write plan command.
- Execute the confirmed flash only with the operator present.
- Record physical observations and reports.

## Phased Execution

### Phase 1 - Worker Read-Only Hardware Preparation

Actions:

- Confirm branch, HEAD, and dirty status.
- Read `AGENTS.md`, `docs/project_state_machine.md`,
  `docs/agent_handoff_log.md`, this plan, and relevant firmware/hardware docs.
- List candidate serial ports with read-only commands.
- Inspect current Makefile flash-plan and flash-execute targets.
- Identify the latest official candidate report and artifact hash.
- Return an operator checklist and exact command sequence.

Acceptance:

- Worker changes no production files and runs no flash/NVS/provider/V21/audio
  command.
- Handoff includes branch/HEAD/dirty, serial candidates, no-write plan command,
  flash execute command template, evidence commands, rollback path, and risks.

### Phase 2 - Foreground No-Write Flash Plan

Actions:

- Main control runs the no-write flash plan with the selected upload port and
  artifact.
- Review the generated report before any write.

Acceptance:

- Report confirms A21 identity, board, artifact name, offsets, hashes, and
  target port.
- Report does not include secrets or full private credentials.

### Phase 3 - Confirmed Flash And Connection

Actions:

- Execute the guarded flash command only after explicit operator confirmation.
- Configure only allowed A21 device settings through the established controlled
  path.
- Start/verify Gateway profile for stock-compatible Xiaozhi.
- Confirm physical device connection.

Acceptance:

- Device boots the official Xiaozhi-compatible A21 candidate.
- Device connects to A21 Gateway.
- Official StackChan avatar/action remains visible and functional.

### Phase 4 - Physical Voice And Behavior Evidence

Actions:

- Run real voice turn evidence.
- Test barge-in while TTS is active.
- Verify avatar/action relay.
- Verify wake/custom wake behavior according to current package status.
- Rotate provider profiles as required by PRD.

Acceptance:

- Evidence reports separate physical, provider, wake, and readiness fields.
- Failures are recorded as missing or blocked, not hidden.

### Phase 5 - PRD Burn-Down Refresh

Actions:

- Run product readiness with latest reports.
- Update `docs/agent_handoff_log.md` and `docs/project_state_machine.md`.
- If physical evidence passes, prepare the next release/test package; if not,
  route the exact failing transition.

Acceptance:

- PRD status is stated in implemented/host-ready/simulated/candidate/physical
  accepted/missing terms.
- No false-green launch claim is made.

## Rollback Plan

- Preserve the previous known-good official StackChan package and restore it
  only through a guarded foreground flash path.
- If the A21 candidate boots but fails Gateway/audio behavior, collect boot and
  connection evidence before rollback.
- If flash planning identifies the wrong artifact, stop before write and rebuild
  from the integrated control branch.

## Risks

- Wrong port or wrong artifact would corrupt the hardware window; mitigate with
  no-write plan and explicit identity checks.
- Official source checkout dirt could confuse provenance; current build reports
  use `git_head_archive_read_only`, but this must remain visible.
- Network/proxy instability can affect Gateway/provider validation; local and
  LAN paths must stay direct.
- Provider testing can leak secrets if reports are not redacted; all provider
  evidence must remain host-side and redacted.
- Hardware behavior may fail despite host readiness; record the failure instead
  of changing acceptance labels.

## Human Confirmation Points

- Confirm the selected serial/upload port before any flash execute command.
- Confirm the operator is present before any flash/NVS write.
- Confirm whether to test providers before or after wake/custom wake physical
  proof if hardware time is limited.
- Confirm rollback package and command before flashing if the device must be
  returned to a known rental/pre-A21 state.
