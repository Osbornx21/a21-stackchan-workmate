# A21 Firmware Release Discipline

## Purpose

A21 firmware must be treated as a device release artifact, not a casual sketch. Wrong package, wrong board, wrong port, or wrong project identity can brick time, damage trust, and contaminate X21/V21 work.

## Non-Negotiables

- A21 firmware lives under `firmware/stackchan/`.
- A21 firmware build artifacts must be named with `a21`, target board, firmware version, git commit, and build timestamp.
- A21 firmware binaries must embed an A21 build identity containing firmware ID, version, board, and git commit.
- Each packaged firmware binary must have a sibling `.sha256` file and `.manifest.json` artifact manifest.
- Gateway must record and validate firmware build identity from device events before real hardware acceptance.
- A21 firmware may not use X21 or V21 project names, service names, artifact names, namespaces, or upload targets.
- Firmware upload is forbidden unless a preflight command verifies manifest identity, board ID, version, git commit, and explicit upload port.
- Raw `pio run -t upload` must be blocked by the A21 PlatformIO script; upload-check remains a dry-run receipt and is not permission to flash.
- Firmware packaging is forbidden when the git worktree is dirty.
- `platformio.ini` must not contain Wi-Fi passwords or other local secrets. Real credentials need an explicit provisioning path or ignored `firmware/stackchan/include/a21_firmware_secrets.local.h`, never a committed build flag.
- Provider API keys, V21 access, proxy config, and long-term memory are forbidden in firmware.
- StackChan/CoreS3 remains a thin device client.

## Current Toolchain

PlatformIO is installed in the repository-local ignored path:

```bash
A21_PLATFORMIO_VERSION=6.1.19 make firmware-tools
PLATFORMIO_CORE_DIR=$PWD/.a21-tools/platformio-core .a21-tools/platformio-venv/bin/pio --version
```

`make firmware-tools` creates or repairs `.a21-tools/platformio-venv` and `.a21-tools/platformio-core`, then installs the pinned `platformio==6.1.19`. `firmware-test`, `firmware-build`, and `firmware-upload-blocker-check` depend on this target, so a fresh checkout or CI worker can bootstrap the A21-local toolchain before touching firmware.

This keeps A21 firmware tooling and PlatformIO package state isolated from X21/V21 and from system-global Python packages. `doctor` reports the repository-local PlatformIO version and warns when it drifts from the pinned version.

## Board Baseline

Primary board ID:

```ini
board = m5stack-cores3
```

Primary sources checked on 2026-05-30:

- PlatformIO M5Stack CoreS3 board documentation: https://docs.platformio.org/en/latest/boards/espressif32/m5stack-cores3.html
- M5Unified PlatformIO registry: https://registry.platformio.org/libraries/m5stack/M5Unified
- M5Stack StackChan documentation: https://docs.m5stack.com/en/StackChan

The StackChan Y-axis servo is clamped to the documented safe range of 5 to 85 degrees in firmware config and covered by native tests.

## Pinned Firmware Dependencies

The first firmware protocol parser uses pinned mature dependencies:

- `platform = espressif32@7.0.1`
- `m5stack/M5Unified @ 0.2.16`
- `bblanchon/ArduinoJson @ 7.4.3`
- `links2004/WebSockets @ 2.7.3`
- native unit-test environment: `a21_stackchan_native` with Unity
- `A21_GATEWAY_HOST` and `A21_GATEWAY_PORT=21080` build flags
- `A21_IGNORE_LOCAL_SECRETS=1` in the native test environment, so ignored hardware bring-up credentials cannot change baseline tests
- `A21_FIRMWARE_ID`, `A21_FIRMWARE_VERSION`, and `A21_FIRMWARE_BOARD` build flags
- `scripts/a21_block_raw_upload.py` PlatformIO pre-build script for failing raw upload targets before flashing can start
- `scripts/a21_build_identity.py` PlatformIO pre-build script for generated commit metadata
- diagnostic-only microphone probe environment: `a21_stackchan_cores3_mic_probe`

`firmware-check --kind manifest` rejects unpinned or missing core firmware dependencies and rejects PlatformIO configs that omit the raw upload blocker. It also treats `firmware/stackchan/a21-firmware.json` as the firmware release identity source of truth: every `A21_FIRMWARE_ID`, `A21_FIRMWARE_VERSION`, and `A21_FIRMWARE_BOARD` build flag in the CoreS3 and native test environments must exactly match the manifest. This is intentional: firmware builds must be reproducible, must not silently drift under A21, and must fail fast before any unguarded flash path can run.

The microphone probe environment is deliberately not the release environment. It enables `A21_ENABLE_CORES3_M5UNIFIED_MIC_CAPTURE=1` and `A21_ENABLE_MIC_DIAGNOSTIC_PROBE=1`, causing the device capability status to report `diagnostic_probe_m5unified_i2s_capture` rather than production `available`. Existing firmware package and upload guards continue to accept only the production `a21_stackchan_cores3` release provenance, so a probe build cannot be silently wrapped as a formal A21 release artifact.

The generated commit header is ignored:

```text
firmware/stackchan/include/a21_firmware_build.generated.h
```

It is created by PlatformIO before native tests and CoreS3 builds. It must not be committed.

## Build Discipline

Before any firmware build:

```bash
make firmware-tools
go run ./cmd/a21 firmware-check --kind manifest
PLATFORMIO_CORE_DIR=$PWD/.a21-tools/platformio-core .a21-tools/platformio-venv/bin/pio run -d firmware/stackchan
```

The preferred wrapper is:

```bash
make firmware-test
make firmware-build
make firmware-mic-probe-build
make firmware-imu-probe-build
make firmware-sensor-probe-build
make firmware-upload-blocker-check
make firmware-mic-probe-upload-blocker-check
make firmware-imu-probe-upload-blocker-check
make firmware-sensor-probe-upload-blocker-check
make firmware-package
make firmware-current-artifact-check
make firmware-artifact-prune-plan
make office-handoff
```

`firmware-test` runs the PlatformIO `native` environment and Unity tests. It must stay hardware-free.

`wake-word-firmware-plan` is the first no-execute control-tower receipt for
custom wake words configured through Gateway. It reads the sanitized A21
wake-word intent, emits `a21.wake_word_firmware_plan.v1`, and keeps both
`build_allowed=false` and `flash_allowed=false`. A custom MultiNet profile
remains a planned firmware-build task until a later reviewed build/package lane
and a guarded foreground T7 flash plan prove the requested model on hardware.
This command must not run PlatformIO, invoke esptool, contact providers, or
store local paths in its JSON output.

