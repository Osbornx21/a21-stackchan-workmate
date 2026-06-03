# A21 Agent Handoff Log

Status: active handoff document.
Last updated: 2026-06-03.

This log is the recovery surface for Codex workers and future control-tower
threads. Every work round must add or update an entry before handoff. Keep this
file operational and redacted: no provider keys, Wi-Fi credentials, raw
transcripts, full private URLs, or local secret values.

Each entry should include:

- goal;
- actual completed work;
- files changed;
- unfinished items;
- known risks or blockers;
- recommended next action;
- test, build, or runtime results;
- failure location and reason, when applicable.

## 2026-06-02 - T-GOV-001 - Establish Repo-Carried Workflow State

Goal:

- Move A21 from long conversation memory toward repository-carried state.
- Add a handoff log, project state machine, and plan directory so new workers
  can resume without redoing finished work.
- Keep this transition documentation-only.

Actual completed work:

- Added control-tower workflow rules to `AGENTS.md`.
- Created this handoff log as the canonical per-round recovery record.
- Created `docs/project_state_machine.md` with initial project/module states,
  active/completed/blocked transitions, and next candidate transitions.
- Created `docs/plans/2026-06-02-a21-workflow-state-machine.md` as the detailed
  plan for this transition.

Files changed:

- `AGENTS.md`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-a21-workflow-state-machine.md`

Current repository state:

- Control branch before this transition: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Worker branch for this transition: `codex/a21-workflow-state-machine-20260602`.
- Baseline commit before governance docs: `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`.
- The correct A21 firmware candidate is `a21-stackchan-official-xiaozhi-compatible`.
- The firmware candidate build evidence recorded before this transition had
  `official_avatar_action_preserved=true`,
  `official_xiaozhi_start_preserved=true`, and
  `minimal_bridge_screen=false`. No flash or NVS write was executed.

Unfinished items:

- Mainline still needs a fresh `make verify` after the governance commit is
  integrated.
- Mainline still needs a fresh
  `make a21-stackchan-official-xiaozhi-compatible-build` if the control tower
  wants a report/artifact generated from the final integrated branch rather
  than the worker branch.
- Physical PRD acceptance remains pending until the official candidate is
  flashed and real StackChan evidence is recorded.

Known risks and blockers:

- Do not treat host/mock/candidate evidence as PRD physical acceptance.
- Do not mix governance documentation work with firmware, provider, Gateway, or
  hardware-write branches.
- The historical official StackChan source path may live near old X21 material;
  A21 may use it only as a read-only official-source export, never as an X21
  firmware package source.

Validation results for this transition:

- `git status --short --branch`: confirmed
  `codex/a21-workflow-state-machine-20260602` before docs edits.
- `git diff --check`: passed with no output before commit.
- Scoped secret scan over the changed docs found only redaction-rule wording,
  not actual credentials.
- No provider, V21, Gateway runtime, hardware, NVS, flash, or Mac-audio command
  should be executed by this transition.

Recommended next action:

- Integrate the governance-doc commit into the control branch.
- Run the documented host-only verification.
- Continue with the next explicit transition from `docs/project_state_machine.md`.

## 2026-06-02 - T-VERIFY-001 - Integrated Host Verification After Governance Merge

Goal:

- Resume cleanly after conversation compaction.
- Verify that the integrated control branch has both the correct official
  Xiaozhi-compatible firmware candidate and the repo-carried workflow docs.
- Generate fresh mainline build evidence for the firmware candidate without
  flashing hardware.

Actual completed work:

- Confirmed control branch `codex/a21-hardware-window-20260602-stackchan-prd`
  at `69c4bbe docs(control): add handoff and state machine workflow`.
- Confirmed the previous firmware candidate commit is integrated at
  `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`.
- Ran full host verification through `make verify`.
- Rebuilt `a21-stackchan-official-xiaozhi-compatible` from the integrated
  control branch.

Files changed:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Current repository state:

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Latest control HEAD before this log update: `69c4bbe`.
- Working tree was clean before this log update.
- Firmware candidate remains `a21-stackchan-official-xiaozhi-compatible`.

Validation results:

- `go test ./internal/app -run 'StackChanOfficial|Official|Firmware|Xiaozhi|Frozen' -count=1`: passed.
- `make verify`: passed, including `go test ./...` and `git diff --check`.
- `make a21-stackchan-official-xiaozhi-compatible-build`: passed.
- Fresh mainline report:
  `reports/a21-stackchan-official-baseline-20260602-193126-1780399886135502000.json`.
- Fresh mainline app artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Fresh mainline app SHA-256:
  `053d3ba0d0c8690898967337a02bce8d3fd957899ebdabe4d9f4ae1e2b28c80d`.
- Report evidence confirms `official_avatar_action_preserved=true`,
  `official_xiaozhi_start_preserved=true`, and
  `minimal_bridge_screen=false`.

Unfinished items:

- No physical StackChan flash, NVS write, provider execution, V21 execution, or
  Mac audio playback was performed in this transition.
- Physical PRD acceptance remains pending until foreground hardware evidence is
  collected.
- The official source checkout used for read-only export was reported dirty by
  the build tool; the build still used `git_head_archive_read_only`, so the
  candidate came from the source Git HEAD rather than local source dirt.

Known risks and blockers:

- Do not claim physical acceptance from this host/build evidence.
- Next hardware flash must be foreground-controlled with explicit port/device
  confirmation.
- Keep the old PCM bridge lane diagnostic-only; it is not the product firmware
  candidate.

Recommended next action:

- Execute `T-HW-001` in a foreground hardware window: no-write flash plan first,
  then explicit confirmed flash of the official Xiaozhi-compatible candidate,
  then collect audio, barge-in, avatar/action, wake, provider, and readiness
  evidence.

## 2026-06-02 - T-HW-001 - Plan And Dispatch No-Write Hardware Preparation

Goal:

- Move from verified host/build candidate into the controlled hardware
  transition without letting the main conversation perform background hardware
  writes.
- Create the detailed hardware flash/evidence plan required before any large
  hardware transition.
- Dispatch a scoped worker for read-only/no-write preparation.

Actual completed work:

- Created `docs/plans/2026-06-02-a21-hardware-flash-evidence.md`.
- Committed the plan on the control branch as
  `114f1e3 docs(control): plan hardware flash evidence transition`.
- Launched worker thread `019e881d-96be-79e1-bc4f-d19f90a19dba` titled
  `A21 T-HW-001 no-write flash plan`.
- Worker branch/worktree: `codex/a21-hw-flash-plan-20260602` in a separate
  Codex worktree.

Files changed:

- `docs/agent_handoff_log.md`
- `docs/plans/2026-06-02-a21-hardware-flash-evidence.md`

Worker boundary:

- Read-only/no-write hardware preparation only.
- Allowed: read docs, inspect Makefile targets, list serial port candidates,
  identify no-write plan command, return flash/evidence/rollback templates.
- Forbidden: flash, NVS write, serial write/open, provider execute, V21 execute,
  long-running Gateway start, Mac audio playback, and business-code edits.

Current status:

- Control branch is clean at `114f1e3` before this handoff-log update.
- Worker is active and has attached to branch
  `codex/a21-hw-flash-plan-20260602`.
- No physical hardware write has been executed by the control thread.

Known risks and blockers:

- The new official Xiaozhi-compatible candidate still needs an explicit
  candidate-specific flash-plan path or a verified existing target; the worker
  is checking this now.
- Serial/upload port must be confirmed in the foreground before any write.
- Rollback package choice must be confirmed before flashing if the device must
  return to a previous known-good state.

Recommended next action:

- Read the worker handoff.
- If a no-write flash-plan command exists, run it in the main foreground
  control thread.
- If the command is missing, route a narrow implementation transition for the
  missing official-candidate flash-plan target before any hardware write.

## 2026-06-02 - T-FW-004 - Dispatch Official Compatible Candidate Flash Seam

Goal:

- Convert the no-write hardware-prep finding into the smallest implementation
  transition needed before physical flash.
- Keep the main conversation in control-tower mode instead of implementing the
  firmware flash seam directly.

Actual completed work:

- Reviewed the existing flash-plan surfaces enough to confirm the worker
  finding: `xiaozhi-firmware-flash-plan` is not valid for the official
  Xiaozhi-compatible A21 product candidate because it expects an app named
  `xiaozhi.bin`, while the correct candidate flash args reference
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Launched implementation worker thread
  `019e8821-5dd0-74b1-8671-58e4fafabdcc` titled
  `A21 T-FW-004 official candidate flash seam`.
- Worker branch/worktree: `codex/a21-official-compatible-flash-plan-20260602`
  in a separate Codex worktree.

Files changed:

- `docs/agent_handoff_log.md`

Worker boundary:

- Add dedicated no-write plan and guarded execute commands for
  `a21-stackchan-official-xiaozhi-compatible`.
- Expected command names:
  `a21-stackchan-official-xiaozhi-compatible-flash-plan` and
  `a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- Use TDD in `internal/app/official_stackchan_test.go`.
- Wire only the necessary CLI/Makefile/report/docs surfaces.
- Forbidden: real flash, NVS write, serial monitor/upload, provider execute,
  V21 execute, Gateway long-running runtime, and Mac audio.

Current status:

- Control branch is clean at `796c9a3` before this handoff-log update.
- Implementation worker is active.
- No hardware write has been executed.

Recommended next action:

- Read the worker handoff.
- If tests and commit pass, cherry-pick the focused implementation commit to
  the control branch.
- Run the new no-write flash-plan command on the foreground control thread for
  `/dev/cu.usbmodem1101`.

## 2026-06-02 - T-FW-004 - Complete Official Compatible Candidate Flash Seam

Goal:

- Finish the dedicated plan/execute seam for the official
  Xiaozhi-compatible A21 StackChan product candidate.
- Keep the product candidate separated from legacy `xiaozhi.bin` and the
  diagnostic PCM bridge lane.
- Produce no-write evidence that the exact current candidate can be planned for
  the current foreground serial port without flashing.

Actual completed work:

- Took over the stalled implementation worker worktree after instructing the
  worker to stop.
- Added dedicated CLI and Make targets:
  `a21-stackchan-official-xiaozhi-compatible-flash-plan` and
  `a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- Added a product-candidate flash report schema that records only basename/file
  artifact information for the candidate receipt, while keeping full paths
  internal to execution.
- Added guarded execute wiring for
  `WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP`.
- Registered the new execute command in `runtimeguard` as a T7 hardware-write
  command.
- Cherry-picked worker commit `cbbd70e` to the control branch as
  `6f34091 feat(firmware): add official xiaozhi compatible flash plan`.

Files changed:

- `Makefile`
- `internal/app/app_plan_execute.go`
- `internal/app/official_stackchan.go`
- `internal/app/official_stackchan_test.go`
- `internal/runtimeguard/control.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Validation results:

- Worker scoped test:
  `go test ./internal/app -run 'Official.*Xiaozhi.*Flash|StackChanOfficial|XiaozhiFirmware|Firmware|Frozen' -count=1`
  passed.
- Worker runtimeguard scoped test:
  `go test ./internal/runtimeguard -run 'Control|Default|Firmware|Xiaozhi' -count=1`
  passed.
- Worker `make verify` passed, including `go test ./...` and
  `git diff --check`.
- Control branch `make verify` passed after cherry-pick, including
  `go test ./...` and `git diff --check`.
- Control branch no-write plan passed:
  `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 make a21-stackchan-official-xiaozhi-compatible-flash-plan`.
- No-write plan report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-195247-1780401167369729000.json`.
- Planned app part:
  `a21-stackchan-official-xiaozhi-compatible.bin` at offset `0x20000`.
- Planned app SHA-256:
  `053d3ba0d0c8690898967337a02bce8d3fd957899ebdabe4d9f4ae1e2b28c80d`.
- The plan receipt reports `dry_run=true`, `flash_allowed=false`, and
  `flash_executed=false`.

Unfinished items:

- No flash, NVS write, serial monitor, provider execution, V21 execution,
  Gateway long-running runtime, or Mac audio playback was performed in this
  transition.
- Physical PRD acceptance remains pending until the official candidate is
  flashed and real StackChan audio, microphone, barge-in, avatar/action, wake,
  provider, and readiness evidence is recorded.

Known risks and blockers:

- The generated no-write report is local evidence under `reports/`; it is not a
  physical acceptance report.
- Execute remains intentionally gated by explicit confirmation and
  `runtimeguard` hardware-write checks.
- The current candidate artifact is under `/tmp/a21-stackchan-official-build`;
  rebuild or re-run the no-write plan before flashing if the build directory is
  refreshed.

Recommended next action:

- Execute the foreground hardware transition: re-run the no-write plan if the
  port/artifact changed, then run the guarded execute command only with operator
  presence and explicit confirmation.
- After flash, collect the PRD evidence bundle: connection, audible TTS,
  microphone input, barge-in stop, official avatar/action, wake, provider
  rotation, and readiness reports.

## 2026-06-02 - T-RECOVERY-001 - Recover Control Tower After Thread Collapse

Goal:

- Recover architecture-control ownership after network instability interrupted
  Codex thread `019e7f81-e218-7df1-8743-1ed66e7ddd37`.
- Read the interrupted thread progress and reconcile it with the current clean
  checkout.
- Update repository-carried workflow/state docs only; do not touch business
  code, firmware logic, runtime services, provider/V21 execution, NVS, flash, or
  Mac audio.

Actual completed work:

- Read the interrupted thread through all available pages.
- Confirmed the latest meaningful hardware-window state:
  - official-compatible flash seam existed and was used;
  - NVS connection settings were written under guard;
  - initial autostart removed the setup/QR gate but caused a watchdog through
    the setup-uninstall path;
  - latest firmware commit `4613946` changed the candidate to enter official
    Xiaozhi runtime directly;
  - latest serial evidence now shows Wi-Fi scan failure and config AP
    `Xiaozhi-6A61`, so the active blocker is network/relay provisioning plus
    physical evidence, not the old setup/QR or WDT failure.
- Updated `AGENTS.md` with explicit plan/worker/handoff summary requirements.
- Added the current continuation plan
  `docs/plans/2026-06-02-a21-hardware-network-evidence-recovery.md`.
- Marked the older hardware flash/evidence plan as partially completed and
  pointed it to the continuation plan.
- Updated `docs/project_state_machine.md` from flash-plan-ready to
  `S-HW-FLASHED-OFFICIAL-RUNTIME-NETWORK-BLOCKED`.

Files changed:

- `AGENTS.md`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-a21-hardware-flash-evidence.md`
- `docs/plans/2026-06-02-a21-hardware-network-evidence-recovery.md`

Current repository state:

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Current HEAD before this documentation update: `4613946`.
- Working tree before this documentation update: clean.
- Current total state after this documentation update:
  `S-HW-FLASHED-OFFICIAL-RUNTIME-NETWORK-BLOCKED`.

Key evidence from current checkout:

- Latest build report:
  `reports/a21-stackchan-official-baseline-20260602-204430-1780404270212940000.json`.
- Latest app SHA-256:
  `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`.
- Latest guarded NVS execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-204515-1780404315388792000.json`.
- Latest guarded flash execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`.
- Latest serial evidence:
  `reports/a21-stackchan-direct-xiaozhi-serial-20260602-2047.log`.

Unfinished items:

- No new physical evidence was collected in this recovery/docs transition.
- Device is not yet recorded as connected to A21 Gateway after the
  direct-runtime firmware fix.
- The latest temporary relay may be stale; network/relay decision must be made
  before the next physical evidence window.
- Full PRD acceptance remains blocked until physical audio, microphone,
  barge-in, official avatar/action, wake, provider, and product-readiness
  evidence pass.

Known risks and blockers:

- Do not let a worker perform background NVS/flash/serial/runtime actions.
- Do not treat the flashed app or NVS write as physical PRD acceptance.
- Do not use Mac audio for future physical prompts; operator speech should
  trigger StackChan.
- Temporary relay hosts expire quickly; stale relay failure must not be
  misdiagnosed as Gateway or firmware protocol failure.

Validation results:

- `git diff --check`: passed.
- Scoped secret scan over changed governance docs: no matches for key, Bearer,
  password, or token patterns.
- `make verify` was not run because this transition intentionally touched only
  governance/state documents and did not change Go, firmware, runtime,
  provider, or V21 code.

Recommended next action:

- Execute `T-HW-002` from
  `docs/plans/2026-06-02-a21-hardware-network-evidence-recovery.md`.
- First decide the network route: operator-visible `Xiaozhi-6A61` Wi-Fi
  configuration or a foreground guarded NVS relay update.
- Then reconnect the device to A21 Gateway and collect physical PRD evidence.

## 2026-06-02 - T-HW-002 - Recover Network Route And Prove Physical Xiaozhi Candidate

Goal:

- Continue under the repo-carried control workflow and quickly run the A21
  physical main flow without redesigning the architecture.
- Recover the flashed official Xiaozhi-compatible StackChan from stale
  relay/Wi-Fi state to an A21 Gateway connection.
- Keep evidence honest: candidate Gateway downlink is progress, not full PRD
  launch acceptance.

Actual completed work:

- Spawned worker thread `019e887f-b94c-78b0-8edc-7315ed56b59d` for read-only
  `T-HW-002 Phase 1` relay/network reconnaissance.
- Worker confirmed the previous temporary relay resolved but OTA/WS probes
  returned `503`, so it should be treated as stale.
- Confirmed a LAN-bound A21 Gateway was already running on port `21081` and
  returning healthy `/healthz`, OTA discovery, and stock Xiaozhi WebSocket
  route information.
- Committed the previous control/state recovery docs as
  `eeacbd3 docs(control): recover hardware network state` so T7 guarded NVS
  execution could run from a clean worktree.
- Ran a no-write NVS plan for the LAN route; status was `ready` and
  `write_executed=false`.
- Ran foreground T7 guarded NVS execution on `/dev/cu.usbmodem1101`; status
  `passed`, `write_executed=true`, Wi-Fi credentials and servo calibration
  preserved, and only Xiaozhi connection keys mutated.
- Hard-reset the device through esptool and captured boot serial evidence.
- Captured physical wake/turn serial evidence after operator wake:
  WakeNet detected `Hi,Stack Chan`, the device connected to
  `ws://192.168.1.20:21081/v1/xiaozhi`, and state moved through
  `listening`/`speaking` cycles.
- Gateway `/v1/devices` showed physical device `44:1b:f6:e2:6a:60` online
  with stock Xiaozhi WebSocket, microphone uplink, and speaker downlink
  capabilities.
- Generated physical Xiaozhi evidence and readiness reports.

Files changed:

- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-a21-hardware-network-evidence-recovery.md`
- `docs/agent_handoff_log.md`

Current repository state:

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- HEAD during foreground hardware run: `eeacbd3`.
- Current total state after this transition:
  `S-HW-PHYSICAL-XIAOZHI-GATEWAY-DOWNLINK-CANDIDATE`.

Key evidence from this round:

- Guarded NVS execution:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-212542-1780406742553040000.json`.
- Reset serial log:
  `reports/a21-stackchan-direct-xiaozhi-serial-reset-20260602-2128.log`.
- Physical wake/turn serial log:
  `reports/a21-stackchan-physical-wake-serial-20260602-2130.log`.
- Physical Xiaozhi evidence:
  `reports/a21-xiaozhi-physical-evidence-20260602-213147.097784000.json`.
- Product readiness:
  `reports/a21-product-readiness-20260602-213204.json`.
- Server-side readiness bundle:
  `reports/a21-server-side-readiness-bundle-20260602-213204.json`.

Important results:

- Physical device is online through stock Xiaozhi profile.
- Physical microphone uplink reached Gateway: 64 frames, delivery ratio 1.
- Gateway downlink reached physical device path; answer first downlink was
  555 ms in the physical evidence report.
- Barge-in trace metrics are present.
- `product-readiness` reports `demo_ready=true`, `launch_ready=false`,
  `status=server_side_blocked`.

Unfinished items:

- Full physical PRD acceptance is not green.
- Missing device playback ack or operator/instrument audible playback
  observation.
- Missing device downlink first-frame timing and speech-end to first audible
  response timing.
- Missing barge-in playback `stop_done` evidence.
- Missing real provider smoke; readiness currently selects `mock`.
- Custom wake product proof is still blocked by guarded wake firmware flash and
  physical wake acceptance.

Known risks and blockers:

- Do not treat `candidate_gateway_downlink` as audible playback acceptance.
- Port `21081` is a foreground LAN-bound Gateway route used to run the hardware
  window quickly; document or retire it before treating it as a durable launch
  route.
- Official StackChan serial shows repeated `Unknown message type: listen`
  warnings during the turn; this did not block uplink/downlink evidence but
  should be reviewed before declaring product polish.
- Old `stackchan-accept` diagnostic gates remain blocked because they expect
  diagnostic probe/runtime echo fields, not the stock Xiaozhi capability shape.

Validation results:

- `git diff --check`: passed.
- Scoped secret scan over changed handoff/state/plan docs: no matches for key,
  Bearer, password, or token patterns.
- Previous docs checkpoint committed as `eeacbd3`.
- Foreground NVS execute passed with T7 control guard and clean worktree.
- `xiaozhi-physical-evidence` passed and wrote candidate physical evidence.
- `product-readiness --use-latest-reports` passed and correctly kept
  `launch_ready=false`.

Recommended next action:

- Execute `T-HW-003: Close Physical Audible Playback And PRD Evidence`.
- Capture either trusted device playback ack/runtime echo or an approved
  operator/instrument audible observation matched to a fresh physical trace.
- Then rerun `xiaozhi-physical-evidence`, `product-readiness`, and
  `server-side-readiness-bundle`.

## 2026-06-02 - T-AUDIO-001 - Isolate Xiaozhi TTS Sound Quality

Goal:

- Answer whether the audio-quality/TTS optimization actually landed.
- Check whether the current physical path is fully Xiaozhi audio/protocol or
  still using the old A21 diagnostic/PCM bridge path.
- Continue the hardware main flow by isolating the bad sound as TTS generation,
  Opus/downlink, firmware speaker playback, or stock-control compatibility.
- Keep the main thread in control-tower mode and route implementation/evidence
  work to a worker.

Actual completed work:

- Confirmed current branch `codex/a21-hardware-window-20260602-stackchan-prd`
  at `49b9e458d44b` with a clean worktree before this docs update.
- Dispatched read-only worker
  `019e888d-f57d-7922-8e48-24b00255a122` for audio/protocol audit.
- Read worker result: audio optimization landed; current physical audio path
  is stock-profile Xiaozhi Opus through A21 Gateway, not the old PCM bridge;
  the most likely bad-sound boundary is Gateway TTS generation/PCM before
  Opus/downlink.
- Rechecked current Gateway health/device state: LAN Gateway on port `21081`
  is healthy; physical device was online with stock Xiaozhi Opus ingress and
  downlink capabilities during the checked window.
- Inspected source around xiaozhi listen handling, TTS pipeline, local TTS
  adapter, PCM quality guard, and Opus downlink.
- Confirmed the stock firmware serial log repeatedly reports `Unknown message
  type: listen` while still entering `speaking`, so the binary audio path works
  but stock-control cleanliness still needs a follow-up.
- Created
  `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md`.
- Updated `docs/project_state_machine.md` with active child transition
  `T-AUDIO-001` and blocked transition `T-AUDIO-002`.
- Dispatched implementation/evidence worker
  `019e8895-43a6-7e23-a4f3-601f0451ab50` titled
  `Isolate TTS audio quality`.

Files changed:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md`

Current repository state:

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Current HEAD before this documentation update: `49b9e458d44b`.
- Worktree before docs update: clean.
- Current total state remains
  `S-HW-PHYSICAL-XIAOZHI-GATEWAY-DOWNLINK-CANDIDATE` with active child
  `T-AUDIO-001`.

Key evidence and conclusions:

- Audio-quality/downlink optimization landed in code and history, including
  the downlink clarity, PCM quality guard, host loopback quality gate, and
  xiaozhi downlink clarity test commits.
- The current physical product route is stock Xiaozhi profile plus A21 Gateway
  `/v1/xiaozhi`: stock-compatible WebSocket, Opus uplink/downlink, listen and
  abort, A21-owned ASR/text/TTS generation, and paced Opus binary downlink.
- The old A21 PCM bridge is not the current product audio path.
- This is not upstream Xiaozhi end-to-end; it is a stock-compatible A21 Gateway
  implementation.
- The physical report remains `candidate_gateway_downlink`, not audible
  playback acceptance.
- The current host-local evidence selects `sherpa_onnx_tts` through
  `A21_TTS_FAST_PROFILE`; an IndexTTS2/voice-clone smoke report exists but was
  not the current physical-path TTS evidence.

Unfinished items:

- Worker `019e8895-43a6-7e23-a4f3-601f0451ab50` must complete Phase 1 host
  downlink objective isolation and return whether any report/test addition was
  needed.
- Physical foreground A/B still needs to compare current TTS against a known
  good or alternate TTS candidate through the same stock Xiaozhi route.
- Device playback ack, operator/instrument audible observation, device
  downlink first-frame timing, first-audible timing, and barge-in `stop_done`
  remain missing.
- Real provider smoke and custom wake proof remain outside this audio RCA
  transition and are still not launch green.

Known risks and blockers:

- Host PCM quality can pass while voice naturalness/prosody still sounds bad.
- Stock firmware `Unknown message type: listen` warnings can pollute protocol
  polish even if they are not the primary TTS-quality root cause.
- Do not switch TTS profiles in a background worker because the physical device
  is currently using a foreground Gateway route.
- Do not claim PRD physical acceptance until audible observation or trusted
  playback ack exists.

Validation results:

- `curl -sS --max-time 3 http://127.0.0.1:21080/healthz`: healthy A21 Gateway.
- `curl -sS --max-time 3 http://127.0.0.1:21081/healthz`: healthy A21 Gateway.
- `curl -sS --max-time 3 http://127.0.0.1:21081/v1/devices`: physical device
  online during the checked window with stock Xiaozhi Opus ingress/downlink
  capabilities.
- `curl -sS --max-time 3 http://127.0.0.1:21081/v1/providers/voice/health`:
  healthy mock voice provider surface; it does not expose xiaozhi product-chain
  TTS selection.
- `go test ./internal/audio ./internal/providers ./internal/gateway ./internal/app -run 'PCMQuality|LocalTTS|VoicePipeline|Xiaozhi.*Opus|Xiaozhi.*Stock|XiaozhiWebSocketListen|XiaozhiPhysicalEvidence' -count=1`:
  passed.
- `git diff --check`: passed.

Recommended next action:

- Read worker `019e8895-43a6-7e23-a4f3-601f0451ab50` when it finishes.
- If worker adds host downlink decoded `audio_quality` evidence and tests pass,
  review and integrate the worker change.
- Then run the foreground physical A/B from the plan, changing only the TTS
  candidate/profile while preserving the same firmware, NVS route, Gateway
  port, and stock Xiaozhi path.

## 2026-06-02 - T-AUDIO-001a - Integrate Xiaozhi Downlink Quality Evidence

Goal:

- Integrate worker `019e8895-43a6-7e23-a4f3-601f0451ab50` Phase 1 output into
  the control branch.
- Keep the change limited to host-only evidence so the bad sound can be
  isolated without touching firmware, NVS, serial, provider/V21 execution, or
  Mac audio playback.
- Preserve the distinction between host objective downlink evidence and
  physical audible acceptance.

Actual completed work:

- Reviewed the worker summary and manually integrated the narrow code/test
  change onto the control branch.
- Added decoded Opus/downlink aggregate quality reporting to
  `xiaozhi-voice-bench`.
- Added regression assertions so the bench output includes
  `downlink_audio_quality`, `codec: opus_decoded_pcm_s16le`,
  `sample_rate_hz: 16000`, and a passed quality status.
- Updated `docs/project_state_machine.md` to mark `T-AUDIO-001a` complete and
  keep `T-AUDIO-001` Phase 2 as the active physical A/B path.
- Updated
  `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md` with Phase 1
  completion status.

Files changed:

- `internal/app/xiaozhi_voice_bench.go`
- `internal/app/app_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md`

Current repository state:

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Starting HEAD for this handoff entry: `49b9e458d44b`.
- Active transition remains `T-AUDIO-001`, with `T-AUDIO-001a` completed and
  physical A/B still pending.

Current conclusions:

- The audio/TTS optimization was already landed before this round.
- The current physical path is stock-profile Xiaozhi Opus uplink/downlink
  through A21 Gateway, not the old A21 PCM bridge.
- The end-to-end route is not upstream Xiaozhi cloud; A21 Gateway still owns
  ASR/text/TTS generation and Opus downlink.
- The most likely blocker remains TTS/model/voice generation unless the new
  host post-Opus metrics or a foreground physical A/B points elsewhere.

Unfinished items:

- Run a fresh `xiaozhi-voice-bench --require-product-chain` against the active
  Gateway route to generate a new report containing `downlink_audio_quality`.
- Run foreground physical A/B with the same firmware, NVS route, Gateway port,
  and stock Xiaozhi path, changing only the approved TTS candidate/profile or
  fixture source.
- Record operator/instrument audible observation or trusted device playback ack
  tied to a fresh trace.
- Regenerate `xiaozhi-physical-evidence` and product readiness after physical
  A/B.
- Clean or gate stock-incompatible `listen` ack behavior if it is confirmed to
  be product-polish or runtime-noise risk.

Known risks and blockers:

- Host PCM and post-Opus objective quality can pass while the voice still
  sounds unnatural, robotic, or unfit for product use.
- Physical speaker/decode/playback can still be the root cause even if host
  downlink metrics are clean.
- Stock firmware still logs `Unknown message type: listen`; do not conflate
  that warning with TTS quality until isolated by evidence.
- Do not treat this host-only report enhancement as PRD physical acceptance.

Validation results:

- `go test ./internal/audio ./internal/providers ./internal/gateway ./internal/app -run 'PCMQuality|LocalTTS|VoicePipeline|Xiaozhi.*Opus|Xiaozhi.*Stock|XiaozhiWebSocketListen|XiaozhiPhysicalEvidence|XiaozhiVoiceBench' -count=1`:
  passed.
- `gofmt` was run on `internal/app/xiaozhi_voice_bench.go` and
  `internal/app/app_test.go`.
- `git diff --check`: passed.

Recommended next action:

- Execute `T-AUDIO-001` Phase 2: foreground physical A/B on the current stock
  Xiaozhi route after operator approval for any Gateway/TTS profile swap.
- Use the new `downlink_audio_quality` evidence to decide whether the next fix
  belongs to TTS/model selection, Opus/downlink pacing/quality, firmware
  playback, or protocol cleanup.

## 2026-06-02 - T-AUDIO-001b - Run Host/Gateway Post-Opus Quality Evidence

Goal:

- Continue converging `T-AUDIO-001` after Phase 1 integration.
- Generate fresh host-only product-chain evidence with `downlink_audio_quality`
  so the bad physical sound can be separated from basic Gateway post-Opus
  waveform quality.
- Keep launch readiness honest and avoid firmware, NVS, serial, provider/V21,
  and Mac audio side effects.

Actual completed work:

- Confirmed control branch `codex/a21-hardware-window-20260602-stackchan-prd`
  at `2fe4947` with a clean worktree before this round.
- Verified Gateway `21081` health, device registry, and provider health:
  physical device `44:1b:f6:e2:6a:60` was online with stock Xiaozhi Opus
  ingress/downlink capabilities; provider health surface was still mock.
- Attempted `xiaozhi-voice-bench` in the sandbox and found Go WebSocket dial
  failed with `operation not permitted` even though curl could perform a raw
  WebSocket upgrade.
- Added a redacted WebSocket dial failure classifier so future reports expose
  low-information categories such as
  `gateway_websocket_unavailable_operation_not_permitted` instead of only
  `gateway_websocket_unavailable`.
- Ran the bench outside the sandbox as required for local Go WebSocket access:
  - `reports/a21-xiaozhi-voice-bench-20260602-220325.719331000.json`:
    `21080`, repeat 3, `candidate_host_only`, `host_product_chain_ready=true`,
    `sherpa_onnx_tts`, answer p95 397 ms, answer `downlink_audio_quality`
    passed.
  - `reports/a21-xiaozhi-voice-bench-20260602-220344.175240000.json`:
    `21081`, repeat 1, `candidate_host_only`, answer
    `downlink_audio_quality` passed. This is a physical-LAN-Gateway precheck,
    not a product-readiness voice evidence candidate because repeat count is
    one.
- Ran product readiness with the real physical device id:
  `reports/a21-product-readiness-20260602-220447.json`.
- Ran server-side readiness bundle:
  `reports/a21-server-side-readiness-bundle-20260602-220501.json`.
- Updated `docs/project_state_machine.md` and
  `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md`.

Files changed:

- `internal/app/xiaozhi_voice_bench.go`
- `internal/app/app_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-a21-xiaozhi-tts-audio-quality-rca.md`

Current repository state:

- Control branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Starting HEAD for this handoff entry: `2fe4947`.
- Active transition remains `T-AUDIO-001`.
- Completed host-only sub-transition: `T-AUDIO-001b`.
- Current module state:
  `S1B-HOST-POST-OPUS-PASS-PHYSICAL-AUDIBLE-PENDING`.

Current conclusions:

- The active A21 host product chain can produce valid post-Opus downlink
  quality with `sherpa_onnx_tts`; the basic Gateway post-Opus waveform path is
  not the leading suspect after this evidence.
- The remaining bad-sound root cause is more likely one of:
  physical firmware decode/playback/speaker path, physical volume/gain, user
  perceived TTS voice/model quality, or protocol/control noise around the stock
  client.
- This still does not prove physical audible quality. The host reports remain
  `candidate_host_only` and `prd_accepted=false`.
- Readiness remains correctly blocked: real provider smoke is missing, physical
  StackChan PRD acceptance is missing, and custom wake word product proof is
  missing.

Unfinished items:

- Run foreground physical audible A/B on the same stock Xiaozhi route.
- Collect operator/instrument audible observation or trusted playback ack
  tied to a fresh physical trace.
- Capture device playback start/downlink first-frame timing and barge-in
  playback stop_done if available.
- Regenerate `xiaozhi-physical-evidence`, `product-readiness`, and
  `server-side-readiness-bundle` after physical A/B.
- Clean/gate stock-incompatible `listen` ack warnings if they remain visible in
  a fresh physical turn.
- Close real provider smoke and custom wake proof in separate transitions.

Known risks and blockers:

- Host post-Opus quality metrics do not judge subjective voice naturalness or
  physical speaker quality.
- `product-readiness` skips the latest one-round `21081` voice bench as a
  contract candidate, which is expected; the repeat-3 `21080` report is the
  current host voice evidence source.
- Running Go WebSocket bench inside the sandbox can fail with
  `operation not permitted`; use sandbox escalation for localhost Gateway bench
  runs.
- Do not claim PRD acceptance from host-only evidence.

Validation results:

- Sandbox `xiaozhi-voice-bench` attempts failed with Go WebSocket
  `operation not permitted`; this was recorded as an execution-environment
  failure, not a Gateway failure.
- `curl` raw WebSocket upgrade to `127.0.0.1:21081/v1/xiaozhi`: returned
  `101 Switching Protocols`.
- Sandbox-external `xiaozhi-voice-bench --gateway-url http://127.0.0.1:21080 --repeat 3 --require-product-chain --output-dir reports`:
  passed.
- Sandbox-external `xiaozhi-voice-bench --gateway-url http://127.0.0.1:21081 --repeat 1 --require-product-chain --output-dir reports`:
  passed.
- Sandbox-external `product-readiness --gateway-url http://127.0.0.1:21081 --device-id 44:1b:f6:e2:6a:60 --use-latest-reports --output-dir reports`:
  passed with `launch_ready=false`.
- Sandbox-external `server-side-readiness-bundle --gateway-url http://127.0.0.1:21081 --device-id 44:1b:f6:e2:6a:60 --use-latest-reports --output-dir reports`:
  passed with `server_side_blocked`.
- `go test ./internal/app -run 'XiaozhiVoiceBench' -count=1`: passed.
- `go test ./internal/audio ./internal/providers ./internal/gateway ./internal/app -run 'PCMQuality|LocalTTS|VoicePipeline|Xiaozhi.*Opus|Xiaozhi.*Stock|XiaozhiWebSocketListen|XiaozhiPhysicalEvidence|XiaozhiVoiceBench' -count=1`:
  passed.
- Sandbox `make verify` failed because `httptest` could not bind `[::1]:0`
  (`operation not permitted`).
- Sandbox-external `make verify`: passed.

Recommended next action:

