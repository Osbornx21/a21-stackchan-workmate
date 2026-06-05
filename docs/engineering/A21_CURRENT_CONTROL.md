# A21 Current Control

Status: current control entry.
Date: 2026-06-05 CST.
Owner: A21 control tower.

This is the short first-read entry for the active launch sprint. It does not
replace `AGENTS.md`, `docs/project_state_machine.md`, the handoff log, release
handoffs, or PRD documents. It points to the current state and the one active
execution plan.

## Current Checkout

- Workspace: `/Users/jiyurun/Documents/New project`
- Branch: `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`
- Sprint start HEAD:
  `b58283b docs(handoff): add internal test 3 master handoff`
- Current source HEAD after the StackChan sys_evt boot-loop recovery and
  avatar relay URL remediation:
  `a95bb868e85e7a0f2fcc9e70e2953aa538af9464`.
- Remote:
  `origin/codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`
- Tracked dirty-state policy:
  do not start launch implementation from unclassified tracked diffs.
- `.DS_Store` policy:
  ignored by `.gitignore`; existing workspace noise may be deleted.
- Source-only control proposal:
  `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md`.
- Current workspace audit:
  `docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`.
- Current integration audit:
  `docs/engineering/A21_INTEGRATION_AUDIT_20260604.md`.

## Current Product State

Internal test 3 is published and accepted for team voice testing, but not full
PRD launch.

- Internal test 3 package:
  `dist/a21-internal-test3-20260603-233245`.
- Tarball:
  `dist/a21-internal-test3-20260603-233245.tar.gz`.
- Package source commit:
  `074e3d877d33`.
- Release docs commit:
  `221c153`.
- Master handoff commit:
  `b58283b`.
- Public Gateway:
  `47.103.57.217`.
- Product Xiaozhi WebSocket:
  `ws://47.103.57.217/v1/xiaozhi`.
- Product device:
  `44:1b:f6:e2:6a:60`.

Runtime truth at the start of this sprint:

- `current_voice_mode`: `dialogue`
- `current_voice_chain_mode`: `cascade`
- `current_asr_profile`: `dashscope_qwen_asr_realtime`
- `current_llm_profile`: `deepseek`
- `current_tts_profile`: `dashscope_qwen_tts_realtime`
- `current_realtime_provider`: `doubao_realtime`
- `current_voice_clone_profile`: `a21_voice_default_dashscope`
- finding: `stepfun_not_selected`

Evidence truth:

- Host bench: `candidate_host_only`.
- Physical report: `candidate_gateway_downlink`.
- Product readiness: `server_side_blocked`.
- Launch ready: false.
- PRD accepted: false.

Live truth after the 2026-06-05 10:20 CST StackChan device-id casefold recovery
deployment:

- Fresh serial evidence after the firmware recovery showed the device remained
  running around the 400 second uptime mark with repeated official avatar relay
  heartbeat pings and no `sys_evt` stack overflow, reboot, Guru, or abort.
- Public Gateway read-only checks exposed a server-side identity split, not a
  fresh firmware brick:
  lowercase `44:1b:f6:e2:6a:60` was online on the Xiaozhi product socket, while
  uppercase `44:1B:F6:E2:6A:60` held the official relay connection record.
- Lowercase official status returned `connected=false`, but uppercase official
  status returned `connected=true`, `delivered_transport=stackchan_official_ws`,
  and `next_action=send_official_control_and_collect_physical_acceptance`.
- Added plan
  `docs/plans/2026-06-05-stackchan-device-id-casefold-recovery.md`.
- Gateway now normalizes MAC-shaped device IDs for official StackChan socket
  register/unregister, status lookup, official control lookup, Xiaozhi fanout
  lookup, and official relay connection registry writes.
- Product recovery now prefers online/latest records when duplicate MAC
  case-variants exist, instead of stopping at the first stale case-insensitive
  match.
- Local verification passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestOfficialStackChanStatusAndControlNormalizeHardwareMACCase|TestOfficialStackChanStatusReportsConnectedFallbackSocket|TestOfficialStackChanControlEndpointDeliversOfficialMotionFrame' -count=1`,
  `GOMAXPROCS=2 go test ./internal/app -run 'TestRunStackChanProductRecovery(PrefersOnlineCaseFoldedDevice|ReadyWhenOnlineAndOfficialRelayConnected|RequiresROMDownloadWhenOfflineWithSerial)' -count=1`,
  `GOMAXPROCS=2 go test ./internal/gateway ./internal/app -run 'OfficialStackChan|ProductRecovery|Xiaozhi|PowerLifecycle' -count=1`,
  and `GOMAXPROCS=2 make verify`.
- This is a server-side state-machine/status fix. It does not flash firmware,
  write NVS, execute provider/V21, perform Git prune/gc, or mark physical
  no-USB power-key/full PRD acceptance.
- Commit `7b32956 fix(gateway): normalize stackchan hardware mac ids` was
  pushed and deployed directly over SSH to ECS
  `i-uf63f4ymqc2dxtljxz2n` / `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path.
- Remote source archive SHA-256:
  `97c78693a6dd1d97a242a497a198d38f79b3bc0e8cbd6e8b5a49b954a516f36b`.
- Remote focused Gateway/App tests passed in `/opt/a21.next`, remote build
  passed, `/opt/a21.next` was safe-swapped to `/opt/a21`, and
  `a21-gateway` restarted active with loopback `/healthz` ok.
- Post-deploy polling showed `/v1/devices` has one normalized lowercase product
  MAC record with `connection_status=online`, and lowercase
  `/v1/stackchan/official/status?device_id=44:1b:f6:e2:6a:60` returns
  `connected=true`, `official_device_id=44:1b:f6:e2:6a:60`, and
  `delivered_transport=stackchan_official_ws`.
- Public read-only product recovery precheck wrote
  `reports/a21-stackchan-product-recovery-20260605-101947.json` with
  `status=product_online_official_relay_ready`, `device_count=1`,
  `device_online=true`, `official_relay.connected=true`, and
  `rom_download_required=false`.
- Public `/v1/devices` also observed live touch input after deploy:
  `last_event=touch.top.swipe_forward`,
  `last_touch_source=top_sensor`.

Live truth after the 2026-06-05 09:54 CST StackChan sys_evt boot-loop recovery
and official avatar relay reconnect transition:

- The product StackChan is not hard-bricked. macOS enumerated the ESP32-S3
  USB/JTAG serial device as `/dev/cu.usbmodem1101`, serial
  `44:1B:F6:E2:6A:60`.
- Serial evidence before the fix showed the visible infinite white flashing
  screen was an app boot loop: the official-compatible app initialized PMIC,
  display, camera, touch, IMU, servo, and audio, then reset after Wi-Fi scan
  with `***ERROR*** A stack overflow in task sys_evt has been detected.`
- Added plan
  `docs/plans/2026-06-05-stackchan-sys-evt-boot-loop-recovery.md`.
- Firmware overlay remediation:
  `Hal::startA21WebSocketAvatarRuntime` now waits for the existing Xiaozhi
  network path instead of starting a second network path, and
  `CONFIG_ESP_SYSTEM_EVENT_TASK_STACK_SIZE=8192` is applied for the heavier
  product runtime.
- Second firmware overlay remediation: the official avatar relay URL now
  percent-encodes MAC-address colons in `device_id`, avoiding the official
  WebSocket parser misreading the query string as part of the host/port and
  failing DNS with `Failed to get host by name`.
- Product app was rebuilt and guarded-flashed through the official-compatible
  product lane only:
  `a21-stackchan-official-xiaozhi-compatible-flash-execute`.
- Current flashed product app artifact SHA-256:
  `c68e9557f065eb40a184ea88d8f111fbc4aa87ca6729c38899cc8555edc7acaf`.
- Current flash report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-095236-1780624356871525000.json`.
- Post-flash serial evidence over a 45 second capture:
  no `stack overflow in task sys_evt`, no `Rebooting...`, no `rst:`, no
  `Guru Meditation`, no `abort()`, Xiaozhi session
  `a21-session-44-1b-f6-e2-6a-60` observed, quiet control websocket kept open,
  official avatar relay waited for Xiaozhi network, DNS failure absent, and
  `WS-Avatar: Connected to server!` with heartbeat pings observed.
- Local tests and build for this transition passed:
  focused official-compatible overlay tests, broader app firmware/product
  recovery tests, `git diff --check`, and
  `GOMAXPROCS=2 make a21-stackchan-official-xiaozhi-compatible-build`.
- The transition does not write NVS, use the generic `xiaozhi.bin` flash lane,
  execute provider/V21, mark no-USB power-key physical acceptance, or claim full
  PRD launch acceptance. Foreground physical UX checks still need the operator
  to observe screen stability, power-key behavior, wake/listen/playback, and
  visible body reactions.

Live truth after the 2026-06-05 08:58 CST Power Lifecycle cold-boot evidence
guard transition:

- Re-read review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` and compared
  the power/lifecycle finding with the current Gateway, protocol, product
  overlay, and recovery state.
- Added plan
  `docs/plans/2026-06-05-power-lifecycle-cold-boot-evidence-guard.md`.
- Tightened `POST /v1/power-lifecycle-acceptance` so product power-key
  acceptance now requires foreground no-USB cold-boot evidence:
  `boot_source=battery_power_key_cold_boot`,
  `usb_connected_during_boot=false`, `power_key_hold_ms` in `250..12000`,
  `pmic_power_key_profile=a21_stackchan_axp2101_pwrkey_v1`, and
  `boot_observed_at_ms>0`.
- Kept the existing requirement that the product device must be online with an
  active Xiaozhi socket before acceptance. An already-online socket alone can no
  longer be recorded as physical cold-boot acceptance.
- `GET /v1/power-lifecycle` now exposes a `pmic_power_key_profile` state item,
  and accepted evidence records redacted boot source, PMIC profile, no-USB
  condition, hold window, and boot observation metadata in device capabilities.
- Focused local Gateway test passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestPowerLifecycle|TestHardwareAcceptance' -count=1`.
- Review-related verification passed:
  `git diff --check`,
  `GOMAXPROCS=2 go test -race ./internal/gateway -run 'PowerLifecycle|OfficialStackChan|Xiaozhi|WorkspaceConsolePageServed' -count=1`,
  `GOMAXPROCS=2 make verify`, `GOMAXPROCS=2 make preflight`, and
  `GOMAXPROCS=2 make doctor`.
- Commit `a181d44 feat(gateway): harden power lifecycle acceptance evidence`
  was pushed to
  `origin/codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
- ECS deployment from this Mac is not completed in this transition. Current
  network/control evidence: Aliyun ECS API TLS closes during handshake through
  the TUN fake-IP route, SSH to `47.103.57.217:22` closes before the SSH banner,
  and public HTTP ports `80`/`21081` accept TCP but return empty application
  replies from this host.
- Recovery package is ready locally at `/tmp/a21-a181d4464af9.tar.gz` with
  SHA-256
  `0d1fd7b40e09d43a10338515189c6c6c7cfc4841729e06cc1c14dde25fb47666`.
  GitHub codeload for commit `a181d4464af90f55f5488f9e78b8ffda61dac16c`
  returns HTTP 200 and can be used from Aliyun Workbench/Cloud Assistant if the
  ECS host has outbound GitHub access.
- This transition does not flash firmware, write NVS, execute provider/V21, or
  mark product power-key physical acceptance. The product still needs the
  guarded official-compatible recovery flash/reconnect window and real no-USB
  cold-boot observation.

Live truth after the 2026-06-05 08:47 CST StackChan product recovery executor
transition:

- Added plan
  `docs/plans/2026-06-05-stackchan-product-recovery-executor.md`.
- `a21 stackchan-product-recovery` remains read-only by default, but now has
  an explicit `--execute-flash` mode for the physical recovery window.
- Execution mode requires confirmation token
  `WRITE_A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_APP`, uses the existing
  hardware control guard, skips flash if the product is already online, and
  calls only the existing official-compatible product flash implementation with
  wait-ROM and `esptool_before=no_reset`.
- The execution report schema is
  `a21.stackchan_product_recovery_execution.v1`; it records precheck, guarded
  flash receipt, and post-flash Gateway/official relay postcheck without
  promoting physical acceptance.
- Added Make targets `stackchan-product-recovery` and
  `stackchan-product-recovery-execute` so the next hardware window has one
  correct command and does not drift to the generic Xiaozhi flash lane.
- Focused local app tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'ProductRecovery|OfficialXiaozhiCompatibleFlash' -count=1`.
- Read-only Make target passed with
  `A21_UPLOAD_PORT=/dev/cu.usbmodem1101 A21_DIRECT_SOURCE_IP=192.168.1.27 GOMAXPROCS=2 make stackchan-product-recovery`
  and wrote
  `reports/a21-stackchan-product-recovery-20260605-084627.json`.
- `git diff --check`, `GOMAXPROCS=2 make verify`,
  `GOMAXPROCS=2 make preflight`, and `GOMAXPROCS=2 make doctor` passed.
- ECS deployment is not required for this transition because the public
  Gateway runtime did not change; the new work is local hardware recovery CLI
  and Makefile workflow.
- Current live product truth is unchanged: Gateway is reachable with direct
  source IP, product device is absent from registry, official relay is
  `connected=false`, USB `/dev/cu.usbmodem1101` is present, and the latest
  guarded flash log still reports ESP32-S3 `No serial data received`.
- No firmware flash, NVS write, provider/V21 execution, reboot/OTA/camera/NFC/IR
  action, or physical acceptance occurred in this transition.

Live truth after the 2026-06-05 08:29 CST Xiaozhi product STT screen and
fast-ack guard transition:

- Re-read review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` and compared
  its final findings against current code and control state. Gateway race,
  namespace/preflight, doctor, power-lifecycle server state, and official MCP
  fallback have already been remediated in later commits; product physical
  recovery and official `/stackChan/ws` acceptance remain open.
- Added plan
  `docs/plans/2026-06-05-xiaozhi-product-stt-screen-and-fast-ack-guard.md`.
- Added `A21_XIAOZHI_STT_SCREEN_POLICY` with `raw`, `status_only`, and `off`
  behavior. Fixture/lab Gateway default remains stock-compatible raw STT text;
  cloud-edge product-chain default is now `status_only`.
- Status-only STT screen policy keeps raw ASR transcript available to the
  internal answer pipeline, but sends the device only a non-sensitive STT
  status phrase and records policy trace markers such as
  `xiaozhi.stt.display.status_only`.
- Cloud-edge product-chain default now sets
  `A21_XIAOZHI_FAST_ACK_ENABLED=false`, so product roleplay waits for real
  answer audio instead of speaking a fast acknowledgement placeholder unless a
  lab run explicitly opts back in.
- Local focused tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'TestGatewayServerOptionsFromEnvCloudEdgeProductChainDoesNotDefaultToLocalSherpa' -count=1` and
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestXiaozhiWebSocketStreamingASRFinalSendsStockSTTBeforeTTS|TestXiaozhiWebSocketSTTScreenPolicyStatusOnlyRedactsDeviceTranscript|TestXiaozhiWebSocketFastAckDisabledWaitsForAnswer' -count=1`.
- Review-related verification passed:
  `git diff --check`,
  `GOMAXPROCS=2 go test -race ./internal/gateway -run 'Xiaozhi|PowerLifecycle|OfficialStackChan|StockProfessionalRoute|WorkspaceConsolePageServed' -count=1`,
  `GOMAXPROCS=2 make verify`, `GOMAXPROCS=2 make preflight`, and
  `GOMAXPROCS=2 make doctor`.
- Commit `79381d4 feat(gateway): guard product stt display and fast ack` was
  pushed and deployed to ECS `47.103.57.217` through Aliyun Cloud Assistant
  over the 5080lab SOCKS path.
- ECS deployment reassembled archive SHA-256
  `ea979684bb4a73952fb0d64c787c2f23089ab5f7f5349798d76e0e8dba444f25`,
  ran remote focused app/Gateway tests, built `/opt/a21.next/bin/a21`, safely
  swapped `/opt/a21`, restarted `a21-gateway`, and passed loopback plus Caddy
  `/healthz`.
- Public SOCKS-path smoke passed for `/healthz`,
  `/v1/stackchan/official/status?device_id=44:1b:f6:e2:6a:60`,
  `/v1/devices`, and `/workspace` containing `allow_mcp_fallback`,
  `fallback_delivered`, and `Relay status`.
- Public live truth is unchanged on the hardware side:
  `/v1/devices` returns `devices=[]`, and official status remains
  `connected=false`, `physical_accepted=false`,
  `next_action=connect_official_stackchan_ws`.
- Post-deploy read-only recovery precheck wrote
  `reports/a21-stackchan-product-recovery-20260605-083905.json` and still
  classified the product as `product_offline_rom_download_required`. It found
  `/dev/cu.usbmodem1101`, but the latest guarded product flash evidence still
  reports `Failed to connect to ESP32-S3: No serial data received`.
- The correct next flash lane remains guarded product-only:
  `a21-stackchan-official-xiaozhi-compatible-flash-execute` with wait-ROM and
  `--esptool-before no_reset` after the device is manually placed in
  ESP32-S3 ROM/download mode.
- This transition does not mark physical power-key, wake, latency, official
  `/stackChan/ws`, or body-action acceptance. Product device recovery and
  post-flash physical validation are still required.

Live truth after the 2026-06-05 08:15 CST official StackChan backend MCP
fallback transition:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was compared against
  current code again. Gateway race/preflight/doctor/verify findings are
  software-remediated; the remaining physical product blocker is still product
  recovery/ROM/download and post-recovery hardware acceptance.
- Added plan
  `docs/plans/2026-06-05-official-stackchan-backend-mcp-fallback.md`.
- `POST /v1/stackchan/official/control` remains strict by default: without an
  official `/stackChan/ws` relay it still returns HTTP 409 unless the request
  explicitly sets `allow_mcp_fallback=true`.
- With `allow_mcp_fallback=true`, Gateway now maps safe official
  state/face/motion semantics to existing bounded Xiaozhi MCP body-preset or
  body-motion sequences and returns `status=fallback_delivered`,
  `delivered_transport=xiaozhi_mcp_sequence`,
  `official_action_fallback_reason=official_stackchan_ws_disconnected`, and
  `official_action_physical_accepted=false`.
- Fallback delivery records `stackchan.official_mcp_fallback.*` trace markers
  and `/v1/devices.runtime_echo` keys including
  `official_stackchan_fallback_status=delivered`,
  `official_stackchan_official_relay=disconnected`, and
  `official_stackchan_packets=0`.
- `/workspace` now sends `allow_mcp_fallback=true` for Official Actions, so the
  product UI no longer depends on catching a 409 before running body fallback.
  The older client-side fallback remains only as compatibility if backend
  fallback fails.
- Focused local Gateway tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestOfficialStackChanControlEndpoint|TestOfficialStackChanStatusReportsDisconnected|TestWorkspaceConsolePageServed' -count=1`.
- Full local verification passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestOfficialStackChanControlEndpoint|TestOfficialStackChanStatus|TestWorkspaceConsolePageServed|TestXiaozhiBodyPreset|TestXiaozhiBodyMotion' -count=1`,
  `GOMAXPROCS=2 go test -race ./internal/gateway -run 'Xiaozhi|PowerLifecycle|OfficialStackChan|StockProfessionalRoute|WorkspaceConsolePageServed' -count=1`,
  `GOMAXPROCS=2 make verify`, `GOMAXPROCS=2 make preflight`, and
  `GOMAXPROCS=2 make doctor`.
- Commit `c5fb24b feat(gateway): add official stackchan mcp fallback` was
  pushed and deployed to ECS `47.103.57.217` through Aliyun Cloud Assistant
  over the existing 5080lab SOCKS path.
- ECS deployment reassembled the source archive at SHA-256
  `7c3c6c03976b01d9e554a140f7a1dc2c54c50fba5dcfddb0e7db54d5f229c287`,
  ran remote focused Gateway tests, built `/opt/a21.next/bin/a21`, safe-swapped
  `/opt/a21.next` to `/opt/a21`, restarted `a21-gateway`, and passed loopback
  plus Caddy `/healthz`.
- Public smoke through the same SOCKS path passed for `/healthz`,
  `/v1/stackchan/official/status?device_id=44:1b:f6:e2:6a:60`, and
  `/workspace` containing `allow_mcp_fallback`, `fallback_delivered`, and
  `official_action_fallback`.
- Public `POST /v1/stackchan/official/control` with
  `allow_mcp_fallback=true` currently returns HTTP 409
  `xiaozhi websocket is not connected` because the product device is still not
  online. This is the expected honest failure mode: backend fallback requires a
  live Xiaozhi MCP socket and does not pretend to move an offline device.
- This transition does not flash firmware, write NVS, execute providers/V21,
  expose reboot/OTA/snapshot/video/camera/NFC/IR/app lifecycle, mark physical
  acceptance, or roll back internal-test3 voice/protocol changes.

Live truth after the 2026-06-05 07:51 CST product recovery precheck
transition:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read and
  compared against current implementation. Its prior P0/P1 software findings
  are no longer the current blocker: the Gateway review race subset passes,
  `make preflight` passes, `make doctor` passes when run outside a parallel
  port collision, and `make verify` passes.
- Added read-only product recovery command:
  `a21 stackchan-accept --check product-recovery` and alias
  `a21 stackchan-product-recovery`.
- The command checks Gateway `/v1/devices`, official relay status
  `/v1/stackchan/official/status`, local USB serial candidates, and the latest
  official-compatible product flash receipt. It never sends device control,
  never flashes, never writes NVS, and never contacts provider or V21 paths.
- Local read-only run against the current product inputs wrote
  `reports/a21-stackchan-product-recovery-20260605-074948.json` and classified
  the current recovery status as `product_offline_rom_download_required`.