`wake-word-firmware-package` is the no-hardware package boundary for that later
reviewed xiaozhi/ESP-SR build lane. `wake-word-firmware-build-receipt` first
turns a matching plan, build directory, and explicit reviewed-build report into
a redacted `a21-wake-word-build.json` receipt without touching hardware. The
review report uses `schema_version=a21.wake_word_firmware_build_review.v1`,
`status=reviewed`, exact plan intent fields, and basename-only `app_binary`,
`sdkconfig`, and `flasher_args` values. The package step then consumes either
`--build-receipt <receipt.json>` or the build-dir-local receipt, verifies CoreS3
`sdkconfig.json`, `flash_args`, the required xiaozhi flash parts, and the exact
requested phrase, pinyin, threshold, and review basename, then writes only an
A21-named app binary, checksum, manifest, and package report. It still sets
`flash_allowed=false`, `flash_executed=false`, and `product_ready=false`;
custom wake-word launch readiness requires a later guarded flash plan plus
physical StackChan proof.
If the build directory or receipt is missing, the command writes only a
redacted diagnostic report and remains below package availability.
The external X21 `xiaozhi-esp32` checkout is frozen read-only reference; A21
`xiaozhi-firmware-flash`, `wake-word-firmware-build-receipt`, and
`wake-word-firmware-package` reject build directories from that frozen source
before inspecting or packaging artifacts.

Firmware expression changes must keep the avatar contract testable without hardware. `a21_firmware_display.h` maps render states into an `A21FaceFrame` with avatar-engine concepts such as expression, gaze, eye-open ratio, breath, and mouth-open ratio. This contract is intentionally compatible with a future pinned `m5stack-avatar` adapter and prevents the product expression layer from being buried as one-off drawing logic.

Current architecture split:

- Xiaozhi firmware plus A21 Gateway owns the stock-compatible voice protocol:
  Opus audio, listen/abort, turn cancel, pacing, and playback lifecycle.
- StackChan official avatar/action owns the product screen and body behavior.
  A21 may send OEM semantic events to an official adapter, but must not replace
  the StackChan avatar with an A21 hand-drawn face or Xiaozhi display fork.
- Personality-specific expressions, clock/time behavior, or richer gestures are
  future OEM layers on top of the official avatar/action adapter. They are not
  allowed to mutate the Xiaozhi audio firmware into an A21 visual runtime.

`internal/transport/stackchan` is the host-side frame contract for that official
adapter path. It builds official StackChan WebSocket packets for
`ControlAvatar` (`0x03`), `ControlMotion` (`0x04`), and `DanceSequence`
(`0x14`) from A21 semantic events, using the same `[type][big-endian
length][payload]` shape consumed by upstream `WebSocketAvatar`. This is
compile-time/protocol evidence only until a guarded official StackChan overlay
and physical acceptance prove the runtime path.

Gateway also serves `/stackChan/ws` for the official avatar/action relay and
`POST /v1/stackchan/official/control` for A21 semantic validation commands.
These endpoints are for official StackChan screen/body behavior; they are not
Xiaozhi audio-firmware extensions and must not become a replacement for stock
Xiaozhi voice compatibility.

`firmware-avatar-spike-build` compiles the isolated `a21_stackchan_cores3_avatar_spike` environment with `meganetaaan/M5Stack-Avatar @ 0.10.0`. It is a compatibility spike for a mature avatar engine only. It is not a production firmware lane, not a package source, and not a flashing command.

`firmware-mic-probe-build` compiles only the isolated CoreS3 microphone diagnostic environment. It is for I2S/microphone bring-up evidence and must not be treated as a release package or production firmware unless a later ADR explicitly promotes the path.

`firmware-imu-probe-build` compiles only the isolated CoreS3 IMU diagnostic environment. It is read-only posture telemetry for bring-up evidence and must not be treated as a release package or production firmware.

`firmware-sensor-probe-build` compiles only the isolated CoreS3 sensor diagnostic environment. It uses the mature M5CoreS3 LTR553 implementation for ambient-light/proximity telemetry and StackChan-BSP INA226 battery telemetry. It must not be treated as a release package or production firmware.

`firmware-upload-blocker-check` intentionally invokes PlatformIO's raw upload target and expects it to fail with the A21 blocker message before any hardware write can start. A successful raw upload target is a release-blocking failure.

`firmware-mic-probe-upload-blocker-check` repeats the same raw-upload negative test for `a21_stackchan_cores3_mic_probe`. The diagnostic build may compile for lab evidence, but it must not create an unguarded flashing lane.

`firmware-imu-probe-upload-blocker-check` and `firmware-sensor-probe-upload-blocker-check` repeat the same raw-upload negative test for their diagnostic environments. Passing those checks proves the new hardware tracks cannot bypass the guarded A21 flash lanes.

## Official StackChan Audio Smoke Lane

The A21 speaker bring-up baseline now uses official StackChan/CoreS3 codec code instead of the rejected hand-written playback path.

The lane is intentionally separate from `firmware/stackchan/` release packaging:

```bash
make stackchan-official-baseline
make stackchan-official-audio-smoke-build
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make stackchan-official-audio-smoke-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE \
make stackchan-official-audio-smoke-flash-execute
```

Rules:

- source is exported from the official StackChan Git `HEAD`; local dirty files in that source checkout are not copied into the build;
- when GitHub dependency fetch is unstable, `A21_STACKCHAN_OFFICIAL_DEP_CACHE`
  may point at a local official StackChan dependency cache; the builder exports
  each dependency from its pinned Git `HEAD`, checks the `repos.json` ref, applies
  the official dependency patch, and must not consume dirty X21/V21 working-tree
  files;
- the plan verifies official codec evidence before build: `AudioCodec::OutputData`, `esp_codec_dev_open`, `esp_codec_dev_write`, `CreateDuplexChannels`, and Xiaozhi `AudioService` output-task usage;
- the overlay must be an A21-owned patch under `firmware/stackchan-official/overlays/`;
- the ESP-IDF build output must include `a21-stackchan-official-audio-smoke.bin` at app offset `0x20000` in `flash_args`;
- flash plan is a no-flash receipt with all bootloader, app, partition table, OTA data, and assets hashes;
- execute requires the exact confirmation token above and uses only `python -m esptool ... write_flash @flash_args` from the build directory;
- this is a speaker hardware acceptance lane only, not production A21 firmware, not a provider lane, and not a license to use raw `idf.py flash` or copied X21/V21 esptool commands.

Physical acceptance criterion: the device plays a repeated two-tone pattern, high tone for about 1.2 seconds, short pause, low tone for about 1.2 seconds, then a longer pause. The sound must be continuous and clear enough to distinguish from the previous "telegraph" artifact.

## Official StackChan PCM Bridge Build Lane

The next M3-prep bridge keeps the successful official codec/HAL boundary, but changes the smoke app into a Gateway-driven PCM receiver:

```bash
make stackchan-official-pcm-bridge-build
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL='ws://HOST:21080/ws/audio?device_id=stackchan-001' \
make stackchan-official-pcm-bridge-nvs-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL='ws://HOST:21080/ws/audio?device_id=stackchan-001' \
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS \
make stackchan-official-pcm-bridge-nvs-execute
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL='ws://HOST:21080/ws/audio?device_id=stackchan-001' \
make stackchan-official-pcm-bridge-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL='ws://HOST:21080/ws/audio?device_id=stackchan-001' \
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP \
make stackchan-official-pcm-bridge-flash-execute
```

Rules:

