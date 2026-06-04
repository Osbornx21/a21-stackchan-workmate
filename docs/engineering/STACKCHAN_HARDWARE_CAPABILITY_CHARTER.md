# A21 StackChan Hardware Capability Charter

## Purpose

StackChan is not a low-end edge endpoint. A21 treats it as an embodied robot surface with a strong CoreS3 base and robot body. Firmware remains thin for model, proxy, and V21 responsibilities, but it must not be thin in product expression.

A21 must either use a hardware capability well, keep it explicitly planned, or leave it out of the user-facing product. It must not ship a custom effect that is worse than the original StackChan experience.

## Capability Contract

Every firmware-originated `device.event` and `runtime.echo` carries a semantic `capabilities` map. This map is not acceptance proof; it is the device's honest declaration of what A21 knows about the hardware surface.

Current stable capability keys:

- `microphone`
- `speaker`
- `screen`
- `screen_touch`
- `top_touch`
- `servo_y`
- `servo_x`
- `rgb`
- `camera`
- `imu`
- `ambient_light`
- `proximity`
- `battery`
- `nfc`
- `infrared`

Status values must be literal and honest:

- `available`: implemented in the current product firmware and covered by tests or acceptance commands.
- `planned_*`: real StackChan hardware or product surface, but not yet product-ready in A21.
- `disabled_*`: intentionally disabled because the current implementation would be unsafe, unstable, or misleading.
- `diagnostic_*`: available only in a special diagnostic build and not part of the product firmware.

## Current Release Declaration

The current product firmware declares:

- `speaker`, `screen`, `screen_touch`, `top_touch`, `servo_y`, and `rgb` as `available`.
- `microphone` as `disabled_m5unified_i2s_stop_crash_guard` in the release build, with a separate diagnostic probe path.
- `imu` as `planned_9_axis_imu` in the release build, with a separate read-only diagnostic probe path.
- `servo_x`, `camera`, `ambient_light`, `proximity`, `battery`, `nfc`, and `infrared` as planned capabilities.

This prevents A21 from silently shrinking StackChan into a screen-plus-LED device while still avoiding false claims about unimplemented hardware.

## Official StackChan Parity Gap Map

Frozen by transition:
`T-STACKCHAN-OFFICIAL-HW-PARITY-GAP-MAP-001`.

Plan:
`docs/plans/2026-06-04-stackchan-official-hardware-parity-full-landing.md`.

This map treats the official StackChan/Xiaozhi package as a reference
hardware/control/status surface only. Official support is not A21 product
acceptance. A capability may be `available`, `diagnostic`, `planned`,
`blocked`, or `product-accepted` only through A21 evidence and gates.

Official source identity:

| Tree | Remote | Branch | HEAD | Dirty state | Reference basis |
| --- | --- | --- | --- | --- | --- |
| `/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan` | `https://github.com/m5stack/StackChan.git` | `main...origin/main` | `da156e1fa0e1c2a5e00b78fbf69b1f7e7bca0483` | Dirty: modified `firmware/dependencies.lock`, `firmware/main/Kconfig.projbuild`, `firmware/main/hal/board/hal_bridge.cc`, `firmware/main/hal/board/stackchan_display.cc`, `firmware/main/hal/hal_ble.cpp`, `firmware/main/hal/hal_io_expander.cpp`, `firmware/main/hal/hal_servo.cpp`, `firmware/main/main.cpp`, `firmware/partitions.csv`, `firmware/sdkconfig.defaults`; untracked `firmware/sources/` | Working tree reference, not a clean upstream baseline |
| `/Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan/firmware/xiaozhi-esp32` | `https://gitclone.com/github.com/78/xiaozhi-esp32.git` | detached `HEAD` | `e77dedb1309153bb63fed285772962c920c97dd4` | Clean | `HEAD` reference |

A21 source identity for this gap map:

| Tree | Remote | Checkout | HEAD | Dirty state before worker edits | Reference basis |
| --- | --- | --- | --- | --- | --- |
| `/Users/jiyurun/.codex/worktrees/be17/New project` | `https://github.com/Osbornx21/a21-stackchan-workmate.git` | detached worker checkout from the main A21 branch line | `15757cad5e7cbcf2df5f2601ffa3904dd766e0ac` | Clean | Current worker tree |

Reference inventory extracted from the official package:

- Official root `README.md` lists CoreS3 screen, capacitive touch, camera,
  proximity and ambient light sensor, IMU, microSD, speaker, dual microphones,
  battery, NFC, infrared, RGB LEDs, and two servos.