- The same report records `/dev/cu.usbmodem1101` present and the latest
  product flash receipt
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-074215-1780616535711678000.json`
  with `flash_executed=false`, `esptool_before=no_reset`, and
  `wait_rom_download_mode=true`.
- Public direct HTTP to `47.103.57.217` remained unstable from this host
  during the run (`EOF` from the CLI direct client and `502` from shell curl),
  so the report keeps Gateway reachability as a finding instead of pretending
  public device state is proven.
- Commit `5ee89f4 feat(app): add stackchan product recovery precheck` was
  pushed and deployed to ECS through Aliyun Cloud Assistant over the 5080lab
  SOCKS path. The source archive was chunked with `SendFile`, SHA-verified on
  the host, tested in `/opt/a21.next`, built, safe-swapped to `/opt/a21`, and
  `a21-gateway` was restarted.
- Remote smoke after deployment passed on the real service port
  `127.0.0.1:21081` and through Caddy on port 80. `/opt/a21/bin/a21
  stackchan-product-recovery --help` exposes the new command. Loopback
  official status still reports `connected=false` and `/v1/devices` still
  reports `devices=[]`.
- 2026-06-05 08:03 CST enhancement: `product-recovery` now accepts
  `--direct-source-ip` and records it in the report, so TUN-safe public Gateway
  checks do not depend on ambient shell environment. It also reads the latest
  guarded flash log and extracts ROM evidence including timeout,
  `rom_no_serial_data`, ESP32-S3 detection, last esptool error, and the
  recovery hint.
- Live run with `--direct-source-ip 192.168.1.27` wrote
  `reports/a21-stackchan-product-recovery-20260605-080244.json`. Gateway
  checks succeeded, `official_relay.checked=true`, `official_relay.connected=false`,
  and `/v1/devices` returned zero devices. The latest flash log evidence is
  `rom_probe_timed_out=true`, `rom_no_serial_data=true`, and
  `LastError="A fatal error occurred: Failed to connect to ESP32-S3: No serial data received."`.
- Commit `9d6c909 feat(app): enrich stackchan recovery diagnostics` was pushed
  and deployed to ECS through the same Cloud Assistant `/opt/a21.next`
  safe-swap path. Remote SHA verification, focused product-recovery app tests,
  remote build, `a21-gateway` restart, loopback `127.0.0.1:21081/healthz`,
  Caddy port 80 `/healthz`, official status, and `/v1/devices` smoke passed.
  Product state remains `official.connected=false` and `devices=[]`.
- Next physical action remains unchanged: put the product StackChan into true
  ESP32-S3 ROM/download mode, rerun the guarded wait-ROM official-compatible
  product flash if needed, then verify product Xiaozhi, official
  `/stackChan/ws`, power-key startup, wake/listen/playback, barge-in, and
  visible screen/RGB/servo/touch behavior.

Live truth after the 2026-06-05 07:30 CST official relay status transition:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was read and compared
  against current HEAD again. Prior review findings for Gateway race,
  namespace/preflight, stock professional mode gating, PMIC power-key parity,
  and physical wake launch-gate semantics remain software-remediated.
- Added plan
  `docs/plans/2026-06-05-official-stackchan-relay-status-surface.md`.
- Gateway now exposes read-only
  `GET /v1/stackchan/official/status?device_id=...` so operators can see
  exact/default official `/stackChan/ws` socket connection, connected age,
  latest trace/session/event, packet count, semantic action surfaces,
  physical acceptance, fallback availability, and `next_action`.
- `/workspace` now has a `Relay status` control and exported
  `official_relay_*` metadata. Official relay delivery remains separate from
  MCP fallback and physical acceptance.
- Verification passed: focused official relay/workspace tests, Gateway review
  race subset,
  `GOMAXPROCS=2 make verify`, `GOMAXPROCS=2 make preflight`, and
  `GOMAXPROCS=2 make doctor`.
- Commit `a139987 feat(gateway): expose official stackchan relay status` was
  pushed and deployed to ECS `47.103.57.217`.
- ECS deployment used Aliyun Cloud Assistant via the 5080lab SOCKS path because
  direct root SSH still closes from this control Mac. Source was transferred
  as chunked Cloud Assistant `SendFile` payloads, reassembled with matching
  SHA-256
  `edaebb34278562f93586bc6487ef46bac6b965c014f9a2b05d29556fcb9eeff5`, then
  deployed through the existing `/opt/a21.next` safe-swap pattern.
- Remote focused Gateway tests passed in `/opt/a21.next`, remote build passed,
  `a21-gateway` restarted active, loopback `/healthz` passed, and loopback
  official status returned
  `connected=false`, `physical_accepted=false`,
  `next_action=connect_official_stackchan_ws`.
- 5080lab public smoke passed for `/healthz`,
  `/v1/stackchan/official/status?device_id=44:1b:f6:e2:6a:60`, and
  `/workspace` containing both `/v1/stackchan/official/status` and
  `Relay status`.
- This transition did not flash firmware, write NVS, execute provider/V21,
  trigger camera/NFC/IR, or promote physical acceptance.
- Remaining product blocker is still physical: recover/flash the product
  StackChan into a state where official `/stackChan/ws`, power-key startup,
  wake/listen/playback/barge-in, and visible body behavior can be accepted.

Live truth after the 2026-06-05 07:42 CST guarded flash retry:

- Public Gateway deployment is live, but product device registry is empty:
  5080lab public `/v1/devices` returned `devices=[]`.
- A short guarded wait-ROM product flash execute was retried on
  `/dev/cu.usbmodem1101` with `--esptool-before no_reset`,
  `--wait-rom`, and a 30 second wait window.
- The T7 flash guard passed on clean HEAD `f6be66d`, but no flash write
  occurred: `flash_executed=false`.
- Report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-074215-1780616535711678000.json`.
- Log:
  `/tmp/a21-stackchan-official-build/a21-official-xiaozhi-compatible-flash-20260605-074141.log`.
- Root evidence remains unchanged:
  `Failed to connect to ESP32-S3: No serial data received`. The product is not
  in ESP32-S3 ROM/download mode.
- Next physical action: manually enter ROM/download mode before retrying
  flash. Hold BOOT/download, press and release RESET, keep holding BOOT until
  the wait-ROM log reports a successful ESP32-S3 `chip_id`, then allow the
  guarded product flash lane to proceed.

Live truth after the 2026-06-05 07:18 CST ROM diagnostic flash attempt:

- A guarded wait-ROM product flash execute was attempted after commit
  `1e8022a`, using the official product artifact and confirmation token. The
  T7 control guard passed, but the command timed out before writing flash:
  `flash_executed=false`.
- A second guarded product flash execute using `default_reset` was attempted
  for comparison. It also failed before writing flash:
  `flash_executed=false`.
- Both failure paths converge on the same root evidence: esptool can see
  `/dev/cu.usbmodem1101`, but cannot synchronize with ESP32-S3 ROM/bootloader
  and reports `No serial data received`.
- macOS USB enumeration still identifies the product device as Espressif
  `USB JTAG/serial debug unit`, serial `44:1B:F6:E2:6A:60`, so the product is
  present on USB but not in a flashable ROM/download state.
- OpenOCD USB-JTAG identify/reset was tried read-only and did not acquire a
  target; it failed at USB descriptor/JTAG setup before any flash action.
- The wait-ROM tool has been enhanced to scan all `/dev/cu.usbmodem*`
  candidates during the wait window, switch to the port where ROM `chip_id`
  succeeds, print periodic candidate status, and include the last esptool
  probe output on timeout.
- Verification passed: focused wait-ROM flash tests, `git diff --check`, and
  `GOMAXPROCS=2 make verify`.
- Next physical action remains: put the product StackChan into true
  ESP32-S3 ROM/download mode. The next software retry should use the enhanced
  wait-ROM product lane so the log shows candidate ports and exact probe
  errors instead of a silent wait.

Live truth after the 2026-06-05 07:07 CST manual-ROM flash guard:

- The review thread and current implementation were compared again. The
  active product blocker remains physical recovery of the product StackChan:
  the device is enumerated on USB serial, but the safe delayed-relay artifact
  still needs a guarded product-lane flash after the operator enters
  ESP32-S3 ROM/download mode.
- The official Xiaozhi-compatible product flash lane now supports an explicit
  wait-for-ROM mode:
  `--wait-rom --wait-rom-timeout-seconds N`.
- The Makefile exposes the same path through
  `A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WAIT_ROM=true` and
  `A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WAIT_ROM_TIMEOUT_SECONDS`.
- Wait-for-ROM mode is guarded: it requires
  `--esptool-before no_reset`, probes only `chip_id` with no reset/no stub
  until the timeout, and then executes the existing exact product artifact
  flash command. It does not relax the A21 control guard, confirmation token,
  product artifact name, clean-worktree guard, or upload-target checks.
- A no-write product flash plan was generated for `/dev/cu.usbmodem1101`:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-070727-1780614447957789000.json`.
  The receipt is `status=ready`, `dry_run=true`,
  `wait_rom_download_mode=true`, `wait_rom_timeout_seconds=90`, and app
  artifact `a21-stackchan-official-xiaozhi-compatible.bin` SHA
  `6c2ba13982efc7570ad0ac9ec0329232af6bedc58cd5ad9b18cc06b7f8f5b8b9`.
- No flash was executed in this transition. The next physical step remains:
  hold BOOT/download, press and release RESET, keep BOOT held until esptool
  detects ESP32-S3 ROM mode, then execute the guarded product flash with
  wait-ROM enabled.
- Verification passed: focused official product flash wait-ROM tests,
  `git diff --check`, `GOMAXPROCS=2 make verify`,
  `GOMAXPROCS=2 make preflight`, and `GOMAXPROCS=2 make doctor`.

Live truth after the 2026-06-05 06:58 CST wake physical launch gate:

- Review-thread wake-word evidence mismatch is now reflected in code:
  server-side wake-word readiness is no longer enough for `launch_ready`.
- `launch_ready` now requires accepted physical wake evidence through the
  existing wake physical acceptance contract.
- Public readiness was re-run with the same source-bound direct-connect setup.
  Result:
  `reports/a21-product-readiness-20260605-065743.json`,
  `status=server_side_candidate_ready`.
- Canonical remaining evidence is now exactly `physical_stackchan_online`,
  `physical_stackchan_prd_acceptance`, and
  `wake_word_physical_acceptance`.
- The report now emits a concrete next action to collect physical wake-word
  acceptance evidence for the active A21 product wake path.
- Verification passed:
  focused wake/launch tests and `GOMAXPROCS=2 make verify`.

Live truth after the 2026-06-05 06:52 CST readiness remote-context alignment:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read. Its
  prior P0/P1 software findings for Gateway race, namespace/preflight, stock
  professional route, and PMIC power-key parity remain remediated on the
  current branch.
- `product-readiness` now adopts the public Gateway voice-chain selected LLM
  profile only for provider-smoke matching when local provider selection is
  unset/mock. Explicit local real providers still win, and no-smoke mock demos
  still report mock honestly.
- Professional external Gateway evidence now suppresses the misleading local
  `A21_V21_ADAPTER_URL` next action when a safe bench/read-record report has
  already proven `external_gateway_ready`; launch readiness still remains
  physical-gated.
- Public readiness was re-run with source-bound direct connect
  `A21_DIRECT_SOURCE_IP=192.168.1.27`, `NO_PROXY=47.103.57.217`, the latest
  StepFun provider smoke report, and `--use-latest-reports`, without setting
  local `A21_PROVIDER_PRIMARY=stepfun`.
- Result:
  `reports/a21-product-readiness-20260605-064932.json`,
  `status=server_side_candidate_ready`. Provider `primary`, `selected`, and
  `smoke_provider` are StepFun. Professional bench/read-record, roleplay voice
  runtime, host voice loopback, wake-word server-side, and voice-chain
  evidence are ready.
- Canonical remaining evidence before the later wake-physical gate was
  `physical_stackchan_online` and `physical_stackchan_prd_acceptance`.
- Tests and gates passed:
  focused app readiness tests, `GOMAXPROCS=2 make verify`,
  `GOMAXPROCS=2 make preflight`, `GOMAXPROCS=2 make doctor`, and
  `GOMAXPROCS=2 go test -race ./internal/gateway -run 'Xiaozhi|PowerLifecycle|OfficialStackChan|StockProfessionalRoute' -count=1`.
- Product device `44:1b:f6:e2:6a:60` remains stale/offline from the public
  Gateway view. The next product action is still physical ROM download entry
  and guarded product app flash of the delayed-relay artifact, followed by
  physical PRD acceptance.

Live truth after the 2026-06-05 05:45 CST stock professional route remediation:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read and
  compared against current implementation after the PMIC flash evidence. A new
  runtime regression was found during fresh ECS readiness: enabling
  `A21_XIAOZHI_STOCK_PROFESSIONAL_ROUTE=true` made stock `realtime` listen
  turns enter professional routing even while A21's selected voice mode was
  roleplay.
- Commit `15a16dc fix(gateway): gate stock professional route by voice mode`
  is deployed on ECS through `/opt/a21.next` safe swap. The stock professional
  route now requires the A21 selected voice mode to be `professional`; default
  roleplay stock turns stay on the normal voice pipeline. Explicit
  `mode=professional` and voice-triggered professional routes remain covered.
- Local verification passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'StockProfessionalRoute|ProfessionalModeDoesNotUsePlaceholder|VoiceTrigger' -count=1`,
  `GOMAXPROCS=2 go test ./internal/app -run 'GatewayServerOptionsFromEnvWiresStockProfessionalRoute|XiaozhiVoiceBench|XiaozhiProfessionalBench' -count=1`,
  `git diff --check`, and `GOMAXPROCS=2 make verify`.
- Remote `/opt/a21.next` focused Gateway/App tests passed, remote build
  passed, `a21-gateway.service` restarted active, and loopback `/healthz`
  passed.
- Fresh real provider smoke passed:
  `reports/a21-provider-smoke-20260605-054222-582464591.json`.
- Fresh roleplay runtime probe passed:
  `reports/a21-roleplay-voice-probe-20260605-054224.json`.
- Fresh repeat-3 Xiaozhi voice bench after the route fix passed with
  `acceptance_status=candidate_host_only`, `failure_count=0`, cloud-edge
  product-chain execution, answer first-audio P95 `1281 ms`, and barge-in stop
  P95 `0 ms`:
  `reports/a21-xiaozhi-voice-bench-20260605-054238.059793905.json`.
- Fresh professional bench passed with `acceptance_status=external_gateway_ready`,
  checking feedback `204 ms`, `read_record.status=completed`, and
  `tts_stop_observed=true`:
  `reports/a21-xiaozhi-professional-bench-20260605-054250.259403329.json`.
- Fresh product readiness is now `server_side_candidate_ready` with
  canonical missing real evidence reduced to `physical_stackchan_prd_acceptance`
  only:
  `reports/a21-product-readiness-20260605-054250.json`.
- Fresh server-side readiness bundle is `server_side_candidate_ready`,
  `candidate_ready=true`, and `collection.status=nothing_missing`:
  `reports/a21-server-side-readiness-bundle-20260605-054250.json`.
- Product roleplay mode ritual was replayed after deploy:
  `a21-trace-mode-ritual-route-fix-15a16dc-20260605` returned HTTP 200
  `status=delivered`, `selected_voice_mode=roleplay`, and 4 steps.
- Product `full_check` was replayed after deploy:
  `a21-trace-full-check-route-fix-15a16dc-20260605` returned HTTP 200
  `status=delivered`, 16 steps, `step_delay_ms=180`, and
  `total_planned_delay_ms=2700`.
- Public hardware acceptance still returns `overall_status=physical_pending`.
  `mode_ritual`, `full_check`, and `power_lifecycle` are delivered but not
  physically accepted.
- Public power lifecycle still returns `overall_status=physical_pending`,
  `xiaozhi_ws_online=true`, and `battery_telemetry=missing`.
- Official `/stackChan/ws` relay remains disconnected. A correct
  `/v1/stackchan/official/control` request returned HTTP 409
  `official stackchan websocket is not connected`. Current body evidence is
  therefore Xiaozhi MCP screen/head/RGB delivery, not official typed-frame
  avatar/action relay product acceptance.
- No firmware flash, no NVS write, no provider secret output, no generic
  `xiaozhi.bin` product flash, no Git prune/gc, and no internal-test3
  voice/protocol rollback occurred.

Live truth after the 2026-06-05 06:38 CST guarded manual bootloader recovery
lane:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read again and
  compared against current HEAD. The remaining unaccepted P0/P1 is still the
  physical official StackChan relay/power recovery path, not a Gateway/ECS
  route absence.
- Commit `409ff0b fix(firmware): allow guarded manual bootloader flash` is
  pushed to
  `origin/codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
- The official Xiaozhi-compatible product flash lane now supports a guarded
  `--esptool-before default_reset|usb_reset|no_reset` option and Makefile env
  `A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_ESPTOOL_BEFORE`. The default
  remains `default_reset`; `no_reset` exists only for manual ROM bootloader
  recovery after a bad product artifact.
- This change does not create an unguarded upload path. Flash execute still
  requires the A21 T7 control guard, clean worktree, exact product artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`, and the confirmation token.
- Local focused App flash/overlay tests, `git diff --check`, and
  `GOMAXPROCS=2 make verify` passed.
- Product rebuild after the recovery-lane commit passed:
  `reports/a21-stackchan-official-baseline-20260605-063356-1780612436653826000.json`.
  Product app SHA:
  `6c2ba13982efc7570ad0ac9ec0329232af6bedc58cd5ad9b18cc06b7f8f5b8b9`.
- Guarded `no_reset` product flash execute was attempted, but failed because
  the chip was not in ROM bootloader/download mode:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-063415-1780612455688329000.json`.
  The flash log shows `--before no_reset` and
  `Failed to connect to ESP32-S3: No serial data received`.
- USB enumeration confirms the connected Espressif USB JTAG/serial device has
  serial/MAC `44:1B:F6:E2:6A:60`; the port is not a wrong-device issue.
- `no_reset_no_sync` direct probe also failed. OpenOCD USB-JTAG read-only probe
  failed at `libusb_get_string_descriptor_ascii() failed with -1`; no JTAG
  flash was attempted.
- The foreground recovery requirement is now physical: hold the board
  `BOOT`/download key, press/release `RESET`, keep holding `BOOT` until
  esptool reports `Chip is ESP32-S3`; if reset is unclear, hold `BOOT` while
  unplugging/replugging USB. Product power key/touch does not enter ROM
  download mode.
- Safe delayed-relay firmware remains build-ready but not flashed. The device
  is still presumed to contain the rejected `61c9fa0` immediate-relay artifact
  until a guarded flash execution succeeds.

Live truth after the 2026-06-05 06:43 CST server-side reconfirmation:

- A fresh esptool `no_reset` probe on `/dev/cu.usbmodem1101` still failed with
  `Failed to connect to ESP32-S3: No serial data received`, so the product
  device has not entered ROM download mode.
- Current Wi-Fi source address is `192.168.1.27`. With TUN active, default
  routes to `47.103.57.217` go through `utun6` / `198.18.0.1` and can return
  false `Empty reply from server`. Source-bind public verification to
  `192.168.1.27` or set `A21_DIRECT_SOURCE_IP=192.168.1.27`.
- Source-bound public checks passed:
  `/healthz`, `/v1/devices`, and `/xiaozhi/ota/` all returned HTTP 200.
- Public `/healthz` returned
  `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`.
- Public `/xiaozhi/ota/` returned
  `ws://47.103.57.217/v1/xiaozhi`.
- Public `/v1/devices` shows product device `44:1b:f6:e2:6a:60` still in the
  registry, but `connection_status=xiaozhi_ws_disconnected` with stale age,
  consistent with the unflashed rejected immediate-relay firmware.
- `GOMAXPROCS=2 make preflight` and `GOMAXPROCS=2 make doctor` passed.
- Review-thread Gateway race regression subset passed:
  `GOMAXPROCS=2 go test -race ./internal/gateway -run 'Xiaozhi|PowerLifecycle|OfficialStackChan|StockProfessionalRoute' -count=1`.
- Re-running readiness with `A21_PROVIDER_PRIMARY=stepfun`,
  `A21_DIRECT_SOURCE_IP=192.168.1.27`, explicit latest provider smoke, and
  `--use-latest-reports` restored the expected server-side state:
  `reports/a21-product-readiness-20260605-064249.json` and
  `reports/a21-server-side-readiness-bundle-20260605-064249.json` are both
  `server_side_candidate_ready`. Canonical missing real evidence is now
  physical: `physical_stackchan_online` and
  `physical_stackchan_prd_acceptance`.

Live truth after the 2026-06-05 06:20 CST official StackChan relay runtime
build:

- Review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was compared against
  the current code after the stock professional route, Gateway race, and PMIC
  power-key remediations. The remaining official StackChan body gap is not an
  ECS endpoint problem: Gateway already exposes `/stackChan/ws`, heartbeat,
  binary official action packets, and `/v1/stackchan/official/control`.
- The product firmware overlay had parked before the Mooncake
  `WebsocketAvatarWorker`, so the official `WebSocketAvatar` was not being
  ticked after WDT-safe direct `GetHAL().startXiaozhi()`.
- The product overlay now keeps direct Xiaozhi start and schedules a delayed
  A21 direct official StackChan avatar relay task before entering the blocking
  `GetHAL().startXiaozhi()` call. The task waits 12 seconds for the Xiaozhi
  Wi-Fi/runtime path to stabilize, starts the official `WebSocketAvatar`
  without system-event logging, and then ticks
  `GetHAL().updateA21WebSocketAvatarRuntime()` every 20 ms.
- The official avatar relay base URL is now controlled by
  `CONFIG_A21_STACKCHAN_OFFICIAL_GATEWAY_BASE_URL="ws://47.103.57.217"`, and
  the official socket appends `device_id` from
  `GetHAL().getFactoryMacString(":")` so product MAC-address controls match
  the registered `/stackChan/ws` socket.
- Focused overlay tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'OfficialXiaozhiCompatibleOverlay(StartsXiaozhiDirectly|RunsOfficialAvatarRelay)' -count=1`.
- Product firmware/app contract tests passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'StackChanOfficial|Official|Firmware|Xiaozhi|Frozen' -count=1`.
- Gateway official/power capability tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'OfficialStackChan|PowerLifecycle|MCPCapabilities' -count=1`.
- `git diff --check` passed.
- First product flash of commit `02955a6c9c28` proved the previous ordering was
  wrong: Xiaozhi reconnected and remained online, but official control still
  returned 409. Boot serial logs showed `A21 starting Xiaozhi mode directly`
  and `HAL start xiaozhi`, but no `A21 starting official StackChan avatar
  relay runtime`, proving `GetHAL().startXiaozhi()` blocks before the later
  relay-start code.
- Flash of the immediate-before-Xiaozhi ordering fix then exposed a second
  firmware issue: startup logs showed the relay runtime beginning Wi-Fi first,
  followed by `***ERROR*** A stack overflow in task sys_evt`. That artifact is
  not acceptable for product use.
- Guarded product rebuild after moving the relay into a delayed background task
  passed. Build report:
  `reports/a21-stackchan-official-baseline-20260605-062018-1780611618644528000.json`.
  Product app artifact:
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`,
  SHA `a1b0262ebf659a268c9c9e578e8b34b1584fed0ec5c20a9c8fc84f5cde7ab92b`.
- This is not yet physical acceptance. Required next evidence is guarded
  product flash of the delayed-task artifact on `/dev/cu.usbmodem1101`,
  reconnect of device
  `44:1b:f6:e2:6a:60`, `/stackChan/ws` online evidence, successful
  `/v1/stackchan/official/control` delivery to the product MAC, and visible
  physical confirmation.
- No NVS write, provider secret output, generic `xiaozhi.bin` product flash,
  Git prune/gc, or internal-test3 voice/protocol rollback occurred.

Live truth after the 2026-06-05 05:31 CST StackChan PMIC power-key parity
product flash:

- Code-review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read after
  the Gateway review-remediation commit. Its remaining power/hardware finding
  was mapped to the product firmware overlay and official StackChan PMIC setup.
- Commit `fda7769 fix(firmware): restore stackchan power key pmic config` is
  pushed to
  `origin/codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
- The product overlay now preserves the WDT-safe direct Xiaozhi start path but
  restores AXP2101 PMIC parity for the physical power-key lifecycle:
  PWRON/OFFLEVEL power-off source handling and 4s hardware power-key
  long-press behavior.
- Focused product-overlay tests passed, including the new PMIC power-key
  lifecycle guard.
- Product build passed with report
  `reports/a21-stackchan-official-baseline-20260605-052214-1780608134186430000.json`.
- Guarded product flash plan and execute passed on `/dev/cu.usbmodem1101`.
  Execution report:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-052359-1780608239212784000.json`.