- this lane builds, plans, provisions NVS, and can flash the app only through the reviewed foreground T7 bridge flash guard;
- source is exported from official StackChan Git `HEAD`, then `firmware/stackchan-official/overlays/a21-official-pcm-bridge.patch` is applied;
- the app artifact must be `a21-stackchan-official-pcm-bridge.bin` at app offset `0x20000` in `flash_args`;
- the bridge reads only `a21/device_id` and `a21/audio_ws_url` from NVS; it does not embed Wi-Fi credentials, provider keys, Gateway IPs, proxy URLs, or V21/X21 identity;
- `stackchan-official-pcm-bridge-nvs` is no-write by default and records the NVS partition offset `0x9000`, size `0x4000`, redacted audio websocket fields, and the exact confirmation required for execution;
- `stackchan-official-pcm-bridge-nvs --execute` first passes `a21 gate --scope hardware`, which rejects detached HEAD, background Codex worktrees, dirty trees, and branches outside `codex/a21-hardware-window-*`; only then does it back up the existing NVS partition, parse it with ESP-IDF `nvs_tool.py`, regenerate a partition with `nvs_partition_gen.py`, preserve existing entries including calibration, mutate only `a21/device_id` and `a21/audio_ws_url`, then write only the NVS partition back to `0x9000`;
- every NVS or flash execution receipt must include the `control_guard` evidence object so later review can prove branch, commit, worktree, dirty-state, and tier;
- NVS run artifacts live under `.a21-run/` and are intentionally gitignored because they may contain Wi-Fi or historical device secrets; reports must remain value-redacted;
- the flash-plan receipt validates the bridge artifact and USB serial port, but stores only the audio websocket scheme, host, path, and `device_id` query presence instead of the full URL;
- when `a21/audio_ws_url` is missing, the device must stay on a black A21 status screen and remain silent;
- playback is accepted only as `pcm_s16le`, mono, 16 kHz or 24 kHz, 1-100 ms chunks, queued through the official `AudioCodec::OutputData` path;
- the real-device bridge app flash lane consumes the same explicit USB serial, artifact hash, flash-part hash, control guard, and confirmation-token discipline before any app write is allowed;
- `docs/engineering/adr/0005-official-pcm-bridge-app-flash.md` records the accepted foreground hardware-window gate for `stackchan-official-pcm-bridge-flash --execute`.

This lane exists to move M3 away from the rejected M5Unified `playRaw` path. It is not production firmware and must not bypass the existing A21 release package/flash discipline.

## Official StackChan Xiaozhi-Compatible Product Lane

Physical product StackChan app flashes use the official-compatible lane and app
artifact `a21-stackchan-official-xiaozhi-compatible.bin`. The matching reports
are `reports/a21-stackchan-official-xiaozhi-compatible-flash-*.json`; dry-run
plans use schema `a21.stackchan.official_xiaozhi_compatible_flash_plan.v1`,
and executed receipts use
`a21.stackchan.official_xiaozhi_compatible_flash_execution.v1` with
`status=passed`, `dry_run=false`, and `flash_executed=true`.

Doctor and office-preflight may summarize these receipts as
`product_lane_artifact_evidence` with schema
`a21.firmware.product_lane_artifact_evidence.v1`. This is metadata evidence
only. It never executes a build or flash command, never authorizes a product
flash, and never converts the generic Xiaozhi `xiaozhi.bin` app into a product
StackChan artifact. Newer dry-run plans must not override an older executed
official-compatible flash receipt.

When a real microphone bring-up window is available, mic-probe flashing uses its own explicit diagnostic lane:

```bash
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make firmware-mic-probe-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_MIC_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE \
make firmware-mic-probe-flash-execute
```

The plan and execute commands rebuild `a21_stackchan_cores3_mic_probe`, require a clean source tree, verify the build output lives under `.pio/build/a21_stackchan_cores3_mic_probe/firmware.bin`, require the embedded A21 firmware identity and git commit, require the `diagnostic_probe_m5unified_i2s_capture` marker, reject busy or non-USB serial ports, and write timestamped `reports/a21-firmware-mic-probe-flash-*.json` receipts. This lane is for microphone diagnosis only; it does not promote microphone capability to production `available`.

The IMU diagnostic lane mirrors that discipline with its own build, marker, confirmation token, and reports:

```bash
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make firmware-imu-probe-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_IMU_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE \
make firmware-imu-probe-flash-execute
```

The IMU plan and execute commands rebuild `a21_stackchan_cores3_imu_probe`, require a clean source tree, verify the build output lives under `.pio/build/a21_stackchan_cores3_imu_probe/firmware.bin`, require the embedded A21 firmware identity and git commit, require the `diagnostic_probe_m5unified_imu` marker, reject busy or non-USB serial ports, and write timestamped `reports/a21-firmware-imu-probe-flash-*.json` receipts. This lane is read-only diagnostic telemetry only; it does not promote IMU gestures, posture behavior, or production `available`.

The sensor diagnostic lane mirrors the same rules and requires all three sensor markers before any write:

```bash
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX make firmware-sensor-probe-flash-plan
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_SENSOR_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE \
make firmware-sensor-probe-flash-execute
```

The sensor plan and execute commands rebuild `a21_stackchan_cores3_sensor_probe`, require a clean source tree, verify the build output lives under `.pio/build/a21_stackchan_cores3_sensor_probe/firmware.bin`, require the embedded A21 firmware identity and git commit, require `diagnostic_probe_ltr553_ambient_light`, `diagnostic_probe_ltr553_proximity`, and `diagnostic_probe_ina226_battery` markers, reject busy or non-USB serial ports, and write timestamped `reports/a21-firmware-sensor-probe-flash-*.json` receipts. This lane is read-only diagnostic telemetry only; it does not promote environmental sensors, battery telemetry, or production `available`.

After a mic-probe flash and a live `audio_probe_only` Gateway validation window, freeze the evidence with:

```bash
A21_DEVICE_ID=stackchan-001 \
A21_MIC_PROBE_WINDOW_MS=5000 \
A21_MIC_PROBE_MIN_FRAMES=90 \
A21_MIC_PROBE_MIN_DELIVERY_RATIO=0.95 \
make stackchan-mic-probe-acceptance
```

For stricter live checks, override the thresholds:

```bash
A21_DEVICE_ID=stackchan-001 \
A21_MIC_PROBE_MIN_FRAMES=180 \
A21_MIC_PROBE_MIN_ABS_PEAK=100 \
A21_MIC_PROBE_MIN_NONZERO_SAMPLES=300 \
A21_MIC_PROBE_MIN_GATEWAY_RMS=0.001 \
A21_MIC_PROBE_MIN_VAD_SPEECH=1 \
A21_MIC_PROBE_MIN_DELIVERY_RATIO=0.95 \
A21_MIC_PROBE_WINDOW_MS=10000 \
make stackchan-mic-probe-acceptance
```