- Official HAL exposes battery, backlight, RGB, servo power, head touch,
  IMU motion events, BLE app-control signals, and WebSocket avatar/action
  signals in `firmware/main/hal/hal.h`.
- Official avatar WebSocket supports typed binary frames for Opus, JPEG,
  avatar control, motion control, camera stream start/stop, text, calls,
  device name, heartbeat, video mode, dance sequence, and audio stream start/
  stop in `firmware/main/hal/hal_ws_avatar.cpp`.
- Official servo HAL implements yaw and pitch servos, angle limits, torque,
  zero calibration, and yaw PWM/rotation mode in
  `firmware/main/hal/hal_servo.cpp`.
- Official IMU and top-touch HALs expose BMI270 shake events and SI12T
  press/release/swipe gestures in `firmware/main/hal/hal_imu.cpp` and
  `firmware/main/hal/hal_head_touch.cpp`.
- Official Xiaozhi MCP common tools include `self.get_device_status`,
  `self.audio_speaker.set_volume`, `self.screen.set_brightness`,
  `self.screen.set_theme`, and `self.camera.take_photo`; user-only tools
  include `self.get_system_info`, `self.reboot`,
  `self.upgrade_firmware`, `self.screen.get_info`, and
  `self.screen.snapshot` in `firmware/xiaozhi-esp32/main/mcp_server.cc`.
- Official Xiaozhi display/status paths expose status, notification, emotion,
  chat message, theme, status bar, and power-save methods in
  `firmware/xiaozhi-esp32/main/display/display.h` and map device states such
  as `starting`, `wifi_configuring`, `idle`, `connecting`, `listening`,
  `speaking`, `upgrading`, `activating`, `audio_testing`, and `fatal_error`
  in `device_state_machine.*` and `application.cc`.
- Official Xiaozhi audio service covers microphone capture, Opus encode/decode,
  wake word, VAD/AEC-capable processing, playback queue, and output volume in
  `firmware/xiaozhi-esp32/main/audio/*`.

Current A21 inventory extracted from the worker checkout:

- A21 Gateway already accepts stock `/v1/xiaozhi` devices and records a
  sanitized stock capability map for `microphone=available_xiaozhi_opus_ingress`
  and `speaker=available_xiaozhi_opus_downlink`, but full PRD physical
  acceptance remains open.
- `POST /v1/xiaozhi/speaker-volume` is live-whitelisted and sends the stock
  MCP tool `self.audio_speaker.set_volume`.
- `POST /v1/stackchan/official/control` maps only selected A21 semantic
  events to official `ControlAvatar`, `ControlMotion`, and `DanceSequence`
  frames.
- A21 runtime/evidence paths know `screen`, `screen_touch`, `top_touch`,
  `servo_y`, and `rgb` as stable declared surfaces, but physical acceptance is
  per-report rather than automatic product acceptance.
- A21 keeps `imu`, `ambient_light`, `proximity`, and `battery` on read-only
  diagnostic tracks; diagnostic probe evidence does not promote product
  availability.
- A21 keeps `servo_x`, `camera`, `nfc`, and `infrared` planned or blocked
  until safety, privacy, product semantics, and physical evidence are approved.

Parity matrix:

| Surface | Official package | A21 current classification | Landing class | Owner transition | Acceptance evidence | Rollback path |
| --- | --- | --- | --- | --- | --- | --- |
| Microphone | Dual microphones through Xiaozhi audio service, Opus uplink, wake word, VAD, and AEC-capable paths | `available_xiaozhi_opus_ingress` as stock transport evidence; not `product-accepted` | Physical evidence gate | `T-XIAOZHI-PHYSICAL-PUBLIC-GATEWAY-TRACE-001` follow-up and audio evidence closure | Fresh device report with `trace_id`, `session_id`, `device_id`, real Opus ingress, ASR final, no raw audio/transcript persistence | Keep stock uplink as candidate only; do not promote PRD acceptance |
| Speaker | Codec output, volume control, playback queue, Opus downlink, speaking UI | `available_xiaozhi_opus_downlink`; MCP volume accepted; playback-start/stop proof still open | Freeze plus playback-start proof | `T-AUDIO-002`, `T-AUDIO-003`, and physical playback ack follow-up | Runtime volume trace plus trusted audible or device playback-start/stop evidence | Keep volume endpoint; block full PRD acceptance |
| Screen | Status, notification, emotion, chat, theme, power save, status bar | Available as coarse A21 registry/expression and selected relay, not full official UI parity | Medium-risk UI parity | `T-STACKCHAN-OFFICIAL-STATUS-DISPLAY-PARITY-001` | Display-state registry tests plus operator screen evidence before product claims | Fall back to coarse A21 state/expression |
| Screen touch | Official display touch can enter listen/config flows | Acceptance CLI cases exist for screen touch; not full official flow parity | Low-medium physical gate | `T-STACKCHAN-OFFICIAL-TOUCH-ACTION-EVIDENCE-001` | Touch acceptance report tied to device and trace/session ids | Keep touch as declared surface; block new behavior |
| Top touch | Three-zone press/release/swipe gestures | Acceptance CLI cases exist for top tap/swipe/barge-in; product proof incomplete | Low-medium physical gate | `T-STACKCHAN-OFFICIAL-TOUCH-ACTION-EVIDENCE-001` | Press/swipe/barge-in report with no false promotion | Keep current limited touch acceptance only |
| Servo Y/pitch | Pitch servo angle/torque/motion with clamps | Official action plan maps state/motion to clamped pitch packets and metadata; `/v1/xiaozhi/mcp-control` now also sends official `self.robot.set_head_angles` to the stock MCP runtime | Keep and verify | `T-STACKCHAN-OFFICIAL-ROBOT-MCP-BODY-CONTROL-001` plus physical evidence window | 2026-06-04 serial HAL evidence for pitch `30`, trace `a21-trace-live-robot-head-responsefix`; broader pose/product acceptance still separate | Remove robot MCP mapping if bounds or official runtime behavior regress |
| Servo X/yaw | Yaw servo angle/PWM/continuous rotation mode | Official action plan marks yaw as candidate; `/v1/xiaozhi/mcp-control` now sends bounded official `self.robot.set_head_angles` yaw values, but full motion semantics remain evidence-gated | Medium-high safety gate | `T-STACKCHAN-OFFICIAL-ROBOT-MCP-BODY-CONTROL-001` plus mechanical safety follow-up | 2026-06-04 serial HAL evidence for yaw `12`, trace `a21-trace-live-robot-head-responsefix`; mechanical range/collision evidence still required | Disable yaw writes if physical range, torque, or collision evidence fails |
| RGB | 12 LED individual/all control | `/v1/xiaozhi/mcp-control` now sends official `self.robot.set_led_color` as an all-LED color control; rich per-LED choreography is not product-exposed | Medium Gateway/action mapping | `T-STACKCHAN-OFFICIAL-ROBOT-MCP-BODY-CONTROL-001` plus future RGB choreography cut | 2026-06-04 serial HAL evidence for RGB `20,0,168`, trace `a21-trace-live-robot-led-responsefix`; visible product expression acceptance still separate | Remove LED mapping if official runtime rejects or unsafe args leak |
| Camera/photo | Camera capture and MCP photo explanation | Planned, privacy-sensitive | High privacy spike | `T-STACKCHAN-OFFICIAL-CAMERA-PRIVACY-SPIKE-001` | ADR/privacy plan, local consent behavior, redacted evidence | Keep camera blocked from product controls |
| Camera stream/video | JPEG frames, start/stop stream, video mode | Not product surface | High privacy/bandwidth spike | `T-STACKCHAN-OFFICIAL-CAMERA-STREAM-SPIKE-001` | Explicit stream lifecycle, bandwidth, privacy, and UI evidence | Keep stream/video blocked |
| IMU | BMI270 shake event, motion hooks | Diagnostic/planned only | Medium diagnostic-to-product | `T-STACKCHAN-OFFICIAL-SENSOR-BATTERY-DIAG-001` | Read-only diagnostic report, then separate product behavior evidence | Keep `imu=planned_9_axis_imu` |
| Ambient/proximity | CoreS3 proximity and ambient light hardware | Diagnostic/planned only | Medium diagnostic-to-product | `T-STACKCHAN-OFFICIAL-SENSOR-BATTERY-DIAG-001` | Sensor-probe report with samples/read-error deltas | Keep planned sensor statuses |
| Battery/charging | Battery status and status-bar display | Diagnostic/planned only | Low read-only, medium product | `T-STACKCHAN-OFFICIAL-SENSOR-BATTERY-DIAG-001` | INA226/battery diagnostic evidence plus display policy | Keep `battery=planned_550mah_battery` |
| NFC | Full-featured NFC module | Planned, no product semantics | High product-semantics spike | `T-STACKCHAN-OFFICIAL-NFC-IR-SPIKE-001` | Opt-in interaction ADR and physical evidence | Keep `nfc=planned_nfc` |
| Infrared | IR transmitter/receiver | Planned, no product semantics | High product-semantics spike | `T-STACKCHAN-OFFICIAL-NFC-IR-SPIKE-001` | Opt-in IR semantics, safety, and evidence | Keep `infrared=planned_infrared_tx_rx` |
| MCP device status | `self.get_device_status` | Live-whitelisted through `/v1/xiaozhi/mcp-control` as low-risk control contract; not product-accepted | Low | `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001` | Endpoint tests, MCP capability check, redacted send marker, no raw response capture | Remove endpoint mapping if raw result or hidden claim leaks |
| MCP speaker volume | `self.audio_speaker.set_volume` | Live-whitelisted through `/v1/xiaozhi/speaker-volume` and unified `/v1/xiaozhi/mcp-control`; physically used for runtime volume | Frozen delivery contract; physical loudness still evidence-gated | `T-STACKCHAN-OFFICIAL-MCP-SPEAKER-VOLUME-FREEZE-001` landed host-local contract | Existing volume traces plus regression tests; physical playback/loudness remains separate | Keep current endpoints; remove unified mapping if unsafe args or raw MCP result leaks |
| MCP brightness/theme/info | `self.screen.set_brightness`, `self.screen.set_theme`, `self.screen.get_info` | Live-whitelisted through `/v1/xiaozhi/mcp-control`; named product-operation endpoints are deployed for device status, screen brightness, screen theme, and MCP capabilities; not product-accepted | Low-medium | `T-STACKCHAN-OFFICIAL-MCP-STATUS-PARITY-001` | Endpoint tests with allowed tools only, bounded arguments, redacted send markers, no raw body logging, ECS deploy, and 2026-06-04 serial evidence for `SetTheme: dark` and `Backlight: Set brightness to 55` | Remove endpoint mapping if raw result, hidden claim, or unsafe argument leaks |
| MCP robot head/LED | `self.robot.get_head_angles`, `self.robot.set_head_angles`, `self.robot.set_led_color` | Live-whitelisted through `/v1/xiaozhi/mcp-control`; deployed to ECS with numeric JSON-RPC ids and redacted inbound MCP responses | Medium physical body-control gate | `T-STACKCHAN-OFFICIAL-ROBOT-MCP-BODY-CONTROL-001` | Focused Go tests, ECS deploy, trace markers `xiaozhi.mcp.robot_led_color.sent`, `xiaozhi.mcp.robot_head_angles_set.sent`, device-session `xiaozhi.mcp.response.received`, and 2026-06-04 serial HAL logs | Disable mappings and keep body parity blocked if official MCP response handling or physical movement regresses |
| MCP reboot/OTA | System info, reboot, upgrade firmware | Not product-exposed | High guardrail | Future guarded system-control ADR only | Explicit ADR, guard, no unapproved reboot/upgrade | Keep reboot/upgrade blocked |
| Device state display | Starting, wifi, idle, connect, listen, speak, upgrade, fatal/error states | Coarser A21 state/trace view | Medium-high | `T-STACKCHAN-OFFICIAL-STATUS-DISPLAY-PARITY-001` | Stable state map, traces, display evidence | Fall back to coarse state/expression |
| Official WS avatar | Avatar, motion, dance, heartbeat, text, call, video, camera, audio stream binary frames | Avatar/motion/dance packet mapping plus action metadata; heartbeat is transport keepalive; text/call/video/camera/audio-stream frames remain blocked | Staged medium/high | `T-STACKCHAN-OFFICIAL-ACTION-PARITY-001` plus camera/audio spikes | Frame-shape tests and physical proof per frame class | Reject unsupported official frame types |
| App lifecycle | Mooncake/AppLauncher/AppAiAgent/AppAvatar/AppDance/AppSetup | Product path parks after direct Xiaozhi start to avoid setup trap | High reconciliation | `T-STACKCHAN-OFFICIAL-APP-LIFECYCLE-PARITY-001` | No-welcome boot proof plus official app-surface reconciliation plan | Keep parked no-welcome product path |
| OTA/provisioning | Official OTA, provisioning, mobile/remote app ecosystem | Product OTA route exists; mobile/app ecosystem not A21 product surface | Medium/high by scope | `T-STACKCHAN-OFFICIAL-OTA-PROVISIONING-SCOPE-001` | ADR and guarded A21 namespace route evidence | Keep official ecosystem out of product claims |