- The flash report records clean worktree commit `fda7769b23da`,
  `flash_executed=true`, product candidate artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`, and app SHA
  `b665af4e78fae0c4dea10db04a2ea90502d88f26322dcca3c234abb8f355fc3e`.
- Product device `44:1b:f6:e2:6a:60` reconnected to public Gateway with fresh
  `device.heartbeat` after flash.
- Public live `full_check` trace
  `a21-trace-full-check-pmic-key-fda7769-20260605` returned HTTP 200
  `status=delivered` with 16 steps.
- Public live roleplay ritual trace
  `a21-trace-mode-ritual-pmic-key-fda7769-20260605` returned HTTP 200
  `status=delivered`, `selected_voice_mode=roleplay`, and 4 steps.
- Public `GET /v1/power-lifecycle?device_id=44:1b:f6:e2:6a:60` still returns
  `overall_status=physical_pending`, `xiaozhi_ws_online=true`, and
  `battery_telemetry=missing`. This is intentional: the flashed PMIC parity
  fix must still be confirmed by a no-USB physical power-button boot.
- Public `GET /v1/hardware-acceptance?device_id=44:1b:f6:e2:6a:60` returns
  `overall_status=physical_pending` for `mode_ritual`, `full_check`, and
  `power_lifecycle`.
- Public `GET /v1/xiaozhi/mcp-capabilities?device_id=44:1b:f6:e2:6a:60`
  reports only the low-risk status/screen/speaker/head/LED MCP tools as
  allowed, with power shutdown/sleep, reboot, firmware upgrade, camera, NFC,
  infrared, and app lifecycle blocked.
- No generic `xiaozhi.bin` product flash, no NVS write, no provider/V21
  execution, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred.

Live truth after the 2026-06-05 05:13 CST review remediation and power
lifecycle state-machine deployment:

- Code-review thread `019e941c-761b-7ee0-a4b8-68103a0850a1` was re-read.
  Its P0/P1 software findings were mapped to the current worktree and
  remediated where software can safely act: Xiaozhi Gateway session races,
  abort/barge-in pacer blocking, fast-ack roleplay interruption risk, and the
  namespace gate that had blocked `make preflight` / `make doctor`.
- Gateway now has a product power lifecycle state machine:
  `GET /v1/power-lifecycle`, `POST /v1/power-lifecycle-acceptance`, and a
  `power_lifecycle` item in `GET /v1/hardware-acceptance`.
- Power shutdown, sleep, reboot, and firmware upgrade remain blocked from the
  low-risk Xiaozhi MCP surface. `/v1/xiaozhi/mcp-capabilities` reports
  `power_shutdown` and `power_sleep` as blocked tool classes.
- Local verification passed: focused power/hardware Gateway tests, touched
  packages, focused Gateway `-race`, `git diff --check`,
  `GOMAXPROCS=2 make verify`, `GOMAXPROCS=2 make preflight`, and
  `GOMAXPROCS=2 make doctor`.
- ECS `47.103.57.217` was deployed through `/opt/a21.next` safe swap. Remote
  focused Gateway/App/runtimeguard tests passed, remote build passed,
  `a21-gateway.service` restarted active, and loopback/public `/healthz`
  passed.
- Public `/xiaozhi/ota/` returned the product WebSocket
  `ws://47.103.57.217/v1/xiaozhi`.
- Public `GET /v1/power-lifecycle?device_id=44:1b:f6:e2:6a:60` returned
  `overall_status=physical_pending`, `xiaozhi_ws_online=true`, and
  `battery_telemetry=missing`; no-cable cold boot and physical power-button
  start remain `physical_accepted=false`.
- Live product roleplay ritual trace
  `a21-trace-mode-ritual-power-state-20260605-0509` and live `full_check`
  trace `a21-trace-full-check-power-state-20260605-0509` both returned HTTP
  200 `status=delivered`.
- Final public hardware acceptance returned `overall_status=physical_pending`:
  `mode_ritual` and `full_check` are machine-delivered, while
  `power_lifecycle` is online but not physically accepted.
- Public repeat-3 Xiaozhi voice bench
  `reports/a21-xiaozhi-voice-bench-20260605-051249.326210000.json` passed
  with 3/3 answers, 3/3 barge-in turns, `failure_count=0`, answer first-audio
  P95 `1493 ms`, and barge-in stop P95 `22 ms`.
- Product readiness
  `reports/a21-product-readiness-20260605-051255.json` remains
  `server_side_blocked`, `launch_ready=false`, `demo_ready=true`.
  Remaining real evidence gaps are `real_provider_smoke`,
  `physical_stackchan_prd_acceptance`, and `roleplay_voice_runtime`.
- No firmware flash, no NVS write, no provider secret output, no generic
  `xiaozhi.bin` product flash, no Git prune/gc, and no internal-test3
  voice/protocol rollback occurred.

Live truth after the 2026-06-05 03:24 CST product deployment and guarded
official-compatible flash:

- Commit `1c9dece fix(workspace): adopt connected hardware device` is pushed
  to `origin/codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`
  and deployed to ECS `47.103.57.217` through `/opt/a21.next` safe swap.
- `/workspace` now has a `Connected device` control and boot-time device
  adoption. It queries `GET /v1/devices` and replaces the default
  `stackchan-sim-001` input with the online product hardware device when the
  operator has not chosen another device.
- Remote `/opt/a21.next` focused Gateway tests passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestHardwareAcceptance|TestVoiceModeRitual' -count=1`.
  Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
  `a21-gateway.service` restarted active and public direct `/healthz` passed.
- Public `/workspace` smoke found `Connected device`,
  `refreshConnectedDevice`, `preferredConnectedDevice`, `/v1/devices`, and
  `connected_device_count`.
- Product device `44:1b:f6:e2:6a:60` returned online through public
  `/v1/devices`; after the guarded flash it produced a fresh heartbeat with
  `device_age_ms=1228`, then later `device_age_ms=430`.
- The product app was flashed through the guarded official-compatible lane on
  `/dev/cu.usbmodem1101`. The no-write plan report was
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-032219-1780600939242091000.json`;
  the execution report was
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-032331-1780601011958120000.json`.
  The execution report records `status=passed`, `flash_executed=true`, T7
  guard ok, clean worktree, commit `1c9dece8b37e`, and app part
  `a21-stackchan-official-xiaozhi-compatible.bin` at offset `0x20000`.
- No NVS write was performed. Existing Wi-Fi/cloud configuration was preserved.
  No generic `xiaozhi.bin` product flash lane was used.
- After the flash, live roleplay mode ritual trace
  `a21-trace-mode-ritual-after-flash-1c9dece-20260605` returned HTTP 200
  `status=delivered`, `selected_voice_mode=roleplay`, `step_delay_ms=180`,
  `total_planned_delay_ms=540`, `provider_executed=false`, and
  `v21_executed=false`.
- After the flash, live `full_check` trace
  `a21-trace-full-check-after-flash-1c9dece-20260605` returned HTTP 200
  `status=delivered`, `scene=full_check`, 16 redacted screen/RGB/head steps,
  `step_delay_ms=180`, and `total_planned_delay_ms=2700`.
- Final public hardware-acceptance summary for product device
  `44:1b:f6:e2:6a:60` returned `overall_status=physical_pending`; both
  `mode_ritual` and `full_check` were `delivery_status=delivered`, with
  trace/session IDs from the after-flash run and next actions
  `accept_visible_mode_ritual` and `accept_visible_full_check`.
- Physical acceptance remains pending until the operator or an instrument
  confirms visible screen/RGB/head movement from the foreground product.

Live truth after the 2026-06-05 04:10 CST no-cable power/lifecycle recovery:

- The operator reported that the physical power button still did not start the
  prototype as a standalone product. This was not previously accepted; the
  earlier flash only proved USB/flash-reset boot, cloud reconnect, and body MCP
  delivery.
- Plan `docs/plans/2026-06-05-no-cable-boot-power-lifecycle-recovery.md`
  records transition `T-NO-CABLE-BOOT-POWER-LIFECYCLE-001`.
- Commit `fabffd4 fix(firmware): preserve official xiaozhi start lifecycle`
  built and flashed through the guarded product lane, but post-flash
  `mode_ritual` and `full_check` returned HTTP 409
  `xiaozhi websocket is not connected`.
- Read-only serial sampling on `/dev/cu.usbmodem1101` showed repeated task
  watchdog triggers with CPU0 running `main`. Decoding the backtrace against
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.elf`
  pointed at `GetMooncake().uninstallAllApps()` from `app_main`, specifically
  AppSetup/AppLauncher LVGL teardown. The request-lifecycle autostart path is
  therefore rejected for the product lane until a separate teardown transition
  proves it non-regressing.
- Commit `eeeb699 fix(firmware): avoid unsafe mooncake teardown autostart` is
  pushed to
  `origin/codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
  It preserves the WDT-safe direct `GetHAL().startXiaozhi()` product path,
  keeps the app artifact `a21-stackchan-official-xiaozhi-compatible.bin`, and
  records the lifecycle finding in the recovery plan.
- Verification passed:
  `GOMAXPROCS=2 go test ./internal/app -run 'TestOfficialXiaozhiCompatibleOverlayStartsXiaozhiDirectlyBeforeMooncakeTeardown|TestOfficialXiaozhiCompatibleOverlayKeepsA21IdleSocketReady|TestStackChanOfficialCandidateContract' -count=1`,
  `git diff --check`, `make a21-stackchan-official-xiaozhi-compatible-build`,
  and `GOMAXPROCS=2 make verify`.
- The restored product build report is
  `reports/a21-stackchan-official-baseline-20260605-040655-1780603615828792000.json`;
  the app artifact SHA is
  `065e23976722aa7630d0dccf8ee80dff2674785ce9bb6221f67769368a960284`.
- Guarded product flash plan passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-040818-1780603698194451000.json`.
  Guarded product flash execute passed:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-040923-1780603763139018000.json`.
  The execution report records clean worktree, commit `eeeb6998a9b0`,
  `flash_executed=true`, and no generic `xiaozhi.bin` lane.
- After the restored flash, public `/v1/devices` showed product device
  `44:1b:f6:e2:6a:60` online with `device_age_ms=719`. Read-only serial
  showed MultiNet wake commands loaded, audio codec open, quiet control
  WebSocket open, and `WS: Session ID: a21-session-44-1b-f6-e2-6a-60`.
- Live trace `a21-trace-mode-ritual-after-wdt-safe-eeeb699-20260605`
  returned HTTP 200 `status=delivered`; live trace
  `a21-trace-full-check-after-wdt-safe-eeeb699-20260605` returned HTTP 200
  `status=delivered` with 16 redacted screen/RGB/head steps.
- Final public hardware-acceptance summary returned
  `overall_status=physical_pending`: machine delivery evidence is restored,
  but physical button cold boot and visible body acceptance remain unaccepted.
- No NVS write, provider execution, V21 execution, generic product flash lane,
  destructive Git cleanup, or internal-test3 voice/protocol rollback occurred.

Live truth after the 2026-06-05 04:05 CST hardware acceptance summary board
deployment:

- Commit `5d786ef feat(gateway): summarize hardware acceptance` is pushed and
  deployed to ECS `47.103.57.217` through `/opt/a21.next` safe swap.
- Gateway now exposes read-only
  `GET /v1/hardware-acceptance?device_id=<device>`, schema
  `a21.gateway.hardware_acceptance.v1`.
- `/workspace` now includes an `Acceptance Board` with
  `Refresh Acceptance`, `hardwareAcceptanceStatus`, and
  `hardwareAcceptanceItems`. It summarizes `mode_ritual` and `full_check`
  delivery state, physical acceptance booleans, matching acceptance endpoints,
  and next operator actions.
- This summary does not send hardware commands and does not upgrade delivered
  evidence to physical acceptance. It is a control/recovery surface.
- Local red/green evidence: before implementation,
  `TestWorkspaceConsolePageServed` missed `/v1/hardware-acceptance`, and the
  endpoint returned HTTP 404. After implementation:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestHardwareAcceptance|TestVoiceModeRitual|TestXiaozhiBodyScenePhysicalAcceptance|TestXiaozhiBodySceneReportsAndAppliesStepPacing' -count=1`
  passed, `git diff --check` passed, and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestHardwareAcceptance|TestVoiceModeRitual' -count=1`.
  Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
  `a21-gateway.service` restarted active; loopback `/healthz` and public
  direct `/healthz` passed.
- Public `/workspace` smoke found `Acceptance Board`, `Refresh Acceptance`,
  `hardwareAcceptanceStatus`, `hardwareAcceptanceItems`, and
  `/v1/hardware-acceptance`.
- First public summary after restart returned
  `overall_status=machine_evidence_pending`, which was correct because the
  in-memory registry had restarted.
- Live roleplay ritual trace
  `a21-trace-mode-ritual-summary-ready-5d786ef-202606050405`
  returned HTTP 200 with `step_delay_ms=180` and
  `total_planned_delay_ms=540`.
- Live `full_check` trace
  `a21-trace-full-check-summary-ready-5d786ef-202606050405`
  returned HTTP 200 with `step_delay_ms=180`,
  `total_planned_delay_ms=2700`, and trace summary
  `last_offset_ms=2713`.
- Final public hardware-acceptance summary for product device
  `44:1b:f6:e2:6a:60` returned `overall_status=physical_pending`; both
  `mode_ritual` and `full_check` were `delivery_status=delivered`,
  `physical_accepted=false`, with next actions
  `accept_visible_mode_ritual` and `accept_visible_full_check`.
- Final public `/v1/devices` check showed product device online and
  `current_voice_mode=roleplay`. The final body state was the full-check reset
  pose: `screen_theme=auto`, `screen_brightness=55`, RGB `0/0/32`, head
  `yaw=0,pitch=18,speed=200`.
- No physical acceptance was recorded in this round because no operator or
  instrument confirmation was provided.
- Git still emits the historical loose objects/gc warning during commits; no
  `git prune` or manual cleanup was run.

Previous live truth after the 2026-06-05 03:45 CST voice-mode ritual physical
acceptance surface deployment:

- Commit `87625a2 feat(gateway): record mode ritual physical acceptance` is
  pushed and deployed to ECS `47.103.57.217` through `/opt/a21.next` safe
  swap.
- This is a physical-acceptance recording surface for the explicit roleplay /
  professional mode-switch body feedback. It is not internal-test3 voice-chain
  revalidation and did not change ASR, TTS, wake, audio protocol, provider
  execution, V21 execution, firmware, flash, NVS, camera, NFC, infrared, or
  app lifecycle.
- Gateway now exposes `POST /v1/voice-mode-ritual-acceptance`. The endpoint
  requires matching delivered mode-ritual `trace_id` and `session_id`,
  `screen_visible=true`, `rgb_visible=true`, `servo_visible=true`, and
  `observer=operator` or `observer=instrument`; otherwise it rejects instead
  of overclaiming.
- `/workspace` Mode Boundary now includes `Accept Visible Mode Ritual` and
  posts to the new acceptance endpoint only from the latest ritual
  trace/session in the browser state.
- Local red/green evidence: before implementation,
  `TestWorkspaceConsolePageServed` missed
  `/v1/voice-mode-ritual-acceptance`, and the endpoint returned HTTP 404.
  After implementation:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestVoiceModeRitual' -count=1`
  passed, `git diff --check` passed, and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestVoiceModeRitual' -count=1`.
  Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
  `a21-gateway.service` restarted active; loopback `/healthz` and public
  direct `/healthz` passed.
- Public `/workspace` smoke found `Accept Visible Mode Ritual`,
  `acceptModeRitualPhysical`, `modeRitualPhysicalStatus`, and
  `/v1/voice-mode-ritual-acceptance`.
- Public negative acceptance smoke without matching delivered ritual evidence
  returned HTTP 409 with
  `matching voice mode ritual evidence is required before physical acceptance`.
- Live roleplay trace
  `a21-trace-mode-ritual-roleplay-acceptance-ready-87625a2-202606050345`
  returned HTTP 200 with `selected_voice_mode=roleplay`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`, and trace summary
  `last_offset_ms=543`.
- Final public `/v1/devices` check showed product device
  `44:1b:f6:e2:6a:60` online, `current_voice_mode=roleplay`,
  `screen_theme=auto`, `screen_brightness=58`, RGB `120/48/96`, head
  `yaw=0,pitch=24,speed=180`, and
  `voice_mode_ritual_physical_accepted=false`.
- No physical acceptance was recorded in this round because no operator or
  instrument confirmation was provided. The user can now watch the ritual and
  click `Accept Visible Mode Ritual` to persist the acceptance marker.
- Git still emits the historical loose objects/gc warning during commits; no
  `git prune` or manual cleanup was run.

Previous live truth after the 2026-06-05 03:31 CST voice-mode hardware ritual pacing
deployment:

- Commit `a5d9b9d fix(gateway): pace voice mode rituals` is pushed and
  deployed to ECS `47.103.57.217` through `/opt/a21.next` safe swap.
- This is a mode-switch body-feedback deployment, not an internal-test3 voice
  chain revalidation. It did not change ASR, TTS, wake, audio protocol,
  provider execution, V21 execution, firmware, flash, NVS, camera, NFC,
  infrared, or app lifecycle.
- `/v1/voice-mode-ritual` now returns `step_delay_ms` and
  `total_planned_delay_ms` and uses the existing
  `A21_BODY_SCENE_STEP_DELAY_MS` pacing policy. ECS runtime currently records
  `A21_BODY_SCENE_STEP_DELAY_MS=180`,
  `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true`, and
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false`.
- Local red/green evidence: before implementation,
  `TestVoiceModeRitualProfessionalSendsHardwareSequence` failed because
  `step_delay_ms` was missing. After implementation:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestVoiceModeRitual|TestWorkspaceConsolePageServed|TestVoiceModesCatalogDefaultsToRoleplayAndListsProfessional|TestVoiceModeSelectionProfessionalReturnsRitualContract' -count=1`
  passed, `git diff --check` passed, and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestVoiceModeRitual' -count=1`.
  Remote build passed:
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`.
  `a21-gateway.service` restarted active; loopback `/healthz` and public
  direct `/healthz` passed.
- Public `/workspace` smoke found `Run Roleplay Ritual`,
  `Run Professional Ritual`, `modeRitualStatus`, `data-mode-ritual`, and
  `/v1/voice-mode-ritual`.
- Product device `44:1b:f6:e2:6a:60` was online before the live run.
- Live professional trace
  `a21-trace-mode-ritual-professional-paced-a5d9b9d-202606050330`
  returned HTTP 200 with `selected_voice_mode=professional`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`,
  `provider_executed=false`, `v21_executed=false`,
  `official_relay_claimed=false`, and `physical_accepted=false`. Trace summary
  `last_offset_ms=541` proves the four MCP writes were not emitted as a 0 ms
  burst.
- Live roleplay restore trace
  `a21-trace-mode-ritual-roleplay-paced-a5d9b9d-202606050331`
  returned HTTP 200 with `selected_voice_mode=roleplay`,
  `step_delay_ms=180`, `total_planned_delay_ms=540`, and trace summary
  `last_offset_ms=541`.
- Final public `/v1/devices` check showed the device online and restored to
  `current_voice_mode=roleplay`; registry state included
  `screen_theme=auto`, `screen_brightness=58`, RGB `120/48/96`, head
  `yaw=0,pitch=24,speed=180`, and
  `voice_mode_ritual_physical_accepted=false`.
- Physical acceptance is still pending because no operator or instrument has
  confirmed visible mode-switch screen/RGB/head movement.
- Git still emits the historical loose objects/gc warning during commits; no
  `git prune` or manual cleanup was run.

Previous live truth after the 2026-06-05 02:50 CST body-scene physical acceptance
surface deployment:

- Commit `98700ab feat(gateway): record body scene physical acceptance` is
  pushed and deployed to ECS `47.103.57.217` through `/opt/a21.next` safe
  swap.
- Gateway now exposes `POST /v1/xiaozhi/body-scene-acceptance` for
  foreground acceptance of the latest delivered `full_check` only. The endpoint
  requires matching `trace_id` and `session_id`, `screen_visible=true`,
  `rgb_visible=true`, `servo_visible=true`, and `observer=operator` or
  `observer=instrument`; otherwise it rejects instead of overclaiming.
- `/workspace` Hardware Scenes now includes `Accept Visible Full Check` and an
  `operator_pending` acceptance status. The button posts only after a
  `full_check` trace/session is present.
- Local verification before deploy:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyScenePhysicalAcceptance|TestXiaozhiBodySceneReportsAndAppliesStepPacing|TestXiaozhiBodySceneFullCheckRunsOperatorVisibleSequence' -count=1`
  passed, and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests passed and
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`
  passed. ECS `a21-gateway.service` restarted active, loopback `/healthz`
  passed, and public `/healthz` passed.
- Public `/workspace` smoke confirmed `Accept Visible Full Check`,
  `acceptHardwareScenePhysical`, `hardwareSceneAcceptanceStatus`, and
  `/v1/xiaozhi/body-scene-acceptance`.
- Public negative acceptance smoke without matching scene evidence returned
  HTTP 409 with
  `matching body scene evidence is required before physical acceptance`.
- Product trace
  `a21-trace-hardware-full-check-acceptance-ready-98700ab-202606050250`
  returned HTTP 200 `scene=full_check`, `step_delay_ms=180`,
  `total_planned_delay_ms=2700`, and 16 redacted steps. The trace endpoint
  recorded 32 ordered markers with `summary.last_offset_ms=2710`.
- `/v1/devices` recorded `last_body_scene=full_check`,
  `last_body_scene_trace_id` and `last_body_scene_session_id` for the latest
  scene, final `screen_theme=auto`, `screen_brightness=55`, final head
  `yaw=0,pitch=18,speed=200`, final RGB `0/0/32`, and product device
  `44:1b:f6:e2:6a:60` stayed online after a follow-up heartbeat check.
- This deployment does not record physical acceptance by itself. The operator
  still needs to observe screen/RGB/head movement and then click
  `Accept Visible Full Check` or provide explicit confirmation.
- No firmware build/flash, no NVS write, no provider/V21 execution, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred in this code deployment.

Previous live truth after the 2026-06-05 02:30 CST paced full body check deployment:

- Commit `6f43646 feat(gateway): pace hardware body scenes` is pushed and
  deployed to ECS `47.103.57.217` through `/opt/a21.next` safe swap.
- Gateway now paces `POST /v1/xiaozhi/body-scene` sequences so screen/RGB/servo
  scenes are operator-visible instead of emitted in a near-instant burst. The
  `gateway` command defaults to 180 ms between body-scene steps; ECS explicitly
  records `A21_BODY_SCENE_STEP_DELAY_MS=180` in `/etc/a21/runtime.env`.
- Responses now include `step_delay_ms` and `total_planned_delay_ms`, while
  `physical_accepted=false` remains until operator or instrument evidence
  confirms visible movement.
- Local verification before deploy:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestXiaozhiBodySceneReportsAndAppliesStepPacing|TestXiaozhiBodySceneFullCheckRunsOperatorVisibleSequence|TestXiaozhiBodySceneSendsScreenAndBodyMCPSequence' -count=1`
  passed, `GOMAXPROCS=2 go test ./internal/app -run 'TestGatewayServerOptionsFromEnvWiresBodySceneStepDelay' -count=1`
  passed, and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway/App tests passed and
  `GOMAXPROCS=2 /usr/local/go/bin/go build -o /opt/a21.next/bin/a21 ./cmd/a21`
  passed. ECS `a21-gateway.service` restarted active, loopback `/healthz`
  passed, and public `/healthz` passed.
- Product trace
  `a21-trace-hardware-full-check-paced-6f43646-202606050230` returned HTTP 200
  `status=delivered`, `delivered_transport=xiaozhi_mcp_sequence`,
  `scene=full_check`, `step_delay_ms=180`, `total_planned_delay_ms=2700`, and
  16 redacted steps.
- The trace endpoint recorded 32 ordered markers from
  `xiaozhi.body_scene.full_check.step1.screen_theme.sent` through
  `xiaozhi.body_scene.full_check.step16.robot_head_angles_set.sent`, with
  `summary.last_offset_ms=2710`.
- `/v1/devices` recorded `last_body_scene=full_check`,
  `last_body_scene_status=delivered`, `last_body_scene_step=16`, final
  `screen_theme=auto`, `screen_brightness=55`, final head
  `yaw=0,pitch=18,speed=200`, and final RGB `0/0/32`. A follow-up public
  `/v1/devices` check about 12 seconds later still showed product device
  `44:1b:f6:e2:6a:60` online with heartbeat updates.