This writes `reports/a21-stackchan-mic-probe-acceptance-YYYYMMDD-HHMMSS.json` with `hardware_acceptance_scope=diagnostic_microphone_only`, `production_capability_promoted=false`, firmware identity, runtime mic counters, latest sample evidence, Gateway metrics, capture/send/ingress rates, and delivery ratios. With `A21_MIC_PROBE_WINDOW_MS>0`, the command first snapshots Gateway/device state, sends a `LISTENING` control event with `audio_probe_only=true`, waits the requested window, snapshots again, sends `IDLE`, and validates deltas. That avoids false failures from cumulative Gateway counters that existed before the probe window.

The gate requires the device to report microphone capability `diagnostic_probe_m5unified_i2s_capture`, captured/sent frame deltas above threshold, microphone-to-audio-WS and audio-WS-to-Gateway delivery ratios above threshold, zero new driver errors, zero new queue drops, non-zero sample evidence, Gateway audio ingress deltas, no new playback chunks, and optional VAD speech deltas. A passing report means the isolated M5Unified/CoreS3 diagnostic microphone path captured and uplinked real PCM frames during this session. It still does not prove production microphone availability, speaker output, AEC, full-duplex, provider latency, or the release firmware path.

After an IMU-probe flash, freeze read-only posture telemetry evidence with:

```bash
A21_DEVICE_ID=stackchan-001 \
A21_IMU_PROBE_WINDOW_MS=1500 \
A21_IMU_PROBE_MIN_SAMPLES=10 \
A21_IMU_PROBE_MIN_ACCEL_TOTAL_MG=500 \
A21_IMU_PROBE_MAX_READ_ERRORS=0 \
make stackchan-imu-probe-acceptance
```

This writes `reports/a21-stackchan-imu-probe-acceptance-YYYYMMDD-HHMMSS.json` with `hardware_acceptance_scope=diagnostic_imu_only`, `production_capability_promoted=false`, firmware identity, capability status, runtime IMU counters, acceleration, gyro, posture, sample-rate evidence, and sample/read-error deltas. A passing report means the isolated M5Unified/CoreS3 IMU diagnostic path is alive and useful as telemetry; it still does not approve product gestures, motion reactions, privacy behavior, or release-firmware IMU promotion.

For read-only ambient/proximity/battery telemetry evidence, use:

```bash
A21_DEVICE_ID=stackchan-001 \
A21_SENSOR_PROBE_WINDOW_MS=1500 \
A21_SENSOR_PROBE_MIN_SAMPLES=10 \
A21_SENSOR_PROBE_MIN_BATTERY_MV=3000 \
A21_SENSOR_PROBE_MAX_READ_ERRORS=0 \
make stackchan-sensor-probe-acceptance
```

This writes `reports/a21-stackchan-sensor-probe-acceptance-YYYYMMDD-HHMMSS.json` with `hardware_acceptance_scope=diagnostic_sensor_only`, `production_capability_promoted=false`, firmware identity, ambient/proximity/battery diagnostic capability statuses, runtime sensor counters, LTR553 ambient/proximity values, INA226 battery voltage/current values, sample-rate evidence, and sample/read-error deltas. A passing report means the isolated sensor diagnostic path is alive and useful as telemetry; it still does not approve adaptive brightness, presence behavior, power-state UI, or release-firmware sensor promotion.

For the first real-device mic-to-speaker loop, use:

```bash
A21_DEVICE_ID=stackchan-001 \
A21_HALF_DUPLEX_WINDOW_MS=1500 \
A21_HALF_DUPLEX_MIN_MIC_FRAMES=1 \
A21_HALF_DUPLEX_MIN_PLAYBACK_CHUNKS=1 \
A21_HALF_DUPLEX_MIN_DELIVERY_RATIO=0.95 \
make stackchan-half-duplex-acceptance
```

This writes `reports/a21-stackchan-half-duplex-acceptance-YYYYMMDD-HHMMSS.json`. The command snapshots Gateway/device state and Gateway metrics, sends `LISTENING` without `audio_probe_only`, arms exactly one `mock_playback_on_next_audio_frame`, waits for live microphone frames to trigger Gateway mock downlink, checks microphone capture/send deltas, Gateway ingress/playback deltas, firmware playback-buffer deltas, and speaker-pump deltas, then clears back to `IDLE`. A passing report confirms the connected StackChan can drive a minimal half-duplex A21 loop through Gateway mock playback instrumentation. It still records `physical_sound_observed=false` and does not claim production ASR, LLM, TTS, AEC, full-duplex, or human-accepted audio quality.

For instrumented speaker/downlink evidence, use:

```bash
A21_DEVICE_ID=stackchan-001 \
A21_SPEAKER_WINDOW_MS=1500 \
A21_SPEAKER_MOCK_AUDIO_CHUNKS=50 \
A21_SPEAKER_MIN_PLAYED_FRAMES=50 \
make stackchan-speaker-acceptance
```

This writes `reports/a21-stackchan-speaker-acceptance-YYYYMMDD-HHMMSS.json`. The command snapshots Gateway/device state and Gateway playback metrics, sends about 1000 ms of non-silent A21 `audio.playback.chunk` frames as bounded `SPEAKING` batches of at most four chunks per `/v1/devices/control` request, waits 1500 ms by default so the last speaker-pump frame and runtime echo have margin, checks playback-buffer and speaker-pump deltas, and then clears back to `IDLE`. A passing report confirms the commanded stream moved through Gateway downlink, firmware buffer, and speaker-pump instrumentation. It intentionally records `physical_sound_observed=false`; use later operator or instrument evidence before claiming physical audibility or product-quality TTS.

`firmware-check --kind current-artifact` validates the newest packaged artifact for the current git commit by reading `a21-firmware-release-index.jsonl`, selecting the latest matching package, and re-running the artifact, release-index, and per-artifact manifest guards. It is part of `make release-check`, so a package step is not considered release-clean until the generated candidate can be independently re-read from the release ledger.

The release ledger is path-bound as well as checksum-bound. A release-index entry and the per-artifact manifest must point back to the same artifact path and checksum path being checked, and current-artifact selection cannot jump from `firmware/artifacts` to an external directory that happens to contain a same-named A21 binary. This prevents old, copied, or hand-assembled packages from being spliced into a current A21 build receipt.

`firmware-artifact-prune-plan` writes a no-delete retention receipt after the current artifact guard. It requires the same release-ledger current artifact, keeps that package plus the configured recent release-ledger-valid packages, lists older release-ledger-valid packages as prune candidates, and lists loose or invalid files for manual review. It always sets `dry_run=true` and `delete_allowed=false`; it is planning evidence, not a deletion command. When `--output-dir` is set, stdout stays summary-only and the full plan goes into the JSON report so release logs are not flooded by old package paths.

`office-handoff` writes the home-to-office handoff manifest. It requires the current release-ledger artifact, embeds the artifact-retention summary, records serial inventory, and lists the next office-only physical acceptance actions. It sets `flash_allowed=false` and `delete_allowed=false`; it is not a substitute for `office-preflight`.

Current native firmware tests cover:

- firmware build identity fields, firmware label construction, and build identity propagation in outgoing device events
- A21 `control.event` parsing
- protocol/device identity rejection
- control event to local render-state transitions
- unsupported state fallback to firmware error state
- A21 Gateway host/port/path validation
- control/audio WebSocket URL construction
- rejection of legacy Gateway ports and X21/V21 route names
- hardware-free connection lifecycle from Wi-Fi connecting to Gateway connected
- reconnect wait timing after Gateway loss
- local fallback on invalid Gateway config
- Wi-Fi config defaults, password redaction, legacy SSID rejection, and local fallback when credentials are missing
- Wi-Fi runtime begin-once behavior, connected transition, and disconnect retry behavior
- Gateway control WebSocket begin-once behavior, control-event parsing, and disconnect retry behavior
- Gateway `device.event` send gating, deterministic seq/trace IDs, interrupt/mock-turn envelope construction, and runtime echo change detection for applied screen/motion/RGB plus microphone and playback/speaker diagnostics
- audio WebSocket begin gating after Gateway connection, full 640-byte mock `audio.frame` envelope construction, queued mic PCM `audio.frame` uplink construction, ack control-event parsing, mock `audio.playback.chunk` downlink parsing, full 20 ms 16 kHz PCM base64 payload capacity, fixed 640-byte PCM encode/decode, bounded playback buffering, M5 speaker pump queue gating, and send rejection while disconnected
- microphone capture policy and four-frame uplink queue that record one PCM16 frame only when the render state is capture-safe and the speaker queue is idle in hardware-free tests; physical CoreS3 release microphone capture is currently disabled by a crash guard, while `a21_stackchan_cores3_mic_probe` exposes the isolated diagnostic status `diagnostic_probe_m5unified_i2s_capture`
- semantic touch intent runtime for `wake_or_listen` and `barge_in`, preserving `screen` vs `top_sensor` source metadata through Gateway `device.event` envelopes
- playback state-machine behavior for starting a speaking stream once, replacing streams with stop/clear, clearing buffered chunks, and stopping plus clearing immediately on barge-in
- StackChan Y-axis servo clamp to 5 to 85 degrees
- semantic render-state to Y-axis motion target mapping, plus write-on-change motion runtime behavior through a driver interface
- semantic render-state to RGB color mapping, plus write-on-change RGB runtime behavior through a driver interface

Current Gateway tests cover:

- `/v1/devices` registration from firmware-originated `device.event` payloads
- A21 firmware identity validation for `a21-stackchan`, semver version, `m5stack-cores3`, and git SHA commit
- StackChan capability visibility for microphone, speaker, screen, screen touch, top touch, Y-axis servo, and RGB
- rejection and metrics for forbidden X21/V21 firmware identity

`make firmware-package` copies PlatformIO's generic `firmware.bin` into `firmware/artifacts/` with an A21-specific filename:

```text
a21-stackchan-<version>-m5stack-cores3-<git-sha>-<YYYYMMDD-HHMMSS>.bin
```

It also writes a sibling `.sha256` file and a per-artifact release manifest:

```text
a21-stackchan-<version>-m5stack-cores3-<git-sha>-<YYYYMMDD-HHMMSS>.bin.manifest.json
```

Only packaged artifacts should be considered candidates for future upload.

`make firmware-package` and `a21 firmware-package` both run a clean-worktree guard. This prevents a binary built from uncommitted sources from being packaged under a misleading git commit, even if someone bypasses Makefile and calls the CLI directly.

`firmware-package` also appends a machine-readable JSONL release record to:

```text
firmware/artifacts/a21-firmware-release-index.jsonl
```

Each line records the A21 firmware ID, version, board, git commit, build timestamp, artifact path, checksum path, SHA-256, and build provenance. The index is a traceability ledger, not flash permission. It makes wrong-package investigations concrete: the candidate binary must be explainable by an A21 package record rather than by a loose PlatformIO `firmware.bin`.

The per-artifact `.manifest.json` records the same identity, checksum, and build provenance beside the binary. Upload-path guards require it so the release candidate is self-describing even before consulting the directory-level index.

Build provenance must say:

- `build_system`: `platformio`
- `platformio_env`: `a21_stackchan_cores3`
- `platformio_board`: `m5stack-cores3`
- `source_path`: the PlatformIO source binary under `firmware/stackchan/.pio/build/a21_stackchan_cores3/firmware.bin`
- `source_name`: `firmware.bin`

Packaging rejects input outside the A21 StackChan firmware lane and rejects input or output paths containing forbidden X21/V21 identities. A21 release candidates cannot be produced from a lookalike `.pio` build folder or written into legacy artifact directories.

Packaging also validates the source PlatformIO `firmware.bin` before copying it. The source binary must already embed the expected A21 firmware ID, version, board, and git commit. This prevents a stale or wrong-board build from being wrapped in a correct-looking A21 artifact name.

Upload-path dry-run guards now require both the release index record and the sibling artifact manifest. `firmware-check --kind upload`, `firmware-check --kind device`, and `firmware-flash-plan` also reject manifest, artifact, and device-report input paths containing forbidden X21/V21 identity before opening those paths, and they do not echo the polluted path back to the operator. A hand-assembled `.bin + .sha256` pair may still be inspected with `firmware-check --kind artifact`, but it cannot pass upload or flash-planning gates unless:

- the same directory's release index contains a matching firmware ID, version, board, commit, timestamp, artifact filename, checksum filename, SHA-256, and build provenance
- the sibling `.manifest.json` contains matching A21 project, firmware ID, version, board, commit, timestamp, artifact filename, checksum filename, SHA-256, and build provenance
- the release index entry and sibling manifest resolve to the same checked artifact path and checksum path
- the release index and sibling manifest full artifact/checksum paths do not contain forbidden X21/V21 identities

`firmware-check --kind upload` also rejects older packages when the release index contains a newer artifact for the same firmware ID, version, board, and commit. This keeps the dry-run path aligned with the latest A21 package for that exact checkout and avoids choosing a stale same-commit binary by accident.

## Artifact Retention Plan

Artifact buildup is operational risk because a human may pick a stale binary under pressure. A21 therefore has a dry-run retention plan:

```bash
go run ./cmd/a21 firmware-artifact-prune-plan \
  --commit <git-sha> \
  --artifact-dir firmware/artifacts \
  --keep-recent 5 \
  --output-dir reports

make firmware-artifact-prune-plan
```

The command writes:

```text
reports/a21-firmware-artifact-prune-plan-YYYYMMDD-HHMMSS.json
```

The receipt contains:

- `delete_allowed: false`
- the release-ledger-validated current artifact
- keep records for current and recent release-ledger-valid packages
- prune candidates for older release-ledger-valid packages
- manual-review records for loose, incomplete, or invalid artifact files

When `--output-dir reports` is used, stdout shows only summary counts plus `report_path`; inspect the report for the full file list. The command rejects artifact directories containing forbidden X21/V21 identity. It never removes files. Any future deletion command must be a separate explicit guarded command and must not reinterpret this dry-run receipt as permission to delete or flash.

## Artifact Guard

Every packaged firmware binary must pass:

```bash
go run ./cmd/a21 firmware-check --kind artifact --artifact firmware/artifacts/<a21-stackchan...bin>
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> make firmware-artifact-check
```

For the current checkout, prefer the ledger-backed command:

```bash
go run ./cmd/a21 firmware-check --kind current-artifact --commit <git-sha>
make firmware-current-artifact-check
```

The guard verifies:

- manifest identity is `A21`
- firmware ID is `a21-stackchan`
- PlatformIO environment uses an `a21_` prefix
- board is `m5stack-cores3`
- artifact filename matches manifest prefix, version, board, git commit, and timestamp
- sibling `.sha256` exists and matches the binary
- binary content embeds the same A21 firmware ID, version, board, and git commit
- artifact filename does not contain forbidden X21/V21 identities
- upload-path guards require a matching `a21-firmware-release-index.jsonl` entry in the artifact directory
- upload-path guards require a matching sibling `.manifest.json` artifact manifest
- upload-path guards require both release records to agree on PlatformIO build provenance
- upload-path guards reject release ledger or manifest paths containing forbidden X21/V21 identities, even when the artifact filename itself looks correct

The embedded-identity check matters because a wrong `firmware.bin` could otherwise be copied into a correctly named artifact with a matching checksum. Build-provenance checks close the next gap: a package is not a valid A21 candidate unless the filename, checksum, artifact manifest, release index, firmware manifest, binary identity, PlatformIO environment, and PlatformIO board all agree.

## Upload Guard

Phase 5B adds an upload dry-run guard, not an upload command:

```bash
go run ./cmd/a21 firmware-check --kind upload \
  --artifact firmware/artifacts/<a21-stackchan...bin> \
  --port /dev/cu.usbmodemXXXX \
  --commit <expected-git-sha>
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
make firmware-upload-check
```

This command verifies the artifact guard and rejects ambiguous upload targets such as `auto`, `default`, `any`, non-`/dev/` paths, and non-serial `/dev/*` paths such as `/dev/null`. Accepted upload paths are deliberately narrow: macOS `cu.*`/`tty.*` names must contain `usbmodem` or `usbserial`, and Linux names must be `ttyUSB*` or `ttyACM*`. Bluetooth, debug-console, and other generic serial-looking ports are rejected before process-ownership checks. It also checks whether the selected serial path is already held by another process. It still does not flash. Actual flashing must only be introduced later as a separate guarded command after physical device identity checks are in place.

Successful `firmware-check --kind upload` output is intentionally a dry-run receipt. The JSON must include:

- `guard_id: a21.firmware.upload_guard.v1`
- `dry_run: true`
- `flash_allowed: false`
- `next_required_confirmation: physical_device_identity`
- `port_usage`

Any future real flashing command must not reinterpret this receipt as permission to flash. It is evidence that the candidate passed preflight checks, and nothing more.

The `--commit` value must match the git sha encoded in the artifact filename. The Makefile wrapper fills it from `git rev-parse --short=12 HEAD`, so an old firmware package cannot pass the dry-run guard for a newer checkout.

If multiple A21 packages exist for the same commit, the upload dry-run accepts only the newest timestamp recorded in `a21-firmware-release-index.jsonl`.

## Device Identity Guard

Phase 5C adds a physical-device identity dry-run guard. It still does not flash. It validates a Gateway `/v1/devices` JSON capture against the exact packaged firmware candidate. Capture the report through the A21 toolchain rather than a hand-written `curl` redirect:

```bash
go run ./cmd/a21 firmware-device-report \
  --gateway-url http://127.0.0.1:21080 \
  --output-dir reports

go run ./cmd/a21 firmware-check --kind device \
  --artifact firmware/artifacts/<a21-stackchan...bin> \
  --device-report reports/a21-devices-<timestamp>.json \
  --device-id stackchan-001 \
  --commit <expected-git-sha> \
  --max-device-age-ms 300000
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_DEVICE_REPORT=reports/a21-devices-<timestamp>.json \
A21_DEVICE_ID=stackchan-001 \
make firmware-device-check
```

`firmware-check --kind device` requires `--max-device-age-ms`; `make firmware-device-report` writes `reports/a21-devices-YYYYMMDD-HHMMSS.json` and uses direct Gateway HTTP without ambient proxy inheritance. It rejects Gateway URLs that target known X21/V21 legacy ports, requires the Gateway response to declare `schema_version=a21.gateway.devices.v1` and `service=a21-gateway`, and rejects Gateway device identity fields that contain forbidden X21/V21 naming before writing the report. `make firmware-device-check` passes `A21_DEVICE_MAX_AGE_MS=300000` by default. Override that value only for an explicitly documented lab reason; physical acceptance should use a freshly captured Gateway `/v1/devices` report.

The captured report preserves Gateway operator fields such as `connection_status`, `device_age_ms`, `current_mode`, `current_expression`, and `playback_stream_id`. These fields help prove what the office operator was looking at during acceptance, but the guard still uses explicit identity, artifact, commit, and freshness checks rather than trusting display state alone. `firmware-check --kind device` requires the report to carry A21 Gateway identity, either as the raw Gateway response fields `schema_version=a21.gateway.devices.v1` and `service=a21-gateway`, or as the `gateway_schema_version` / `gateway_service` fields written by `firmware-device-report`. A naked hand-written `{"devices":[...]}` file is not valid acceptance evidence. It also requires `connection_status=online`; stale or unknown Gateway state is a hard stop.

The guard verifies:

- the artifact still passes `firmware-check --kind artifact`
- the artifact commit matches the expected git commit
- the device report contains the explicit `A21_DEVICE_ID`
- `identity_status` is `ok`
- reported firmware ID is `a21-stackchan`
- reported version, board, and commit match the artifact manifest and filename
- `last_seen_ms` is present and recent enough under the required `--max-device-age-ms`
- device ID and firmware identity contain no forbidden X21/V21 names

Successful output includes:

- `guard_id: a21.firmware.device_identity_guard.v1`
- `device_identity_confirmed: true`
- `flash_allowed: false`
- `next_required_confirmation: explicit_guarded_flash_command`

This receipt is a stronger identity confirmation than `firmware-check --kind upload`, but it is still not permission to flash. It proves that Gateway has seen a StackChan-like A21 device identity matching the candidate artifact. A future real flashing command must require both the upload dry-run receipt and this device identity receipt, then perform its own final confirmation.

## Flash Plan Guard

Phase 5D adds a combined flash-plan guard. It still does not flash. It composes the existing artifact, upload-port, and device-identity guards into one receipt so a future real flashing command cannot accidentally combine an old artifact, a new checkout, the wrong serial port, or the wrong device report:

```bash
go run ./cmd/a21 firmware-flash-plan \
  --artifact firmware/artifacts/<a21-stackchan...bin> \
  --port /dev/cu.usbmodemXXXX \
  --device-report reports/a21-devices.json \
  --device-id stackchan-001 \
  --commit <expected-git-sha> \
  --max-device-age-ms 300000 \
  --output-dir reports
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_DEVICE_REPORT=reports/a21-devices.json \
A21_DEVICE_ID=stackchan-001 \
make firmware-flash-plan
```

`firmware-flash-plan` requires `--max-device-age-ms`; the Makefile wrapper passes the same `A21_DEVICE_MAX_AGE_MS` freshness guard as `firmware-check --kind device` and writes a timestamped no-flash receipt to:

```text
reports/a21-firmware-flash-plan-YYYYMMDD-HHMMSS.json
```

This makes flash planning auditable instead of ephemeral stdout. The receipt includes `generated_at_ms` and `report_path`, but it still sets `flash_allowed: false`.

The guard verifies:

- the artifact passes `firmware-check --kind artifact`
- the upload port is explicit, exists, and is not busy
- the Gateway device report contains the explicit `A21_DEVICE_ID`
- the Gateway device report has `connection_status=online`
- the Gateway device report has `last_seen_ms` and is fresh under the required `--max-device-age-ms`
- the Gateway device report does not show active playback or `current_expression=speaking`
- the Gateway device report does not show unsafe runtime states: `current_expression=thinking|professional|error` or `current_mode=professional|local_fallback|error`
- the upload guard and device-identity guard reference the same artifact checksum and commit
- every identity remains in the A21 namespace

Successful output includes:

- `guard_id: a21.firmware.flash_plan_guard.v1`
- `generated_at_ms`
- `dry_run: true`
- `flash_allowed: false`
- `next_required_confirmation: future_explicit_guarded_flash_command`
- `report_path` when `--output-dir` is provided

This receipt is the strongest no-flash receipt in the current repository. It is a precondition design for future guarded flashing, not permission to flash.

## Current Flashing Status

Phase 5E introduces the first explicit guarded bootstrap flash path for a real StackChan/CoreS3 unit that cannot yet report an A21 firmware identity to Gateway. This is intentionally separate from `firmware-flash-plan`: bootstrap flashing is only for initial bring-up or recovery, while `firmware-flash-plan` remains the stronger steady-state path after Gateway has seen a matching A21 device.

Bootstrap flash planning is still a dry-run receipt:

```bash
go run ./cmd/a21 firmware-bootstrap-flash \
  --artifact firmware/artifacts/<a21-stackchan...bin> \
  --port /dev/cu.usbmodemXXXX \
  --commit <expected-git-sha> \
  --output-dir reports
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
make firmware-bootstrap-flash-plan
```

The receipt writes:

```text
reports/a21-firmware-bootstrap-flash-plan-YYYYMMDD-HHMMSS.json
```

It verifies the packaged A21 artifact, latest same-commit release-ledger selection, explicit non-busy serial port, and the exact ESP32-S3 image parts to write:

- `0x0000` bootloader from `firmware/stackchan/.pio/build/a21_stackchan_cores3/bootloader.bin`
- `0x8000` partitions from `firmware/stackchan/.pio/build/a21_stackchan_cores3/partitions.bin`
- `0xe000` `boot_app0.bin` from the repository-local `.a21-tools/platformio-core`
- `0x10000` the packaged `a21-stackchan-...bin` artifact

Successful plan output includes:

- `guard_id: a21.firmware.bootstrap_flash_plan.v1`
- `dry_run: true`
- `flash_allowed: false`
- `next_required_confirmation: firmware-bootstrap-flash-execute_with_confirmation_token`
- SHA-256 for every image part

The only real write command is:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_BOOTSTRAP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_FIRMWARE \
make firmware-bootstrap-flash-execute
```

or the equivalent CLI command:

```bash
go run ./cmd/a21 firmware-bootstrap-flash --execute \
  --artifact firmware/artifacts/<a21-stackchan...bin> \
  --port /dev/cu.usbmodemXXXX \
  --commit <expected-git-sha> \
  --confirm WRITE_A21_STACKCHAN_FIRMWARE \
  --output-dir reports
```

The execute command rebuilds the same plan immediately before flashing, requires the confirmation token, writes through the repository-local esptool, and records:

```text
reports/a21-firmware-bootstrap-flash-execution-YYYYMMDD-HHMMSS.json
```

Raw PlatformIO upload remains forbidden. Bootstrap flashing is not permission to use `pio run -t upload`, `uploadfs`, `uploadota`, copied X21/V21 esptool snippets, or generic `firmware.bin` paths.

A21 firmware work must continue to use the repository-local `.a21-tools/` PlatformIO environment and `firmware/artifacts/a21-stackchan-...` packages. Do not point A21 upload checks at X21/V21 build directories, generic `firmware.bin` paths, or auto-selected serial ports.

Before any future firmware upload:

1. Confirm physical device identity.
2. Confirm serial/upload port.
3. Confirm `firmware/stackchan/a21-firmware.json`.
4. Confirm git commit and version.
5. Confirm generated artifact filename begins with `a21-stackchan-`.
6. Run `firmware-check --kind artifact`.
7. Run `firmware-check --kind upload`.
8. Capture `/v1/devices` from the A21 Gateway and run `firmware-check --kind device`.
9. Run `firmware-flash-plan`.
10. If the device already reports A21 identity, prefer `firmware-flash-plan` and keep bootstrap flashing out of the path.
11. If this is initial bring-up or recovery and no A21 identity can be captured, run `firmware-bootstrap-flash`.
12. Only after the bootstrap plan passes may `firmware-bootstrap-flash --execute` run with `WRITE_A21_STACKCHAN_FIRMWARE`.

## Office Preflight

The home/travel handoff command is:

```bash
make office-handoff
```

It writes:

```text
reports/a21-office-handoff-YYYYMMDD-HHMMSS.json
```

This manifest proves which current artifact and retention state will be carried into the office. It does not contact Gateway, does not require StackChan to be connected, and does not authorize flashing. It exists so the operator does not reconstruct travel state from scattered reports.

Phase 5J adds a no-flash office preflight receipt for the Shanghai handoff path:

```bash
A21_DEVICE_ID=stackchan-001 \
make office-preflight
```

or:

```bash
go run ./cmd/a21 office-preflight \
  --gateway-url http://127.0.0.1:21080 \
  --device-id stackchan-001 \
  --commit <expected-git-sha> \
  --max-device-age-ms 300000 \
  --output-dir reports
