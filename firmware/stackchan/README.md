# A21 StackChan Firmware

This is the A21 firmware lane for M5Stack StackChan/CoreS3.

## Identity

- Project: `A21`
- Firmware ID: `a21-stackchan`
- Board: `m5stack-cores3`
- Version: `0.1.0`

PlatformIO runs `scripts/a21_build_identity.py` before each firmware build. The script writes ignored `include/a21_firmware_build.generated.h` with the current git commit, so the device build identity can be shown on screen and sent in device events without committing generated metadata.

## Build

From the repository root:

```bash
make firmware-tools
go run ./cmd/a21 firmware-check
make firmware-test
make firmware-build
make firmware-mic-probe-build
make firmware-imu-probe-build
make firmware-sensor-probe-build
make stackchan-official-audio-smoke-build
make stackchan-official-pcm-bridge-build
make firmware-mic-probe-upload-blocker-check
make firmware-imu-probe-upload-blocker-check
make firmware-sensor-probe-upload-blocker-check
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make firmware-mic-probe-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make firmware-imu-probe-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make firmware-sensor-probe-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make stackchan-official-audio-smoke-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL='ws://HOST:21080/ws/audio?device_id=stackchan-001' \
make stackchan-official-pcm-bridge-flash-plan
```

`make firmware-tools` creates the repository-local `.a21-tools/` PlatformIO virtualenv pinned to `platformio==6.1.19`. `make firmware-test` runs host-native protocol and state-machine tests. It does not flash hardware.

`firmware-check` treats `a21-firmware.json` as the release identity source of truth. The CoreS3 and native PlatformIO environments must use matching `A21_FIRMWARE_ID`, `A21_FIRMWARE_VERSION`, and `A21_FIRMWARE_BOARD` build flags, or the build gate fails before packaging.

`make firmware-mic-probe-build` compiles `a21_stackchan_cores3_mic_probe`, an isolated diagnostic build that turns on M5Unified microphone capture with `A21_ENABLE_MIC_DIAGNOSTIC_PROBE=1`. It is not the default environment and must not be packaged as a production release artifact.

`make firmware-imu-probe-build` compiles `a21_stackchan_cores3_imu_probe`, an isolated diagnostic build that turns on read-only M5Unified IMU sampling with `A21_ENABLE_IMU_DIAGNOSTIC_PROBE=1`. It reports IMU sample counters, acceleration, gyro, and coarse posture through runtime echo. It is not the default environment and must not be packaged as a production release artifact.

`make firmware-sensor-probe-build` compiles `a21_stackchan_cores3_sensor_probe`, an isolated read-only diagnostic build that turns on M5CoreS3 LTR553 ambient-light/proximity sampling and StackChan-BSP INA226 battery voltage/current sampling. It reports sensor counters and values through runtime echo. It is not the default environment and must not be packaged as a production release artifact.

`make firmware-mic-probe-upload-blocker-check` proves the diagnostic environment also fails raw PlatformIO upload targets before a flash can start.

`make firmware-imu-probe-upload-blocker-check` provides the same no-raw-upload proof for the IMU diagnostic environment.

`make firmware-sensor-probe-upload-blocker-check` provides the same no-raw-upload proof for the sensor diagnostic environment.

`make firmware-mic-probe-flash-plan` and `make firmware-mic-probe-flash-execute` are the only diagnostic microphone flash lane. They rebuild the probe firmware, require a clean git source tree, check the embedded A21 identity and `diagnostic_probe_m5unified_i2s_capture` marker, and require `A21_MIC_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE` before any write. Use this only for microphone bring-up, not for production release packaging.

`make firmware-imu-probe-flash-plan` and `make firmware-imu-probe-flash-execute` are the matching read-only diagnostic IMU flash lane. They rebuild `a21_stackchan_cores3_imu_probe`, require a clean git source tree, check the embedded A21 identity and `diagnostic_probe_m5unified_imu` marker, and require `A21_IMU_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE` before any write. Use this only for IMU bring-up evidence, not for production release packaging.

`make firmware-sensor-probe-flash-plan` and `make firmware-sensor-probe-flash-execute` are the matching read-only diagnostic sensor flash lane. They rebuild `a21_stackchan_cores3_sensor_probe`, require a clean git source tree, check the embedded A21 identity plus `diagnostic_probe_ltr553_ambient_light`, `diagnostic_probe_ltr553_proximity`, and `diagnostic_probe_ina226_battery` markers, and require `A21_SENSOR_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE` before any write. Use this only for sensor bring-up evidence, not for production release packaging.