- This is a hardware body-scene pacing/product-socket evidence cut only. It
  does not reopen, retest, or roll back the internal-test3 voice chain.
- No firmware build/flash, no NVS write, no provider/V21 execution, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred in this code deployment.

Previous live truth after the 2026-06-05 02:17 CST full body check deployment:

- Commit `9171751 feat(gateway): add full body check scene` is pushed and
  deployed to ECS `47.103.57.217` through `/opt/a21.next` safe swap.
- Gateway now accepts `scene=full_check` on `POST /v1/xiaozhi/body-scene`.
  The scene runs a bounded 16-step operator-visible diagnostic sequence:
  screen theme/brightness, RGB, left/right head motion, center pose, focus
  pose, and final reset pose. It uses only the already whitelisted stock MCP
  tools `self.screen.set_theme`, `self.screen.set_brightness`,
  `self.robot.set_led_color`, and `self.robot.set_head_angles`.
- `/workspace` Hardware Scenes now exposes `Full Check` via
  `data-hardware-scene="full_check"` alongside Showtime, Focus, and Reset.
- Local verification before deploy:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyScene' -count=1`
  passed, and `GOMAXPROCS=2 make verify` passed.
- Remote `/opt/a21.next` focused Gateway tests passed and
  `go build -o /opt/a21.next/bin/a21 ./cmd/a21` passed. A non-product
  `--help` binary probe returned unsupported-command status and was not used
  as a deployment gate.
- ECS `a21-gateway.service` restarted active, loopback `/healthz` passed, and
  public `/healthz` plus `/workspace` smoke confirmed `Full Check`,
  `data-hardware-scene="full_check"`, and `/v1/xiaozhi/body-scene`.
- After ECS restart, public `/v1/devices` showed product device
  `44:1b:f6:e2:6a:60` online across 8 heartbeat polls.
- Product trace `a21-trace-hardware-full-check-9171751-202606050217`
  returned HTTP 200 `status=delivered`, `delivered_transport=
  xiaozhi_mcp_sequence`, `scene=full_check`, and 16 redacted steps. The trace
  endpoint recorded 32 markers from
  `xiaozhi.body_scene.full_check.step1.screen_theme.sent` through
  `xiaozhi.body_scene.full_check.step16.robot_head_angles_set.sent`.
- `/v1/devices` recorded `last_body_scene=full_check`,
  `last_body_scene_status=delivered`, `last_body_scene_step=16`, final
  `screen_theme=auto`, `screen_brightness=55`, final head
  `yaw=0,pitch=18,speed=200`, and final RGB `0/0/32`.
- A follow-up public `/v1/devices` check about 12 seconds later still showed
  the device online with heartbeat updates and the full_check registry state.
- This improves the operator hardware-check surface and body-control product
  feel, but it is still machine-readable delivery evidence only:
  `physical_accepted=false` remains until visible operator or instrument
  confirmation.
- No firmware build/flash, no NVS write, no provider/V21 execution, no
  camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app-lifecycle
  exposure, no Git prune/gc, and no internal-test3 voice/protocol rollback
  occurred in this code deployment.

Live truth after the 2026-06-05 02:08 CST guarded product-lane reconnect
flash and hardware scene run:

- The reconnect firmware candidate from commit
  `b9c0baa fix(firmware): tick quiet xiaozhi reconnect checks` was flashed to
  product device `44:1b:f6:e2:6a:60` through the guarded
  `a21-stackchan-official-xiaozhi-compatible` product lane only.
- Flash plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-020621-1780596381380115000.json`
  returned `status=ready`, app artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`, and app SHA-256
  `3eef974929aed78cdd77232897485aaac25bce8aa98daa4d8d78b3d96662b7ac`.
- Flash execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-020726-1780596446779669000.json`
  returned `status=passed`, `flash_allowed=true`, and
  `flash_executed=true`; its control guard verified branch
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`, commit
  `dc8c752f9fee`, and a clean worktree before the write.
- No NVS write occurred. No generic `xiaozhi.bin` or
  `xiaozhi-firmware-flash-*` product-device flash path was used.
- Public `/v1/devices` changed from `connection_status=xiaozhi_ws_disconnected`
  to `connection_status=online` after reboot and kept heartbeat updates for
  device `44:1b:f6:e2:6a:60`.
- Product `showtime` scene trace
  `a21-trace-hardware-showtime-flash-b9c0baa-202606050208` returned HTTP 200
  with `status=delivered`, `delivered_transport=xiaozhi_mcp_sequence`, and 8
  redacted steps: screen theme, screen brightness, RGB, head yaw/pitch,
  RGB, head yaw/pitch, head reset, and final RGB.
- The trace endpoint recorded 16 sent markers for the 8 body-scene steps, and
  `/v1/devices` recorded `last_body_scene=showtime`,
  `last_body_scene_status=delivered`, `screen_theme=dark`,
  `screen_brightness=72`, final head `yaw=0,pitch=24,speed=220`, and final
  RGB `0/36/96`.
- A follow-up public `/v1/devices` check about 12 seconds later still showed
  the device online with heartbeat updates and the same body-scene registry
  state.
- This is strong product-socket and machine-readable MCP delivery evidence for
  the body scene after the reconnect flash. It is not yet physical acceptance:
  `physical_accepted=false` remains until an operator or instrument confirms
  visible screen/RGB/head movement on the device.
- No provider/V21 execution, no camera/NFC/IR expansion, no reboot/OTA/
  snapshot/video/app-lifecycle exposure, no Git prune/gc, and no
  internal-test3 voice/protocol rollback occurred.

Live truth after the 2026-06-05 02:00 CST listen-start state-reaction
suppression and reconnect-candidate cut:

- Commit `72e6bcc fix(gateway): suppress listen-start state reactions` is
  pushed and deployed to ECS `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path.
- Root evidence: product trace `a21-trace-44-1b-f6-e2-6a-60` showed device
  `xiaozhi.hello`, then `xiaozhi.listen.start`, automatic state-reaction MCP
  `robot_led_color`, `xiaozhi.state_reaction.failed`, and socket close
  cancellation markers within 232 ms. This made the hardware scene live smoke
  impossible after Gateway restart.
- Gateway now suppresses automatic state-reaction MCP writes at the stock
  physical `listening/listen_start` boundary and records
  `xiaozhi.state_reaction.listen_start_suppressed` plus
  `last_state_reaction_status=suppressed_listen_start`. Non-listen-start
  state reactions such as `thinking` remain tested.
- Runtime safety action on ECS:
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=false`; playback events, touch events,
  and touch reactions remain enabled. This is a runtime gate rollback for a
  regressed automatic state reaction, not a source rollback.
- Local focused state-reaction/body-scene tests passed, full local
  `GOMAXPROCS=2 make verify` passed, remote focused Gateway tests/build
  passed, `a21-gateway` restarted active, and public `/healthz` plus
  `/workspace` smoke passed.
- Commit `b9c0baa fix(firmware): tick quiet xiaozhi reconnect checks` is
  pushed as a no-flash firmware overlay candidate. It moves the quiet Xiaozhi
  control websocket reconnect check from VAD-change-only probing to periodic
  `MAIN_EVENT_CLOCK_TICK` probing, so idle devices can keep retrying even when
  no new VAD events arrive.
- No-flash firmware build passed:
  `make a21-stackchan-official-xiaozhi-compatible-build`. The generated app
  artifact is
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  with SHA-256
  `3eef974929aed78cdd77232897485aaac25bce8aa98daa4d8d78b3d96662b7ac`.
- Final public `/v1/devices` check showed product device `44:1b:f6:e2:6a:60`
  had reappeared but remained `connection_status=xiaozhi_ws_disconnected`.
  Its capabilities no longer included `xiaozhi_product_state_reactions`, which
  proves the runtime gate rollback took effect. The refreshed trace had no
  `xiaozhi.state_reaction.*` MCP markers; it still showed `listen.start`
  followed by `asr.stream.error`, `asr.stream.cancelled`, and
  `xiaozhi.opus_ingress.queue_cancelled.socket_closed`. Physical
  hardware-scene acceptance is therefore still pending device reconnect
  resilience or a foreground guarded product-lane firmware flash/power-cycle
  window.
- No firmware flash, no NVS write, no provider/V21 execution, no camera/NFC/IR
  expansion, no reboot/OTA/snapshot/video/app-lifecycle exposure, no Git
  prune/gc, and no internal-test3 voice/protocol rollback occurred.

Live truth after the 2026-06-05 01:43 CST workspace hardware-scene cut:

- Commit `88e1549 feat(gateway): add xiaozhi hardware scene sequences` is
  pushed and deployed to ECS `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path.
- Gateway now exposes `POST /v1/xiaozhi/body-scene` for bounded one-click
  `showtime`, `focus`, and `reset` scenes. `showtime` combines screen
  theme/brightness plus RGB/head MCP writes; `focus` and `reset` provide
  shorter bounded workspace states. Responses keep redacted step metadata and
  `physical_accepted=false`.
- `/workspace` now exposes a Hardware Scenes section with Showtime, Focus, and
  Reset controls, safe trace/status display, and `hardware_scene_*` metadata
  export fields. Existing Body Presets, Hardware Screen, and Official Actions
  controls remain available.
- Local focused tests passed:
  `GOMAXPROCS=2 go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestXiaozhiBodyPreset|TestXiaozhiBodyMotion|TestXiaozhiBodyScene' -count=1`.
  Full local `GOMAXPROCS=2 make verify` passed. Remote focused Gateway tests
  and build passed in `/opt/a21.next`; `a21-gateway` restarted active;
  loopback and public direct-source `/healthz` returned ok.
- Public `/workspace` HTML smoke found `Hardware Scenes`,
  `/v1/xiaozhi/body-scene`, `runHardwareScene`,
  `refreshHardwareSceneTrace`, `hardware_scene_trace_id`, and
  `data-hardware-scene="showtime"`.
- Public `showtime` scene execution against product device
  `44:1b:f6:e2:6a:60` returned HTTP 409 `xiaozhi websocket is not connected`,
  and public `/v1/devices` remained empty for the immediate post-deploy poll.
  This is device socket absence after Gateway restart, not a voice-chain
  rollback. Product-socket/physical evidence for the scene remains pending
  until the product device reconnects.
- This cut did not build or flash firmware, write NVS, execute providers/V21,
  expose reboot/OTA/snapshot/video/camera/NFC/IR/app lifecycle, run Git
  prune/gc, or renew internal-test3 voice-chain acceptance.

Live truth after the 2026-06-05 01:32 CST workspace official-action fallback cut:

- Commit `7dfbb10 feat(gateway): fallback official actions to body motion` is
  pushed and deployed to ECS `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path.
- `/workspace` Official Actions still attempts the real
  `/v1/stackchan/official/control` relay first. If the separate `/stackChan/ws`
  official avatar/action socket is disconnected, the UI now records the
  blocked reason and automatically runs the matching
  `/v1/xiaozhi/body-preset` or `/v1/xiaozhi/body-motion` fallback so the
  product still moves through the live Xiaozhi MCP socket.
- Exported metadata now keeps `official_action_blocked_reason` and
  `official_action_fallback` separate from official-frame delivery, and the UI
  shows `fallback_delivered` rather than pretending official packets were sent.
- Local focused workspace/body-motion/official-action tests passed and full
  local `GOMAXPROCS=2 make verify` passed. Remote focused Gateway tests and
  build passed in `/opt/a21.next`; `a21-gateway` restarted active; loopback
  and public direct-source `/healthz` returned ok.
- Public `/workspace` HTML smoke found `officialActionFallback`,
  `runOfficialActionFallback`, `fallback_delivered`,
  `official_action_blocked_reason`, `official_action_fallback`,
  `/v1/stackchan/official/control`, and `/v1/xiaozhi/body-motion`.
- Public official action relay on product device `44:1b:f6:e2:6a:60` still
  truthfully returns HTTP 409 `official stackchan websocket is not connected`.
  Public fallback body-motion `dance` passed for trace
  `a21-trace-workspace-body-motion-dance-7dfbb10`, with 10 generic/body-motion
  markers and `/v1/devices` recording `last_body_motion=dance`, final robot
  head `yaw=0,pitch=24,speed=220`, LED `red=0,green=168,blue=80`, and the
  device online.
- This cut is a product-control ergonomics deployment. It is not a renewed
  internal-test3 voice-chain acceptance run and not an official-frame physical
  acceptance claim. No firmware build/flash, no NVS write, no provider or V21
  execution, no camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app
  lifecycle exposure, and no Git prune/gc occurred.

Live truth after the 2026-06-05 01:26 CST workspace official-action/body-motion cut:

- Commit `6ce372e feat(gateway): expose official actions in workspace console`
  is pushed and deployed to ECS. `/workspace` now exposes an Official Actions
  section for semantic state, face, and motion controls through the existing
  `/v1/stackchan/official/control` relay, with safe status/trace/surface
  metadata and `official_action_physical_accepted=false`.
- Commit `e5ae4d1 feat(gateway): add xiaozhi body motion sequences` is pushed
  and deployed to ECS through the existing `/opt/a21.next` safe-swap path.
  Gateway now exposes `POST /v1/xiaozhi/body-motion` for bounded MCP-backed
  `look_up`, `nod`, `shake`, `dance`, and `stop` sequences, and `/workspace`
  Body Presets now includes those MCP motion controls.
- Local focused workspace/official-action/body-motion tests passed and full
  local `GOMAXPROCS=2 make verify` passed. Remote focused Gateway tests and
  build passed in `/opt/a21.next`; `a21-gateway` restarted active; loopback
  and public direct-source `/healthz` returned ok.
- Public `/workspace` HTML smoke found Official Actions, Body Presets,
  `/v1/stackchan/official/control`, `/v1/xiaozhi/body-motion`,
  `runOfficialAction`, `runBodyMotion`, `official_action_trace_id`, and
  `body_motion_trace_id`.
- Public official action relay on product device `44:1b:f6:e2:6a:60`
  currently returns HTTP 409 `official stackchan websocket is not connected`.
  This is the truthful runtime state for the separate `/stackChan/ws` official
  avatar/action socket and is now visible from the product console instead of
  being hidden.
- Live public `dance` body-motion on product device `44:1b:f6:e2:6a:60`
  passed for trace `a21-trace-workspace-body-motion-dance-e5ae4d1`.
  Response status was `delivered`, transport `xiaozhi_mcp_sequence`, with 5
  redacted steps and `physical_accepted=false`. Trace recorded ordered
  `xiaozhi.body_motion.dance.step*.sent` markers plus generic robot MCP
  markers. Public `/v1/devices` recorded `last_body_motion=dance`,
  `last_body_motion_status=delivered`, `last_body_motion_step=5`, final robot
  head `yaw=0,pitch=24,speed=220`, LED `red=0,green=168,blue=80`, and the
  device remained online.
- This cut is a product action/body-control surface deployment. It is not a
  renewed internal-test3 voice-chain acceptance run and not a voice/protocol
  rollback. No firmware build/flash, no NVS write, no provider or V21
  execution, no camera/NFC/IR expansion, no reboot/OTA/snapshot/video/app
  lifecycle exposure, and no Git prune/gc occurred.

Live truth after the 2026-06-05 01:14 CST workspace hardware-screen console cut:

- Commit `c74261d feat(gateway): expose screen controls in workspace console`
  is pushed and deployed to ECS `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path.
- `/workspace` now exposes a Hardware Screen product control section for
  screen brightness, screen theme, device status, screen info, and MCP
  capabilities. The controls call the existing low-risk MCP/status endpoints
  and display safe trace/status/tool/capability summaries plus safe
  `screen_control_*` metadata.
- Local focused workspace/screen tests passed and full local
  `GOMAXPROCS=2 make verify` passed. Remote focused Gateway tests and build
  passed in `/opt/a21.next`; `a21-gateway` restarted active; loopback and
  public direct-source `/healthz` returned ok.
- Public `/workspace` HTML smoke found `Hardware Screen`, the screen/status
  endpoint calls, `runHardwareScreenAction`, and `screen_control_trace_id`.
- Public MCP capabilities on product device `44:1b:f6:e2:6a:60` returned
  `connection_status=online`, `mcp_advertised=true`, 8 allowed tools, blocked
  high-risk classes for reboot/firmware upgrade/camera/snapshot/video/NFC/IR/
  app lifecycle, `result_redacted=true`, and `physical_accepted=false`.
- Live public brightness/theme/screen-info actions passed for traces
  `a21-trace-workspace-screen-brightness-c74261d`,
  `a21-trace-workspace-screen-theme-c74261d`, and
  `a21-trace-workspace-screen-info-c74261d`. Responses were delivered through
  `xiaozhi_mcp`; trace markers recorded
  `xiaozhi.mcp.screen_brightness.sent`, `xiaozhi.mcp.screen_theme.sent`, and
  `xiaozhi.mcp.screen_info.sent`. Public `/v1/devices` recorded
  `screen_brightness=62`, `screen_theme=dark`, and the device remained online.
- This cut is a hardware screen/status product surface deployment. It is not a
  renewed internal-test3 voice-chain acceptance run and not a voice/protocol
  rollback. No firmware build/flash, no NVS write, no provider or V21
  execution, no camera/NFC/IR expansion, and no Git prune/gc occurred.

Live truth after the 2026-06-05 01:07 CST workspace body-preset console cut:

- Commit `d361176 feat(gateway): expose body presets in workspace console` is
  pushed and deployed to ECS `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path.
- `/workspace` now exposes a Body Presets product control section for
  `ready`, `listening`, `thinking`, `speaking`, `celebrate`, and `reset_idle`.
  The controls call the existing `/v1/xiaozhi/body-preset` endpoint, display
  safe trace/status/LED/head/transport summaries, and export safe
  `body_preset_*` metadata.
- Local focused workspace/body tests passed and full local
  `GOMAXPROCS=2 make verify` passed. Remote focused Gateway tests and build
  passed in `/opt/a21.next`; `a21-gateway` restarted active; loopback and
  public direct-source `/healthz` returned ok.
- Public `/workspace` HTML smoke found `Body Presets`,
  `/v1/xiaozhi/body-preset`, `bodyPresetActions`, and the `celebrate` control.
- Live public `ready` preset against product device `44:1b:f6:e2:6a:60`
  passed for trace `a21-trace-workspace-body-ready-d361176`.
  Response status was `delivered`, transport `xiaozhi_mcp_sequence`,
  `physical_accepted=false`, LED args `red=0,green=36,blue=96`, and head args
  `yaw=0,pitch=22,speed=180`. Trace markers recorded both official MCP sends
  and body-preset markers. Public `/v1/devices` recorded
  `last_body_preset=ready`; the device remained online.
- This cut is a body-control product surface deployment. It is not a renewed
  internal-test3 voice-chain acceptance run and not a voice/protocol rollback.
  No firmware build/flash, no NVS write, no provider or V21 execution, no
  camera/NFC/IR expansion, and no Git prune/gc occurred.

Live truth after the 2026-06-05 00:58 CST Gateway body-preset sequence cut:

- Commit `da77d21 feat(gateway): add xiaozhi body preset sequences` is pushed
  and deployed to ECS `47.103.57.217` through the existing `/opt/a21.next`
  safe-swap path.
- Gateway now exposes `POST /v1/xiaozhi/body-preset` for bounded product
  expression presets: `ready`, `listening`, `thinking`, `speaking`,
  `celebrate`, and `reset_idle`. Each preset expands to exactly two official
  MCP writes on the live Xiaozhi socket: `self.robot.set_led_color` and
  `self.robot.set_head_angles`.
- Remote focused body/MCP tests passed, remote
  `go build -o /opt/a21.next/bin/a21 ./cmd/a21` passed, `a21-gateway`
  restarted active, and public `/healthz` returned ok.
- Live public body-preset execution against product device `44:1b:f6:e2:6a:60`
  passed for trace `a21-trace-live-body-preset-celebrate-202606050058`.
  Response status was `delivered`, transport `xiaozhi_mcp_sequence`,
  `physical_accepted=false`, LED args `red=0,green=168,blue=80`, and head args
  `yaw=18,pitch=36,speed=260`.
- Live trace recorded
  `xiaozhi.body_preset.celebrate.robot_led_color.sent` and
  `xiaozhi.body_preset.celebrate.robot_head_angles_set.sent`. Public
  `/v1/devices` recorded `last_body_preset=celebrate`, robot LED/head values,
  and the device remained online.
- No firmware build/flash, no NVS write, no provider or V21 execution, no
  camera/NFC/IR expansion, and no internal-test3 voice/protocol rollback
  occurred. This is product-socket body-control evidence, not operator-accepted
  visible movement yet.

Live truth after the 2026-06-05 00:48 CST Gateway host-say interrupt
classification cut:

- Root cause of the medium/long `/v1/xiaozhi/say` `502` observation is now
  classified: a user/device touch barge-in cancels the active Xiaozhi turn, but
  the HTTP host-say handler previously treated all aborted downlinks as
  BadGateway.
- Gateway now distinguishes intentional user/device interruption from true
  downlink failure. `barge`, `wake`, `abort`, or `interrupt` cancellation
  reasons return HTTP 200 with `status=interrupted`,
  `interrupt_reason=<safe reason>`, partial `audio_chunks`, and trace marker
  `xiaozhi.say.interrupted`.
- Actual downlink/TTS errors still remain HTTP 502 and continue to record
  `xiaozhi.say.downlink_error`; this cut does not mask provider, codec,
  websocket, or audio delivery failures.
- Focused tests passed for normal host-say delivery, touch interruption, real
  downlink error, WAV host-say delivery, post-host-say suppression, product
  touch barge-in, and product touch reactions. Full
  `GOMAXPROCS=2 make verify` also passed.
- Commit `45f363f fix(gateway): classify xiaozhi say barge-in as interrupted`
  is pushed and deployed to ECS `47.103.57.217` through the existing
  `/opt/a21.next` safe-swap path. Remote focused Gateway tests passed, remote
  `go build -o /opt/a21.next/bin/a21 ./cmd/a21` passed, `a21-gateway`
  restarted active, and loopback plus public `/healthz` returned ok.
- Post-deploy public `/v1/devices` through the TUN-safe direct-source path
  showed product device `44:1b:f6:e2:6a:60` online with fresh
  `last_event=xiaozhi.hello`; the product device reconnected after this
  Gateway restart without firmware flash or NVS write.
- No firmware build/flash, no NVS write, no provider or V21 execution, and no
  internal-test3 voice/protocol rollback occurred in this cut.
- Remaining physical PRD blockers are unchanged: collect one operator-side
  window with wake/listen mic ingress, answer downlink/playback, trusted
  audible or instrument observation, and touch/wake barge-in stop_done.

Live truth after the 2026-06-05 00:36 CST StackChan touch barge-in/body cut:

- Current source HEAD for the flashed product app:
  `9413ed5 fix(stackchan): keep product barge-in alive while speaking`.
- ECS Gateway was already safe-swapped to `9413ed5`; public `/healthz` is ok
  and device `44:1b:f6:e2:6a:60` is online on `47.103.57.217`.
- Guarded official-compatible product flash passed on `/dev/cu.usbmodem1101`
  with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260605-002920-1780590560252437000.json`.
  The flashed app artifact was
  `a21-stackchan-official-xiaozhi-compatible.bin` with SHA-256
  `4af28d25013111777f2bc82befd6b697ea27da3484e1acd7b8eff661007b0de6`.
- The physical speaking/touch trace
  `a21-trace-speaking-barge-cn-20260605003524` proved top-touch barge-in while
  the product device was in a speaking/downlink window. Gateway markers include
  `device.touch.barge_in.received`, `xiaozhi.touch.barge_in`,
  `barge_in.detected`, `playback.stop`, `xiaozhi.abort.received`, and
  `device.playback.stop_done`.