Promotion rule:

- `available` means the current A21 runtime or firmware can expose the surface
  honestly, but it is not sufficient for product acceptance.
- `diagnostic` means a special probe or read-only report exists and must not be
  shipped as product behavior.
- `planned` means StackChan has the hardware or official surface, but A21 has
  not landed an accepted implementation.
- `blocked` means privacy, safety, key-isolation, proxy, firmware, or product
  semantics prevent exposure.
- `product-accepted` requires explicit A21 evidence with `trace_id`,
  `session_id`, `device_id`, no raw secrets/transcripts/audio/provider payloads,
  and no regression of the official-compatible product flash guard.

## Product Gate

Before a planned capability can become `available`, it needs:

- a firmware unit or native test for its parser, state machine, clamp, or driver boundary;
- a Gateway or CLI acceptance surface when the capability leaves the device;
- an operator or instrumented evidence report for physical behavior;
- no leakage of provider keys, V21 internals, X21 names, or global proxy assumptions;
- product behavior that is at least not worse than the original StackChan capability and ideally clearly better for A21's desk-workmate experience.

If a capability fails this gate, keep it as `planned_*`, `diagnostic_*`, or `disabled_*`.

## Near-Term Hardware Tracks

- IMU: posture, bump/pickup detection, attention cues, and safe expression transitions.
- Ambient light and proximity: adaptive brightness, wake/sleep, and interaction distance.
- Battery: honest power state, local fallback behavior, and low-power expression.
- Second servo axis: coordinated head pose once mechanical range and safety clamps are verified.
- Camera: local presence/gesture or privacy-safe visual context, never silent surveillance.
- NFC/infrared: explicit opt-in desk interactions, not hidden automation.