- Execute `T-AUDIO-001` Phase 2 foreground physical audible A/B.
- Keep firmware, NVS route, Gateway port, and stock Xiaozhi path stable.
- Change only the approved TTS candidate/profile or fixture source, and require
  operator/instrument audible observation plus fresh trace linkage before
  updating physical acceptance.

## 2026-06-02 22:16 CST - T-AUDIO-001c Foreground Long-TTS Playback Attempt

Round goal:

- Respond to the operator request to set volume to maximum and play a long TTS
  passage for a fresh phone recording.
- Keep the work inside `T-AUDIO-001` Phase 2 and avoid firmware, NVS, provider,
  V21, or business-code changes.

Actual completed work:

- Confirmed physical device `44:1b:f6:e2:6a:60` was online on Gateway
  `127.0.0.1:21081` with stock Xiaozhi Opus ingress/downlink capabilities.
- Set macOS output volume to 100 after explicit operator approval.
- Attempted to deliver a long Chinese diagnostic passage through
  `stackchan-local-tts-playback` with `--engine sherpa_onnx` against the
  physical Gateway/device.
- The command failed before physical playback with:
  `gateway device control returned status 409: device audio websocket is not connected`.
- Inspected Gateway routing and confirmed the failure is expected for the
  current physical session: the old `/v1/devices/control` PCM playback surface
  requires the legacy A21 audio WebSocket, while the current hardware is online
  through stock `/v1/xiaozhi` WebSocket.
- Confirmed the stock Xiaozhi physical path sends TTS only inside a
  device-driven listen/audio turn. Existing HTTP control for Xiaozhi supports
  debug state/face/display/motion events only, not arbitrary stock TTS audio
  injection.
- Updated `docs/project_state_machine.md` to preserve this boundary and prevent
  future false evidence from the wrong playback surface.

Files changed:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Current repository state:

- Branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Active transition: `T-AUDIO-001`.
- Current TTS/audio state:
  `S1C-STOCK-XIAOZHI-OPERATOR-RECORDING-PENDING`.
- No business code, firmware, NVS, provider, or V21 surfaces were modified.

Current unfinished items:

- Collect a fresh physical audible recording from a real stock Xiaozhi
  listen/audio turn.
- Tie the recording to the latest trace/session and regenerate
  `xiaozhi-physical-evidence`, `product-readiness`, and readiness bundle.
- If the team wants host-driven long TTS on the physical stock Xiaozhi session,
  create a separate plan/worker transition for a stock-safe injection seam
  before implementation.

Known risks and blockers:

- The long TTS passage was not played through the physical StackChan speaker in
  this round; do not treat this attempt as audible playback evidence.
- Running `xiaozhi-voice-bench` or any virtual client with the physical device
  id would only prove host/bench downlink and could mask the real physical
  socket, so it must not be used as the operator recording path.
- macOS volume being set to 100 does not necessarily affect physical StackChan
  speaker loudness on stock Xiaozhi Opus downlink.

Validation results:

- `curl http://127.0.0.1:21081/v1/devices`: physical device online,
  `xiaozhi_profile=stock`, `xiaozhi_transport=websocket`,
  `xiaozhi_audio=opus_16000hz_mono_60ms`.
- `osascript -e 'set volume output volume 100'`: passed after explicit
  operator approval.
- `go run ./cmd/a21 stackchan-local-tts-playback --gateway-url http://127.0.0.1:21081 --device-id 44:1b:f6:e2:6a:60 --engine sherpa_onnx ...`:
  failed with Gateway `409 device audio websocket is not connected`.
- No build/test suite was rerun because this round only updated governance
  docs after a foreground runtime attempt.

Recommended next action:

- For immediate recording, trigger a real stock Xiaozhi turn on the device and
  record the physical response from 20-30 cm in front of the speaker.
- Use a prompt that encourages a long spoken response, then upload the new
  phone recording for analysis.
- If a deterministic host-pushed long TTS is required, first open
  `T-AUDIO-003: Stock-safe physical TTS injection plan` rather than reusing
  the legacy PCM control path.

## 2026-06-02 22:31 CST - T-AUDIO-001d Recording Analysis And StackChan Volume Boundary

Round goal:

- Analyze the operator recording
  `/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 2.m4a`.
- Correct the previous mistaken macOS-volume interpretation and build a
  desktop helper focused on StackChan/Gateway/device control truth.
- Do not modify firmware, write NVS, flash hardware, or claim volume/action
  control where the current stock session rejects it.

Actual completed work:

- Analyzed the new 13.03 s AAC recording from 2026-06-02 22:19:34 CST.
- Built `tools/desktop/a21-stackchan-control.command` and copied it to
  `/Users/jiyurun/Desktop/A21-StackChan-Control.command`.
- Added `docs/plans/2026-06-02-stackchan-volume-action-control.md` because
  true StackChan speaker-volume control is a firmware/protocol/device-behavior
  transition.
- Updated `docs/project_state_machine.md` with the current volume/action
  control state and next transition.
- Removed the earlier untracked macOS-volume helper before completion so the
  repository keeps only the StackChan-focused tool.

Files changed:

- `tools/desktop/a21-stackchan-control.command`
- `docs/plans/2026-06-02-stackchan-volume-action-control.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current audio findings:

- Container/codec: M4A AAC-LC, 48 kHz, stereo, 13.034667 s, about 132 kb/s.
- `volumedetect`: mean volume `-32.3 dB`, max volume `-4.3 dB`.
- EBU R128: integrated loudness `-25.8 LUFS`, true peak `-4.2 dBFS`,
  loudness range `7.8 LU`.
- `astats`: overall peak `-4.278839 dB`, RMS `-32.344824 dB`, noise floor
  about `-47.029328 dB`; no clipping/NaN/Inf evidence.
- Frame analysis: only about `2.16%` of 20 ms frames exceeded `-25 dBFS` RMS,
  `7.24%` exceeded `-30 dBFS`, and `17.26%` exceeded `-35 dBFS`; p50 20 ms
  RMS was `-48.41 dBFS`.
- Active-frame spectrum above `-35 dBFS` was concentrated around speech
  presence bands: about `77.7%` energy in `800-2000 Hz`, `21.8%` in
  `2000-4000 Hz`, almost no low band and almost no `>4 kHz`.

Current conclusion:

- This recording is much stronger than the prior phone sample: peak level is
  close to full scale and therefore the phone recording path is not simply
  "too quiet".
- Average loudness and active-frame ratio are still low, and the spectrum is
  narrow. This supports the operator observation that physical output may not
  be at desired loudness, but it does not by itself prove TTS generation,
  Opus downlink, or speaker gain as the sole cause.
- The next meaningful fix is StackChan-side output gain/volume control or a
  controlled A/B volume trial, not macOS system volume.

StackChan control findings:

- `tools/desktop/a21-stackchan-control.command status` passed when run outside
  the Codex sandbox against Gateway `21081`.
- The physical device was online as stock Xiaozhi:
  `speaker=available_xiaozhi_opus_downlink`,
  `xiaozhi_audio=opus_16000hz_mono_60ms`.
- Runtime StackChan speaker-volume setter is not exposed on current stock
  `/v1/xiaozhi` Gateway path.
- `face happy` and `motion nod` through `/v1/xiaozhi/control` both returned
  HTTP 409: `xiaozhi device events require debug profile negotiation`.
- `diagnostic-tone 255` through `/v1/devices/control` returned HTTP 409:
  `device audio websocket is not connected`.
- Known code-level volume knobs are firmware-side, such as official codec
  `SetOutputVolume(...)` overlays or old A21 `M5.Speaker.setVolume(96)`;
  those require a planned firmware/protocol transition before use.

Validation results:

- `zsh -n tools/desktop/a21-stackchan-control.command`: passed.
- `tools/desktop/a21-stackchan-control.command status` outside sandbox:
  passed and printed current volume/action boundaries.
- `tools/desktop/a21-stackchan-control.command face happy` outside sandbox:
  failed honestly with HTTP 409 debug-profile negotiation block.
- `tools/desktop/a21-stackchan-control.command motion nod` outside sandbox:
  failed honestly with HTTP 409 debug-profile negotiation block.
- `tools/desktop/a21-stackchan-control.command diagnostic-tone 255` outside
  sandbox: failed honestly with HTTP 409 audio WebSocket block.
- `/Users/jiyurun/Desktop/A21-StackChan-Control.command` exists and is
  executable.

Current unfinished items:

- Decide and execute `T-HW-VOLUME-001`: fixed official codec output volume
  patch vs stock-safe runtime volume control seam.
- If fixed firmware volume is chosen, patch the official Xiaozhi-compatible
  overlay, rebuild, flash only through guarded foreground hardware commands,
  and record before/after phone samples.
- If runtime action/volume control is desired, add a debug-profile negotiation
  or MCP/device-tool path deliberately; do not rely on the current stock socket.

Known risks and blockers:

- Current desktop helper cannot set StackChan TTS loudness because no runtime
  setter exists on the active path.
- Raising firmware output volume can introduce clipping, resonance, or worse
  perceived TTS quality; it needs A/B recordings.
- Device registry capability strings can show feature hints while the live
  socket still rejects host-pushed events.
- Diagnostic tone volume is not product TTS volume.

Recommended next action:

- Start `T-HW-VOLUME-001` with the fixed firmware codec volume path unless a
  live official runtime volume setter is verified first.
- Use the new plan:
  `docs/plans/2026-06-02-stackchan-volume-action-control.md`.

## 2026-06-02 22:36 CST - T-HW-VOLUME-001 Worker Dispatch And State Control

Round goal:

- Continue under the control-tower workflow after the user asked to keep
  pushing quickly with branch/thread tools.
- Turn the StackChan device-volume problem into an active, scoped transition.
- Keep the main thread as architecture/control only and move implementation
  into a worker worktree.

Actual completed work:

- Re-read the active state machine, handoff log, and StackChan volume/action
  plan.
- Confirmed the main checkout was clean at `f49abdee54bc` on branch
  `codex/a21-hardware-window-20260602-stackchan-prd` before this docs update.
- Created and pinned worker thread
  `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9`, titled
  `A21 T-HW-VOLUME-001 fixed codec volume worker`.
- Worker worktree:
  `/Users/jiyurun/.codex/worktrees/bb16/New project`.
- Worker assignment: implement only the fixed official codec output-volume path
  for the official Xiaozhi-compatible firmware overlay, with minimal guard/test
  and handoff/state docs.
- Updated `docs/project_state_machine.md` so the active child transition is
  now `T-HW-VOLUME-001` rather than the already host-isolated `T-AUDIO-001`.
- Recorded explicit transition current state, target state, trigger, actions,
  acceptance, failure states, rollback path, and next state.

Files changed:

- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current unfinished items:

- Wait for worker `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9` to return one of:
  `DONE`, `DONE_WITH_CONCERNS`, or `BLOCKED`.
- Review the worker diff before integration. Expected write set is limited to
  the official Xiaozhi-compatible overlay, focused guard/test files, and state
  docs.
- If the worker produces a fixed-volume candidate, run no-write build/report
  checks before asking for any foreground flash.
- Physical loudness remains unaccepted until the operator approves a hardware
  window, the candidate is flashed through guarded commands, and before/after
  recordings are analyzed.

Known risks and blockers:

- Current stock `/v1/xiaozhi` session still has no runtime StackChan
  speaker-volume setter.
- `/v1/xiaozhi/control` action probes still require debug profile negotiation
  and must not be treated as product stock control.
- Firmware output gain may make loudness better while worsening clipping,
  resonance, or TTS intelligibility; it needs A/B evidence.
- No PRD physical acceptance may be claimed from this worker alone.

Recommended next action:

- Read the worker result from thread
  `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9`.
- If `DONE`, inspect diff, run the focused tests it reports, merge/cherry-pick
  only if boundaries held, then run the official-compatible build/report path.
- If build/report passes, open a foreground hardware window for guarded flash
  and phone-recorded before/after loudness comparison.

Validation results:

- `git status --short --branch`: clean before this docs update.
- `git log --oneline -5`: latest commit before this round was
  `f49abde docs(audio): plan stackchan volume control`.
- `git diff --check`: passed after this handoff entry was written.

## 2026-06-02 22:47 CST - T-HW-VOLUME-001a Main Integration And No-Write Build

Round goal:

- Continue under the control-tower workflow after the user asked to push quickly
  with branch/thread tools.
- Review the worker result for StackChan device loudness, integrate only the
  fixed official codec volume candidate, and advance toward hardware A/B
  without touching flash/NVS/provider/V21/Mac audio.

Actual completed work:

- Read worker thread/worktree result from
  `/Users/jiyurun/.codex/worktrees/bb16/New project` on branch
  `codex/t-hw-volume-001-official-codec-volume`.
- Integrated the worker's code-only patch into the main branch:
  the official Xiaozhi-compatible overlay now includes official
  `audio_codec`/`board` headers and calls
  `Board::GetInstance().GetAudioCodec()->SetOutputVolume(92)` immediately
  before `GetHAL().startXiaozhi()`.
- Added a focused Go guard test requiring the official codec volume setter,
  preserved `GetHAL().startXiaozhi()`, and volume setting before Xiaozhi
  runtime entry.
- Updated `docs/project_state_machine.md` from worker-dispatched to
  no-write-build-passed state.
- Ran the official Xiaozhi-compatible no-write build/report. Initial sandboxed
  attempts failed in `fetch_repos.py` due to stale local Git/proxy settings
  (`127.0.0.1:7897`) and then DNS/network sandboxing. The approved escalated
  no-flash build passed after clearing proxy env/config for that command only.

Files changed:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current repository state:

- Main branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Worker branch remains dirty with the same intended four-file patch; main
  branch manually integrated the code and state instead of blindly applying the
  worker's stale-baseline docs diff.
- Build reports are under ignored `reports/`; record paths below because they
  are evidence but not committed.

Current unfinished items:

- Commit this integration after final `git diff --check`/status review.
- Open a foreground hardware window for guarded official-compatible flash only
  if the operator approves hardware execution.
- After flash, collect before/after phone or instrument recordings and
  regenerate physical/readiness evidence.
- Runtime StackChan volume control is still not exposed on the current stock
  `/v1/xiaozhi` path; any `self.audio_speaker.set_volume`-style runtime path is
  a separate planned stock-safe protocol transition.

Known risks and blockers:

- `SetOutputVolume(92)` may improve loudness but can introduce clipping,
  enclosure resonance, or worse perceived TTS. Physical A/B is required before
  product acceptance.
- The current stock session still rejects host-pushed action/device events
  without debug-profile negotiation and rejects legacy diagnostic tone because
  the old PCM audio WebSocket is not connected.
- Source checkout
  `/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan` is dirty, but
  the build tool exported from git `HEAD` only and reported
  `source_worktree_dirty` honestly.
- No PRD physical acceptance may be claimed from code/build evidence alone.

Validation results:

- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlaySetsCodecVolumeBeforeRuntime|TestRunStackChanOfficialXiaozhiCompatiblePlanReportsProductCandidateContract|TestApplyStackChanOfficialCandidateContractKeepsXiaozhiCompatibleAfterExecute' -count=1`:
  passed.
- `go test ./internal/app -run 'StackChanOfficial|Official|Firmware|Xiaozhi' -count=1`:
  passed.
- Clean official `HEAD` patch apply check against
  `/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan`: passed.
- `git diff --check`: passed before the no-write build.
- `make a21-stackchan-official-xiaozhi-compatible-build`:
  passed with no flash/NVS/serial writes after approved network escalation and
  command-local proxy clearing.
- Passing build report:
  `reports/a21-stackchan-official-baseline-20260602-225207-1780411927040145000.json`.
- Built app SHA-256:
  `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.

Recommended next action:

- Commit this integration as the code/build candidate.
- Next transition: `T-HW-VOLUME-001b` foreground StackChan volume A/B
  acceptance. Use guarded flash only in an operator-approved hardware window,
  then record before/after audio and rerun physical/readiness evidence.

## 2026-06-02 22:59 CST - T-HW-VOLUME-001b Flash Gate And A/B Worker Dispatch

Round goal:

- Continue under the control-tower workflow after the user asked to keep
  pushing with branch/thread tools.
- Advance the fixed StackChan codec-volume candidate from no-write build passed
  to foreground flash-window readiness.
- Use worker threads for bounded review/runbook prep while keeping the main
  thread in control and avoiding silent hardware writes.

Actual completed work:

- Recovered current state from `AGENTS.md`, `docs/project_state_machine.md`,
  `docs/agent_handoff_log.md`, and the StackChan volume/action plan.
- Confirmed main branch `codex/a21-hardware-window-20260602-stackchan-prd` was
  clean at `f0603f5e0f7f1744c73042c2e335b1719bf2c8b7`.
- Verified USB serial candidate `/dev/cu.usbmodem1101` and current official
  build artifacts under `/tmp/a21-stackchan-official-build`.
- Ran only the no-write flash plan:
  `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 make a21-stackchan-official-xiaozhi-compatible-flash-plan`.
- Created and pinned read-only worker
  `019e88d6-1734-75d2-897b-aa2da3069885`
  (`A21 T-HW-VOLUME-001b flash gate review`) in worktree
  `/Users/jiyurun/.codex/worktrees/37f0/New project`.
- Created and pinned read-only worker
  `019e88d6-ad61-7513-b379-aa21ae7db150`
  (`A21 T-HW-VOLUME-001b A/B evidence runbook`) in worktree
  `/Users/jiyurun/.codex/worktrees/4f7d/New project`.
- Integrated the completed flash-gate worker conclusion into
  `docs/project_state_machine.md`: the gate is ready to request an
  operator-approved foreground flash window, but not physical acceptance.
- Main-thread runbook inventory confirmed the post-flash evidence path should
  use real stock Xiaozhi physical turns, phone/instrument recordings,
  `xiaozhi-instrument-observation`, `xiaozhi-physical-evidence`, and
  `product-readiness --use-latest-reports`; the legacy
  `stackchan-local-tts-playback` path remains invalid for current stock
  Xiaozhi physical loudness because it previously returned HTTP 409 audio
  WebSocket disconnected.

Files changed:

- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current repository state:

- Main branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Reports remain under ignored `reports/`; paths are recorded as evidence but
  not committed.
- Worker `019e88d6-1734-75d2-897b-aa2da3069885` returned `STATUS=DONE`.
- Worker `019e88d6-ad61-7513-b379-aa21ae7db150` returned `STATUS=DONE`.
  It stayed read-only, found that ignored `reports/` evidence lives in the main
  checkout rather than its detached worktree, and produced the foreground A/B
  runbook.

Flash gate evidence:

- No-write build report:
  `reports/a21-stackchan-official-baseline-20260602-225207-1780411927040145000.json`.
- Built app SHA-256:
  `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.
- No-write flash-plan report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-225600-1780412160509265000.json`.
- Flash-plan fields: `status=ready`, `port=/dev/cu.usbmodem1101`,
  `dry_run=true`, `flash_allowed=false`, `flash_executed=false`, app offset
  `0x20000`, app SHA-256
  `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.
- Rollback baseline:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`,
  app SHA-256
  `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`.
- Required execute confirmation token:
  `WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP`.

Current unfinished items:

- Ask the operator to explicitly open the foreground hardware flash window
  before running the execute command.
- After flash, trigger real stock Xiaozhi physical turns on the device and
  capture before/after phone or instrument recordings.
- Convert the audible observation into a redacted sidecar with
  `xiaozhi-instrument-observation`, regenerate `xiaozhi-physical-evidence`, and
  refresh readiness with `product-readiness --use-latest-reports`.

Known risks and blockers:

- No physical loudness result exists yet for the `SetOutputVolume(92)` build.
- The flash plan is ready, but `flash_allowed=false` by design until the
  foreground execute command is deliberately run with the confirmation token.
- Raising codec output may introduce clipping, enclosure resonance, or worse
  TTS intelligibility; before/after recordings are required before acceptance.
- Current stock `/v1/xiaozhi` still has no Gateway runtime volume setter.
- `stackchan-local-tts-playback` and diagnostic tone remain non-product paths
  for this stock physical session.
- A/B worker confirmed invalid evidence includes Mac volume changes,
  `stackchan-local-tts-playback`, `/v1/devices/control` diagnostic tone, old
  PCM/audio WebSocket proof, host-only `xiaozhi-voice-bench`, decoded Opus
  quality, dry-run reports, and debug playback ack unless explicitly negotiated
  in a debug profile.

Recommended next action:

- If the operator approves hardware execution, run exactly:
  `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP make a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- Then trigger a real stock Xiaozhi turn on StackChan, record before/after
  phone or instrument samples with the same phone position and prompt class,
  analyze LUFS/peak/RMS/active-frame ratio/clipping, create an instrument
  observation report, rerun physical evidence, and refresh product readiness.

Validation results:

- `git status --short --branch`: clean before this docs update.
- `ls -1 /dev/cu.usb*`: found `/dev/cu.usbmodem1101`.
- `ls -l /tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin /tmp/a21-stackchan-official-build/flash_args`:
  artifacts present.
- `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 make a21-stackchan-official-xiaozhi-compatible-flash-plan`:
  passed; no flash/NVS/serial write executed.
- `go run ./cmd/a21 xiaozhi-physical-evidence --help`: printed expected
  physical-evidence command shape.
- `go run ./cmd/a21 xiaozhi-instrument-observation --help`: printed expected
  instrument-observation command shape.
- `go run ./cmd/a21 product-readiness --help`: printed expected latest-report
  readiness refresh flags.

## 2026-06-02 23:24 CST - T-HW-VOLUME-001c Foreground Flash Executed And State Converged

Round goal:

- Continue the control-tower flow after the operator explicitly requested the
  guarded StackChan flash execute command.
- Move the fixed official codec volume candidate from flash-plan-ready to
  flashed-but-not-audibly-accepted.
- Keep other work moving through scoped worker threads without expanding the
  main thread into broad code changes.

Actual completed work:

- Executed the operator-approved foreground hardware command:
  `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP make a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- Flash execute passed and wrote report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`.
- Verified the flash report fields: `status=passed`, `port=/dev/cu.usbmodem1101`,
  `dry_run=false`, `flash_allowed=true`, `flash_executed=true`, app SHA-256
  `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.
- Checked post-flash Gateway health on `127.0.0.1:21081`; `/healthz` returned
  `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`.
- Checked the Gateway device registry; physical device
  `44:1b:f6:e2:6a:60` remained registered `online` on stock Xiaozhi WebSocket
  with speaker `available_xiaozhi_opus_downlink`.
- Pinned and collected worker `019e88dd-1517-77c2-a002-7db771ec016a`
  (`T-PROTOCOL-001`). It found the strongest `Unknown message type: listen`
  hypothesis: Gateway sends server-to-device `type=listen` ack frames that
  stock firmware does not accept, while binary Opus/TTS downlink still reaches
  `speaking`.
- Added mainline plan
  `docs/plans/2026-06-02-stock-xiaozhi-protocol-cleanup.md` from the protocol
  worker result.
- Pinned and collected read-only worker
  `019e88dd-3efa-78e2-a7d3-7063089cbf30` (`T-PROVIDER-001`). It found the
  state machine was stale: usable explicit DeepSeek and local Ollama provider
  evidence exists, but newer generic readiness reports fell back to `mock`
  because they omitted explicit provider report/env.
- Updated `docs/project_state_machine.md` to record the foreground flash,
  active physical A/B blocker, protocol cleanup plan, and provider evidence
  truth.

Files changed:

- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-stock-xiaozhi-protocol-cleanup.md`
- `docs/agent_handoff_log.md`

Current repository state:

- Branch: `codex/a21-hardware-window-20260602-stackchan-prd`.
- HEAD before this docs update: `18dc5539455e`.
- Active transition: `T-HW-VOLUME-001`.
- Current state: `S3-FOREGROUND-FLASHED-A-B-PENDING`.
- Target state: `S4-PHYSICAL-LOUDNESS-A-B-RECORDED`.

Current unfinished items:

- Trigger a real stock Xiaozhi turn on the physical StackChan and collect the
  post-flash phone recording with the same phone position and prompt class.
- Compare post-flash loudness, peak, RMS/LUFS, clipping, active speech ratio,
  and intelligibility against the previous operator recording.
- Convert the result into an operator/instrument observation sidecar, rerun
  `xiaozhi-physical-evidence`, and refresh product readiness.
- Dispatch the scoped implementation worker for
  `docs/plans/2026-06-02-stock-xiaozhi-protocol-cleanup.md`.
- Refresh readiness with the selected provider report/env pinned if the control
  tower needs a fresh readiness bundle; do not run generic readiness in a way
  that silently selects `mock`.

Known risks and blockers:

- Flash success is not physical loudness acceptance.
- `SetOutputVolume(92)` can still clip, distort, resonate, or fail to fix the
  user's TTS complaint; only a post-flash physical recording can decide.
- Current stock `/v1/xiaozhi` still has no Gateway runtime volume setter.
- `stackchan-local-tts-playback`, macOS volume, diagnostic tone, host-only
  `xiaozhi-voice-bench`, and dry-run reports remain invalid as physical
  StackChan loudness acceptance.
- The `listen` warning likely needs Gateway protocol cleanup, but it is
  separate from the physical TTS/loudness verdict.
- Provider evidence exists, but a careless generic readiness refresh can make
  the status look red again by selecting `mock`.

Recommended next action:

- Operator: record one post-flash real stock Xiaozhi long-TTS sample from the
  same phone position, then provide the file for analysis.
- Control tower: after the recording arrives, run objective audio analysis and
  update `T-HW-VOLUME-001d`.
- Worker lane: dispatch `T-PROTOCOL-001 implementation` from
  `docs/plans/2026-06-02-stock-xiaozhi-protocol-cleanup.md`.
- Host readiness lane: refresh product readiness with the explicit selected
  provider report/env pinned, not with generic mock fallback.

Validation results:

- Foreground flash execute command: passed.
- Flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`.
- `curl --noproxy '*' -sS --max-time 3 http://127.0.0.1:21081/healthz`:
  returned Gateway ok.
- `curl --noproxy '*' -sS --max-time 3 http://127.0.0.1:21081/v1/devices`:
  returned physical device `44:1b:f6:e2:6a:60` as `online`.
- Protocol worker `019e88dd-1517-77c2-a002-7db771ec016a`: `STATUS=DONE`,
  docs-only plan.
- Provider worker `019e88dd-3efa-78e2-a7d3-7063089cbf30`: `STATUS=DONE`,
  read-only.
- No business code changed in this main-thread convergence step.

## 2026-06-02 23:34 CST - T-AUDIO-002 Stock Xiaozhi Audio Parity Hotfix Integrated

Round goal:

- Stop treating the bad physical sound as only a volume issue.
- Read official Xiaozhi/StackChan behavior in parallel worker threads and
  integrate the smallest hotfix that moves the physical main flow forward.
- Give the operator a real StackChan volume control path on the current stock
  `/v1/xiaozhi` session.

Actual completed work:

- Spawned three read-only subagents:
  - `019e88eb-7665-7f82-86d4-6e7e996e9839`: official `xiaozhi-esp32`
    audio/protocol chain. It confirmed client/uplink `16000 Hz`, CoreS3/output
    `24000 Hz`, official codec/NVS volume behavior, and official decode/playback
    queue paths.
  - `019e88eb-7e31-7930-84ea-834e3d91849b`: StackChan codec/I2S/volume control.
    It confirmed official MCP tool `self.audio_speaker.set_volume` is the
    fastest runtime volume path for the stock Xiaozhi session.
  - `019e88eb-837b-78a1-a1b0-ce51b023d179`: A21 versus official protocol/audio
    comparison. It identified the minimal urgent set: 16k/24k downlink split,
    unsupported `listen` ack, per-frame Opus encoder rebuild, and prebuffer
    burst risk.
- Added plan
  `docs/plans/2026-06-02-stackchan-audio-official-parity-hotfix.md`.
- Restored stock-compatible server/downlink audio to `24000 Hz` while keeping
  stock client/uplink at `16000 Hz`.
- Changed local TTS adapter output/downlink chunking to `24000 Hz` mono
  `60 ms`.
- Suppressed server-to-device `listen` replies for stock physical MAC-address
  devices while preserving debug/virtual test behavior.
- Reused a turn-level downlink Opus encoder for contiguous same-format TTS
  frames.
- Reduced downlink prebuffer from five frames to one frame.
- Added `POST /v1/xiaozhi/speaker-volume`, which sends a stock MCP
  `tools/call` for `self.audio_speaker.set_volume` to an online
  `/v1/xiaozhi` WebSocket when `hello.features.mcp=true`.
- Updated repo and Desktop StackChan control helpers so `volume 100` calls the
  new runtime volume endpoint.
- Updated `docs/engineering/PROTOCOL.md` and `docs/project_state_machine.md`
  to reflect the new active `T-AUDIO-002` transition.

Files changed:

- `docs/agent_handoff_log.md`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-02-stackchan-audio-official-parity-hotfix.md`
- `docs/project_state_machine.md`
- `internal/app/app_test.go`
- `internal/app/xiaozhi_voice_bench.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `tools/desktop/a21-stackchan-control.command`
- `/Users/jiyurun/Desktop/A21-StackChan-Control.command`

Current unfinished items:

- The running Gateway on `21081`, if still active from before this hotfix, must
  be restarted or relaunched from this working tree before the new endpoint and
  audio behavior are live.
- After Gateway restart, send StackChan volume `100` through
  `/v1/xiaozhi/speaker-volume` or the Desktop helper.
- Trigger a long real stock Xiaozhi TTS turn and collect a new physical
  recording from the same phone position.
- Analyze LUFS/peak/RMS/active ratio and compare against
  `/Users/jiyurun/Downloads/军民公路259号 7.m4a` and the 22:19 reference.
- If physical audio is still blurred/intermittent, continue with TTS model/profile
  and device playback instrumentation rather than more blind loudness changes.

Known risks and blockers:

- MCP volume endpoint proves delivery to the live WebSocket, not device-side
  application, until a device response or physical recording confirms it.
- The current Go Opus wrapper still uses a 48 kHz frame-size interface
  internally; this round reduces churn and aligns the downlink target but does
  not replace the Opus library.
- Stock action controls for face/motion/display still require debug
  `device_events` negotiation; this round only fixes audio/volume path.
- Full PRD acceptance remains blocked on physical audible evidence and custom
  wake proof.

Recommended next action:

- Restart/relaunch the Gateway from this tree on the physical port, verify
  `/healthz`, verify device registry, send `volume 100`, then run a long TTS
  turn for a fresh recording.
- If the live device does not advertise MCP or the endpoint returns 409, fall
  back to the guarded firmware volume-100 candidate path without claiming
  runtime volume success.

Validation results:

- `go test ./internal/gateway ./internal/providers ./internal/transport/xiaozhi ./internal/app -run 'TestXiaozhiSpeakerVolumeUsesStockMCPToolCall|TestXiaozhiListenReplySuppressedForStockPhysicalMACDevice|TestWriteXiaozhiOpusDownlinkReusesTurnEncoderForContiguousFrames|TestWriteXiaozhiOpusDownlinkUsesPacerAndCurrentTurn|TestXiaozhiOpusDownlinkFansOutOfficialStackChanSpeaking|TestBuildServerHello|TestVoicePipelineRunnerProducesDownlinkReadyMockChunksAndRedactedReport|TestLocalTTSAdapterConvertsWAVToDownlinkReady24KChunks|XiaozhiVoiceBench|TestRunXiaozhiVoiceBenchReportsHostOnlyCandidateEvidence' -count=1`:
  passed.
- `go test ./internal/app ./internal/gateway ./internal/providers ./internal/audio/opuscodec ./internal/transport/xiaozhi -run 'Xiaozhi|StackChan|VoicePipeline|Opus|LocalTTS|Playback|Barge|Abort|ProviderLatencyFixture|FastCompanion' -count=1`:
  passed.
- `git diff --check`: passed before documentation/helper updates.
- `zsh -n tools/desktop/a21-stackchan-control.command`: passed.
- `zsh -n /Users/jiyurun/Desktop/A21-StackChan-Control.command`: passed.

## 2026-06-02 23:51 CST - T-AUDIO-002 Live Say And Wake-Word Reality Check

Round goal:

- Respond to the operator's new recording
  `/Users/jiyurun/Downloads/军民公路259号 8.m4a` from 3 seconds onward.
- Stop relying on voice prompts or legacy controls for speaker tests by adding a
  direct host-to-physical StackChan TTS path on the live stock `/v1/xiaozhi`
  socket.
- Verify whether custom wake-word firmware is actually active on the current
  physical device.

Actual completed work:

- Analyzed `8.m4a` from 3 seconds onward with FFmpeg loudness/stats:
  - `8.m4a`: `input_i=-31.78 LUFS`, `input_tp=-10.56 dBTP`,
    `input_lra=18.50`, `input_thresh=-43.70`.
  - Same 3-second window comparison for `7.m4a`: `input_i=-35.56 LUFS`,
    `input_tp=-14.65 dBTP`, `input_lra=5.40`, `input_thresh=-46.97`.
  - Interpretation: the hotfix improved sustained loudness by about `3.8 LUFS`
    and peak by about `4.1 dB`, matching the operator's "current noise is much
    better" observation, but there is still about `10 dB` of peak headroom, so
    physical loudness is not accepted as "max".
- Added `POST /v1/xiaozhi/say`, which sends a foreground host text prompt over
  the already connected stock Xiaozhi WebSocket as TTS lifecycle plus binary
  Opus downlink.
- Stored the live `xiaozhiSession` pointer with each registered stock socket so
  `/v1/xiaozhi/say` uses the same write lock, turn cancellation, downlink codec,
  and trace path as normal voice turns.
- Reduced `startXiaozhiTurn` prebuffer from five 60 ms frames to one 60 ms
  frame; the old five-frame setting could create a 300 ms startup burst that
  sounded like blur/stutter on the physical speaker.
- Added `TestXiaozhiSayDeliversTextAsStockTTSDownlink` to prove that `/say`
  writes `tts/start`, `tts/sentence_start`, binary Opus, and `tts/stop` to the
  same stock socket.
- Restarted the Gateway from this working tree on `0.0.0.0:21081`; health
  returned ok.
- Waited for physical device `44:1b:f6:e2:6a:60` to reconnect, then delivered
  stock MCP volume `100`:
  `trace_id=a21-trace-live-volume-100-hotfix`,
  `status=delivered`, `delivered_transport=xiaozhi_mcp`,
  `tool_name=self.audio_speaker.set_volume`.
- Sent a long physical TTS through `/v1/xiaozhi/say`:
  `trace_id=a21-trace-live-long-tts-hotfix`, `text_chars=189`,
  `audio_chunks=676`, `status=delivered`, `delivered_transport=xiaozhi_ws`.
- Verified current device registry still shows physical device online with
  `speaker_volume=100`, `xiaozhi_feature_mcp=true`, and
  `last_event=xiaozhi.tts.opus_frame.downlink`.
- Confirmed custom wake word is not active in the currently flashed physical
  app:
  `reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json`
  is `status=packaged`, `package_written=true`, but `product_ready=false`,
  `flash_allowed=false`, and `flash_executed=false`. The current physical
  official-compatible firmware still depends on stock Xiaozhi WakeNet/touch.
- Updated the repo and Desktop StackChan control helpers so `say` can play a
  foreground physical TTS test from the helper.
- Updated `docs/engineering/PROTOCOL.md` and
  `docs/project_state_machine.md` with the new `/v1/xiaozhi/say` boundary,
  live TTS evidence, and wake-word truth.

Files changed:

- `docs/agent_handoff_log.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `tools/desktop/a21-stackchan-control.command`
- `/Users/jiyurun/Desktop/A21-StackChan-Control.command`

Current unfinished items:

- The operator should provide the recording captured during
  `a21-trace-live-long-tts-hotfix`; analyze it against `7.m4a` and `8.m4a`.
- If the new recording is still not loud enough, move from protocol parity to
  TTS source/gain staging: local TTS WAV amplitude, limiter/headroom before
  Opus encode, and optional guarded firmware volume-100 persistence if MCP
  volume does not persist across sessions.
- The live trace shows device microphone ingress after speaker playback, which
  can retrigger the voice pipeline; half-duplex/AEC/speaker-ducking remains a
  next transition.
- Custom wake-word product readiness remains blocked until the packaged
  MultiNet firmware is flashed through a guarded hardware window and followed
  by physical custom wake proof.

Known risks and blockers:

- `/v1/xiaozhi/say` is an operator foreground test path, not product dialogue
  acceptance.
- Runtime MCP volume delivery is proven at the WebSocket/Gateway layer; physical
  loudness still needs the fresh recording analysis.
- The current physical firmware does not contain the custom wake package, so
  voice wake behavior remains stock and can be weak depending on phrase,
  environment, and microphone/AEC state.

Recommended next action:

- Analyze the operator recording for `a21-trace-live-long-tts-hotfix`.
- Dispatch a focused worker for `T-AUDIO-003: TTS Gain And Half-Duplex
  Stabilization` if loudness or self-trigger remains bad.
- Dispatch a separate firmware worker for `T-FW-004: Guarded Custom Wake Flash`
  using the existing wake package if the contest flow requires voice wake rather
  than touch.

Validation results:

- `go test ./internal/gateway -run 'TestXiaozhiSayDeliversTextAsStockTTSDownlink|TestXiaozhiSpeakerVolumeUsesStockMCPToolCall|TestWriteXiaozhiOpusDownlinkReusesTurnEncoderForContiguousFrames' -count=1`:
  passed.
- `go test ./internal/gateway ./internal/providers ./internal/transport/xiaozhi ./internal/app -run 'TestXiaozhiSayDeliversTextAsStockTTSDownlink|TestXiaozhiSpeakerVolumeUsesStockMCPToolCall|TestXiaozhiListenReplySuppressedForStockPhysicalMACDevice|TestWriteXiaozhiOpusDownlinkReusesTurnEncoderForContiguousFrames|TestWriteXiaozhiOpusDownlinkUsesPacerAndCurrentTurn|TestXiaozhiOpusDownlinkFansOutOfficialStackChanSpeaking|TestBuildServerHello|TestVoicePipelineRunnerProducesDownlinkReadyMockChunksAndRedactedReport|TestLocalTTSAdapterConvertsWAVToDownlinkReady24KChunks|XiaozhiVoiceBench|TestRunXiaozhiVoiceBenchReportsHostOnlyCandidateEvidence' -count=1`:
  passed.
- `go test ./internal/app ./internal/gateway ./internal/providers ./internal/audio/opuscodec ./internal/transport/xiaozhi -run 'Xiaozhi|StackChan|VoicePipeline|Opus|LocalTTS|Playback|Barge|Abort|ProviderLatencyFixture|FastCompanion' -count=1`:
  passed.
- `zsh -n tools/desktop/a21-stackchan-control.command`: passed.
- `zsh -n /Users/jiyurun/Desktop/A21-StackChan-Control.command`: passed.
- `git diff --check`: passed.

## 2026-06-03 00:29 CST - T-INTEGRATE-001 Accepted Audio Hotfix Review

Round goal:

- Review the T-AUDIO-002 and T-AUDIO-003 hotfixes after operator acceptance.
- Record the final accepted recording evidence.
- If no blocking findings are found, integrate the hotfix set into the current
  branch with tests and state-machine updates.

Actual completed work:

- Analyzed accepted operator recording
  `/Users/jiyurun/Downloads/军民公路259号 10.m4a`:
  - duration `37.781333 s`, AAC 48 kHz stereo.
  - from 3 seconds: `-26.5 LUFS`, `-8.6 dBFS` true peak, `LRA=4.3 LU`.
  - This matches the final 3x gain lane and the operator explicitly marked the
    audio acceptance as passed.
- Reviewed hotfix diff boundaries:
  - stock protocol remains clean: physical MAC stock devices no longer receive
    unsupported `type=listen` replies; debug/virtual paths keep replies.
  - runtime volume path uses stock MCP tool
    `self.audio_speaker.set_volume`.
  - `/v1/xiaozhi/say` is a Gateway/operator foreground path that sends stock
    TTS lifecycle and Opus binary downlink over the live `/v1/xiaozhi` socket.
  - downlink audio aligns to 24 kHz mono 60 ms; uplink remains stock 16 kHz.
  - turn-level Opus encoder reuse and one-frame prebuffer remove avoidable
    churn/startup burst.
  - bounded PCM leveling now uses 3x max gain with a noise gate and headroom
    cap after 4x was rejected by listening feedback.
  - host-say input suppression is limited to stock physical MAC devices and
    does not claim normal dialogue half-duplex acceptance.
- Fixed an inaccurate earlier handoff file list that mentioned
  `internal/transport/xiaozhi/*` even though those files are not in the final
  diff.
- Updated `docs/project_state_machine.md`:
  - total state is now
    `S-HW-PHYSICAL-XIAOZHI-AUDIO-ACCEPTED-WAKE-PENDING`;
  - `T-AUDIO-002` and `T-AUDIO-003` are recorded as completed;
  - stale audio/volume blockers were removed or narrowed;
  - next candidates are custom wake flash/proof, normal-dialogue half-duplex,
    and selected-provider readiness refresh.

Files changed this round:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Current unfinished items:

- Commit the reviewed integration set if verification remains green.
- Full PRD physical acceptance remains false until custom wake proof, broader
  normal-dialogue half-duplex, and selected-provider readiness refresh are
  closed.

Known risks and blockers:

- `/v1/xiaozhi/say` is still an operator foreground path, not a replacement for
  normal product dialogue acceptance.
- Phone recordings remain room-path evidence, though the operator accepted the
  latest result.
- Custom wake package is still not flashed into the current physical app.

Recommended next action:

- Commit the accepted hotfix set.
- Start `T-FW-003` guarded custom wake flash/proof or `T-HALF-DUPLEX-001`
  normal-dialogue echo suppression depending on contest priority.

Validation results:

- `go test ./internal/gateway ./internal/providers ./internal/transport/xiaozhi ./internal/app -run 'TestXiaozhiSayDeliversTextAsStockTTSDownlink|TestXiaozhiSpeakerVolumeUsesStockMCPToolCall|TestXiaozhiListenReplySuppressedForStockPhysicalMACDevice|TestWriteXiaozhiOpusDownlinkReusesTurnEncoderForContiguousFrames|TestWriteXiaozhiOpusDownlinkUsesPacerAndCurrentTurn|TestXiaozhiOpusDownlinkFansOutOfficialStackChanSpeaking|TestBuildServerHello|TestVoicePipelineRunnerProducesDownlinkReadyMockChunksAndRedactedReport|TestLocalTTSAdapterConvertsWAVToDownlinkReady24KChunks|XiaozhiVoiceBench|TestRunXiaozhiVoiceBenchReportsHostOnlyCandidateEvidence|TestXiaozhiSaySuppressesImmediateListenRestartForStockPhysical|TestXiaozhiDownlinkPCM16' -count=1`:
  passed.
- `go test ./internal/app ./internal/gateway ./internal/providers ./internal/audio/opuscodec ./internal/transport/xiaozhi -run 'Xiaozhi|StackChan|VoicePipeline|Opus|LocalTTS|Playback|Barge|Abort|ProviderLatencyFixture|FastCompanion' -count=1`:
  passed.
- `zsh -n tools/desktop/a21-stackchan-control.command`: passed.
- `zsh -n /Users/jiyurun/Desktop/A21-StackChan-Control.command`: passed.
- `git diff --check`: passed.
- `make verify`: passed, including `go test ./...` and `git diff --check`.
- `tools/desktop/a21-stackchan-control.command status`: Gateway health ok;
  physical device record present but stale after the acceptance window.
- `curl --noproxy '*' http://127.0.0.1:21081/healthz`: returned Gateway ok
  after restart.
- `POST /v1/xiaozhi/speaker-volume`: delivered volume `100` to
  `44:1b:f6:e2:6a:60`.
- `POST /v1/xiaozhi/say`: delivered `676` audio chunks to the live physical
  stock socket.
- After the foreground tool session was closed, Gateway was relaunched in tmux
  session `a21-gateway-21081`; `curl --noproxy '*' http://127.0.0.1:21081/healthz`
  returned ok.

## 2026-06-03 00:07 CST - T-AUDIO-003 Bounded Gain And Host-Say Echo Suppression

Round goal:

- Continue from T-AUDIO-002 without changing principles or widening scope.
- Analyze operator recording `/Users/jiyurun/Downloads/军民公路259号 9.m4a`.
- Fix the remaining "not loud enough" path only after evidence.
- Reduce immediate host-say speaker-to-mic self-trigger risk.

Actual completed work:

- Verified current repo state before edits: branch
  `codex/a21-hardware-window-20260602-stackchan-prd`, HEAD `ebc0db4`, dirty
  with prior T-AUDIO-002 changes.
- Verified Gateway tmux session `a21-gateway-21081` was healthy and the physical
  device `44:1b:f6:e2:6a:60` was online.
- Analyzed `9.m4a`:
  - full file duration `48.618667 s`, AAC 48 kHz stereo.
  - from 3 seconds: `-36.99 LUFS`, `-16.05 dBTP`,
    `input_lra=5.80`, `input_thresh=-47.70`.
  - best early 10-second window from 3 seconds:
    `-34.79 LUFS`, `-16.05 dBTP`.
  - no clear `-45 dB` long-silence splits were detected, so the weak integrated
    loudness was not only a blank-tail artifact.
- Spawned two read-only sidecar agents:
  - `019e8910-7ecb-7e01-a468-9480df0ba61d`: confirmed Gateway downlink only
    attenuated hot PCM and never lifted quiet TTS frames; recommended bounded
    Gateway-level leveling as the smallest safe patch.
  - `019e8910-b7b4-7f92-98c8-1eb21286b7c0`: confirmed `/v1/xiaozhi/say` did not
    arm input suppression, allowing speaker playback to be accepted as new
    listen/voice-pipeline input; recommended host-say-only stock physical input
    suppression.
- Added plan
  `docs/plans/2026-06-02-stackchan-tts-gain-half-duplex-hotfix.md`.
- Replaced pure `xiaozhiDownlinkPCM16` headroom limiting with bounded PCM
  leveling:
  - noise gate leaves tiny noise unchanged;
  - quiet non-silent TTS frames can be lifted up to the target peak;
  - max gain cap is `4000` milli (`4x`);
  - headroom cap remains `29490`.
- Added host-say-only input suppression:
  - constant `xiaozhiHostSayInputCooldownMS=1200`;
  - only stock physical MAC devices, not debug profile;
  - records `xiaozhi.say.input_suppression_armed`;
  - immediate `listen/start` during cooldown records
    `xiaozhi.listen.start.input_suppressed` and remains ignored.
- Sent one boosted physical `/v1/xiaozhi/say` turn after loading the first
  4x gain build:
  `trace_id=a21-trace-live-long-tts-gain`, `text_chars=226`,
  `audio_chunks=669`.
- Analyzed the operator's boosted recording
  `/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a`:
  - duration `29.077333 s`, AAC 48 kHz stereo.
  - from 3 seconds: `-21.65 LUFS`, `-4.72 dBTP`,
    `input_lra=6.20`, `input_thresh=-32.61`.
  - 3s/10s window: `-20.63 LUFS`, `-4.72 dBTP`.
  - FFmpeg `astats` from 3 seconds: overall peak `-4.774703 dB`,
    RMS `-27.138800 dB`, no NaNs/Infs/denormals and no obvious clipping.
  - This materially improves 9.m4a and moves the issue from "too quiet" to
    operator listening confirmation plus half-duplex stability.
- Updated `docs/engineering/PROTOCOL.md` with bounded downlink leveling and
  post-host-say input suppression.
- Updated `docs/project_state_machine.md` to make `T-AUDIO-003` the active
  child transition.

Files changed:

- `docs/agent_handoff_log.md`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-02-stackchan-tts-gain-half-duplex-hotfix.md`
- `docs/project_state_machine.md`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`

Current unfinished items:

- Restart tmux Gateway from the final tree after the latest host-say suppression
  patch, then verify `/healthz`.
- Run broader focused tests and `git diff --check`.
- Operator should confirm whether the boosted recording sounds clear, not harsh,
  and acceptable for contest demo use.
- If boosted audio is too hot or harsh, reduce max gain from `4000` milli to the
  sidecar's safer `3000` milli while preserving the same tests.
- Normal dialogue half-duplex after non-host-say TTS still needs physical proof;
  this round only guards the operator foreground `/v1/xiaozhi/say` path.

Known risks and blockers:

- Per-frame leveling can cause pumping on highly variable TTS; current physical
  recording did not show obvious clipping, but operator listening judgement is
  still required.
- Phone recordings are room-path evidence, not exact SPL.
- Custom wake word remains packaged but not flashed into the current physical
  app.

Recommended next action:

- Restart Gateway from the final working tree.
- Ask operator for subjective verdict on
  `/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a`.
- If accepted, proceed to `T-FW-004` custom wake flash or `T-HALF-DUPLEX-001`
  normal dialogue echo suppression, depending on contest-critical priority.

Validation results so far:

- `go test ./internal/gateway -run 'TestXiaozhiDownlinkPCM16|TestXiaozhiSayDeliversTextAsStockTTSDownlink|TestXiaozhiSpeakerVolumeUsesStockMCPToolCall|TestWriteXiaozhiOpusDownlinkReusesTurnEncoderForContiguousFrames' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestXiaozhiSaySuppressesImmediateListenRestartForStockPhysical|TestXiaozhiSayDeliversTextAsStockTTSDownlink|TestXiaozhiWebSocketTouchAbortSuppressesImmediateListenRestart|TestXiaozhiSessionRecentDownlinkCanBeInterruptedAfterTurnCompletion|TestXiaozhiWebSocketListenStopRunsVoicePipelineAndSendsPacedOpus|TestXiaozhiDownlinkPCM16' -count=1`:
  passed.

## 2026-06-03 00:22 CST - T-AUDIO-003 Roll Back Gain Cap To 3x

Round goal:

- Respond to operator feedback that the 4x Gateway gain candidate felt like a
  slight regression with subtle electrical interruption and reduced clarity.
- Roll the Xiaozhi downlink max-gain cap back from 4x to 3x without changing
  stock protocol, firmware, provider routing, or V21 boundaries.
- Keep the physical 3x candidate live for immediate listening retest.

Actual completed work:

- Analyzed the new operator recording
  `/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 3.m4a`:
  - duration `35.818667 s`, AAC 48 kHz stereo.
  - from 3 seconds: `-26.4 LUFS`, `-9.0 dBFS` true peak, `LRA=3.8 LU`.
  - FFmpeg `astats` from 3 seconds: overall peak `-9.086959 dB`, RMS
    `-31.859961 dB`, no NaNs/Infs/denormals.
- Used TDD for the gain rollback:
  - first changed
    `TestXiaozhiDownlinkPCM16BoostsQuietTTSFramesWithinHeadroom` to expect
    bounded 3x boost from peak `6000` to `18000`;
  - verified the test failed against the 4x code with
    `quiet TTS pcm peak = 24000, want bounded 3x boost to 18000`;
  - changed `xiaozhiDownlinkPCM16MaxGainMilli` from `4000` to `3000`.
- Updated plan/state docs to record that 4x was rejected by listening feedback
  and 3x is the current candidate.
- Relaunched the Gateway from the 3x working tree in tmux session
  `a21-gateway-21081`; `/healthz` returned ok.
- Confirmed physical StackChan `44:1b:f6:e2:6a:60` re-registered online over
  stock `/v1/xiaozhi`.
- Delivered runtime volume `100` through stock MCP:
  `trace_id=a21-trace-stackchan-volume-1780417211`.
- Played a 3x long physical TTS through `/v1/xiaozhi/say`:
  `trace_id=a21-trace-stackchan-say-1780417217`, `text_chars=176`,
  `audio_chunks=579`.
- Trace summary for `a21-trace-stackchan-say-1780417217`:
  - `answer_first_audio_total_ms=1656`;
  - `tts.first_audio` at offset `1772 ms`;
  - `audio.downlink.first_frame` at offset `1773 ms`;
  - `xiaozhi.say.input_suppression_armed=1`;
  - `xiaozhi.listen.start.input_suppressed=1`;
  - `xiaozhi.opus_frame.ignored_not_listening=259`.

Files changed this round:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-02-stackchan-tts-gain-half-duplex-hotfix.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current unfinished items:

- Operator must listen to the just-played 3x physical TTS and decide whether
  clarity is now acceptable.
- If 3x is still unclear or electrically interrupted, stop tuning gain upward
  and route the next transition to TTS source/profile quality, device codec
  instrumentation, or guarded firmware playback gain rather than stacking more
  Gateway amplification.
- Normal dialogue half-duplex after non-host-say TTS still needs separate
  physical acceptance.
- Custom wake word remains packaged but not flashed into the current physical
  app.

Known risks and blockers:

- Phone recordings are room-path evidence, not exact SPL.
- 3x may be slightly less loud than 4x; this is intentional to preserve
  clarity.
- The physical device currently uses stock Xiaozhi WakeNet/touch activation;
  custom wake proof remains blocked until a guarded flash/run is executed.

Recommended next action:

- If the operator accepts 3x listening quality, freeze Gateway gain at 3x and
  move to `T-FW-004` custom wake flash or `T-HALF-DUPLEX-001` normal dialogue
  echo suppression.
- If the operator rejects 3x clarity, start a new small transition focused on
  TTS voice/model/source quality and device-side playback instrumentation.

Validation results:

- `go test ./internal/gateway -run TestXiaozhiDownlinkPCM16BoostsQuietTTSFramesWithinHeadroom -count=1`:
  failed before production-code rollback as expected.
- `go test ./internal/gateway -run 'TestXiaozhiDownlinkPCM16BoostsQuietTTSFramesWithinHeadroom|TestXiaozhiDownlinkPCM16DoesNotBoostTinyNoise|TestXiaozhiDownlinkPCM16AppliesHeadroomToHotTTSFrames|TestXiaozhiDownlinkPCM16Accepts48KProviderFrames' -count=1`:
  passed after rollback.
- `go test ./internal/gateway ./internal/providers ./internal/transport/xiaozhi ./internal/app -run 'TestXiaozhiSayDeliversTextAsStockTTSDownlink|TestXiaozhiSpeakerVolumeUsesStockMCPToolCall|TestXiaozhiListenReplySuppressedForStockPhysicalMACDevice|TestWriteXiaozhiOpusDownlinkReusesTurnEncoderForContiguousFrames|TestWriteXiaozhiOpusDownlinkUsesPacerAndCurrentTurn|TestXiaozhiOpusDownlinkFansOutOfficialStackChanSpeaking|TestBuildServerHello|TestVoicePipelineRunnerProducesDownlinkReadyMockChunksAndRedactedReport|TestLocalTTSAdapterConvertsWAVToDownlinkReady24KChunks|XiaozhiVoiceBench|TestRunXiaozhiVoiceBenchReportsHostOnlyCandidateEvidence|TestXiaozhiSaySuppressesImmediateListenRestartForStockPhysical|TestXiaozhiDownlinkPCM16' -count=1`:
  passed.
- `go test ./internal/app ./internal/gateway ./internal/providers ./internal/audio/opuscodec ./internal/transport/xiaozhi -run 'Xiaozhi|StackChan|VoicePipeline|Opus|LocalTTS|Playback|Barge|Abort|ProviderLatencyFixture|FastCompanion' -count=1`:
  passed.
- `zsh -n tools/desktop/a21-stackchan-control.command`: passed.
- `zsh -n /Users/jiyurun/Desktop/A21-StackChan-Control.command`: passed.
- `git diff --check`: passed.

## 2026-06-03 00:32 CST - T-INTEGRATE-001 Tail Pointer After Commit

Round goal:

- Keep the latest handoff visible at the end of the log after integrating the
  accepted audio hotfix set.

Actual completed work:

- Operator accepted the 3x physical audio result.
- Accepted recording `/Users/jiyurun/Downloads/军民公路259号 10.m4a` measured
  `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds.
- Review found no blocking issue in the two hotfix rounds:
  - stock physical devices no longer receive unsupported `listen` replies;
  - runtime volume uses stock MCP `self.audio_speaker.set_volume`;
  - `/v1/xiaozhi/say` is an operator foreground path using stock TTS lifecycle
    plus Opus binary downlink;
  - downlink TTS uses 24 kHz mono 60 ms;
  - bounded leveling is capped at 3x with noise gate and headroom;
  - host-say input suppression is physical-observed but not overclaimed as
    normal-dialogue half-duplex acceptance.
- `docs/project_state_machine.md` now records
  `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED` with active
  `T-NEXT-PENDING`.

Files changed this tail-pointer update:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Current unfinished items:

- Full PRD acceptance remains blocked by custom wake proof, broader normal
  dialogue half-duplex, and selected-provider readiness refresh.

Known risks and blockers:

- Gateway health is ok; after the acceptance window the physical device registry
  record was present but stale.
- Phone recordings are accepted operator evidence but not exact SPL.

Recommended next action:

- Next transitions: `T-FW-003`, `T-HALF-DUPLEX-001`, `T-PROVIDER-001b`.

Validation results:

- Focused Gateway/provider/transport/app tests: passed.
- Broader Xiaozhi/StackChan/voice/Opus focused tests: passed.
- Desktop helper syntax checks: passed.
- `git diff --check`: passed.
- `make verify`: passed, including `go test ./...` and `git diff --check`.

## 2026-06-03 01:10 CST - T-PROVIDER-002 Hot-Plug TTS/LLM And Voice Clone Recovery

Round goal:

- Respond to the operator correction that A21 must keep voice-clone capability.
- Preserve `voice_clone_cli` while adding the immediate contest candidate path:
  StepFun `step-1-8k` text stream plus Iflytek/Xfyun real-time TTS.
- Keep stock Xiaozhi firmware/protocol, accepted 3x Gateway gain, and Codex/global
  proxy settings unchanged.

Actual completed work:

- Confirmed by code search and read-only subagent `019e8943-e565-7c62-9fff-23fcb2c904c4`
  that `voice_clone_cli` is still the retained A21 voice-clone seam:
  `internal/audio/local_tts.go`, `internal/app/app_audio.go`,
  `internal/providers/voice_pipeline_adapters.go`,
  `internal/app/product_demo.go`, `scripts/a21_5080_indextts2_bridge.py`,
  `scripts/a21_5080_indextts2_bridge_test.py`, and
  `docs/engineering/VOICE_CLONE_TTS.md`.
- Integrated host-side Iflytek TTS as `iflytek_tts`:
  - env names: `A21_IFLYTEK_TTS_APP_ID`,
    `A21_IFLYTEK_TTS_API_KEY`, `A21_IFLYTEK_TTS_API_SECRET`;
  - HMAC WebSocket auth URL generation;
  - 16 kHz PCM request and WAV writing;
  - redacted `a21.audio.local_tts.v1` report with endpoint host, direct network
    mode, timing, and PCM quality;
  - direct WebSocket HTTP client that ignores ambient `HTTP_PROXY` /
    `HTTPS_PROXY`.
- Exposed `iflytek_tts` through:
  - `local-tts-smoke`;
  - `local-voice-loopback`;
  - `stackchan-local-tts-playback`;
  - `stackchan-fast-companion-turn`;
  - Gateway voice-pipeline TTS selection via `A21_TTS_FAST_PROFILE`.
- Relaxed runtime text-stream hot-plug selection so explicit compatibility
  candidates such as `stepfun` can execute without being promoted to product
  `route_eligible=true`.
- Added failure-report behavior for `local-tts-smoke`: provider synthesis
  errors now still write a redacted report when the synthesizer returns one.
- Updated docs to separate the four TTS concepts:
  - Iflytek: immediate fast real-time contest TTS candidate;
  - StepFun: explicit text-stream candidate, not product route-eligible yet;
  - `voice_clone_cli`: retained voice-clone/persona capability;
  - `sherpa_onnx`: emergency/diagnostic fallback only for this contest window.
- Read the 5080 report through SSH without storing it in the repo. The report
  contains plaintext credentials; repo docs/logs record only env names and
  redacted evidence.

Files changed this round:

- `internal/audio/local_tts.go`
- `internal/audio/local_tts_test.go`
- `internal/app/app.go`
- `internal/app/app_audio.go`
- `internal/app/app_audio_loopback.go`
- `internal/app/app_stackchan_playback.go`
- `internal/app/app_test.go`
- `internal/app/fast_companion_turn.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `docs/plans/2026-06-03-provider-tts-real-dialogue-acceptance.md`
- `docs/project_state_machine.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/NETWORK.md`
- `docs/engineering/VOICE_CLONE_TTS.md`
- `docs/agent_handoff_log.md`

Current unfinished items:

- Iflytek TTS is coded and tested, but Mac direct live WebSocket smoke is
  blocked:
  `reports/provider-tts-candidate/a21-local-tts-smoke-20260603-010638.json`
  has `status=failed`, `network_mode=direct`, finding
  `iflytek_tts_websocket_dial_failed`.
- StepFun direct provider-smoke passed, but `local-voice-loopback` with StepFun
  was unstable on the Mac direct path:
  - first retry failed awaiting headers after client timeout;
  - second retry failed during TLS handshake.
- No physical StackChan playback was attempted in this round because the real
  TTS source did not synthesize successfully.
- Full PRD acceptance remains blocked by custom wake proof, normal dialogue
  half-duplex proof, and selected-provider/live-TTS readiness.

Known risks and blockers:

- The 5080 report contains plaintext credentials; do not copy values into repo
  docs, shell snippets, reports, or final messages. Consider rotating them
  outside this repo.
- Mac direct provider network is not reliable enough for the real-time loopback
  path even though `provider-smoke` can pass.
- If explicit WebSocket provider proxying is needed, it requires a separate
  adapter transition; the current Iflytek WebSocket path is direct-only by
  design.
- `voice_clone_cli` readiness is configuration/file-existence readiness until
  a wrapper/model smoke and physical playback acceptance run.

Recommended next action:

- Execute `T-PROVIDER-002b: Iflytek/Real-TTS Live Chain Unblock`.
- Fastest likely path: use 5080/Alibaba as the TTS egress/relay rather than
  continuing to rely on Mac direct WebSocket, then rerun:
  - `local-tts-smoke --engine iflytek_tts`;
  - `local-voice-loopback --engine iflytek_tts --text-provider stepfun
    --execute-text-provider`;
  - physical `stackchan-local-tts-playback` or `stackchan-fast-companion-turn`
    through Gateway `21081`.
- Keep `voice_clone_cli` available for a parallel clone/persona smoke after the
  immediate Iflytek real-time path is unblocked.

Validation results:

- `go test ./internal/audio ./internal/providers ./internal/app -run 'Iflytek|VoiceClone|LocalTTS|LocalVoiceLoopback|StackChanLocalTTSPlayback|ProductVoiceReadiness|ProviderCompatMatrix|VoicePipelineAdapters' -count=1`:
  passed.
- `python3 scripts/a21_5080_indextts2_bridge_test.py`: passed.
- `go test ./internal/app ./internal/gateway ./internal/providers ./internal/audio -run 'Iflytek|VoiceClone|VoicePipeline|LocalVoiceLoopback|StackChan|FastCompanion|Xiaozhi' -count=1`:
  passed.
- `git diff --check`: passed.
- `make verify`: passed, including `go test ./...` and `git diff --check`.
- Live StepFun provider smoke:
  `reports/provider-tts-candidate/a21-provider-smoke-20260603-010652-957877000.json`
  passed with `provider=stepfun`, `network_mode=direct`, `route_eligible=false`,
  `repeat=2`, first-content p50 `442.975 ms`, p95 `1095.192 ms`.
- Live Iflytek TTS smoke:
  `reports/provider-tts-candidate/a21-local-tts-smoke-20260603-010638.json`
  failed at WebSocket dial as described above.

## 2026-06-03 01:18 CST - T-HALF-DUPLEX-001 Readiness Blocked By Stale Device

Round goal:

- Execute Worker HALF-DUPLEX readiness/probe/report work for
  `T-HALF-DUPLEX-001` without firmware flash, NVS writes, provider/TTS
  changes, wake-word changes, or product-ready overclaim.

Actual completed work:

- Read `AGENTS.md`, `docs/project_state_machine.md`,
  `docs/agent_handoff_log.md`, and
  `docs/plans/2026-06-03-normal-dialogue-half-duplex-acceptance.md`.
- Confirmed checkout baseline `b5a405eedabd11b2b5e23e38fdf1b220c5b69ce6`
  on branch `codex/a21-hardware-window-20260602-stackchan-prd`.
- Checked live Gateway health on `127.0.0.1:21081`; response was
  `service=a21-gateway`, `status=ok`, `version=0.1.0-dev`.
- Checked `/v1/devices` on `127.0.0.1:21081` twice. Physical device
  `44:1b:f6:e2:6a:60` was registered but stale:
  - `connection_status=stale`;
  - latest observed `device_age_ms=534028`;
  - `identity_status=unknown`;
  - `firmware={}`;
  - `microphone=available_xiaozhi_opus_ingress`;
  - `speaker=available_xiaozhi_opus_downlink`;
  - `speaker_volume=100`;
  - `last_event=xiaozhi.tts.opus_frame.downlink`;
  - `last_trace_id=a21-trace-44-1b-f6-e2-6a-60`;
  - `last_session_id=a21-session-44-1b-f6-e2-6a-60`.
- Checked default Gateway `127.0.0.1:21080`; health was ok but `/v1/devices`
  only showed stale virtual device `stackchan-virtual-a21-bench-001`.
- Did not run `stackchan-half-duplex-acceptance` because the plan permits the
  physical probe only if the live device is online/fresh. Current device state
  would block before valid normal-dialogue half-duplex evidence and lacks the
  diagnostic mic capability required by the plan.

Files changed this round:

- `docs/agent_handoff_log.md`

Current unfinished items:

- No half-duplex acceptance report was generated in this round.
- Normal dialogue half-duplex remains unaccepted.
- The live physical device must reconnect freshly before the instrumented probe
  can be rerun.
- The current stock Xiaozhi firmware reports microphone capability
  `available_xiaozhi_opus_ingress`, not
  `diagnostic_probe_m5unified_i2s_capture`, so this transition may still need
  an explicit foreground diagnostic-capability decision before acceptance.

Known risks and blockers:

- `connection_status=stale` means a control write would not be valid physical
  acceptance evidence.
- Empty firmware identity and `identity_status=unknown` would block the current
  half-duplex acceptance report even if a control request were attempted.
- Do not interpret prior host-say suppression markers as normal-dialogue
  half-duplex acceptance.

Recommended next action:

- Foreground operator should wake/reconnect the physical StackChan to Gateway
  `127.0.0.1:21081`, confirm `/v1/devices` shows
  `connection_status=online`, then rerun:
  `A21_GATEWAY_URL=http://127.0.0.1:21081 A21_DEVICE_ID=44:1b:f6:e2:6a:60 make stackchan-half-duplex-acceptance`.
- If the device still does not expose
  `diagnostic_probe_m5unified_i2s_capture`, block honestly or dispatch a
  separate guarded diagnostic firmware/capability plan; do not flash from this
  worker.

Validation results:

- `curl http://127.0.0.1:21081/healthz`: passed.
- `curl http://127.0.0.1:21081/v1/devices`: passed twice, physical device
  stale.
- `curl http://127.0.0.1:21080/healthz`: passed.
- `curl http://127.0.0.1:21080/v1/devices`: passed, virtual stale device only.
- `stackchan-half-duplex-acceptance`: not run; blocked by stale physical device
  state.

## 2026-06-03 01:31 CST - T-PROVIDER-002b 5080 Relay TTS Chain Unblocked, Physical Playback Stale-Blocked

Round goal:

- Execute Worker PROVIDER for `T-PROVIDER-002b`: unblock the real TTS live
  chain using 5080/Alibaba relay first, falling back to an explicit WebSocket
  egress adapter only if the relay was blocked.

Actual completed work:

- Read `AGENTS.md`, `docs/project_state_machine.md`,
  `docs/agent_handoff_log.md`, and
  `docs/plans/2026-06-03-provider-tts-live-chain-unblock.md`.
- Confirmed checkout baseline `b5a405eedabd11b2b5e23e38fdf1b220c5b69ce6`
  on branch `codex/a21-hardware-window-20260602-stackchan-prd`.
- Confirmed 5080 SSH was reachable through the established inbox/outbox lane.
- Used the existing 5080 Iflytek helper as the relay base without printing
  credentials.
- Ran a throwaway remote 5080 harness that imported the existing helper,
  synthesized Iflytek 16 kHz PCM TTS, wrote a WAV, and returned only a redacted
  report bundle.
- Ran a second throwaway remote 5080 harness that executed StepFun
  `step-1-8k` streaming text, fed the returned content directly into Iflytek
  TTS, wrote a WAV, and returned only a redacted loopback report bundle.
- Imported the returned redacted reports and WAV files under
  `reports/provider-tts-candidate/`.
- Ran a secret/text/path scan on the imported JSON reports; it found no auth
  query, API key, API secret, bearer token, raw/base64 audio marker, full URL,
  provider env name, local path, or probe text.
- Checked Gateway `127.0.0.1:21081` health; it was ok.
- Checked `/v1/devices`; physical device `44:1b:f6:e2:6a:60` was registered
  but `connection_status=stale`, so no physical playback command was sent.

Files changed this round:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `reports/provider-tts-candidate/a21-local-tts-smoke-5080-relay-20260603-0125.json`
- `reports/provider-tts-candidate/a21-iflytek-tts-5080-relay-20260603-0125.wav`
- `reports/provider-tts-candidate/a21-local-voice-loopback-5080-relay-20260603-0128.json`
- `reports/provider-tts-candidate/a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`

Current unfinished items:

- Physical StackChan playback of the StepFun+Iflytek chain was not attempted
  because the physical socket was stale.
- Operator listening acceptance remains missing.
- Mac direct Iflytek WebSocket remains blocked; relay is the working path for
  this window.

Known risks and blockers:

- The remote 5080 source report still contains plaintext credentials outside
  this repo; do not copy values into repo docs, reports, commands, or final
  messages.
- The imported WAV is candidate audio evidence, not physical acceptance until
  replayed through the live stock Xiaozhi path and judged by the operator.
- StepFun remains an explicit compatibility candidate in this evidence; it is
  not promoted to product `route_eligible=true`.

Recommended next action:

- Foreground operator should wake/reconnect StackChan to Gateway `21081` until
  `/v1/devices` shows `connection_status=online`, then run:
  `go run ./cmd/a21 stackchan-local-tts-playback --gateway-url http://127.0.0.1:21081 --device-id 44:1b:f6:e2:6a:60 --wav reports/provider-tts-candidate/a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav --output-dir reports/provider-tts-candidate`
  and record operator listening acceptance or rejection.

Validation results:

- 5080 Iflytek TTS relay smoke: passed; report
  `reports/provider-tts-candidate/a21-local-tts-smoke-5080-relay-20260603-0125.json`;
  WAV `reports/provider-tts-candidate/a21-iflytek-tts-5080-relay-20260603-0125.wav`;
  `tts_first_audio_ms=100.299`.
- 5080 StepFun+Iflytek relay chain: passed; report
  `reports/provider-tts-candidate/a21-local-voice-loopback-5080-relay-20260603-0128.json`;
  WAV `reports/provider-tts-candidate/a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`;
  `text_stream_first_content_ms=229.077`, `tts_first_audio_ms=87.947`.
- Imported report redaction scan: passed.
- `curl http://127.0.0.1:21081/healthz`: passed.
- `curl http://127.0.0.1:21081/v1/devices`: passed, physical device stale.
- Physical playback: not attempted because stale device state would not produce
  valid acceptance evidence.

## 2026-06-03 01:42 CST - T-FW-003 Wake Package Integrity Passed, Guarded Flash Blocked By Missing Build Dir

Round goal:

- Execute Worker WAKE for `T-FW-003` package integrity and no-write gate prep
  without firmware flash, NVS writes, provider/TTS edits, half-duplex edits, or
  product-ready overclaim.

Actual completed work:

- Verified wake package report
  `reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json`
  is below activation: `status=packaged`, `package_written=true`,
  `product_ready=false`, `flash_allowed=false`, `flash_executed=false`.
- Verified artifact exists:
  `reports/a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-a2d3dc882b42-20260602-075112.bin`.
- Verified artifact SHA-256 matches package report and `.sha256`:
  `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`.
- Verified artifact size matches app part: `2853680` bytes.
- Verified manifest exists and matches package metadata.
- Ran product readiness with the wake package report; report
  `reports/a21-product-readiness-20260603-011911.json` stayed blocked as
  expected with `launch_ready=false`, `wake_word.product_ready=false`,
  `firmware_package_available=true`,
  `firmware_package_flash_allowed=false`, and
  `physical_firmware_flash_executed=false`.
- Did not produce an exact guarded flash command because the reviewed A21
  ESP-IDF build directory is missing from the workspace/reports lane. The
  package lane only has copied app artifact plus manifest/hash, not full flash
  inputs.

Files changed this round:

- Ignored/generated report:
  `reports/a21-product-readiness-20260603-011911.json`
- No tracked files changed by Worker WAKE.

Current unfinished items:

- No wake flash-plan report was generated.
- No custom wake firmware was flashed.
- No custom wake physical proof exists.
- Need the reviewed A21 scratch Xiaozhi build directory that produced SHA
  `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`,
  outside any frozen X21/external source path.

Known risks and blockers:

- Current repo command family `xiaozhi-firmware-flash` requires the reviewed
  build dir with `sdkconfig.json`, flash args, app bin, bootloader, partition,
  OTA data, and assets.
- External Xiaozhi/X21-adjacent build dirs are explicitly rejected by A21
  guardrails and cannot be used as valid flash inputs for this transition.
- Product readiness must remain red until guarded flash plus physical custom
  wake proof.

Recommended next action:

- Restore/provide the reviewed A21 build directory for the wake artifact SHA.
- Confirm foreground upload port, currently expected as `/dev/cu.usbmodem1101`.
- Run no-write `xiaozhi-firmware-flash-plan` first; only execute after that
  plan is ready and the operator explicitly opens the guarded write window.

Validation results:

- `shasum -a 256 reports/a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-a2d3dc882b42-20260602-075112.bin`:
  passed.
- `wc -c reports/a21-wake-word-xiaozhi-esp-sr-multinet-m5stack-cores3-a2d3dc882b42-20260602-075112.bin`:
  passed.
- `go run ./cmd/a21 product-readiness --wake-word-firmware-package-report reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json --output-dir reports --require-real`:
  exited `1` as expected because launch readiness is false and wrote
  `reports/a21-product-readiness-20260603-011911.json`.
- `go test ./internal/app -run 'WakeWordFirmware|XiaozhiFirmwareFlash' -count=1`:
  passed.
- `go test ./internal/runtimeguard -count=1`: passed.

## 2026-06-03 01:48 CST - Control Tower Integrates Three Active Transition Results

Round goal:

- Maintain main-thread control while three scoped workers advance
  `T-PROVIDER-002b`, `T-HALF-DUPLEX-001`, and `T-FW-003`.
- Keep changes limited to governance/planning/state docs and generated/ignored
  evidence; do not modify business code, flash firmware, write NVS, or change
  global proxy settings.

Actual completed work:

- Confirmed current branch
  `codex/a21-hardware-window-20260602-stackchan-prd` at baseline commit
  `b5a405e feat(audio): add hot-pluggable iflytek tts candidate`.
- Added transition plan docs:
  - `docs/plans/2026-06-03-provider-tts-live-chain-unblock.md`;
  - `docs/plans/2026-06-03-normal-dialogue-half-duplex-acceptance.md`;
  - `docs/plans/2026-06-03-guarded-wake-flash-physical-proof.md`.
- Dispatched Provider worker `019e8954-e65c-72a2-9847-a17a59a0ad6b`,
  Half-duplex worker `019e8956-7c31-78e3-bd96-f04b94f52d0b`, and Wake worker
  `019e8956-9692-7ae1-bc14-99a311b1c080`.
- Main-thread Gateway probe confirmed `http://127.0.0.1:21081/healthz` is ok.
- Main-thread device probe confirmed physical device `44:1b:f6:e2:6a:60` is
  registered but stale on Gateway `21081`.
- Updated `docs/project_state_machine.md` so active transitions now accurately
  reflect:
  - provider/TTS relay chain passed on 5080 but physical playback is stale
    blocked;
  - half-duplex is blocked by stale device and missing diagnostic mic-probe
    capability;
  - wake package integrity passed but guarded flash is blocked by missing
    reviewed A21 build dir.

Files changed this round:

- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`
- `docs/plans/2026-06-03-provider-tts-live-chain-unblock.md`
- `docs/plans/2026-06-03-normal-dialogue-half-duplex-acceptance.md`
- `docs/plans/2026-06-03-guarded-wake-flash-physical-proof.md`
- Ignored/generated provider candidate reports and WAVs under
  `reports/provider-tts-candidate/`
- Ignored/generated readiness report
  `reports/a21-product-readiness-20260603-011911.json`

Current unfinished items:

- Physical StackChan must reconnect fresh before provider playback and
  half-duplex acceptance can continue.
- The relay-produced StepFun+Iflytek WAV has not yet been played through the
  live stock Xiaozhi path.
- Normal dialogue half-duplex remains unaccepted.
- Custom wake remains unflashed and unaccepted because full guarded flash inputs
  are missing.
- Selected-provider readiness/bundle refresh should wait until the chosen
  provider/TTS physical playback result is known.

Known risks and blockers:

- Do not copy 5080 plaintext credentials into repo docs, logs, reports, shell
  snippets, or final messages.
- Candidate WAV files are host/relay evidence only until physical playback and
  operator listening acceptance happen.
- A stale device registry must not be treated as valid physical acceptance.
- Wake package integrity is not enough for product readiness or flash execution.

Recommended next action:

- Execute `T-DEVICE-REFRESH-001`: foreground wake/touch/reboot StackChan until
  `/v1/devices` on Gateway `21081` shows `connection_status=online`.
- Then execute `T-PROVIDER-PLAYBACK-001`: play
  `reports/provider-tts-candidate/a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`
  through the stock Xiaozhi path and collect operator listening acceptance.
- In parallel, execute `T-WAKE-FLASH-GATE-001`: restore/provide the reviewed
  A21 ESP-IDF wake build dir and run a no-write flash plan before any guarded
  foreground flash.

Validation results:

- Worker Provider: 5080 relay Iflytek TTS smoke passed; StepFun+Iflytek relay
  chain passed; JSON redaction scan passed; Gateway health passed.
- Worker Half-duplex: Gateway health passed; device probe passed but stale;
  `stackchan-half-duplex-acceptance` correctly not run.
- Worker Wake: package hash/size checks passed; wake-related Go tests passed;
  readiness stayed red as expected.

## 2026-06-03 - T-PROVIDER-PLAYBACK-001 Stock Xiaozhi Relay WAV Playback Host Support

Round goal:

- Execute Worker PLAYBACK for `T-PROVIDER-PLAYBACK-001`: TDD-implement
  `/v1/xiaozhi/say` optional `wav_path` support so a local A21-compatible
  16 kHz mono WAV can be delivered through the same stock TTS lifecycle and
  Opus downlink path as text say.

Actual completed work:

- Read `AGENTS.md`, `docs/project_state_machine.md`,
  `docs/agent_handoff_log.md`, and
  `docs/plans/2026-06-03-stock-xiaozhi-relay-wav-playback.md`.
- Confirmed checkout branch `codex/a21-hardware-window-20260602-stackchan-prd`
  at baseline `6eb8062`.
- Added the focused Gateway RED test first. It failed as expected because the
  pre-change handler ignored `wav_path` and returned `400: text is required`.
- Added `wav_path` to `XiaozhiSayRequest`, requiring exactly one playable
  source: `text` or `wav_path`.
- Implemented local 16 kHz mono PCM WAV chunk loading through existing audio
  helpers and fed those chunks into the existing `writeXiaozhiOpusDownlink`
  path.
- Preserved the stock `tts/start`, `tts/sentence_start`, binary Opus downlink,
  `tts/stop`, and post-say input-suppression lifecycle.
- Added basename-only response metadata for WAV playback:
  `audio_source=wav_file` and `audio_basename`; no full local path, raw/base64
  audio, transcript/provider output, credentials, or proxy values are returned.
- Updated the protocol and state-machine docs for the new host support.
- Probed Gateway `21081`: health passed and device `44:1b:f6:e2:6a:60` was
  online/fresh.
- Sent a gated foreground `/v1/xiaozhi/say` `wav_path` request to the live
  Gateway using relay WAV basename
  `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`. The live process
  returned `400: text is required`, proving it was still running the old
  text-only handler; no audio was delivered.

Files changed this round:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current unfinished items:

- The live Gateway on `21081` must be restarted or redeployed from this branch
  before physical relay WAV playback can actually deliver audio.
- Operator listening acceptance or rejection remains missing.

Known risks and blockers:

- Do not expose relay WAV full paths, raw/base64 audio, transcript/provider
  output, credentials, proxy values, or other local path values beyond safe
  basenames.
- Host tests prove the transport path only; physical acceptance still requires
  live Gateway code plus operator listening evidence.

Recommended next action:

- Restart/deploy Gateway `21081` from the branch containing this `wav_path`
  support, confirm `/v1/devices` shows `44:1b:f6:e2:6a:60` online/fresh, then
  resend `/v1/xiaozhi/say` with the relay WAV and capture operator feedback.

Validation results:

- RED: `go test ./internal/gateway -run TestXiaozhiSayDeliversWAVAsStockTTSDownlink -count=1`
  failed with `say wav status = 400: text is required`.
- GREEN: `go test ./internal/gateway -run TestXiaozhiSayDeliversWAVAsStockTTSDownlink -count=1`
  passed.
- Regression:
  `go test ./internal/gateway -run 'TestXiaozhiSay(DeliversWAVAsStockTTSDownlink|DeliversTextAsStockTTSDownlink|SuppressesImmediateListenRestartForStockPhysical)' -count=1`
  passed.
- Package: `go test ./internal/gateway -count=1` passed.
- Repo: `make verify` passed, including `go test ./...` and
  `git diff --check`.
- Physical playback: gated request attempted only after online/fresh device
  proof; no physical audio was delivered because the live Gateway rejected
  `wav_path` before starting a turn.
- Live rejection trace query returned `event_count=0`, confirming no delivery
  trace was produced by the old handler.

## 2026-06-03 01:48 CST - T-WAKE-FLASH-GATE-001 Build Dir Search Remains Blocked By Provenance

Round goal:

- Run Worker WAKE-BUILDDIR as a read-only search for the reviewed A21 ESP-IDF
  build directory or enough provenance to safely run a no-write wake flash plan
  for artifact SHA
  `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`.

Actual completed work:

- Searched A21 workspace, `reports`, `.a21-run`, `dist`, `firmware`,
  `/Users/jiyurun/.codex/worktrees`, `/private/tmp`, `/tmp`, user Desktop,
  Downloads, and adjacent document folders.
- Found the only full build-shaped matching dir:
  `/private/tmp/a21-xiaozhi-wake-build/xiaozhi-esp32/build`.
- Verified `xiaozhi.bin` in that scratch dir exactly matches SHA
  `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b` and
  matches the package artifact by `cmp`.
- Verified required flash-shaped files exist in the scratch dir, including
  `flash_args`, `flasher_args.json`, `config/sdkconfig.json`, `xiaozhi.bin`,
  bootloader, partition table, OTA data, generated assets, review JSON, and
  receipt JSON.
- Verified config/assets contain the intended custom wake payload:
  CoreS3, `USE_CUSTOM_WAKE_WORD=true`, command `xiao a er yi`, display
  `小阿二一`, and threshold `35` / `0.35`.

Files changed this round:

- None by Worker WAKE-BUILDDIR.

Current unfinished items:

- No safe no-write flash plan was run from this dir.
- The matching dir is reviewed for the no-hardware package lane, but it is not
  valid A21-owned flash input under current governance because provenance shows
  it was copied from the frozen X21 source tree.

Known risks and blockers:

- Using the scratch X21-derived build dir mechanically would bypass the current
  A21-owned flash-input rule.
- A21 workspace still lacks a durable reviewed full build dir for the wake
  artifact.
- Product readiness must remain red until guarded flash and physical proof.

Recommended next action:

- Rebuild or restore the wake firmware from an A21-owned/reviewed source path,
  then run no-write `xiaozhi-firmware-flash-plan`.
- Do not execute wake flash from `/private/tmp/a21-xiaozhi-wake-build/...`
  unless the governance rule is explicitly changed by the operator.

Validation results:

- `find`, `rg`, `shasum -a 256`, `wc -c`, `cmp -s`, `strings`, `realpath`,
  `stat`, and USB port listing were run read-only.
- No edits, no flash, no NVS write, no provider/V21/runtime execution.

## 2026-06-03 01:55 CST - T-PROVIDER-PLAYBACK-001 Physical Relay WAV Delivered, Half-Duplex Blocked

Round goal:

- Integrate Worker PLAYBACK implementation, restart live Gateway with the new
  stock Xiaozhi `wav_path` support, physically play the 5080 relay
  StepFun+Iflytek WAV, and immediately probe half-duplex while preserving
  launch-grade honesty.

Actual completed work:

- Main-thread review found Worker PLAYBACK stayed in scope:
  - code changes only in Gateway stock say path;
  - no firmware, NVS, provider adapter, global proxy, wake, or half-duplex
    implementation changes;
  - `wav_path` response metadata is basename-only.
- Focused Gateway regression passed locally after worker return.
- `make verify` passed after the implementation.
- Restarted tmux Gateway session `a21-gateway-21081` from this working tree
  with the same host-local parameters and direct `NO_PROXY` coverage.
- Confirmed `http://127.0.0.1:21081/healthz` returned ok after restart.
- Waited for physical device `44:1b:f6:e2:6a:60` to reconnect; `/v1/devices`
  showed it `online` with fresh `device_age_ms`.
- Delivered runtime speaker volume `100` over stock MCP:
  - trace `a21-trace-relay-playback-volume-1780450901`;
  - response `delivered_transport=xiaozhi_mcp`;
  - tool `self.audio_speaker.set_volume`.
- Delivered the 5080 relay StepFun+Iflytek WAV through stock `/v1/xiaozhi/say`
  `wav_path`:
  - trace `a21-trace-provider-playback-wav-1780450901`;
  - response `delivered_transport=xiaozhi_ws`;
  - `audio_chunks=40`;
  - `audio_source=wav_file`;
  - `audio_basename=a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`;
  - no full WAV path in the response.
- Trace summary for `a21-trace-provider-playback-wav-1780450901` recorded:
  - `event_count=452`;
  - `xiaozhi.say.start=1`;
  - `tts.first_audio=1`;
  - `audio.downlink.first_frame=1`;
  - `xiaozhi.say.downlink=1`;
  - `xiaozhi.tts.opus_frame.downlink=40`;
  - `xiaozhi.say.input_suppression_armed=1`;
  - `xiaozhi.say.delivered=1`.
- Operator listening feedback immediately after playback: "好多了".
- Ran `A21_GATEWAY_URL=http://127.0.0.1:21081 A21_DEVICE_ID=44:1b:f6:e2:6a:60 make stackchan-half-duplex-acceptance`.
- Half-duplex report
  `reports/a21-stackchan-half-duplex-acceptance-20260603-014233.json`
  was generated and correctly blocked with
  `half_duplex_acceptance_status=blocked`.

Files changed this round:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`
- `docs/plans/2026-06-03-stock-xiaozhi-relay-wav-playback.md`
- Ignored/generated report:
  `reports/a21-stackchan-half-duplex-acceptance-20260603-014233.json`

Current unfinished items:

- Relay WAV has positive physical listening feedback, but a longer normal
  dialogue run is still needed before full conversation acceptance.
- Instrumented half-duplex remains blocked by current stock firmware
  capabilities and missing runtime echo counters.
- Wake remains blocked by missing A21-owned reviewed build dir and physical
  custom wake proof.
- Selected-provider readiness/bundle still needs a pinned refresh that does not
  fall back to `mock`.

Known risks and blockers:

- The live Gateway is now running from the dirty working tree until this round
  is committed.
- `reports/a21-stackchan-half-duplex-acceptance-20260603-014233.json` proves
  the half-duplex gate is blocked, not accepted. Findings include
  `device_identity_not_ok`, firmware id/board/commit mismatch,
  `microphone_not_diagnostic_probe`, `speaker_not_available`, and missing
  runtime echo fields.
- Operator "好多了" is strong positive voice feedback, but not full PRD launch
  acceptance.

Recommended next action:

- Commit this host playback support and state update.
- Execute `T-PROVIDER-001b`: refresh selected-provider/server-side readiness
  using the StepFun/Iflytek relay evidence without leaking keys or promoting
  mock evidence.
- Decide `T-HALF-DUPLEX-002`: no-flash normal-dialogue observation versus a
  separate guarded diagnostic-capability firmware plan.
- Rebuild/restore an A21-owned wake firmware build dir before any wake flash
  plan.

Validation results:

- `go test ./internal/gateway -run 'TestXiaozhiSayDeliversWAVAsStockTTSDownlink|TestXiaozhiSayDeliversTextAsStockTTSDownlink|TestXiaozhiSaySuppressesImmediateListenRestartForStockPhysical' -count=1`:
  passed.
- `git diff --check`: passed before this log update.
- `make verify`: passed before live Gateway restart.
- Live Gateway restart: passed; `healthz` ok.
- Live relay WAV playback: delivered as above.
- Half-duplex acceptance command: exited nonzero with blocked report as
  expected; no flash or NVS write occurred.

## 2026-06-03 01:52 CST - T-PROVIDER-001b Closed, Half-Duplex/Wake Gates Preserved

Round goal:

- Continue the three active control-tower transitions quickly but safely:
  selected provider readiness refresh, normal dialogue half-duplex, and custom
  wake flash gate.

Actual completed work:

- Spawned three scoped workers with disjoint responsibilities and explicit
  no-flash/no-NVS/no-secret/no-global-proxy boundaries:
  - provider worker `019e8974-346e-7912-93b2-77cdbb9f3acf`;
  - half-duplex worker `019e8974-6474-7333-8a78-3e9cf2ea1884`;
  - wake gate worker `019e8974-9784-7c11-be91-cbc234a0a6dd`.
- Provider worker found the best current voice-chain candidate evidence:
  - Iflytek TTS relay report
    `reports/provider-tts-candidate/a21-local-tts-smoke-5080-relay-20260603-0125.json`
    passed with first audio `100.299 ms`;
  - StepFun+Iflytek relay chain
    `reports/provider-tts-candidate/a21-local-voice-loopback-5080-relay-20260603-0128.json`
    passed with text first content `229.077 ms` and TTS first audio
    `87.947 ms`;
  - StepFun direct smoke
    `reports/provider-tts-candidate/a21-provider-smoke-20260603-010652-957877000.json`
    passed but remains `route_eligible=false`, so it was not promoted into
    product provider readiness.
- Provider worker refreshed selected-provider readiness using route-eligible
  DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json`
  instead of letting the latest selector fall back to `mock`.
- New ignored provider reports:
  - `reports/provider-tts-candidate/a21-product-readiness-20260603-015100.json`
    has `provider.selected=deepseek`, `provider.real_provider_ready=true`,
    `provider.smoke_status=passed`, and `status=server_side_blocked`;
  - `reports/provider-tts-candidate/a21-server-side-readiness-bundle-20260603-015102.json`
    has `provider.ready=true` and `status=server_side_blocked`.
- Provider redaction check found no API key, secret, auth query, raw
  transcript, base64 audio, or full local path in JSON string values.
- Half-duplex worker checked Gateway and device state:
  - `http://127.0.0.1:21081/healthz` is healthy;
  - device `44:1b:f6:e2:6a:60` was `connection_status=stale`, so no new
    no-flash half-duplex command was forced.
- Final main-thread status check later found device `44:1b:f6:e2:6a:60`
  `connection_status=online`, so the main thread immediately ran the no-flash
  half-duplex command.
- New ignored half-duplex report:
  `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json` is
  `half_duplex_acceptance_status=blocked`, `dry_run=true`,
  `flash_allowed=false`, and `physical_sound_observed=false`.
- Half-duplex remains blocked by the prior report
  `reports/a21-stackchan-half-duplex-acceptance-20260603-014233.json`, the new
  online report above, and current stock firmware lacking diagnostic identity,
  mic-probe capability, available speaker echo fields, and runtime echo
  counters.
- Wake gate worker rechecked package integrity:
  - package/report/artifact/manifest/sha agree;
  - SHA-256 remains
    `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`;
  - intent is `小阿二一` / `xiao a er yi`, threshold `35`;
  - package remains below activation with `product_ready=false`,
    `flash_allowed=false`, and `flash_executed=false`.
- Wake gate remains blocked because no A21-owned reviewed full ESP-IDF build
  directory with full flash inputs was found; no no-write flash plan was run.
- Updated `docs/project_state_machine.md` to move `T-PROVIDER-001b` to
  completed and keep half-duplex/wake as explicit blocked transitions.

Files changed this round:

- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`
- Ignored/generated provider reports:
  - `reports/provider-tts-candidate/a21-product-readiness-20260603-015100.json`
  - `reports/provider-tts-candidate/a21-server-side-readiness-bundle-20260603-015102.json`
- Ignored/generated half-duplex report:
  - `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json`

Current unfinished items:

- Normal dialogue half-duplex still needs an operator-observed no-flash
  dialogue run; the instrumented counter gate is blocked even when the device
  is online because stock firmware lacks the required diagnostic fields.
- If machine-verifiable mic/playback counters are required, diagnostic
  capability firmware needs its own guarded plan.
- Custom wake still needs an A21-owned reviewed full wake build directory
  before any no-write flash plan.
- StepFun+Iflytek relay is the current best voice-chain candidate, but product
  readiness does not yet ingest it as a selected voice-chain evidence surface.

Known risks and blockers:

- Full PRD remains blocked; do not treat selected-provider readiness as launch
  acceptance.
- StepFun is still a compatibility candidate, not route-eligible product
  provider readiness evidence.
- The physical device was stale during the worker check but came online during
  final main-thread status; the online no-flash run still blocked on
  non-diagnostic stock firmware fields.
- Wake package integrity alone is not flash or product readiness.

Recommended next action:

- `T-HALF-DUPLEX-002`: collect no-flash normal dialogue observation first;
  only open diagnostic firmware if machine-verifiable counters are required.
- `T-WAKE-FLASH-GATE-001`: restore or rebuild the reviewed A21-owned wake
  ESP-IDF build dir, then run no-write flash plan only.
- `T-VOICE-CHAIN-EVIDENCE-001`: add or reuse a redacted selected voice-chain
  evidence ingress so StepFun+Iflytek relay evidence can close the correct
  server-side gap without changing provider route eligibility.

Validation results:

- `git status --short --branch` was clean before doc edits.
- Worker checks were read-only except for ignored readiness report generation.
- `A21_GATEWAY_URL=http://127.0.0.1:21081 A21_DEVICE_ID=44:1b:f6:e2:6a:60 make stackchan-half-duplex-acceptance`:
  exited nonzero with blocked report
  `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json`, as
  expected for non-diagnostic stock firmware; no flash was performed.
- No tracked code was modified.
- No flash, NVS write, global proxy change, provider secret output, or V21
  execution occurred.

## 2026-06-03 02:09 CST - T-HALF-DUPLEX-002 No-Flash Observation Passed, 5080 Clone Check Routed

Round goal:

- Keep convergence fast without letting "no firmware flash" or "operator did
  not click" become a hard blocker.
- Run no-flash normal dialogue self-trigger observation while the device is
  online.
- Dispatch a parallel 5080 worker to check whether local CosyVoice or another
  clone-capable TTS path already exists and can produce a smoke WAV.

Actual completed work:

- Spawned 5080/CosyVoice worker `019e897d-d4b4-78c3-9358-ac27a4f61d0d` with
  strict no-repo-edit, no-secret, no-hardware, no-global-proxy boundaries.
- Ran no-flash normal dialogue observation from the main thread:
  - Gateway `http://127.0.0.1:21081` was healthy.
  - Device `44:1b:f6:e2:6a:60` was online.
  - Runtime speaker volume `100` was delivered before playback.
  - Accepted relay WAV
    `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav` was played via
    stock `/v1/xiaozhi/say`.
  - Trace: `a21-trace-no-flash-dialogue-observe-1780423245`.
  - Session: `a21-session-no-flash-dialogue-observe-1780423245`.
  - Playback delivered `40` audio chunks.
  - Post-say observation window was extended to `20000 ms`.
  - Trace event count reached `869`.
  - Self-trigger event names were empty for `xiaozhi.listen.start`,
    `provider.start_turn.start`, `provider.realtime_session.start`, and
    `xiaozhi.voice_pipeline.start`.
  - `xiaozhi.listen.start.input_suppressed=1` was observed.
- Generated ignored no-flash report
  `reports/a21-no-flash-normal-dialogue-observation-20260603-020055.json` with
  `status=candidate_passed_no_self_trigger`.
- Redaction scan of the no-flash report found no API key, secret,
  Authorization, base64 audio, raw audio, transcript, provider output, full
  local path, or non-loopback URL.
- Updated the half-duplex plan to separate the contest no-flash observation
  track from the optional diagnostic-counter firmware path.
- Ran a fresh product readiness sweep:
  `reports/a21-product-readiness-20260603-020743.json`.
  It remains `server_side_blocked`; because this quick sweep did not pin the
  provider report/env it selected `mock`, so use it only as a gap inventory.
- 5080/CosyVoice worker result:
  - 5080 LAN SSH is online; `D:/a21-mainland-latency-lab`, `inbox`, and
    `outbox` exist.
  - CosyVoice source and venv exist, but the CosyVoice venv lacks `torch` and
    `tqdm`, and no usable `pretrained_models/CosyVoice-*` weights were found.
  - CosyVoice classes can be imported from the IndexTTS venv and CUDA is
    available there, but no CosyVoice weights are ready for generation.
  - IndexTTS2 source, venv, runner, CUDA, and partial checkpoints exist, but
    inference fails because `checkpoints/qwen0.6bemo4-merge/` is missing or not
    loadable.
  - F5-TTS and GPT-SoVITS source traces exist, but no ready checkpoint/run path
    was confirmed.
  - No clone WAV was produced.
- Created plan
  `docs/plans/2026-06-03-cosyvoice-5080-local-clone-candidate.md` before any
  model restoration/download task.
- Updated `docs/project_state_machine.md` and
  `docs/plans/2026-06-03-provider-tts-real-dialogue-acceptance.md` with the
  5080 clone status.

Files changed this round:

- `docs/plans/2026-06-03-normal-dialogue-half-duplex-acceptance.md`
- `docs/plans/2026-06-03-provider-tts-real-dialogue-acceptance.md`
- `docs/plans/2026-06-03-cosyvoice-5080-local-clone-candidate.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`
- Ignored/generated reports:
  - `reports/a21-no-flash-normal-dialogue-observation-20260603-020055.json`
  - `reports/a21-product-readiness-20260603-020743.json`

Current unfinished items:

- No-flash self-trigger observation is candidate-passed, but touch/barge-in
  operator proof is still pending before full PRD physical acceptance.
- Instrumented half-duplex counter acceptance remains optional/diagnostic and
  would require guarded diagnostic firmware if machine counters are mandatory.
- Wake remains blocked by missing A21-owned reviewed full build dir and
  physical custom wake proof.
- CosyVoice/IndexTTS2 local clone path is not usable until weights/dependencies
  are restored; StepFun+Iflytek remains the current accepted contest voice
  candidate.
- Product readiness still needs explicit provider pinning when run; generic
  `--use-latest-reports` can select `mock`.

Known risks and blockers:

- Do not promote the no-flash observation to full diagnostic half-duplex
  acceptance.
- Do not start large 5080 model downloads without the CosyVoice recovery plan
  and a redacted source/size summary.
- Do not switch the contest default to clone TTS until a WAV is generated and
  operator listening accepts it.

Recommended next action:

- Continue with `T-WAKE-FLASH-GATE-001` by restoring/rebuilding the A21-owned
  wake build dir.
- In parallel, run the next 5080 clone recovery worker under
  `docs/plans/2026-06-03-cosyvoice-5080-local-clone-candidate.md`, preferring
  the quickest cached IndexTTS2 `qwen0.6bemo4-merge` restoration.
- Add or reuse a voice-chain evidence ingress so the accepted StepFun+Iflytek
  relay path closes the right readiness gap without changing provider route
  eligibility.

Validation results:

- No-flash normal dialogue observation report generated and redaction-scanned.
- No flash, NVS write, global proxy change, provider secret output, V21
  execution, or StackChan firmware change occurred.
- 5080 worker did not modify tracked A21 files.

## 2026-06-03 02:24 CST - Bare Wake Flash Incident Rolled Back, StackChan-Compatible App Restored

Round goal:

- Continue hardware convergence after operator explicitly allowed build,
  compile, and flash.
- Try to close custom wake activation without losing the accepted StackChan
  avatar/audio path.
- Preserve control-tower rules and update state from the actual hardware
  result.

Actual completed work:

- Restored the previously reviewed custom wake ESP-IDF flash input into an
  ignored A21 run path and verified its key values:
  - app file `xiaozhi.bin`;
  - app SHA-256
    `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`;
  - phrase `小阿二一` / `xiao a er yi`;
  - threshold `35`.
- Ran generic bare Xiaozhi no-write flash plan:
  `reports/a21-xiaozhi-firmware-flash-20260603-021247-1780423967840224000.json`;
  it reported `status=ready`, `dry_run=true`, `flash_allowed=false`, and
  `flash_executed=false`.
- Executed the generic bare Xiaozhi flash after operator hardware approval:
  `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json`;
  it reported `status=passed`, `flash_allowed=true`, and
  `flash_executed=true`.
- Operator immediately reported the device returned to the plain Xiaozhi UI.
  Root cause: the flashed app part was `xiaozhi.bin`, not the StackChan product
  app `a21-stackchan-official-xiaozhi-compatible.bin`. This was the wrong
  product lane even though the generic hardware guard passed.
- Located the correct StackChan-compatible product candidate in
  `/tmp/a21-stackchan-official-build`:
  - app file `a21-stackchan-official-xiaozhi-compatible.bin`;
  - app SHA-256
    `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`;
  - flash app offset `0x20000`.
- Ran correct StackChan-compatible no-write flash plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021802-1780424282862523000.json`;
  it reported `status=ready`, `firmware_candidate=a21-stackchan-official-xiaozhi-compatible`,
  `dry_run=true`, `flash_allowed=false`, and `flash_executed=false`.
- Executed the corrective StackChan-compatible flash:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`;
  it reported `status=passed`, `flash_allowed=true`, and
  `flash_executed=true`.
- Verified post-restore Gateway/device state:
  - Gateway `http://127.0.0.1:21081/healthz` returned ok.
  - Device `44:1b:f6:e2:6a:60` reconnected with a fresh `xiaozhi.hello`.
  - Runtime speaker volume `100` delivered through stock MCP on trace
    `a21-trace-recovery-stackchan-volume-1780424386012`.
  - Accepted 5080 StepFun+Iflytek relay WAV
    `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav` delivered through
    stock `/v1/xiaozhi/say` on trace
    `a21-trace-recovery-stackchan-relay-wav-1780424386012` with
    `audio_chunks=40`.
- Spawned read-only sidecar `019e898e-5c43-7050-9cff-1ecb6f646e2d` to review
  the wake/flash lane boundary. It confirmed:
  - product StackChan app flashes must use
    `a21-stackchan-official-xiaozhi-compatible-flash-execute`;
  - schema must be `a21.stackchan.official_xiaozhi_compatible_flash_execution.v1`;
  - app file must be `a21-stackchan-official-xiaozhi-compatible.bin`;
  - generic `xiaozhi-firmware-flash-execute` / `xiaozhi.bin` must be rejected
    for product StackChan devices.
- Spawned read-only sidecar `019e898e-75f1-7a81-9969-70de7112271f` to check
  CosyVoice/clone TTS. It confirmed:
  - A21 already has the `voice_clone_cli` seam;
  - an old IndexTTS2 smoke passed but was too slow at about `23899 ms` first
    audio;
  - no ready CosyVoice weights or `qwen0.6bemo4-merge` IndexTTS2 checkpoint
    were found on the Mac/known pullback paths;
  - StepFun+Iflytek remains the current accepted contest voice candidate while
    clone weights are restored in parallel.
- Spawned code worker `019e8993-0987-7601-9b8a-3aa4e88ebfaa` for
  `T-FLASH-GUARD-001`, scoped to a small artifact-lane guard and focused tests;
  no hardware writes were allowed in that worker.
- Integrated `T-FLASH-GUARD-001`:
  - generic `xiaozhi-firmware-flash-*` now rejects product-looking
    `xiaozhi.bin` at app offset `0x20000` unless the command explicitly passes
    `--non-product-dev`;
  - the rejection points to
    `a21-stackchan-official-xiaozhi-compatible-flash-execute` for the product
    StackChan app;
  - allowed generic dev receipts include
    `"build_lane_role": "non_product_dev"`;
  - official StackChan-compatible flash commands are unchanged.
- Proved the guard against the real incident build:
  - `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 A21_XIAOZHI_FIRMWARE_BUILD_DIR=... make xiaozhi-firmware-flash-plan`
    now exits nonzero before any write with
    `xiaozhi-firmware-flash is not a product StackChan flash lane for xiaozhi.bin at 0x20000`;
  - `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 make a21-stackchan-official-xiaozhi-compatible-flash-plan`
    still passes and wrote
    `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-023008-1780425008542994000.json`.
- Updated:
  - `AGENTS.md`;
  - `internal/app/xiaozhi_firmware.go`;
  - `internal/app/official_stackchan_test.go`;
  - `docs/project_state_machine.md`;
  - `docs/plans/2026-06-03-guarded-wake-flash-physical-proof.md`;
  - `docs/plans/2026-06-03-flash-lane-identity-guard.md`;
  - this handoff log.

Files changed this round:

- `AGENTS.md`
- `internal/app/xiaozhi_firmware.go`
- `internal/app/official_stackchan_test.go`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-03-guarded-wake-flash-physical-proof.md`
- `docs/plans/2026-06-03-flash-lane-identity-guard.md`
- `docs/agent_handoff_log.md`
- Ignored/runtime flash inputs under `.a21-run/wake-word/`
- Ignored/generated flash reports under `reports/`

Current unfinished items:

- Custom wake is not product-ready. The bare wake artifact is evidence only and
  must not be flashed again onto the product StackChan device.
- Custom wake must be rebuilt or overlaid into
  `a21-stackchan-official-xiaozhi-compatible.bin`, with official avatar/action
  and Xiaozhi runtime preservation proven before no-write flash.
- Touch/barge-in operator proof and final physical evidence regeneration remain
  open for full PRD acceptance.
- Clone-capable local TTS is not ready; keep StepFun+Iflytek as current contest
  voice path until a redacted clone smoke and listening acceptance pass.

Known risks and blockers:

- Do not treat
  `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json`
  as a successful product flash; it is an incident report.
- Do not run another wake flash unless the app file is
  `a21-stackchan-official-xiaozhi-compatible.bin`.

Recommended next action:

- Start `T-WAKE-INTEGRATE-001` as a worker task: port custom wake assets into
  the StackChan-compatible app lane and produce an official-compatible no-write
  flash plan only.
- Continue `T-VOICE-CHAIN-EVIDENCE-001` so the accepted StepFun+Iflytek relay
  chain is represented in the correct readiness surface without replacing
  route-eligible provider evidence.

Validation results:

- Generic bare Xiaozhi flash executed and is classified as incident evidence.
- Corrective StackChan-compatible flash executed successfully and restored the
  product app.
- Post-restore Gateway health, device fresh reconnect, runtime volume `100`,
  and relay WAV playback all passed.
- Focused tests passed:
  `go test ./internal/app -run 'TestRunXiaozhiFirmwareFlash|TestRunStackChanOfficialXiaozhiCompatibleFlash|TestCollectOfficialStackChanBuildArtifactsFindsXiaozhiCompatibleAppFromFlashArgs|TestRunWakeWordFirmware(BuildReceipt|Package)' -count=1`.
- `git diff --check` passed.
- `make verify` passed.
- No NVS write, global proxy change, provider secret output, or V21 execution
  occurred in this recovery round.

## 2026-06-03 - T-WAKE-002 - Zi Yue Wake Built In StackChan-Compatible Lane

Goal:

- Continue the wake convergence without repeating the bare `xiaozhi.bin`
  product regression.
- Change the requested wake word to `紫悦`.
- Build the custom wake into the official StackChan-compatible product app lane
  and prepare a guarded product flash.
- Launch a separate comparison thread for raw Xiaozhi/StackChan hardware and
  audio-behavior parity research.

Actual completed work before flash:

- Created plan
  `docs/plans/2026-06-03-zi-yue-wake-stackchan-compatible.md`.
- Launched and received a separate read-only background comparison thread for:
  - current A21 versus raw `xiaozhi.bin` protocol/audio handling;
  - raw Xiaozhi whole-device behavior versus A21 StackChan behavior;
  - official StackChan docs/source and StackChan-for-Xiaozhi implementation
    cross-check.
- Recorded the comparison result as state evidence: bare `xiaozhi.bin` being
  louder/clearer is treated as an operator-confirmed fact, but the artifact
  remains non-product incident evidence. The migration direction is official
  CoreS3 codec/HAL parity, source TTS/mastering, runtime volume/NVS/MCP
  evidence, downlink Opus/pacer parity, and official StackChan app/action
  initialization inside the compatible product lane.
- Added `紫悦` wake config to
  `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
  without leaving the product lane:
  - `CONFIG_USE_CUSTOM_WAKE_WORD=y`;
  - `CONFIG_CUSTOM_WAKE_WORD="zi yue"`;
  - `CONFIG_CUSTOM_WAKE_WORD_DISPLAY="紫悦"`;
  - `CONFIG_CUSTOM_WAKE_WORD_THRESHOLD=20`;
  - `CONFIG_SR_MN_CN_MULTINET7_QUANT=y`;
  - `CONFIG_SEND_WAKE_WORD_DATA=n`;
  - `# CONFIG_USE_AFE_WAKE_WORD is not set`;
  - `# CONFIG_SR_WN_WN9_HISTACKCHAN_TTS3 is not set`.
- Tried the sidecar-audit suggestion to move autostart closer to the official
  app flow, then recorded the physical regression: the device got stuck on the
  welcome/setup screen and Skip/Start were ineffective.
- Applied the contest recovery hotfix: keep `紫悦` custom wake, but start
  Xiaozhi directly before the welcome/setup flow. Retaining official app loading
  without showing welcome/setup is now a separate transition, not part of this
  recovery flash.
- Added a focused app test that asserts the official-compatible overlay keeps
  the StackChan product identity and does not point to the bare Xiaozhi app
  lane.
- Updated `docs/project_state_machine.md` from bare-wake recovery toward the
  `紫悦` product-lane flash-ready state.

Files changed before flash:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/plans/2026-06-03-zi-yue-wake-stackchan-compatible.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current unfinished items:

- The `紫悦` build is not yet physical-wake accepted until the guarded flash is
  executed and the operator confirms saying `紫悦` wakes the StackChan UI
  without touching the screen.
- The separate raw Xiaozhi/StackChan comparison thread returned evidence. Its
  immediate follow-up should be a new plan for
  `T-AUDIO-BARE-XIAOZHI-PARITY-001`, not an unplanned audio rewrite.
- Touch/barge-in operator proof and final physical evidence regeneration remain
  open for full PRD acceptance.

Known risks and blockers:

- `zi yue` is short; threshold `20` improves wake sensitivity but may false
  wake. If false wakes appear, keep the product lane and tune only threshold,
  first toward `35`.
- First ESP-IDF build attempt failed at Python 3.13 `_csv` dynamic-library load
  during `gen_crt_bundle.py`; manual replay and second build passed. Treat as a
  local toolchain hiccup unless it repeats.
- Do not flash `xiaozhi.bin` as product StackChan firmware. Product flash must
  remain `a21-stackchan-official-xiaozhi-compatible-flash-*`.

Recommended next action:

- Commit the `紫悦` product-lane build change.
- Execute the guarded official-compatible product flash on
  `/dev/cu.usbmodem1101`.
- After reboot, verify StackChan UI is still present and ask the operator to
  try `紫悦`.

Validation results before flash:

- `go test ./internal/app -run 'OfficialXiaozhiCompatibleOverlay|XiaozhiFirmwareFlash|OfficialXiaozhiCompatibleFlash' -count=1`:
  passed.
- `git diff --check`: passed.
- `make verify`: passed.
- First
  `make a21-stackchan-official-xiaozhi-compatible-build`: failed at local
  Python `_csv` dynamic-library system policy in `gen_crt_bundle.py`; overlay
  had already applied and `sdkconfig.json` showed the expected wake config.
- Second
  `make a21-stackchan-official-xiaozhi-compatible-build`: passed but was
  superseded by the autostart-order correction.
- Final
  `make a21-stackchan-official-xiaozhi-compatible-build` after the autostart
  correction: passed.
- Build report:
  `reports/a21-stackchan-official-baseline-20260603-030428-1780427068064697000.json`.
- Product app artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Product app SHA-256:
  `3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`.
- Build config inspection passed for StackChan board identity, `紫悦` custom
  wake, MultiNet7, disabled AFE WakeNet, and disabled HiStackChan WakeNet.
- Patched `main.cpp` inspection passed: A21 sets codec volume and starts
  Xiaozhi before the welcome/setup flow.
- No-write flash plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030447-1780427087258380000.json`
  is `status=ready`, `dry_run=true`, `flash_allowed=false`,
  `flash_executed=false`, app `a21-stackchan-official-xiaozhi-compatible.bin`,
  offset `0x20000`, port `/dev/cu.usbmodem1101`.

Actual completed work after flash:

- A first execute attempt after the welcome-screen hotfix was correctly blocked
  by the T7 guard because the worktree had two dirty files.
- Committed the recovery hotfix as
  `7f3225e fix(firmware): bypass stackchan welcome setup`.
- Re-ran guarded official-compatible product flash after the worktree was clean.
- Flash execution passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030818-1780427298506934000.json`.
- Flash details:
  - port `/dev/cu.usbmodem1101`;
  - app `a21-stackchan-official-xiaozhi-compatible.bin`;
  - app offset `0x20000`;
  - app SHA-256
    `3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`;
  - control commit `7f3225ee1fe7`;
  - `flash_allowed=true`;
  - `flash_executed=true`.
- Gateway `127.0.0.1:21081` stayed healthy after flash.
- Device `44:1b:f6:e2:6a:60` reconnected online after flash; polling observed
  `online` at 03:08:51 with trace `a21-trace-44-1b-f6-e2-6a-60` and speaker
  volume `100`.

Current unfinished items after flash:

- Operator must confirm the screen is no longer stuck on
  "Welcome! Let's get started".
- Operator must say `紫悦` and report whether the device wakes without screen
  touch.
- Official app loading / professional-mode frontend parity should be handled in
  a separate planned transition:
  `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001`.

Failure location and reason:

- The only build failure in this round was local ESP-IDF Python 3.13 importing
  `_csv` under ninja for certificate bundle generation. Manual command replay
  and rerun succeeded, so no firmware code rollback was needed.
- The request-start autostart variant physically regressed to the official
  welcome/setup screen with ineffective Skip/Start buttons. It was superseded
  by direct Xiaozhi autostart before product recovery flash.

## 2026-06-03 - T-WAKE-003 / T-ASR-GREEN-LATENCY-001 - Wake Rejected And Green Listen Bounded

Goal:

- Preserve the last round's progress after the operator pasted the prior
  status output back into the thread.
- Record that `紫悦` physical wake validation failed instead of keeping it
  pending or green.
- Separate the ASR green-light waiting problem from wake-word acceptance.
- Create the smallest contest-path hotfix candidate for both issues without
  touching provider, V21, NVS, Wi-Fi, or TTS gain.

Actual completed work:

- Pulled live Gateway trace
  `a21-trace-44-1b-f6-e2-6a-60` from `127.0.0.1:21081`.
- Summarized the trace:
  - `event_count=22129`;
  - `xiaozhi.listen.start=19`;
  - `vad.speech.start=13`;
  - `vad.speech.end=13`;
  - `xiaozhi.listen.auto_stop=13`;
  - `xiaozhi.voice_pipeline.start=13`;
  - some listen windows were far too long, including about 25s, 38s, 70s, and
    108s before auto-stop or replacement by another listen.
- Created plan
  `docs/plans/2026-06-03-wake-and-asr-green-latency-recovery.md`.
- Updated the official-compatible firmware overlay wake command list from
  `zi yue` to
  `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue`, while keeping display
  `紫悦`, threshold `20`, custom MultiNet, and the product app lane.
- Added Gateway stock Xiaozhi max-listen safety stop:
  - default `7000 ms`;
  - env override `A21_XIAOZHI_LISTEN_MAX_MS`;
  - active only after speech has been detected;
  - records `xiaozhi.listen.max_duration_auto_stop` then
    `xiaozhi.listen.auto_stop`;
  - starts the normal voice pipeline task rather than inventing a new path.
- Improved trace summary pairing so reused hardware trace ids use the latest
  complete event pair instead of pairing the first old event with a later turn.
- Updated `docs/project_state_machine.md`:
  - total state now records wake failure and ASR latency hotfix candidate;
  - `T-WAKE-002` is rejected, not accepted;
  - `T-WAKE-003-ZI-YUE-PHRASE-TUNING` is active;
  - `T-ASR-GREEN-LATENCY-001-XIAOZHI-LISTEN-AUTO-STOP` is active.

Modified files:

- `docs/plans/2026-06-03-wake-and-asr-green-latency-recovery.md`
- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `internal/app/official_stackchan_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current unfinished items:

- Full `make verify` passed for this round.
- The tuned wake phrase has not been built or flashed yet.
- The live Gateway has not yet been restarted with the max-listen hotfix.
- Physical wake acceptance remains failed until the operator confirms a tuned
  phrase wakes the device without touch.
- Green-light latency remains a candidate fix until the operator retests the
  live device.

Known risks and blockers:

- MultiNet may still not reliably recognize the two-syllable `紫悦`; the longer
  aliases are a pragmatic contest-path improvement, not proof.
- A 7s max listen cap can cut off very long utterances. It is env configurable
  through `A21_XIAOZHI_LISTEN_MAX_MS`.
- The serial diagnostic read path was unreliable on this machine because Python
  lacked `serial` and a Perl read blocked; no serial proof was collected this
  round.
- No new background worker could be spawned initially because the subagent
  thread limit was reached; the main control thread executed the bounded
  changes directly.

Next recommended actions:

1. Commit the hotfix candidate.
2. Build the official-compatible product app and inspect `sdkconfig.json`.
3. Run a no-write official-compatible flash plan, then guarded flash execute
   only from a clean worktree.
4. Restart/deploy the Gateway hotfix or otherwise ensure the live Gateway is
   running this commit before retesting green-light latency.
5. Ask the operator to test `紫悦`, `紫悦紫悦`, `你好紫悦`, and `小紫悦`, and to
   report whether green ASR wait is shorter.

Test/build/run results so far:

- `go test ./internal/gateway -run 'TestTraceEndpointUsesLatestCompletePairForReusedHardwareTrace|TestTraceEndpointReturnsVoicePipelineSplitSummary|TestTraceEndpointUsesASRFinalWhenPartialIsUnavailable|TestXiaozhiWebSocketVADSpeechEndAutoStopsRealtimeTurn|TestXiaozhiWebSocketMaxListenDurationAutoStopsAfterSpeech' -count=1`:
  passed.
- `go test ./internal/app -run 'TestGatewayServerOptionsFromEnvWiresXiaozhiListenMaxDuration|TestGatewayServerOptionsFromEnvWiresSileroVADConfig|TestOfficialXiaozhiCompatibleOverlaySetsZiYueCustomWake|TestOfficialXiaozhiCompatibleOverlayPreservesOfficialStackChanAppSurface' -count=1`:
  passed.
- `git diff --check`: passed.
- `make verify`: passed.

Post-commit build/flash/runtime results:

- Committed as `5242349 fix(voice): bound xiaozhi listen and tune zi yue wake`.
- `make a21-stackchan-official-xiaozhi-compatible-build`: passed.
- Build report:
  `reports/a21-stackchan-official-baseline-20260603-032615-1780428375746120000.json`.
- Product app:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Product app SHA-256:
  `e13a6cbd63596cd1388f5a540b488b3e35b449e49a5e31891cf133436561dc6e`.
- Generated `sdkconfig.json` proves:
  - `BOARD_TYPE_M5STACK_STACK_CHAN=true`;
  - `USE_CUSTOM_WAKE_WORD=true`;
  - `CUSTOM_WAKE_WORD="zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue"`;
  - `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`;
  - `CUSTOM_WAKE_WORD_THRESHOLD=20`;
  - `SR_MN_CN_MULTINET7_QUANT=true`;
  - `USE_AFE_WAKE_WORD=false`;
  - `SR_WN_WN9_HISTACKCHAN_TTS3=false`.
- No-write flash plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-032627-1780428387219884000.json`,
  `status=ready`, app offset `0x20000`, app file
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Guarded flash execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-032734-1780428454747199000.json`,
  `status=passed`, `flash_executed=true`, control commit `5242349a2095`,
  port `/dev/cu.usbmodem1101`.
- Restarted tmux Gateway `a21-gateway-21081` from this commit with
  `A21_XIAOZHI_LISTEN_MAX_MS=7000`; health returned ok.
- Device `44:1b:f6:e2:6a:60` reconnected online after the Gateway restart.
- Delivered runtime speaker volume `100` through stock MCP on trace
  `a21-trace-wake-asr-hotfix-volume-1780428544`.

Current operator validation needed:

- Try wake phrases: `紫悦`, `紫悦紫悦`, `你好紫悦`, `小紫悦`.
- Verify whether the green ASR wait now stops within roughly 7 seconds after
  speech is detected.

## 2026-06-03 - T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001 / T-AUDIO-BARE-XIAOZHI-PARITY-001 - Quiet Socket Candidate

Goal:

- Continue from the operator clarification that wake cannot be physically
  validated before the stock Xiaozhi socket is connected.
- Keep the official StackChan app/hardware preload surface, but bypass the
  visible welcome/setup flow.
- Add graceful not-connected feedback and prevent touch from leaving the device
  stuck in infinite green listening when no speech is detected.
- Keep `T-AUDIO-BARE-XIAOZHI-PARITY-001` moving in a separate bounded worker
  without flashing bare `xiaozhi.bin` or changing accepted audio gain.

Actual completed work:

- Created plan
  `docs/plans/2026-06-03-boot-idle-socket-and-not-connected-ux.md`, with main
  transition `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001` and sub-transition
  `T-BOOT-IDLE-SOCKET-001`.
- Pulled read-only worker result from thread
  `019e89d6-2cde-73d1-8865-5073d46baf66` and integrated its non-conflicting
  operator checklist as
  `docs/testing/stackchan-boot-idle-socket-ux-runbook.md`.
- Rebuilt the official-compatible overlay patch so ordinary `git apply` works
  against the full official StackChan source tree, including the ignored nested
  `firmware/xiaozhi-esp32` source.
- Updated the overlay candidate:
  - install official StackChan apps before the immediate Xiaozhi request;
  - set codec volume `92`;
  - request Xiaozhi immediately with A21 autostart copy instead of direct
    `startXiaozhi()` before app install;
  - add `CONFIG_A21_STACKCHAN_KEEP_CONTROL_CHANNEL=y`;
  - explicitly disable `CONFIG_X21_STACKCHAN_DEVICE_EVENTS`;
  - use A21 names for quiet idle socket state;
  - show `紫悦` connecting/ready copy;
  - keep `protocol_->OpenAudioChannel()` as quiet idle preconnect;
  - add `A21_NO_SPEECH_LISTENING_TIMEOUT_MS=7000` so touch-started no-speech
    listening stops.
- Updated focused app tests for the new app-preload/no-welcome contract, A21
  idle socket contract, and Zi Yue custom wake contract.
- Updated `docs/project_state_machine.md`:
  - firmware candidate state is now
    `S5H-STACKCHAN-COMPATIBLE-APP-PRELOAD-QUIET-SOCKET-CANDIDATE`;
  - `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001` is active;
  - `T-AUDIO-BARE-XIAOZHI-PARITY-001` is active and worker-dispatched.
- Dispatched a bounded worktree worker for
  `T-AUDIO-BARE-XIAOZHI-PARITY-001`; worker is read-only/docs-only, no flash,
  no provider/V21 execution, no audio playback.

Modified files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/plans/2026-06-03-boot-idle-socket-and-not-connected-ux.md`
- `docs/testing/stackchan-boot-idle-socket-ux-runbook.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Current unfinished items:

- No-write flash plan and guarded flash execute are still pending.
- Physical proof is still pending for:
  - boot connects without touch;
  - screen shows connecting/ready feedback;
  - boot preconnect does not send `listen.start`;
  - touch/no-speech exits green within about 7 seconds;
  - `紫悦` wake variants work once socket is ready.
- Audio parity worker has not yet returned.

Known risks and blockers:

- The direct-start package recovered from welcome/setup but skipped app preload;
  this candidate intentionally changes that behavior, so physical boot must be
  watched carefully for a welcome-screen regression.
- A quiet idle Xiaozhi WebSocket may time out if the server expects active
  traffic; trace reconnection behavior after boot.
- The patch file needed whitespace-safe unified-diff handling because patch
  context lines can look like trailing whitespace to `git diff --check`.

Next recommended actions:

1. Run focused tests, ordinary `git apply --check`, and `make verify`.
2. Build `a21-stackchan-official-xiaozhi-compatible` and inspect
   `sdkconfig.json` for A21 quiet socket, Zi Yue custom wake, and no X21 device
   events.
3. Commit from a clean verified worktree, then run no-write flash plan and
   guarded product flash on `/dev/cu.usbmodem1101`.
4. Ask the operator to validate boot/no-touch connection, not-connected UI,
   no-speech green timeout, and Zi Yue wake variants.

Test/build/run results so far:

- `git -C /Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan apply --check firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`: passed.
- `git diff --check`: passed.
- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady|TestOfficialXiaozhiCompatibleOverlaySetsZiYueCustomWake|TestOfficialXiaozhiCompatibleOverlaySetsCodecVolumeBeforeRuntime|TestRunStackChanOfficialXiaozhiCompatiblePlanReportsProductCandidateContract|TestApplyStackChanOfficialCandidateContractKeepsXiaozhiCompatibleAfterExecute' -count=1`: passed.
- `make verify`: passed.
- `make a21-stackchan-official-xiaozhi-compatible-build`: passed after moving
  official overlay application to after `fetch_repos.py`.
- Build report:
  `reports/a21-stackchan-official-baseline-20260603-040613-1780430773893634000.json`.
- Product app:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Product app SHA-256:
  `10fb2896d6096ab12beb81519166f0cb790226894e6d9451222904d2ff9f65f0`.
- Generated `sdkconfig.json` proves:
  - `BOARD_TYPE_M5STACK_STACK_CHAN=true`;
  - `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`;
  - `USE_CUSTOM_WAKE_WORD=true`;
  - `CUSTOM_WAKE_WORD="zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue"`;
  - `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`;
  - `CUSTOM_WAKE_WORD_THRESHOLD=20`;
  - `SR_MN_CN_MULTINET7_QUANT=true`;
  - `USE_AFE_WAKE_WORD=false`;
  - `SR_WN_WN9_HISTACKCHAN_TTS3=false`;
  - `SEND_WAKE_WORD_DATA=false`;
  - `OTA_URL="http://101.132.117.182/xiaozhi/ota/"`.

Physical regression and immediate hotfix:

- The `requestXiaozhiStart()` after app preload package was flashed by report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-040909-1780430949793206000.json`
  and then physically rejected by the operator: the device again showed
  "Welcome! Let's get started".
- Root cause candidate: even a single Mooncake/app setup frame before Xiaozhi
  start can render the setup trap.
- Hotfix changed the overlay to install official apps, set volume `92`, and
  call `GetHAL().startXiaozhi()` directly before any Mooncake update loop.
- Focused app tests and `git diff --check` passed for this hotfix.
- Rebuild passed:
  `reports/a21-stackchan-official-baseline-20260603-041402-1780431242345821000.json`.
- Hotfix product app SHA-256:
  `7ff81bb0e564e020128e02068bc7d83b36f90c21de8cbd3d4b89b1dd1d6e9cf3`.
- Committed as `9ba8bc1 fix(firmware): skip welcome after stackchan app preload`.
- No-write flash plan passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-041530-1780431330430536000.json`.
- Guarded flash execute passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-041636-1780431396539566000.json`.
- Flash details:
  - port `/dev/cu.usbmodem1101`;
  - app `a21-stackchan-official-xiaozhi-compatible.bin`;
  - app offset `0x20000`;
  - app SHA-256
    `7ff81bb0e564e020128e02068bc7d83b36f90c21de8cbd3d4b89b1dd1d6e9cf3`;
  - control commit `9ba8bc11e2d4`;
  - `flash_allowed=true`;
  - `flash_executed=true`.
- Gateway `127.0.0.1:21081` saw device `44:1b:f6:e2:6a:60` reconnect
  `online` after flash.
- Runtime speaker volume `100` delivered through stock MCP on trace
  `a21-trace-direct-preload-hotfix-volume-1780431400`.

Current operator validation needed:

- Confirm the screen is no longer on "Welcome! Let's get started".
- Confirm the device reaches the A21/Xiaozhi runtime without tapping Skip or
  Start.
- Try wake phrases: `紫悦`, `紫悦紫悦`, `你好紫悦`, `小紫悦`.
- If tapping the screen enters green listening with no speech, confirm it exits
  in roughly 7 seconds.

## 2026-06-03 04:32 CST - T-ASR-GREEN-LATENCY-001 firmware listen-bound candidate

Round goal:

- Continue from the confirmed setup/welcome fix.
- Treat the current physical failure as `无限 ASR + wake disabled while
  listening`, not as a standalone wake-word threshold issue.
- Produce a focused product-firmware candidate that keeps the official
  Xiaozhi-compatible lane and does not touch provider/TTS/gain.

Actual completed:

- Pulled live Gateway state for physical device `44:1b:f6:e2:6a:60` on
  Gateway `127.0.0.1:21081`; device was online and last event was
  `xiaozhi.opus_frame.decoded`.
- Pulled trace `a21-trace-44-1b-f6-e2-6a-60` and confirmed:
  - `xiaozhi.listen.start=110`;
  - `xiaozhi.opus_frame.received=8034`;
  - `xiaozhi.opus_frame.decoded=8034`;
  - repeated `listen.stop -> listen.start`;
  - wake is expected to be ineffective while listening because the product
    config has `WAKE_WORD_DETECTION_IN_LISTENING=false`.
- Root cause candidate:
  - official `Application::HandleStartListeningEvent()` forces
    `kListeningModeManualStop`;
  - existing A21 no-speech timer only armed for `kListeningModeAutoStop`;
  - therefore any official `StartListening()` path can bypass the 7 second
    no-speech timeout and keep the device in green/listening, suppressing wake.
- Updated the official-compatible overlay so:
  - `HandleStartListeningEvent()` uses `GetDefaultListeningMode()` instead of
    `kListeningModeManualStop`;
  - `SetListeningMode()` arms the A21 no-speech timer for all non-realtime
    listening modes;
  - no-speech timeout stops listening for any mode;
  - VAD silence after speech remains restricted to AutoStop, preserving the
    normal Xiaozhi speech turn behavior.
- Added focused guard tests to prevent reintroducing the unbounded manual
  listening path.
- Rebuilt the product firmware candidate.

Modified files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Test/build/run results:

- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady|TestOfficialXiaozhiCompatibleOverlaySetsZiYueCustomWake|TestOfficialXiaozhiCompatibleOverlaySetsCodecVolumeBeforeRuntime' -count=1`: passed.
- `git diff --check`: passed.
- `make verify`: passed.
- `make a21-stackchan-official-xiaozhi-compatible-build`: passed.
- Build report:
  `reports/a21-stackchan-official-baseline-20260603-042940-1780432180247638000.json`.
- Product app:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Product app SHA-256:
  `ca0877d09eecfb9c69f2279ce14c95942a2cf966e05f118e82551e92c14665b9`.
- Generated `sdkconfig.json` confirms:
  - `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`;
  - `USE_CUSTOM_WAKE_WORD=true`;
  - `CUSTOM_WAKE_WORD="zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue"`;
  - `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`;
  - `CUSTOM_WAKE_WORD_THRESHOLD=20`;
  - `SR_MN_CN_MULTINET7_QUANT=true`;
  - `USE_AFE_WAKE_WORD=false`;
  - `WAKE_WORD_DETECTION_IN_LISTENING=false`;
  - `SEND_WAKE_WORD_DATA=false`.

Current unfinished items:

- Commit the focused firmware/test/docs change.
- Run no-write flash plan and guarded flash execute on `/dev/cu.usbmodem1101`.
- After flash, deliver runtime speaker volume `100` again.
- Physical operator validation is still required:
  - boot reaches Xiaozhi runtime without welcome/setup;
  - touch/no-speech green listening exits instead of looping forever;
  - wake variants work from idle: `紫悦`, `紫悦紫悦`, `你好紫悦`, `小紫悦`.

Known risks and blockers:

- This fixes the most likely firmware state-machine cause, but physical proof is
  not yet collected.
- If an official frontend path repeatedly sends `StartListening()` on a tight
  loop, the timer should now stop each no-speech listen, but a second transition
  may still be needed to suppress that trigger source.
- Do not mark wake product-ready until the operator confirms wake from idle.

Next recommended actions:

1. Commit this focused candidate.
2. Run guarded product flash on `/dev/cu.usbmodem1101`.
3. Poll Gateway trace after flash and ask the operator to test no-speech green
   timeout plus the four wake variants.

Follow-up execution in the same round:

- Committed focused fix:
  `88cb8a9 fix(firmware): bound stackchan listening state`.
- No-write flash plan passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-043116-1780432276384588000.json`.
- Guarded flash execute passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-043222-1780432342953949000.json`.
- Flash details:
  - port `/dev/cu.usbmodem1101`;
  - product app `a21-stackchan-official-xiaozhi-compatible.bin`;
  - app offset `0x20000`;
  - app SHA-256
    `ca0877d09eecfb9c69f2279ce14c95942a2cf966e05f118e82551e92c14665b9`;
  - control commit `88cb8a92069d`;
  - T7 guard saw `dirty_file_count=0`;
  - `flash_allowed=true`;
  - `flash_executed=true`.
- Device `44:1b:f6:e2:6a:60` reconnected online on Gateway
  `127.0.0.1:21081`.
- Runtime speaker volume `100` delivered through stock MCP:
  trace `a21-trace-listen-bound-volume-1780432365`,
  tool `self.audio_speaker.set_volume`, status `delivered`.
- Post-flash trace check on `a21-trace-44-1b-f6-e2-6a-60` for events after
  `1780432350000` showed:
  - `events_after_flash=1`;
  - `xiaozhi.hello.received=1`;
  - no automatic `xiaozhi.listen.start` observed in that window.

Updated validation request:

- Ask the operator to confirm the screen still reaches the Xiaozhi/StackChan
  runtime without welcome/setup.
- Tap the screen once and wait without speaking; expected result is that green
  listening exits instead of looping indefinitely.
- After it is idle, test wake phrases:
  `紫悦`, `紫悦紫悦`, `你好紫悦`, `小紫悦`.
- If wake still fails from idle, keep `T-WAKE-003` open and tune the wake
  phrase/threshold next; do not revert to bare `xiaozhi.bin`.

## 2026-06-03 04:39 CST - T-WAKE-003b Zi Yue MultiNet command splitter candidate

Round goal:

- Continue from the listen-bound flashed package.
- Investigate wake from idle as a separate issue now that post-flash Gateway
  evidence shows no automatic `listen.start`.
- Keep product lane as `a21-stackchan-official-xiaozhi-compatible`; do not
  flash bare `xiaozhi.bin`.

Actual completed:

- Rechecked current Gateway state:
  - device `44:1b:f6:e2:6a:60` online;
  - speaker volume capability `100`;
  - last event before this slice was `xiaozhi.mcp.speaker_volume.sent`;
  - post-flash trace still showed only `xiaozhi.hello.received=1` after the
    listen-bound flash window, so the old infinite-ASR trace is pre-fix data.
- Inspected official `CustomWakeWord` implementation:
  - `Kconfig.projbuild` documents `CUSTOM_WAKE_WORD` as a single pinyin command
    separated by spaces;
  - `CustomWakeWord::Initialize()` previously pushed
    `CONFIG_CUSTOM_WAKE_WORD` as one command and later called
    `esp_mn_commands_add` once for that entry;
  - therefore the previous config
    `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue` was likely registered as
    one invalid/overlong MultiNet command, not four alternatives.
- Updated the official-compatible overlay to split
  `CONFIG_CUSTOM_WAKE_WORD` on `|`, trim each segment, and register each phrase
  as a separate MultiNet wake command with display/greeting `紫悦`.
- Added focused guard test strings so the overlay must contain the splitter and
  multi-command push.
- Rebuilt the official-compatible product app successfully.

Modified files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Test/build/run results:

- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlaySetsZiYueCustomWake|TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady' -count=1`: passed.
- `git diff --check`: passed.
- `make a21-stackchan-official-xiaozhi-compatible-build`: passed.
- Build report:
  `reports/a21-stackchan-official-baseline-20260603-043851-1780432731137364000.json`.
- Product app SHA-256:
  `a0638a3b9872c98456c4dee9c8503dfa6d6d27f2374b499fe7c36033aeccb575`.
- Generated `sdkconfig.json` confirms:
  - `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`;
  - `USE_CUSTOM_WAKE_WORD=true`;
  - `CUSTOM_WAKE_WORD="zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue"`;
  - `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`;
  - `CUSTOM_WAKE_WORD_THRESHOLD=20`;
  - `WAKE_WORD_DETECTION_IN_LISTENING=false`.
- Binary strings confirm:
  - `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue`;
  - `Loaded %d A21 custom wake command(s) for %s`.

Current unfinished items:

- Commit the focused splitter candidate.
- Run no-write flash plan and guarded flash execute.
- Deliver runtime speaker volume `100` after flash.
- Physical validation still required for:
  - no welcome/setup regression;
  - no automatic `listen.start` after boot;
  - touch/no-speech green exits;
  - wake variants from idle: `紫悦`, `紫悦紫悦`, `你好紫悦`, `小紫悦`.

Known risks and blockers:

- This fixes the strongest source-level wake config bug, but physical wake is
  still unproven.
- If all four phrases fail after this package, next likely causes are ESP-SR
  threshold/sensitivity, microphone input to MultiNet while idle, or direct
  Xiaozhi startup not enabling wake detection as expected.
- Do not mark wake product-ready without operator proof.

Next recommended actions:

1. Commit this focused candidate.
2. Flash the new app on `/dev/cu.usbmodem1101`.
3. Capture post-flash trace and operator wake result for all four phrases.

Follow-up execution in the same round:

- Read-only wake worker `019e8a0c-5d2d-7992-bb20-0ed64a889351`
  independently confirmed the strongest root cause: `|` phrase list is likely
  invalid/treated as one malformed command; `USE_CUSTOM_WAKE_WORD=true` with
  `USE_AFE_WAKE_WORD=false` is structurally valid for MultiNet; old infinite
  ASR is not the leading cause after the listen-bound flash.
- Read-only hardware parity worker `019e8a0c-5d2d-7992-bb20-0ec6fcba6dbf`
  found direct `GetHAL().startXiaozhi()` preserves the official Xiaozhi
  runtime/HAL/audio path, and avatar/RGB/touch/modifiers are expected after
  Xiaozhi `StackChanAvatarDisplay::SetupUI()`, while Mooncake launcher/app
  runtime surfaces remain bypassed to avoid welcome/setup regression.
- `make verify`: passed.
- Committed focused fix:
  `8e4df0b fix(firmware): split zi yue wake commands`.
- No-write flash plan passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-044128-1780432888876682000.json`.
- Guarded flash execute passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-044234-1780432954645477000.json`.
- Flash details:
  - port `/dev/cu.usbmodem1101`;
  - product app `a21-stackchan-official-xiaozhi-compatible.bin`;
  - app offset `0x20000`;
  - app SHA-256
    `a0638a3b9872c98456c4dee9c8503dfa6d6d27f2374b499fe7c36033aeccb575`;
  - control commit `8e4df0b30c0f`;
  - T7 guard saw `dirty_file_count=0`;
  - `flash_allowed=true`;
  - `flash_executed=true`.
- Device `44:1b:f6:e2:6a:60` reconnected online on Gateway
  `127.0.0.1:21081`, last event `xiaozhi.hello`.
- Runtime speaker volume `100` delivered through stock MCP:
  trace `a21-trace-multi-wake-volume-1780432980`,
  tool `self.audio_speaker.set_volume`, status `delivered`.
- Post-flash trace check on `a21-trace-44-1b-f6-e2-6a-60` for events after
  `1780432954000` showed:
  - `events_after_multi_wake_flash=1`;
  - `xiaozhi.hello.received=1`;
  - no automatic `xiaozhi.listen.start` observed in that window.

Current validation request:

- Confirm the screen still reaches the Xiaozhi/StackChan runtime without
  welcome/setup.
- Tap once and wait without speaking; expected result is that green listening
  exits instead of looping indefinitely.
- From idle, test wake phrases in this order:
  `小紫悦`, `你好紫悦`, `紫悦紫悦`, `紫悦`.
- If all wake variants still fail, next transition should add
  `CustomWakeWord` init/feed/logging or tune threshold; do not revert to bare
  `xiaozhi.bin`.

## 2026-06-03 - Gateway No-Speech Cooldown And Xiaozhi Parity Control-Tower Round

本轮目标:

- Continue from recovered state without redesigning the project.
- Investigate why the live device again showed long/repeated green listening
  after the multi-wake flash.
- Keep the main thread as control tower, use parallel read-only workers for
  Xiaozhi protocol/audio, Gateway/provider streaming, and StackChan
  no-welcome/hardware parity.
- Make only the minimal runtime hotfix needed to reduce the green-listening
  loop; do not flash firmware, change gain, change provider, or touch V21.

实际完成内容:

- Dispatched three read-only subagents:
  - `019e8a21-8fed-7442-9de0-10a774f61ae4` reported that the product firmware
    device side is still official Xiaozhi WebSocket JSON + binary Opus +
    AudioService/AudioCodec, while remaining audio parity gaps are mainly
    host-side provider/TTS PCM/WAV source quality, chunking, leveling, and
    Gateway re-encoding.
  - `019e8a21-d3e3-7802-93b6-a7190e08e55f` reported that `/v1/xiaozhi` is the
    fastest real basic dialogue path, but current ASR/provider still starts
    after VAD/listen stop rather than true live streaming ASR.
  - `019e8a21-b1f5-7ef2-bec1-f084d4b3a4b2` reported that no-welcome/direct
    Xiaozhi is useful but full official AppAvatar/AppDance/AppSetup lifecycle
    parity is not proven because the Mooncake app update path is bypassed.
- Confirmed live pre-fix trace `a21-trace-44-1b-f6-e2-6a-60` had
  `xiaozhi.listen.start=333`, `audio.ingress.buffered=25115`,
  `stackchan.official_auto.not_connected=1025`, and repeated
  `listen.stop -> placeholder tts.stop -> listen.start`.
- Added Gateway stock-physical no-speech input cooldown after
  `placeholder_no_asr_tts`.
- Made listen-start suppression reason-specific:
  - `xiaozhi.listen.start.suppressed_after_no_speech`;
  - `xiaozhi.listen.start.suppressed_after_host_say`;
  - `xiaozhi.listen.start.suppressed_after_barge`.
- Restarted the existing `a21-gateway-21081` tmux Gateway from the patched
  worktree and verified `/healthz`.
- Device `44:1b:f6:e2:6a:60` reconnected online after the final restart; fresh
  trace contained only `xiaozhi.hello.received=1` before operator touch/wake.
- Delivered runtime speaker volume `100` through stock MCP on trace
  `a21-trace-no-speech-cooldown-final-volume-1780434593`.
- Updated `docs/plans/2026-06-03-wake-and-asr-green-latency-recovery.md`.
- Updated `docs/project_state_machine.md`.

修改过的文件:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-03-wake-and-asr-green-latency-recovery.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Physical operator validation is still required:
  - tap once and confirm green listening exits instead of looping;
  - from idle, test `小紫悦`, `你好紫悦`, `紫悦紫悦`, `紫悦`;
  - verify no welcome/setup regression.
- Real basic dialogue smoke is not yet run. The next correct path is real
  `/v1/xiaozhi` listen + Opus ingress + voice pipeline + Opus downlink, not
  `/v1/xiaozhi/say` or `fast-companion-turn`.

已知风险和阻塞点:

- Wake is still not accepted; do not mark `wake_word.product_ready=true`.
- Full official StackChan app lifecycle parity is not proven; current contest
  package preserves the Xiaozhi display/HAL path but bypasses Mooncake app
  running lifecycle to avoid welcome/setup.
- Audio is official on the device side, but source TTS/mastering remains a
  host-side quality and streaming gap.
- `stackchan.official_auto.not_connected` remains a separate official relay
  issue and should not be mistaken for Xiaozhi voice socket failure.

下一轮建议动作:

1. Commit this Gateway cooldown patch if review finds no blocking issue.
2. Ask the operator to tap once from idle and observe whether green listening
   exits; inspect trace for `suppressed_after_no_speech` if it loops.
3. Run `T-XIAOZHI-HOST-LOCAL-REAL-BASIC-DIALOGUE-SMOKE` on the live device
   using real `/v1/xiaozhi`, with evidence for Opus ingress, VAD/listen stop,
   `asr.final`, `provider.first_content`, `tts.first_audio`,
   `audio.downlink.first_frame`, and `xiaozhi.voice_pipeline.completed`.

测试/构建/运行结果:

- `go test ./internal/gateway -run 'TestXiaozhiWebSocket(NoSpeechPlaceholderSuppressesImmediateListenRestartForStockPhysical|TouchAbortSuppressesImmediateListenRestart|SaySuppressesImmediateListenRestartForStockPhysical|VADSpeechEndAutoStopsRealtimeTurn|MaxListenDurationAutoStopsAfterSpeech)$' -count=1`:
  passed.
- `make verify`: passed after the final reason-specific cooldown edits and docs.
- Final live Gateway restart command succeeded:
  `tmux kill-session -t a21-gateway-21081 && tmux new-session -d -s a21-gateway-21081 ... go run ./cmd/a21 gateway --addr 0.0.0.0:21081 ...`.
- `/healthz`: `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`.
- `/v1/devices`: device `44:1b:f6:e2:6a:60` reconnected `online` with
  `last_event=xiaozhi.hello`.
- Fresh trace `a21-trace-44-1b-f6-e2-6a-60` after final restart contained only
  `xiaozhi.hello.received=1` before operator touch/wake.
- A later live trace read showed the cooldown fired once and blocked the
  immediate restart loop:
  - `xiaozhi.no_speech.input_suppression_armed=1`;
  - `xiaozhi.listen.start.input_suppressed=1`;
  - `xiaozhi.listen.start.suppressed_after_no_speech=1`;
  - `xiaozhi.turn.start=1`.
- Runtime volume `100` delivered by stock MCP trace
  `a21-trace-no-speech-cooldown-final-volume-1780434593`.

如果中途失败，记录失败位置和原因:

- No failure yet. Remaining work is physical validation and optional commit,
  not a code blocker.

## 2026-06-03 - T-XIAOZHI-REALTIME-VOICE-PARITY-001 - Realtime Parity Gate Landed

本轮目标:

- Respond to the user's Xiaozhi protocol review by separating "stock Opus
  transport works" from "full Xiaozhi-style realtime ASR/LLM/TTS works".
- Use parallel read-only workers to audit device-side protocol/audio, host
  voice pipeline streaming, and official-compatible firmware setup/wake state.
- Add a non-invasive evidence gate for real `/v1/xiaozhi` traces without
  driving `/say`, synthetic host loopback, provider execution, V21 execution,
  flash, NVS, or audio playback.

实际完成内容:

- Dispatched and received three read-only worker audits:
  - `019e8a30-6e40-7d60-970c-2795e4bae94b`: confirmed product-compatible
    firmware is the official Xiaozhi/CoreS3 path, while current `/v1/xiaozhi`
    host flow is still not streaming-ASR-first.
  - `019e8a30-87b8-7422-8874-c3a14d7911e3`: confirmed Gateway is
    `Opus ingress/downlink shell + turn-buffered ASR input + LLM/TTS answer
    streaming`, not full streaming ASR/LLM/TTS.
  - `019e8a30-a839-74c0-98cd-4bd2723be831`: confirmed no-welcome direct
    `startXiaozhi()` overlay is the protected product path; `requestXiaozhiStart`
    or bare `xiaozhi.bin` can reintroduce the setup/plain-Xiaozhi regression.
- Added plan `docs/plans/2026-06-03-xiaozhi-realtime-voice-parity.md`.
- Added CLI `a21 xiaozhi-realtime-parity`:
  - reads `/v1/devices` and `/v1/traces`;
  - does not trigger conversation or audio;
  - classifies traces as `blocked`, `stock_opus_transport_only`,
    `turn_buffered_xiaozhi_candidate`, or `xiaozhi_realtime_candidate`;
  - rejects `/say`, fast-companion, and local-fallback markers;
  - returns success only for `turn_buffered_xiaozhi_candidate` or
    `xiaozhi_realtime_candidate`.
- Updated `docs/project_state_machine.md` with
  `T-XIAOZHI-REALTIME-VOICE-PARITY-001` and follow-up candidate
  `T-XIAOZHI-STREAMING-ASR-001`.
- Ran the new CLI against current live trace
  `a21-trace-44-1b-f6-e2-6a-60`:
  - device online, stock WebSocket, physical id, and Opus ingress/decode were
    present;
  - counts included `opus_frames_decoded=25` and `pcm_ingress_frames=25`;
  - no VAD speech end, ASR final, provider first content, TTS first audio,
    downlink first frame, or pipeline completion existed;
  - classification was `stock_opus_transport_only`;
  - command returned non-zero as intended.

修改过的文件:

- `internal/app/app.go`
- `internal/app/xiaozhi_realtime_parity.go`
- `internal/app/xiaozhi_realtime_parity_test.go`
- `docs/plans/2026-06-03-xiaozhi-realtime-voice-parity.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- `T-XIAOZHI-STREAMING-ASR-001` is not implemented. Current production path is
  still turn-buffered at the ASR boundary until a streaming ASR session feeds
  decoded PCM frames before listen stop/VAD end.
- Need an operator-triggered real `/v1/xiaozhi` turn, then rerun
  `xiaozhi-realtime-parity` on the resulting trace.
- Wake remains unaccepted physically; do not mark
  `wake_word.product_ready=true`.
- Setup/no-welcome must still be physically guarded after any future firmware
  package/flash.

已知风险和阻塞点:

- Current live trace has stock transport and Opus ingress only; it is not a
  real basic dialogue smoke.
- `xiaozhi-voice-bench` remains host-loopback/synthetic unless explicitly
  proven otherwise; do not use it as physical StackChan acceptance.
- `/v1/xiaozhi/say` can sound good but is not the realtime dialogue path.
- Local TTS and many local ASR/TTS paths still cross WAV/file boundaries and
  must not be labeled true streaming.

下一轮建议动作:

1. Trigger a real physical `/v1/xiaozhi` turn by tap or wake, then run:
   `go run ./cmd/a21 xiaozhi-realtime-parity --gateway-url http://127.0.0.1:21081 --device-id 44:1b:f6:e2:6a:60 --trace-id <trace> --session-id <session> --output-dir reports`.
2. If classification is `turn_buffered_xiaozhi_candidate`, write and execute
   `T-XIAOZHI-STREAMING-ASR-001`: pre-open ASR on listen start, feed decoded
   PCM frames on ingress, and let VAD end commit/finalize.
3. In parallel, keep `T-WAKE-003` physical wake validation and
   `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001` regression checks alive; do not
   flash bare `xiaozhi.bin` or return to `requestXiaozhiStart()`.

测试/构建/运行结果:

- `go test ./internal/app -run 'TestXiaozhiRealtimeParity' -count=1`: passed.
- `git diff --check`: passed.
- `make verify`: passed.
- Gateway live health:
  `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`.
- `/v1/devices`: device `44:1b:f6:e2:6a:60` online, stock Xiaozhi WebSocket,
  `speaker_volume=100`, last trace `a21-trace-44-1b-f6-e2-6a-60`.
- Live `xiaozhi-realtime-parity` against current trace returned exit code `1`
  with classification `stock_opus_transport_only`, which is expected because
  the trace did not contain a completed voice turn.

如果中途失败，记录失败位置和原因:

- No implementation failure. The live trace classification is red by design:
  current trace has transport/Opus ingress evidence only, not full real
  dialogue or streaming parity evidence.

## 2026-06-03 - T-XIAOZHI-STREAMING-ASR-001 - Streaming ASR Session Seam

本轮目标:

- Continue from the realtime parity gate toward actual Xiaozhi-style voice:
  pre-open ASR on stock `/v1/xiaozhi` listen start, feed decoded Opus PCM frames
  as they arrive, and use VAD/listen stop only to commit/finalize.
- Preserve the existing turn-buffered ASR fallback and do not disturb firmware,
  wake, provider credentials, V21, audio gain, or TTS source selection.

实际完成内容:

- Added plan `docs/plans/2026-06-03-xiaozhi-streaming-asr.md`.
- Dispatched and received three read-only workers:
  - `019e8a3a-fbe7-7bf1-af8a-2d3fd3263985`: Gateway seam audit; confirmed
    attach points at listen start, `observeXiaozhiDecodedIngress`, stop/VAD
    commit, and abort/close cancellation.
  - `019e8a3b-166c-73b2-b2d1-c9d7cd5ce95b`: provider ASR audit; confirmed
    existing Sherpa path is WAV/buffered fallback and must not be marked true
    streaming.
  - `019e8a3b-3224-7d21-8a17-0676824dec38`: trace/parity audit; confirmed
    `xiaozhi-realtime-parity` should require ASR stream markers and answer
    downlink markers to avoid fast-ack/fallback false green.
- Added optional provider interfaces:
  - `providers.StreamingASRAdapter`;
  - `providers.StreamingASRSession`;
  - `providers.StreamingASRStartRequest`.
- Added `providers.NewMockStreamingASRAdapter` for tests. It also satisfies the
  existing `ASRAdapter` fallback contract, so old batch path remains available.
- Added Gateway streaming ASR lifecycle:
  - start session on `/v1/xiaozhi` listen start if selected ASR supports it;
  - feed decoded PCM frames from `observeXiaozhiDecodedIngress`;
  - record `asr.stream.start`, `asr.audio.append`, `asr.stream.commit`,
    `asr.first_partial`, and `asr.final`;
  - commit on listen stop / VAD auto-stop;
  - cancel on abort/socket close/reset.
- Added `VoicePipelineRequest.ASRTranscript` so a streaming final transcript can
  be reused without re-running batch ASR. The transcript is in-memory only and
  not stored in reports.
- Added answer downlink marker `xiaozhi.voice_pipeline.answer.downlink`.
- Hardened `xiaozhi-realtime-parity`:
  - counts `asr.stream.start`, `asr.audio.append`, and `asr.stream.commit`;
  - requires ASR stream start/append and answer downlink for
    `xiaozhi_realtime_candidate`;
  - treats `xiaozhi.local_fallback.sent` as a forbidden fake/fallback marker.
- Updated `docs/project_state_machine.md` with active
  `T-XIAOZHI-STREAMING-ASR-001`.

修改过的文件:

- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/xiaozhi_realtime_parity.go`
- `internal/app/xiaozhi_realtime_parity_test.go`
- `docs/plans/2026-06-03-xiaozhi-streaming-asr.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Real streaming ASR provider/local engine is not connected yet. This phase
  proves the Gateway/session seam with a mock streaming ASR adapter only.
- Current live Gateway process was not restarted in this round, so live hardware
  remains on the previous binary until a deliberate restart/deploy.
- `xiaozhi-realtime-parity` live classification should be rerun after a Gateway
  restart and a real physical `/v1/xiaozhi` turn.
- `T-XIAOZHI-STREAMING-ASR-PROVIDER-001` must select and implement a real
  streaming ASR provider/session; Sherpa WAV fallback is explicitly not enough.

已知风险和阻塞点:

- Fake streaming ASR tests prove architecture and ordering only; they do not
  prove provider readiness, wake readiness, or physical product acceptance.
- Existing TTS may still be file/segment based depending on selected adapter;
  full Xiaozhi parity also needs true streaming TTS/provider evidence.
- `device.playback.start` remains debug/device-events evidence, not stock
  baseline acceptance.
- Stock protocol direction allowlist remains a separate transition and was not
  mixed into this work.

下一轮建议动作:

1. Start `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`: choose the real streaming ASR
   backend/session API and wire it into `providers.StreamingASRAdapter`.
2. Restart Gateway from the new commit only after review/commit, then run one
   real physical `/v1/xiaozhi` turn and collect `xiaozhi-realtime-parity`.
3. Keep wake/setup validation parallel: no bare `xiaozhi.bin`, no
   `requestXiaozhiStart()`, and no `wake_word.product_ready=true` without
   physical proof.

测试/构建/运行结果:

- `go test ./internal/providers -run 'TestMockStreamingASRAdapter' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestXiaozhiWebSocket(StreamingASRStartsBeforeListenStop|ListenStopRunsVoicePipelineAndSendsPacedOpus|VADSpeechEndAutoStopsRealtimeTurn)$' -count=1`:
  passed.
- `go test ./internal/app -run 'TestXiaozhiRealtimeParity' -count=1`: passed.
- `go test ./internal/gateway -count=1`: passed.
- `go test ./internal/providers -count=1`: passed.
- `make verify`: passed.
- `git diff --check`: passed.

如果中途失败，记录失败位置和原因:

- No failure. Scope is intentionally Phase 1: streaming ASR session seam and
  trace-order proof with a mock adapter, not real provider acceptance.

## 2026-06-03 - T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001b - Park After Direct Xiaozhi Start

本轮目标:

- Recover from the operator report that the physical device returned to the
  first-run "Welcome! Let's get started" page after the multi-wake flash.
- Preserve the official-compatible product lane and avoid bare `xiaozhi.bin`.
- Find the root cause before changing firmware again.

实际完成内容:

- Recovered current control state from `AGENTS.md`, `docs/project_state_machine.md`,
  recent handoff log entries, recent build/flash reports, and current git
  status.
- Confirmed the latest product flash was not the bare Xiaozhi lane:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-044234-1780432954645477000.json`
  flashed `a21-stackchan-official-xiaozhi-compatible.bin` with app SHA-256
  `a0638a3b9872c98456c4dee9c8503dfa6d6d27f2374b499fe7c36033aeccb575`.
- Inspected the built binary strings and source export. The binary contains
  both the A21 direct-start log and the upstream welcome strings, proving the
  app includes both code paths.
- Identified the actual root cause: official `Hal::startXiaozhi()` starts
  Xiaozhi/StackChan tasks and returns. The previous A21 direct-start overlay
  then fell through into the Mooncake main loop, allowing `AppLauncher` to
  create `StartupWorker` and render the welcome/setup page.
- Updated the official-compatible overlay so after direct
  `GetHAL().startXiaozhi()` it parks `app_main` in a watchdog-feeding sleep
  loop before the Mooncake main loop can run.
- Added a focused app test guard requiring the watchdog/delay parking before
  the Mooncake main loop anchor.
- Read-only worker notifications also confirmed the streaming ASR provider
  work is not ready for real acceptance and should not distract from this
  firmware welcome regression.

修改过的文件:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Operator physical no-welcome validation is pending: visually confirm the
  screen is no longer on "Welcome! Let's get started".
- Wake remains physically unaccepted; do not mark
  `wake_word.product_ready=true`.
- Streaming ASR provider is still not real-provider accepted; workers confirmed
  existing Iflytek/cloud ASR evidence is final-only and Sherpa
  `streaming_zipformer` is currently WAV/file based.

已知风险和阻塞点:

- Parking `app_main` preserves the Xiaozhi runtime path but bypasses Mooncake
  app running lifecycle after install. This is intentional for the contest
  recovery path; full official-app lifecycle parity remains a later transition.
- If physical device still shows welcome after this build/flash, the next
  suspect is stale artifact/partition flash mismatch, not this control-flow
  path.

下一轮建议动作:

1. Ask the operator to confirm the welcome/setup page is gone after the
   `e3758cff...` product flash.
2. After no-welcome is physically clean, validate idle socket, touch no-speech
   exit, and wake variants `小紫悦`, `你好紫悦`, `紫悦紫悦`, `紫悦`.
3. If welcome still appears, treat stale artifact/partition mismatch as the
   next hypothesis and inspect serial boot logs plus app partition SHA.

测试/构建/运行结果:

- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlaySetsCodecVolumeBeforeRuntime|TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady|TestOfficialXiaozhiCompatibleOverlaySetsZiYueCustomWake' -count=1`:
  passed.
- `git diff --check`: passed.
- `make verify`: passed.
- Official-compatible product build:
  `reports/a21-stackchan-official-baseline-20260603-054837-1780436917458986000.json`,
  status `passed`, app SHA-256
  `e3758cffdc294e74220f5288a33306d4aace164279bea6acb3a2cea5af17e1c3`.
- No-write flash plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-054848-1780436928156547000.json`,
  status `ready`, same app SHA.
- Guarded flash execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-054954-1780436994114555000.json`,
  status `passed`, `flash_executed=true`, clean control commit
  `e694550f1739`, port `/dev/cu.usbmodem1101`.
- Gateway health after flash returned
  `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`.
- Device `44:1b:f6:e2:6a:60` reconnected `online` after reboot with
  `last_event=xiaozhi.hello`.
- Runtime volume `100` delivered through stock MCP on trace
  `a21-trace-no-welcome-park-volume-1780437008`.
- Post-flash trace events filtered since `1780436933516` contained only
  `xiaozhi.hello.received=2`, with no automatic `listen.start`.

如果中途失败，记录失败位置和原因:

- First focused test run failed because the new test looked for `+    // Main loop`;
  that line is patch context, not an added line. The test anchor was corrected
  to search the real patch context line.

## 2026-06-03 - T-XIAOZHI-STREAMING-ASR-PROVIDER-001a - Static Streaming Provider Gate

本轮目标:

- Continue the user-requested Xiaozhi realtime parity work after the no-welcome
  flash hotfix.
- Dispatch independent read-only workers to re-audit official Xiaozhi protocol,
  A21 Gateway `/v1/xiaozhi`, and real streaming provider candidates.
- Add a machine-readable gate that prevents mock ASR, `streaming_zipformer`
  name-only evidence, `/say`, or WAV/file TTS from being promoted to
  Xiaozhi-style realtime voice parity.

实际完成内容:

- Confirmed current state: branch
  `codex/a21-hardware-window-20260602-stackchan-prd`, HEAD `4829a9b`, clean
  worktree at start.
- Dispatched three projectless read-only workers:
  - `019e8a52-c184-7e53-ba3c-c3b9205950fb`: official Xiaozhi protocol/audio
    source audit; completed and confirmed stock hello/Opus, `tts` start/stop,
    local wake/AFE, AudioService, and 60 ms pacer are the non-negotiable parity
    shape. It also warned that `/v1/xiaozhi/say` and host-only good audio must
    not replace real `/v1/xiaozhi` physical trace evidence.
  - `019e8a52-c5ee-73a2-a28b-40d5f6624fba`: A21 Gateway `/v1/xiaozhi`
    realtime gap audit; completed and confirmed Gateway has the streaming ASR
    seam but real provider/TTS streaming remains missing.
  - `019e8a52-cb20-79f0-af7a-b70420ed848b`: streaming ASR/TTS provider
    candidate audit; completed and recommended Sherpa streaming_zipformer as
    the fastest real ASR session path, while confirming Iflytek IAT has no
    runtime adapter and current Iflytek TTS still writes WAV.
- Added plan
  `docs/plans/2026-06-03-xiaozhi-streaming-provider-readiness.md`.
- Added CLI `a21 xiaozhi-streaming-provider-readiness`.
- Added Makefile target `xiaozhi-streaming-provider-readiness` for the same
  static gate. It is expected to exit non-zero while the provider chain remains
  blocked.
- The new gate is static/no-execute: it reads selected voice pipeline profiles
  and classifies whether ASR, LLM, and TTS satisfy strict Xiaozhi realtime
  provider requirements. It stores no transcript, raw audio, provider output,
  credential values, full URLs, or local absolute paths.
- The gate explicitly blocks:
  - mock ASR/LLM/TTS;
  - Sherpa ASR because current adapter writes accumulated PCM to a WAV before
    running the local ASR script;
  - Iflytek TTS because current adapter receives provider chunks but writes a
    WAV report and only then re-reads chunks for downlink;
  - unknown or unconfigured ASR/TTS profiles.
- Added tests for default mock blocking, Sherpa+StepFun+Iflytek blocking, and a
  test-only all-streaming fixture that proves the gate can pass only when all
  three provider stages are marked streaming.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-streaming-provider-readiness.md`
- `Makefile`
- `internal/app/app.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- No real streaming ASR provider is implemented yet.
- No true streaming TTS adapter is implemented yet.
- All three read-only worker reports have now completed and should be used as
  input to the next provider implementation transition.
- Physical `xiaozhi-realtime-parity` remains unaccepted until a real
  operator-triggered `/v1/xiaozhi` turn proves streaming ASR/LLM/TTS/downlink
  ordering.

已知风险和阻塞点:

- This gate is a truthful blocker, not a voice-quality improvement by itself.
- Static profile classification must be updated when a real streaming adapter
  lands, otherwise it will continue to block correctly but conservatively.
- Current StepFun+Iflytek candidate improves audio but still has ASR WAV/batch
  and TTS WAV/file boundaries, so it cannot satisfy the user's Xiaozhi realtime
  architecture requirement yet.

下一轮建议动作:

1. Read final W1/W2/W3 worker outputs.
2. Implement the first real `providers.StreamingASRAdapter`: either a long-lived
   Sherpa streaming subprocess/session or an Iflytek IAT streaming WebSocket
   adapter.
3. After real ASR streaming is green, implement true streaming TTS emission
   that forwards provider audio chunks toward Gateway downlink before a full
   WAV/file exists.
4. Rerun:
   `go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`.
5. Then run a physical `/v1/xiaozhi` turn and evaluate
   `xiaozhi-realtime-parity`.

测试/构建/运行结果:

- `go test ./internal/app -run 'TestXiaozhiStreamingProviderReadiness' -count=1`:
  passed.
- `git diff --check`: passed.
- `make verify`: passed.
- `go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`:
  exited `1` by design; report
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-055502.json`
  has `gate_status=blocked` with mock ASR/LLM/TTS findings.
- `A21_ASR_LOCAL_PROFILE=sherpa_onnx A21_TEXT_STREAM_PROFILE=stepfun A21_TTS_FAST_PROFILE=iflytek_tts go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`:
  exited `1` by design; report
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-055512.json`
  has `gate_status=blocked`,
  `asr_batch_wav_boundary_not_xiaozhi_streaming`, and
  `tts_wav_file_boundary_not_xiaozhi_streaming`.

如果中途失败，记录失败位置和原因:

- No implementation failure. The two CLI runs are intentionally non-zero
  because the current provider chain is not yet Xiaozhi-realtime compliant.

## 2026-06-03 - T-XIAOZHI-SHERPA-STREAMING-ASR-ADAPTER-001 - Selectable Sherpa Streaming ASR Seam

本轮目标:

- Continue from the recovered control-tower state without reopening the already
  fixed welcome/setup issue.
- Implement the next strict provider transition for Xiaozhi realtime parity:
  make Sherpa streaming ASR selectable as a `providers.StreamingASRAdapter`
  without falsely promoting the existing batch/WAV `sherpa_onnx` adapter.
- Keep the work inside provider/readiness/docs scope: no firmware, no flash, no
  service restart, no real provider/ASR/V21 execution, and no audio playback.

实际完成内容:

- Confirmed current branch `codex/a21-hardware-window-20260602-stackchan-prd`,
  HEAD `0cb5573`, clean at the start except new work from this round.
- Added plan
  `docs/plans/2026-06-03-sherpa-streaming-asr-adapter.md`.
- Launched worker thread `019e8a5c-383d-7571-8b4b-14a5a885154f` in isolated
  worktree `/Users/jiyurun/.codex/worktrees/be99/New project` for the same
  scoped transition. It wrote red provider tests, then the control tower took
  over the minimal implementation to avoid waiting; the worker was instructed
  to stop further edits and not commit.
- Added `StreamingASRSessionFactory` and a
  `NewLocalSherpaONNXStreamingASRAdapter` wrapper. The wrapper keeps the old
  batch Sherpa adapter as fallback for `Transcribe`, but only starts streaming
  through a configured session factory.
- Added selectable streaming ASR profiles:
  `sherpa_onnx_streaming`, `local_sherpa_onnx_streaming`, and
  `streaming_zipformer`.
- Preserved existing behavior for `sherpa_onnx` and `local_sherpa_onnx`: they
  remain batch/WAV and do not implement `StreamingASRAdapter`.
- Updated the static provider readiness gate so streaming Sherpa is no longer
  mislabeled as a WAV boundary. It blocks missing helper/model proof with
  `asr_sherpa_streaming_helper_or_model_missing`.
- Updated readiness report filenames to include `UnixNano` so concurrent gate
  runs do not overwrite each other.
- Updated `docs/project_state_machine.md` with the completed adapter seam and
  the remaining TTS/runtime-helper blockers.

修改过的文件:

- `docs/plans/2026-06-03-sherpa-streaming-asr-adapter.md`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- The new Sherpa streaming profile is an adapter seam only; a real long-lived
  helper/model runtime has not been executed or proven.
- Streaming TTS is still not implemented. Iflytek TTS remains a WAV/file
  boundary in the strict Xiaozhi provider gate.
- Physical `xiaozhi-realtime-parity` is still blocked until real ASR helper
  proof, streaming TTS, and an operator-triggered stock `/v1/xiaozhi` turn are
  recorded.
- Wake word physical proof remains separate and still not accepted.

已知风险和阻塞点:

- Do not treat `sherpa_onnx_streaming` selection as product readiness by itself.
  It only proves Gateway can select a streaming ASR adapter when a session
  factory/helper exists.
- The readiness gate intentionally allows ASR stage readiness when helper/model
  env are present, but that is still static/no-execute; runtime proof must come
  from a future helper smoke/trace transition.
- TTS remains the next hard blocker for Xiaozhi-style realtime voice.

下一轮建议动作:

1. Implement or connect a long-lived Sherpa streaming helper process/session
   and prove `AppendFrame -> partial -> Commit -> final` without WAV.
2. Implement streaming TTS chunk emission before complete WAV/file synthesis.
3. After ASR helper and TTS streaming are both real, run physical
   `xiaozhi-realtime-parity` from an operator-triggered `/v1/xiaozhi` turn.

测试/构建/运行结果:

- `go test ./internal/providers -run 'TestLocalSherpaONNX.*ASR|TestVoicePipelineAdaptersFromEnv.*Sherpa|TestVoicePipelineAdaptersFromEnvDefaultsMockAndSelectsHostLocal' -count=1`:
  passed.
- `go test ./internal/app -run TestXiaozhiStreamingProviderReadiness -count=1`:
  passed.
- `go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`:
  intentionally exited non-zero; report
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-060836-1780438116871814000.json`
  is blocked by mock ASR/LLM/TTS.
- `A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming A21_TEXT_STREAM_PROFILE=stepfun A21_TTS_FAST_PROFILE=iflytek_tts go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`:
  intentionally exited non-zero; report
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-060837-1780438117185327000.json`
  is blocked by missing Sherpa streaming helper/model and TTS WAV boundary, not
  by ASR WAV boundary.
- `A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming A21_SHERPA_ONNX_STREAMING_HELPER=/redacted/a21-sherpa-streaming-helper A21_SHERPA_ONNX_ASR_MODEL_DIR=/redacted/a21-sherpa-model A21_TEXT_STREAM_PROFILE=stepfun A21_TTS_FAST_PROFILE=iflytek_tts go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`:
  intentionally exited non-zero; report
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-060837-1780438117326052000.json`
  marks ASR+LLM ready but still blocks on
  `tts_wav_file_boundary_not_xiaozhi_streaming`.
- `git diff --check`: passed.
- `make verify`: passed.

如果中途失败，记录失败位置和原因:

- Initial parallel CLI gate runs used the prior second-resolution filename and
  wrote the same report path. The report writer was fixed to include
  `UnixNano`; reruns produced unique report files.

## 2026-06-03 - T-XIAOZHI-STREAMING-TTS-ADAPTER-001 - Doubao Realtime TTS Seam

本轮目标:

- Continue the full Xiaozhi parity objective from the user's protocol review:
  local wake/VAD, long stock socket, Opus 60 ms frames, streaming ASR, streaming
  LLM, streaming TTS, and paced Opus playback.
- Dispatch multiple read-only workers to re-read Xiaozhi/A21 protocol and audio
  paths before further implementation.
- Remove the next known false-streaming boundary: TTS adapters that return a
  channel only after a complete WAV/file exists.

实际完成内容:

- Confirmed branch `codex/a21-hardware-window-20260602-stackchan-prd`, starting
  HEAD `399a2a6`, and clean worktree at start.
- Dispatched read-only worker threads for:
  - official Xiaozhi protocol/audio implementation parity;
  - A21 Gateway `/v1/xiaozhi` streaming/state-machine gap map;
  - A21 provider ASR/TTS streaming gap review.
- Added plan
  `docs/plans/2026-06-03-xiaozhi-streaming-tts-adapter.md`.
- Added `providers.StreamingTTSAdapter` as an explicit capability marker. Local
  WAV/file TTS adapters do not implement it.
- Added a Doubao realtime TTS pipeline adapter selected by
  `A21_TTS_FAST_PROFILE=doubao_tts_realtime` or `doubao_realtime_tts`.
- Reused the existing `DoubaoRealtimeTTSProvider` and realtime WebSocket session
  primitives; no new provider protocol was invented.
- Added a PCM16 mono chunker that accumulates arbitrary provider
  `response.audio.delta` payloads and emits exact `pcm_s16le`, mono, 60 ms
  `VoiceAudioChunk` values for Gateway Opus downlink.
- The new adapter writes no WAV/file and tests use fake realtime connection
  messages only; no real provider call was made.
- Updated `xiaozhi-streaming-provider-readiness` so:
  - local/Iflytek/voice-clone TTS remain blocked as WAV/file boundary;
  - `doubao_tts_realtime` is recognized as streaming and implemented in
    Gateway;
  - missing Doubao env yields `tts_doubao_realtime_config_missing`;
  - configured Doubao env makes the TTS stage ready without storing secrets.
- Updated `docs/project_state_machine.md` with the completed adapter seam.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-streaming-tts-adapter.md`
- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- No real Doubao/Iflytek/5080 provider execution was run in this transition.
- No physical `/v1/xiaozhi` realtime parity trace was captured.
- Sherpa streaming ASR still needs a real long-lived helper/runtime proof.
- Wake word physical proof remains separate and not accepted.
- Full PRD launch remains red.

已知风险和阻塞点:

- Static provider-shape gate can now pass when ASR helper env, StepFun, and
  Doubao realtime TTS env are all present; this must not be confused with
  runtime provider proof or physical acceptance.
- Doubao realtime TTS is one reusable streaming adapter seam, not necessarily
  the final contest voice. Iflytek/5080 streaming adapters can reuse the same
  60 ms chunking shape.
- Provider audio deltas may arrive in arbitrary sizes; the chunker buffers
  partial frames and pads only on final drain.

下一轮建议动作:

1. Implement/connect the long-lived Sherpa streaming ASR helper and prove real
   runtime `AppendFrame -> partial -> Commit -> final` without WAV.
2. Run a no-audio-output realtime TTS provider smoke against the selected
   provider path, keeping reports redacted.
3. Once ASR helper and TTS provider runtime evidence are real, run physical
   `xiaozhi-realtime-parity` from an operator-triggered stock `/v1/xiaozhi`
   turn.

测试/构建/运行结果:

- `go test ./internal/providers -run 'TestDoubaoRealtimeTTS|TestVoicePipelineAdaptersFromEnv.*TTS|TestLocalTTSAdapter' -count=1`:
  passed.
- `go test ./internal/app -run TestXiaozhiStreamingProviderReadiness -count=1`:
  passed.
- `A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming A21_TEXT_STREAM_PROFILE=stepfun A21_TTS_FAST_PROFILE=doubao_tts_realtime go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`:
  intentionally exited non-zero; report
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061849-1780438729911945000.json`
  blocks missing ASR helper/model and missing Doubao realtime TTS config.
- `A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming A21_TEXT_STREAM_PROFILE=stepfun A21_TTS_FAST_PROFILE=doubao_tts_realtime A21_DOUBAO_API_KEY=sk-a21-secret A21_DOUBAO_TTS_MODEL=doubao-tts A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`:
  intentionally exited non-zero; report
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061850-1780438730222971000.json`
  marks TTS ready but blocks missing ASR helper/model.
- `A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming A21_SHERPA_ONNX_STREAMING_HELPER=/redacted/a21-sherpa-streaming-helper A21_SHERPA_ONNX_ASR_MODEL_DIR=/redacted/a21-sherpa-model A21_TEXT_STREAM_PROFILE=stepfun A21_TTS_FAST_PROFILE=doubao_tts_realtime A21_DOUBAO_API_KEY=sk-a21-secret A21_DOUBAO_TTS_MODEL=doubao-tts A21_DOUBAO_TTS_VOICE=zh_female_kailangjiejie_moon_bigtts go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`:
  exited zero; report
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061850-1780438730359984000.json`
  has `gate_status=passed` for the static provider-shape gate and
  `prd_accepted=false`.
- `git diff --check`: passed.
- `make verify`: passed.

如果中途失败，记录失败位置和原因:

- No failing implementation attempt. Worker thread creation/listing was
  asynchronous, so the control tower implemented the bounded no-provider TTS
  seam directly after writing the plan.

## 2026-06-03 - T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001b - Current HEAD Reflash Recovery

本轮目标:

- Recover from the operator report that the physical StackChan was again stuck
  on the official first-run "Welcome! Let's get started" setup page.
- Preserve the current control-tower rules: no bare `xiaozhi.bin`, no broad
  business-code changes, no provider/V21 execution, and product firmware only
  through the guarded `a21-stackchan-official-xiaozhi-compatible` lane.

实际完成内容:

- Confirmed the main worktree is on branch
  `codex/a21-hardware-window-20260602-stackchan-prd`, clean at HEAD
  `790697518c3e1c24e72668bd5da3decfe3fab626`.
- Read the current no-welcome state from `docs/project_state_machine.md`,
  `docs/agent_handoff_log.md`, recent plan/report files, and the compatible
  overlay.
- Verified the current overlay still contains the no-welcome hotfix:
  `GetHAL().startXiaozhi()` is followed by a watchdog-feeding infinite park
  loop, preventing fall-through into the Mooncake welcome/setup loop.
- Rebuilt the current HEAD product firmware with
  `make a21-stackchan-official-xiaozhi-compatible-build`.
- Ran the no-write flash plan for `/dev/cu.usbmodem1101`.
- Executed the guarded product flash on `/dev/cu.usbmodem1101`.
- Verified Gateway `127.0.0.1:21081` stayed healthy.
- Verified the physical device `44:1b:f6:e2:6a:60` reconnected online after
  flash.
- Verified post-flash trace events since the flash execution contain only
  `xiaozhi.hello.received=1` and no `xiaozhi.listen.start`, so the previous
  infinite green/listening loop did not automatically restart after this flash.
- Delivered runtime speaker volume `100` over stock MCP to the live device.
- Opened a read-only background thread request to inspect stale artifact /
  partition / welcome-regression causes; it was not available before this log
  entry, so the main thread did not wait for it.

修改过的文件:

- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Operator visual confirmation is still required: the screen must be checked
  physically to confirm it is no longer on "Welcome! Let's get started".
- Wake word physical proof remains red; this flash only restores the current
  no-welcome product candidate.
- Normal click/touch barge-in and no-speech UX still need physical acceptance.
- The full Xiaozhi realtime parity objective remains open: real streaming ASR
  helper/runtime, streaming TTS provider execution, and physical stock
  `/v1/xiaozhi` turn proof are not complete.

已知风险和阻塞点:

- If the welcome screen still appears after this current-HEAD flash, treat it as
  a stale partition/OTA/runtime boot selection problem, not as evidence that
  the overlay source lacks the park-after-start hotfix.
- Do not use the generic `xiaozhi-firmware-flash-*` lane for product recovery;
  it can regress the device to plain Xiaozhi UI.
- The Gateway process is still the 05:09 `a21-gateway-21081` tmux service. It
  is healthy and has the no-speech cooldown env, but it was not restarted in
  this round.

下一轮建议动作:

1. Ask the operator to visually confirm the welcome/setup page is gone. If it
   is still visible, read the serial boot partition/app logs before changing
   source code.
2. From the idle screen, test wake phrases `紫悦`, `紫悦紫悦`, `你好紫悦`, and
   `小紫悦`; record whether a `xiaozhi.listen.start` appears without a touch.
3. If visual no-welcome is clean but wake still fails, continue `T-WAKE-003`
   with device-side custom wake init/feed logging or threshold tuning.

测试/构建/运行结果:

- `make a21-stackchan-official-xiaozhi-compatible-build`: passed.
  - Report:
    `reports/a21-stackchan-official-baseline-20260603-062637-1780439197207182000.json`
  - App SHA-256:
    `fc7788736ced71c98cee846892a867d306d663c5ceb31014cc01079a381e766c`
- `make a21-stackchan-official-xiaozhi-compatible-flash-plan A21_UPLOAD_PORT=/dev/cu.usbmodem1101`:
  passed as no-write ready.
  - Report:
    `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062715-1780439235852990000.json`
- `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP make a21-stackchan-official-xiaozhi-compatible-flash-execute`:
  passed.
  - Report:
    `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062853-1780439333539408000.json`
  - Guard commit: `790697518c3e`
  - Dirty file count at flash: `0`
  - App part: `a21-stackchan-official-xiaozhi-compatible.bin` at `0x20000`
- `curl http://127.0.0.1:21081/healthz`: passed with
  `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`.
- `curl http://127.0.0.1:21081/v1/devices`: passed; device
  `44:1b:f6:e2:6a:60` was `online`, last event `xiaozhi.hello`, speaker volume
  capability `100`.
- `curl http://127.0.0.1:21081/v1/traces?trace_id=a21-trace-44-1b-f6-e2-6a-60`
  plus local `jq` filter: post-flash event count `1`, only
  `xiaozhi.hello.received`, no automatic `xiaozhi.listen.start`.
- `POST /v1/xiaozhi/speaker-volume` with trace
  `a21-trace-current-head-volume-1780439333`: passed, status `delivered`,
  transport `xiaozhi_mcp`, tool `self.audio_speaker.set_volume`, volume `100`.

如果中途失败，记录失败位置和原因:

- No code fix failed in this round. The key recovery action was re-building and
  re-flashing the current HEAD product app to eliminate stale artifact or stale
  flash-state ambiguity.

## 2026-06-03 - T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001 - Plan And Worker Dispatch

本轮目标:

- Continue the user's full Xiaozhi realtime voice objective:
  local wake/VAD, long stock Xiaozhi socket, Opus 60 ms frames, streaming ASR,
  streaming LLM, streaming TTS, paced Opus playback, and clean device states.
- Move the ASR provider side from a profile/factory seam toward a real
  long-lived streaming helper session.
- Preserve control-tower discipline: plan first, worker execution in scoped
  worktree, no broad business-code edits in the main thread.

实际完成内容:

- Confirmed main branch
  `codex/a21-hardware-window-20260602-stackchan-prd`, starting HEAD
  `0dd575c3dc46758398f1fe544a2c96cc36d1a992`, and clean worktree.
- Re-read current state around `T-XIAOZHI-STREAMING-ASR-001`, existing Sherpa
  streaming adapter seam, Doubao realtime TTS seam, and strict provider
  readiness gate.
- Used memory only as a cautionary guardrail: wake/voice false-green, stock
  Xiaozhi profile cleanliness, and host-only evidence not being PRD acceptance.
- Dispatched three read-only background review threads:
  - `W-XIAOZHI-OFFICIAL-STREAMING-PARITY`: official Xiaozhi / StackChan
    realtime protocol, AudioService, state-machine, wake/listen/speak parity.
  - `W-A21-XIAOZHI-GATEWAY-GAP`: A21 Gateway/provider gap against
    local wake + long socket + Opus frames + streaming ASR/LLM/TTS.
  - `W-SHERPA-STREAMING-RUNTIME-SPEC`: JSONL helper protocol and Go session
    design for Sherpa streaming ASR.
- Added plan
  `docs/plans/2026-06-03-sherpa-streaming-asr-runtime-helper.md`.
- Committed the plan as
  `13e9e82 docs(plans): define sherpa streaming asr runtime helper`.
- Dispatched implementation worker
  `T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001` in a scoped worktree, using the
  new plan and strict boundaries.
- Updated `docs/project_state_machine.md` so the active transition is visible
  even before the worker returns.

修改过的文件:

- `docs/plans/2026-06-03-sherpa-streaming-asr-runtime-helper.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Implementation worker has not yet returned in the main thread.
- No subprocess helper code has been integrated in the main worktree yet.
- No real Sherpa model/runtime execution has been run.
- No Gateway restart, firmware flash, NVS write, provider/V21 execution, or
  audio playback occurred in this dispatch round.
- Full Xiaozhi realtime parity remains incomplete: ASR runtime helper,
  streaming TTS runtime proof, wake proof, and physical `/v1/xiaozhi` realtime
  turn evidence are still open.

已知风险和阻塞点:

- The plan intentionally keeps static helper env separate from real runtime
  proof. Do not mark ASR PRD-ready merely because
  `A21_SHERPA_ONNX_STREAMING_HELPER` and model dir env exist.
- The helper must not write WAV in the streaming path.
- Helper errors must not leak transcripts, local paths, provider output, URLs,
  proxy values, or credentials.
- If the worker cannot be read through Codex thread tools, continue from the
  committed plan and inspect any new worktree/branch before duplicating work.

下一轮建议动作:

1. Read the implementation worker result when available.
2. If worker completed, review its branch/diff, run the focused tests,
   `git diff --check`, and `make verify` in the main integration context.
3. If worker did not start or cannot be recovered, implement the committed plan
   in a new scoped worktree rather than improvising in the main thread.

测试/构建/运行结果:

- `git diff --check -- docs/plans/2026-06-03-sherpa-streaming-asr-runtime-helper.md`:
  passed before committing the plan.
- No full test run yet after the state/log update; it should be run after
  worker integration or before the next commit.

如果中途失败，记录失败位置和原因:

- Background thread discovery returned pending worktree ids but did not
  immediately list readable thread ids. The main thread therefore recorded the
  dispatch and continued with repo-carried plan/state instead of waiting on UI
  state.

## 2026-06-03 - T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001 - Worker Implementation Candidate

本轮目标:

- Execute the committed runtime-helper plan in an isolated worktree:
  `/Users/jiyurun/.codex/worktrees/a21-sherpa-streaming-asr-runtime-manual/New project`.
- Add a real subprocess-backed `StreamingASRSessionFactory` for
  `sherpa_onnx_streaming` without touching firmware, Gateway runtime, providers,
  V21, hardware, or audio playback.

实际完成内容:

- Created branch
  `codex/a21-sherpa-streaming-asr-runtime-manual-20260603` from main HEAD
  `20a75f9`.
- Verified isolated-worktree baseline with `make verify`: passed.
- Added repo-owned helper script
  `scripts/a21_sherpa_onnx_streaming_asr_session.py`.
  - JSONL stdin/stdout commands: `start`, `append`, `commit`, `cancel`.
  - JSONL events: `ready`, `partial`, `final`, `error`.
  - Fake mode via `A21_SHERPA_STREAMING_ASR_FAKE=1` for no-model dry checks.
  - Real mode supports `streaming_zipformer` shape and emits stable error
    codes when sherpa-onnx/model files are missing.
- Added a Go subprocess-backed Sherpa streaming ASR session:
  - starts helper through `A21_SHERPA_ONNX_STREAMING_HELPER`;
  - optional `A21_SHERPA_ONNX_STREAMING_PYTHON`;
  - requires `A21_SHERPA_ONNX_ASR_MODEL_DIR`;
  - defaults family to `streaming_zipformer`;
  - sends redacted trace/session IDs to the helper;
  - sends each PCM frame as base64 in an `append` command;
  - reads `partial`/`final` helper events into `ASRAdapterEvent`;
  - discards stderr and returns stable redacted Go errors.
- Wired `VoicePipelineAdaptersFromEnv` so only streaming Sherpa profiles use
  the subprocess factory when helper/model env are configured:
  `sherpa_onnx_streaming`, `local_sherpa_onnx_streaming`, and
  `streaming_zipformer`.
- Preserved batch `sherpa_onnx` / `local_sherpa_onnx` behavior.
- Added fake-helper tests proving:
  - `start -> append -> commit` command order;
  - partial arrives after `AppendFrame`;
  - final arrives after `Commit`;
  - streaming path does not call the batch WAV runner;
  - trace/session IDs are redacted in helper commands.
- Found and fixed one race/flakiness during `make verify`: the initial
  implementation let the `cmd.Wait()` goroutine close the events channel while
  stdout reader was still responsible for sending helper events. The fix gives
  channel close ownership to the stdout reader only.

修改过的文件:

- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `scripts/a21_sherpa_onnx_streaming_asr_session.py`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- No real sherpa-onnx model was executed.
- No live Gateway `/v1/xiaozhi` turn used this helper yet.
- No provider, V21, firmware, NVS, flash, hardware, or audio playback occurred.
- Static `xiaozhi-streaming-provider-readiness` still must not be read as PRD
  acceptance.
- Full realtime chain remains incomplete until streaming TTS runtime proof,
  wake proof, and physical stock Xiaozhi turn evidence are collected.

已知风险和阻塞点:

- The helper is process-per-ASR-session; long-run backpressure and crash
  recovery still need a real no-audio model smoke.
- Real sherpa-onnx model layout may differ from the assumed
  `streaming_zipformer` files; helper returns stable model-missing errors but
  has not been run against local model weights.
- Reports must continue to avoid storing transcripts, paths, URLs, proxy
  values, credentials, or raw audio.

下一轮建议动作:

1. Main control thread should review this worker diff and merge/cherry-pick the
   branch into the main hardware branch if acceptable.
2. Run a no-audio model smoke only after confirming actual model path/weights.
3. After ASR helper runtime and streaming TTS runtime are both proven, run the
   physical stock `/v1/xiaozhi` realtime parity gate from an operator-triggered
   turn.

测试/构建/运行结果:

- Baseline in isolated worktree before edits: `make verify` passed.
- `A21_SHERPA_STREAMING_ASR_FAKE=1 ... scripts/a21_sherpa_onnx_streaming_asr_session.py`:
  passed with `ready`, `partial`, `final`.
- `go test ./internal/providers -run 'TestLocalSherpaONNX.*Streaming|TestVoicePipelineAdaptersFromEnv.*Sherpa' -count=1`:
  passed.
- `go test ./internal/providers -run 'TestLocalSherpaONNX.*ASR|TestVoicePipelineAdaptersFromEnv.*Sherpa|TestVoicePipelineAdaptersFromEnvDefaultsMockAndSelectsHostLocal' -count=1`:
  passed.
- `go test ./internal/app -run TestXiaozhiStreamingProviderReadiness -count=1`:
  passed.
- `python3 -m py_compile scripts/a21_sherpa_onnx_streaming_asr_session.py`:
  passed.
- `git diff --check`: passed.
- `go test ./internal/providers -count=1`: passed after fixing the events
  close race.
- Final `make verify`: passed.

如果中途失败，记录失败位置和原因:

- First final `make verify` failed in
  `TestLocalSherpaONNXStreamingASRAdapterRunsSubprocessHelper` with a timeout
  waiting for a helper ASR event. Root cause was event-channel close ownership:
  `cmd.Wait()` could close the event channel before the stdout reader completed.
  The fix moved close ownership to the stdout reader; rerun `make verify`
  passed.

## 2026-06-03 - T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001 - Mainline Integration Verification

本轮目标:

- Resume from the interrupted control-thread state without restarting design.
- Integrate the scoped Sherpa streaming ASR worker branch into the main
  hardware branch.
- Verify the integrated mainline while preserving the already repaired
  StackChan setup/no-welcome state and avoiding hardware/provider/V21 side
  effects.

实际完成内容:

- Fast-forward merged branch
  `codex/a21-sherpa-streaming-asr-runtime-manual-20260603` into
  `codex/a21-hardware-window-20260602-stackchan-prd`.
- Mainline HEAD after merge: `041ad69`.
- Re-ran focused mainline verification for the new ASR helper and the existing
  Xiaozhi streaming provider readiness guard.
- Found a mainline `make verify` flake in the new subprocess-helper test:
  `TestLocalSherpaONNXStreamingASRAdapterRunsSubprocessHelper` timed out after
  one second waiting for the fake helper event under full `go test ./...`
  package parallelism.
- Reproduced the subprocess-helper test directly with
  `go test ./internal/providers -run TestLocalSherpaONNXStreamingASRAdapterRunsSubprocessHelper -count=20 -failfast -v`:
  passed.
- Reproduced the full provider package with
  `go test ./internal/providers -count=20 -failfast -v`: passed.
- Root-cause judgment: production helper path did not fail; the test's
  one-second ASR event timeout was too tight for subprocess startup/stdout
  scheduling during full-repo parallel verification.
- Hardened the provider test helper wait window from one second to five seconds.

修改过的文件:

- `internal/providers/voice_pipeline_real_adapters_test.go`
- `docs/agent_handoff_log.md`

当前未完成事项:

- `make verify` was rerun after the timeout hardening patch and passed.
- The timeout hardening patch plus this handoff update were committed as
  `987a532`.
- No real sherpa-onnx model, live Gateway turn, provider call, V21 call,
  firmware build, flash, NVS write, or audio playback occurred in this
  integration verification step.

已知风险和阻塞点:

- Static tests still do not prove physical wake, realtime provider quality, or
  end-to-end Xiaozhi turn acceptance.
- Process-per-ASR-session helper remains a candidate path pending real model
  smoke and live Gateway integration.

下一轮建议动作:

1. Continue with real-model no-audio Sherpa smoke once the model path/weights
   are confirmed.
2. Continue streaming TTS runtime proof without provider key leakage.
3. Continue physical `/v1/xiaozhi` realtime parity from an operator-triggered
   turn only after setup/wake state is stable.

测试/构建/运行结果:

- `go test ./internal/providers -run 'TestLocalSherpaONNX.*ASR|TestVoicePipelineAdaptersFromEnv.*Sherpa|TestVoicePipelineAdaptersFromEnvDefaultsMockAndSelectsHostLocal' -count=1`:
  passed on mainline.
- `go test ./internal/app -run TestXiaozhiStreamingProviderReadiness -count=1`:
  passed on mainline.
- `python3 -m py_compile scripts/a21_sherpa_onnx_streaming_asr_session.py`:
  passed; generated `scripts/__pycache__` was removed.
- Fake helper dry check with `A21_SHERPA_STREAMING_ASR_FAKE=1`: passed with
  `ready`, `partial`, `final`.
- First mainline `make verify`: failed in
  `TestLocalSherpaONNXStreamingASRAdapterRunsSubprocessHelper` due to the
  one-second test timeout described above.
- Direct subprocess-helper stress:
  `go test ./internal/providers -run TestLocalSherpaONNXStreamingASRAdapterRunsSubprocessHelper -count=20 -failfast -v`:
  passed.
- Full provider package stress:
  `go test ./internal/providers -count=20 -failfast -v`: passed.
- `git diff --check`: passed after the timeout hardening patch.
- `go test ./internal/providers -run TestLocalSherpaONNXStreamingASRAdapterRunsSubprocessHelper -count=20 -failfast`:
  passed after the timeout hardening patch.
- `go test ./internal/providers -run 'TestLocalSherpaONNX.*ASR|TestVoicePipelineAdaptersFromEnv.*Sherpa|TestVoicePipelineAdaptersFromEnvDefaultsMockAndSelectsHostLocal' -count=1`:
  passed after the timeout hardening patch.
- `go test ./internal/app -run TestXiaozhiStreamingProviderReadiness -count=1`:
  passed after the timeout hardening patch.
- Final `make verify`: passed after the timeout hardening patch.

如果中途失败，记录失败位置和原因:

- Failure location: `internal/providers/voice_pipeline_real_adapters_test.go`
  ASR event receive helper.
- Failure reason: brittle one-second test timeout under full-repo parallel
  verification load, not observed as a helper protocol or runtime failure in
  direct repeated tests.

## 2026-06-03 - T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001 - Mainline State Closeout

本轮目标:

- Remove stale "need rerun/need commit" wording after the successful mainline
  verification commit.
- Update the project state machine so a fresh model can see that the helper is
  now a mainline candidate, not only a worker candidate.

实际完成内容:

- Updated `docs/project_state_machine.md` current/next state to
  `S-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-HELPER-MAINLINE-CANDIDATE`.
- Recorded mainline integration commit `041ad69` and test hardening commit
  `987a532`.
- Updated this handoff log to reflect that `make verify` passed and the
  hardening commit exists.

修改过的文件:

- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- No code/runtime work remains in this closeout step after commit.
- No runtime, hardware, provider, V21, firmware, NVS, or audio playback action
  occurred in this closeout step.

已知风险和阻塞点:

- Mainline candidate status is still not real-model, live-Gateway, wake, or
  physical PRD acceptance.

下一轮建议动作:

1. Continue with real-model no-audio Sherpa smoke.
2. Continue streaming TTS runtime proof.
3. Continue physical Xiaozhi parity under the
   existing hardware-window discipline.

测试/构建/运行结果:

- `git diff --check`: passed for the documentation closeout.

如果中途失败，记录失败位置和原因:

- No closeout failure at the time of writing.

## 2026-06-03 - T-XIAOZHI-FULL-REALTIME-VOICE-CONVERGENCE-001 - Control Plan And Read-Only Worker Fan-Out

本轮目标:

- Accept the user's Xiaozhi review as the realtime voice target, without
  redefining success around helper seams, static readiness, `/say`, host
  loopback, WAV/file TTS, or mock tests.
- Dispatch independent read-only workers to cross-check official Xiaozhi
  protocol/state, ESP32/CoreS3 audio HAL/wake behavior, and A21 actual runtime
  gaps.
- Write a repo-carried convergence plan before any further broad code changes.

实际完成内容:

- Confirmed main worktree state before edits:
  branch `codex/a21-hardware-window-20260602-stackchan-prd`, HEAD `61b27ac`,
  clean.
- Used the Codex thread tools to dispatch three strict read-only workers:
  - `019e8a8a-7268-7210-bf9c-eca8ab1c5c6d`:
    Xiaozhi protocol/state-machine parity audit.
  - `019e8a8a-7263-7b20-94b9-9b2847aa741d`:
    Xiaozhi ESP32/CoreS3 audio HAL, wake/VAD, MCP volume, and hardware behavior
    parity audit.
  - `019e8a8a-7265-71d3-ae47-0dd708965ec6`:
    A21 `/v1/xiaozhi` realtime runtime gap audit.
- Added the control plan
  `docs/plans/2026-06-03-xiaozhi-full-realtime-voice-convergence.md`.
- Updated `docs/project_state_machine.md` with active transition
  `T-XIAOZHI-FULL-REALTIME-VOICE-CONVERGENCE-001`.
- Main-thread code read confirmed the current nuanced state:
  - Gateway has stock-shaped long `/v1/xiaozhi` WebSocket and binary Opus
    ingress/downlink.
  - Gateway can start a streaming ASR session on listen start and append decoded
    PCM frames.
  - `VoicePipelineRunner` can reuse `ASRTranscript` from streaming ASR and avoid
    re-running batch ASR.
  - `RunStream()` can emit audio chunks as LLM text segments arrive.
  - Full Xiaozhi realtime acceptance is still unproven because real Sherpa model
    runtime, real streaming TTS/provider runtime, wake proof, and physical
    stock `/v1/xiaozhi` realtime trace are missing.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-full-realtime-voice-convergence.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Need read worker returns and incorporate any corrected official-source
  findings.
- Need run `git diff --check` and commit the plan/state/log if clean.
- Need dispatch the next implementation worker for exactly one transition:
  likely real Sherpa streaming ASR no-audio smoke or streaming TTS runtime
  proof.
- No real provider execution, V21 execution, hardware, firmware build, flash,
  NVS write, service restart, or audio playback occurred in this round.

已知风险和阻塞点:

- `xiaozhi-streaming-provider-readiness` can pass a static shape gate under
  configured env, but that still is not runtime/provider/physical acceptance.
- `sherpa_onnx_streaming` helper is mainline candidate only until a real model
  smoke or stable missing-model report exists.
- `doubao_tts_realtime` is an adapter seam and static readiness candidate only
  until runtime provider proof exists.
- Wake remains physical-red; screen tap can be an explicitly labeled fallback
  trigger, not a wake acceptance substitute.
- Bare `xiaozhi.bin` must not be used as product firmware even if it sounds
  better; it is comparison evidence only.

下一轮建议动作:

1. Read the three worker returns and adjust the convergence plan/state if they
   identify a stronger P0 than the current main-thread assessment.
2. If no contradiction appears, dispatch `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001`
   as the next scoped worker, because ASR runtime proof is the first missing
   chain segment after Gateway streaming-session wiring.
3. In parallel after ASR smoke, dispatch `T-STREAMING-TTS-RUNTIME-PROOF-001`
   for provider/runtime TTS proof before any physical `/v1/xiaozhi` parity
   acceptance attempt.

测试/构建/运行结果:

- `git diff --check`: passed for the initial convergence plan/state/log update.

如果中途失败，记录失败位置和原因:

- No failure at the time of writing; worker results are still pending.

## 2026-06-03 - T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001 - Plan Dispatch Prep

本轮目标:

- Prepare the next implementation transition after the Xiaozhi convergence
  plan: prove or truthfully block real local Sherpa streaming ASR model runtime
  without audio/hardware/provider side effects.

实际完成内容:

- Added plan
  `docs/plans/2026-06-03-sherpa-realmodel-no-audio-smoke.md`.
- Updated `docs/project_state_machine.md` with active transition
  `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001`.
- Scoped the worker boundary to no audio playback, no microphone capture, no
  provider/V21 execution, no Gateway/service restart, no firmware/flash/NVS,
  no model download, and no WAV boundary.

修改过的文件:

- `docs/plans/2026-06-03-sherpa-realmodel-no-audio-smoke.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Need run final `git diff --check`.
- Need commit these planning/state/log updates.
- Need dispatch the implementation worker for
  `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` after the docs commit or from the
  recorded plan.

已知风险和阻塞点:

- Local model files may be absent or in a different layout. The accepted
  outcome is then a stable missing-model blocker, not a fake pass.
- Real Sherpa Python package availability is unknown.
- Even a passing no-audio smoke remains below physical Xiaozhi realtime
  acceptance.

下一轮建议动作:

1. Commit the convergence and Sherpa smoke plans if diff checks pass.
2. Dispatch one scoped worker to implement `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001`.
3. Continue reading the read-only Xiaozhi parity worker returns in parallel.

测试/构建/运行结果:

- Pending final `git diff --check`.

如果中途失败，记录失败位置和原因:

- No failure at the time of writing.

## 2026-06-03 - T-XIAOZHI-FULL-REALTIME-VOICE-CONVERGENCE-001 - Worker Returns And Sherpa Smoke Dispatch

本轮目标:

- Record the committed convergence planning bundle.
- Record read-only worker conclusions that arrived after the initial plan.
- Dispatch the next scoped implementation worker without expanding scope.

实际完成内容:

- Committed the convergence/Sherpa-smoke plan bundle as `5047b5d`
  (`docs(control): plan xiaozhi realtime convergence`).
- Read-only worker `019e8a8a-7268-7210-bf9c-eca8ab1c5c6d`
  completed protocol/state-machine parity audit:
  - A21 already has stock-shaped WebSocket/Opus shell and streaming seams.
  - Full Xiaozhi realtime parity is not already true.
  - LLM/TTS still start after ASR transcript/final availability.
  - Real Sherpa streaming model proof and live provider/physical evidence are
    missing.
  - Suggested next transitions align with the new convergence plan.
- Read-only worker `019e8a8a-7265-71d3-ae47-0dd708965ec6`
  completed A21 runtime path audit:
  - Opus ingress and ASR append are realtime.
  - Answer path still begins after `listen.stop` or VAD/max-duration auto-stop.
  - `RunStream()` can stream LLM segments into TTS, but only after ASR commit.
  - Defaults can still select batch Sherpa ASR and WAV Sherpa TTS unless env is
    explicit.
  - Recommended `T-XIAOZHI-CONTINUOUS-TURN-OVERLAP-001` after ASR/TTS runtime
    proof.
- Read-only worker `019e8a8a-7263-7b20-94b9-9b2847aa741d`
  initially remained active on ESP32/CoreS3 audio HAL and wake/VAD parity.
- Dispatched implementation worker
  `019e8a8e-df69-70e3-a18f-c083d33023ea` for
  `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` in isolated worktree
  `/Users/jiyurun/.codex/worktrees/cdf0/New project`.

修改过的文件:

- `docs/agent_handoff_log.md`

当前未完成事项:

- Need read the audio HAL worker final result when it completes.
- Need read and review the Sherpa real-model smoke worker result.
- Need integrate the worker branch only after its tests pass and scope is
  confirmed.
- No provider execution, V21 execution, hardware, firmware build, flash, NVS
  write, service restart, or audio playback occurred in the main control thread.

已知风险和阻塞点:

- The current Gateway is realtime at Opus ingress and ASR append, but not yet
  true Xiaozhi continuous overlap because answer generation starts after ASR
  commit.
- Static readiness and adapter seams still cannot satisfy physical PRD
  acceptance.
- Audio HAL/wake worker may reprioritize the next firmware/audio transition
  after it returns.

下一轮建议动作:

1. Poll `019e8a8e-df69-70e3-a18f-c083d33023ea` for the Sherpa smoke worker
   summary and commit.
2. Poll `019e8a8a-7263-7b20-94b9-9b2847aa741d` for audio HAL/wake findings.
3. If Sherpa smoke is clean, review and fast-forward/cherry-pick it; then
   proceed to `T-STREAMING-TTS-RUNTIME-PROOF-001`.

测试/构建/运行结果:

- Main control thread ran `git diff --check` before commit `5047b5d`: passed.
- No code tests were run in this main-thread dispatch update.

如果中途失败，记录失败位置和原因:

- No failure in main control thread; worker results are pending.

## 2026-06-03 - T-AUDIO-BARE-XIAOZHI-PARITY-001 / T-WAKE - Audio HAL Worker Return

本轮目标:

- Record the final read-only audio HAL/wake worker findings so the next model
  does not lose them during realtime-ASR/TTS work.

实际完成内容:

- Read-only worker `019e8a8a-7263-7b20-94b9-9b2847aa741d` completed.
- Confirmed no edits, no commits, no builds, no runtime/service changes, no
  provider/V21 execution, no hardware/audio/flash/NVS actions in that worker.
- Key findings:
  - A21 product overlay still bypasses the official StackChan app lifecycle by
    calling `GetHAL().startXiaozhi()` and then parking, while official
    StackChan installs AppLauncher/AppAiAgent/AppAvatar/AppDance/AppSetup and
    runs Mooncake before Xiaozhi start. This is interaction-critical for
    display, touch, RGB, servo, and avatar coupling.
  - Wake path differs from official default: official S3 favors AFE WakeNet
    when available; A21 product overlay disables AFE wake and uses custom
    MultiNet aliases for `紫悦`. This is a likely P0 for wake failure and must
    remain separate from loudness/TTS work.
  - Device HAL is official-like, but A21 still applies host-side downlink
    leveling, Go Opus encode, pacing, and provider/TTS decisions before device
    decode. This supports keeping official codec/HAL while proving provider and
    Gateway differences with evidence.
  - Official StackChan CoreS3 source uses `input_gain=60`; upstream Xiaozhi
    CoreS3 source uses `input_gain=30`. Recognition impact is unproven but
    source-level difference is real and wake/ASR-relevant.
  - `/v1/xiaozhi/say` remains useful for downlink/audio checks only; it is not
    a wake/VAD-driven physical turn acceptance path.
- Suggested future transitions from the worker:
  - `T-WAKE-004-AFE-VS-CUSTOM-PARITY`
  - `T-STACKCHAN-APP-LIFECYCLE-PARITY`
  - `T-HAL-AUDIO-CONFIG-PARITY`
  - Continue `T-XIAOZHI-REALTIME-VOICE-PARITY-RUNTIME`

修改过的文件:

- `docs/agent_handoff_log.md`

当前未完成事项:

- Need add the three worker-suggested transitions to state machine candidate
  queue when the main thread next edits governance docs.
- Need review the active Sherpa smoke worker result.
- Need avoid firmware/lifecycle/wake edits until a scoped plan/worker is
  created; current implementation worker is ASR smoke only.

已知风险和阻塞点:

- If we only fix provider streaming while leaving app lifecycle/wake diverged,
  physical UX can still feel unlike official StackChan/Xiaozhi.
- If we restore official app lifecycle naively, the welcome/setup trap may come
  back; it needs its own scoped transition.
- `紫悦` wake should not be marked accepted without physical proof.

下一轮建议动作:

1. Continue current `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` worker to completion.
2. After ASR/TTS runtime proof, plan `T-WAKE-004-AFE-VS-CUSTOM-PARITY`.
3. Plan `T-STACKCHAN-APP-LIFECYCLE-PARITY` before touching overlay lifecycle.

测试/构建/运行结果:

- No code tests run for this log-only recording.

如果中途失败，记录失败位置和原因:

- No failure in main control thread.

## 2026-06-03 - T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001 - Recovery From Lost Progress

本轮目标:

- Recover from control-thread context loss without restarting the design.
- Continue from the operator-confirmed fact that setup/welcome is fixed.
- Verify current physical Gateway state before deciding whether to flash or
  hotfix again.

实际完成内容:

- Re-read `AGENTS.md` instructions provided in-thread, the latest handoff log,
  the boot idle socket plan, and `docs/project_state_machine.md`.
- Confirmed the current branch is
  `codex/a21-hardware-window-20260602-stackchan-prd`, clean, at
  `e7d3a40c6e50`; the firmware product flash evidence was created earlier from
  clean commit `790697518c3e` before later docs/control commits.
- Confirmed the current overlay already contains:
  - direct Xiaozhi start followed by `GetHAL().feedTheDog()` /
    `GetHAL().delay(1000)` parking to avoid the Mooncake welcome/setup loop;
  - A21 quiet idle websocket preconnect;
  - A21 no-speech / VAD listen bounding;
  - split `紫悦` MultiNet command registration.
- Verified latest guarded product flash identity:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062853-1780439333539408000.json`
  passed through T7 from clean commit `790697518c3e`, flashed product app
  `a21-stackchan-official-xiaozhi-compatible.bin`, app SHA-256
  `fc7788736ced71c98cee846892a867d306d663c5ceb31014cc01079a381e766c`.
- Confirmed Gateway health on `127.0.0.1:21080` and `127.0.0.1:21081`.
- Queried current physical device registry on `21081`: device
  `44:1b:f6:e2:6a:60` is `online`, `last_event=xiaozhi.hello`,
  `speaker_volume=100`, trace `a21-trace-44-1b-f6-e2-6a-60`.
- Parsed physical trace `a21-trace-44-1b-f6-e2-6a-60`:
  - `event_count=1016`;
  - `xiaozhi.listen.start=1`;
  - `xiaozhi.listen.stop=1`;
  - `xiaozhi.turn.start=1`;
  - `xiaozhi.no_speech.input_suppression_armed=1`;
  - `xiaozhi.listen.start.input_suppressed=1`;
  - `xiaozhi.listen.start.suppressed_after_no_speech=1`;
  - latest event is `xiaozhi.hello.received`.
- Delivered runtime speaker volume `100` again through stock MCP on trace
  `a21-trace-recovery-volume-1780441733`; response status was `delivered`.
- Updated `docs/project_state_machine.md` from the stale welcome failure state
  to `S-APP-PRELOAD-NO-WELCOME-IDLE-SOCKET-CANDIDATE`, without marking wake or
  touch acceptance green.
- Read existing worker threads:
  - `019e8a8e-df69-70e3-a18f-c083d33023ea` Sherpa smoke worker is still
    active and had reached an honest `model_dir_missing` blocked smoke result
    before its final docs/commit verification.
  - Prior read-only provider/ASR workers confirmed there is no real streaming
    ASR provider evidence yet; cloud ASR evidence is final-only and local
    `streaming_zipformer` evidence was WAV/file-smoke only.

修改过的文件:

- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Operator physical retry is still required:
  - confirm the screen remains out of `Welcome! Let's get started`;
  - tap once without speaking and confirm green listening exits instead of
    looping;
  - from idle, try `紫悦`, `紫悦紫悦`, `你好紫悦`, and `小紫悦`.
- Poll and integrate the Sherpa no-audio smoke worker if it commits cleanly.
- If all wake variants still fail from idle while socket is online, start
  `T-WAKE-004-AFE-VS-CUSTOM-PARITY` with custom wake init/feed logging or
  AFE-vs-MultiNet parity as the next firmware transition.

已知风险和阻塞点:

- `S-APP-PRELOAD-NO-WELCOME-IDLE-SOCKET-CANDIDATE` is not full acceptance:
  wake and touch physical checks are still open.
- The trace still contains an old touch-start/listen episode, so future
  reviewers must distinguish historical events from the latest idle hello
  state.
- Real Xiaozhi realtime ASR provider evidence remains blocked; this should not
  block the immediate wake/touch physical validation but still blocks full PRD
  acceptance.

下一轮建议动作:

1. Ask the operator to perform the three physical checks above while keeping
   Gateway `21081` running.
2. Immediately re-query `/v1/devices` and
   `/v1/traces?trace_id=a21-trace-44-1b-f6-e2-6a-60` after the operator action.
3. If wake fails with the device online and idle, dispatch/execute
   `T-WAKE-004-AFE-VS-CUSTOM-PARITY`; do not re-open setup or flash bare
   `xiaozhi.bin`.

测试/构建/运行结果:

- `curl http://127.0.0.1:21080/healthz`: passed.
- `curl http://127.0.0.1:21081/healthz`: passed.
- `curl http://127.0.0.1:21081/v1/devices`: returned physical StackChan
  online with `last_event=xiaozhi.hello`.
- `curl http://127.0.0.1:21081/v1/xiaozhi/speaker-volume`: delivered MCP
  volume `100` on trace `a21-trace-recovery-volume-1780441733`.
- No firmware build, flash, NVS write, provider execution, V21 execution, audio
  playback, or service restart was performed in this recovery round.

如果中途失败，记录失败位置和原因:

- A first ad-hoc Python JSON scan and two piped curl/json-tool queries hung;
  only those diagnostic processes were killed. Gateway and device services were
  not stopped.

## 2026-06-03 - T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001 - Implementation Closeout

本轮目标:

- Implement the smallest no-audio/no-hardware Sherpa real-model streaming ASR
  smoke so the repo can either prove real local model startup/append/commit or
  truthfully record a stable blocker.

实际完成内容:

- Created scoped branch
  `codex/a21-sherpa-realmodel-no-audio-smoke-20260603` from detached `5047b5d`.
- Added `a21 local-asr-streaming-smoke`, registered it in the app command
  router, and added `make local-asr-streaming-smoke`.
- The smoke validates helper/model preflight, writes only redacted report
  fields, and sends one generated in-memory PCM16LE silence frame only when
  helper and streaming model files are present.
- Ran the local smoke with helper
  `scripts/a21_sherpa_onnx_streaming_asr_session.py` and no model download. It
  produced ignored report
  `reports/a21-local-asr-streaming-smoke-20260603-070451-1780441491984219000.json`
  with `status=blocked`, finding `model_dir_missing`,
  `model_files_present=false`, and zero appended frames/events.
- Updated `docs/project_state_machine.md` to mark this transition completed as
  a truthful blocker, not ASR runtime or physical Xiaozhi acceptance.

修改过的文件:

- `Makefile`
- `internal/app/app_plan_execute.go`
- `internal/app/local_asr_streaming_smoke.go`
- `internal/app/local_asr_streaming_smoke_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Real local Sherpa streaming ASR remains unproven until
  `A21_SHERPA_ONNX_ASR_MODEL_DIR` points to a valid local streaming Zipformer
  cache and the smoke can start/append/commit against real model files.

已知风险和阻塞点:

- Current stable blocker: `model_dir_missing`.
- This no-audio smoke is below stock `/v1/xiaozhi` physical realtime parity and
  below PRD acceptance.

下一轮建议动作:

1. Configure or stage a valid local Sherpa streaming Zipformer model cache
   without downloading inside an implementation worker, then rerun
   `a21 local-asr-streaming-smoke`.
2. Continue `T-STREAMING-TTS-RUNTIME-PROOF-001` for the next missing realtime
   chain segment.

测试/构建/运行结果:

- Red test first:
  `go test ./internal/app -run TestRunLocalASRStreamingSmoke -count=1` failed
  because `local-asr-streaming-smoke` was not registered.
- Focused app tests:
  `go test ./internal/app -run 'TestRunLocalASRStreamingSmoke|TestLocalASRStreaming' -count=1`
  passed.
- Focused provider tests:
  `go test ./internal/providers -run 'TestLocalSherpaONNX.*ASR|TestVoicePipelineAdaptersFromEnv.*Sherpa' -count=1`
  passed.
- Runtime smoke command:
  `go run ./cmd/a21 local-asr-streaming-smoke --helper scripts/a21_sherpa_onnx_streaming_asr_session.py --output-dir reports`
  returned nonzero as expected for the truthful blocker and wrote the ignored
  `model_dir_missing` report named above.
- Final `git diff --check`: passed.
- Final `make verify`: passed.

如果中途失败，记录失败位置和原因:

- Earlier focused app test assertion was too broad and matched the required
  `audio_payload_policy` redaction field; the assertion was narrowed to actual
  encoded payload field names.

## 2026-06-03 - Mainline Integration - No-Welcome Recovery And Sherpa Smoke

本轮目标:

- Integrate the recovered no-welcome/idle-socket state and the completed Sherpa
  no-audio smoke worker into the main control branch.

实际完成内容:

- Committed main-thread recovery/state update as `7ea6e78`
  (`docs(control): recover stackchan no-welcome state`).
- Cherry-picked worker commit `8dbd5b9` into the main branch as `9c7cdbb`
  (`feat(a21): add sherpa streaming asr smoke`).
- Resolved the only cherry-pick conflict in `docs/agent_handoff_log.md` by
  preserving both the hardware recovery handoff and the Sherpa worker closeout.
- Verified the current physical StackChan state before integration:
  `44:1b:f6:e2:6a:60` is online on Gateway `21081`, last event is
  `xiaozhi.hello`, and runtime volume `100` was delivered through stock MCP on
  trace `a21-trace-recovery-volume-1780441733`.

修改过的文件:

- `Makefile`
- `internal/app/app_plan_execute.go`
- `internal/app/local_asr_streaming_smoke.go`
- `internal/app/local_asr_streaming_smoke_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Operator physical validation remains the immediate hardware task:
  - screen must stay out of the welcome/setup page;
  - tap once without speaking and verify green listening exits;
  - from idle, try `紫悦`, `紫悦紫悦`, `你好紫悦`, and `小紫悦`.
- Real local Sherpa streaming ASR remains blocked by missing local model dir:
  `model_dir_missing`.

已知风险和阻塞点:

- Wake is still not accepted until physical proof passes from idle while the
  socket is online.
- The new Sherpa smoke is an honest no-audio blocker/reporting tool, not a
  realtime ASR acceptance proof.

下一轮建议动作:

1. Run the physical wake/touch checks and immediately re-query
   `/v1/devices` plus trace `a21-trace-44-1b-f6-e2-6a-60`.
2. If wake still fails, start `T-WAKE-004-AFE-VS-CUSTOM-PARITY` and add
   device-side custom wake init/feed evidence without reopening setup.
3. Continue `T-STREAMING-TTS-RUNTIME-PROOF-001` or configure the local Sherpa
   model cache, but do not claim real Xiaozhi realtime parity from the current
   no-audio smoke.

测试/构建/运行结果:

- `go test ./internal/app -run 'TestRunLocalASRStreamingSmoke|TestLocalASRStreaming' -count=1`: passed.
- `go test ./internal/providers -run 'TestLocalSherpaONNX.*ASR|TestVoicePipelineAdaptersFromEnv.*Sherpa' -count=1`: passed.
- `git diff --check`: passed.
- `make verify`: passed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The only merge conflict was in
  `docs/agent_handoff_log.md` during worker cherry-pick and was resolved by
  preserving both handoff entries.

## 2026-06-03 - Control Recovery - Setup Fixed, Resume Realtime Chain

本轮目标:

- Recover the control-tower state after context loss without reopening the
  already-fixed setup/welcome issue.
- Preserve the current transition state in repo documents and continue from the
  latest explicit next step toward Xiaozhi-style realtime voice.

实际完成内容:

- Re-read the live branch status, recent commits, latest handoff entries,
  current state-machine wake/no-welcome/realtime sections, and latest plan list.
- Confirmed the current repo baseline records the operator's later result that
  setup/no-welcome is fixed.
- Confirmed the working tree had only governance-document changes from the
  in-progress `T-STREAMING-TTS-RUNTIME-PROOF-001` plan/state update.
- Found the three latest read-only audit workers for Xiaozhi realtime parity,
  streaming TTS runtime proof, and wake endpoint parity ended in `systemError`;
  their outputs are not accepted as evidence.
- Kept the continuation point at `T-STREAMING-TTS-RUNTIME-PROOF-001` while
  leaving wake/touch physical validation explicitly open.

修改过的文件:

- `docs/plans/2026-06-03-streaming-tts-runtime-proof.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Implement `T-STREAMING-TTS-RUNTIME-PROOF-001` in a scoped worker branch.
- Physically retry wake/touch from the current setup-free, idle socket state:
  `紫悦`, `紫悦紫悦`, `你好紫悦`, and `小紫悦`.
- Continue to keep full PRD/Xiaozhi realtime parity red until real ASR runtime,
  real TTS runtime, stock `/v1/xiaozhi` trace, wake, touch/barge-in, and
  physical behavior are all proven.

已知风险和阻塞点:

- Setup/welcome should not be reopened unless it physically regresses again.
- The failed audit workers created no usable evidence; future workers must
  restate branch/HEAD/dirty and boundaries.
- `T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` remains a truthful blocker because
  `A21_SHERPA_ONNX_ASR_MODEL_DIR` was missing.
- `T-STREAMING-TTS-RUNTIME-PROOF-001` must not promote static env/config or
  fake provider tests as real provider/runtime or physical StackChan acceptance.

下一轮建议动作:

1. Commit this recovery/state/plan update.
2. Dispatch a scoped implementation worker for
   `T-STREAMING-TTS-RUNTIME-PROOF-001`.
3. In parallel, keep the physical validation checklist focused on wake/touch
   from idle socket; do not change the setup path unless the operator reports a
   new welcome regression.

测试/构建/运行结果:

- `git status --short --branch`: branch
  `codex/a21-hardware-window-20260602-stackchan-prd`; modified
  `docs/project_state_machine.md`; new
  `docs/plans/2026-06-03-streaming-tts-runtime-proof.md`.
- `git diff --check`: passed.
- No code tests, provider execution, Gateway restart, firmware build, flash,
  NVS write, serial access, V21 execution, or audio playback was performed.

如果中途失败，记录失败位置和原因:

- Read-only worker threads `019e8a9e-02dc-7cf2-9e3c-0bb159c4e00b`,
  `019e8a9e-02da-7b41-9cec-0e9a767fe16b`, and
  `019e8a9e-02d9-7453-a475-2df57956b865` ended with `systemError`, so their
  results were discarded.

## 2026-06-03 07:49 CST - Worker Completes Streaming TTS Runtime Smoke

本轮目标:

- Execute scoped transition `T-STREAMING-TTS-RUNTIME-PROOF-001`.
- Add redacted `a21 streaming-tts-runtime-smoke` and
  `make streaming-tts-runtime-smoke` without real provider execution unless
  `--execute` is explicit.

实际完成内容:

- Added a redacted streaming TTS runtime smoke report schema.
- Default no-execute path writes `status=blocked` with finding
  `execute_flag_required`.
- `--execute` with incomplete env writes stable env names only:
  `A21_TTS_FAST_PROFILE`, `A21_DOUBAO_API_KEY`, `A21_DOUBAO_TTS_MODEL`, and
  `A21_DOUBAO_TTS_VOICE`.
- Fake realtime tests prove `tts_session.update`, `input_text.append`, and
  `input_text.done` are sent, the first provider audio delta is observed while
  the stream is open, and at least one exact 60 ms PCM16 mono chunk is counted
  without a WAV/file boundary.
- Added a tiny Doubao realtime TTS provider-session read seam for the smoke.
- Ran the safe default Make target; it wrote
  `reports/a21-streaming-tts-runtime-smoke-20260603-074721-1780444041851710000.json`
  with `status=blocked` and finding `execute_flag_required`.

修改过的文件:

- `Makefile`
- `internal/app/app_plan_execute.go`
- `internal/app/streaming_tts_runtime_smoke.go`
- `internal/app/streaming_tts_runtime_smoke_test.go`
- `internal/providers/doubao_realtime_tts_provider.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Real Doubao realtime TTS provider execution remains blocked until an operator
  explicitly runs `a21 streaming-tts-runtime-smoke --execute` with complete env.
- Physical `/v1/xiaozhi` / StackChan acceptance remains a later transition.

已知风险和阻塞点:

- Runtime result is `blocked`, stable finding `execute_flag_required`.
- Fake realtime proof is test evidence only and must not be promoted to real
  provider runtime or physical PRD acceptance.

下一轮建议动作:

1. If operator authorizes provider cost/network use, run
   `a21 streaming-tts-runtime-smoke --execute --output-dir reports` with
   complete Doubao realtime TTS env.
2. Feed an accepted executed TTS runtime report into the next Xiaozhi realtime
   parity gate without claiming physical acceptance.

测试/构建/运行结果:

- Red test first:
  `go test ./internal/app -run TestRunStreamingTTSRuntimeSmoke -count=1`
  failed before implementation because `streamingTTSRuntimeSmokeDialer` and
  `streamingTTSRuntimeSmokeReport` were undefined.
- Focused app tests:
  `go test ./internal/app -run TestRunStreamingTTSRuntimeSmoke -count=1`
  passed.
- Focused provider tests:
  `go test ./internal/providers -run 'TestDoubaoRealtimeTTS|TestDoubaoRealtimeTTSTTSAdapterStreamsProviderDeltasAsDownlinkChunks' -count=1`
  passed.
- Broader focused app/provider tests:
  `go test ./internal/app -run 'TestRunStreamingTTSRuntimeSmoke|TestRunLocalASRStreamingSmoke|TestXiaozhiStreamingProviderReadiness' -count=1`
  passed.
  `go test ./internal/providers -run 'TestDoubaoRealtimeTTS|TestDoubaoRealtimeTTSTTSAdapterStreamsProviderDeltasAsDownlinkChunks|TestVoicePipelineAdaptersFromEnv.*Doubao' -count=1`
  passed.
- Runtime smoke command:
  `make streaming-tts-runtime-smoke` returned nonzero as expected for the
  no-execute blocker and wrote the report named above.
- Final `git diff --check`: passed after this handoff entry.
- Final `make verify`: passed.
- No ASR, LLM, V21, `/v1/xiaozhi/say`, Gateway start/stop, provider execution,
  physical device path, firmware build, flash, NVS write, serial access, or
  audio playback was performed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The Make target failure is expected default blocker
  behavior because `--execute` was not supplied.

## 2026-06-03 07:56 CST - Control Integrates Streaming TTS Runtime Smoke

本轮目标:

- Review and integrate worker branch `codex/a21-streaming-tts-runtime-proof-001`.
- Keep the result below real provider/runtime and physical StackChan PRD
  acceptance unless explicit executed evidence exists.
- Fold read-only Xiaozhi parity audits back into the next-transition queue.

实际完成内容:

- Cherry-picked worker commit `37ffcca feat(a21): add streaming tts runtime
  smoke` into control branch as `0d2fc73`.
- Reviewed the implementation boundary: default path requires no provider call,
  `--execute` is the only real runtime path, reports use stable findings and
  do not record credentials, user text, provider output text, raw/base64 audio,
  full URLs, proxy values, or absolute paths.
- Confirmed read-only protocol audit `019e8ab7-3467-7d13-8541-a45abb062a6d`
  found A21 now has stock-shaped `/v1/xiaozhi` websocket/Opus and streaming ASR
  hooks, but real Sherpa ASR remains blocked on `model_dir_missing`, TTS runtime
  was only static before this worker, and physical stock turn ordering remains
  unaccepted.
- Confirmed read-only endpoint audit `019e8ab7-7795-7252-901e-b46aab40ce8e`
  recommended incremental migration inside the official-compatible product lane,
  not a wholesale rewrite. Highest-risk endpoint gap remains wake parity because
  A21 disables the stock AFE/HiStackChan WakeNet path and relies on custom
  MultiNet phrases that still need physical proof.

修改过的文件:

- `docs/agent_handoff_log.md`

当前未完成事项:

- Real Doubao realtime TTS runtime proof requires explicit operator
  authorization to run `a21 streaming-tts-runtime-smoke --execute --output-dir
  reports` with complete env.
- Full target chain remains incomplete: real ASR runtime, streaming LLM turn,
  stock `/v1/xiaozhi` physical ordering, wake from idle, touch/barge-in, and
  physical Opus playback evidence are still open.

下一轮建议动作:

1. If provider execution is authorized, run the TTS runtime smoke with
   `--execute` and complete env, still host-only and below physical acceptance.
2. Dispatch `T-WAKE-004-AFE-VS-CUSTOM-PARITY` as a bounded product-lane worker
   or physical operator checklist: no provider/V21, no bare `xiaozhi.bin`, no
   gain/TTS changes, acceptance by idle wake logs and false-wake rejection.
3. Keep `/v1/xiaozhi/say`, host loopback, static provider-shape gates, and fake
   realtime tests out of PRD/full-realtime acceptance.

测试/构建/运行结果:

- Main control review:
  `git show --stat --oneline 37ffcca`, `git show --name-only 37ffcca`, and
  `git diff --check 02cbb21..37ffcca` passed/clean in the worker worktree.
- Main branch focused tests after cherry-pick:
  `go test ./internal/app -run 'TestRunStreamingTTSRuntimeSmoke|TestRunXiaozhiStreamingProviderReadiness|TestRunLocalASRStreamingSmoke' -count=1`
  passed.
- Main branch provider focused tests after cherry-pick:
  `go test ./internal/providers -run 'TestDoubaoRealtimeTTS|TestVoicePipeline|TestStreamingTTS' -count=1`
  passed.
- Main branch `git diff --check HEAD~1..HEAD`: passed.
- Main branch `make verify`: passed.
- No provider execution, Gateway start/stop, ASR/LLM/V21 execution,
  `/v1/xiaozhi/say`, physical device path, firmware build, flash, NVS write,
  serial access, or audio playback was performed by control integration.

如果中途失败，记录失败位置和原因:

- No unresolved integration failure. `T-STREAMING-TTS-RUNTIME-PROOF-001` is
  integrated as a truthful blocker, not runtime/physical acceptance.

## 2026-06-03 08:22 CST - Control Cross-Checks Xiaozhi Realtime Convergence

本轮目标:

- Recover the main control thread after integrating the streaming TTS runtime
  smoke.
- Re-run strict read-only cross-checks across Xiaozhi protocol, endpoint
  parity, Gateway/provider runtime gaps, and architecture reuse strategy.
- Choose the next scoped implementation transition without redefining full
  Xiaozhi realtime success around static gates, `/say`, WAV, host loopback,
  mock tests, or plan-only progress.

实际完成内容:

- Confirmed main branch `codex/a21-hardware-window-20260602-stackchan-prd` at
  HEAD `188b341`, clean.
- Collected final structured reports from four read-only workers:
  - Protocol/state worker `019e8ac3-c9f3-7cc3-b8a1-c27cc2748168`.
  - Endpoint/HAL/wake worker `019e8ac3-c9f7-7721-9f6c-1bce1e69af4c`.
  - Gateway/provider runtime gap worker `019e8ac3-c9f6-7350-a66e-e51dcdc8109e`.
  - Architecture strategy worker `019e8ac3-c9fa-7ed0-8b61-625a418a84c2`.
- Cross-check conclusion:
  - Immediate product lane can stay WebSocket/Opus; MQTT+UDP is a planned
    transport-parity gap.
  - Endpoint parity risk is concentrated in custom wake vs official AFE/WakeNet
    and the parked direct-Xiaozhi path bypassing normal Mooncake/AppLauncher
    lifecycle.
  - Runtime host blocker is now sharper: ASR partials exist as evidence markers
    but do not yet drive LLM/TTS before ASR final/listen stop.
  - Architecture direction remains incremental A21 convergence using Xiaozhi
    firmware/protocol/audio-service patterns. Full server-stack embedding would
    need an ADR and must preserve A21 Go/provider/V21 boundaries.
- Added plan
  `docs/plans/2026-06-03-xiaozhi-asr-partial-to-llm-realtime-bridge.md`.
- Updated the full realtime convergence plan and state machine to make
  `T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001` the next scoped
  host-side implementation candidate.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-asr-partial-to-llm-realtime-bridge.md`
- `docs/plans/2026-06-03-xiaozhi-full-realtime-voice-convergence.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Full Xiaozhi realtime objective is still incomplete.
- Real Sherpa streaming ASR remains blocked by missing model/config evidence.
- Real Doubao realtime TTS runtime remains blocked until explicit operator
  authorization and complete env for `--execute`.
- Physical product evidence still needs wake or labeled tap trigger, stock
  `/v1/xiaozhi` trace ordering, device playback, touch/barge-in, and idle
  recovery.

下一轮建议动作:

1. Dispatch scoped worker
   `T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001`.
2. Worker must not build/flash firmware, start/stop Gateway, call providers/V21,
   play audio, touch NVS/serial/hardware, or use `/v1/xiaozhi/say` as
   acceptance.
3. Acceptance is host-side ordered trace/test evidence only, below physical PRD
   acceptance.

测试/构建/运行结果:

- No tests run yet for this documentation/planning update.
- No provider execution, Gateway start/stop, ASR/LLM/V21 execution,
  `/v1/xiaozhi/say`, physical device path, firmware build, flash, NVS write,
  serial access, or audio playback was performed.

如果中途失败，记录失败位置和原因:

- No failure. This is a control-state update and worker-routing step only.

## 2026-06-03 08:38 CST - Worker Completes ASR Partial To LLM Realtime Bridge

本轮目标:

- Execute scoped transition
  `T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001`.
- Prove host-side stock `/v1/xiaozhi` ASR partials can start LLM/TTS streaming
  before ASR final/listen stop, while keeping final/batch fallback intact.
- Tighten `xiaozhi-realtime-parity` so turn-buffered-only and fake traces are
  not confused with realtime candidate traces.

实际完成内容:

- Added `VoicePipelineRequest.ASRTranscriptSource` with explicit
  `streaming_partial` vs `streaming_final` semantics.
- Gateway now records the first streaming ASR partial, starts exactly one
  workmate voice-pipeline answer task from that partial while the stock
  `/v1/xiaozhi` turn is still listening, and skips a duplicate final-start task
  after `listen.stop` or VAD/max-duration auto-stop.
- Partial-driven pipeline requests no longer synthesize `asr.final=0`; final
  transcript reuse remains the default source when a final transcript is
  available.
- `xiaozhi-realtime-parity` now requires `asr.stream.commit` for
  `xiaozhi_realtime_candidate` and blocks host-loopback fake markers in
  addition to `/say`, fast-companion, and local fallback markers.
- Updated the state machine to record this as host-side candidate evidence only,
  below real provider/runtime and physical PRD acceptance.

修改过的文件:

- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/xiaozhi_realtime_parity.go`
- `internal/app/xiaozhi_realtime_parity_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Full Xiaozhi realtime PRD acceptance is still incomplete.
- Real Sherpa streaming ASR remains blocked until model/helper runtime evidence
  is provided.
- Real streaming TTS runtime remains blocked until explicit provider execution
  is authorized and complete env is present.
- Physical product evidence still needs wake or labeled tap trigger, stock
  `/v1/xiaozhi` device trace ordering, audible device playback, touch/barge-in,
  and idle recovery.

已知风险和阻塞点:

- The bridge starts from first partial text and is host-side test evidence; it
  does not prove real ASR model quality, real provider latency, or physical
  user experience.
- No raw transcripts, raw/base64 audio, credentials, full URLs, proxy values,
  or absolute local paths were added to reports.

下一轮建议动作:

1. Run the real Sherpa streaming ASR no-audio/runtime unblock when model
   artifacts are available.
2. Run realtime TTS provider execution only with explicit operator
   authorization.
3. After real ASR/TTS runtime evidence, collect a physical stock `/v1/xiaozhi`
   parity trace and keep `/say`, host loopback, mock, and WAV/file paths out of
   acceptance.

测试/构建/运行结果:

- Red tests first:
  `go test ./internal/providers -run TestVoicePipelineRunStreamStartsLLMFromStreamingASRPartialBeforeFinal -count=1`
  failed before implementation because `ASRTranscriptSource` and
  `VoicePipelineASRTranscriptSourcePartial` were undefined.
- Red parity tests first:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity(ClassifiesRealtimeCandidateOrdering|DoesNotAcceptRealtimeOrderingWithoutASRCommit|BlocksHostLoopbackMarkers)' -count=1`
  failed because no-commit and host-loopback traces were accepted as realtime.
- Red Gateway test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal -count=1`
  failed because `xiaozhi.voice_pipeline.start` did not occur before
  `listen.stop`.
- Focused provider tests passed:
  `go test ./internal/providers -run 'TestVoicePipelineRunStreamStartsLLMFromStreamingASRPartialBeforeFinal|TestVoicePipelineRunStreamDeliversDoneAfterBufferedChunksDrain|TestVoicePipelineRunStreamCarriesFallbackReportOnAudioChunks|TestMockStreamingASRAdapterEmitsPartialOnFrameAndFinalOnCommit' -count=1`.
- Focused app tests passed:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity(ClassifiesRealtimeCandidateOrdering|DoesNotAcceptRealtimeOrderingWithoutASRCommit|BlocksHostLoopbackMarkers|BlocksFakeSayPath|DoesNotAcceptTransportOnlyTrace)' -count=1`.
- Focused Gateway tests passed:
  `go test ./internal/gateway -run 'TestXiaozhiWebSocketASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal|TestXiaozhiWebSocketStreamingASRStartsBeforeListenStop|TestXiaozhiWebSocketStreamsVoicePipelineAnswerChunks' -count=1`.
- Broader relevant package tests passed:
  `go test ./internal/providers -count=1`;
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity|TestRunXiaozhiStreamingProviderReadiness|TestRunStreamingTTSRuntimeSmoke|TestRunLocalASRStreamingSmoke' -count=1`;
  `go test ./internal/gateway -count=1`.
- Required verification passed before this handoff entry:
  `git diff --check`; `make verify`.
- No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host
  loopback, WAV/file acceptance path, firmware build, flash, NVS/serial access,
  hardware action, or audio playback was performed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The only failures were intentional red tests before
  implementation and one updated legacy Gateway assertion that previously froze
  turn-buffered behavior.

## 2026-06-03 08:41 CST - Control Integrates ASR Partial Bridge

本轮目标:

- Integrate worker `019e8ade-4fd4-7032-b508-2b786b71c162` into the main A21
  control branch.
- Preserve the result as host-side ordered evidence only, below real
  provider/runtime and physical StackChan PRD acceptance.

实际完成内容:

- Applied the worker diff from worktree
  `/Users/jiyurun/.codex/worktrees/4d95/New project` onto main branch
  `codex/a21-hardware-window-20260602-stackchan-prd`.
- Revalidated the same scope on main: Gateway/provider/parity tests plus full
  `make verify`.
- Confirmed the transition remains honest: it proves the host-side bridge and
  tighter gates, not real model/provider execution or physical acceptance.

修改过的文件:

- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/xiaozhi_realtime_parity.go`
- `internal/app/xiaozhi_realtime_parity_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Real Sherpa streaming ASR runtime/model proof.
- Authorized real streaming TTS provider execution.
- Physical stock `/v1/xiaozhi` trace with wake or labeled tap trigger, audible
  playback, touch/barge-in, and idle recovery.

测试/构建/运行结果:

- Main branch focused provider tests passed:
  `go test ./internal/providers -count=1`.
- Main branch focused app tests passed:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity|TestRunXiaozhiStreamingProviderReadiness|TestRunStreamingTTSRuntimeSmoke|TestRunLocalASRStreamingSmoke' -count=1`.
- Main branch focused Gateway tests passed:
  `go test ./internal/gateway -count=1`.
- Main branch `git diff --cached --check`: passed.
- Main branch `make verify`: passed.
- No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host
  loopback, WAV/file acceptance path, firmware build, flash, NVS/serial access,
  hardware action, or audio playback was performed by control integration.

如果中途失败，记录失败位置和原因:

- No unresolved integration failure.

## 2026-06-03 08:52 CST - Control Unblocks Real Sherpa Streaming ASR Smoke

本轮目标:

- Continue the full Xiaozhi realtime convergence after the host-side ASR
  partial bridge.
- Replace the stale `model_dir_missing` ASR blocker with current local evidence
  if the repo-local streaming Zipformer cache is usable.
- Keep the work host-local: no provider/V21, no Gateway restart, no hardware,
  no firmware, no `/v1/xiaozhi/say`, no audio playback.

实际完成内容:

- Verified the canonical local assets exist:
  - `scripts/a21_sherpa_onnx_streaming_asr_session.py`
  - `.a21-tools/sherpa-onnx-venv/bin/python`
  - `.a21-tools/sherpa-onnx-asr-models/sherpa-onnx-streaming-zipformer-zh-int8-2025-06-30`
- Confirmed the helper can load the real model and emit `ready`/`final` for an
  in-memory PCM16 frame.
- Added default canonical discovery to:
  - `a21 local-asr-streaming-smoke`
  - `VoicePipelineAdaptersFromEnv` when `A21_ASR_LOCAL_PROFILE` explicitly
    selects `sherpa_onnx_streaming`
  - `xiaozhi-streaming-provider-readiness`
- `make local-asr-streaming-smoke` now passes without extra env on this
  machine and writes
  `reports/a21-local-asr-streaming-smoke-20260603-085636-1780448196060587000.json`.
- `A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming a21
  xiaozhi-streaming-provider-readiness` now marks ASR ready from the canonical
  cache and remains blocked only on mock LLM/TTS in the default env; latest
  report:
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-085641-1780448201880789000.json`.

修改过的文件:

- `internal/app/local_asr_streaming_smoke.go`
- `internal/app/local_asr_streaming_smoke_test.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Full Xiaozhi realtime PRD acceptance is still incomplete.
- Real streaming TTS provider runtime execution remains blocked until explicit
  authorization and complete env.
- Physical stock `/v1/xiaozhi` evidence still needs wake or labeled tap
  trigger, real trace ordering, audible device playback, touch/barge-in, and
  idle recovery.

测试/构建/运行结果:

- Red test first:
  `go test ./internal/app -run TestRunLocalASRStreamingSmokeDiscoversCanonicalLocalModelCache -count=1`
  failed before implementation with `streaming_helper_missing`.
- Red provider test first:
  `go test ./internal/providers -run TestVoicePipelineAdaptersFromEnvDiscoversCanonicalSherpaStreamingCache -count=1`
  failed before implementation with `sherpa-onnx streaming ASR helper is not configured`.
- Red readiness test first:
  `go test ./internal/app -run TestXiaozhiStreamingProviderReadinessAcceptsCanonicalSherpaStreamingCache -count=1`
  failed before implementation because ASR was still reported missing.
- Focused tests passed:
  `go test ./internal/app -run 'TestRunLocalASRStreamingSmoke(DiscoversCanonicalLocalModelCache|WritesRedactedPassReport|RecordsMissingModelBlocker|RecordsSherpaPackageBlocker)' -count=1`;
  `go test ./internal/providers -run 'TestVoicePipelineAdaptersFromEnv(DiscoversCanonicalSherpaStreamingCache|WiresSherpaStreamingSubprocessHelper|SelectsSherpaONNXStreamingASR)|TestLocalSherpaONNXStreamingASRAdapterRequiresConfiguredSession' -count=1`;
  `go test ./internal/app -run 'TestXiaozhiStreamingProviderReadiness' -count=1`.
- Runtime host-local smoke:
  `make local-asr-streaming-smoke` passed.
- Static readiness:
  `A21_ASR_LOCAL_PROFILE=sherpa_onnx_streaming go run ./cmd/a21 xiaozhi-streaming-provider-readiness --output-dir reports`
  returned blocked as expected, with ASR ready and default LLM/TTS still mock.
- No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host
  loopback, WAV/file acceptance path, firmware build, flash, NVS/serial access,
  hardware action, or audio playback was performed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The only failures were intentional red tests and the
  expected readiness block on mock LLM/TTS.

## 2026-06-03 09:06 CST - Xiaozhi Realtime Parity Real Profile Evidence

本轮目标:

- Continue the persistent full Xiaozhi realtime convergence goal without
  claiming acceptance from a narrower slice.
- Follow the user's instruction to use multiple subthreads for read-only
  Xiaozhi/A21 inspection before the next implementation cut.
- Harden the realtime parity gate so a stock-shaped, well-ordered trace cannot
  pass as `xiaozhi_realtime_candidate` unless it also proves real streaming
  ASR/LLM/TTS profile classes.
- Keep scope host-local and no-execute: no provider/V21, no Gateway lifecycle,
  no `/v1/xiaozhi/say`, no firmware/hardware/audio playback.

子线程/只读审查:

- Protocol explorer `019e8afe-6a0f-7f23-8c4c-8cfcbacde774` read upstream
  Xiaozhi docs plus local protocol/firmware material. Conclusion: A21's
  WebSocket JSON plus binary Opus shell matches the immediate Xiaozhi product
  lane direction, but full voice parity still requires real streaming
  ASR/LLM/TTS, barge-in, wake, playback, and idle recovery evidence.
- Runtime explorer `019e8afe-8dc1-75c0-9239-47c44ef18379` read A21 Gateway,
  transport, audio, provider, and tests. Conclusion: current `/v1/xiaozhi`
  ingress/downlink is real Opus, but the middle still has mock defaults,
  batch/WAV fallback seams, a blocking ASR commit path, and mixed state flags;
  the next state-machine cut should make commit/final handling non-blocking and
  explicit.

实际完成内容:

- Added plan
  `docs/plans/2026-06-03-xiaozhi-realtime-parity-real-profile-evidence.md`.
- Added Gateway profile-class trace markers for Xiaozhi voice-pipeline turns:
  - `xiaozhi.voice_pipeline.asr.real_streaming`
  - `xiaozhi.voice_pipeline.llm.real_streaming`
  - `xiaozhi.voice_pipeline.tts.real_streaming`
  - blocker markers for mock, batch, and file-boundary stages
- Added parity-gate counts and stage availability:
  - `asr.real_streaming_profile`
  - `llm.real_streaming_profile`
  - `tts.real_streaming_profile`
  - `realtime_profile.blocker_absent`
- `xiaozhi-realtime-parity` now requires all three real streaming profile
  markers and no profile blockers before returning `xiaozhi_realtime_candidate`.
  Ordered traces without those markers downgrade to
  `turn_buffered_xiaozhi_candidate` and report
  `xiaozhi_realtime_real_profile_evidence_missing`.
- Updated `docs/project_state_machine.md` with the completed transition and
  next recommended state-machine cut.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-realtime-parity-real-profile-evidence.md`
- `internal/app/xiaozhi_realtime_parity.go`
- `internal/app/xiaozhi_realtime_parity_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Full Xiaozhi realtime PRD acceptance remains incomplete.
- Real streaming TTS provider execution still needs explicit authorization and
  complete env.
- A physical stock `/v1/xiaozhi` trace still needs wake or labeled tap trigger,
  real profile markers, audible playback, touch/barge-in, and idle recovery.
- Runtime explorer recommends the next narrow transition: make streaming ASR
  commit/final handling explicit and non-blocking so the WebSocket read loop is
  not held by `commitXiaozhiStreamingASR`.

测试/构建/运行结果:

- Red app test first:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity(ClassifiesRealtimeCandidateOrdering|BlocksRealtimeOrderingWithoutRealProfileEvidence)' -count=1`
  failed because the gate accepted a realtime-looking trace without real
  profile markers as `xiaozhi_realtime_candidate`.
- Red Gateway test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal -count=1`
  failed because mock partial-bridge turns lacked mock/blocker profile markers.
- Focused tests passed after implementation:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity' -count=1`;
  `go test ./internal/gateway -run 'TestXiaozhiWebSocketASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal|TestXiaozhiWebSocketStreamingASRStartsBeforeListenStop|TestXiaozhiVoicePipeline' -count=1`.
- No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host
  loopback runtime, firmware build, flash, NVS/serial access, hardware action,
  or audio playback was performed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The only failures were intentional red tests.

## 2026-06-03 09:19 CST - Xiaozhi Nonblocking ASR Commit

本轮目标:

- Continue the persistent Xiaozhi realtime convergence goal after real-profile
  parity hardening.
- Implement the next state-machine cut from the runtime read-only audit:
  streaming ASR commit/final handling must not block the `/v1/xiaozhi`
  WebSocket read loop.
- Keep this host-local: no provider/V21, no Gateway lifecycle, no
  `/v1/xiaozhi/say`, no firmware/hardware/audio playback.

实际完成内容:

- Added plan `docs/plans/2026-06-03-xiaozhi-nonblocking-asr-commit.md`.
- Replaced synchronous `listen.stop` / VAD auto-stop ASR commit handling with
  async streaming-ASR commit/final handling.
- While commit is pending, the WebSocket read loop can process abort/barge-in.
- If a streaming ASR final arrives and no partial-driven answer already
  started, the async path starts the voice pipeline from the streaming final.
- Removed the unused synchronous commit helper so future code is less likely to
  reintroduce the blocking path.
- Updated `docs/project_state_machine.md` with the completed transition and the
  next live evidence direction.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-nonblocking-asr-commit.md`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Full Xiaozhi realtime PRD acceptance remains incomplete.
- Real streaming TTS provider execution still needs explicit authorization and
  complete env.
- Physical stock `/v1/xiaozhi` trace still needs wake or labeled tap trigger,
  real profile markers, audible playback, touch/barge-in, and idle recovery.
- Next useful transition: feed the canonical real Sherpa streaming ASR helper
  through a stock `/v1/xiaozhi` host-local trace, then keep realtime parity
  blocked until real streaming LLM/TTS and physical evidence exist.

测试/构建/运行结果:

- Red Gateway test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketListenStopDoesNotBlockAbortWhileStreamingASRCommitPending -count=1`
  failed because `xiaozhi.abort.received` was not recorded while ASR commit was
  pending.
- Focused Gateway tests passed after implementation:
  `go test ./internal/gateway -run 'TestXiaozhiWebSocket(ListenStopDoesNotBlockAbortWhileStreamingASRCommitPending|StreamingASRFinalStartsPipelineWithoutBatchFallback|StreamingASRStartsBeforeListenStop|ASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal)|TestXiaozhiVoicePipeline' -count=1`.
- Focused app parity tests passed:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity' -count=1`.
- No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host
  loopback runtime, firmware build, flash, NVS/serial access, hardware action,
  or audio playback was performed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The only failure was the intentional red test.

## 2026-06-03 10:19 CST - Xiaozhi Stock STT And Wake Preroll

本轮目标:

- Continue the persistent Xiaozhi realtime convergence goal with a
  protocol-faithful, host-local cut after source-reading Xiaozhi behavior.
- Improve stock `/v1/xiaozhi` message/audio fidelity without provider/V21
  execution, Gateway lifecycle changes, `/v1/xiaozhi/say`, host-loopback
  runtime acceptance, firmware, hardware, or audio playback.

实际完成内容:

- Added plan
  `docs/plans/2026-06-03-xiaozhi-official-protocol-source-read-and-next-cut.md`.
- Dispatched two read-only source explorers:
  - Explorer A confirmed stock Xiaozhi is device-driven for `listen`; server
    should send `hello`, `stt`, `llm`, `tts`, `mcp`, and related messages, not
    rely on server-to-device `listen` for stock physical devices.
  - Explorer B identified wake-word pre-roll Opus as the biggest immediate
    host-side audio mismatch; upstream Xiaozhi can send audio before/around
    `listen/detect`, while A21 had dropped binary Opus when not listening.
- Explorer C could not be spawned because the thread limit was reached; the
  main thread performed the A21 gap read locally.
- Added stock `stt` WebSocket emission before `tts/start` when streaming ASR
  partial/final text exists, without storing transcript text in traces.
- Added bounded true-idle wake pre-roll buffering: up to five decoded Opus
  frames received before `listen/start` are attached to the next turn, counted
  in audio ingress evidence, and fed to streaming ASR when active.
- Preserved no-speech/host-say cooldown behavior so cooldown or current-turn
  Opus remains `xiaozhi.opus_frame.ignored_not_listening` rather than false
  wake pre-roll.
- Updated `docs/project_state_machine.md` with the completed transition,
  remaining PRD blockers, and the next physical/runtime evidence direction.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-official-protocol-source-read-and-next-cut.md`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Full Xiaozhi realtime PRD acceptance remains incomplete.
- Real streaming provider execution is still not done.
- Physical stock `/v1/xiaozhi` trace still needs wake or labeled tap trigger,
  real streaming ASR/LLM/TTS profile markers, audible playback,
  touch/barge-in, and idle recovery.
- Next useful transition: with explicit authorization, either collect a
  physical stock `/v1/xiaozhi` trace and run `xiaozhi-realtime-parity`, or run
  the real streaming TTS provider runtime path with complete env.

测试/构建/运行结果:

- Red Gateway test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketStreamingASRFinalSendsStockSTTBeforeTTS -count=1`
  failed because the first post-ASR message was `tts/start` instead of stock
  `stt`.
- Red Gateway test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketWakePrerollOpusFeedsNextTurn -count=1`
  failed because the voice pipeline did not receive the pre-listen Opus frame.
- Focused Gateway tests passed:
  `go test ./internal/gateway -run 'TestXiaozhiWebSocket(WakePrerollOpusFeedsNextTurn|StreamingASRFinalSendsStockSTTBeforeTTS|StreamingASRFinalStartsPipelineWithoutBatchFallback|ListenStopRunsVoicePipelineAndSendsPacedOpus|StreamingASRStartsBeforeListenStop|ASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal)' -count=1`.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Focused app parity/readiness tests passed:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity|TestXiaozhiStreamingProviderReadiness' -count=1`.
- No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`,
  host-loopback runtime acceptance, firmware build, flash, NVS/serial access,
  hardware action, or audio playback was performed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The failures above were intentional red tests.