- Fresh physical report
  `reports/a21-xiaozhi-physical-evidence-20260605-003553.699946000.json`
  is still `candidate_gateway_downlink`, but now carries real barge-in stop
  evidence: playback start `59 ms`, `barge_in.detected=true`,
  `barge_in.stop=true`, and stop_done `26 ms`.
- Fresh half-duplex report
  `reports/a21-xiaozhi-half-duplex-acceptance-20260605-003553.730698000.json`
  remains `blocked` only for the remaining physical review blockers:
  `xiaozhi_half_duplex_mic_ingress_missing` and
  `xiaozhi_half_duplex_audible_observation_missing`. It no longer lacks
  barge-in detection, playback stop, or stop_done.
- Evidence reader adaptation: product runtime echo may use the normalized
  device default session `a21-session-44-1b-f6-e2-6a-60` within the same trace
  while the report target remains the explicit request session. The matcher
  still requires the requested session to appear in the trace and still rejects
  cross-device or cross-trace evidence.
- Observed behavior that still needs a follow-up: medium/long host-say downlink
  can return `502` after a long speaking window, but short host-say recovers
  the device to `idle` and records `device.playback.stop_done`. Do not treat
  this as an internal-test3 voice rollback.

Live truth after the 2026-06-04 20:07 CST hardware/network recovery:

- Product app flash is complete through the official-compatible product lane:
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-195734-1780574254123911000.json`,
  app SHA-256
  `e66a41ef486b866b076746bd064af2e3afb75e0a316515921bbc681b89fb36a8`.
- Corrected product NVS is complete:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-200257-1780574577392353000.json`,
  with `wifi_credentials_written=true`, `mutated_entry_count=5`, and
  `servo_calibration_present=true`.
- The correct SSID is `ChinaNet-N6e3` with no spaces around the hyphen.
- Serial evidence shows device `44:1b:f6:e2:6a:60` found `ChinaNet-N6e3`,
  connected with IP `192.168.1.26`, and opened
  `ws://47.103.57.217/v1/xiaozhi`.
- This Mac currently has TUN mode routing `47.103.57.217` through
  `utun6` / `198.18.0.1`. Default public curls may therefore show false
  empty replies. Use `A21_DIRECT_SOURCE_IP=192.168.1.20` or
  `curl --interface 192.168.1.20 --noproxy '*' ...` for public A21
  verification while TUN is active.
- With the source bind, public `/healthz`, `/xiaozhi/ota/`, `/v1/devices`,
  `/v1/voice-chain-profiles`, and `/v1/traces` are reachable and healthy.
- Live `/v1/devices` reports `current_mode=roleplay`,
  `current_voice_chain_mode=cascade`, `current_llm_profile=stepfun`,
  `current_tts_profile=voice_clone_cli`, and last event
  `xiaozhi.tts.opus_frame.downlink`.
- Live trace `a21-trace-44-1b-f6-e2-6a-60` has physical Opus ingress/decode,
  ASR partial/final, provider first content, TTS first audio, Opus downlink,
  four voice-pipeline completions, and one barge-in/playback-stop marker.
- The earlier default SSH path failed because the default key was not accepted.
  A source-bound explicit local Aliyun identity has since been used to deploy
  and restart the public Gateway; SSH is not the current product blocker.

Live truth after the 2026-06-04 20:36 CST official robot MCP body-control cut:

- Gateway commits `8e2f890`, `e93d5d4`, and `60a2132` are deployed on ECS.
  They whitelist official robot head/LED MCP tools, send official-compatible
  numeric JSON-RPC ids, and accept redacted device-side `type=mcp` responses
  without replying with stock-unknown `type=error`.
- Physical serial evidence on `/dev/cu.usbmodem1101` showed official HAL
  execution on product device `44:1b:f6:e2:6a:60`:
  `[HAL-MCP] set_led_color: r=20, g=0, b=168` and
  `[HAL-MCP] motion set_angles: yaw: 12, pitch: 30, speed: 150`.
- Command traces `a21-trace-live-robot-led-responsefix` and
  `a21-trace-live-robot-head-responsefix` recorded the send markers, while
  device session trace `a21-trace-44-1b-f6-e2-6a-60` recorded two
  `xiaozhi.mcp.response.received` markers.
- `/v1/devices` now records `robot_led_red=20`, `robot_led_green=0`,
  `robot_led_blue=168`, `robot_head_yaw=12`, `robot_head_pitch=30`,
  `robot_head_speed=150`, and `xiaozhi_mcp_response=received_redacted`.
- This closes the first visible body-control gap for RGB and head servo MCP
  execution. It does not promote camera, NFC, infrared, no-cable boot,
  app-lifecycle parity, screen visual acceptance, or full PRD physical
  acceptance.
- Operational caveat: after a public Gateway restart, the current device path
  does not reliably auto-reconnect; a hard reset brought the device back
  online for this evidence window. Treat that as a firmware/app-lifecycle
  follow-up, not as a reason to roll back internal-test3 voice changes.

Live truth after the 2026-06-04 20:48 CST named screen/status MCP endpoint cut:

- Gateway commit `4df52b3` is deployed on ECS. The low-risk screen/status
  operation surface now has named public Gateway endpoints:
  `POST /v1/xiaozhi/device-status`,
  `POST /v1/xiaozhi/screen-brightness`,
  `POST /v1/xiaozhi/screen-theme`, and
  `GET /v1/xiaozhi/mcp-capabilities?device_id=<device_id>`.
- These endpoints reuse the same official MCP whitelist and redaction path as
  `/v1/xiaozhi/mcp-control`; they do not expose reboot, upgrade, camera,
  snapshot, stream/video, NFC, infrared, or app-lifecycle controls.
- Public `mcp-capabilities` returned the allowed official tools and blocked
  high-risk classes with `result_redacted=true` and
  `physical_accepted=false`.
- Physical serial evidence on `/dev/cu.usbmodem1101` showed official screen
  execution after the product device reconnected:
  `StackChanAvatarDisplay: SetTheme: dark` and
  `Backlight: Set brightness to 55`.
- Command traces `a21-trace-live-named-device-status-ready`,
  `a21-trace-live-named-brightness-ready`, and
  `a21-trace-live-named-theme` recorded the corresponding
  `xiaozhi.mcp.*.sent` markers. Device session trace
  `a21-trace-44-1b-f6-e2-6a-60` recorded three
  `xiaozhi.mcp.response.received` events after this window.
- `/v1/devices` recorded `screen_theme=dark` and `screen_brightness=55`.
- This is screen/status MCP physical execution evidence, not full visual
  screen product acceptance.
- Reconfirmed lifecycle gap: after the public Gateway safe-swap restart,
  `/v1/devices` was empty until a hardware reset of the product device. The
  reset used repo-local esptool `chip_id` with `--after hard_reset` only; no
  firmware flash or NVS write occurred. The first product keepalive repair is
  now implemented locally but not yet deployed/flashed.

Live truth after the 2026-06-04 21:48 CST Xiaozhi control-channel keepalive
physical cut:

- `T-STACKCHAN-XIAOZHI-CONTROL-KEEPALIVE-001` adds a product-only idle
  control-channel liveness contract for the official-compatible Xiaozhi path.
- Gateway parses `hello.features.keepalive_events`, returns
  `a21.keepalive_events=true` only for hardware-MAC product clients under the
  existing `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS=true` gate, records
  `device.heartbeat`, and still rejects product `state`, `face`, `display`,
  and `motion` device events.
- The official-compatible product overlay now advertises
  `features.keepalive_events=true`, parses the product allowance, sends
  `type=device, kind=heartbeat` through a protocol-layer keepalive timer while
  the product websocket is open, and starts a protocol-layer reconnect task when
  heartbeat send fails after a Gateway restart.
- ECS runtime was updated to include
  `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS=true` in `/etc/a21/runtime.env`; provider
  secrets were not printed.
- Final no-flash product build passed through the guarded lane with
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  at app sha256
  `d43989209ee1be056f8d59b539bc48c6cdd2821fd9645793c18f134b6dee9179`.
- Guarded product flash passed on device `44:1b:f6:e2:6a:60` with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-214705-1780580825810604000.json`.
- Foreground restart evidence passed without device hard reset: Gateway
  restarted at `2026-06-04 21:47:49 CST`; `/v1/devices` was empty for the
  restart window, then device `44:1b:f6:e2:6a:60` reappeared with
  `xiaozhi.hello` at `21:47:57` and resumed `device.heartbeat` at `21:48:08`.
  A later public snapshot recorded `last_event=device.heartbeat` and
  `device_age_ms=1909`.

Live truth after the 2026-06-04 22:28 CST StackChan product touch physical cut:

- `T-STACKCHAN-OFFICIAL-TOUCH-ACTION-EVIDENCE-001` now has product-lane
  physical touch evidence for screen tap, top tap, top directional swipes, and
  top-touch barge-in on device `44:1b:f6:e2:6a:60`.
- Gateway commit `f16e71b` was deployed to ECS `47.103.57.217`; remote focused
  Gateway/App tests passed, remote build passed, `a21-gateway` restarted
  active, and `/healthz` returned ok. ECS root-only runtime env now includes
  `A21_XIAOZHI_PRODUCT_TOUCH_EVENTS=true`; provider secrets were not printed.
- The official-compatible product overlay now advertises
  `features.touch_events=true`, parses `a21.touch_events=true`, bridges screen
  touch and official top-touch HAL gestures into product-safe
  `type=device, kind=touch` events, and still does not advertise debug
  `features.device_events`.
- Guarded no-flash product build passed with app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  at sha256
  `9b8366e387b10ffa784394e965f702734753f1c4c68192f17a11135e3b713216`, report
  `reports/a21-stackchan-official-baseline-20260604-221937-1780582777246064000.json`.
- Guarded product flash passed on `/dev/cu.usbmodem1101` with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-222334-1780583014061627000.json`.
- After flash, public `/v1/devices` showed `xiaozhi_feature_touch_events=true`
  and `xiaozhi_product_touch_events=true`.
- Physical touch acceptance passed:
  `reports/a21-stackchan-touch-acceptance-20260604-222700.json`
  (`screen_touch`, event `device.touch.wake_or_listen.received`, source
  `screen`);
  `reports/a21-stackchan-touch-acceptance-20260604-222708.json`
  (`top_tap`, event `device.touch.top.tap.received`, source `top_sensor`);
  `reports/a21-stackchan-touch-acceptance-20260604-222719.json`
  (`top_swipe_forward`);
  `reports/a21-stackchan-touch-acceptance-20260604-222744.json`
  (`top_swipe_backward`); and
  `reports/a21-stackchan-touch-acceptance-20260604-222818.json`
  (`top_barge_in`, trace `a21-trace-touch-barge-in-20260604`, event
  `device.touch.barge_in.received`, source `top_sensor`).
- Directional top swipes are physically accepted as guided directional touch,
  but still carry `needs_affordance=true`; they should get screen guidance,
  product training, or physical labeling before being treated as frictionless
  everyday UX.
- This cut does not close full physical PRD acceptance, screen visual
  acceptance, richer RGB/servo choreography, camera, NFC, infrared, app
  lifecycle, or no-cable boot/power acceptance.

Live truth after the 2026-06-04 22:48 CST StackChan touch body-reaction cut:

- `T-STACKCHAN-OFFICIAL-TOUCH-BODY-REACTION-001` is deployed on ECS and
  product-gated behind `A21_XIAOZHI_PRODUCT_TOUCH_REACTIONS=true`.
- Gateway now returns `a21.touch_reactions=true` only when the same hardware
  MAC stock Xiaozhi client has product touch allowance and advertises
  `hello.features.mcp=true`; it still rejects debug-only device events from
  the product profile.
- Accepted product touch events now trigger bounded official MCP body
  feedback on the live `/v1/xiaozhi` socket: `self.robot.set_led_color` and
  `self.robot.set_head_angles`. The mapping is deliberately small: low
  RGB values and small pitch/yaw values within the existing robot MCP clamps.
- Device registry now has stable touch evidence fields
  `last_touch_event`, `last_touch_source`, `last_touch_trace_id`,
  `last_touch_session_id`, and `last_touch_seen_ms` so later
  `xiaozhi.mcp.response.received`, Opus, or heartbeat events do not erase the
  touch acceptance signal.
- Physical/runtime evidence passed on device `44:1b:f6:e2:6a:60`:
  report `reports/a21-stackchan-touch-reaction-evidence-20260604-224756.json`,
  trace `a21-trace-44-1b-f6-e2-6a-60`, session
  `a21-session-44-1b-f6-e2-6a-60`.
- The observed touch was `top_swipe_backward` from `top_sensor`; Gateway sent
  LED `red=120, green=60, blue=0` and head `yaw=-18, pitch=24, speed=200`;
  trace markers include `device.touch.top.swipe_backward.received`,
  `xiaozhi.touch_reaction.robot_led_color.sent`,
  `xiaozhi.touch_reaction.robot_head_angles_set.sent`, and two redacted
  `xiaozhi.mcp.response.received` events.
- This is the first real "touch -> body" foreground reaction proof. It does
  not close screen visual acceptance, richer choreography, camera, NFC,
  infrared, no-cable boot/power, app lifecycle, or full PRD physical
  acceptance.

Live truth after the 2026-06-04 23:34 CST Xiaozhi reconnect/voice diagnosis:

- Gateway commit `5852e20` is deployed on ECS and fixes a registry regression
  introduced by the stale-socket guard: a fresh product Xiaozhi hello or
  heartbeat now restores `connection_status=online` after the previous socket
  was marked `xiaozhi_ws_disconnected`.
- ECS remote focused Gateway tests, remote Go build, `a21-gateway` restart,
  and local `/healthz` passed. The deployment used the existing source-bound
  SSH path and did not touch provider secrets.
- Device `44:1b:f6:e2:6a:60` was recovered with read-only chip-id plus
  hard-reset only. After reconnect, `/v1/devices` showed stable
  `connection_status=online` with `xiaozhi.hello` and later
  `device.heartbeat`.
- The physical evidence tooling fix now works: live
  `xiaozhi-physical-evidence` reports the device online instead of
  `gateway data unsafe`. The current blocked report is honest: audio frame
  count is still zero and the trace still lacks Opus decode, PCM ingress, VAD
  speech end, TTS downlink, playback ack, and audible observation.
- Serial capture during a controlled wake/listen attempt showed wake-word
  detection, Gateway reconnect/listen start, and official MCP state-reaction
  execution, but the device transitioned `listening -> idle` almost
  immediately and only then logged wake-word Opus packet encoding. The next
  active root-cause thread is Xiaozhi natural audio ingress, with special
  attention to the current stock-physical server `type=listen` reply
  suppression invariant.
- This is a runtime blocker diagnosis, not a rollback authorization. Internal
  test 3 voice protocol changes, provider routing, product flash guards,
  keepalive, touch, and body-reaction cuts remain preserved.

Live truth after the 2026-06-04 23:48 CST Xiaozhi wake overlay root-cause cut:

- Root-cause review rejected the first listen-reply hypothesis: the official
  Xiaozhi WebSocket runtime forwards non-hello JSON to Application, and the
  Application handler does not process server `type=listen` replies.
- The stronger firmware-side root cause was an overlay hunk with weak context.
  The intended A21 idle-control-channel fix for
  `Application::ContinueWakeWordInvoke` actually matched a nearby helper with
  the same comment, so the product wake path could still return early when
  the device was `Idle` and the quiet control websocket was already open.
- The overlay now carries function-signature context so `git apply --recount`
  patches `ContinueWakeWordInvoke` itself. A targeted host test checks this
  exact hunk and prevents the old false positive.
- `GOMAXPROCS=2 make verify` passed after the overlay/test repair. A flaky
  doctor test was also tightened so it rejects full provider proxy URLs and
  secrets without treating arbitrary timestamp digits as a proxy leak.
- Guarded no-flash product build passed:
  `reports/a21-stackchan-official-baseline-20260604-234848-1780588128571683000.json`.
  The rebuilt app artifact is
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  with SHA-256
  `e8880adbe7982a2e59bf58319d34097cbc16fbfcc2988975c0a19e186d32b305`.
- Post-build source inspection confirmed
  `ContinueWakeWordInvoke` now accepts
  `state == kDeviceStateIdle && protocol_->IsAudioChannelOpened()`. The first
  guarded flash attempt correctly stopped before hardware write because the
  tracked worktree was dirty. Next step is to commit this repair, rerun the
  product-lane flash guard, and collect natural audio ingress evidence.

Live truth after the 2026-06-04 23:55 CST Xiaozhi natural audio ingress cut:

- Commit `43fcd16` was pushed, leaving the worktree clean for the hardware
  guard. The guarded product-lane app flash then passed on
  `/dev/cu.usbmodem1101` with report
  `reports/a21-stackchan-official-xiaozhi-compatible-flash-20260604-235129-1780588289127583000.json`.
  The app artifact was
  `a21-stackchan-official-xiaozhi-compatible.bin` at SHA-256
  `e8880adbe7982a2e59bf58319d34097cbc16fbfcc2988975c0a19e186d32b305`.
- No NVS write was executed in this cut. The device reused the existing
  product NVS, reconnected to public Gateway `47.103.57.217`, and resumed
  `device.heartbeat` after `xiaozhi.hello`.
- A controlled physical wake/listen/speak attempt after the flash produced
  serial evidence of wake detection, `idle -> listening`, AFE startup, device
  VAD stop, `idle -> speaking`, and official MCP state-reaction execution.
- The live trace now has real natural audio ingress:
  `xiaozhi_listen_to_audio_ingress_ms=114`, `asr_first_partial_ms=207`,
  `llm_first_content_ms=434`, `audio_downlink_first_frame_ms=467`,
  `tts_first_audio_ms=766`, `device_playback_start_ms=51`, and
  `answer_first_audio_total_ms=641`.
- The local evidence reader needed one more safe-marker repair because the
  real roleplay trace contains redacted marker `roleplay.prompt_input.used`.
  It now allows that exact marker while still rejecting unsafe transcript,
  prompt, URL, path, raw-audio, and secret values.
- Physical evidence report
  `reports/a21-xiaozhi-physical-evidence-20260604-235541.527765000.json`
  passed as `candidate_gateway_downlink`: device online, stock profile,
  `audio_frame_count=64`, mic available, Opus decode available, PCM ingress
  available, VAD speech end available, TTS downlink available, and device
  playback ack available.
- Half-duplex acceptance report
  `reports/a21-xiaozhi-half-duplex-acceptance-20260604-235541.288482000.json`
  remains `blocked` for the right remaining reasons: missing barge-in
  detected/stop/stop_done evidence and missing operator or instrumented
  audible observation. Full PRD physical acceptance is therefore still not
  green.

## Active Transition

Current active plan:

- `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`

Transition:

- `T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001`

Target:

- Advance internal test 4 without regressing internal test 3: make `roleplay`
  and `professional` the two user-facing modes, keep `dialogue` as a
  compatibility alias, and define the A21 Cloud/Web/App plus V21 Knowledge
  Service/Adapter shape for public-only and personal+public professional query.

Current focused cut:

- Server-side roleplay voice runtime gate plan:
  `docs/plans/2026-06-04-server-side-roleplay-voice-runtime-gate.md`
- Transition:
  `T-SERVER-SIDE-ROLEPLAY-VOICE-RUNTIME-GATE-001`
- Status:
  ready as server-side evidence. Fresh runtime report
  `reports/a21-roleplay-voice-probe-20260604-172311.json` first passed before
  the formal ECS swap; after deploying commit `431c7ec`, final deployed
  evidence `reports/a21-roleplay-voice-probe-20260604-172658.json` also passed
  with selected `a21_roleplay_wry_peer`, scenario `engineer_pushback`,
  `a21_voice_clone_default`, StepFun text stream execution, prompt input,
  voice-clone profile usage, audio downlink, device playback start, and 45
  audio chunks. Server-side bundle
  `reports/a21-server-side-readiness-bundle-20260604-172722.json` is
  `server_side_candidate_ready`. Product readiness
  `reports/a21-product-readiness-20260604-172722.json` remains
  `server_side_candidate_ready` only because physical StackChan PRD acceptance
  is still missing.

Current next cut:

- `T-XIAOZHI-PHYSICAL-PRD-PROMOTE-GATE-001`
- Target:
  close physical StackChan PRD acceptance without weakening internal test 3.
  Required missing evidence is playback ack/start from device or trusted
  runtime echo, playback stop_done/auto_stop where applicable, and operator or
  instrumented audible observation. Do not treat Gateway downlink or host-only
  voice bench as physical PRD acceptance.
- Current implementation cut:
  product-safe playback acknowledgement negotiation is now implemented on both
  sides locally. Gateway parses `hello.features.playback_events` and exposes
  explicit env `A21_XIAOZHI_PRODUCT_PLAYBACK_EVENTS`, default off. The
  official-compatible product overlay advertises `playback_events`, parses the
  product allowance, sends playback `start` after the official audio output
  task begins playback, and sends `stop_done` after TTS stop or local abort
  queue clear. When enabled, only a hardware-MAC device that does not request
  debug features receives `a21.profile=product` /
  `a21.playback_events=true`; only playback `start` / `stop_done` events are
  accepted. This is an adaptation path for physical evidence collection, not
  accepted PRD evidence by itself.
- Build evidence:
  no-flash product build passed with local dependency cache. Report
  `reports/a21-stackchan-official-baseline-20260604-175756-1780567076043462000.json`
  produced app artifact
  `/tmp/a21-stackchan-official-build/a21-stackchan-official-xiaozhi-compatible.bin`
  at offset `0x20000`, SHA-256
  `e66a41ef486b866b076746bd064af2e3afb75e0a316515921bbc681b89fb36a8`.
  This is build evidence only; no flash or NVS write occurred.

Current cloud/workspace product cut:

- `T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001`
- Target:
  make the PRD "users bind devices, upload personal workspace documents, and
  control what hardware may consult in professional mode" shape concrete
  without waiting for real indexing or physical PRD promotion.
- Status:
  Gateway now exposes memory-only `GET/POST/PUT
  /v1/workspace-device-bindings`, adds device-binding summary to
  `/v1/professional-workspace`, exposes bind/revoke controls in `/workspace`,
  and gates professional mock/Xiaozhi/Web-App consult turns before V21
  execution whenever a workspace has binding records. Gateway now also exposes
  formal `POST /v1/professional-query` with schema
  `a21.gateway.professional_query.v1`; `/workspace` professional probe uses
  that endpoint rather than `/v1/mock-turn`. No binding records keeps old
  internal-test flows `open_until_binding_configured`; bound devices may query;
  unbound, revoked, deleted, or query-scope-denied devices fail before
  `v21.query.start` with safe read-ledger failure codes.
- Boundary:
  this is a Gateway cloud/workspace product-surface cut. It does not perform
  real document indexing, durable cloud auth/tenant ACL, V21 merge/release,
  provider deployment, store pairing secrets, touch firmware, flash, write NVS,
  or promote physical StackChan PRD acceptance.

## Scoped Hardware Parity Transition

Transition:

- `T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001`

Plan:

- `docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`

Status:

- Gap map frozen in
  `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`.
- This is a docs/state baseline only. It does not expose new Gateway controls,
  start Gateway, execute providers or V21, build firmware, flash firmware,
  touch serial, or write NVS.
- Low-risk MCP/status parity has also landed as a Gateway contract through
  `POST /v1/xiaozhi/mcp-control` for only
  `self.audio_speaker.set_volume`, `self.get_device_status`,
  `self.screen.set_brightness`, `self.screen.set_theme`, and
  `self.screen.get_info`.
- Status-display registry parity has landed as Gateway/protocol state:
  `/v1/devices` now carries normalized `display_state` metadata and
  trace/session linkage from stock Xiaozhi turn states or A21 device events.
  This is not physical screen acceptance; `display_state_physical_accepted`
  remains false until a separate hardware evidence report exists.