```

The command composes:

- newest release-indexed firmware artifact for the expected git commit
- direct no-ambient-proxy Gateway `/v1/devices` capture
- required Gateway identity: `schema_version=a21.gateway.devices.v1`, `service=a21-gateway`
- device identity check with `connection_status=online` and `last_seen_ms` freshness
- device quiescence using the same runtime-state rule as `firmware-flash-plan`
- serial inventory and available USB serial candidates
- proxy/fingerprint metadata for the current network

The preflight rejects manifest and artifact input paths that contain forbidden X21/V21 identity before reading them. It also refuses `ready_for_flash_plan=true` while the device is speaking, has active playback, is thinking, is in professional mode, or is in error/local fallback. It writes `reports/a21-office-preflight-YYYYMMDD-HHMMSS.json` plus a paired `reports/a21-devices-YYYYMMDD-HHMMSS.json`. It sets `dry_run=true` and `flash_allowed=false`. A passing office preflight only means the operator has enough evidence to run `firmware-flash-plan` with an explicit USB serial path; it is still not permission to flash.

If the output reports directory already contains official-compatible
product-lane evidence, office-preflight includes
`product_lane_artifact_evidence`. When a release-ledger artifact is still
missing, this remains an annotation and the report stays
`ready_for_flash_plan=false` with
`official_product_lane_artifact_evidence_present`; a prior executed app flash is
not treated as permission or as a new generic flash-plan artifact.

## Office Acceptance Gate

After the office has a handoff receipt and a ready office-preflight receipt, run:

```bash
A21_HANDOFF_REPORT=reports/a21-office-handoff-YYYYMMDD-HHMMSS.json \
A21_OFFICE_PREFLIGHT_REPORT=reports/a21-office-preflight-YYYYMMDD-HHMMSS.json \
make office-acceptance
```

If a no-flash flash-plan receipt exists, include it:

```bash
A21_FIRMWARE_FLASH_PLAN=reports/a21-firmware-flash-plan-YYYYMMDD-HHMMSS.json
```

The command writes:

```text
reports/a21-office-acceptance-YYYYMMDD-HHMMSS.json
```

The gate cross-checks:

- handoff schema is `a21.office_handoff.v1`
- office-preflight schema is `a21.office_preflight.v1`
- all receipts keep `flash_allowed=false`
- handoff keeps `delete_allowed=false`
- office-preflight has `ready_for_flash_plan=true`
- commit and artifact path match across reports
- artifact SHA matches when both receipts provide it
- optional flash-plan report references the same commit, artifact, and device

The success status is `ready_for_physical_acceptance`, not "flashed" or "launched". This preserves the difference between software evidence and physical StackChan acceptance.

## StackChan Identity Acceptance

When the StackChan is physically present and the office acceptance receipt is ready, run a fresh identity-only acceptance gate:

```bash
A21_OFFICE_ACCEPTANCE_REPORT=reports/a21-office-acceptance-YYYYMMDD-HHMMSS.json \
A21_DEVICE_ID=stackchan-001 \
make stackchan-identity-acceptance
```

or:

```bash
go run ./cmd/a21 stackchan-identity-acceptance \
  --office-acceptance reports/a21-office-acceptance-YYYYMMDD-HHMMSS.json \
  --gateway-url http://127.0.0.1:21080 \
  --device-id stackchan-001 \
  --commit <expected-git-sha> \
  --max-device-age-ms 300000 \
  --output-dir reports
```

The command writes:

```text
reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json
```

It reuses the release-ledger artifact guard, takes a fresh direct Gateway `/v1/devices` capture, validates A21 firmware identity and freshness, checks device quiescence, and records serial inventory plus USB serial candidates. The report uses `hardware_acceptance_scope=identity_only` and `identity_acceptance_status=identity_confirmed` only when all checks pass. It still sets `flash_allowed=false` and does not claim microphone, speaker, screen, servo, RGB, OTA, latency, or flashing acceptance.

## StackChan Capability Acceptance

After identity-only acceptance passes, generate a standard physical evidence template:

```bash
A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT=reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
A21_DEVICE_ID=stackchan-001 \
make stackchan-physical-evidence
```

or, when the operator already has physical observations, use explicit `--pass capability=evidence_type` entries:

```bash
go run ./cmd/a21 stackchan-physical-evidence \
  --identity-acceptance reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
  --device-id stackchan-001 \
  --commit <expected-git-sha> \
  --pass microphone=gateway_audio_frame \
  --pass speaker=audible_playback \
  --pass screen=operator_visible_state \
  --pass screen_touch=touch_event \
  --pass top_touch=touch_event \
  --pass servo_y=servo_clamped_motion \
  --pass rgb=operator_visible_state \
  --output-dir reports
```

This command writes:

```text
reports/a21-stackchan-physical-evidence-YYYYMMDD-HHMMSS.json
```

To derive Gateway-observable observations from A21 runtime state and traces:

```bash
A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT=reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
A21_DEVICE_ID=stackchan-001 \
A21_DERIVE_GATEWAY_EVIDENCE=1 \
make stackchan-physical-evidence
```

Gateway-derived evidence is intentionally limited:

- `microphone`: latest trace includes `audio.frame.received`
- `speaker`: latest trace includes `audio.playback.chunk.sent`
- `screen`: device registry has `runtime_echo.screen`; if absent, it can fall back to current Gateway mode/expression
- `screen_touch`: latest touch event source is `screen`
- `top_touch`: latest touch event source is `top_sensor`
- `servo_y`: device registry has `runtime_echo.servo_y`
- `rgb`: device registry has `runtime_echo.rgb`

`runtime_echo` is emitted by firmware after applying screen, motion, and RGB state, so it is a device-applied echo rather than only Gateway intent. It still does not prove physical audibility, screen visibility, servo movement, RGB output, OTA, AEC, or full-duplex quality. Those still require explicit physical observations or later instrumented hardware probes.

Capability acceptance then consumes that evidence file:

```bash
A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT=reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT=reports/a21-stackchan-physical-evidence-YYYYMMDD-HHMMSS.json \
A21_DEVICE_ID=stackchan-001 \
make stackchan-capability-acceptance
```

or:

```bash
go run ./cmd/a21 stackchan-capability-acceptance \
  --identity-acceptance reports/a21-stackchan-identity-acceptance-YYYYMMDD-HHMMSS.json \
  --evidence reports/a21-stackchan-physical-evidence.json \
  --device-id stackchan-001 \
  --commit <expected-git-sha> \
  --output-dir reports
```

The command writes:

```text
reports/a21-stackchan-capability-acceptance-YYYYMMDD-HHMMSS.json
```

The required physical evidence schema is:

```json
{
  "schema_version": "a21.stackchan_physical_evidence.v1",
  "device_id": "stackchan-001",
  "commit": "<expected-git-sha>",
  "artifact_sha256": "<identity-acceptance-artifact-sha256>",
  "observations": [
    {
      "capability": "microphone",
      "status": "passed",
      "evidence_type": "gateway_audio_frame",
      "observed_at_ms": 1780000001000
    }
  ]
}
```

Required capabilities are:

- `microphone`
- `speaker`
- `screen`
- `screen_touch`
- `top_touch`
- `servo_y`
- `rgb`

For each required capability, the fresh identity acceptance report must declare it as `available`, and the physical evidence report must include a `passed` observation with non-empty `evidence_type` and `observed_at_ms`. Missing or failed capability evidence produces `capability_acceptance_status=blocked`.

This gate deliberately separates declaration from acceptance. Gateway `capabilities` prove what the firmware says the device surface is; `stackchan-capability-acceptance` proves that an operator or later hardware probe produced per-capability evidence. The report still sets `flash_allowed=false` and does not claim OTA, latency, AEC, full-duplex quality, or future flashing permission.

## Serial Inventory

Before choosing an upload port, inspect the current serial state:

```bash
go run ./cmd/a21 serial-list
```

Only USB-looking candidates from that inventory should be used with `firmware-check --kind upload` or `firmware-flash-plan`. A visible `/dev/cu.*` path is not enough by itself.

The inventory lists `/dev/cu.*` paths, marks `usbmodem` candidates, and reports whether `lsof` sees another process holding the path. A busy port is a hard stop for `firmware-check --kind upload`.
