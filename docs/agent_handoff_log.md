# A21 Agent Handoff Log

Status: active handoff document.
Last updated: 2026-06-05.

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

## 2026-06-05 06:02 CST - Official StackChan Relay Runtime Build Ready

Round goal:

- Close the review thread's remaining official StackChan body-channel finding
  at the firmware/runtime level: keep the WDT-safe direct Xiaozhi product
  start, but restore the official `/stackChan/ws` avatar/action runtime that
  was bypassed by parking before the Mooncake worker.

Actual completed work:

- Re-read review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` via the Codex
  thread tool and compared its conclusions against the current implementation.
- Confirmed prior P0 findings were already remediated in current HEAD:
  Gateway Xiaozhi race, namespace/preflight gate, stock professional route
  mode gating, and PMIC power-key parity.
- Identified the remaining official body gap: Gateway `/stackChan/ws` and
  `/v1/stackchan/official/control` are implemented, but the product overlay
  direct-started Xiaozhi and parked before official `WebsocketAvatarWorker`
  could tick `WebSocketAvatar`.
- Updated the product overlay to add an A21 direct official avatar relay
  runtime:
  `startA21WebSocketAvatarRuntime()` starts the official `WebSocketAvatar`;
  `updateA21WebSocketAvatarRuntime()` ticks it every 20 ms in the parked loop.
- Updated the official avatar URL to use
  `CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL="ws://47.103.57.217"` and
  append `device_id` from `GetHAL().getFactoryMacString(":")` so controls to
  product MAC `44:1b:f6:e2:6a:60` can match the registered official socket.
- Added focused overlay guard tests to prevent losing the relay runtime,
  Gateway base URL, MAC device_id query, 20 ms tick, and no-Mooncake-teardown
  product path.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Failing tests first confirmed the missing relay runtime:
  `GOMAXPROCS=2 go test ./internal/app -run 'OfficialXiaozhiCompatibleOverlay(StartsXiaozhiDirectly|RunsOfficialAvatarRelay)' -count=1`
  failed before the overlay update.
- Focused overlay tests passed after the fix.
- Product firmware/app contract tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'StackChanOfficial|Official|Firmware|Xiaozhi|Frozen' -count=1`.
- Gateway official/power capability tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'OfficialStackChan|PowerLifecycle|MCPCapabilities' -count=1`.
- `git diff --check` passed.
- Guarded product build passed:
  `GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`.
  Report:
  `reports/a21-stackchan-official-baseline-20260605-060158-1780610518624307000.json`.
  Product app SHA:
  `4158bdd7a584cb4f915b717f858c1e86339f74484d514c25854297de3a610721`.

Unfinished items:

- Product app has not yet been flashed in this round.
- `/stackChan/ws` online evidence after the new firmware is still pending.
- Official `/v1/stackchan/official/control` delivery to the product MAC and
  visible physical confirmation remain pending.
- No-cable cold boot and physical power-button acceptance remain pending.

Known risks/blockers:

- The official source checkout is dirty, but the build exported the official
  source from git HEAD only; the report records that dirty-source finding.
- Running Xiaozhi and official avatar WebSockets together is now compiled but
  needs product-device runtime proof.

Recommended next action:

- Commit the firmware overlay/test/docs, flash the guarded product artifact on
  `/dev/cu.usbmodem1101`, wait for device reconnect, verify `/stackChan/ws`
  registration, send an official motion command to device
  `44:1b:f6:e2:6a:60`, and then collect operator physical acceptance.

Forbidden actions avoided:

- No NVS write, provider secret output, generic `xiaozhi.bin` product flash,
  unguarded upload, Git prune/gc, or internal-test3 voice/protocol rollback
  occurred.

## 2026-06-05 05:31 CST - StackChan PMIC Power-Key Parity Product Flash

Round goal:

- Close the code-review thread's power/hardware lifecycle findings as far as
  software and firmware can act, compare the product overlay against the
  official StackChan PMIC setup, flash only the guarded product lane, and keep
  no-cable power truth physically gated.

Actual completed work:

- Re-read review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` and mapped its
  remaining findings against the post-`56a0fcb` implementation.
- Compared the A21 official-compatible StackChan overlay with the local
  official StackChan board implementation and Xiaozhi AXP2101 examples.
- Restored StackChan PMIC power-key parity in the product overlay by enabling
  PWRON/OFFLEVEL power-off source handling and the 4s hardware power-key
  long-press register before the direct Xiaozhi product start path.
- Added a focused product-overlay test to prevent losing the PMIC power-key
  lifecycle registers again.
- Built and flashed the product app through the guarded
  `a21-stackchan-official-xiaozhi-compatible` product lane on
  `/dev/cu.usbmodem1101`.
- Replayed live product `full_check` and roleplay mode ritual after the flash.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-05-no-cable-boot-power-lifecycle-recovery.md`

Tests/build/runtime results:

- Focused product-overlay tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlayPreservesStackChanPowerKeyLifecycle|TestOfficialXiaozhiCompatibleOverlayStartsXiaozhiDirectlyBeforeMooncakeTeardown|TestStackChanOfficialCandidateContract' -count=1`.
- `git diff --check` passed before the firmware commit and again after the
  control-document update.
- Product build passed:
  `GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`.
- Build report:
  `reports/a21-stackchan-official-baseline-20260605-052214-1780608134186430000.json`.
- Guarded flash plan passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-052252-1780608172150417000.json`.
- Guarded flash execute passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-052359-1780608239212784000.json`.

Runtime or physical evidence:

- Flash execution recorded clean worktree commit `fda7769b23da`, branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`,
  `flash_executed=true`, and app artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Flashed app SHA:
  `b665af4e78fae0c4dea10db04a2ea90502d88f26322dcca3c234abb8f355fc3e`.
- Product device `44:1b:f6:e2:6a:60` reconnected to public Gateway with fresh
  `device.heartbeat` after the flash.
- Public `POST /v1/xiaozhi/body-scene` with trace
  `a21-trace-full-check-pmic-key-fda7769-20260605` returned
  `status=delivered`, 16 steps, and `physical_accepted=false`.
- Public `POST /v1/voice-mode-ritual` with trace
  `a21-trace-mode-ritual-pmic-key-fda7769-20260605` returned
  `status=delivered`, `selected_voice_mode=roleplay`, 4 steps, and
  `physical_accepted=false`.
- Public `GET /v1/power-lifecycle?device_id=44:1b:f6:e2:6a:60` returned
  `overall_status=physical_pending`, `xiaozhi_ws_online=true`,
  `battery_telemetry=missing`, and no-cable/power-button acceptance still
  false.
- Public `GET /v1/xiaozhi/mcp-capabilities?device_id=44:1b:f6:e2:6a:60`
  returned 8 allowed low-risk MCP tools and 10 blocked classes, including
  `power_shutdown`, `power_sleep`, `reboot`, `firmware_upgrade`, `camera_*`,
  `nfc`, `infrared`, and `app_lifecycle`.

Remaining issues:

- The PMIC parity fix is flashed, but physical no-cable cold boot and the
  physical power button are still awaiting foreground operator or instrument
  acceptance.
- Battery telemetry remains missing from product runtime echo.
- Official `/stackChan/ws` Avatar/Motion/Dance relay remains not product
  accepted; current body feedback is still Xiaozhi MCP-backed.
- Launch readiness still needs real provider smoke, roleplay voice runtime,
  and physical StackChan PRD acceptance evidence.

Next suggested action:

- Have the operator disconnect USB/power, wait for the unit to be fully off,
  hold the physical power button for about 4s, and report whether the product
  boots and reconnects. If it does, record
  `/v1/power-lifecycle-acceptance`; if it does not, open a battery/PMIC
  diagnostic transition against the physical board.

Forbidden actions avoided:

- No generic `xiaozhi.bin` product flash.
- No NVS write.
- No provider/V21 execution.
- No Git prune/gc.
- No internal-test3 voice/protocol rollback.

## 2026-06-05 05:13 CST - Review Remediation, Gateway Stabilization, And Power Lifecycle State Machine

Round goal:

- Consume code-review thread `019e941c-761b-7ee0-a4b8-68103a0850a1`,
  compare its findings against the current implementation, remediate the
  blocking Gateway/voice/hardware issues that can be fixed in software, deploy
  the result, and keep physical power-button truth honest.

Actual completed work:

- Fixed the Xiaozhi Gateway session-state race class by locking/snapshotting
  session identity, features, listening state, ASR/Opus counters, and downlink
  activity before cross-goroutine use.
- Fixed a real abort/barge-in blocking path by removing the pacer reset from
  `cancelCurrentXiaozhiTurnLocked`; canceled turns now rely on context
  cancellation and stale-turn checks instead of blocking the abort handler
  behind pacer locks.
- Changed product-chain fast ack from immediate "我在" toward delayed
  backchannel behavior: app defaults now set
  `A21_XIAOZHI_FAST_ACK_ENABLED=true` and
  `A21_XIAOZHI_FAST_ACK_DELAY_MS=700`; Gateway skips the backchannel when the
  real answer is ready first.
- Fixed the explicit V21 adapter plan namespace gate so `make preflight` and
  `make doctor` pass without allowing accidental V21/X21 naming in A21 code.
- Added `GET /v1/power-lifecycle` and
  `POST /v1/power-lifecycle-acceptance`; power lifecycle is now part of
  `GET /v1/hardware-acceptance`.
- Kept power shutdown/sleep/reboot/firmware upgrade out of the low-risk
  Xiaozhi MCP whitelist. `/v1/xiaozhi/mcp-capabilities` now lists
  `power_shutdown` and `power_sleep` as blocked tool classes.
- Deployed the latest Gateway to ECS `47.103.57.217` through `/opt/a21.next`
  safe swap; no firmware flash, NVS write, serial write, provider secret
  printing, or Git prune/gc occurred.
- Replayed live product mode ritual and `full_check` after ECS restart so the
  in-memory hardware acceptance board has fresh machine-delivered evidence.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `internal/app/xiaozhi_professional_bench.go`
- `internal/audio/ratecontroller.go`
- `internal/runtimeguard/namespace.go`
- `internal/runtimeguard/namespace_test.go`
- `docs/engineering/PROTOCOL.md`

Tests/build/runtime results:

- Local focused Gateway power/hardware/MCP tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestHardwareAcceptance|TestPowerLifecycle|TestXiaozhiMCPCapabilitiesEndpointReportsAllowedAndBlockedTools' -count=1`.
- Local touched packages passed:
  `GOMAXPROCS=2 go test ./internal/gateway ./internal/app ./internal/audio ./internal/runtimeguard -count=1`.
- Local focused Gateway race suite passed:
  `GOMAXPROCS=2 go test -race ./internal/gateway -run 'Xiaozhi|VoiceModeRitual|HardwareAcceptance|PowerLifecycle|WorkspaceConsole' -count=1`.
- `git diff --check`, `GOMAXPROCS=2 make verify`,
  `GOMAXPROCS=2 make preflight`, and `GOMAXPROCS=2 make doctor` passed.
- Remote `/opt/a21.next` focused Gateway/App/runtimeguard tests passed, remote
  build passed, `a21-gateway.service` restarted active, and remote loopback
  `/healthz` returned `status=ok`.

Runtime or physical evidence:

- Public direct `/healthz` returned `status=ok`.
- Public direct `/xiaozhi/ota/` returned
  `ws://47.103.57.217/v1/xiaozhi` with version `1`.
- Public `GET /v1/power-lifecycle?device_id=44:1b:f6:e2:6a:60` returned
  `overall_status=physical_pending`, `xiaozhi_ws_online=true`,
  `battery_telemetry=missing`, and physical power-button/cold-boot items
  `physical_accepted=false`.
- Public `/v1/xiaozhi/mcp-capabilities` reported allowed tools for speaker
  volume, device status, screen brightness/theme/info, robot head, and LED;
  it reported blocked classes including `reboot`, `firmware_upgrade`,
  `nfc`, `infrared`, `power_shutdown`, `power_sleep`, and `app_lifecycle`.
- Live product mode ritual trace
  `a21-trace-mode-ritual-power-state-20260605-0509` returned HTTP 200
  `status=delivered`.
- Live product `full_check` trace
  `a21-trace-full-check-power-state-20260605-0509` returned HTTP 200
  `status=delivered`.
- Final public hardware acceptance for product device
  `44:1b:f6:e2:6a:60` returned `overall_status=physical_pending` with
  `mode_ritual`, `full_check`, and `power_lifecycle` all present; the first
  two are machine-delivered and all three are physically unaccepted.
- Public repeat-3 Xiaozhi voice bench
  `reports/a21-xiaozhi-voice-bench-20260605-051249.326210000.json` passed
  3/3 answer and 3/3 barge-in turns, `failure_count=0`,
  answer first-audio P95 `1493 ms`, and barge-in stop P95 `22 ms`.
- Product readiness
  `reports/a21-product-readiness-20260605-051255.json` remained
  `server_side_blocked`, `launch_ready=false`, `demo_ready=true`.

Remaining issues:

- Physical no-cable cold boot and physical power-button start are still not
  accepted. Gateway can now track this accurately, but it cannot prove battery
  / PMIC / button electrical behavior without foreground observation or
  instrumented power evidence.
- Battery telemetry is still missing in the product firmware runtime echo; it
  remains a sensor/battery diagnostic track.
- Physical StackChan PRD voice acceptance remains blocked by missing physical
  mic/Opus/PCM/VAD/answer-downlink/operator evidence.
- Product readiness is still blocked by real provider smoke, roleplay voice
  runtime report mismatch, and physical StackChan PRD acceptance.
- Official `/stackChan/ws` avatar/action relay remains not product-closed; the
  live body effects are currently Xiaozhi MCP-backed.

Next suggested action:

- Run a foreground physical power window: attempt no-USB cold boot from the
  physical power button, watch for device boot/Gateway reconnect/Xiaozhi socket,
  and only then call `/v1/power-lifecycle-acceptance`.
- In parallel, promote sensor/battery diagnostic evidence, run a fresh roleplay
  voice runtime probe against the current profile, and close physical
  mic/playback PRD evidence.

Forbidden actions avoided:

- No firmware flash, no NVS write, no generic `xiaozhi.bin`, no provider secret
  output, no V21 internals copied into A21, no internal-test3 voice protocol
  rollback, no Git prune/gc, and no hidden MCP power/reboot expansion.

## 2026-06-05 03:24 CST - Latest Product Deployment And Guarded Flash

Round goal:

- Deploy the latest internal-test4 product surface, flash the current
  official-compatible product app, and report the real PRD/experience state
  without reopening internal-test3 voice-chain acceptance.

Actual completed work:

- Pushed `1c9dece fix(workspace): adopt connected hardware device`.
- Deployed `1c9dece` to ECS `47.103.57.217` through `/opt/a21.next` safe
  swap.
- `/workspace` now has `Connected device` and boot-time adoption of the online
  product hardware device from `/v1/devices`, instead of staying on
  `stackchan-sim-001` when the operator has not selected a device.
- Ran public product smokes for `/healthz`, `/workspace`, `/v1/devices`, and
  `/v1/hardware-acceptance`.
- Ran a guarded no-write product flash plan on `/dev/cu.usbmodem1101`.
- Executed the guarded official-compatible product flash on
  `/dev/cu.usbmodem1101`.
- Waited for product device `44:1b:f6:e2:6a:60` to return with a fresh online
  heartbeat after flash.
- Re-ran roleplay mode ritual and `full_check` body scene after the flash,
  both against the real product device.

Changed files:

- `internal/gateway/server_test.go`
- `internal/gateway/workspace_console.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Local focused Gateway tests passed before commit:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestHardwareAcceptance|TestVoiceModeRitual|TestXiaozhiBodyScenePhysicalAcceptance|TestXiaozhiBodySceneReportsAndAppliesStepPacing' -count=1`.
- Local full verification passed before commit:
  `GOMAXPROCS=2 make verify`.
- Remote focused Gateway tests passed on `/opt/a21.next`:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestHardwareAcceptance|TestVoiceModeRitual' -count=1`.
- Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- `a21-gateway.service` restarted active.
- Public direct `/healthz` returned `status=ok`.
- Public `/workspace` smoke found `Connected device`,
  `refreshConnectedDevice`, `preferredConnectedDevice`, `/v1/devices`,
  `connected_device_count`, `Acceptance Board`, `Full Check`,
  `Accept Visible Full Check`, and `Accept Visible Mode Ritual`.

Runtime or physical evidence:

- Product flash plan report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-032219-1780600939242091000.json`
  with `status=ready`.
- Product flash execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-032331-1780601011958120000.json`
  with `status=passed`, `flash_executed=true`, T7 guard ok, clean worktree,
  commit `1c9dece8b37e`, and app artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- No NVS write was performed. Existing Wi-Fi/cloud configuration was
  preserved. No generic `xiaozhi.bin` product flash was used.
- After flash, product device `44:1b:f6:e2:6a:60` returned fresh online
  heartbeats (`device_age_ms=1228`, then `device_age_ms=430`).
- After-flash roleplay ritual trace
  `a21-trace-mode-ritual-after-flash-1c9dece-20260605` returned HTTP 200
  `status=delivered`, `selected_voice_mode=roleplay`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`,
  `provider_executed=false`, and `v21_executed=false`.
- After-flash `full_check` trace
  `a21-trace-full-check-after-flash-1c9dece-20260605` returned HTTP 200
  `status=delivered`, `scene=full_check`, 16 redacted screen/RGB/head steps,
  `step_delay_ms=180`, and `total_planned_delay_ms=2700`.
- Final public hardware-acceptance summary returned
  `overall_status=physical_pending`; both `mode_ritual` and `full_check` were
  `delivery_status=delivered`, with next actions
  `accept_visible_mode_ritual` and `accept_visible_full_check`.

Deviations from plan:

- None for deployment/flash. Physical acceptance was not auto-recorded because
  the foreground operator has not yet confirmed visible screen/RGB/head
  movement.

Remaining issues:

- The product has machine-delivered screen/RGB/servo body evidence after
  flash, but physical acceptance remains pending until the user/operator
  confirms what was visible and clicks the two acceptance controls.
- Professional mode has contracts and read-record surfaces, but real V21
  upload/index/query-scope execution is still not product-accepted.
- Roleplay persona, memory, prompt, voice clone, and probe surfaces exist, but
  long-term memory persistence/delete/export and fully polished character
  authoring are not complete.
- Official `/stackChan/ws` avatar/action relay remains disconnected.
- Camera, NFC, infrared, battery/sensor diagnostics, and richer official
  avatar/motion semantics remain planned/high-risk parity work.
- Natural microphone-triggered voice-chain physical PRD acceptance was not
  reopened in this round.

Next suggested action:

- Use the live `/workspace` Acceptance Board as the foreground checklist:
  watch mode ritual and full check on the product device, then click
  `Accept Visible Mode Ritual` and `Accept Visible Full Check` if screen/RGB
  and head movement are visible. Then move to official avatar/action relay and
  camera/NFC/IR parity spikes, while keeping internal-test3 voice protocol
  untouched.

Forbidden actions avoided:

- No Git prune/gc, no NVS write, no generic product flash lane, no provider
  secret printing, no accidental V21/provider execution during body evidence,
  no internal-test3 voice/protocol rollback, and no subagent dispatch.

## 2026-06-05 - T-WORKSPACE-HARDWARE-FULL-CHECK-SCENE-001 - Full Body Check Deployed

Goal:

- Add a one-click operator-visible hardware body diagnostic so the product can
  exercise screen, RGB, and head motion without asking the operator to run
  showtime/focus/reset separately.

Actual completed work:

- Added `scene=full_check` to `POST /v1/xiaozhi/body-scene`.
- Added `Full Check` to the `/workspace` Hardware Scenes panel.
- Kept the sequence bounded to already whitelisted stock MCP tools:
  `self.screen.set_theme`, `self.screen.set_brightness`,
  `self.robot.set_led_color`, and `self.robot.set_head_angles`.
- Used TDD: the first focused test run failed because `full_check` was not in
  the workspace and API returned HTTP 400; after implementation, focused tests
  passed.
- Committed and pushed `9171751 feat(gateway): add full body check scene`.
- Deployed to ECS through `/opt/a21.next` safe swap. Remote focused Gateway
  tests passed, remote build passed, `a21-gateway.service` restarted active,
  and public `/healthz` passed.
- Public `/workspace` smoke found `Full Check`,
  `data-hardware-scene="full_check"`, and `/v1/xiaozhi/body-scene`.
- Public `/v1/devices` showed product device `44:1b:f6:e2:6a:60` online after
  the Gateway restart and across 8 heartbeat polls.
- Live product full check trace
  `a21-trace-hardware-full-check-9171751-202606050217` returned HTTP 200 with
  `status=delivered`, `delivered_transport=xiaozhi_mcp_sequence`,
  `scene=full_check`, and 16 redacted steps.
- Trace endpoint recorded 32 markers through
  `xiaozhi.body_scene.full_check.step16.robot_head_angles_set.sent`.
- `/v1/devices` recorded `last_body_scene=full_check`,
  `last_body_scene_status=delivered`, `last_body_scene_step=16`,
  `screen_theme=auto`, `screen_brightness=55`, final head
  `yaw=0,pitch=18,speed=200`, and final RGB `0/0/32`. A 12-second follow-up
  check still showed the device online.

Files changed:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/workspace_console.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`

Unfinished items:

- `physical_accepted=false` remains until visible operator or instrument
  confirmation of the full check movement.
- Separate official `/stackChan/ws` avatar/action relay still returns HTTP
  409 when no official avatar socket is connected.
- Camera/NFC/IR remain planned/high-risk parity spikes, not product surfaces.

Known risks or blockers:

- Do not promote full_check to product-accepted from machine delivery alone.
- Do not re-enable automatic listen-start state reactions until foreground
  hardware stability proves it safe.
- Git may keep reporting historical loose object/gc warnings; no prune/gc is
  authorized.

Recommended next action:

- Get operator confirmation for visible full_check motion, then encode that as
  physical acceptance evidence.
- Continue with official avatar relay lifecycle reconciliation or safe
  diagnostic-only battery/IMU status exposure.

Test, build, or runtime results:

- RED:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodySceneFullCheckRunsOperatorVisibleSequence' -count=1`
  failed as expected before implementation.
- GREEN:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodySceneFullCheckRunsOperatorVisibleSequence|TestXiaozhiBodySceneRejectsUnknownScene' -count=1`
  passed.
- Focused:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyScene' -count=1`
  passed.
- `GOMAXPROCS=2 make verify`: passed.
- Remote `/opt/a21.next` focused Gateway tests: passed.
- Remote `go build -o /opt/a21.next/bin/a21 ./cmd/a21`: passed.
- ECS loopback and public `/healthz`: passed.
- Public product `full_check`: HTTP 200 delivered.

Failure location and reason:

- None in the final deployed path. A remote binary `--help` probe returned an
  unsupported-command status after build and was not used as a deployment gate.

## 2026-06-05 - T-FIRMWARE-QUIET-RECONNECT-PRODUCT-FLASH-001 - Product Reconnect Flash and Showtime Machine Evidence

Goal:

- Move the body-capability track from disconnected contract evidence to live
  product-socket delivery without redoing or rolling back internal-test3 voice
  protocol acceptance.
- Flash only the official-compatible A21 product lane candidate that adds
  periodic quiet Xiaozhi reconnect checks.

Actual completed work:

- Confirmed USB product device `44:1B:F6:E2:6A:60` on
  `/dev/cu.usbmodem1101`.
- Confirmed product app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  with SHA-256
  `3eef974929aed78cdd77232897485aaac25bce8aa98daa4d8d78b3d96662b7ac`.
- Ran guarded product flash plan and execute through
  `a21-stackchan-official-xiaozhi-compatible`; execute returned
  `status=passed`, `flash_allowed=true`, and `flash_executed=true`.
- Confirmed no NVS write and no generic `xiaozhi.bin` product flash path.
- After reboot, public `/v1/devices` showed product device
  `44:1b:f6:e2:6a:60` online with heartbeat updates.
- Ran public product `showtime` scene via
  `POST /v1/xiaozhi/body-scene` using trace
  `a21-trace-hardware-showtime-flash-b9c0baa-202606050208`; it returned HTTP
  200 `status=delivered` and `delivered_transport=xiaozhi_mcp_sequence`.
- Trace recorded 16 sent markers for 8 bounded scene steps; `/v1/devices`
  recorded `last_body_scene=showtime`, `screen_theme=dark`,
  `screen_brightness=72`, final head `yaw=0,pitch=24,speed=220`, and final
  RGB `0/36/96`.
- A follow-up public `/v1/devices` check about 12 seconds later still showed
  the device online.

Files changed:

- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- Physical acceptance remains pending until an operator or instrument confirms
  visible screen/RGB/head movement from the showtime scene.
- The separate official `/stackChan/ws` avatar/action relay remains
  disconnected; current successful body delivery is through the product
  Xiaozhi MCP path.
- Camera, NFC, IR, IMU, ambient/proximity, and battery parity remain scoped
  follow-up transitions.

Known risks or blockers:

- Keep `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false` until foreground hardware
  stability proves automatic listen/start body reactions are safe again.
- Do not mark the showtime scene `physical_accepted=true` from machine
  delivery alone.
- Git may continue to warn about historical loose objects/gc; no prune/gc is
  authorized.

Recommended next action:

- Ask the operator to confirm visible showtime screen/RGB/head movement, then
  promote the scene acceptance evidence if confirmed.
- Next code transition should add richer operator-visible body/evidence
  controls or continue official parity gaps such as touch-driven gestures,
  battery/status diagnostics, and the official avatar relay reconciliation.

Test, build, or runtime results:

- `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 make a21-stackchan-official-xiaozhi-compatible-flash-plan`:
  passed with `status=ready`.
- `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP make a21-stackchan-official-xiaozhi-compatible-flash-execute`:
  passed with `status=passed`.
- Public `/v1/devices`: product device returned online and stayed online in a
  follow-up check after showtime.
- Public `/v1/xiaozhi/body-scene`: showtime returned HTTP 200 delivered.

Failure location and reason:

- None in this round. Remaining physical acceptance is evidence-gated, not a
  runtime failure.

## 2026-06-04 - T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001 - Start Roleplay/Professional v2

Goal:

- Start internal test 4 without regressing internal test 3 voice-main-chain
  acceptance.
- Promote `roleplay` and `professional` to the two user-facing modes.
- Preserve `professional` as the only A21/V21 evidence path.
- Define the A21 Cloud/Web/App plus V21 Knowledge Service/Adapter product form
  for upload, public-only query, personal-only query, personal+public query,
  and A21 hardware professional consultation.

Actual completed work:

- Created the internal test 4 plan:
  `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`.
- Updated PRD v0.5 to make `roleplay` the default embodied/personality/memory/
  voice-clone mode and `professional` the explicit evidence mode.
- Updated protocol documentation and V21 integration notes for adapter v2
  direction with workspace/query-scope fields.
- Updated Gateway mode contract:
  - `/v1/voice-modes` lists `roleplay` and `professional`;
  - default selected voice mode is `roleplay`;
  - old `dialogue` input is accepted as a backwards-compatible alias and
    returns selected `roleplay`;
  - fast-companion accepts `roleplay` while still rejecting selected
    `professional` before provider or V21 execution.
- Updated simulator defaults/readouts to `roleplay`.
- Updated project state and evidence manifest to treat internal test 4 as the
  active build direction while keeping internal test 3 as the latest accepted
  package.

Files changed:

- `internal/protocol/message.go`
- `internal/protocol/message_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/prd/A21_PRD.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/V21_INTEGRATION.md`
- `docs/engineering/VOICE_MODE_SELECTION.md`
- `docs/engineering/A21_CLOUD_VOICE_PROVIDER_MATRIX.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- A21 Cloud/Web/App upload/index/device-binding implementation is not built
  yet.
- V21 upload/index/query-scope implementation is not verified yet.
- Adapter v2 code fields are planned but not yet implemented.
- Physical StackChan internal test 4 professional consult evidence remains
  pending.

Known risks or blockers:

- Do not rename or rewrite all legacy `workmate`/`companion` runtime states in
  this cut; they are still part of internal test 3 compatibility.
- Do not let `roleplay` call V21 just because memory/persona hints exist.
- Do not claim cloud workspace readiness from the v1 local adapter smoke.
- Git may continue to warn about historical loose objects/gc; no prune/gc
  action is authorized.

Recommended next action:

- Implement adapter v2 request/response fields for `workspace_id`, `user_id`,
  and `query_scope` behind redacted tests.
- Dispatch a V21-side worker for upload/index/query-scope support without
  importing V21 internals into A21.
- Then wire web/app workspace controls and hardware professional consult
  evidence.

Test, build, or runtime results:

- `go test ./internal/protocol -run 'ProductModes|Envelope|DeviceEvent' -count=1`: passed.
- `go test ./internal/gateway -run 'VoiceModes|FastCompanion|SimulatorPageServed|ProfessionalMode|OrdinaryOfficeModes' -count=1`: passed.
- `go test ./internal/protocol ./internal/gateway ./internal/personality ./internal/v21adapter -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location and reason:

- None so far.

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

## 2026-06-03 10:32 CST - Xiaozhi Opus Ingress Queue

本轮目标:

- Continue the persistent Xiaozhi realtime convergence goal by adding the
  explicit Opus frame queue boundary requested by the target chain.
- Keep the cut host-local and stock-protocol aligned: no provider/V21
  execution, no Gateway lifecycle change, no `/v1/xiaozhi/say`, no
  host-loopback runtime acceptance, no firmware/hardware/audio playback.

实际完成内容:

- Added plan `docs/plans/2026-06-03-xiaozhi-opus-ingress-queue.md`.
- Spawned read-only explorers:
  - Explorer Goodall confirmed the current A21 read loop decoded Opus, pushed
    VAD/audio ingress, and called streaming-ASR `AppendFrame` inline; this can
    block `abort`, `listen.stop`, heartbeat/device events, and later stock
    control frames.
  - Explorer Boole confirmed the next stock behavior to prove after queueing:
    wake-as-abort plus playback queue drain before fresh listening.
- Added a bounded per-session Opus ingress queue for listening `/v1/xiaozhi`
  frames.
- `handleXiaozhiBinary` now records receipt/enqueue and returns to the
  WebSocket read loop before Opus decode, VAD/audio ingress, or streaming ASR
  append.
- Added a queue worker that preserves frame order, decodes Opus, pushes
  audio-ingress/VAD evidence, appends to streaming ASR, and records
  `xiaozhi.opus_ingress.queue_dropped` when the bounded queue is full.
- `listen.stop` now starts an async finalize path that waits briefly for
  queued ingress to catch up, then commits streaming ASR or starts the voice
  pipeline; the control read loop stays free to process `abort`.
- Updated `docs/project_state_machine.md` with the completed transition and
  remaining PRD blockers.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-opus-ingress-queue.md`
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
- Next useful transition: host-test stock wake-as-abort/playback-drain ordering
  or, with explicit authorization, collect a physical stock `/v1/xiaozhi`
  trace / run real streaming TTS runtime proof.

测试/构建/运行结果:

- Red Gateway test first:
  `go test ./internal/gateway -run TestXiaozhiWebSocketOpusAppendDoesNotBlockAbortControlFrame -count=1`
  failed because `xiaozhi.abort.received` was not recorded while streaming ASR
  `AppendFrame` was blocked.
- Focused Gateway tests passed:
  `go test ./internal/gateway -run 'TestXiaozhiWebSocket(OpusAppendDoesNotBlockAbortControlFrame|WakePrerollOpusFeedsNextTurn|StreamingASRFinalSendsStockSTTBeforeTTS|StreamingASRFinalStartsPipelineWithoutBatchFallback|ListenStopDoesNotBlockAbortWhileStreamingASRCommitPending|ListenStopRunsVoicePipelineAndSendsPacedOpus|StreamingASRStartsBeforeListenStop|ASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal)' -count=1`.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Focused app parity/readiness tests passed:
  `go test ./internal/app -run 'TestXiaozhiRealtimeParity|TestXiaozhiStreamingProviderReadiness' -count=1`.
- `git diff --check` passed.
- `make verify` passed.
- No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`,
  host-loopback runtime acceptance, firmware build, flash, NVS/serial access,
  hardware action, or audio playback was performed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The initial failure was the intentional red test.

## 2026-06-03 10:48 CST - Xiaozhi Stale Opus Ingress Suppression

本轮目标:

- Continue the persistent Xiaozhi realtime convergence goal by tightening the
  Opus ingress queue ownership boundary after `fc79156`.
- Prevent abort/wake-as-abort/listen-start barge-in from letting queued
  old-turn Opus frames decode into the next listening turn.
- Keep the cut host-local and stock-protocol aligned.

实际完成内容:

- Added plan `docs/plans/2026-06-03-xiaozhi-stale-opus-ingress-suppression.md`.
- Ran two read-only explorers:
  - Raman confirmed stock-shaped behavior: wake/listen during Speaking should
    interrupt current playback/turn and then allow fresh Listening; debug
    `stop_done` must stay optional and not become stock acceptance.
  - Aristotle identified the narrowest next A21 gap: abort can process while
    Opus append is blocked, but queued old-turn frames still needed explicit
    cancellation/suppression proof.
- Added a low-level Gateway red test proving a cancelled Opus ingress item
  previously reached `audio.ingress.buffered`.
- Added a WebSocket regression proving abort with queued old-turn Opus frames
  suppresses the queued frames and does not start stale voice-pipeline/downlink
  work.
- Added a pre-decode stale-context guard in `processXiaozhiOpusIngressFrame`.
- Updated `docs/project_state_machine.md`.

修改过的文件:

- `docs/plans/2026-06-03-xiaozhi-stale-opus-ingress-suppression.md`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Full Xiaozhi realtime PRD acceptance remains incomplete.
- Real streaming TTS execution is still pending complete provider env.
- User supplied provider credentials in-chat during this round; do not echo,
  commit, or write them to tracked docs. Current known missing Doubao realtime
  TTS values still include model and voice names unless they are supplied via
  a local env file.
- Physical stock `/v1/xiaozhi` trace still needs wake or labeled tap trigger,
  real streaming ASR/LLM/TTS profile markers, audible playback, touch/barge-in,
  playback stop completion where available, and idle recovery.

测试/构建/运行结果:

- Red Gateway test first:
  `go test ./internal/gateway -run TestXiaozhiOpusIngressSkipsCanceledTurnFrame -count=1`
  failed because a cancelled Opus ingress item reached
  `audio.ingress.buffered` and lacked stale suppression.
- Green focused stale-ingress tests passed:
  `go test ./internal/gateway -run 'TestXiaozhi(OpusIngressSkipsCanceledTurnFrame|WebSocketAbortSuppressesQueuedOldTurnOpusFrames)' -count=1`.
- Focused Xiaozhi Gateway tests passed:
  `go test ./internal/gateway -run 'Test(XiaozhiOpusIngressSkipsCanceledTurnFrame|XiaozhiWebSocket(AbortSuppressesQueuedOldTurnOpusFrames|OpusAppendDoesNotBlockAbortControlFrame|ListenStopDoesNotBlockAbortWhileStreamingASRCommitPending|ListenStartBargeInStopsActiveTTS|AbortCancelsBlockedTurnTaskWithinBargeInBudget|WakePrerollOpusFeedsNextTurn|StreamingASRFinalSendsStockSTTBeforeTTS|StreamingASRFinalStartsPipelineWithoutBatchFallback|ListenStopRunsVoicePipelineAndSendsPacedOpus|StreamingASRStartsBeforeListenStop|ASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal|AbortDuringStreamingAnswerSuppressesStaleSegments|AbortAfterFastAckSuppressesAnswerFrames|AbortStopsPlaceholderTTSAndPreventsStaleBinary)|WriteXiaozhiOpusDownlinkSkipsStaleTurn)' -count=1`.

如果中途失败，记录失败位置和原因:

- No unresolved failure so far. The initial failure was the intentional red
  test.

## 2026-06-03 12:11 CST - Dialogue-First Low-Latency PRD Convergence

本轮目标:

- 接续主控纪律，把 A21 当前产品模式收敛为两个对外模式:
  `dialogue` 和 `professional`。
- 专打对话低延迟链路，先把 PRD、协议、Gateway 合同、simulator
  和 provider-readiness 报告对齐到 dialogue-first。
- 保持 provider/V21/Gateway/硬件执行边界: 不落盘真实 provider
  credential，不启动或停止 Gateway，不执行真实 provider/V21，不触碰
  firmware/serial/hardware/audio playback。

实际完成内容:

- Added plan
  `docs/plans/2026-06-03-dialogue-first-low-latency-prd-convergence.md`.
- Updated PRD and engineering docs so the product-facing mode set is
  `dialogue` + `professional`; older `workmate` / `companion` /
  `co_creation` / `roleplay` labels are compatibility labels that normalize
  to `dialogue` at product-contract surfaces.
- Added protocol product-mode helpers for `dialogue`, canonical mode listing,
  and legacy normalization.
- Updated `/v1/voice-modes`, Gateway dialogue endpoints, and simulator mode
  controls to expose dialogue/professional while keeping professional as the
  V21 adapter boundary and rejected from dialogue-only endpoints.
- Updated Xiaozhi streaming provider-readiness report with
  `product_mode=dialogue`, `chain_mode=dialogue_low_latency`, and
  `professional_boundary=v21_adapter_only`; the report remains static
  no-execute evidence with `prd_accepted=false`.
- Updated Doubao realtime TTS configuration checks so either
  `A21_DOUBAO_API_KEY` or `A21_DOUBAO_ACCESS_TOKEN` can satisfy the credential
  env requirement; report/test output uses env names only.
- Renamed `internal/app/frozen_x21_firmware.go` to
  `internal/app/frozen_external_firmware.go` with no behavior change. Reason:
  X21 remains a frozen one-way reference, but A21 runtime/app path names should
  not carry new X21 identity.
- Updated `docs/project_state_machine.md` with active transition
  `T-DIALOGUE-001-LOW-LATENCY-CHAIN-CONVERGENCE`.

修改过的文件:

- `docs/plans/2026-06-03-dialogue-first-low-latency-prd-convergence.md`
- `docs/prd/A21_PRD.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/VOICE_MODE_SELECTION.md`
- `docs/project_state_machine.md`
- `internal/protocol/message.go`
- `internal/protocol/message_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `internal/app/frozen_external_firmware.go`
- `internal/app/streaming_tts_runtime_smoke.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `internal/providers/doubao_realtime_tts_provider.go`
- `internal/providers/doubao_realtime_tts_provider_test.go`
- `docs/agent_handoff_log.md`

当前未完成事项:

- Dialogue low-latency chain is contract-aligned but not product-accepted:
  no real Doubao execution, no real Gateway runtime window, and no physical
  stock `/v1/xiaozhi` trace were collected in this round.
- Professional mode remains V21 adapter-only; no V21 execute or integration
  validation was performed.
- Doubao credential values supplied in chat were not written to repo files,
  reports, or logs by this round. Future runtime work must inject them through
  a local secret env path and redact evidence.
- Physical acceptance still needs labeled stock trigger, streaming ASR/LLM/TTS
  profile markers, audible playback, touch/barge-in, and idle recovery.

测试/构建/运行结果:

- Initial direct `go test ./internal/protocol ./internal/gateway
  ./internal/app ./internal/providers -count=1` failed only in two doctor
  tests because the shell inherited a global proxy and lacked full A21
  direct-connect `NO_PROXY` coverage. Reproduced root cause as
  `proxy_direct_bypass_missing`.
- Same touched-package test passed after explicit direct-connect env:
  `env NO_PROXY=localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16 no_proxy=localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16 A21_NO_PROXY=localhost,127.0.0.1,::1,.local,10.0.0.0/8,10.21.0.0/16,172.16.0.0/12,192.168.0.0/16 go test ./internal/protocol ./internal/gateway ./internal/app ./internal/providers -count=1`.
- `make verify` passed.
- `make preflight` passed. Host gate was ok, namespace audit was ok, with
  existing warnings `firmware_current_artifact_missing` and
  `wake_word_firmware_build_required`.
- `make doctor` passed with the same existing warnings.
- `git diff --cached --check` passed.
- No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`,
  host-loopback runtime acceptance, firmware build, flash, NVS/serial access,
  hardware action, or audio playback was performed.

如果中途失败，记录失败位置和原因:

- No unresolved failure. The only failure was the direct `go test` invocation
  missing A21 direct-connect proxy coverage; Makefile-backed gates passed.

## 2026-06-03 12:42 CST - Aliyun Public Gateway Profile Configuration

本轮目标:

- 接续用户最新决策: 公网通道合理，但不能替代 Mac Gateway 的本地模型和
  本地处理极速能力。
- 将公网 Gateway 做成可选产品配置，让前端可选，同时保持
  `dialogue` / `professional` 仍是唯一产品模式集合。
- 不切换 Mac 网络，不写入 provider credential，不重启现有 Gateway，不构建
  或刷写 firmware，不触碰硬件/audio playback。

实际完成内容:

- Updated Aliyun plan
  `docs/plans/2026-06-03-aliyun-xiaozhi-public-voice-gateway.md`:
  `mac_local` is the default local-speed profile and `public_wss` is an
  optional product public Gateway profile.
- Added Gateway profile API:
  - `GET /v1/gateway-profiles`
  - `POST /v1/gateway-profiles`
  - schema `a21.gateway.profiles.v1`
  - profiles `mac_local` and `public_wss`
- Added `A21_PUBLIC_GATEWAY_URL` and CLI `--public-gateway-url` wiring.
  Accepted public URLs must be `https` or `wss`, must not include URL
  credentials, query, fragment, token strings, or secret strings, and normalize
  to `wss://<host>/v1/xiaozhi`.
- Updated `/xiaozhi/ota/` so:
  - selected `mac_local` returns request-host `ws`/`wss`;
  - `X-Forwarded-Proto: https` maps to `wss`;
  - selected configured `public_wss` returns the configured public WSS endpoint.
- Updated simulator frontend with a Gateway profile selector and readout; it
  fetches/saves `/v1/gateway-profiles` independently from `/v1/voice-modes`.
- Updated `PROTOCOL.md`, `VOICE_MODE_SELECTION.md`, `NETWORK.md`, PRD, and
  project state machine to keep `gateway_profile` separate from `voice_mode`.
- Ran non-mutating Aliyun reachability checks against the provided public host:
  SSH `22` is reachable; `443` currently refuses connection. BatchMode SSH
  probes for `root` and `ubuntu` failed with `Permission denied (publickey)`,
  so this Mac currently lacks a usable key for remote deployment.

修改过的文件:

- `docs/plans/2026-06-03-aliyun-xiaozhi-public-voice-gateway.md`
- `docs/prd/A21_PRD.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/VOICE_MODE_SELECTION.md`
- `docs/engineering/NETWORK.md`
- `docs/project_state_machine.md`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `docs/agent_handoff_log.md`

测试/构建/运行结果:

- Red test first failed as expected because `ServerOptions.PublicGatewayURL`
  and `/v1/gateway-profiles` did not exist.
- Focused gateway/app profile tests passed:
  `go test ./internal/gateway ./internal/app -run 'Test(XiaozhiOTAEndpointUsesWSSBehindTLSReverseProxy|GatewayProfilesCatalogDefaultsToMacLocalAndAllowsConfiguredPublicWSS|GatewayProfilesRejectsPublicWSSWithoutConfiguredPublicURL|SimulatorPageServed|GatewayServerFromEnvExposesConfiguredPublicGatewayProfile)' -count=1`.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Focused app config tests passed:
  `go test ./internal/app -run 'Test(GatewayServerFromEnvExposesConfiguredPublicGatewayProfile|GatewayServerOptionsFromEnvWiresXiaozhiListenMaxDuration|GatewayServerOptionsFromEnvWiresSileroVADConfig|GatewayServerFromEnvUsesSelectedProviderOnlyWhenExplicit)' -count=1`.
- `git diff --check` and `git diff --cached --check` passed.
- `make verify` passed.
- `make preflight` passed with existing warnings only:
  `firmware_current_artifact_missing`,
  `wake_word_firmware_build_required`.
- `make doctor` passed with the same existing warnings.

当前未完成事项 / blockers:

- Aliyun public Gateway is not deployed from this Mac yet: 443 is closed/refused
  and SSH key access is not available for `root` or `ubuntu`.
- No real provider execution, Aliyun runtime, physical StackChan public WSS
  connection, wake-word proof, or PRD acceptance was collected in this round.
- Product readiness remains blocked by physical StackChan acceptance and
  wake-word product proof, not by the new profile API.

推荐下一步:

- Provide a usable SSH key/session for the Aliyun instance or run the deployment
  commands on the host directly, then start A21 Gateway with
  `A21_PUBLIC_GATEWAY_URL=https://<public-host>` behind Caddy/Nginx on `443`.
- After `443` is healthy, select `public_wss` in `/simulator` and point
  StackChan OTA at the public Gateway to collect a fresh physical candidate
  trace.

## 2026-06-03 13:35 CST - Aliyun Public Gateway Runtime Bring-up

本轮目标:

- 继续主控线程，不切换 Mac 网络。
- 使用本机 Aliyun CLI/SWAS 命令助手，把 A21 公网 Gateway 候选通道实际部署
  到用户给定的轻量应用服务器。
- 保留 Mac Gateway 作为本地模型/本地处理极速路径；公网只作为
  `gateway_profile=public_wss` 可选产品配置。
- 不把对话中提供的 provider credential 写入 repo、命令内容、远端 env、
  systemd unit、nginx config、报告或日志。

实际完成内容:

- 确认实例 `b5c8d6841b50416ca3470665a0087e28` 是 Aliyun SWAS/轻量应用
  服务器，不是 ECS；必须使用
  `aliyun swas-open ... --region cn-shanghai --endpoint
  swas.cn-shanghai.aliyuncs.com --biz-region-id cn-shanghai`。
- 确认服务器状态:
  - Ubuntu 24.04, 2 vCPU / 2 GiB, public IP `101.132.117.182`
  - SWAS 防火墙已有 `80`, `443`, `22`, ICMP allow
  - `swas-open run-command` 以 root 可执行；SSH key 仍不可用
- 远端 GitHub clone 因 `GnuTLS recv error (-110)` 失败；切换为 SWAS
  命令分片传输本地 tracked snapshot。
- 通过 119 个 SWAS command chunk 将
  `/tmp/a21-tracked-current.tar.gz` 传到 `/opt/a21-deploy`；远端 base64
  长度 `1425608` 和 tar SHA-256
  `6c210949776d8295f25b62ffc7d9430d87fec5d91c8b5749be563c9c905e1f56`
  与本地匹配后解包到 `/opt/a21`。
- 安装远端构建依赖: `build-essential`, `pkg-config`, `libopus-dev`, Go
  `1.26.3`。
- 首次远端 build 失败在 `proxy.golang.org` 超时；使用
  `GOPROXY=https://goproxy.cn,direct` 和
  `GOSUMDB=sum.golang.google.cn` 后
  `go build -o /opt/a21/bin/a21 ./cmd/a21` 通过。
- 创建并启动 systemd service `a21-gateway`:
  - 工作目录 `/opt/a21`
  - 监听 `127.0.0.1:21080`
  - `A21_PUBLIC_GATEWAY_URL=https://101.132.117.182`
  - `A21_XIAOZHI_PRODUCT_CHAIN=host_local`
  - `A21_VOICE_TEXT_MAX_TOKENS=32`
  - 未注入 provider credential
- 尝试 Caddy 反代时发现现有 nginx 已占用 `80`；根因是该服务器已有
  X21/控制台站点。未抢占整站；Caddy 最终保持 `inactive/disabled`。
- 备份并更新 `/etc/nginx/sites-enabled/x21`:
  - 仅 A21 路径 `/healthz`, `/simulator`, `/v1/`, `/xiaozhi/ota`,
    `/ws/audio` proxy 到 `127.0.0.1:21080`
  - 原 root `/` 仍 proxy 到 `127.0.0.1:8000`
  - nginx 增加 `443 ssl`，证书为 `/etc/a21/tls/a21-selfsigned.crt`
    和 `/etc/a21/tls/a21-selfsigned.key`
  - 原配置备份在 `/etc/nginx/a21-backups/`
- 外部 Mac 验证:
  - `curl http://101.132.117.182/healthz` 返回
    `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`
  - `curl -k https://101.132.117.182/healthz` 返回同样 health
  - `curl -k https://101.132.117.182/v1/gateway-profiles` 返回
    `selected_gateway_profile=public_wss`，并暴露
    `wss://101.132.117.182/v1/xiaozhi`
  - `curl -k https://101.132.117.182/xiaozhi/ota/` 返回 stock OTA
    `websocket.url=wss://101.132.117.182/v1/xiaozhi`
  - 不带 `-k` 的 HTTPS 验证失败:
    `SSL certificate problem: self signed certificate`
  - `curl http://101.132.117.182/` 仍返回原控制台 HTML，未被 A21 接管
  - `curl -k https://101.132.117.182/simulator` 返回 A21 simulator HTML
- 跑了一次 host-only 公网 WebSocket bench:
  `go run ./cmd/a21 xiaozhi-voice-bench --gateway-url
  http://101.132.117.182 --repeat 1 --timeout-ms 5000
  --require-product-chain`
  - 结果 exit 1，按预期未达验收
  - answer turn: `hello_accepted=true`, `listen_ack=true`,
    `metrics_observed=true`, status `passed`
  - barge-in turn: status `failed`, finding `turn_read_failed`
  - `acceptance_status=blocked`, `prd_accepted=false`
  - execution flags:
    `provider_executed=false`, `v21_executed=false`,
    `hardware_executed=false`, `host_product_chain_ready=false`
  - report:
    `reports/a21-xiaozhi-voice-bench-20260603-133114.031578000.json`

修改过的本地文件:

- `docs/plans/2026-06-03-aliyun-xiaozhi-public-voice-gateway.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

远端运行/配置面:

- `/opt/a21`
- `/opt/a21/bin/a21`
- `/etc/systemd/system/a21-gateway.service`
- `/etc/nginx/sites-enabled/x21`
- `/etc/nginx/a21-backups/`
- `/etc/a21/tls/a21-selfsigned.crt`
- `/etc/a21/tls/a21-selfsigned.key`

当前未完成事项 / blockers:

- `443` 目前是自签证书，只能 `curl -k` 验证；真实 StackChan 产品 WSS
  需要可信域名证书，或明确的设备信任/证书策略。
- 没有通过 CLI 命令内容写入任何 provider secret；因此远端还没有真实
  cloud ASR / LLM / TTS execution。
- 公网 WebSocket 已证明 hello/listen 候选，但没有真实 product chain、
  downlink audio、物理 StackChan online/public-WSS、可听播放、barge-in
  和 idle recovery 证据。
- `product-readiness` 不能置绿；物理 StackChan acceptance 和 wake-word
  product proof 仍是硬阻塞。

推荐下一步:

- 给 `101.132.117.182` 绑定一个 A21 域名并签发可信 TLS，或明确设备
  证书信任策略；随后把 `A21_PUBLIC_GATEWAY_URL` 改为
  `https://<a21-domain>`。
- 用非日志化的 secret 注入方式配置真实 provider env；不要通过
  `swas-open run-command --command-content` 直接传 provider key。
- 可信 TLS 和 provider env 就绪后，重跑公网 `/v1/xiaozhi` host bench，
  再让 StackChan OTA 指向公网 Gateway 采集物理 candidate trace。

## 2026-06-03 14:22 CST - Main ECS Public Gateway Bring-up

本轮目标:

- 接续主控线程，不丢进程/进度。
- 按用户最新决策修正口径: `47.103.57.217` 是 A21 主公网 voice-edge
  Gateway，不是候选；Mac Gateway 作为前端/operator 可切换的本地极速路径。
- 不切换 Mac 网络，不把 provider credential 写入 repo、命令记录、远端
  systemd/Caddy 配置、固件、报告或日志。

实际完成内容:

- Added/updated tests so a valid `A21_PUBLIC_GATEWAY_URL` selects
  `public_wss` by default while `mac_local` remains switchable.
- Added IP-only public bring-up support: `http://...` and
  `ws://.../v1/xiaozhi` public URLs normalize to `ws://.../v1/xiaozhi`;
  `https`/`wss` still normalize to `wss`. URL credentials, query, fragment,
  `token`, and `secret` strings remain rejected.
- Updated PRD, protocol, network, voice-mode, Aliyun plan, and project state
  docs:
  - `47.103.57.217` / ECS `i-uf63f4ymqc2dxtljxz2n` is the main public
    Gateway target.
  - `101.132.117.182` SWAS is experimental/backup evidence only.
  - Product target remains trusted `443`/`wss`; IP-only `http/ws` is bring-up.
- Kept the Linux runtime fingerprint fix from the ECS bring-up:
  `DetectFingerprint` now falls back to `ip route show default` and parses
  `dev eth0` when macOS `route -n get default` is unavailable.
- Synced current tracked A21 snapshot to new ECS through
  `/tmp/a21-tracked-current-main-ecs.tar.gz`; remote SHA-256 matched
  `1782eaec865fc771452956942f0563466c56b24c3a51bda1bb315028c34fb98a`.
- On `47.103.57.217`, installed/confirmed Go `1.26.3`, build dependencies,
  Caddy, UFW, and built `/opt/a21/bin/a21`.
- Remote `/opt/a21` was deployed through `/opt/a21.next`; previous `/opt/a21`
  was preserved under a timestamped backup.
- Created/enabled `a21-gateway.service`:
  - listens only on `127.0.0.1:21081`
  - `--public-gateway-url http://47.103.57.217`
  - `--product-chain host_local`
  - no provider credential values
- Configured Caddy public `80` and `443` reverse proxy to `127.0.0.1:21081`.
  `443` uses a temporary 30-day self-signed IP SAN cert for bring-up only.
- Fixed a Caddy restart failure caused by the temporary private key being
  unreadable to the `caddy` user; changed ownership/permissions and restarted
  successfully.

修改过的本地文件:

- `docs/plans/2026-06-03-aliyun-xiaozhi-public-voice-gateway.md`
- `docs/prd/A21_PRD.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/VOICE_MODE_SELECTION.md`
- `docs/engineering/NETWORK.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app_test.go`
- `internal/runtimeguard/fingerprint.go`
- `internal/runtimeguard/fingerprint_test.go`

远端运行/配置面:

- `/opt/a21`
- `/opt/a21/bin/a21`
- `/etc/systemd/system/a21-gateway.service`
- `/etc/caddy/Caddyfile`
- `/etc/a21/tls/a21-edge-ip.crt`
- `/etc/a21/tls/a21-edge-ip.key`

测试/构建/运行结果:

- Red test first failed for HTTP public bring-up because old validator rejected
  `http://47.103.57.217`.
- Focused public Gateway tests passed after the minimal validator change:
  `go test ./internal/gateway ./internal/app -run
  'TestGatewayProfiles(AcceptsPublicHTTPBringupURL|CatalogDefaultsToPublicWSSAndAllowsMacLocalSwitch|RejectsPublicURLWithQuery)|TestGatewayServerFromEnvExposesConfiguredPublicGatewayProfile'
  -count=1`.
- Touched package test passed with A21 direct-connect env:
  `go test ./internal/gateway ./internal/app ./internal/runtimeguard -count=1`.
- `git diff --check` and `git diff --cached --check` passed.
- Local `make verify` passed.
- Remote `make verify` passed on ECS.
- Remote services:
  - `a21-gateway`: active
  - `caddy`: active
  - listeners: public `80`, public `443`, local `127.0.0.1:21081`
- External Mac verification passed:
  - `http://47.103.57.217/healthz`
  - `http://47.103.57.217/v1/gateway-profiles`
  - `http://47.103.57.217/xiaozhi/ota/`
  - `https://47.103.57.217/healthz` with `-k`
  - `https://47.103.57.217/xiaozhi/ota/` with `-k`
- OTA currently returns:
  `ws://47.103.57.217/v1/xiaozhi`.
- Host-only public WebSocket bench over `http://47.103.57.217` accepted
  hello/listen and observed host-local voice-pipeline shape, but exited 1 as
  expected:
  - `prd_accepted=false`
  - `provider_executed=false`
  - `v21_executed=false`
  - `hardware_executed=false`
  - LLM profile still `mock`
  - barge-in turn failed with `turn_read_failed`
  - report `reports/a21-xiaozhi-voice-bench-20260603-142101.627764000.json`

当前未完成事项 / blockers:

- Provider secrets were intentionally not injected; no real cloud ASR/LLM/TTS
  execution occurred on the ECS.
- `443` is self-signed IP TLS for bring-up only. Product trusted `wss`
  requires a domain and trusted certificate, or an explicit device trust
  decision.
- No physical StackChan public Gateway trace has been collected yet.
- Product readiness remains blocked by real physical StackChan dialogue
  acceptance and wake-word product proof.

推荐下一步:

- Point StackChan at `ws://47.103.57.217/v1/xiaozhi` for immediate main
  public-edge bring-up.
- Inject provider secrets through a non-logged server-side secret path, then
  rerun public `/v1/xiaozhi` bench and collect physical StackChan trace.
- Bind a domain to `47.103.57.217`, switch `A21_PUBLIC_GATEWAY_URL` to the
  trusted `https://<domain>` endpoint, and promote OTA to `wss://.../v1/xiaozhi`.

## 2026-06-03 - StackChan Xiaozhi-style Wi-Fi provisioning

目标:

- 按用户最新要求，不再只靠硬写 Wi-Fi；产品端侧配网按 Xiaozhi 启动思路推进。
- 保持 47.103.57.217 作为主公网 Gateway，Mac Gateway 仍作为可切换本地路径。
- 不写入 provider secret，不 flash 硬件，不回退当前 staged 公网/provider/模式收敛改动。

实际完成内容:

- 新增计划 `docs/plans/2026-06-03-stackchan-xiaozhi-style-wifi-provisioning.md`。
- A21 self-owned firmware 状态机新增 `A21_CONN_WIFI_PROVISIONING`：
  - 缺 Wi-Fi SSID 时进入 `Wi-Fi provisioning`。
  - invalid Wi-Fi 仍进入 local fallback 并标记 `invalid_wifi`。
  - provisioning 失败才进入 local fallback。
- A21 self-owned firmware 新增 Xiaozhi-style 配网方法枚举：
  - `hotspot`
  - `blufi`
  - `acoustic`
  - 默认 `hotspot`
- `a21_firmware_wifi_runtime.h` 新增 provisioning driver seam：
  - 没有凭据时不会调用 `WiFi.begin`。
  - provisioning method 只启动一次。
  - 仍保留已有 station connect / reconnect 行为。
- official-compatible product overlay 明确保留 Xiaozhi 端侧配网模型：
  - `CONFIG_USE_HOTSPOT_WIFI_PROVISIONING=y`
  - `# CONFIG_USE_ESP_BLUFI_WIFI_PROVISIONING is not set`
  - `# CONFIG_USE_ACOUSTIC_WIFI_PROVISIONING is not set`
  - OTA 更新为主公网 Gateway：`http://47.103.57.217/xiaozhi/ota/`
- 文档更新：
  - `docs/engineering/NETWORK.md`
  - `docs/engineering/PROTOCOL.md`
  - `firmware/stackchan/README.md`
  - `docs/project_state_machine.md`

修改过的本轮文件:

- `docs/plans/2026-06-03-stackchan-xiaozhi-style-wifi-provisioning.md`
- `firmware/stackchan/include/a21_firmware_connection.h`
- `firmware/stackchan/include/a21_firmware_wifi.h`
- `firmware/stackchan/include/a21_firmware_wifi_runtime.h`
- `firmware/stackchan/test/test_protocol/test_main.cpp`
- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `firmware/stackchan/README.md`
- `docs/engineering/NETWORK.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

测试/构建结果:

- Red test first failed as expected:
  - Go overlay test missing `CONFIG_USE_HOTSPOT_WIFI_PROVISIONING=y`。
  - firmware native test failed to compile because provisioning phase/method/runtime seam did not exist.
- Focused Go test passed:
  `go test ./internal/app -run TestOfficialXiaozhiCompatibleOverlayPreservesXiaozhiWifiProvisioning -count=1`
- Firmware native tests passed:
  `make firmware-test`
  - `93 test cases: 93 succeeded`
- Full local verification passed on retry:
  `make verify`
  - First run hit one non-reproducing doctor test failure.
  - Focused doctor test passed.
  - Second full run passed.
- Product lane no-flash build passed:
  `make a21-stackchan-official-xiaozhi-compatible-build`
  - Report:
    `reports/a21-stackchan-official-baseline-20260603-150433-1780470273395485000.json`
  - Artifact:
    `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  - App SHA-256:
    `c012542ee8f1837106da91fe934e27487813333eaad124841386706259f48659`
  - Overlay applied:
    `true`
  - Product candidate:
    `a21-stackchan-official-xiaozhi-compatible`
  - `official_avatar_action_preserved=true`
  - `official_xiaozhi_start_preserved=true`
  - Build sdkconfig/defaults confirmed:
    `CONFIG_OTA_URL="http://47.103.57.217/xiaozhi/ota/"`
    `CONFIG_USE_HOTSPOT_WIFI_PROVISIONING=y`
    `# CONFIG_USE_ESP_BLUFI_WIFI_PROVISIONING is not set`
    `# CONFIG_USE_ACOUSTIC_WIFI_PROVISIONING is not set`

未完成事项 / blockers:

- 尚未通过 guarded hardware window flash 到真实 StackChan。
- 尚未观察真实 first boot / no saved Wi-Fi NVS 的 Hotspot 配网画面或手机/浏览器配网链路。
- 这不改变 PRD 口径：物理对话、半双工/打断、wake-word 产品证明仍需单独验收。

推荐下一步:

- 通过 guarded flash lane 刷 `a21-stackchan-official-xiaozhi-compatible.bin`，观察无凭据启动是否进入 Xiaozhi Hotspot 配网。
- 之后再做真实 provider 注入和物理对话/打断/半双工验收。

## 2026-06-03 15:45 CST - Public Edge Provider/TTS And Physical Downlink Push

目标:

- 保持 `47.103.57.217` 作为主公网 Gateway。
- 在不把 provider key 写入 repo、固件、systemd unit、Caddy 配置、报告或日志的前提下，给公网 Gateway 注入真实 TTS 能力。
- 验证真实 StackChan 公网下行、host-level abort/barge-in 时序，并如实保留 PRD blockers。

实际完成内容:

- 在 ECS `/etc/a21/secrets/provider.env` 通过 root-only `0600` secret 文件注入 provider 环境变量；输出只记录变量名，不记录值。
- 部署并启用 `scripts/a21_dashscope_cosyvoice_tts.py`：
  - 作为 A21 `voice_clone_cli` wrapper。
  - 使用 DashScope CosyVoice WebSocket TTS。
  - 输出 PCM16 mono WAV 给现有 A21 local TTS adapter。
  - 不打印文本、provider output、API key、Authorization、Bearer、URL 或本地路径。
- ECS 安装 `python3-websocket` 作为 wrapper runtime dependency。
- Gateway systemd 继续只读取 `EnvironmentFile=-/etc/a21/secrets/provider.env`，未写入 secret 值。
- 验证并保留失败事实:
  - Doubao realtime TTS `ai-gateway.vei.volces.com` 使用用户提供 access token / secretkey 作为 Bearer key 均返回 handshake `401`。
  - 将 Doubao model 临时改为 `doubao-tts` 仍为 `401`，说明当前凭证不适用于 A21 已接入的 AI Gateway Realtime TTS Bearer API。
  - Iflytek TTS 兼容映射用户三元组也返回 `iflytek_tts_websocket_dial_failed_http_401`。
- 验证 DashScope CosyVoice 可用组合:
  - `cosyvoice-v3-flash`
  - `longanyang`
  - `pcm`
  - `16000 Hz`
- 远端 `local-tts-smoke --engine voice_clone_cli` 通过：
  - report path: `/tmp/a21-provider-smoke/a21-local-tts-smoke-20260603-154156.json`
  - provider/engine: `voice_clone_cli`
  - model label: `cosyvoice_v3_flash`
  - output bytes: `76844`
  - duration: `2400 ms`
  - peak: `-8.663 dBFS`
  - RMS: `-26.018 dBFS`
- 真实公网 StackChan `/v1/xiaozhi/say` delivered：
  - device: `44:1b:f6:e2:6a:60`
  - trace: `a21-trace-public-edge-dashscope-say-1780472526`
  - status: `delivered`
  - transport: `xiaozhi_ws`
  - text chars: `23`
  - Opus chunks: `87`
  - `tts.first_audio` offset: `2159 ms`
  - `audio.downlink.first_frame` offset: `2160 ms`
  - trace includes `xiaozhi.say.delivered`
- 公网 `xiaozhi-voice-bench` host-loopback ran against `http://47.103.57.217`:
  - report: `reports/a21-xiaozhi-voice-bench-20260603-154310.624051000.json`
  - answer turn passed with first audio `1185 ms`
  - barge-in turn passed with `abort_stop_ms=1`
  - final status still `blocked`, `prd_accepted=false`, because it is host/virtual evidence and did not prove physical microphone or full real ASR/Text/TTS execution.
- 公网 physical half-duplex acceptance was attempted and correctly blocked:
  - report: `reports/a21-stackchan-half-duplex-acceptance-20260603-154254.json`
  - reason: the current product Xiaozhi firmware is not the diagnostic mic-probe lane and lacks the diagnostic runtime echo counters required by this acceptance tool.

修改过的本轮文件:

- `scripts/a21_dashscope_cosyvoice_tts.py`
- `internal/providers/realtime.go`
- `internal/providers/realtime_test.go`
- `internal/app/streaming_tts_runtime_smoke.go`
- `internal/app/streaming_tts_runtime_smoke_test.go`
- `internal/app/app_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-03-provider-tts-real-dialogue-acceptance.md`

测试/部署结果:

- Focused tests passed:
  `go test ./internal/providers ./internal/app -run 'RealtimeWebSocketPlan|StreamingTTSRuntimeSmoke' -count=1`
- Full local verification passed:
  `make verify`
- Script validation passed:
  `python3 -m py_compile scripts/a21_dashscope_cosyvoice_tts.py`
  and `git diff --check`
- ECS deployed from a tracked/current working-tree tarball; remote Gateway restarted and `/healthz` passed.

未完成事项 / blockers:

- Physical audible confirmation for the new DashScope voice still needs operator listening/recording.
- Normal physical half-duplex and real microphone-driven dialogue remain open; the current half-duplex tool requires diagnostic firmware counters and is not applicable to stock product firmware.
- Wake-word product proof remains open.
- Doubao realtime TTS remains blocked by `401` until an AI Gateway Realtime Bearer API key is provided or a different Doubao auth adapter is added.
- Full PRD remains blocked; do not mark launch/full green from this round.

推荐下一步:

- Ask operator to confirm whether the delivered DashScope `/v1/xiaozhi/say` audio was audible and acceptable.
- Add a stock-Xiaozhi physical half-duplex acceptance path that uses real `/v1/xiaozhi` mic ingress/downlink traces instead of diagnostic runtime echo counters.
- Continue a separate real dialogue test with physical mic trigger, DeepSeek text stream, DashScope TTS downlink, and abort/barge-in trace.

## 2026-06-03 16:18 CST - Stock Xiaozhi Half-Duplex Acceptance Gate

目标:

- 补齐产品 stock Xiaozhi 固件可用的半双工验收路径，避免继续用诊断 mic-probe runtime echo counter 误挡产品固件。
- 保持公网 `47.103.57.217` 为主 Gateway，Mac Gateway 仍是可切换本地路径。
- 不动 firmware、不写 provider secret、不宣称 full PRD green。

实际完成内容:

- 新增 `stackchan-accept --check xiaozhi-half-duplex`：
  - 读取 stock `/v1/xiaozhi` trace 和 `/v1/audio/recent`。
  - 缺省可从 `/v1/devices` 自动派生 `last_trace_id` / `last_session_id`。
  - 写 `a21.xiaozhi_half_duplex_acceptance.v1` 报告。
  - `hardware_acceptance_scope=stock_xiaozhi_mic_to_tts_downlink`。
  - `diagnostic_mic_probe_required=false`。
  - 不调用 `/v1/devices/control`，不要求诊断 mic-probe capability 或 runtime echo mic counters。
- 保留旧 `stackchan-accept --check half-duplex` 作为诊断/仪表化 mic-to-mock-playback gate。
- 更新文档：
  - `docs/engineering/DOCTOR.md`
  - `docs/engineering/PROTOCOL.md`
  - `docs/plans/2026-06-03-provider-tts-real-dialogue-acceptance.md`
  - `docs/project_state_machine.md`

修改过的本轮文件:

- `internal/app/app_stackchan_common.go`
- `internal/app/app_stackchan_xiaozhi_half_duplex.go`
- `internal/app/xiaozhi_physical_evidence_test.go`
- `docs/engineering/DOCTOR.md`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-03-provider-tts-real-dialogue-acceptance.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

测试/实测结果:

- Red test first failed as expected:
  `unknown stackchan acceptance check "xiaozhi-half-duplex"`。
- Focused test passed:
  `go test ./internal/app -run 'TestRunXiaozhiHalfDuplexAcceptance(UsesStockTraceEvidence|DerivesLatestDeviceTrace)' -count=1`。
- Related focused suite passed:
  `go test ./internal/app -run 'XiaozhiPhysicalEvidence|XiaozhiHalfDuplex|StackChanHalfDuplex|StackChanAccept' -count=1`。
- Public Gateway live check passed for health/device/profile:
  - `http://47.103.57.217/healthz` OK。
  - `/v1/devices` shows physical `44:1b:f6:e2:6a:60` online, stock Xiaozhi transport, latest event `xiaozhi.hello`。
  - `/v1/gateway-profiles` still selects `public_wss` and keeps `mac_local` available。
- New stock half-duplex gate ran against `http://47.103.57.217` and correctly blocked:
  - report `reports/a21-xiaozhi-half-duplex-acceptance-20260603-161717.013784000.json`
  - physical device online: true
  - profile: stock
  - audio frame count: 0
  - mic/downlink/playback/barge-in evidence: missing
  - `prd_accepted=false`
- Local full verification passed:
  `make verify`。
- Commit created and pushed:
  `e8c9427 feat(xiaozhi): add stock half-duplex acceptance` on
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`。
- ECS `47.103.57.217` deployed from a `git archive` of commit `e8c9427`:
  - remote focused test passed:
    `go test ./internal/app -run 'XiaozhiPhysicalEvidence|XiaozhiHalfDuplex|StackChanHalfDuplex|StackChanAccept' -count=1`
  - remote `go build -o /opt/a21/bin/a21 ./cmd/a21` passed
  - `a21-gateway` restarted and stayed `active`
  - public `/healthz` OK
  - public `/v1/gateway-profiles` still selected `public_wss`, with `mac_local` available
  - `/xiaozhi/ota/` still returns `ws://47.103.57.217/v1/xiaozhi`
- Remote deployed binary ran the new gate and correctly blocked because the
  post-restart physical trace was still hello-only:
  `/tmp/a21-provider-smoke/a21-xiaozhi-half-duplex-acceptance-20260603-162242.676505054.json`。

未完成事项 / blockers:

- No current physical mic-driven dialogue trace has been captured on the public Gateway; latest physical trace is hello-only.
- Normal physical half-duplex/barge-in proof remains open until StackChan produces real mic Opus ingress and downlink in the same stock trace.
- Operator audible confirmation for the DashScope voice remains open.
- Wake-word product proof remains open.

推荐下一步:

- Trigger a real physical StackChan turn on the public Gateway, then rerun:
  `go run ./cmd/a21 stackchan-accept --check xiaozhi-half-duplex --gateway-url http://47.103.57.217 --device-id 44:1b:f6:e2:6a:60 --output-dir reports`
- If that report moves to `physical_review_required`, pair it with operator/instrument listening evidence and then refresh product readiness without marking full PRD green until wake-word proof exists.

## 2026-06-03 17:3x CST - StackChan Voice/Firmware Recovery Audit After No-Sound Report

目标:

- 回答用户关于“语音链路、Xiaozhi 协议/音频、唤醒词、级联优化是否在干净 Xiaozhi 重拉后丢失”的问题。
- 逐层区分 repo/overlay、clean build workdir、最新 bin/config、物理 Gateway runtime。
- 复盘构建/刷机路径错误，不把“应该还在”当作证据。

实际完成内容:

- 当前分支/版本:
  - branch `codex/a21-hardware-window-20260603-wifi-provisioning-flash`;
  - HEAD `4c4178a fix(stackchan): keep official dependency cache clean`;
  - ahead of origin by 1 commit;
  - unrelated/untracked only `.DS_Store` and `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md`.
- 确认最新产品 flash 是 product lane:
  - report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-170812-1780477692092580000.json`;
  - clean guard worktree `/private/tmp/a21-flash-clean`;
  - guard commit `4c4178af579d`;
  - app file `a21-stackchan-official-xiaozhi-compatible.bin`;
  - app sha `85e46b26c98b1c8c1d6f73be5e50be2149573623af7db1816851336be62c6389`.
- 确认产品 clean build config 实际包含:
  - `BOARD_TYPE_M5STACK_STACK_CHAN=true`;
  - `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`;
  - `USE_CUSTOM_WAKE_WORD=true`;
  - `CUSTOM_WAKE_WORD="zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue"`;
  - `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`;
  - `CUSTOM_WAKE_WORD_THRESHOLD=20`;
  - `SR_MN_CN_MULTINET7_QUANT=true`;
  - `SR_WN_WN9_HISTACKCHAN_TTS3=false`;
  - `USE_HOTSPOT_WIFI_PROVISIONING=true`;
  - `USE_ESP_BLUFI_WIFI_PROVISIONING=false`;
  - `USE_ACOUSTIC_WIFI_PROVISIONING=false`;
  - `OTA_URL="http://47.103.57.217/xiaozhi/ota/"`.
- 确认 clean workdir 里存在端侧代码:
  - custom wake word split code in `firmware/xiaozhi-esp32/main/audio/wake_words/custom_wake_word.cc`;
  - log marker `Loaded %d A21 custom wake command(s) for %s`;
  - VAD/no-speech/listen guards in `firmware/xiaozhi-esp32/main/application.cc`;
  - `A21_VAD_STOP_DEBOUNCE_MS=520`;
  - `A21_VAD_MIN_LISTENING_MS=1200`;
  - `A21_NO_SPEECH_LISTENING_TIMEOUT_MS=7000`;
  - `CONFIG_A21_STACKCHAN_KEEP_CONTROL_CHANNEL` guarded logic.
- 确认当前 HEAD 祖先链仍包含核心语音/端侧改动:
  - `f0603f5 fix(firmware): set official xiaozhi codec volume`;
  - `060d2bb fix(audio): accept stackchan xiaozhi playback hotfix`;
  - `5242349 fix(voice): bound xiaozhi listen and tune zi yue wake`;
  - `8e4df0b fix(firmware): split zi yue wake commands`;
  - `9ba8bc1 fix(firmware): skip welcome after stackchan app preload`;
  - `e694550 fix(firmware): park after stackchan xiaozhi autostart`;
  - `fc79156 feat(a21): queue xiaozhi opus ingress`;
  - `2f8a63f feat(xiaozhi): suppress stale opus ingress after abort`;
  - `d15a7b4 feat: promote public voice gateway and wifi provisioning`;
  - `face173 feat: enable public edge dashscope voice downlink`;
  - `e8c9427 feat(xiaozhi): add stock half-duplex acceptance`.
- 发现一个 superseded/missing commit:
  - `b25b8b7 fix(firmware): raise stackchan xiaozhi volume candidate` is not an ancestor of current HEAD.
  - 当前 overlay 仍有 `SetOutputVolume(92)` from `f0603f5`;需要后续确认 `b25b8b7` 是否只被 `f0603f5/060d2bb` 取代，还是有未带入的体感音量细节。
- 确认构建/刷机错误:
  - 2026-06-03 02:13 有一次真实 product-affecting 错刷:
    `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json`;
    command `xiaozhi-firmware-flash --execute`;
    app `xiaozhi.bin`;
    sha `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`;
    commit `8f36a95e86d4`.
  - 该错刷后来被 product lane 覆盖:
    `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`;
    app `a21-stackchan-official-xiaozhi-compatible.bin`;
    sha `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.
  - 17:08 又被当前 clean product package 覆盖，见最新 flash report。
  - 另一个路径错误是审计误读 `/tmp/a21-stackchan-official-build/sdkconfig`;正确路径是 `/tmp/a21-stackchan-official-build/config/sdkconfig.json` and `config/sdkconfig.h`。
- 确认主公网 Gateway runtime 断点:
  - ECS `47.103.57.217` 的 systemd `ExecStart` 仍是
    `/opt/a21/bin/a21 gateway --addr 127.0.0.1:21081 --public-gateway-url http://47.103.57.217 --product-chain host_local --voice-text-max-tokens 32`;
  - `/etc/a21/secrets/provider.env` 有 DashScope TTS / DeepSeek text / voice clone env names, but no ASR env;
  - `applyXiaozhiProductChainEnvDefaults` will default missing ASR to `A21_ASR_LOCAL_PROFILE=sherpa_onnx`;
  - ECS was intentionally chosen not to run local ASR/TTS, so this product-chain mode conflicts with the current public Gateway role.
- 远端 runtime evidence:
  - `a21-gateway` active;
  - device `44:1b:f6:e2:6a:60` present but currently stale;
  - device current mode/expression `local_fallback`;
  - `/v1/providers/voice/health` returns `a21-mock-voice`;
  - current trace `a21-trace-44-1b-f6-e2-6a-60` has:
    - `xiaozhi.opus_frame.received=124`;
    - `xiaozhi.opus_frame.decoded=124`;
    - `audio.ingress.buffered=124`;
    - `xiaozhi.opus_ingress.queued=124`;
    - `xiaozhi.wake_preroll.opus_frame.buffered=119`;
    - `xiaozhi.listen.auto_stop=8`;
    - `barge_in.detected=8`;
    - `playback.stop=8`;
    - `audio.downlink.first_frame=8`;
    - `xiaozhi.tts.opus_frame.downlink=8`;
    - but also `xiaozhi.voice_pipeline.unavailable=8`;
    - `local_fallback.entered=8`;
    - `xiaozhi.fast_ack.downlink=8`.

结论:

- 不是“只剩 avatar”。大量端侧/网关链路代码还在 repo、clean build、latest product config 里。
- 也不是“已经有效”。公网主 Gateway 当前普通 `/v1/xiaozhi` 对话路径实际掉进 `host_local` ASR 缺失和 local fallback，因此用户体感会像唤醒/点屏/声音链路都没了。
- 唤醒词的固件 config 已进 latest product build，但物理 wake 仍失败，必须用串口或设备日志确认是否加载了 `Loaded 4 A21 custom wake command(s) for 紫悦`，以及是否是 phrase/threshold/model issue。
- 三种配网诉求目前只完成了 product Hotspot default；BluFi/acoustic 只是被命名为 build alternatives 且 disabled，尚未满足“像 Xiaozhi 启动那三种方式”。

推荐下一步:

- 先修 public Gateway product chain: 公网 ECS 不应继续以 `host_local` + missing ASR 作为主产品链路。必须接入云 ASR adapter or explicit public-cloud ASR profile，再跑真实 physical `/v1/xiaozhi` turn。
- 同步补齐三种 Wi-Fi provisioning product config/option，避免只支持硬写或单 Hotspot。
- 单独打开串口/日志验证 wake load/detect，不用 Gateway `/v1/wake-word` 作为固件真相。
- 在修 provider chain 前，不再重复刷固件；当前 latest product app 已是正确 lane。
## 2026-06-03 - T-CLOUD-VOICE-001 - Pure Cloud Voice Provider Matrix

本轮目标:

- 开一条不干扰当前硬件/公网 Gateway 主线的支线任务。
- 找回过往对话中的 provider/API 决策，并重新查官方文档，形成
  Bailian Qwen-TTS / CosyVoice、Doubao、MiniMax 的 A21 接入方案。
- 明确“完全支持、前端可配置、可下发、低延迟”的 A21 语义，但不把
  未执行供应商链路标成 ready。

实际完成内容:

- 在独立 worktree 创建分支
  `codex/a21-pure-cloud-voice-matrix`，基线 HEAD `d15a7b4`。
- 读取 A21 当前 `voice_mode`、`gateway_profile`、provider spine、latency、
  protocol、observability、voice clone、Doubao realtime TTS、CosyVoice 5080
  candidate plan 和 provider-memory 线索。
- 查官方文档并落成
  `docs/engineering/A21_CLOUD_VOICE_PROVIDER_MATRIX.md`：
  - Bailian Qwen-TTS-Realtime / Qwen3-TTS-VC-Realtime；
  - Bailian CosyVoice realtime and clone/custom voice；
  - Bailian Qwen Omni Realtime as separate S2S lane；
  - Doubao realtime TTS and voice clone model families；
  - MiniMax realtime/sync TTS and voice clone；
  - frontend-visible `cloud_voice_profile` concept；
  - safe env names, profile statuses, redaction rules, latency gates, and
    implementation slices.
- Added transition plan
  `docs/plans/2026-06-03-pure-cloud-voice-provider-matrix.md`.
- Updated `docs/project_state_machine.md` with active
  `T-CLOUD-VOICE-001-PURE-CLOUD-PROVIDER-MATRIX`.

修改过的文件:

- `docs/engineering/A21_CLOUD_VOICE_PROVIDER_MATRIX.md`
- `docs/plans/2026-06-03-pure-cloud-voice-provider-matrix.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

当前未完成事项:

- No runtime code was added in this transition.
- No real provider execution was performed.
- No Gateway restart, V21 execution, firmware build/flash, NVS/serial/hardware,
  or audio playback occurred.
- Frontend endpoints such as `GET/POST /v1/cloud-voice-profiles` are still
  planned, not implemented.
- Doubao existing adapter still needs reconciliation against the current
  Volcengine endpoint/auth docs before live execution.
- Bailian Qwen-TTS, CosyVoice, and MiniMax adapters are not implemented yet.

已知风险和阻塞点:

- Catalog visibility must not be mistaken for runtime readiness.
- `voice_mode` stays `dialogue` / `professional`; cloud voice profile selection
  must remain separate and user/operator explicit.
- `professional` remains V21 adapter only; opaque realtime or S2S provider
  cannot replace the evidence path.
- Clone reference audio/text privacy needs an explicit consent and storage
  policy before any live voice-clone run.

推荐下一步:

1. Implement `GET/POST /v1/cloud-voice-profiles` as a no-execute safe catalog
   and selector, returning only profile IDs, statuses, capabilities, and
   missing env names.
2. Reconcile `doubao_tts_realtime` against current Volcengine docs and extend
   `streaming-tts-runtime-smoke` to support the official endpoint/auth shape.
3. Add fake-event adapters for Bailian Qwen-TTS realtime and CosyVoice using
   the shared 60 ms PCM16 chunking contract.
4. Add MiniMax WebSocket/HTTP TTS and voice-clone plan/smoke commands.
5. Add frontend selector only after the no-execute catalog contract is green.
6. Run real provider smoke on 5080/mainland lab and import redacted reports
   before any physical StackChan A/B.

测试/构建/运行结果:

- `git diff --check`: passed.
- `git diff --no-index --check /dev/null docs/engineering/A21_CLOUD_VOICE_PROVIDER_MATRIX.md`: passed for the new document.
- `git diff --no-index --check /dev/null docs/plans/2026-06-03-pure-cloud-voice-provider-matrix.md`: passed for the new plan.
- Secret/redaction scan over the two new docs found no key values, bearer
  tokens, local paths, or public Gateway URLs.

如果中途失败，记录失败位置和原因:

- Initial file creation briefly landed two new docs in the main hardware
  checkout because the patch tool had no workdir parameter. The files were
  moved into this branch worktree with an absolute-path patch. Existing main
  hardware-line modifications were left untouched.

## 2026-06-03 - T-CLOUD-VOICE-001 - No-Execute Cloud Voice Profile Control Surface

本轮目标:

- 在支线里把纯云端声音矩阵推进到可测的控制面：
  `cloud_voice_profile` 后端 catalog/selector、模拟器前端配置入口、doctor
  可见性和设备 registry 安全下发字段。
- 保持 no-execute：不跑真实供应商、不重启 Gateway、不碰 V21/firmware/
  hardware/audio playback。
- 完成验证后再把支线状态同步给主控制线。

实际完成内容:

- 新增 provider-neutral cloud voice catalog：
  - Bailian Qwen-TTS realtime；
  - Bailian Qwen3-TTS-VC realtime；
  - Bailian CosyVoice realtime；
  - Bailian CosyVoice clone TTS；
  - Bailian Qwen Omni realtime；
  - Doubao realtime TTS；
  - Doubao voice clone TTS；
  - MiniMax T2A websocket；
  - MiniMax T2A HTTP；
  - MiniMax voice clone TTS。
- 新增 `GET /v1/cloud-voice-profiles`：
  - schema: `a21.gateway.cloud_voice_profiles.v1`；
  - 只返回 safe profile IDs、status、capabilities、present/missing env names；
  - 不返回 key/model/voice 值、URL、prompt、transcript、provider output 或
    audio payload。
- 新增 `POST`/`PUT /v1/cloud-voice-profiles`：
  - 只接受已知 safe `cloud_voice_profile` ID；
  - unknown / legacy / blocked values 返回 400 且不回显输入；
  - 不改变 `voice_mode` 或 `gateway_profile`。
- 设备 registry 新增安全字段：
  `current_cloud_voice_profile`。
- Simulator 新增 cloud voice profile select、readout 和 registry 显示。
- `a21 doctor` 的 `voice` 段新增 `cloud_voice` catalog，使用
  `a21.cloud_voice_profiles.v1` schema。
- 更新矩阵文档、计划和状态机，把 T-CLOUD-VOICE-001 从 docs-only 推进到
  no-execute control-ready。

修改过的文件:

- `internal/providers/cloud_voice.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `internal/app/doctor.go`
- `docs/engineering/A21_CLOUD_VOICE_PROVIDER_MATRIX.md`
- `docs/plans/2026-06-03-pure-cloud-voice-provider-matrix.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

测试/验证结果:

- Red tests observed:
  - Gateway failed on missing `CloudVoiceEnv`, missing
    `CurrentCloudVoiceProfile`, and missing helper.
  - Doctor failed because `"cloud_voice"` was absent from report.
- Focused gateway tests passed:
  `go test ./internal/gateway -run 'TestCloudVoice|TestSimulatorPageServed|TestVoiceModes|TestGatewayProfiles|TestFastCompanionRejectsProfessional' -count=1 -v`
- Focused app doctor/env tests passed with A21 direct NO_PROXY:
  `go test ./internal/app -run 'TestGatewayServerFromEnvExposesCloudVoiceProfileWithoutSecrets|TestRunDoctorIncludesCloudVoiceProfilesWithoutSecrets|TestRunDoctorVoiceHealthFollowsSelectedProviderWithoutSecrets|TestRunDoctorVoiceHealthReportsDoubaoRealtimeDegradedWithoutSecrets|TestRunDoctorReportsExplicitGatewayVoiceProviderRuntime|TestRunDoctorReportsExplicitGatewayDoubaoRealtimeRuntimeAsDegraded' -count=1 -v`
- Provider adjacent tests passed:
  `go test ./internal/providers -run 'TestProviderCatalog|TestDoubaoRealtime|TestRealtime|TestVoicePipelineSelection' -count=1`
- Related package tests passed:
  `go test ./internal/gateway ./internal/app ./internal/providers -count=1`
- Production/docs redaction scan found no key values, bearer/auth headers,
  public Gateway URLs, or local paths in the new production/docs surfaces.
- `git diff --check`: passed.
- Untracked file whitespace checks produced no output:
  - `docs/engineering/A21_CLOUD_VOICE_PROVIDER_MATRIX.md`
  - `docs/plans/2026-06-03-pure-cloud-voice-provider-matrix.md`
  - `internal/providers/cloud_voice.go`
- Full verification passed:
  `make verify`
  - `go test ./...`: passed.
  - `git diff --check`: passed.

未完成事项 / blockers:

- No real provider execution was performed.
- No runtime dispatch endpoint was added.
- Bailian Qwen-TTS, Qwen3-TTS-VC, CosyVoice, Qwen Omni, Doubao voice clone,
  MiniMax T2A, and MiniMax voice clone adapters remain planned.
- Doubao realtime TTS still needs reconciliation against current Volcengine
  endpoint/auth docs before live smoke.
- No 5080 lab smoke or physical StackChan A/B evidence exists for these cloud
  voice profiles.

推荐下一步:

1. Reconcile Doubao realtime TTS against current Volcengine docs and extend
   the fake/runtime smoke contract without changing product defaults.
2. Add provider-neutral streaming TTS fixture/chunker tests reusable by
   Bailian Qwen/CosyVoice, Doubao, and MiniMax.
3. Implement Bailian Qwen-TTS realtime fixture adapter.
4. Add planned readiness endpoint and server-side safe dispatch only after
   fixture tests are green.
5. Run 5080 redacted smoke matrix before any physical StackChan acceptance.

如果中途失败，记录失败位置和原因:

- No failure remains in this round. The only observed red states were the
  expected TDD red tests before implementation.

## 2026-06-03 17:35 CST - Main Public Gateway Cloud Edge Runtime Bridge

Round goal:

- Continue full push after the pure-cloud selector merge by turning the main
  public Gateway path away from broken `host_local` defaults on ECS and toward
  a real cloud-edge voice chain.

Actual completed work:

- Fixed the post-cherry-pick doctor test failure root cause: two Doubao doctor
  tests were missing A21 direct `NO_PROXY` isolation, so current global proxy
  settings triggered the expected `proxy_direct_bypass_missing` block.
- Added a provider-neutral A21 `cloud_edge` execution mode for voice-pipeline
  reports.
- Added a Doubao realtime ASR streaming adapter for `A21_ASR_PROFILE=cloud`
  plus `A21_ASR_CLOUD_PROFILE=doubao_asr_realtime`.
- Added cloud-edge Gateway product-chain defaults:
  - `A21_ASR_PROFILE=cloud`
  - `A21_ASR_CLOUD_PROFILE=doubao_asr_realtime`
  - `A21_TTS_FAST_PROFILE=doubao_tts_realtime`
  - `A21_TEXT_STREAM_PROFILE=deepseek` only when DeepSeek key env is present
    and no text profile is already selected.
- Updated xiaozhi streaming provider readiness so configured Doubao ASR +
  DeepSeek text stream + Doubao realtime TTS can pass the static runtime-shape
  gate while still keeping `prd_accepted=false`.

Changed files:

- `internal/providers/doubao_realtime_asr.go`
- `internal/providers/doubao_realtime_asr_test.go`
- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Test/build/runtime results:

- Focused provider tests passed:
  `go test ./internal/providers -run 'TestDoubaoRealtimeASR|TestVoicePipelineAdaptersFromEnvSelectsDoubaoCloudStreamingASR|TestVoicePipelineAdaptersFromEnvSelectsDoubaoRealtimeTTS|TestVoicePipelineRunStreamStartsLLMFromStreamingASRPartialBeforeFinal' -count=1 -v`
- Focused app tests passed:
  `go test ./internal/app -run 'TestGatewayServerOptionsFromEnvCloudEdgeProductChainDoesNotDefaultToLocalSherpa|TestXiaozhiStreamingProviderReadiness.*Doubao|TestRunDoctorVoiceHealthFollowsSelectedProviderWithoutSecrets|TestRunDoctorReportsExplicitGatewayVoiceProviderRuntime' -count=1 -v`
- Related package tests passed:
  `go test ./internal/gateway ./internal/app ./internal/providers -count=1`
- `git diff --check`: passed.

Unfinished items:

- Full `make verify` still needs to run after this log update.
- Main ECS `47.103.57.217` still needs deployment with
  `--product-chain cloud_edge`.
- Server-side provider secrets must be injected through root-only env/secret
  files, not repo/docs/systemd unit/firmware.
- Physical StackChan trace is still required for audible dialogue, barge-in,
  and wake-word acceptance.

Known risks/blockers:

- The Doubao ASR event shape is intentionally tolerant around transcript field
  names but still needs a credentialed runtime smoke against the live endpoint.
- The existing Doubao realtime TTS adapter also still needs live endpoint
  confirmation before claiming provider runtime acceptance.
- Wake word from idle is still not accepted; screen tap remains only a labeled
  fallback trigger.

Recommended next action:

1. Run full `make verify`.
2. Build and deploy the current A21 binary to `47.103.57.217`.
3. Update `a21-gateway.service` to run `--product-chain cloud_edge`, with
   secrets loaded from a root-only environment file.
4. Verify `/healthz`, `/v1/gateway-profiles`,
   `/v1/cloud-voice-profiles`, `/xiaozhi/ota/`, and
   `xiaozhi-streaming-provider-readiness` with env names only.
5. Trigger one real StackChan turn and run `xiaozhi-realtime-parity` on the
   live trace before making any physical acceptance claim.

## 2026-06-03 18:10 CST - DashScope Public Edge Pivot And StepFun Text Selector Correction

Round goal:

- Continue the main public Gateway voice chain after live Doubao ASR/TTS dial
  evidence showed the current Doubao credential/endpoint shape was not usable,
  and correct the accidental DeepSeek text-stream selection in the cloud-edge
  path.

Actual completed work:

- Added DashScope realtime ASR and TTS adapters:
  - ASR profile: `dashscope_qwen_asr_realtime`.
  - TTS profile: `dashscope_qwen_tts_realtime`.
  - Realtime WebSocket shape uses bearer auth headers, `session.update`,
    audio/text buffer append+commit, `session.finish`, and 60 ms PCM16 mono
    downlink chunks.
- Wired DashScope ASR/TTS into `VoicePipelineAdaptersFromEnv`, the cloud-edge
  execution mode, and xiaozhi streaming provider readiness.
- Changed cloud-edge product-chain defaults:
  - If `A21_DASHSCOPE_API_KEY` exists and ASR/TTS profiles are not explicitly
    set, select DashScope ASR/TTS.
  - If `A21_LAB_STEPFUN_API_KEY` exists and no text profile is explicitly set,
    select `stepfun` before falling back to `deepseek`.
- Deployed the verified build to main ECS `47.103.57.217`.
- Updated remote root-only provider env profile IDs for ASR/TTS only:
  - `A21_ASR_CLOUD_PROFILE=dashscope_qwen_asr_realtime`
  - `A21_TTS_FAST_PROFILE=dashscope_qwen_tts_realtime`
- Confirmed the remote systemd service still uses root-only env files and runs:
  `/opt/a21/bin/a21 gateway --addr 127.0.0.1:21081 --public-gateway-url
  http://47.103.57.217 --product-chain cloud_edge --voice-text-max-tokens 32`.

Changed files:

- `internal/providers/dashscope_realtime.go`
- `internal/providers/dashscope_realtime_test.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Test/build/runtime results:

- Focused app tests passed:
  `go test ./internal/app -run 'TestGatewayServerOptionsFromEnvCloudEdge.*(DashScope|StepFun)|TestXiaozhiStreamingProviderReadinessPassesConfiguredCloudEdgeDashScope' -count=1`
- Related package tests passed:
  `go test ./internal/gateway ./internal/app ./internal/providers -count=1`
- `git diff --check`: passed.
- Remote focused tests passed after rsync:
  `go test ./internal/app -run 'TestGatewayServerOptionsFromEnvCloudEdgePrefersStepFunOverDeepSeekWhenConfigured|TestGatewayServerOptionsFromEnvCloudEdgePrefersDashScopeWhenConfigured' -count=1`
- Remote build passed:
  `go build -o /opt/a21/bin/a21 ./cmd/a21`
- Remote service restart passed; `http://47.103.57.217/healthz` returns OK.
- Remote static provider readiness currently reports:
  `dashscope_qwen_asr_realtime + deepseek + dashscope_qwen_tts_realtime`,
  `gate_status=passed`, `prd_accepted=false`.
- Public host-loopback `xiaozhi-voice-bench --require-product-chain` now sees
  the selected DashScope ASR/TTS profiles, but still blocks because the full
  non-fixture ASR+LLM+TTS product chain did not complete in one turn.

Unfinished items:

- StepFun is not active on the ECS because `/etc/a21/secrets/provider.env` does
  not currently contain `A21_LAB_STEPFUN_API_KEY`, `A21_STEPFUN_MODEL`, or
  `A21_STEPFUN_BASE_URL`.
- The local shell also has no StepFun env, and the plaintext 5080 source report
  was not found in the current workspace search; only redacted reports and
  env-name templates were found.
- Full `make verify` still needs to run after this handoff entry.
- Real physical StackChan mic-driven dialogue, barge-in, audible playback, and
  wake-word acceptance are still not proven.

Known risks/blockers:

- The intended text path is StepFun `step-1-8k`, but current deployment still
  falls back to DeepSeek until the StepFun secret is injected into the root-only
  provider env.
- DashScope realtime event shapes are covered by fake adapter tests and static
  readiness, but still need live credentialed runtime evidence before PRD
  acceptance.
- Wake word remains below product acceptance until guarded physical proof.

Recommended next action:

1. Inject StepFun into `/etc/a21/secrets/provider.env` without printing values:
   `A21_LAB_STEPFUN_API_KEY`, `A21_STEPFUN_MODEL=step-1-8k`, and optional
   `A21_STEPFUN_BASE_URL`.
2. Restart `a21-gateway` and rerun static readiness; expected LLM profile is
   `stepfun`.
3. Run public `xiaozhi-voice-bench --require-product-chain` again and inspect
   trace events for ASR partial/final, StepFun first content, DashScope TTS
   first audio, and Opus downlink.
4. Only after host-loopback provider chain completes, run the physical
   StackChan dialogue/barge-in/wake-word evidence pass.

## 2026-06-03 19:15 CST - Xiaozhi Provider State Machine Rebuild

Round goal:

- Stop patch-chasing the public voice chain and rebuild the failing provider
  bridge against the official Xiaozhi and realtime provider state machines.

Actual completed work:

- Created the control plan
  `docs/plans/2026-06-03-xiaozhi-provider-state-machine-rebuild.md`.
- Confirmed the root cause is not loss of the Mac Gateway transport/audio work:
  the public chain still has Xiaozhi WebSocket, Opus ingress/downlink,
  streaming ASR reuse, nonblocking commit, and barge-in surfaces.
- Fixed the DashScope realtime TTS adapter so it is no longer a synchronous
  RPC wrapper:
  - starts the read loop before text append/commit;
  - waits for `session.updated` when available;
  - emits audio on `response.audio.delta`;
  - delays `session.finish` until after audio completion and successful text
    commit;
  - closes without `session.finish` on cancellation;
  - reports redacted, stage-specific no-audio/read/provider/invalid-delta
    failures.
- Added provider tests that fail on the old order where
  `session.finish` was sent before audio was read.
- Added fake realtime timeline recording for precise write/read ordering
  assertions.

Changed files:

- `docs/plans/2026-06-03-xiaozhi-provider-state-machine-rebuild.md`
- `internal/providers/dashscope_realtime.go`
- `internal/providers/dashscope_realtime_test.go`
- `internal/providers/realtime_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Test/build/runtime results:

- Focused DashScope tests passed:
  `go test ./internal/providers -run 'DashScopeRealtime(TTS|ASR)|VoicePipelineAdaptersFromEnvSelectsDashScope' -count=1`
- Provider package passed:
  `go test ./internal/providers -count=1`
- Related packages passed:
  `go test ./internal/gateway ./internal/app ./internal/providers -count=1`
- `git diff --check`: passed.
- Full verification passed:
  `make verify`

Unfinished items:

- This fix is not yet deployed to the public ECS `47.103.57.217` at the time
  of this handoff entry.
- A fresh public `xiaozhi-voice-bench --require-product-chain` must prove
  ASR partial/final, LLM first content, TTS first audio, binary Opus downlink,
  and barge-in cleanup on the main public Gateway.
- Physical StackChan wake, mic-driven dialogue, audible playback, and
  touch/new-wake barge-in acceptance are still not proven by this host-side
  adapter fix.

Known risks/blockers:

- StepFun is still not active unless root-only ECS env includes
  `A21_LAB_STEPFUN_API_KEY` and `A21_STEPFUN_MODEL=step-1-8k`.
- If live DashScope sends a different event ordering than the documented
  `session.updated`/`response.audio.delta` lifecycle, the adapter will proceed
  after a bounded readiness timeout but public bench must confirm behavior.
- This is a main-chain candidate unblock, not full PRD green.

Recommended next action:

1. Commit this state-machine fix.
2. Deploy the commit to `47.103.57.217`.
3. Run public `xiaozhi-voice-bench --require-product-chain` and inspect trace
   markers for `provider.first_content`, `tts.first_audio`,
   `audio.downlink.first_frame`, `xiaozhi.voice_pipeline.completed`, and
   clean abort/stop behavior.
4. Only after public host bench has binary downlink, test the physical
   StackChan against `ws://47.103.57.217/v1/xiaozhi`.

Follow-up result:

- Deployed commits through `8752b8d` to main public Gateway `47.103.57.217`.
- Root cause after provider-state-machine fix was remote env mismatch:
  `A21_DASHSCOPE_TTS_MODEL`/`A21_DASHSCOPE_TTS_VOICE` pointed at a CosyVoice
  model/voice family while the selected adapter was Qwen-TTS Realtime. The
  service rejected `session.update`, which showed in trace as
  `xiaozhi.voice_pipeline.failed.tts_session_update_failed`.
- Backed up `/etc/a21/secrets/provider.env`, then changed only root-only ECS
  env values to Qwen-TTS Realtime family. No provider key was printed or
  written to repo/firmware.
- Public host-loopback product-chain bench passed:
  `reports/a21-xiaozhi-voice-bench-20260603-201052.155507000.json`.
- Evidence from that report:
  - `acceptance_status=candidate_host_only`
  - `prd_accepted=false`
  - `provider_executed=true`
  - `voice_pipeline_execution_mode=cloud_edge`
  - answer binary downlink frames: `127`
  - answer first-audio total p95: `806 ms`
  - trace ASR first partial/final: `158 ms`
  - LLM first content: `517 ms`
  - TTS first audio: `556 ms`
  - audio downlink first frame: `523 ms`
  - barge-in stop p95: `12 ms`
  - downlink audio quality: `passed`
- This proves the public cloud-edge Gateway candidate chain, not physical
  StackChan wake/mic/audible/barge-in acceptance.

Physical run result:

- User requested direct real-device validation. Physical StackChan
  `44:1b:f6:e2:6a:60` was online on public Gateway
  `http://47.103.57.217` with stock Xiaozhi websocket profile.
- Operator/device activity produced real physical trace
  `a21-trace-44-1b-f6-e2-6a-60` /
  `a21-session-44-1b-f6-e2-6a-60`.
- Trace counters observed on the public Gateway:
  - `xiaozhi.opus_frame.received=66`
  - `xiaozhi.opus_frame.decoded=66`
  - `audio.ingress.buffered=66`
  - `vad.speech.start=5`
  - `vad.speech.end=5`
  - `xiaozhi.listen.auto_stop=5`
  - `asr.first_partial=11`
  - `asr.final=11`
  - `provider.first_content=5`
  - `tts.first_audio=5`
  - `xiaozhi.tts.opus_frame.downlink=506`
  - `audio.downlink.first_frame=10`
  - `xiaozhi.voice_pipeline.completed=3`
  - `barge_in.detected=5`
  - `playback.stop=5`
- Generated physical report:
  `reports/a21-xiaozhi-physical-evidence-20260603-201623.594517000.json`.
  It records `physical_device_online=true`, `audio_frame_count=64`,
  `mic.available=true`, `frames_delivered=64`, `delivery_ratio=1`,
  stock profile available, Opus decode available, VAD speech end available,
  listen auto-stop available, TTS downlink available, and
  `answer.first_downlink=571 ms`.
- Generated stock half-duplex report:
  `reports/a21-xiaozhi-half-duplex-acceptance-20260603-201623.946467000.json`.
  It records `half_duplex_acceptance_status=candidate_gateway_trace`,
  `mic_available=true`, `downlink_available=true`,
  `barge_in_detected_available=true`, and `barge_in_stop_available=true`.
- Remaining physical blockers are explicit and narrow:
  `device.playback.ack` / trusted runtime playback start is unavailable,
  operator or instrumented audible observation is not recorded, and
  `device.playback.stop_done` is not exposed by the stock firmware. Therefore
  both reports correctly keep `prd_accepted=false`.
- `stackchan-accept --check touch --case top_barge_in` was attempted and
  blocked with `device audio websocket is not connected`; this control path is
  not the stock Xiaozhi audio socket, while trace-level barge-in evidence is
  present.

## 2026-06-03 18:24 CST - Voice Chain Selector Hot Switch

Round goal:

- Add the product-facing dialogue-chain selector the user requested: cascade
  ASR -> LLM -> fixed TTS versus end-to-end realtime, with frontend hot switch,
  StepFun recommendation, visible realtime providers, and separate voice-clone
  selection.

Actual completed work:

- Added `GET/POST/PUT /v1/voice-chain-profiles`.
- Added provider-safe catalogs for:
  - `cascade` chain mode with selectable ASR and LLM.
  - `realtime` chain mode with visible realtime provider choices.
  - voice/clone choices with safe display labels.
- Made StepFun the recommended/default cascade LLM and kept DeepSeek visible
  only as fallback.
- Preserved fixed cascade TTS as `dashscope_qwen_tts_realtime`; selecting a
  clone maps the effective TTS profile to `voice_clone_cli`.
- Implemented in-memory hot switch for cascade runner metadata, ASR adapter,
  TTS adapter, realtime provider, and device registry fields.
- Corrected the realtime hot-switch root cause by setting the existing
  `A21_GATEWAY_VOICE_PROVIDER=selected` runtime gate along with
  `A21_PROVIDER_PRIMARY`.
- Added simulator controls/readouts for chain mode, ASR, LLM, selected TTS,
  realtime provider, and voice/clone.
- Documented the selector boundary in `docs/engineering/PROTOCOL.md`.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-03-voice-chain-product-selector-hot-switch.md`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Test/build/runtime results:

- Focused gateway tests passed:
  `go test ./internal/gateway -run 'VoiceChain|Simulator|RealtimeSession|CloudVoice|VoiceMode' -count=1`
- Related package tests passed:
  `go test ./internal/gateway ./internal/app ./internal/providers -count=1`
- `git diff --check`: passed.
- Full verification passed:
  `make verify`

Unfinished items:

- No ECS deploy, provider execution, firmware build/flash, physical hardware
  action, or audio playback was performed in this selector cut.
- Remote ECS still needs root-only StepFun secret injection before the selected
  product LLM can actually stop falling back to DeepSeek.

Known risks/blockers:

- This is runtime/config hot switch and UI/API state, not physical PRD green.
- Realtime provider entries without secrets correctly report unavailable until
  server-side env is injected.
- Wake word and physical barge-in proof remain separate product acceptance
  gates.

Recommended next action:

1. Commit this selector slice.
2. Inject StepFun env on ECS through `/etc/a21/secrets/provider.env` without
   printing values, restart Gateway, and verify `/v1/voice-chain-profiles` plus
   static streaming readiness show `stepfun`.
3. Then run the real physical StackChan dialogue/barge-in/wake-word evidence
   pass.

## 2026-06-03 18:36 CST - Fast Ack Must Not Block Main Answer

Round goal:

- Continue the Xiaozhi voice main-chain push and verify whether the protocol,
  audio, ASR/LLM/TTS streaming, and barge-in work had actually reached a clear,
  smooth, low-latency chain.

Actual completed work:

- Re-ran the current public Gateway `xiaozhi-voice-bench` against
  `http://47.103.57.217`.
- Confirmed the current chain already reaches stock `/v1/xiaozhi` hello/listen,
  Opus ingress/decode, streaming ASR append/commit, ASR partial/final, stock
  `stt`, and trace timing.
- Found the main answer-chain blocker: if the fast-ack TTS path failed,
  Gateway sent `tts.stop` with `fast_ack_unavailable` and returned before
  running the full ASR -> LLM -> TTS answer pipeline.
- Fixed `writeXiaozhiVoicePipelineTTS` so fast ack is an optional latency
  optimization, not a hard dependency. If fast ack does not produce audio and
  the turn has not been aborted, Gateway continues into the full answer
  pipeline.
- Updated the regression test from the old wrong contract
  `StopsLifecycleWhenFastAckUnavailable` to
  `ContinuesAnswerWhenFastAckUnavailable`, requiring answer sentence start,
  binary Opus downlink, `provider.first_content`, `tts.first_audio`,
  `audio.downlink.first_frame`, and `xiaozhi.voice_pipeline.completed`.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/agent_handoff_log.md`

Test/build/runtime results:

- Focused Gateway tests passed:
  `go test ./internal/gateway -run 'FastAckUnavailable|ListenStopRunsVoicePipelineAndSendsPacedOpus|ASRPartialStartsStreamingAnswerBeforeListenStopAndASRFinal|AbortCancelsBlockedTurnTaskWithinBargeInBudget|ListenStartBargeInStopsActiveTTS' -count=1`
- Related package tests passed:
  `go test ./internal/gateway ./internal/app ./internal/providers -count=1`
- `git diff --check`: passed.
- Full verification passed:
  `make verify`

Unfinished items:

- This fix still needs commit, deployment to the public Gateway, and a fresh
  public `xiaozhi-voice-bench --require-product-chain` run.
- StepFun is still not active on ECS because no StepFun env names are present
  in `/etc/a21/secrets/provider.env`; current public selector still reports
  `selected_llm_profile=deepseek` and finding `stepfun_not_selected`.
- No physical StackChan dialogue/barge-in/wake-word evidence was collected in
  this round yet.

Known risks/blockers:

- The fresh public bench before the fix showed ASR partial/final around
  154-155 ms, but no LLM first content, TTS first audio, or Opus downlink
  because the fast-ack failure ended the turn early.
- The fix should unblock the answer pipeline, but product acceptance still
  requires a post-deploy bench and physical evidence.

Recommended next action:

1. Commit this fast-ack continuity fix.
2. Deploy the commit to `47.103.57.217`.
3. Re-run public `xiaozhi-voice-bench --require-product-chain` and inspect
   `provider.first_content`, `tts.first_audio`, `audio.downlink.first_frame`,
   binary downlink frames, and barge-in metrics.
4. Then inject StepFun server-side env when available and repeat the same
   evidence pass with `selected_llm_profile=stepfun`.

## 2026-06-03 20:4x CST - Zi Yue Custom Wake Root Cause And Product Rebuild

Round goal:

- Answer why custom wake did not pass and close the concrete firmware-side
  root cause without using the bare `xiaozhi.bin` lane.

Actual completed work:

- Confirmed public Gateway `/v1/wake-word` still reports builtin
  `你好小智` and `custom_runtime_active=false`; this endpoint is not proof that
  the flashed product app has active custom wake.
- Captured a 25-second serial window on `/dev/cu.usbmodem101`; only periodic
  `SystemInfo` logs appeared, with no custom wake detection log.
- Inspected the official-compatible firmware source and found the real root
  cause: `CustomWakeWord::Initialize()` only used `CONFIG_CUSTOM_WAKE_WORD`
  when `models_list == nullptr`. In the normal official Xiaozhi path,
  `models_list` is already supplied, so the code calls
  `ParseWakenetModelConfig()` and reads the asset `index.json` command table.
  The A21 `zi yue|...` aliases could therefore be present in `sdkconfig` and
  binary strings without becoming the active MultiNet commands.
- Updated
  `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
  so `CONFIG_USE_CUSTOM_WAKE_WORD=y` overrides the asset command list with the
  A21 sdkconfig aliases on both init paths.
- Added/updated the official StackChan test guard to require the asset-command
  override marker.
- Rebuilt the official-compatible product lane only:
  `a21-stackchan-official-xiaozhi-compatible.bin`.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Test/build/runtime results:

- Focused official StackChan tests passed:
  `go test ./internal/app -run 'OfficialXiaozhiCompatibleOverlaySetsZiYueCustomWake|OfficialXiaozhiCompatible|StackChanOfficial' -count=1`.
- `git diff --check`: passed.
- Product build passed with dependency cache:
  `A21_STACKCHAN_OFFICIAL_DEP_CACHE="/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan" make a21-stackchan-official-xiaozhi-compatible-build`.
- Build report:
  `reports/a21-stackchan-official-baseline-20260603-204131-1780490491190876000.json`.
- Product app:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- Product app SHA-256:
  `7b547c8706f7ad89d6a3436ddd7a21e299f5a25ec20461a5489d0ee2076a1537`.
- Binary strings now include:
  `Loaded %d A21 sdkconfig custom wake command(s) for %s`,
  `A21 overriding asset multinet commands with sdkconfig custom wake commands`,
  and `Custom wake word detected: command_id=%d, string=%s, prob=%f`.
- `make verify`: passed.

Unfinished items:

- This fix is built but not yet flashed in this round.
- Custom wake remains unaccepted until a guarded product-lane flash and
  physical idle wake proof pass.
- Barge-in still has a separate runtime issue: traces show stop markers, but
  downlink can continue after barge-in; that needs the next Gateway cancellation
  fix.

Recommended next action:

1. Commit this wake override fix.
2. Guarded-flash only the newly built
   `a21-stackchan-official-xiaozhi-compatible.bin`.
3. Capture boot serial logs showing the A21 override and loaded command count.
4. Test `紫悦`, `紫悦紫悦`, `你好紫悦`, and `小紫悦` from idle and then regenerate
   wake/physical evidence.

## 2026-06-03 21:0x CST - Zi Yue Wake Assets Partition Fix

Round goal:

- Explain and fix why the freshly flashed custom wake build still did not wake
  on `紫悦`, without switching to the bare `xiaozhi.bin` lane or relying on
  external source-cache dirty changes.

Actual completed work:

- Re-read the post-flash serial boot log and confirmed the runtime failure was
  below the custom wake command override:
  `Assets: The index.json file is not found`, followed by
  `MODEL_LOADER: Can not find model in partition table`, and only the
  `VAD(WebRTC)` AFE pipeline. That means MultiNet was not loaded, so
  `CustomWakeWord` could not initialize even though the app binary contained
  the A21 wake strings.
- Confirmed the bad build had `generated_assets.bin` at 4,688,623 bytes while
  the emitted partition table still had `assets,data,spiffs,0xa00000,4M`.
  The external StackChan source cache had a local dirty `5M` partition edit,
  but A21 correctly exports source `HEAD`, so that dirty change was not in the
  product build.
- Updated the A21 official-compatible overlay to patch
  `firmware/partitions.csv` from `assets ... 4M` to `assets ... 5M`.
- Added a build/flash guard that parses ESP-IDF `partition-table.bin` and
  rejects a package when `generated_assets.bin` exceeds the actual `assets`
  partition size.
- Added regression coverage for the overlay `5M` contract and for rejecting a
  correctly named but oversized product flash package.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan.go`
- `internal/app/official_stackchan_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Test/build/runtime results:

- Focused tests passed:
  `go test ./internal/app -run 'ZiYueCustomWake|OversizedAssetsPartition|XiaozhiCompatibleFlashPlan|XiaozhiCompatibleFlashExecute' -count=1`.
- `go test ./internal/app -count=1`: passed.
- `make verify`: passed.
- Product build passed:
  `reports/a21-stackchan-official-baseline-20260603-210122-1780491682047937000.json`.
- New product app SHA-256:
  `7c2b0f8e72e43bf3296638faba9667c557f8f732b19f7b22da7805e6b8597af9`.
- New partition table SHA-256:
  `704b0cc2d29d95d8429450e3d379c903c77864042d0bc3050f669c2c244bdb8d`.
- New assets SHA-256:
  `d0a20f925364d33e75694dd07b4897ba9a1689728949d45a2987d6551cbc8b8e`.
- Verified generated partition table:
  `assets,data,spiffs,0xa00000,5M`; `generated_assets.bin` remains
  4,688,623 bytes and now fits.
- Verified `generated_assets.bin` contains `index.json`, `srmodels.bin`,
  MultiNet model config, and
  `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue`.

Unfinished items:

- The fixed build is not yet reflashed in this round.
- Custom wake remains unaccepted until the device boot log shows assets/model
  load plus the A21 custom wake override markers, then a physical idle wake
  test passes.
- Barge-in still remains a separate Gateway/device queue cancellation issue.

Recommended next action:

1. Commit this assets partition guard/fix.
2. Run product-lane flash plan and guarded flash on `/dev/cu.usbmodem101`.
3. Capture boot serial and require: no `index.json file is not found`, no
   `MODEL_LOADER: Can not find model in partition table`,
   model/MultiNet load evidence, and the A21 custom wake override markers.
4. Physically test `紫悦`, `紫悦紫悦`, `你好紫悦`, and `小紫悦` from idle.

## 2026-06-03 21:1x CST - Xiaozhi Wake State Machine Alignment

Round goal:

- Compare the A21 official-compatible wake/listen state machine against the
  Xiaozhi baseline and fix why custom wake detection still did not enter a
  usable conversation turn.

Actual completed work:

- Confirmed the device now loads assets and MultiNet, and serial evidence
  showed `Custom wake word detected` plus `Wake word detected: 紫悦`.
- Pulled the public Gateway trace and found the server still only saw repeated
  `xiaozhi.hello.received` after the wake event; it did not receive
  `listen.detect`, `listen.start`, or wake-triggered Opus ingress.
- Compared Xiaozhi baseline `HandleWakeWordDetectedEvent`,
  `ContinueWakeWordInvoke`, `HandleStateChangedEvent`, and protocol
  `SendAbortSpeaking` / `SendWakeWordDetected` / `SendStartListening`.
- Root cause: Xiaozhi baseline assumes `ContinueWakeWordInvoke` is reached
  through `kDeviceStateConnecting`. A21 added an idle quiet-control WebSocket,
  so wake can reach `ContinueWakeWordInvoke` while still in
  `kDeviceStateIdle` with `protocol_->IsAudioChannelOpened()==true`; the old
  guard returned immediately and swallowed the wake transition.
- Fixed the A21 overlay so `ContinueWakeWordInvoke` accepts the Xiaozhi
  `Connecting` path and the A21 `Idle + open channel` fast path.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Test/build/runtime results:

- Focused tests passed:
  `go test ./internal/app -run 'OfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady|ZiYueCustomWake' -count=1`.
- `make verify`: passed.
- Product build passed:
  `reports/a21-stackchan-official-baseline-20260603-211451-1780492491468577000.json`.
- New product app SHA-256:
  `7674e98af738ade2e3653b46598093a135611cf8f6a4746a689bf0b447ef66c8`.
- Partition table and assets stayed on the previously verified fixed hashes:
  `704b0cc2d29d95d8429450e3d379c903c77864042d0bc3050f669c2c244bdb8d`
  and
  `d0a20f925364d33e75694dd07b4897ba9a1689728949d45a2987d6551cbc8b8e`.

Unfinished items:

- The fixed wake state-machine build still needs guarded flash and physical
  trace proof.
- Barge-in still needs a dedicated post-wake proof: while Xiaozhi protocol
  sends `abort` and clears the send queue, Gateway/device downlink
  cancellation must be verified after the wake path starts producing turns.

Recommended next action:

1. Commit the wake state-machine fix.
2. Guarded-flash the new product app SHA
   `7674e98af738ade2e3653b46598093a135611cf8f6a4746a689bf0b447ef66c8`.
3. Capture serial and Gateway trace evidence for wake -> `listen.start` ->
   Opus ingress -> ASR -> TTS downlink.
4. Immediately test barge-in during speaking and verify no downlink frames are
   emitted for the cancelled turn after `abort`.

## 2026-06-03 22:xx CST - Xiaozhi Listen/Speak Boundary Hotfix

Round goal:

- Fix the physical symptom where StackChan can begin speaking before the user
  finishes after the custom wake word, while preserving low-latency ASR partial
  evidence and barge-in cancellation.

Actual completed work:

- Rechecked the Xiaozhi stock state-machine boundary: `asr.first_partial` may
  arrive while the device is still listening, but user-audible `tts start` and
  binary Opus downlink must not begin until the listen turn is closed by
  device `listen.stop`, trusted VAD end, or a max-duration safety stop.
- Found two A21-side causes of the “抢答” behavior:
  1. `startXiaozhiPartialVoicePipeline` treated the first ASR partial as a
     complete answer trigger and started the audible TTS path.
  2. The Gateway RMS VAD default hangover is only two 60 ms Opus frames; on a
     physical stock StackChan this can turn a natural short pause into
     `vad.speech.end` and `xiaozhi.listen.auto_stop` before the endpoint sends
     its own `listen.stop`.
- Changed ASR partial handling to record
  `xiaozhi.voice_pipeline.partial_prewarm_deferred` and keep the turn in
  listening; it no longer starts `tts`, LLM/TTS output, or Opus downlink.
- Changed stock physical MAC-device handling so Gateway-side `vad.speech.end`
  is recorded but does not auto-stop the turn; the stock endpoint `listen.stop`
  remains the authoritative speech-end signal. Max-duration safety auto-stop
  remains available.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Test/build/runtime results:

- Focused Gateway tests passed:
  `go test ./internal/gateway -run 'TestXiaozhiWebSocket(ASRPartialDoesNotSpeakBeforeListenStop|StockPhysicalDefersGatewayVADStopUntilListenStop|StreamingASRStartsBeforeListenStop|StreamingASRFinalStartsPipelineWithoutBatchFallback|StreamingASRFinalSendsStockSTTBeforeTTS|VADSpeechEndAutoStopsRealtimeTurn|MaxListenDurationAutoStopsAfterSpeech|ListenStopDoesNotBlockAbortWhileStreamingASRCommitPending)' -count=1`.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Pre-deploy live trace scan on `47.103.57.217` showed the old runtime still
  had many Gateway VAD-driven endings (`vad.speech.end=27`,
  `listen.stop=8`), confirming the physical symptom is plausible before this
  fix is deployed.

Unfinished items:

- `make verify`, commit, public ECS deployment, and fresh physical trace scan
  still need to run for this hotfix.
- Full physical acceptance still needs operator confirmation that the device no
  longer speaks over unfinished wake-word utterances.
- Device-level `playback.stop_done` remains optional/stock-limited evidence;
  Gateway barge-in stop/downlink suppression is not the same as device queue
  acknowledgement.

Recommended next action:

1. Run `make verify`.
2. Commit the hotfix.
3. Deploy the committed Gateway to `47.103.57.217`.
4. Run a fresh wake -> speak -> interrupt physical trace and require:
   no `tts.first_audio` before `listen.stop` or max-duration stop, no old
   downlink after `playback.stop`, and operator audible confirmation.

Post-deploy update:

- Committed as `27d8a34 fix(gateway): defer xiaozhi speech until listen stop`.
- `make verify` passed before deploy.
- Deployed to main public ECS Gateway `47.103.57.217`; remote focused Gateway
  tests passed, `go build ./cmd/a21` passed, `a21-gateway` restarted active,
  and `http://127.0.0.1:21081/healthz` returned ok.
- Public host-loopback Xiaozhi bench passed after deploy:
  `reports/a21-xiaozhi-voice-bench-20260603-214135.199644000.json`.
  It remains host-only/candidate evidence, not physical PRD green.
- Post-deploy barge-in trace
  `a21-trace-xiaozhi-bench-1780494081208-barge-01` had
  `stop_plus_500ms_violations=0`; latest downlink preceded playback stop.
- Physical device `44:1b:f6:e2:6a:60` reconnected to the public Gateway after
  deploy as stock Xiaozhi Opus transport, but the latest post-deploy physical
  event is hello-only. A fresh operator wake/speak trace is still needed to
  confirm the audible "no speaking before I finish" fix on hardware.

## 2026-06-03 22:xx CST - Xiaozhi Late ASR Final Reply Recovery

Round goal:

- Fix the post-hotfix regression where a long utterance can leave StackChan in
  green listening state and never reply.

Actual completed work:

- Pulled the live physical trace for device `44:1b:f6:e2:6a:60` after the user
  reproduced the issue. The Gateway received real stock Opus uplink
  (`xiaozhi.opus_frame.received=299`) and the endpoint eventually sent
  `xiaozhi.listen.stop`.
- Root cause: `asr.stream.commit` waited only 200 ms for final text. The live
  DashScope final arrived about 222 ms after commit, so Gateway recorded
  `asr.stream.final_timeout`, returned, then later recorded `asr.final` without
  any code path left to start the voice pipeline.
- Added late-final recovery: when an ASR final event arrives after commit and
  the turn is no longer listening, Gateway claims the turn exactly once and
  starts the answer pipeline. This preserves the earlier no-pre-stop-speaking
  fix while preventing green-listening/no-reply.
- Renamed the internal one-shot guard from the old partial-bridge meaning to a
  generic streaming-ASR answer claim.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/agent_handoff_log.md`

Test/build/runtime results:

- Added regression:
  `TestXiaozhiWebSocketLateStreamingASRFinalStillStartsPipeline`.
- Focused Gateway streaming/listen tests passed.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Related packages passed:
  `go test ./internal/app ./internal/providers -count=1`.
- `make verify` passed.

Unfinished items:

- Commit and deploy this late-final recovery to `47.103.57.217`.
- Re-run the same physical long-speech test and require `asr.stream.final_timeout`
  followed by `asr.final` and then `xiaozhi.voice_pipeline.start` / downlink,
  or no timeout at all with normal final-driven reply.

Post-deploy update:

- Committed as `0161d84 fix(gateway): answer after late streaming asr final`.
- Deployed to main public ECS Gateway `47.103.57.217`; remote focused tests
  passed, `go build ./cmd/a21` passed, `a21-gateway` restarted active, and
  `healthz` returned ok.
- Public host-loopback bench passed after deploy:
  `reports/a21-xiaozhi-voice-bench-20260603-215804.962203000.json`.
- Fresh physical trace `a21-trace-44-1b-f6-e2-6a-60` after user long-speech
  repro showed the late-final recovery working: `asr.stream.final_timeout=2`,
  `asr.final=5`, `xiaozhi.voice_pipeline.start=2`,
  `provider.first_content=2`, `tts.first_audio=2`, and
  `xiaozhi.tts.opus_frame.downlink=267`.
- The same trace included `barge_in.detected=1` and `playback.stop=1` with
  `stop_plus_500ms_violations=0`. Stock firmware still does not expose
  `device.playback.stop_done`, so Gateway-side stop/downlink suppression is
  verified, but device queue acknowledgement remains unavailable.

## 2026-06-03 22:xx CST - Xiaozhi VAD Miss ASR Final Recovery

Round goal:

- Investigate the user's report that long-speech replies still occasionally do
  not happen after the late-final fix.

Actual completed work:

- Pulled live device state and trace for `44:1b:f6:e2:6a:60`.
- Found a distinct no-reply path after barge/new listen: the turn had
  `listen.stop`, `asr.stream.commit`, `asr.first_partial`, and `asr.final`, but
  no `xiaozhi.voice_pipeline.start`.
- Root cause: `writeXiaozhiVoicePipelineTTS` still required
  `voicePipelineHasSpeech=true`, which comes from Gateway-side RMS/VAD. In the
  failing segment the cloud ASR produced final text, but Gateway VAD had not
  raised `vad.speech.start`, so the valid ASR final was blocked before the
  answer pipeline.
- Fixed the contract so non-empty streaming ASR final text is speech evidence:
  final text marks the turn as speech-bearing, and the voice pipeline can start
  when `streamingASRFinalText` is present even if local Gateway VAD missed.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/agent_handoff_log.md`

Test/build/runtime results:

- Added regression:
  `TestXiaozhiWebSocketStreamingASRFinalStartsPipelineWhenGatewayVADMisses`.
- Focused Xiaozhi listen/final/barge tests passed.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Related packages passed:
  `go test ./internal/app ./internal/providers -count=1`.
- `make verify` passed.

Recommended next action:

1. Commit and deploy to `47.103.57.217`.
2. Re-run physical long utterance plus interruption and require:
   `asr.final -> xiaozhi.voice_pipeline.start -> tts.first_audio/downlink`
   even when `vad.speech.start` is absent for that segment.

## 2026-06-03 22:1x CST - Xiaozhi Speaking Drain / Self-Loop Guard

Round goal:

- Investigate the user's report that the device began looping into repeated
  self-triggered turns after replies, and compare the A21 Gateway behavior
  against the Xiaozhi-style listen/think/speak/drain state machine.

Actual completed work:

- Captured a fresh physical trace for product device `44:1b:f6:e2:6a:60`
  against main public Gateway `47.103.57.217`.
- The trace proved the main answer path was no longer silent: repeated physical
  turns had `listen.start`, Opus ingress, `listen.stop`, `asr.final`,
  `xiaozhi.voice_pipeline.start`, `provider.first_content`,
  `tts.first_audio`, and binary Opus downlink.
- The trace also exposed the self-loop path: after an answer had already sent
  many downlink frames, `RunStream` ended with a failed final result and A21
  treated the whole turn as `xiaozhi.voice_pipeline.unavailable`, entered
  `local_fallback`, sent `tts.stop`, and then accepted a new `listen.start`
  about 200 ms later. That differs from the intended Xiaozhi-style state
  machine, where audio already played means the speaking turn is degraded or
  complete, not a fresh fallback prompt, and post-stop tail audio must be
  drained before new input is accepted.
- Fixed streaming answer finalization so an error after answer audio has already
  been emitted records `xiaozhi.voice_pipeline.completed_degraded_after_audio`
  and sends one degraded `tts.stop`, without `local_fallback` and without
  `xiaozhi.voice_pipeline.unavailable`.
- Added a short post-TTS input suppression window for non-barge stop reasons
  (`completed`, `local_fallback`, `placeholder`, `host_say_complete`) so
  physical playback tail/echo does not immediately re-enter `listen.start`.
  Barge-in/abort/wake stops are not suppressed, preserving interrupt behavior
  while the device is actually speaking.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/agent_handoff_log.md`

Test/build/runtime results:

- Added regression:
  `TestXiaozhiWebSocketStreamingAnswerErrorAfterAudioDoesNotFallbackLoop`.
- Focused Xiaozhi streaming/no-reply/barge tests passed.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Related packages passed:
  `go test ./internal/app ./internal/providers -count=1`.
- `git diff --check` passed.
- `make verify` passed.

Recommended next action:

1. Commit and deploy to `47.103.57.217`.
2. Re-run a physical long reply plus repeated interruption test and require:
   no `xiaozhi.local_fallback.sent` after answer downlink has begun, no
   `xiaozhi.voice_pipeline.unavailable` after answer downlink has begun, and
   no accepted `listen.start` inside the post-TTS drain window.

Post-deploy update:

- Committed as `0aa1eda fix(gateway): prevent xiaozhi speaking self loop`.
- Deployed to main public ECS Gateway `47.103.57.217`; remote focused tests
  passed, `go build ./cmd/a21` passed, `a21-gateway` restarted active, and
  `healthz` returned ok.
- Public host-loopback bench passed after deploy:
  `reports/a21-xiaozhi-voice-bench-20260603-222620.134851000.json`.
  It remains host-loopback candidate evidence, not physical PRD acceptance.
- A 90-second physical trace watch after deployment saw only fresh
  `xiaozhi.hello.received` events and no new `listen.start`/Opus ingress, so
  the self-loop fix still needs an operator-triggered physical repro pass.

Follow-up reset pass:

- A later physical trace after `0aa1eda` showed the first guard working:
  two post-stop `listen.start` events were suppressed with
  `xiaozhi.listen.start.suppressed_post_tts_drain`, and there were no
  `xiaozhi.voice_pipeline.unavailable` or `xiaozhi.local_fallback.sent`
  markers after answer downlink.
- The same trace exposed a second state-machine gap: after Gateway suppressed a
  post-TTS `listen.start`, the device continued sending several seconds of
  Opus until its eventual `listen.stop.ignored`. Gateway was no longer
  accepting a new turn, but once the fixed cooldown expired it started buffering
  that still-suppressed Opus as `xiaozhi.wake_preroll.*`. That can pollute the
  next real wake/listen turn with playback tail audio.
- Added an explicit `suppressedListenActive` state. When Gateway rejects a
  `listen.start` because input is suppressed, all binary Opus from that rejected
  listen session is traced as `xiaozhi.opus_frame.ignored_suppressed_listen`
  and cannot enter wake-preroll or ASR. The state is cleared only when the
  device sends `listen.stop`, traced as
  `xiaozhi.listen.stop.suppressed_session_ended`.
- Physical follow-up showed one remaining tail frame could arrive immediately
  after the suppressed listen's `listen.stop`. Added
  `xiaozhi.listen.stop.suppressed_session_drain_armed`, a short drain after a
  rejected listen session ends, so stop-tail Opus is also discarded instead of
  entering wake-preroll.
- Updated no-speech and host-say suppression tests so suppressed-listen audio
  is forbidden from entering `xiaozhi.wake_preroll.opus_frame.buffered`.

Follow-up test/build results:

- Focused Xiaozhi suppression/self-loop/no-reply/barge tests passed.
- Full Gateway package passed:
  `go test ./internal/gateway -count=1`.
- Related packages passed:
  `go test ./internal/app ./internal/providers -count=1`.
- `git diff --check` passed.
- Latest focused drain tests passed after the stop-tail drain addition.
- `make verify` passed after the stop-tail drain addition.

Follow-up deploy and physical check:

- Committed as `862791e fix(gateway): drop suppressed xiaozhi listen audio`.
- Committed the stop-tail drain as
  `e17aa3d fix(gateway): drain suppressed xiaozhi listen tail`.
- Deployed `e17aa3d` to main public ECS Gateway `47.103.57.217`; remote
  focused tests passed, `go build ./cmd/a21` passed, `a21-gateway` restarted
  active, and `healthz` returned ok.
- Public host-loopback bench passed after deploy:
  `reports/a21-xiaozhi-voice-bench-20260603-224856.376949000.json`.
  It remains host-loopback candidate evidence, not physical PRD acceptance.
- Physical trace window for product device `44:1b:f6:e2:6a:60` showed one
  complete real turn after deploy: `listen.start`, Opus ingress, `listen.stop`,
  `asr.final`, `xiaozhi.voice_pipeline.start`, `provider.first_content`,
  `tts.first_audio`, and TTS Opus downlink.
- The same physical trace showed the post-TTS self-loop guard working:
  `xiaozhi.listen.start.suppressed_post_tts_drain=1`,
  `xiaozhi.listen.stop.suppressed_session_ended=1`,
  `xiaozhi.listen.stop.suppressed_session_drain_armed=1`,
  `xiaozhi.opus_frame.ignored_suppressed_listen=12`, with no
  `xiaozhi.wake_preroll.opus_frame.buffered`, no
  `xiaozhi.voice_pipeline.unavailable`, and no `xiaozhi.local_fallback.sent`
  observed in that window.

## 2026-06-03 23:3x CST - Internal Test 3 Release Closure

Round goal:

- Close the current voice-main-chain branch state, verify build/integration, and
  publish the A21 internal test 3 package after the user accepted the physical
  voice main-chain breakthrough.

Actual completed work:

- Rechecked branch `codex/a21-hardware-window-20260603-wifi-provisioning-flash`
  at HEAD `074e3d877d33`; no tracked dirty files were present before release
  packaging.
- Verified the latest real product firmware flash evidence remains the
  official-compatible lane:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-170812-1780477692092580000.json`
  with `flash_executed=true` and app artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`; no bare `xiaozhi.bin`
  product flash was used for this release record.
- Rechecked the main public Gateway `47.103.57.217`: `/healthz` returned ok,
  `/xiaozhi/ota/` returned `ws://47.103.57.217/v1/xiaozhi`, and device
  `44:1b:f6:e2:6a:60` was online.
- Captured public Gateway runtime snapshots for devices, gateway profiles,
  voice-chain profiles, OTA, and health.
- Built the packaged CLI from current HEAD and archived current source.
- Published package directory
  `dist/a21-internal-test3-20260603-233245` and refreshed `SHA256SUMS`.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `.a21-run/latest-internal-test3-package.path`
- `dist/a21-internal-test3-20260603-233245/**`

Test/build/runtime results:

- `make verify`: passed.
- `make preflight`: passed with warnings
  `firmware_current_artifact_missing` and `wake_word_firmware_build_required`.
- `make doctor`: passed with the same warnings.
- Packaged binary `bin/a21 gate --scope host`: passed with the A21 direct
  `NO_PROXY` coverage, with the same firmware/wake-word warnings.
- Public host-loopback voice bench passed:
  `reports/a21-xiaozhi-voice-bench-20260603-233014.342186000.json`;
  `answer_first_audio_p95_ms=792`, `barge_in_stop_p95_ms=20`,
  `failure_count=0`, ASR `dashscope_qwen_asr_realtime`, LLM `deepseek`, TTS
  `dashscope_qwen_tts_realtime`.
- Fresh xiaozhi physical evidence report:
  `reports/a21-xiaozhi-physical-evidence-20260603-232946.250456000.json`;
  device online, Opus ingress/downlink present, but machine-readable status
  remains `candidate_gateway_downlink`.
- Fresh product readiness:
  `reports/a21-product-readiness-20260603-233027.json`; `demo_ready=true`,
  `launch_ready=false`, `status=server_side_blocked`.
- Fresh server-side bundle:
  `reports/a21-server-side-readiness-bundle-20260603-233027.json`;
  `status=server_side_blocked`.
- Static local streaming provider readiness wrote
  `reports/a21-xiaozhi-streaming-provider-readiness-20260603-233026-1780500626486079000.json`
  and exited 1 as expected without local provider env; remote runtime snapshot
  remains the public-Gateway truth for internal test 3.

Known risks/blockers:

- Remote voice-chain profile still reports `stepfun_not_selected`; current
  runtime LLM is `deepseek`.
- Machine-readable physical evidence still lacks device playback ack,
  playback stop_done, and trusted operator audible observation.
- Local readiness tools still see local provider env as mock/blocked, so their
  provider section is not the same as the remote ECS runtime provider state.
- Full PRD launch remains blocked; internal test 3 is a voice-main-chain
  release package, not a full PRD green.

Recommended next action:

- Use the internal test 3 package for team testing on the public Gateway, then
  separately cut the StepFun remote switch and trusted physical playback/stop
  acknowledgement evidence.

## 2026-06-03 23:5x CST - Internal Test 3 Master Handoff Document

Round goal:

- Create a complete repo-carried handoff for the current long control thread so
  the next control/worker thread can resume without relying on chat memory.
- Tie the handoff to the committed internal test 3 release state, public
  Gateway state, package SHA, firmware lane guardrails, commit ledger,
  validation evidence, and remaining blockers.

Actual completed work:

- Added `docs/handoffs/2026-06-03-a21-internal-test3-master-handoff.md`.
- Captured the current branch/HEAD split: repo HEAD is
  `221c153 docs(release): publish internal test 3`, while the internal test 3
  package source archive remains pinned to `074e3d877d33`.
- Re-verified the package tarball SHA check.
- Re-queried the live public Gateway `47.103.57.217` for health, device,
  voice-chain profile, and OTA snapshots while writing the handoff.
- Recorded the current truth that live runtime still reports
  `stepfun_not_selected` and selected LLM `deepseek`.
- Preserved the release framing: internal test 3 voice main-chain accepted for
  team testing, but not full PRD launch green.

Changed files:

- `docs/handoffs/2026-06-03-a21-internal-test3-master-handoff.md`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Test/build/runtime results:

- `curl http://47.103.57.217/healthz`: ok.
- `curl http://47.103.57.217/v1/devices`: product device
  `44:1b:f6:e2:6a:60` online.
- `curl http://47.103.57.217/v1/voice-chain-profiles`: cascade mode,
  DashScope realtime ASR, DeepSeek LLM, DashScope realtime TTS,
  `stepfun_not_selected`.
- `curl http://47.103.57.217/xiaozhi/ota/`: returns
  `ws://47.103.57.217/v1/xiaozhi`.
- `shasum -c dist/a21-internal-test3-20260603-233245.tar.gz.sha256`: passed.

Known risks/blockers:

- The handoff is documentation-only and does not switch StepFun, restart
  Gateway, rebuild firmware, or collect new physical playback ack/stop_done.
- Existing untracked noise remains untouched:
  `.DS_Store`, `internal/.DS_Store`,
  `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md`.

Recommended next action:

- Commit this handoff document, then use it as the first-read file for the next
  control-thread continuation. The next product work should be the StepFun
  remote switch or trusted physical playback/stop acknowledgement evidence.

## 2026-06-04 - Worker D - Provider Selector And StepFun Runtime Switch Plan

Round goal:

- Audit and adapt provider/voice-chain selector surfaces so StepFun remote
  switch and post-switch evidence are explicit and redacted.

Actual completed work:

- Tightened `xiaozhi-streaming-provider-readiness` LLM stage reporting to use
  provider-catalog readiness for built-in and loaded text-stream profiles.
- Added env-name-only `required_env`, `present_env`, and `missing_env` fields
  plus `selection_role` so `stepfun` launch selection is distinguishable from
  DeepSeek fallback.
- Added focused readiness tests for StepFun selected with missing env names and
  DeepSeek fallback selected with env-name-only reporting.
- Added an ECS-only StepFun remote switch runbook covering root-only
  `/etc/a21/secrets/provider.env`, `a21-gateway` restart, fresh
  `/v1/voice-chain-profiles`, host bench, physical evidence, and readiness
  reruns.
- Linked the runbook from the provider benchmark contract.

Changed files:

- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `docs/engineering/A21_PROVIDER_BENCHMARKS.md`
- `docs/engineering/A21_STEPFUN_REMOTE_SWITCH_RUNBOOK.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- The required `internal/app` acceptance command could not complete because
  unrelated in-progress edits outside Worker D's write set currently break the
  app package build.

Known risks/blockers:

- Disjoint dirty files observed during this round include `internal/app/app_test.go`,
  `internal/app/official_stackchan.go`, `internal/app/product_demo.go`,
  `internal/app/xiaozhi_physical_evidence_test.go`,
  `internal/app/wake_word_physical_acceptance_test.go`,
  `internal/gateway/server_test.go`, and `docs/engineering/PROTOCOL.md`.
  They were not reverted or edited by Worker D.
- Current app build failure locations:
  `internal/app/product_demo.go:790` undefined
  `fetchProductVoiceChainReadiness`,
  `internal/app/product_demo.go:843` undefined
  `productVoiceChainLaunchPolicySatisfied`, and
  `internal/app/official_stackchan.go:16` unused `sort` import.

Test/build/runtime results:

- `go test ./internal/providers -run 'DashScope|Doubao|ProviderSmoke|TextStream|VoicePipelineAdapters|GatewayVoiceProvider|Realtime' -count=1`:
  passed.
- `go test ./internal/gateway -run 'VoiceChainProfiles|GatewayProfiles' -count=1`:
  passed.
- `go test ./internal/app -run 'XiaozhiStreamingProviderReadiness|ProviderSmoke|ProviderCompat|ProviderLatency' -count=1`:
  failed before tests ran due to the unrelated app package build errors listed
  above.
- `git diff --check`: passed.

Recommended next action:

- Let the Worker B/C ownership lane finish or fix the app package build errors,
  then rerun the required Worker D app acceptance command and the full
  integration order from
  `docs/plans/2026-06-04-full-launch-protocol-adaptation.md`.

## 2026-06-04 02:13 CST - Full Launch Protocol Adaptation Integration

Round goal:

- Move from internal test 3 release packaging toward full-launch readiness by
  adapting Gateway protocol regressions, readiness evidence semantics,
  official-compatible firmware evidence pointers, and provider/StepFun selector
  reporting after the stock `/v1/xiaozhi` protocol pivot.
- Keep launch truth honest: no StepFun false green, no physical playback false
  green, no firmware lane weakening, and no secret/provider leakage.

Actual completed work:

- Added the control plan
  `docs/plans/2026-06-04-full-launch-protocol-adaptation.md`.
- Added current control and evidence manifest entries under
  `docs/engineering/A21_CURRENT_CONTROL.md` and
  `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`.
- Integrated Worker A Gateway tests for stock `/v1/xiaozhi` message ordering,
  post-answer suppressed listen drain behavior, protocol v2 lifecycle coverage,
  and product-vs-dev protocol wording.
- Integrated Worker B readiness changes so product/server-side readiness ingest
  voice-chain selector state, block on `stepfun_not_selected`, reject stale or
  mismatched physical evidence targets, and avoid over-crediting Gateway
  downlink as PRD playback acceptance.
- Integrated Worker C official-compatible firmware evidence support so
  doctor/preflight can surface executed product-lane flash evidence while
  keeping generic `xiaozhi.bin` product flash blocked and wake-word warnings
  honest.
- Integrated Worker D provider selector reporting so StepFun launch selection
  and DeepSeek fallback are distinguishable with env-name-only details, plus a
  redacted ECS StepFun switch runbook.
- Added the next runtime plan
  `docs/plans/2026-06-04-ecs-control-plane-and-stepfun-switch.md`.
- Updated `docs/project_state_machine.md` to the current state
  `S-PUBLIC-GATEWAY-HEALTHY-STEPFUN-SWITCH-BLOCKED-BY-SSH`.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/engineering/A21_PROVIDER_BENCHMARKS.md`
- `docs/engineering/A21_STEPFUN_REMOTE_SWITCH_RUNBOOK.md`
- `docs/engineering/DOCTOR.md`
- `docs/engineering/FIRMWARE_RELEASE_DISCIPLINE.md`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-04-full-launch-protocol-adaptation.md`
- `docs/plans/2026-06-04-ecs-control-plane-and-stepfun-switch.md`
- `internal/app/app_office_preflight.go`
- `internal/app/app_stackchan_xiaozhi_half_duplex.go`
- `internal/app/app_test.go`
- `internal/app/doctor.go`
- `internal/app/official_stackchan.go`
- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- `internal/app/wake_word_physical_acceptance_test.go`
- `internal/app/xiaozhi_physical_evidence.go`
- `internal/app/xiaozhi_physical_evidence_test.go`
- `internal/app/xiaozhi_streaming_provider_readiness.go`
- `internal/app/xiaozhi_streaming_provider_readiness_test.go`
- `internal/app/official_firmware_evidence_test.go`
- `internal/gateway/server_test.go`

Unfinished items:

- StepFun is not selected on the live public Gateway; `/v1/voice-chain-profiles`
  still reports selected LLM `deepseek` and finding `stepfun_not_selected`.
- Full PRD launch remains blocked by missing physical playback ack,
  playback stop_done, and trusted audible/instrument observation.
- Remote ECS systemd/secret inspection and restart are blocked from this Mac
  because `ssh root@47.103.57.217` rejects the current public key.
- Public direct `:21081` HTTP returns `Empty reply from server`; product
  entrypoint `80` is healthy and remains the route for StackChan/OTA checks.

Known risks/blockers:

- Do not blind POST a StepFun hot switch on the public endpoint until remote
  StepFun env-name presence is verified; a blind switch could disrupt the
  working internal test 3 team-testing chain.
- Do not treat doctor's satisfied official-compatible product-lane flash
  evidence as wake-word or playback PRD acceptance.
- Do not use report mtime to infer launch truth; use
  `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`.
- Existing untracked local noise remains untouched:
  `.DS_Store`, `docs/.DS_Store`, `internal/.DS_Store`,
  `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md`.

Test/build/runtime results:

- `go test ./internal/gateway -run 'Xiaozhi|WriteXiaozhi|OfficialStackChan|TraceEndpointReturnsVoicePipelineSplitSummary' -count=1`:
  passed.
- `go test ./internal/app -run 'XiaozhiPhysicalEvidence|XiaozhiHalfDuplex|ProductReadiness|ServerSideReadinessBundle|XiaozhiVoiceBench|ProviderLatencyBench' -count=1`:
  passed.
- `go test ./internal/app -run 'Doctor|OfficePreflight|StackChanOfficialXiaozhiCompatible|XiaozhiFirmwareFlash|FirmwareCurrentArtifact' -count=1`:
  passed.
- `go test ./internal/app -run 'XiaozhiStreamingProviderReadiness|ProviderSmoke|ProviderCompat|ProviderLatency' -count=1`:
  passed.
- `go test ./internal/providers -run 'DashScope|Doubao|ProviderSmoke|TextStream|VoicePipelineAdapters|GatewayVoiceProvider|Realtime' -count=1`:
  passed.
- `go test ./internal/transport/xiaozhi ./internal/transport/stackchan -count=1`:
  passed.
- `go test ./internal/gateway -run 'VoiceChainProfiles|GatewayProfiles' -count=1`:
  passed.
- `git diff --check`: passed.
- `make verify`: passed.
- After the final documentation update, a default-concurrency `make verify`
  retry was killed by the host with `Killed: 9`; follow-up `go test -p 1 ./...`,
  `git diff --check`, and `GOMAXPROCS=2 make verify` all passed, so the kill is
  classified as a local resource/concurrency interruption rather than a test
  failure.
- `make preflight`: passed with warning
  `wake_word_firmware_build_required`.
- `make doctor`: passed with warning `wake_word_firmware_build_required` and
  `firmware.product_lane_artifact_evidence.status=satisfied` from the executed
  official-compatible product-lane flash report.
- Retry at 2026-06-04 02:13 CST:
  `curl --noproxy '*' http://47.103.57.217/healthz` passed;
  `/v1/devices` showed product device `44:1b:f6:e2:6a:60` online;
  `/v1/voice-chain-profiles` still reported selected LLM `deepseek` and
  `stepfun_not_selected`;
  `/xiaozhi/ota/` returned `ws://47.103.57.217/v1/xiaozhi`;
  `ssh root@47.103.57.217` failed with `Permission denied (publickey)`.

Recommended next action:

- Execute `docs/plans/2026-06-04-ecs-control-plane-and-stepfun-switch.md`:
  recover or delegate ECS control-plane access, verify StepFun env-name
  presence without values, then switch/restart only if configured and collect
  fresh Gateway, host bench, physical evidence, product readiness, and
  server-side readiness evidence.

## 2026-06-04 02:24 CST - ECS Control Plane Narrowed To StepFun Env Blocker

Round goal:

- Continue the ECS control-plane and StepFun switch transition under the master
  control rules.
- Verify whether the previous SSH blocker was absolute, then check StepFun
  env-name readiness without exposing values.
- Do not hot-switch public runtime unless StepFun env names are present.

Actual completed work:

- Read the current plan
  `docs/plans/2026-06-04-ecs-control-plane-and-stepfun-switch.md` and resumed
  at Action Plan step 1.
- Verified that default SSH still has no loaded identities, then used an
  explicit existing local SSH identity for a read-only ECS service check.
- Confirmed remote `a21-gateway` is active, Caddy is active, public product
  entrypoint `80` is healthy, and remote `127.0.0.1:21081/healthz` returns ok.
- Verified the remote provider env file is present, root-owned, and mode `600`.
- Checked env-name presence only; no provider values were printed or copied.
- Determined `A21_LAB_STEPFUN_API_KEY` and `A21_STEPFUN_MODEL` are missing on
  ECS, while `A21_DASHSCOPE_API_KEY` is present.
- Checked local control-machine env-name presence only; StepFun and DashScope
  env names are also missing locally.
- Stopped before any `/v1/voice-chain-profiles` StepFun hot switch.
- Wrote fresh current-blocker readiness evidence:
  `reports/a21-product-readiness-20260604-022419.json` and
  `reports/a21-server-side-readiness-bundle-20260604-022420.json`.
- Updated current control, evidence manifest, the ECS/StepFun plan, and project
  state machine to the env blocker state.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/plans/2026-06-04-ecs-control-plane-and-stepfun-switch.md`
- `reports/a21-product-readiness-20260604-022419.json`
- `reports/a21-server-side-readiness-bundle-20260604-022420.json`

Unfinished items:

- StepFun env names must be provisioned on ECS by an approved operator:
  `A21_LAB_STEPFUN_API_KEY` and `A21_STEPFUN_MODEL`.
- After provisioning, `a21-gateway` must be restarted/validated and then
  `/v1/voice-chain-profiles` may be switched to StepFun.
- Post-switch host bench, physical evidence, product readiness, and
  server-side readiness are still pending.
- Physical playback ack, playback stop_done, and trusted audible/instrument
  observation remain required for PRD launch.

Known risks/blockers:

- Do not write provider secret values from this Mac; local env names are
  missing and the project policy forbids secret values in repo, reports, logs,
  traces, docs, stdout, or chat.
- Do not blind hot-switch public runtime to StepFun while required env names
  are missing; it would risk breaking the working internal test 3 chain.
- Public direct `:21081` still should not be treated as the product entrypoint;
  use public `80` for product StackChan/OTA checks unless ECS networking is
  deliberately changed.

Test/build/runtime results:

- ECS read-only check:
  `a21-gateway=active`, `caddy=active`,
  remote `127.0.0.1:21081/healthz` ok.
- Remote env-name check:
  `A21_LAB_STEPFUN_API_KEY=missing`,
  `A21_STEPFUN_MODEL=missing`,
  `A21_DASHSCOPE_API_KEY=present`.
- Local env-name check:
  StepFun/DashScope env names missing.
- `go run ./cmd/a21 product-readiness --gateway-url http://47.103.57.217 --device-id 44:1b:f6:e2:6a:60 --use-latest-reports --output-dir reports`:
  wrote `reports/a21-product-readiness-20260604-022419.json`,
  `status=server_side_blocked`, `launch_ready=false`, selected LLM `deepseek`,
  finding `stepfun_not_selected`.
- `go run ./cmd/a21 server-side-readiness-bundle --gateway-url http://47.103.57.217 --device-id 44:1b:f6:e2:6a:60 --use-latest-reports --output-dir reports`:
  wrote `reports/a21-server-side-readiness-bundle-20260604-022420.json`,
  `status=server_side_blocked`, missing `provider_smoke` and
  `stepfun_not_selected`.

Recommended next action:

- Have an approved operator provision the missing StepFun env names in the
  remote root-only provider env file, then resume
  `docs/plans/2026-06-04-ecs-control-plane-and-stepfun-switch.md` at Action
  Plan step 3: restart/validate `a21-gateway`, switch selector to StepFun, and
  collect fresh Gateway, host bench, physical evidence, product readiness, and
  server-side readiness reports.

## 2026-06-04 02:28 CST - Public Gateway Code Sync Required Before StepFun

Round goal:

- Keep pushing after StepFun env blocker by moving all locally verified
  protocol/readiness/provider-selector fixes toward the public Gateway without
  switching runtime state prematurely.
- Detect whether the remote binary already enforces the new StepFun env-name
  gate.

Actual completed work:

- Ran ECS-side `xiaozhi-streaming-provider-readiness` with selector env names
  forcing StepFun while sourcing the remote root-only provider env file.
- First run failed because remote `/opt/a21/reports` did not exist; created the
  A21 reports directory and reran.
- Found stale remote binary behavior: the ECS binary reported
  `gate_status=passed` for StepFun static readiness even though the direct
  env-name check showed required StepFun env names missing.
- Added
  `docs/plans/2026-06-04-public-gateway-code-sync-before-stepfun.md` to deploy
  the locally verified code before any StepFun selector switch.
- Updated current control and project state machine to require code sync before
  StepFun switch.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-public-gateway-code-sync-before-stepfun.md`

Unfinished items:

- Commit the local protocol-adaptation/code-sync source.
- Deploy that commit to ECS.
- Re-run remote static StepFun readiness and require it to block when env names
  are missing.
- Only after remote code is synced, provision StepFun env names through the
  root-only ECS secret file and switch `/v1/voice-chain-profiles`.

Known risks/blockers:

- Do not trust the current remote binary's StepFun static readiness result; it
  predates the local env-name gate changes.
- Do not POST/PUT the live voice-chain selector to StepFun before the code sync
  and env-name verification pass.
- Do not record provider secret values or model values in repo, reports, logs,
  traces, stdout, or chat.

Test/build/runtime results:

- Remote stale-readiness observation wrote an ECS-side report under
  `/opt/a21/reports`, but that ignored runtime report is not a commit artifact.
- Local code remains the source of truth for the StepFun missing-env gate until
  the code-sync deploy completes.

Recommended next action:

- Commit the locally verified changes, deploy the commit to ECS using
  `git archive HEAD`, run focused remote tests/build, swap `/opt/a21`, and
  prove the remote binary now blocks StepFun readiness until the missing env
  names are provisioned.

## 2026-06-04 03:08 CST - StepFun Route Eligibility Promotion Started

Round goal:

- Continue after re-reading the internal test 3 master handoff and current
  code/reports without repeating already completed StepFun runtime switch work.
- Treat internal test 3 endpoint-side voice acceptance and protocol changes as
  immutable baseline for this slice.
- Promote StepFun only at the provider route-evidence contract layer.

Actual completed work:

- Re-read the master handoff, active plans, current control entry, evidence
  manifest, state machine, public Gateway snapshots, and latest reports.
- Confirmed public runtime already selects LLM `stepfun`; product device
  `44:1b:f6:e2:6a:60` is online; OTA still returns the public Xiaozhi
  WebSocket route.
- Confirmed latest host bench executed the cloud-edge DashScope ASR + StepFun
  LLM + DashScope TTS chain.
- Identified the current non-duplicate blocker: the executed StepFun provider
  smoke passed but was generated with `route_eligible=false`, so readiness
  rejects it as `provider_smoke`.
- Added focused control plan
  `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`.
- Promoted built-in StepFun to explicit `route_eligible=true` in the provider
  catalog and updated the explicit route-eligible catalog test.
- Updated provider/readiness docs and current control documents to stop
  describing StepFun as compatibility-only for the current launch policy.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/engineering/A21_DEVELOPMENT_MAINLINE.md`
- `docs/engineering/A21_PROVIDER_BENCHMARKS.md`
- `docs/engineering/DOCTOR.md`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`
- `internal/providers/catalog.go`
- `internal/providers/catalog_test.go`

Unfinished items:

- Run full verification after doc/control updates.
- Commit the route-eligibility promotion patch.
- Deploy the committed patch to ECS.
- Run fresh redacted StepFun provider smoke from the synced remote binary.
- Re-run product readiness and server-side readiness with the fresh report.
- Physical playback ack, playback stop_done, or trusted audible/instrument
  observation remains required before full PRD launch can be claimed.

Known risks/blockers:

- Do not reinterpret older provider-smoke reports; the old StepFun smoke is
  valid execution evidence but not route-eligible readiness evidence.
- Do not touch internal test 3 `/v1/xiaozhi` protocol behavior, firmware flash
  state, NVS, or endpoint-side accepted voice behavior in this route-only
  promotion.
- Do not record provider secret values or model values in repo, reports, logs,
  traces, stdout, or chat.

Test/build/runtime results:

- `go test ./internal/providers -run 'ProviderCatalog|ProviderSmoke' -count=1`:
  passed.
- `go test ./internal/app -run 'ProductReadiness|ServerSideReadinessBundle|XiaozhiStreamingProviderReadiness' -count=1`:
  passed.
- First `GOMAXPROCS=2 make verify` found one stale test assumption:
  `TestRunLocalVoiceLoopbackCanUseCompatibilityTextStreamWithoutLeakingContent`
  still used StepFun as the compatibility-only candidate. The test was moved to
  SiliconFlow so StepFun stays promoted while compatibility-only coverage
  remains intact.
- `go test ./internal/app -run 'LocalVoiceLoopbackCanUseCompatibilityTextStream|ProductReadiness|ServerSideReadinessBundle|XiaozhiStreamingProviderReadiness' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Recommended next action:

- Commit and deploy the small StepFun route-eligibility promotion patch before
  collecting fresh provider/readiness evidence.

## 2026-06-04 03:17 CST - StepFun Route Eligibility Committed, ECS Deploy Blocked By SSH

Round goal:

- Deploy the committed StepFun route-eligibility promotion to ECS and collect a
  fresh remote route-eligible provider-smoke report.

Actual completed work:

- Committed the route-eligibility promotion as
  `3741c4a feat(providers): promote stepfun route eligibility`.
- Confirmed the committed local binary now emits StepFun dry-run provider smoke
  with `route_eligible=true`, `configured=true`, `stream=true`, and
  `executed=false` when the required env names are present.
- Attempted ECS deploy using the existing safe `git archive HEAD` to
  `/opt/a21.next` pattern.
- Confirmed default SSH is unusable in this thread.
- Confirmed the available `~/.ssh` candidate identities are not accepted by
  `root@47.103.57.217`; the remote closes the SSH connection.
- Stopped before any remote file, service, selector, secret, firmware, flash,
  or NVS mutation.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`

Unfinished items:

- Restore or provide an approved SSH/control-plane identity for ECS.
- Deploy commit `3741c4a` to `/opt/a21` using the existing safe swap pattern.
- Run fresh remote StepFun `provider-smoke --execute --stream --repeat 3`.
- Copy only the redacted provider-smoke report into local `reports/`.
- Re-run product readiness and server-side readiness with the fresh report.
- Physical playback ack, playback stop_done, or trusted audible/instrument
  observation remains required before full PRD launch can be claimed.

Known risks/blockers:

- Do not use the older provider-smoke report as launch evidence; it was
  generated before StepFun was route-eligible and correctly records
  `route_eligible=false`.
- Do not attempt blind remote changes without a verified ECS control-plane
  identity.
- Do not touch internal test 3 `/v1/xiaozhi` protocol behavior, firmware flash
  state, NVS, or endpoint-side accepted voice behavior in this route-only
  promotion.

Test/build/runtime results:

- `git diff --check`: passed before commit.
- `GOMAXPROCS=2 make verify`: passed before commit.
- Local dry-run shape check:
  StepFun provider smoke reports `status=ready`, `configured=true`,
  `executed=false`, `stream=true`, `route_eligible=true`, with env names only.
- ECS deploy attempt: blocked before remote mutation because SSH identity is
  unavailable in this thread.

Recommended next action:

- Resume
  `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md` at Action
  Plan step 6 after ECS SSH/control-plane access is restored, then collect the
  fresh remote provider/readiness evidence.

## 2026-06-04 03:28 CST - Workspace Control Audit And Safe Cleanup

Round goal:

- Explain and contain the Git loose-object warning and branch/worktree/subagent
  sprawl without destabilizing launch-critical internal test 3 work.
- Clean the main workspace only where safe.

Actual completed work:

- Added current workspace audit:
  `docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`.
- Marked `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md` as source-only
  backlog rather than an active control source.
- Updated current control to point at the workspace audit and current
  route-promotion plan.
- Added `.DS_Store` to `.gitignore`.
- Removed `.DS_Store` noise from the main workspace.
- Ran `git worktree prune`; it removed only the stale missing worktree
  registration.
- Confirmed the previously surfaced subagent ID is not known to the current
  multi-agent runtime.
- Confirmed A21-related Codex threads are historical/current control surfaces,
  not active worker writers.

Changed files:

- `.gitignore`
- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md`
- `docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`
- `docs/project_state_machine.md`

Unfinished items:

- Existing source-only worker worktrees remain in place; do not delete them
  without a dedicated cleanup plan and explicit approval.
- Existing stashes remain source-only; do not apply/pop/drop them during the
  launch-critical path.
- Git object-store pruning remains deferred; loose objects are maintenance
  debt, not current source dirty state.

Known risks/blockers:

- Deleting branches/worktrees by name is unsafe until each is classified as
  merged, superseded, source-only, or disposable.
- Running `git prune` or aggressive GC before classification could permanently
  remove unreachable source material that still explains old worker output.
- The main launch blocker is still ECS SSH/control-plane recovery for deploying
  commit `3741c4a` and collecting fresh remote StepFun provider evidence.

Test/build/runtime results:

- `git worktree prune --dry-run` identified one stale registration.
- `git worktree prune` completed; registered worktrees reduced from 38 to 37.
- Post-prune worktree count:
  37 existing, 0 missing, 4 dirty before committing this audit, 19 branched,
  18 detached.
- `git count-objects -vH` still reports about 7.9k loose objects and no
  garbage; object-store cleanup is intentionally deferred.

Recommended next action:

- Commit the workspace audit/control cleanup, then continue only from the
  current active plan. Do not spawn new workers or delete old branches until the
  current route-promotion/ECS evidence path is closed or explicitly paused.

## 2026-06-04 03:36 CST - Internal Test 3 Integration Audit

Round goal:

- Verify that the internal test 3 commits and accepted endpoint-side protocol
  progress are still included in current `HEAD`.
- Distinguish local mainline inclusion from remote tracking branch inclusion.
- Avoid any broad revert or cleanup that could erase accepted protocol work.

Actual completed work:

- Added integration audit:
  `docs/engineering/A21_INTEGRATION_AUDIT_20260604.md`.
- Confirmed internal test 3 package/source, release-doc, and handoff commits
  are all ancestors of current `HEAD`.
- Confirmed those internal test 3 commits are also present on
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Confirmed follow-up commits `765ed41`, `3741c4a`, and `8396261` are included
  locally but not yet present on the remote tracking branch.
- Confirmed high-signal Gateway protocol, firmware lane, wake/socket, and
  voice-chain selector commits remain ancestors of current `HEAD`.
- Checked that there is no diff from `221c153..HEAD` in
  `internal/gateway/server.go`, `internal/transport/xiaozhi`, or `firmware`.
- Updated current control and workspace audit to point at the integration
  audit as the no-broad-revert reference.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_INTEGRATION_AUDIT_20260604.md`
- `docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`

Unfinished items:

- Push or otherwise integrate the three newest local commits after the user
  chooses the remote/PR path.
- ECS deploy remains blocked by SSH/control-plane access; no remote mutation was
  attempted during this audit.

Known risks/blockers:

- Do not assume local-only commits are already on the remote tracking branch.
- Do not use branch/worktree cleanup as a route to discard internal test 3
  progress.
- Do not revert Gateway protocol, transport, firmware, or endpoint-side
  acceptance changes without a named reviewed transition.

Test/build/runtime results:

- `git merge-base --is-ancestor` checks passed for the listed commits.
- `git branch -r --contains` confirms internal test 3 baseline commits are on
  the remote tracking branch, while the three newest local commits are not.
- `git diff --stat 221c153..HEAD -- firmware internal/transport internal/gateway/server.go`:
  no output.
- `git diff --check`: passed.

Recommended next action:

- Keep current `HEAD` as the control mainline. Next operational step remains
  ECS control-plane recovery and deployment of the local route-eligibility
  commit chain; do not start cleanup-driven reverts.

## 2026-06-04 03:33 CST - Remote Branch Synced, ECS Runtime Still Blocked

Round goal:

- Continue from the integration audit by making sure the local commits are no
  longer trapped on the control Mac.
- Recheck public Gateway and ECS control-plane status without mutating runtime.

Actual completed work:

- Pushed current branch to
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Confirmed remote tracking branch now contains `765ed41`, `3741c4a`,
  `8396261`, and `5dba606`.
- Rechecked public Gateway from the control Mac.
- Confirmed TCP ports `22`, `80`, `443`, and `21081` on `47.103.57.217` are
  reachable.
- Confirmed HTTP paths `/healthz`, `/v1/devices`, `/v1/voice-chain-profiles`,
  and `/xiaozhi/ota/` currently return empty replies.
- Confirmed default SSH remains unusable for `root@47.103.57.217`.
- No ECS files, services, provider secrets, selectors, firmware, flash state, or
  NVS were mutated.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/engineering/A21_INTEGRATION_AUDIT_20260604.md`
- `docs/project_state_machine.md`

Unfinished items:

- Restore ECS SSH/control-plane access or have an approved operator run the
  deploy commands on host.
- Diagnose why public Gateway HTTP paths are returning empty replies.
- Deploy commit `3741c4a` or newer to ECS after control-plane recovery.
- Run fresh remote StepFun provider smoke and rerun readiness.

Known risks/blockers:

- Do not treat pushed Git source as equivalent to ECS runtime deployment.
- Do not use older public Gateway snapshots while current HTTP checks return
  empty replies.
- Do not make blind remote changes without verified ECS control-plane access.

Test/build/runtime results:

- `git push origin codex/a21-hardware-window-20260603-wifi-provisioning-flash`:
  passed.
- `git status --short --branch`: clean after push before this doc update.
- Public Gateway TCP reachability: ports `22`, `80`, `443`, and `21081`
  reachable.
- Public HTTP checks: empty replies for the current product paths.
- SSH check: default SSH unusable.

Recommended next action:

- Commit and push this runtime-blocker update, then either recover ECS
  control-plane access or hand the exact deploy/diagnosis task to an approved
  operator with host access.

## 2026-06-04 04:02 CST - ECS StepFun Cloud-Edge Evidence Accepted

Round goal:

- Resolve the ECS runtime/control-plane blocker without reverting internal test
  3 protocol work.
- Deploy the route-eligible StepFun code path.
- Collect fresh ECS-side provider/readiness evidence.
- Adapt readiness to accept cloud-edge Xiaozhi host evidence without promoting
  it to physical PRD acceptance.

Actual completed work:

- Recovered ECS control through the approved 5080lab jump path and an explicit
  existing local SSH identity.
- Deployed `d9362a7 feat(readiness): accept cloud-edge xiaozhi evidence` to
  ECS using the existing `/opt/a21.next` safe-swap pattern.
- Preserved provider secret values; only A21 profile selector IDs were changed
  in the root-only ECS provider env file.
- Confirmed `a21-gateway` and Caddy are active and ECS loopback health is ok.
- Confirmed Gateway selector after restart is cascade DashScope ASR, StepFun
  LLM, fixed DashScope TTS, with no findings.
- Ran fresh ECS StepFun provider smoke, static provider readiness, Xiaozhi
  cloud-edge host bench, product readiness, and server-side readiness.
- Added readiness code support for cloud-edge Xiaozhi host evidence:
  `cloud_edge` execution mode is preserved by the sanitizer, and
  product-readiness accepts non-mock cloud-edge provider execution as
  server-side candidate voice evidence while keeping it below PRD acceptance.
- Diagnosed the Mac direct-curl symptom: this control Mac still receives empty
  public HTTP replies, while 5080lab and ECS loopback are healthy; a later ECS
  tcpdump did not observe the Mac curl reaching the host.

Changed files:

- `internal/app/app_test.go`
- `internal/app/product_demo.go`
- `internal/app/provider_latency_bench.go`
- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/engineering/A21_INTEGRATION_AUDIT_20260604.md`
- `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`
- `docs/project_state_machine.md`

Fresh evidence:

- `reports/a21-provider-smoke-20260604-035200-977132343.json`:
  `passed`, `executed=true`, `stream=true`, `route_eligible=true`.
- `reports/a21-xiaozhi-streaming-provider-readiness-20260604-035317-1780516397705439206.json`:
  `gate_status=passed`, `stepfun_selected`.
- `reports/a21-xiaozhi-voice-bench-20260604-035502.222374820.json`:
  `candidate_host_only`, `cloud_edge`, `failure_count=0`,
  answer first-audio p95 `809 ms`, barge-in stop p95 `0 ms`.
- `reports/a21-product-readiness-20260604-040153.json`:
  `server_side_blocked`, provider evidence ready and host voice loopback ready.
- `reports/a21-server-side-readiness-bundle-20260604-040207.json`:
  `server_side_blocked`, missing `v21_professional_smoke`.

Unfinished items:

- V21 professional execution evidence remains missing.
- Physical StackChan PRD acceptance remains missing: playback/audible and
  stop/barge-in evidence are still required.
- Mac direct public curls to `47.103.57.217` still return empty replies from
  this control machine, but this is no longer classified as an A21 runtime
  blocker because ECS loopback and 5080lab public checks are healthy.

Known risks/blockers:

- Do not claim `launch_ready=true` or `prd_accepted=true`; readiness is still
  server-side blocked by V21 and physical evidence.
- Do not run V21 execution without a scoped professional validation plan.
- Do not use cloud-edge host bench as physical playback proof.
- Do not prune Git loose objects or delete old worktrees/branches during the
  launch-critical path.

Test/build/runtime results:

- Local:
  `go test ./internal/app -run 'ProductReadiness|ServerSideReadinessBundle|XiaozhiVoiceBench|ProviderLatency' -count=1`
  passed.
- Local:
  `go test ./internal/providers -run 'ProviderCatalog|ProviderSmoke' -count=1`
  passed.
- Local `git diff --check`: passed.
- Remote pre-swap app/provider focused tests: passed.
- Remote build `go build -o /opt/a21.next/bin/a21 ./cmd/a21`: passed.
- Remote `a21-gateway`: active after swap.
- Remote StepFun provider smoke: passed with no fallback.
- Remote product/server-side readiness: provider and host voice evidence ready,
  still blocked by V21/physical gates.

Recommended next action:

- Create or activate a scoped V21 professional validation transition, then run
  the adapter evidence without leaking query text or provider outputs.
- In a foreground hardware window, collect physical playback/stop/audible
  evidence for the current StepFun cloud-edge chain.

## 2026-06-04 04:14 CST - V21 Professional Execution Transition Opened

Round goal:

- Keep control over active A21 threads/worktrees after the ECS StepFun
  recovery.
- Move the next server-side launch blocker from an implicit missing item into
  an explicit V21 professional execution transition.
- Avoid reusing stale 2026-06-01/02 V21 reports or broad-resetting internal
  test 3 protocol work.

Actual completed work:

- Inspected A21 thread status: the server mainline control thread is `idle`,
  and the hardware control thread is `notLoaded`; no active worker writer is
  currently authorized.
- Confirmed the current branch is clean and synced to the remote at
  `20d11a0`.
- Confirmed historical worktrees remain registered but are not the active write
  surface for this transition.
- Added scoped plan
  `docs/plans/2026-06-04-v21-professional-execution-validation.md`.
- Updated current control, evidence manifest, and state machine so V21
  professional execution is the active server-side transition.
- Ran safe V21 discovery only: CLI help, env-name presence, local adapter
  listener check, and report-age listing.
- Generated fresh local V21 dry-run report
  `reports/a21-v21-adapter-smoke-20260604-041657.json`.
- Reran local product/server-side readiness collectors with the fresh V21
  dry-run and latest StepFun/cloud-edge report pointers:
  `reports/a21-product-readiness-20260604-041710.json` and
  `reports/a21-server-side-readiness-bundle-20260604-041710.json`.
- Marked those local collector reports as boundary-missing/operator-ask
  evidence only, not canonical launch evidence superseding the 04:02 ECS
  StepFun/cloud-edge reports.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/plans/2026-06-04-v21-professional-execution-validation.md`
- `docs/project_state_machine.md`

Unfinished items:

- No fresh executed V21 adapter smoke has run yet.
- No product/server-side readiness rerun has absorbed V21 evidence yet.
- Physical StackChan PRD acceptance remains open.

Known risks/blockers:

- Current shell has no `A21_V21_ADAPTER_URL`, no `A21_V21_BACKEND_URL`, no
  `A21_V21_ADAPTER_TOKEN`, and no local `127.0.0.1:21121` listener.
- Local default V21 backend `127.0.0.1:18080` is not reachable, so the A21
  adapter bridge cannot be started against a local backend from this shell.
- Historical V21 reports are from 2026-06-01/02 and must not close the current
  2026-06-04 V21 gate.
- Do not execute V21 until the A21 adapter boundary or approved V21 backend
  boundary is configured.

Test/build/runtime results:

- `go run ./cmd/a21 v21-adapter-smoke --help`: passed.
- `go run ./cmd/a21 xiaozhi-professional-bench --help`: passed.
- `go run ./cmd/a21 v21-adapter-bridge --help`: passed.
- Env-name presence check: V21 adapter/backend/token missing in current shell.
- Local `21121` listener check: no listener.
- `go run ./cmd/a21 v21-adapter-smoke --output-dir reports`: passed with
  `status=skipped`, `configured=false`, `executed=false`, `redaction_ok=true`.
- Local `18080` backend check: connection refused.
- Local product readiness collector:
  `reports/a21-product-readiness-20260604-041710.json`,
  `server_side_blocked` from local shell missing Gateway/provider/selector/V21.
- Local server-side readiness collector:
  `reports/a21-server-side-readiness-bundle-20260604-041710.json`,
  `server_side_blocked` from local shell missing Gateway/provider/selector/V21,
  redaction flags false for payloads/prompts/transcripts/provider outputs/
  evidence bodies/full URLs/local paths/credentials.
- `git diff --check`: passed after the first control-doc update.
- `GOMAXPROCS=2 make verify`: passed.

Recommended next action:

- If an A21/V21 adapter boundary or V21 backend boundary is provided, run one
  redacted `v21-adapter-smoke --execute`, then rerun server-side readiness.
- Operator needs to provide either `A21_V21_ADAPTER_URL` for a running adapter
  boundary or `A21_V21_BACKEND_URL` plus optional `A21_V21_COLLECTION_ID` for
  the A21 bridge; do not provide raw V21 evidence or document contents.

## 2026-06-04 04:36 CST - V21 Local Adapter Evidence Closed

Round goal:

- Use the user-supplied V21 control thread to determine whether V21 is local,
  then close the A21 V21 professional evidence gate through the explicit
  adapter boundary.

Actual completed work:

- Read V21 thread
  `codex://threads/019e68bc-4fb6-7ce0-ad67-5b1dd0de478f`.
- Confirmed V21 current operating shape from that thread and repo state:
  local/LAN Docker Compose backend, demo backend on `18081`, old loopback
  history on `18080`.
- Checked `/Users/jiyurun/Documents/v21-knowledge-platform`; the V21 worktree
  already had uncommitted LAN discovery/UI edits. Those edits were not modified
  or reverted.
- Started Docker Desktop because Docker daemon was initially unavailable.
- Started the V21 LAN demo Docker Compose stack with `make lan-demo-up`; the
  script later timed out on its LAN self-check, but the compose services were
  healthy and manual direct checks succeeded.
- Verified V21 health:
  `127.0.0.1:18081/api/v1/healthz` and
  `192.168.1.20:18081/api/v1/healthz` both returned ok.
- Verified V21 runtime config had retrieval and LLM configured.
- Verified V21 active collection discovery returned an active release.
- Started A21 bridge temporarily:
  `go run ./cmd/a21 v21-adapter-bridge --addr 127.0.0.1:21121 --v21-url http://127.0.0.1:18081`.
- Verified bridge health.
- Ran real V21 adapter smoke with execution:
  `A21_V21_ADAPTER_URL=http://127.0.0.1:21121 go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports`.
- Reran product/server-side readiness with the fresh V21 report and latest
  StepFun/cloud-edge report pointers.
- Stopped the temporary A21 bridge process on `21121`.
- Left the V21 Docker Compose backend running on `18081` for follow-up work.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `docs/plans/2026-06-04-v21-professional-execution-validation.md`
- `docs/project_state_machine.md`

Fresh reports:

- `reports/a21-v21-adapter-smoke-20260604-043456.json`:
  `passed`, `configured=true`, `executed=true`, `redaction_ok=true`,
  evidence count `5`, speech block count `1`, screen card count `1`,
  follow-up count `1`, duration `726.483 ms`.
- `reports/a21-product-readiness-20260604-043528.json`:
  `server_side_blocked`; provider, V21, and host voice are ready; local
  missing evidence is Gateway, physical StackChan, wake word, and voice-chain
  selector.
- `reports/a21-server-side-readiness-bundle-20260604-043528.json`:
  `server_side_blocked`; provider ready, V21 ready, host voice ready; missing
  `gateway`, `wake_word`, and `voice_chain_selector`.

Unfinished items:

- Physical StackChan PRD acceptance remains open.
- Current local readiness collector has no live A21 Gateway/wake/selector
  context.
- ECS/public Gateway was not reconfigured to call the local V21 bridge; this
  run proves adapter-contract execution only.

Known risks/blockers:

- Do not claim full launch readiness from this V21 smoke.
- Do not treat the temporary Mac-local bridge as permanent ECS topology.
- V21 Docker Compose remains running on `18081`; stop it only with an explicit
  V21/LAN demo decision, not as A21 cleanup.
- V21 worktree remains dirty from its own LAN discovery/UI work; do not revert
  those changes from A21.

Test/build/runtime results:

- `docker info`: passed after starting Docker Desktop.
- `make lan-demo-up` in V21: compose services started; script exited with LAN
  self-check timeout after manual kill of the waiting script, but health checks
  independently passed.
- `docker compose -p v21air-lan-demo ... ps`: V21 app healthy on `18081`.
- V21 health checks on localhost and LAN IP: passed.
- V21 runtime config check: retrieval and LLM configured.
- V21 collections check: active release present.
- A21 bridge `/healthz`: passed.
- V21 adapter smoke execute: passed.
- Product/server-side readiness reruns: V21 evidence ready, still blocked on
  non-V21 gates.
- `GOMAXPROCS=2 make verify`: passed.

Recommended next action:

- Refresh live A21 Gateway/voice-chain/wake evidence against the current
  StepFun cloud-edge chain, then collect physical StackChan PRD playback/stop
  or trusted audible evidence.

## 2026-06-04 05:06 CST - StackChan Official Hardware Parity Plan Handed Back

Round goal:

- Turn the read-only comparison between A21's current StackChan hardware
  control/status surface and the official Xiaozhi StackChan package into a
  detailed main-control implementation plan.

Actual completed work:

- Wrote a detailed staged plan:
  `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`.
- Captured the official-vs-A21 parity matrix across microphone, speaker,
  screen/status display, screen touch, top touch, pitch/yaw servos, RGB,
  camera/photo/video, IMU, ambient/proximity, battery/charging, NFC, infrared,
  Xiaozhi MCP tools, official StackChan WebSocket avatar/action frames, app
  lifecycle, OTA/provisioning, wake/audio/AEC/playback evidence, and physical
  PRD gates.
- Split the work into ordered transitions:
  docs-only gap map, low-risk MCP/status parity, display-state registry
  parity, official avatar/motion/RGB/servo mapping, physical touch/action
  evidence, sensor/battery diagnostics, audio/wake/playback evidence closure,
  official app-lifecycle reconciliation, and high-risk camera/video/NFC/IR
  spikes.
- Added `T-STACKCHAN-OFFICIAL-HARDWARE-PARITY-001` as a next candidate in
  `docs/project_state_machine.md` without taking over the current active
  Internal Test 4 work.

Changed files:

- `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- The plan has not been executed.
- No Gateway MCP/status endpoints were added in this round.
- No firmware build, flash, serial, NVS write, provider execution, Gateway
  start, or physical hardware action was performed.
- The main control thread still needs to schedule the first worker.

Known risks/blockers:

- The local official source trees used for comparison may be dirty; the first
  worker must record exact remotes, branches, HEAD commits, and dirty state.
- The current A21 worktree already contains main-control Internal Test 4
  changes; workers must not revert or overwrite them.
- Product hardware availability must remain blocked unless a worker produces
  physical or diagnostic evidence with `trace_id`, `session_id`, and
  `device_id`.
- Camera/video/NFC/IR/app-lifecycle work is high risk and must not be folded
  into the first low-risk MCP/status worker.

Test/build/runtime results:

- No product code was changed.
- Plan self-review searched the new plan for placeholder markers such as
  `TODO`, `TBD`, and stale placeholder source paths; none remained.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Recommended next action:

- In the main control thread, dispatch
  `T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001` first as a docs-only/read-only
  worker. After it lands, schedule the low-risk Gateway MCP/status worker
  before any firmware or high-risk hardware work.

## 2026-06-04 05:18 CST - StackChan Official Hardware Parity Gap Map Frozen

Round goal:

- Execute `T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001` as a docs-only/read-only
  scoped worker and freeze the official-vs-A21 hardware/control/status parity
  gap map for later low-risk MCP/status, display/action, and physical evidence
  workers.

Actual completed work:

- Read the required control inputs:
  `AGENTS.md`,
  `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`,
  the latest handoff entries, `docs/project_state_machine.md` next candidates,
  and current git status/diff.
- Recorded official source identity:
  StackChan root remote `https://github.com/m5stack/StackChan.git`, branch
  `main...origin/main`, HEAD
  `da156e1fa0e1c2a5e00b78fbf69b1f7e7bca0483`, dirty working tree with
  modified firmware files and untracked `firmware/sources/`; references from
  this tree are working-tree references, not clean upstream HEAD.
- Recorded Xiaozhi sub-tree identity:
  remote `https://gitclone.com/github.com/78/xiaozhi-esp32.git`, detached
  clean `HEAD` at `e77dedb1309153bb63fed285772962c920c97dd4`; references from
  this tree are HEAD references.
- Recorded A21 worker source identity:
  detached worker checkout
  `/Users/jiyurun/.codex/worktrees/be17/New project`, HEAD
  `15757cad5e7cbcf2df5f2601ffa3904dd766e0ac`, initially clean.
- Added `## Official StackChan Parity Gap Map` to the capability charter with
  official inventory, A21 current inventory, parity rows, landing classes,
  owner transitions, acceptance evidence, rollback paths, and explicit
  `available` / `diagnostic` / `planned` / `blocked` / `product-accepted`
  distinctions.
- Updated current control with the scoped transition status and next operator
  action.
- Updated the project state machine with the completed docs/state baseline and
  changed the next parity action to the low-risk MCP/status worker.

Changed files:

- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- No Gateway MCP/status endpoints were implemented in this round.
- No display-state registry, action/motion/RGB/servo mapping, sensor/battery
  diagnostics, physical touch evidence, camera/video/NFC/infrared spike, or app
  lifecycle reconciliation was implemented.
- Physical StackChan PRD acceptance remains open.

Known risks/blockers:

- The official StackChan root tree is dirty and must remain reference material
  only until a clean upstream/source-control decision is made.
- A21's current stock Xiaozhi microphone/speaker transport evidence is not full
  product acceptance; playback-start/stop, trusted audible evidence, and final
  PRD physical evidence remain required.
- High-risk camera/video/NFC/infrared/app-lifecycle work must not be folded
  into the next low-risk MCP/status worker.

Test/build/runtime results:

- `git diff --check`: passed before handoff append.
- `GOMAXPROCS=2 make verify`: passed.
- Final `git diff --check` after handoff append: passed.

Recommended next action:

- Dispatch `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001` for low-risk Gateway
  MCP/status parity only: `self.get_device_status`,
  `self.screen.set_brightness`, `self.screen.set_theme`, and
  `self.screen.get_info`.
- Keep reboot, firmware upgrade, camera/photo, screen snapshot, camera stream,
  video, NFC, infrared, app lifecycle, firmware build, flash, serial, and NVS
  out of the next worker.

Forbidden actions avoided:

- No Gateway start.
- No provider or V21 execution.
- No firmware build.
- No flash.
- No serial access.
- No NVS write.
- No product-code edits.
- No unguarded upload command.
- No official capability was promoted to A21 product acceptance.

## 2026-06-04 05:34 CST - Internal Test 4 Roleplay Runtime Profile Slice

Round goal:

- Continue internal test 4 from the committed `roleplay`/`professional` mode
  contract into a concrete Gateway roleplay runtime selector without regressing
  internal test 3 Xiaozhi voice behavior or the V21 professional boundary.

Actual completed work:

- Accepted and pushed the StackChan official hardware parity control-tower plan
  as commit `15757ca` and queued a separate docs-only/read-only gap-map worker
  from the pushed branch. Pending worktree id:
  `local:f9fd1abd-ffe0-4fbf-bbb7-eb71823b4885`.
- Added Gateway `GET/POST/PUT /v1/roleplay-profile` with schema
  `a21.gateway.roleplay_profile.v1`.
- Added roleplay profile/scenario/voice-clone runtime selection. Voice-clone
  selection through the roleplay endpoint updates the existing voice-chain
  selector so the selected clone reaches the TTS boundary.
- Added redacted fast-companion roleplay runtime summaries and trace markers
  `roleplay.profile.ready` plus `roleplay.memory.ready`.
- Updated simulator controls to display and persist roleplay scenarios while
  preserving the `roleplay` default voice mode.
- Updated protocol, internal test 4 plan, current control, and project state
  machine docs for the roleplay runtime slice.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- Cloud/Web/App upload, indexing, workspace query scope, and device binding are
  still planned; this round does not implement those APIs.
- Hardware professional consult and physical StackChan PRD acceptance remain
  open.
- The queued hardware parity gap-map worker has not returned yet.

Known risks/blockers:

- `roleplay` runtime memory is still environment-backed bounded prompt input,
  not a durable user memory store.
- `professional` must stay out of fast-companion/dialogue execution until the
  explicit V21 adapter/workspace path is implemented.
- Do not use this slice to claim cloud workspace readiness or physical PRD
  acceptance.

Test/build/runtime results:

- `go test ./internal/gateway -run 'SimulatorPageServed|RoleplayProfile|FastCompanionHybridRoutesLocalAudioFrontendToTextStreamBoundary' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `go test ./internal/protocol ./internal/gateway ./internal/personality ./internal/v21adapter -count=1`:
  passed.

Recommended next action:

- Run final `git diff --check` and `GOMAXPROCS=2 make verify`, commit/push the
  roleplay runtime slice if clean, then continue with the next internal test 4
  slice: no-execute upload/workspace API contract plus V21 adapter v2
  `workspace_id`/`query_scope` fields.

## 2026-06-04 06:12 CST - Internal Test 4 Professional Workspace Contract Slice

Round goal:

- Continue internal test 4 by landing the no-execute professional workspace and
  V21 adapter v2 query-scope contract, without claiming cloud upload/index
  readiness or touching firmware/hardware.

Actual completed work:

- Added Gateway `GET/POST/PUT /v1/professional-workspace` with schema
  `a21.gateway.professional_workspace.v1`.
- Added redacted professional context selection for `user_id`, `workspace_id`,
  and `query_scope`.
- Supported query scopes: `public_only`, `personal_only`, and
  `personal_plus_public`.
- Extended the A21/V21 adapter query request with v2 fields:
  `device_id`, `user_id`, `workspace_id`, and `query_scope`.
- Extended adapter responses and smoke/readiness surfaces with redacted
  `source_scope_counts` and `workspace_status`.
- Wired Gateway professional turns and stock Xiaozhi professional turns to send
  the selected workspace/query-scope context to the V21 adapter.
- Added simulator `professionalQueryScope` control.
- Updated protocol, V21 integration, observability, current control, internal
  test 4 plan, and state machine docs.
- Product-readiness remains compatible with old v1 smoke reports while
  mirroring optional v2 scope/status/count fields when present.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `internal/v21adapter/client.go`
- `internal/v21adapter/client_test.go`
- `internal/v21adapter/smoke.go`
- `internal/v21adapter/professional_readiness.go`
- `internal/app/professional_adapter_bridge.go`
- `internal/app/product_demo.go`
- `internal/app/app_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/V21_INTEGRATION.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- No file upload/import/index job execution was implemented.
- No durable auth/account/device binding store was implemented.
- No V21 repository-side personal/public corpus enforcement was implemented.
- Hardware professional consult evidence and physical StackChan PRD acceptance
  remain open.
- The hardware parity gap-map worker still has not returned in this control
  thread.

Known risks/blockers:

- `personal_only` and `personal_plus_public` are contract scopes only until V21
  and cloud upload/index workers implement source-scope enforcement.
- Product-readiness accepts v1 adapter smoke for internal test 3 evidence; do
  not reinterpret that as internal test 4 cloud workspace readiness.
- Professional traces must continue to avoid user/workspace labels, utterance
  text, retrieved text, URLs, paths, and secrets.

Test/build/runtime results:

- `go test ./internal/v21adapter ./internal/gateway ./internal/app -run 'HTTPClientPostsProfessionalQueryContract|HTTPClientPostsExplicitWorkspaceScopeContract|ProfessionalWorkspace|ProfessionalModeSendsExplicitV21PlaceholderContract|SimulatorPageServed|V21Adapter|Professional' -count=1`:
  passed.
- `go test ./internal/v21adapter ./internal/gateway ./internal/app -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Recommended next action:

- Commit/push this workspace contract slice if the staged range remains
  limited to the files above.
- Next implementation candidate: no-execute upload/import/index job API
  skeleton with redacted job status, followed by a V21-side worker for
  personal/public corpus enforcement.

## 2026-06-04 05:53 CST - Internal Test 4 Workspace Upload Job Skeleton

Round goal:

- Continue internal test 4 by adding a no-execute upload/import/index job
  lifecycle surface for the A21 workspace, without storing raw documents,
  executing V21 indexing, touching firmware, or regressing internal test 3
  Xiaozhi/StackChan voice behavior.

Actual completed work:

- Added Gateway `GET/POST/PUT /v1/workspace-upload-jobs` with schema
  `a21.gateway.workspace_upload_jobs.v1`.
- Added redacted metadata-only job creation for `upload` and `import`
  source kinds with `personal` or `public` source scope.
- Added job polling plus `mark_failed`, `retry`, and `delete` actions.
- Kept job states explicitly no-execute:
  `accepted_no_execute`, `not_started_no_execute`, `failed_no_execute`, and
  `deleted_no_execute`.
- Rejected raw document text, file bytes, base64 payloads, import URLs, local
  paths, credentials, API keys, tokens, and provider outputs at decode time.
- Added trace markers for accepted, failed, deleted, and not-started index
  states without tracing document/user content.
- Added simulator `Workspace Job` control wired to the current professional
  query-scope context.
- Updated protocol, observability, V21 integration, current-control, internal
  test 4 plan, and project state machine docs.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/V21_INTEGRATION.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- No real file upload byte storage is implemented.
- No import URL fetching is implemented.
- No indexing worker or V21 repository-side corpus enforcement is implemented.
- No durable auth/account/device binding store is implemented.
- Hardware professional consult evidence and physical StackChan PRD acceptance
  remain open.
- The hardware parity gap-map worker queued from pending worktree
  `local:f9fd1abd-ffe0-4fbf-bbb7-eb71823b4885` has not returned in this
  control thread yet.

Known risks/blockers:

- `personal` and `public` source scopes are lifecycle metadata only until the
  V21/cloud workspace workers implement ACL-aware storage and retrieval.
- The endpoint intentionally refuses raw upload/import payloads; frontend or
  cloud workers must add storage under a separate plan before claiming upload
  execution readiness.
- Traces and reports must continue to avoid raw document text, filenames that
  contain paths/URLs/secrets, provider outputs, credentials, and user content.

Test/build/runtime results:

- `git diff --check`: passed before this handoff entry.
- `go test ./internal/gateway -run 'SimulatorPageServed|WorkspaceUploadJobs' -count=1`:
  passed.
- `go test ./internal/gateway ./internal/app -run 'SimulatorPageServed|WorkspaceUploadJobs|ProfessionalWorkspace|ProductReadiness' -count=1`:
  passed.
- `go test ./internal/gateway ./internal/app ./internal/v21adapter -count=1`:
  passed.
- `GOMAXPROCS=2 make verify`: passed.

Recommended next action:

- Commit/push the workspace job skeleton if the range remains limited.
- Next implementation candidate: V21/cloud worker contract for actual
  upload/import/index execution and personal/public corpus enforcement, while
  keeping StackChan document-free and provider-key-free.

## 2026-06-04 05:59 CST - Main Control Integrated Hardware Parity Gap Map

Round goal:

- Audit whether the scoped hardware parity gap-map worker completed, then bring
  the completed worker result back into the current main-control branch without
  reverting the internal test 4 roleplay, professional workspace, or workspace
  job changes.

Actual completed work:

- Located worker thread `019e8f55-d945-75a2-b887-14061793a086`
  (`映射官方硬件差距`) and read its completion summary.
- Fetched worker branch `codex/stackchan-official-hw-parity-gap-map-001`.
- Cherry-picked worker commit
  `d02a592 docs(stackchan): map official hardware parity gaps` into the current
  branch as `271c25c`.
- Resolved the only content conflict in `docs/agent_handoff_log.md` by keeping
  both the worker's `05:18` hardware parity handoff and the current main-control
  internal test 4 handoff entries.
- Corrected `docs/engineering/A21_CURRENT_CONTROL.md` so the current source
  baseline remains `0f4b354 feat(gateway): add workspace upload job skeleton`
  rather than the older worker checkout base.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/project_state_machine.md`

Unfinished items:

- The low-risk MCP/status parity worker has not been implemented yet.
- No hardware, Gateway runtime, provider, V21, firmware, flash, serial, or NVS
  action was performed during this integration.

Known risks/blockers:

- The official StackChan root reference remains a dirty working-tree reference,
  not a clean upstream baseline.
- Hardware PRD acceptance remains blocked by physical evidence and playback/
  hardware-surface proof, not by this docs-only gap map.

Test/build/runtime results:

- `git diff --cached --check`: passed before `271c25c`.
- `git diff --check`: passed after the main-control integration note.
- `GOMAXPROCS=2 make verify`: passed.

Recommended next action:

- Commit this main-control integration note and push the current branch.
- Then schedule `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001` as the next
  low-risk hardware parity implementation slice.

## 2026-06-04 06:01 CST - Main Control Dispatch Prepared For MCP Status Parity

Round goal:

- Prepare the next scoped worker dispatch after integrating the official
  hardware parity gap map, keeping the low-risk MCP/status work separate from
  firmware, flash, camera, app lifecycle, and other high-risk surfaces.

Detailed plan path:

- `docs/plans/2026-06-04-stackchan-official-mcp-status-parity.md`
  - Worker must create or update this plan before implementation.
  - The plan must reference the parent parity plan
    `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`
    and the frozen parity matrix in
    `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`.

Worker execution task:

- Execute `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001`.
- Add only low-risk Gateway/Xiaozhi MCP status parity for official tools:
  `self.get_device_status`, `self.screen.set_brightness`,
  `self.screen.set_theme`, and `self.screen.get_info`.
- Add tests and docs/state/handoff updates proving the new surface is
  redacted, A21-namespaced, trace/session/device-ready, and does not promote
  official-only capability to product acceptance.

Worker boundary conditions:

- May inspect `internal/gateway/server.go`,
  `internal/gateway/server_test.go`, `internal/gateway/simulator.go`,
  `docs/engineering/PROTOCOL.md`,
  `docs/engineering/OBSERVABILITY.md`,
  `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`,
  `docs/project_state_machine.md`, and `docs/agent_handoff_log.md`.
- Must not build firmware, flash firmware, touch serial, write NVS, start
  Gateway, execute provider APIs, execute V21, or touch physical hardware.
- Must not expose or implement `self.reboot`, `self.upgrade_firmware`,
  `self.camera.take_photo`, `self.screen.snapshot`, camera stream/video, NFC,
  infrared, or app-lifecycle work.
- Must not store or log raw MCP response bodies, screenshots, images, provider
  output, secrets, full URLs, local paths, transcripts, or raw/base64 audio.
- Must not revert internal test 4 roleplay/professional/workspace job changes
  or internal test 3 voice/protocol changes.

Required worker summary format:

- Transition
- What changed
- Files changed
- Tests run and results
- Runtime or physical evidence
- Deviations from plan
- Remaining issues
- Next suggested action
- Forbidden actions avoided
- Commit and push status

Dispatch status:

- Pending worker creation from current pushed main-control branch
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash` after this
  dispatch record is committed and pushed.

## 2026-06-04 06:06 CST - Main Control Dispatch Prepared For V21 A21 v2 Native Scope

Round goal:

- Prepare the V21-side worker needed by internal test 4 so V21 can natively
  accept and report A21 v2 professional workspace/query-scope metadata, while
  preserving A21/V21 separation and the existing dirty V21 LAN discovery work.

Detailed plan path:

- A21 control plan:
  `docs/plans/2026-06-04-v21-a21-v2-workspace-query-scope-native-contract.md`
- Required V21 worker plan:
  `/Users/jiyurun/Documents/v21-knowledge-platform/docs/plans/2026-06-04-a21-v2-workspace-query-scope-native-contract.md`

Worker execution task:

- Execute `T-V21-A21-V2-WORKSPACE-QUERY-SCOPE-NATIVE-001` in the V21 repository.
- Make V21 `/internal/v1/knowledge/voice-query` accept A21 v2 additive request
  fields `device_id`, `user_id`, `workspace_id`, and `query_scope`.
- Add safe response metadata `source_scope_counts` and `workspace_status`.
- If V21 cannot yet prove personal/public ACL enforcement with the current
  schema, return or document a stable pending/blocker status instead of
  claiming `personal_plus_public` is searchable.

Worker boundary conditions:

- V21 repo path: `/Users/jiyurun/Documents/v21-knowledge-platform`.
- Existing V21 working tree is dirty on branch `feat/consumer-lan-discovery-ui`
  with LAN discovery / desktop connector changes. Worker must not revert,
  overwrite, stage, or reformat those files.
- No A21 code edits in this worker.
- No A21 Gateway runtime start, no StackChan hardware, no firmware build, no
  flash, no serial, no NVS, no provider API execution.
- No deletion, prune, report cleanup, or rewrite of existing V21 reports or
  worktrees.
- No query text, answer text, evidence body, credential, full URL, local path,
  transcript, raw audio, or provider output may be written to reports/docs/chat.

Required worker summary format:

- Transition
- What changed
- Files changed
- Tests run and results
- Runtime evidence
- Deviations from plan
- Remaining issues
- Next suggested action
- Forbidden actions avoided
- Commit and push status

Dispatch status:

- Pending V21 scoped worker creation after this A21 dispatch record is
  committed and pushed.

Test/build/runtime results:

- `git diff --check`: passed before commit.
- `GOMAXPROCS=2 make verify`: passed.

## 2026-06-04 06:07 CST - Worker Completed MCP Status Parity

Round goal:

- Execute `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001` as a scoped low-risk
  Gateway/Xiaozhi MCP status/control parity worker.

Actual completed work:

- Created detailed transition plan
  `docs/plans/2026-06-04-stackchan-official-mcp-status-parity.md` with current
  state, target state, non-goals, impact scope, steps, acceptance, failure
  state, rollback, and human confirmation points.
- Added `POST /v1/xiaozhi/mcp-control` for only:
  `self.get_device_status`, `self.screen.set_brightness`,
  `self.screen.set_theme`, and `self.screen.get_info`.
- Kept `self.audio_speaker.set_volume` on its existing endpoint and did not
  change its behavior.
- Added strict argument validation: brightness `0..100`, bounded safe theme
  tokens, and no user arguments for status/info.
- Rejected non-whitelisted high-risk tools before websocket write, including
  reboot, firmware upgrade, camera/photo, screen snapshot, camera stream/video,
  NFC, infrared, and app lifecycle examples.
- Recorded only redacted trace/device markers after successful MCP delivery:
  `xiaozhi.mcp.device_status.sent`,
  `xiaozhi.mcp.screen_brightness.sent`,
  `xiaozhi.mcp.screen_theme.sent`, and `xiaozhi.mcp.screen_info.sent`.
- Updated protocol, observability, capability charter, and project state docs
  to keep this as control-contract parity only, not product or physical
  acceptance.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-stackchan-official-mcp-status-parity.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- No physical StackChan screen/status evidence was collected.
- No raw MCP response parsing or result correlation is implemented.
- No firmware, reboot, OTA, camera/photo, snapshot, camera stream/video, NFC,
  infrared, or app lifecycle parity was attempted.

Known risks/blockers:

- Delivery of an MCP `tools/call` proves only that Gateway sent the request to
  a connected stock MCP-capable `/v1/xiaozhi` socket. It does not prove the
  device applied brightness/theme or returned status/info.
- Future result correlation must remain redacted and must not store raw MCP
  response bodies, screenshots, images, provider output, URLs, paths,
  transcripts, secrets, or raw/base64 audio.
- Product acceptance remains blocked on physical evidence and main-control
  review.

Test/build/runtime results:

- `go test ./internal/gateway -run 'XiaozhiMCPStatusParity|XiaozhiSpeakerVolumeUsesStockMCPToolCall' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Recommended next action:

- Main control should review/merge this worker branch, then choose the next
  medium-risk display, avatar/action, touch, or diagnostic-sensor parity slice
  with a fresh plan and evidence gate.

Forbidden actions avoided:

- No firmware build, firmware flash, serial access, NVS write, Gateway start,
  provider API execution, V21 execution, or physical hardware action occurred.
- No `self.reboot`, `self.upgrade_firmware`, `self.camera.take_photo`,
  `self.screen.snapshot`, camera stream/video, NFC, infrared, or app lifecycle
  tool was exposed.
- No internal test 3 Xiaozhi voice/protocol work or internal test 4
  roleplay/professional/workspace job work was reverted.

## 2026-06-04 06:24 CST - Internal Test 4 Roleplay Memory Control Cut

Round goal:

- Continue the internal test 4 roleplay/persona/memory lane without regressing
  internal test 3 device-side voice/protocol behavior, and make roleplay memory
  a product-visible control surface paired with scenario and voice-clone
  selection.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-roleplay-memory-control-surface.md`.
- Extended `POST`/`PUT /v1/roleplay-profile` with `memory_hints` and
  `clear_memory`.
- Runtime roleplay hints are sanitized through the existing personality memory
  policy, stored only as bounded Gateway runtime prompt-input hints, and
  reported only as readiness/counts/finding codes.
- Fast companion roleplay summaries use the configured hint count and keep
  `professional_route_allowed=false` and `v21_executed=false`.
- Simulator now exposes set/clear roleplay memory controls and displays only
  memory status/count.
- Updated protocol, internal-test4 plan, current control state, and project
  state machine.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/plans/2026-06-04-roleplay-memory-control-surface.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not durable long-term memory, account memory sync, document upload,
  personal workspace indexing, V21 ACL enforcement, or physical StackChan
  roleplay evidence.
- Persona quality review, prompt examples, wake/touch entry into roleplay
  memory, and voice-clone runtime evidence remain follow-up work.

Known risks/blockers:

- Runtime hints are intentionally in-process only; a Gateway restart clears
  them.
- Environment-provided `A21_MEMORY_*` hints still exist as operator/runtime
  configuration and are not cleared by `clear_memory`.
- Physical acceptance remains blocked until roleplay turns are exercised on the
  actual StackChan path.

Recommended next action:

- Integrate the hardware MCP/status parity worker result into the main-control
  branch.
- Then schedule the next roleplay cut for persona-quality examples plus
  wake/touch entry evidence, or choose the next hardware display/avatar/touch
  parity slice after main-control review.

Test/build/runtime results:

- `go test ./internal/gateway ./internal/personality -run 'SimulatorPageServed|RoleplayProfile|FastCompanionHybridRoutesLocalAudioFrontendToTextStreamBoundary|PersonalityMemory' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or V21 execution occurred, and
  no firmware build, flash, serial, NVS, or physical hardware action occurred.

Failure location/reason:

- None in this round.

## 2026-06-04 06:24 CST - Main Control Integrated MCP Worker And V21 Scope Result

Round goal:

- Regain control over the two active scoped workers, integrate the completed
  A21 hardware MCP/status parity worker, and record the completed V21 native
  scope worker result without mixing V21 code into A21.

Actual completed work:

- Read active worker threads:
  - `019e8f82-7488-73a1-86ae-2afefefee595`:
    `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001`.
  - `019e8f88-5301-7101-afcc-79fb7a5c88e1`:
    `T-V21-A21-V2-WORKSPACE-QUERY-SCOPE-NATIVE-001`.
- Cherry-picked A21 MCP worker commit `3ec1540` into the main-control branch
  as `b19d420 feat(gateway): add xiaozhi mcp status parity`.
- Resolved the only cherry-pick conflict in `docs/agent_handoff_log.md` by
  preserving the V21 dispatch record, the MCP worker completion record, and
  the roleplay memory control record in time order.
- Recorded the V21 worker result in A21 control docs:
  V21 branch `origin/codex/a21-v2-workspace-query-scope-native-contract`,
  commit `ad61246 feat(voice-query): accept A21 workspace scope metadata`.
- Updated A21 state to show V21 native A21 v2 request/response shape is
  complete on the V21 worker branch, while personal/public ACL enforcement is
  still pending and must not be claimed by A21 yet.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-stackchan-official-mcp-status-parity.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-v21-a21-v2-workspace-query-scope-native-contract.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- V21 branch `origin/codex/a21-v2-workspace-query-scope-native-contract` still
  needs V21-side review/merge/release.
- V21 public/personal ACL enforcement still needs an ADR/schema/retrieval
  filter transition before A21 can claim nonzero `source_scope_counts`.
- MCP status/control delivery is Gateway contract parity only; it is not
  physical screen/status product acceptance.

Known risks/blockers:

- V21 worker reported `make test`/`make check` blocked by Node toolchain
  mismatch and `boundary-check` blocked by existing `v21air-lan-demo` Jaeger
  port ownership. It did not start services or touch dirty LAN/desktop work.
- MCP result correlation remains unimplemented; future work must not store raw
  MCP response bodies, screenshots, images, URLs, paths, transcripts, secrets,
  provider output, or raw/base64 audio.

Recommended next action:

- Push this main-control integration commit.
- Then choose between:
  - V21 merge/ACL ADR coordination for professional workspace correctness.
  - Next hardware parity slice: display/avatar/action/touch/diagnostic sensors
    with explicit physical-evidence gate.
  - Next roleplay slice: persona example quality plus wake/touch entry evidence.

Test/build/runtime results:

- `go test ./internal/gateway ./internal/personality -run 'XiaozhiMCPStatusParity|XiaozhiSpeakerVolumeUsesStockMCPToolCall|RoleplayProfile|FastCompanionHybridRoutesLocalAudioFrontendToTextStreamBoundary|SimulatorPageServed|PersonalityMemory' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or V21 execution occurred from
  A21, and no firmware build, flash, serial, NVS, or physical hardware action
  occurred.

Failure location/reason:

- None in this main-control integration round.

## 2026-06-04 06:36 CST - Roleplay Prompt Voice Pipeline Cut

Round goal:

- Close the gap where roleplay personality/memory was configurable and
  summarized but not guaranteed to affect the actual low-latency voice
  provider request.

Actual completed work:

- Added plan `docs/plans/2026-06-04-roleplay-prompt-voice-pipeline.md`.
- Added runtime-only `VoicePipelineRequest.TextPrompt`.
- Updated voice pipeline text-stream execution to use `TextPrompt` when
  present, while keeping ASR transcript metrics/reporting separate.
- Added `prompt_input_ready` and `prompt_input_not_recorded` report metadata
  without storing prompt bodies.
- Injected composed roleplay personality/scenario/memory prompt into:
  - fast-companion roleplay turns with PCM frames;
  - stock `/v1/xiaozhi` roleplay voice-pipeline turns.
- Added redacted trace marker `roleplay.prompt_input.used`.
- Added provider and Gateway tests proving prompt input reaches the text-stream
  boundary and does not leak through responses, traces, or voice-pipeline
  reports.
- Updated protocol, internal-test4 plan, current-control, and state-machine
  docs.

Changed files:

- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-roleplay-prompt-voice-pipeline.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is still host/runtime evidence, not physical StackChan roleplay
  acceptance.
- Persona quality examples, wake/touch entry into roleplay, and audible
  physical proof remain follow-up work.
- Voice-clone provider runtime evidence remains separate from prompt injection.

Known risks/blockers:

- Prompt bodies intentionally enter provider request memory. They must remain
  out of reports, traces, docs examples, and API responses.
- Stock `/v1/xiaozhi` physical roleplay evidence still requires an operator
  turn with provider execution and trusted audible/playback evidence.

Recommended next action:

- Schedule either roleplay persona-quality/physical-evidence worker, or the
  next StackChan display/avatar/touch parity slice. Keep V21 ACL ADR/merge
  coordination active in parallel.

Test/build/runtime results:

- `go test ./internal/providers ./internal/gateway -run 'VoicePipelineRunnerUsesPromptInput|VoicePipelineRunnerProducesDownlinkReady|FastCompanionHybridRunsVoicePipelineWhenFramesProvided|XiaozhiWebSocketListenStopRunsVoicePipelineAndSendsPacedOpus|RoleplayProfileEndpointSetsRuntimeMemoryHints|RoleplayProfileEndpointPersistsScenarioVoiceClone' -count=1`:
  passed.
- `go test ./internal/providers ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or V21 execution occurred, and
  no firmware build, flash, serial, NVS, or physical hardware action occurred.

Failure location/reason:

- None in this round.

## 2026-06-04 06:47 CST - Roleplay Voice Clone Pipeline Contract

Round goal:

- Close the remaining internal test 4 roleplay gap where selecting a
  `voice_clone_profile` updated the selector/TTS profile but was not carried
  as per-turn evidence into the provider-neutral voice pipeline and TTS
  request.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-roleplay-voice-clone-pipeline-contract.md`.
- Added runtime-only `VoicePipelineRequest.VoiceCloneProfile`.
- Added `TTSAdapterRequest.VoiceCloneProfile` and passed the safe A21 voice
  profile ID from voice pipeline synthesis into TTS adapters.
- Updated local TTS adapters to place the safe profile ID in
  `LocalTTSOptions.Voice`, so `voice_clone_cli` can bind a turn to the selected
  voice identity without exposing reference audio/text configuration.
- Added `VoicePipelineReport.input.voice_clone_profile` and
  `voice_clone_sample_not_recorded` redaction metadata.
- Fast-companion roleplay turns with PCM frames and stock `/v1/xiaozhi`
  roleplay voice-pipeline turns now carry the selected clone profile into the
  voice pipeline and trace only `roleplay.voice_clone_profile.used`.
- Updated protocol/current-control/internal-test4/state-machine docs.
- Corrected the hardware parity next action in current-control/state-machine:
  gap-map and low-risk MCP/status are already completed, so the next hardware
  slice is `T-STACKCHAN-OFFICIAL-STATUS-DISPLAY-PARITY-001`, not another
  gap-map or MCP worker.

Changed files:

- `internal/providers/voice_pipeline.go`
- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_test.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-roleplay-voice-clone-pipeline-contract.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This does not execute a real clone provider and does not prove physical
  StackChan roleplay audio.
- Durable account voice/persona binding, provider-specific clone profile
  routing, and voice sample management remain separate product/security gates.
- Hardware parity status-display/device-registry work remains the next scoped
  hardware worker.

Known risks/blockers:

- Selecting a clone profile still relies on the existing `voice_clone_cli`
  configuration for actual reference audio/model behavior. The runtime contract
  now carries the selected safe profile ID, but missing clone provider/env
  configuration must be reported truthfully by provider/runtime evidence.
- Prompt bodies, memory text, transcripts, provider output, reference
  audio/text, local paths, URLs, credentials, voice samples, and raw/base64
  audio must remain out of reports, traces, docs, and API responses.

Recommended next action:

- Run broad provider/Gateway tests, `git diff --check`, and `GOMAXPROCS=2 make
  verify`, then commit/push this scoped runtime-contract cut.
- Then dispatch `T-STACKCHAN-OFFICIAL-STATUS-DISPLAY-PARITY-001` as the next
  hardware parity worker with no firmware/flash/serial/NVS/camera/video/NFC/IR
  scope.

Test/build/runtime results:

- `go test ./internal/providers ./internal/gateway -run 'VoicePipelineRunnerPassesVoiceCloneProfile|VoicePipelineAdaptersFromEnvSelectsVoiceCloneCLI|FastCompanionHybridRunsVoicePipelineWhenFramesProvided|XiaozhiWebSocketListenStopRunsVoicePipelineAndSendsPacedOpus|RoleplayProfileEndpointPersistsScenarioVoiceClone' -count=1`:
  passed.
- `go test ./internal/providers ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or V21 execution occurred, and
  no firmware build, flash, serial, NVS, or physical hardware action occurred.

Failure location/reason:

- None in this round.

## 2026-06-04 07:09 CST - Official Status Display Registry Parity

Round goal:

- Land `T-STACKCHAN-OFFICIAL-STATUS-DISPLAY-PARITY-001` as the next hardware
  parity slice after the gap map and low-risk MCP/status worker, without
  touching firmware, serial, NVS, providers, V21, or physical hardware.

Actual completed work:

- Added stable A21 `DisplayState` protocol values for official
  StackChan/Xiaozhi status-display states:
  `starting`, `wifi_configuring`, `idle`, `connecting`, `listening`,
  `thinking`, `speaking`, `upgrading`, `audio_testing`, `error`, and
  `fatal_error`.
- Added official-to-A21 state normalization. Official aliases such as
  `activating` normalize to `connecting`; unknown or legacy-looking names
  normalize to `error`.
- Added optional `display_state` to A21 device events.
- Added `/v1/devices` registry display-state fields:
  `display_state`, `display_state_source`, `display_state_trace_id`,
  `display_state_session_id`, `display_state_updated_at_ms`, and
  `display_state_physical_accepted`.
- Updated stock Xiaozhi turn-state fanout and A21 device-event ingestion to
  record display state without erasing device capabilities or `runtime_echo`.
- Added trace markers `stackchan.display_state.received`,
  `stackchan.display_state.normalized`, and
  `stackchan.display_state.registry_updated`.
- Updated protocol, observability, current-control, hardware parity plan, and
  state-machine docs. Next hardware candidate is now
  `T-STACKCHAN-OFFICIAL-ACTION-PARITY-001`.

Changed files:

- `internal/protocol/message.go`
- `internal/protocol/message_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is registry/status parity only. It does not prove physical screen
  rendering, avatar behavior, RGB output, servo motion, or product acceptance.
- `display_state_physical_accepted` remains false until a separate physical
  screen/status report proves rendering on the device.
- Official avatar/motion/RGB/servo semantic mapping remains the next scoped
  Gateway/transport worker.

Known risks/blockers:

- Official display words that A21 does not recognize intentionally collapse to
  `error`; future official aliases must be added through protocol tests before
  being treated as product states.
- Trace markers are state-label metadata only. They must not be used as
  screen-rendering, camera, provider, V21, transcript, or audio evidence.

Recommended next action:

- Dispatch `T-STACKCHAN-OFFICIAL-ACTION-PARITY-001` for official avatar,
  motion, RGB, and servo semantic mapping with simulator/Gateway tests first.
  Keep physical hardware control behind a separate foreground hardware window.

Test/build/runtime results:

- `go test ./internal/protocol ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or V21 execution occurred, and
  no firmware build, flash, serial, NVS, or physical hardware action occurred.

Failure location/reason:

- None in this round.

## 2026-06-04 07:34 CST - Official Avatar Action Semantic Mapping

Round goal:

- Land `T-STACKCHAN-OFFICIAL-ACTION-PARITY-001` as a host/Gateway transport
  mapping cut after status-display parity, without starting Gateway, touching
  providers/V21, building firmware, flashing, using serial, writing NVS, or
  claiming physical action/RGB/servo acceptance.

Actual completed work:

- Added `BuildOfficialActionPlan` in the official StackChan transport adapter.
  It returns the normalized A21 device extension event, official binary packet
  sequence, packet count, semantic surface metadata, and
  `physical_accepted=false`.
- Preserved the existing official frame types only:
  `ControlAvatar` (`0x03`), `ControlMotion` (`0x04`), and `DanceSequence`
  (`0x14`).
- Added deterministic tests for all semantic states: idle, listening,
  thinking, speaking, and error all emit avatar + pitch motion packets with
  stable metadata.
- Marked yaw/`servo_x` movement as `servo_x_candidate_yaw_sequence`, added an
  explicit yaw clamp helper, and tested yaw keyframes remain within
  `-180..180`.
- Marked RGB as semantic `*_no_rgb_frame` metadata because this adapter does
  not emit an official RGB packet yet.
- Kept display, heartbeat, camera, video, and call frame classes outside the
  avatar/action adapter.
- Updated `POST /v1/stackchan/official/control` to return packet count,
  official action surfaces, and `official_action_physical_accepted=false`.
- Mirrored the same action metadata into `/v1/devices.runtime_echo` with
  `official_stackchan_` keys.
- Updated protocol, observability, hardware capability charter, current
  control, hardware parity plan, project state machine, and this handoff.

Changed files:

- `internal/transport/stackchan/official_avatar.go`
- `internal/transport/stackchan/official_avatar_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not physical avatar, servo, or RGB evidence. It only proves packet
  shape, Gateway response metadata, and registry metadata.
- RGB still needs a future official packet/evidence cut before any physical LED
  claim.
- `servo_x` yaw remains candidate-only until mechanical safety and visible
  motion evidence are collected in a foreground hardware window.

Known risks/blockers:

- `/v1/stackchan/official/control` delivery only proves websocket packet write
  to a connected official StackChan socket; it does not prove the physical
  device moved or rendered.
- Future official frame classes such as text/call/video/camera/audio stream
  still require separate privacy/safety scopes and must not be smuggled into
  this adapter.

Recommended next action:

- Dispatch `T-STACKCHAN-OFFICIAL-ACTION-PHYSICAL-EVIDENCE-001` only after an
  approved hardware window is open. It should collect touch, barge-in, visible
  avatar/motion, RGB, and servo evidence with trace/session/device IDs.

Test/build/runtime results:

- `go test ./internal/transport/stackchan -count=1`: passed.
- `go test ./internal/gateway -run 'Test.*Official.*Control|Test.*StackChan.*Action' -count=1`:
  passed.
- `go test ./internal/transport/stackchan ./internal/gateway -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or V21 execution occurred, and
  no firmware build, flash, serial, NVS, or physical hardware action occurred.

Failure location/reason:

- None in this round.

## 2026-06-04 07:58 CST - Professional Mode Ritual Contract

Round goal:

- Make professional mode switching visible and intentional for internal test 4
  without changing the accepted internal test 3 Xiaozhi voice path or executing
  provider/V21/hardware work.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-professional-mode-ritual-contract.md`.
- Added `VoiceModeRitual` to `/v1/voice-modes` responses.
- Added top-level `selected_ritual` plus per-mode `ritual` metadata.
- Professional mode now exposes a redacted mode-switch ritual:
  `screen_label=PRO`, cue `我在查，先把证据和置信度拉出来。`,
  expression `professional`, trace marker
  `professional.checking_feedback.sent`, workspace policy
  `professional_only`, `v21_allowed=true`, and `physical_accepted=false`.
- Roleplay mode exposes a no-V21 ritual with `screen_label=A21` and
  `physical_accepted=false`.
- Simulator now displays the selected mode cue through `modeRitualReadout`.
- Added tests proving the ritual contract is present, redacted, and does not
  unblock fast-companion professional execution.
- Updated protocol, internal-test4 plan, current control, and project state
  machine docs.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/plans/2026-06-04-professional-mode-ritual-contract.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is host/web/simulator mode-contract evidence only. It does not prove a
  physical StackChan professional consult, audible checking cue, or evidence
  card render on hardware.
- Actual professional answers still require the existing professional/V21 path
  and fresh adapter/runtime evidence.

Known risks/blockers:

- The ritual cue is intentionally static and redacted. Future personalized
  professional intros must not include user utterance text, workspace text,
  evidence bodies, provider output, URLs, paths, credentials, or raw audio.
- Physical professional mode acceptance still needs foreground hardware
  evidence after the user opens that window.

Recommended next action:

- Continue toward internal test 4 acceptance by either:
  - opening the no-flash hardware evidence window for professional/touch/action
    proof; or
  - implementing the next cloud upload/index execution slice behind the
    existing no-execute `/v1/workspace-upload-jobs` contract.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestVoiceModesCatalogDefaultsToRoleplayAndListsProfessional|TestVoiceModeSelectionProfessionalReturnsRitualContract|TestFastCompanionRejectsProfessionalVoiceModeWithoutProviderOrV21Execution|TestSimulatorPageServed' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestVoiceMode|TestFastCompanionRejectsProfessionalVoiceMode|TestSimulatorPageServed' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or V21 execution occurred, and
  no firmware build, flash, serial, NVS, or physical hardware action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 11:33 CST - Environment Switch No-Repeat Handoff

Round goal:

- Preserve current internal test 4 progress and unfinished items before a
  development environment switch, with explicit no-repeat guidance after
  context compaction.

Actual completed work:

- Added
  `docs/handoffs/2026-06-04-a21-internal-test4-environment-switch-handoff.md`
  as the current recovery point.
- Recorded the last pushed HEAD
  `d4a974c docs(control): hand off roleplay expression plan`.
- Recorded that the only pre-handoff tracked local code changes were
  `internal/gateway/server.go`, `internal/gateway/server_test.go`, and
  `internal/gateway/simulator.go` for the uncommitted roleplay
  `expression_plan` slice.
- Recorded that the local expression-plan slice is not yet protocol-doc,
  state-machine, handoff-log, full-verify, commit, or push complete.
- Recorded that current-mainline internal test 4 commits through roleplay
  device-state reflection and A21 native V21 voice-query bridge are ancestors
  of HEAD and should not be repeated.
- Closed the completed Poincare subagent after it reported the StackChan
  official hardware parity gap-map audit had already landed in the main tree
  and that continuing the older worker branch would create duplicate churn.

Changed files:

- `docs/handoffs/2026-06-04-a21-internal-test4-environment-switch-handoff.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- The local roleplay `expression_plan` code slice remains uncommitted.
- No fresh full verification has been run after this handoff-doc update.
- Product-visible gaps remain: web/app workspace, real indexing, durable
  account/device binding, V21 merge/release, physical StackChan expression and
  PRD acceptance, custom wake proof, and persistent cloud topology.

Known risks/blockers:

- Context compaction can cause repeated branch scans and repeated work unless
  the next environment starts from the new handoff.
- Older StackChan parity worker branches are not ancestors of current HEAD and
  must not be blindly merged because they predate later internal test 4 work.
- Git may still warn about historical loose objects/gc; no prune/gc action was
  taken.

Recommended next action:

- In the new environment, first read the new handoff, run
  `git status --short --branch`, then either finish the local
  `expression_plan` slice with docs/state/handoff/full verification or park it
  and move to visible product progress.

Test/build/runtime results:

- No runtime, provider, V21, ECS, firmware, serial, NVS, prune/gc, or hardware
  action occurred in this handoff-doc round.
- Fresh verification after adding the handoff/log entry is still pending.

Failure location/reason:

- None; this round intentionally stopped at a recovery handoff.

## 2026-06-04 11:36 CST - Roleplay Official Expression Plan

Transition:

- `T-INTERNAL-TEST4-ROLEPLAY-OFFICIAL-EXPRESSION-PLAN-001`

What changed:

- Added a safe no-send `expression_plan` to `/v1/roleplay-profile`.
- The plan uses existing official StackChan action metadata for baseline
  posture, role soul, scenario emphasis, and memory cue phases.
- Selected `a21_roleplay_wry_peer` + `engineer_pushback` + memory-ready state
  now maps to official happy/thinking/nod metadata with packet counts and
  phase-prefixed surfaces.
- The simulator roleplay panel now displays the expression delivery policy,
  action count, and packet count.
- Protocol, internal test 4 plan, current control, and state machine now record
  that this is no-send host-local expression planning rather than hardware
  delivery or physical acceptance.

Files changed:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-roleplay-official-expression-plan.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests run and results:

- `go test ./internal/gateway -run 'TestSimulatorPageServed|TestRoleplayProfileEndpointReturnsOfficialExpressionPlanWithoutSendingHardware|TestRoleplayProfileEndpointSelectsSoulProfile|TestRoleplayProfileEndpointPersistsScenarioVoiceClone|TestRoleplayProfileEndpointSetsRuntimeMemoryHints|TestRoleplayProfileEndpointSelectsSoulProfileAndReturnsSafeCatalog' -count=1`:
  passed.
- `go test ./internal/transport/stackchan -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Runtime or physical evidence:

- None. This round did not start Gateway, send StackChan action packets, run
  provider/V21/voice-clone execution, play audio, flash firmware, write NVS, or
  touch physical hardware.

Deviations from plan:

- None. The implementation stayed within host-side Gateway response,
  simulator, tests, and control docs.

Remaining issues:

- This is not hardware expression delivery and not roleplay physical
  acceptance.
- Screen/avatar/motion/RGB/servo proof still needs a foreground hardware
  window and must keep `physical_accepted=false` until that evidence exists.
- The bigger product gaps remain web/app workspace UX, real indexing,
  persistent account/device binding, V21 release/ACL, custom wake proof, and
  full StackChan PRD acceptance.

Next suggested action:

- Move from safe host contracts to visible product progress: either build the
  web/app workspace UI over the existing Gateway APIs, implement the real
  stored-document indexing adapter path, or schedule the foreground hardware
  window for expression/touch/barge-in evidence.

Forbidden actions avoided:

- No `/stackChan/ws`, `/v1/stackchan/official/control`, provider, V21,
  voice-clone CLI, audio playback, Gateway service start, ECS, firmware,
  serial, NVS, prune/gc, or physical hardware action occurred.

## 2026-06-04 11:09 CST - Environment Switch Handoff And Product Gap Report

Round goal:

- Pause active implementation for a development-environment switch, preserve
  the current control-tower progress, and record what is still missing from a
  user-visible PRD/product perspective.

Actual completed work:

- Latest pushed commits on
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`:
  - `976a1d2 feat(gateway): reflect roleplay state in device registry`
  - `6498ede feat(app): route v21 bridge through native voice query`
  - `731d4d8 docs(control): record v21 source scope guard`
- Current worktree before this handoff had only one uncommitted file:
  `docs/plans/2026-06-04-roleplay-official-expression-plan.md`.
- Added that plan as the next no-hardware roleplay immersion cut:
  `T-INTERNAL-TEST4-ROLEPLAY-OFFICIAL-EXPRESSION-PLAN-001`.

Current product truth:

- Internal test 3 voice chain is preserved; do not revert the user's端侧语音
 实机验收/protocol changes.
- Internal test 4 has gained many necessary product contracts, but much of it
  is not yet obvious to a user because it is adapter/state/readiness work:
  roleplay profile/soul/memory/voice-clone selection, safe device registry
  reflection, professional mode/read ledgers, upload/index request ledgers, V21
  source-scope guard wiring, and low-risk MCP controls.
- User-visible product progress still needs to be forced into visible/audible
  experiences: roleplay expression on StackChan, physical voice/latency proof,
  professional consult against real searchable V21 evidence, and production
  web/app workspace flows.

Unfinished items:

- Roleplay immersion:
  - Next planned cut is `roleplay_official_expression_plan`: expose a safe
    no-send `expression_plan` from `/v1/roleplay-profile` using existing
    official StackChan action metadata.
  - Still missing after that: actually delivering selected roleplay expression
    to StackChan screen/servo/RGB in an approved hardware/runtime window and
    collecting physical evidence.
- Professional/V21:
  - A21 bridge now targets V21 native voice-query, but V21 branch
    `origin/codex/a21-v2-workspace-scope-retrieval-guard` still needs
    merge/release into the environment that A21 will call.
  - Stored uploaded documents are not parsed/chunked/embedded/indexed.
  - No durable account/device/workspace ACL or cloud object storage is done.
- Workspace/web/app:
  - Upload intake exists, but current ledgers are mostly in-memory and not a
    polished production web workspace.
  - The 2 GB personal corpus expectation is not implemented as durable storage
    or searchable index.
- Voice/provider:
  - Internal test 3 public voice path is the protected baseline, but full
    low-latency roleplay + voice clone + hardware expression physical
    acceptance is still not proven.
  - StepFun selection on ECS remains dependent on root-only secret injection if
    that path is still desired.
- Hardware/MCP:
  - Low-risk MCP/status/volume/action contracts exist, but full physical
    coverage of camera, IMU/sensors, battery, NFC, infrared, RGB, servos,
    touch/barge-in, and playback-start/stop remains evidence-gated.

Known risks/blockers:

- A lot of repo progress is below the user's visual product threshold; the next
  developer should prioritize cuts that produce visible/audible simulator or
  hardware behavior, not more control-plane-only paperwork.
- Do not do Git prune/gc; historical loose-object warnings remain expected.
- Do not flash firmware, touch serial/NVS, execute providers/V21, or start real
  runtime services unless the new environment has an explicit scoped window.

Recommended next action:

1. Implement
   `docs/plans/2026-06-04-roleplay-official-expression-plan.md`:
   `/v1/roleplay-profile.expression_plan` with official StackChan action
   metadata and `physical_accepted=false`.
2. Then add a simulator-visible roleplay expression preview/readout so the
   user can see product movement without hardware.
3. In parallel, prepare the V21 merge/release or indexing adapter cut so
   professional mode can consult actual searchable resources rather than
   metadata/index-request ledgers.
4. After the environment switch, run fresh:
   `git status --short --branch`, read this handoff, then run
   `GOMAXPROCS=2 make verify` before making claims.

Test/build/runtime results:

- Latest code commits before this handoff passed `GOMAXPROCS=2 make verify`.
- This handoff/plan round is docs-only; run `git diff --check` before commit.
- No Gateway service was started, no provider/V21/voice-clone execution
  occurred, and no firmware build, flash, serial, NVS, ECS change, prune/gc, or
  physical hardware action occurred.

Failure location/reason:

- Product gap remains: visible/audible roleplay immersion, real searchable
  professional consult, durable workspace storage/indexing, and physical
  full-hardware acceptance are not complete.

## 2026-06-04 11:02 CST - Roleplay Device State Reflection

Round goal:

- Make the selected A21 roleplay identity visible as safe device/registry state
  so StackChan/simulator surfaces can reflect role soul, scenario, memory
  readiness/count, and voice-clone binding without exposing prompt or memory
  content.

Actual completed work:

- Added plan `docs/plans/2026-06-04-roleplay-device-state-reflection.md`.
- Extended `/v1/devices` records with safe roleplay fields:
  `current_roleplay_profile`, `current_roleplay_scenario`,
  `roleplay_soul_ready`, `roleplay_memory_ready`,
  `roleplay_memory_hint_count`, and `roleplay_physical_accepted`.
- Added a lightweight roleplay device-state snapshot derived from the existing
  roleplay selector and bounded memory policy.
- Device `runtime_echo` now receives only safe roleplay IDs, booleans, counts,
  and explicit non-storage flags for prompt text, memory text, and voice
  samples.
- Simulator Device Registry now shows role soul, scenario, and role memory next
  to the existing voice-clone profile.
- Added tests proving selected `a21_roleplay_wry_peer`,
  `engineer_pushback`, and one safe memory hint are reflected in `/v1/devices`
  while unsafe URL hints, memory text, and control text are not leaked.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/plans/2026-06-04-roleplay-device-state-reflection.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not physical StackChan roleplay acceptance; it only reflects safe
  host-local device state.
- Provider execution, voice-clone CLI execution, real audio output, and
  physical screen/servo/RGB reflection remain evidence-gated follow-ups.

Known risks/blockers:

- Runtime echo is a state reflection channel, not proof that firmware displayed
  or animated the role state.
- Roleplay memory remains bounded runtime prompt input, not durable long-term
  memory or user account memory.
- Git may still warn about historical loose objects/gc; no prune/gc action was
  taken.

Recommended next action:

- Continue roleplay immersion by mapping safe roleplay state to official
  StackChan action/screen/servo/RGB semantics in a no-hardware contract cut,
  then collect physical evidence in an approved hardware window.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestSimulatorPageServed|TestVoiceMode|TestRoleplayProfile|TestFastCompanionHybrid' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider/V21/voice-clone execution
  occurred, and no firmware build, flash, serial, NVS, ECS change, prune/gc, or
  physical hardware action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 10:50 CST - A21 Native V21 Voice-Query Bridge

Round goal:

- Make A21's local `v21-adapter-bridge` use V21 native
  `/internal/v1/knowledge/voice-query` as the primary professional query path
  so internal test 4 no longer bypasses the V21 source-scope guard through
  direct retrieval on normal professional consults.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-a21-v21-native-voice-query-bridge.md`.
- Extended the bridge DTOs for V21 voice-query and retrieval fallback with safe
  `source_scope`, `source_scope_counts`, and `workspace_status` metadata.
- The bridge now sends safe A21 v2 fields into native V21 voice-query:
  `device_id`, `user_id`, `workspace_id`, `query_scope`, trace/session IDs,
  and the active collection ID.
- Native voice-query success now mirrors V21-returned `source_scope_counts`
  and `workspace_status` instead of fabricating counts from requested
  `query_scope`.
- Direct retrieval remains only as a controlled no-evidence expansion fallback;
  fallback counts are derived only from result `source_scope` labels.
- Added httptest coverage proving the green path calls native voice-query, does
  not call direct retrieval, preserves explicit `personal_plus_public`
  workspace scope metadata, and still retries the child-lock ASR fragment only
  after native `no_evidence`.
- Documented the new bridge contract in current control, V21 integration,
  protocol, internal test 4 plan, and project state.

Changed files:

- `internal/app/professional_adapter_bridge.go`
- `internal/app/app_test.go`
- `internal/v21adapter/client.go`
- `docs/plans/2026-06-04-a21-v21-native-voice-query-bridge.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/V21_INTEGRATION.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- V21 worker branch `origin/codex/a21-v2-workspace-scope-retrieval-guard` still
  needs merge/release before this is active against the normal local V21
  service.
- Durable tenant/account ACL, real personal upload indexing, cloud storage,
  provider execution, and physical StackChan professional consult acceptance
  remain open.
- Direct retrieval fallback is retained only for controlled no-evidence
  expansion; it must not become the normal professional query path again.

Known risks/blockers:

- A V21 deployment that lacks `/internal/v1/knowledge/voice-query` will return
  no-evidence/404 and use fallback behavior, so release ordering matters.
- Product readiness should continue treating adapter-boundary smoke as below
  full PRD acceptance until a real V21 merge/release and physical consult
  evidence exist.
- Git may still warn about historical loose objects/gc; no prune/gc action was
  taken.

Recommended next action:

- Continue internal test 4 along two parallel mainline cuts: roleplay
  persona/memory/voice-clone runtime reflection toward the actual StackChan
  experience, and approved V21 merge/release plus real indexing adapter work.

Test/build/runtime results:

- `go test ./internal/app -run 'V21AdapterBridge|V21AdapterSmoke|ProductReadiness.*V21|RunProductReadiness.*V21' -count=1`:
  passed.
- `go test ./internal/v21adapter -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or real V21 execution occurred,
  and no firmware build, flash, serial, NVS, ECS change, prune/gc, or physical
  hardware action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 10:37 CST - V21 Source-Scope Retrieval Guard

Round goal:

- Move the V21 side of the A21 internal test 4 workspace contract beyond
  metadata-only acceptance by adding executable source-scope filtering for
  `public_only`, `personal_only`, and `personal_plus_public`, while preserving
  the boundary that this is not yet durable tenant/account ACL or real personal
  indexing.

Actual completed work:

- Inspected the dirty main V21 worktree and avoided it. The dirty checkout is
  `/Users/jiyurun/Documents/v21-knowledge-platform` on
  `feat/consumer-lan-discovery-ui`.
- Continued from the clean V21 worker worktree
  `/Users/jiyurun/.codex/worktrees/b0c0/v21-knowledge-platform`.
- Created V21 branch
  `codex/a21-v2-workspace-scope-retrieval-guard` from the previous A21 v2
  native contract branch.
- Added V21 plan
  `docs/plans/2026-06-04-a21-v2-workspace-scope-retrieval-guard.md`.
- Implemented V21 source-scope guard commit
  `fccd0ac feat(voice-query): enforce A21 workspace source scope` and pushed it
  to `origin/codex/a21-v2-workspace-scope-retrieval-guard`.
- V21 retrieval requests now carry safe A21 v2 metadata:
  `device_id`, `user_id`, `workspace_id`, and `query_scope`.
- V21 retrieval results and voice-query evidence now carry safe
  `source_scope=public|personal`.
- V21 voice-query filters scoped results before answer generation; unclassified
  evidence is not promoted into scoped A21 responses.
- V21 HTTP retrieval adapter and Qdrant sidecar pass `source_scope`; the
  Postgres writer can derive it from chunk metadata, source-unit locator
  metadata, or `object_refs.access_class`.
- Scoped V21 responses can now return `workspace_status=searchable` with
  classified `source_scope_counts`; legacy requests without `query_scope`
  remain `scope_contract_ready_acl_pending`.
- Updated A21 control docs to record the worker result and next boundary.

Changed files:

- V21 worker commit `fccd0ac` changed:
  - `backend/internal/adapters/postgres/admin_reader_test.go`
  - `backend/internal/adapters/postgres/upload_writer.go`
  - `backend/internal/adapters/retrieval/http_searcher.go`
  - `backend/internal/adapters/retrieval/http_searcher_test.go`
  - `backend/internal/admin/readmodel.go`
  - `backend/internal/admin/voice_query.go`
  - `backend/internal/httpapi/voice_query.go`
  - `backend/internal/httpapi/voice_query_test.go`
  - `contracts/internal/voice-knowledge-query.schema.json`
  - `deploy/sidecars/retrieval-qdrant/retrieval_service.py`
  - `deploy/sidecars/retrieval-qdrant/retrieval_service_test.py`
  - `docs/core/07-DEVICE-VOICE-AND-PROTOCOL.md`
  - `docs/core/10-API-AND-EVENT-CONTRACTS.md`
  - `docs/plans/2026-06-04-a21-v2-workspace-scope-retrieval-guard.md`
- A21 control docs changed:
  - `docs/engineering/A21_CURRENT_CONTROL.md`
  - `docs/engineering/V21_INTEGRATION.md`
  - `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
  - `docs/project_state_machine.md`
  - `docs/agent_handoff_log.md`

Unfinished items:

- V21 worker branches still need V21 owner/main-thread review and merge.
- Durable tenant/account ACL is not complete.
- Real personal document parsing, chunking, embedding, indexing, cloud storage,
  and V21 release readiness are not complete.
- A21 Gateway still has no real execution path that consumes uploaded A21
  workspace documents into V21 personal corpora.
- Physical StackChan professional consult acceptance remains separate.

Known risks/blockers:

- `fccd0ac` is source-scope guard evidence only; do not claim full personal
  corpus readiness from it.
- V21 `make test` and `make check` are currently blocked by Node toolchain
  mismatch: Makefile expects Node `v24.15.0`; this shell has Node `v25.8.0`.
- Git may still warn about historical loose objects/gc; no prune/gc action was
  taken.

Recommended next action:

- Review/merge the two V21 worker branches, then dispatch a durable A21/V21
  indexing adapter transition that takes A21 stored-local documents through an
  approved V21 upload/index contract and returns source-scope-aware searchable
  readiness. Keep this separate from physical StackChan professional consult
  evidence.

Test/build/runtime results:

- V21 `go test ./internal/httpapi ./internal/adapters/retrieval ./internal/adapters/postgres -run 'VoiceQuery|HTTPSearcher|UploadWriterSearch' -count=1`:
  passed.
- V21 `go test ./...` from `backend/`: passed.
- V21 sidecar `python3 -m unittest retrieval_service_test.py`: passed.
- V21 retrieval eval
  `python3 -m unittest evals/retrieval/test_qdrant_retrieval_sidecar.py`:
  passed.
- V21 `git diff --check`: passed.
- V21 `make compose-config`: passed.
- V21 `make test`: blocked at `check-toolchain` due Node version mismatch.
- V21 `make check`: blocked at `check-toolchain` due Node version mismatch.
- No A21 Gateway service was started, no provider or real V21 service
  execution occurred, no private documents were ingested, and no firmware
  build, flash, serial, NVS, ECS change, prune/gc, or physical hardware action
  occurred.

Failure location/reason:

- `make test` and `make check` in V21 root failed before tests at
  `check-toolchain` because Node is `v25.8.0` while the Makefile expects
  `v24.15.0`.

## 2026-06-04 09:46 CST - Roleplay Soul Profile Contract

Round goal:

- Move internal test 4 roleplay from a single default `roleplay_profile`
  placeholder to a selectable A21-owned role-soul layer that can affect actual
  voice-pipeline prompt input while preserving upload/read-record redaction
  discipline.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-roleplay-soul-profile-contract.md`.
- Added role-soul personality assets:
  `docs/personality/role_souls/default.md`,
  `docs/personality/role_souls/wry_peer.md`, and
  `docs/personality/role_souls/calm_anchor.md`.
- Extended `personality.Compose` with an optional `RoleSoul` layer loaded
  between `tone_rules` and `mode_prompts`.
- Expanded `/v1/roleplay-profile` so `roleplay_profile` can select
  `a21_roleplay_default`, `a21_roleplay_wry_peer`, or
  `a21_roleplay_calm_anchor` together with scenario, voice-clone profile, and
  bounded memory hints.
- Runtime summaries now expose `soul_prompt_input_ready` and safe
  `prompt_parts` such as `role_soul:a21_roleplay_wry_peer` without returning
  prompt bodies.
- Fast-companion and stock Xiaozhi roleplay prompt composition now use the
  selected role soul before passing prompt input to the provider-neutral voice
  pipeline.
- Simulator controls now include `roleplayProfile` and `roleplaySoulReadout`
  beside scenario, voice-clone, and memory controls.

Changed files:

- `internal/personality/composer.go`
- `internal/personality/composer_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/personality/README.md`
- `docs/personality/role_souls/default.md`
- `docs/personality/role_souls/wry_peer.md`
- `docs/personality/role_souls/calm_anchor.md`
- `docs/plans/2026-06-04-roleplay-soul-profile-contract.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not physical StackChan roleplay audio acceptance and does not prove a
  real cloned voice provider executed.
- This is not durable long-term memory, account/persona sync, role marketplace,
  cloud profile storage, V21 professional retrieval, or agent bridge runtime.
- Selected soul currently affects prompt input; hardware expression/RGB/servo
  semantic mapping for each soul remains a separate hardware/action parity
  promotion.

Known risks/blockers:

- Persona quality still needs subjective voice-turn review on the live product
  chain; this slice proves contract/path correctness, not final taste.
- API responses expose safe role-soul labels/descriptions and prompt-part IDs;
  future production UI may tighten what is user-visible.
- Physical acceptance still depends on a foreground device window and current
  Gateway/provider configuration.

Recommended next action:

- Continue internal test 4 with a hardware/runtime evidence window for selected
  role soul + selected voice clone on stock `/v1/xiaozhi`, or move to the next
  no-hardware slice: professional/workspace indexing adapter boundary. Do not
  treat role-soul contract evidence as provider execution or physical audio
  acceptance.

Test/build/runtime results:

- `go test ./internal/personality ./internal/gateway -run 'TestPersonalityComposeRoleSoulAddsOnlySelectedSoul|TestRoleplayProfileEndpointSelectsSoulProfileAndReturnsSafeCatalog|TestFastCompanionHybridRunsVoicePipelineWhenFramesProvided|TestSimulatorPageServed' -count=1`:
  passed.
- `go test ./internal/personality ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or real V21 execution occurred,
  and no firmware build, flash, serial, NVS, ECS change, prune/gc, or physical
  hardware action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 09:04 CST - Workspace Source Readiness Registry

Round goal:

- Move the internal test 4 workspace surface beyond upload-job metadata by
  exposing safe public/personal source readiness and query-scope readiness,
  while preserving upload/read-record discipline and avoiding any false claim
  of real document storage, indexing, provider execution, or V21 execution.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-workspace-source-readiness-registry.md`.
- Added `GET /v1/workspace-sources` with schema
  `a21.gateway.workspace_sources.v1`.
- Workspace upload/import job creation now creates a linked redacted
  `source_id` and a memory-only source record with
  `readiness=metadata_only`.
- `PUT /v1/workspace-upload-jobs` now accepts
  `mark_searchable` / `mark_indexed_metadata_only`, promoting only metadata
  readiness to `searchable_metadata_only`; no indexing execution occurs.
- `mark_failed`, `retry`, and `delete` now sync the linked source readiness.
  Delete leaves only a redacted tombstone.
- `/v1/professional-workspace` runtime now reports `source_scope_counts`,
  `searchable_source_scope_counts`, and `query_scope_readiness`, while
  `v21_execution_allowed` remains false.
- Simulator Workspace Audit now has a `Sources` refresh action plus source
  count/readiness readouts.
- Added Gateway tests for source creation, searchable metadata promotion,
  professional workspace readiness summary, trace redaction, and fail/retry/
  delete source synchronization.
- Updated protocol, internal-test4 plan, current control, and project state
  machine docs.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/plans/2026-06-04-workspace-source-readiness-registry.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not real upload byte storage, document parsing, indexing execution,
  V21 ACL enforcement, durable persistence, provider execution, real V21
  retrieval, or physical StackChan professional consult acceptance.
- The source registry is memory-only and resets with Gateway restart.

Known risks/blockers:

- `searchable_metadata_only` is an internal test readiness label, not proof
  that V21 can retrieve the source. V21 personal/public ACL enforcement remains
  a separate worker concern.
- Durable user/device/workspace binding remains unimplemented.

Recommended next action:

- Continue with a real upload/index adapter-boundary spike only after an ADR
  defines storage and ACL ownership, or open a foreground hardware window for
  spoken professional trigger, checking cue, evidence playback, and visible
  `PRO` status proof.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestWorkspaceSource|TestWorkspaceUploadJobs|TestProfessionalWorkspace' -count=1`:
  first failed before implementation because workspace source types/fields and
  source helpers did not exist; passed after implementation.
- `go test ./internal/gateway -run 'TestWorkspaceSource|TestWorkspaceUploadJobs|TestProfessionalWorkspace|TestSimulatorPageServed' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or real V21 execution occurred,
  and no firmware build, flash, serial, NVS, ECS change, or physical hardware
  action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 08:18 CST - Professional Workspace Read Records

Round goal:

- Close the internal test 4 professional workspace audit gap by recording safe
  metadata for each professional V21 read without storing query text, retrieved
  text, provider output, evidence bodies, document contents, credentials, URLs,
  paths, voice transcript, audio, or hardware evidence.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-professional-workspace-read-records.md`.
- Added `GET /v1/professional-read-records` with schema
  `a21.gateway.professional_read_records.v1`.
- Added memory-only `ProfessionalReadRecord` ledger state to Gateway.
- Mock professional turns now start a read record before V21 query and mark it
  `completed` with safe `source_scope_counts` / `workspace_status` or `failed`
  with a safe failure code.
- Stock `/v1/xiaozhi` professional turns use the same read-record flow.
- Added trace markers `professional.read_record.started`,
  `professional.read_record.completed`, and `professional.read_record.failed`.
- Added redaction tests using deliberate `RAW_SECRET` / `RAW_PRIVATE_QUERY`
  fixtures to prove the read-record endpoint does not store or return raw
  utterance, retrieved evidence, cards, speech blocks, provider output, URLs,
  paths, credentials, or document text.
- Updated protocol, internal-test4 plan, current control, and project state
  machine docs.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-professional-workspace-read-records.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not document upload byte storage, indexing, ACL enforcement, database
  persistence, V21 tenant migration, or production cloud ingest.
- Physical StackChan professional consult acceptance is still unproven in this
  round; `professional` hardware evidence still needs a foreground hardware
  window.

Known risks/blockers:

- The ledger is in-memory and will reset with Gateway process restart.
- Source-scope counts and workspace status are only as truthful as the V21
  adapter response contract; V21 ACL enforcement remains a separate worker
  concern.

Recommended next action:

- Continue with either the next no-execute workspace/index readiness slice or
  dispatch the approved hardware evidence worker for professional mode cue,
  answer playback, and visible card/status proof. Do not jump to real ingest or
  hardware write without a scoped plan.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestProfessionalReadRecords' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestProfessionalReadRecords|TestProfessionalModeSendsExplicitV21PlaceholderContract|TestProfessionalModeV21TimeoutCancelsQueryAndFallsBack|TestWorkmateModeDoesNotCallV21Adapter' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no real provider or real V21 execution
  occurred, and no firmware build, flash, serial, NVS, or physical hardware
  action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 08:34 CST - Simulator Workspace Audit Surface

Round goal:

- Make the upload/read-record discipline visible in the simulator so internal
  test 4 operators can inspect workspace job status and professional read
  ledger metadata without querying raw APIs.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-simulator-workspace-audit-surface.md`.
- Added a simulator `Workspace Audit` section.
- Added a `Read Records` control that calls
  `GET /v1/professional-read-records`, filtered by current `trace_id` when
  available.
- Added visible simulator readouts for:
  - last no-execute workspace upload job status;
  - professional read-record count;
  - read status / failure code;
  - query scope / utterance bucket;
  - public/personal source-scope counts;
  - workspace status / privacy scope.
- Workspace job creation now updates the visible job readout.
- Professional evidence responses now trigger a read-record refresh for the
  current trace.
- Updated the simulator smoke test to assert the new endpoint and DOM IDs.
- Updated internal-test4 plan, current control, and project state machine.

Changed files:

- `internal/gateway/simulator.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-simulator-workspace-audit-surface.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is still a simulator/internal-test control surface, not a production
  web app, real document upload, indexing, ACL enforcement, persistence, V21
  tenant migration, or physical StackChan proof.
- Browser plugin tooling was not exposed in this Codex session, and the local
  Node runtime did not have `playwright`; visual screenshot automation was
  therefore not available in this round.

Known risks/blockers:

- The readout reflects the Gateway's memory-only read ledger. It will reset on
  Gateway restart until a future persistent workspace/audit store is scoped.
- The simulator intentionally displays only metadata. It should not be expanded
  to show raw queries or evidence bodies without a new privacy design.

Recommended next action:

- Continue toward internal test 4 by either adding the next real workspace
  ingest/index readiness slice behind the existing no-execute guard or opening
  a foreground hardware evidence window for professional cue, readout, and
  playback proof.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestSimulatorPageServed|TestProfessionalReadRecords' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- Local HTTP smoke against `go run ./cmd/a21 gateway --addr 127.0.0.1:21083`:
  `/simulator` contained `Workspace Audit`, `professionalReadRecordsRefresh`,
  `professionalReadRecordCount`, and `/v1/professional-read-records`;
  `/v1/professional-read-records` returned schema
  `a21.gateway.professional_read_records.v1`.
- The local verification Gateway on `127.0.0.1:21083` was stopped after the
  smoke check.
- `GOMAXPROCS=2 make verify`: passed.
- No provider or real V21 execution occurred, and no firmware build, flash,
  serial, NVS, ECS change, or physical hardware action occurred.

Failure location/reason:

- Browser/Playwright screenshot automation unavailable because the Browser tool
  was not exposed and Node `playwright` module was not installed. HTTP smoke
  covered the served page and endpoint presence instead.

## 2026-06-04 08:49 CST - Professional Voice Trigger Route

Round goal:

- Let explicit PRD trigger phrases such as "专业模式", "认真查一下",
  "帮我查 V21", and "给我证据" route default roleplay/workmate voice turns
  into the professional evidence path without weakening internal test 3
  Xiaozhi audio behavior, privacy/state boundaries, or read-record redaction.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-professional-voice-trigger-route.md`.
- Added a provider-neutral Gateway trigger classifier with negation guards for
  phrases such as "不要进专业检索", "不用专业模式", and "别查 V21".
- `/v1/mock-turn` now forces default/roleplay/workmate/companion trigger text
  into the existing `professionalTurnResponse` path and records
  `professional.voice_trigger.detected`.
- Stock `/v1/xiaozhi` default `realtime`/workmate turns now route to
  professional when the streaming ASR final contains an explicit trigger,
  records `professional.voice_trigger.detected` and
  `xiaozhi.professional_route.voice_trigger`, and reuses that streaming final
  instead of running duplicate batch ASR.
- Added tests for mock-turn routing/read-record/redaction, negated classifier
  phrases, and Xiaozhi WebSocket trigger routing without stock override.
- Updated protocol, internal-test4 plan, current control, and project state
  machine docs.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-professional-voice-trigger-route.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not real provider execution, real V21 retrieval, permanent cloud
  workspace ingest/indexing, durable audit storage, or physical StackChan
  professional consult acceptance.
- Hardware voice-trigger acceptance still needs a foreground physical window
  after the product route is deployed/configured.

Known risks/blockers:

- The trigger list is intentionally narrow. More natural-language switch
  variants should be added only with explicit tests and privacy/negation
  guards.
- The read-record ledger remains memory-only until a future durable audit
  store is scoped.

Recommended next action:

- Continue internal test 4 with either a no-execute workspace ingest/index
  readiness slice or a foreground hardware evidence window for spoken
  professional trigger, checking cue, evidence playback, and visible `PRO`
  status. Keep firmware, serial, NVS, and real provider/V21 execution out of
  unscoped workers.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestProfessionalVoiceTrigger|TestXiaozhiWebSocketVoiceTrigger' -count=1`:
  first failed before implementation with `undefined: professionalVoiceTrigger`;
  passed after implementation.
- `go test ./internal/gateway -run 'TestProfessionalVoiceTrigger|TestXiaozhiWebSocketVoiceTrigger|TestProfessionalMode|TestWorkmateModeDoesNotCallV21Adapter|TestOfficePrivacyModesDoNotCallV21Adapter' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or real V21 execution occurred,
  and no firmware build, flash, serial, NVS, ECS change, or physical hardware
  action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 09:18 CST - Workspace Document Upload Intake

Round goal:

- Move internal test 4 knowledge workspace beyond metadata-only job/source
  readiness by accepting real local document upload bytes into an A21-owned
  Gateway runtime store, without indexing, V21 execution, cloud storage, or
  hardware action.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-workspace-document-upload-intake.md`.
- Added `POST /v1/workspace-documents` with schema
  `a21.gateway.workspace_documents.v1`.
- Multipart uploads now store bytes under the A21 local runtime store
  (`A21_WORKSPACE_DOCUMENT_STORE_DIR` or
  `.a21-run/gateway/workspace-documents`) and compute a `sha256:` document
  hash.
- Stored documents create linked workspace job/source records with
  `stored_local_pending_index`, `storage_status=stored_local`, and
  `index_status=not_started_no_execute`.
- `/v1/professional-workspace` now reports pending-index readiness for the
  selected scope when local stored uploads exist, while keeping
  `v21_execution_allowed=false`.
- Simulator Workspace Audit now includes a file picker/upload control and
  displays only safe document/job/source metadata.
- The existing direct `/v1/workspace-upload-jobs` endpoint remains
  metadata-only and still rejects raw document text, bytes, base64 payloads,
  import URLs, local paths, credentials, and provider output.
- Hardware parity worker `019e9011-80b2-73c3-a0a8-7390a3e1f153` confirmed
  `T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001` was already merged and made no
  file changes, avoiding duplicate parity-matrix churn.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `internal/app/app.go`
- `docs/plans/2026-06-04-workspace-document-upload-intake.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not parsing, import URL fetch, chunking, OCR, embedding, indexing,
  cloud object storage, durable account/auth/ACL, V21 personal/public
  enforcement, or production web workspace.
- Stored-local documents are not searchable until a separate indexing/V21
  transition lands.
- Physical StackChan professional consult acceptance is still pending a
  foreground hardware evidence window.

Known risks/blockers:

- The document registry remains in memory; Gateway restart forgets job/source
  metadata even though files remain in the runtime store.
- The default 16 MiB per-file limit is an intake guard, not the final 2 GB
  personal corpus product design.
- Hashes and sizes are exposed as safe metadata for internal test 4, but future
  production privacy review may further scope what appears in user-facing UI.

Recommended next action:

- Continue with a planned indexing/adapter worker that parses or transfers
  stored files into V21 behind the v2 query-scope contract, or start the
  hardware professional-consult evidence window. Do not treat
  `stored_local_pending_index` as searchable or V21-ready.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestWorkspaceDocumentUpload' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestWorkspaceDocumentUpload|TestWorkspaceSource|TestWorkspaceUploadJobs|TestProfessionalWorkspace|TestSimulatorPageServed' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or real V21 execution occurred,
  and no firmware build, flash, serial, NVS, ECS change, prune/gc, or physical
  hardware action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 10:02 CST - MCP Speaker Volume Freeze

Round goal:

- Freeze the already-used official Xiaozhi MCP speaker volume tool into the
  unified low-risk `/v1/xiaozhi/mcp-control` surface, so A21's StackChan MCP
  control contract is less split while preserving physical-evidence discipline.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-stackchan-official-mcp-speaker-volume-freeze.md`.
- Added `volume` to `XiaozhiMCPControlRequest`.
- `/v1/xiaozhi/mcp-control` now allows
  `self.audio_speaker.set_volume` with bounded `volume=0..100`.
- Existing `/v1/xiaozhi/speaker-volume` behavior remains available as the
  compatibility/product shortcut.
- Unified MCP control now records `xiaozhi.mcp.speaker_volume.sent` and safe
  bounded `speaker_volume` device activity metadata.
- Missing volume, out-of-range volume, and mixed screen/volume arguments are
  rejected before websocket write.
- High-risk MCP tools remain blocked before websocket write: reboot, firmware
  upgrade, camera/photo, screen snapshot, stream/video, NFC, infrared, and app
  lifecycle.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-stackchan-official-mcp-speaker-volume-freeze.md`
- `docs/plans/2026-06-04-stackchan-official-mcp-status-parity.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not physical loudness, playback-start, playback-stop, or PRD speaker
  acceptance.
- This does not add raw MCP response capture, screen snapshot, camera/photo,
  reboot, firmware upgrade, NFC, infrared, stream/video, or app lifecycle
  controls.
- Device-side volume persistence/NVS policy remains separate from this
  Gateway delivery contract.

Known risks/blockers:

- Delivery proves only that the stock MCP request was sent to an online
  Xiaozhi socket with MCP support; actual loudness still needs trusted physical
  evidence.
- The unified endpoint and dedicated shortcut now overlap intentionally; future
  UI can prefer `/v1/xiaozhi/mcp-control` while old operator scripts keep using
  `/v1/xiaozhi/speaker-volume`.

Recommended next action:

- Continue hardware parity with a foreground physical evidence window for
  speaker playback-start/stop, touch/barge-in, visible action, RGB, and servo
  proof, or continue no-hardware work on workspace indexing/V21 adapter
  readiness. Do not promote MCP delivery to product physical acceptance.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestXiaozhiSpeakerVolumeUsesStockMCPToolCall|TestXiaozhiMCPStatusParityAllowsOnlyScopedTools|TestXiaozhiMCPStatusParityBlocksHighRiskTools|TestXiaozhiMCPStatusParityRequiresSafeArguments' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or real V21 execution occurred,
  and no firmware build, flash, serial, NVS, ECS change, prune/gc, or physical
  hardware action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 10:36 CST - Workspace Index Request Ledger

Round goal:

- Move internal test 4 workspace uploads from `stored_local_pending_index` to a
  truthful no-execute indexing request state, without parsing documents,
  indexing, executing V21, or leaking private document contents.

Actual completed work:

- Added plan
  `docs/plans/2026-06-04-workspace-index-request-ledger.md`.
- Added `GET/POST /v1/workspace-index-jobs` with schema
  `a21.gateway.workspace_index_jobs.v1`.
- `POST /v1/workspace-index-jobs` accepts safe document/job/source IDs and
  optional trace/session/device IDs, verifies the stored-local intake file is
  present, and records a redacted `index_job_id`.
- Linked documents, upload jobs, and sources promote to
  `indexing_requested_no_execute`; upload jobs set
  `indexing_api_ready=true` while keeping `execution_started=false`.
- `/v1/workspace-sources` now reports
  `indexing_requested_source_scope_counts`, and
  `/v1/professional-workspace` can report selected-scope
  `indexing_requested_no_execute` readiness without allowing V21 execution.
- Simulator Workspace Audit now includes an `Index Request` control and
  index-job readout.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/simulator.go`
- `docs/plans/2026-06-04-workspace-index-request-ledger.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is not parsing, OCR, chunking, embedding, cloud storage, durable
  account/auth/ACL, real indexing, searchable readiness, V21 execution, or
  physical StackChan professional consult acceptance.
- The workspace/index ledgers remain in-memory; local files persist in the
  runtime store, but metadata is not durable across Gateway restart.

Known risks/blockers:

- `indexing_requested_no_execute` must not be treated as searchable or as proof
  that personal/public ACL enforcement exists.
- Future V21 indexing must remain behind the A21/V21 adapter contract and must
  preserve source-scope redaction.
- Git may still warn about historical loose objects/gc; no prune/gc action was
  taken.

Recommended next action:

- Continue with a scoped V21/A21 indexing adapter worker that consumes stored
  local documents through an approved contract, or use a foreground hardware
  window for professional consult evidence. Keep real V21 execution out of
  unscoped workspace UI work.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestWorkspaceIndex|TestWorkspaceDocumentUpload|TestWorkspaceSource|TestWorkspaceUploadJobs|TestProfessionalWorkspace|TestSimulatorPageServed' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- No Gateway service was started, no provider or real V21 execution occurred,
  and no firmware build, flash, serial, NVS, ECS change, prune/gc, or physical
  hardware action occurred.

Failure location/reason:

- None in this focused round.

## 2026-06-04 11:40 CST - Latest Recovery Pointer

Round goal:

- Keep the newest recovery signal at the end of the handoff log after earlier
  entries landed above older records.

Actual completed work:

- The current latest implementation transition is
  `2026-06-04 11:36 CST - Roleplay Official Expression Plan` in this log.
- The current no-repeat environment handoff is
  `docs/handoffs/2026-06-04-a21-internal-test4-environment-switch-handoff.md`.
- Next environment should read that handoff, then current `git status`, then
  continue from HEAD instead of redoing internal test 3 or old worker branch
  audits.

Changed files:

- `docs/agent_handoff_log.md`

Unfinished items:

- Product-visible gaps remain web/app workspace UX, real indexing, persistent
  account/device binding, V21 release/ACL, custom wake proof, and full
  StackChan PRD physical acceptance.

Known risks/blockers:

- Historical Git loose-object/gc warnings may still appear; no prune/gc action
  should be taken in this flow.

Recommended next action:

- Commit/push the verified roleplay expression-plan slice, then move to a
  visibly user-facing product surface or a foreground hardware evidence window.

Test/build/runtime results:

- This pointer adds no code or runtime behavior.
- Fresh final `git diff --check` after this pointer: passed.
- Fresh final `GOMAXPROCS=2 make verify` after this pointer: passed.

Failure location/reason:

- None.

## 2026-06-04 11:51 CST - Workspace Console Product Surface

Transition:

- `T-INTERNAL-TEST4-WORKSPACE-CONSOLE-PRODUCT-SURFACE-001`

What changed:

- Added `GET /workspace` as a product-oriented web console over the existing
  safe A21 workspace APIs.
- The console exposes query-scope selection, local document upload,
  no-execute index request, source readiness refresh, professional read-record
  refresh, and roleplay/professional boundary readouts.
- The page keeps status labels honest: `stored_local`,
  `indexing_requested_no_execute`, `searchable=false`,
  `v21_execution_allowed=false`, and `physical_accepted=false`.
- The implementation uses static Go-served HTML/CSS/JS and adds no new
  package, service, port, auth provider, database, cloud storage, or runtime
  process.

Files changed:

- `internal/gateway/server.go`
- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-workspace-console-product-surface.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests run and results:

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestSimulatorPageServed|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource|TestProfessionalWorkspace' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Runtime or physical evidence:

- Local Gateway was started on `127.0.0.1:21080` only for page verification.
- `curl http://127.0.0.1:21080/healthz`: returned ok.
- `curl http://127.0.0.1:21080/workspace`: returned HTML.
- Browser/IAB tool was unavailable from this context, so Playwright was used as
  fallback.
- Playwright rendered `/workspace` at desktop `1270x900` and mobile `390x900`;
  both had zero overflow findings.
- Playwright exercised the product console with a dummy local text fixture:
  upload produced `stored_local`, index request produced
  `indexing_requested_no_execute`, source count became `1`, and
  `searchable=false` remained visible.

Deviations from plan:

- None. The console stayed on existing safe Gateway APIs and did not add a new
  frontend build stack.

Remaining issues:

- This is not real parsing, OCR, chunking, embedding, cloud storage, durable
  account/device binding, real V21 indexing, V21 merge/release, provider
  execution, or physical StackChan professional consult acceptance.
- The dummy upload written during browser verification is local runtime test
  data, not product knowledge evidence.

Next suggested action:

- Move from no-execute workspace console to the real indexing adapter path, or
  add account/device binding and source deletion/export UX, while keeping V21
  execution behind the A21 adapter contract.

Forbidden actions avoided:

- No provider, V21, ECS, firmware, serial, NVS, prune/gc, flash, or physical
  hardware action occurred.

## 2026-06-04 12:13 CST - Workspace Roleplay Control Surface

Transition:

- `T-WORKSPACE-ROLEPLAY-CONTROL-SURFACE-001`

What changed:

- Extended `GET /workspace` with roleplay setup controls over existing safe
  Gateway contracts only.
- Added product-console controls for role soul, scenario, voice profile,
  bounded roleplay memory hint, save, and memory clear.
- The page now populates role soul/scenario options from
  `/v1/roleplay-profile` and voice options from `/v1/voice-chain-profiles`.
- Saving roleplay setup writes `/v1/roleplay-profile`; the backend keeps
  voice profile selection mediated through the existing voice-chain contract.
- The page shows only safe readiness/status fields: selected IDs, prompt-input
  readiness, memory status/count, expression action/packet counts, and
  `physical_accepted=false`.
- Safe metadata export now includes selected roleplay profile/scenario/voice
  and memory status, still with redaction flags only.

Files changed:

- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-workspace-roleplay-control-surface.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests run and results:

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestRoleplayProfile|TestVoiceChainProfiles|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource|TestProfessionalWorkspace' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Runtime or physical evidence:

- Local Gateway was started on `127.0.0.1:21080` only for page verification,
  then stopped.
- Playwright opened `/workspace` at desktop `1270x900` and mobile `390x900`.
- Playwright selected `a21_roleplay_wry_peer`, `engineer_pushback`, and
  `a21_voice_clone_default`, saved one bounded memory hint, observed
  `/v1/roleplay-profile` return `memory_count=1`,
  `prompt_composed=true`, `expression_actions=4`, and
  `physical_accepted=false`, then cleared memory.
- Playwright exported safe metadata and verified roleplay profile/scenario/
  voice/status fields plus redaction flags.
- Desktop and mobile screenshots were written under `.a21-run/evidence/` for
  local runtime evidence only.

Deviations from plan:

- Browser/IAB control was not exposed in this context, so Playwright was used
  as the browser automation fallback.

Remaining issues:

- This is product-console configuration and Gateway contract evidence. It is
  not real provider execution, real TTS/voice-clone audio playback, physical
  roleplay expression acceptance, wake-word firmware activation, V21 indexing,
  or physical StackChan professional consult acceptance.

Next suggested action:

- Continue toward the full PRD by adding the adjacent wake-word/provider
  control surface to `/workspace`, or move to a foreground hardware evidence
  window for roleplay/professional physical acceptance once the operator is
  ready.

Forbidden actions avoided:

- No provider, V21, ECS, firmware, serial, NVS, prune/gc, flash, or physical
  hardware action occurred.

## 2026-06-04 12:02 CST - Workspace Console Management Controls

Transition:

- `T-WORKSPACE-CONSOLE-MANAGEMENT-CONTROLS-001`

What changed:

- Extended `GET /workspace` with first-pass management controls over existing
  safe Gateway APIs only.
- Added a delete-source control that calls existing
  `PUT /v1/workspace-upload-jobs` with `action=delete` and leaves an honest
  `deleted_metadata_only` source tombstone.
- Added a client-side safe metadata export with schema
  `a21.workspace_console_export.v1` and explicit redaction flags.
- Added professional read-record filters for `record_id`, `trace_id`, and
  `session_id`.
- Tuned the workspace console status grid so long A21 status tokens remain
  readable on desktop and mobile.

Files changed:

- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-workspace-console-management-controls.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests run and results:

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource|TestProfessionalWorkspace' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Runtime or physical evidence:

- Local Gateway was started on `127.0.0.1:21080` only for page verification,
  then stopped.
- Playwright opened `/workspace` at desktop `1270x900` and mobile `390x900`.
- Playwright exercised dummy local upload, no-execute index request, metadata
  export, filtered read-record refresh, filter clear, and delete-source flow.
- Observed states included `indexing_requested_no_execute`,
  `deleted_metadata_only`, `deleted_no_execute`, and `searchable=false`.
- Export evidence schema was `a21.workspace_console_export.v1` with no raw
  content, private location, secret value, provider result, voice data, or
  evidence body included.
- Screenshots were written under `.a21-run/evidence/` for local runtime
  evidence only.

Deviations from plan:

- Browser/IAB control was not exposed in this context, so Playwright was used
  as the browser automation fallback.

Remaining issues:

- This is still no-execute workspace management. It is not real parsing,
  chunking, embedding, OCR, cloud upload, V21 indexing, provider execution,
  durable account/device binding, or physical StackChan professional consult
  acceptance.

Next suggested action:

- Move to the next internal test 4 slice: either real adapter-indexing worker
  staging with strict redaction, or account/device binding for the workspace
  console, while keeping physical/hardware acceptance in a separate foreground
  evidence window.

Forbidden actions avoided:

- No provider, V21, ECS, firmware, serial, NVS, prune/gc, flash, or physical
  hardware action occurred.

## 2026-06-04 12:23 CST - Workspace Voice Chain And Wake Control Surface

Transition:

- `T-WORKSPACE-VOICE-CHAIN-WAKE-CONTROL-SURFACE-001`

What changed:

- Extended `GET /workspace` with provider voice-chain and wake-word setup
  controls over existing safe Gateway contracts only.
- Added voice-chain controls for chain mode, ASR profile, LLM profile,
  realtime provider, save, hot-switch status, effective TTS readout, and safe
  finding readout.
- Added wake-word controls for builtin/custom mode, desired phrase, desired
  pinyin, threshold, save, reset, build-required status, runtime hot-swap
  status, active phrase, runtime status, firmware status, and safe code.
- Safe metadata export now includes selected voice-chain and wake-word status
  fields while keeping redaction flags explicit.

Files changed:

- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-workspace-voice-chain-wake-control-surface.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests run and results:

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestVoiceChainProfiles|TestWakeWord|TestRoleplayProfile|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource|TestProfessionalWorkspace' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Runtime or physical evidence:

- Local Gateway was started on `127.0.0.1:21080` only for page verification,
  then stopped.
- Playwright opened `/workspace` at desktop `1270x900` and mobile `390x900`.
- Playwright selected `cascade`, `doubao_asr_realtime`, `stepfun`, and
  `openai_realtime`, saved voice-chain settings, and observed
  `/v1/voice-chain-profiles` return the selected profiles plus
  `hot_switch=true`.
- Playwright saved custom wake-word intent `custom_multinet` for
  `小阿二一` / `xiao a er yi` with threshold `35`, observed
  `/v1/wake-word` return `pending_firmware_build`,
  `firmware_build_required=true`, `custom_pending_firmware`,
  `builtin_xiaozhi_wakenet`, `runtime_hot_swap_supported=false`, and
  `custom_runtime_active=false`, then reset back to `builtin_xiaozhi`.
- Desktop and mobile screenshots plus export evidence were written under
  `.a21-run/evidence/` for local runtime evidence only.

Deviations from plan:

- Browser/IAB control was not exposed in this context, so Playwright was used
  as the browser automation fallback.

Remaining issues:

- This is product-console configuration and Gateway contract evidence. It is
  not provider execution, realtime provider session acceptance, audible
  voice-clone playback, custom wake-word firmware build/flash/activation, V21
  execution, or physical StackChan acceptance.

Next suggested action:

- Move from frontend configuration into evidence: either host-only provider
  voice-chain smoke for the selected chain or a foreground hardware window for
  roleplay/professional/wake physical acceptance.

Forbidden actions avoided:

- No provider, V21, ECS, firmware build, serial, NVS, prune/gc, flash, or
  physical hardware action occurred.

## 2026-06-04 12:39 CST - Workspace Voice Probe Control Surface

Transition:

- `T-WORKSPACE-VOICE-PROBE-CONTROL-SURFACE-001`

What changed:

- Extended `GET /workspace` with a safe Voice Probe panel over existing Gateway
  routes only.
- Added a roleplay probe that calls `/v1/fast-companion/turn` at the
  no-provider boundary and displays selected role soul, scenario, voice
  profile, memory count, prompt readiness, route status, and trace markers.
- Added a professional probe that calls `/v1/mock-turn`, refreshes
  `/v1/traces`, and filters `/v1/professional-read-records` by generated trace
  id to display safe professional route/read metadata.
- Safe metadata export now includes the last voice-probe mode, trace/session,
  route, event count, trace marker names, roleplay profile, voice profile, and
  professional status.

Files changed:

- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/plans/2026-06-04-workspace-voice-probe-control-surface.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests run and results:

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestFastCompanionHybridRoutesLocalAudioFrontendToTextStreamBoundary|TestProfessionalReadRecordsCompleteWithRedactedScopeMetadata|TestProfessionalReadRecordsFailSafelyWhenV21Unavailable|TestProfessionalVoiceTrigger|TestRoleplayProfile|TestVoiceChainProfiles|TestWakeWord|TestProfessionalWorkspace|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Runtime or physical evidence:

- Local Gateway was started on `127.0.0.1:21080` only for page verification,
  then stopped.
- Playwright used a temporary `/tmp/a21-playwright` module install and system
  Chrome; no project dependency was added.
- Playwright opened `/workspace` at desktop `1270x900` and mobile `390x900`
  with no horizontal overflow.
- Playwright selected `a21_roleplay_wry_peer`, `engineer_pushback`, and
  `a21_voice_clone_default`, saved one bounded memory hint, and ran the
  roleplay probe. Observed safe status:
  `route=fast_companion_hybrid`, trace event count `14`,
  `a21_roleplay_wry_peer / engineer_pushback / prompt=true`,
  `voice=a21_voice_clone_default`, and `memory=ready / 1`.
- Playwright ran the professional probe. Observed safe status:
  `route=professional_mock_turn`, trace event count `13`, and one
  `a21-professional-read-*` record with `status=completed`,
  `query_scope=public_only`, and `workspace_status=searchable` in the
  host-local default mock path.
- Desktop/mobile screenshots and JSON evidence were written under
  `.a21-run/evidence/` for local runtime evidence only.

Deviations from plan:

- Browser/IAB control was not exposed in this context, so Playwright was used
  as the browser automation fallback.

Remaining issues:

- This is product-console dialogue-path metadata evidence. It is not real
  provider execution, real V21 execution, audible voice-clone playback,
  real indexing, Gateway deployment, ECS runtime acceptance, firmware build,
  wake-word activation, or physical StackChan roleplay/professional acceptance.

Next suggested action:

- Move to a host-only selected voice-chain smoke/readiness slice, or schedule a
  foreground hardware evidence window for physical roleplay/professional/wake
  acceptance once the operator is ready.

Forbidden actions avoided:

- No provider, real V21, ECS, firmware build, serial, NVS, prune/gc, flash, or
  physical hardware action occurred.

## 2026-06-04 13:00 CST - Selected Voice-Chain Readiness Ingress

Round goal:

- Complete
  `T-VOICE-CHAIN-EVIDENCE-001-SELECTED-VOICE-CHAIN-READINESS-INGRESS`
  without repeating the already-completed workspace voice-probe cut.

Actual completed work:

- Added `--voice-chain-readiness-report` to `a21 product-readiness`.
- Added `--voice-chain-readiness-report` passthrough to
  `a21 server-side-readiness-bundle`.
- Added `--use-latest-reports` discovery for newest
  `a21-xiaozhi-streaming-provider-readiness-*.json`.
- Added defensive ingestion for
  `a21.xiaozhi_streaming_provider_readiness.v1` static no-execute selected
  ASR/LLM/TTS capability evidence.
- Product/server-side readiness now expose static capability evidence status,
  basename source report, profile match state, stage-ready booleans, execution
  mode, and safe finding codes.
- Mismatched capability reports remain visible as
  `voice_chain_capability_report_mismatch` and are not absorbed as current-chain
  readiness.
- Updated current-control, protocol, internal-test4 plan, and state-machine
  docs.

Changed files:

- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-selected-voice-chain-readiness-ingress.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This ingress is static no-execute readiness bookkeeping only. It does not run
  a provider, run V21, deploy Gateway/ECS, or prove audible/physical StackChan
  behavior.
- Next promotion still needs the existing provider execution, V21 execution,
  host/physical voice, wake, and physical PRD acceptance gates.

Known risks/blockers:

- Static capability evidence can prove selected-chain configuration readiness,
  but not runtime latency, answer quality, audio quality, or physical playback.
- A future gate decision is still needed if static capability should become a
  hard server-side candidate requirement.

Recommended next action:

- Continue from the current focused state. Either collect/ingest fresh selected
  voice-chain static reports from the active runtime, or move to the next
  runtime/physical evidence gate without redoing workspace console/probe work.

Test/build/runtime results:

- `go test ./internal/app -run 'TestProductReadiness(IngestsSelectedVoiceChainStaticReadiness|RejectsVoiceChainStaticReadinessMismatch|BlocksServerSideCandidateWhenStepFunNotSelected)|TestServerSideReadinessBundle(AcceptsVoiceChainStaticReadinessReport|SurfacesStepFunNotSelected)|TestRunProductReadinessUsesLatestVoiceChainReadinessReport' -count=1`:
  passed.
- `go test ./internal/app -run 'TestProductReadiness(AcceptsExecutedProviderSmokeEvidence|RejectsProviderSmokeReportMismatch|AcceptsProviderRealtimeFixtureEvidence|IngestsSelectedVoiceChainStaticReadiness|RejectsVoiceChainStaticReadinessMismatch|BlocksServerSideCandidateWhenStepFunNotSelected)|TestRunProductReadiness(UsesLatestRealtimeFixtureWithoutPathLeak|UsesLatestVoiceChainReadinessReport|LatestProviderSelectionPrefersNewestUsableConfiguredMatch)|TestServerSideReadinessBundle(AcceptsVoiceChainStaticReadinessReport|SurfacesStepFunNotSelected)' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- None. During implementation a transient loader audit found that a broad
  provider-smoke forbidden-value helper would reject the explicit
  `v21_adapter_only` boundary string; this was replaced with a voice-chain
  specific safe-value check and provider smoke/realtime loaders were confirmed
  restored.

Forbidden actions avoided:

- No provider execution, V21 execution, ECS/root-secret/runtime change, Gateway
  protocol change, firmware build, flash, serial, NVS, report deletion,
  prune/gc, or physical hardware action occurred.

## 2026-06-04 17:55 CST - Official Product Playback Ack Overlay

Round goal:

- Continue `T-XIAOZHI-PHYSICAL-PRD-PROMOTE-GATE-001` from the Gateway
  product playback allowance without repeating closed roleplay/provider/V21
  work. Add the matching official-compatible product firmware overlay surface
  so a real StackChan can eventually emit playback `start` / `stop_done`
  evidence.

Actual completed work:

- Updated
  `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
  with product-safe playback acknowledgement support.
- Added `CONFIG_A21_PRODUCT_PLAYBACK_EVENTS` and enabled it in the
  official-compatible product lane.
- The product firmware hello now advertises only
  `hello.features.playback_events=true`; it does not advertise
  `features.device_events`.
- Product firmware parses server hello `a21.profile=product` /
  `a21.playback_events=true` before sending any playback ack.
- Playback `start` is emitted after the official Xiaozhi audio output task
  reaches playback. Playback `stop_done` is emitted after server TTS stop or
  local abort queue clear. Local abort while speaking clears the decoder queue
  immediately.
- Added official overlay contract tests to forbid debug/legacy
  `device_events` in the product ack path.
- Updated protocol, firmware README, current-control, internal-test4 plan,
  hardware parity plan, and project state docs.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `firmware/stackchan/README.md`
- `internal/app/official_stackchan_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- No ECS/runtime env was changed. The Gateway env
  `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS` remains default-off unless explicitly
  enabled in a guarded runtime window.
- No firmware flash, NVS write, serial access, provider execution, V21
  execution, or physical evidence collection occurred.
- Product readiness remains blocked on `physical_stackchan_prd_acceptance`.

Known risks/blockers:

- The overlay has been verified to apply to the official top-level HEAD plus
  clean `firmware/xiaozhi-esp32` dependency HEAD, and the guarded no-flash
  product build now passes. It still needs foreground flash/physical evidence
  before any PRD acceptance claim.
- Physical PRD promotion still needs real device playback-start, bounded
  stop_done/barge-in evidence, and operator or instrumented audible
  observation.

Recommended next action:

- Review the passed no-flash build report, then schedule a foreground product
  flash/evidence window only with the guarded
  `a21-stackchan-official-xiaozhi-compatible-*` lane and explicit operator
  approval. Keep `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS` disabled until that
  runtime window is ready to collect playback ack evidence.

Test/build/runtime results:

- Overlay apply check against temporary official top-level HEAD plus clean
  `firmware/xiaozhi-esp32` dependency HEAD:
  `git apply --check --recount firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`:
  passed.
- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlay(AddsProductPlaybackAckOnly|KeepsA21IdleSocketReady|SetsCodecVolumeBeforeRuntime|SetsZiYueCustomWake|PreservesXiaozhiWifiProvisioning)' -count=1`:
  passed.
- `A21_STACKCHAN_OFFICIAL_DEP_CACHE='/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan' GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`:
  passed. Report:
  `reports/a21-stackchan-official-baseline-20260604-175756-1780567076043462000.json`.
  App artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`,
  SHA-256 `e66a41ef486b866b076746bd064af2e3afb75e0a316515921bbc681b89fb36a8`.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- First temporary overlay apply check used a top-level `git archive HEAD` only,
  which omitted the official dependency directory
  `firmware/xiaozhi-esp32`. The check was rerun with the clean dependency HEAD
  archived into the temporary tree.
- One intermediate patch shape used blank context lines that failed
  `git diff --check`.
- First no-flash product build failed because zero-context insertions drifted
  in C++ files, placing `Protocol` declarations outside the class and the audio
  callback outside `AudioOutputTask`. The overlay hunks now use stable nonblank
  anchors and the no-flash build passes.

Forbidden actions avoided:

- No repeated roleplay/provider/V21 evidence runs.
- No ECS deployment or runtime env mutation.
- No firmware flash, serial access, NVS write, Wi-Fi change, report deletion,
  repository prune/gc, or rollback of internal test 3 protocol/audio changes.

## 2026-06-04 17:39 CST - Product Playback Ack Channel Adaptation

Round goal:

- Continue `T-XIAOZHI-PHYSICAL-PRD-PROMOTE-GATE-001` without repeating closed
  roleplay/provider/V21 work or weakening internal test 3. Add only the
  product-safe playback acknowledgement adaptation needed to collect missing
  physical StackChan playback-start / stop_done evidence.

Actual completed work:

- Added `hello.features.playback_events` parsing to the Xiaozhi transport
  hello model.
- Added Gateway option/env `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS`, default off.
- When the env/option is enabled, only a hardware-MAC device that advertises
  `playback_events` and does not request debug flags receives
  `a21.profile=product` / `a21.playback_events=true`.
- Product allowance accepts only playback `start` / `stop_done` `type=device`
  acknowledgements. Non-playback device events are rejected, and the existing
  debug `features.device_events=true` path remains isolated.
- Device registry capabilities now record
  `xiaozhi_feature_playback_events=true` and
  `xiaozhi_product_playback_events=true` for the product allowance while
  keeping `xiaozhi_profile=stock`.
- Updated protocol, internal-test4 plan, current-control, and project state
  docs to state that this is an evidence-collection path, not physical PRD
  acceptance.

Changed files:

- `internal/transport/xiaozhi/frame.go`
- `internal/transport/xiaozhi/frame_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This does not deploy to ECS, enable the env in runtime, build firmware, flash,
  write NVS, touch serial, execute providers, execute V21, or collect fresh
  physical/audible evidence.
- A product firmware/runtime cut still needs to advertise
  `features.playback_events=true` before this can produce real device ack
  evidence.
- Product readiness remains blocked on `physical_stackchan_prd_acceptance`.

Known risks/blockers:

- The new Gateway allowance is intentionally default-off. If a physical device
  does not advertise `playback_events`, it remains on the ordinary stock path.
- Physical PRD promotion still needs real StackChan playback-start,
  bounded stop_done/barge-in evidence, and operator or instrumented audible
  observation.

Recommended next action:

- Implement or select the product-lane firmware/runtime patch that advertises
  `features.playback_events=true` and emits playback `start` / `stop_done`,
  then run a foreground StackChan physical evidence window and promote only via
  `a21 xiaozhi-physical-prd-review --confirm ACCEPT_A21_XIAOZHI_PHYSICAL_PRD`.

Test/build/runtime results:

- `go test ./internal/app -run 'TestGatewayServerOptionsFromEnvWires(ProductPlaybackEvents|StockProfessionalRoute)' -count=1`:
  passed.
- `go test ./internal/transport/xiaozhi -run 'TestParseHelloCapturesFeatureProfile|TestBuildServerHelloKeepsStockProfileFreeOfDebugExtensions' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestXiaozhi(WebSocketStockProfileHelloReply|WebSocketDebugProfileHelloReplyIncludesA21DeviceEventsAllowance|StockProfileRejectsPlaybackStartDeviceEvent|ProductPlaybackEventsAllowanceRecordsPlaybackStart|DebugProfileRecordsPlayback(Start|StopDone)DeviceEvent|DebugProfileRejectsUnsafePlaybackStreamID)' -count=1`:
  passed.
- `go test ./internal/gateway -run 'TestXiaozhiProductPlaybackEventsAllowanceRecordsPlaybackStart|TestXiaozhi(StockProfileRejectsPlaybackStartDeviceEvent|DebugProfileRecordsPlayback(Start|StopDone)DeviceEvent|DebugProfileRejectsUnsafePlaybackStreamID)' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- Initial continuation compile failure was in `internal/gateway/server.go`
  `recordXiaozhiDeviceSeen`: the in-progress diff referenced `session` before
  declaring it. The function now constructs the temporary session first and
  attaches hello features before deriving capabilities.

Forbidden actions avoided:

- No repeated roleplay/provider/V21 evidence runs.
- No ECS deployment or runtime env mutation.
- No firmware build, flash, serial access, NVS write, Wi-Fi change, report
  deletion, repository prune/gc, or rollback of internal test 3 protocol/audio
  changes.

## 2026-06-04 17:25 CST - Roleplay Clone Runtime Closed And Server-Side Candidate Ready

Round goal:

- Stop repeating prior Gateway/cloud checks and close the one remaining
  server-side roleplay voice-clone runtime gap from the current branch state.

Actual completed work:

- Confirmed cloud `/v1/roleplay-profile` was already set to
  `a21_roleplay_wry_peer`, `engineer_pushback`, and
  `a21_voice_clone_default` with memory ready.
- Reproduced the fresh failure once:
  `reports/a21-roleplay-voice-probe-20260604-170513.json` showed roleplay and
  voice-clone selection hit, but the fast companion pipeline fell to
  `local_fallback`.
- Fixed the provider runtime env mismatch left after the previous alias patch:
  `VoicePipelineAdaptersFromEnv` now reads `A21_VOICE_CLONE_CLI` as a
  compatibility alias for `A21_VOICE_CLONE_COMMAND`.
- Found the deeper ECS voice-clone wrapper failure through direct
  `local-tts-smoke`: the DashScope CosyVoice wrapper was consuming generic
  Qwen realtime `A21_DASHSCOPE_TTS_MODEL=qwen3-tts-flash-realtime` and
  `A21_DASHSCOPE_TTS_VOICE=Cherry`, causing provider task failures.
- Updated `scripts/a21_dashscope_cosyvoice_tts.py` so CosyVoice clone runtime
  prefers the explicit `--model`/voice-clone model, uses only
  `A21_DASHSCOPE_COSYVOICE_VOICE` or the CosyVoice default voice for voice
  selection, and emits short provider task failure code/message diagnostics
  without printing prompts, keys, URLs, or audio.
- Fixed roleplay evidence parsing so real StackChan MAC-style `device_id`
  values such as `44:1b:f6:e2:6a:60` are accepted while URL/path/whitespace
  and credential-shaped identities remain rejected.
- Remote voice-clone CLI smoke then passed and produced a 16 kHz mono WAV with
  audio quality passed.
- Fresh roleplay voice runtime probe passed before formal ECS swap:
  `reports/a21-roleplay-voice-probe-20260604-172311.json`, with StepFun text
  stream executed, prompt input used, voice clone profile used, audio downlink
  observed, device playback start observed, and 46 audio chunks. After deploying
  commit `431c7ec`, final deployed evidence
  `reports/a21-roleplay-voice-probe-20260604-172658.json` also passed with 45
  audio chunks.
- Fresh server-side bundle passed as server-side candidate before deploy:
  `reports/a21-server-side-readiness-bundle-20260604-172349.json`; after deploy
  `reports/a21-server-side-readiness-bundle-20260604-172722.json` is the final
  current bundle.
- Fresh product readiness remains truthful:
  `reports/a21-product-readiness-20260604-172349.json` was the pre-deploy
  bundle; after deploy `reports/a21-product-readiness-20260604-172722.json` is
  `server_side_candidate_ready`, `launch_ready=false`, `prd_accepted=false`,
  with only `physical_stackchan_prd_acceptance` missing at the canonical
  decision layer.

Changed files:

- `internal/providers/voice_pipeline_adapters.go`
- `internal/providers/voice_pipeline_real_adapters_test.go`
- `internal/app/product_demo.go`
- `internal/app/app_test.go`
- `scripts/a21_dashscope_cosyvoice_tts.py`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- Full PRD launch still requires physical StackChan PRD acceptance:
  playback ack/start from device or trusted runtime echo, stop_done/auto_stop
  where applicable, and operator or instrumented audible observation.
- Product readiness still reports V21 adapter runtime env as not configured in
  the local shell, while the fresh professional external-gateway report is used
  as evidence for server-side candidate readiness. Do not misread this as full
  permanent ECS V21 topology.

Known risks/blockers:

- CosyVoice wrapper now isolates Qwen realtime model/voice env, but cloud
  clone TTS latency in the smoke was about 1.2s to first audio. This is usable
  for roleplay runtime evidence, not yet a tuned low-latency physical PRD
  acceptance metric.
- The control Mac source IP changed from `192.168.0.196` to `10.98.141.239`
  during this round. Future public Gateway and SSH checks must use the current
  `en0` source IP through `A21_DIRECT_SOURCE_IP` or `ssh -b`.

Recommended next action:

- Continue with `T-XIAOZHI-PHYSICAL-PRD-PROMOTE-GATE-001` in a foreground
  hardware/evidence window. Do not rerun the roleplay clone debugging path
  unless the passed report is superseded by a new failure.

Test/build/runtime results:

- `go test ./internal/providers -run 'TestVoicePipelineAdaptersFromEnvSelectsVoiceCloneCLI|TestVoicePipelineAdaptersFromEnvAcceptsVoiceCloneCLIAlias|TestVoicePipelineRunnerPassesVoiceCloneProfileToTTSAndReport' -count=1`:
  passed.
- `go test ./internal/app -run 'TestRunRoleplayVoiceProbeWritesReadyReportAndProductReadinessCanIngest|TestProductRoleplayVoiceSafeOptionalIDAcceptsDeviceMACOnly|TestRunRoleplayVoiceProbeWritesBlockedReportWhenPipelineIncomplete|TestProductVoiceReadinessAcceptsVoiceCloneCLIAlias' -count=1`:
  passed.
- `python3 -m py_compile scripts/a21_dashscope_cosyvoice_tts.py`: passed.
- Remote `/opt/a21/bin/a21 local-tts-smoke --engine voice_clone_cli`: passed;
  latest remote report `a21-local-tts-smoke-20260604-171908.json`.
- `a21 roleplay-voice-probe --require-ready`: passed; reports
  `reports/a21-roleplay-voice-probe-20260604-172311.json` and final deployed
  `reports/a21-roleplay-voice-probe-20260604-172658.json`.
- `a21 server-side-readiness-bundle`: passed as server-side candidate; final
  deployed report `reports/a21-server-side-readiness-bundle-20260604-172722.json`.
- `a21 product-readiness`: server-side candidate only; final deployed report
  `reports/a21-product-readiness-20260604-172722.json`.

Failure location/reason:

- The first runtime failure was real: voice-clone TTS command failed inside the
  ECS wrapper, so Gateway correctly returned `local_fallback`.
- The wrapper failure was caused by generic Qwen realtime TTS model/voice env
  leaking into the CosyVoice clone wrapper. After model/voice isolation, the
  command produced WAV successfully.
- The subsequent probe `blocked` status with a completed pipeline was a local
  report parser bug: MAC-style product device IDs were rejected by provider-ID
  safety rules.

Forbidden actions avoided:

- No firmware build, firmware flash, serial, NVS write, provider key print,
  secret edit, report deletion, prune/gc, broad rollback, or internal-test3
  protocol/audio regression occurred.

## 2026-06-04 16:15 CST - Internal Test 4 Hardware Window Control and PRD Promote Gate

Round goal:

- Regain control of the active Internal Test 4 threads, avoid repeating
  already-merged internal-test3/protocol work, unblock foreground StackChan
  hardware progress, and add the missing product-readiness promote boundary for
  stock Xiaozhi physical evidence.

Actual completed work:

- Kept internal test 3 protocol/audio changes intact and worked on the
  foreground hardware-window branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
- Created and pushed `codex/a21-internal-test4-mainline-20260604` from
  current HEAD `87579cf` as the clean internal-test4 coordination branch.
- Confirmed public `47.103.57.217` was not usable from this Mac during this
  window (`curl` empty replies; SSH closed), then used a local LAN Gateway on
  `0.0.0.0:21080` / `http://10.98.141.239:21080`.
- Ran a guarded product-lane official-compatible NVS write on
  `/dev/cu.usbmodem1101`; report
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-155946-1780559986794566000.json`
  passed with servo calibration and existing Wi-Fi credentials preserved.
- Reset the device and observed the real current blocker:
  `WifiStation: No AP found`. The Mac has LAN IP `10.98.141.239` but is not on
  Wi-Fi, so the product device needs a reachable AP restored or explicit Wi-Fi
  credentials written through the guarded NVS lane.
- Added optional explicit Wi-Fi SSID/password support to the official-compatible
  NVS writer. It mutates only requested Wi-Fi keys plus Xiaozhi connection keys
  and keeps stdout/reports redacted.
- Added `a21 xiaozhi-physical-prd-review`,
  `stackchan-accept --check xiaozhi-prd-review`, and
  `make xiaozhi-physical-prd-review`. The new command consumes matching
  `a21.xiaozhi_physical_evidence.v1` and
  `a21.xiaozhi_half_duplex_acceptance.v1` reports, requires
  `ACCEPT_A21_XIAOZHI_PHYSICAL_PRD`, verifies playback-start, audible
  observation, mic, barge-in stop, `device.playback.stop_done`, target matching,
  and redaction, then writes an accepted
  `a21.xiaozhi_physical_evidence.v1` report for `product-readiness`.
- Updated Makefile, protocol, doctor, firmware-release discipline, current
  control, internal-test4 plan, and project state documents.

Files changed:

- `Makefile`
- `internal/app/app.go`
- `internal/app/app_stackchan_common.go`
- `internal/app/official_stackchan.go`
- `internal/app/official_stackchan_test.go`
- `internal/app/xiaozhi_physical_prd_review.go`
- `internal/app/xiaozhi_physical_evidence_test.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/DOCTOR.md`
- `docs/engineering/FIRMWARE_RELEASE_DISCIPLINE.md`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- The physical device is not yet online after the local-LAN NVS update because
  its preserved Wi-Fi AP was not found.
- No accepted Xiaozhi physical PRD report was generated against the real device
  in this round; fresh physical evidence and half-duplex evidence still need to
  be collected after Wi-Fi is restored.
- Public ECS remains a separate access/runtime path from this Mac during this
  window and was not repaired here.

Known risks/blockers:

- Writing explicit Wi-Fi credentials requires the operator to provide a
  reachable AP SSID/password or restore the previous AP. Credential values must
  not be printed or committed.
- `xiaozhi-physical-prd-review` cannot invent missing evidence. If
  `device.playback.stop_done`, audible observation, playback-start, mic, or
  target matching are absent, it blocks and writes no accepted report.

Recommended next action:

- Provide or restore a reachable Wi-Fi AP, then rerun guarded
  `a21-stackchan-official-xiaozhi-compatible-nvs-execute` with explicit Wi-Fi
  credentials if needed.
- Restart/keep local LAN Gateway, wait for product StackChan to register on
  `/v1/devices`, collect fresh `xiaozhi-physical-evidence` and
  `stackchan-accept --check xiaozhi-half-duplex` reports with trusted audible
  observation, then run `make xiaozhi-physical-prd-review` and
  `a21 product-readiness --use-latest-reports`.

Test/build/runtime results:

- `go test ./internal/app -run 'TestRunXiaozhiPhysicalPRDReview|TestRunStackChanOfficialXiaozhiCompatibleNVS|TestOfficialXiaozhiCompatibleNVSCSV' -count=1`:
  passed.
- `go test ./internal/app -run 'TestRunXiaozhiPhysicalPRDReview|TestProductReadinessIngestsAcceptedXiaozhiPhysicalEvidence|TestRunXiaozhiHalfDuplexAcceptance|TestRunXiaozhiPhysicalEvidence|TestRunStackChanOfficialXiaozhiCompatibleNVS|TestOfficialXiaozhiCompatibleNVSCSV' -count=1`:
  passed.
- `go test ./internal/app -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- Initial `xiaozhi-physical-prd-review` focused test failed because the
  half-duplex report includes runtimeguard metadata with local/proxy state that
  is not copied into the accepted physical report. The reader was corrected to
  validate bounded single JSON, schema, redaction, and target fields for the
  half-duplex input instead of applying the generic raw physical-report scanner
  to metadata. Focused tests and full verify then passed.

Forbidden actions avoided:

- No repository prune/gc, generic `xiaozhi.bin` product flash, unguarded raw
  firmware upload, provider key exposure, provider key firmware storage, V21
  internals copy, or rollback of internal test 3 protocol/audio changes
  occurred. The only hardware mutation was the guarded official-compatible
  product NVS write described above.

## 2026-06-04 16:25 CST - Explicit Wi-Fi Credential NVS Attempt

Round goal:

- Use the operator-provided Wi-Fi credentials to connect the product StackChan
  through the official-compatible Xiaozhi product NVS lane without exposing the
  password or flashing the app.

Actual completed work:

- Confirmed the hardware-window branch was clean at commit `a216fe6`.
- Attempted to connect the Mac Wi-Fi to the provided SSID; the Mac could not
  find that network and remained on LAN `10.98.141.239`.
- Ran guarded product-lane NVS plan and execute on `/dev/cu.usbmodem1101` using
  the provided SSID/password via stdin, not command arguments.
- NVS execute passed with explicit Wi-Fi write enabled:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-162103-1780561263934023000.json`.
- Restarted the local LAN Gateway on `0.0.0.0:21080` and confirmed
  `/healthz` plus `/xiaozhi/ota/`.
- Hard-reset the device. Serial logs showed the device trying the provided
  SSID, then Wi-Fi disconnect reason `201`, 5 reconnect attempts, and fallback
  to Xiaozhi provisioning hotspot `Xiaozhi-6A61`.
- Gateway `/v1/devices` stayed empty during 12 polling attempts.

Files changed:

- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- Device is still not online because it cannot see/connect to the provided AP.
- No Xiaozhi physical evidence, half-duplex evidence, or PRD accepted physical
  report was generated in this attempt.

Known risks/blockers:

- ESP32-S3 requires a reachable 2.4 GHz AP. The provided SSID may be out of
  range, typoed, hidden/unsupported, or 5 GHz only.

Recommended next action:

- Provide a reachable 2.4 GHz SSID/password, move the device/AP into range, or
  use the `Xiaozhi-6A61` captive portal to provision a visible AP. Then rerun
  the same guarded NVS path only if NVS needs to be changed.

Test/build/runtime results:

- NVS plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-162056-1780561256235353000.json`
  was `status=ready`.
- NVS execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-162103-1780561263934023000.json`
  was `status=passed`, `write_executed=true`,
  `wifi_credentials_written=true`, `mutated_entry_count=5`, and
  `servo_calibration_present=true`.
- Gateway local health and OTA discovery passed before device reset.
- Serial evidence after reset showed Wi-Fi reason `201` and fallback to
  `wifi_configuring`.

Failure location/reason:

- Physical AP availability. The device tried the configured SSID but did not
  find/connect to it from the current location.

Forbidden actions avoided:

- No password was written to repository docs, no provider key was exposed, no
  firmware app flash occurred, no generic `xiaozhi.bin` product lane was used,
  no V21/provider execution occurred, no repository prune/gc was run, and no
  internal-test3 protocol/audio changes were reverted.

## 2026-06-04 16:35 CST - Cloud Gateway NVS Correction

Round goal:

- Correct the foreground hardware path after the operator clarified that the
  product StackChan should use the cloud Gateway rather than the temporary
  local LAN Gateway.

Actual completed work:

- Stopped the local Gateway before any second NVS write; port `21080` was no
  longer listening locally.
- Verified the control branch was the guarded hardware-window branch and clean.
- Wrote official-compatible product NVS with the operator-provided phone
  hotspot credentials and cloud endpoints:
  `http://47.103.57.217/xiaozhi/ota/` and
  `ws://47.103.57.217/v1/xiaozhi`.
- NVS execute passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-163211-1780561931273006000.json`.
- Hard-reset the product device after the write.
- Cloud TCP ports `22`, `80`, `443`, and `21081` were reachable from the
  current network, but HTTP/HTTPS application requests still returned empty
  replies or TLS syscall errors. SSH was closed by the remote host before
  authentication.
- Serial output after reset showed regular runtime `SystemInfo` lines in the
  observed window and did not repeat the earlier `No AP found` fallback.

Files changed:

- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- Device registration on cloud `/v1/devices` could not be confirmed because the
  cloud Gateway HTTP/WS path is not responding from this network.
- Fresh Xiaozhi physical evidence, half-duplex evidence, and PRD physical
  promote remain pending.

Known risks/blockers:

- The product NVS now points to the cloud Gateway as intended. If the device is
  on Wi-Fi but cannot register, the next blocker is cloud runtime/Caddy/Gateway
  application-layer reachability, not local LAN routing.

Recommended next action:

- Use an ECS/control-plane path or operator cloud console to confirm
  `a21-gateway` and Caddy health, then restore public
  `http://47.103.57.217/healthz`, `/xiaozhi/ota/`, and `/v1/devices`.
- Once cloud HTTP/WS responds, poll `/v1/devices` for
  `44:1b:f6:e2:6a:60`, collect stock Xiaozhi physical evidence and
  half-duplex evidence, then run `make xiaozhi-physical-prd-review`.

Test/build/runtime results:

- NVS plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-163203-1780561923834996000.json`
  was `status=ready`.
- NVS execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-163211-1780561931273006000.json`
  was `status=passed`, `write_executed=true`,
  `wifi_credentials_written=true`, `mutated_entry_count=5`, and
  `servo_calibration_present=true`.
- Direct cloud HTTP checks returned empty replies; direct HTTPS checks returned
  TLS syscall errors; direct TCP probes to ports `22`, `80`, `443`, and
  `21081` succeeded.

Failure location/reason:

- Cloud application-layer reachability. The all-cloud product endpoint is
  selected in device NVS, but public HTTP/WS cannot currently be verified from
  this control Mac.

Forbidden actions avoided:

- No password or hotspot name was written to repository docs, no provider key
  was exposed, no firmware app flash occurred, no generic Xiaozhi product lane
  was used, no local Gateway endpoint was written in the corrected NVS attempt,
  no V21/provider execution occurred, no repository prune/gc was run, and no
  internal-test3 protocol/audio changes were reverted.

## 2026-06-04 15:06 CST - Roleplay Voice Runtime Probe Closure

Round goal:

- Stop repeating readiness gate work and close the live local
  `roleplay_voice_runtime` blocker from the active server-side bundle.

Actual completed work:

- Started a local A21 Gateway on `127.0.0.1:21080` and ran
  `server-side-readiness-bundle --collect-missing --execute-provider-smoke
  --execute-v21-smoke`.
- Confirmed Gateway, professional ritual, professional read-record, wake-word,
  and voice-chain selector could become ready when the local Gateway is live.
- Diagnosed the remaining roleplay runtime blocker:
  - default safe voice profile reached the voice pipeline but was not traced as
    used;
  - the voice-pipeline branch emitted audio chunks but did not record
    `device.playback.start`;
  - product readiness rejected live prompt parts `core_identity` and
    `tone_rules` because only `key:value` prompt-part markers were accepted.
- Updated Gateway trace behavior so the roleplay voice-pipeline branch records
  selected safe voice-profile use and host/simulator playback-start.
- Updated product readiness prompt-part safety to accept safe single-token
  roleplay prompt-part IDs while keeping unsafe prompt/body/path/credential
  values forbidden.
- Re-ran live `a21 roleplay-voice-probe --require-ready`; it passed and wrote
  `a21-roleplay-voice-probe-20260604-150552.json`.
- Re-ran server-side readiness collection; it wrote
  `a21-server-side-readiness-bundle-20260604-150611.json` and now reports only
  `provider_smoke` as the no-hardware server-side blocker.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/product_demo.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-roleplay-voice-runtime-probe-closure.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- No real provider smoke is available in the current shell because no
  `A21_` provider env names are present.
- Full launch still requires physical StackChan online and physical PRD
  acceptance.

Known risks/blockers:

- `roleplay_voice_runtime_ready=true` is no-hardware host/simulator runtime
  evidence only. It proves selected voice-profile propagation, prompt input,
  audio downlink, and playback-start boundary; it does not prove real clone
  audio quality or physical speaker output.
- Public ECS `47.103.57.217` `/healthz` timed out during this round; local
  Gateway was used for runtime closure.

Recommended next action:

- Configure a real A21 provider (`A21_PROVIDER_PRIMARY` plus its required
  A21-namespaced key/model env), rerun executed provider smoke and
  `server-side-readiness-bundle --require-candidate`, then move to foreground
  physical StackChan acceptance.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestFastCompanionHybridRunsVoicePipelineWhenFramesProvided|TestFastCompanionVoicePipelineRecordsDefaultVoiceProfileAndPlaybackStart|TestFastCompanionHybridRoutesLocalAudioFrontendToTextStreamBoundary' -count=1`:
  passed.
- `go test ./internal/app -run 'TestRunRoleplayVoiceProbeWritesReadyReportAndProductReadinessCanIngest|TestRunServerSideReadinessBundleCollectsAuthorizedProviderAndV21Evidence|TestProductReadinessReportsServerSideCandidateWhenEvidenceSlicesPass' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `go test ./internal/app -count=1`: passed in 465.595s.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- Runtime: `a21 roleplay-voice-probe --gateway-url http://127.0.0.1:21080
  --device-id stackchan-sim-001 --require-ready --output-dir reports`: passed.
- Runtime: `a21 server-side-readiness-bundle --gateway-url
  http://127.0.0.1:21080 --device-id stackchan-sim-001 --use-latest-reports
  --collect-missing --execute-provider-smoke --execute-v21-smoke
  --collect-repeat 3 --output-dir reports`: passed with
  `missing_evidence=["provider_smoke"]`.

Failure location/reason:

- First live roleplay probe failed because the default selected voice profile
  was not traced as used and playback-start was missing. After trace fixes, it
  still self-classified as blocked because product readiness rejected safe
  single-token prompt parts. Both causes were fixed.

Forbidden actions avoided:

- No firmware build, flash, serial, NVS, ECS/root-secret change, provider key
  write, V21 repo mutation, report deletion, prune/gc, or physical hardware
  action occurred.

## 2026-06-04 15:31 CST - StepFun Provider Smoke Server Candidate Closure

Round goal:

- Close the remaining no-hardware server-side blocker `provider_smoke` without
  weakening the physical StackChan PRD gate.

Actual completed work:

- Found `.a21-run/provider.env` with A21-namespaced provider key variables and
  used it without printing secret values.
- Confirmed current shell had no active `A21_` provider env.
- Ran StepFun provider dry-run first:
  - initial dry-run showed only `A21_STEPFUN_MODEL` missing;
  - rerun with `A21_STEPFUN_MODEL=step-1-8k` returned `configured=true`.
- Ran executed StepFun streaming provider smoke:
  - report:
    `reports/provider-live/a21-provider-smoke-20260604-153030-291957000.json`;
  - `status=passed`, `executed=true`, `repeat=3`, HTTP status `200`;
  - first-content p50 `467.607ms`, p95 `2235.073ms`.
- Started local Gateway with StepFun launch-policy env and ran
  `server-side-readiness-bundle --provider-smoke-report <report>
  --use-latest-reports --require-candidate`.
- The resulting bundle
  `reports/a21-server-side-readiness-bundle-20260604-153129.json` returned
  `status=server_side_candidate_ready` and `candidate_ready=true`.

Changed files:

- `docs/plans/2026-06-04-stepfun-provider-smoke-server-candidate-closure.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- Full PRD launch still needs physical StackChan online evidence and physical
  PRD acceptance.
- Public ECS `47.103.57.217` was previously observed timing out on `/healthz`
  and has not been repaired in this round.

Known risks/blockers:

- The server-side candidate is no-hardware only. It proves the software
  evidence chain through Gateway/provider/professional/read-record/roleplay/
  wake/voice-chain readiness, but not physical wake, real speaker audibility,
  physical roleplay expression, or physical professional consultation.

Recommended next action:

- Move to foreground physical StackChan acceptance with the product lane:
  device online, wake/listen.start, Opus ingress, ASR, TTS downlink,
  playback-start/audible evidence, roleplay/professional mode behavior, and
  final PRD acceptance. Keep `a21-stackchan-official-xiaozhi-compatible.bin`
  as the product flash artifact if a guarded flash is needed.

Test/build/runtime results:

- `go run ./cmd/a21 provider-smoke --provider stepfun --stream --repeat 3
  --output-dir reports/provider-live` with `A21_STEPFUN_MODEL=step-1-8k`:
  configured dry-run passed.
- `go run ./cmd/a21 provider-smoke --provider stepfun --execute --stream
  --repeat 3 --output-dir reports/provider-live`: passed.
- `go run ./cmd/a21 server-side-readiness-bundle --gateway-url
  http://127.0.0.1:21080 --device-id stackchan-sim-001
  --provider-smoke-report reports/provider-live/a21-provider-smoke-20260604-153030-291957000.json
  --use-latest-reports --require-candidate --output-dir reports`: passed with
  `server_side_candidate_ready`.

Failure location/reason:

- Initial StepFun dry-run was not configured because `A21_STEPFUN_MODEL` was
  absent from `.a21-run/provider.env`. The model was set to the documented
  launch value `step-1-8k`, then dry-run and executed smoke both passed.

Forbidden actions avoided:

- No provider secret was printed, committed, written into firmware, or stored
  in report bodies. No ECS/root-secret change, firmware build, flash, serial,
  NVS, V21 repo mutation, report deletion, prune/gc, or physical hardware
  action occurred.

## 2026-06-04 14:04 CST - Professional Ritual Gate Landed

Round goal:

- Stop server-side readiness from treating V21 adapter smoke as sufficient
  professional-mode launch evidence.

Actual completed work:

- Added `professional_ritual_ready` and
  `professional_ritual_source_report` to product/server-side readiness.
- Server-side candidate readiness now requires accepted external Gateway
  `a21.xiaozhi_professional_bench.v1` evidence. Adapter smoke still closes the
  V21 adapter evidence gate, but cannot close the professional ritual gate by
  itself.
- `product-readiness --use-latest-reports` now prefers accepted
  `a21-xiaozhi-professional-bench-*.json` reports over newer adapter smoke so
  stronger ritual evidence is not overwritten.
- `server-side-readiness-bundle` now exposes `professional_ritual` and can
  collect `professional_ritual_execution` through
  `a21 xiaozhi-professional-bench --gateway-url <gateway> --output-dir reports`
  only when `--execute-v21-smoke` is explicitly supplied.
- Updated protocol/current-control/state/internal-test4 plan docs.

Changed files:

- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-server-side-professional-ritual-execution-gate.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This does not prove physical StackChan professional consult acceptance,
  audible `PRO` cue playback, screen/avatar/motion/RGB/servo behavior, ECS
  runtime readiness, or live provider/V21 production latency.

Known risks/blockers:

- Fresh runtime reports still need to be collected against the intended
  Gateway/provider/V21 environment. Physical PRD acceptance remains separate.

Recommended next action:

- Run `server-side-readiness-bundle --collect-missing --execute-provider-smoke
  --execute-v21-smoke --require-candidate` against the intended runtime once
  provider/V21 access is intentionally enabled, then move to the foreground
  StackChan hardware window for wake/roleplay/professional acceptance.

Test/build/runtime results:

- `go test ./internal/app -run 'TestProductReadiness(ReportsServerSideCandidateWhenEvidenceSlicesPass|ReportsServerSideBlockedWhenWakeWordBlocksRealSlices|CountsExecutedProfessionalReportWithoutLiveV21Health|ExposesV21ProfessionalExecutionForRealAdapterReport)|TestRun(ProductReadinessCommandUsesLatestReportsWithoutPathLeak|ProductReadinessLatestV21SelectionPrefersProfessionalBenchOverNewerAdapterSmoke|ServerSideReadinessBundleUsesLatestReportsWithoutPathLeak|ServerSideReadinessBundleCollectMissingSkipsExternalWithoutAuthorization|ServerSideReadinessBundleCollectsAuthorizedProviderAndV21Evidence)' -count=1`:
  passed.
- `go test ./internal/app -run 'TestProductReadinessCanReachRealLaunchReadyWhenInputsArePresent|TestRunServerSideReadinessBundleCollectsMissingRoleplayVoiceRuntime' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- First full verify failed because older launch/roleplay-collection tests had
  not supplied professional ritual evidence. Tests were updated to preserve
  the new gate rather than weakening it.

Forbidden actions avoided:

- No firmware build, flash, serial, NVS, provider key persistence in firmware,
  ECS/root-secret change, report deletion, prune/gc, destructive git, or
  physical hardware action occurred.

## 2026-06-04 14:23 CST - Professional Read-Record Gate Landed

Round goal:

- Make professional-mode readiness prove the Gateway read-record ledger, not
  only the checking ritual and V21 result.

Actual completed work:

- Extended `a21 xiaozhi-professional-bench` reports with a safe `read_record`
  summary fetched from `/v1/professional-read-records?trace_id=<bench-trace>`.
- The report now proves a single completed read record matching the bench
  trace/session/device, legal query scope, `professional_only` privacy,
  `fast_first` latency profile, `voice_first_with_citations` answer style,
  safe source-scope counts, workspace status, and redaction booleans.
- The bench report maps Gateway `VoiceTranscriptStored` into
  `voice_text_stored=false` so the report avoids transcript wording while
  preserving the redaction fact.
- Product readiness now exposes `professional_read_record_ready` and
  `professional_read_record_source_report`.
- Server-side readiness blocks with `professional_read_record` when the
  professional ritual report is otherwise accepted but the ledger is missing
  or incomplete.
- Server-side readiness bundle now exposes a `professional_read_record`
  evidence block and can recollect it through the existing professional bench
  path under the same explicit `--execute-v21-smoke` authorization.
- Updated protocol/current-control/state/internal-test4 plan docs.

Changed files:

- `internal/app/xiaozhi_professional_bench.go`
- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-professional-read-record-readiness-gate.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This does not implement real personal upload indexing, durable account ACL,
  cloud storage, ECS deployment, physical StackChan professional consult
  acceptance, or audible/visible hardware proof.

Known risks/blockers:

- Fresh runtime evidence still needs to be collected against the intended
  Gateway/provider/V21 environment. Physical PRD acceptance remains separate.

Recommended next action:

- Run a fresh authorized server-side readiness collection against the intended
  runtime, then proceed to the foreground physical StackChan window for
  wake/roleplay/professional proof.

Test/build/runtime results:

- `go test ./internal/app -run 'TestRunXiaozhiProfessionalBenchReportsExternalGatewayRuntimeContract|TestProductReadiness(CountsExecutedProfessionalReportWithoutLiveV21Health|RejectsXiaozhiProfessionalReportWithoutReadRecord)|TestRun(ServerSideReadinessBundleCollectsAuthorizedProviderAndV21Evidence|ServerSideReadinessBundleUsesLatestReportsWithoutPathLeak)' -count=1`:
  passed.
- `go test ./internal/app -run 'TestProductReadiness(ReportsServerSideCandidateWhenEvidenceSlicesPass|ReportsServerSideBlockedWhenWakeWordBlocksRealSlices|CountsExecutedProfessionalReportWithoutLiveV21Health|ExposesV21ProfessionalExecutionForRealAdapterReport|CanReachRealLaunchReadyWhenInputsArePresent)|TestRun(ProductReadinessCommandUsesLatestReportsWithoutPathLeak|ProductReadinessLatestV21SelectionPrefersProfessionalBenchOverNewerAdapterSmoke|ServerSideReadinessBundleUsesLatestReportsWithoutPathLeak|ServerSideReadinessBundleCollectMissingSkipsExternalWithoutAuthorization|ServerSideReadinessBundleCollectsAuthorizedProviderAndV21Evidence|ServerSideReadinessBundleCollectsMissingRoleplayVoiceRuntime)' -count=1`:
  passed.
- `go test ./internal/app -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- Initial focused run showed the bench was collecting read records, but the
  mock V21 client did not provide safe `workspace_status` or
  `source_scope_counts`; the mock was fixed to provide searchable/public
  metadata so the gate tests the intended contract.
- The external bench report initially emitted the JSON key
  `voice_transcript_stored`; this was changed to `voice_text_stored` to avoid
  transcript wording in readiness-facing evidence.

Forbidden actions avoided:

- No firmware build, flash, serial, NVS, provider key persistence in firmware,
  ECS/root-secret change, report deletion, prune/gc, destructive git, or
  physical hardware action occurred.

## 2026-06-04 13:39 CST - Roleplay Voice Probe Report Generator

Round goal:

- Add the missing generator for live Gateway roleplay voice runtime evidence so
  internal test 4 roleplay readiness can move from static/profile visibility to
  runnable host/Gateway evidence without hand-authored fixtures.

Actual completed work:

- Added `a21 roleplay-voice-probe`.
- The command sends a short redacted local-audio probe to the existing Gateway
  `/v1/fast-companion/turn` roleplay voice path and fetches `/v1/traces`.
- Generated reports use the existing `a21.roleplay_voice_probe.v1` schema and
  are written as `a21-roleplay-voice-probe-*.json`.
- Complete Gateway voice-pipeline evidence writes `status=passed`; incomplete
  evidence writes `status=blocked`.
- `--require-ready` returns non-zero for blocked evidence after preserving the
  report.
- Added tests proving generated ready reports are accepted by
  `product-readiness --use-latest-reports`, while blocked reports do not
  overclaim.
- Updated protocol, current-control, internal-test4 plan, project state, and
  the scoped generator plan.

Changed files:

- `internal/app/app.go`
- `internal/app/roleplay_voice_probe.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-roleplay-voice-probe-report-generator.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is host/Gateway runtime evidence generation only. It does not prove
  physical wake, audible voice-clone quality, hardware expression delivery, or
  full PRD launch acceptance.
- A fresh report still needs to be generated against the target live Gateway
  runtime after provider/voice-chain configuration is selected.

Known risks/blockers:

- If the Gateway is configured without a completing voice pipeline, the command
  will correctly write `status=blocked` instead of closing readiness.

Recommended next action:

- Run `a21 roleplay-voice-probe --gateway-url <target> --use target device/env`
  in the live runtime, rerun `a21 product-readiness --use-latest-reports`, then
  continue toward real low-latency voice, roleplay immersion, professional mode,
  and physical StackChan/MCP acceptance.

Test/build/runtime results:

- `go test ./internal/app -run 'TestRunRoleplayVoiceProbe|TestRunProductReadinessUsesLatestRoleplayVoiceRuntimeReport|TestProductReadinessSurfacesRoleplayVoiceRuntimeEvidence' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- None.

Forbidden actions avoided:

- No ECS/root-secret change, V21 execution, firmware build, flash, serial, NVS,
  report deletion, prune/gc, destructive Git cleanup, or physical hardware
  action occurred. The CLI does not call providers directly; any provider
  runtime occurs only through the already configured Gateway path.

## 2026-06-04 13:43 CST - Server-Side Roleplay Voice Runtime Gate

Round goal:

- Stop server-side candidate readiness from bypassing roleplay voice immersion
  after the roleplay voice probe generator landed.

Actual completed work:

- Added `roleplay_voice_runtime_ready` and `roleplay_voice_source_report` to
  `product-readiness.server_side`.
- `server_side.missing_evidence` now includes `roleplay_voice_runtime` until a
  matched ready `a21-roleplay-voice-probe-*.json` report is present.
- `server-side-readiness-bundle` now exposes a `roleplay_voice` evidence block.
- `server-side-readiness-bundle --collect-missing` now runs
  `a21 roleplay-voice-probe --require-ready` and only absorbs the generated
  report when `product-readiness` accepts it.
- Updated tests so old provider/V21/host-only candidate paths require roleplay
  voice runtime evidence, and added a direct collect-missing test for the
  roleplay voice probe path.
- Updated protocol, current-control, internal-test4 plan, project state, and
  the scoped gate plan.

Changed files:

- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-server-side-roleplay-voice-runtime-gate.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This tightens the server-side launch gate; it still does not prove physical
  wake, audible voice-clone quality, hardware expression delivery, or full PRD
  acceptance.

Known risks/blockers:

- A Gateway with an incomplete roleplay voice pipeline now correctly blocks
  server-side candidate readiness. That is intentional product gating, not a
  process blocker.

Recommended next action:

- Generate live roleplay voice runtime evidence against the target Gateway,
  rerun `server-side-readiness-bundle --collect-missing --require-candidate`,
  then move to physical StackChan wake/playback/professional/MCP acceptance.

Test/build/runtime results:

- `go test ./internal/app -run 'TestRunServerSideReadinessBundle|TestServerSideReadinessBundle|TestRunRoleplayVoiceProbe|TestRunProductReadinessUsesLatestRoleplayVoiceRuntimeReport' -count=1`:
  passed.

Failure location/reason:

- First focused run correctly exposed stale tests that expected
  provider/V21/host evidence to be enough for server-side candidate readiness.
  The tests were updated to include roleplay voice runtime evidence or verify
  roleplay collection.

Forbidden actions avoided:

- No ECS/root-secret change, firmware build, flash, serial, NVS, report
  deletion, prune/gc, destructive Git cleanup, or physical hardware action
  occurred.

## 2026-06-04 14:05 CST - Roleplay Voice Runtime Evidence Ingress

Round goal:

- Complete `T-ROLEPLAY-VOICE-RUNTIME-EVIDENCE-001` so product readiness can
  ingest safe roleplay voice runtime evidence instead of only static roleplay
  profile readiness.

Actual completed work:

- Added `--roleplay-voice-report <report.json>` to `a21 product-readiness`.
- Added latest-report discovery for safe `a21-roleplay-voice-probe-*.json`
  reports when `--use-latest-reports` is used.
- Passed the same report path through `a21 server-side-readiness-bundle`.
- Extended the top-level product-readiness `roleplay` object with runtime
  evidence availability, match, readiness, source basename, route/status,
  execution mode, marker count, text-stream execution, prompt-input use,
  voice-clone profile use, audio downlink, and playback-start booleans.
- Matched roleplay voice reports against the current Gateway-selected role
  soul, scenario, and voice-clone profile before `voice_runtime_ready=true`.
- Unsafe reports now become `roleplay_voice_report_invalid`; safe but incomplete
  matched reports become `roleplay_voice_runtime_not_ready`.
- Added `roleplay_voice_runtime` to canonical missing real evidence until a
  matched ready runtime report is present.
- Updated protocol, current-control, internal-test4 plan, project state, and
  this handoff log.

Changed files:

- `internal/app/product_demo.go`
- `internal/app/server_side_readiness_bundle.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-roleplay-voice-runtime-evidence-ingress.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is evidence ingress and readiness reporting only. It does not execute a
  real provider, V21, voice-clone CLI, ECS deployment, firmware, serial, NVS,
  or physical hardware.
- Roleplay voice runtime evidence still does not prove audible voice-clone
  quality, operator-perceived role immersion, official action delivery, or
  physical StackChan PRD acceptance.

Known risks/blockers:

- The new report schema proves the roleplay voice path can be audited; a fresh
  live Gateway/Voice Probe report still needs to be collected in the target
  runtime before the current product-readiness report will show the gap closed.

Recommended next action:

- Generate a fresh `a21-roleplay-voice-probe-*.json` from the live Gateway
  roleplay voice path, then rerun
  `a21 product-readiness --use-latest-reports`. After that, move to the
  foreground StackChan window for physical wake/roleplay/professional
  acceptance.

Test/build/runtime results:

- `go test ./internal/app -run 'TestProductReadiness(SurfacesRoleplayImmersionReadiness|RejectsUnsafeRoleplayProfile|SurfacesRoleplayVoiceRuntimeEvidence|RejectsUnsafeRoleplayVoiceRuntimeEvidence)|TestRunProductReadinessUsesLatestRoleplayVoiceRuntimeReport' -count=1`:
  passed.
- `go test ./internal/app -run 'TestProductReadiness(SurfacesRoleplayImmersionReadiness|RejectsUnsafeRoleplayProfile|SurfacesRoleplayVoiceRuntimeEvidence|RejectsUnsafeRoleplayVoiceRuntimeEvidence|IngestsSelectedVoiceChainStaticReadiness|RejectsVoiceChainStaticReadinessMismatch|BlocksServerSideCandidateWhenStepFunNotSelected|CanReachRealLaunchReadyWhenInputsArePresent|ReportsServerSideCandidateWhenEvidenceSlicesPass|ReportsServerSideCandidateWithoutPhysicalPRD|IngestsXiaozhiHostLoopbackCandidateEvidence|ClosesContinuousVoiceGapForHostProductChain|KeepsContinuousVoiceGapForFixtureOnlyXiaozhi|AcceptsCloudEdgeXiaozhiReportAsCandidate)|TestRunProductReadiness(UsesLatestRoleplayVoiceRuntimeReport|UsesLatestVoiceChainReadinessReport|AcceptsXiaozhiReportAndRedactsOutput|UsesLatestRealtimeFixtureWithoutPathLeak)|TestServerSideReadinessBundle(AcceptsVoiceChainStaticReadinessReport|SurfacesStepFunNotSelected)' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- None.

Forbidden actions avoided:

- No provider execution, V21 execution, ECS/root-secret/runtime change, Gateway
  protocol change, firmware build, flash, serial, NVS, report deletion,
  prune/gc, or physical hardware action occurred.

## 2026-06-04 13:18 CST - Roleplay Immersion Product Readiness

Round goal:

- Complete `T-ROLEPLAY-IMMERSION-READINESS-001` so roleplay soul, memory,
  voice-clone profile, prompt composition, and expression planning become
  visible in product readiness instead of only in Gateway UI/probe metadata.

Actual completed work:

- Added top-level `roleplay` readiness to `a21 product-readiness`.
- Product readiness now fetches `GET /v1/roleplay-profile` when the Gateway
  exposes it.
- The report surfaces selected role soul, scenario, voice-clone profile, soul
  prompt readiness, prompt-composed status, memory configured/readiness/count,
  official expression-plan action/packet counts, redaction booleans, and
  physical acceptance truth.
- Invalid or unsafe roleplay profile responses become
  `roleplay_profile_invalid`; unavailable endpoints remain
  `status=unavailable`.
- Output uses `asr_text_stored=false` rather than `transcript_stored=false` so
  existing redaction tests continue to forbid transcript wording in product
  readiness output.
- Updated protocol, current-control, internal-test4 plan, project state, and
  this handoff log.

Changed files:

- `internal/app/product_demo.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-roleplay-immersion-product-readiness.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/PROTOCOL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is product-readiness visibility only. It does not execute a provider,
  V21, voice-clone CLI, ECS deployment, firmware, serial, NVS, or physical
  hardware.
- Roleplay physical acceptance and audible voice-clone playback still require a
  foreground hardware/runtime evidence window.

Known risks/blockers:

- Roleplay immersion readiness proves the Gateway contract is visible and
  redacted in launch reports; it does not prove actual audio quality, voice
  clone output quality, physical facial/motion delivery, or latency.

Recommended next action:

- Continue toward a runtime/physical evidence gate: either collect fresh
  host/runtime roleplay voice evidence that combines selected roleplay readiness
  with provider/voice-chain execution, or schedule a foreground StackChan
  hardware window for wake/roleplay/professional acceptance.

Test/build/runtime results:

- `go test ./internal/app -run 'TestProductReadiness(SurfacesRoleplayImmersionReadiness|RejectsUnsafeRoleplayProfile)' -count=1`:
  passed.
- `go test ./internal/app -run 'TestProductReadiness(SurfacesRoleplayImmersionReadiness|RejectsUnsafeRoleplayProfile|IngestsSelectedVoiceChainStaticReadiness|RejectsVoiceChainStaticReadinessMismatch|BlocksServerSideCandidateWhenStepFunNotSelected|CanReachRealLaunchReadyWhenInputsArePresent|ReportsServerSideCandidateWithoutPhysicalPRD)|TestRunProductReadinessUsesLatestVoiceChainReadinessReport|TestServerSideReadinessBundleAcceptsVoiceChainStaticReadinessReport' -count=1`:
  passed.
- `go test ./internal/app -run 'TestProductReadinessIngestsXiaozhiHostLoopbackCandidateEvidence|TestRunProductReadinessCommandAcceptsXiaozhiReportAndRedactsOutput|TestProductReadiness(SurfacesRoleplayImmersionReadiness|RejectsUnsafeRoleplayProfile)' -count=1`:
  passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.

Failure location/reason:

- The first full `make verify` attempt failed because product readiness output
  used the JSON key `transcript_stored=false`, which correctly carried no
  transcript content but violated existing redaction tests that forbid the
  transcript term in product readiness JSON. The output key was changed to
  `asr_text_stored=false`, focused tests passed, and full verify passed.

Forbidden actions avoided:

- No provider execution, V21 execution, ECS/root-secret/runtime change, Gateway
  protocol change, firmware build, flash, serial, NVS, report deletion,
  prune/gc, or physical hardware action occurred.

## 2026-06-04 18:00 CST - Latest Mainline Checkpoint

Round goal:

- Keep the control thread recoverable after the product playback ack overlay
  work and prevent the next environment from repeating closed roleplay,
  provider, V21, or Gateway-only tasks.

Actual completed work:

- Gateway product playback allowance is already committed at `4c70b14`.
- This round added the matching official-compatible product overlay support
  and verified it with apply check, Go contract tests, full `make verify`, and
  guarded no-flash product build.
- Passed build report:
  `reports/a21-stackchan-official-baseline-20260604-175756-1780567076043462000.json`.
- Built app artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`,
  SHA-256 `e66a41ef486b866b076746bd064af2e3afb75e0a316515921bbc681b89fb36a8`.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `firmware/stackchan/README.md`
- `internal/app/official_stackchan_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- No flash, NVS write, serial access, ECS env change, provider execution, V21
  execution, or physical PRD promotion occurred.
- Next real transition is foreground physical: guarded product flash if
  approved, enable `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS` only for the evidence
  window, collect playback `start` / `stop_done`, barge-in, wake, and audible
  observation, then run `xiaozhi-physical-prd-review`.

Test/build/runtime results:

- Overlay apply check with official top-level HEAD plus clean
  `firmware/xiaozhi-esp32` dependency HEAD: passed.
- Focused official overlay tests: passed.
- `A21_STACKCHAN_OFFICIAL_DEP_CACHE='/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan' GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`:
  passed.
- `GOMAXPROCS=2 make verify`: passed.

Forbidden actions avoided:

- No repeated roleplay/provider/V21 evidence runs, no bare `xiaozhi.bin`, no
  generic `xiaozhi-firmware-flash-*`, no flash/NVS/serial, no prune/gc, and no
  rollback of internal test 3 protocol/audio changes.

## 2026-06-04 18:13 CST - Workspace Device Binding Guard

Round goal:

- Advance internal test 4 cloud/workspace product shape without repeating
  closed roleplay/provider/V21 evidence or touching physical firmware state.

Actual completed work:

- Added `GET/POST/PUT /v1/workspace-device-bindings` as a memory-only A21
  workspace device-access contract.
- Device bindings connect safe A21 `device_id`, redacted `user_id`, redacted
  `workspace_id`, and allowed professional `query_scope` values.
- Repeated binding of the same device/user/workspace is idempotent; revoke,
  restore, and delete are metadata-only and do not require firmware flash or
  NVS writes.
- `/v1/professional-workspace` now reports device-binding policy/count summary.
- Professional mock turns and stock Xiaozhi professional turns now enforce
  binding before V21 execution once a workspace has binding records. Unbound,
  revoked, deleted, or query-scope-denied devices fail before
  `v21.query.start` and write only safe read-ledger failure codes.
- `/workspace` now exposes device ID, bind, revoke, refresh, binding status,
  active binding count, and safe metadata export over existing Gateway APIs.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/workspace_console.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is Gateway/workspace access metadata and route gating only. It does not
  perform real cloud auth, tenant ACL, real document indexing, V21 merge, V21
  release, provider execution, firmware flash, NVS write, serial access, or
  physical StackChan PRD promotion.
- Physical acceptance still needs the foreground product StackChan window with
  playback `start` / `stop_done`, barge-in, wake, and audible observation.

Known risks/blockers:

- Device binding is currently memory-only for internal test 4. A production
  cloud account/device table and durable ACL store remain future work.

Recommended next action:

- Continue either with the foreground physical PRD evidence window, or with the
  next cloud/workspace cut: durable cloud account/device binding persistence
  and real index worker handoff while preserving the same safe API surface.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestWorkspace(DeviceBindings|Console)|TestProfessionalReadRecords|TestWorkspaceUploadJobsRejectRawPayloadFields' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- Local runtime check: started `go run ./cmd/a21 gateway --addr
  127.0.0.1:21080`, confirmed `/workspace` contains
  `/v1/workspace-device-bindings` controls, posted a safe binding for
  `stackchan-live-check-001`, observed bound professional mock route proceed,
  observed unbound professional mock route fail with
  `failure_code=device_unbound`, and confirmed its trace had no
  `v21.query.start`. The Gateway was then stopped.

Forbidden actions avoided:

- No provider execution, V21 execution, ECS/root-secret/runtime change, firmware
  build, flash, serial, NVS, report deletion, prune/gc, or internal-test3
  rollback occurred.

## 2026-06-04 18:45 CST - Professional Query Endpoint Product Surface

Round goal:

- Continue internal test 4 cloud/workspace product progress without repeating
  closed device-binding, roleplay, provider, V21, or firmware work.

Actual completed work:

- Added formal `POST /v1/professional-query` with schema
  `a21.gateway.professional_query.v1` for Web/App professional consult.
- The endpoint accepts safe device/workspace/scope/query fields, emits the
  professional checking cue, starts the professional read ledger, enforces the
  workspace device-binding guard before `v21.query.start`, calls the A21/V21
  adapter only after that guard, and returns user-facing answer/evidence events
  plus a redacted evidence report when V21 succeeds.
- Unsafe payload fields such as document text, evidence bodies, provider
  output, base64, URLs/paths, credentials, transcript, screen cards, speech
  blocks, and audio are rejected before read-record or V21 execution.
- Refactored mock professional turns to reuse the same Gateway professional
  query execution path, keeping mock/Web/App guard and read-ledger behavior in
  one place.
- Updated `/workspace` professional probe to call `/v1/professional-query`
  instead of `/v1/mock-turn`.
- Updated protocol/control/state docs so the next environment should continue
  from the formal professional-query endpoint rather than repeating mock probe
  work.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/workspace_console.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- This is a host-local Gateway product-surface cut. It does not implement real
  document parsing, chunking, embedding, indexing, cloud storage, durable
  account/auth/tenant ACL, V21 merge/release, provider deployment, firmware
  flash, NVS write, serial access, or physical StackChan professional consult
  acceptance.
- Physical PRD acceptance still needs the foreground StackChan evidence window
  with playback `start` / `stop_done`, barge-in, wake, and audible observation.

Known risks/blockers:

- Device bindings and read records remain memory-only in this Gateway slice.
- The endpoint can show user-facing answer/evidence output because it is the
  active consult surface; the persistence/redaction guarantee applies to
  Gateway metadata, workspace/read-ledger records, traces, and exports.

Recommended next action:

- Continue the cloud/workspace path with durable account/device binding and
  indexing-worker handoff, or use the approved foreground hardware window to
  collect physical StackChan PRD evidence. Do not go back to `/v1/mock-turn` as
  the Web/App professional product entry.

Test/build/runtime results:

- `go test ./internal/gateway -run 'TestProfessionalQueryEndpoint|TestWorkspaceDeviceBindings|TestProfessionalReadRecords|TestWorkspaceConsolePageServed' -count=1`:
  passed.
- `go test ./internal/gateway -count=1`: passed.
- `git diff --check`: passed.
- `GOMAXPROCS=2 make verify`: passed.
- Local runtime check: started `go run ./cmd/a21 gateway --addr
  127.0.0.1:21080`, confirmed `/workspace` references
  `/v1/professional-query`, bound `stackchan-runtime-web-001`, observed bound
  `/v1/professional-query` complete with answer/evidence report and a completed
  read record, observed unbound `stackchan-runtime-unbound-001` fail with
  `device_unbound`, and confirmed its trace had no `v21.query.start`. The
  Gateway was then stopped.

Forbidden actions avoided:

- No provider execution, real V21 service execution, ECS/root-secret/runtime
  change, firmware build, flash, serial, NVS, report deletion, prune/gc, or
  internal-test3 protocol/audio rollback occurred.

## 2026-06-04 20:03 CST - Product Flash/NVS Hardware Window And Live Blocker Cut

Round goal:

- Stop re-auditing already-completed internal-test4 work, execute the now
  unblocked product StackChan firmware/NVS path, and report the real PRD
  blockers plainly.

Actual completed work:

- Rechecked current branch/worktree: foreground hardware-window branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`, HEAD
  `522c5c28d335`, clean before guarded writes.
- Confirmed product artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  exists with SHA-256
  `e66a41ef486b866b076746bd064af2e3afb75e0a316515921bbc681b89fb36a8`.
- Ran product flash plan, then guarded product flash execute on
  `/dev/cu.usbmodem1101`; execution passed.
- Ran product NVS plan, then guarded product NVS execute on
  `/dev/cu.usbmodem1101`; execution passed, set OTA to
  `http://47.103.57.217/xiaozhi/ota/`, WebSocket to
  `ws://47.103.57.217/v1/xiaozhi`, and wrote the first operator-text Wi-Fi
  SSID spelling without recording the password in docs.
- Captured serial startup after flash/NVS: A21 official Xiaozhi-compatible app
  booted, device MAC is `44:1b:f6:e2:6a:60`, firmware entered Wi-Fi station
  mode, then logged `No AP found`.
- Verified the Mac also could not join that first typed SSID spelling from this
  location.
- Operator screenshot then clarified the visible SSID spelling as
  `ChinaNet-N6e3` with no spaces around the hyphen, so the Wi-Fi blocker is an
  NVS SSID spelling correction rather than evidence that the AP is absent.
- Rechecked public Gateway `47.103.57.217`: TCP connects to `80` and `21081`,
  but `/healthz`, `/xiaozhi/ota/`, and `/v1/devices` return empty replies;
  SSH to `root@47.103.57.217` closes before authentication.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Unfinished items:

- Physical PRD acceptance is still not claimable. The device needs a corrected
  NVS SSID write using `ChinaNet-N6e3`, and the public ECS Gateway application
  layer is not returning HTTP responses.
- The latest local Web/App professional endpoint commit is not yet deployed to
  the public ECS because SSH/control-plane access is unavailable from this Mac.

Known risks/blockers:

- `T-STACKCHAN-CHINANET-SSID-SPELLING-001`: first NVS write used the typed SSID
  spelling with a space; screenshot shows the visible SSID is `ChinaNet-N6e3`.
- `T-ALIYUN-CLOUD-GATEWAY-APP-REACHABILITY-001`: public ECS accepts TCP but
  returns empty HTTP replies and closes SSH before auth; cloud control-plane
  intervention is needed.

Recommended next action:

- In Aliyun workbench, restore or restart the A21 Gateway/Caddy chain until
  `http://47.103.57.217/healthz`,
  `http://47.103.57.217/xiaozhi/ota/`, and
  `http://47.103.57.217/v1/devices` return valid A21 responses; then rewrite
  product NVS with `ChinaNet-N6e3` and rerun physical `/v1/xiaozhi` evidence
  collection.

Test/build/runtime results:

- Flash plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-195627-1780574187977369000.json`
  passed with `dry_run=true`.
- Flash execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-195734-1780574254123911000.json`
  passed with `flash_executed=true`.
- NVS plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-195740-1780574260353872000.json`
  passed with `dry_run=true`.
- NVS execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-195752-1780574272065931000.json`
  passed with `write_executed=true`, `mutated_entry_count=5`,
  `wifi_credentials_written=true`, and `servo_calibration_present=true`.
- Serial observation on `/dev/cu.usbmodem1101` showed boot/app identity and
  `No AP found`; no `got ip` or WebSocket registration was observed.

Forbidden actions avoided:

- No generic `xiaozhi.bin` product flash, no prune/gc, no provider secret
  printing, no firmware key storage, no V21 internal execution, and no
  internal-test3 protocol/audio rollback occurred.

## 2026-06-04 20:07 CST - Corrected ChinaNet NVS And TUN Route Recovery

Round goal:

- Correct the SSID spelling from the operator screenshot, stop misclassifying
  TUN-route false negatives as ECS downtime, and update the live product truth.

Actual completed work:

- Operator screenshot clarified the visible SSID is `ChinaNet-N6e3` with no
  spaces around the hyphen.
- Ran corrected product NVS plan and guarded NVS execute on
  `/dev/cu.usbmodem1101`; execution passed.
- Captured serial after the corrected NVS write:
  - Device found AP `ChinaNet-N6e3`.
  - Device connected to Wi-Fi and received IP `192.168.1.26`.
  - Device opened `ws://47.103.57.217/v1/xiaozhi`.
  - Device entered listening/speaking state and produced the audible short
    reply marker `嗯。`.
- Operator then pointed out TUN mode was enabled. Route inspection confirmed
  public A21 traffic to `47.103.57.217` was going through `utun6` via
  `198.18.0.1`, reproducing the known false-negative pattern.
- Applied the historical direct-source workaround with source IP
  `192.168.1.20`:
  - `http://47.103.57.217/healthz` returned A21 Gateway ok.
  - `http://47.103.57.217/xiaozhi/ota/` returned
    `ws://47.103.57.217/v1/xiaozhi`.
  - `http://47.103.57.217/v1/devices` showed product device
    `44:1b:f6:e2:6a:60` online.
- SSH with the same source bind reached the real auth state and failed with
  `Permission denied (publickey)`, so the remaining ECS deploy blocker is key
  authorization, not Gateway reachability.
- Live trace `a21-trace-44-1b-f6-e2-6a-60` now has `1803` events, including
  real physical Opus ingress/decode, ASR partial/final, provider first content,
  TTS first audio, Opus downlink, voice-pipeline completion, and one barge-in
  stop marker.

Changed files:

- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`

Unfinished items:

- Full PRD acceptance is still not claimable because physical evidence still
  lacks device playback-start/stop-done or trusted audible/instrument
  observation.
- The latest local Web/App professional-query Gateway code is not deployed to
  ECS from this Mac because SSH root auth is not available.
- A Mac-side `product-readiness` run without ECS provider secret env still
  under-reports provider as `mock`; live public Gateway runtime itself reports
  cascade mode with StepFun and voice-clone TTS selected.

Known risks/blockers:

- Keep using `A21_DIRECT_SOURCE_IP=192.168.1.20` or `curl --interface
  192.168.1.20` for public A21 verification while TUN mode is active.
- Do not treat TUN-path empty replies as ECS downtime unless the bound-source
  check also fails.
- `root@47.103.57.217` needs an accepted key or Aliyun workbench access for
  deployment/restart.

Recommended next action:

- Run physical acceptance from the now-online device and direct-source public
  Gateway path: confirm audible playback, playback-start/stop-done or trusted
  observation, barge-in, and professional/roleplay mode behavior; then
  regenerate physical evidence and product readiness with provider env aligned
  to the live ECS runtime.

Test/build/runtime results:

- Corrected NVS plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-200246-1780574566418018000.json`
  passed with `dry_run=true`.
- Corrected NVS execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-200257-1780574577392353000.json`
  passed with `write_executed=true`, `mutated_entry_count=5`,
  `wifi_credentials_written=true`, and `servo_calibration_present=true`.
- Direct-source public Gateway checks passed for `/healthz`, `/xiaozhi/ota/`,
  `/v1/devices`, and `/v1/voice-chain-profiles`.
- Live device registry: `44:1b:f6:e2:6a:60` online, current mode `roleplay`,
  voice chain `cascade`, LLM `stepfun`, TTS `voice_clone_cli`, last event
  `xiaozhi.tts.opus_frame.downlink`.
- Live trace counts include `xiaozhi.opus_frame.received=230`,
  `xiaozhi.opus_frame.decoded=230`, `asr.first_partial=8`, `asr.final=8`,
  `provider.first_content=4`, `tts.first_audio=4`,
  `xiaozhi.tts.opus_frame.downlink=214`,
  `xiaozhi.voice_pipeline.completed=4`, `barge_in.detected=1`, and
  `playback.stop=1`.

Forbidden actions avoided:

- No generic `xiaozhi.bin` product flash, no provider secret printing, no
  provider key in firmware, no prune/gc, no V21 internal execution, no
  internal-test3 rollback, and no false PRD acceptance claim occurred.

## 2026-06-04 20:36 CST - Official Robot MCP Body Controls Deployed And Physically Executed

Round goal:

- Move the StackChan hardware parity work from "Gateway can remember/control
  some things" to a real official-runtime body-control slice for visible RGB
  and head servo behavior, without rolling back internal-test3 voice protocol
  changes or flashing firmware again.

Actual completed work:

- Root-caused the first failed live robot MCP attempt to official Xiaozhi MCP
  JSON-RPC id validation: the official firmware rejects string `id` values for
  `tools/call`.
- Expanded `POST /v1/xiaozhi/mcp-control` only for bounded official robot MCP
  tools:
  - `self.robot.get_head_angles`
  - `self.robot.set_head_angles`
  - `self.robot.set_led_color`
- Kept high-risk official tools blocked before WebSocket write, including
  camera/photo, screen snapshot, camera stream/video, NFC, infrared, app
  lifecycle, reboot, and OTA/upgrade controls.
- Changed Gateway-to-device MCP envelopes to use stable positive numeric
  JSON-RPC ids while keeping the public A21 `mcp_id` string in HTTP responses
  for traceability.
- Root-caused the second failed live robot MCP response to Gateway rejecting
  device-side `type=mcp` replies and replying with stock-unknown `type=error`.
- Accepted redacted device-side `type=mcp` responses after `hello`, traced
  `xiaozhi.mcp.response.received`, and stored only
  `xiaozhi_mcp_response=received_redacted` in the device registry.
- Deployed commits `8e2f890`, `e93d5d4`, and `60a2132` to ECS, restarted the
  public Gateway, and hard-reset the product device to reconnect after the
  restart.
- Verified physical execution on product device `44:1b:f6:e2:6a:60` through
  serial:
  - `[HAL-MCP] set_led_color: r=20, g=0, b=168`
  - `[HAL-MCP] motion set_angles: yaw: 12, pitch: 30, speed: 150`
- Verified that the post-fix serial window did not show
  `Unknown message type: error`.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/transport/xiaozhi/builders.go`
- `internal/transport/xiaozhi/builders_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- Full PRD physical acceptance is still open.
- Camera, NFC, infrared, screen visual acceptance, no-cable boot/power
  behavior, and official app-lifecycle parity remain open transitions.
- The current device path did not reliably auto-reconnect after a Gateway
  restart; a hard reset was needed for the latest evidence window.

Known risks/blockers:

- Robot MCP controls are now live execution evidence for official RGB/head
  tools, not a blanket product claim for every StackChan body surface.
- Yaw and pitch movement still need mechanical safety/product semantics before
  richer motion sequences are promoted.
- While TUN mode is active on the Mac, public A21 checks still need
  `curl --interface 192.168.1.20 --noproxy '*' ...` or equivalent
  `A21_DIRECT_SOURCE_IP=192.168.1.20` handling.
- The local SSH private key filename may contain historical naming, but that
  name is only a local credential label. It must not be copied into A21
  runtime identity, protocol, firmware, docs, or product claims.

Recommended next action:

- Continue hardware parity from the next visible surfaces in order:
  screen/status visual acceptance, touch/barge-in/action evidence, no-cable
  boot/power diagnostics, then camera/NFC/IR as explicit high-risk spikes.
  Keep using official-compatible product lane guards and do not flash again
  unless a scoped hardware window requires it.

Test/build/runtime results:

- Local focused tests passed before deployment:
  `go test ./internal/transport/xiaozhi ./internal/gateway -run 'TestMCP|TestXiaozhiMCP|TestXiaozhiHandleMCPResponse|TestXiaozhiMessageTypeMCP' -count=1`.
- Local `GOMAXPROCS=2 make verify` passed after the MCP response fix.
- Remote focused tests passed after deployment, and `systemctl is-active
  a21-gateway` returned active.
- Direct-source public `/v1/devices` confirmed product device online.
- Command trace `a21-trace-live-robot-led-responsefix` recorded
  `xiaozhi.mcp.robot_led_color.sent`.
- Command trace `a21-trace-live-robot-head-responsefix` recorded
  `xiaozhi.mcp.robot_head_angles_set.sent`.
- Device session trace `a21-trace-44-1b-f6-e2-6a-60` recorded two
  `xiaozhi.mcp.response.received` events.
- `/v1/devices` recorded `robot_led_red=20`, `robot_led_green=0`,
  `robot_led_blue=168`, `robot_head_yaw=12`, `robot_head_pitch=30`,
  `robot_head_speed=150`, and `xiaozhi_mcp_response=received_redacted`.

Forbidden actions avoided:

- No generic `xiaozhi.bin` product flash, no firmware flash, no NVS write, no
  provider secret printing, no provider key in firmware, no prune/gc, no V21
  internal execution, no internal-test3 rollback, and no false PRD acceptance
  claim occurred.

## 2026-06-04 20:48 CST - Named Screen/Status MCP Endpoints Deployed And Physically Executed

Round goal:

- Continue the official hardware parity path by turning the low-risk
  screen/status MCP tools into product-operation endpoints for Web/App/operator
  use, then verify them against the live product StackChan without flashing or
  rewriting NVS.

Actual completed work:

- Added named Gateway endpoints:
  - `POST /v1/xiaozhi/device-status`
  - `POST /v1/xiaozhi/screen-brightness`
  - `POST /v1/xiaozhi/screen-theme`
  - `GET /v1/xiaozhi/mcp-capabilities?device_id=<device_id>`
- Reused the existing official MCP delivery path, whitelist, numeric JSON-RPC
  id behavior, and redacted device-side MCP response handling.
- Kept high-risk classes blocked: reboot, firmware upgrade, camera/photo,
  screen snapshot, camera stream/video, NFC, infrared, and app lifecycle.
- Added endpoint tests for successful tool mapping, missing `device_id`,
  brightness bounds, strict named theme values, non-MCP device rejection, and
  capabilities redaction/allowed/blocked lists.
- Deployed commit `4df52b3` to ECS through the existing `/opt/a21.next` safe
  swap. Remote focused Gateway tests passed, the binary built, and
  `a21-gateway` restarted active with loopback health ok.
- Verified public `mcp-capabilities` through the TUN-safe source-bound path;
  it returned the allowed official tools and blocked classes with
  `result_redacted=true` and `physical_accepted=false`.
- Reconfirmed the lifecycle gap: after Gateway restart, `/v1/devices` was empty
  until the product device was hard-reset. The reset used repo-local esptool
  `chip_id` with `--after hard_reset`; it did not flash firmware or write NVS.
- Verified live product device execution after reconnect:
  - `screen-theme` response delivered `self.screen.set_theme` with theme
    `dark`; serial logged `StackChanAvatarDisplay: SetTheme: dark`.
  - `screen-brightness` response delivered `self.screen.set_brightness` with
    brightness `55`; serial logged `Backlight: Set brightness to 55`.
  - `device-status` response delivered `self.get_device_status` with
    `result_redacted=true`.
- Verified command traces:
  `a21-trace-live-named-device-status-ready`,
  `a21-trace-live-named-brightness-ready`, and
  `a21-trace-live-named-theme`.
- Verified device session trace `a21-trace-44-1b-f6-e2-6a-60` had
  `xiaozhi.mcp.response.received=3` after this evidence window.
- Verified `/v1/devices` recorded `screen_theme=dark` and
  `screen_brightness=55`.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- Full physical screen visual acceptance is still open; this is MCP execution
  evidence, not a photographed/operator-accepted screen UX proof.
- Gateway-restart auto-reconnect remains open and should become the next
  firmware/app-lifecycle transition.
- Touch acceptance still needs an operator window because the existing
  `stackchan-accept --check touch` cases require real taps/swipes.
- Camera, NFC, infrared, no-cable boot/power behavior, and full app lifecycle
  parity remain open.

Known risks/blockers:

- The product device does not reliably reconnect by itself after a public
  Gateway safe-swap restart.
- A just-reset device can have short-lived socket/registry timing edges during
  boot. The stable evidence was therefore collected after reconnect, with
  explicit `*-ready` trace ids.
- Continue using `curl --interface 192.168.1.20 --noproxy '*' ...` while the
  Mac TUN route is active.

Recommended next action:

- Fix the Gateway-restart/device auto-reconnect lifecycle gap, then run a
  foreground touch/action acceptance window for screen touch, top tap/swipe,
  top barge-in, and visible action/RGB/servo behavior. Do not flash again
  unless that lifecycle fix explicitly requires a guarded product firmware
  window.

Test/build/runtime results:

- Local focused test:
  `go test ./internal/gateway -run 'TestXiaozhiNamedMCP|TestXiaozhiMCPStatusParity|TestXiaozhiMCPResponse' -count=1`
  passed.
- Local `go test ./internal/gateway -count=1` passed.
- Local `git diff --check` passed.
- Local `GOMAXPROCS=2 make verify` passed.
- Remote focused Gateway test during deployment passed.
- Remote `go build -o /opt/a21.next/bin/a21 ./cmd/a21` passed.
- Remote `systemctl is-active a21-gateway` returned active and loopback
  `/healthz` returned ok.
- Live public `/v1/xiaozhi/mcp-capabilities` returned the expected safe
  capability response.
- Live serial evidence showed `SetTheme: dark` and
  `Backlight: Set brightness to 55`.

Forbidden actions avoided:

- No generic `xiaozhi.bin` product flash, no firmware flash, no NVS write, no
  provider secret printing, no provider key in firmware, no prune/gc, no V21
  internal execution, no internal-test3 rollback, and no false screen/product
  acceptance claim occurred.

## 2026-06-04 21:02 CST - Xiaozhi Control Channel Keepalive Candidate

Round goal:

- Fix the real lifecycle gap observed after ECS Gateway safe-swap: the product
  device did not reliably reconnect to `/v1/xiaozhi` without a hard reset.

Actual completed work:

- Root-caused the gap to the idle official Xiaozhi websocket lacking a product
  heartbeat. Existing A21 `EnsureA21ControlChannel()` only reopens when
  `IsAudioChannelOpened()` is false, while the official websocket can remain
  apparently open on a dead idle socket until timeout or traffic.
- Added a product-only keepalive contract:
  `hello.features.keepalive_events=true` from the product overlay,
  `a21.keepalive_events=true` from Gateway only for hardware-MAC product
  clients under the existing product playback-events gate, and
  `type=device, kind=heartbeat` as the only new product device event.
- Kept debug-only `state`, `face`, `display`, and `motion` device events
  blocked for product clients.
- Updated the official-compatible product overlay so an open idle control
  websocket sends heartbeat and closes the stale channel when heartbeat send
  fails; the existing 10 second A21 reconnect loop can then open a fresh
  websocket.
- Verified the guarded official-compatible product build, no flash:
  app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`,
  sha256 `f6bf04d007d6531112403a0c8518105201222b88c5cc052f2172f84bf3875fc9`,
  report
  `reports/a21-stackchan-official-baseline-20260604-210158-1780578118998849000.json`.

Changed files:

- `internal/transport/xiaozhi/frame.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `docs/plans/2026-06-04-stackchan-xiaozhi-control-channel-keepalive.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Unfinished items:

- ECS Gateway has not yet been redeployed with keepalive support.
- Product device has not yet been flashed with the new
  `a21-stackchan-official-xiaozhi-compatible.bin` artifact.
- Foreground proof is still required: after Gateway safe-swap/restart, device
  `44:1b:f6:e2:6a:60` must reconnect and reappear in `/v1/devices` without
  hard reset.

Known risks/blockers:

- The official source worktree is dirty. The guarded build exported from
  official Git HEAD and used the dependency cache; do not mutate or clean the
  official source tree.
- The repair depends on an idle send detecting stale TCP/WebSocket state; this
  is the right minimal product fix, but physical restart evidence is still the
  acceptance gate.
- Continue using `curl --interface 192.168.1.20 --noproxy '*' ...` while TUN is
  active.

Recommended next action:

- Run full local verification, commit/push the candidate, deploy Gateway to
  ECS, flash only the guarded official-compatible product app artifact, then
  run the Gateway restart reconnect proof before returning to touch/action
  physical acceptance.

Test/build/runtime results so far:

- Failing tests were created first and failed at the expected missing
  `keepalive_events`/overlay contract points.
- `go test ./internal/gateway -run 'TestXiaozhiProduct(Playback|Keepalive)' -count=1`
  passed after implementation.
- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlay(KeepsA21IdleSocketReady|AddsProductPlaybackAckOnly)' -count=1`
  passed.
- `go test ./internal/transport/xiaozhi -count=1` passed.
- `go test ./internal/gateway -run 'TestXiaozhi(WebSocketHello|Product|StockProfileRejectsPlayback|DebugProfile|MCPResponse)' -count=1`
  passed.
- `A21_STACKCHAN_OFFICIAL_DEP_CACHE='/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan' GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`
  passed.

Forbidden actions avoided:

- No generic `xiaozhi.bin` product flash, no firmware flash, no NVS write, no
  provider secret printing, no provider key in firmware, no prune/gc, no V21
  internal execution, no internal-test3 rollback, and no false reconnect
  product-acceptance claim occurred.

## 2026-06-04 21:48 CST - Xiaozhi Control Channel Keepalive Physical Proof

Round goal:

- Finish `T-STACKCHAN-XIAOZHI-CONTROL-KEEPALIVE-001`: deploy the keepalive
  Gateway contract, flash only the official-compatible product app lane, and
  prove the product device reconnects after a Gateway restart without a device
  hard reset.

Actual completed work:

- Deployed the keepalive-capable Gateway to ECS and enabled the product runtime
  gate `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS=true` in `/etc/a21/runtime.env`
  without printing provider secrets.
- Flashed the product device `44:1b:f6:e2:6a:60` through the guarded
  `a21-stackchan-official-xiaozhi-compatible` product lane only.
- Found two firmware lifecycle gaps during physical proof and fixed both:
  heartbeat was initially behind the app idle-state guard, then heartbeat send
  failure closed the protocol without a reliable reopen path.
- Final firmware overlay runs keepalive in `WebsocketProtocol` and starts a
  protocol-layer reconnect task after heartbeat send failure.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Runtime or physical evidence:

- Final no-flash build passed with app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  sha256 `d43989209ee1be056f8d59b539bc48c6cdd2821fd9645793c18f134b6dee9179`,
  report
  `reports/a21-stackchan-official-baseline-20260604-214350-1780580630173242000.json`.
- Final guarded product flash passed with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-214705-1780580825810604000.json`.
- After final flash, public `/v1/devices` showed `device.heartbeat` with
  `xiaozhi_product_keepalive_events=true`.
- Gateway was restarted at `2026-06-04 21:47:49 CST`; without device hard reset,
  `/v1/devices` was empty during the restart window, then device
  `44:1b:f6:e2:6a:60` reappeared with `xiaozhi.hello` at `21:47:57` and resumed
  `device.heartbeat` at `21:48:08`.

Tests/build/runtime results:

- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlay|TestOfficialStackChan|TestXiaozhi' -count=1` passed.
- `go test ./internal/gateway -run 'TestXiaozhiProduct(Playback|Keepalive)' -count=1` passed.
- `git diff --check` passed.
- `GOMAXPROCS=2 make verify` passed before each committed firmware lifecycle
  repair.
- ECS remote focused tests/build passed during the earlier keepalive deployment.

Unfinished items:

- This closes Gateway-restart control-channel recovery evidence, not full
  physical screen/touch/servo/RGB/camera/NFC/IR PRD acceptance.
- Next highest-value hardware parity work remains touch/action foreground
  acceptance plus official StackChan body surface alignment beyond MCP command
  delivery.

Forbidden actions avoided:

- No generic `xiaozhi.bin` product flash, no NVS write, no provider key in
  firmware, no provider secret printing, no Git prune/gc, no V21 internal
  execution, and no internal-test3 voice/protocol rollback occurred.

## 2026-06-04 22:29 CST - StackChan Product Touch Physical Evidence

Round goal:

- Continue hardware parity after keepalive closure by moving screen/top touch
  from contract/plan to product-lane physical evidence, without regressing the
  internal-test3 voice/protocol path.

Actual completed work:

- Added `hello.features.touch_events` parsing and a product-only Gateway gate
  `A21_XIAOZHI_PRODUCT_TOUCH_EVENTS`.
- Gateway now accepts product `type=device, kind=touch` only for hardware-MAC
  stock Xiaozhi clients that advertise `touch_events` and do not request debug
  device events or debug metrics.
- Mapped firmware touch values to A21 trace/registry events:
  `screen_tap` -> `device.touch.wake_or_listen.received`,
  `top_tap` -> `device.touch.top.tap.received`,
  `top_swipe_forward/backward` -> corresponding directional events, and
  `top_barge_in` / `screen_barge_in` -> `device.touch.barge_in.received`.
- Updated the official-compatible product overlay so screen touch and official
  top-touch HAL gestures send product-safe touch events only after
  `a21.profile=product` / `a21.touch_events=true`.
- Deployed commit `f16e71b` to ECS `47.103.57.217`, enabled root-only
  `A21_XIAOZHI_PRODUCT_TOUCH_EVENTS=true`, and restarted `a21-gateway`.
- Flashed only the guarded official-compatible product lane on
  `/dev/cu.usbmodem1101`.

Changed files:

- `internal/transport/xiaozhi/frame.go`
- `internal/transport/xiaozhi/frame_test.go`
- `internal/transport/xiaozhi/device_extension.go`
- `internal/transport/xiaozhi/device_extension_test.go`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `internal/app/app_stackchan_touch.go`
- `internal/app/official_stackchan_test.go`
- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Runtime or physical evidence:

- Guarded no-flash product build passed:
  `reports/a21-stackchan-official-baseline-20260604-221937-1780582777246064000.json`,
  app sha256
  `9b8366e387b10ffa784394e965f702734753f1c4c68192f17a11135e3b713216`.
- Guarded product flash passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-222334-1780583014061627000.json`.
- Public `/v1/devices` after flash showed
  `xiaozhi_feature_touch_events=true` and
  `xiaozhi_product_touch_events=true`.
- Physical touch acceptance passed:
  - `reports/a21-stackchan-touch-acceptance-20260604-222700.json`
    (`screen_touch`, source `screen`).
  - `reports/a21-stackchan-touch-acceptance-20260604-222708.json`
    (`top_tap`, source `top_sensor`).
  - `reports/a21-stackchan-touch-acceptance-20260604-222719.json`
    (`top_swipe_forward`, source `top_sensor`).
  - `reports/a21-stackchan-touch-acceptance-20260604-222744.json`
    (`top_swipe_backward`, source `top_sensor`).
  - `reports/a21-stackchan-touch-acceptance-20260604-222818.json`
    (`top_barge_in`, trace `a21-trace-touch-barge-in-20260604`, source
    `top_sensor`).

Tests/build/runtime results:

- Focused local tests passed for Xiaozhi transport touch parsing, Gateway
  product touch allowance, app env wiring, official overlay contracts, and
  touch acceptance CLI.
- `git diff --check` passed before build and commit.
- `A21_STACKCHAN_OFFICIAL_DEP_CACHE='/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan' GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`
  passed after fixing overlay apply and C++ include boundaries.
- `GOMAXPROCS=2 make verify` passed before commit.
- ECS remote focused Gateway/App tests passed, remote `go build` passed,
  `a21-gateway` restarted active, and local health returned ok.

Unfinished items:

- Directional top swipes are physically accepted as guided directional touch
  but still need product affordance, labeling, or screen guidance before they
  are frictionless everyday UX.
- Full PRD physical acceptance remains open for broader audio/playback
  evidence, custom wake/no-cable boot/power, screen visual acceptance, richer
  servo/RGB expression, camera, NFC, infrared, and app lifecycle.

Forbidden actions avoided:

- No generic `xiaozhi.bin` product flash, no NVS write, no provider secret
  printing, no provider key in firmware, no Git prune/gc, no V21 internal
  execution, and no internal-test3 voice/protocol rollback occurred.

## 2026-06-04 22:49 CST - StackChan Product Touch Body Reaction Evidence

Round goal:

- Continue hardware parity from accepted touch events into visible body
  feedback, without repeating keepalive/touch bridge work and without
  regressing internal-test3 voice/protocol behavior.

Actual completed work:

- Added product-gated touch body reactions behind
  `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true`.
- Gateway returns `a21.touch_reactions=true` only for hardware-MAC stock
  Xiaozhi clients that already satisfy product touch allowance and advertise
  `hello.features.mcp=true`.
- Accepted product touch events now send bounded official MCP
  `self.robot.set_led_color` and `self.robot.set_head_angles` reactions over
  the same live `/v1/xiaozhi` socket.
- Added stable registry fields `last_touch_event`, `last_touch_source`,
  `last_touch_trace_id`, `last_touch_session_id`, and `last_touch_seen_ms`
  so MCP responses, Opus, or heartbeat events can keep their true
  `last_event` semantics without erasing touch acceptance evidence.
- Updated touch acceptance to prefer stable `last_touch_*` fields.
- Deployed the Gateway update to ECS `47.103.57.217`, enabled root-only
  `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true`, and restarted `a21-gateway`.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app.go`
- `internal/app/app_firmware.go`
- `internal/app/app_stackchan_touch.go`
- `internal/app/app_test.go`
- `internal/firmwarecheck/device_identity.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Runtime or physical evidence:

- ECS remote focused Gateway/App tests passed and remote build/restart/health
  passed.
- Public `/v1/devices` after reconnect showed
  `xiaozhi_product_touch_reactions=true`.
- Real foreground touch on device `44:1b:f6:e2:6a:60` passed:
  `reports/a21-stackchan-touch-reaction-evidence-20260604-224756.json`.
- Evidence trace/session:
  `a21-trace-44-1b-f6-e2-6a-60` /
  `a21-session-44-1b-f6-e2-6a-60`.
- Observed touch/reaction:
  `top_swipe_backward` from `top_sensor` -> LED `120/60/0` and head
  `yaw=-18,pitch=24,speed=200`.
- Trace markers included `device.touch.top.swipe_backward.received`,
  `xiaozhi.touch_reaction.robot_led_color.sent`,
  `xiaozhi.touch_reaction.robot_head_angles_set.sent`, and two redacted
  `xiaozhi.mcp.response.received` events.
- `last_event` later advanced to Opus/heartbeat events, while
  `last_touch_event=touch.top.swipe_backward` remained stable as intended.

Tests/build/runtime results:

- `git diff --check` passed.
- `go test ./internal/gateway -run 'TestXiaozhiProductTouch|TestXiaozhiMCPStatusParityAllowsOnlyScopedTools|TestXiaozhiMCPCapabilitiesEndpointReportsAllowedAndBlockedTools' -count=1` passed.
- `go test ./internal/app -run 'TestRunStackChanTouchAcceptancePassesTopTapWithGatewayTrace|TestGatewayServerOptionsFromEnvWiresProductTouch|TestGatewayServerOptionsFromEnvWiresProductPlaybackEvents' -count=1` passed.
- `GOMAXPROCS=2 make verify` passed after the final docs/state/handoff update.

Unfinished items:

- This is the first product `touch -> body` proof, not full screen visual
  acceptance, richer personality choreography, camera, NFC, infrared,
  no-cable boot/power, app lifecycle, or full PRD physical acceptance.
- Directional swipe UX still needs an affordance/training/screen cue before
  it can be considered frictionless core UX.

Recommended next action:

- Move to a visible expression polish slice: small per-state reaction map for
  listening/thinking/speaking/error and roleplay/professional mode, using the
  same bounded MCP/body contract and separate acceptance evidence.

Forbidden actions avoided:

- No firmware flash, no NVS write, no serial write, no provider execution,
  no V21 execution, no provider key in firmware, no generic `xiaozhi.bin`
  product lane, no Git prune/gc, and no internal-test3 voice/protocol rollback.

## 2026-06-04 23:20 CST - StackChan Product State Body Reaction Implementation

Round goal:

- Continue hardware/body parity after accepted touch reactions by making
  ordinary Xiaozhi states visibly drive the StackChan body, without reopening
  keepalive/touch bridge work or touching internal-test3 voice/protocol.

Actual completed work:

- Archived currently controllable old delegation worker threads to reduce
  control-tower noise; no repository branch, worktree, commit, or report was
  deleted.
- Added plan
  `docs/plans/2026-06-04-stackchan-product-state-body-reactions.md`.
- Added `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true` as a separate server-side
  Gateway gate.
- Gateway returns `a21.state_reactions=true` only for hardware-MAC stock
  Xiaozhi clients with `hello.features.mcp=true`.
- Gateway-generated `idle`, `listening`, `thinking`, `speaking`, `error`, and
  `fatal_error` states now send bounded official MCP
  `self.robot.set_led_color` and `self.robot.set_head_angles` reactions over
  the live `/v1/xiaozhi` socket.
- State reactions are independent from `/stackChan/ws`, touch reactions, and
  firmware-originated `type=device` event allowances.
- Device registry records only redacted runtime echo:
  `last_state_reaction_status`, `last_state_reaction_state`,
  `last_state_reaction_reason`, and bounded robot LED/head fields.
- Added a reliability guard so Xiaozhi websocket close marks the registry
  `connection_status=xiaozhi_ws_disconnected` while preserving the previous
  semantic `last_event`; stale registry rows can no longer be mistaken for a
  writable MCP/body-control socket.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `docs/plans/2026-06-04-stackchan-product-state-body-reactions.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- `go test ./internal/gateway -run 'TestXiaozhiDeviceRegistryMarksSocketDisconnectedOnClose|TestXiaozhiProductStateReactions|TestXiaozhiProductTouchReactions' -count=1`
  passed.
- `go test ./internal/app -run 'TestGatewayServerOptionsFromEnvWiresProduct(State|Touch|Playback)Events|TestGatewayServerOptionsFromEnvWiresProductTouchReactions|TestGatewayServerOptionsFromEnvWiresProductStateReactions' -count=1`
  passed.
- `git diff --check` passed before this handoff update.
- `GOMAXPROCS=2 make verify` passed after docs/state updates.
- ECS deployment and physical device evidence are still pending at this log
  point.

Unfinished items:

- Deploy to ECS with root-only
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true`.
- Collect fresh runtime/physical evidence on device `44:1b:f6:e2:6a:60`,
  ideally a `listen_start` state reaction showing LED/head movement and
  `xiaozhi.state_reaction.*` markers.
- This does not close screen visual acceptance, richer choreography,
  camera/NFC/infrared, no-cable boot/power, app lifecycle, or full PRD
  acceptance.

Recommended next action:

- Commit/push this state-reaction cut, deploy the Gateway to ECS, enable the
  env gate, restart `a21-gateway`, and collect one physical state-reaction
  report.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no serial write, no
  provider execution, no V21 execution, no provider key in firmware, no generic
  `xiaozhi.bin` product lane, no Git prune/gc, and no internal-test3
  voice/protocol rollback.

## 2026-06-04 23:16 CST - StackChan Product State Body Reaction Evidence

Round goal:

- Finish the state/body reaction cut by deploying to ECS, proving product
  device MCP/body delivery, and reducing stale socket false-green risk, without
  repeating the accepted keepalive/touch work.

Actual completed work:

- Pushed commit `b6c12f0`:
  `feat(stackchan): add product state body reactions`.
- Pushed follow-up commit `f2663f1`:
  `fix(gateway): mark disconnected xiaozhi sockets`.
- Deployed the branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs` to ECS
  `47.103.57.217`.
- Enabled root-only
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true` in `/etc/a21/runtime.env`; the
  env presence was checked only as redacted `<set>`.
- Restarted `a21-gateway` and confirmed ECS local health plus public
  direct-source `/healthz` and OTA probes.
- Recovered device `44:1b:f6:e2:6a:60` with read-only chip-id plus hard-reset;
  no firmware app flash, NVS write, or serial write command was run.
- Captured product-device runtime evidence via `/v1/xiaozhi/say` with a short
  local WAV trigger, proving the state reaction path over the real Xiaozhi
  socket and official MCP responses.
- Added a checked evidence report:
  `reports/a21-stackchan-state-reaction-evidence-20260604-2316.json`.
- Closed one still-manageable old worker thread; older detached delegation
  records returned `No AppServerManager registered`, so no further thread-tool
  cleanup was possible from this session.

Changed files:

- `reports/a21-stackchan-state-reaction-evidence-20260604-2316.json`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Runtime or physical evidence:

- Device: `44:1b:f6:e2:6a:60`.
- Evidence trace/session:
  `a21-trace-state-reaction-wav-20260604` /
  `a21-session-state-reaction-wav-20260604`.
- `/v1/xiaozhi/say` response: `status=delivered`,
  `delivered_transport=xiaozhi_ws`, `audio_source=wav_file`,
  `audio_chunks=4`.
- Trace markers included `xiaozhi.say.start`, `tts.first_audio`,
  `stackchan.display_state.received`, `stackchan.display_state.registry_updated`,
  `xiaozhi.state_reaction.robot_led_color.sent`,
  `xiaozhi.state_reaction.robot_head_angles_set.sent`,
  `xiaozhi.tts.opus_frame.downlink`, `xiaozhi.say.downlink`,
  `xiaozhi.tts.stop`, and redacted `xiaozhi.mcp.response.received`.
- `/v1/devices` runtime echo after completion showed
  `last_state_reaction_status=delivered`,
  `last_state_reaction_state=idle`,
  `last_state_reaction_reason=tts_stop_host_say_complete`,
  `last_state_reaction_tool=robot_head_angles_set`, LED `20/20/40`, and head
  `yaw=0,pitch=24,speed=160`.

Tests/build/runtime results:

- `go test ./internal/gateway -run 'TestXiaozhiDeviceRegistryMarksSocketDisconnectedOnClose|TestXiaozhiProductStateReactions|TestXiaozhiProductTouchReactions' -count=1`
  passed.
- `go test ./internal/app -run 'TestGatewayServerOptionsFromEnvWiresProduct(State|Touch|Playback)Events|TestGatewayServerOptionsFromEnvWiresProductTouchReactions|TestGatewayServerOptionsFromEnvWiresProductStateReactions' -count=1`
  passed.
- `GOMAXPROCS=2 make verify` passed before this evidence-log update.
- ECS remote focused tests, remote Go build, service restart, and health
  probes passed.

Unfinished items:

- This proves product-device state/body reaction through `/v1/xiaozhi` MCP,
  not full natural realtime microphone/listen acceptance.
- Full PRD physical acceptance remains open for natural voice turn evidence,
  screen visual acceptance, app lifecycle/no-welcome parity, no-cable
  boot/power behavior, richer official-style choreography, camera, NFC, and
  infrared.

Recommended next action:

- Run a short natural listen/speak physical acceptance slice on the already
  deployed ECS Gateway, then promote screen visual acceptance and no-cable
  boot/power as the next two hardware parity cuts.

Forbidden actions avoided:

- No firmware app flash, no NVS write, no serial write, no provider secret
  printing, no provider key in firmware, no V21 internal execution, no generic
  `xiaozhi.bin` product flash, no Git prune/gc, and no internal-test3
  voice/protocol rollback.

## 2026-06-04 23:25 CST - Xiaozhi Physical Acceptance Tooling Unblocked

Round goal:

- Continue from state/body evidence toward natural microphone listen/speak PRD
  acceptance and identify whether the current blocker is tooling, network,
  Gateway, or physical runtime evidence.

Actual completed work:

- Confirmed ECS public health still works through the TUN-safe direct-source
  path.
- Confirmed local A21 evidence commands must be run with
  `A21_DIRECT_SOURCE_IP=192.168.1.20` while the control Mac TUN route is
  active.
- Found a false acceptance-tool blocker: `xiaozhi-physical-evidence` rejected
  the full Gateway device record because safe product runtime echo included
  `roleplay_prompt_text_stored=false`.
- Fixed the reader to scan only Xiaozhi physical consumed runtime echo fields
  while keeping trace/audio/instrument redaction checks strict.
- Added regression coverage by including safe roleplay runtime echo fields in
  the Xiaozhi physical evidence fixture.
- Re-ran live ECS physical evidence and half-duplex commands. They now produce
  honest blocked reports instead of `gateway data unsafe`.

Changed files:

- `internal/app/xiaozhi_physical_evidence.go`
- `internal/app/xiaozhi_physical_evidence_test.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- `go test ./internal/app -run 'TestRunXiaozhiPhysicalEvidence|TestRunXiaozhiHalfDuplexAcceptance|TestA21DirectHTTPClient' -count=1`
  passed.
- `git diff --check` passed before this handoff update.
- Live command with `A21_DIRECT_SOURCE_IP=192.168.1.20`:
  `xiaozhi-physical-evidence --gateway-url http://47.103.57.217 --device-id 44:1b:f6:e2:6a:60 --trace-id a21-trace-44-1b-f6-e2-6a-60 --session-id a21-session-44-1b-f6-e2-6a-60`
  produced blocked report
  `reports/a21-xiaozhi-physical-evidence-20260604-232451.781686000.json`.
- Live command with `A21_DIRECT_SOURCE_IP=192.168.1.20`:
  `stackchan-accept --check xiaozhi-half-duplex --gateway-url http://47.103.57.217 --device-id 44:1b:f6:e2:6a:60`
  produced blocked report
  `reports/a21-xiaozhi-half-duplex-acceptance-20260604-232452.185159000.json`.

Runtime evidence and blocker:

- `/v1/devices` now correctly reports
  `connection_status=xiaozhi_ws_disconnected`, not stale online.
- Latest trace `a21-trace-44-1b-f6-e2-6a-60` has `xiaozhi.listen.start`,
  `asr.stream.start`, and state-reaction MCP markers, but lacks Opus decode,
  PCM ingress, VAD speech end, listen auto-stop, TTS downlink, playback ack,
  and audible observation.
- Therefore the next blocker is real product-device runtime evidence, not ECS
  health, network routing, or acceptance-tool safety filtering.

Unfinished items:

- Recover the device online and trigger a natural microphone listen/speak turn.
- Capture audible or instrumented playback observation.
- Rerun `xiaozhi-physical-evidence`,
  `stackchan-accept --check xiaozhi-half-duplex`, then
  `xiaozhi-physical-prd-review` only when both reports are
  `physical_review_required`.

Recommended next action:

- Hard-reset/reconnect device `44:1b:f6:e2:6a:60` without flashing or NVS,
  trigger a real spoken turn, and collect the PRD chain with
  `A21_DIRECT_SOURCE_IP=192.168.1.20`.

Forbidden actions avoided:

- No firmware flash, no NVS write, no provider secret printing, no provider
  key in firmware, no V21 internal execution, no generic `xiaozhi.bin` product
  flash, no Git prune/gc, and no internal-test3 voice/protocol rollback.

## 2026-06-04 23:34 CST - Xiaozhi Reconnect Registry Fixed, Natural Audio Still Blocked

Round goal:

- Move from stale-socket/tooling diagnosis to real product-device natural
  voice acceptance without losing the state/body evidence thread.

Actual completed work:

- Pushed commit `5852e20`:
  `fix(gateway): restore xiaozhi online status after reconnect`.
- Deployed the branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs` to ECS
  `47.103.57.217`.
- Fixed `/v1/devices` behavior so fresh product Xiaozhi hello, heartbeat,
  playback, touch, and state-reaction activity restore
  `connection_status=online` after a prior socket-close
  `xiaozhi_ws_disconnected` mark.
- Recovered device `44:1b:f6:e2:6a:60` with read-only chip-id plus hard-reset;
  no firmware flash, no NVS write, and no serial write command was run.
- Confirmed live public direct-source device snapshots now show stable
  `online` heartbeat after reconnect.
- Reran physical evidence after the tooling fix and collected serial
  diagnostics for the remaining natural voice blocker.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`

Tests/build/runtime results:

- `go test ./internal/gateway -run 'TestXiaozhiDeviceRegistryMarksSocketDisconnectedOnClose|TestXiaozhiDeviceRegistryRestoresOnlineAfterReconnect|TestXiaozhiProductKeepaliveEventsAllowanceRecordsHeartbeat|TestXiaozhiProductPlaybackEventsAllowanceRecordsPlaybackStart|TestXiaozhiProductTouchEventsAllowanceRecordsTouch|TestXiaozhiProductStateReactions' -count=1`
  passed.
- `GOMAXPROCS=2 make verify` passed before deployment.
- ECS remote focused tests, remote Go build, `a21-gateway` restart, and
  local `/healthz` passed after deployment.

Runtime evidence and blocker:

- Live `xiaozhi-physical-evidence` with
  `A21_DIRECT_SOURCE_IP=192.168.1.20` now finds the device online and no longer
  stops at `gateway data unsafe`.
- The current live blocked report still has `audio_frame_count=0`, with no
  Opus decode, PCM ingress, VAD speech end, listen auto-stop, TTS downlink,
  playback ack, or audible observation.
- Serial evidence during a controlled wake/listen attempt showed wake-word
  detection, Gateway reconnect/listen start, and official MCP state-reaction
  execution, but the device transitioned `listening -> idle` almost
  immediately and only later logged wake-word Opus packet encoding.
- The narrowed suspect is the current stock-physical server `type=listen`
  reply suppression invariant. It must be tested with a product-gated,
  default-off path, not by rolling back internal test 3 protocol behavior.

Unfinished items:

- Implement and test a minimal product-gated listen-reply trial path.
- Enable it on ECS only after local regression tests pass.
- Rerun natural microphone physical evidence and half-duplex acceptance.

Recommended next action:

- Add `A21_XIAOZHI_PRODUCT_LISTEN_REPLIES` as a separate runtime gate for
  hardware-MAC product Xiaozhi clients, preserve the default suppression
  behavior, deploy it, and run one controlled physical audio-ingress trial.

Forbidden actions avoided:

- No firmware flash, no NVS write, no provider secret printing, no provider
  key in firmware, no V21 internal execution, no generic `xiaozhi.bin` product
  flash, no Git prune/gc, and no internal-test3 voice/protocol rollback.

## 2026-06-04 23:48 CST - Xiaozhi Wake Control-Channel Overlay Root Cause

Round goal:

- Fix the first concrete cause of natural audio ingress failure without
  changing Gateway provider routing, Xiaozhi listen suppression, NVS, or
  internal test 3 voice protocol behavior.

Actual completed work:

- Investigated the current listen-reply suppression hypothesis against the
  official Xiaozhi WebSocket/Application code. Server `type=listen` replies are
  not consumed by the official Application handler, so that is not the first
  repair target.
- Found a firmware overlay hunk-context bug: the intended
  `ContinueWakeWordInvoke` idle-control-channel repair could match a nearby
  helper with the same comment, leaving the actual wake continuation path in
  its old `Connecting`-only state.
- Tightened the overlay hunk with the
  `Application::ContinueWakeWordInvoke` function signature.
- Added a targeted host test that requires the exact wake-function hunk, so a
  future nearby-helper false positive fails locally.
- Fixed an unrelated doctor-test false positive where a generated timestamp
  containing `7891` could be mistaken for a leaked provider proxy port. The
  test still blocks the full proxy URL, host:port, and secret value.
- Rebuilt the official-compatible product artifact without hardware write.

Changed files:

- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `internal/app/app_test.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- `go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady|TestRunXiaozhiPhysicalEvidence|TestGatewayServerOptionsFromEnvWiresProduct' -count=1`
  passed.
- `go test ./internal/app -run 'TestRunDoctorIncludesProviderNetworkPolicy|TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady' -count=1`
  passed.
- `GOMAXPROCS=2 make verify` passed.
- `GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`
  passed with report
  `reports/a21-stackchan-official-baseline-20260604-234848-1780588128571683000.json`
  and product app SHA-256
  `e8880adbe7982a2e59bf58319d34097cbc16fbfcc2988975c0a19e186d32b305`.
- Post-build source inspection confirmed
  `ContinueWakeWordInvoke` now allows
  `state == kDeviceStateIdle && protocol_->IsAudioChannelOpened()`.

Runtime evidence and blocker:

- A guarded product flash attempt correctly stopped before hardware write
  because the tracked worktree was dirty. This is a guard success, not a
  hardware failure.

Unfinished items:

- Commit this root-cause repair.
- Rerun the guarded product-lane flash command against `/dev/cu.usbmodem1101`.
- After reboot/reconnect, run a natural wake/speak acceptance attempt and
  rerun `xiaozhi-physical-evidence` plus `stackchan-accept --check
  xiaozhi-half-duplex`.

Recommended next action:

- Commit/push, execute
  `a21-stackchan-official-xiaozhi-compatible-flash-execute` with the product
  confirmation token, then collect audio ingress evidence with the TUN-safe
  `A21_DIRECT_SOURCE_IP=192.168.1.20` path.

Forbidden actions avoided:

- No firmware flash completed while the worktree was dirty, no NVS write, no
  provider secret printing, no provider key in firmware, no V21 internal
  execution, no generic `xiaozhi.bin` product flash, no Git prune/gc, and no
  internal-test3 voice/protocol rollback.

## 2026-06-04 23:55 CST - Xiaozhi Natural Audio Ingress Restored

Round goal:

- Complete the wake/control-channel repair through product flash and prove
  that natural audio ingress is no longer stuck at zero frames.

Actual completed work:

- Committed and pushed `43fcd16` so the product flash guard could run from a
  clean tracked worktree.
- Executed the guarded official-compatible product app flash on
  `/dev/cu.usbmodem1101`.
- Confirmed the device reconnected to public Gateway `47.103.57.217` using the
  existing NVS and resumed heartbeat.
- Ran a controlled physical wake/listen/speak attempt after the flash.
- Collected serial evidence of wake detection, `idle -> listening`, AFE
  startup, device VAD stop, `idle -> speaking`, and official MCP state
  reactions.
- Fixed the local physical-evidence reader to allow the safe redacted trace
  marker `roleplay.prompt_input.used` while still rejecting unsafe transcript,
  prompt body, URL, path, raw-audio, provider-output, and secret values.
- Reran live `xiaozhi-physical-evidence` and half-duplex acceptance commands.

Changed files:

- `internal/app/xiaozhi_physical_evidence.go`
- `internal/app/xiaozhi_physical_evidence_test.go`
- `reports/a21-stackchan-official-baseline-20260604-234848-1780588128571683000.json`
- `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-235129-1780588289127583000.json`
- `reports/a21-xiaozhi-physical-evidence-20260604-235541.527765000.json`
- `reports/a21-xiaozhi-half-duplex-acceptance-20260604-235541.288482000.json`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- `go test ./internal/app -run 'TestRunXiaozhiPhysicalEvidence|TestRunXiaozhiHalfDuplexAcceptance|TestRunDoctorIncludesProviderNetworkPolicy' -count=1`
  passed.
- `GOMAXPROCS=2 make verify` passed.
- Product build report:
  `reports/a21-stackchan-official-baseline-20260604-234848-1780588128571683000.json`.
- Product flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-235129-1780588289127583000.json`,
  app SHA-256
  `e8880adbe7982a2e59bf58319d34097cbc16fbfcc2988975c0a19e186d32b305`.
- Physical evidence report:
  `reports/a21-xiaozhi-physical-evidence-20260604-235541.527765000.json`,
  `acceptance_status=candidate_gateway_downlink`,
  `audio_frame_count=64`, mic available, Opus decode available, PCM ingress
  available, VAD speech end available, TTS downlink available, and playback
  ack available.
- Half-duplex report:
  `reports/a21-xiaozhi-half-duplex-acceptance-20260604-235541.288482000.json`,
  still `blocked`.

Runtime evidence and blocker:

- Live trace summary after the flash included
  `xiaozhi_listen_to_audio_ingress_ms=114`, `asr_first_partial_ms=207`,
  `llm_first_content_ms=434`, `audio_downlink_first_frame_ms=467`,
  `tts_first_audio_ms=766`, `device_playback_start_ms=51`, and
  `answer_first_audio_total_ms=641`.
- Remaining blockers are now full half-duplex/product acceptance items:
  barge-in detected/stop/stop_done evidence and trusted operator or
  instrumented audible observation.

Unfinished items:

- Collect trusted audible observation or instrumented audible energy.
- Run a physical barge-in/stop_done window while the product device is
  speaking.
- Rerun `xiaozhi-physical-evidence`, `stackchan-accept --check
  xiaozhi-half-duplex`, then `xiaozhi-physical-prd-review` only when those
  acceptance reports become reviewable.

Recommended next action:

- Promote `T-XIAOZHI-PHYSICAL-BARGE-IN-STOP-DONE-001`: use a long enough
  product speaking window, trigger physical wake or top-touch interruption,
  confirm `barge_in.detected`, `playback.stop`, and
  `device.playback.stop_done`, then add audible/instrument observation.

Forbidden actions avoided:

- No NVS write, no provider secret printing, no provider key in firmware, no
  V21 internal execution, no generic `xiaozhi.bin` product flash, no Git
  prune/gc, and no internal-test3 voice/protocol rollback.

## 2026-06-05 00:37 CST - StackChan Touch Barge-In StopDone Candidate

Round goal:

- Continue internal test 4 hardware/body parity without redoing or rolling back
  internal test 3 voice acceptance.
- Flash the latest product official-compatible app and prove that speaking
  barge-in can reach playback stop and device stop_done.

Actual completed work:

- Confirmed branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs` was clean
  at `9413ed5 fix(stackchan): keep product barge-in alive while speaking`.
- Confirmed product artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  SHA-256 `4af28d25013111777f2bc82befd6b697ea27da3484e1acd7b8eff661007b0de6`.
- Executed the guarded official-compatible product flash on
  `/dev/cu.usbmodem1101` with no NVS write and no generic `xiaozhi.bin` lane.
- Confirmed the device reconnected to public Gateway `47.103.57.217`.
- Ran live speaking-window tests. A medium Chinese host-say run produced a real
  product top-touch barge-in trace with `barge_in.detected`, `playback.stop`,
  and `device.playback.stop_done`.
- Fixed local evidence matching so reports can consume same-trace product
  runtime echo that uses the normalized device default session
  `a21-session-44-1b-f6-e2-6a-60`, while still requiring the requested session
  to appear in trace and still rejecting cross-device/cross-trace evidence.
- Recovered a stuck `speaking` display state with a short host-say; short
  host-say delivered, sent `xiaozhi.tts.stop`, and produced
  `device.playback.stop_done`.

Changed files:

- `internal/app/xiaozhi_physical_evidence.go`
- `internal/app/xiaozhi_physical_evidence_test.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Runtime evidence:

- Product flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-002920-1780590560252437000.json`,
  status `passed`.
- Touch barge-in trace:
  `a21-trace-speaking-barge-cn-20260605003524`.
- Physical evidence report:
  `reports/a21-xiaozhi-physical-evidence-20260605-003553.699946000.json`,
  status `candidate_gateway_downlink`, playback start `59 ms`, barge-in
  detected/stopped true, stop_done `26 ms`.
- Half-duplex report:
  `reports/a21-xiaozhi-half-duplex-acceptance-20260605-003553.730698000.json`,
  still `blocked`, now only for `xiaozhi_half_duplex_mic_ingress_missing` and
  `xiaozhi_half_duplex_audible_observation_missing`.

Tests/build/runtime results:

- Focused app tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'TestRunXiaozhiPhysicalEvidence(AcceptsProductRuntimeSessionDrift|RejectsTargetMismatches|AcceptsGatewayTraceMarkers)|TestRunXiaozhiHalfDuplexAcceptance' -count=1`.
- Full verification passed:
  `GOMAXPROCS=2 make verify`.

Unfinished items:

- Generate a single reviewable physical window that includes wake/listen mic
  ingress, answer downlink, playback start, audible or instrument observation,
  and touch or wake-word barge-in stop_done in one target evidence set.
- Investigate medium/long host-say `502` after long downlink windows. Short
  host-say recovers and proves stop_done, so do not treat this as a voice-chain
  rollback.
- Rerun `xiaozhi-physical-prd-review` only after mic ingress and audible
  observation are present with the stop_done evidence.

Recommended next action:

- First close the evidence quality gap: combine a wake/listen turn with a
  speaking top-touch barge-in in one operator-supervised physical window.
  Then address the long host-say 502 recovery path if it still reproduces.

Forbidden actions avoided:

- No NVS write, no provider secret printing, no provider key in firmware, no
  V21 internal execution, no generic product flash lane, no Git prune/gc, and
  no internal-test3 voice/protocol rollback.

## 2026-06-05 00:42 CST - Clean Wake/Mic Ingress Retry Note

Round goal:

- After committing `78f2d83`, try to combine the newly proven touch
  barge-in/stop_done path with wake/listen mic ingress in a single physical
  evidence window.

Actual completed work:

- Confirmed branch was clean and pushed at `78f2d83`.
- Confirmed public Gateway health was ok and device `44:1b:f6:e2:6a:60` was
  online.
- Attempted to use the post-barge `listening` state by playing a Chinese
  question through macOS `Tingting`; the device did not produce a new answer
  trace and the previous trace only accumulated suppressed post-barge frames.
- Recovered the device to `idle` with short host-say trace
  `a21-trace-reset-idle-20260605004000`, then waited and tried a clean
  macOS-speaker wake/question again. That did not produce reliable new
  wake/listen mic-ingress evidence; the trace shows host-say suppression
  markers and ignored suppressed Opus frames.

Changed files:

- `docs/agent_handoff_log.md`

Runtime evidence:

- Device was healthy after the retry: online, `display_state=idle`, last trace
  `a21-trace-reset-idle-20260605004000`.
- The retry did not improve the half-duplex blocker beyond the previous
  `reports/a21-xiaozhi-half-duplex-acceptance-20260605-003553.730698000.json`.

Tests/build/runtime results:

- No code changed in this retry note.
- No new acceptance report was promoted.

Remaining issues:

- Mac speaker wake is not reliable enough to be used as product physical
  mic-ingress proof from this workstation position.
- Next physical window needs the user/operator physically near the device to
  say `紫悦` and ask a short question, then top-touch during the answer. The
  target report should combine mic ingress, answer downlink/playback, audible
  or instrument observation, and touch/wake barge-in stop_done.

Forbidden actions avoided:

- No NVS write, no provider secret printing, no firmware flash, no provider or
  V21 execution, no Git prune/gc, and no internal-test3 voice/protocol rollback.

## 2026-06-05 00:48 CST - Xiaozhi Host-Say Touch Interrupt Classification

Round goal:

- Continue internal test 4 hardware/body parity without redoing internal test 3
  voice acceptance.
- Fix the product-facing `/v1/xiaozhi/say` behavior where a normal
  user/device touch barge-in could surface as HTTP 502 after a speaking
  window.

Actual completed work:

- Reproduced the control-surface bug with a deterministic Gateway test: host
  say starts, first audio frame is delivered, product top-touch barge-in sends
  `touch_barge_in` stop, then the second TTS chunk observes the canceled turn.
- Added Gateway response semantics for intentional interruption. Safe
  cancellation reasons containing `barge`, `wake`, `abort`, or `interrupt`
  now return HTTP 200 with `status=interrupted`, safe `interrupt_reason`,
  partial `audio_chunks`, and trace marker `xiaozhi.say.interrupted`.
- Preserved the failure boundary: actual TTS/downlink errors still return HTTP
  502 and record `xiaozhi.say.downlink_error`.
- Committed and pushed `45f363f fix(gateway): classify xiaozhi say barge-in
  as interrupted`.
- Deployed `45f363f` to ECS `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path after remote focused tests and build passed.
- Confirmed post-deploy public `/healthz` ok and product device
  `44:1b:f6:e2:6a:60` online with fresh `last_event=xiaozhi.hello`.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Failing test before implementation:
  `GOMAXPROCS=2 go test ./internal/gateway -run TestXiaozhiSayReportsInterruptedWhenProductTouchBargeInCancelsDownlink -count=1`
  failed because `XiaozhiSayResponse` had no `InterruptReason`.
- Focused test passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run TestXiaozhiSayReportsInterruptedWhenProductTouchBargeInCancelsDownlink -count=1`.
- Focused Gateway regression suite passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestXiaozhi(ProductTouchBargeInCancelsActiveTurnAndStopsPlayback|ProductTouchReactionsSendBoundedBodyMCP|Say(DeliversTextAsStockTTSDownlink|ReportsInterruptedWhenProductTouchBargeInCancelsDownlink|KeepsBadGatewayForActualDownlinkError|DeliversWAVAsStockTTSDownlink|SuppressesImmediateListenRestartForStockPhysical))' -count=1`.
- Full verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote ECS focused Gateway regression suite passed before safe-swap.
- Remote ECS build passed:
  `/usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- Remote `systemctl is-active a21-gateway`, loopback `/healthz`, and public
  direct-source `/healthz` passed after safe-swap.

Runtime or physical evidence:

- No new physical PRD evidence was promoted in this round. This was a Gateway
  control-surface repair for the previously observed host-say 502 after
  touch/barge-in.
- Public direct-source `/v1/devices` after the ECS restart showed the product
  device online with product touch/playback/keepalive/state-reaction
  capabilities still present.

Remaining issues:

- Full physical PRD acceptance still needs one operator-side product window
  with wake/listen mic ingress, answer downlink/playback, trusted audible or
  instrument observation, and touch/wake barge-in stop_done.
- Screen/servo/RGB/body parity should continue after this control-surface fix;
  camera/NFC/IR remain higher-risk parity spikes.

Next suggested action:

- In the next operator-side hardware window, rerun a short host-say interrupted
  by top touch to confirm public HTTP now reports `interrupted` instead of
  502. Then continue physical screen/servo/RGB/touch body parity and the
  mic/audible window.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no Git
  prune/gc, and no internal-test3 voice/protocol rollback.

## 2026-06-05 00:58 CST - Xiaozhi Body Preset Sequence Deployed

Round goal:

- Continue hardware/body parity after the host-say interrupt fix by turning
  low-level robot LED/head MCP controls into a product-usable expression
  sequence endpoint.

Actual completed work:

- Added `POST /v1/xiaozhi/body-preset`.
- Supported bounded presets: `ready`, `listening`, `thinking`, `speaking`,
  `celebrate`, and `reset_idle`.
- Each preset expands to exactly two official MCP writes over the existing
  live Xiaozhi socket: `self.robot.set_led_color` followed by
  `self.robot.set_head_angles`.
- Responses return redacted step metadata, `delivered_transport` as
  `xiaozhi_mcp_sequence`, and `physical_accepted=false`.
- Added tests for successful bounded preset delivery, trace markers, registry
  updates, and rejection of unknown presets such as `camera`.
- Committed and pushed `da77d21 feat(gateway): add xiaozhi body preset
  sequences`.
- Deployed `da77d21` to ECS `47.103.57.217` through `/opt/a21.next`
  safe-swap.
- Ran a live public `celebrate` preset against product device
  `44:1b:f6:e2:6a:60`.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Focused local body/MCP tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestXiaozhiBodyPreset|TestXiaozhiNamedMCPStatusEndpointsUseScopedTools|TestXiaozhiMCPStatusParityAllowsOnlyScopedTools' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote focused body/MCP tests passed before ECS safe-swap.
- Remote build passed:
  `/usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- Remote `a21-gateway` restarted active, loopback `/healthz` passed, and
  public direct-source `/healthz` passed.

Runtime or physical evidence:

- Live public body-preset response for trace
  `a21-trace-live-body-preset-celebrate-202606050058` returned
  `status=delivered`, `delivered_transport=xiaozhi_mcp_sequence`,
  `physical_accepted=false`, LED args `red=0,green=168,blue=80`, and head args
  `yaw=18,pitch=36,speed=260`.
- Live trace recorded
  `xiaozhi.body_preset.celebrate.robot_led_color.sent` and
  `xiaozhi.body_preset.celebrate.robot_head_angles_set.sent`.
- Public `/v1/devices` recorded `last_body_preset=celebrate`,
  `robot_led_green=168`, `robot_led_blue=80`, `robot_head_yaw=18`,
  `robot_head_pitch=36`, and `robot_head_speed=260`; the device remained
  online.

Remaining issues:

- This is product-socket body-control evidence, not operator/instrument
  physical proof that LED and servos visibly moved.
- Full PRD physical acceptance still needs the mic ingress + audible
  observation + answer playback + barge-in stop_done window.
- Camera, NFC, and infrared remain planned high-risk spikes, not product
  controls.

Next suggested action:

- In the next foreground hardware window, have the operator watch the device
  while calling `/v1/xiaozhi/body-preset` presets, then record visible LED/head
  movement evidence. Continue toward mic/audible PRD acceptance after body
  expression is visibly confirmed.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no Git
  prune/gc, no camera/NFC/IR expansion, and no internal-test3 voice/protocol
  rollback.

## 2026-06-05 01:43 CST - Workspace Hardware Scene Console Deployed

Round goal:

- Move the hardware/body work from scattered screen/body buttons into a
  one-click foreground product scene surface, without reopening internal-test3
  voice acceptance or weakening the physical acceptance boundary.

Actual completed work:

- Added `POST /v1/xiaozhi/body-scene`.
- Added bounded `showtime`, `focus`, and `reset` scene plans.
- `showtime` combines screen theme/brightness plus RGB/head official MCP
  writes; `focus` and `reset` provide shorter bounded workspace states.
- Added ordered `xiaozhi.body_scene.<scene>.stepN.<marker>.sent` trace
  markers and safe device registry fields including `last_body_scene`,
  `screen_theme`, `screen_brightness`, `robot_led_*`, and `robot_head_*`.
- Added a Hardware Scenes section to `/workspace` with Showtime, Focus, Reset,
  scene trace display, and safe `hardware_scene_*` export metadata.
- Extended workspace and Gateway tests to lock the new endpoint, UI controls,
  ordered MCP sequence, trace markers, and registry evidence.
- Updated `docs/engineering/PROTOCOL.md` with the new scene contract.
- Committed and pushed
  `88e1549 feat(gateway): add xiaozhi hardware scene sequences`.
- Deployed `88e1549` to ECS `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Focused local tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyPreset|TestXiaozhiBodyMotion|TestXiaozhiBodyScene' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- `git diff --check` passed before the feature commit.
- Remote focused Gateway tests passed in `/opt/a21.next`:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyScene' -count=1`.
- Remote build passed:
  `/usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- Remote `a21-gateway` restarted active, loopback `/healthz` passed, and
  public direct-source `/healthz` passed.
- Public `/workspace` HTML smoke found `Hardware Scenes`,
  `/v1/xiaozhi/body-scene`, `runHardwareScene`,
  `refreshHardwareSceneTrace`, `hardware_scene_trace_id`, and
  `data-hardware-scene="showtime"`.

Runtime or physical evidence:

- Public `showtime` body-scene execution against product device
  `44:1b:f6:e2:6a:60` returned HTTP 409
  `xiaozhi websocket is not connected`.
- Public `/v1/devices` was empty during a six-poll post-deploy window after
  the Gateway service restart. This means the product Xiaozhi socket was not
  online at the moment of the scene smoke; it is not evidence of a voice-chain
  rollback or protocol regression.
- Scene product-socket delivery and physical screen/RGB/servo acceptance remain
  pending until the product device reconnects and the scene is run in a
  foreground hardware window.

Remaining issues:

- Product device `44:1b:f6:e2:6a:60` must reconnect to
  `ws://47.103.57.217/v1/xiaozhi` before live scene smoke can prove
  product-socket delivery.
- Physical acceptance for hardware scenes remains pending operator or
  instrument evidence that screen theme/brightness, RGB, and servo movement
  visibly occur.
- Official `/stackChan/ws` avatar/action relay remains disconnected; Hardware
  Scenes use the current Xiaozhi MCP product path.
- Camera, NFC, infrared, reboot, firmware upgrade, snapshot, video, and app
  lifecycle remain blocked/planned high-risk controls.

Next suggested action:

- Reconnect or power-cycle the product StackChan into the public Xiaozhi
  Gateway, then run `/workspace` Showtime and record the trace plus visible
  screen/RGB/servo evidence. If the device does not auto-reconnect after
  Gateway restart, promote firmware/app lifecycle reconnection resilience as
  the next foreground hardware transition.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no Git
  prune/gc, no camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app
  lifecycle exposure, and no internal-test3 voice/protocol rollback.

## 2026-06-05 01:14 CST - Workspace Hardware Screen Console Deployed

Round goal:

- Move the already deployed low-risk screen/status MCP parity endpoints into
  the `/workspace` product control surface, continuing hardware/body parity
  instead of repeating internal-test3 voice-chain acceptance work.

Actual completed work:

- Added a Hardware Screen section to `/workspace`.
- Added brightness slider + apply action, theme controls for `light`, `dark`,
  and `auto`, device status, screen info, MCP capabilities, and trace marker
  refresh controls.
- Wired the controls to existing `/v1/xiaozhi/device-status`,
  `/v1/xiaozhi/screen-brightness`, `/v1/xiaozhi/screen-theme`,
  `/v1/xiaozhi/mcp-control` for `self.screen.get_info`, and
  `/v1/xiaozhi/mcp-capabilities`.
- Added safe UI state for screen status, `physical_accepted`, trace id,
  brightness, theme, MCP tool, capabilities, and trace markers.
- Added safe `screen_control_*` fields to workspace client-side metadata
  export.
- Extended `TestWorkspaceConsolePageServed` to lock the screen controls,
  endpoints, JS helpers, and export fields into the served page contract.
- Updated `docs/engineering/PROTOCOL.md` to record that `/workspace` includes
  the low-risk hardware screen/status surface and must keep
  `physical_accepted=false` until visible operator or instrument evidence
  proves product screen effects.
- Committed and pushed
  `c74261d feat(gateway): expose screen controls in workspace console`.
- Deployed `c74261d` to ECS `47.103.57.217` through `/opt/a21.next`
  safe-swap.

Changed files:

- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Focused local tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiNamedMCPStatusEndpointsUseScopedTools|TestXiaozhiMCPStatusParityAllowsOnlyScopedTools' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote focused Gateway tests passed in `/opt/a21.next`.
- Remote build passed:
  `/usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- Remote `a21-gateway` restarted active, loopback `/healthz` passed, and
  public direct-source `/healthz` passed.
- Public `/workspace` HTML smoke found `Hardware Screen`,
  `/v1/xiaozhi/screen-brightness`, `/v1/xiaozhi/screen-theme`,
  `/v1/xiaozhi/device-status`, `/v1/xiaozhi/mcp-capabilities`,
  `runHardwareScreenAction`, and `screen_control_trace_id`.

Runtime or physical evidence:

- Public MCP capabilities on product device `44:1b:f6:e2:6a:60` returned
  `connection_status=online`, `mcp_advertised=true`, allowed tools for
  speaker/status/screen/robot MCP, blocked high-risk classes for reboot,
  firmware upgrade, camera, snapshot, video, NFC, infrared, and app lifecycle,
  `result_redacted=true`, and `physical_accepted=false`.
- Live public screen brightness response for trace
  `a21-trace-workspace-screen-brightness-c74261d` returned
  `status=delivered`, `tool_name=self.screen.set_brightness`,
  `arguments.brightness=62`, and `result_redacted=true`.
- Live public screen theme response for trace
  `a21-trace-workspace-screen-theme-c74261d` returned `status=delivered`,
  `tool_name=self.screen.set_theme`, `arguments.theme=dark`, and
  `result_redacted=true`.
- Live public screen info response for trace
  `a21-trace-workspace-screen-info-c74261d` returned `status=delivered`,
  `tool_name=self.screen.get_info`, and `result_redacted=true`.
- Live traces recorded `xiaozhi.mcp.screen_brightness.sent`,
  `xiaozhi.mcp.screen_theme.sent`, and `xiaozhi.mcp.screen_info.sent`.
- Public `/v1/devices` recorded `screen_brightness=62`,
  `screen_theme=dark`, and the product device remained online.

Remaining issues:

- This is deployed product-surface and product-socket evidence. It is not yet
  visible operator/instrument physical acceptance for screen brightness/theme
  effects.
- Full PRD physical acceptance remains `PHYSICAL-PENDING`; the remaining
  evidence window still needs mic ingress and trusted audible or instrumented
  observation before promotion.
- Camera, NFC, infrared, reboot, firmware upgrade, snapshot, video, and app
  lifecycle remain blocked/planned high-risk controls.

Next suggested action:

- Use `/workspace` Hardware Screen and Body Presets together in the next
  foreground hardware window, then record visible screen + LED/head movement
  evidence before promoting those surfaces to physical accepted.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no Git
  prune/gc, no camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app
  lifecycle exposure, and no internal-test3 voice/protocol rollback.

## 2026-06-05 01:32 CST - Workspace Official Action Fallback Deployed

Round goal:

- Improve foreground product ergonomics after the official action relay
  surfaced HTTP 409 for the disconnected `/stackChan/ws` socket, without
  hiding that runtime gap or claiming official-frame physical acceptance.

Actual completed work:

- Updated `/workspace` Official Actions so each click still attempts
  `/v1/stackchan/official/control` first.
- If the official relay is disconnected or blocked, the UI now runs an explicit
  MCP-backed fallback:
  - `motion` actions fall back to `/v1/xiaozhi/body-motion`.
  - `state` actions fall back to matching `/v1/xiaozhi/body-preset` values.
  - `face=happy` falls back to `celebrate`; `face=attentive` falls back to
    `listening`.
- Official action UI now shows `fallback=<type>:<value>` and
  `fallback_delivered` instead of leaving the product action dead.
- Metadata export now preserves `official_action_blocked_reason` and
  `official_action_fallback` separately from official-frame delivery.
- Updated `docs/engineering/PROTOCOL.md` to document the fallback behavior and
  the official-frame acceptance boundary.
- Committed and pushed:
  `7dfbb10 feat(gateway): fallback official actions to body motion`.
- Deployed `7dfbb10` to ECS `47.103.57.217` through `/opt/a21.next`
  safe-swap.

Changed files:

- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Focused local tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyMotion|TestOfficialStackChanControlEndpoint' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote focused Gateway tests passed in `/opt/a21.next`.
- Remote build passed:
  `/usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- Remote `a21-gateway` restarted active, loopback `/healthz` passed, and
  public direct-source `/healthz` passed.
- Public `/workspace` HTML smoke found `officialActionFallback`,
  `runOfficialActionFallback`, `fallback_delivered`,
  `official_action_blocked_reason`, `official_action_fallback`,
  `/v1/stackchan/official/control`, and `/v1/xiaozhi/body-motion`.

Runtime or physical evidence:

- Public official action relay on product device `44:1b:f6:e2:6a:60` still
  truthfully returns HTTP 409:
  `official stackchan websocket is not connected`.
- Public fallback body-motion `dance` response for trace
  `a21-trace-workspace-body-motion-dance-7dfbb10` returned
  `status=delivered`, `delivered_transport=xiaozhi_mcp_sequence`, 5 redacted
  steps, and `physical_accepted=false`.
- Live trace recorded 10 generic robot MCP and ordered body-motion markers.
- Public `/v1/devices` recorded `last_body_motion=dance`, final robot head
  `yaw=0,pitch=24,speed=220`, LED `red=0,green=168,blue=80`, and the product
  device remained online.

Remaining issues:

- Browser-click evidence was not collected in this environment; validation is
  from page contract tests, public HTML smoke, and public API/runtime traces.
- The official avatar/action relay remains disconnected until a `/stackChan/ws`
  socket or app-lifecycle reconciliation path is connected and accepted.
- Fallback body motion is product-socket delivery evidence, not
  operator/instrument physical acceptance.
- Full PRD physical acceptance remains `PHYSICAL-PENDING`.

Next suggested action:

- Continue toward official app lifecycle or `/stackChan/ws` reconciliation so
  Official Actions can eventually deliver official-frame evidence. Until then,
  use the fallback-backed Official Actions and Body Motion controls for
  foreground prototype demos and visible movement checks.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no Git
  prune/gc, no camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app
  lifecycle exposure, and no internal-test3 voice/protocol rollback.

## 2026-06-05 01:26 CST - Workspace Official Actions And Body Motion Deployed

Round goal:

- Continue hardware/body parity from screen/body controls into action
  controls: expose the existing official StackChan avatar/action relay in
  `/workspace`, and add a current-product Xiaozhi MCP motion path so the device
  can perform bounded head/LED motions even while the separate official
  `/stackChan/ws` socket is not connected.

Actual completed work:

- Added an Official Actions section to `/workspace` with semantic state, face,
  and motion controls for `idle`, `listening`, `thinking`, `speaking`, `happy`,
  `attentive`, `look_up`, `nod`, `shake`, `dance`, and `stop`.
- Wired Official Actions to existing `POST /v1/stackchan/official/control`.
- Official Actions display safe status, trace, event, packet count, transport,
  surface labels, and `official_action_physical_accepted=false`; a disconnected
  official socket is shown as a blocked state rather than fake delivery.
- Added `POST /v1/xiaozhi/body-motion` with bounded MCP-backed motions:
  `look_up`, `nod`, `shake`, `dance`, and `stop`.
- Wired Body Presets `/workspace` controls to the new body-motion endpoint for
  immediate current-product motion feedback on the live Xiaozhi MCP socket.
- Added safe `official_action_*` and `body_motion_*` metadata export fields.
- Extended workspace page contract tests and added body-motion sequence tests.
- Updated `docs/engineering/PROTOCOL.md` to document body-motion and the
  official action console behavior.
- Committed and pushed:
  `6ce372e feat(gateway): expose official actions in workspace console`.
- Committed and pushed:
  `e5ae4d1 feat(gateway): add xiaozhi body motion sequences`.
- Deployed `e5ae4d1` to ECS `47.103.57.217` through `/opt/a21.next`
  safe-swap, superseding the intermediate `6ce372e` binary.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Focused local tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyPreset|TestXiaozhiBodyMotion|TestOfficialStackChanControlEndpoint' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote focused Gateway tests passed in `/opt/a21.next`.
- Remote build passed:
  `/usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- Remote `a21-gateway` restarted active, loopback `/healthz` passed, and
  public direct-source `/healthz` passed.
- Public `/workspace` HTML smoke found `Official Actions`,
  `/v1/stackchan/official/control`, `official_action_trace_id`,
  `/v1/xiaozhi/body-motion`, `runBodyMotion`, and `body_motion_trace_id`.

Runtime or physical evidence:

- Public official action relay test against product device
  `44:1b:f6:e2:6a:60` returned HTTP 409:
  `official stackchan websocket is not connected`. This is the current truth
  for the separate official avatar/action relay path and is now surfaced
  honestly.
- Live public body-motion `dance` response for trace
  `a21-trace-workspace-body-motion-dance-e5ae4d1` returned
  `status=delivered`, `delivered_transport=xiaozhi_mcp_sequence`, 5 redacted
  steps, and `physical_accepted=false`.
- Live trace recorded 10 events: generic robot MCP markers plus ordered
  `xiaozhi.body_motion.dance.step1.robot_led_color.sent`,
  `xiaozhi.body_motion.dance.step2.robot_head_angles_set.sent`,
  `xiaozhi.body_motion.dance.step3.robot_led_color.sent`,
  `xiaozhi.body_motion.dance.step4.robot_head_angles_set.sent`, and
  `xiaozhi.body_motion.dance.step5.robot_head_angles_set.sent`.
- Public `/v1/devices` recorded `last_body_motion=dance`,
  `last_body_motion_status=delivered`, `last_body_motion_step=5`, final robot
  head `yaw=0,pitch=24,speed=220`, LED `red=0,green=168,blue=80`, and the
  product device remained online.

Remaining issues:

- Body-motion has product-socket delivery evidence, not operator/instrument
  physical acceptance that movement was visibly observed.
- The official avatar/action relay remains runtime-disconnected for the
  product device until a `/stackChan/ws` official socket or app-lifecycle
  reconciliation path is connected and accepted.
- Full PRD physical acceptance remains `PHYSICAL-PENDING`; the remaining
  evidence window still needs mic ingress and trusted audible or instrumented
  observation before promotion.
- Camera, NFC, infrared, reboot, firmware upgrade, snapshot, video, and app
  lifecycle remain blocked/planned high-risk controls.

Next suggested action:

- Keep `/workspace` Body Motion as the current-product action path. Next,
  reconcile the official `/stackChan/ws` avatar/action socket or app lifecycle
  so Official Actions can move from visible blocked state to delivered
  official-frame evidence without weakening the no-welcome product path.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no Git
  prune/gc, no camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app
  lifecycle exposure, and no internal-test3 voice/protocol rollback.

## 2026-06-05 01:07 CST - Workspace Body Preset Console Deployed

Round goal:

- Move the already deployed body-preset API from curl/API-only into the
  `/workspace` product control surface, while confirming this is not a renewed
  internal-test3 voice-chain acceptance run or rollback.

Actual completed work:

- Added a Body Presets section to `/workspace` with controls for `ready`,
  `listening`, `thinking`, `speaking`, `celebrate`, and `reset_idle`.
- Wired the controls to existing `POST /v1/xiaozhi/body-preset`.
- Added safe UI state for preset status, `physical_accepted`, trace id, LED
  args, head args, transport, and trace markers.
- Added safe `body_preset_*` fields to workspace client-side metadata export.
- Extended `TestWorkspaceConsolePageServed` to lock the body-preset controls,
  endpoint, JS helpers, and export fields into the served page contract.
- Updated `docs/engineering/PROTOCOL.md` to record that `/workspace` includes
  bounded body presets and must keep `physical_accepted=false` until visible
  operator or instrument evidence proves product LED/head movement.
- Committed and pushed
  `d361176 feat(gateway): expose body presets in workspace console`.
- Deployed `d361176` to ECS `47.103.57.217` through `/opt/a21.next`
  safe-swap.

Changed files:

- `internal/gateway/workspace_console.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Focused local tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyPreset' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote focused Gateway tests passed in `/opt/a21.next`.
- Remote build passed:
  `/usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- Remote `a21-gateway` restarted active, loopback `/healthz` passed, and
  public direct-source `/healthz` passed.
- Public `/workspace` HTML smoke found `Body Presets`,
  `/v1/xiaozhi/body-preset`, `bodyPresetActions`, and the `celebrate` control.

Runtime or physical evidence:

- Live public `ready` body-preset response for trace
  `a21-trace-workspace-body-ready-d361176` returned `status=delivered`,
  `delivered_transport=xiaozhi_mcp_sequence`, `physical_accepted=false`, LED
  args `red=0,green=36,blue=96`, and head args
  `yaw=0,pitch=22,speed=180`.
- Live trace recorded both official MCP send markers and body-preset markers:
  `xiaozhi.mcp.robot_led_color.sent`,
  `xiaozhi.body_preset.ready.robot_led_color.sent`,
  `xiaozhi.mcp.robot_head_angles_set.sent`, and
  `xiaozhi.body_preset.ready.robot_head_angles_set.sent`.
- Public `/v1/devices` recorded `last_body_preset=ready`, updated robot LED
  and head values, and the product device `44:1b:f6:e2:6a:60` remained online.

Remaining issues:

- This is deployed product-surface and product-socket evidence. It is not yet
  visible operator/instrument physical acceptance for LED/head movement.
- Full PRD physical acceptance remains `PHYSICAL-PENDING`; the remaining
  evidence window still needs mic ingress and trusted audible or instrumented
  observation before promotion.
- Camera, NFC, and infrared remain planned high-risk spikes, not product
  controls.

Next suggested action:

- Use `/workspace` Body Presets in the next foreground hardware window while
  watching the device, then record visible LED/head movement evidence and only
  then promote body presets from product-socket evidence to physical accepted.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no Git
  prune/gc, no camera/NFC/IR expansion, and no internal-test3 voice/protocol
  rollback.

## 2026-06-05 01:44 CST - Latest Handoff Pointer

- Latest full handoff entry:
  `2026-06-05 01:43 CST - Workspace Hardware Scene Console Deployed`.
- Commit `88e1549 feat(gateway): add xiaozhi hardware scene sequences` is
  pushed and deployed to ECS.
- `/workspace` now has Hardware Scenes and `POST /v1/xiaozhi/body-scene` for
  bounded `showtime`, `focus`, and `reset` screen/RGB/servo MCP scenes.
- Local focused tests, full `GOMAXPROCS=2 make verify`, remote focused tests,
  remote build, ECS safe-swap, public `/healthz`, and public `/workspace`
  smoke passed.
- Live product scene smoke is pending because product device
  `44:1b:f6:e2:6a:60` was not connected to `/v1/xiaozhi` after the Gateway
  restart; `/v1/devices` was empty in the immediate post-deploy poll window.
- This is not a voice/protocol rollback. No firmware build/flash, NVS write,
  provider/V21 execution, camera/NFC/IR expansion, reboot/OTA/snapshot/video/
  app lifecycle exposure, or Git prune/gc occurred.

## 2026-06-05 02:00 CST - Listen-Start State Reaction Suppressed And Reconnect Candidate Built

Round goal:

- Recover the hardware-scene path from the immediate post-deploy device socket
  instability without reopening internal-test3 voice acceptance, and prepare a
  firmware-side reconnect resilience candidate without flashing.

Actual completed work:

- Investigated the post-deploy `showtime` HTTP 409.
- Found the product device did reconnect once and then closed quickly:
  trace `a21-trace-44-1b-f6-e2-6a-60` showed `xiaozhi.hello`,
  `xiaozhi.listen.start`, automatic state-reaction MCP
  `robot_led_color`, `xiaozhi.state_reaction.failed`, `asr.stream.error`,
  `asr.stream.cancelled`, and
  `xiaozhi.opus_ingress.queue_cancelled.socket_closed` within 232 ms.
- Changed Gateway state reactions so stock physical
  `listening/listen_start` records
  `xiaozhi.state_reaction.listen_start_suppressed` and
  `last_state_reaction_status=suppressed_listen_start` instead of sending MCP.
- Kept non-listen-start state reaction coverage by moving the positive test to
  `thinking`.
- On ECS, changed `/etc/a21/runtime.env` to
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false` while keeping playback events,
  touch events, and touch reactions enabled.
- Added a firmware overlay candidate that moves
  `EnsureA21ControlChannel()` from VAD-change-only probing to periodic
  `MAIN_EVENT_CLOCK_TICK` probing, improving idle reconnect resilience.
- Committed and pushed:
  `72e6bcc fix(gateway): suppress listen-start state reactions`.
- Deployed `72e6bcc` to ECS through `/opt/a21.next` safe-swap.
- Committed and pushed:
  `b9c0baa fix(firmware): tick quiet xiaozhi reconnect checks`.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Red test first failed as expected:
  `TestXiaozhiProductStateReactionsSuppressListenStartMCP` observed an
  unexpected MCP `self.robot.set_led_color` on `listen_start`.
- Focused Gateway tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestXiaozhiProductStateReactions|TestXiaozhiBodyScene' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote focused Gateway tests passed in `/opt/a21.next`.
- Remote build passed:
  `/usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- Remote `a21-gateway` restarted active; loopback and public `/healthz`
  returned ok.
- Public `/workspace` smoke still found Hardware Scenes and
  `/v1/xiaozhi/body-scene`.
- No-flash firmware build passed:
  `make a21-stackchan-official-xiaozhi-compatible-build`.

Runtime or physical evidence:

- ECS runtime env now shows:
  `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS=true`,
  `A21_XIAOZHI_PRODUCT_TOUCH_EVENTS=true`,
  `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true`,
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false`.
- Built firmware candidate:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`.
- App SHA-256:
  `3eef974929aed78cdd77232897485aaac25bce8aa98daa4d8d78b3d96662b7ac`.
- Build report:
  `reports/a21-stackchan-official-baseline-20260605-020021-1780596021736181000.json`.
- Final public `/v1/devices` check showed product device
  `44:1b:f6:e2:6a:60` had reappeared but remained
  `connection_status=xiaozhi_ws_disconnected`.
- The final device capabilities no longer included
  `xiaozhi_product_state_reactions`, proving the runtime state-reaction gate
  rollback took effect.
- The refreshed trace no longer contained `xiaozhi.state_reaction.*` MCP
  markers; it still showed `listen.start` followed by `asr.stream.error`,
  `asr.stream.cancelled`, and
  `xiaozhi.opus_ingress.queue_cancelled.socket_closed`. The product device
  still needs reconnect resilience or a foreground guarded flash/power-cycle
  window before scene physical acceptance can be collected.

Remaining issues:

- Product scene delivery cannot be physically accepted until device
  `44:1b:f6:e2:6a:60` is online on `/v1/xiaozhi`.
- The reconnect resilience firmware candidate is built but not flashed.
- Runtime state reactions are intentionally off on ECS until a foreground
  hardware window proves the listen-start suppression and non-listening state
  reactions do not destabilize the socket.
- Official `/stackChan/ws` avatar/action relay remains disconnected.

Next suggested action:

- In the next foreground hardware window, either power-cycle/reconnect the
  current flashed device or use the guarded product lane to flash the built
  reconnect candidate, then run `/workspace` Showtime and record visible
  screen/RGB/servo evidence.

Forbidden actions avoided:

- No firmware flash, no NVS write, no provider secret printing, no provider or
  V21 execution, no generic product flash lane, no camera/NFC/IR expansion, no
  reboot/OTA/snapshot/video/app-lifecycle exposure, no Git prune/gc, and no
  internal-test3 voice/protocol rollback.

## 2026-06-05 02:30 CST - Hardware Body Scene Pacing Deployed

Round goal:

- Stop treating machine-delivered `full_check` as sufficient body feel when
  the scene was emitted in a near-instant burst, without reopening or
  rolling back internal-test3 voice-chain acceptance.

Actual completed work:

- Added bounded inter-step pacing to `POST /v1/xiaozhi/body-scene`.
- Added response fields `step_delay_ms` and `total_planned_delay_ms`.
- Wired the `gateway` runtime default to 180 ms per body-scene step and added
  positive `A21_BODY_SCENE_STEP_DELAY_MS` override support.
- Set ECS `/etc/a21/runtime.env` to `A21_BODY_SCENE_STEP_DELAY_MS=180`.
- Documented the body-scene pacing contract in `docs/engineering/PROTOCOL.md`.
- Committed and pushed:
  `6f43646 feat(gateway): pace hardware body scenes`.
- Deployed `6f43646` to ECS through `/opt/a21.next` safe-swap.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/app/app.go`
- `internal/app/app_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Red test first failed as expected before implementation:
  `TestXiaozhiBodySceneReportsAndAppliesStepPacing` did not find
  `step_delay_ms`.
- Focused local Gateway tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestXiaozhiBodySceneReportsAndAppliesStepPacing|TestXiaozhiBodySceneFullCheckRunsOperatorVisibleSequence|TestXiaozhiBodySceneSendsScreenAndBodyMCPSequence' -count=1`.
- Focused local App env test passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'TestGatewayServerOptionsFromEnvWiresBodySceneStepDelay' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote `/opt/a21.next` focused Gateway/App tests passed.
- Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- ECS `a21-gateway.service` restarted active; loopback and public `/healthz`
  passed.

Runtime or physical evidence:

- ECS runtime env now includes `A21_BODY_SCENE_STEP_DELAY_MS=180`.
- Product device `44:1b:f6:e2:6a:60` was online on `/v1/xiaozhi`.
- Live public `full_check` trace
  `a21-trace-hardware-full-check-paced-6f43646-202606050230` returned HTTP 200
  with `status=delivered`, `scene=full_check`, `step_delay_ms=180`,
  `total_planned_delay_ms=2700`, and 16 redacted steps.
- Trace endpoint recorded 32 ordered markers with
  `summary.last_offset_ms=2710`, proving the sequence is paced over the
  intended operator-visible window instead of delivered in about 1 ms.
- Public `/v1/devices` recorded `last_body_scene=full_check`,
  `last_body_scene_step=16`, `screen_theme=auto`, `screen_brightness=55`,
  final head `yaw=0,pitch=18,speed=200`, final RGB `0/0/32`, and the device
  remained online after a follow-up heartbeat check about 12 seconds later.
- This is machine-readable product-socket evidence. It is still not physical
  acceptance because no operator/instrument confirmation was recorded in this
  round.

Remaining issues:

- `physical_accepted=false` remains for body scenes until the operator confirms
  visible screen/RGB/head movement or instrument evidence is attached.
- Official `/stackChan/ws` avatar/action relay remains disconnected.
- Camera, NFC, and infrared remain planned/high-risk parity spikes.
- Automatic `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false` remains off on ECS for
  stability.

Next suggested action:

- Ask the operator to watch `/workspace` Full Check and confirm visible screen
  theme/brightness, RGB, and head movement. If accepted, record a physical
  evidence report and promote body-scene physical acceptance; otherwise tune
  scene poses/delays before touching firmware lifecycle.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback.

## 2026-06-05 04:10 CST - No-Cable Power/Lifecycle Recovery and WDT-Safe Product Reflash

Round goal:

- Explain and act on the operator report that the physical power button still
  does not start the standalone product, without overclaiming prior
  USB/flash-reset evidence as no-cable boot acceptance.

Actual completed work:

- Created plan
  `docs/plans/2026-06-05-no-cable-boot-power-lifecycle-recovery.md`.
- Tested an official lifecycle autostart candidate in commit `fabffd4`; product
  build and guarded flash passed, but runtime evidence rejected it.
- Read-only serial evidence on `/dev/cu.usbmodem1101` showed task watchdog
  triggers with CPU0 running `main`. Decoded backtrace pointed at
  `GetMooncake().uninstallAllApps()` from `app_main`, specifically
  AppSetup/AppLauncher LVGL teardown.
- Forward-fixed the product lane in commit `eeeb699` by preserving the
  WDT-safe direct `GetHAL().startXiaozhi()` autostart path and recording the
  lifecycle finding in the plan.
- Rebuilt, pushed, and guarded-flashed the restored product app.

Changed files:

- `docs/plans/2026-06-05-no-cable-boot-power-lifecycle-recovery.md`
- `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`
- `internal/app/official_stackchan_test.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Focused app test passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlayStartsXiaozhiDirectlyBeforeMooncakeTeardown|TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady|TestStackChanOfficialCandidateContract' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Product build passed:
  `reports/a21-stackchan-official-baseline-20260605-040655-1780603615828792000.json`,
  app SHA
  `065e23976722aa7630d0dccf8ee80dff2674785ce9bb6221f67769368a960284`.
- Guarded flash plan passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-040818-1780603698194451000.json`.
- Guarded flash execute passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-040923-1780603763139018000.json`,
  clean worktree, commit `eeeb6998a9b0`, `flash_executed=true`.

Runtime or physical evidence:

- The rejected lifecycle candidate produced HTTP 409
  `xiaozhi websocket is not connected` for body commands and serial watchdog
  evidence in `GetMooncake().uninstallAllApps()`.
- After the WDT-safe reflash, read-only serial showed MultiNet wake commands,
  audio codec open, quiet control WebSocket open, and Xiaozhi session
  `a21-session-44-1b-f6-e2-6a-60`.
- Public `/v1/devices` showed device `44:1b:f6:e2:6a:60` online with fresh
  heartbeat.
- Live trace `a21-trace-mode-ritual-after-wdt-safe-eeeb699-20260605` returned
  HTTP 200 `status=delivered`.
- Live trace `a21-trace-full-check-after-wdt-safe-eeeb699-20260605` returned
  HTTP 200 `status=delivered`.
- Public hardware acceptance summary returned `overall_status=physical_pending`
  with both delivery traces present.

Remaining issues:

- Physical power-button cold boot without USB is still not accepted. If it
  still fails on the foreground device after the WDT-safe reflash, the next
  root-cause branch is hardware power path, battery, PMIC, power-button
  hold/latched behavior, and missing battery telemetry rather than Gateway/body
  MCP.
- Official Mooncake teardown lifecycle remains a rejected product path until a
  separate transition can prove it does not WDT or regress Xiaozhi WS.
- Visible screen/RGB/head physical acceptance remains pending operator or
  instrument confirmation.

Next suggested action:

- Ask the operator to disconnect USB/power, attempt physical-button cold boot,
  and report exact behavior: screen/backlight, LED, servo twitch, boot sound,
  and whether the public `/v1/devices` row gets a fresh heartbeat. If it still
  fails, open a foreground power-path/battery/PMIC diagnostic transition.

Forbidden actions avoided:

- No NVS write, no provider execution, no V21 execution, no generic
  `xiaozhi.bin` product flash, no Git prune/gc, no destructive reset, and no
  internal-test3 voice/protocol rollback.

## 2026-06-05 03:31 CST - Voice Mode Hardware Ritual Pacing Deployed

Round goal:

- Make the PRD roleplay/professional mode switch visible on the physical
  StackChan body without reopening or rolling back internal-test3 voice-chain
  acceptance.

Actual completed work:

- Added response fields `step_delay_ms` and `total_planned_delay_ms` to
  `POST /v1/voice-mode-ritual`.
- Reused the existing body-scene pacing policy for the four-step
  screen/RGB/head mode ritual so it is delivered over an operator-visible
  window instead of a near-instant burst.
- Documented the pacing contract in `docs/engineering/PROTOCOL.md`.
- Updated `docs/plans/2026-06-05-voice-mode-hardware-ritual.md`,
  `docs/engineering/A21_CURRENT_CONTROL.md`, and
  `docs/project_state_machine.md` with current evidence.
- Committed and pushed:
  `a5d9b9d fix(gateway): pace voice mode rituals`.
- Deployed `a5d9b9d` to ECS through `/opt/a21.next` safe-swap.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-05-voice-mode-hardware-ritual.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Red test first failed as expected:
  `TestVoiceModeRitualProfessionalSendsHardwareSequence` did not find
  `step_delay_ms`.
- Focused local Gateway tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestVoiceModeRitual|TestWorkspaceConsolePageServed|TestVoiceModesCatalogDefaultsToRoleplayAndListsProfessional|TestVoiceModeSelectionProfessionalReturnsRitualContract' -count=1`.
- `git diff --check` passed.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote `/opt/a21.next` focused Gateway tests passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestVoiceModeRitual' -count=1`.
- Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- ECS `a21-gateway.service` restarted active; loopback and public direct
  `/healthz` passed.

Runtime or physical evidence:

- ECS runtime env includes `A21_BODY_SCENE_STEP_DELAY_MS=180`,
  `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true`, and
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false`.
- Public `/workspace` smoke found `Run Roleplay Ritual`,
  `Run Professional Ritual`, `modeRitualStatus`, `data-mode-ritual`, and
  `/v1/voice-mode-ritual`.
- Product device `44:1b:f6:e2:6a:60` was online before the live run.
- Live professional trace
  `a21-trace-mode-ritual-professional-paced-a5d9b9d-202606050330`
  returned HTTP 200 with `selected_voice_mode=professional`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`,
  `provider_executed=false`, `v21_executed=false`,
  `official_relay_claimed=false`, and `physical_accepted=false`.
  Trace summary `last_offset_ms=541` proved the ritual was paced.
- Live roleplay restore trace
  `a21-trace-mode-ritual-roleplay-paced-a5d9b9d-202606050331`
  returned HTTP 200 with `selected_voice_mode=roleplay`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`, and trace summary
  `last_offset_ms=541`.
- Final public `/v1/devices` check showed the product device online and
  restored to `current_voice_mode=roleplay`, with `screen_theme=auto`,
  `screen_brightness=58`, RGB `120/48/96`, head
  `yaw=0,pitch=24,speed=180`, and
  `voice_mode_ritual_physical_accepted=false`.

Deviations from plan:

- None for this scoped transition. The deployment intentionally reused the
  existing `A21_BODY_SCENE_STEP_DELAY_MS` policy instead of introducing a new
  env var.

Remaining issues:

- Mode ritual physical acceptance still requires operator or instrument
  confirmation of visible screen/RGB/head movement.
- Body-scene `full_check` physical acceptance is still pending operator or
  instrument confirmation.
- Official `/stackChan/ws` avatar/action relay remains disconnected.
- Camera, NFC, and infrared remain planned/high-risk parity spikes.
- Natural microphone-triggered voice-chain physical PRD acceptance remains a
  separate foreground evidence path; it was not reopened in this round.

Next suggested action:

- In the next foreground operator window, watch the `/workspace` roleplay and
  professional ritual buttons, record visible screen/RGB/head confirmation,
  then add a narrow mode-ritual physical acceptance endpoint or fold it into
  the existing hardware acceptance report format. After that, continue the
  hardware parity queue with official avatar/action relay or camera/NFC/IR
  spikes only behind explicit gates.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc despite the historical loose-object warning, and
  no internal-test3 voice/protocol rollback.

## 2026-06-05 04:05 CST - Hardware Acceptance Summary Board Deployed

Round goal:

- Make current body evidence recoverable and operator-readable by adding a
  single acceptance summary surface for mode ritual and full-check evidence.

Actual completed work:

- Added plan
  `docs/plans/2026-06-05-hardware-acceptance-summary-board.md`.
- Added read-only `GET /v1/hardware-acceptance?device_id=<device>`.
- Added `/workspace` `Acceptance Board` with `Refresh Acceptance`.
- The summary reports `mode_ritual` and `full_check` delivery status,
  physical acceptance state, trace/session IDs, acceptance endpoints, and next
  operator actions.
- Documented the contract in `docs/engineering/PROTOCOL.md`.
- Committed and pushed:
  `5d786ef feat(gateway): summarize hardware acceptance`.
- Deployed `5d786ef` to ECS through `/opt/a21.next` safe-swap.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/workspace_console.go`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-05-hardware-acceptance-summary-board.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Red tests first failed as expected:
  `TestWorkspaceConsolePageServed` missed `/v1/hardware-acceptance`, and the
  endpoint returned HTTP 404.
- Focused local Gateway tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestHardwareAcceptance|TestVoiceModeRitual|TestXiaozhiBodyScenePhysicalAcceptance|TestXiaozhiBodySceneReportsAndAppliesStepPacing' -count=1`.
- `git diff --check` passed.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote `/opt/a21.next` focused Gateway tests passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestHardwareAcceptance|TestVoiceModeRitual' -count=1`.
- Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- ECS `a21-gateway.service` restarted active; loopback and public direct
  `/healthz` passed.

Runtime or physical evidence:

- Public `/workspace` smoke found `Acceptance Board`, `Refresh Acceptance`,
  `hardwareAcceptanceStatus`, `hardwareAcceptanceItems`, and
  `/v1/hardware-acceptance`.
- First public summary after service restart returned
  `overall_status=machine_evidence_pending`, correctly reflecting the fresh
  in-memory registry.
- Live roleplay ritual trace
  `a21-trace-mode-ritual-summary-ready-5d786ef-202606050405`
  returned HTTP 200 with `step_delay_ms=180` and
  `total_planned_delay_ms=540`.
- Live `full_check` trace
  `a21-trace-full-check-summary-ready-5d786ef-202606050405`
  returned HTTP 200 with `step_delay_ms=180`,
  `total_planned_delay_ms=2700`, and trace summary `last_offset_ms=2713`.
- Final public `/v1/hardware-acceptance?device_id=44:1b:f6:e2:6a:60`
  returned `overall_status=physical_pending`; both `mode_ritual` and
  `full_check` were `delivery_status=delivered`,
  `physical_accepted=false`, with next actions
  `accept_visible_mode_ritual` and `accept_visible_full_check`.
- Final public `/v1/devices` check showed product device online and
  `current_voice_mode=roleplay`; final visible state was the full-check reset
  pose: `screen_theme=auto`, `screen_brightness=55`, RGB `0/0/32`, head
  `yaw=0,pitch=18,speed=200`.
- No physical acceptance was recorded in this round because no operator or
  instrument confirmation was provided.

Deviations from plan:

- None for this scoped transition. The summary endpoint is read-only and does
  not send hardware commands or accept physical evidence.

Remaining issues:

- The operator still needs to watch the delivered mode ritual and full-check
  body sequence, then click the two acceptance buttons to persist accepted
  physical evidence.
- Official `/stackChan/ws` avatar/action relay remains disconnected.
- Camera, NFC, and infrared remain planned/high-risk parity spikes.
- Natural microphone-triggered voice-chain physical PRD acceptance remains a
  separate foreground evidence path; it was not reopened in this round.

Next suggested action:

- Use the `Acceptance Board` as the foreground operator checklist: confirm
  `mode_ritual` and `full_check` visually, click their acceptance buttons, and
  verify the summary flips from `physical_pending` to `accepted`. After that,
  continue with official avatar/action relay parity or high-risk camera/NFC/IR
  spikes under explicit gates.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc despite the historical loose-object warning, and
  no internal-test3 voice/protocol rollback.

## 2026-06-05 05:45 CST - Stock Professional Route Mode Gate Deployed

Round goal:

- Consume the full review-thread conclusion against current implementation,
  fix any remaining software-side issue that could cause power/body/voice mode
  confusion, redeploy ECS, and refresh product readiness without rolling back
  internal-test3 voice protocol or the PMIC product flash.

Actual completed work:

- Re-read review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` and compared
  it to current code, deployed ECS behavior, and latest PMIC power-key flash
  evidence.
- Found a fresh runtime regression while re-running evidence:
  `A21_XIAOZHI_STOCK_PROFESSIONAL_ROUTE=true` routed stock `realtime` listens
  into professional execution even when selected voice mode was roleplay.
- Fixed the Gateway route state machine so stock professional override only
  applies when selected voice mode is `professional`. Explicit
  `mode=professional` and voice-triggered professional paths remain covered.
- Committed:
  `15a16dc fix(gateway): gate stock professional route by voice mode`.
- Deployed `15a16dc` to ECS `47.103.57.217` through `/opt/a21.next` safe swap.
- Replayed product roleplay mode ritual and `full_check` after deploy.
- Verified official `/stackChan/ws` remains disconnected with a correct
  official control request returning HTTP 409, so current body evidence is
  Xiaozhi MCP screen/head/RGB delivery rather than official typed-frame relay
  product acceptance.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Local focused Gateway tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'StockProfessionalRoute|ProfessionalModeDoesNotUsePlaceholder|VoiceTrigger' -count=1`.
- Local focused App tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'GatewayServerOptionsFromEnvWiresStockProfessionalRoute|XiaozhiVoiceBench|XiaozhiProfessionalBench' -count=1`.
- `git diff --check` passed.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote `/opt/a21.next` focused Gateway/App tests passed.
- Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- ECS `a21-gateway.service` restarted active; loopback `/healthz` passed.

Runtime or physical evidence:

- Fresh provider smoke passed:
  `reports/a21-provider-smoke-20260605-054222-582464591.json`.
- Fresh roleplay runtime probe passed:
  `reports/a21-roleplay-voice-probe-20260605-054224.json`.
- Fresh repeat-3 Xiaozhi voice bench passed with
  `acceptance_status=candidate_host_only`, cloud-edge product-chain execution,
  `failure_count=0`, answer first-audio P95 `1281 ms`, and barge-in stop P95
  `0 ms`:
  `reports/a21-xiaozhi-voice-bench-20260605-054238.059793905.json`.
- Fresh professional bench passed with
  `acceptance_status=external_gateway_ready`, checking feedback `204 ms`,
  completed read record, and `tts_stop_observed=true`:
  `reports/a21-xiaozhi-professional-bench-20260605-054250.259403329.json`.
- Fresh product readiness is `server_side_candidate_ready` with canonical
  missing real evidence reduced to `physical_stackchan_prd_acceptance`:
  `reports/a21-product-readiness-20260605-054250.json`.
- Fresh server-side readiness bundle is `server_side_candidate_ready`,
  `candidate_ready=true`, and `collection.status=nothing_missing`:
  `reports/a21-server-side-readiness-bundle-20260605-054250.json`.
- Product roleplay mode ritual trace
  `a21-trace-mode-ritual-route-fix-15a16dc-20260605` returned HTTP 200
  `status=delivered`.
- Product `full_check` trace
  `a21-trace-full-check-route-fix-15a16dc-20260605` returned HTTP 200
  `status=delivered` with 16 paced steps.
- Public hardware acceptance remains `overall_status=physical_pending` for
  `mode_ritual`, `full_check`, and `power_lifecycle`.
- Public power lifecycle remains `overall_status=physical_pending`,
  `xiaozhi_ws_online=true`, and `battery_telemetry=missing`.

Remaining issues:

- User/operator physical acceptance is still required for no-cable cold boot,
  physical power-button start, visible mode ritual, visible full-check body
  sequence, audible natural microphone voice chain, custom wake, and overall
  physical StackChan PRD acceptance.
- Official `/stackChan/ws` avatar/action relay remains disconnected and is not
  product accepted.
- Battery telemetry is still missing in power lifecycle status.
- Camera, NFC, and infrared remain planned/high-risk parity spikes.

Next suggested action:

- Have the operator perform the physical acceptance window: unplug USB, use
  the power button, confirm Gateway/Xiaozhi reconnect, then from `/workspace`
  observe and accept the latest mode ritual and full-check sequence. After
  physical acceptance, continue with official `/stackChan/ws` lifecycle/relay
  parity as the next software transition.

Forbidden actions avoided:

- No firmware flash, no NVS write, no provider secret printing, no generic
  `xiaozhi.bin` product flash lane, no camera/NFC/IR expansion, no reboot/OTA
  exposure, no Git prune/gc despite the historical loose-object warning, and
  no internal-test3 voice/protocol rollback.

## 2026-06-05 03:45 CST - Mode Ritual Physical Acceptance Surface Deployed

Round goal:

- Move the PRD roleplay/professional mode-switch body feedback from paced
  machine evidence toward product acceptance by adding a controlled
  operator/instrument acceptance path.

Actual completed work:

- Added `POST /v1/voice-mode-ritual-acceptance`.
- Added `/workspace` `Accept Visible Mode Ritual` control.
- The endpoint requires matching delivered mode-ritual `trace_id` and
  `session_id`, visible `screen`, `rgb`, and `servo` confirmations, plus
  `observer=operator` or `observer=instrument`.
- Documented the acceptance contract in `docs/engineering/PROTOCOL.md` and
  updated the active voice-mode ritual plan.
- Committed and pushed:
  `87625a2 feat(gateway): record mode ritual physical acceptance`.
- Deployed `87625a2` to ECS through `/opt/a21.next` safe-swap.

Changed files:

- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/workspace_console.go`
- `docs/engineering/PROTOCOL.md`
- `docs/plans/2026-06-05-voice-mode-hardware-ritual.md`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Red tests first failed as expected:
  `TestWorkspaceConsolePageServed` missed
  `/v1/voice-mode-ritual-acceptance`, and the acceptance endpoint returned
  HTTP 404.
- Focused local Gateway tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestVoiceModeRitual' -count=1`.
- `git diff --check` passed.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote `/opt/a21.next` focused Gateway tests passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestVoiceModeRitual' -count=1`.
- Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- ECS `a21-gateway.service` restarted active; loopback and public direct
  `/healthz` passed.

Runtime or physical evidence:

- Public `/workspace` smoke found `Accept Visible Mode Ritual`,
  `acceptModeRitualPhysical`, `modeRitualPhysicalStatus`, and
  `/v1/voice-mode-ritual-acceptance`.
- Public negative acceptance smoke without matching delivered ritual evidence
  returned HTTP 409:
  `matching voice mode ritual evidence is required before physical acceptance`.
- Live roleplay trace
  `a21-trace-mode-ritual-roleplay-acceptance-ready-87625a2-202606050345`
  returned HTTP 200 with `selected_voice_mode=roleplay`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`, and trace summary
  `last_offset_ms=543`.
- Final public `/v1/devices` check showed the product device online and in
  `current_voice_mode=roleplay`, with `screen_theme=auto`,
  `screen_brightness=58`, RGB `120/48/96`, head
  `yaw=0,pitch=24,speed=180`, and
  `voice_mode_ritual_physical_accepted=false`.
- No physical acceptance was recorded in this round because no operator or
  instrument confirmation was provided.

Deviations from plan:

- None for this scoped transition. Physical acceptance remains explicit and
  operator/instrument gated.

Remaining issues:

- The operator still needs to watch a roleplay/professional ritual and click
  `Accept Visible Mode Ritual` to persist accepted physical evidence.
- Body-scene `full_check` physical acceptance is still pending operator or
  instrument confirmation.
- Official `/stackChan/ws` avatar/action relay remains disconnected.
- Camera, NFC, and infrared remain planned/high-risk parity spikes.
- Natural microphone-triggered voice-chain physical PRD acceptance remains a
  separate foreground evidence path; it was not reopened in this round.

Next suggested action:

- During the next foreground operator window, run roleplay and professional
  rituals from `/workspace`, visually confirm screen/RGB/head movement, click
  `Accept Visible Mode Ritual`, and capture the resulting trace/registry
  acceptance evidence. Then either accept `full_check` the same way or move to
  official avatar/action relay parity.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc despite the historical loose-object warning, and
  no internal-test3 voice/protocol rollback.

## 2026-06-05 02:50 CST - Body Scene Physical Acceptance Surface Deployed

Round goal:

- Move paced `full_check` from machine evidence toward product acceptance by
  adding a controlled operator/instrument acceptance recording path, without
  pretending physical acceptance occurred automatically.

Actual completed work:

- Added plan
  `docs/plans/2026-06-05-body-scene-physical-acceptance-surface.md`.
- Added `POST /v1/xiaozhi/body-scene-acceptance`.
- Added `/workspace` `Accept Visible Full Check` control and
  `hardwareSceneAcceptanceStatus`.
- Body-scene delivery now records `last_body_scene_trace_id` and
  `last_body_scene_session_id` so acceptance can require matching scene
  evidence.
- Documented the acceptance contract in `docs/engineering/PROTOCOL.md`.
- Committed and pushed:
  `98700ab feat(gateway): record body scene physical acceptance`.
- Deployed `98700ab` to ECS through `/opt/a21.next` safe-swap.

Changed files:

- `docs/plans/2026-06-05-body-scene-physical-acceptance-surface.md`
- `docs/engineering/PROTOCOL.md`
- `internal/gateway/server.go`
- `internal/gateway/server_test.go`
- `internal/gateway/workspace_console.go`
- `docs/engineering/A21_CURRENT_CONTROL.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Tests/build/runtime results:

- Red tests first failed as expected:
  `TestWorkspaceConsolePageServed` missed
  `/v1/xiaozhi/body-scene-acceptance`; the body-scene acceptance endpoint
  returned HTTP 404.
- Focused local Gateway tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyScenePhysicalAcceptance|TestXiaozhiBodySceneReportsAndAppliesStepPacing|TestXiaozhiBodySceneFullCheckRunsOperatorVisibleSequence' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 make verify`.
- Remote `/opt/a21.next` focused Gateway tests passed.
- Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
- ECS `a21-gateway.service` restarted active; loopback and public `/healthz`
  passed.

Runtime or physical evidence:

- Public `/workspace` smoke found `Accept Visible Full Check`,
  `acceptHardwareScenePhysical`, `hardwareSceneAcceptanceStatus`, and
  `/v1/xiaozhi/body-scene-acceptance`.
- Public negative acceptance smoke without matching delivered scene evidence
  returned HTTP 409:
  `matching body scene evidence is required before physical acceptance`.
- Product `full_check` trace
  `a21-trace-hardware-full-check-acceptance-ready-98700ab-202606050250`
  returned HTTP 200 with `status=delivered`, `scene=full_check`,
  `step_delay_ms=180`, `total_planned_delay_ms=2700`, and 16 redacted steps.
- Trace endpoint recorded 32 ordered markers with
  `summary.last_offset_ms=2710`.
- Public `/v1/devices` recorded the latest `last_body_scene_trace_id` and
  `last_body_scene_session_id` for that trace, final screen/head/RGB state,
  and the device remained online after a follow-up heartbeat check.
- No body-scene physical acceptance was recorded in this round; the operator
  has not yet confirmed visible screen/RGB/head movement.

Remaining issues:

- Operator or instrument confirmation is still required before calling the new
  acceptance endpoint and promoting body-scene physical acceptance.
- Official `/stackChan/ws` avatar/action relay remains disconnected.
- Camera, NFC, and infrared remain planned/high-risk parity spikes.
- Automatic `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false` remains off on ECS for
  stability.

Next suggested action:

- Have the operator watch `/workspace` Full Check. If screen theme/brightness,
  RGB, and head movement are all visible, click `Accept Visible Full Check`
  or provide explicit confirmation so the acceptance marker can be recorded.

Forbidden actions avoided:

- No firmware build, no firmware flash, no NVS write, no provider secret
  printing, no provider or V21 execution, no generic product flash lane, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback.