`make stackchan-official-audio-smoke-build`, `make stackchan-official-audio-smoke-flash-plan`, and `make stackchan-official-audio-smoke-flash-execute` are the official StackChan/CoreS3 codec speaker-smoke lane. This lane exports official StackChan from Git `HEAD`, applies only the A21 audio-smoke overlay, verifies mature codec evidence, builds with ESP-IDF, records all `flash_args` parts and hashes, and requires `A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE` before any write. Use this only to validate clear physical speaker output through the official codec/HAL boundary. It is not production A21 firmware and does not replace A21 release packaging.

`make stackchan-official-pcm-bridge-build` exports official StackChan from Git `HEAD`, applies the A21 PCM bridge overlay, and builds `a21-stackchan-official-pcm-bridge.bin` through ESP-IDF. `make stackchan-official-pcm-bridge-nvs-plan` is no-write and records the redacted `a21/device_id` plus `a21/audio_ws_url` provisioning plan for the NVS partition at `0x9000/0x4000`. `make stackchan-official-pcm-bridge-nvs-execute` requires `A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS`, backs up the existing NVS partition, preserves current entries including calibration, mutates only the A21 namespace keys, and writes only the NVS partition. The run files live under `.a21-run/` because they may contain local Wi-Fi or historical device secrets. `make stackchan-official-pcm-bridge-flash-plan` remains no-flash for the app image: it checks the bridge app, required flash parts, USB serial port, `device_id`, and a redacted A21 audio websocket endpoint. No bridge app flash-execute target exists yet. If flashed by a future guarded lane without `a21/audio_ws_url`, the bridge must stay on a black A21 status page and remain silent.

## Runtime Surface

The current firmware starts in local fallback render state and renders the A21 identity, firmware version plus commit, state label, Gateway target, connection lifecycle status, and short status text on the CoreS3 screen. Gateway `control.event` messages are parsed by `a21_firmware_protocol.h` and applied to a thin device-local state model in `a21_firmware_state.h`.

Gateway configuration is currently compile-time and A21-only:

- `A21_GATEWAY_HOST=10.21.0.1`
- `A21_GATEWAY_PORT=21080`
- control path `/ws/control`
- audio path `/ws/audio`

Local hardware bring-up can override the Gateway host and Wi-Fi credentials through ignored `include/a21_firmware_secrets.local.h`. That header is included early enough to override the compile-time `A21_GATEWAY_HOST` macro, which is useful when a home Mac is at a temporary LAN address such as `192.168.1.20`. The native test environment defines `A21_IGNORE_LOCAL_SECRETS=1`, so local provisioning cannot change baseline unit-test expectations. Do not put local SSID/password values in `platformio.ini`, docs, commits, or artifact names.

`a21_firmware_connection.h` owns the hardware-free connection lifecycle model:

- Wi-Fi connecting
- Gateway connecting
- Gateway connected
- reconnect wait with bounded backoff
- local fallback on invalid A21 Gateway config

`a21_firmware_wifi.h` owns the first Wi-Fi configuration guard:

- default builds have no Wi-Fi SSID and stay in local fallback
- Wi-Fi status text redacts the password
- SSIDs containing legacy project identity are rejected
- `platformio.ini` must not contain `A21_WIFI_PASSWORD`
- local hardware bring-up may copy `include/a21_firmware_secrets.example.h` to ignored `include/a21_firmware_secrets.local.h`

`a21_firmware_wifi_runtime.h` provides the first guarded Wi-Fi runtime. It uses a small driver interface in native tests and Arduino `WiFi.h` on CoreS3. The runtime will not call `WiFi.begin` without valid credentials, calls it once per connection attempt, transitions to Gateway connecting when Wi-Fi reports connected, and enters reconnect wait after Wi-Fi loss.

`a21_firmware_gateway_ws.h` owns the first guarded Gateway control WebSocket runtime:

- connects only after Wi-Fi moves the device into Gateway connecting
- uses the A21-only Gateway host, port, and `/ws/control` path
- applies incoming `control.event` envelopes through the tested parser
- sends A21 `device.event` envelopes with deterministic seq/trace IDs
- includes firmware id, version, board, and commit in outgoing `device.event` payloads
- includes semantic capability status for microphone, speaker, screen, screen touch, top touch, Y-axis servo, and RGB in outgoing `device.event` payloads
- enters reconnect wait when the control socket disconnects
- uses `links2004/WebSockets @ 2.7.3` behind a small driver interface on CoreS3

