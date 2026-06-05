# A21 Project State Machine

Status: active state document.
Last updated: 2026-06-05.

## Stabilization Override

Active override transition:
`T-A21-LEAN-CARVE-MAINLINE-001`.

Current override state:
`S-LEAN-CARVE-GOVERNANCE-GATES-PASSED-RUNTIME-GATEWAY-EOF-BLOCKED`.

Internal test 4 is now the protected recovery baseline. Until P0 stabilization
exits, no new feature work, product flash, provider chain change, side-branch
merge, or worker expansion is allowed unless it is explicitly tied to rollback,
recovery, or a P0 acceptance transition.

The 2026-06-05 lean carve temporarily supersedes the P0 voice-loop RCA as the
active control transition because the user explicitly ordered governance
contraction after freezing internal test 4. The prior P0 RCA state remains
preserved below and must not be re-expanded until the carve's runtime blocker
is closed.

Recovery baseline:
`docs/engineering/A21_INTERNAL_TEST4_RECOVERY_BASELINE.md`.

Active stabilization plan:
`docs/plans/2026-06-05-a21-stabilization-after-internal-test4.md`.

Latest P0 power boot RCA:
`docs/engineering/A21_P0_POWER_BOOT_RCA_20260605.md`.

Latest P0 voice-loop RCA:
`docs/engineering/A21_P0_VOICE_LOOP_RCA_20260605.md`.

Active transition:
`T-A21-LEAN-CARVE-MAINLINE-001`.

Latest lean carve update, 2026-06-05 21:20 CST:

- Branch `main-lean` was created from frozen HEAD `090eca6`.
- Local branch fan-in completed to exactly three branches:
  `main-lean`, `codex/a21-stabilization-after-internal-test4-20260605`, and
  `codex/a21-hardware-window-20260605-product-flash-9f4532a`.
- Product/lab split completed at command-dispatch and binary-reachability
  level. Product `cmd/a21` no longer dispatches lab demo/bench/evidence/
  professional commands; `cmd/a21-lab` owns those execution paths.
- Product readiness no longer promotes simulator/mock demo proof as
  product-ready, and product Gateway no longer serves the simulator route by
  default.
- Oversized Go files were mechanically split; current maximum product
  implementation file is 773 lines and current maximum test file is 896 lines.
- Lean verification passed: `go test ./...`, `make verify`, sequential
  `make preflight`, `make doctor`, and `bash scripts/lean-gate.sh`.
- Product dependency and product-binary checks passed with no
  bench/demo/evidence/professional dependency path matches and no
  lab/simulator binary symbol matches.
- The carve is not complete: `make stackchan-fast-companion-turn` with true
  provider intent failed before timing evidence because public Gateway
  `/v1/devices` returned EOF. No current p95/barge-in lock exists.
- Recovery probing from this workspace is externally blocked: public direct
  `/healthz` and `/v1/devices` return `Empty reply from server`, and SSH to
  `root@47.103.57.217` closes the port 22 connection before remote commands can
  run.
- Next state candidate:
  `S-LEAN-CARVE-RUNTIME-GATEWAY-RECOVERED-FAST-COMPANION-LOCK-PENDING`.
- Next action: recover the ECS Gateway/public `/v1/devices` path, rerun the
  true-provider StackChan fast companion turn, and record p95 plus barge-in
  stop metrics in `docs/lean/CARVE_LOG.md`.

This document records A21 as a set of explicit transitions. A conversation is an
execution surface; the repository state, plans, handoff log, tests, and evidence
are the project memory.

## Project State

Current total state: `S-INTERNAL-TEST4-ROLEPLAY-SOUL-PROFILE-READY-ROLEPLAY-PROMPT-VOICE-CLONE-PIPELINE-READY-ROLEPLAY-DEVICE-STATE-REFLECTION-READY-ROLEPLAY-OFFICIAL-EXPRESSION-PLAN-READY-ROLEPLAY-IMMERSION-READINESS-READY-ROLEPLAY-VOICE-RUNTIME-EVIDENCE-READY-ROLEPLAY-VOICE-PROBE-REPORT-GENERATOR-READY-SERVER-SIDE-ROLEPLAY-VOICE-RUNTIME-GATE-READY-WORKSPACE-CONSOLE-PRODUCT-SURFACE-READY-WORKSPACE-DEVICE-BINDING-GUARD-READY-WORKSPACE-PROFESSIONAL-QUERY-ENDPOINT-READY-WORKSPACE-VOICE-PROBE-CONTROL-SURFACE-READY-WORKSPACE-BODY-PRESET-CONTROL-SURFACE-DEPLOYED-WORKSPACE-HARDWARE-SCREEN-CONTROL-SURFACE-DEPLOYED-WORKSPACE-OFFICIAL-ACTION-CONTROL-SURFACE-DEPLOYED-WORKSPACE-OFFICIAL-ACTION-FALLBACK-READY-WORKSPACE-HARDWARE-SCENE-CONTROL-SURFACE-DELIVERED-WORKSPACE-HARDWARE-FULL-CHECK-DEPLOYED-XIAOZHI-LISTEN-START-STATE-REACTION-SUPPRESSED-XIAOZHI-BODY-MOTION-SEQUENCE-DEPLOYED-SELECTED-VOICE-CHAIN-READINESS-INGRESS-READY-WORKSPACE-SOURCE-READINESS-READY-WORKSPACE-DOCUMENT-UPLOAD-INTAKE-READY-V21-SOURCE-SCOPE-RETRIEVAL-GUARD-READY-A21-V21-NATIVE-VOICE-QUERY-BRIDGE-READY-PROFESSIONAL-VOICE-TRIGGER-READY-MCP-SPEAKER-VOLUME-FROZEN-OFFICIAL-ROBOT-MCP-BODY-CONTROLS-DEPLOYED-CLOUD-UPLOAD-INDEX-EXECUTION-PLANNED-FIRMWARE-QUIET-RECONNECT-CANDIDATE-FLASHED-BODY-SCENE-MACHINE-EVIDENCE-READY-BODY-FULL-CHECK-MACHINE-EVIDENCE-READY-BODY-FULL-CHECK-PACED-MACHINE-EVIDENCE-READY-BODY-SCENE-PHYSICAL-ACCEPTANCE-SURFACE-DEPLOYED-VOICE-MODE-HARDWARE-RITUAL-PACED-DEPLOYED-VOICE-MODE-RITUAL-PHYSICAL-ACCEPTANCE-SURFACE-DEPLOYED-HARDWARE-ACCEPTANCE-SUMMARY-BOARD-DEPLOYED-CONNECTED-HARDWARE-AUTO-ADOPTION-DEPLOYED-PRODUCT-OFFICIAL-COMPATIBLE-FLASHED-AFTER-FLASH-BODY-EVIDENCE-READY-PHYSICAL-PENDING-PRODUCT-DIRECT-START-WDT-SAFE-RESTORED-GATEWAY-XIAOZHI-REVIEW-REMEDIATED-POWER-LIFECYCLE-STATE-MACHINE-DEPLOYED-STACKCHAN-PMIC-POWER-KEY-PARITY-FLASHED-NO-CABLE-POWER-PHYSICAL-PENDING-STOCK-PROFESSIONAL-ROUTE-MODE-GATED-SERVER-SIDE-CANDIDATE-READY-OFFICIAL-STACKCHAN-RELAY-RUNTIME-BUILD-READY-GUARDED-MANUAL-BOOTLOADER-FLASH-READY-PHYSICAL-ROM-DOWNLOAD-PENDING-SERVER-SIDE-CANDIDATE-RECONFIRMED-READINESS-REMOTE-CONTEXT-ALIGNED-WAKE-PHYSICAL-LAUNCH-GATE-ENFORCED-PRODUCT-FLASH-WAIT-ROM-GUARD-READY-PRODUCT-FLASH-ROM-DIAGNOSTIC-READY-OFFICIAL-STACKCHAN-RELAY-STATUS-SURFACE-READY-PRODUCT-ROM-DOWNLOAD-STILL-PENDING-PRODUCT-RECOVERY-PRECHECK-READY-OFFICIAL-STACKCHAN-BACKEND-MCP-FALLBACK-READY-XIAOZHI-PRODUCT-STT-SCREEN-AND-FAST-ACK-GUARD-READY-STACKCHAN-PRODUCT-RECOVERY-EXECUTOR-READY-POWER-LIFECYCLE-COLD-BOOT-EVIDENCE-GUARD-READY-STACKCHAN-SYS-EVT-BOOT-LOOP-RECOVERED-OFFICIAL-AVATAR-RELAY-CONNECTED-SERIAL-EVIDENCE-READY-STACKCHAN-DEVICE-ID-CASEFOLD-RECOVERY-DEPLOYED-PRODUCT-ONLINE-OFFICIAL-RELAY-READY-STACKCHAN-NATIVE-MOTION-POWER-BUILD-READY-STACKCHAN-NATIVE-MOTION-POWER-DEPLOYED-FLASHED-PHYSICAL-PENDING-STACKCHAN-OFFICIAL-POWER-UI-PARITY-BUILD-READY-STACKCHAN-OFFICIAL-POWER-UI-PARITY-FLASHED-PHYSICAL-PENDING-STACKCHAN-OFFICIAL-FRONTEND-RESTORED-BLE-APP-BINDING-SECRET-PENDING-STACKCHAN-OFFICIAL-HOME-PMIC-POWER-FLASHED-PHYSICAL-PENDING-STACKCHAN-PMIC-COLD-BOOT-DIAGNOSTIC-FLASHED-PHYSICAL-PENDING-STACKCHAN-PMIC-BOOT-SNAPSHOT-FLASHED-SERIAL-EVIDENCE-READY-STOCK-OFFICIAL-AB-BASELINE-BUILT-STOCK-OFFICIAL-AB-FLASH-GUARD-READY-STOCK-OFFICIAL-AB-FLASHED-PHYSICAL-PENDING-PRODUCT-PMIC-STARTUP-STOCK-PARITY-RESTORED-PRODUCT-FLASHED-PHYSICAL-PENDING`

Latest foreground hardware-window update, 2026-06-05 16:34 CST:

- Commit `998a4bba6e9d fix(stackchan): preserve stock pmic startup parity`
  restored A21 product PMIC startup parity with stock official and removed the
  speculative `REG10`/`REG22`/`REG24` write hunk.
- The A21 product candidate was restored through the guarded
  official-compatible flash lane on `/dev/cu.usbmodem1101`.
- Product flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-163318-1780648398612612000.json`.
- Flashed app artifact:
  `a21-stackchan-official-xiaozhi-compatible.bin`, SHA-256
  `5968211923f788666e08bca51740e691dd17ae36d2535d8c265ced73d3abbf23`.
- USB serial evidence shows the product boots to official Launcher, initializes
  the expected hardware stack, and records read-only PMIC snapshot
  `r00=28,r01=14,r10=34,r12=00,r14=65,r20=04,r21=20,r22=06,r23=3f,r24=00,r26=08,r27=00,r30=3f,r61=05,r62=0d,r63=15,r64=03,r80=05,r82=12,r90=3f,r91=00,r92=0d,r94=1c,r95=1c,r97=1c,r99=18,a4=64,a5=00`.
- Current device state is restored A21 product candidate, not stock baseline.
  No-USB PWRKEY and AI.AGENT voice/body UX need foreground physical evidence.

Latest Gateway state-machine update, 2026-06-05 16:50 CST:

- Official Xiaozhi reference comparison found that normal server `tts.stop`
  after a completed answer intentionally drives the device back to
  `kDeviceStateListening` unless the device is in manual-stop mode.
- In the official `Listening` state, the device waits for playback drain in
  auto mode, then sends `listen.start` and enables voice processing; the
  Gateway must not suppress that normal auto-listen restart.
- The current Gateway candidate removes post-`tts.stop` input suppression for
  normal completed answers and keeps suppression only for non-normal local
  fallback, placeholder, host-say, degraded, error, or unavailable paths.
- The candidate also prevents an already-stopped turn from being classified as
  a new barge-in when the official device immediately re-enters listening.
- Focused Gateway tests and broad Xiaozhi/official StackChan tests passed
  before this state update; deployment to ECS is the next action.

Latest AI.AGENT body update, 2026-06-05 17:05 CST:

- Post-deploy public status showed the product Xiaozhi socket online but the
  official `/stackChan/ws` relay disconnected, with fallback available.
- Official StackChan source comparison found that the WebSocket avatar/body
  relay is an `AppAvatar` ability. The official `AI.AGENT` path requests
  Xiaozhi start, unloads Mooncake apps, and enters Xiaozhi runtime; therefore
  `/stackChan/ws` is not a reliable body control surface inside `AI.AGENT`.
- The Gateway candidate now keeps official relay as the first transport when
  it is connected, but falls back to official Xiaozhi MCP robot tools
  `self.robot.set_led_color` and `self.robot.set_head_angles` when the relay
  is disconnected or delivery fails.
- Focused Gateway tests passed for both official relay delivery and
  AI.AGENT/Xiaozhi MCP fallback delivery.

Latest firmware state-machine parity candidate, 2026-06-05 17:18 CST:

- Official Xiaozhi `HandleStartListeningEvent` comparison found manual
  touch/start always uses `kListeningModeManualStop`.
- The A21 official-compatible overlay had changed that path to
  `GetDefaultListeningMode()`, which can turn manual touch into auto/realtime
  listening and diverge from official stop/listen behavior.
- The current firmware candidate removes that overlay hunk so the official
  manual-start semantics remain intact while keeping the A21 quiet control
  channel and wake-word invoke patches.
- Focused official-compatible overlay tests passed before this state update;
  full verify and guarded product build/flash remain next actions.

Latest foreground hardware-window update, 2026-06-05 16:11 CST:

- T7-guarded stock-official baseline diagnostic flash passed at commit
  `a923ac17b5ce`.
- The flashed app is pure official no-overlay
  `/tmp/a21-stackchan-official-stock-build/stack-chan.bin`, SHA-256
  `a0cd9129b9e5f4718893d4fa672cb62e57088d5585057a1e4fa8ec835018135e`.
- Flash report:
  `reports/a21-stackchan-official-baseline-flash-20260605-161057-1780647057743653000.json`.
- The device is temporarily on stock official baseline pending physical no-USB
  PWRKEY test. A21 product candidate must be restored after this A/B evidence
  is captured.

Latest foreground hardware-window update, 2026-06-05 16:32 CST:

- USB serial capture from the current stock-official baseline proved the device
  is not bricked under USB: stock `stack-chan` 1.4.1 initializes PMIC, display,
  camera, touch, MCP, head touch, IO expander, RTC, IMU, servos, and reaches
  official Launcher.
- A stock-vs-A21 generated-tree comparison found `main.cpp` and `AI.AGENT`
  startup entry identical. The official front-end remains the first runtime;
  A21/Xiaozhi starts only after the operator selects `AI.AGENT`.
- The only PMIC startup delta was the A21 overlay hunk writing AXP2101
  registers `0x10`, `0x22`, and `0x24`. It did not close the no-USB symptom and
  diverged from stock official, so it has been removed.
- The rebuilt A21 product candidate now preserves stock PMIC startup writes and
  adds only read-only PMIC diagnostics. Generated-tree diff confirms
  `WriteReg(0x27, 0x00)` is the only PMIC startup write in both stock and A21.
- Verification passed: focused overlay/product contract tests, `git diff
  --check`, `GOMAXPROCS=2 make verify`, and guarded product candidate build.
- New product app artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`,
  SHA-256 `5968211923f788666e08bca51740e691dd17ae36d2535d8c265ced73d3abbf23`.
- Next action is to commit/push this correction, restore the A21 product
  candidate through the official-compatible product flash lane, capture USB
  serial PMIC snapshot, and repeat no-USB PWRKEY with explicit hold-time
  evidence. Do not reintroduce PMIC write churn without new evidence.

Latest foreground hardware-window update, 2026-06-05 16:08 CST:

- The first attempt to execute the stock-official baseline diagnostic flash
  correctly stopped before flashing because the new command had not yet been
  registered as a known T7 hardware-write command.
- Added the missing runtimeguard spec for
  `stackchan-official-baseline-flash --execute` and
  `stackchan-official-baseline-flash-execute`, including `--execute`
  classification.
- Added a focused test proving the baseline diagnostic flash remains a guarded
  hardware write requiring a foreground `codex/a21-hardware-window-*` branch,
  clean worktree, and non-detached HEAD.
- Focused runtimeguard tests passed, focused app flash-lane contract tests
  passed, `git diff --check` passed, and `GOMAXPROCS=2 make verify` passed.
- No firmware flash occurred in this update. Next action is a controlled
  stock-official physical A/B flash followed by immediate product-lane restore.

Latest foreground hardware-window update, 2026-06-05 15:56 CST:

- After the operator reported the same no-USB cold-boot failure after the PMIC
  boot-snapshot flash, the uncommitted `REG27=0x10` experiment was reverted
  before any build or flash.
- A clean state-machine comparison found current A21 product `main.cpp` still
  matches official StackChan `HEAD`: official Launcher/Home runs first, and
  Xiaozhi starts only after `AI.AGENT` requests it.
- The unresolved failure boundary is now explicitly before Gateway/voice:
  `physical upper-left PWRKEY -> AXP2101 battery/PMIC rail hold -> ESP32 boot`.
- A clean stock-official no-overlay build passed in isolated directories
  without flashing. Report:
  `reports/a21-stackchan-official-baseline-20260605-155153-1780645913655503000.json`.
- Stock-official no-overlay app artifact:
  `/tmp/a21-stackchan-official-stock-build/stack-chan.bin`, SHA-256
  `a0cd9129b9e5f4718893d4fa672cb62e57088d5585057a1e4fa8ec835018135e`.
- Added and dry-run verified the guarded stock-official diagnostic flash lane:
  `stackchan-official-baseline-flash-plan` /
  `stackchan-official-baseline-flash-execute`.
- No-flash plan report on `/dev/cu.usbmodem1101`:
  `reports/a21-stackchan-official-baseline-flash-20260605-160124-1780646484411288000.json`.
- Next action: choose a controlled physical A/B path. Either temporarily flash
  stock-official as explicit non-product diagnostic evidence and restore the
  A21 product candidate immediately afterward, or run a single-variable
  product-lane candidate with a recorded hypothesis. Do not add PMIC register
  churn without this decision.

Latest foreground hardware-window update, 2026-06-05 15:33 CST:

- The first attempt to flash the PMIC boot-snapshot candidate failed during app
  write because USB serial disconnected at about 40% with
  `Device not configured`; this left the flash attempt incomplete.
- After the operator reconnected the cable, the same guarded product lane was
  rerun successfully at commit `4812f85bad8a`.
- Successful flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-153312-1780644792918047000.json`.
- The flashed app artifact is
  `a21-stackchan-official-xiaozhi-compatible.bin` with SHA-256
  `a99ac8b1311a122865136ff52f06160c659e2e89465361cefd26eacfc7322684`.
- USB serial boot evidence captured the new PMIC snapshot:
  `r00=38,r01=35,r10=34,r12=00,r14=65,r20=04,r21=20,r22=06,r23=3f,r24=00,r26=08,r27=00,r30=3f,r61=05,r62=0d,r63=15,r64=03,r80=05,r82=12,r90=3f,r91=00,r92=0d,r94=1c,r95=1c,r97=1c,r99=18,a4=64,a5=00`.
- USB boot reaches official Launcher with PMIC, display, camera, touch, MCP,
  head touch, IO expander, RTC, IMU, and servos initialized. Gateway remains
  stale until the operator opens `AI.AGENT`.
- Next action: operator unplugs USB completely, waits for full power-off, then
  presses the upper-left power key and reports whether the device reaches
  official Home.

Previous foreground hardware-window update, 2026-06-05 15:29 CST:

- Gateway still shows the product device stale from the earlier session, so the
  15:15 flashed candidate has not produced new `pmic_power_status` evidence
  through `AI.AGENT`.
- USB serial capture proves the product app boots through official Launcher and
  initializes PMIC, display, camera, touch, IMU, servos, and official app list;
  no WDT/boot-loop or immediate power-save shutdown was observed.
- Added a stronger PMIC evidence candidate: boot-time serial log
  `A21 PMIC boot-after-init: ...` plus expanded PMIC raw snapshot fields for
  charger and rail registers.
- Focused overlay tests passed, `git diff --check` passed, and
  `GOMAXPROCS=2 make verify` passed.
- Guarded product build passed. App artifact:
  `a21-stackchan-official-xiaozhi-compatible.bin`; SHA-256
  `a99ac8b1311a122865136ff52f06160c659e2e89465361cefd26eacfc7322684`;
  build report:
  `reports/a21-stackchan-official-baseline-20260605-152919-1780644559915124000.json`.
- Next action: commit and push this build-ready transition, flash it through
  the guarded official-compatible product lane, then capture USB boot serial
  logs to inspect the PMIC register values before repeating no-USB acceptance.

Previous foreground hardware-window update, 2026-06-05 15:15 CST:

- Commit `1106dd49dc8e fix(stackchan): add pmic cold boot diagnostics` was
  pushed and product-flashed through the guarded official-compatible lane on
  `/dev/cu.usbmodem1101`.
- Flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-151446-1780643686932286000.json`.
- The flashed app artifact is
  `a21-stackchan-official-xiaozhi-compatible.bin` with SHA-256
  `31615f23fb2dc5d8e746fc6b6ed3186b65a020f29e9c72f114b89b39b4aaf36e`.
- T7 control guard accepted only the clean foreground hardware-window branch at
  commit `1106dd49dc8e`; flash executed successfully and wrote bootloader, app,
  partition table, OTA data, and generated assets.
- Public Gateway `/healthz` passed after flash. `/v1/devices` still shows the
  product device stale from the earlier session, and
  `/v1/stackchan/official/status` reports `connected=false` until the operator
  opens official `AI.AGENT`.
- Next action: operator confirms the device reaches official Home on USB, opens
  `AI.AGENT` once so Gateway records `pmic_power_status`, then physically tests
  no-USB upper-left power-key cold boot and shutdown/restart.

Previous foreground hardware-window update, 2026-06-05 15:10 CST:

- After the operator confirmed the restored official Home/AI.AGENT entry works,
  physical no-USB cold boot still failed; the previous PMIC parity flash did
  not complete product acceptance.
- The active failure boundary remains before Gateway/provider/voice: pressing
  the upper-left power key on battery flashes screen/red LED but does not
  sustain startup to official Home.
- Added the next official-compatible product candidate with `REG24 = 0x00` to
  lower the battery-voltage power-off threshold for cold-boot inrush tolerance,
  while preserving official Home/setup/mobile surfaces and `AI.AGENT` entry.
- Added read-only AXP2101 raw register heartbeat evidence as
  `pmic_power_status`; no remote power-control surface was added.
- Focused overlay tests passed, `git diff --check` passed, and
  `GOMAXPROCS=2 make verify` passed.
- Guarded product build passed. App artifact:
  `a21-stackchan-official-xiaozhi-compatible.bin`; SHA-256
  `31615f23fb2dc5d8e746fc6b6ed3186b65a020f29e9c72f114b89b39b4aaf36e`;
  build report:
  `reports/a21-stackchan-official-baseline-20260605-150958-1780643398410587000.json`.
- Next action: commit and push this build-ready transition, then run the
  guarded official-compatible product flash lane on `/dev/cu.usbmodem1101`.

Previous foreground hardware-window update, 2026-06-05 14:28 CST:

- Commit `9e6bfad92abc fix(stackchan): restore official home pmic power parity`
  was pushed and product-flashed through the guarded official-compatible lane on
  `/dev/cu.usbmodem1101`.
- Flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-142805-1780640885284551000.json`.
- The flashed app artifact is
  `a21-stackchan-official-xiaozhi-compatible.bin` with SHA-256
  `1f1348156a3312c76059f0a084d956d13ca41cc74a22ca2c0b8884b4ea1a9aa3`.
- T7 control guard accepted only the clean foreground hardware-window branch at
  commit `9e6bfad92abc`; flash executed successfully and wrote bootloader, app,
  partition table, OTA data, and generated assets.
- Public Gateway `/healthz` passed after flash. `/v1/devices` and
  `/v1/power-lifecycle` still show the product device as stale until the
  operator opens official `AI.AGENT` or the device reconnects; no-USB physical
  power acceptance remains pending.
- Next action: operator unplugs USB, lets the unit fully power down, presses
  the upper-left power key, and confirms whether it cold boots to official Home
  and can shut down/restart without USB.

Previous foreground hardware-window update, 2026-06-05 14:24 CST:

- After the operator confirmed official Home entry, no-USB power-key boot still
  failed with screen/red LED flash and no sustained startup.
- Added
  `docs/plans/2026-06-05-stackchan-official-home-pmic-power-key-parity.md`
  to isolate this transition from the already-accepted official Home/NVS fix.
- Restored only the conservative AXP2101 PMIC power-key parity hunk in the
  official-compatible product overlay: `REG10 |= 0x04`, `REG22 = 0b110`, and
  `REG27 = 0x00`.
- Updated overlay guard tests so official launcher/setup/Home, BLE, and
  no-direct-Xiaozhi boot remain protected while the exact PMIC parity hunk is
  required.
- Focused overlay tests passed, `git diff --check` passed, the overlay applied
  cleanly to the clean official StackChan export, and `GOMAXPROCS=2 make verify`
  passed.
- Guarded product build passed. App artifact:
  `a21-stackchan-official-xiaozhi-compatible.bin`; SHA-256
  `1f1348156a3312c76059f0a084d956d13ca41cc74a22ca2c0b8884b4ea1a9aa3`;
  build report:
  `reports/a21-stackchan-official-baseline-20260605-142444-1780640684585168000.json`.
- Next action: commit this build-ready transition, run the guarded
  official-compatible product flash lane, then physically test no-USB cold boot
  and shutdown.

Previous foreground hardware-window update, 2026-06-05 14:13 CST:

- Guarded official-compatible NVS execution passed at commit `01f38f21e763`
  using `A21_IDF_PYTHON` pointed at the verified ESP-IDF Python 3.14 venv.
- T7 control guard passed with a clean foreground hardware-window branch.
- esptool connected to ESP32-S3 MAC `44:1b:f6:e2:6a:60`, wrote NVS at
  `0x9000`, verified flash hash, and hard reset the device.
- Execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260605-141249-1780639969993980000.json`.
- Summary evidence: `write_executed=true`,
  `wifi_credentials_written=true`, `app_config_marked_configured=true`, and
  `servo_calibration_present=true`.
- Expected post-reset state is official StackChan Home, not immediate A21
  voice/body connection. Gateway should reconnect only after the operator opens
  official `AI.AGENT`.
- Next action: operator visually confirms Home, then opens `AI.AGENT`; control
  thread queries Gateway for fresh `/v1/xiaozhi` and `/stackChan/ws` activity.

Previous foreground hardware-window update, 2026-06-05 14:09 CST:

- Side-branch official Home unlock commit was integrated into the foreground
  hardware-window branch as `843c59f` and pushed to origin.
- Focused tests, `git diff --check`, and `GOMAXPROCS=2 make verify` passed
  before hardware execution.
- The first guarded NVS execution passed T7 control guard with a clean
  foreground worktree but failed before reading flash. Report:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260605-140250-1780639370066907000.json`;
  `write_executed=false`.
- Failure root cause is local ESP-IDF Python selection: `export.sh` chose a
  broken system Python 3.13 whose `_ssl` and `hashlib` imports are blocked by
  macOS code-signing policy. This is not a serial, Gateway, Wi-Fi, PMIC, or
  firmware-state failure.
- The foreground branch now supports `A21_IDF_PYTHON` / `--idf-python` for the
  official-compatible NVS executor, allowing it to use the verified
  `idf5.5_py3.14_env` Python directly for esptool and NVS scripts.
- Next action: commit/push the Python override, rerun `make verify`, execute
  guarded NVS with `A21_IDF_PYTHON`, and physically confirm official Home.

Latest side-branch update, 2026-06-05 13:54 CST:

- Created isolated branch `codex/a21-official-config-fallback-20260605` to
  solve the official front-end `Ready to Configure` trap without replacing
  official StackChan PMIC, Home, Setup, up-swipe, mobile surfaces, or
  `AI.AGENT` lifecycle.
- Root cause is now explicit: the official launcher gates Home on
  `GetHAL().isAppConfiged()`, which reads NVS `app_config/is_configed`.
  Because the stock official mobile App BLE association currently fails on the
  closed/placeholder `notifyState` type `4` encryption path, the App cannot set
  that NVS flag for the A21 product candidate.
- Side-branch candidate extends the existing guarded
  `a21-stackchan-official-xiaozhi-compatible-nvs` path to write
  `app_config/is_configed=1` when Wi-Fi credentials are present, while still
  preserving existing entries and only mutating A21/Xiaozhi connection gates.
- The execution summary now exposes `app_config_marked_configured`, and
  verification rejects a provisioned NVS image that has Wi-Fi credentials but
  lacks `app_config/is_configed=1`.
- Focused tests, `git diff --check`, and a direct rerun of the previously
  killed audio Silero help test passed on the side branch.
- T7 hardware-write guard means the actual NVS execution must happen only
  after this side-branch commit is integrated into a clean foreground
  `codex/a21-hardware-window-*` branch.

Latest mainline control update, 2026-06-05 13:31 CST:

- Foreground operator confirmed the official StackChan front-end is restored
  after the guarded product flash at commit `290a35073651`.
- The official mobile App now reports `Failed to process device data` during
  BLE association. Code review mapped this exact toast to
  `app/lib/view/popup/select_blue_device.dart`, where the App handles BLE
  `notifyState` type `4`, decrypts `data.state` through
  `RsaUtil.decryptStackChanBlue`, and requires the decrypted plaintext to be at
  least 12 characters before taking the first 12 as the device MAC.
- The firmware-side source path is `firmware/main/hal/hal_ble.cpp`
  `handle_handshake()`, which calls
  `secret_logic::generate_handshake_token(data)` and sends the result as
  `notify_state(4, token)`.
- The public official StackChan source currently leaves
  `secret_logic::generate_handshake_token()` as the placeholder
  `hi-stack-chan`, while the public App source leaves
  `ValueConstant.stackChanBluePrivateKey` empty. A21 therefore cannot make the
  stock official App accept BLE device data from the public source alone.
- The operator-provided physical device ID is `441BF6E26A60`, equivalent to the
  Gateway-normalized device ID `44:1b:f6:e2:6a:60`; this is the plaintext value
  the App needs after RSA-OAEP(SHA-256) decryption, but the matching official
  BLE public key or closed `secret_logic` implementation is still required to
  produce the encrypted `state` accepted by the stock official App.
- This is not a Gateway, public network, PMIC, or A21 voice-provider failure.
  The restored front-end can continue to be used for on-device mode entry and
  A21 Gateway routing; full stock-App BLE association remains pending on
  official secret material or an A21 companion-App/key-pair path.
- The accepted product state machine is now: official boot/PMIC/Home/Setup and
  Wi-Fi/NVS first; opening the official `AI.AGENT` app is the only transition
  into A21 runtime; after that transition, A21 Gateway owns `/v1/xiaozhi` voice
  and `/stackChan/ws` body/action links.
- Added focused guard coverage for this boundary in
  `internal/app/official_stackchan_test.go`, and both the focused overlay test
  and `GOMAXPROCS=2 make verify` passed.

Previous control update, 2026-06-05 13:26 CST:

- Commit `290a35073651 fix(stackchan): restore official power ui lifecycle`
  was product-flashed through the guarded official-compatible lane on
  `/dev/cu.usbmodem1101`.
- Flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-132511-1780637111677933000.json`.
- The flashed app artifact is
  `a21-stackchan-official-xiaozhi-compatible.bin` with SHA-256
  `2f50122b1a05879b393faf8490a6e3adec4e61f947417bdd4d53c16d7184e8a9`.
- The flash guard accepted only a clean worktree at commit `290a35073651` on
  branch `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
- Public Gateway checks through the correct Caddy entry
  `http://47.103.57.217/...` passed for `/healthz`, `/v1/devices`,
  `/v1/stackchan/official/status`, `/v1/power-lifecycle`, and
  `/xiaozhi/ota/`; direct `:21080`/`:21081` public probes returned 502 and
  are not the correct public operator entry from this host.
- `/xiaozhi/ota/` returns `ws://47.103.57.217/v1/xiaozhi`; the official body
  relay remains visible in Gateway status when the device is connected.
- Physical no-USB power-key cold boot is still not accepted until the operator
  unplugs power, lets the unit turn fully off, presses the upper-left power key,
  and confirms it reaches the official StackChan front-end instead of only
  flashing the screen/red LED.

Previous control update, 2026-06-05 13:18 CST:

- Added
  `docs/plans/2026-06-05-stackchan-official-power-ui-parity-recovery.md`
  after the operator reported that the no-USB upper-left power button still
  only flashed the screen/red LED and stopped.
- Root-cause review split the boot path into physical PWRKEY/AXP2101 hold,
  ESP/HAL init, official launcher/setup/app selection, and Xiaozhi/Gateway
  voice. The observed failure is before Gateway/provider/voice can participate.
- Restored official PMIC behavior by removing the A21 product overlay hunk that
  wrote AXP2101 `0x10` and `0x22`; the built source now leaves the official
  StackChan `WriteReg(0x27, 0x00)` lifecycle untouched.
- Restored the official launcher/setup/mobile-association front-end as the
  default by removing the `A21_OEM_AUTOSTART_XIAOZHI` define and the product
  overlay `main.cpp` direct `GetHAL().startXiaozhi()` parking path.
- Kept A21 as the selected backend/mode service by making
  `secret_logic::get_server_url()` return
  `CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL`; the official
  `AppAvatar` relay still appends an A21 `device_id` query parameter.
- Focused overlay tests passed, `GOMAXPROCS=2 make verify` passed, and the
  guarded product build passed with app SHA-256
  `22fde627ca518dedfa261fafd94a5ddf4bacbd0bbde91b9be8a25a788de5d5fc`.
- Build report:
  `reports/a21-stackchan-official-baseline-20260605-131727-1780636647604865000.json`.
- Physical no-USB power-key validation remains pending until this build is
  committed, guarded-flashed, and tested on the product device.

Previous control update, 2026-06-05 13:01 CST:

- Commit `2356745a8586 fix(stackchan): restore native motion power diagnostics`
  was deployed to ECS `47.103.57.217` through `/opt/a21.next` safe swap.
- Remote focused Gateway/App tests passed, remote Go build passed,
  `a21-gateway` restarted active, and public `/healthz` returned OK.
- Guarded product flash passed on `/dev/cu.usbmodem1101` through the
  official-compatible product lane. App artifact
  `a21-stackchan-official-xiaozhi-compatible.bin` SHA-256:
  `357211684819d915106f8b72acdfe3bbf7d4b2f63d4547970599d05efdcd9c51`.
- Public Gateway post-flash evidence shows the product device online, official
  relay connected through `stackchan_official_ws`, and device runtime echo now
  carrying battery/power diagnostics:
  `battery_level`, `battery_charging`, `battery_discharging`,
  `external_power`, `power_source`, and
  `pmic_power_key_profile=a21_stackchan_axp2101_pwrkey_v1`.
- Post-flash official motion control trace
  `a21-trace-post-native-motion-flash` returned `status=delivered`,
  `delivered_transport=stackchan_official_ws`, and
  `motion=official_dance_sequence`.
- `/v1/power-lifecycle` now marks battery telemetry ready through
  `diagnostic_runtime_echo`; no-cable cold boot, physical power button, and
  PMIC profile operator acceptance remain explicitly missing rather than
  inferred.
- Physical foreground acceptance is still pending for no-USB cold boot/shutdown,
  wake sensitivity, no self-loop, reply latency, and visible body/touch
  amplitude. No generic `xiaozhi.bin` flash, NVS write, provider-key firmware
  change, Git prune/gc, or internal-test3 voice/protocol rollback occurred.

Previous control update, 2026-06-05 12:54 CST:

- Added
  `docs/plans/2026-06-05-stackchan-native-motion-power-diagnostics.md`.
- Increased official StackChan body motion amplitude in Gateway official
  avatar state fanout, body presets, body motions, and touch-reaction MCP
  fallback while preserving existing A21 product voice protocol behavior.
- Increased firmware-local screen/top touch StackChan RGB, vibration sound, and
  servo motion feedback amplitudes and removed the `isMoving()` suppression
  that made repeated touches feel dropped.
- Added Xiaozhi device heartbeat battery/power-source runtime echo parsing and
  Gateway recording so no-USB power-key failure can be separated from
  battery/supply path versus PMIC state-machine behavior.
- Fixed the official-compatible overlay `protocol.cc` include hunk so it
  applies reliably with `git apply --recount`.
- Verification passed: focused app/Gateway/transport tests, `git diff --check`,
  clean source overlay apply check, and `GOMAXPROCS=2 make verify`.
- Product build passed through
  `a21-stackchan-official-xiaozhi-compatible-build`; app artifact SHA-256 is
  `357211684819d915106f8b72acdfe3bbf7d4b2f63d4547970599d05efdcd9c51`.
- First guarded product flash attempt on `/dev/cu.usbmodem1101` was correctly
  blocked because the implementation worktree was dirty. Next action is commit,
  deploy Gateway to ECS, then guarded product flash through the same product
  lane.
- No generic `xiaozhi.bin` flash, NVS write, provider-key firmware change, Git
  prune/gc, or internal-test3 voice/protocol rollback occurred.

Previous control update, 2026-06-05 11:12 CST:

- Responded to the foreground P0 regression report covering voice self-loop,
  delayed replies, no-USB power-key behavior, and missing touch/RGB/body
  feedback.
- Commit `843e4cc fix(stackchan): stabilize voice state and touch feedback`
  restored Xiaozhi official speaking-state wake semantics so custom wake word
  detection is not forced on during TTS playback, enabled delayed fast ack by
  default for the cloud-edge product chain, and added firmware-local
  touch feedback through StackChan RGB, servo motion, and the vibration sound.
- Product firmware build passed through the official-compatible lane and was
  guarded-flashed on `/dev/cu.usbmodem1101` with app SHA-256
  `84698ea172a1070adca45d29b005e87dca74e4a39cd8f22ba7558797dc9d261f`.
- ECS `47.103.57.217` was deployed from commit `843e4cc` through the
  `/opt/a21.next` safe-swap path. Remote focused tests and build passed,
  `a21-gateway` restarted active, and runtime env explicitly has
  `A21_XIAOZHI_FAST_ACK_ENABLED=true` and
  `A21_XIAOZHI_FAST_ACK_DELAY_MS=700`.
- Commit `8fdc95d fix(stackchan): align PMIC power key timing` then aligned
  StackChan PMIC power-key timing back to official `REG27=0x00`, preserved the
  explicit `REG22` PWRON/OFFLEVEL power-off source handling, and enabled the
  AXP2101 16s PWRON hardware shutdown fallback.
- The second product firmware build passed and was guarded-flashed on
  `/dev/cu.usbmodem1101` with app SHA-256
  `568da12f52bc050b3512c3ac71d5f3afd02f1fa6bd9debd89afe208efc1fe53b`.
- After the second flash, public Gateway heartbeat updated, official relay
  reconnected through `stackchan_official_ws`, and a post-flash
  `/v1/stackchan/official/control` motion command returned `status=delivered`
  with trace `a21-trace-post-pmic-flash-official-motion`.
- Physical acceptance remains pending and must be performed in the foreground:
  wake sensitivity/no self-loop, first audible reply timing, screen/top-touch
  local RGB/servo/sound feedback, and no-USB power key cold boot/shutdown.
- The attempted A21 idle websocket exception for `CanEnterSleepMode()` was not
  shipped because it created an unstable overlay hunk and failed product
  compile; the current flashed power fix is PMIC-only plus official timing.
- No generic `xiaozhi.bin` product flash, NVS write, provider-key firmware
  change, Git prune/gc, or internal-test3 protocol rollback occurred.

Previous control update, 2026-06-05 10:20 CST:

- Fresh serial capture after firmware recovery showed repeated official avatar
  relay heartbeat pings at roughly 400 seconds uptime and no `sys_evt` stack
  overflow, reboot, Guru, or abort.
- Public Gateway status exposed a device-id casefold split rather than a fresh
  hardware failure: lowercase MAC was online as the Xiaozhi product socket,
  while uppercase MAC held the official relay connection.
- Added
  `docs/plans/2026-06-05-stackchan-device-id-casefold-recovery.md`.
- Gateway now normalizes MAC-shaped device IDs for official socket/status,
  official control, Xiaozhi-to-official fanout, and official relay registry
  writes.
- Product recovery now prefers online/latest records when duplicate
  case-variant MAC records are present.
- Focused Gateway/App casefold tests, broader App/Gateway
  `OfficialStackChan|ProductRecovery|Xiaozhi|PowerLifecycle` subset, and
  `GOMAXPROCS=2 make verify` passed.
- Commit `7b32956 fix(gateway): normalize stackchan hardware mac ids` was
  deployed directly over SSH to ECS `47.103.57.217` through `/opt/a21.next`
  safe swap. Remote archive SHA verification, focused Gateway/App tests,
  remote build, `a21-gateway` restart, and loopback `/healthz` passed.
- Post-deploy polling showed one normalized lowercase MAC device record,
  product socket online, lowercase official relay status connected through
  `stackchan_official_ws`, and public recovery precheck
  `reports/a21-stackchan-product-recovery-20260605-101947.json` returned
  `product_online_official_relay_ready` with `rom_download_required=false`.
- No firmware flash, NVS write, provider/V21 execution, Git prune/gc, or
  physical no-USB power-key acceptance occurred in this transition.

Previous control update, 2026-06-05 09:54 CST:

- The product StackChan was confirmed not hard-bricked: USB/JTAG serial
  enumerated as `/dev/cu.usbmodem1101`, serial `44:1B:F6:E2:6A:60`.
- Added
  `docs/plans/2026-06-05-stackchan-sys-evt-boot-loop-recovery.md`.
- Pre-fix serial capture showed the white flashing screen was an app boot loop
  caused by `***ERROR*** A stack overflow in task sys_evt has been detected.`
  after Wi-Fi scan/connect.
- Firmware overlay now prevents the A21 official avatar relay task from
  starting a second network path and sets
  `CONFIG_ESP_SYSTEM_EVENT_TASK_STACK_SIZE=8192`.
- Firmware overlay also percent-encodes MAC-address colons in the official
  avatar relay `device_id` query parameter, fixing the official WebSocket URL
  parser failure that previously produced `Failed to get host by name`.
- Official-compatible product app rebuilt and guarded-flashed through
  `a21-stackchan-official-xiaozhi-compatible-flash-execute`; current flashed
  app SHA-256 is
  `c68e9557f065eb40a184ea88d8f111fbc4aa87ca6729c38899cc8555edc7acaf`.
- Flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-095236-1780624356871525000.json`.
- Post-flash 45 second serial capture showed no `sys_evt` stack overflow, no
  reset/Guru/abort loop, Xiaozhi session
  `a21-session-44-1b-f6-e2-6a-60`, quiet control websocket active, and
  `WS-Avatar: Connected to server!` with heartbeat pings.
- Local focused overlay/app tests, `git diff --check`, and product app build
  passed.
- No generic `xiaozhi.bin` product flash, NVS write, provider/V21 execution,
  Git prune/gc, or physical no-USB power-key acceptance occurred.

Previous control update, 2026-06-05 08:58 CST:

- Re-read review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` and compared
  the power/lifecycle blocker against the current Gateway, protocol, product
  overlay, and product recovery state.
- Added
  `docs/plans/2026-06-05-power-lifecycle-cold-boot-evidence-guard.md`.
- Tightened `POST /v1/power-lifecycle-acceptance` to require explicit
  foreground no-USB cold-boot evidence:
  `boot_source=battery_power_key_cold_boot`,
  `usb_connected_during_boot=false`, `power_key_hold_ms` in `250..12000`,
  `pmic_power_key_profile=a21_stackchan_axp2101_pwrkey_v1`, and
  `boot_observed_at_ms>0`.
- Kept online product registry plus active Xiaozhi WebSocket as a required
  prerequisite, so an already-online socket cannot be mistaken for a real
  physical power-key cold boot.
- `GET /v1/power-lifecycle` now exposes `pmic_power_key_profile` separately
  from generic runtime online status, and accepted evidence records redacted
  boot-source and PMIC-profile metadata in device capabilities.
- Review-related verification passed:
  `git diff --check`,
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestPowerLifecycle|TestHardwareAcceptance' -count=1`,
  `GOMAXPROCS=2 go test -race ./internal/gateway -run 'PowerLifecycle|OfficialStackChan|Xiaozhi|WorkspaceConsolePageServed' -count=1`,
  `GOMAXPROCS=2 make verify`, `GOMAXPROCS=2 make preflight`, and
  `GOMAXPROCS=2 make doctor`.
- Commit `a181d44 feat(gateway): harden power lifecycle acceptance evidence`
  was pushed to the current branch.
- ECS deployment from this Mac is pending because Aliyun ECS API TLS closes
  during handshake through the current TUN fake-IP route, SSH to
  `47.103.57.217:22` closes before the SSH banner, and public HTTP ports accept
  TCP but return empty application replies from this host.
- Local deployment archive is ready at `/tmp/a21-a181d4464af9.tar.gz` with
  SHA-256
  `0d1fd7b40e09d43a10338515189c6c6c7cfc4841729e06cc1c14dde25fb47666`; GitHub
  codeload for commit `a181d4464af90f55f5488f9e78b8ffda61dac16c` returns
  HTTP 200.
- No firmware flash, NVS write, provider/V21 execution, or physical acceptance
  occurred in this transition.

Previous control update, 2026-06-05 08:47 CST:

- Added
  `docs/plans/2026-06-05-stackchan-product-recovery-executor.md`.
- Extended `a21 stackchan-product-recovery` with explicit
  `--execute-flash` mode. The default command remains read-only.
- Execution mode requires confirmation token
  `WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP`, skips flash when the
  product is already online, calls only the existing
  `a21-stackchan-official-xiaozhi-compatible` wait-ROM product flash path with
  `esptool_before=no_reset`, writes
  `a21.stackchan_product_recovery_execution.v1`, and runs a post-flash Gateway
  plus official relay check.
- Added Make targets `stackchan-product-recovery` and
  `stackchan-product-recovery-execute` to keep the hardware recovery window on
  the correct product lane.
- Focused app tests for product recovery execution and official-compatible
  wait-ROM flash passed.
- Read-only recovery through the new Make target passed with
  `A21_DIRECT_SOURCE_IP=192.168.1.27` and wrote
  `reports/a21-stackchan-product-recovery-20260605-084627.json`; live product
  truth is still `product_offline_rom_download_required`, `devices=[]`, and
  official relay `connected=false`.
- `git diff --check`, `make verify`, `make preflight`, and `make doctor`
  passed.
- ECS deployment is not required for this transition because the public Gateway
  runtime did not change.
- No firmware flash was executed in this transition.

Previous control update, 2026-06-05 08:29 CST:

- Re-read review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` and applied
  the remaining software-closeable voice experience findings to product
  defaults.
- Added
  `docs/plans/2026-06-05-xiaozhi-product-stt-screen-and-fast-ack-guard.md`.
- Added `A21_XIAOZHI_STT_SCREEN_POLICY` with `raw`, `status_only`, and `off`.
  Gateway fixture/lab default remains `raw`; cloud-edge product-chain default
  is `status_only`.
- `status_only` preserves raw ASR transcript for the internal answer pipeline
  but sends only a non-sensitive device STT status phrase and policy trace
  marker, avoiding raw transcript screen echo in product roleplay.
- Cloud-edge product-chain default now sets
  `A21_XIAOZHI_FAST_ACK_ENABLED=false`, while explicit lab overrides can still
  opt in to fast ack and tune `A21_XIAOZHI_FAST_ACK_DELAY_MS`.
- Focused app and Gateway tests for cloud-edge defaults, stock raw STT order,
  status-only STT display redaction, and fast-ack disabled behavior passed.
- Review-related `git diff --check`, Gateway `-race` subset, `make verify`,
  `make preflight`, and `make doctor` passed.
- Commit `79381d4 feat(gateway): guard product stt display and fast ack` was
  pushed and deployed to ECS `47.103.57.217` through Aliyun Cloud Assistant
  over the 5080lab SOCKS path. Remote archive SHA verification, focused
  app/Gateway tests, build, `/opt/a21.next` safe-swap, `a21-gateway` restart,
  loopback health, Caddy health, official status, and public SOCKS smoke
  passed.
- Public hardware truth remains `devices=[]`, official relay
  `connected=false`, and `next_action=connect_official_stackchan_ws`.
- Post-deploy read-only recovery precheck wrote
  `reports/a21-stackchan-product-recovery-20260605-083905.json` with
  `status=product_offline_rom_download_required`, USB candidate
  `/dev/cu.usbmodem1101`, and latest flash-log evidence
  `No serial data received`.
- This transition does not claim product physical power-key, wake, latency,
  official `/stackChan/ws`, or body-action acceptance.

Previous control update, 2026-06-05 08:15 CST:

- Re-read and applied the review-thread conclusion that official StackChan body
  parity should not split relay state and fallback execution outside Gateway.
- Added
  `docs/plans/2026-06-05-official-stackchan-backend-mcp-fallback.md`.
- `POST /v1/stackchan/official/control` is still strict by default and still
  returns HTTP 409 without a connected official `/stackChan/ws` relay.
- Requests with explicit `allow_mcp_fallback=true` now map safe official
  state/face/motion events to bounded Xiaozhi MCP body-preset/body-motion
  sequences and return `status=fallback_delivered`,
  `delivered_transport=xiaozhi_mcp_sequence`,
  `official_action_fallback_reason=official_stackchan_ws_disconnected`, and
  `official_action_physical_accepted=false`.
- Fallback writes `stackchan.official_mcp_fallback.*` trace markers and
  `/v1/devices.runtime_echo` metadata such as
  `official_stackchan_fallback_status=delivered`,
  `official_stackchan_official_relay=disconnected`, and
  `official_stackchan_packets=0`.
- `/workspace` Official Actions now sends `allow_mcp_fallback=true`; the older
  client-side catch-and-fallback path remains only as compatibility.
- Focused Gateway official-control/status/workspace tests, the review-related
  Gateway `-race` subset, `make verify`, `make preflight`, and `make doctor`
  passed locally.
- Commit `c5fb24b feat(gateway): add official stackchan mcp fallback` is pushed
  and deployed to ECS `47.103.57.217` through Aliyun Cloud Assistant over the
  existing 5080lab SOCKS path. Remote archive SHA verification, focused Gateway
  tests, build, `/opt/a21.next` safe-swap, `a21-gateway` restart, loopback
  health, Caddy health, official relay status, and `/workspace` smoke passed.
- Public `/workspace` now exposes `allow_mcp_fallback`; public official status
  remains `connected=false`, `physical_accepted=false`, and
  `next_action=connect_official_stackchan_ws`.
- Public fallback control currently returns HTTP 409
  `xiaozhi websocket is not connected` because the product device is offline;
  this is the expected honest failure mode until product Xiaozhi reconnects.
- This is a backend state-machine/control-surface remediation only. It does
  not mark product physical acceptance, flash firmware, write NVS, execute
  providers/V21, or resolve the current product-offline ROM/download blocker.

Previous control update, 2026-06-05 07:51 CST:

- Re-read review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` and compared
  its findings against the current implementation. Gateway review race subset,
  `make preflight`, `make doctor`, and `make verify` now pass in the current
  checkout.
- Added read-only product recovery CLI
  `a21 stackchan-accept --check product-recovery` plus alias
  `a21 stackchan-product-recovery`.
- The recovery precheck reads Gateway `/v1/devices`, official relay
  `/v1/stackchan/official/status`, local USB serial inventory, and latest
  official-compatible product flash receipts. It does not send control frames,
  flash firmware, write NVS, contact provider paths, or execute V21.
- Live local recovery run wrote
  `reports/a21-stackchan-product-recovery-20260605-074948.json` and classified
  the current state as `product_offline_rom_download_required`.
- The report records `/dev/cu.usbmodem1101` present and latest guarded product
  flash receipt
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-074215-1780616535711678000.json`
  with `flash_executed=false` and `wait_rom_download_mode=true`.
- Public direct Gateway access from this host was still unstable during the
  precheck (`EOF` via the CLI direct client, `502` via shell curl), so Gateway
  reachability remains a finding rather than accepted product-online evidence.
- Commit `5ee89f4 feat(app): add stackchan product recovery precheck` was
  pushed and deployed to ECS using Aliyun Cloud Assistant through the 5080lab
  SOCKS path. Remote `/opt/a21.next` app tests and build passed, then the host
  was safe-swapped to `/opt/a21` and `a21-gateway` restarted active.
- Remote post-deploy smoke passed: `/opt/a21/bin/a21
  stackchan-product-recovery --help`, loopback `/healthz` on
  `127.0.0.1:21081`, Caddy `/healthz` on port 80, loopback official status,
  and loopback `/v1/devices`. Product state remains
  `official.connected=false` and `devices=[]`.
- 2026-06-05 08:03 CST enhancement: product recovery precheck now supports
  explicit `--direct-source-ip` and parses the latest guarded flash log for
  ROM evidence. A live run with `--direct-source-ip 192.168.1.27` wrote
  `reports/a21-stackchan-product-recovery-20260605-080244.json`; Gateway
  status checks succeeded, `official.connected=false`, `devices=[]`, and the
  flash log evidence records `rom_probe_timed_out=true`,
  `rom_no_serial_data=true`, and
  `Failed to connect to ESP32-S3: No serial data received`.
- Commit `9d6c909 feat(app): enrich stackchan recovery diagnostics` was
  deployed to ECS through Cloud Assistant and the `/opt/a21.next` safe-swap
  path. Remote app tests, build, service restart, loopback/Caddy health
  checks, official status, and `/v1/devices` smoke passed. Product state is
  still `official.connected=false` and `devices=[]`.
- Next transition remains physical: enter ESP32-S3 ROM/download mode, execute
  the guarded product flash if needed, then verify product Xiaozhi,
  official `/stackChan/ws`, power-key startup, wake/listen/playback,
  barge-in, and visible body behavior.

Previous control update, 2026-06-05 07:42 CST:

- Public Gateway deployment is live for the official relay status surface, but
  the product is not currently online: 5080lab public `/v1/devices` returned
  `devices=[]`.
- A short guarded wait-ROM product flash execute was retried on
  `/dev/cu.usbmodem1101` with `--esptool-before no_reset`,
  `--wait-rom`, and a 30 second wait window.
- The T7 flash guard passed on clean HEAD `f6be66d`, but the command timed
  out before writing flash: `flash_executed=false`.
- Report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-074215-1780616535711678000.json`.
- Log:
  `/tmp/a21-stackchan-official-build/a21-official-xiaozhi-compatible-flash-20260605-074141.log`.
- Root evidence remains
  `Failed to connect to ESP32-S3: No serial data received`; the product is not
  in ESP32-S3 ROM/download mode.
- Next physical action is manual ROM/download entry before retrying the
  guarded product flash lane.

Previous control update, 2026-06-05 07:30 CST:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read and
  compared against current HEAD. Previously reported software blockers for
  Gateway race, namespace gates, stock professional mode routing,
  power-lifecycle state, and physical wake launch-gate semantics remain
  closed in the current implementation.
- Added plan
  `docs/plans/2026-06-05-official-stackchan-relay-status-surface.md`.
- Gateway now exposes read-only
  `GET /v1/stackchan/official/status?device_id=...` with schema
  `a21.stackchan.official.status.v1`.
- The status surface reports exact/default official `/stackChan/ws` socket
  connection, connected timing, last trace/session/event, last packet count,
  semantic official action surfaces, physical acceptance, fallback
  availability, and next action.
- `/workspace` now shows relay connection and next action separately from
  official action delivery and MCP fallback, and exports `official_relay_*`
  metadata.
- Verification passed: focused official relay/workspace tests,
  `git diff --check`, Gateway review race subset,
  `GOMAXPROCS=2 make verify`, `GOMAXPROCS=2 make preflight`, and
  `GOMAXPROCS=2 make doctor`.
- Commit `a139987 feat(gateway): expose official stackchan relay status` was
  pushed and deployed to ECS `47.103.57.217` using Aliyun Cloud Assistant via
  the 5080lab SOCKS path, chunked `SendFile` transfer, remote SHA-256
  reassembly verification, and the existing `/opt/a21.next` safe-swap
  pattern.
- Remote focused Gateway tests and remote build passed before swap;
  `a21-gateway` restarted active; loopback `/healthz` and loopback
  `/v1/stackchan/official/status?device_id=44:1b:f6:e2:6a:60` passed.
- 5080lab public smoke confirmed `/healthz`, the official status endpoint,
  and `/workspace` with `Relay status` are live.
- No firmware build, flash, NVS write, provider/V21 execution, or physical
  acceptance promotion occurred in this transition.
- Next transition remains physical recovery and acceptance: enter ESP32-S3
  ROM/download mode, execute the guarded product flash if needed, reconnect
  product Xiaozhi and official `/stackChan/ws`, then collect power-key, wake,
  playback, barge-in, and visible body evidence.

Previous control update, 2026-06-05 07:18 CST:

- Guarded physical product flash was attempted on commit `1e8022a` through
  both wait-ROM/no-reset and default-reset official product flash lanes.
- Both attempts failed before any flash write. The reports recorded
  `flash_executed=false`; no generic product lane and no `xiaozhi.bin` were
  used.
- The common failure evidence is that `/dev/cu.usbmodem1101` exists and the
  product USB serial `44:1B:F6:E2:6A:60` enumerates, but esptool cannot
  synchronize with ESP32-S3 ROM/bootloader and reports `No serial data
  received`.
- A read-only OpenOCD USB-JTAG identify/reset attempt also did not acquire a
  target before descriptor/JTAG setup failed.
- The wait-ROM product flash script was enhanced to scan all
  `/dev/cu.usbmodem*` candidates, switch to whichever port answers ROM
  `chip_id`, print periodic candidate status, and print the last esptool
  probe output on timeout.
- Verification passed: focused official flash wait-ROM tests,
  `git diff --check`, and `GOMAXPROCS=2 make verify`.
- Next transition remains physical: enter real ESP32-S3 ROM/download mode,
  rerun the enhanced guarded product flash, then collect reconnect, official
  relay/body, wake, voice, playback, and barge-in evidence.

Previous control update, 2026-06-05 07:07 CST:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read and
  compared to current implementation again. The remaining product blocker is
  physical recovery/acceptance, not a missing generic flash lane.
- The official Xiaozhi-compatible product flash lane now supports explicit
  wait-for-ROM mode through CLI flags
  `--wait-rom --wait-rom-timeout-seconds N` and Makefile env
  `A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WAIT_ROM=true`.
- Wait-for-ROM is constrained to `--esptool-before no_reset`, probes only
  ESP32-S3 ROM `chip_id` with no reset/no stub until timeout, and then runs
  the existing guarded product app flash against
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- A no-write product flash plan for `/dev/cu.usbmodem1101` produced
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-070727-1780614447957789000.json`
  with `status=ready`, `dry_run=true`, `wait_rom_download_mode=true`, and
  `wait_rom_timeout_seconds=90`.
- No firmware flash was executed and no physical acceptance was promoted in
  this transition.
- Verification passed: focused official flash wait-ROM tests,
  `git diff --check`, `GOMAXPROCS=2 make verify`,
  `GOMAXPROCS=2 make preflight`, and `GOMAXPROCS=2 make doctor`.
- Next transition remains physical: put the product StackChan into
  ESP32-S3 ROM/download mode, execute the guarded product flash with wait-ROM
  enabled, then collect reconnect, official relay/body, physical wake, and
  PRD physical voice evidence.

Previous control update, 2026-06-05 06:58 CST:

- Wake-word launch readiness was tightened to match the review-thread
  evidence distinction: server-side wake-word status does not equal physical
  wake acceptance.
- `launch_ready` now requires `productWakeWordPhysicalAccepted`, using the
  existing wake physical acceptance contract.
- Canonical missing evidence now includes `wake_word_physical_acceptance` when
  no accepted physical wake proof is attached.
- The real-launch readiness fixture now includes guarded custom wake package
  evidence and physical wake acceptance evidence.
- Public product readiness was rerun with source-bound direct connect and
  returned `reports/a21-product-readiness-20260605-065743.json`,
  `status=server_side_candidate_ready`.
- Canonical missing real evidence is now exactly `physical_stackchan_online`,
  `physical_stackchan_prd_acceptance`, and
  `wake_word_physical_acceptance`.
- Verification passed: focused wake/launch tests and `GOMAXPROCS=2 make
  verify`.
- Next transition remains physical: guarded delayed-relay product flash,
  reconnect, official relay/body acceptance, physical wake acceptance, and PRD
  physical voice acceptance.

Previous control update, 2026-06-05 06:52 CST:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read and
  compared to current implementation. Software findings for Gateway race,
  namespace/preflight, stock professional route, and PMIC power-key parity
  remain remediated; the active launch gap remains physical StackChan
  recovery and PRD acceptance.
- `product-readiness` now aligns provider-smoke matching with the remote
  Gateway voice-chain selected LLM profile when local provider selection is
  unset/mock and a provider smoke report is supplied or selected through
  `--use-latest-reports`.
- Explicit local real providers remain authoritative, and no-smoke mock demos
  continue to report mock instead of silently promoting the Gateway selector.
- Safe external Gateway professional bench/read-record evidence no longer
  leaves the misleading local `A21_V21_ADAPTER_URL` next action, while
  `launch_ready` remains false until physical evidence is accepted.
- Public product readiness was rerun without local
  `A21_PROVIDER_PRIMARY=stepfun` and returned
  `reports/a21-product-readiness-20260605-064932.json`,
  `status=server_side_candidate_ready`.
- Canonical missing real evidence before the later wake physical launch gate
  was `physical_stackchan_online` and `physical_stackchan_prd_acceptance`.
- Verification passed: focused app readiness tests, `GOMAXPROCS=2 make
  verify`, `GOMAXPROCS=2 make preflight`, `GOMAXPROCS=2 make doctor`, and
  `GOMAXPROCS=2 go test -race ./internal/gateway -run
  'Xiaozhi|PowerLifecycle|OfficialStackChan|StockProfessionalRoute' -count=1`.
- Next transition remains physical: enter ESP32-S3 ROM download mode, execute
  the guarded product app flash for the delayed-relay artifact, then collect
  serial/no-WDT, Xiaozhi reconnect, official relay, screen/RGB/head/touch,
  wake/listen/playback/barge-in, and PRD physical acceptance evidence.

Previous control update, 2026-06-05 06:43 CST:

- A short esptool `no_reset` probe still failed with
  `Failed to connect to ESP32-S3: No serial data received`; the device is not
  yet in ROM download mode.
- `GOMAXPROCS=2 make preflight` and `GOMAXPROCS=2 make doctor` both passed on
  current HEAD.
- The current Wi-Fi source address is `192.168.1.27`. Default routing to
  `47.103.57.217` goes through TUN `utun6` / `198.18.0.1`, so default curls
  may return false `Empty reply from server`. Source-bound public checks using
  `curl --interface 192.168.1.27 --noproxy '*'` passed for `/healthz`,
  `/v1/devices`, and `/xiaozhi/ota/`.
- Re-running readiness with `A21_PROVIDER_PRIMARY=stepfun`,
  `A21_DIRECT_SOURCE_IP=192.168.1.27`, explicit provider smoke report
  `reports/a21-provider-smoke-20260605-054222-582464591.json`, and
  `--use-latest-reports` restored server-side candidate status:
  `reports/a21-product-readiness-20260605-064249.json` and
  `reports/a21-server-side-readiness-bundle-20260605-064249.json`.
- Focused Gateway review regression passed under race detector:
  `GOMAXPROCS=2 go test -race ./internal/gateway -run 'Xiaozhi|PowerLifecycle|OfficialStackChan|StockProfessionalRoute' -count=1`.
- Remaining canonical evidence is physical: bring product StackChan back
  online by flashing the safe delayed-relay artifact, then record PRD physical
  acceptance.

Previous control update, 2026-06-05 06:38 CST:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read and
  compared against current HEAD. Prior review findings for Gateway race,
  namespace/preflight, stock professional route, and PMIC power-key parity
  remain software-closed; the still-unaccepted product issue is the physical
  official StackChan relay/power recovery path.
- Commit `409ff0b fix(firmware): allow guarded manual bootloader flash` is
  pushed. The official Xiaozhi-compatible product flash lane now accepts
  `--esptool-before default_reset|usb_reset|no_reset` and Makefile env
  `A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_ESPTOOL_BEFORE`, while retaining
  the A21 T7 control guard, clean-worktree guard, exact product artifact, and
  confirmation token.
- Focused App flash/overlay tests, `git diff --check`, and
  `GOMAXPROCS=2 make verify` passed.
- Product rebuild passed:
  `reports/a21-stackchan-official-baseline-20260605-063356-1780612436653826000.json`.
  Product app SHA:
  `6c2ba13982efc7570ad0ac9ec0329232af6bedc58cd5ad9b18cc06b7f8f5b8b9`.
- Guarded `no_reset` product flash execute failed because the chip was not in
  ROM bootloader/download mode:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-063415-1780612455688329000.json`.
  USB enumeration confirmed the connected Espressif USB JTAG/serial unit is
  product device `44:1B:F6:E2:6A:60`.
- `no_reset_no_sync` direct probe and OpenOCD USB-JTAG read-only probe did not
  produce a usable recovery path. No JTAG flash was attempted.
- Next transition requires operator physical ROM download entry: hold
  `BOOT`/download, press/release `RESET`, keep holding `BOOT` until esptool
  reports `Chip is ESP32-S3`, then run the guarded `no_reset` product flash.

Previous control update, 2026-06-05 06:20 CST:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was compared against
  current HEAD after the stock professional route and PMIC remediation. The
  remaining official StackChan body gap was traced to the product firmware
  overlay: WDT-safe direct `startXiaozhi()` parked before the Mooncake worker
  that normally ticks official `WebSocketAvatar`.
- The product overlay now keeps direct Xiaozhi start, but schedules a delayed
  A21 direct official StackChan avatar relay task before entering the blocking
  `GetHAL().startXiaozhi()` call. The task waits 12 seconds, starts the
  official `WebSocketAvatar` without system-event logging, and ticks
  `updateA21WebSocketAvatarRuntime()` every 20 ms.
- The official avatar socket URL now uses
  `CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL="ws://47.103.57.217"` and
  appends `device_id` from `GetHAL().getFactoryMacString(":")`, so Gateway
  controls addressed to product MAC `44:1b:f6:e2:6a:60` can match the
  official `/stackChan/ws` socket instead of the fallback `stackchan-official`
  registry key.
- The first product flash of commit `02955a6c9c28` preserved Xiaozhi online
  behavior but left official control at HTTP 409. Serial boot logs proved the
  relay-start code did not run because `GetHAL().startXiaozhi()` is blocking.
- The immediate-before-Xiaozhi ordering artifact was flashed next and rejected:
  serial boot logs showed the relay runtime starting Wi-Fi first and then
  `***ERROR*** A stack overflow in task sys_evt`.
- Focused overlay tests, official firmware/App tests, Gateway official/power
  tests, `git diff --check`, and a guarded product firmware rebuild passed
  after moving the relay into a delayed background task.
- Product rebuild report:
  `reports/a21-stackchan-official-baseline-20260605-062018-1780611618644528000.json`.
  App artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`,
  SHA `a1b0262ebf659a268c9c9e578e8b34b1584fed0ec5c20a9c8fc84f5cde7ab92b`.
- This is build-ready, not product accepted. The next state requires guarded
  product flash of the delayed-task artifact, device reconnect, `/stackChan/ws`
  online evidence, a
  successful `/v1/stackchan/official/control` delivery to the product MAC, and
  physical/operator confirmation of visible head/RGB/screen behavior.

Previous control update, 2026-06-05 05:45 CST:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read and
  compared against deployed ECS behavior after the PMIC flash. Fresh
  post-restart evidence showed a real mode-state regression:
  `A21_XIAOZHI_STOCK_PROFESSIONAL_ROUTE=true` routed stock `realtime` listens
  to professional execution even while selected voice mode was roleplay.
- Commit `15a16dc fix(gateway): gate stock professional route by voice mode`
  is deployed on ECS. Stock professional override now requires selected voice
  mode `professional`; default roleplay stock turns stay in normal voice
  pipeline. Explicit professional listens and voice-triggered professional
  routes remain covered.
- Local verification passed: focused Gateway stock-professional route tests,
  focused App bench/env tests, `git diff --check`, and
  `GOMAXPROCS=2 make verify`.
- Remote `/opt/a21.next` focused Gateway/App tests and build passed; ECS
  `a21-gateway.service` restarted active and `/healthz` passed.
- Fresh reports after deployment:
  `reports/a21-provider-smoke-20260605-054222-582464591.json`,
  `reports/a21-roleplay-voice-probe-20260605-054224.json`,
  `reports/a21-xiaozhi-voice-bench-20260605-054238.059793905.json`,
  `reports/a21-xiaozhi-professional-bench-20260605-054250.259403329.json`,
  `reports/a21-product-readiness-20260605-054250.json`, and
  `reports/a21-server-side-readiness-bundle-20260605-054250.json`.
- Fresh voice bench passed `candidate_host_only` with cloud-edge product-chain
  execution, `failure_count=0`, answer first-audio P95 `1281 ms`, and
  barge-in stop P95 `0 ms`.
- Fresh professional bench passed `external_gateway_ready` with checking
  feedback `204 ms`, completed read record, and TTS stop observed.
- Fresh product readiness and server-side readiness are
  `server_side_candidate_ready`. The canonical missing real evidence is now
  only `physical_stackchan_prd_acceptance`.
- Product mode ritual and `full_check` were replayed after deploy and both
  returned `status=delivered`; public hardware acceptance remains
  `physical_pending`.
- Official `/stackChan/ws` remains disconnected: a correct official control
  request returned HTTP 409 `official stackchan websocket is not connected`.
  Current body evidence remains Xiaozhi MCP delivery, not official typed-frame
  relay acceptance.
- No firmware flash, NVS write, provider secret output, generic product flash,
  Git prune/gc, or internal-test3 voice/protocol rollback occurred.

Previous control update, 2026-06-05 05:31 CST:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read after
  the Gateway remediation. Remaining power/hardware findings were compared
  against the product overlay and official StackChan PMIC setup.
- Commit `fda7769 fix(firmware): restore stackchan power key pmic config` is
  pushed. The product overlay now keeps the WDT-safe direct Xiaozhi start path
  while restoring AXP2101 PWRON/OFFLEVEL source handling and the 4s hardware
  power-key long-press register.
- Focused product-overlay tests passed, `git diff --check` passed, and the
  guarded product build passed.
- Guarded product flash plan and execute passed on `/dev/cu.usbmodem1101`.
  Execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-052359-1780608239212784000.json`.
  The flashed app artifact is
  `a21-stackchan-official-xiaozhi-compatible.bin`, SHA
  `b665af4e78fae0c4dea10db04a2ea90502d88f26322dcca3c234abb8f355fc3e`.
- Product device `44:1b:f6:e2:6a:60` reconnected to public Gateway after
  flash and returned fresh `device.heartbeat`.
- Public `full_check` trace
  `a21-trace-full-check-pmic-key-fda7769-20260605` and roleplay ritual trace
  `a21-trace-mode-ritual-pmic-key-fda7769-20260605` both returned HTTP 200
  `status=delivered`; both remain `physical_accepted=false`.
- Public `GET /v1/power-lifecycle?device_id=44:1b:f6:e2:6a:60` returned
  `overall_status=physical_pending`, `xiaozhi_ws_online=true`,
  `battery_telemetry=missing`, and no physical acceptance for no-cable cold
  boot or the power button.
- Public `GET /v1/xiaozhi/mcp-capabilities?device_id=44:1b:f6:e2:6a:60`
  returned the expected safe tool set: status/screen/speaker/head/LED allowed,
  power shutdown/sleep, reboot, firmware upgrade, camera, NFC, infrared, and
  app lifecycle blocked.
- No generic `xiaozhi.bin` product flash, NVS write, provider/V21 execution,
  Git prune/gc, or internal-test3 voice/protocol rollback occurred.

Previous control update, 2026-06-05 05:13 CST:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was consumed and
  mapped to current code. Software-remediable findings are closed in the
  current worktree: Gateway Xiaozhi data races, abort/barge-in pacer blocking,
  delayed fast ack defaults, and namespace preflight/doctor failures.
- Gateway now exposes `GET /v1/power-lifecycle` and
  `POST /v1/power-lifecycle-acceptance`; `GET /v1/hardware-acceptance` now
  includes `power_lifecycle` alongside `mode_ritual` and `full_check`.
- Xiaozhi MCP power/shutdown/sleep remains blocked. The allowed MCP surface is
  still screen/status, speaker volume, robot head, and robot LED only.
- Local verification passed: focused Gateway power/hardware tests, touched
  package tests, focused Gateway `-race`, `git diff --check`,
  `GOMAXPROCS=2 make verify`, `GOMAXPROCS=2 make preflight`, and
  `GOMAXPROCS=2 make doctor`.
- ECS `47.103.57.217` was deployed through `/opt/a21.next` safe swap. Remote
  focused Gateway/App/runtimeguard tests and build passed, service restarted
  active, and `/healthz` passed.
- Public `/xiaozhi/ota/` returned `ws://47.103.57.217/v1/xiaozhi`.
- Public product mode ritual
  `a21-trace-mode-ritual-power-state-20260605-0509` and `full_check`
  `a21-trace-full-check-power-state-20260605-0509` both returned HTTP 200
  `status=delivered`.
- Public `GET /v1/hardware-acceptance?device_id=44:1b:f6:e2:6a:60` returned
  `overall_status=physical_pending`: `mode_ritual` and `full_check` are
  machine-delivered; `power_lifecycle` is online but physically unaccepted.
- Public `GET /v1/power-lifecycle?device_id=44:1b:f6:e2:6a:60` returned
  `overall_status=physical_pending`, `xiaozhi_ws_online=true`,
  `battery_telemetry=missing`, and no physical power-button/cold-boot
  acceptance.
- Public repeat-3 voice bench
  `reports/a21-xiaozhi-voice-bench-20260605-051249.326210000.json` passed
  3/3 answer and 3/3 barge-in turns, `failure_count=0`,
  answer first-audio P95 `1493 ms`, and barge-in stop P95 `22 ms`.
- Product readiness
  `reports/a21-product-readiness-20260605-051255.json` remains
  `server_side_blocked`, `launch_ready=false`, `demo_ready=true`. Missing
  real evidence is still `real_provider_smoke`,
  `physical_stackchan_prd_acceptance`, and `roleplay_voice_runtime`.
- No firmware flash, NVS write, provider secret output, generic product flash,
  Git prune/gc, or internal-test3 voice/protocol rollback occurred.

Previous control update, 2026-06-05 04:10 CST:

- `T-NO-CABLE-BOOT-POWER-LIFECYCLE-001` is active and recorded in
  `docs/plans/2026-06-05-no-cable-boot-power-lifecycle-recovery.md`.
- The operator reported that the physical power button still did not start the
  standalone prototype. This remains unaccepted; previous evidence only proved
  USB/flash-reset boot, cloud reconnect, and body MCP delivery.
- Commit `fabffd4` attempted a request-through-official-lifecycle autostart.
  It built and flashed successfully, but post-flash body commands returned
  HTTP 409 `xiaozhi websocket is not connected`.
- Read-only serial evidence on `/dev/cu.usbmodem1101` showed repeated task
  watchdog triggers with CPU0 running `main`. Decoded backtrace pointed at
  `GetMooncake().uninstallAllApps()` from `app_main`, specifically
  AppSetup/AppLauncher LVGL teardown. That lifecycle request path is rejected
  for the product lane until a separate teardown transition proves it safe.
- Commit `eeeb699 fix(firmware): avoid unsafe mooncake teardown autostart` is
  pushed. It restores the WDT-safe direct `GetHAL().startXiaozhi()` product
  path while keeping the guarded official-compatible artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- Product build passed with report
  `reports/a21-stackchan-official-baseline-20260605-040655-1780603615828792000.json`
  and app SHA
  `065e23976722aa7630d0dccf8ee80dff2674785ce9bb6221f67769368a960284`.
- `GOMAXPROCS=2 make verify` passed. Guarded flash plan and execute passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-040818-1780603698194451000.json`
  and
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-040923-1780603763139018000.json`.
  The execution report records clean worktree, commit `eeeb6998a9b0`, and
  `flash_executed=true`.
- After flash, serial showed MultiNet wake commands loaded, audio codec open,
  quiet control WebSocket open, and Xiaozhi session
  `a21-session-44-1b-f6-e2-6a-60`. Public `/v1/devices` showed the product
  device online with fresh heartbeat.
- Live traces `a21-trace-mode-ritual-after-wdt-safe-eeeb699-20260605` and
  `a21-trace-full-check-after-wdt-safe-eeeb699-20260605` both returned HTTP
  200 `status=delivered`; final public hardware-acceptance summary remained
  `overall_status=physical_pending`.
- No NVS write, provider execution, V21 execution, generic `xiaozhi.bin`
  product flash, Git prune/gc, or internal-test3 voice/protocol rollback
  occurred.

Previous control update, 2026-06-05 03:24 CST:

- `T-WORKSPACE-CONNECTED-HARDWARE-AUTO-ADOPTION-001` is pushed and deployed
  on ECS. Commit `1c9dece` adds `/workspace` boot-time adoption of the online
  product hardware device from `GET /v1/devices`, plus a `Connected device`
  refresh control and export metadata `connected_device_count`.
- Remote `/opt/a21.next` focused Gateway tests passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestHardwareAcceptance|TestVoiceModeRitual' -count=1`.
  Remote build passed and `a21-gateway.service` restarted active. Public
  direct `/healthz` and `/workspace` smoke passed.
- The product app was flashed on `/dev/cu.usbmodem1101` through the guarded
  official-compatible product lane. Plan report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-032219-1780600939242091000.json`.
  Execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-032331-1780601011958120000.json`.
  The execution report records `status=passed`, `flash_executed=true`, clean
  worktree, commit `1c9dece8b37e`, and app artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- No NVS write was performed and no generic `xiaozhi.bin` product lane was
  used.
- After the flash, product device `44:1b:f6:e2:6a:60` produced fresh online
  heartbeats, then live roleplay ritual trace
  `a21-trace-mode-ritual-after-flash-1c9dece-20260605` and live `full_check`
  trace `a21-trace-full-check-after-flash-1c9dece-20260605` both returned
  HTTP 200 `status=delivered`.
- Final public hardware-acceptance summary returned
  `overall_status=physical_pending`; both `mode_ritual` and `full_check` were
  `delivery_status=delivered`, with next actions
  `accept_visible_mode_ritual` and `accept_visible_full_check`.
- Physical acceptance remains pending until the operator or an instrument
  confirms visible screen/RGB/head movement.

Previous control update, 2026-06-05 04:05 CST:

- `T-HARDWARE-ACCEPTANCE-SUMMARY-BOARD-001` is pushed and deployed on ECS.
  Commit `5d786ef` adds `GET /v1/hardware-acceptance` and the `/workspace`
  `Acceptance Board`.
- The summary is read-only. It reads redacted device registry capabilities and
  reports `mode_ritual` and `full_check` delivery/physical state plus the next
  operator action; it never upgrades machine evidence into physical
  acceptance.
- Local TDD evidence: before implementation,
  `TestWorkspaceConsolePageServed` missed `/v1/hardware-acceptance`, and the
  endpoint returned HTTP 404. After implementation, focused Gateway tests,
  `git diff --check`, and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests and build passed, then
  `a21-gateway.service` safe-swapped active. Loopback and public direct
  `/healthz` passed.
- Public `/workspace` smoke found `Acceptance Board`, `Refresh Acceptance`,
  `hardwareAcceptanceStatus`, `hardwareAcceptanceItems`, and
  `/v1/hardware-acceptance`.
- First public summary after restart correctly returned
  `overall_status=machine_evidence_pending` because the in-memory registry had
  restarted.
- Live roleplay ritual trace
  `a21-trace-mode-ritual-summary-ready-5d786ef-202606050405` returned HTTP
  200 with `step_delay_ms=180` and `total_planned_delay_ms=540`.
- Live `full_check` trace
  `a21-trace-full-check-summary-ready-5d786ef-202606050405` returned HTTP 200
  with `step_delay_ms=180`, `total_planned_delay_ms=2700`, and trace summary
  `last_offset_ms=2713`.
- Final public hardware-acceptance summary for product device
  `44:1b:f6:e2:6a:60` returned `overall_status=physical_pending`; both
  `mode_ritual` and `full_check` were `delivery_status=delivered`,
  `physical_accepted=false`, with next actions
  `accept_visible_mode_ritual` and `accept_visible_full_check`.
- Product device remained `connection_status=online` and
  `current_voice_mode=roleplay`.
- No firmware build/flash, no NVS write, no provider/V21 execution, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred.

Previous control update, 2026-06-05 03:45 CST:

- `T-VOICE-MODE-RITUAL-PHYSICAL-ACCEPTANCE-SURFACE-001` is pushed and
  deployed on ECS. Commit `87625a2` adds
  `POST /v1/voice-mode-ritual-acceptance` and the `/workspace`
  `Accept Visible Mode Ritual` control.
- The endpoint is intentionally narrow: it requires `device_id`,
  `voice_mode`, matching delivered `trace_id` and `session_id`,
  `screen_visible=true`, `rgb_visible=true`, `servo_visible=true`, and
  `observer=operator` or `observer=instrument`. Missing matching evidence
  returns HTTP 409 instead of overclaiming physical acceptance.
- Successful acceptance records schema
  `a21.gateway.voice_mode_ritual_acceptance.v1`, trace marker
  `voice_mode.ritual.<mode>.physical_acceptance.accepted`, and redacted
  registry fields such as `voice_mode_ritual_physical_accepted=true`,
  `voice_mode_ritual_screen_physical_accepted=true`,
  `voice_mode_ritual_rgb_physical_accepted=true`, and
  `voice_mode_ritual_servo_physical_accepted=true`.
- Local TDD evidence: before implementation,
  `TestWorkspaceConsolePageServed` missed
  `/v1/voice-mode-ritual-acceptance`, and the acceptance endpoint returned
  HTTP 404. After implementation, focused Gateway tests, `git diff --check`,
  and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests and build passed, then
  `a21-gateway.service` safe-swapped active. Loopback and public direct
  `/healthz` passed.
- Public `/workspace` smoke found `Accept Visible Mode Ritual`,
  `acceptModeRitualPhysical`, and `/v1/voice-mode-ritual-acceptance`.
- Public negative acceptance smoke without matching delivered ritual evidence
  returned HTTP 409
  `matching voice mode ritual evidence is required before physical acceptance`.
- Live roleplay trace
  `a21-trace-mode-ritual-roleplay-acceptance-ready-87625a2-202606050345`
  returned HTTP 200 with `step_delay_ms=180`,
  `total_planned_delay_ms=540`, and trace summary `last_offset_ms=543`.
- Final public `/v1/devices` check showed product device
  `44:1b:f6:e2:6a:60` online, `current_voice_mode=roleplay`,
  `screen_theme=auto`, `screen_brightness=58`, RGB `120/48/96`, head
  `yaw=0,pitch=24,speed=180`, and
  `voice_mode_ritual_physical_accepted=false`. No physical acceptance was
  recorded because no operator or instrument confirmation was provided.
- No firmware build/flash, no NVS write, no provider/V21 execution, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred.

Previous control update, 2026-06-05 03:31 CST:

- `T-VOICE-MODE-HARDWARE-RITUAL-PACING-001` is pushed and deployed on ECS.
  Commit `a5d9b9d` adds visible pacing to
  `POST /v1/voice-mode-ritual` by reusing the existing body-scene step-delay
  policy.
- This transition is not a voice-chain revalidation. It does not touch
  ASR/TTS/wake/audio protocol, provider, V21, firmware, NVS, flash, camera,
  NFC, infrared, or app lifecycle.
- The endpoint response now includes `step_delay_ms` and
  `total_planned_delay_ms`. ECS runtime keeps
  `A21_BODY_SCENE_STEP_DELAY_MS=180`, so the four-step screen/RGB/head ritual
  plans 540 ms of visible spacing.
- Local TDD evidence: before implementation,
  `TestVoiceModeRitualProfessionalSendsHardwareSequence` failed because
  `step_delay_ms` was missing. After implementation, focused Gateway tests,
  `git diff --check`, and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests and build passed, then
  `a21-gateway.service` safe-swapped active. Loopback and public direct
  `/healthz` passed.
- Public `/workspace` smoke found `Run Roleplay Ritual`,
  `Run Professional Ritual`, `modeRitualStatus`,
  `data-mode-ritual`, and `/v1/voice-mode-ritual`.
- Live professional trace
  `a21-trace-mode-ritual-professional-paced-a5d9b9d-202606050330`
  delivered HTTP 200 with `selected_voice_mode=professional`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`,
  `provider_executed=false`, `v21_executed=false`, and trace summary
  `last_offset_ms=541`.
- Live roleplay restore trace
  `a21-trace-mode-ritual-roleplay-paced-a5d9b9d-202606050331`
  delivered HTTP 200 with `selected_voice_mode=roleplay`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`, and trace summary
  `last_offset_ms=541`.
- Final public `/v1/devices` check showed product device
  `44:1b:f6:e2:6a:60` online and restored to
  `current_voice_mode=roleplay`, with `screen_theme=auto`,
  `screen_brightness=58`, RGB `120/48/96`, head
  `yaw=0,pitch=24,speed=180`, and
  `voice_mode_ritual_physical_accepted=false`.
- Physical acceptance remains pending until the operator or an instrument
  confirms visible mode-switch screen/RGB/head movement.
- No Git prune/gc was run despite the historical loose-object warning.

Previous control update, 2026-06-05 02:50 CST:

- `T-WORKSPACE-HARDWARE-BODY-SCENE-PHYSICAL-ACCEPTANCE-001` is pushed and
  deployed on ECS. Commit `98700ab` adds
  `POST /v1/xiaozhi/body-scene-acceptance` plus the `/workspace`
  `Accept Visible Full Check` control.
- The acceptance endpoint is intentionally narrow: it accepts only
  `scene=full_check`, requires matching delivered `trace_id` and `session_id`,
  requires `screen_visible=true`, `rgb_visible=true`,
  `servo_visible=true`, and requires `observer=operator` or
  `observer=instrument`. Missing matching evidence returns HTTP 409.
- Successful acceptance records schema
  `a21.gateway.xiaozhi_body_scene_acceptance.v1`, trace marker
  `xiaozhi.body_scene.full_check.physical_acceptance.accepted`, and redacted
  registry fields such as `body_scene_physical_accepted=true`,
  `body_scene_screen_physical_accepted=true`,
  `body_scene_rgb_physical_accepted=true`, and
  `body_scene_servo_physical_accepted=true`.
- Local TDD evidence: before implementation,
  `TestWorkspaceConsolePageServed` missed
  `/v1/xiaozhi/body-scene-acceptance` and
  `TestXiaozhiBodyScenePhysicalAcceptanceRecordsOperatorEvidence` returned
  HTTP 404. After implementation, focused Gateway tests passed and
  `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests and build passed, then
  `a21-gateway.service` safe-swapped active. Loopback and public `/healthz`
  passed, and public `/workspace` smoke found
  `Accept Visible Full Check`, `acceptHardwareScenePhysical`,
  `hardwareSceneAcceptanceStatus`, and `/v1/xiaozhi/body-scene-acceptance`.
- Public negative smoke proved the acceptance guard: a request without matching
  delivered body-scene evidence returned HTTP 409
  `matching body scene evidence is required before physical acceptance`.
- Live product trace
  `a21-trace-hardware-full-check-acceptance-ready-98700ab-202606050250`
  returned HTTP 200 `scene=full_check`, `step_delay_ms=180`,
  `total_planned_delay_ms=2700`, and trace summary
  `last_offset_ms=2710`. `/v1/devices` recorded
  `last_body_scene_trace_id` and `last_body_scene_session_id`, so the
  operator can now record physical acceptance from `/workspace`.
- Physical acceptance has not been recorded in this round because no operator
  or instrument confirmation was provided. The body-scene acceptance surface is
  ready; `PHYSICAL-PENDING` remains.
- No firmware build/flash, no NVS write, no provider/V21 execution, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred.

Previous control update, 2026-06-05 02:30 CST:

- `T-WORKSPACE-HARDWARE-BODY-SCENE-PACING-001` is pushed and deployed on ECS.
  Commit `6f43646` adds bounded inter-step pacing to
  `POST /v1/xiaozhi/body-scene` so machine-delivered screen/RGB/servo scenes
  are operator-visible instead of delivered in a single burst.
- The `gateway` command now defaults body-scene step pacing to 180 ms and ECS
  explicitly records `A21_BODY_SCENE_STEP_DELAY_MS=180` in
  `/etc/a21/runtime.env`. Positive env values can override the default; the
  server caps delays at 1000 ms. Direct `NewServer()` tests keep a 20 ms
  low-latency default.
- Body-scene responses now include `step_delay_ms` and
  `total_planned_delay_ms` while still returning `physical_accepted=false`.
- Local TDD evidence: the pacing test first failed without
  `step_delay_ms`; after implementation, focused Gateway/App tests passed and
  `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway/App tests and build passed, then
  `a21-gateway.service` safe-swapped active. Loopback and public `/healthz`
  passed.
- Live product trace
  `a21-trace-hardware-full-check-paced-6f43646-202606050230` returned HTTP 200
  `status=delivered`, `scene=full_check`, `step_delay_ms=180`,
  `total_planned_delay_ms=2700`, and 16 redacted steps.
- The trace endpoint recorded 32 ordered markers with
  `summary.last_offset_ms=2710`, proving the scene is now paced over the
  intended window instead of emitted in about 1 ms.
- `/v1/devices` recorded `last_body_scene=full_check`,
  `last_body_scene_step=16`, final `screen_theme=auto`,
  `screen_brightness=55`, final head `yaw=0,pitch=18,speed=200`, final RGB
  `0/0/32`, and the product device remained online after a follow-up heartbeat
  check about 12 seconds later.
- This is improved machine-readable product-socket body evidence and operator
  check ergonomics, not physical acceptance. Operator or instrument
  confirmation is still required before promoting the body scene to
  product-accepted.
- No firmware build/flash, no NVS write, no provider/V21 execution, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred.

Previous control update, 2026-06-05 02:17 CST:

- `T-WORKSPACE-HARDWARE-FULL-CHECK-SCENE-001` is pushed and deployed on ECS.
  Commit `9171751` adds `scene=full_check` to
  `POST /v1/xiaozhi/body-scene` and adds a `Full Check` button to the
  `/workspace` Hardware Scenes panel.
- The scene uses only bounded official MCP tools already accepted by the
  product socket path: screen theme, screen brightness, RGB LED, and head
  yaw/pitch/speed. It combines showtime, focus, and reset poses into a
  16-step operator-visible diagnostic sequence and keeps
  `physical_accepted=false`.
- Local TDD red/green completed: before implementation,
  `TestWorkspaceConsolePageServed` missed
  `data-hardware-scene="full_check"` and
  `TestXiaozhiBodySceneFullCheckRunsOperatorVisibleSequence` returned HTTP
  400. After implementation, focused Gateway tests passed and
  `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests and build passed, then
  `a21-gateway.service` safe-swapped active. Public `/healthz` passed and
  `/workspace` smoke found `Full Check`,
  `data-hardware-scene="full_check"`, and `/v1/xiaozhi/body-scene`.
- Product device `44:1b:f6:e2:6a:60` returned online after the ECS restart and
  stayed online through 8 public heartbeat polls.
- Live product full check trace
  `a21-trace-hardware-full-check-9171751-202606050217` returned HTTP 200
  `status=delivered`, `scene=full_check`, and 16 redacted steps. The trace
  endpoint recorded 32 ordered markers through
  `xiaozhi.body_scene.full_check.step16.robot_head_angles_set.sent`.
- `/v1/devices` recorded `last_body_scene=full_check`,
  `last_body_scene_step=16`, final `screen_theme=auto`,
  `screen_brightness=55`, final head `yaw=0,pitch=18,speed=200`, and final
  RGB `0/0/32`. A follow-up public check about 12 seconds later still showed
  the device online with heartbeat updates.
- This is machine-readable product-socket body evidence and an improved
  operator check surface, not physical acceptance. Operator or instrument
  confirmation is still required before promoting the body scene to
  product-accepted.
- No firmware build/flash, no NVS write, no provider/V21 execution, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred.

Latest control update, 2026-06-05 02:08 CST:

- `T-FIRMWARE-QUIET-RECONNECT-PRODUCT-FLASH-001` is executed on the product
  device through the guarded official-compatible product lane. The flashed app
  artifact was
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  with SHA-256
  `3eef974929aed78cdd77232897485aaac25bce8aa98daa4d8d78b3d96662b7ac`.
- Flash plan report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-020621-1780596381380115000.json`
  returned `status=ready`; flash execution report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-020726-1780596446779669000.json`
  returned `status=passed`, `flash_allowed=true`, and
  `flash_executed=true`.
- The flash guard verified branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`, commit
  `dc8c752f9fee`, non-detached HEAD, and zero dirty tracked files before
  writing. No NVS write and no generic `xiaozhi.bin` product flash path were
  used.
- After reboot, public `/v1/devices` showed product device
  `44:1b:f6:e2:6a:60` returning from
  `connection_status=xiaozhi_ws_disconnected` to `connection_status=online`
  with heartbeat updates. Capabilities kept product keepalive/playback/touch
  and touch reaction gates; `xiaozhi_product_state_reactions` stayed absent
  per the current ECS runtime safety gate.
- Product body scene trace
  `a21-trace-hardware-showtime-flash-b9c0baa-202606050208` returned HTTP 200
  `status=delivered` through `xiaozhi_mcp_sequence`. The scene emitted 8
  bounded screen/RGB/head MCP steps and 16 trace markers. `/v1/devices`
  recorded `last_body_scene=showtime`, `screen_theme=dark`,
  `screen_brightness=72`, final head `yaw=0,pitch=24,speed=220`, and final
  RGB `0/36/96`.
- A follow-up public `/v1/devices` check about 12 seconds later still showed
  the device online with heartbeat updates and the showtime registry state.
- This closes the previous socket absence blocker for machine-readable
  hardware-scene delivery after Gateway restart. It is not yet physical
  acceptance: `physical_accepted=false` remains until operator or instrument
  evidence confirms visible screen/RGB/head movement.
- No provider/V21 execution, no camera/NFC/IR expansion, no reboot/OTA/
  snapshot/video/app-lifecycle exposure, no Git prune/gc, and no
  internal-test3 voice/protocol rollback occurred.

Latest control update, 2026-06-05 02:00 CST:

- `T-XIAOZHI-LISTEN-START-STATE-REACTION-SUPPRESS-001` is pushed and
  deployed on ECS. Commit `72e6bcc` suppresses automatic
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS` MCP writes for the stock physical
  `listening/listen_start` boundary and records
  `xiaozhi.state_reaction.listen_start_suppressed` plus redacted runtime echo
  instead. `thinking` and other accepted state reactions remain covered in
  tests.
- Root evidence before the fix: after the product device sent `xiaozhi.hello`,
  trace `a21-trace-44-1b-f6-e2-6a-60` showed `xiaozhi.listen.start`, immediate
  state-reaction `robot_led_color`, `xiaozhi.state_reaction.failed`, then
  `asr.stream.error`, `asr.stream.cancelled`, and
  `xiaozhi.opus_ingress.queue_cancelled.socket_closed` within 232 ms.
- Runtime safety action: `/etc/a21/runtime.env` on ECS now has
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false` while keeping playback events,
  touch events, and touch reactions enabled. This is a runtime gate rollback
  for the regressed automatic state reaction path, not a code rollback.
- Commit `b9c0baa` is pushed as a firmware overlay candidate. It moves
  `EnsureA21ControlChannel()` from VAD-change-only probing to periodic
  `MAIN_EVENT_CLOCK_TICK` probing, so an idle product device can keep trying
  the quiet Xiaozhi control websocket even without new VAD events.
- No-flash firmware validation passed:
  `make a21-stackchan-official-xiaozhi-compatible-build`, producing
  `a21-stackchan-official-xiaozhi-compatible.bin` with app SHA-256
  `3eef974929aed78cdd77232897485aaac25bce8aa98daa4d8d78b3d96662b7ac`.
- Post-deploy public `/healthz` and `/workspace` smoke passed. Final public
  `/v1/devices` showed product device `44:1b:f6:e2:6a:60` had reappeared but
  remained `connection_status=xiaozhi_ws_disconnected`. Its capabilities no
  longer included `xiaozhi_product_state_reactions`, and the refreshed trace
  had no `xiaozhi.state_reaction.*` MCP markers; it still showed
  `listen.start` followed by ASR/socket-close cancellation. Live scene physical
  acceptance is still pending product device reconnect resilience or a
  foreground guarded firmware flash/power-cycle window.
- No firmware flash, no NVS write, no provider/V21 execution, no camera/NFC/IR
  expansion, no reboot/OTA/snapshot/video/app-lifecycle exposure, no Git
  prune/gc, and no internal-test3 voice/protocol rollback occurred.

Previous control update, 2026-06-05 01:43 CST:

- `T-WORKSPACE-HARDWARE-SCENE-CONTROL-SURFACE-001` is pushed and deployed on
  ECS. Commit `88e1549` adds `POST /v1/xiaozhi/body-scene` plus a
  `/workspace` Hardware Scenes panel for one-click `showtime`, `focus`, and
  `reset` sequences that combine bounded screen theme/brightness MCP writes
  with bounded RGB/head MCP writes.
- Local focused workspace/body-scene/body-preset/body-motion tests passed,
  full local `GOMAXPROCS=2 make verify` passed, remote focused Gateway tests
  and build passed in `/opt/a21.next`, `a21-gateway` safe-swapped active,
  loopback `/healthz` passed, and public direct-source `/healthz` plus
  `/workspace` HTML smoke passed.
- Public `/workspace` HTML now contains `Hardware Scenes`,
  `/v1/xiaozhi/body-scene`, `runHardwareScene`,
  `refreshHardwareSceneTrace`, `hardware_scene_trace_id`, and the
  `data-hardware-scene="showtime"` control.
- A live product `showtime` call against device `44:1b:f6:e2:6a:60` returned
  HTTP 409 `xiaozhi websocket is not connected` after the service restart, and
  public `/v1/devices` was empty during the post-deploy poll window. This is
  device socket absence after Gateway safe-swap, not a voice/protocol rollback.
  Physical scene acceptance remains pending until the product device reconnects
  and an operator or instrument confirms screen, RGB, and servo movement.
- This transition did not build or flash firmware, write NVS, execute
  providers/V21, expose reboot/OTA/snapshot/video/camera/NFC/IR/app lifecycle,
  run Git prune/gc, or renew internal-test3 voice-chain acceptance.

Previous control update, 2026-06-05 01:32 CST:

- `T-WORKSPACE-OFFICIAL-ACTION-FALLBACK-001` is deployed on ECS. Commit
  `7dfbb10` keeps `/v1/stackchan/official/control` truthful, but changes the
  `/workspace` Official Actions panel so a disconnected `/stackChan/ws`
  official relay automatically falls back to the already-delivered
  `/v1/xiaozhi/body-preset` or `/v1/xiaozhi/body-motion` product path.
- Local focused workspace/body-motion/official-action tests passed, full local
  `GOMAXPROCS=2 make verify` passed, remote focused Gateway tests/build passed
  in `/opt/a21.next`, `a21-gateway` safe-swapped active, loopback `/healthz`
  passed, and public direct-source `/healthz` plus `/workspace` HTML smoke
  passed.
- Public `/workspace` HTML now contains `officialActionFallback`,
  `runOfficialActionFallback`, `fallback_delivered`,
  `official_action_blocked_reason`, and `official_action_fallback`. Public API
  evidence still shows official relay HTTP 409 for product device
  `44:1b:f6:e2:6a:60`, while fallback body-motion `dance` delivered through
  trace `a21-trace-workspace-body-motion-dance-7dfbb10` with 10 MCP/body-motion
  markers and `/v1/devices` updated to `last_body_motion=dance`.
- This transition improves foreground product ergonomics while preserving the
  official-frame acceptance boundary. No firmware build/flash, no NVS write,
  no provider/V21 execution, no camera/NFC/IR expansion, no reboot/OTA/
  snapshot/video/app-lifecycle exposure, no Git prune/gc, and no internal-test3
  voice/protocol rollback occurred.

Previous control update, 2026-06-05 01:26 CST:

- `T-WORKSPACE-OFFICIAL-ACTION-CONTROL-SURFACE-001` and
  `T-XIAOZHI-BODY-MOTION-SEQUENCE-001` are deployed on ECS. Commit `6ce372e`
  adds an Official Actions section to `/workspace` for semantic
  state/face/motion/dance relay through `/v1/stackchan/official/control`, and
  commit `e5ae4d1` adds `POST /v1/xiaozhi/body-motion` plus `/workspace` MCP
  motion controls for `look_up`, `nod`, `shake`, `dance`, and `stop`.
- Local focused tests passed for workspace, official action, body preset, and
  body motion. Full local `GOMAXPROCS=2 make verify` passed. Remote focused
  Gateway tests/build passed in `/opt/a21.next`, `a21-gateway` safe-swapped
  active, loopback `/healthz` passed, and public direct-source `/healthz` plus
  `/workspace` HTML smoke passed.
- Public official action relay on product device `44:1b:f6:e2:6a:60` returned
  HTTP 409 `official stackchan websocket is not connected`; this is now an
  honest surfaced runtime gap for the separate `/stackChan/ws` official avatar
  relay, not a hidden failure or voice rollback.
- Public MCP-backed body motion `dance` passed on product device
  `44:1b:f6:e2:6a:60` with trace
  `a21-trace-workspace-body-motion-dance-e5ae4d1`. Response was
  `status=delivered`, `delivered_transport=xiaozhi_mcp_sequence`, 5 redacted
  steps, and `physical_accepted=false`. Trace recorded 10 generic/body-motion
  markers, and `/v1/devices` recorded `last_body_motion=dance`,
  `last_body_motion_step=5`, final head `yaw=0,pitch=24,speed=220`, LED
  `0/168/80`, and device online.
- These transitions improve the product action surface while preserving the
  official-compatible product lane. No firmware build/flash, no NVS write, no
  provider/V21 execution, no camera/NFC/IR expansion, no reboot/OTA/snapshot/
  video/app-lifecycle exposure, no Git prune/gc, and no internal-test3
  voice/protocol rollback occurred.

Previous control update, 2026-06-05 01:14 CST:

- `T-WORKSPACE-HARDWARE-SCREEN-CONTROL-SURFACE-001` is deployed on ECS.
  Commit `c74261d` adds a Hardware Screen section to `/workspace` with a
  brightness slider, theme controls for `light`, `dark`, and `auto`, device
  status, screen info, MCP capability refresh, safe trace display, and
  `screen_control_*` metadata export fields.
- Local focused workspace/screen tests passed, full local `GOMAXPROCS=2 make
  verify` passed, remote focused Gateway tests/build passed in `/opt/a21.next`,
  `a21-gateway` safe-swapped active, loopback `/healthz` passed, and public
  direct-source `/healthz` plus `/workspace` HTML smoke passed.
- Public screen/status executions through the deployed Gateway on product
  device `44:1b:f6:e2:6a:60` passed for traces
  `a21-trace-workspace-screen-brightness-c74261d`,
  `a21-trace-workspace-screen-theme-c74261d`, and
  `a21-trace-workspace-screen-info-c74261d`. Responses were
  `status=delivered`, `delivered_transport=xiaozhi_mcp`, and
  `result_redacted=true`. Trace markers recorded
  `xiaozhi.mcp.screen_brightness.sent`, `xiaozhi.mcp.screen_theme.sent`, and
  `xiaozhi.mcp.screen_info.sent`; `/v1/devices` recorded
  `screen_brightness=62` and `screen_theme=dark`; the device stayed online.
- This transition is a hardware screen/status product surface deployment, not a
  voice chain re-acceptance or protocol rollback. No firmware build/flash, no
  NVS write, no provider/V21 execution, no camera/NFC/IR expansion, no Git
  prune/gc, and no internal-test3 voice/protocol rollback occurred.

Previous control update, 2026-06-05 01:07 CST:

- `T-WORKSPACE-BODY-PRESET-CONTROL-SURFACE-001` is deployed on ECS. Commit
  `d361176` adds a Body Presets section to `/workspace` with buttons for
  `ready`, `listening`, `thinking`, `speaking`, `celebrate`, and `reset_idle`,
  plus safe trace/status display and metadata export fields.
- Local focused workspace/body tests passed, full local `GOMAXPROCS=2 make
  verify` passed, remote focused Gateway tests/build passed in `/opt/a21.next`,
  `a21-gateway` safe-swapped active, loopback `/healthz` passed, and public
  direct-source `/healthz` plus `/workspace` HTML smoke passed.
- Public body-preset execution through the deployed Gateway on product device
  `44:1b:f6:e2:6a:60` passed for trace
  `a21-trace-workspace-body-ready-d361176`. Response was `status=delivered`,
  `delivered_transport=xiaozhi_mcp_sequence`, LED `0/36/96`, head
  `yaw=0,pitch=22,speed=180`, and `physical_accepted=false`. Trace markers and
  `/v1/devices` registry updated with `last_body_preset=ready`; the device
  stayed online.
- This transition is a body-control product surface deployment, not a voice
  chain re-acceptance or protocol rollback. No firmware build/flash, no NVS
  write, no provider/V21 execution, no camera/NFC/IR expansion, no Git
  prune/gc, and no internal-test3 voice/protocol rollback occurred.

Previous control update, 2026-06-05 00:58 CST:

- `T-STACKCHAN-OFFICIAL-BODY-PRESET-SEQUENCE-001` is deployed and has live
  product-socket evidence. Commit `da77d21` adds
  `POST /v1/xiaozhi/body-preset`, mapping `ready`, `listening`, `thinking`,
  `speaking`, `celebrate`, and `reset_idle` into bounded official
  `self.robot.set_led_color` plus `self.robot.set_head_angles` MCP writes.
- Local `GOMAXPROCS=2 make verify` passed before deployment. Remote focused
  body/MCP tests/build passed before ECS safe-swap, `a21-gateway` restarted
  active, and public `/healthz` returned ok.
- Live public execution on product device `44:1b:f6:e2:6a:60` with trace
  `a21-trace-live-body-preset-celebrate-202606050058` returned
  `status=delivered`, `delivered_transport=xiaozhi_mcp_sequence`, LED
  `0/168/80`, head `yaw=18,pitch=36,speed=260`, and
  `physical_accepted=false`. Public `/v1/devices` recorded
  `last_body_preset=celebrate` and the robot LED/head values.
- This transition did not flash firmware, write NVS, execute providers/V21, or
  expand camera/NFC/IR. It improves the product body-control surface, but the
  total state remains `PHYSICAL-PENDING` until operator or instrument evidence
  confirms visible LED/head movement and the remaining mic/audible PRD window.

Previous control update, 2026-06-05 00:48 CST:

- `T-XIAOZHI-HOST-SAY-INTERRUPT-CLASSIFICATION-001` is ready in code and
  tests. The previously observed medium/long `/v1/xiaozhi/say` `502` is now
  classified as a Gateway control-surface semantics bug when a product
  touch/wake/barge/abort cancellation intentionally interrupts an active
  downlink.
- Gateway `/v1/xiaozhi/say` now returns HTTP 200 with `status=interrupted`,
  safe `interrupt_reason`, partial `audio_chunks`, and
  `xiaozhi.say.interrupted` trace marker for intentional user/device
  interruption. True downlink/TTS errors remain HTTP 502 and are covered by
  `xiaozhi.say.downlink_error` tests.
- Focused Gateway tests passed for normal host-say, touch interruption, actual
  downlink error, WAV host-say, post-host-say suppression, product touch
  barge-in, and product touch reactions. Full `GOMAXPROCS=2 make verify`
  passed.
- Commit `45f363f` is pushed and deployed to ECS `47.103.57.217`; remote
  focused Gateway tests/build passed, `a21-gateway` restarted active, public
  `/healthz` returned ok, and direct-source `/v1/devices` showed product
  device `44:1b:f6:e2:6a:60` online with fresh `xiaozhi.hello`.
- This transition did not flash firmware, write NVS, execute providers/V21, or
  roll back internal-test3 voice/protocol changes. Total state remains
  `PHYSICAL-PENDING` because mic ingress and trusted audible or instrument
  observation are still missing from one product physical window.

Previous control update, 2026-06-05 00:36 CST:

- `T-XIAOZHI-PHYSICAL-BARGE-IN-STOP-DONE-001` is partially promoted from
  blocked to product trace candidate for the touch/body path. The product app
  at `9413ed5` was flashed through the guarded official-compatible lane on
  `/dev/cu.usbmodem1101` without NVS write, using
  `a21-stackchan-official-xiaozhi-compatible.bin` SHA-256
  `4af28d25013111777f2bc82befd6b697ea27da3484e1acd7b8eff661007b0de6`.
- Live trace `a21-trace-speaking-barge-cn-20260605003524` on device
  `44:1b:f6:e2:6a:60` recorded real top-touch barge-in during a speaking
  window: `device.touch.barge_in.received`, `barge_in.detected`,
  `playback.stop`, and `device.playback.stop_done`.
- Fresh evidence
  `reports/a21-xiaozhi-physical-evidence-20260605-003553.699946000.json`
  remains `candidate_gateway_downlink` but now proves playback start `59 ms`
  and stop_done `26 ms`. Fresh half-duplex report
  `reports/a21-xiaozhi-half-duplex-acceptance-20260605-003553.730698000.json`
  remains `blocked` only for mic-ingress and trusted audible observation.
- Local evidence-reader code now accepts same-trace product runtime session
  drift from explicit request session to normalized device default session
  while preserving strict trace/device matching.
- Next action: collect a single physical window with wake/listen mic ingress,
  audible/instrument observation, and either top-touch or wake-word barge-in in
  the same reviewable evidence set; then rerun `xiaozhi-physical-prd-review`.

Previous control update, 2026-06-04 17:23 CST:

- `T-SERVER-SIDE-ROLEPLAY-VOICE-RUNTIME-GATE-001` now has fresh cloud runtime
  evidence rather than only static readiness. Pre-deploy report
  `reports/a21-roleplay-voice-probe-20260604-172311.json` passed, and final
  deployed report `reports/a21-roleplay-voice-probe-20260604-172658.json`
  also passed, proving selected role soul/scenario/memory prompt input,
  selected `a21_voice_clone_default`, StepFun text-stream execution, audio
  downlink, device playback start marker, and 45 audio playback chunks.
- Root causes fixed in this control cut: provider pipeline adapters now honor
  the `A21_VOICE_CLONE_CLI` compatibility alias, the DashScope CosyVoice
  wrapper no longer consumes generic Qwen realtime `A21_DASHSCOPE_TTS_MODEL`
  or `A21_DASHSCOPE_TTS_VOICE` values, and roleplay evidence parsing accepts
  real StackChan MAC-style device IDs without allowing URLs, paths, whitespace,
  or credential-shaped identities.
- Fresh `reports/a21-server-side-readiness-bundle-20260604-172722.json` is
  `server_side_candidate_ready`; fresh
  `reports/a21-product-readiness-20260604-172722.json` still correctly keeps
  launch/PRD false because `physical_stackchan_prd_acceptance` remains missing.

Follow-on physical transition update, 2026-06-04:

- `T-XIAOZHI-PHYSICAL-PRD-PROMOTE-GATE-001` now has a local product-safe
  playback acknowledgement adaptation path on Gateway and product overlay.
  Gateway can parse `hello.features.playback_events`; with explicit
  `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS=true`, a hardware-MAC product device
  that does not request debug features receives only `a21.profile=product` /
  `a21.playback_events=true`. The official-compatible product overlay now
  advertises that feature and may report playback `start` / `stop_done` after
  server allowance.
- The guarded no-flash official-compatible product build has passed for this
  overlay, producing
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  with SHA-256
  `e66a41ef486b866b076746bd064af2e3afb75e0a316515921bbc681b89fb36a8`
  and report
  `reports/a21-stackchan-official-baseline-20260604-175756-1780567076043462000.json`.
- This is a protocol adaptation for collecting the missing physical evidence.
  It does not change the total state away from `PHYSICAL-PENDING` until a real
  StackChan run supplies playback-start, bounded stop_done/barge-in evidence,
  and operator or instrumented audible observation.

Workspace device-binding update, 2026-06-04 18:13 CST:

- `T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001` now has a concrete
  cloud/workspace device-access guard rather than only PRD language. Gateway
  exposes memory-only `GET/POST/PUT /v1/workspace-device-bindings`, records
  safe bindings between A21 `device_id`, redacted `user_id`, redacted
  `workspace_id`, and allowed professional `query_scope` values, and adds a
  binding summary to `/v1/professional-workspace`.
- Professional mock turns and stock Xiaozhi professional turns now check the
  binding registry before V21 execution once a workspace has any binding
  record. Bound devices can proceed; unbound, revoked, deleted, or
  query-scope-denied devices fail before `v21.query.start` with safe
  read-ledger failure codes.
- `/workspace` now exposes device ID, bind, revoke, refresh, binding status,
  active binding count, and safe binding metadata export. This is not physical
  PRD evidence, real indexing, V21 merge/release, provider execution, firmware
  flash, or NVS work.

Workspace professional-query update, 2026-06-04:

- `T-INTERNAL-TEST4-PROFESSIONAL-QUERY-ENDPOINT-001` closes the Web/App
  professional consult-entry gap. Gateway now exposes
  `POST /v1/professional-query` with schema
  `a21.gateway.professional_query.v1`, reuses the professional checking cue,
  V21 adapter, device-binding guard, and read ledger, and rejects unsafe raw
  payload/evidence/provider fields before read-record/V21 execution.
- `/workspace` professional probe now calls `/v1/professional-query` instead
  of `/v1/mock-turn`, so the product web surface and the hardware/mock
  professional paths share the same guard and ledger behavior.
- This is not real indexing, durable cloud auth/tenant ACL, V21 merge/release,
  provider deployment, firmware flash, NVS, or physical StackChan professional
  consult acceptance.

Official robot MCP body-control update, 2026-06-04 20:36 CST:

- `T-STACKCHAN-OFFICIAL-ROBOT-MCP-BODY-CONTROL-001` deployed the first
  official body-control parity slice to ECS without flashing firmware or
  changing NVS. Gateway now allows only bounded official robot MCP tools
  `self.robot.get_head_angles`, `self.robot.set_head_angles`, and
  `self.robot.set_led_color`, emits official-compatible numeric JSON-RPC ids,
  and accepts redacted device-side `type=mcp` responses instead of sending a
  stock-unknown error reply.
- Physical serial evidence on device `44:1b:f6:e2:6a:60` showed official HAL
  execution for RGB `20,0,168` and head angles yaw `12`, pitch `30`, speed
  `150`. Gateway traces recorded
  `xiaozhi.mcp.robot_led_color.sent`,
  `xiaozhi.mcp.robot_head_angles_set.sent`, and two
  `xiaozhi.mcp.response.received` markers.
- The cut improves the visible StackChan body surface for internal test 4, but
  full PRD physical acceptance, no-cable boot/power behavior, screen product
  evidence, camera, NFC, infrared, and official app-lifecycle parity remain
  open transitions.

Screen/status MCP operation-surface update, 2026-06-04:

- `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001` now has named endpoint
  implementation deployed on ECS for `POST /v1/xiaozhi/device-status`,
  `POST /v1/xiaozhi/screen-brightness`,
  `POST /v1/xiaozhi/screen-theme`, and
  `GET /v1/xiaozhi/mcp-capabilities?device_id=<device_id>`.
- The endpoints reuse the same stock MCP delivery, whitelist, numeric
  JSON-RPC id, and redacted response behavior as the unified
  `/v1/xiaozhi/mcp-control` path.
- Fresh live evidence on device `44:1b:f6:e2:6a:60` recorded public capability
  response, command traces for device status, screen brightness, and screen
  theme, device-session MCP responses, `/v1/devices` registry values
  `screen_theme=dark` and `screen_brightness=55`, and serial logs
  `StackChanAvatarDisplay: SetTheme: dark` plus
  `Backlight: Set brightness to 55`.
- This is not full physical screen visual acceptance. Gateway-restart
  auto-reconnect remains open because the device did not reconnect after the
  ECS safe-swap until a hard reset.

Xiaozhi control-channel keepalive physical evidence, 2026-06-04:

- `T-STACKCHAN-XIAOZHI-CONTROL-KEEPALIVE-001` has product-lane physical
  evidence for the Gateway-restart auto-reconnect gap.
- Gateway now parses `hello.features.keepalive_events`, returns product
  `a21.keepalive_events=true` only for hardware-MAC Xiaozhi clients under the
  existing product playback-events runtime gate, records `device.heartbeat`,
  and still blocks product debug-only `state`, `face`, `display`, and `motion`
  device events.
- The official-compatible product overlay now sends
  `type=device, kind=heartbeat` through a protocol-layer keepalive timer and
  starts a protocol-layer reconnect task when heartbeat send fails.
- Final guarded no-flash product build passed with app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  and sha256
  `d43989209ee1be056f8d59b539bc48c6cdd2821fd9645793c18f134b6dee9179`.
- Guarded product flash passed on device `44:1b:f6:e2:6a:60` with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-214705-1780580825810604000.json`.
- Foreground restart proof passed without device hard reset: Gateway restarted
  at `2026-06-04 21:47:49 CST`; the device reappeared with `xiaozhi.hello` at
  `21:47:57` and resumed `device.heartbeat` at `21:48:08`.

StackChan product touch physical evidence, 2026-06-04:

- `T-STACKCHAN-OFFICIAL-TOUCH-ACTION-EVIDENCE-001` has product-lane physical
  evidence for the first official touch/body slice.
- Gateway commit `f16e71b` adds `hello.features.touch_events` parsing,
  product-only `A21_XIAOZHI_PRODUCT_TOUCH_EVENTS` allowance, Gateway trace /
  registry mapping for screen/top touch events, and stock Xiaozhi touch
  acceptance observation mode.
- The official-compatible product overlay advertises
  `features.touch_events=true`, only sends touch events after
  `a21.profile=product` / `a21.touch_events=true`, bridges screen touch and
  official top-touch HAL gestures, and keeps debug `features.device_events`
  disabled.
- ECS `47.103.57.217` was updated with the commit and root-only
  `A21_XIAOZHI_PRODUCT_TOUCH_EVENTS=true`; remote focused tests/build passed
  and `a21-gateway` restarted active.
- Guarded no-flash product build passed with app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  sha256
  `9b8366e387b10ffa784394e965f702734753f1c4c68192f17a11135e3b713216`.
- Guarded product flash passed with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-222334-1780583014061627000.json`.
- Public `/v1/devices` then showed `xiaozhi_feature_touch_events=true` and
  `xiaozhi_product_touch_events=true`.
- Touch acceptance reports passed for device `44:1b:f6:e2:6a:60`:
  `reports/a21-stackchan-touch-acceptance-20260604-222700.json`
  (`screen_touch`),
  `reports/a21-stackchan-touch-acceptance-20260604-222708.json`
  (`top_tap`),
  `reports/a21-stackchan-touch-acceptance-20260604-222719.json`
  (`top_swipe_forward`),
  `reports/a21-stackchan-touch-acceptance-20260604-222744.json`
  (`top_swipe_backward`), and
  `reports/a21-stackchan-touch-acceptance-20260604-222818.json`
  (`top_barge_in` on trace `a21-trace-touch-barge-in-20260604`).
- Directional swipes remain `guided_directional_touch` with
  `needs_affordance=true`; full PRD physical acceptance remains open for
  broader audio/playback evidence, screen visual acceptance, camera, NFC,
  infrared, app lifecycle, no-cable boot/power, and richer body expression.

StackChan touch body-reaction evidence, 2026-06-04:

- `T-STACKCHAN-OFFICIAL-TOUCH-BODY-REACTION-001` is deployed on ECS and
  closes the first foreground `touch -> body` reaction proof without firmware
  flash, NVS write, provider execution, V21 execution, or serial write.
- Gateway adds a separate `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true` gate,
  returns `a21.touch_reactions=true` only for hardware-MAC stock Xiaozhi
  clients with product touch allowance and `hello.features.mcp=true`, and
  uses only the already whitelisted official robot MCP tools
  `self.robot.set_led_color` and `self.robot.set_head_angles`.
- Device registry now exposes stable `last_touch_event`,
  `last_touch_source`, `last_touch_trace_id`, `last_touch_session_id`, and
  `last_touch_seen_ms` fields so later `xiaozhi.mcp.response.received`, Opus,
  or heartbeat events do not erase touch acceptance evidence.
- Runtime/physical evidence report
  `reports/a21-stackchan-touch-reaction-evidence-20260604-224756.json`
  passed on device `44:1b:f6:e2:6a:60`, trace
  `a21-trace-44-1b-f6-e2-6a-60`, session
  `a21-session-44-1b-f6-e2-6a-60`: observed `top_swipe_backward` from
  `top_sensor`, LED `120/60/0`, head `yaw=-18,pitch=24,speed=200`, markers
  `device.touch.top.swipe_backward.received`,
  `xiaozhi.touch_reaction.robot_led_color.sent`,
  `xiaozhi.touch_reaction.robot_head_angles_set.sent`, and redacted MCP
  responses.
- This is not screen visual acceptance, richer choreography, camera, NFC,
  infrared, no-cable boot/power, app lifecycle, or full PRD physical
  acceptance.

StackChan state body-reaction implementation, 2026-06-04:

- `T-STACKCHAN-OFFICIAL-STATE-BODY-REACTION-001` now has a Gateway/App
  implementation, focused tests, ECS deployment, and product-device runtime
  evidence.
- Gateway adds a separate `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true` gate,
  returns `a21.state_reactions=true` only for hardware-MAC stock Xiaozhi
  clients with `hello.features.mcp=true`, and sends bounded official MCP
  `self.robot.set_led_color` plus `self.robot.set_head_angles` reactions for
  Gateway-generated `idle`, `listening`, `thinking`, `speaking`, `error`, and
  `fatal_error` state transitions.
- State reactions are independent from touch reactions and do not require
  `/stackChan/ws` to be online, because they use the live product
  `/v1/xiaozhi` MCP socket.
- Focused tests prove listening-state MCP delivery, trace markers
  `xiaozhi.state_reaction.robot_led_color.sent` and
  `xiaozhi.state_reaction.robot_head_angles_set.sent`, redacted registry
  runtime echo, MCP-required behavior, and env wiring.
- A follow-up registry reliability guard marks closed Xiaozhi sockets as
  `connection_status=xiaozhi_ws_disconnected` while preserving the previous
  semantic `last_event`, preventing stale `/v1/devices` rows from being used
  as MCP/body-control proof.
- Commits `b6c12f0` and `f2663f1` were pushed on branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`, deployed
  to ECS `47.103.57.217`, and enabled through root-only
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true`.
- Device `44:1b:f6:e2:6a:60` was recovered with a read-only chip-id/hard-reset
  operation, not a firmware flash or NVS write. The product socket reconnected
  and accepted state-reaction MCP delivery.
- Runtime evidence report
  `reports/a21-stackchan-state-reaction-evidence-20260604-2316.json` records
  trace `a21-trace-state-reaction-wav-20260604`, session
  `a21-session-state-reaction-wav-20260604`, host WAV-triggered
  `speaking -> idle` state reactions, four audio chunks, MCP send markers,
  redacted MCP responses, and registry echo for idle LED `20/20/40` plus head
  `yaw=0,pitch=24,speed=160`.
- This is product-device runtime/body evidence, not full realtime voice or full
  PRD physical acceptance. Natural microphone-driven listen evidence, screen
  visual acceptance, richer choreography, camera, NFC, infrared, no-cable
  boot/power, and app lifecycle parity remain open.

Xiaozhi physical acceptance tooling update, 2026-06-04:

- `T-XIAOZHI-PHYSICAL-ACCEPTANCE-TOOLING-001` fixed an acceptance-tooling false
  block discovered while moving from state/body evidence to natural
  listen/speak acceptance.
- The local `xiaozhi-physical-evidence` and `xiaozhi-half-duplex` commands
  already support the TUN-safe `A21_DIRECT_SOURCE_IP=192.168.1.20` path. With
  that env set, ECS Gateway queries work from the control Mac.
- The evidence reader no longer rejects safe, unrelated device runtime echo
  booleans such as `roleplay_prompt_text_stored=false`, because the
  Xiaozhi physical report does not serialize the full runtime echo. It still
  scans consumed runtime echo, trace events, audio metadata, and instrument
  reports for prompt/transcript/provider/secret/raw-audio/locator leaks.
- Fresh live diagnosis against ECS generated a real blocked candidate instead
  of `gateway data unsafe`: device `44:1b:f6:e2:6a:60` was correctly marked
  `xiaozhi_ws_disconnected`, latest trace had `xiaozhi.listen.start` and state
  reaction markers, but no Opus frames, PCM ingress, VAD speech end, TTS
  downlink, playback ack, or audible observation.
- Next state is not more acceptance-tool plumbing. The next physical action is
  to recover the device online, trigger a real microphone listen/speak turn,
  then rerun `xiaozhi-physical-evidence`,
  `stackchan-accept --check xiaozhi-half-duplex`, and the explicit PRD review.

Xiaozhi reconnect registry repair and natural-voice blocker, 2026-06-04:

- `T-XIAOZHI-DEVICE-RECONNECT-REGISTRY-001` repaired a live regression where
  the stale-socket fix correctly marked a closed product Xiaozhi socket as
  `xiaozhi_ws_disconnected`, but a later hello/heartbeat from the same device
  did not always restore `connection_status=online`.
- Gateway now promotes active product Xiaozhi hello, heartbeat, playback,
  touch, and state-reaction activity back to `online`, and `/v1/devices`
  snapshots also treat a live Xiaozhi socket as online before applying
  freshness checks.
- Commit `5852e20` was pushed and deployed to ECS. Remote focused tests,
  remote build, service restart, and health probe passed.
- Read-only chip-id/hard-reset recovery brought device
  `44:1b:f6:e2:6a:60` back online without firmware flash or NVS write.
  Subsequent direct-source public checks showed stable `online` heartbeats.
- Natural microphone PRD acceptance remains blocked for a narrower reason:
  live evidence now finds the device online, wake/listen trace markers, and
  state-reaction MCP delivery, but still records zero audio ingress frames.
  Serial evidence shows wake detection followed by a near-immediate
  `listening -> idle` transition, before wake-word Opus packets are logged.
- The active next transition is
  `T-XIAOZHI-NATURAL-AUDIO-INGRESS-ROOTCAUSE-001`: test whether the current
  stock-physical server `type=listen` reply suppression is preventing the
  official-compatible product firmware from staying in listening/audio-upload
  state. Any change must be product-gated and default-off until physical
  evidence proves it.

Xiaozhi wake/control-channel overlay root cause, 2026-06-04:

- `T-XIAOZHI-WAKE-CONTROL-CHANNEL-OVERLAY-001` identified the actual first
  repair for natural audio ingress. The stock-physical listen-reply
  suppression hypothesis is now lower confidence because the official
  Application runtime does not consume server `type=listen` replies.
- The official-compatible overlay intended to allow wake continuation when the
  device is `Idle` and the quiet control websocket is already open. However,
  its hunk context was too weak and could match the nearby
  `ContinueOpenAudioChannel` helper instead of
  `Application::ContinueWakeWordInvoke`.
- The overlay now includes the `ContinueWakeWordInvoke` function signature in
  the hunk context, and the host test checks the exact targeted hunk. This
  prevents future builds from falsely passing while leaving the wake path in
  the old `Connecting`-only state.
- `GOMAXPROCS=2 make verify` passed. Guarded product no-flash build passed
  with report
  `reports/a21-stackchan-official-baseline-20260604-234848-1780588128571683000.json`
  and app SHA-256
  `e8880adbe7982a2e59bf58319d34097cbc16fbfcc2988975c0a19e186d32b305`.
- Build-source inspection confirms
  `ContinueWakeWordInvoke` now accepts
  `state == kDeviceStateIdle && protocol_->IsAudioChannelOpened()`.
- Product-lane flash is pending after the code/docs commit because the flash
  guard correctly refused to write hardware while tracked files were dirty.

Xiaozhi natural audio ingress physical evidence, 2026-06-04:

- `T-XIAOZHI-NATURAL-AUDIO-INGRESS-ROOTCAUSE-001` is no longer blocked at
  zero audio frames. Commit `43fcd16` was pushed, the product flash guard ran
  from a clean worktree, and the official-compatible product app flash passed
  with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-235129-1780588289127583000.json`.
- No NVS write occurred. The device reconnected through the existing product
  NVS and resumed heartbeat after `xiaozhi.hello`.
- Physical serial/runtime evidence after the flash showed wake detection,
  `idle -> listening`, AFE startup, device VAD stop, `idle -> speaking`, and
  state-reaction MCP execution.
- Live trace metrics now show natural audio ingress and answer downlink:
  listen-to-audio-ingress `114ms`, ASR first partial `207ms`, LLM first
  content `434ms`, audio downlink first frame `467ms`, TTS first audio
  `766ms`, device playback start `51ms`, and answer first audio total
  `641ms`.
- The local physical-evidence reader now treats `roleplay.prompt_input.used`
  as a safe trace marker because it proves prompt use without storing prompt
  text. Unsafe transcript/prompt/body/URL/path/raw-audio/secret checks remain.
- Physical evidence report
  `reports/a21-xiaozhi-physical-evidence-20260604-235541.527765000.json`
  is `candidate_gateway_downlink` with device online, `audio_frame_count=64`,
  Opus decode, PCM ingress, VAD speech end, TTS downlink, and playback ack.
- Half-duplex report
  `reports/a21-xiaozhi-half-duplex-acceptance-20260604-235541.288482000.json`
  remains blocked because barge-in detected/stop/stop_done and trusted audible
  observation are still missing. Next transition should focus on physical
  audible observation plus barge-in/stop_done, not on reconnect or zero-frame
  audio ingress.

Active child transitions:

- `T-WAKE-003-ZI-YUE-PHRASE-TUNING`
- `T-ASR-GREEN-LATENCY-001-XIAOZHI-LISTEN-AUTO-STOP`
- `T-ASR-GREEN-LATENCY-002-XIAOZHI-NO-SPEECH-COOLDOWN`
- `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001`
- `T-AUDIO-BARE-XIAOZHI-PARITY-001`
- `T-XIAOZHI-FULL-REALTIME-VOICE-CONVERGENCE-001`
- `T-XIAOZHI-REALTIME-VOICE-PARITY-001`
- `T-XIAOZHI-STREAMING-ASR-001`
- `T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001`
- `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`
- `T-XIAOZHI-STALE-OPUS-INGRESS-SUPPRESSION-001`
- `T-XIAOZHI-LISTEN-SPEAK-BOUNDARY-001`
- `T-DIALOGUE-001-LOW-LATENCY-CHAIN-CONVERGENCE`
- `T-ALIYUN-001-XIAOZHI-PUBLIC-VOICE-GATEWAY`
- `T-XIAOZHI-STOCK-HALF-DUPLEX-ACCEPTANCE-001`
- `T-STACKCHAN-WIFI-PROVISIONING-001-XIAOZHI-STYLE`
- `T-XIAOZHI-HOST-LOCAL-REAL-BASIC-DIALOGUE-SMOKE`
- `T-VOICE-CHAIN-EVIDENCE-001-SELECTED-VOICE-CHAIN-READINESS-INGRESS`
- `T-ROLEPLAY-IMMERSION-READINESS-001`
- `T-ROLEPLAY-VOICE-RUNTIME-EVIDENCE-001`
- `T-ROLEPLAY-VOICE-PROBE-REPORT-GENERATOR-001`
- `T-SERVER-SIDE-ROLEPLAY-VOICE-RUNTIME-GATE-001`
- `T-COSYVOICE-5080-LOCAL-CLONE-CANDIDATE-CHECK`
- `T-CLOUD-VOICE-001-PURE-CLOUD-PROVIDER-MATRIX`
- `T-ECS-STEPFUN-001-CONTROL-PLANE-AND-RUNTIME-SWITCH`
- `T-PUBLIC-GATEWAY-002-CODE-SYNC-BEFORE-STEPFUN`
- `T-STEPFUN-ROUTE-001-LAUNCH-POLICY-PROMOTION`
- `T-V21-PROFESSIONAL-EXECUTION-001`
- `T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001`
- `T-INTERNAL-TEST4-PROFESSIONAL-QUERY-ENDPOINT-001`
- `T-XIAOZHI-PHYSICAL-PRD-PROMOTE-GATE-001`
- `T-OFFICIAL-XIAOZHI-COMPATIBLE-NVS-WIFI-OVERRIDE-001`
- `T-STACKCHAN-OFFICIAL-ROBOT-MCP-BODY-CONTROL-001`

A21 has a Go-first Gateway/Core foundation, stock-compatible Xiaozhi transport,
official StackChan avatar/action relay, provider/V21 boundaries, a repo-carried
control workflow, and a freshly rebuilt official StackChan Xiaozhi-compatible
firmware candidate. The official-compatible candidate has been flashed in a
foreground hardware window, the latest firmware enters the official Xiaozhi
runtime directly, and the device has now connected to an A21 Gateway over the
stock Xiaozhi WebSocket path. A physical wake/turn produced Gateway uplink,
downlink, and barge-in candidate evidence. The user reports the audible sound
is now materially cleaner after the stock-audio hotfix, but still not at
expected maximum loudness. The 2026-06-02 emergency RCA split the problem into
official audio parity, runtime speaker-volume control, TTS source quality, and
wake-word activation rather than treating it as only loudness. The accepted
3x Gateway gain is now frozen for the contest path. `T-PROVIDER-002` has
landed host-side hot-plug code for the 5080-recommended StepFun text stream
plus Iflytek 16 kHz PCM TTS while preserving stock Xiaozhi firmware/protocol
boundaries; commit `b5a405e` is the current integration baseline. The old
`sherpa_onnx` local voice is now only an emergency/diagnostic fallback for this
contest path; `voice_clone_cli` remains the retained A21 voice-clone seam for
IndexTTS2/CosyVoice/F5-TTS/GPT-SoVITS and must not be removed just because the
immediate fast-dialogue path selects Iflytek. The 2026-06-03 control-tower
worker fan-out refreshed selected-provider readiness without falling back to
`mock`, confirmed the StepFun+Iflytek relay chain remains voice-chain candidate
evidence, and separated the no-flash normal-dialogue observation path from the
optional diagnostic half-duplex counter path. The no-flash self-trigger
observation now has candidate evidence, while custom wake and clone-capable
local TTS remain explicit recovery tasks rather than hidden blockers.
The 2026-06-03 internal test 3 closure packages the public-Gateway voice
main-chain breakthrough at HEAD `074e3d877d33` into
`dist/a21-internal-test3-20260603-233245`. Fresh verification passed
`make verify`, host preflight/doctor with firmware/wake warnings, packaged
binary host gate with A21 direct `NO_PROXY`, public Gateway health/profile/OTA
checks, and public host-loopback voice bench. The product StackChan
`44:1b:f6:e2:6a:60` was online on `ws://47.103.57.217/v1/xiaozhi`; remote
runtime snapshot shows `cascade` with ASR `dashscope_qwen_asr_realtime`, LLM
`deepseek`, fixed TTS `dashscope_qwen_tts_realtime`, hot switch enabled, and
`stepfun_not_selected` still present. Machine-readable physical evidence is
still `candidate_gateway_downlink`, so this is an internal voice-main-chain
release rather than full PRD launch readiness. The detailed recovery surface
for this long control thread is
`docs/handoffs/2026-06-03-a21-internal-test3-master-handoff.md`; read it before
continuing Gateway, provider, firmware, or physical StackChan work from this
state.
The 2026-06-04 full-launch protocol-adaptation sprint landed locally and
passed the unified verification surface. Gateway stock `/v1/xiaozhi`
regressions now pin product message order and suppressed post-answer listen
drain behavior; readiness/server-side readiness ingest voice-chain selector
state and block on `stepfun_not_selected`; physical evidence matching rejects
stale or mismatched device/trace/session targets and does not over-credit
Gateway downlink as audible playback; doctor/preflight now recognize executed
official-compatible product-lane flash evidence without weakening firmware
guards; provider selector readiness distinguishes StepFun launch selection
from DeepSeek fallback using env-name-only reporting and a redacted remote
switch runbook. Local verification passed `make verify`, `make preflight`, and
`make doctor`; the only host warning is still
`wake_word_firmware_build_required`. Public Gateway retry on 2026-06-04
02:13 CST recovered `http://47.103.57.217/healthz`, `/v1/devices`,
`/v1/voice-chain-profiles`, `/v1/gateway-profiles`, and `/xiaozhi/ota/`, with
the product device `44:1b:f6:e2:6a:60` online. The live selector still reports
selected LLM `deepseek` and finding `stepfun_not_selected`, and SSH from this
control machine is blocked by `Permission denied (publickey)`. The next
runtime transition is
`docs/plans/2026-06-04-ecs-control-plane-and-stepfun-switch.md`; do not blind
POST a StepFun hot switch until remote StepFun env-name presence is verified.
The follow-up ECS control-plane check succeeded with an explicit existing local
SSH identity: `a21-gateway` and Caddy are active, `127.0.0.1:21081/healthz`
returns ok, and public `80` remains the product entrypoint. The root-only
remote provider env file exists with owner `root:root` and mode `600`, but
`A21_LAB_STEPFUN_API_KEY` and `A21_STEPFUN_MODEL` are missing. Therefore the
StepFun runtime switch is now blocked by env provisioning rather than SSH. Do
not switch `/v1/voice-chain-profiles` to `stepfun` until an approved operator
injects the missing A21 env names on ECS and restarts/validates
`a21-gateway`.
Before secret provisioning, the control tower also found remote binary drift:
the current ECS `/opt/a21/bin/a21` still reports StepFun static readiness as
passed even when the direct env-name check shows required StepFun env names are
missing. Therefore
`T-PUBLIC-GATEWAY-002-CODE-SYNC-BEFORE-STEPFUN` must deploy the locally verified
readiness/provider-selector code to ECS before the StepFun selector switch.
This sync must not edit secrets, execute providers, flash firmware, write NVS,
or POST the selector to StepFun.
The current control thread then re-read the master handoff and live runtime
before continuing: public `/v1/voice-chain-profiles` reported selected LLM
`stepfun`, `/v1/devices` showed product StackChan `44:1b:f6:e2:6a:60` online on
the selected StepFun cascade chain, and host bench
`reports/a21-xiaozhi-voice-bench-20260604-023616.742713000.json` executed the
cloud-edge DashScope ASR + StepFun LLM + DashScope TTS path. The remaining
server-side blocker was no longer `stepfun_not_selected`; it was that the
executed StepFun provider-smoke report
`reports/a21-provider-smoke-20260604-023711-678466985.json` was produced before
StepFun was promoted and therefore had `route_eligible=false`. Active
transition `T-STEPFUN-ROUTE-001-LAUNCH-POLICY-PROMOTION` promoted the built-in
StepFun profile to explicit route-eligible launch-policy status without
changing internal test 3 `/v1/xiaozhi` protocol behavior, firmware flash state,
or endpoint-side voice acceptance evidence.

The 2026-06-04 04:02 CST recovery resolved the ECS blocker. The control thread
used the approved jump path to deploy `d9362a7 feat(readiness): accept
cloud-edge xiaozhi evidence` to ECS through the existing `/opt/a21.next`
safe-swap, kept `/etc/a21/secrets/provider.env` root-owned and mode `600`, and
set only A21 profile IDs/selectors needed for the StepFun cascade. Fresh ECS
evidence now shows StepFun provider smoke passed with `executed=true`,
`stream=true`, `route_eligible=true`, static Xiaozhi provider readiness passed
with `stepfun_selected`, and the 15s cloud-edge Xiaozhi host bench passed as
`candidate_host_only` with `failure_count=0`. Product readiness and
server-side readiness now absorb provider evidence and cloud-edge host voice
evidence; they remain blocked only by `v21_professional_execution` and
`physical_stackchan_prd_acceptance`. Full PRD remains blocked until physical
playback ack, stop_done, or trusted audible/instrument observation is present.

The control Mac still receives empty HTTP replies when directly curling
`47.103.57.217`, while 5080lab, ECS loopback, Caddy, and Gateway health are
normal. ECS tcpdump did not observe the later Mac curl attempt reaching the
host, so this is tracked as a source-path/network issue rather than an A21
runtime blocker. Do not use the Mac direct-curl symptom to invalidate the fresh
ECS/5080lab runtime evidence, and do not claim launch readiness without the
remaining V21 and physical evidence gates.
`T-V21-PROFESSIONAL-EXECUTION-001` is now the active server-side transition
after the StepFun cloud-edge evidence cut. The current control shell has no
`A21_V21_ADAPTER_URL`, no `A21_V21_BACKEND_URL`, no `A21_V21_ADAPTER_TOKEN`,
and no local `127.0.0.1:21121` listener. Historical V21 reports from
2026-06-01/02 must not be used to close the current gate. The plan is
`docs/plans/2026-06-04-v21-professional-execution-validation.md`; it allows a
fresh redacted adapter-boundary smoke only after the boundary is configured and
keeps physical StackChan PRD acceptance separate.
Fresh control-shell reports
`reports/a21-v21-adapter-smoke-20260604-041657.json`,
`reports/a21-product-readiness-20260604-041710.json`, and
`reports/a21-server-side-readiness-bundle-20260604-041710.json` are
boundary-missing/operator-ask evidence only. They must not supersede the 04:02
ECS StepFun/cloud-edge provider and host-voice evidence.
The user then supplied V21 control thread
`codex://threads/019e68bc-4fb6-7ce0-ad67-5b1dd0de478f`. That thread confirmed
V21 is a local/LAN Docker Compose service. Docker Desktop was started, the V21
LAN demo backend was brought up on `18081`, and A21 temporarily bridged
`127.0.0.1:21121` to `127.0.0.1:18081`. Fresh adapter evidence
`reports/a21-v21-adapter-smoke-20260604-043456.json` passed with
`configured=true`, `executed=true`, `redaction_ok=true`, evidence count `5`,
speech count `1`, screen-card count `1`, and follow-up count `1`. Fresh
readiness reports
`reports/a21-product-readiness-20260604-043528.json` and
`reports/a21-server-side-readiness-bundle-20260604-043528.json` now mark
provider, V21, and host voice evidence ready in the local control context; they
remain blocked by local Gateway/wake/selector and physical PRD evidence. This
closes V21 adapter-contract execution evidence, not permanent ECS Gateway V21
topology.
`T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001` is now a passed real-model no-audio
smoke: the repo-local canonical helper, sherpa-onnx Python environment, and
streaming Zipformer model cache were discovered automatically, the JSONL
helper loaded the real model, appended one in-memory 60 ms PCM16 frame, and
committed a final event without writing WAV, recording transcript text, storing
raw audio, calling providers/V21, starting Gateway, or touching hardware.
`T-STREAMING-TTS-RUNTIME-PROOF-001` is now recorded as a redacted runtime smoke
tool with truthful no-execute blocker: fake realtime tests prove provider audio
delta to exact 60 ms PCM16 mono chunking before any WAV/file boundary, while the
local default report records `execute_flag_required` and no real provider
runtime pass is claimed.
`T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001` is now a host-side bridge
candidate: stock `/v1/xiaozhi` streaming ASR partial events can start the
workmate LLM/TTS streaming answer before explicit `listen.stop` or `asr.final`,
while the existing final-transcript and batch ASR fallback paths remain
available. This is unit/Gateway/parity evidence only, not real Sherpa runtime,
real provider execution, or physical PRD acceptance.
`T-XIAOZHI-REALTIME-PARITY-REAL-PROFILE-EVIDENCE-001` now hardens the
realtime parity gate against a subtler false green: a trace with good event
ordering but missing real streaming ASR/LLM/TTS profile-class markers is
downgraded from `xiaozhi_realtime_candidate` and receives
`xiaozhi_realtime_real_profile_evidence_missing`. Gateway records only
category markers such as `llm.mock_blocked`, `tts.file_boundary_blocked`, or
`tts.real_streaming`; it does not store raw provider names, transcripts,
provider outputs, URLs, credentials, paths, or audio payloads.
`T-XIAOZHI-NONBLOCKING-ASR-COMMIT-001` now moves `listen.stop` and VAD
auto-stop off the blocking ASR commit path: streaming ASR commit/finalization
runs asynchronously, the WebSocket read loop can process abort/barge-in while
commit is pending, and final-driven voice-pipeline startup uses the streaming
ASR final without falling back to batch `Transcribe`.
`T-XIAOZHI-OFFICIAL-PROTOCOL-SOURCE-READ-AND-NEXT-CUT-001` is now a
host-local stock-protocol fidelity cut: A21 emits stock `stt` before `tts`
when streaming ASR partial/final text exists, and buffers a bounded idle
wake pre-roll of decoded Opus frames so wake-adjacent audio can feed the next
`listen/start` turn instead of being dropped. This remains below PRD
acceptance because it is unit/Gateway evidence only, with no real provider
execution, physical stock trace, wake/tap proof, barge-in/touch evidence, or
audible playback.
`T-XIAOZHI-OPUS-INGRESS-QUEUE-001` now adds the missing host-side Opus frame
queue boundary: listening Opus frames are enqueued on a bounded per-session
queue before decode/VAD/streaming-ASR append, so the stock WebSocket control
loop can still read `abort` while ASR append is slow. `listen.stop`
finalization now waits asynchronously for queued ingress to catch up before
committing streaming ASR or starting the voice pipeline. This is still
host-local evidence, not physical Xiaozhi PRD acceptance.
`T-XIAOZHI-STALE-OPUS-INGRESS-SUPPRESSION-001` now closes the next queue
ownership gap: Opus ingress items whose turn context has already been cancelled
are suppressed before decode, audio ingress, VAD, or streaming ASR append. This
keeps abort/wake-as-abort/listen-start barge-in from letting old queued audio
pollute the next listening turn. It is unit/Gateway evidence only, not real
provider execution or physical PRD acceptance.
`T-XIAOZHI-LISTEN-SPEAK-BOUNDARY-001` is now the active hotfix for the
post-wake "speaks before the user finishes" physical symptom. A21 no longer
lets ASR partial text start an audible Xiaozhi answer: partials now record
`xiaozhi.voice_pipeline.partial_prewarm_deferred` and keep the device in
listening until device `listen.stop`, trusted turn end, or max-duration safety
stop closes the turn. For stock physical MAC devices, Gateway-side RMS
`vad.speech.end` is also deferred and no longer triggers
`xiaozhi.listen.auto_stop`, because the two-frame RMS hangover can mistake a
natural short pause for speech end. This preserves streaming ASR partial
observability without allowing pre-stop `tts.first_audio` or Opus downlink.
`T-DIALOGUE-001-LOW-LATENCY-CHAIN-CONVERGENCE` is preserved as the internal
test 3 low-latency spoken-chain history. Internal test 4 supersedes its
user-facing mode contraction: product modes now converge to `roleplay` and
`professional`, while `dialogue`, `workmate`, `companion`, and `co_creation`
remain compatibility aliases or playbook labels under `roleplay`. The
underlying low-latency Xiaozhi/ASR/text/TTS chain remains intact, and
`professional` stays on the V21 adapter boundary with `prd_accepted=false`.
`T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001` is active as the
mode-contract v2 and cloud knowledge workspace transition. The plan is
`docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`.
Its first cut makes `roleplay` the default user-facing `voice_mode`, keeps
`dialogue` as a backwards-compatible alias, leaves internal test 3 Xiaozhi
audio/barge-in behavior untouched, and now exposes
`GET/POST/PUT /v1/roleplay-profile` for roleplay scenario plus voice-clone
runtime selection. Fast companion returns only a redacted roleplay runtime
summary and traces roleplay readiness without storing prompt text, memory text,
transcripts, provider output, voice samples, or V21 evidence. The A21
Cloud/Web/App plus V21 Knowledge Service/Adapter roadmap now has a Gateway
contract entrypoint: `GET/POST/PUT /v1/professional-workspace` selects redacted
`user_id`, `workspace_id`, and `query_scope`, and professional V21 turns send
those v2 scope fields to the A21/V21 adapter. Upload/import/index execution,
durable account binding, and personal corpus enforcement remain planned work.
Gateway also now exposes `GET/POST/PUT /v1/workspace-upload-jobs` for
no-execute upload/import/index job metadata: create, poll, mark failed, retry,
and delete. This proves the workspace lifecycle API shape without storing
document text/bytes or executing V21 indexing.
Professional workspace reads now also produce a memory-only safe read ledger at
`GET /v1/professional-read-records`, and the simulator Workspace Audit surface
shows only upload/read metadata for operator checks. Explicit user-spoken
professional triggers such as "专业模式", "认真查一下", "帮我查 V21", and
"给我证据" now route default/roleplay/workmate/companion mock and stock
Xiaozhi turns into the professional evidence path, while negated phrases and
privacy/state modes remain out of V21. Stock Xiaozhi trigger routing reuses the
streaming ASR final instead of running duplicate batch ASR.
Workspace upload/import metadata now also creates memory-only source records at
`GET /v1/workspace-sources`, with public/personal source-scope counts,
`metadata_only`, `searchable_metadata_only`, `failed_metadata_only`, and
`deleted_metadata_only` readiness, plus professional workspace
`query_scope_readiness`. That source-readiness cut was below real upload
storage, real indexing, V21 ACL enforcement, and physical hardware acceptance.
Workspace document upload intake now adds `POST /v1/workspace-documents`, which
accepts multipart file bytes into an A21 Gateway local runtime store and links
the stored document to the same job/source registry as
`stored_local_pending_index` with `storage_status=stored_local` and
`index_status=not_started_no_execute`. `/v1/professional-workspace` can show
that pending-index readiness for the selected scope while still keeping
`v21_execution_allowed=false`. This is local intake only: it does not parse,
chunk, embed, index, upload to cloud storage, execute V21, or store documents
on StackChan.
`T-INTERNAL-TEST4-PROFESSIONAL-MODE-RITUAL-001` gives `/v1/voice-modes` a
selected professional ritual contract: `PRO` screen label, evidence-first cue,
professional expression, `professional.checking_feedback.sent` trace marker,
`professional_only` workspace policy, `v21_allowed=true`, and
`physical_accepted=false`. The simulator shows the selected mode cue, while
fast-companion remains blocked for selected professional mode.
`T-ALIYUN-001-XIAOZHI-PUBLIC-VOICE-GATEWAY` is active as the main public
transport profile slice: with a valid `A21_PUBLIC_GATEWAY_URL`, `public_wss`
is the default product voice-edge Gateway, while `mac_local` remains
selectable for Mac-local models and local processing. Gateway exposes
`/v1/gateway-profiles`, simulator can select the profile, and
`/xiaozhi/ota/` returns either request-host local `ws`/`wss` or configured
public `ws`/`wss` `/v1/xiaozhi`. The earlier Aliyun SWAS host `101.132.117.182`
proved the experimental chain under systemd/nginx with self-signed `443`, but
the new ECS `47.103.57.217` / `i-uf63f4ymqc2dxtljxz2n` is now running the main
A21 voice-edge Gateway under systemd on `127.0.0.1:21081` behind Caddy public
`80/443`. External HTTP health/profile/OTA checks pass, OTA returns
`ws://47.103.57.217/v1/xiaozhi`, and temporary self-signed HTTPS passes only
with `-k`. Trusted TLS, secure provider-secret injection, real cloud
ASR/LLM/TTS execution, physical StackChan public-WSS evidence, and wake-word
product proof remain open before PRD acceptance.
The 2026-06-03 public-edge provider push moved this slice beyond host-only
reachability: ECS `47.103.57.217` now reads provider configuration from
root-only `/etc/a21/secrets/provider.env`, runs a DashScope CosyVoice wrapper
through A21's existing `voice_clone_cli` TTS seam, and delivered real
DashScope-generated audio to the physical StackChan over stock
`/v1/xiaozhi`. Evidence trace
`a21-trace-public-edge-dashscope-say-1780472526` returned
`status=delivered`, `delivered_transport=xiaozhi_ws`, `audio_chunks=87`,
`tts.first_audio=2159 ms`, `audio.downlink.first_frame=2160 ms`, and
`xiaozhi.say.delivered` for device `44:1b:f6:e2:6a:60`. The matching remote
TTS smoke `/tmp/a21-provider-smoke/a21-local-tts-smoke-20260603-154156.json`
passed with `provider=voice_clone_cli`, `model=cosyvoice_v3_flash`, and a
redacted PCM16 mono WAV quality report. Doubao realtime TTS and the attempted
Iflytek mapping remain blocked by provider `401`; their credentials must not
be treated as A21-ready Realtime/Iflytek credentials. Public host-loopback
bench report `reports/a21-xiaozhi-voice-bench-20260603-154310.624051000.json`
proved virtual answer and abort timing (`barge_in_stop_p95_ms=1`) but stayed
`prd_accepted=false`. Physical half-duplex report
`reports/a21-stackchan-half-duplex-acceptance-20260603-154254.json` correctly
blocked because the stock product firmware is not the diagnostic mic-probe
runtime echo lane. Full PRD remains blocked by operator audible confirmation,
stock physical mic-driven dialogue, normal half-duplex/barge-in proof, and
wake-word product proof.
`T-XIAOZHI-STOCK-HALF-DUPLEX-ACCEPTANCE-001` is active as the product-firmware
half-duplex evidence correction. A new `stackchan-accept --check
xiaozhi-half-duplex` gate now reads stock `/v1/xiaozhi` Gateway traces and
`/v1/audio/recent`, can derive the latest trace/session from `/v1/devices`, and
writes `a21.xiaozhi_half_duplex_acceptance.v1` reports with
`hardware_acceptance_scope=stock_xiaozhi_mic_to_tts_downlink` and
`diagnostic_mic_probe_required=false`. It does not require the diagnostic
mic-probe capability or runtime echo counters, so product stock Xiaozhi
firmware is no longer misclassified by the older diagnostic half-duplex gate.
The first public run against `47.103.57.217` produced
`reports/a21-xiaozhi-half-duplex-acceptance-20260603-161717.013784000.json`:
device `44:1b:f6:e2:6a:60` was online with stock Xiaozhi transport, but the
latest trace was hello-only with zero audio frames, no downlink, no playback
ack, and no barge-in stop evidence, so status correctly remained `blocked` and
`prd_accepted=false`.
The 2026-06-03 post-rebuild/no-sound audit found that the endpoint and
Gateway voice changes were not broadly erased by the clean Xiaozhi refresh:
the current HEAD still contains the wake/listen/VAD, Opus queue, stale ingress
suppression, public Gateway, DashScope downlink, and stock half-duplex
acceptance commits; the latest clean product build/flash still used
`a21-stackchan-official-xiaozhi-compatible.bin` with custom `紫悦` wake config
and A21 control-channel guards. The current runtime break is on the public
Gateway product chain: ECS `47.103.57.217` is still launched with
`--product-chain host_local`, has TTS/text env keys but no ASR env, defaults
missing ASR to `A21_ASR_LOCAL_PROFILE=sherpa_onnx`, and therefore drives real
physical `/v1/xiaozhi` turns into `xiaozhi.voice_pipeline.unavailable` and
`local_fallback`. This keeps physical dialogue and audible acceptance blocked
until the public Gateway has a cloud-compatible ASR/LLM/TTS chain and a fresh
physical trace proves real provider execution.
`T-STACKCHAN-WIFI-PROVISIONING-001-XIAOZHI-STYLE` is active as the endpoint
provisioning contract: A21 no longer treats missing Wi-Fi credentials in the
self-owned firmware state model as local fallback; it enters
`Wi-Fi provisioning` and exposes Xiaozhi's three startup provisioning methods
as `hotspot`, `blufi`, and `acoustic`, with `hotspot` as default. The
official-compatible product overlay now explicitly selects Hotspot/SoftAP
provisioning, keeps BluFi and acoustic disabled but named as build-time
alternatives, points OTA at the main public Gateway
`http://47.103.57.217/xiaozhi/ota/`, and still stores no provider keys or Wi-Fi
credentials in repo/firmware. The product lane was rebuilt no-flash with the
current overlay and produced
`/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
with app SHA-256
`c012542ee8f1837106da91fe934e27487813333eaad124841386706259f48659`; report
`reports/a21-stackchan-official-baseline-20260603-150433-1780470273395485000.json`
is `status=passed`, `official_avatar_action_preserved=true`, and
`official_xiaozhi_start_preserved=true`. This is still below physical
acceptance until flashed and observed through a guarded hardware window.
`T-CLOUD-VOICE-001-PURE-CLOUD-PROVIDER-MATRIX` is active as a side-branch
research/control transition for the newly deployed public Gateway era. It
creates the provider-neutral contract for fully supporting Bailian Qwen-TTS
Realtime and CosyVoice, Doubao realtime/clone voice families, and MiniMax TTS
and clone families without changing A21 defaults. The contract keeps
`cloud_voice_profile` separate from `voice_mode` and `gateway_profile`, keeps
`professional` on the V21 adapter path, and requires frontend/operator
selection, server-side dispatch, A21 latency reports, redacted provider smoke,
and physical StackChan evidence before any cloud voice profile can become
accepted. The side branch now also implements the first no-execute control
surface: `GET/POST /v1/cloud-voice-profiles`, simulator selector/readout,
device-registry `current_cloud_voice_profile`, and `a21 doctor`
`voice.cloud_voice` report. The implemented surface returns only safe profile
IDs, statuses, capabilities, present env names, and missing env names; it does
not execute providers, does not change `voice_mode` or `gateway_profile`, and
does not expose provider keys, model values, voice IDs, URLs, prompt text,
transcripts, or audio payloads.
The fixed
official codec output-volume candidate is already prepared in the repo-owned
Xiaozhi-compatible overlay. The no-write
official-compatible build/report passed with app SHA-256
`2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`.
The matching candidate was flashed through an operator-approved foreground
hardware window on `/dev/cu.usbmodem1101`; flash report
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`
is `status=passed`, `dry_run=false`, `flash_allowed=true`, and
`flash_executed=true`. Gateway `127.0.0.1:21081` remained healthy after flash
and the physical device `44:1b:f6:e2:6a:60` remained registered online. Physical
audible A/B for the volume-92 candidate failed: operator recording
`/Users/jiyurun/Downloads/军民公路259号 7.m4a` measured `-33.4 LUFS` integrated
loudness and `-14.3 dBFS` true peak, materially weaker than the 22:19 reference
sample at `-25.8 LUFS` and `-4.2 dBFS`. Analysis report
`reports/a21-operator-recording-audio-analysis-20260602-2310-stackchan-volume92.json`
sets `post_flash_loudness_accepted=false`. The follow-up operator recording
`/Users/jiyurun/Downloads/军民公路259号 8.m4a`, analyzed from 3 seconds onward,
improved to `-31.78 LUFS` integrated and `-10.56 dBTP` true peak versus
`7.m4a` from 3 seconds at `-35.56 LUFS` and `-14.65 dBTP`, but still leaves
roughly 10 dB peak headroom. Host hotfix tests now pass for
stock MCP speaker-volume delivery, stock physical `listen` reply suppression,
24 kHz downlink TTS chunks, turn-level Opus encoder reuse, foreground
`/v1/xiaozhi/say` delivery, Gateway playback, voice-bench, and
app/gateway/provider/transport focused paths. The live Gateway on
`127.0.0.1:21081` delivered runtime volume `100` through stock MCP and pushed a
long physical TTS turn through `/v1/xiaozhi/say` with trace
`a21-trace-live-long-tts-hotfix`, `text_chars=189`, and `audio_chunks=676`.
Explicit
real/local provider evidence exists, but
generic readiness refreshes can still fall back to `mock` if they omit the
selected provider report/env. It is not yet full PRD accepted because audible
playback observation, broader half-duplex acceptance, and custom wake proof
remain missing. Follow-up recording
`/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a` after bounded Gateway
downlink gain measured `-21.65 LUFS` and `-4.72 dBTP` from 3 seconds, with no
obvious clipping in FFmpeg `astats`, but the operator later reported that this
4x gain candidate felt like a slight regression with subtle electrical
interruption and reduced clarity. Follow-up recording
`/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 3.m4a` measured
`-26.4 LUFS` and `-9.0 dBFS` true peak from 3 seconds; the active code
candidate now uses a safer 3x max-gain cap while preserving the same stock
protocol, 24 kHz downlink, Opus reuse, and host-say suppression. The 3x code
was relaunched in tmux Gateway session `a21-gateway-21081`, runtime volume
`100` was delivered by trace `a21-trace-stackchan-volume-1780417211`, and
physical `/v1/xiaozhi/say` trace `a21-trace-stackchan-say-1780417217`
delivered `text_chars=176`, `audio_chunks=579`, `answer_first_audio_total_ms=1656`,
`xiaozhi.say.input_suppression_armed=1`, and
`xiaozhi.listen.start.input_suppressed=1`. Operator acceptance recording
`/Users/jiyurun/Downloads/军民公路259号 10.m4a` measured `-26.5 LUFS` and
`-8.6 dBFS` true peak from 3 seconds, and the operator explicitly accepted the
3x audio result. Audio quality is accepted for the current contest path, while
normal dialogue half-duplex remains a separate follow-up and full PRD launch
acceptance is still blocked by custom wake proof. Custom wake-word package
evidence exists, but the 2026-06-03 foreground hardware window proved an
important negative guard: flashing the restored package's bare `xiaozhi.bin`
through `xiaozhi-firmware-flash-execute` overwrote the StackChan avatar/product
app with a plain Xiaozhi UI. That incident was rolled back immediately by
flashing the correct `a21-stackchan-official-xiaozhi-compatible.bin` product
candidate on `/dev/cu.usbmodem1101`; restore flash report
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`
is `status=passed`, `flash_allowed=true`, and `flash_executed=true`.
Post-restore Gateway `127.0.0.1:21081` saw a fresh physical hello from
`44:1b:f6:e2:6a:60`; runtime volume `100` delivered on trace
`a21-trace-recovery-stackchan-volume-1780424386012`; the accepted 5080
StepFun+Iflytek relay WAV delivered through stock `/v1/xiaozhi/say` on trace
`a21-trace-recovery-stackchan-relay-wav-1780424386012` with
`audio_chunks=40`. `T-WAKE-002` rebuilt custom wake inside the
StackChan-compatible product lane instead of the bare `xiaozhi.bin` lane:
plan `docs/plans/2026-06-03-zi-yue-wake-stackchan-compatible.md`; build report
`reports/a21-stackchan-official-baseline-20260603-030428-1780427068064697000.json`
is `status=passed`, app SHA-256
`3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`, with
`official_avatar_action_preserved=true`,
`official_xiaozhi_start_preserved=true`, and `minimal_bridge_screen=false`.
Build config proves StackChan board identity, `USE_CUSTOM_WAKE_WORD=true`,
`CUSTOM_WAKE_WORD="zi yue"`, `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`, threshold
`20`, `SR_MN_CN_MULTINET7_QUANT=true`, `USE_AFE_WAKE_WORD=false`, and stock
HiStackChan WakeNet disabled. The overlay intentionally uses a contest-recovery
direct autostart: it sets codec volume and calls `GetHAL().startXiaozhi()`
before the welcome/setup app flow, because the official request path trapped the
physical device on the welcome screen. Retaining official app loading without
showing welcome/setup is now a follow-up transition. No-write flash plan
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030447-1780427087258380000.json`
was clean for `/dev/cu.usbmodem1101`. The first post-commit flash execute was
blocked by the T7 guard because the welcome-recovery hotfix had made the
worktree dirty; commit `7f3225e` then restored a clean control state. Guarded
flash execute
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030818-1780427298506934000.json`
passed with app `a21-stackchan-official-xiaozhi-compatible.bin`, app offset
`0x20000`, app SHA-256
`3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`, and
control commit `7f3225ee1fe7`. Gateway `127.0.0.1:21081` stayed healthy; the
physical device `44:1b:f6:e2:6a:60` reconnected online at 03:08:51 with trace
`a21-trace-44-1b-f6-e2-6a-60` and speaker volume `100`. Physical wake proof is
now rejected by the operator: saying `紫悦` produced no response. A separate
green-ASR latency problem is also active: live trace reuse showed multiple long
stock Xiaozhi listen windows, including about 25s, 38s, 70s, and 108s before
auto-stop or replacement by a new listen. The current unflashed hotfix candidate
keeps the display identity `紫悦`, adds longer MultiNet command aliases, adds a
Gateway max-listen safety stop with `A21_XIAOZHI_LISTEN_MAX_MS`, and improves
trace summary pairing for reused hardware trace ids. Commit `5242349` was
built and flashed through the official-compatible product lane: build report
`reports/a21-stackchan-official-baseline-20260603-032615-1780428375746120000.json`;
no-write flash plan
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-032627-1780428387219884000.json`;
flash execute
`reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-032734-1780428454747199000.json`;
app SHA-256
`e13a6cbd63596cd1388f5a540b488b3e35b449e49a5e31891cf133436561dc6e`.
Gateway `a21-gateway-21081` was restarted from this commit with
`A21_XIAOZHI_LISTEN_MAX_MS=7000`, device `44:1b:f6:e2:6a:60` reconnected
online, and runtime speaker volume `100` was delivered on trace
`a21-trace-wake-asr-hotfix-volume-1780428544`. `T-FLASH-GUARD-001` is now integrated: the
generic `xiaozhi-firmware-flash-*` lane rejects product-looking `xiaozhi.bin`
at app offset `0x20000` unless explicitly marked `--non-product-dev`, while
the official-compatible product flash plan still passes.

Current control branch:

- `codex/a21-hardware-window-20260602-stackchan-prd`

Current notable baseline:

- `ebc0db4 docs(control): record stackchan flash convergence`
- `18dc553 docs(control): prepare stackchan volume flash gate`
- `f0603f5 fix(firmware): set official xiaozhi codec volume`
- `4613946 fix(firmware): enter official xiaozhi runtime directly`
- `eeacbd3 docs(control): recover hardware network state`
- `e7e9b03 feat(firmware): autostart official xiaozhi candidate`
- `37ef8f3 feat(firmware): add official xiaozhi nvs connection config`
- `6f34091 feat(firmware): add official xiaozhi compatible flash plan`
- `d28560a docs(control): dispatch stackchan volume worker`
- `f49abde docs(audio): plan stackchan volume control`
- `751de08 docs(audio): record stock xiaozhi playback boundary`
- `59f30f4 docs(control): record official flash seam worker dispatch`
- `69c4bbe docs(control): add handoff and state machine workflow`
- `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`
- `060d2bb fix(audio): accept stackchan xiaozhi playback hotfix`
- `b5a405e feat(audio): add hot-pluggable iflytek tts candidate`

## Module States

| Module | State | Evidence | Next state |
| --- | --- | --- | --- |
| Control workflow | `S1-REPO-CARRIED-CONTROL` | Commit `69c4bbe`; `docs/agent_handoff_log.md`, `docs/project_state_machine.md`, and `docs/plans/` exist | `S2-WORKER-TRANSITION-OPERATING` |
| Gateway `/v1/xiaozhi` | `S3-PHYSICAL-DEVICE-CONNECTED` | Gateway on `127.0.0.1:21081` / LAN port `21081` accepted the physical device via stock Xiaozhi WebSocket; trace `a21-trace-44-1b-f6-e2-6a-60` has Opus uplink, VAD, ASR final, provider first content, TTS first audio, and Opus downlink | `S4-AUDIBLE-PLAYBACK-ACCEPTED` |
| Official StackChan avatar/action relay | `S2-HOST-READY` | Gateway/transport mapping exists for official StackChan packets | `S3-FLASHED-OFFICIAL-CANDIDATE` |
| Firmware candidate | `S5N-NO-WELCOME-IDLE-SOCKET-CANDIDATE` | Commit `a986d6b` request-start physical flash regressed to the welcome/setup trap. Hotfix commit `9ba8bc1` moved the direct Xiaozhi start before visible setup, and `e694550` parked `app_main` after `GetHAL().startXiaozhi()` because `startXiaozhi()` returns and otherwise falls into the Mooncake welcome/setup loop. After the operator reported a welcome regression again, the control tower rebuilt the current HEAD `7906975` product app through the guarded official-compatible lane. Build report `reports/a21-stackchan-official-baseline-20260603-062637-1780439197207182000.json` passed with app SHA-256 `fc7788736ced71c98cee846892a867d306d663c5ceb31014cc01079a381e766c`; flash plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062715-1780439235852990000.json` was ready; guarded flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062853-1780439333539408000.json` passed on `/dev/cu.usbmodem1101` from clean commit `790697518c3e`. Device `44:1b:f6:e2:6a:60` is online on Gateway `21081`; `/v1/devices` reports `last_event=xiaozhi.hello`, `speaker_volume=100`, and trace `a21-trace-44-1b-f6-e2-6a-60`. The operator later confirmed setup is fixed. Recovery trace analysis found the old touch-start episode had only one `xiaozhi.listen.start` and one `xiaozhi.listen.stop`, with `xiaozhi.no_speech.input_suppression_armed=1`, `xiaozhi.listen.start.input_suppressed=1`, and `xiaozhi.listen.start.suppressed_after_no_speech=1`; the latest trace event is `xiaozhi.hello.received`, so current state is idle socket connected rather than infinite listening. Runtime speaker volume `100` was re-delivered on trace `a21-trace-recovery-volume-1780441733`. Physical wake and touch retry remain pending. | `S5O-WAKE-AND-TOUCH-PHYSICAL-ACCEPTED` |
| Device connection/NVS | `S3-LAN-GATEWAY-CONNECTED` | Latest guarded NVS execution `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260602-212542-1780406742553040000.json` pointed OTA/WS to the LAN-bound A21 Gateway; reset serial log shows OTA connection to `21081` and activation | `S4-STABLE-RECONNECT-EVIDENCE` |
| Provider hot-plug | `S4A-SELECTED-PROVIDER-READY-VOICE-CHAIN-CANDIDATE` | Read-only worker `019e88dd-3efa-78e2-a7d3-7063089cbf30` confirmed usable explicit provider evidence: DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json` passed with `executed=true`, `stream=true`, `repeat=5`, `route_eligible=true`, and `first_content_p95_ms=745.516`; local Ollama smoke `reports/a21-provider-smoke-20260602-075644-199710000.json` passed when selected explicitly; readiness `reports/a21-product-readiness-20260602-160841.json` has `provider.real_provider_ready=true` and `server_side.provider_evidence_ready=true`. 5080 report `outbox/A21-VOICE-FULL-REPORT.md` was read through the established `5080lab` outbox lane and recommends StepFun `step-1-8k` for real-time text stream plus Iflytek TTS for real-time 16 kHz PCM synthesis. The source report contained plaintext credentials, so repo docs/logs record only env names and redacted provider identity. Current code lets explicit StepFun/compatibility text-stream profiles run without promoting product route eligibility, exposes `iflytek_tts` through CLI and voice-pipeline TTS selection, and preserves `voice_clone_cli` as the voice-clone path. Worker `019e8954-e65c-72a2-9847-a17a59a0ad6b` unblocked the host chain through 5080 relay: Iflytek TTS report `reports/provider-tts-candidate/a21-local-tts-smoke-5080-relay-20260603-0125.json` passed with first audio `100.299 ms`, and StepFun+Iflytek chain report `reports/provider-tts-candidate/a21-local-voice-loopback-5080-relay-20260603-0128.json` passed with text first content `229.077 ms` and TTS first audio `87.947 ms`. After deploying the new Gateway handler, relay WAV playback through stock `/v1/xiaozhi/say` delivered `audio_chunks=40` on trace `a21-trace-provider-playback-wav-1780450901`; operator feedback was positive: "好多了". Worker `019e8974-346e-7912-93b2-77cdbb9f3acf` then refreshed selected-provider readiness with route-eligible DeepSeek evidence: `reports/provider-tts-candidate/a21-product-readiness-20260603-015100.json` has `provider.selected=deepseek`, `provider.real_provider_ready=true`, and `provider.smoke_status=passed`; `reports/provider-tts-candidate/a21-server-side-readiness-bundle-20260603-015102.json` has `provider.ready=true`. StepFun/Iflytek remains voice-chain candidate evidence rather than being forced into the product provider slot. | `S5-VOICE-CHAIN-EVIDENCE-INGRESS-OR-LONG-DIALOGUE` |
| V21 adapter | `S2-HOST-READY` | Adapter contract exists; no firmware key or V21 internals should leak into A21 | `S3-PROFESSIONAL-EVIDENCE-RUN` |
| Memory/personality | `S1-IMPLEMENTED-HOST` | Host-side memory/personality work exists but needs current PRD burn-down refresh | `S2-READINESS-REVIEWED` |
| Physical StackChan acceptance | `S3D-STACKCHAN-COMPATIBLE-RESTORED-RELAY-WAV-OK` | Physical stock Xiaozhi path was online; runtime volume `100` and 3x `/v1/xiaozhi/say` were delivered to `44:1b:f6:e2:6a:60`; accepted recording `/Users/jiyurun/Downloads/军民公路259号 10.m4a` measured `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds. On 2026-06-03 the Gateway was restarted from this branch, device `44:1b:f6:e2:6a:60` reconnected online, runtime volume `100` was delivered via stock MCP trace `a21-trace-relay-playback-volume-1780450901`, and relay WAV playback trace `a21-trace-provider-playback-wav-1780450901` delivered `40` stock Xiaozhi Opus downlink chunks. After the mistaken bare wake/Xiaozhi flash, corrective StackChan-compatible app flash restored the product candidate; post-restore runtime volume `100` delivered on trace `a21-trace-recovery-stackchan-volume-1780424386012`, and the accepted relay WAV delivered on trace `a21-trace-recovery-stackchan-relay-wav-1780424386012` with `audio_chunks=40`. Full PRD accepted remains false because custom wake and broader dialogue half-duplex evidence are still pending. | `S4-PRD-PHYSICAL-ACCEPTED` |
| Xiaozhi audio/protocol | `S3D-LIVE-SAY-HOTFIX-RUNNING` | Plan `docs/plans/2026-06-02-stackchan-audio-official-parity-hotfix.md`; server/downlink hello restored to stock-compatible 24 kHz while client/uplink remains 16 kHz; stock physical MAC devices no longer receive unsupported server `type=listen` replies; `/v1/xiaozhi/say` delivers TTS lifecycle and Opus binary to the live stock socket for text and now accepts a local 16 kHz mono WAV through `wav_path` with basename-only response metadata; focused Gateway/transport/app tests pass | `S4-PHYSICAL-STOCK-AUDIO-RETESTED` |
| TTS/audio quality | `S7A-RELAY-VOICE-PHYSICAL-OPERATOR-POSITIVE` | Recording `9.m4a` from 3 seconds stayed weak at `-36.99 LUFS` and `-16.05 dBTP`; bounded Gateway PCM leveling at 4x produced `/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a` at `-21.65 LUFS` and `-4.72 dBTP`, but the operator reported subtle electrical interruption and reduced clarity. Follow-up recording `/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 3.m4a` measured `-26.4 LUFS` and `-9.0 dBFS` true peak from 3 seconds. Current code uses bounded 3x gain, noise gate, and headroom; live 3x trace `a21-trace-stackchan-say-1780417217` delivered `579` chunks. Operator accepted final recording `/Users/jiyurun/Downloads/军民公路259号 10.m4a`, measured `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds. Iflytek TTS is now coded as the immediate clearer TTS source; `voice_clone_cli` remains the voice-clone/persona seam; old `sherpa_onnx` is emergency/diagnostic only. Live Mac direct Iflytek smoke extracted the 5080 report credentials in memory without printing them, then failed at WebSocket dial; redacted report `reports/provider-tts-candidate/a21-local-tts-smoke-20260603-010638.json` has `status=failed`, `network_mode=direct`, finding `iflytek_tts_websocket_dial_failed`. 5080 relay host-chain WAV `reports/provider-tts-candidate/a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav` was delivered physically through stock `/v1/xiaozhi/say`; operator reported it was much better. Next improvement is selected-provider/readiness evidence and longer normal dialogue acceptance, not additional Gateway gain. | `S7-IFLYTEK-TTS-PHYSICAL-ACCEPTED` |
| Local clone TTS / CosyVoice | `S1-5080-SOURCE-PRESENT-WEIGHTS-MISSING` | Worker `019e897d-d4b4-78c3-9358-ac27a4f61d0d` confirmed 5080 is online and found CosyVoice source plus venv, IndexTTS2 source/venv/runner with CUDA, and traces of F5-TTS/GPT-SoVITS. No clone WAV was produced: CosyVoice venv lacks `torch`/`tqdm` and has no usable `pretrained_models/CosyVoice-*` weights; IndexTTS2 is closest but fails because `checkpoints/qwen0.6bemo4-merge/` is missing or not loadable; F5-TTS/GPT-SoVITS have no confirmed ready checkpoint/run path. Plan `docs/plans/2026-06-03-cosyvoice-5080-local-clone-candidate.md` now tracks restoration. | `S2-LOCAL-CLONE-SMOKE-WAV-PRODUCED` |
| StackChan volume/action control | `S5-RUNTIME-VOLUME100-PHYSICAL-ACCEPTED` | `POST /v1/xiaozhi/speaker-volume` delivered official MCP `self.audio_speaker.set_volume` with `volume=100` to live device `44:1b:f6:e2:6a:60`; latest 3x trace is `a21-trace-stackchan-volume-1780417211`. Desktop helper supports both `volume` and `say`; physical loudness is accepted through final operator recording `10.m4a`. | `S6-FROZEN-FOR-CONTEST-FLOW` |
| Half-duplex / echo control | `S3A-NO-FLASH-NORMAL-DIALOGUE-NO-SELF-TRIGGER-CANDIDATE` | `/v1/xiaozhi/say` now arms a short input-suppression window for stock physical devices after host-say completion; focused test proves immediate listen restart and Opus echo are ignored without starting a new voice pipeline. Physical 3x trace `a21-trace-stackchan-say-1780417217` recorded `xiaozhi.say.input_suppression_armed=1` and `xiaozhi.listen.start.input_suppressed=1`. Diagnostic counter reports `reports/a21-stackchan-half-duplex-acceptance-20260603-014233.json` and `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json` are blocked because current stock firmware lacks A21 identity, diagnostic mic-probe capability, available speaker echo fields, and runtime echo counters. That diagnostic blocker no longer blocks the contest path: no-flash normal-dialogue observation report `reports/a21-no-flash-normal-dialogue-observation-20260603-020055.json` delivered the accepted StepFun+Iflytek WAV through stock `/v1/xiaozhi/say` on trace `a21-trace-no-flash-dialogue-observe-1780423245`, waited `20000 ms`, observed `869` trace events, recorded `xiaozhi.listen.start.input_suppressed=1`, and found no `xiaozhi.listen.start`, `provider.start_turn.start`, `provider.realtime_session.start`, or `xiaozhi.voice_pipeline.start` self-trigger events. | `S4-TOUCH-BARGE-IN-OPERATOR-CHECK-OR-DIAGNOSTIC-COUNTERS` |
| Wake word | `S4G-ZI-YUE-PHRASE-TUNED-FLASHED-AWAITING-PROOF` | Custom MultiNet package `reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json` and bare flash report `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json` remain incident/background evidence only because `xiaozhi.bin` regressed the product UI. `T-WAKE-002` integrated the requested wake word into the product app lane and flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030818-1780427298506934000.json` passed, but operator physical verification failed: saying `紫悦` produced no response. `T-WAKE-003` keeps display `紫悦` and updates the custom MultiNet command string to `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue` while staying in the official-compatible product lane. Build report `reports/a21-stackchan-official-baseline-20260603-032615-1780428375746120000.json` passed; flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-032734-1780428454747199000.json` passed; device reconnected online. | `S5-ZI-YUE-TUNED-PHYSICAL-WAKE-ACCEPTED` |

## Active Transitions

### Active T-WAKE-003: Zi Yue Phrase Tuning

Current state:

- `S4G-ZI-YUE-PHRASE-TUNED-FLASHED-AWAITING-PROOF`

Target state:

- `S5-ZI-YUE-TUNED-PHYSICAL-WAKE-ACCEPTED`

Trigger:

- Operator reported that saying `紫悦` produced no response after the guarded
  `T-WAKE-002` product flash.
- Current `zi yue` command is only two syllables and is likely too short for
  robust ESP-SR/MultiNet custom command recognition.

Actions:

- Use
  `docs/plans/2026-06-03-wake-and-asr-green-latency-recovery.md`.
- Keep product lane `a21-stackchan-official-xiaozhi-compatible.bin`.
- Keep visible wake identity `紫悦`.
- Tune only the custom MultiNet command list to
  `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue`.

Current result:

- Commit `5242349` passed focused tests and `make verify`.
- Official-compatible build passed with app SHA-256
  `e13a6cbd63596cd1388f5a540b488b3e35b449e49a5e31891cf133436561dc6e`.
- Guarded official-compatible flash execute passed on `/dev/cu.usbmodem1101`.
- Device `44:1b:f6:e2:6a:60` reconnected online.
- Physical wake proof is still pending operator retry.

Acceptance conditions:

- Focused overlay test passes.
- Official-compatible build config proves the tuned command string.
- Guarded official-compatible flash execute passes after clean-worktree checks.
- Operator physically confirms one of the `紫悦` variants wakes the device
  without screen touch.

Failure state:

- `F-WAKE-003-PHYSICAL-WAKE-REJECTED` if the tuned phrases still do not wake.
- `F-WAKE-003-FALSE-WAKE` if the tuned phrases cause unacceptable false wakes.

Rollback path:

- Keep the same product lane and tune threshold/phrases only.
- Reflash the last accepted official-compatible app if UI or audio regresses.

Next state:

- `S5-ZI-YUE-TUNED-PHYSICAL-WAKE-ACCEPTED`

### Active T-ASR-GREEN-LATENCY-001: Xiaozhi Listen Auto-Stop

Current state:

- `S-GREEN-ASR-LISTEN-HOTFIX-RUNNING-AWAITING-PHYSICAL-RETEST`

Target state:

- `S-GREEN-ASR-LISTEN-BOUNDED`

Trigger:

- Operator reported a long wait after ASR turns green.
- Live trace `a21-trace-44-1b-f6-e2-6a-60` showed multiple long reused-trace
  listen windows, including approximately 25s, 38s, 70s, and 108s before
  auto-stop or another listen replaced the window.

Actions:

- Use
  `docs/plans/2026-06-03-wake-and-asr-green-latency-recovery.md`.
- Add a Gateway max-listen safety stop after speech has been detected.
- Default max listen is 7000 ms and can be overridden with
  `A21_XIAOZHI_LISTEN_MAX_MS`.
- Improve trace summary pairing for reused physical trace ids.

Current result:

- Commit `5242349` passed focused tests and `make verify`.
- Gateway `a21-gateway-21081` was restarted with
  `A21_XIAOZHI_LISTEN_MAX_MS=7000`.
- Device `44:1b:f6:e2:6a:60` reconnected online to the restarted Gateway.
- Runtime volume `100` was redelivered through stock MCP on trace
  `a21-trace-wake-asr-hotfix-volume-1780428544`.
- Operator retest is still required.

Acceptance conditions:

- Focused Gateway test proves `xiaozhi.listen.max_duration_auto_stop`,
  `xiaozhi.listen.auto_stop`, and `xiaozhi.voice_pipeline.start`.
- Focused app test proves env wiring.
- Operator reports the green-light wait is materially shorter after restart or
  flash/runtime refresh.

Failure state:

- `F-ASR-GREEN-LATENCY-STILL-LONG` if green waits still exceed the configured
  bound.
- `F-ASR-GREEN-LATENCY-CUTS-SPEECH` if the bound cuts acceptable normal
  utterances too aggressively.

Rollback path:

- Increase `A21_XIAOZHI_LISTEN_MAX_MS` or revert only this Gateway change.

Next state:

- `S-GREEN-ASR-LISTEN-BOUNDED`

### Active T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001: Official Frontend Without Setup Trap

Current state:

- `S-APP-PRELOAD-NO-WELCOME-IDLE-SOCKET-CANDIDATE`

Target state:

- `S-APP-PRELOAD-NO-WELCOME-IDLE-SOCKET-READY`

Trigger:

- The operator confirmed the request/start variant trapped the physical device
  on "Welcome! Let's get started", but also asked to preserve official app
  loading so the full StackChan hardware experience is not lost.
- The operator also clarified wake cannot be verified before socket connection:
  before the Xiaozhi socket is established, saying the wake word does nothing;
  tapping the screen opens the socket by entering an unbounded green listening
  state.

Actions:

- Use
  `docs/plans/2026-06-03-boot-idle-socket-and-not-connected-ux.md`.
- Treat `T-BOOT-IDLE-SOCKET-001` as the sub-transition for quiet socket
  preconnect and not-connected feedback.
- Patch the official-compatible product overlay to install official StackChan
  apps first, then set codec volume and request Xiaozhi immediately without
  showing the welcome/setup trap.
- Add A21-named idle control-channel preconnect and `紫悦` connecting/ready
  feedback.
- Add device-side no-speech timeout so a touch-started listen without speech
  exits instead of staying green indefinitely.

Current result:

- Focused overlay tests pass for app-preload autostart, tuned `紫悦` custom
  wake, A21 quiet idle socket contract, and product-candidate reporting.
- The overlay patch passes ordinary `git apply --check` against the official
  source tree and `git diff --check`.
- `make verify` passed after the build-tool change that applies overlays after
  `fetch_repos.py`.
- Official-compatible build passed:
  `reports/a21-stackchan-official-baseline-20260603-040613-1780430773893634000.json`.
- Product app SHA-256:
  `7ff81bb0e564e020128e02068bc7d83b36f90c21de8cbd3d4b89b1dd1d6e9cf3`.
- Generated `sdkconfig.json` proves
  `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`,
  `USE_CUSTOM_WAKE_WORD=true`,
  `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`,
  `CUSTOM_WAKE_WORD_THRESHOLD=20`,
  `SR_MN_CN_MULTINET7_QUANT=true`,
  `USE_AFE_WAKE_WORD=false`,
  `SR_WN_WN9_HISTACKCHAN_TTS3=false`, and
  `SEND_WAKE_WORD_DATA=false`.
- Flash and physical operator validation are still pending.
- The `requestXiaozhiStart()` variant was physically rejected because it showed
  the welcome/setup page; the current hotfix starts Xiaozhi directly after app
  install and before any Mooncake update/setup frame.
- Commit `9ba8bc1` was guarded-flashed successfully through the
  official-compatible product lane:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-041636-1780431396539566000.json`.
- Device `44:1b:f6:e2:6a:60` reconnected `online` after flash, and runtime
  speaker volume `100` was delivered on trace
  `a21-trace-direct-preload-hotfix-volume-1780431400`.
- Operator later reported a welcome/setup regression again after the
  multi-wake flash. Root cause is now corrected in the overlay: official
  `Hal::startXiaozhi()` starts Xiaozhi and returns, so the previous direct
  autostart could still fall through into the Mooncake main loop and let
  `AppLauncher` create `StartupWorker`. The current hotfix parks `app_main`
  after direct `GetHAL().startXiaozhi()` by feeding the watchdog and sleeping,
  preventing AppLauncher/AppSetup from rendering `Welcome! Let's get started`.
- Focused app test now guards this control-flow contract by requiring
  `GetHAL().feedTheDog()` and `GetHAL().delay(1000)` before the Mooncake main
  loop anchor.
- The latest guarded product flash report is
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062853-1780439333539408000.json`;
  T7 guard recorded clean commit `790697518c3e`, product app
  `a21-stackchan-official-xiaozhi-compatible.bin`, and app SHA-256
  `fc7788736ced71c98cee846892a867d306d663c5ceb31014cc01079a381e766c`.
- The operator later confirmed setup is fixed. Current live Gateway evidence
  from `/v1/devices` on `127.0.0.1:21081` shows physical device
  `44:1b:f6:e2:6a:60` is `online`, `last_event=xiaozhi.hello`, and
  `speaker_volume=100`.
- Recovery trace analysis for `a21-trace-44-1b-f6-e2-6a-60` shows the old
  touch-start episode had only one `xiaozhi.listen.start`, one
  `xiaozhi.listen.stop`, one `xiaozhi.turn.start`, and no ongoing loop. The
  no-speech cooldown fired once with
  `xiaozhi.no_speech.input_suppression_armed=1`,
  `xiaozhi.listen.start.input_suppressed=1`, and
  `xiaozhi.listen.start.suppressed_after_no_speech=1`; the latest event is
  `xiaozhi.hello.received`.
- Runtime speaker volume `100` was re-delivered through stock MCP on trace
  `a21-trace-recovery-volume-1780441733`.
- Physical wake with `紫悦` variants and touch/no-speech retry remain pending.

Acceptance conditions:

- `make verify` passes.
- Official-compatible build config proves
  `A21_STACKCHAN_KEEP_CONTROL_CHANNEL=true`.
- Product flash uses only
  `a21-stackchan-official-xiaozhi-compatible.bin`.
- On boot, the device opens the stock Xiaozhi socket without requiring touch
  and without sending `listen.start`.
- The screen shows connecting/ready feedback while the socket is not ready.
- Touch without speech returns from green listening within the device timeout.

Failure state:

- `F-APP-PRELOAD-WELCOME-TRAP` if the welcome/setup screen appears again.
- `F-IDLE-SOCKET-STARTS-LISTENING` if boot preconnect sends `listen.start`.
- `F-TOUCH-GREEN-STILL-INFINITE` if touch still strands the device in green.

Rollback path:

- Revert only this overlay/test/doc transition and reflash the last known
  official-compatible product app if physical boot, audio, or wake behavior
  regresses.

Next state:

- `S-APP-PRELOAD-NO-WELCOME-IDLE-SOCKET-READY`

### Active T-AUDIO-BARE-XIAOZHI-PARITY-001: Migrate Bare-Package Audio Advantages

Current state:

- `S-BARE-XIAOZHI-PARITY-FACT-RECORDED`

Target state:

- `S-COMPATIBLE-PRODUCT-AUDIO-PARITY-CHECKLIST-ACTIONABLE`

Trigger:

- The operator-confirmed fact is that the bare `xiaozhi.bin` sounded louder and
  clearer, but it is a non-product incident artifact because it regressed the
  StackChan avatar/product UI.

Actions:

- Keep product lane `a21-stackchan-official-xiaozhi-compatible.bin`; do not
  flash bare `xiaozhi.bin`.
- Split parity into volume/NVS/MCP proof, TTS source mastering, downlink
  PCM/Opus/pacer metrics, and official StackChan app/action initialization.
- Use an independent worker thread for read-only or docs-only parity checklist
  generation.

Current result:

- Worker dispatch is active for this transition.
- No new audio/gain/provider behavior has been claimed in the main thread.

Acceptance conditions:

- A parity checklist identifies which differences are already closed, which are
  host-configurable, and which require physical A/B.
- The checklist preserves the accepted 3x contest path and does not promote
  additional Gateway gain without evidence.

Failure state:

- `F-AUDIO-PARITY-BARE-FLASH-REGRESSION` if a worker or main thread attempts to
  use bare `xiaozhi.bin` as product firmware.
- `F-AUDIO-PARITY-UNVERIFIED-TUNING` if a change claims louder/clearer physical
  output without operator recording or trace evidence.

Rollback path:

- Keep current accepted 3x audio path and runtime volume `100`; defer parity
  tuning until a bounded A/B plan exists.

Next state:

- `S-COMPATIBLE-PRODUCT-AUDIO-PARITY-CHECKLIST-ACTIONABLE`

### Active T-XIAOZHI-FULL-REALTIME-VOICE-CONVERGENCE-001: Full Xiaozhi Realtime Voice Convergence

Current state:

- `S-XIAOZHI-REALTIME-SEAMS-MAINLINE-NOT-OVERLAPPED`

Trigger:

- The user's review correctly defines Xiaozhi voice as a realtime media system:
  local wake/VAD, long Xiaozhi transport, small Opus frames, streaming ASR,
  streaming LLM, streaming TTS, and paced Opus playback.
- A21 has multiple correct-looking seams, but seams/static gates are not the
  same as real runtime or physical acceptance.

Target state:

- `S-XIAOZHI-FULL-REALTIME-PHYSICAL-CANDIDATE`

Action:

- Use plan
  `docs/plans/2026-06-03-xiaozhi-full-realtime-voice-convergence.md`.
- Keep the product firmware lane on
  `a21-stackchan-official-xiaozhi-compatible`.
- Read-only worker audits for official Xiaozhi protocol/state, CoreS3/audio
  HAL/wake behavior, A21 runtime gaps, and architecture strategy have completed
  again on HEAD `188b341`. They agree that A21 should continue incremental
  convergence on stock Xiaozhi protocol/framework patterns, not wholesale
  server-stack embedding, while keeping full realtime acceptance red.
- Sequence the next implementation transitions as:
  the now-landed host ASR partial-to-LLM bridge, real Sherpa streaming ASR
  unblock, real streaming TTS runtime execution if authorized, physical stock
  `/v1/xiaozhi` realtime parity, local wake/state closure, and official
  StackChan audio/HAL behavior parity.
- Preserve strict evidence boundaries: host/static/mock/provider-shape evidence
  cannot claim physical PRD acceptance.

Acceptance conditions:

- Plan exists and is committed.
- Worker fan-out ids and boundaries are recorded in the plan/handoff.
- `docs/project_state_machine.md` names this convergence transition so future
  models do not treat individual ASR/TTS seams as final success.
- The next code execution worker receives one scoped transition with explicit
  no-flash/no-provider/no-hardware boundaries unless separately authorized.

Failure states:

- `F-XIAOZHI-CONVERGENCE-FAKE-GREEN` if mock, `/say`, host-loopback, static
  readiness, or WAV/file paths are described as Xiaozhi realtime acceptance.
- `F-XIAOZHI-CONVERGENCE-SCOPE-SPRAWL` if one worker silently changes firmware,
  provider execution, wake, gain, and Gateway behavior together.
- `F-XIAOZHI-CONVERGENCE-STOCK-PROFILE-LEAK` if debug/device extensions become
  required in the stock Xiaozhi hello/runtime.

Rollback path:

- Revert the convergence plan/state/log entries. Existing ASR/TTS seams and
  accepted contest audio path remain untouched.

Next state:

- `S-XIAOZHI-CONTINUOUS-TURN-BRIDGE-CANDIDATE-HOST-TESTED`
- Next transition: unblock real Sherpa streaming ASR runtime evidence, run real
  streaming TTS runtime only if explicitly authorized, then collect a physical
  stock `/v1/xiaozhi` parity trace without `/say`, host loopback, mock, or
  WAV/file evidence.

### Active T-XIAOZHI-REALTIME-VOICE-PARITY-001: Xiaozhi Realtime Voice Parity Gate

Current state:

- `S-XIAOZHI-STOCK-OPUS-TRANSPORT-TURN-BUFFERED-HOST`

Trigger:

- User review requires A21 to match Xiaozhi's realtime voice design rather than
  behaving like a normal full-WAV/full-HTTP voice bot.
- Three read-only worker audits on 2026-06-03 agreed that the product device
  lane is stock-shaped at the `/v1/xiaozhi` Opus/WebSocket boundary, but the
  host voice pipeline is still turn-buffered at the ASR boundary.

Target state:

- `S-XIAOZHI-REALTIME-PARITY-GATE-LANDED`

Action:

- Keep product firmware on `a21-stackchan-official-xiaozhi-compatible.bin`.
- Add a trace-only `xiaozhi-realtime-parity` evidence command that reads
  Gateway `/v1/devices` and `/v1/traces` without driving `/say`, synthetic
  host-loopback WebSocket audio, provider execution, V21 execution, flash, NVS,
  or audio playback.
- Classify live traces as `blocked`, `stock_opus_transport_only`,
  `turn_buffered_xiaozhi_candidate`, or `xiaozhi_realtime_candidate`.
- Reject fake path markers such as `/v1/xiaozhi/say`,
  `fast_companion.voice_pipeline.*`, and local fallback control events.

Acceptance conditions:

- Focused tests pass for physical stock Opus traces, streaming-ordering traces,
  and fake `/say` traces.
- The report redacts payloads, credentials, full URLs, local paths,
  transcripts, and raw audio.
- The report never sets `prd_accepted=true` and does not claim local wake,
  audible playback, interruption, or setup-free product acceptance.

Failure states:

- `F-XIAOZHI-REALTIME-PARITY-FAKE-GREEN` if host-loopback, `/say`, or
  fast-companion traces can pass as realtime parity.
- `F-XIAOZHI-REALTIME-PARITY-UNSAFE-REPORT` if report output stores secrets,
  raw audio, transcripts, full URLs, or local paths.
- `F-XIAOZHI-REALTIME-PARITY-BEHAVIOR-REGRESSION` if adding the evidence gate
  changes Gateway or firmware runtime behavior.

Rollback path:

- Remove the additive CLI/report/tests and revert this state/log entry. No
  firmware, provider, Gateway runtime, or NVS rollback is required for Phase 1.

Next state:

- `S-XIAOZHI-REALTIME-PARITY-GATE-LANDED`
- Follow-up transition if the live report remains turn-buffered:
  `T-XIAOZHI-STREAMING-ASR-001`.

### Completed T-XIAOZHI-REALTIME-PARITY-REAL-PROFILE-EVIDENCE-001: Realtime Parity Real Profile Evidence

Current state:

- `S-XIAOZHI-REALTIME-PARITY-ORDERING-GATE-HARDENED`

Trigger:

- The parity gate could classify a trace as `xiaozhi_realtime_candidate` based
  on stock Opus transport plus ordered `asr.first_partial`,
  `provider.first_content`, `tts.first_audio`, and downlink markers, even when
  the trace did not prove real streaming ASR/LLM/TTS profile classes.
- User review explicitly warns that A21 must not hide a normal buffered
  question/answer path behind Xiaozhi-shaped protocol markers.

Target state:

- `S-XIAOZHI-REALTIME-PARITY-REQUIRES-REAL-PROFILE-EVIDENCE`

Action:

- Added plan
  `docs/plans/2026-06-03-xiaozhi-realtime-parity-real-profile-evidence.md`.
- Gateway voice-pipeline turns now emit redacted category markers for profile
  classes:
  `xiaozhi.voice_pipeline.asr.real_streaming`,
  `xiaozhi.voice_pipeline.llm.real_streaming`,
  `xiaozhi.voice_pipeline.tts.real_streaming`, or blocker markers such as
  `asr.mock_blocked`, `asr.batch_blocked`, `llm.mock_blocked`,
  `tts.mock_blocked`, and `tts.file_boundary_blocked`.
- `xiaozhi-realtime-parity` now counts those markers and requires all three
  real streaming stage markers, with no profile blockers, before it can return
  `xiaozhi_realtime_candidate`.

Acceptance conditions:

- Red app test first proved an ordered realtime-looking trace without real
  profile markers was incorrectly accepted as `xiaozhi_realtime_candidate`.
- Red Gateway test first proved mock partial-bridge turns lacked blocker
  markers.
- Focused app/Gateway tests pass and reports stay redacted.

Failure states:

- `F-REALTIME-PARITY-MOCK-GREEN` if mock or file-boundary profile classes can
  still reach `xiaozhi_realtime_candidate`.
- `F-TRACE-SECRET-LEAK` if profile evidence stores provider values, URLs,
  paths, transcripts, credentials, proxy values, or raw audio.

Rollback path:

- Revert the profile marker additions, parity-gate checks, tests, plan, and
  state/log entries. Existing stock `/v1/xiaozhi` transport, ASR partial
  bridge, Sherpa smoke, and TTS adapter seam remain intact.

Next state:

- `S-XIAOZHI-REALTIME-PARITY-REQUIRES-REAL-PROFILE-EVIDENCE`
- Next transition: non-blocking turn reducer/queue hardening so streaming ASR
  commit/final handling cannot stall the Xiaozhi WebSocket read loop.

### Completed T-XIAOZHI-NONBLOCKING-ASR-COMMIT-001: Xiaozhi Nonblocking ASR Commit

Current state:

- `S-XIAOZHI-ASR-COMMIT-NONBLOCKING-HOST-TESTED`

Trigger:

- Runtime read-only audit found `commitXiaozhiStreamingASR` was called
  synchronously from `listen.stop` and VAD auto-stop, which could hold the
  `/v1/xiaozhi` WebSocket read loop while commit/final polling waited.
- Xiaozhi-style voice requires the control channel to remain responsive to
  abort/barge-in while ASR finalization is pending.

Target state:

- `S-XIAOZHI-ASR-COMMIT-ASYNC-CONTROL-LOOP-RESPONSIVE`

Action:

- Added plan `docs/plans/2026-06-03-xiaozhi-nonblocking-asr-commit.md`.
- Replaced synchronous stop/auto-stop ASR commit with an async commit task.
- The async task records `asr.stream.commit`, waits briefly for a streaming
  ASR final, starts the voice-pipeline task from the streaming final when no
  partial-driven answer already started, and records truthful timeout/error
  markers instead of silently promoting missing final evidence.
- Removed the unused synchronous commit helper to avoid future regression.

Acceptance conditions:

- Red test first proved abort was not processed while streaming ASR commit was
  pending.
- New Gateway tests prove abort is recorded during pending commit and that a
  streaming ASR final starts the pipeline without calling batch `Transcribe`.
- Existing Xiaozhi streaming ASR, partial bridge, voice-pipeline, and parity
  tests continue to pass.

Failure states:

- `F-XIAOZHI-ASR-COMMIT-BLOCKS-WS` if abort/barge-in cannot be read during ASR
  commit.
- `F-XIAOZHI-ASR-COMMIT-BATCH-REGRESSION` if a streaming ASR final path calls
  batch `Transcribe` or writes a WAV.
- `F-XIAOZHI-ASR-COMMIT-DOUBLE-TURN` if partial-driven and final-driven paths
  start duplicate voice-pipeline tasks.

Rollback path:

- Revert the async commit helper, stop/auto-stop call-site changes, tests, plan,
  and state/log entries. Existing stock `/v1/xiaozhi` transport, ASR partial
  bridge, real-profile gate, Sherpa smoke, and TTS adapter seam remain intact.

Next state:

- `S-XIAOZHI-ASR-COMMIT-ASYNC-CONTROL-LOOP-RESPONSIVE`
- Next transition: feed the real Sherpa streaming ASR session through a stock
  `/v1/xiaozhi` host-local trace, then keep realtime parity blocked until real
  streaming LLM/TTS profile evidence and physical playback are present.

### Completed T-XIAOZHI-OFFICIAL-PROTOCOL-SOURCE-READ-AND-NEXT-CUT-001: Stock STT And Wake Preroll

Current state:

- `S-XIAOZHI-STOCK-STT-WAKE-PREROLL-HOST-TESTED`

Trigger:

- Source-read workers compared upstream Xiaozhi protocol/audio behavior with
  A21 and found two narrow host-side fidelity gaps that did not require live
  providers, Gateway lifecycle changes, firmware, hardware, or audio playback.
- Stock Xiaozhi expects server `stt` display messages before TTS when
  transcripts exist, and wake-adjacent Opus can arrive before/around
  `listen/detect`.

Target state:

- `S-XIAOZHI-STOCK-STT-AND-IDLE-PREROLL-PRESERVED`

Action:

- Added plan
  `docs/plans/2026-06-03-xiaozhi-official-protocol-source-read-and-next-cut.md`.
- Emitted stock `stt` WebSocket messages before `tts/start` when a streaming
  ASR partial or final transcript is available, without recording transcript
  text in traces.
- Added bounded idle wake pre-roll buffering for decoded Opus frames received
  before `listen/start`, attaching those frames to the next turn and feeding
  them into streaming ASR when active.
- Kept cooldown and current-turn suppression intact so post-placeholder or
  host-say Opus remains `ignored_not_listening` instead of becoming false wake
  evidence.

Acceptance conditions:

- Red tests first proved TTS began before stock `stt` and pre-listen Opus was
  dropped from the next voice pipeline request.
- Focused Gateway tests prove stock `stt` precedes TTS, wake pre-roll feeds the
  next turn, streaming ASR/final/partial paths still work, and paced Opus TTS
  downlink still works.
- Full Gateway package and focused app realtime parity/readiness tests pass.

Failure states:

- `F-XIAOZHI-STT-ORDER-REGRESSION` if TTS starts before stock `stt` when a
  transcript exists.
- `F-XIAOZHI-WAKE-PREROLL-DROPPED` if idle pre-listen Opus cannot feed the next
  streaming ASR/voice-pipeline turn.
- `F-XIAOZHI-COOLDOWN-FALSE-WAKE` if cooldown or current-turn Opus is
  misclassified as wake pre-roll.

Rollback path:

- Revert the stock STT helper, wake pre-roll buffering/attach logic, tests,
  plan, and state/log entries. Existing nonblocking ASR commit, partial bridge,
  real-profile gate, Sherpa smoke, and realtime TTS seam remain intact.

Next state:

- `S-XIAOZHI-STOCK-STT-AND-IDLE-PREROLL-PRESERVED`
- Next transition: with authorization, collect a physical stock `/v1/xiaozhi`
  trace or execute the real streaming TTS provider path; keep realtime PRD
  acceptance false until real streaming ASR/LLM/TTS profile evidence, physical
  playback, barge-in/touch, and idle recovery are proven.

### Completed T-XIAOZHI-OPUS-INGRESS-QUEUE-001: Xiaozhi Opus Ingress Queue

Current state:

- `S-XIAOZHI-OPUS-INGRESS-QUEUED-HOST-TESTED`

Trigger:

- The full realtime objective explicitly requires an Opus frame queue between
  local wake/VAD capture and streaming ASR.
- Source-read workers confirmed upstream Xiaozhi separates protocol callbacks
  from encode/decode/playback queues, while A21 still decoded Opus, ran
  audio-ingress/VAD, and appended to streaming ASR inline on the WebSocket read
  loop.

Target state:

- `S-XIAOZHI-OPUS-FRAME-QUEUE-CONTROL-LOOP-RESPONSIVE`

Action:

- Added plan `docs/plans/2026-06-03-xiaozhi-opus-ingress-queue.md`.
- Added a bounded per-session Opus ingress queue for listening frames.
- `handleXiaozhiBinary` now parses/adopts incoming Opus, records receipt,
  enqueues the frame, and returns to the WebSocket read loop without waiting
  for decode, VAD/audio ingress, or streaming ASR append.
- The queue worker preserves frame order, decodes Opus, pushes audio ingress
  evidence, appends to streaming ASR, and records queue drops explicitly.
- `listen.stop` now starts an async finalize path that waits briefly for
  queued ingress to catch up before committing streaming ASR or starting the
  voice pipeline, keeping the control loop free to process abort.

Acceptance conditions:

- Red test first proved a blocking streaming ASR `AppendFrame` prevented
  `abort` from being processed.
- Focused Gateway tests prove `abort` is recorded while ASR append remains
  blocked and that wake pre-roll, stock STT, streaming final, nonblocking
  commit, partial bridge, and paced Opus downlink still work.
- Full Gateway package, focused app parity/readiness tests, `git diff --check`,
  and `make verify` pass.

Failure states:

- `F-XIAOZHI-OPUS-INGRESS-BLOCKS-CONTROL` if audio decode/VAD/ASR append can
  block reading `abort`.
- `F-XIAOZHI-OPUS-INGRESS-EMPTY-STOP` if `listen.stop` starts a voice pipeline
  before queued frames are processed.
- `F-XIAOZHI-OPUS-INGRESS-HIDDEN-DROP` if queue overflow loses frames without
  trace evidence.

Rollback path:

- Revert the queue helper code, async stop-finalize path, tests, plan, and
  state/log entries. Existing stock STT, wake pre-roll, nonblocking ASR commit,
  partial bridge, real-profile gate, Sherpa smoke, and realtime TTS seam remain
  intact.

Next state:

- `S-XIAOZHI-OPUS-FRAME-QUEUE-CONTROL-LOOP-RESPONSIVE`
- Next transition: prove wake-as-abort/playback-drain ordering in a host stock
  WebSocket test, then move to authorized physical stock `/v1/xiaozhi` trace or
  real streaming TTS runtime proof. Full PRD acceptance remains false until
  real streaming ASR/LLM/TTS profile evidence, physical playback, wake/tap,
  barge-in/touch, and idle recovery are proven.

### Active T-XIAOZHI-STREAMING-ASR-001: Stock Xiaozhi Streaming ASR Session

Current state:

- `S-XIAOZHI-STREAMING-ASR-SESSION-SEAM-IMPLEMENTED-FAKE-ONLY`

Trigger:

- User review requires A21 to match Xiaozhi's realtime chain:
  local wake/VAD, long socket, Opus frames, streaming ASR, streaming LLM,
  streaming TTS, and paced Opus playback.
- Read-only workers confirmed current `/v1/xiaozhi` was previously
  turn-buffered at the ASR boundary: decoded PCM frames were accumulated, then
  ASR ran after stop/VAD end.

Target state:

- `S-XIAOZHI-STREAMING-ASR-PROVIDER-READY`

Action:

- Use `docs/plans/2026-06-03-xiaozhi-streaming-asr.md`.
- Add optional `providers.StreamingASRAdapter` and `StreamingASRSession`
  alongside existing batch `ASRAdapter`.
- Start a streaming ASR session on stock `/v1/xiaozhi` listen start when the
  selected ASR adapter supports it.
- Feed decoded Opus PCM frames into the streaming ASR session in
  `observeXiaozhiDecodedIngress`.
- Commit the streaming ASR session on `listen.stop` or VAD auto-stop.
- Reuse streaming ASR final text in the voice pipeline so batch ASR is not
  re-run when streaming final is available.
- Keep existing accumulated `voicePipelineFrames` and batch ASR fallback when no
  streaming session/final exists.

Acceptance conditions:

- Gateway test proves `asr.first_partial` appears before `xiaozhi.listen.stop`
  and before `xiaozhi.voice_pipeline.start`.
- Existing listen-stop and VAD auto-stop tests still pass.
- Provider test proves mock streaming ASR emits partial on frame append and
  final only on commit.
- `xiaozhi-realtime-parity` counts `asr.stream.start`,
  `asr.audio.append`, and `asr.stream.commit`, and requires stream start/append
  for `xiaozhi_realtime_candidate`.
- This phase is not real provider acceptance; it is a session seam and fake
  adapter proof.

Failure states:

- `F-XIAOZHI-STREAMING-ASR-BATCH-REGRESSION` if non-streaming ASR fallback no
  longer works.
- `F-XIAOZHI-STREAMING-ASR-FAKE-PROVIDER-GREEN` if fake streaming ASR is
  recorded as real provider readiness.
- `F-XIAOZHI-STREAMING-ASR-GOROUTINE-LEAK` if abort/socket close does not cancel
  sessions.
- `F-XIAOZHI-STREAMING-ASR-TRACE-ORDERING-REGRESSION` if partial/final markers
  occur only after listen stop.

Rollback path:

- Revert the additive provider interfaces, Gateway session lifecycle hooks,
  tests, and this state entry. No firmware, NVS, provider credential, or audio
  gain rollback is involved.

Next state:

- `S-XIAOZHI-STREAMING-ASR-PROVIDER-READY`
- Next transition:
  `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`.

### Active T-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-001: Sherpa Streaming ASR Runtime Helper

Current state:

- `S-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-HELPER-MAINLINE-CANDIDATE`

Trigger:

- The full Xiaozhi parity objective requires real streaming ASR, not a batch
  WAV runner or a selectable profile name.
- `T-XIAOZHI-SHERPA-STREAMING-ASR-ADAPTER-001` added profile selection and an
  injected factory seam, but no default runtime helper/session process.

Target state:

- `S-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-HELPER-MAINLINE-CANDIDATE`

Action:

- Use `docs/plans/2026-06-03-sherpa-streaming-asr-runtime-helper.md`.
- Add a JSONL subprocess helper script for Sherpa streaming ASR sessions.
- Add a Go subprocess-backed `StreamingASRSessionFactory` for
  `sherpa_onnx_streaming` when helper path and model dir are configured.
- Prove with fake helper tests that `AppendFrame` can produce partial ASR
  before `Commit`, and `Commit` can produce final ASR without writing WAV.
- Preserve batch `sherpa_onnx` and keep readiness/PRD acceptance red until real
  model/provider/physical evidence exists.

Acceptance conditions:

- Plan exists and is committed: `13e9e82`.
- Worker runs in a scoped worktree with explicit no-firmware/no-provider/no-audio
  boundaries.
- Focused provider/app tests, `git diff --check`, and `make verify` pass before
  integration.
- Handoff logs state that this is runtime-helper candidate evidence only, not
  full Xiaozhi realtime acceptance.
- Worker branch
  `codex/a21-sherpa-streaming-asr-runtime-manual-20260603` adds the JSONL
  helper and subprocess session; fake-helper tests prove `AppendFrame` produces
  partial and `Commit` produces final without calling the batch WAV runner.
- Mainline fast-forward integration commit: `041ad69`.
- Mainline test hardening commit: `987a532`.
- Mainline focused tests and `make verify` pass after increasing the
  subprocess-helper test event wait to tolerate full-repo package parallelism.

Failure states:

- `F-SHERPA-STREAMING-HELPER-WAV-REGRESSION` if the streaming path writes WAV.
- `F-SHERPA-STREAMING-HELPER-SECRET-LEAK` if helper errors expose local paths,
  transcripts, provider output, URLs, proxy values, or credentials.
- `F-SHERPA-STREAMING-HELPER-HANG` if subprocess cancellation leaves pipes or
  child processes alive.
- `F-SHERPA-STREAMING-ASR-FALSE-GREEN` if static helper env is recorded as real
  model/physical PRD acceptance.

Rollback path:

- Revert helper script, subprocess factory, env wiring, tests, and docs. The
  adapter seam and batch Sherpa path remain available.

Next state:

- `S-XIAOZHI-SHERPA-STREAMING-ASR-RUNTIME-HELPER-MAINLINE-CANDIDATE`
- Next transition: real no-audio model smoke, then stock `/v1/xiaozhi`
  operator-triggered physical realtime parity.

### Completed T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001: Sherpa Streaming ASR Real-Model No-Audio Smoke

Current state:

- `S-SHERPA-STREAMING-ASR-REALMODEL-SMOKE-RECORDED`
- Local outcome: `passed` with `real_model_no_audio_streaming_smoke`.
- Report:
  `reports/a21-local-asr-streaming-smoke-20260603-085636-1780448196060587000.json`
  stores helper/model basenames, env names, booleans, event counts, and
  redaction policies only.

Trigger:

- The Sherpa streaming helper is mainline candidate, but fake helper tests do
  not prove local real-model streaming ASR can start or commit.
- Xiaozhi realtime parity requires ASR runtime proof before a physical
  `/v1/xiaozhi` turn can be classified as more than a seam/static candidate.

Target state:

- `S-SHERPA-STREAMING-ASR-REALMODEL-SMOKE-RECORDED`

Action:

- Use plan `docs/plans/2026-06-03-sherpa-realmodel-no-audio-smoke.md`.
- Add or reuse a redacted no-audio smoke report for
  `sherpa_onnx_streaming`.
- Run the JSONL helper against real local model files if present; otherwise
  record a stable model/package blocker.
- Do not write WAV, play audio, capture mic, call providers/V21, start/stop
  Gateway, flash firmware, or write NVS.
- Completed by adding `a21 local-asr-streaming-smoke` and a Make target.
  Later control integration found the repo-local canonical model cache and
  taught the smoke/provider runtime/readiness paths to discover it when explicit
  env is absent. `make local-asr-streaming-smoke` now passes locally with the
  real helper/model and redacted evidence.

Acceptance conditions:

- Smoke outcome is recorded as either real-model pass or truthful blocker.
- Report stores no transcript, raw audio, raw helper payload, credential, full
  URL, or absolute local path.
- Focused tests, `git diff --check`, and `make verify` pass before commit.
- State/log make clear this remains below physical PRD acceptance.
- Current smoke outcome is a host-local real-model pass, not PRD or physical
  realtime parity acceptance.

Failure states:

- `F-SHERPA-REALMODEL-SMOKE-WAV-BOUNDARY` if the smoke writes or reads WAV.
- `F-SHERPA-REALMODEL-SMOKE-SECRET-LEAK` if reports leak path/transcript/audio
  or secret values.
- `F-SHERPA-REALMODEL-SMOKE-FAKE-GREEN` if missing model files or missing
  sherpa-onnx package are treated as pass.

Rollback path:

- Revert the smoke CLI/report/tests/docs. Keep the streaming helper seam.

Next state:

- `S-SHERPA-STREAMING-ASR-REALMODEL-SMOKE-RECORDED`
- Next transition: authorized real streaming TTS runtime execution or physical
  stock `/v1/xiaozhi` parity trace after the selected runtime env is complete.

### Completed T-STREAMING-TTS-RUNTIME-PROOF-001: Streaming TTS Runtime Proof

Current state:

- `S-STREAMING-TTS-RUNTIME-SMOKE-RECORDED-BLOCKED-NO-EXECUTE`

Target state:

- `S-STREAMING-TTS-RUNTIME-SMOKE-RECORDED`

Trigger:

- `T-XIAOZHI-STREAMING-TTS-ADAPTER-001` added a selectable
  `doubao_tts_realtime` streaming TTS adapter seam, but no real runtime smoke
  has proven or truthfully blocked provider session start, text append, first
  audio delta, 60 ms PCM chunk framing, and session close.
- Xiaozhi realtime parity still requires evidence that selected TTS can emit
  audio before a complete WAV/file or provider EOF exists.

Actions:

- Use plan `docs/plans/2026-06-03-streaming-tts-runtime-proof.md`.
- Add a scoped app smoke such as `a21 streaming-tts-runtime-smoke` plus Make
  target.
- Default to no-provider execution and report `execute_flag_required` unless
  `--execute` is explicit.
- With missing env, report stable blocker names instead of failing obscurely or
  echoing secret values.
- With explicit `--execute` and complete env, start only the selected realtime
  TTS session, send a short A21-owned test text, observe first provider audio
  delta, count valid 60 ms PCM chunks, close the session, and write redacted
  timing/count evidence.
- Worker result: added `a21 streaming-tts-runtime-smoke` and
  `make streaming-tts-runtime-smoke`. Default local runtime report
  `reports/a21-streaming-tts-runtime-smoke-20260603-074721-1780444041851710000.json`
  is `status=blocked` with finding `execute_flag_required`; no real provider
  call, Gateway call, ASR/LLM/V21 call, `/v1/xiaozhi/say`, physical device
  path, audio playback, firmware build, flash, or NVS write occurred.

Acceptance conditions:

- Focused app/provider tests prove no-execute blocker, missing-env blocker,
  fake realtime first-audio-before-EOF behavior, no WAV/file boundary, and
  redaction.
- Runtime report stores no user transcript, provider output text, raw/base64
  audio, credentials, full URL, proxy value, or absolute path.
- If real provider execution is not authorized or fails, the transition records
  a truthful blocked/failed status and remains below PRD acceptance.
- `git diff --check` and relevant focused tests pass before integration.

Failure states:

- `F-STREAMING-TTS-RUNTIME-SECRET-LEAK` if output/report leaks credentials,
  raw/base64 audio, full URL, proxy value, user text, or absolute path.
- `F-STREAMING-TTS-RUNTIME-WAV-BOUNDARY` if the smoke writes/reads WAV or waits
  for a complete file before first chunk.
- `F-STREAMING-TTS-RUNTIME-FAKE-GREEN` if static env/config or fake provider
  tests are promoted as real provider/runtime or physical Xiaozhi acceptance.

Next candidate transition:

- Operator-authorized real Doubao realtime TTS runtime execution with
  `--execute` and complete env, still below physical `/v1/xiaozhi` acceptance.

Rollback path:

- Revert the smoke command, Make target, tests, and docs. Keep the existing
  static adapter seam and accepted physical 3x contest audio path.

Next state:

- `S-STREAMING-TTS-RUNTIME-SMOKE-RECORDED`
- Next transition: stock `/v1/xiaozhi` realtime parity proof after ASR and TTS
  runtime evidence are both available.

### Active T-PROVIDER-002b: Iflytek/Real-TTS Live Chain Unblock

Current state:

- `S4A-SELECTED-PROVIDER-READY-VOICE-CHAIN-CANDIDATE`
- `S7A-RELAY-VOICE-PHYSICAL-OPERATOR-POSITIVE`

Target state:

- `S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

Trigger:

- `T-PROVIDER-002` integrated host-side StepFun and Iflytek hot-plug code in
  commit `b5a405e`, but live Mac direct Iflytek WebSocket dial failed and
  StepFun loopback was unstable on the Mac direct path.
- The operator rejected the old local TTS source for contest dialogue quality.

Actions:

- Use `docs/plans/2026-06-03-provider-tts-live-chain-unblock.md`.
- Worker `019e8954-e65c-72a2-9847-a17a59a0ad6b` used 5080 relay first instead
  of adding a WebSocket adapter.
- Keep provider credentials host-side, reports redacted, global proxy
  unchanged, and firmware/NVS out of scope.

Current result:

- 5080 relay Iflytek TTS passed:
  `reports/provider-tts-candidate/a21-local-tts-smoke-5080-relay-20260603-0125.json`,
  first audio `100.299 ms`.
- 5080 relay StepFun `step-1-8k` plus Iflytek TTS chain passed:
  `reports/provider-tts-candidate/a21-local-voice-loopback-5080-relay-20260603-0128.json`,
  text first content `229.077 ms`, TTS first audio `87.947 ms`.
- Candidate WAVs were imported under `reports/provider-tts-candidate/`.
- Physical relay WAV playback was later completed through stock
  `/v1/xiaozhi/say` on trace
  `a21-trace-provider-playback-wav-1780450901`; operator feedback was
  positive: "好多了".
- Selected-provider readiness was refreshed without falling back to `mock`:
  `reports/provider-tts-candidate/a21-product-readiness-20260603-015100.json`
  selects route-eligible `deepseek` with `real_provider_ready=true`; bundle
  `reports/provider-tts-candidate/a21-server-side-readiness-bundle-20260603-015102.json`
  has `provider.ready=true`.
- StepFun remains an explicit compatibility voice-chain candidate, not product
  route-eligible provider readiness evidence.

Acceptance conditions:

- Physical StackChan playback uses the relay-generated StepFun+Iflytek WAV or
  equivalent live chain through the stock Xiaozhi path.
- Operator accepts or rejects the new voice with a concrete listening reason.
- StepFun remains an explicit compatibility candidate until a separate
  readiness promotion.
- Product selected-provider readiness remains pinned to a route-eligible real
  provider report unless an explicit ADR/transition promotes StepFun.

Failure state:

- `F-PROVIDER-002B-SECRETS-LEAKED` if credentials, auth URL/query, transcript,
  provider output, raw/base64 audio, proxy value, or local path enter reports
  or docs.
- `F-PROVIDER-002B-PHYSICAL-STALE` while no fresh physical socket is available
  for playback.
- `F-PROVIDER-002B-PHYSICAL-REJECTED` if operator rejects the relay voice after
  stock Xiaozhi playback.

Rollback path:

- Keep accepted 3x Gateway gain and stock firmware.
- Use `sherpa_onnx` only as emergency/diagnostic fallback, or `voice_clone_cli`
  when a configured clone wrapper is selected.

Next state:

- `S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

### Active T-PROVIDER-PLAYBACK-001: Stock Xiaozhi Relay WAV Playback

Current state:

- `S-PROVIDER-PLAYBACK-STOCK-WAV-DELIVERED-OPERATOR-POSITIVE`

Target state:

- `S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

Trigger:

- The 5080 relay produced a candidate StepFun+Iflytek WAV, but the legacy
  `/v1/devices/control` playback path is not valid evidence for stock Xiaozhi
  firmware.

Actions:

- Use
  `docs/plans/2026-06-03-stock-xiaozhi-relay-wav-playback.md`.
- Worker PLAYBACK added TDD-covered `/v1/xiaozhi/say` `wav_path` support for
  local A21-compatible 16 kHz mono PCM WAV files.
- Keep response/report surfaces basename-only and exclude raw/base64 audio,
  transcript/provider output, credentials, proxy values, and full local paths.

Current result:

- Focused Gateway test first failed RED with `400: text is required`.
- Minimal Gateway implementation now sends WAV chunks through the same stock
  TTS lifecycle, Opus downlink path, and post-say input-suppression path as
  text say.
- Gateway `21081` was restarted from this branch, the target device reconnected
  online/fresh, and runtime speaker volume `100` was delivered through stock
  MCP trace `a21-trace-relay-playback-volume-1780450901`.
- Foreground `/v1/xiaozhi/say` `wav_path` delivery succeeded for relay WAV
  basename `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav` with
  trace `a21-trace-provider-playback-wav-1780450901`, `audio_chunks=40`,
  `delivered_transport=xiaozhi_ws`, `audio_source=wav_file`, and no full path
  in the response.
- Trace `a21-trace-provider-playback-wav-1780450901` recorded
  `xiaozhi.say.start=1`, `tts.first_audio=1`,
  `audio.downlink.first_frame=1`, `xiaozhi.tts.opus_frame.downlink=40`,
  `xiaozhi.say.input_suppression_armed=1`, and `xiaozhi.say.delivered=1`.
- Operator feedback after hearing it: "好多了".

Acceptance conditions:

- `/v1/devices` shows device `44:1b:f6:e2:6a:60` online/fresh.
- The relay WAV basename `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav`
  is delivered through `/v1/xiaozhi/say` `wav_path`.
- Operator accepts or rejects the voice with a concrete listening reason.

Failure state:

- `F-PROVIDER-PLAYBACK-PHYSICAL-STALE` while no fresh physical socket is
  available for playback.
- `F-PROVIDER-PLAYBACK-LIVE-BINARY-OLD` if the live Gateway process is still
  running a pre-`wav_path` build and rejects the request before delivery.
- `F-PROVIDER-PLAYBACK-SECRET-LEAK` if a response, report, or doc captures
  full WAV paths, raw/base64 audio, transcript/provider output, credentials,
  proxy values, or other local path values beyond safe basenames.

Rollback path:

- Revert the `wav_path` handler/test/doc changes only; keep accepted 3x gain,
  stock firmware, provider hot-plug code, wake package, and half-duplex state.

Next state:

- `S-PROVIDER-TTS-REAL-DIALOGUE-CANDIDATE-RUNNING`

### Active T-HALF-DUPLEX-001: Normal Dialogue Echo Suppression Acceptance

Current state:

- `S2C-NORMAL-DIALOGUE-ACCEPTANCE-IN-PROGRESS`

Target state:

- `S-HALF-DUPLEX-NORMAL-DIALOGUE-PHYSICAL-ACCEPTED`

Trigger:

- Host-say suppression is physically observed, but normal dialogue after a real
  user turn has not proven no self-trigger after assistant TTS.

Actions:

- Use `docs/plans/2026-06-03-normal-dialogue-half-duplex-acceptance.md`.
- Worker `019e8956-7c31-78e3-bd96-f04b94f52d0b` owns safe readiness and probe
  work only.

Current result:

- Gateway `127.0.0.1:21081` is healthy.
- Device `44:1b:f6:e2:6a:60` is registered but stale, with
  `identity_status=unknown`, empty firmware identity, microphone
  `available_xiaozhi_opus_ingress`, and speaker
  `available_xiaozhi_opus_downlink`.
- `stackchan-half-duplex-acceptance` was not run because the device was not
  online/fresh and does not expose the diagnostic mic-probe capability required
  by the plan.

Acceptance conditions:

- Device is online/fresh and exposes the needed speaker/microphone evidence.
- `stackchan-half-duplex-acceptance` or normal-dialogue trace proves no
  unwanted self-trigger after TTS while preserving touch/barge-in.

Failure state:

- `F-HALF-DUPLEX-001-DEVICE-STALE` if device cannot be refreshed for live
  acceptance.
- `F-HALF-DUPLEX-001-NO-DIAGNOSTIC-MIC` if current firmware lacks the required
  mic-probe counters for instrumented acceptance.
- `F-HALF-DUPLEX-001-OVER-SUPPRESSED` if a fix makes touch/barge-in feel deaf.

Rollback path:

- Revert only half-duplex suppression tuning if it blocks user interruption.
- Keep accepted 3x gain, stock protocol, provider selection, wake firmware, and
  NVS unchanged.

Next state:

- `S-HALF-DUPLEX-NORMAL-DIALOGUE-PHYSICAL-ACCEPTED`

### Rejected T-WAKE-002: Zi Yue Wake In StackChan-Compatible App

Current state:

- `S4F-ZI-YUE-PHYSICAL-WAKE-REJECTED`

Target state:

- `S4-ZI-YUE-PHYSICAL-WAKE-PROOF`

Trigger:

- Custom wake package existed only in the bare `xiaozhi.bin` lane, but that lane
  regressed the physical product UI to plain Xiaozhi.
- The live physical firmware still needs custom wake that preserves the
  StackChan avatar/action surface.
- The operator requested the wake word be changed to `紫悦`.

Actions:

- Use `docs/plans/2026-06-03-zi-yue-wake-stackchan-compatible.md`.
- Configure the official-compatible product overlay for custom MultiNet wake:
  `zi yue` / `紫悦`, threshold `20`, MultiNet7, no wake-word data send, AFE
  WakeNet disabled, and HiStackChan WakeNet disabled.
- Preserve `a21-stackchan-official-xiaozhi-compatible.bin`, official StackChan
  app surface, and `GetHAL().startXiaozhi()`.
- Build the product candidate, inspect `sdkconfig.json`, run no-write flash
  plan, then use only the official-compatible guarded flash execute path.

Current result:

- Focused app tests passed for the official-compatible overlay, `紫悦` wake
  contract, and product flash-lane guard.
- `make verify` passed.
- First ESP-IDF attempt failed at Python 3.13 `_csv` dynamic-library system
  policy during `gen_crt_bundle.py`; manual replay and a second build passed,
  so this is recorded as an environment hiccup, not a firmware/config failure.
- Build report
  `reports/a21-stackchan-official-baseline-20260603-030428-1780427068064697000.json`
  is `status=passed`; app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  has SHA-256
  `3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`.
- Build config inspection proved:
  `BOARD_TYPE_M5STACK_STACK_CHAN=true`,
  `BOARD_TYPE_M5STACK_CORE_S3=false`,
  `USE_CUSTOM_WAKE_WORD=true`,
  `CUSTOM_WAKE_WORD="zi yue"`,
  `CUSTOM_WAKE_WORD_DISPLAY="紫悦"`,
  `CUSTOM_WAKE_WORD_THRESHOLD=20`,
  `SEND_WAKE_WORD_DATA=false`,
  `WAKE_WORD_DETECTION_IN_LISTENING=false`,
  `USE_AFE_WAKE_WORD=false`,
  `SR_MN_CN_MULTINET7_QUANT=true`, and
  `SR_WN_WN9_HISTACKCHAN_TTS3=false`.
- No-write flash plan
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030447-1780427087258380000.json`
  is `status=ready`, `dry_run=true`, `flash_allowed=false`,
  `flash_executed=false`, app `a21-stackchan-official-xiaozhi-compatible.bin`,
  offset `0x20000`, port `/dev/cu.usbmodem1101`.
- Guarded flash execute
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-030818-1780427298506934000.json`
  passed on `/dev/cu.usbmodem1101` with control commit `7f3225ee1fe7`,
  clean worktree, app `a21-stackchan-official-xiaozhi-compatible.bin`, app
  offset `0x20000`, and app SHA-256
  `3f7dcd291a2efb586f11aa3b6c6ca3003cae0d53be45225854502ccd0e6fa4cf`.
- Post-flash Gateway `127.0.0.1:21081` remained healthy; device
  `44:1b:f6:e2:6a:60` reconnected online at 03:08:51 with volume `100`, trace
  `a21-trace-44-1b-f6-e2-6a-60`.
- Operator physical verification failed: saying `紫悦` produced no response.

Acceptance conditions:

- Guarded flash execute passes with app
  `a21-stackchan-official-xiaozhi-compatible.bin` at offset `0x20000`.
- Physical device reboots into the StackChan-compatible avatar/action UI, not
  plain Xiaozhi.
- Operator says `紫悦` and the device wakes without screen touch. This was not
  met.
- If `紫悦` misses or false-wakes, the next transition tunes only custom wake
  phrase/threshold while keeping the product app lane.

Failure state:

- `F-WAKE-002-UNGUARDED-WRITE` if any worker performs a background flash/NVS
  write.
- `F-WAKE-002-BARE-XIAOZHI-REGRESSION` if product hardware is flashed with
  `xiaozhi.bin` again.
- `F-WAKE-002-STACKCHAN-UI-REGRESSION` if the flashed app shows plain Xiaozhi
  rather than StackChan-compatible UI.
- `F-WAKE-002-PHYSICAL-WAKE-REJECTED` if the operator cannot wake it by saying
  `紫悦`.

Rollback path:

- Reflash the last accepted official-compatible app artifact through
  `a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- If only wake sensitivity is wrong, keep the same product app lane and tune
  `CONFIG_CUSTOM_WAKE_WORD_THRESHOLD` from `20` toward `35`.
- Preserve NVS connection settings unless a rollback plan explicitly requires
  reprovisioning.

Next state:

- `S4F-ZI-YUE-PHYSICAL-WAKE-REJECTED`

### Completed T-WAKE-INTEGRATE-001: Custom Wake Bare Flash Guard And Recovery

Current state:

- `S2D-CUSTOM-WAKE-PACKAGE-BARE-FLASH-INVALID-ROLLBACK-DONE`

Target state:

- `S2E-CUSTOM-WAKE-BARE-FLASH-GUARDED`

Trigger:

- Custom wake package existed, but the live physical firmware used stock
  Xiaozhi WakeNet/touch activation and the operator reported weak wake
  response.
- A foreground attempt to flash the restored bare wake build succeeded
  technically but restored plain Xiaozhi UI, proving that raw `xiaozhi.bin`
  is the wrong product artifact for StackChan.

Actions:

- Use `docs/plans/2026-06-03-guarded-wake-flash-physical-proof.md`.
- Worker `019e8993-0987-7601-9b8a-3aa4e88ebfaa` owns only the small flash-lane
  guard hotfix; it must not run hardware writes.

Current result:

- Wake package integrity passed below activation:
  `reports/a21-wake-word-firmware-package-20260602-075112-1780357872711914000.json`.
- Artifact exists and matches SHA-256
  `1815bda17a9ec5dcd0052e86526670ab17f3c5bff1e11944f5ac3731ea8e349b`.
- Bare wake flash execution
  `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json`
  passed but is now incident evidence only because it flashed app file
  `xiaozhi.bin`.
- Corrective StackChan-compatible flash execution
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json`
  restored app file `a21-stackchan-official-xiaozhi-compatible.bin`.
- Post-restore volume/TTS evidence passed on traces
  `a21-trace-recovery-stackchan-volume-1780424386012` and
  `a21-trace-recovery-stackchan-relay-wav-1780424386012`.

Acceptance conditions:

- No future product StackChan runbook uses `xiaozhi-firmware-flash-execute` or
  app file `xiaozhi.bin`.
- Generic `xiaozhi-firmware-flash-*` rejects product-looking `xiaozhi.bin`
  unless explicitly marked `--non-product-dev`.
- Corrective StackChan-compatible flash path is documented and verified.

Failure state:

- `F-FW-003-UNGUARDED-WRITE` if any background flash/NVS write is attempted.
- `F-FW-003-BARE-XIAOZHI-PRODUCT-FLASH` if a product StackChan device is
  flashed again with `xiaozhi.bin`.
- `F-FW-003-PACKAGE-STALE` if future wake assets cannot be reconciled with the
  accepted audio/official-compatible baseline.

Rollback path:

- Reflash the last accepted official-compatible app artifact through a guarded
  foreground path if custom wake regresses behavior.
- Preserve NVS connection settings unless a rollback plan explicitly requires
  reprovisioning.

Next state:

- `S2E-CUSTOM-WAKE-BARE-FLASH-GUARDED`

### Completed T-PROVIDER-002: Real Provider/TTS Hot-Plug Candidate

Current state:

- `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`
- `S3C-STEPFUN-IFLYTEK-HOTPLUG-CODED`

Result:

- Commit `b5a405e` added host-side `iflytek_tts`, StepFun explicit
  compatibility selection, voice-pipeline TTS selection, and retained
  `voice_clone_cli`.
- Focused tests and `make verify` passed before this follow-up transition.
- Mac direct Iflytek TTS remained blocked, which created active
  `T-PROVIDER-002b`.

Next state:

- `S3D-STEPFUN-IFLYTEK-LIVE-UNBLOCK-IN-PROGRESS`

### Completed T-INTEGRATE-001: Review And Integrate Accepted Audio Hotfixes

Current state:

- `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`

Target state:

- `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`

Trigger:

- Operator accepted the 3x physical audio result after recording
  `/Users/jiyurun/Downloads/军民公路259号 10.m4a`.

Actions:

- Review the T-AUDIO-002 and T-AUDIO-003 hotfix diff for stock-protocol,
  audio-leveling, session/trace, and half-duplex boundaries.
- Update project state and handoff docs with accepted recording evidence.
- Run focused Go tests, desktop helper syntax checks, `git diff --check`, and
  live Gateway/device status checks.
- Commit the accepted hotfixes into the current integration branch if review
  finds no blocking issue.

Acceptance conditions:

- No blocking review findings.
- Verification commands pass.
- Commit captures code, tests, plans, protocol docs, state machine, and handoff
  log.
- Integration commit has been created on the current branch.

Failure state:

- `F-INTEGRATE-001-BLOCKING-REVIEW-FINDING` if review finds a protocol,
  session, safety, or acceptance-labeling issue.
- `F-INTEGRATE-001-VERIFY-FAILED` if tests or syntax/diff checks fail.

Rollback path:

- Do not commit; keep the dirty branch available for targeted fixes.

Next state:

- `S-HW-PHYSICAL-XIAOZHI-AUDIO-HOTFIX-INTEGRATED`

### Previous T-AUDIO-003: Bounded TTS Gain And Host-Say Input Suppression

Current state:

- `S5-OPERATOR-LOUDNESS-AND-CLARITY-ACCEPTED`

Target state:

- `S5-OPERATOR-LOUDNESS-AND-CLARITY-ACCEPTED`

Trigger:

- Operator recording `9.m4a` after `/v1/xiaozhi/say` remained too quiet:
  from 3 seconds it measured `-36.99 LUFS`, `-16.05 dBTP`; best early
  10-second window measured `-34.79 LUFS`, `-16.05 dBTP`.
- Sidecar analysis found the Gateway downlink PCM path only attenuated
  full-scale frames and never lifted quiet TTS with available headroom.
- Half-duplex sidecar found `/v1/xiaozhi/say` did not arm input suppression,
  so speaker playback could be captured by the microphone and retrigger listen
  or voice-pipeline flow.

Actions:

- Use `docs/plans/2026-06-02-stackchan-tts-gain-half-duplex-hotfix.md`.
- Replace pure headroom limiting with bounded downlink PCM leveling:
  tiny noise below gate is unchanged, quiet non-silent TTS is lifted toward a
  target peak with max-gain cap, and hot frames remain below headroom.
- Keep the live stock protocol unchanged.
- Add host-say-only short input suppression for stock physical devices.
- Keep normal dialogue half-duplex acceptance separate.

Acceptance conditions:

- Focused tests prove quiet TTS frames are boosted, tiny noise is not boosted,
  and full-scale frames remain capped.
- Focused test proves immediate listen restart plus speech Opus after host-say
  is ignored for stock physical devices without starting a new voice pipeline.
- Physical recording after the gain hotfix improves materially without obvious
  clipping, but listening quality must remain acceptable. 4x candidate evidence
  `/Users/jiyurun/Downloads/纳仕张江国际社区云庐B区.m4a` measured `-21.65 LUFS` and
  `-4.72 dBTP` from 3 seconds but was rejected by operator listening feedback.
  The current 3x candidate is expected to preserve clarity with less peak
  stress; follow-up recording
  `/Users/jiyurun/Downloads/浦东新区第二中心小学(申江校区) 3.m4a` measured
  `-26.4 LUFS` and `-9.0 dBFS` true peak from 3 seconds.
- Final 3x Gateway trace `a21-trace-stackchan-say-1780417217` delivered
  `579` TTS chunks to the live physical stock socket after volume `100` trace
  `a21-trace-stackchan-volume-1780417211`.
- Operator accepted final recording
  `/Users/jiyurun/Downloads/军民公路259号 10.m4a`, which measured
  `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds.

Failure state:

- `F-AUDIO-003-CLIPPING-OR-HARSHNESS` if operator reports harshness,
  electrical interruption, or the next recording shows clipping.
- `F-AUDIO-003-STILL-QUIET` if physical loudness remains unacceptable after
  bounded Gateway gain.
- `F-AUDIO-003-SELF-TRIGGER` if physical host-say still retriggers voice flow.

Rollback path:

- Remove or lower the bounded PCM leveling constants while leaving stock
  protocol, firmware, provider, and V21 untouched.
- Remove the host-say suppression helper if it blocks legitimate operator
  foreground tests.
- If safe 3x Gateway gain is insufficient, move to guarded firmware codec-gain
  or persistent volume work instead of stacking more host gain.

Next state:

- `S5-OPERATOR-LOUDNESS-AND-CLARITY-ACCEPTED`

### Previous T-AUDIO-002: Stock Xiaozhi Audio Parity And Runtime Volume Hotfix

Current state:

- `S3-LIVE-HOTFIX-DEPLOYED-AND-LONG-TTS-SENT`

Target state:

- `S4-PHYSICAL-STOCK-AUDIO-RETESTED`

Trigger:

- The operator recording after the volume-92 flash still sounded blurred,
  intermittent, and unclear.
- Parallel read-only workers found that official StackChan Xiaozhi behavior is
  not just "same protocol": client/uplink is 16 kHz, server/downlink/CoreS3
  output is 24 kHz, stock firmware does not accept `listen` replies, official
  volume is exposed through MCP, and A21 was rebuilding Opus downlink encoders
  per frame.

Actions:

- Use `docs/plans/2026-06-02-stackchan-audio-official-parity-hotfix.md`.
- Keep client/uplink validation at 16 kHz while restoring server/downlink TTS
  to 24 kHz mono 60 ms.
- Suppress unsupported `listen` replies for stock physical MAC-address devices.
- Reuse the turn-level Opus encoder for contiguous same-format downlink frames
  and reduce initial prebuffer from five frames to one frame.
- Add `POST /v1/xiaozhi/speaker-volume` as a stock MCP
  `self.audio_speaker.set_volume` delivery path.
- Add `POST /v1/xiaozhi/say` as an operator foreground path that sends stock
  TTS lifecycle plus Opus binary downlink over the live `/v1/xiaozhi` socket.
- Update the desktop StackChan control helper to call the runtime volume
  endpoint and host-say endpoint.
- Do not flash firmware, write NVS, execute V21, or claim physical acceptance
  in this transition.

Acceptance conditions:

- Focused Gateway/provider/transport/app tests pass.
- `git diff --check` passes.
- Desktop helper syntax checks pass.
- A Gateway built from this tree can set volume `100` on the live stock
  Xiaozhi socket, then play a long TTS turn for physical recording. This has
  been demonstrated by trace `a21-trace-live-long-tts-hotfix`.
- Physical acceptance remains red until the operator/instrument recording is
  analyzed and logged.

Failure state:

- `F-AUDIO-002-MCP-NOT-SUPPORTED` if the live device does not advertise
  `features.mcp=true` or does not react to the official volume tool.
- `F-AUDIO-002-PHYSICAL-STILL-BROKEN` if post-hotfix audio remains blurred or
  intermittent, shifting focus to TTS model/profile and device-side playback
  instrumentation.
- `F-AUDIO-002-REGRESSION` if focused playback/barge-in tests fail.

Rollback path:

- Revert the Gateway/provider/transport patches and keep the already flashed
  firmware unchanged.
- If runtime volume delivery behaves unexpectedly, stop using
  `/v1/xiaozhi/speaker-volume` and resume the guarded firmware volume-100
  candidate path.

Next state:

- `S4-PHYSICAL-STOCK-AUDIO-RETESTED`

### Candidate T-HW-VOLUME-002: Raise StackChan Fixed Output After Volume92 A/B Failed

Current state:

- `S3B-VOLUME92-A-B-FAILED`

Target state:

- `S4-VOLUME100-CANDIDATE-BUILD-FLASH-A-B`

Trigger:

- The user clarified the target is StackChan device loudness, not macOS system
  volume.
- The latest phone recording is stronger than the previous one but still has
  low sustained loudness and narrow active speech spectrum.
- Current stock `/v1/xiaozhi` has no Gateway runtime speaker-volume setter, so
  the fastest honest path is a guarded firmware-side output-gain candidate.
- Worker thread `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9` has prepared the fixed
  official codec output-volume patch for main-thread review.
- Main-thread no-write build/report has passed for the patched official
  Xiaozhi-compatible candidate.
- Main-thread no-write flash-plan report is ready, and read-only flash-gate
  worker `019e88d6-1734-75d2-897b-aa2da3069885` confirmed the artifact hash,
  USB port, dry-run safety fields, and rollback baseline.
- The operator explicitly opened the foreground hardware window and supplied the
  guarded execute command with confirmation token.
- Main-thread flash execute passed with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`.
- Post-flash recording `/Users/jiyurun/Downloads/军民公路259号 7.m4a`
  measured `-33.4 LUFS` integrated loudness and `-14.3 dBFS` true peak,
  weaker than the 22:19 reference at `-25.8 LUFS` and `-4.2 dBFS`.
- The analysis report
  `reports/a21-operator-recording-audio-analysis-20260602-2310-stackchan-volume92.json`
  records `post_flash_loudness_accepted=false` and no clipping evidence.

Actions:

- Use `docs/plans/2026-06-02-stackchan-volume-action-control.md`.
- Keep the main thread as architecture/control only.
- Keep the failed volume-92 flash execute report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json`
  as the rollback/comparison evidence.
- Dispatch a scoped worker to raise the official Xiaozhi-compatible fixed
  output candidate to `SetOutputVolume(100)` and verify the official codec
  `SetOutputVolume` ordering without touching hardware.
- Main thread reviews the worker commit, runs no-write build and flash-plan,
  then opens a foreground flash window only after operator approval.
- After any volume-100 flash, trigger a real stock Xiaozhi turn, collect a
  post-flash phone recording, and compare it against both `7.m4a` and the
  22:19 reference sample.
- Do not add runtime volume protocol, do not use macOS volume as evidence, do
  not use legacy local audio playback, do not write NVS, do not execute
  provider/V21, and do
  not promote diagnostic tone or host-only evidence to PRD acceptance.

Acceptance conditions:

- The official Xiaozhi-compatible overlay explicitly sets official codec output
  volume `100` before entering the Xiaozhi runtime.
- Existing `GetHAL().startXiaozhi()` behavior remains preserved.
- Focused guard/test or build-report evidence proves the volume setting exists
  in the candidate artifact path.
- A no-write official-compatible build/report passes for the volume-100
  candidate before any flash is requested.
- A no-write flash plan is ready and keeps `flash_allowed=false` until the
  operator explicitly runs the guarded execute command with the confirmation
  token.
- Foreground flash execute and before/after audible A/B are recorded before
  physical loudness is accepted.
- `git diff --check` passes, and any touched Go guard tests pass.
- State and handoff docs record that physical before/after audibility remains
  unaccepted until foreground flash plus phone/instrument A/B.

Failure state:

- `F-HW-VOLUME-001-NO-OFFICIAL-CODEC-SEAM` if the worker cannot locate a safe
  official codec setting point.
- `F-HW-VOLUME-001-SCOPE-DRIFT` if runtime protocol, Gateway behavior,
  provider/V21, NVS, flash, or macOS audio is touched outside the plan.
- `F-HW-VOLUME-001-PHYSICAL-OVERCLAIM` if the code candidate is treated as
  product voice acceptance without foreground physical evidence.
- `F-HW-VOLUME-002-NO-AUDIBLE-GAIN` if volume 100 also fails to improve the
  physical recording, in which case control returns to `T-AUDIO-001` TTS/model
  and playback-chain RCA.

Rollback path:

- Revert the overlay/test/doc patch if the candidate build or guard fails.
- Keep the currently flashed firmware and NVS route unchanged until an explicit
  hardware window approves flash.
- If a flashed volume trial sounds worse, flash back to the last accepted
  official-compatible app SHA recorded in the firmware flash reports.

Next state:

- `S4-VOLUME100-CANDIDATE-BUILD-FLASH-A-B`

### T-AUDIO-001: Isolate Xiaozhi TTS Quality From Opus/Device Playback

Current state:

- `S1C-STOCK-XIAOZHI-OPERATOR-RECORDING-PENDING`

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
- Run host-only product-chain bench against current A21 Gateways to verify
  post-Opus quality before touching physical firmware or TTS profile routing.
- Avoid hardware writes, NVS writes, provider/V21 execution, unapproved Mac
  audio playback, and PRD overclaiming.
- After worker return, run a foreground physical A/B only if the operator
  approves any Gateway/TTS profile change.
- For the current stock Xiaozhi physical connection, do not use
  `stackchan-local-tts-playback` as evidence: it requires the legacy A21 audio
  WebSocket and returned `409 device audio websocket is not connected` during
  the 2026-06-02 foreground recording window.
- Collect the next physical audible sample by triggering a real stock Xiaozhi
  listen/audio turn on the device, or first create a separate planned
  transition for a stock-safe TTS injection seam.

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
| T-AUDIO-001b: Host/Gateway post-Opus quality run | Completed host-only candidate | `reports/a21-xiaozhi-voice-bench-20260602-220325.719331000.json` passed 3-repeat host product-chain bench with `sherpa_onnx_tts`, answer p95 397 ms, and post-Opus quality passed; `reports/a21-xiaozhi-voice-bench-20260602-220344.175240000.json` passed a one-round check on the physical LAN Gateway; readiness `reports/a21-product-readiness-20260602-220447.json` and bundle `reports/a21-server-side-readiness-bundle-20260602-220501.json` still correctly block launch. |
| T-AUDIO-001c: Foreground long-TTS push attempt | Blocked, evidence-preserving | macOS output volume was set to 100 after explicit operator request; physical device `44:1b:f6:e2:6a:60` was online on stock Xiaozhi Opus via Gateway `21081`; `go run ./cmd/a21 stackchan-local-tts-playback --gateway-url http://127.0.0.1:21081 --device-id 44:1b:f6:e2:6a:60 --engine sherpa_onnx ...` returned `409 device audio websocket is not connected`, so no valid physical StackChan playback was claimed. |
| T-OPS-001: Desktop StackChan control boundary helper | Completed | Added `tools/desktop/a21-stackchan-control.command` and copied it to `/Users/jiyurun/Desktop/A21-StackChan-Control.command`; status check passed against Gateway `21081`; action probes and diagnostic tone correctly reported current stock-session 409 blockers instead of claiming control. |
| T-HW-VOLUME-001a: Fixed official codec volume candidate prep/build | Completed no-hardware candidate | Worker thread `019e88c3-8fa7-7e53-8e46-ab3ff6e637b9` prepared official codec `SetOutputVolume(92)` in the Xiaozhi-compatible overlay before `GetHAL().startXiaozhi()` and added a Go guard test; main branch integrated the patch, focused Go tests, official StackChan Go subset, clean official `HEAD` patch-apply check, `git diff --check`, and no-write build report `reports/a21-stackchan-official-baseline-20260602-225207-1780411927040145000.json` passed. App SHA-256: `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`. |
| T-HW-VOLUME-001b: Fixed official codec volume flash gate | Completed no-write gate | No-write flash plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-225600-1780412160509265000.json` is `status=ready`, `port=/dev/cu.usbmodem1101`, `dry_run=true`, `flash_allowed=false`, `flash_executed=false`, and app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`; read-only worker `019e88d6-1734-75d2-897b-aa2da3069885` confirmed it is enough to request an operator-approved hardware window, not enough for physical acceptance. Read-only worker `019e88d6-ad61-7513-b379-aa21ae7db150` prepared the A/B evidence runbook and confirmed `stackchan-local-tts-playback`, Mac volume, diagnostic tone, host-only bench, and dry-run reports are not valid physical loudness acceptance. Rollback baseline: `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-204606-1780404366296223000.json`, app SHA-256 `8a759546961f5244622d8a1ebd9cbfc92274893bbe0ce0bc460922eb2490dd6d`. |
| T-HW-VOLUME-001c: Foreground fixed codec volume flash | Completed hardware write, A/B pending | Operator supplied guarded execute command; flash report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260602-230350-1780412630917666000.json` is `status=passed`, `port=/dev/cu.usbmodem1101`, `dry_run=false`, `flash_allowed=true`, `flash_executed=true`, app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`. Post-flash Gateway `127.0.0.1:21081` health returned ok and device `44:1b:f6:e2:6a:60` remained registered online. This is not physical loudness acceptance. |
| T-FW-003a: Bare wake flash incident and StackChan-compatible recovery | Completed recovery | Bare wake flash execution `reports/a21-xiaozhi-firmware-flash-20260603-021354-1780424034336886000.json` passed generic T7 but flashed app `xiaozhi.bin`, reverting the physical device to plain Xiaozhi UI. Corrective no-write plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021802-1780424282862523000.json` and execute report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-021849-1780424329771759000.json` restored `a21-stackchan-official-xiaozhi-compatible.bin` with app SHA-256 `2b42e91226a4e21538999a283312d1754882e1654cbf6fc20115f6210ce4864e`. Post-restore runtime volume and relay WAV playback succeeded on traces `a21-trace-recovery-stackchan-volume-1780424386012` and `a21-trace-recovery-stackchan-relay-wav-1780424386012`. |
| T-FLASH-GUARD-001: Product artifact-lane guard | Completed | Generic `xiaozhi-firmware-flash-*` now rejects product-looking app `xiaozhi.bin` at `0x20000` unless explicitly marked `--non-product-dev`; rejection points to `a21-stackchan-official-xiaozhi-compatible-flash-execute`. Real incident build no-write plan now exits nonzero with the guard message. Correct product no-write plan still passes in `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-023008-1780425008542994000.json`. Focused app tests, `git diff --check`, and `make verify` passed. |
| T-PROTOCOL-001a: Stock Xiaozhi listen warning plan-first audit | Completed plan | Worker `019e88dd-1517-77c2-a002-7db771ec016a` created `docs/plans/2026-06-02-stock-xiaozhi-protocol-cleanup.md`; strongest hypothesis is Gateway sends server-to-device `type=listen` ack frames that stock firmware does not accept, while binary Opus/TTS downlink still reaches `speaking`. |
| T-PROVIDER-001a: Provider readiness truth audit | Completed read-only audit | Worker `019e88dd-3efa-78e2-a7d3-7063089cbf30` found `T-PROVIDER-001` stale as a missing-smoke blocker. Usable explicit provider evidence exists: DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json`, local Ollama smoke `reports/a21-provider-smoke-20260602-075644-199710000.json`, provider-ready readiness `reports/a21-product-readiness-20260602-160841.json`, and server-side bundle `reports/a21-server-side-readiness-bundle-20260602-160841.json`. Newer generic readiness reports that selected `mock` are invocation/context regressions, not absence of provider evidence. |
| T-PROVIDER-001b: Selected provider readiness refresh | Completed, server-side still blocked by non-provider gates | Worker `019e8974-346e-7912-93b2-77cdbb9f3acf` pinned readiness to route-eligible DeepSeek smoke `reports/a21-provider-smoke-20260602-112710-368364000.json`; generated `reports/provider-tts-candidate/a21-product-readiness-20260603-015100.json` with `provider.selected=deepseek`, `real_provider_ready=true`, and `smoke_status=passed`; generated `reports/provider-tts-candidate/a21-server-side-readiness-bundle-20260603-015102.json` with `provider.ready=true`. Reports remain `server_side_blocked` because V21, host voice/continuous pipeline, physical StackChan, and wake-word gates are still open. |
| T-AUDIO-002: Stock Xiaozhi audio parity and runtime volume hotfix | Completed physical candidate | Restored 24 kHz downlink while keeping 16 kHz uplink, suppressed unsupported stock physical `listen` replies, reused turn-level Opus encoder, reduced prebuffer to one frame, added stock MCP speaker volume endpoint, and added foreground `/v1/xiaozhi/say`; focused Gateway/provider/transport/app tests passed. |
| T-AUDIO-003: Bounded 3x TTS gain and host-say suppression | Completed accepted | 4x gain was rejected by operator listening feedback; final 3x gain delivered `a21-trace-stackchan-say-1780417217`, final accepted recording `/Users/jiyurun/Downloads/军民公路259号 10.m4a` measured `-26.5 LUFS` and `-8.6 dBFS` true peak from 3 seconds, and host-say suppression markers were observed. |
| T-AUDIO-004a: Bare Xiaozhi audio and whole-device parity read-only audit | Completed read-only audit | Sidecar thread `019e899d-e8c2-71e0-a10c-9e7e34ac4cff` reported the operator-confirmed fact that bare `xiaozhi.bin` was louder and clearer, while keeping it classified as a non-product/incident artifact. The explanation is not protocol alone: the key differences are official CoreS3 codec/HAL behavior, source TTS/mastering, runtime volume/NVS/MCP state, bounded leveling, gopus downlink, pacer/encoder state, and official StackChan app/action initialization. It recommends migrating parity conditions into `a21-stackchan-official-xiaozhi-compatible`, not returning to the bare `xiaozhi.bin` or the old M5Unified PCM bridge. |
| T-HALF-DUPLEX-002: No-flash normal dialogue self-trigger observation | Completed candidate | Main thread used the online stock Xiaozhi device without firmware flash or operator click, delivered relay WAV `a21-stepfun-iflytek-chain-5080-relay-20260603-0128.wav` through `/v1/xiaozhi/say`, and generated `reports/a21-no-flash-normal-dialogue-observation-20260603-020055.json` with `status=candidate_passed_no_self_trigger`, trace `a21-trace-no-flash-dialogue-observe-1780423245`, `audio_chunks=40`, `wait_after_say_ms=20000`, `event_count=869`, empty `self_trigger_event_names`, and `input_suppressed_count=1`. Diagnostic half-duplex counters remain a separate optional firmware path. |
| T-ASR-GREEN-LATENCY-001: Bound Xiaozhi listen from firmware | Flashed, physical validation pending | Physical trace `a21-trace-44-1b-f6-e2-6a-60` showed `xiaozhi.listen.start=110`, `xiaozhi.opus_frame.received=8034`, repeated `listen.stop -> listen.start`, and wake disabled while listening. Root cause candidate: official `HandleStartListeningEvent()` forced `kListeningModeManualStop`, while the A21 no-speech timeout only armed for `kListeningModeAutoStop`. The overlay now routes StartListening through `GetDefaultListeningMode()`, starts the no-speech timer for non-realtime listening, and keeps VAD silence auto-stop scoped to AutoStop after speech. Focused tests, `make verify`, and product build passed; build report `reports/a21-stackchan-official-baseline-20260603-042940-1780432180247638000.json`, app SHA-256 `ca0877d09eecfb9c69f2279ce14c95942a2cf966e05f118e82551e92c14665b9`. No-write plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-043116-1780432276384588000.json` passed, and guarded flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-043222-1780432342953949000.json` passed on `/dev/cu.usbmodem1101` from clean commit `88cb8a92069d`. Post-flash Gateway evidence: device `44:1b:f6:e2:6a:60` reconnected online, runtime volume `100` was delivered by stock MCP trace `a21-trace-listen-bound-volume-1780432365`, and post-flash trace events since `1780432350000` contained only `xiaozhi.hello.received=1` with no automatic `listen.start`. |
| T-WAKE-003b: Split Zi Yue MultiNet command list | Flashed, physical validation pending | Source inspection found `CUSTOM_WAKE_WORD` is documented as one pinyin command, while `CustomWakeWord::Initialize()` adds each command through `esp_mn_commands_add`. The earlier `zi yue|zi yue zi yue|ni hao zi yue|xiao zi yue` config therefore likely registered as one invalid/overlong command instead of four alternatives. The overlay now splits `CONFIG_CUSTOM_WAKE_WORD` on `|`, trims each command, and registers each as a separate wake command with display/greeting `紫悦`. Focused tests, `make verify`, and official-compatible build passed; build report `reports/a21-stackchan-official-baseline-20260603-043851-1780432731137364000.json`; app SHA-256 `a0638a3b9872c98456c4dee9c8503dfa6d6d27f2374b499fe7c36033aeccb575`. Binary strings include `Loaded %d A21 custom wake command(s) for %s` and the multi-phrase config. Commit `8e4df0b` was flashed through no-write report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-044128-1780432888876682000.json` and guarded execute report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-044234-1780432954645477000.json` on `/dev/cu.usbmodem1101`; T7 guard saw clean commit `8e4df0b30c0f`, app SHA `a0638a3b9872c98456c4dee9c8503dfa6d6d27f2374b499fe7c36033aeccb575`. Device `44:1b:f6:e2:6a:60` reconnected online, runtime volume `100` delivered by trace `a21-trace-multi-wake-volume-1780432980`, and post-flash trace since `1780432954000` showed only `xiaozhi.hello.received=1` with no automatic `listen.start`. |
| T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001b: Park after direct Xiaozhi start | Current HEAD reflashed, visual validation pending | Operator reported the welcome/setup page reappeared after the multi-wake product flash even though the binary contained the direct autostart log. Source inspection proved the false assumption: `Hal::startXiaozhi()` starts Xiaozhi tasks and returns, so `main.cpp` could continue into the Mooncake main loop and AppLauncher setup worker. The overlay now parks `app_main` after direct `GetHAL().startXiaozhi()` with `GetHAL().feedTheDog()` and `GetHAL().delay(1000)`, before the Mooncake main loop. Focused app tests, `git diff --check`, and `make verify` passed in the original hotfix. Product build report `reports/a21-stackchan-official-baseline-20260603-054837-1780436917458986000.json` passed with app SHA-256 `e3758cffdc294e74220f5288a33306d4aace164279bea6acb3a2cea5af17e1c3`; guarded flash execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-054954-1780436994114555000.json` passed on `/dev/cu.usbmodem1101` from clean commit `e694550f1739`. After later provider commits, the control tower rebuilt and flashed current HEAD `7906975`: build `reports/a21-stackchan-official-baseline-20260603-062637-1780439197207182000.json`, flash plan `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062715-1780439235852990000.json`, and execute `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260603-062853-1780439333539408000.json` all passed; app SHA-256 is `fc7788736ced71c98cee846892a867d306d663c5ceb31014cc01079a381e766c`. Device `44:1b:f6:e2:6a:60` reconnected online, runtime speaker volume `100` was delivered by trace `a21-trace-current-head-volume-1780439333`, and post-flash trace events since `1780439333539` contain only `xiaozhi.hello.received=1` with no automatic `listen.start`. |
| T-ASR-GREEN-LATENCY-002: Gateway no-speech cooldown | Completed runtime hotfix, physical validation pending | Live trace before the fix showed `xiaozhi.listen.start=333`, `audio.ingress.buffered=25115`, `stackchan.official_auto.not_connected=1025`, and repeated `listen.stop -> placeholder tts.stop -> listen.start` loops. Gateway now arms a short stock-physical input suppression after `placeholder_no_asr_tts`, recording `xiaozhi.no_speech.input_suppression_armed` and reason-specific `xiaozhi.listen.start.suppressed_after_no_speech` so an immediate restart is ignored instead of opening a new turn. Focused Gateway tests passed, `make verify` passed, Gateway `a21-gateway-21081` was restarted from this worktree, `/healthz` returned `service=a21-gateway,status=ok`, device `44:1b:f6:e2:6a:60` reconnected online, runtime volume `100` was delivered by `a21-trace-no-speech-cooldown-final-volume-1780434593`, and live trace `a21-trace-44-1b-f6-e2-6a-60` later showed the cooldown firing once: `xiaozhi.no_speech.input_suppression_armed=1`, `xiaozhi.listen.start.input_suppressed=1`, `xiaozhi.listen.start.suppressed_after_no_speech=1`, with only one `xiaozhi.turn.start`. |
| T-XIAOZHI-STREAMING-ASR-PROVIDER-001a: Static provider readiness gate | Completed truthful blocker | Plan `docs/plans/2026-06-03-xiaozhi-streaming-provider-readiness.md` defines the strict provider requirements for Xiaozhi realtime parity. New CLI `a21 xiaozhi-streaming-provider-readiness` is static/no-execute and blocks mock, batch WAV, and file-boundary paths. Default report `reports/a21-xiaozhi-streaming-provider-readiness-20260603-055502.json` is `gate_status=blocked` with mock ASR/LLM/TTS findings. Selected StepFun+Iflytek check `reports/a21-xiaozhi-streaming-provider-readiness-20260603-055512.json` is also `gate_status=blocked`: LLM text stream is ready, but Sherpa ASR is `asr_batch_wav_boundary_not_xiaozhi_streaming` and Iflytek TTS is `tts_wav_file_boundary_not_xiaozhi_streaming`. This prevents mock `/say`, `streaming_zipformer` name-only, or WAV TTS evidence from being promoted to Xiaozhi realtime parity. |
| T-XIAOZHI-SHERPA-STREAMING-ASR-ADAPTER-001: Selectable Sherpa streaming ASR seam | Completed adapter seam, runtime helper not proven | Plan `docs/plans/2026-06-03-sherpa-streaming-asr-adapter.md` scoped the work. Added `sherpa_onnx_streaming` / `local_sherpa_onnx_streaming` / `streaming_zipformer` selection for a `providers.StreamingASRAdapter` wrapper that keeps the old batch Sherpa adapter as fallback and starts streaming only through a configured session factory. Existing `sherpa_onnx` remains batch/WAV and is still blocked as `asr_batch_wav_boundary_not_xiaozhi_streaming`. New readiness reports prove the distinction: `reports/a21-xiaozhi-streaming-provider-readiness-20260603-060837-1780438117185327000.json` blocks missing streaming helper/model without WAV-boundary findings, while `reports/a21-xiaozhi-streaming-provider-readiness-20260603-060837-1780438117326052000.json` marks ASR+LLM ready when helper/model env are present but still blocks on `tts_wav_file_boundary_not_xiaozhi_streaming`. Focused provider/app tests, `git diff --check`, and `make verify` passed. No firmware, provider execution, service restart, flash, or audio playback occurred. |
| T-SHERPA-REALMODEL-NO-AUDIO-SMOKE-001: Sherpa streaming ASR real-model no-audio smoke | Completed host-local real-model pass | `local-asr-streaming-smoke` now discovers the repo-local canonical helper, sherpa-onnx Python env, and `sherpa-onnx-streaming-zipformer-zh-int8-2025-06-30` model cache when explicit env is absent. `make local-asr-streaming-smoke` generated `reports/a21-local-asr-streaming-smoke-20260603-085636-1780448196060587000.json` with `status=passed`, `evidence_mode=real_model_no_audio_streaming_smoke`, `model_files_present=true`, `frames_appended=1`, `ready_events=1`, `final_events=1`, `transcript_policy=transcript_not_recorded`, and `audio_payload_policy=raw_audio_not_recorded`. Static readiness report `reports/a21-xiaozhi-streaming-provider-readiness-20260603-085641-1780448201880789000.json` marks ASR ready from the same canonical cache while still blocking mock LLM/TTS. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host loopback, WAV/file acceptance path, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-XIAOZHI-STREAMING-TTS-ADAPTER-001: Doubao realtime TTS adapter seam | Completed adapter seam, runtime provider not executed | Plan `docs/plans/2026-06-03-xiaozhi-streaming-tts-adapter.md` scoped the work. Added an explicit `providers.StreamingTTSAdapter` marker, a Doubao realtime TTS pipeline adapter selected by `A21_TTS_FAST_PROFILE=doubao_tts_realtime`, and a PCM16 mono chunker that turns provider audio deltas into exact 60 ms downlink-ready chunks without writing or reading a WAV. Local/Iflytek/voice-clone TTS remain classified as WAV/file boundaries. New readiness reports prove the distinction: `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061849-1780438729911945000.json` blocks missing Doubao TTS config, `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061850-1780438730222971000.json` marks TTS ready but still blocks ASR helper proof, and `reports/a21-xiaozhi-streaming-provider-readiness-20260603-061850-1780438730359984000.json` passes the static provider-shape gate when ASR helper env, StepFun, and Doubao realtime TTS env are all present. `prd_accepted` remains false because no real provider execution or physical `/v1/xiaozhi` trace was captured. Focused provider/app tests, `git diff --check`, and `make verify` passed. |
| T-STREAMING-TTS-RUNTIME-PROOF-001: Streaming TTS runtime smoke | Completed truthful blocker | Added redacted `a21 streaming-tts-runtime-smoke` plus `make streaming-tts-runtime-smoke`. Without `--execute`, local report `reports/a21-streaming-tts-runtime-smoke-20260603-074721-1780444041851710000.json` is `status=blocked`, finding `execute_flag_required`. Tests use a fake realtime dialer/session to prove `tts_session.update`, `input_text.append`, and `input_text.done` are sent, the first provider audio delta is observed while the stream is open, and at least one exact 60 ms PCM16 mono chunk is counted without WAV/file boundary. No real provider execution or physical Xiaozhi/StackChan path was used. |
| T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001: ASR partial to LLM realtime bridge | Completed host-side candidate | Added a partial transcript source for `VoicePipelineRequest`, a stock `/v1/xiaozhi` partial bridge that starts exactly one workmate streaming answer from the first ASR partial before `listen.stop`/`asr.final`, and parity gates that require ordered `asr.stream.commit` while blocking host-loopback fake markers. Focused provider/Gateway/app tests passed. This does not execute real providers/V21, start Gateway, flash firmware, play audio, or claim physical PRD acceptance. |
| T-XIAOZHI-REALTIME-PARITY-REAL-PROFILE-EVIDENCE-001: Realtime parity real profile evidence | Completed evidence hardening | Plan `docs/plans/2026-06-03-xiaozhi-realtime-parity-real-profile-evidence.md` scoped the no-execute/no-hardware cut. Gateway now records redacted profile-class markers for Xiaozhi voice-pipeline turns, distinguishing real streaming ASR/LLM/TTS from mock, batch, and file-boundary stages without storing raw provider names, transcripts, provider outputs, URLs, credentials, paths, or audio payloads. `xiaozhi-realtime-parity` now requires all three real streaming profile markers and no profile blockers before returning `xiaozhi_realtime_candidate`; ordered traces without those markers downgrade to `turn_buffered_xiaozhi_candidate` with `xiaozhi_realtime_real_profile_evidence_missing`. Focused app/Gateway tests passed. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host loopback runtime, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-XIAOZHI-NONBLOCKING-ASR-COMMIT-001: Xiaozhi nonblocking ASR commit | Completed host-local control-loop hardening | Plan `docs/plans/2026-06-03-xiaozhi-nonblocking-asr-commit.md` scoped the cut. `listen.stop` and VAD auto-stop now start async streaming-ASR commit/final handling instead of blocking the `/v1/xiaozhi` WebSocket read loop. Focused Gateway tests prove abort can be processed while commit remains pending and that a streaming ASR final starts the voice pipeline without calling batch `Transcribe`. This keeps A21 closer to Xiaozhi's responsive control/media state machine while remaining below full PRD acceptance. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host-loopback runtime, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-XIAOZHI-OFFICIAL-PROTOCOL-SOURCE-READ-AND-NEXT-CUT-001: Stock STT and wake preroll | Completed host-local stock fidelity cut | Plan `docs/plans/2026-06-03-xiaozhi-official-protocol-source-read-and-next-cut.md` scoped source-read workers plus a narrow red/green implementation. A21 now sends stock `stt` before `tts/start` when streaming ASR text exists, and buffers up to five true-idle wake pre-roll Opus frames for the next `listen/start` while preserving cooldown/current-turn `ignored_not_listening` behavior. Focused Gateway tests, full Gateway package tests, and focused app realtime parity/readiness tests passed. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host-loopback runtime acceptance, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-XIAOZHI-OPUS-INGRESS-QUEUE-001: Xiaozhi Opus ingress queue | Completed host-local control-loop hardening | Plan `docs/plans/2026-06-03-xiaozhi-opus-ingress-queue.md` scoped the cut. Listening Opus frames now enter a bounded per-session queue before decode/VAD/streaming-ASR append, and `listen.stop` finalization waits asynchronously for queued ingress to catch up. Focused Gateway tests, full Gateway package tests, focused app parity/readiness tests, `git diff --check`, and `make verify` passed. No provider/V21 execution, Gateway start/stop, `/v1/xiaozhi/say`, host-loopback runtime acceptance, firmware build/flash, NVS/serial/hardware action, or audio playback was performed. |
| T-DIALOGUE-001-LOW-LATENCY-CHAIN-CONVERGENCE: Dialogue-first product mode and readiness gate | Superseded by internal test 4 mode contract | Plan `docs/plans/2026-06-03-dialogue-first-low-latency-prd-convergence.md` scoped the internal test 3 cut. The low-latency Xiaozhi/ASR/text/TTS chain remains valid, but user-facing product mode surfaces now converge on `roleplay` plus `professional` through `T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001`. `dialogue` remains a backwards-compatible alias under `roleplay`; `professional` remains V21-only. |
| T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001: Roleplay/professional v2 and cloud workspace | Active roleplay soul/profile + voice-clone + workspace/read-record/source/index-request/device-binding/professional-query cut | Plan `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md` scopes the work. A21 internal test 4 makes `roleplay` the default user-facing embodied mode with role/personality, memory hints, and voice-clone selection, keeps `professional` as the explicit evidence/V21 path, and defines the cloud workspace product form: upload documents, query public-only or personal+public resources, and consult through web/app or A21 hardware without putting provider/V21 secrets or documents on StackChan. Current Gateway runtime exposes `/v1/voice-modes`, `/v1/roleplay-profile`, `/v1/professional-workspace`, `/v1/professional-query`, `/v1/professional-read-records`, `/v1/workspace-device-bindings`, `/v1/workspace-upload-jobs`, `/v1/workspace-documents`, `/v1/workspace-index-jobs`, and `/v1/workspace-sources`; accepts `dialogue` as a compatibility alias; lets `/v1/roleplay-profile` select A21 role-soul profiles, scenarios, voice-clone profile, and bounded runtime `memory_hints` while responses/traces show only readiness/count/finding/prompt-part metadata; passes composed role-soul/personality/scenario/memory prompt into fast-companion and stock `/v1/xiaozhi` voice-pipeline text-stream requests while reports only expose `prompt_input_ready`/`prompt_input_not_recorded`; carries the selected safe `voice_clone_profile` into `VoicePipelineRequest`, `TTSAdapterRequest`, and redacted report input metadata with `voice_clone_sample_not_recorded`; sends redacted `device_id`/`user_id`/`workspace_id`/`query_scope` fields through the professional V21 adapter v2 contract; records memory-only professional read metadata before/after V21 queries without storing raw utterance, retrieved text, evidence bodies, provider output, document text, URLs, paths, credentials, voice transcript, or audio; supports local document upload intake plus no-execute workspace job create/poll/fail/retry/delete/searchable-metadata metadata and a no-execute index-request ledger that promotes stored uploads to `indexing_requested_no_execute` without making them searchable; enforces workspace device bindings before V21 execution once configured; exposes the formal Web/App professional consult surface through `/v1/professional-query`; exposes upload/read/source/index/device-binding audit metadata in the simulator and `/workspace` surfaces; and routes explicit user-spoken professional triggers from default/roleplay/workmate/companion mock and stock Xiaozhi turns into the professional evidence path without triggering on negated phrases or privacy/state modes. |
| T-INTERNAL-TEST4-ROLEPLAY-SOUL-PROFILE-001: Roleplay soul profile selector | Completed host-local runtime cut | Plan `docs/plans/2026-06-04-roleplay-soul-profile-contract.md` scopes the cut. Personality composition now supports `docs/personality/role_souls/*.md`; `/v1/roleplay-profile` can select `a21_roleplay_default`, `a21_roleplay_wry_peer`, or `a21_roleplay_calm_anchor` together with scenario, memory hints, and voice-clone profile. Runtime summaries expose only safe role-soul/prompt-part metadata such as `role_soul:a21_roleplay_wry_peer`, `soul_prompt_input_ready`, memory counts, and redaction flags. Fast-companion and stock Xiaozhi roleplay prompt composition use the selected role soul before passing prompt input to the provider-neutral voice pipeline, and the simulator exposes a role soul selector/readout. No prompt body, memory text, transcript, provider output, voice clone sample, local path, URL, credential, V21 evidence, audio, provider/V21 execution, Gateway service start, firmware, serial, NVS, ECS, or physical hardware action occurred. |
| T-ROLEPLAY-MEMORY-CONTROL-SURFACE-001: Roleplay runtime memory control | Completed host-local contract cut | Plan `docs/plans/2026-06-04-roleplay-memory-control-surface.md` scopes the cut. Gateway `POST`/`PUT /v1/roleplay-profile` now accepts `memory_hints` and `clear_memory`, sanitizes runtime hints through the personality memory policy, stores only sanitized prompt-input hints in Gateway memory, reports only readiness/counts/finding codes, and keeps fast-companion roleplay `professional_route_allowed=false` and `v21_executed=false`. Simulator exposes set/clear memory controls while displaying only status/count. |
| T-ROLEPLAY-PROMPT-VOICE-PIPELINE-001: Roleplay prompt into voice pipeline | Completed host-local runtime cut | Plan `docs/plans/2026-06-04-roleplay-prompt-voice-pipeline.md` scopes the cut. `VoicePipelineRequest` now carries runtime-only `TextPrompt`; text-stream adapters receive the composed roleplay personality/scenario/memory prompt when present, while reports record only `prompt_input_ready` and `prompt_input_not_recorded`. Fast-companion turns with PCM frames and stock `/v1/xiaozhi` turns both inject the selected roleplay prompt before provider text streaming, and traces record only `roleplay.prompt_input.used`. No V21/professional route, provider execution, Gateway service start, firmware, serial, NVS, or hardware action occurred. |
| T-ROLEPLAY-VOICE-CLONE-PIPELINE-CONTRACT-001: Roleplay voice clone into TTS boundary | Completed host-local runtime cut | Plan `docs/plans/2026-06-04-roleplay-voice-clone-pipeline-contract.md` scopes the cut. `VoicePipelineRequest` now carries runtime-only `VoiceCloneProfile`; `TTSAdapterRequest` receives the selected safe A21 profile ID; local TTS adapters pass it through `LocalTTSOptions.Voice`; `VoicePipelineReport.input.voice_clone_profile` exposes only that safe ID and redaction includes `voice_clone_sample_not_recorded`. Fast-companion roleplay turns with PCM frames and stock `/v1/xiaozhi` roleplay turns both pass the selected clone profile to the voice pipeline and trace only `roleplay.voice_clone_profile.used`. No provider/V21 execution, Gateway service start, firmware, serial, NVS, or physical hardware action occurred. |
| T-INTERNAL-TEST4-ROLEPLAY-DEVICE-STATE-REFLECTION-001: Roleplay device state reflection | Completed host-local registry/simulator cut | Plan `docs/plans/2026-06-04-roleplay-device-state-reflection.md` scopes the cut. `/v1/devices` now reflects safe roleplay state fields for selected roleplay profile, scenario, soul readiness, memory readiness/count, and `roleplay_physical_accepted=false`; device `runtime_echo` carries only safe roleplay IDs, booleans, counts, and non-storage flags. The simulator Device Registry panel shows role soul, scenario, and role memory next to voice clone. Tests prove selected role soul/scenario/memory reflect into the registry without leaking memory text, unsafe URLs, control text, prompt bodies, provider output, voice-clone samples, audio, or physical acceptance. No provider/V21 execution, Gateway service start, ECS, firmware, serial, NVS, prune/gc, or physical hardware action occurred. |
| T-INTERNAL-TEST4-ROLEPLAY-OFFICIAL-EXPRESSION-PLAN-001: Roleplay official expression plan | Completed host-local no-send expression contract | Plan `docs/plans/2026-06-04-roleplay-official-expression-plan.md` scopes the cut. `/v1/roleplay-profile` now returns `expression_plan` with schema `a21.roleplay_expression_plan.v1`, using existing official StackChan action-plan metadata for baseline posture, role soul, scenario emphasis, and memory cue phases. The simulator shows the no-send policy plus action/packet counts. Tests prove selected `a21_roleplay_wry_peer` + `engineer_pushback` + memory-ready state produces expected official action metadata, does not leak memory text, unsafe URLs, prompt bodies, provider output, audio, or voice-clone samples, and keeps aggregate/per-action `physical_accepted=false`. No provider/V21 execution, Gateway service start, ECS, firmware, serial, NVS, prune/gc, `/stackChan/ws` delivery, or physical hardware action occurred. |
| T-ROLEPLAY-IMMERSION-READINESS-001: Roleplay immersion product readiness | Completed product-readiness visibility cut | Plan `docs/plans/2026-06-04-roleplay-immersion-product-readiness.md` scopes the cut. `a21 product-readiness` now fetches `GET /v1/roleplay-profile` when available and exposes a top-level `roleplay` object with selected role soul, scenario, voice-clone profile, soul prompt readiness, prompt-composed status, memory configured/readiness/count, official expression-plan action/packet counts, redaction flags, and physical acceptance truth. Invalid or unsafe roleplay profile responses become `roleplay_profile_invalid`; unavailable endpoints remain `status=unavailable`. This makes roleplay immersion visible in launch reports while still not executing providers, V21, voice-clone CLI, Gateway deployment, ECS, firmware, serial, NVS, or physical StackChan roleplay acceptance. |
| T-ROLEPLAY-VOICE-RUNTIME-EVIDENCE-001: Roleplay voice runtime evidence ingress | Completed product-readiness evidence-ingress cut | Plan `docs/plans/2026-06-04-roleplay-voice-runtime-evidence-ingress.md` scopes the cut. `a21 product-readiness` now accepts `--roleplay-voice-report` and latest `a21-roleplay-voice-probe-*.json` reports; `server-side-readiness-bundle` passes the same report through. The top-level `roleplay` object now exposes safe runtime evidence availability/match/readiness, source basename, route/status/execution mode, marker count, and booleans for text-stream execution, prompt input use, voice-clone profile use, audio downlink, and playback-start observation. Unsafe reports become `roleplay_voice_report_invalid`; incomplete matched reports become `roleplay_voice_runtime_not_ready`. This closes the roleplay voice-path reporting gap while still not executing providers/V21, accepting voice-clone audio quality, deploying Gateway/ECS, touching firmware/serial/NVS, or claiming physical StackChan roleplay acceptance. |
| T-ROLEPLAY-VOICE-PROBE-REPORT-GENERATOR-001: Roleplay voice probe report generator | Completed host/Gateway report-generation cut | Plan `docs/plans/2026-06-04-roleplay-voice-probe-report-generator.md` scopes the cut. `a21 roleplay-voice-probe` now sends a short redacted local-audio probe to the existing Gateway `/v1/fast-companion/turn` roleplay voice path, fetches `/v1/traces`, filters trace markers through the existing roleplay voice evidence allow-list, and writes `a21-roleplay-voice-probe-*.json`. Complete Gateway voice-pipeline evidence is written as `status=passed`; incomplete evidence is preserved as `status=blocked`, with `--require-ready` returning non-zero after writing the report. Tests prove generated ready reports are accepted by `product-readiness --use-latest-reports` and blocked reports do not overclaim. No ECS, V21, firmware, serial, NVS, report deletion, prune/gc, or physical hardware action occurred. |
| T-SERVER-SIDE-ROLEPLAY-VOICE-RUNTIME-GATE-001: Server-side roleplay voice runtime gate | Completed server-side launch-gate cut | Plan `docs/plans/2026-06-04-server-side-roleplay-voice-runtime-gate.md` scopes the cut. `product-readiness.server_side` now requires matched ready roleplay voice runtime evidence through `roleplay_voice_runtime_ready` before `candidate_ready=true`, and carries `roleplay_voice_source_report`. `server-side-readiness-bundle` exposes a `roleplay_voice` evidence block, lists `roleplay_voice_runtime` in missing evidence/collection commands, and `--collect-missing` runs `a21 roleplay-voice-probe --require-ready` before absorbing the generated report. Focused tests prove old provider/V21/host-only candidate paths no longer bypass roleplay voice runtime and the bundle can collect it. No ECS, firmware, serial, NVS, report deletion, prune/gc, or physical hardware action occurred. |
| T-INTERNAL-TEST4-PROFESSIONAL-MODE-RITUAL-001: Professional mode ritual contract | Completed host-local mode contract cut | Plan `docs/plans/2026-06-04-professional-mode-ritual-contract.md` scopes the cut. `/v1/voice-modes` now returns selected/per-mode ritual metadata for roleplay and professional. Professional selection exposes `PRO`, the shared checking cue, professional expression, `professional.checking_feedback.sent`, `professional_only`, `v21_allowed=true`, and `physical_accepted=false`; simulator displays the selected cue through `modeRitualReadout`. Fast-companion still rejects selected professional mode without provider or V21 execution. |
| T-INTERNAL-TEST4-PROFESSIONAL-WORKSPACE-READ-RECORDS-001: Professional workspace read records | Completed host-local read-ledger contract cut | Plan `docs/plans/2026-06-04-professional-workspace-read-records.md` scopes the cut. Gateway now exposes `GET /v1/professional-read-records` with schema `a21.gateway.professional_read_records.v1`; mock professional turns and stock `/v1/xiaozhi` professional turns start a memory-only read record before V21 query and mark it `completed` with safe `source_scope_counts`/`workspace_status` or `failed` with a safe failure code. Records include trace/session/device IDs, redacted user/workspace labels, scope, privacy, latency profile, answer style, utterance bucket, timestamps, and redaction flags only. No raw utterance, retrieved text, evidence body, screen-card text, speech block, provider output, document text, URL, local path, credential, voice transcript, audio, provider execution, real V21 execution, Gateway service start, firmware, serial, NVS, or physical hardware action occurred. |
| T-INTERNAL-TEST4-SIMULATOR-WORKSPACE-AUDIT-001: Simulator workspace audit surface | Completed host-local simulator audit cut | Plan `docs/plans/2026-06-04-simulator-workspace-audit-surface.md` scopes the cut. The simulator now has a Workspace Audit section and a `Read Records` control. It shows the latest no-execute workspace upload job status and safe professional read-record metadata: record count, completion/failure status, query scope, utterance bucket, source-scope counts, workspace status, and privacy scope. Professional evidence responses trigger a read-record refresh for the current trace. The surface does not display raw query text, retrieved text, evidence body, screen-card text, provider output, document text, URLs, paths, credentials, voice transcript, or audio. No provider/V21 execution, Gateway service start, real ingest/indexing, firmware, serial, NVS, or physical hardware action occurred. |
| T-INTERNAL-TEST4-PROFESSIONAL-VOICE-TRIGGER-001: Professional voice trigger route | Completed host-local Gateway route cut | Plan `docs/plans/2026-06-04-professional-voice-trigger-route.md` scopes the cut. Gateway now recognizes explicit user-spoken professional phrases such as "专业模式", "认真查一下", "帮我查 V21", and "给我证据" from default/roleplay/workmate/companion contexts. `/v1/mock-turn` routes trigger text into `professionalTurnResponse`; stock `/v1/xiaozhi` routes a default `realtime` turn into professional when the streaming ASR final contains a trigger and reuses that final text instead of running duplicate batch ASR. Negated phrases such as "不要进专业检索", "不用专业模式", and "别查 V21" stay out of V21. Traces store only `professional.voice_trigger.detected` and `xiaozhi.professional_route.voice_trigger`, not utterance text, evidence bodies, provider output, document text, URLs, paths, credentials, voice transcript, or audio. No real provider/V21 execution, Gateway service start, firmware build/flash, serial, NVS, or physical hardware action occurred. |
| T-INTERNAL-TEST4-WORKSPACE-SOURCE-READINESS-001: Workspace source readiness registry | Completed host-local metadata registry cut | Plan `docs/plans/2026-06-04-workspace-source-readiness-registry.md` scopes the cut. Gateway now exposes `GET /v1/workspace-sources` with schema `a21.gateway.workspace_sources.v1`; workspace job creation creates a linked redacted `source_id`; `mark_searchable` / `mark_indexed_metadata_only` promotes only metadata readiness to `searchable_metadata_only`; fail/retry/delete keep source readiness synchronized and delete leaves a redacted tombstone. `/v1/professional-workspace` runtime now reports `source_scope_counts`, `searchable_source_scope_counts`, and `query_scope_readiness` while keeping `v21_execution_allowed=false`. The simulator Workspace Audit surface can refresh source count/readiness. No document text, bytes, base64 payload, import URL, local path, credential, provider output, real upload storage, real indexing, provider/V21 execution, Gateway service start, firmware, serial, NVS, ECS, or physical hardware action occurred. |
| T-INTERNAL-TEST4-WORKSPACE-DOCUMENT-UPLOAD-INTAKE-001: Workspace document upload intake | Completed host-local local-storage intake cut | Plan `docs/plans/2026-06-04-workspace-document-upload-intake.md` scopes the cut. Gateway now exposes `POST /v1/workspace-documents` with schema `a21.gateway.workspace_documents.v1`; it accepts multipart file bytes only, stores them under the A21 runtime store, computes a safe `sha256:` document hash, and links the stored document to workspace job/source records. Linked jobs and sources report `stored_local_pending_index`, `storage_status=stored_local`, and `index_status=not_started_no_execute`; `/v1/professional-workspace` reports that pending-index readiness for the selected scope while keeping `v21_execution_allowed=false`. Simulator Workspace Audit has a file picker/upload control and safe metadata readout. Responses/traces do not expose raw document text, raw bytes, base64 payloads, original private filenames, local paths, import URLs, credentials, provider output, V21 evidence, voice transcript, or audio. No parsing, chunking, embedding, indexing, cloud storage, provider/V21 execution, Gateway service start, firmware, serial, NVS, ECS, or physical hardware action occurred. |
| T-INTERNAL-TEST4-WORKSPACE-INDEX-REQUEST-LEDGER-001: Workspace index request ledger | Completed host-local no-execute index-request cut | Plan `docs/plans/2026-06-04-workspace-index-request-ledger.md` scopes the cut. Gateway now exposes `GET/POST /v1/workspace-index-jobs` with schema `a21.gateway.workspace_index_jobs.v1`; it accepts only safe document/job/source IDs and optional trace/session/device IDs, verifies the stored-local file exists, and records a redacted `index_job_id`. Linked document/job/source readiness promotes to `indexing_requested_no_execute`; source summaries now include `indexing_requested_source_scope_counts`; `/v1/professional-workspace` reports selected-scope indexing-request readiness and `indexing_api_ready=true` only after a request is recorded, while still keeping `v21_execution_allowed=false`. Simulator Workspace Audit has an index request control/readout. Responses/traces do not expose document text, raw bytes, base64 payloads, original private filenames, local paths, import URLs, credentials, provider output, V21 evidence, voice transcript, or audio. No parsing, chunking, embedding, OCR, cloud storage, provider/V21 execution, Gateway service start, firmware, serial, NVS, ECS, or physical hardware action occurred. |
| T-INTERNAL-TEST4-WORKSPACE-CONSOLE-PRODUCT-SURFACE-001: Workspace console product surface | Completed host-local web console cut | Plan `docs/plans/2026-06-04-workspace-console-product-surface.md` scopes the cut. Gateway now serves `GET /workspace` with a product-oriented workspace console over existing safe APIs. The console lets users select query scope, upload a local document, request no-execute indexing, refresh source readiness, refresh professional read records, and see roleplay/professional boundary state. Playwright verified desktop and mobile render with no visible overflow and exercised a dummy upload plus no-execute index request, producing `stored_local`, `indexing_requested_no_execute`, source count `1`, and `searchable=false`. No new service/port/dependency, provider/V21 execution, real indexing, Gateway deployment, ECS, firmware, serial, NVS, prune/gc, or physical hardware action occurred. |
| T-WORKSPACE-CONSOLE-MANAGEMENT-CONTROLS-001: Workspace console management controls | Completed host-local web console management cut | Plan `docs/plans/2026-06-04-workspace-console-management-controls.md` scopes the cut. `/workspace` now exposes delete source, safe metadata export, and professional read-record filter controls on top of existing Gateway APIs only. Delete uses `PUT /v1/workspace-upload-jobs` with `action=delete` and leaves an honest `deleted_metadata_only` tombstone; export writes client-side `a21.workspace_console_export.v1` metadata with explicit redaction flags; read filters call `GET /v1/professional-read-records` with safe `record_id`, `trace_id`, or `session_id`. Playwright verified upload, no-execute index, export, filter, delete, desktop render, and mobile render with no horizontal overflow. No new API/service/port/dependency, provider/V21 execution, real indexing, Gateway deployment, ECS, firmware, serial, NVS, prune/gc, flash, or physical hardware action occurred. |
| T-WORKSPACE-ROLEPLAY-CONTROL-SURFACE-001: Workspace roleplay control surface | Completed host-local web console roleplay cut | Plan `docs/plans/2026-06-04-workspace-roleplay-control-surface.md` scopes the cut. `/workspace` now exposes role soul, scenario, voice profile, bounded memory hint, save, and clear-memory controls over existing `/v1/roleplay-profile` and `/v1/voice-chain-profiles` APIs. Playwright verified selecting `a21_roleplay_wry_peer`, `engineer_pushback`, and `a21_voice_clone_default`, saving one bounded memory hint, seeing `/v1/roleplay-profile` return `memory_count=1`, `prompt_composed=true`, expression actions, and `physical_accepted=false`, then clearing memory and exporting safe metadata. No raw memory text, prompt body, voice data, provider output, V21 evidence, new API/service/port/dependency, provider/V21 execution, real indexing, Gateway deployment, ECS, firmware, serial, NVS, prune/gc, flash, or physical hardware action occurred. |
| T-WORKSPACE-VOICE-CHAIN-WAKE-CONTROL-SURFACE-001: Workspace voice-chain and wake control surface | Completed host-local web console voice/wake cut | Plan `docs/plans/2026-06-04-workspace-voice-chain-wake-control-surface.md` scopes the cut. `/workspace` now exposes voice-chain mode, ASR profile, LLM profile, realtime provider, effective TTS readout, and wake-word mode/phrase/pinyin/threshold controls over existing `/v1/voice-chain-profiles` and `/v1/wake-word` APIs. Playwright verified saving `cascade`, `doubao_asr_realtime`, `stepfun`, and `openai_realtime` without provider execution, then saving custom MultiNet wake intent and confirming `pending_firmware_build`, built-in Xiaozhi WakeNet still active, runtime hot swap false, custom runtime false, and reset back to builtin. No provider/V21 execution, real indexing, Gateway deployment, ECS, firmware build, serial, NVS, prune/gc, flash, or physical hardware action occurred. |
| T-WORKSPACE-VOICE-PROBE-CONTROL-SURFACE-001: Workspace voice probe control surface | Completed host-local web console voice-probe cut | Plan `docs/plans/2026-06-04-workspace-voice-probe-control-surface.md` scopes the cut. `/workspace` now exposes a safe Voice Probe panel over existing `/v1/fast-companion/turn`, `/v1/mock-turn`, `/v1/traces`, and `/v1/professional-read-records`. Playwright selected `a21_roleplay_wry_peer`, `engineer_pushback`, and `a21_voice_clone_default`, saved one bounded memory hint, and observed the roleplay probe return `fast_companion_hybrid`, trace event count `14`, `prompt=true`, selected voice clone, and `memory=ready / 1`. Professional probe returned `professional_mock_turn`, trace event count `13`, and one `a21-professional-read-*` record with `status=completed`, `query_scope=public_only`, and `workspace_status=searchable` in the host-local default mock path. No provider execution, real V21 execution, real indexing, Gateway deployment, ECS, firmware build, serial, NVS, prune/gc, flash, or physical hardware action occurred. |
| T-INTERNAL-TEST4-PROFESSIONAL-QUERY-ENDPOINT-001: Web/App professional consult endpoint | Completed host-local Gateway product-surface cut | Plan `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md` scopes the cut. Gateway now exposes `POST /v1/professional-query` with schema `a21.gateway.professional_query.v1`; it accepts safe device/workspace/scope/query fields, emits professional checking cue events, starts the professional read ledger, enforces workspace device bindings before `v21.query.start`, calls the A21/V21 adapter only after that guard, and returns user-facing answer/evidence events plus a redacted evidence report when V21 succeeds. Unsafe raw document/evidence/provider/base64/path/URL/credential/audio fields are rejected before query execution, and Gateway metadata/read-records/traces do not echo the user's query text. `/workspace` professional probe now calls `/v1/professional-query` instead of `/v1/mock-turn`. Focused Gateway tests and full Gateway package tests passed. No real indexing, durable cloud auth/tenant ACL, V21 merge/release, provider deployment, Gateway ECS change, firmware, flash, serial, NVS, prune/gc, or physical hardware action occurred. |
| T-V21-A21-V2-WORKSPACE-QUERY-SCOPE-NATIVE-001: V21 native A21 v2 workspace query-scope contract | Completed V21 worker branch, ACL enforcement pending | A21 plan `docs/plans/2026-06-04-v21-a21-v2-workspace-query-scope-native-contract.md` scoped a V21-repo worker. V21 branch `origin/codex/a21-v2-workspace-query-scope-native-contract` commit `ad61246` makes `/internal/v1/knowledge/voice-query` natively accept `device_id`, `user_id`, `workspace_id`, and `query_scope`, validates `public_only`, `personal_only`, and `personal_plus_public`, and returns safe `source_scope_counts` plus `workspace_status`. Current V21 schema still cannot prove personal/public ACL enforcement, so the worker truthfully returns `scope_contract_ready_acl_pending` and zero classified counts. No A21 code, Gateway runtime, StackChan hardware, firmware, provider execution, or dirty V21 LAN/desktop work was touched. |
| T-V21-A21-WORKSPACE-SCOPE-RETRIEVAL-GUARD-001: V21 source-scope retrieval guard | Completed V21 worker branch, merge/release pending | V21 plan `/Users/jiyurun/.codex/worktrees/b0c0/v21-knowledge-platform/docs/plans/2026-06-04-a21-v2-workspace-scope-retrieval-guard.md` scoped the worker. V21 branch `origin/codex/a21-v2-workspace-scope-retrieval-guard` commit `fccd0ac` carries A21 v2 `device_id`/`user_id`/`workspace_id`/`query_scope` into retrieval requests, carries safe `source_scope=public|personal` through HTTP/Qdrant/Postgres retrieval results and voice-query evidence, filters `public_only`, `personal_only`, and `personal_plus_public` before answer generation, fails closed for unclassified evidence in scoped A21 responses, and can return classified `source_scope_counts` with `workspace_status=searchable`. V21 `go test ./...`, sidecar unittest, retrieval eval unittest, `git diff --check`, and `make compose-config` passed. V21 `make test`/`make check` were blocked by Node toolchain mismatch (`v25.8.0` present, `v24.15.0` expected). This is source-scope guard evidence only; durable tenant/account ACL, real personal upload indexing, cloud storage, V21 merge/release, and physical StackChan professional consult acceptance remain open. No A21 code, Gateway runtime, services, provider execution, private document ingest, ECS, firmware, serial, NVS, or dirty V21 LAN/desktop work was touched. |
| T-A21-V21-NATIVE-VOICE-QUERY-BRIDGE-001: A21 native V21 voice-query bridge | Completed host-local adapter contract cut | Plan `docs/plans/2026-06-04-a21-v21-native-voice-query-bridge.md` scopes the cut. A21 local `v21-adapter-bridge` now calls V21 native `/internal/v1/knowledge/voice-query` as the primary professional query path, passes safe A21 v2 `device_id`/`user_id`/`workspace_id`/`query_scope` fields, mirrors V21-returned `source_scope_counts` and `workspace_status`, and uses direct retrieval only as a controlled no-evidence expansion fallback with counts derived only from result `source_scope` labels. Focused app/adapter tests passed. This is adapter-boundary contract evidence only; V21 merge/release, real personal upload indexing, durable tenant/account ACL, cloud storage, provider execution, Gateway service startup, ECS deployment, firmware, serial, NVS, and physical StackChan professional consult acceptance remain open. |
| T-ALIYUN-001-XIAOZHI-PUBLIC-VOICE-GATEWAY: Main public Gateway profile | Active main-edge running | Plan `docs/plans/2026-06-03-aliyun-xiaozhi-public-voice-gateway.md` scopes the cut. Gateway now separates `gateway_profile` from `voice_mode`: valid `A21_PUBLIC_GATEWAY_URL` selects `public_wss` as the main product public path, while `mac_local` remains available for Mac/local-model switching. Product deployment targets trusted `443`/`wss`; IP-only bring-up can use public `http/ws`. `/v1/gateway-profiles`, simulator selection, env `A21_PUBLIC_GATEWAY_URL`, CLI `--public-gateway-url`, and OTA public URL behavior are implemented. The earlier `101.132.117.182` SWAS path is experimental/backup. New ECS `47.103.57.217` is active behind Caddy, returns `ws://47.103.57.217/v1/xiaozhi` from OTA, and passed remote `make verify`; host-only bench remains blocked below PRD because no real provider/V21/hardware execution was injected. |
| T-VOICE-CHAIN-SELECTOR-001-CASCADE-REALTIME-HOTSWITCH: Product voice-chain selector | Completed host-local selector cut | Plan `docs/plans/2026-06-03-voice-chain-product-selector-hot-switch.md` scoped the cut. Gateway now exposes `GET/POST/PUT /v1/voice-chain-profiles` for the frontend to choose `cascade` or `realtime` independently from `voice_mode`, `gateway_profile`, and catalog-only `cloud_voice_profile`. Cascade exposes ASR and LLM choices, recommends StepFun, keeps DeepSeek as fallback, and fixes default TTS at DashScope realtime TTS unless voice clone maps the effective TTS to `voice_clone_cli`. Realtime selection updates the existing `A21_GATEWAY_VOICE_PROVIDER=selected` gate plus `A21_PROVIDER_PRIMARY`. Simulator and device registry now display chain mode, ASR, LLM, effective TTS, realtime provider, and voice/clone profile. Focused Gateway tests passed; no provider execution, deployment, firmware, hardware, or audio playback occurred. |
| T-XIAOZHI-FAST-ACK-CONTINUITY-001: Fast ack cannot block main answer | Completed host-local runtime fix | Public bench against `47.103.57.217` showed the chain reached Opus ingress/decode, streaming ASR append/commit, ASR partial/final, and stock `stt`, but then stopped at `xiaozhi.fast_ack.unavailable` without running the full answer pipeline. Gateway now treats fast ack as optional: if fast-ack TTS fails and the turn is not aborted, the full ASR -> LLM -> TTS answer pipeline continues. Focused Gateway tests, related package tests, `git diff --check`, and `make verify` passed. This is not yet deployed/bench-verified in the public Gateway at the time of this state entry, and it does not claim physical PRD acceptance. |
| T-XIAOZHI-PROVIDER-STATE-MACHINE-REBUILD-001: DashScope realtime TTS lifecycle | Completed public cloud-edge host candidate | Plan `docs/plans/2026-06-03-xiaozhi-provider-state-machine-rebuild.md` scoped the official-state-machine rebuild. DashScope realtime TTS now starts the read loop before text append/commit, waits for `session.updated` when available, emits chunks on `response.audio.delta`, sends `session.finish` after text commit per Qwen-TTS realtime semantics, and closes cleanly on cancellation. Focused DashScope/provider tests, related Gateway/App/Provider tests, `git diff --check`, and `make verify` passed before deployment. Public Gateway `47.103.57.217` was updated through commit `8752b8d`; root-only ECS env was corrected from CosyVoice model/voice family to Qwen-TTS Realtime family after traces showed `tts_session_update_failed`. Public `xiaozhi-voice-bench --require-product-chain` report `reports/a21-xiaozhi-voice-bench-20260603-201052.155507000.json` passed as `candidate_host_only` with `provider_executed=true`, `voice_pipeline_execution_mode=cloud_edge`, `127` answer binary downlink frames, answer first-audio p95 `806 ms`, and barge-in stop p95 `12 ms`. This is still below physical PRD acceptance; next action is physical StackChan wake/mic/audible/barge-in proof on `ws://47.103.57.217/v1/xiaozhi`. |
| T-XIAOZHI-PHYSICAL-PUBLIC-GATEWAY-TRACE-001: Public real-device voice trace | Completed physical gateway-trace candidate | Physical StackChan `44:1b:f6:e2:6a:60` connected to public Gateway `47.103.57.217` on stock Xiaozhi websocket profile and produced trace `a21-trace-44-1b-f6-e2-6a-60` / session `a21-session-44-1b-f6-e2-6a-60`. Counters showed real mic/Opus ingress (`xiaozhi.opus_frame.received=66`, decoded `66`), VAD speech start/end (`5/5`), listen auto-stop `5`, ASR partial/final (`11/11`), LLM first content `5`, TTS first audio `5`, TTS Opus downlink `506`, answer downlink first-frame markers `10`, completed voice pipelines `3`, and trace-level barge-in/playback stop evidence (`barge_in.detected=5`, `playback.stop=5`). Reports `reports/a21-xiaozhi-physical-evidence-20260603-201623.594517000.json` and `reports/a21-xiaozhi-half-duplex-acceptance-20260603-201623.946467000.json` record `candidate_gateway_downlink` / `candidate_gateway_trace`, mic delivery ratio `1`, `answer.first_downlink=571 ms`, downlink available, barge-in stop available, and `prd_accepted=false`. Remaining blockers are device playback ack or trusted runtime playback start, operator/instrumented audible observation, and stock firmware playback `stop_done` exposure. |
| T-STACKCHAN-CLOUD-GATEWAY-NVS-CORRECTION-001: Product NVS cloud endpoint correction | Completed NVS write, cloud app verification blocked | After the operator clarified that the product StackChan must use the all-cloud path, the local LAN Gateway was stopped and local port `21080` was confirmed not listening. Official-compatible product NVS was rewritten with the operator-provided phone hotspot credentials plus cloud endpoints `http://47.103.57.217/xiaozhi/ota/` and `ws://47.103.57.217/v1/xiaozhi`; execute report `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-163211-1780561931273006000.json` passed with `wifi_credentials_written=true`, `mutated_entry_count=5`, and `servo_calibration_present=true`. After hard reset, observed serial output showed regular `SystemInfo` lines and did not repeat the earlier `No AP found` or hotspot-provisioning fallback. Current network probes reach TCP ports `22`, `80`, `443`, and `21081`, but public HTTP/HTTPS requests return empty replies or TLS syscall errors and SSH closes before authentication, so device registration cannot be confirmed until the cloud Gateway/Caddy application layer is restored or an ECS control path is available. |
| T-STACKCHAN-PRODUCT-FLASH-CHINANET-NVS-001: Product app flash and ChinaNet NVS refresh | Completed guarded hardware writes, physical reconnect restored | In the foreground hardware-window branch `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`, the control tower used only the product lane and flashed `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin` on `/dev/cu.usbmodem1101`; execute report `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-195734-1780574254123911000.json` passed with app SHA-256 `e66a41ef486b866b076746bd064af2e3afb75e0a316515921bbc681b89fb36a8`. First product NVS execute report `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-195752-1780574272065931000.json` passed but used the first typed SSID spelling; operator screenshot clarified the visible SSID is `ChinaNet-N6e3`. Corrected NVS execute report `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-200257-1780574577392353000.json` passed with `wifi_credentials_written=true`, `mutated_entry_count=5`, `servo_calibration_present=true`, OTA `http://47.103.57.217/xiaozhi/ota/`, and WebSocket `ws://47.103.57.217/v1/xiaozhi`. Serial startup confirmed the device found `ChinaNet-N6e3`, connected with IP `192.168.1.26`, opened the public Xiaozhi WebSocket, and entered listening/speaking. Public checks must bind source IP `192.168.1.20` while TUN is active; with that workaround `/healthz`, OTA, `/v1/devices`, and live trace access pass. This is hardware reconnect and live runtime evidence, not full PRD physical acceptance. |
| T-STACKCHAN-OFFICIAL-TOUCH-ACTION-EVIDENCE-001: Product touch/body event bridge | Completed product-lane physical touch slice | Commit `f16e71b` added a product-only touch event bridge: Gateway parses `hello.features.touch_events`, only returns `a21.touch_events=true` under `A21_XIAOZHI_PRODUCT_TOUCH_EVENTS=true` for hardware-MAC stock clients without debug features, records screen/top touch events as `device.touch.*.received`, and keeps debug `state`/`face`/`display`/`motion` blocked. The official-compatible overlay advertises `touch_events`, bridges screen touch and official top-touch HAL gestures, and sends `type=device, kind=touch` only after product allowance. ECS `47.103.57.217` was deployed with the new commit and env gate, guarded product build/flash passed with app SHA-256 `9b8366e387b10ffa784394e965f702734753f1c4c68192f17a11135e3b713216`, and physical acceptance passed for `screen_touch`, `top_tap`, `top_swipe_forward`, `top_swipe_backward`, and `top_barge_in` on device `44:1b:f6:e2:6a:60`. Directional swipes still need product affordance; this is not camera/NFC/IR/screen visual/no-cable/full PRD acceptance. |
| T-STACKCHAN-OFFICIAL-TOUCH-BODY-REACTION-001: Product touch body reaction | Completed product-gated runtime/physical slice | Gateway now has `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true` as a separate server-side gate. Product touch events from hardware-MAC stock Xiaozhi clients with `hello.features.mcp=true` trigger bounded official MCP `self.robot.set_led_color` and `self.robot.set_head_angles` reactions, while stable `last_touch_*` registry fields preserve touch evidence after MCP responses, Opus, or heartbeat events update `last_event`. ECS was deployed with the gate; focused local/remote Gateway/App tests and `make verify` passed. Runtime evidence report `reports/a21-stackchan-touch-reaction-evidence-20260604-224756.json` passed on device `44:1b:f6:e2:6a:60`: `top_swipe_backward` from `top_sensor` produced LED `120/60/0`, head `yaw=-18,pitch=24,speed=200`, touch reaction markers, and redacted MCP responses on trace `a21-trace-44-1b-f6-e2-6a-60`. No firmware flash, NVS write, provider execution, V21 execution, serial write, git prune/gc, or internal-test3 voice/protocol rollback occurred. |
| T-STACKCHAN-OFFICIAL-STATE-BODY-REACTION-001: Product state body reaction | Completed local Gateway/App implementation, ECS/physical evidence pending | Plan `docs/plans/2026-06-04-stackchan-product-state-body-reactions.md` scopes the cut. Gateway now has `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true` as a separate server-side gate. Hardware-MAC stock Xiaozhi clients with `hello.features.mcp=true` receive `a21.state_reactions=true`; Gateway-generated `idle`, `listening`, `thinking`, `speaking`, `error`, and `fatal_error` transitions trigger bounded official MCP `self.robot.set_led_color` and `self.robot.set_head_angles` reactions over the live `/v1/xiaozhi` socket. State reactions record `xiaozhi.state_reaction.robot_led_color.sent`, `xiaozhi.state_reaction.robot_head_angles_set.sent`, or `xiaozhi.state_reaction.failed`, and store only redacted runtime echo such as `last_state_reaction_state`, `last_state_reaction_reason`, `robot_head_pitch`, and `robot_led_blue`. Gateway also marks closed Xiaozhi sockets as `connection_status=xiaozhi_ws_disconnected` without erasing the last semantic event, so stale registry rows are not accepted as writable MCP sockets. Focused Gateway/App tests, `git diff --check`, and `GOMAXPROCS=2 make verify` passed. No firmware flash, NVS write, serial write, provider execution, V21 execution, or internal-test3 voice/protocol rollback occurred. |

| T-XIAOZHI-SECOND-READONLY-CROSSCHECK-001: Protocol/endpoint/runtime/strategy cross-check | Completed read-only audit | Four strict read-only workers on HEAD `188b341` returned structured final reports. Protocol thread `019e8ac3-c9f3-7cc3-b8a1-c27cc2748168` confirmed WebSocket/Opus parity is enough for the immediate product lane but MQTT+UDP must remain a planned Xiaozhi transport gap. Endpoint thread `019e8ac3-c9f7-7721-9f6c-1bce1e69af4c` identified custom wake vs official AFE/WakeNet and parked direct-Xiaozhi app lifecycle as the highest product-lane parity risks. Runtime thread `019e8ac3-c9f6-7350-a66e-e51dcdc8109e` identified the host chain blocker: ASR partials do not yet drive LLM/TTS before ASR final/listen stop. Strategy thread `019e8ac3-c9fa-7ed0-8b61-625a418a84c2` recommends incremental A21 convergence using Xiaozhi firmware/protocol/audio-service patterns, with ADR-backed B-lite voice-engine adapter only if phased physical evidence fails. No worker edited files, built, flashed, started services, called providers/V21, or touched audio/hardware. |
| T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001: Official hardware parity gap map | Completed docs/state baseline | Worker froze the official-vs-A21 hardware/control/status gap map in `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md` using the local official StackChan root `da156e1fa0e1c2a5e00b78fbf69b1f7e7bca0483` as a dirty working-tree reference and the Xiaozhi sub-tree `e77dedb1309153bb63fed285772962c920c97dd4` as a clean detached-HEAD reference. The map distinguishes `available`, `diagnostic`, `planned`, `blocked`, and `product-accepted`, assigns owner transitions, acceptance evidence, and rollback paths for every surface in the parity plan, and updates `docs/engineering/A21_CURRENT_CONTROL.md`. No Gateway start, provider/V21 execution, firmware build, flash, serial, or NVS write occurred. Next candidate is the low-risk MCP/status worker, not firmware or high-risk hardware. |
| T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001: Official MCP status/control parity | Completed low-risk Gateway contract | Worker added `POST /v1/xiaozhi/mcp-control` for only `self.get_device_status`, `self.screen.set_brightness`, `self.screen.set_theme`, and `self.screen.get_info`. Requests require A21 `device_id`, carry or generate `trace_id` and `session_id`, require an online `/v1/xiaozhi` socket with `hello.features.mcp=true`, and record only redacted send markers after delivery. High-risk tools such as reboot, firmware upgrade, camera/photo, screen snapshot, stream/video, NFC, infrared, and app lifecycle are rejected before websocket write. This is not firmware work, physical evidence, or product acceptance. |
| T-STACKCHAN-OFFICIAL-MCP-SPEAKER-VOLUME-FREEZE-001: Official MCP speaker volume freeze | Completed host-local Gateway contract | Plan `docs/plans/2026-06-04-stackchan-official-mcp-speaker-volume-freeze.md` scopes the cut. The existing dedicated `/v1/xiaozhi/speaker-volume` endpoint remains available, and unified `POST /v1/xiaozhi/mcp-control` now also allows `self.audio_speaker.set_volume` with bounded `volume=0..100`. The control path still requires A21 `device_id`, trace/session identity, online Xiaozhi socket, and `hello.features.mcp=true`; it records only redacted marker `xiaozhi.mcp.speaker_volume.sent` plus bounded `speaker_volume` activity metadata. Missing/out-of-range volume and mixed screen/volume arguments are rejected before websocket write, and high-risk MCP tools remain blocked. No raw MCP response, prompt, transcript, provider output, V21 evidence, audio, firmware, serial, NVS, ECS, Gateway service start, or physical hardware action occurred. Physical loudness/playback acceptance remains evidence-gated. |
| T-STACKCHAN-OFFICIAL-STATUS-DISPLAY-PARITY-001: Official status-display registry parity | Completed Gateway/protocol registry contract | A21 protocol now has stable `DisplayState` values for official Xiaozhi/StackChan status words and normalizes unknown or legacy-looking states to `error`. Gateway records latest status-display metadata in `/v1/devices` with source, trace/session IDs, update timestamp, and `display_state_physical_accepted=false`; stock Xiaozhi turn state writes and A21 device events update the registry without erasing capabilities or runtime echo. Trace markers `stackchan.display_state.received`, `stackchan.display_state.normalized`, and `stackchan.display_state.registry_updated` are redacted state markers only. No Gateway service start, provider/V21 execution, firmware build, flash, serial, NVS write, or physical screen acceptance occurred. |
| T-STACKCHAN-OFFICIAL-ACTION-PARITY-001: Official avatar/action semantic mapping | Completed Gateway/transport mapping contract | `BuildOfficialActionPlan` now returns official `ControlAvatar`, `ControlMotion`, or `DanceSequence` packets plus redacted metadata for packet count, semantic surfaces, and `physical_accepted=false`. Transport tests cover all semantic states, dance, yaw candidate clamping, unsupported display/heartbeat/camera/video/call frame classes, and no RGB-frame overclaim. Gateway `POST /v1/stackchan/official/control` returns action metadata and mirrors it into `/v1/devices.runtime_echo` with `official_stackchan_` prefixes. No Gateway service start, provider/V21 execution, firmware build, flash, serial, NVS write, or physical action acceptance occurred. |

## Blocked Transitions

| Transition | Blocker | Required unblock |
| --- | --- | --- |
| T-HW-002b: Full StackChan physical acceptance after Gateway downlink | Audible playback is accepted for the 3x foreground path, relay WAV playback received positive operator feedback, no-flash self-trigger observation passed, and touch/barge-in proof now has product-lane physical reports; custom wake, broader device playback timing, and final physical evidence regeneration are still missing | Collect custom wake proof and device playback timing or trusted playback-start evidence, then regenerate `xiaozhi-physical-evidence`. |
| T-PRD-001: Declare full PRD physical acceptance | Audio path is accepted, selected-provider readiness is refreshed, and no-flash self-trigger observation passed, but PRD accepted remains false because custom wake, continuous voice pipeline/host voice evidence, V21 professional execution, and final physical evidence regeneration are still pending | Close custom wake proof, collect continuous voice/host voice evidence and V21 professional evidence, regenerate physical evidence, and rerun product readiness. |
| T-HALF-DUPLEX-DIAG-001: Instrumented half-duplex counter acceptance | Latest online diagnostic-counter run `reports/a21-stackchan-half-duplex-acceptance-20260603-015622.json` is blocked because current stock firmware lacks A21 identity, diagnostic mic-probe capability, available speaker echo fields, and runtime echo counters; this is no longer a contest-path blocker because no-flash self-trigger observation passed | Create a separate guarded diagnostic-capability firmware plan only if machine-verifiable counters are required. |
| T-ALIYUN-CLOUD-GATEWAY-APP-REACHABILITY-001: Cloud Gateway application reachability | Default public probes from this Mac are false-negative while TUN routes `47.103.57.217` through `utun6`/`198.18.0.1`; direct-source probes with `192.168.1.20` return healthy A21 Gateway, OTA, device registry, and live traces. SSH with source bind reaches the real auth state but fails `Permission denied (publickey)`, so ECS deploy/restart remains blocked by key authorization, not app reachability. | Use `A21_DIRECT_SOURCE_IP=192.168.1.20` or `curl --interface 192.168.1.20` for verification under TUN; use Aliyun workbench or an accepted SSH key for ECS deploy/restart. |

## Next Candidate Transitions

Priority candidate added from the 2026-06-04 hardware parity comparison:

- `T-STACKCHAN-OFFICIAL-HARDWARE-PARITY-001`
  - Current phase: docs-only gap map worker
    `T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001` completed, and the low-risk
    Gateway MCP/status worker
    `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001` landed as a control contract.
    The official source identity, A21 source identity, parity matrix, landing
    class, owner transition, evidence, and rollback path remain frozen in
    `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`.
  - Plan:
    `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`.
  - Next action: main control should dispatch
    `T-STACKCHAN-OFFICIAL-ACTION-PHYSICAL-EVIDENCE-001` only in an approved
    foreground hardware window for touch, barge-in, visible action, RGB, and
    servo evidence.
  - Boundary for next worker: no firmware build, flash, serial, NVS write,
    reboot/upgrade exposure, camera/photo, screen snapshot, video stream,
    NFC, infrared, or app-lifecycle changes.

Priority candidate added from the 2026-06-04 internal test 4 workspace plan:

- `T-V21-A21-V2-WORKSPACE-QUERY-SCOPE-NATIVE-001`
  - Current phase: native metadata worker completed at
    `origin/codex/a21-v2-workspace-query-scope-native-contract` commit
    `ad61246`; follow-up source-scope guard worker completed at
    `origin/codex/a21-v2-workspace-scope-retrieval-guard` commit `fccd0ac`.
  - Plan:
    `docs/plans/2026-06-04-v21-a21-v2-workspace-query-scope-native-contract.md`.
  - Next action: V21 owner/main thread should review/merge both worker
    branches, then define the durable tenant/account ACL plus real personal
    upload/index transition that binds A21 workspace documents to V21 indexed
    personal corpora.
  - Boundary: A21 may cite `fccd0ac` as source-scope guard evidence only. Do
    not claim durable account ACL, real personal upload indexing, cloud
    storage, V21 release readiness, or physical professional consult acceptance
    from this branch alone.

1. `T-WAKE-003: Zi Yue Phrase Tuning`
   - Current phase: root causes found and product-lane rebuild passed. First,
     `CustomWakeWord::Initialize` used the asset `index.json` command path
     whenever a model list was already loaded, so the A21
     `CONFIG_CUSTOM_WAKE_WORD` aliases were not guaranteed to become the active
     MultiNet command table. Second, the post-flash boot log showed
     `index.json` and model-loader failures because `generated_assets.bin` was
     4,688,623 bytes while the emitted product partition table still allocated
     only `assets ... 4M`. The overlay now overrides the asset command list
     and patches `firmware/partitions.csv` to `assets ... 5M`; the build/flash
     path now rejects assets images larger than the binary partition table.
     Product build
     `reports/a21-stackchan-official-baseline-20260603-210122-1780491682047937000.json`
     passed with app SHA-256
     `7c2b0f8e72e43bf3296638faba9667c557f8f732b19f7b22da7805e6b8597af9`,
     partition-table SHA-256
     `704b0cc2d29d95d8429450e3d379c903c77864042d0bc3050f669c2c244bdb8d`,
     and assets SHA-256
     `d0a20f925364d33e75694dd07b4897ba9a1689728949d45a2987d6551cbc8b8e`.
   - Latest update: serial evidence after the assets fix proved MultiNet and
     custom wake did load and detected `紫悦`, but the public Gateway trace
     still only saw `xiaozhi.hello.received`. Xiaozhi baseline review found
     that `ContinueWakeWordInvoke` assumes the `kDeviceStateConnecting` path;
     A21's quiet idle WebSocket creates a new valid path,
     `kDeviceStateIdle && protocol_->IsAudioChannelOpened()`, which the old
     guard returned from before sending `listen.start`. The overlay now accepts
     that A21 fast path. Product build
     `reports/a21-stackchan-official-baseline-20260603-211451-1780492491468577000.json`
     passed with app SHA-256
     `7674e98af738ade2e3653b46598093a135611cf8f6a4746a689bf0b447ef66c8`.
   - Next action: commit the wake state-machine fix, guarded-flash only
     `a21-stackchan-official-xiaozhi-compatible.bin`, capture serial plus
     Gateway trace proving wake -> `listen.start` -> Opus ingress -> ASR ->
     TTS downlink, then physically retry `紫悦`, `紫悦紫悦`, `你好紫悦`, and
     `小紫悦` from idle.

2. `T-ASR-GREEN-LATENCY-001: Xiaozhi Listen Auto-Stop`
   - Current phase: product app flashed, firmware no-speech timer is present,
     and Gateway now suppresses immediate no-speech placeholder listen restarts
     for stock physical devices.
   - Next action: ask the operator to tap once and confirm no-speech green
     listening exits instead of looping, then test wake from idle with `紫悦`,
     `紫悦紫悦`, `你好紫悦`, and `小紫悦`.

3. `T-STACKCHAN-APP-PRELOAD-NO-WELCOME-001: Official Frontend Without Setup Trap`
   - Current phase: no-welcome/direct Xiaozhi path is physically useful, but
     the latest read-only audit says full official AppAvatar/AppDance/AppSetup
     lifecycle parity is not proven because the Mooncake app update path is
     bypassed.
   - Next action: keep current contest package unless welcome regresses; plan a
     separate official-app-lifecycle parity transition after wake/tap validation.

4. `T-AUDIO-BARE-XIAOZHI-PARITY-001: Migrate Bare-Package Audio Advantages`
   - Current phase: read-only parity audit confirms the device side is official
     Xiaozhi Opus/AudioService, while the remaining audio gap is mainly host
     provider/TTS PCM/WAV source quality, chunking, leveling, and re-encoding.
   - Next action: keep 3x accepted gain frozen; plan TTS streaming/mastering A/B
     before changing provider, gain, codec, or firmware.

5. `T-VOICE-CHAIN-EVIDENCE-001: Selected Voice-Chain Readiness Ingress`
   - Current phase: product readiness and server-side readiness now ingest the
     existing static no-execute
     `a21.xiaozhi_streaming_provider_readiness.v1` report through
     `--voice-chain-readiness-report` or `--use-latest-reports`. The report is
     matched against the current Gateway-selected ASR/LLM/TTS profiles before
     `static_capability_ready` is set. Mismatches remain safe findings and are
     not absorbed as current-chain readiness.
   - Next action: use this ingress as readiness bookkeeping only; real
     provider execution, host/physical voice evidence, V21 execution, and
     physical StackChan PRD acceptance still require their existing gates.

6. `T-XIAOZHI-HOST-LOCAL-REAL-BASIC-DIALOGUE-SMOKE`
   - Current phase: read-only Gateway/provider audit confirmed `/v1/xiaozhi`
     is the closest product path: real Opus ingress, VAD buffering, voice
     pipeline, and Opus downlink, while ASR/TTS are still not fully streaming.
   - Next action: after an operator-triggered real `/v1/xiaozhi` turn, run
     `xiaozhi-realtime-parity` against the live trace. Do not use
     `/v1/xiaozhi/say`, `xiaozhi-voice-bench`, or `fast-companion-turn` as
     physical parity evidence.

7. `T-XIAOZHI-ASR-PARTIAL-TO-LLM-REALTIME-BRIDGE-001`
   - Current phase: completed host-side candidate in scoped worker.
   - Next action: keep this evidence below PRD acceptance, then use it as the
     ordering gate for real Sherpa ASR runtime proof, authorized realtime TTS
     execution, and a physical stock `/v1/xiaozhi` trace.

8. `T-XIAOZHI-STREAMING-ASR-001`
   - Current phase: host-side streaming ASR session, partial-to-LLM bridge,
     nonblocking commit, stock `stt` ordering, and true-idle wake pre-roll
     buffering are implemented and covered by Gateway tests. Listening Opus
     frames now pass through a bounded per-session ingress queue so decode,
     VAD, and ASR append do not block the WebSocket control loop. Cancelled
     queue items are now suppressed before audio ingress so old-turn audio
     cannot leak into a fresh listening turn. Full runtime proof is still
     missing because no physical stock `/v1/xiaozhi` trace has shown real
     streaming ASR/LLM/TTS profile markers plus playback.
   - Next action: in an approved runtime/hardware window, collect an
     operator-triggered physical stock `/v1/xiaozhi` trace and run
     `xiaozhi-realtime-parity`; do not promote host-loopback, `/say`,
     voice-bench, or file-boundary evidence.

9. `T-XIAOZHI-STREAMING-ASR-PROVIDER-001`
   - Current phase: public Gateway `cloud_edge` runtime bridge is implemented
     in code. `sherpa_onnx_streaming` remains the local streaming ASR seam,
     `doubao_asr_realtime` and `dashscope_qwen_asr_realtime` are now
     selectable cloud `StreamingASRAdapter` profiles, and
     `doubao_tts_realtime` plus `dashscope_qwen_tts_realtime` are selectable
     `StreamingTTSAdapter` profiles. `A21_XIAOZHI_PRODUCT_CHAIN=cloud_edge`
     now defaults the public product chain away from local Sherpa and toward
     cloud ASR + text-stream LLM + realtime TTS. With `A21_DASHSCOPE_API_KEY`
     present and no explicit ASR/TTS override, the cloud-edge default selects
     DashScope ASR/TTS. With `A21_LAB_STEPFUN_API_KEY` present and no explicit
     text override, the cloud-edge default now selects StepFun before falling
     back to DeepSeek. Static readiness can pass for configured cloud ASR,
     configured StepFun or DeepSeek text stream, and configured realtime TTS,
     while `prd_accepted` remains false.
   - Current deployment note: main ECS `47.103.57.217` is running the verified
     cloud-edge build behind Caddy, and remote readiness currently reports
     `dashscope_qwen_asr_realtime + deepseek + dashscope_qwen_tts_realtime`
     because `/etc/a21/secrets/provider.env` contains no StepFun key/model
     entries. The remaining correction for the intended `stepfun` LLM path is
     root-only injection of `A21_LAB_STEPFUN_API_KEY` and
     `A21_STEPFUN_MODEL=step-1-8k`, then service restart and a fresh provider
     bench. Credentialed live provider execution and physical trace proof are
     still pending.
   - Next action: inject StepFun secrets into the ECS root-only provider env if
     available, rerun `xiaozhi-streaming-provider-readiness` to prove
     `llm_profile=stepfun`, then collect an operator-triggered physical stock
     `/v1/xiaozhi` trace and run `xiaozhi-realtime-parity`.

10. `T-WAKE-004-AFE-VS-CUSTOM-PARITY`
   - Current phase: candidate from read-only audio HAL/wake worker
     `019e8a8a-7263-7b20-94b9-9b2847aa741d`.
   - Next action: plan whether to restore official AFE WakeNet behavior or
     harden custom MultiNet `紫悦`; acceptance must be physical wake from idle,
     not screen tap.

11. `T-STACKCHAN-APP-LIFECYCLE-PARITY`
    - Current phase: candidate from read-only audio HAL/wake worker.
    - Next action: plan how to preserve official StackChan AppLauncher,
      AppAiAgent, AppAvatar, AppDance, AppSetup/Mooncake lifecycle while still
      avoiding the welcome/setup trap.

12. `T-HAL-AUDIO-CONFIG-PARITY`
    - Current phase: candidate from read-only audio HAL/wake worker.
    - Next action: record and, if evidence supports it, align CoreS3/StackChan
      codec constants and init order such as ES7210 input gain, AFE/AEC/VAD,
      AW88298 output, and runtime MCP/NVS volume behavior.

13. `T-SERVER-SIDE-PROFESSIONAL-RITUAL-EXECUTION-GATE-001`
    - Current phase: completed code/test/doc transition on 2026-06-04.
      Server-side readiness now requires `professional_ritual_ready` from an
      accepted external Gateway `a21.xiaozhi_professional_bench.v1` report.
      V21 adapter smoke remains adapter-boundary evidence but no longer
      satisfies the professional ritual by itself.
    - Next action: collect fresh provider/V21/professional/roleplay/wake
      reports against the intended runtime, then move the remaining acceptance
      work into the foreground physical StackChan window.

14. `T-PROFESSIONAL-READ-RECORD-READINESS-GATE-001`
    - Current phase: completed code/test/doc transition on 2026-06-04.
      External Gateway professional bench evidence now includes a safe
      `read_record` summary from `/v1/professional-read-records`, and
      server-side readiness exposes `professional_read_record_ready`.
    - Next action: collect fresh runtime reports against the intended Gateway
      and keep real upload indexing, durable account ACL, ECS deployment, and
      physical StackChan professional acceptance as separate transitions.

15. `T-ROLEPLAY-VOICE-RUNTIME-PROBE-CLOSURE-001`
    - Current phase: completed code/test/runtime transition on 2026-06-04.
      Fast Companion roleplay voice-pipeline traces now show selected safe
      voice-profile use and host/simulator playback-start. Product readiness
      accepts safe single-token roleplay prompt-part IDs, so the live local
      `a21 roleplay-voice-probe --require-ready` report now passes.
    - Runtime evidence: `a21-roleplay-voice-probe-20260604-150552.json` and
      `a21-server-side-readiness-bundle-20260604-150611.json`.
    - Next action: configure real A21 provider env and run executed provider
      smoke, then collect physical StackChan PRD acceptance in a foreground
      hardware window.

16. `T-STEPFUN-PROVIDER-SMOKE-SERVER-CANDIDATE-CLOSURE-001`
    - Current phase: completed runtime/docs transition on 2026-06-04.
      StepFun executed streaming provider smoke passed with the local
      A21-namespaced provider env and `A21_STEPFUN_MODEL=step-1-8k`.
      Server-side readiness with the executed StepFun provider report now
      returns `server_side_candidate_ready`.
    - Runtime evidence:
      `reports/provider-live/a21-provider-smoke-20260604-153030-291957000.json`
      and `reports/a21-server-side-readiness-bundle-20260604-153129.json`.
    - Next action: foreground physical StackChan acceptance: device online,
      wake/listen/Opus ingress, ASR/TTS downlink, playback-start/audible
      evidence, roleplay/professional mode behavior, and PRD acceptance.