- Official avatar/action parity has landed as a host/Gateway mapping contract:
  `BuildOfficialActionPlan` maps state/face/motion to official
  `ControlAvatar`, `ControlMotion`, or `DanceSequence` packets plus redacted
  action metadata. `servo_x` remains candidate-only and RGB is
  `*_no_rgb_frame` metadata until a hardware evidence window proves more.
- Official StackChan/Xiaozhi capabilities remain reference material. A21
  product availability still requires A21 evidence and the promotion gates in
  the capability charter.

Current official-source reference:

- StackChan root:
  `da156e1fa0e1c2a5e00b78fbf69b1f7e7bca0483`, dirty working tree, read as
  working-tree reference.
- Xiaozhi sub-tree:
  `e77dedb1309153bb63fed285772962c920c97dd4`, detached clean `HEAD`, read as
  `HEAD` reference.

Next operator/control action:

- Dispatch `T-STACKCHAN-OFFICIAL-ACTION-PHYSICAL-EVIDENCE-001` only in an
  approved foreground hardware window for touch/barge-in/visible action
  evidence.
- Keep reboot, upgrade, camera/photo, screen snapshot, camera stream/video,
  NFC, infrared, app lifecycle, firmware, flash, serial, and NVS out of the
  next worker.

## Current Evidence Manifest

Current manifest:

- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`

Launch decisions must use the manifest plus the named reports. Do not infer
launch truth from raw report directory mtime.

## Active Workers

No active worker writer is currently authorized.

Recent thread control snapshot:

- A21 server mainline control thread:
  `019e7f81-e218-7df1-8743-1ed66e7ddd37`, status `idle`.
- A21 hardware control thread:
  `019e8873-52f0-7570-8efd-04b899db7d4e`, status `notLoaded`.
- Historical worktrees remain registered for evidence and branch history, but
  they are not the active write surface for this transition.
- Do not delete old worktrees, prune loose objects, or reinterpret old worker
  outputs as current evidence during the launch-critical path.

## Allowed Commands For This Transition

Host-only and read-only commands:

```bash
git status --short --branch
git diff --stat
go test ./internal/gateway -run 'Xiaozhi|WriteXiaozhi|OfficialStackChan|TraceEndpointReturnsVoicePipelineSplitSummary' -count=1
go test ./internal/app -run 'XiaozhiPhysicalEvidence|XiaozhiHalfDuplex|ProductReadiness|ServerSideReadinessBundle|XiaozhiVoiceBench|ProviderLatencyBench' -count=1
go test ./internal/providers -run 'DashScope|Doubao|ProviderSmoke|TextStream|VoicePipelineAdapters|GatewayVoiceProvider|Realtime' -count=1
go test ./internal/transport/xiaozhi ./internal/transport/stackchan ./internal/v21adapter ./internal/personality -count=1
git diff --check
make verify
make preflight
make doctor
```

Public Gateway read-only checks require direct routing:

```bash
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 curl -sS http://47.103.57.217/healthz
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 curl -sS http://47.103.57.217/v1/devices
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 curl -sS http://47.103.57.217/v1/voice-chain-profiles
NO_PROXY=47.103.57.217 no_proxy=47.103.57.217 curl -sS http://47.103.57.217/xiaozhi/ota/
```

## Forbidden Commands For This Transition

- No bare `xiaozhi.bin` product flash.
- No generic `xiaozhi-firmware-flash-*` product StackChan flashing.
- No provider execute from Mac except the already scoped StepFun evidence
  recorded in the manifest.
- No V21 execute unless the adapter boundary is configured and this active
  professional validation plan's redaction/contract rules are followed.
- No ECS root secret edits until the StepFun switch has its own guarded
  runtime task.
- No hardware write or NVS write from this sprint unless a foreground hardware
  window is opened with its own plan.
- No report deletion or archive moves.

## Last Verified Facts

From the internal test 3 master handoff and sprint kickoff:

- Internal test 3 tarball SHA check passed.
- Public Gateway health returned `status=ok`.
- Product device `44:1b:f6:e2:6a:60` was online.
- `/v1/voice-chain-profiles` still reported selected LLM `deepseek` and
  finding `stepfun_not_selected`.
- `/xiaozhi/ota/` returned `ws://47.103.57.217/v1/xiaozhi`.
- Targeted read-only test listings showed existing Gateway/App/Provider/
  Transport/V21/Personality coverage for the relevant protocol surfaces.

## Latest Control-Tower Result - 2026-06-04 02:13 CST

The protocol-adaptation worker round has landed locally and passed the unified
verification surface.

Accepted local results:

- Gateway stock `/v1/xiaozhi` regression coverage passed.
- Product readiness, server-side readiness, physical evidence, and
  half-duplex evidence adaptation passed.
- Official-compatible product-lane firmware evidence pointer support passed.
- Provider selector and StepFun/DeepSeek role reporting passed.
- `make verify`: passed.
- A post-documentation default `make verify` retry was killed by the host with
  `Killed: 9`; follow-up `go test -p 1 ./...`, `git diff --check`, and
  `GOMAXPROCS=2 make verify` all passed, classifying the kill as a local
  resource/concurrency interruption rather than a test failure.
- `make preflight`: passed with only
  `wake_word_firmware_build_required`.
- `make doctor`: passed with only `wake_word_firmware_build_required`; doctor
  now reports official-compatible product-lane flash evidence as satisfied.

Public Gateway retry result:

- `http://47.103.57.217/healthz`: recovered and returned
  `{"service":"a21-gateway","status":"ok","version":"0.1.0-dev"}`.
- `http://47.103.57.217/v1/devices`: product device
  `44:1b:f6:e2:6a:60` online, current mode `local_fallback`, selector state
  cascade/DashScope ASR/DeepSeek/DashScope TTS.
- `http://47.103.57.217/v1/voice-chain-profiles`: still reports selected LLM
  `deepseek` and finding `stepfun_not_selected`.
- `http://47.103.57.217/xiaozhi/ota/`: returns
  `ws://47.103.57.217/v1/xiaozhi`.
- Public `:21081` still gives `Empty reply from server` from this Mac; this is
  not the product entrypoint while public `80` is healthy.
- `ssh root@47.103.57.217`: currently rejects this local key with
  `Permission denied (publickey)`, so remote systemd/secret inspection and
  service restart are blocked from this control thread.

Current conclusion:

- Local protocol/readiness/provider/firmware-evidence adaptation is accepted.
- Public product HTTP entrypoint is currently reachable again.
- Full launch remains blocked by `stepfun_not_selected`, physical playback
  ack/stop_done/trusted audible evidence, and unavailable remote control-plane
  authentication for a guarded StepFun switch.

## Latest Control-Tower Result - 2026-06-04 02:24 CST

The ECS control-plane blocker was narrowed.

- SSH succeeds when using an explicit existing local SSH identity; the default
  SSH agent still has no loaded identities.
- Remote `a21-gateway`: active.
- Remote `caddy`: active.
- Remote `127.0.0.1:21081/healthz`: ok.
- Remote provider env file: present, owner `root:root`, mode `600`.
- `A21_LAB_STEPFUN_API_KEY`: missing on ECS.
- `A21_STEPFUN_MODEL`: missing on ECS.
- `A21_DASHSCOPE_API_KEY`: present on ECS.
- Local control-machine StepFun/DashScope env names are also missing.

Per the transition stop rules, no StepFun hot switch was attempted.

Fresh readiness evidence:

- `reports/a21-product-readiness-20260604-022419.json`:
  `status=server_side_blocked`, `launch_ready=false`, selected LLM `deepseek`,
  finding `stepfun_not_selected`, missing real evidence includes
  `real_provider_smoke`, `physical_stackchan_prd_acceptance`, and
  `stepfun_not_selected`.
- `reports/a21-server-side-readiness-bundle-20260604-022420.json`:
  `status=server_side_blocked`, `candidate_ready=false`, missing evidence
  `provider_smoke` and `stepfun_not_selected`.

Current conclusion:

- Public Gateway and ECS service health are good.

## Latest Control-Tower Result - 2026-06-04 02:28 CST

Remote binary drift was identified before the StepFun switch.

- ECS-side `xiaozhi-streaming-provider-readiness` was run with selector env
  names forcing StepFun while sourcing the root-only provider env file.
- The remote binary reported `gate_status=passed` for StepFun even though the
  direct env-name check showed `A21_LAB_STEPFUN_API_KEY` and
  `A21_STEPFUN_MODEL` missing.
- This is stale remote binary behavior relative to the local verified code,
  where StepFun selected with missing env names is blocked and reports
  env-name-only missing fields.

New active runtime safety plan:

- `docs/plans/2026-06-04-public-gateway-code-sync-before-stepfun.md`

Current execution order:

1. Commit the locally verified protocol/readiness/provider-selector changes.
2. Deploy that commit to ECS without changing provider secrets or selector
   state.
3. Re-run remote static StepFun readiness and require it to block on missing
   env names.
4. Resume StepFun env provisioning and switch only after the remote binary is
   synced.

## Latest Control-Tower Result - 2026-06-04 03:07 CST

The control thread re-read the internal test 3 master handoff, current plans,
code, public Gateway snapshots, and latest reports before continuing.

Runtime truth has moved beyond the 02:28 env/code-sync blocker:

- Public `http://47.103.57.217/healthz`: healthy.
- Public `/v1/voice-chain-profiles`: selected LLM `stepfun`.
- Public `/v1/devices`: product device `44:1b:f6:e2:6a:60` online with
  cascade DashScope ASR, StepFun LLM, and fixed DashScope TTS.
- Public `/xiaozhi/ota/`: still returns
  `ws://47.103.57.217/v1/xiaozhi`.
- Host bench `reports/a21-xiaozhi-voice-bench-20260604-023616.742713000.json`
  executed the cloud-edge chain with ASR `dashscope_qwen_asr_realtime`, LLM
  `stepfun`, and TTS `dashscope_qwen_tts_realtime`.
- Remote provider smoke
  `reports/a21-provider-smoke-20260604-023711-678466985.json` passed execution
  and streaming checks but still has `route_eligible=false`, so readiness
  rejects it as `provider_smoke`.

New active control plan:

- `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`

Current conclusion:

- Do not repeat the DeepSeek-to-StepFun runtime switch work; it is already
  reflected in the public selector snapshot.
- The current server-side blocker is StepFun route-eligibility promotion and a
  fresh route-eligible provider smoke report.
- Full PRD launch still remains blocked by physical playback ack, stop_done, or
  trusted audible/instrument observation.

## One Recommended Next Action

Execute `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`: promote
the built-in StepFun profile to explicit route-eligible launch-policy status,
deploy the committed patch to ECS, run fresh redacted StepFun provider smoke,
and then re-run product/server-side readiness. Do not touch internal test 3
Gateway protocol, firmware flash state, or endpoint-side voice acceptance.

## Latest Control-Tower Result - 2026-06-04 03:17 CST

Local promotion is committed; remote deploy is blocked by ECS SSH/control-plane
access.

- Commit:
  `3741c4a feat(providers): promote stepfun route eligibility`.
- Local verification before commit:
  `go test ./internal/providers -run 'ProviderCatalog|ProviderSmoke' -count=1`
  passed;
  `go test ./internal/app -run 'LocalVoiceLoopbackCanUseCompatibilityTextStream|ProductReadiness|ServerSideReadinessBundle|XiaozhiStreamingProviderReadiness' -count=1`
  passed;
  `git diff --check` passed;
  `GOMAXPROCS=2 make verify` passed.
- Local dry-run shape after commit:
  StepFun provider smoke reports `route_eligible=true` when configured, but
  remains `executed=false` and is not launch evidence.
- ECS deploy attempt stopped before remote mutation because this thread has no
  accepted SSH identity for `root@47.103.57.217`; default SSH and local
  `~/.ssh` candidates are unusable.

Current conclusion:

- The source-level StepFun route-eligibility blocker is closed locally.
- Public Gateway has not yet been updated to commit `3741c4a` from this
  thread.
- The next action is control-plane recovery, then deploy `3741c4a`, run fresh
  remote executed StepFun provider smoke, and rerun readiness.
- Workspace cleanup is constrained by
  `docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`: clean current
  mainline noise and stale registrations only; do not delete real branches,
  existing worker worktrees, stashes, reports, firmware artifacts, or evidence
  during launch-critical work.
- Integration cleanup/review is constrained by
  `docs/engineering/A21_INTEGRATION_AUDIT_20260604.md`: internal test 3 commits
  are confirmed ancestors of current `HEAD`; do not perform broad revert or
  branch cleanup that could erase accepted endpoint-side protocol progress.

## Latest Control-Tower Result - 2026-06-04 03:33 CST

The current mainline commits are now pushed to the remote tracking branch, but
the ECS runtime remains blocked by control-plane and public Gateway health.

- Pushed to
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`:
  `765ed41`, `3741c4a`, `8396261`, and `5dba606`.
- `git branch -r --contains` confirms those commits are now present on the
  remote tracking branch.
- Public TCP ports `22`, `80`, `443`, and `21081` on `47.103.57.217` accept
  connections.
- Public HTTP paths `/healthz`, `/v1/devices`, `/v1/voice-chain-profiles`, and
  `/xiaozhi/ota/` currently return empty HTTP replies from this control Mac.
- Default SSH remains unusable for `root@47.103.57.217`; no ECS files,
  services, secrets, selectors, firmware, flash state, or NVS were mutated.

Current conclusion:

- Repository integration is now recovered on the remote branch.
- ECS deploy of `3741c4a` or newer is still pending.
- Public Gateway health is now an active runtime blocker that requires ECS
  control-plane access or an approved operator on the host.
- Do not rerun or reinterpret older provider/readiness reports as launch
  evidence while the public Gateway is returning empty replies.

## Latest Control-Tower Result - 2026-06-04 04:02 CST

The ECS runtime blocker is resolved for the A21 launch path, and the StepFun
route-eligibility transition has moved from provider blocker to V21/physical
blocker.

- Deployed to ECS through the approved jump path and `/opt/a21.next` safe swap:
  `d9362a7 feat(readiness): accept cloud-edge xiaozhi evidence`.
- Remote focused tests before swap:
  `go test ./internal/app -run 'ProductReadiness|ServerSideReadinessBundle|XiaozhiVoiceBench|ProviderLatency' -count=1`
  passed;
  `go test ./internal/providers -run 'ProviderCatalog|ProviderSmoke' -count=1`
  passed.
- Remote `a21-gateway`: active.
- Remote Caddy: active.
- Remote `127.0.0.1:21081/healthz`: ok.
- Remote selector after restart: cascade DashScope ASR, StepFun LLM, fixed
  DashScope TTS, `findings=null`.
- ECS provider env file remains root-owned and mode `600`; only A21 profile
  selector IDs were changed, not provider secret values.
- Fresh StepFun provider smoke:
  `reports/a21-provider-smoke-20260604-035200-977132343.json`, `passed`,
  `executed=true`, `stream=true`, `route_eligible=true`, no fallback.
- Fresh static Xiaozhi provider readiness:
  `reports/a21-xiaozhi-streaming-provider-readiness-20260604-035317-1780516397705439206.json`,
  `gate_status=passed`, `stepfun_selected`.
- Fresh cloud-edge Xiaozhi host bench:
  `reports/a21-xiaozhi-voice-bench-20260604-035502.222374820.json`,
  `candidate_host_only`, `cloud_edge`, `failure_count=0`,
  answer first-audio p95 `809 ms`, barge-in stop p95 `0 ms`.
- Fresh product readiness:
  `reports/a21-product-readiness-20260604-040153.json`,
  `server_side_blocked`, with provider evidence and host voice loopback ready.
- Fresh server-side readiness bundle:
  `reports/a21-server-side-readiness-bundle-20260604-040207.json`,
  `server_side_blocked`, missing `v21_professional_smoke`.

Current conclusion:

- StepFun route eligibility and cloud-edge host voice evidence are now accepted
  by readiness.
- Full launch remains blocked by `v21_professional_execution` and
  `physical_stackchan_prd_acceptance`.
- Superseded by the 2026-06-04 20:07 CST TUN diagnosis above: default curls
  may return empty replies while TUN is active, but direct-source public A21
  probes with `192.168.1.20` are healthy. Do not reopen this as an ECS runtime
  blocker unless the direct-source check also fails.

## Latest Control-Tower Result - 2026-06-04 04:14 CST

The StepFun cloud-edge server-side slice is accepted at source and ECS runtime,
and the active server-side gap is now V21 professional execution.

Current control facts:

- Current branch is clean and synced to
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash` at
  `20d11a0`.
- A21 thread inspection shows no active worker writer.
- Registered historical worktrees are not being cleaned or pruned.
- Fresh StepFun provider and cloud-edge host voice reports remain the current
  server-side evidence.
- `A21_V21_ADAPTER_URL`, `A21_V21_BACKEND_URL`, and
  `A21_V21_ADAPTER_TOKEN` are missing in the current shell.
- Local `127.0.0.1:21121` is not listening.
- Existing V21 reports are old 2026-06-01/02 evidence and must not be used to
  close the 2026-06-04 V21 gate without a fresh adapter run.
- Fresh V21 dry-run report:
  `reports/a21-v21-adapter-smoke-20260604-041657.json`, `skipped`,
  `configured=false`, `executed=false`, `redaction_ok=true`.
- Fresh local collector reports
  `reports/a21-product-readiness-20260604-041710.json` and
  `reports/a21-server-side-readiness-bundle-20260604-041710.json` are
  control-shell blocker evidence only; they do not supersede the 04:02 ECS
  StepFun/cloud-edge canonical evidence because the local shell has no live
  Gateway/provider selector context.

Active transition:

- `docs/plans/2026-06-04-v21-professional-execution-validation.md`
- `T-V21-PROFESSIONAL-EXECUTION-001`

Current conclusion:

- Server-side readiness is ready for V21 adapter evidence, but V21 is blocked
  on a missing adapter/backend boundary from this control shell.

## Latest Control-Tower Result - 2026-06-04 04:36 CST

The V21 professional execution gate is closed for local adapter-boundary
evidence.

What happened:

- Read V21 control thread
  `codex://threads/019e68bc-4fb6-7ce0-ad67-5b1dd0de478f`.
- Confirmed V21 is currently operated as a local/LAN Docker Compose service.
- Started Docker Desktop and brought up the V21 LAN demo backend from
  `/Users/jiyurun/Documents/v21-knowledge-platform`.
- Verified V21 health:
  - `127.0.0.1:18081/api/v1/healthz`: ok.
  - `192.168.1.20:18081/api/v1/healthz`: ok.
- Verified V21 runtime is configured for retrieval and LLM.
- Verified V21 active collection discovery has an active release.
- Temporarily started A21 adapter bridge:
  `127.0.0.1:21121 -> 127.0.0.1:18081`.
- Ran fresh executed adapter smoke:
  `reports/a21-v21-adapter-smoke-20260604-043456.json`, `passed`,
  `executed=true`, `redaction_ok=true`.
- Ran fresh readiness collectors:
  `reports/a21-product-readiness-20260604-043528.json` and
  `reports/a21-server-side-readiness-bundle-20260604-043528.json`.

Current evidence truth:

- Provider evidence: ready from
  `reports/a21-provider-smoke-20260604-035200-977132343.json`.
- V21 professional evidence: ready from
  `reports/a21-v21-adapter-smoke-20260604-043456.json`.
- Host voice evidence: ready as cloud-edge candidate from
  `reports/a21-xiaozhi-voice-bench-20260604-035502.222374820.json`.
- The local collector remains `server_side_blocked` because this shell has no
  live A21 Gateway, wake-word status, or voice-chain selector context.

Current conclusion:

- V21 is no longer the active server-side evidence blocker.
- This is local adapter-boundary execution evidence, not a permanent ECS
  Gateway V21 topology.
- Full PRD remains blocked by physical StackChan acceptance and current live
  Gateway/wake/selector evidence refresh.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Start

The active product direction has moved from internal test 3 evidence closure to
internal test 4 PRD completion.

Current internal test 4 decisions:

- User-facing modes are now `roleplay` and `professional`.
- `roleplay` is the default embodied mode for personality, memory hints,
  role-play playbooks, voice-clone selection, low-latency speech, and Xiaozhi
  playback.
- `professional` remains the only mode allowed to call the A21/V21 adapter.
- `dialogue`, `workmate`, `companion`, and `co_creation` are compatibility
  aliases or lower-level playbook labels under `roleplay`.
- Mode switching must be user-initiated by voice, touch, app, or web control.
- A21 Cloud/Web/App is the target workspace surface for upload, indexing,
  public-only query, personal-only query, personal+public query, and device
  binding.
- V21 should evolve into a Knowledge Service/Adapter boundary with scoped
  `workspace_id`, `user_id`, and `query_scope` fields; A21 must not read V21
  internals or store uploaded documents on StackChan.

First code/doc cut in progress:

- Plan:
  `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`.
- PRD v0.5 mode/workspace update.
- Protocol mode contract update.
- Gateway `/v1/voice-modes` catalog defaults to `roleplay`, lists
  `roleplay`/`professional`, and accepts old `dialogue` input as a
  compatibility alias.
- `/v1/voice-modes` now carries `selected_ritual` and per-mode `ritual`
  metadata. Selecting `professional` returns `screen_label=PRO`, the shared
  evidence-first checking cue, `expression=professional`, trace marker
  `professional.checking_feedback.sent`, `workspace_policy=professional_only`,
  `v21_allowed=true`, and `physical_accepted=false`.
- Fast companion accepts `roleplay` while still rejecting selected
  `professional` before provider or V21 execution.

Guardrails:

- Do not regress internal test 3 Xiaozhi audio state machine, StepFun/DashScope
  selector evidence, V21 adapter evidence, or product firmware lane.
- Do not claim internal test 4 cloud workspace readiness until upload/index
  scope, adapter v2, and hardware professional consult evidence are separately
  proven.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Roleplay Runtime Slice

The first internal test 4 runtime cut now moves beyond mode naming into a
Gateway-owned roleplay profile contract.

Current implementation state:

- Gateway exposes `GET/POST/PUT /v1/roleplay-profile` with schema
  `a21.gateway.roleplay_profile.v1`.
- The runtime selector carries `roleplay_profile`, scenario/playbook, and
  `voice_clone_profile`.
- Selecting a roleplay voice clone also updates the existing voice-chain
  selector, so the roleplay path reaches the selected TTS/voice-clone boundary
  without adding a second provider control plane.
- Fast companion roleplay turns return a redacted roleplay runtime summary and
  trace `roleplay.profile.ready`; memory readiness adds
  `roleplay.memory.ready`.
- The runtime summary reports prompt composition readiness and memory hint
  counts, but keeps prompt text, memory text, transcripts, provider output,
  voice-clone samples, and V21 evidence out of responses and traces.

Current conclusion:

- `roleplay` is now the concrete default embodied/personality lane for internal
  test 4, with scenario and voice-clone selection exposed to the simulator.
- `professional` remains separate and still cannot execute through
  fast-companion/dialogue endpoints.
- Cloud upload/index/query-scope and hardware professional consult remain
  planned work, not readiness claims.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Roleplay Memory Control

The roleplay runtime now has a product-visible memory control surface without
turning memory into persisted logs or V21 context.

Current implementation state:

- `POST`/`PUT /v1/roleplay-profile` accepts `memory_hints` and
  `clear_memory`.
- Runtime hints are sanitized through the existing personality memory policy:
  unsafe URLs, local paths, and credential-looking strings are rejected; long
  hints are bounded before prompt use.
- Gateway stores only sanitized runtime hint text in memory and returns only
  readiness, counts, and finding codes.