`a21_firmware_audio_ws.h` owns the first guarded audio WebSocket runtime:

- opens a separate `/ws/audio` socket only after the control Gateway is connected
- sends deterministic mock `audio.frame` envelopes with a full 640-byte `pcm_s16le`, 16 kHz, mono, 20 ms silence payload encoded in base64
- sends queued microphone capture frames as A21 `audio.frame` envelopes with real PCM payloads when the device is in a capture-safe state
- applies Gateway ack `control.event` messages through the same tested parser
- accepts Gateway `audio.playback.chunk` envelopes into a bounded playback buffer keyed by `stream_id`
- keeps all mic/speaker switching inside the firmware driver layer so Gateway still sees only A21 protocol envelopes

Current local controls are intentionally minimal and routed through semantic device intents:

- `BtnA`: emit `touch.wake_or_listen` with `touch_source=screen`
- `BtnB`: emit `touch.barge_in` with `touch_source=top_sensor`
- `BtnC`: send one mock silence `audio.frame` to Gateway `/ws/audio`

`a21_firmware_touch.h` keeps touch as a semantic runtime (`wake_or_listen`, `barge_in`) instead of exposing coordinates or hardware registers to Gateway. Real screen touch and top-sensor calibration remain future hardware work; this slice only proves the tested intent pipeline and preserves the source label.

`a21_firmware_playback.h` owns the first playback state machine. It starts a stream when the local state becomes `speaking` with a `stream_id`, writes only once for the same stream, and immediately stops plus clears pending audio when the state changes to `interrupted`, `listening`, `error`, `local_fallback`, or another non-speaking state. On CoreS3 the stop/clear callbacks now call `M5.Speaker.stop(...)` for the A21 speaker channel.

`a21_firmware_audio_playback.h` owns the hardware-free playback chunk parser and bounded buffer. It accepts only A21 `audio.playback.chunk` envelopes for the current device, validates current Phase 5 PCM mono chunk metadata (`pcm_s16le`, 16 kHz, 20 ms), decodes each accepted payload into a fixed 640-byte PCM frame, tracks queue depth/drop counts, and clears the buffer when render state leaves `speaking`.

`a21_firmware_speaker.h` owns the first A21 release-firmware speaker pump boundary, but the real-device acceptance baseline is now the official StackChan/CoreS3 codec smoke lane. The earlier hand-written A21 playback variants produced audible "telegraph" artifacts on the real StackChan and must not be used to claim speaker acceptance. Future production downlink work should migrate this boundary toward the official `AudioCodec` / `OutputData` path or an A21 adapter over the same mature codec/HAL layer. The current PCM/base64 envelope remains a diagnostic bridge; the mature production media direction is A21-owned binary Opus over the device audio WebSocket.

Firmware runtime echo includes playback-buffer and speaker-pump counters: queued chunks, total accepted chunks, dropped chunks, clear count, played frames, busy ticks, driver errors, and last stream ID. These fields are diagnostic evidence for the speaker/downlink lane; they do not by themselves prove that physical sound was heard.

`a21_firmware_mic.h` owns the first microphone capture policy boundary and a bounded four-frame uplink queue. The hardware-free policy still records one 20 ms / 16 kHz / mono PCM16 frame only when the render state is capture-safe and the speaker channel queue is empty, then the audio WebSocket runtime can encode that frame into an A21 `audio.frame`. On the default physical CoreS3 release build, the M5Unified mic capture path is still guarded off because earlier repeated I2S stop/start behavior could crash inside `Mic_Class::mic_task` / ESP-IDF `i2s_stop`; firmware reports the microphone capability as `disabled_m5unified_i2s_stop_crash_guard`.

The dedicated `a21_stackchan_cores3_mic_probe` environment enables the same M5Unified mic path only for diagnosis. Its capability status is `diagnostic_probe_m5unified_i2s_capture`, not production `available`. The CoreS3 driver now switches I2S based on real M5Unified task state (`isRunning`) instead of pin-configuration state (`isEnabled`) so probe builds do not end/restart speaker or mic on every capture frame.

The firmware still does not run VAD on device, prove acoustic echo cancellation, prove physical microphone quality, or claim full-duplex behavior. Current speaker output is a guarded CoreS3 build path, while physical microphone uplink is intentionally disabled behind the crash guard and must not be counted as real user-facing audio until hardware acceptance passes.