## Hardware Mainline Command

Use this command before starting any planned-hardware driver work:

```bash
go run ./cmd/a21 stackchan-hardware-mainline --gateway-url http://127.0.0.1:21080 --device-id stackchan-001 --output-dir reports
```

The report schema is `a21.stackchan_hardware_mainline.v1`.

The command:

- reads the current Gateway `/v1/devices` report with direct no-proxy LAN access;
- confirms the intended StackChan device is online and declares every planned hardware key;
- emits an ordered hardware track list;
- keeps `dry_run=true`, `flash_allowed=false`, and `delete_allowed=false`;
- writes `reports/a21-stackchan-hardware-mainline-*.json`.

The ordered track list is:

1. `imu`: read-only diagnostic probe.
2. `ambient_light`: read-only diagnostic probe.
3. `proximity`: read-only diagnostic probe.
4. `battery`: read-only diagnostic probe.
5. `servo_x`: motion safety spike.
6. `camera`: privacy-safe vision spike.
7. `nfc`: explicit opt-in interaction spike.
8. `infrared`: explicit opt-in interaction spike.

This report is not proof that a planned capability is product-ready. It is the gate that prevents A21 from forgetting the hardware surface while preserving firmware build and flash discipline.

IMU diagnostic discipline:

- default release firmware must keep `imu=planned_9_axis_imu`;
- `a21_stackchan_cores3_imu_probe` is the only current IMU diagnostic build;
- `make firmware-imu-probe-build` must pass before any physical IMU test window;
- `make firmware-imu-probe-upload-blocker-check` must continue proving raw PlatformIO uploads are blocked;
- `make firmware-imu-probe-flash-plan` and `make firmware-imu-probe-flash-execute` are the only guarded IMU diagnostic flash lane;
- `make stackchan-imu-probe-acceptance` is required before treating IMU telemetry as live diagnostic evidence;
- IMU runtime echo is telemetry, not product promotion.

Sensor diagnostic discipline:

- default release firmware must keep `ambient_light=planned_ambient_light_sensor`, `proximity=planned_proximity_sensor`, and `battery=planned_550mah_battery`;
- `a21_stackchan_cores3_sensor_probe` is the current read-only diagnostic build for CoreS3 LTR553 ambient/proximity telemetry plus StackChan-BSP INA226 battery telemetry;
- `make firmware-sensor-probe-build` must pass before any physical sensor test window;
- `make firmware-sensor-probe-upload-blocker-check` must continue proving raw PlatformIO uploads are blocked;
- `make firmware-sensor-probe-flash-plan` and `make firmware-sensor-probe-flash-execute` are the only guarded sensor diagnostic flash lane;
- `make stackchan-sensor-probe-acceptance` is required before treating ambient/proximity/battery telemetry as live diagnostic evidence;
- sensor runtime echo is telemetry, not product promotion.
