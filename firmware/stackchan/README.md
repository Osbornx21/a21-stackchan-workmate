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
```

`make firmware-tools` creates the repository-local `.a21-tools/` PlatformIO virtualenv pinned to `platformio==6.1.19`. `make firmware-test` runs host-native protocol and state-machine tests. It does not flash hardware.

## Runtime Surface

The current firmware starts in local fallback render state and renders the A21 identity, firmware version plus commit, state label, Gateway target, connection lifecycle status, and short status text on the CoreS3 screen. Gateway `control.event` messages are parsed by `a21_firmware_protocol.h` and applied to a thin device-local state model in `a21_firmware_state.h`.

Gateway configuration is currently compile-time and A21-only:

- `A21_GATEWAY_HOST=10.21.0.1`
- `A21_GATEWAY_PORT=21080`
- control path `/ws/control`
- audio path `/ws/audio`

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

`a21_firmware_speaker.h` owns the first real speaker pump boundary. It consumes decoded PCM frames only while the render state is `speaking`, waits when the M5Unified speaker channel already has two queued buffers, copies each 20 ms frame into one of three stable slots, and then calls `M5.Speaker.playRaw(...)`. This avoids handing M5Unified a pointer to buffer memory that may be cleared on barge-in.

`a21_firmware_mic.h` owns the first microphone capture policy boundary and a bounded four-frame uplink queue. It records one 20 ms / 16 kHz / mono PCM16 frame only when the render state is capture-safe and the speaker channel queue is empty, then the audio WebSocket runtime encodes that frame into an A21 `audio.frame`. This encodes the current M5Unified internal mic/speaker constraint directly in firmware tests instead of pretending full-duplex is solved.

The firmware still does not run VAD on device, prove acoustic echo cancellation, prove physical microphone quality, or claim full-duplex behavior. Current mic uplink and speaker output are guarded CoreS3 build paths and still need physical hardware acceptance before they count as real user-facing audio.

Servo safety currently lives in `a21_firmware_config.h`:

- `A21_SERVO_Y_MIN_DEG=5`
- `A21_SERVO_Y_MAX_DEG=85`
- `a21ClampServoY(...)`

`a21_firmware_motion.h` maps semantic render states to safe Y-axis targets and writes only when the target angle changes. The CoreS3 main loop is wired through a no-op motion driver for now, so the runtime path is exercised without actuating hardware. Real servo hardware output still requires a future calibrated driver.

`a21_firmware_rgb.h` maps semantic render states to RGB state colors and writes only when the target color changes. The CoreS3 main loop is wired through a no-op RGB driver for now, so the runtime path is exercised without lighting hardware. Real RGB output still requires a future calibrated driver.

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

`make firmware-package` and `a21 firmware-package` refuse to run when the git worktree is dirty. This is intentional: a firmware binary must not be packaged under a commit SHA that does not fully describe its source. The package step writes the A21-named `.bin`, sibling `.sha256`, sibling `.manifest.json`, and `a21-firmware-release-index.jsonl`; upload-path dry-run guards require all of them to agree.