- Fast companion roleplay turns can use the configured hint count while still
  reporting `prompt_stored=false`, `memory_text_stored=false`,
  `professional_route_allowed=false`, and `v21_executed=false`.
- Simulator exposes a one-hint roleplay memory control and clear button, but
  displays only memory status/count.

Current conclusion:

- Roleplay now covers the PRD's first user-visible persona/memory/voice-clone
  control loop.
- This is not durable long-term memory, document upload, personal workspace
  indexing, provider transcript storage, or V21 professional retrieval.

## Latest Control-Tower Result - 2026-06-04 Roleplay Prompt Enters Voice Pipeline

The roleplay lane now carries the selected personality/scenario/memory prompt
into the low-latency text-stream provider boundary instead of stopping at
configuration readiness.

Current implementation state:

- `VoicePipelineRequest` has a redacted runtime-only `TextPrompt` field.
- Voice pipeline text-stream adapters receive `TextPrompt` when present;
  reports continue to record ASR/transcript counts separately and do not store
  prompt text.
- `VoicePipelineReport` exposes only `prompt_input_ready` and
  `prompt_input_not_recorded`.
- Fast-companion roleplay turns with PCM frames compose the selected roleplay
  prompt and pass it into the voice pipeline.
- Stock `/v1/xiaozhi` roleplay turns also compose the selected roleplay prompt
  before the text-stream provider boundary.
- Gateway traces only `roleplay.prompt_input.used`; prompt bodies, memory text,
  transcripts, and provider output remain unrecorded.

Current conclusion:

- Roleplay persona and memory can now affect actual voice answers on the
  low-latency provider path, not only simulator/profile summaries.
- This is still host/runtime evidence. Physical StackChan roleplay acceptance
  requires an operator-triggered stock `/v1/xiaozhi` turn and audible/provider
  evidence.

## Latest Control-Tower Result - 2026-06-04 Roleplay Voice Clone Enters TTS Boundary

The roleplay lane now carries the selected safe voice-clone profile into the
provider-neutral voice pipeline and TTS adapter request instead of stopping at
the selector/catalog layer.

Current implementation state:

- `VoicePipelineRequest` has a redacted runtime-only `VoiceCloneProfile`.
- `TTSAdapterRequest` carries the same safe A21 voice profile ID.
- Local TTS adapters pass the safe profile through `LocalTTSOptions.Voice`, so
  `voice_clone_cli` can bind the turn to the selected voice identity without
  exposing reference audio/text paths.
- `VoicePipelineReport.input.voice_clone_profile` exposes only safe A21
  profile IDs, and report redaction includes
  `voice_clone_sample_not_recorded`.
- Fast-companion roleplay turns with PCM frames and stock `/v1/xiaozhi`
  roleplay turns both pass the selected profile to the voice pipeline.
- Gateway traces only `roleplay.voice_clone_profile.used` when a clone profile
  is selected; it does not trace samples, prompt bodies, memory text,
  transcripts, provider output, local paths, URLs, or credentials.

Current conclusion:

- Roleplay voice selection now has runtime evidence at the TTS boundary:
  selected profile -> voice pipeline request -> TTS request -> redacted report.
- This still does not execute a real clone provider or prove physical
  StackChan roleplay audio. Provider execution and physical acceptance remain
  separate gates.

## Latest Control-Tower Result - 2026-06-04 Professional Mode Ritual Contract

Professional mode selection now has a first-class switching ritual contract
instead of only changing an enum.

Current implementation state:

- `GET/POST /v1/voice-modes` returns `selected_ritual` and per-mode `ritual`
  metadata.
- The professional ritual uses screen label `PRO`, cue
  `我在查，先把证据和置信度拉出来。`, expression `professional`, trace marker
  `professional.checking_feedback.sent`, workspace policy
  `professional_only`, `v21_allowed=true`, and `physical_accepted=false`.
- The simulator displays the selected mode cue through `modeRitualReadout`.
- Fast companion remains blocked when the selected voice mode is
  `professional`; no provider or V21 execution occurs from that endpoint.

Current conclusion:

- Web/app/hardware controls now have a shared redacted ritual payload for
  entering professional mode. Actual professional answers and evidence still
  require the existing professional/V21 path, and physical StackChan
  acceptance remains separate.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Professional Workspace Contract

The next internal test 4 cut gives professional mode an explicit workspace and
query-scope contract without implementing cloud upload/index execution yet.

Current implementation state:

- Gateway exposes `GET/POST/PUT /v1/professional-workspace` with schema
  `a21.gateway.professional_workspace.v1`.
- The selected professional context carries redacted `user_id`,
  `workspace_id`, and `query_scope`.
- Supported contract scopes are `public_only`, `personal_only`, and
  `personal_plus_public`.
- Professional turns now send `device_id`, `user_id`, `workspace_id`, and
  `query_scope` to the A21/V21 adapter contract v2.
- Gateway traces only `professional.workspace.ready` and
  `professional.query_scope.<scope>`; it does not trace user/workspace labels
  or utterance text.
- V21 adapter responses and smoke reports can carry redacted
  `source_scope_counts` and `workspace_status`.

Current conclusion:

- A21 now has the no-execute professional workspace/query-scope API contract
  needed for cloud/app and V21-side workers.
- Upload/import/index job APIs, durable account binding, and personal corpus
  enforcement remain separate unshipped slices.
- Full PRD readiness remains blocked by cloud workspace execution, hardware
  professional consult evidence, and physical StackChan acceptance.

## Latest Control-Tower Result - 2026-06-04 V21 Native Scope Worker Completed

The V21 scoped worker completed the first native A21 v2 voice-query contract
cut in the V21 repository.

Current implementation state:

- V21 worker branch:
  `origin/codex/a21-v2-workspace-query-scope-native-contract`.
- V21 worker commit:
  `ad61246 feat(voice-query): accept A21 workspace scope metadata`.
- V21 `/internal/v1/knowledge/voice-query` now natively accepts
  `device_id`, `user_id`, `workspace_id`, and `query_scope`.
- Valid V21 `query_scope` values match A21 internal test 4:
  `public_only`, `personal_only`, and `personal_plus_public`.
- V21 responses now include safe metadata fields `source_scope_counts` and
  `workspace_status`.
- Because current V21 schema cannot yet prove personal/public ACL enforcement,
  the worker truthfully returns `workspace_status=scope_contract_ready_acl_pending`
  and zero classified `source_scope_counts` instead of claiming personal corpus
  search is enforced.

Current conclusion:

- The A21/V21 adapter contract is now aligned on request/response shape at the
  V21 worker branch level.
- V21 merge/release and an ACL/schema ADR remain required before A21 can claim
  personal/public corpus enforcement or nonzero personal/public scope counts.
- The worker did not touch A21 code, A21 Gateway runtime, StackChan hardware,
  firmware, provider APIs, or the dirty V21 LAN/desktop connector work.

## Latest Control-Tower Result - 2026-06-04 V21 Scope Retrieval Guard Completed

The next V21 scoped worker moved the native A21 v2 bridge from metadata-only
acceptance to executable source-scope filtering.

Current implementation state:

- V21 worker branch:
  `origin/codex/a21-v2-workspace-scope-retrieval-guard`.
- V21 worker commit:
  `fccd0ac feat(voice-query): enforce A21 workspace source scope`.
- V21 retrieval requests now carry safe A21 v2 metadata:
  `device_id`, `user_id`, `workspace_id`, and `query_scope`.
- V21 retrieval results and voice-query evidence now carry safe
  `source_scope=public|personal` labels.
- `public_only`, `personal_only`, and `personal_plus_public` are filtered
  before answer generation; unclassified evidence is not promoted into scoped
  A21 responses.
- V21 HTTP retrieval adapter and Qdrant sidecar both pass through
  `source_scope`; the Postgres retrieval writer can derive scope from chunk
  metadata, source-unit locator metadata, or `object_refs.access_class`.
- Scoped responses can report `workspace_status=searchable` with classified
  `source_scope_counts`; legacy requests without `query_scope` still keep
  `scope_contract_ready_acl_pending` and zero claimed scope counts.

Verification:

- V21 `go test ./...` from `backend/`: passed.
- V21 sidecar `python3 -m unittest retrieval_service_test.py`: passed.
- V21 retrieval eval
  `python3 -m unittest evals/retrieval/test_qdrant_retrieval_sidecar.py`:
  passed.
- V21 `git diff --check`: passed.
- V21 `make compose-config`: passed.
- V21 `make test` and `make check`: blocked at `check-toolchain` because the
  Makefile expects Node `v24.15.0` and this shell has Node `v25.8.0`.

Current conclusion:

- A21 can now treat the V21 worker branch as source-scope guard evidence, not
  merely shape evidence.
- This still does not complete durable tenant/account ACL, real personal
  upload indexing, cloud storage, V21 merge/release, or physical StackChan
  professional consult acceptance.
- The worker did not start V21/A21 services, execute providers, ingest private
  documents, touch ECS, firmware, serial, NVS, or the dirty V21 LAN/desktop
  connector worktree.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Workspace Job Skeleton

The workspace surface now has a no-execute upload/import/index job contract.

Current implementation state:

- Gateway exposes `GET/POST/PUT /v1/workspace-upload-jobs` with schema
  `a21.gateway.workspace_upload_jobs.v1`.
- `POST` creates a redacted metadata job for `upload` or `import` source kinds.
- `GET` polls all jobs or a single `job_id`.
- `PUT` supports `mark_failed`, `retry`, and `delete`.
- Jobs store only redacted labels, source scope, content type, size, status,
  attempt count, and trace/session/device IDs.
- Jobs explicitly report `accepted_no_execute`,
  `not_started_no_execute`, and redaction flags; no document text, bytes,
  base64 payload, import URL, local path, credential, or provider output is
  accepted.
- Simulator exposes a `Workspace Job` button that creates a no-execute metadata
  job from the current professional query-scope context.

Current conclusion:

- The A21 Cloud/Web/App workspace API skeleton is now present for upload/import
  job lifecycle UX and worker integration.
- Real upload storage, indexing execution, delete propagation, source ACLs,
  durable user/device binding, and V21 personal/public corpus enforcement
  remain separate unshipped slices.

## Latest Control-Tower Result - 2026-06-04 Professional Workspace Read Records

Professional evidence turns now leave a safe read ledger for upload/read-record
discipline without storing query text or retrieved content.

Current implementation state:

- Gateway exposes `GET /v1/professional-read-records` with schema
  `a21.gateway.professional_read_records.v1`.
- Mock professional turns and stock Xiaozhi professional turns start a
  memory-only read record before V21 query and complete or fail it with safe
  status/scope/count/timing metadata.
- Records store trace/session/device IDs, redacted user/workspace labels,
  query scope, privacy scope, latency profile, answer style, utterance length
  bucket, source-scope counts, workspace status, and redaction flags only.
- Traces add `professional.read_record.started`,
  `professional.read_record.completed`, and
  `professional.read_record.failed`.

Current conclusion:

- Internal test 4 now has an auditable professional-read metadata surface.
- The ledger is memory-only; durable audit storage and real cloud document
  indexing remain separate work.

## Latest Control-Tower Result - 2026-06-04 Simulator Workspace Audit Surface

The simulator now exposes the workspace/read-record metadata operators need for
internal test 4 checks.

Current implementation state:

- The simulator has a `Workspace Audit` section with a no-execute workspace job
  action and a `Read Records` refresh action.
- The surface shows last workspace job status plus read-record count, status,
  failure code, query scope, utterance bucket, source-scope counts, workspace
  status, and privacy scope.
- Professional evidence responses refresh read records for the current trace.
- The simulator displays metadata only, not raw queries, retrieved text,
  evidence bodies, provider output, document text, URLs, paths, credentials,
  voice transcripts, or audio.

Current conclusion:

- The internal-test operator surface can inspect upload/read discipline without
  leaving the safe metadata boundary.
- Browser screenshot tooling was unavailable in that round; local HTTP smoke
  covered the served page and endpoint presence.

## Latest Control-Tower Result - 2026-06-04 Professional Voice Trigger Route

Professional mode can now be entered from speech by explicit user trigger
phrases instead of only API/app selection.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-professional-voice-trigger-route.md`.
- `/v1/mock-turn` routes default/roleplay/workmate/companion trigger text such
  as "认真查一下" into `professionalTurnResponse`.
- Stock `/v1/xiaozhi` default `realtime`/workmate turns route to
  professional when the streaming ASR final contains an explicit trigger such
  as "给我证据".
- Xiaozhi triggered professional turns reuse the streaming ASR final and avoid
  a duplicate batch ASR pass.
- Negated phrases such as "不要进专业检索", "不用专业模式", and
  "别查 V21" do not trigger V21.
- Trace markers are redacted:
  `professional.voice_trigger.detected` and
  `xiaozhi.professional_route.voice_trigger`.

Current conclusion:

- The PRD's user-initiated mode switch now has a Gateway route on both the mock
  control path and the stock Xiaozhi socket.
- This is host-local Gateway evidence only. It does not execute real provider
  or real V21, start Gateway as a runtime service, build/flash firmware, write
  serial/NVS, or prove physical StackChan professional consult acceptance.

## Latest Control-Tower Result - 2026-06-04 Workspace Source Readiness Registry

The workspace surface now has a memory-only source/readiness registry layered
on top of the no-execute upload/import job contract.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-source-readiness-registry.md`.
- Gateway exposes `GET /v1/workspace-sources` with schema
  `a21.gateway.workspace_sources.v1`.
- Creating a workspace upload/import job now creates a linked redacted
  `source_id` with `readiness=metadata_only` and
  `index_status=not_started_no_execute`.
- `PUT /v1/workspace-upload-jobs` accepts
  `mark_searchable` / `mark_indexed_metadata_only` and marks the source as
  `searchable_metadata_only` without real indexing.
- `mark_failed`, `retry`, and `delete` keep source readiness in sync; deletion
  leaves only a redacted tombstone.
- `/v1/professional-workspace` runtime now includes source-scope counts,
  searchable source-scope counts, and query-scope readiness while keeping
  `v21_execution_allowed=false`.
- The simulator Workspace Audit surface can refresh source readiness and show
  source count plus readiness summary.
- Trace markers remain metadata-only:
  `workspace.source.created_metadata_only` and
  `workspace.index_job.searchable_metadata_only`.

Current conclusion:

- Internal test 4 can now show "uploaded/imported source exists" and
  "searchable metadata candidate" per public/personal scope, which is closer to
  the cloud workspace PRD shape.
- This still does not store documents, run indexing, enforce V21 ACLs, execute
  providers/V21, start Gateway, deploy ECS, or prove physical hardware
  professional consult.

## Latest Control-Tower Result - 2026-06-04 Workspace Document Upload Intake

The workspace surface now accepts real local document upload bytes without
promoting indexing or V21 execution.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-document-upload-intake.md`.
- Gateway exposes `POST /v1/workspace-documents` with schema
  `a21.gateway.workspace_documents.v1`.
- The endpoint accepts `multipart/form-data` only, stores bytes in an A21
  Gateway runtime directory, computes a safe `sha256:` document hash, and links
  the document to a workspace job/source pair.
- Linked jobs and sources report `stored_local_pending_index`,
  `storage_status=stored_local`, and `index_status=not_started_no_execute`.
- The local store defaults to `.a21-run/gateway/workspace-documents` and can
  be overridden by `A21_WORKSPACE_DOCUMENT_STORE_DIR`; a conservative per-file
  limit can be overridden by `A21_WORKSPACE_DOCUMENT_MAX_BYTES`.
- `/v1/professional-workspace` now shows `stored_local_pending_index` readiness
  when the selected scope has local stored uploads but no searchable source.
- Simulator Workspace Audit now includes a file picker and upload control, then
  displays only document/job/source metadata.
- Existing `/v1/workspace-upload-jobs` direct creation remains metadata-only
  and still rejects raw document text, bytes, base64 payloads, import URLs,
  local paths, credentials, and provider output.

Current conclusion:

- Internal test 4 has a real local upload intake layer: user file bytes can
  enter A21-controlled local storage and become visible as a safe pending-index
  source.
- This still does not parse, chunk, embed, index, upload to cloud storage,
  enforce durable auth/ACLs, execute provider/V21, start Gateway as a service,
  deploy ECS, or prove physical StackChan professional consult acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Soul Profile Contract

The roleplay profile selector now controls a real A21-owned role-soul layer
instead of a single default placeholder.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-soul-profile-contract.md`.
- Personality composition now supports an optional `role_souls/*.md` layer
  between `tone_rules` and `mode_prompts`.
- A21 roleplay profiles now include `a21_roleplay_default`,
  `a21_roleplay_wry_peer`, and `a21_roleplay_calm_anchor`.
- `/v1/roleplay-profile` accepts the selected `roleplay_profile` together with
  scenario, voice-clone profile, and bounded memory hints.
- Runtime summaries expose safe prompt-part identifiers such as
  `role_soul:a21_roleplay_wry_peer`, `soul_prompt_input_ready`, memory counts,
  and redaction flags, but not prompt bodies or memory text.
- Fast-companion and stock Xiaozhi roleplay prompt composition use the selected
  role soul before passing prompt input to the provider-neutral voice pipeline.
- Simulator controls now include a role soul selector and readout alongside
  scenario, voice clone, and memory controls.

Current conclusion:

- Internal test 4 roleplay now has a configurable persona/soul surface that can
  affect actual voice-pipeline prompt input, not only metadata.
- This is host-local contract evidence only. It does not execute provider/V21,
  start Gateway as a runtime service, deploy ECS, build/flash firmware, write
  serial/NVS, or prove physical StackChan roleplay audio acceptance.

## Latest Control-Tower Result - 2026-06-04 MCP Speaker Volume Freeze

The already-used official speaker volume MCP tool is now part of the unified
low-risk MCP control surface.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-stackchan-official-mcp-speaker-volume-freeze.md`.
- `/v1/xiaozhi/mcp-control` now allows `self.audio_speaker.set_volume` with
  bounded `volume=0..100`.
- The existing `/v1/xiaozhi/speaker-volume` compatibility/product shortcut is
  preserved.
- Unified MCP control still requires valid A21 `device_id`, an online Xiaozhi
  socket, `hello.features.mcp=true`, and trace/session/device identity.
- The endpoint records only the redacted send marker
  `xiaozhi.mcp.speaker_volume.sent` and safe device activity metadata such as
  bounded `speaker_volume`.
- Missing, out-of-range, or mixed screen/volume arguments are rejected before
  websocket write.

Current conclusion:

- Low-risk MCP parity now has one consistent Gateway control surface for
  speaker volume, device status, brightness, theme, and screen info.
- This is delivery-contract evidence only. Physical loudness/playback
  acceptance still requires fresh device evidence; no provider/V21, Gateway
  runtime service start, ECS, firmware, serial, NVS, or hardware action
  occurred.

## Latest Control-Tower Result - 2026-06-04 Workspace Index Request Ledger

Internal test 4 workspace upload readiness now has a no-execute indexing
request ledger.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-index-request-ledger.md`.
- Gateway now exposes `GET/POST /v1/workspace-index-jobs` with schema
  `a21.gateway.workspace_index_jobs.v1`.
- `POST /v1/workspace-index-jobs` accepts only safe document/job/source IDs
  and optional trace/session/device IDs. It verifies the stored local file is
  present, but does not read, parse, chunk, embed, OCR, upload, or send it to
  V21.
- Linked document, upload job, and source readiness now promote to
  `indexing_requested_no_execute`; the source summary exposes
  `indexing_requested_source_scope_counts`, and professional workspace
  readiness can report `indexing_api_ready=true` after a request is recorded.
- The simulator Workspace Audit surface now has an `Index Request` control and
  index-job readout using only safe IDs/status metadata.

Current conclusion:

- Local uploads now have a truthful adapter-ready pre-index state instead of a
  dead-end pending flag.
- This is not searchable readiness, real indexing, parsing, embedding, cloud
  storage, provider/V21 execution, durable auth/ACL, Gateway service startup,
  ECS deployment, firmware, serial, NVS, or physical StackChan professional
  consult acceptance.

## Latest Control-Tower Result - 2026-06-04 A21 Native V21 Voice-Query Bridge

The local A21 `v21-adapter-bridge` now consumes V21's native A21 v2
`/internal/v1/knowledge/voice-query` contract as its primary professional
query path.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-a21-v21-native-voice-query-bridge.md`.
- Bridge requests now pass safe A21 v2 fields into V21 native voice-query:
  `device_id`, `user_id`, `workspace_id`, `query_scope`, trace/session IDs,
  and the active collection ID.
- Native voice-query success mirrors V21-returned `source_scope_counts` and
  `workspace_status`; A21 no longer fabricates counts from `query_scope` on the
  green path.
- Direct retrieval remains only as a controlled no-evidence expansion fallback,
  and fallback counts are derived only from V21 result `source_scope` labels.
- Focused httptest coverage proves the bridge calls native voice-query on the
  green path, avoids direct retrieval there, preserves explicit
  `personal_plus_public` workspace scope metadata, and still retries the
  child-lock ASR fragment through the legacy retrieval fallback after a
  native `no_evidence` response.

Current conclusion:

- A21 is now wired to the V21 source-scope guard worker contract instead of
  silently bypassing it through direct retrieval for normal professional
  queries.
- This is adapter-boundary contract evidence only. It is not V21 merge/release,
  real personal upload indexing, durable tenant/account ACL, cloud storage,
  provider execution, Gateway service startup, ECS deployment, firmware,
  serial, NVS, or physical StackChan professional consult acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Device State Reflection

The selected roleplay identity is now visible as safe device/registry state,
not only as hidden prompt input.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-device-state-reflection.md`.
- `/v1/devices` now reflects safe roleplay state:
  `current_roleplay_profile`, `current_roleplay_scenario`,
  `roleplay_soul_ready`, `roleplay_memory_ready`,
  `roleplay_memory_hint_count`, and `roleplay_physical_accepted=false`.
- Device `runtime_echo` now receives only safe roleplay IDs, booleans, counts,
  and explicit non-storage flags for prompt text, memory text, and voice
  samples.
- The simulator Device Registry panel now shows role soul, scenario, and role
  memory alongside the existing voice-clone profile.
- Focused tests prove selected `a21_roleplay_wry_peer` +
  `engineer_pushback` + one safe memory hint reaches the device registry while
  unsafe URLs, memory text, and control text stay out of `/v1/devices`.

Current conclusion:

- Roleplay is now better reflected as an embodied runtime state: the device
  registry can tell which persona/scenario/memory posture is active without
  exposing the role prompt or memory content.
- This is host-local registry/simulator evidence only. It does not execute
  providers, V21, or voice-clone CLI; it does not start Gateway as a service,
  deploy ECS, build/flash firmware, write serial/NVS, or prove physical
  StackChan roleplay audio/visual acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Official Expression Plan

Roleplay identity now has a safe host-side expression plan instead of stopping
at hidden prompt/device-state metadata.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-official-expression-plan.md`.
- `/v1/roleplay-profile` now returns `expression_plan` with schema
  `a21.roleplay_expression_plan.v1`.
- The plan maps baseline posture, selected role soul, scenario emphasis, and
  memory cue phases to existing official StackChan action-plan metadata.
- The simulator shows the expression delivery policy plus action and packet
  counts.
- The plan records only safe profile/scenario/voice-clone IDs, memory
  readiness/count, event kinds, event values, packet counts, official semantic
  surfaces, and redaction flags.

Current conclusion:

- Roleplay is one step closer to embodied expression because the Gateway can
  now describe which official StackChan semantics should represent the selected
  role state.
- This is still a no-send contract with
  `delivery_policy=no_send_plan_only` and `physical_accepted=false`. It does
  not execute providers, V21, voice-clone CLI, Gateway runtime delivery,
  `/stackChan/ws`, firmware, serial, NVS, ECS, or physical hardware action.

## Latest Control-Tower Result - 2026-06-04 Workspace Console Product Surface

Internal test 4 workspace progress is now visible as a user-facing web console,
not only as simulator/debug controls.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-console-product-surface.md`.
- Gateway serves `GET /workspace` with a compact app shell for workspace setup,
  upload/index, readiness, and mode-boundary state.