`a21_firmware_imu.h` owns the first read-only IMU diagnostic boundary. The default release build keeps `imu=planned_9_axis_imu`. The isolated `a21_stackchan_cores3_imu_probe` build reports `imu=diagnostic_probe_m5unified_imu` and samples M5Unified IMU through a small driver interface, producing runtime echo fields for sample count, read errors, acceleration in mg, gyro in mdps, and coarse posture. These fields are diagnostic telemetry only; they do not promote IMU to product `available`.

`a21_firmware_sensors.h` owns the first read-only environmental and power diagnostic boundary. The default release build keeps `ambient_light=planned_ambient_light_sensor`, `proximity=planned_proximity_sensor`, and `battery=planned_550mah_battery`. The isolated `a21_stackchan_cores3_sensor_probe` build reports `diagnostic_probe_ltr553_ambient_light`, `diagnostic_probe_ltr553_proximity`, and `diagnostic_probe_ina226_battery`, then emits sample count, read errors, raw ambient/proximity values, and battery voltage/current through runtime echo. These fields are diagnostic telemetry only; they do not promote sensors to product `available`.

Servo safety currently lives in `a21_firmware_config.h`:

- `A21_SERVO_Y_MIN_DEG=5`
- `A21_SERVO_Y_MAX_DEG=85`
- `a21ClampServoY(...)`

`a21_firmware_motion.h` maps semantic render states to safe Y-axis targets and writes only when the target angle changes. The CoreS3 main loop uses the official `m5stack/StackChan-BSP` driver and sends the clamped Y-axis target through `M5StackChan.Motion.moveY(...)`. Motion remains bounded by the A21 servo clamp; do not bypass that clamp for physical validation.

`a21_firmware_rgb.h` maps semantic render states to RGB state colors and writes only when the target color changes. The CoreS3 main loop uses the official `m5stack/StackChan-BSP` driver to apply the color to the 12 body RGB LEDs and calls `M5StackChan.refreshRgb()` after each state color change.

## Upload

There is still no upload target. Raw PlatformIO upload targets are blocked by `scripts/a21_block_raw_upload.py`; do not bypass it or remove it. A future explicit guarded flash command must require the A21 upload and device-identity receipts before any real hardware write can exist.

The blocker is part of release verification:

```bash
make firmware-upload-blocker-check
```

That command intentionally invokes PlatformIO's raw upload target and expects the A21 blocker to fail it before flashing can start.

Phase 5B/5C adds only dry-run guards:

```bash
go run ./cmd/a21 serial-list
go run ./cmd/a21 firmware-artifact-check --artifact firmware/artifacts/<a21-stackchan...bin>
go run ./cmd/a21 firmware-upload-check --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --commit <expected-git-sha>
go run ./cmd/a21 firmware-device-report --gateway-url http://127.0.0.1:21080 --output-dir reports
go run ./cmd/a21 firmware-device-check --artifact firmware/artifacts/<a21-stackchan...bin> --device-report reports/a21-devices-<timestamp>.json --device-id stackchan-001 --commit <expected-git-sha> --max-device-age-ms 300000
```

These commands inventory serial devices and validate artifact identity, board, version, checksum, sibling artifact manifest, release-index record, latest same-commit package selection, expected git commit, embedded binary identity, explicit serial target, port existence, serial-like path form, whether another process is already holding the serial path, and whether the A21 Gateway has recently seen the expected device ID with matching A21 firmware identity. They do not flash the device.

Successful upload-check and device-check output are dry-run receipts with `flash_allowed: false`. They are preflight records, not permission to run `pio run -t upload`, and the PlatformIO blocker is expected to fail raw upload attempts.

Initial bring-up or recovery uses the explicit A21 bootstrap flash path, not raw PlatformIO upload:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
make firmware-bootstrap-flash-plan

A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_BOOTSTRAP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_FIRMWARE \
make firmware-bootstrap-flash-execute
```

The plan command writes a no-flash receipt with SHA-256 for bootloader, partitions, `boot_app0.bin`, and the packaged A21 application artifact. The execute command rebuilds the plan, requires the confirmation token, uses repository-local esptool, and writes a timestamped execution receipt under `reports/`.

`make firmware-package` and `a21 firmware-package` refuse to run when the git worktree is dirty. This is intentional: a firmware binary must not be packaged under a commit SHA that does not fully describe its source. The package step writes the A21-named `.bin`, sibling `.sha256`, sibling `.manifest.json`, and `a21-firmware-release-index.jsonl`; upload-path dry-run guards require all of them to agree.
