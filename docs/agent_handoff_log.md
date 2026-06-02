# A21 Agent Handoff Log

Status: active handoff document.
Last updated: 2026-06-02.

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