- The console calls existing safe APIs only:
  `/v1/professional-workspace`, `/v1/workspace-documents`,
  `/v1/workspace-index-jobs`, `/v1/workspace-sources`,
  `/v1/professional-read-records`, `/v1/roleplay-profile`, and
  `/v1/voice-modes`.
- Playwright rendered the page at desktop `1270x900` and mobile `390x900`
  with no overflow findings.
- A dummy local upload through the page produced `stored_local`, and the page's
  index request produced `indexing_requested_no_execute`, source count `1`,
  and `searchable=false`.

Current conclusion:

- This is visible product movement for the cloud/web/app workspace shape:
  users can now operate the safe upload/index/readiness contract from a web
  surface.
- This is not real parsing, embedding, indexing, cloud storage, auth/device
  binding, provider execution, V21 execution, Gateway deployment, ECS,
  firmware, serial, NVS, or physical StackChan professional consult acceptance.

## Latest Control-Tower Result - 2026-06-04 Workspace Voice Probe Surface

Internal test 4 workspace progress now includes a safe dialogue-path probe,
not only configuration controls.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-workspace-voice-probe-control-surface.md`.
- Gateway serves `GET /workspace` with a Voice Probe panel over existing safe
  routes only:
  `/v1/fast-companion/turn`, `/v1/mock-turn`, `/v1/traces`, and
  `/v1/professional-read-records`.
- Playwright selected `a21_roleplay_wry_peer`, `engineer_pushback`, and
  `a21_voice_clone_default`, saved one bounded memory hint, and observed the
  roleplay probe return `fast_companion_hybrid`, trace event count `14`,
  `prompt=true`, `voice=a21_voice_clone_default`, and `memory=ready / 1`.
- Playwright ran the professional probe and observed
  `professional_mock_turn`, trace event count `13`, and one
  `a21-professional-read-*` record with `status=completed`,
  `query_scope=public_only`, and `workspace_status=searchable` in the
  host-local default mock path.
- Desktop and mobile render checks had no horizontal overflow.

Follow-on product-surface update:

- The professional probe path has now moved from `/v1/mock-turn` to the formal
  Web/App consult endpoint `POST /v1/professional-query`. The old observations
  above remain historical evidence for the mock probe cut; current workspace
  console behavior should be read from the professional-query endpoint and its
  read-record trace.

Current conclusion:

- This closes the product-console gap between frontend roleplay/voice/memory
  setup and observable safe dialogue metadata.
- This is not provider execution, real V21 execution, audible voice-clone
  playback, real document indexing, Gateway deployment, ECS, firmware, serial,
  NVS, or physical StackChan professional/roleplay acceptance.

## Latest Control-Tower Result - 2026-06-04 Selected Voice-Chain Readiness Ingress

Product/server-side readiness now has a safe selected voice-chain capability
evidence ingress instead of only the Gateway selector readout.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-selected-voice-chain-readiness-ingress.md`.
- `a21 product-readiness` accepts
  `--voice-chain-readiness-report <report.json>`.
- `a21 product-readiness --use-latest-reports` can discover the newest
  `a21-xiaozhi-streaming-provider-readiness-*.json` report.
- `a21 server-side-readiness-bundle` passes the same evidence through to the
  underlying product-readiness report.
- The report must match the current Gateway-selected ASR, LLM, and TTS profile
  IDs before `static_capability_ready` becomes true.
- Mismatched reports remain visible as
  `voice_chain_capability_report_mismatch` and are not absorbed as current-chain
  readiness.

Current conclusion:

- This closes the bookkeeping gap between the existing static no-execute
  provider-chain classifier and product/server-side readiness.
- This is not provider execution, real V21 execution, audible playback, Gateway
  deployment, ECS runtime acceptance, firmware, serial, NVS, or physical
  StackChan PRD acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Immersion Readiness

Product readiness now has an explicit roleplay immersion surface instead of
leaving persona/memory/voice-clone readiness only in Gateway UI/probe metadata.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-immersion-product-readiness.md`.
- `a21 product-readiness` fetches `GET /v1/roleplay-profile` when the Gateway
  exposes it.
- The product report now includes a top-level `roleplay` object with selected
  role soul, scenario, voice-clone profile, soul prompt readiness, prompt
  composed status, memory configured/readiness/count, expression-plan
  availability, action/packet counts, redaction flags, and physical acceptance
  truth.
- Invalid or unsafe roleplay profile responses become
  `roleplay_profile_invalid`; unavailable endpoints stay `status=unavailable`
  so older Gateway/test surfaces are not overinterpreted.
- `roleplay.physical_accepted` remains false until separate physical evidence
  exists.

Current conclusion:

- This makes roleplay immersion measurable in the launch readiness report:
  the PRD reviewer can see whether role soul, memory, voice clone, prompt, and
  expression planning are ready without inspecting runtime traces by hand.
- This is not provider execution, real voice-clone audio, V21 execution,
  Gateway deployment, ECS runtime acceptance, firmware, serial, NVS, or
  physical StackChan roleplay acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Voice Runtime Evidence

Product readiness now has a safe ingress for roleplay voice runtime evidence
instead of only static roleplay profile readiness.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-voice-runtime-evidence-ingress.md`.
- `a21 product-readiness` accepts
  `--roleplay-voice-report <report.json>`.
- `a21 product-readiness --use-latest-reports` can discover the newest safe
  `a21-roleplay-voice-probe-*.json` report.
- `a21 server-side-readiness-bundle` passes the same report path through to the
  underlying product-readiness report.
- The product report's top-level `roleplay` object now shows whether runtime
  evidence is available, matched to the current Gateway roleplay profile, and
  ready.
- Accepted evidence exposes only safe route/status/execution mode, source
  basename, marker count, and booleans for text-stream execution, prompt input
  use, voice-clone profile use, audio downlink, and playback-start observation.
- Unsafe roleplay voice reports become `roleplay_voice_report_invalid`; matched
  but incomplete reports become `roleplay_voice_runtime_not_ready`.

Current conclusion:

- This closes the reporting gap between workspace Voice Probe / fast-companion
  roleplay runtime behavior and product readiness: the reviewer can now see
  whether role soul, memory, and voice-clone selection reached the roleplay
  voice path.
- This is not a real provider execution, voice-clone audio quality acceptance,
  real V21 execution, Gateway deployment, ECS runtime acceptance, firmware,
  serial, NVS, or physical StackChan roleplay acceptance.

## Latest Control-Tower Result - 2026-06-04 Professional Ritual Gate

Server-side readiness now separates V21 adapter evidence from the user-facing
professional ritual/execution evidence.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-server-side-professional-ritual-execution-gate.md`.
- `a21 product-readiness` now reports
  `server_side.professional_ritual_ready` and
  `server_side.professional_ritual_source_report`.
- Adapter smoke still closes `v21_professional_evidence_ready`, but only a
  safe `a21.xiaozhi_professional_bench.v1` report with
  `acceptance_status=external_gateway_ready` closes
  `professional_ritual_ready`.
- `a21 product-readiness --use-latest-reports` prefers an accepted
  `a21-xiaozhi-professional-bench-*.json` report over newer adapter smoke so a
  fresh adapter smoke cannot overwrite stronger professional ritual evidence.
- `a21 server-side-readiness-bundle` now exposes a `professional_ritual`
  evidence block and can collect missing ritual evidence with
  `a21 xiaozhi-professional-bench --gateway-url <gateway> --output-dir reports`
  only when `--execute-v21-smoke` is explicitly supplied.

Current conclusion:

- Provider smoke, V21 adapter smoke, host voice, roleplay voice, wake-word, and
  voice-chain evidence are no longer enough for a server-side candidate unless
  the professional checking cue/result-order/stale-suppression/abort-stop path
  is also proven.
- This is still below physical StackChan PRD acceptance and does not flash
  firmware, touch serial/NVS, persist provider keys in firmware, deploy ECS, or
  prove audible/visible hardware professional consult behavior.

## Latest Control-Tower Result - 2026-06-04 Professional Read-Record Gate

Professional readiness now requires the execution report to prove the Gateway
read ledger, not only the V21 answer.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-professional-read-record-readiness-gate.md`.
- `a21 xiaozhi-professional-bench` now fetches
  `/v1/professional-read-records?trace_id=<bench-trace>` and writes a safe
  `read_record` summary into `a21.xiaozhi_professional_bench.v1` reports.
- Product readiness rejects an external Gateway professional bench report if
  the read record is missing, incomplete, mismatched, not `professional_only`,
  missing legal source-scope counts/workspace status, or redaction is unsafe.
- `a21 product-readiness` now reports
  `server_side.professional_read_record_ready` and
  `server_side.professional_read_record_source_report`.
- `a21 server-side-readiness-bundle` exposes `professional_read_record` as a
  separate evidence block. If the professional ritual is ready but the ledger
  is not, the missing evidence is `professional_read_record`.

Current conclusion:

- The internal test 4 professional path now has launch-gate evidence for
  both execution and read-ledger discipline: same trace, completed record,
  scope/status/counts, and redaction.
- This is still not real personal upload indexing, cloud storage, durable
  account ACL, ECS deployment, firmware, serial, NVS, or physical StackChan
  professional consult acceptance.

## Latest Control-Tower Result - 2026-06-04 Roleplay Voice Runtime Probe Closure

The live local Gateway roleplay voice probe now passes instead of remaining a
blocked fixture boundary.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-roleplay-voice-runtime-probe-closure.md`.
- Fast Companion roleplay voice-pipeline traces now record
  `roleplay.voice_clone_profile.used` whenever a safe selected voice profile is
  carried into the provider-neutral voice pipeline, including the default
  `a21_voice_default_dashscope` profile.
- The same voice-pipeline branch now records `device.playback.start` when the
  Gateway emits the first host/simulator playback chunk.
- Product readiness now accepts safe single-token roleplay prompt parts such as
  `core_identity` and `tone_rules`, in addition to `key:value` prompt-part
  markers, while still rejecting unsafe text/URL/path/credential-like values.
- Runtime evidence:
  - `a21 roleplay-voice-probe --require-ready` passed and wrote
    `a21-roleplay-voice-probe-20260604-150552.json`.
  - `a21 server-side-readiness-bundle --collect-missing
    --execute-provider-smoke --execute-v21-smoke` wrote
    `a21-server-side-readiness-bundle-20260604-150611.json`.

Current conclusion:

- Local no-hardware server-side readiness now has Gateway, professional
  ritual, professional read-record, host voice, roleplay voice runtime,
  wake-word, and voice-chain selector ready.
- The remaining server-side blocker is real provider smoke. The remaining full
  PRD blockers are physical StackChan online and physical PRD acceptance.
- This does not prove real clone-audio quality, real provider quality, ECS
  deployment, firmware, serial, NVS, or physical StackChan roleplay acceptance.

## Latest Control-Tower Result - 2026-06-04 StepFun Provider Smoke Closure

The no-hardware server-side candidate gate is now closed on the local runtime.

Current implementation state:

- The active cut is
  `docs/plans/2026-06-04-stepfun-provider-smoke-server-candidate-closure.md`.
- `.a21-run/provider.env` was sourced without printing secret values.
- `A21_PROVIDER_PRIMARY=stepfun`, `A21_TEXT_STREAM_PROFILE=stepfun`, and
  `A21_STEPFUN_MODEL=step-1-8k` were used for the launch-policy text provider.
- Dry-run provider smoke first reported `configured=true`.
- Executed streaming provider smoke then passed:
  `reports/provider-live/a21-provider-smoke-20260604-153030-291957000.json`.
- Local Gateway was started with the same StepFun launch-policy env.
- `a21 server-side-readiness-bundle --provider-smoke-report <stepfun-report>
  --use-latest-reports --require-candidate` passed and wrote
  `reports/a21-server-side-readiness-bundle-20260604-153129.json`.

Current conclusion:

- `server_side.candidate_ready=true` and
  `status=server_side_candidate_ready`.
- The server-side candidate now has Gateway, executed provider smoke,
  professional ritual, professional read-record, host voice, roleplay voice
  runtime, wake-word, and selected voice-chain evidence.
- Full PRD launch remains blocked only by physical StackChan online evidence
  and physical PRD acceptance.
- No provider key was printed, committed, stored in firmware, or put into a
  report body. No ECS/root-secret change, firmware build, flash, serial, NVS,
  or physical hardware action occurred.

## Latest Control-Tower Result - 2026-06-04 Internal Test 4 Hardware Window

Current implementation state:

- Mainline branch `codex/a21-internal-test4-mainline-20260604` was created
  from HEAD `87579cf` and pushed. The active foreground hardware branch is
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
- Local Gateway was run on `0.0.0.0:21080` with public LAN URL
  `http://10.98.141.239:21080`; `/healthz`, `/xiaozhi/ota/`, and local
  stock-WebSocket OTA discovery responded.
- Product StackChan serial `/dev/cu.usbmodem1101` was reachable. A guarded
  official-compatible product NVS write passed on the hardware-window branch
  and changed only `wifi/ota_url`, `websocket/url`, and `websocket/version`.
  Servo calibration and existing Wi-Fi credentials were preserved.
- After reset, the device repeatedly reported `WifiStation: No AP found`. The
  control Mac is on LAN at `10.98.141.239` but not associated with a Wi-Fi AP,
  so the physical blocker is now network provisioning, not Gateway URL/NVS
  product-lane routing.
- The official-compatible NVS writer now supports explicit
  `A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WIFI_SSID` and
  `A21_STACKCHAN_OFFICIAL_XIAOZHI_COMPATIBLE_WIFI_PASSWORD` values for this
  foreground case. It writes only requested Wi-Fi keys plus Xiaozhi connection
  keys and redacts values from reports/stdout.
- `a21 xiaozhi-physical-prd-review` and `make xiaozhi-physical-prd-review`
  now provide the missing report-only promote boundary from matching
  `physical_review_required` Xiaozhi physical and half-duplex reports to an
  accepted `a21.xiaozhi_physical_evidence.v1` report.

Current conclusion:

- Server-side candidate remains closed locally; full PRD is still waiting on
  the device joining a reachable Wi-Fi network, then fresh stock Xiaozhi
  physical evidence, stock half-duplex evidence, and the explicit PRD review
  promote command.
- Public ECS direct curl from this Mac still returned empty replies during this
  window, and SSH was closed from this machine; local LAN Gateway was used for
  guarded hardware work instead.
- No firmware app flash, provider key exposure, generic `xiaozhi.bin` product
  flash, repository prune/gc, or rollback of internal test 3 protocol/audio
  changes occurred.

## Latest Control-Tower Result - 2026-06-04 Wi-Fi Credential NVS Attempt

Current implementation state:

- The operator provided a target AP SSID/password out of band. The password was
  used only as stdin for the guarded NVS writer and was not committed or copied
  into documentation.
- The official-compatible NVS plan and execute reports were generated:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-162056-1780561256235353000.json`
  and
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-162103-1780561263934023000.json`.
- The execution report passed with `wifi_credentials_written=true`,
  `mutated_entry_count=5`, `servo_calibration_present=true`, and a clean
  hardware-window control guard on commit `a216fe6fcaa1`.
- Local Gateway was restarted on `0.0.0.0:21080` with public LAN URL
  `http://10.98.141.239:21080`; health and `/xiaozhi/ota/` returned the
  expected stock WebSocket URL.
- After hard reset, the device attempted the provided SSID but serial logs
  showed Wi-Fi disconnect reason `201`, exhausted 5 attempts, then entered
  Xiaozhi hotspot provisioning as `Xiaozhi-6A61`.
- Gateway `/v1/devices` stayed empty during the polling window.

Current conclusion:

- The NVS write path is now proven for explicit Wi-Fi credentials, but the
  provided AP was not visible/reachable to the device in this physical location
  or band. The next hardware action needs a 2.4 GHz AP the ESP32-S3 can see, or
  the operator should use the `Xiaozhi-6A61` captive portal to provision a
  reachable network.
- No firmware app flash, generic Xiaozhi product flash, provider execution,
  V21 execution, repository prune/gc, or internal-test3 rollback occurred.

## Latest Control-Tower Result - 2026-06-04 Cloud Gateway NVS Correction

Current implementation state:

- The operator clarified that the product device should use the cloud Gateway,
  not the temporary local LAN Gateway.
- The local LAN Gateway was stopped before any second NVS write. Port `21080`
  was released.
- The product-lane NVS writer then wrote the operator-provided phone hotspot
  credentials together with the canonical cloud Gateway endpoints:
  `http://47.103.57.217/xiaozhi/ota/` and
  `ws://47.103.57.217/v1/xiaozhi`.
- NVS plan:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-163203-1780561923834996000.json`.
- NVS execute:
  `reports/a21-stackchan-official-xiaozhi-compatible-nvs-20260604-163211-1780561931273006000.json`.
- The execute report passed with `wifi_credentials_written=true`,
  `mutated_entry_count=5`, `servo_calibration_present=true`, and a clean
  hardware-window control guard at commit `bf20e56522aa`.
- The device was hard-reset after the cloud NVS write. Serial output observed
  regular `SystemInfo` lines and did not repeat the earlier `No AP found` or
  hotspot-provisioning fallback in the observed window.
- Superseded by the 2026-06-04 20:07 CST TUN diagnosis above: default public
  probes were routed through TUN and returned false empty replies. Direct-source
  probes with `192.168.1.20` return healthy A21 Gateway responses. SSH now
  reaches the real auth state and fails with `Permission denied (publickey)`.

Current conclusion:

- The product NVS is now aligned with the all-cloud architecture, and the later
  corrected SSID NVS write restored physical reconnect. The remaining cloud
  blocker is ECS SSH authorization for deployment/restart, not Gateway HTTP/WS
  health.
- Device registration can be confirmed from this control Mac only through the
  direct-source path while TUN is active.
- No provider key exposure, firmware app flash, generic Xiaozhi product flash,
  V21 execution, repository prune/gc, or internal-test3 rollback occurred.

## Latest Control-Tower Result - 2026-06-04 State Body Reactions

Current implementation state:

- Current branch:
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs`.
- After the accepted touch body-reaction proof, the next visible expression
  slice is implemented behind
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true`.
- The new gate returns `a21.state_reactions=true` only for hardware-MAC stock
  Xiaozhi clients that advertise stock `hello.features.mcp=true`.
- Gateway-generated `idle`, `listening`, `thinking`, `speaking`, `error`, and
  `fatal_error` states now map to bounded official MCP
  `self.robot.set_led_color` and `self.robot.set_head_angles` calls on the
  live product `/v1/xiaozhi` socket.
- This does not depend on `/stackChan/ws`, does not broaden accepted
  firmware-originated `type=device` events, and does not touch firmware,
  flash, NVS, provider execution, or V21.
- Follow-up reliability guard: when the Xiaozhi websocket closes, Gateway now
  preserves prior semantic `last_event` evidence but reports
  `connection_status=xiaozhi_ws_disconnected`, so MCP/body-control probes do
  not trust stale registry rows as writable sockets.
- Commits `b6c12f0` and `f2663f1` are pushed on
  `codex/a21-hardware-window-20260604-internal-test4-local-lan-nvs` and
  deployed to ECS `47.103.57.217`.
- ECS `/etc/a21/runtime.env` has root-only
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true` alongside the existing playback,
  touch-event, and touch-reaction gates. The env value was checked only as
  present/redacted.
- Remote focused tests, remote Go build, Gateway restart, local health check,
  and public direct-source `/healthz` and `/xiaozhi/ota/` probes passed.
- Device `44:1b:f6:e2:6a:60` was recovered with read-only chip-id plus
  hard-reset. No firmware flash and no NVS write occurred.
- Product-device runtime evidence:
  `reports/a21-stackchan-state-reaction-evidence-20260604-2316.json`.
  Trace `a21-trace-state-reaction-wav-20260604` / session
  `a21-session-state-reaction-wav-20260604` delivered host WAV-triggered
  `speaking -> idle` state reactions through the real Xiaozhi socket, with
  `xiaozhi.state_reaction.robot_led_color.sent`,
  `xiaozhi.state_reaction.robot_head_angles_set.sent`, redacted MCP
  responses, and registry echo for idle LED `20/20/40` plus head
  `yaw=0,pitch=24,speed=160`.
- Focused local tests pass for state reaction delivery, MCP-required gating,
  socket disconnect registry behavior, and app env wiring. `GOMAXPROCS=2 make
  verify` passed before the final evidence-log update.

Current conclusion:

- State/body reaction is now product-device evidenced at the Gateway/MCP/body
  level, and stale online registry false-greens are fixed.
- This is not full realtime voice parity or full PRD physical acceptance
  because the evidence used `/v1/xiaozhi/say` with a short WAV to trigger the
  state path. Natural microphone-triggered listen/speak evidence, screen
  visual acceptance, app lifecycle, camera, NFC, infrared, no-cable boot/power,
  and richer StackChan choreography remain open.

## Latest Control-Tower Result - 2026-06-04 Xiaozhi Physical Acceptance Tooling

Current implementation state:

- The next PRD path after state/body evidence is the stock Xiaozhi physical
  acceptance chain:
  `xiaozhi-physical-evidence` -> `stackchan-accept --check xiaozhi-half-duplex`
  -> `xiaozhi-physical-prd-review`.
- Under the active TUN route, local evidence commands must be run with
  `A21_DIRECT_SOURCE_IP=192.168.1.20`; this is already implemented in the A21
  direct HTTP helper and is equivalent to the successful
  `curl --interface 192.168.1.20 --noproxy '*' ...` checks.
- A false blocker was found and fixed: the physical evidence reader rejected
  the whole Gateway device record when safe product runtime echo keys contained
  words such as `prompt`, for example `roleplay_prompt_text_stored=false`.
  The report does not serialize full runtime echo, so the reader now scans
  only the runtime echo fields it consumes while keeping trace/audio/instrument
  redaction checks strict.
- Focused tests passed:
  `go test ./internal/app -run 'TestRunXiaozhiPhysicalEvidence|TestRunXiaozhiHalfDuplexAcceptance|TestA21DirectHTTPClient' -count=1`.
- After the fix, live ECS evidence generation no longer fails as
  `gateway data unsafe`. It produces an honest blocked report:
  `reports/a21-xiaozhi-physical-evidence-20260604-232451.781686000.json`.
- The paired half-duplex check also produces an honest blocked report:
  `reports/a21-xiaozhi-half-duplex-acceptance-20260604-232452.185159000.json`.
  These are local generated reports under the ignored `reports/` directory and
  are not product acceptance artifacts.
- Current live device state from `/v1/devices`: device `44:1b:f6:e2:6a:60` is
  `connection_status=xiaozhi_ws_disconnected`; latest trace has
  `xiaozhi.listen.start`, `asr.stream.start`, and state-reaction MCP delivery,
  but no `xiaozhi.opus_frame.decoded`, no `audio.ingress.buffered`, no
  `vad.speech.end`, no `xiaozhi.listen.auto_stop`, no TTS downlink, no playback
  ack, and no audible observation.

Current conclusion:

- The acceptance command path is unblocked and accurate. The remaining blocker
  for PRD physical voice acceptance is real device/runtime evidence: bring the
  product device back online, trigger a natural microphone listen/speak turn,
  capture audible/playback observation, then run the existing PRD chain.
- No firmware flash, NVS write, provider execution, V21 execution, generic
  `xiaozhi.bin` product flash, Git prune/gc, or internal-test3 rollback
  occurred in this tooling fix.
