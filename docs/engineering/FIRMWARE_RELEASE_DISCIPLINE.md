# A21 Firmware Release Discipline

## Purpose

A21 firmware must be treated as a device release artifact, not a casual sketch. Wrong package, wrong board, wrong port, or wrong project identity can brick time, damage trust, and contaminate X21/V21 work.

## Non-Negotiables

- A21 firmware lives under `firmware/stackchan/`.
- A21 firmware build artifacts must be named with `a21`, target board, firmware version, git commit, and build timestamp.
- A21 firmware binaries must embed an A21 build identity containing firmware ID, version, board, and git commit.
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
PLATFORMIO_CORE_DIR=$PWD/.a21-tools/platformio-core .a21-tools/platformio-venv/bin/pio --version
```

This keeps A21 firmware tooling and PlatformIO package state isolated from X21/V21 and from system-global Python packages.

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
- `A21_FIRMWARE_ID`, `A21_FIRMWARE_VERSION`, and `A21_FIRMWARE_BOARD` build flags
- `scripts/a21_block_raw_upload.py` PlatformIO pre-build script for failing raw upload targets before flashing can start
- `scripts/a21_build_identity.py` PlatformIO pre-build script for generated commit metadata

`firmware-check` rejects unpinned or missing core firmware dependencies and rejects PlatformIO configs that omit the raw upload blocker. This is intentional: firmware builds must be reproducible, must not silently drift under A21, and must fail fast before any unguarded flash path can run.

The generated commit header is ignored:

```text
firmware/stackchan/include/a21_firmware_build.generated.h
```

It is created by PlatformIO before native tests and CoreS3 builds. It must not be committed.

## Build Discipline

Before any firmware build:

```bash
go run ./cmd/a21 firmware-check
PLATFORMIO_CORE_DIR=$PWD/.a21-tools/platformio-core .a21-tools/platformio-venv/bin/pio run -d firmware/stackchan
```

The preferred wrapper is:

```bash
make firmware-test
make firmware-build
make firmware-package
```

`firmware-test` runs the PlatformIO `native` environment and Unity tests. It must stay hardware-free.

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
- Gateway `device.event` send gating, deterministic seq/trace IDs, and interrupt/mock-turn envelope construction
- audio WebSocket begin gating after Gateway connection, mock `audio.frame` envelope construction, ack control-event parsing, mock `audio.playback.chunk` downlink parsing, full 20 ms 16 kHz PCM base64 payload capacity, bounded playback buffering, and send rejection while disconnected
- semantic touch intent runtime for `wake_or_listen` and `barge_in`, preserving `screen` vs `top_sensor` source metadata through Gateway `device.event` envelopes
- playback state-machine behavior for starting a speaking stream once, replacing streams with stop/clear, clearing buffered chunks, and stopping plus clearing immediately on barge-in
- StackChan Y-axis servo clamp to 5 to 85 degrees
- semantic render-state to Y-axis motion target mapping, plus write-on-change motion runtime behavior through a driver interface
- semantic render-state to RGB color mapping, plus write-on-change RGB runtime behavior through a driver interface

Current Gateway tests cover:

- `/v1/devices` registration from firmware-originated `device.event` payloads
- A21 firmware identity validation for `a21-stackchan`, semver version, `m5stack-cores3`, and git SHA commit
- rejection and metrics for forbidden X21/V21 firmware identity

`make firmware-package` copies PlatformIO's generic `firmware.bin` into `firmware/artifacts/` with an A21-specific filename:

```text
a21-stackchan-<version>-m5stack-cores3-<git-sha>-<YYYYMMDD-HHMMSS>.bin
```

It also writes a sibling `.sha256` file. Only packaged artifacts should be considered candidates for future upload.

`make firmware-package` first runs a clean-worktree guard. This prevents a binary built from uncommitted sources from being packaged under a misleading git commit.

## Artifact Guard

Every packaged firmware binary must pass:

```bash
go run ./cmd/a21 firmware-artifact-check --artifact firmware/artifacts/<a21-stackchan...bin>
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> make firmware-artifact-check
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

The embedded-identity check matters because a wrong `firmware.bin` could otherwise be copied into a correctly named artifact with a matching checksum. A package is not a valid A21 candidate unless the filename, checksum, manifest, and binary identity all agree.

## Upload Guard

Phase 5B adds an upload dry-run guard, not an upload command:

```bash
go run ./cmd/a21 firmware-upload-check \
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

This command verifies the artifact guard and rejects ambiguous upload targets such as `auto`, `default`, `any`, non-`/dev/` paths, and non-serial `/dev/*` paths such as `/dev/null`. Accepted serial path forms are macOS `cu.*`/`tty.*` and Linux `ttyUSB*`/`ttyACM*`. It also checks whether the selected serial path is already held by another process. It still does not flash. Actual flashing must only be introduced later as a separate guarded command after physical device identity checks are in place.

Successful `firmware-upload-check` output is intentionally a dry-run receipt. The JSON must include:

- `guard_id: a21.firmware.upload_guard.v1`
- `dry_run: true`
- `flash_allowed: false`
- `next_required_confirmation: physical_device_identity`
- `port_usage`

Any future real flashing command must not reinterpret this receipt as permission to flash. It is evidence that the candidate passed preflight checks, and nothing more.

The `--commit` value must match the git sha encoded in the artifact filename. The Makefile wrapper fills it from `git rev-parse --short=12 HEAD`, so an old firmware package cannot pass the dry-run guard for a newer checkout.

## Device Identity Guard

Phase 5C adds a physical-device identity dry-run guard. It still does not flash. It validates a Gateway `/v1/devices` JSON capture against the exact packaged firmware candidate:

```bash
curl -fsS http://127.0.0.1:21080/v1/devices > reports/a21-devices.json

go run ./cmd/a21 firmware-device-check \
  --artifact firmware/artifacts/<a21-stackchan...bin> \
  --device-report reports/a21-devices.json \
  --device-id stackchan-001 \
  --commit <expected-git-sha>
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_DEVICE_REPORT=reports/a21-devices.json \
A21_DEVICE_ID=stackchan-001 \
make firmware-device-check
```

The guard verifies:

- the artifact still passes `firmware-artifact-check`
- the artifact commit matches the expected git commit
- the device report contains the explicit `A21_DEVICE_ID`
- `identity_status` is `ok`
- reported firmware ID is `a21-stackchan`
- reported version, board, and commit match the artifact manifest and filename
- device ID and firmware identity contain no forbidden X21/V21 names

Successful output includes:

- `guard_id: a21.firmware.device_identity_guard.v1`
- `device_identity_confirmed: true`
- `flash_allowed: false`
- `next_required_confirmation: explicit_guarded_flash_command`

This receipt is a stronger identity confirmation than `firmware-upload-check`, but it is still not permission to flash. It proves that Gateway has seen a StackChan-like A21 device identity matching the candidate artifact. A future real flashing command must require both the upload dry-run receipt and this device identity receipt, then perform its own final confirmation.

## Flash Plan Guard

Phase 5D adds a combined flash-plan guard. It still does not flash. It composes the existing artifact, upload-port, and device-identity guards into one receipt so a future real flashing command cannot accidentally combine an old artifact, a new checkout, the wrong serial port, or the wrong device report:

```bash
go run ./cmd/a21 firmware-flash-plan \
  --artifact firmware/artifacts/<a21-stackchan...bin> \
  --port /dev/cu.usbmodemXXXX \
  --device-report reports/a21-devices.json \
  --device-id stackchan-001 \
  --commit <expected-git-sha>
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> \
A21_UPLOAD_PORT=/dev/cu.usbmodemXXXX \
A21_DEVICE_REPORT=reports/a21-devices.json \
A21_DEVICE_ID=stackchan-001 \
make firmware-flash-plan
```

The guard verifies:

- the artifact passes `firmware-artifact-check`
- the upload port is explicit, exists, and is not busy
- the Gateway device report contains the explicit `A21_DEVICE_ID`
- the upload guard and device-identity guard reference the same artifact checksum and commit
- every identity remains in the A21 namespace

Successful output includes:

- `guard_id: a21.firmware.flash_plan_guard.v1`
- `dry_run: true`
- `flash_allowed: false`
- `next_required_confirmation: future_explicit_guarded_flash_command`

This receipt is the strongest no-flash receipt in the current repository. It is a precondition design for future guarded flashing, not permission to flash.

## Current Flashing Status

Real flashing is intentionally locked. The repository has build, package, artifact-check, upload-check dry-run, device-identity dry-run, flash-plan dry-run gates, and a PlatformIO raw-upload blocker. No command is allowed to write an A21 binary to hardware yet. The next unlock must introduce a separate guarded flash command with a name that cannot be confused with X21 or V21 tooling.

A21 firmware work must continue to use the repository-local `.a21-tools/` PlatformIO environment and `firmware/artifacts/a21-stackchan-...` packages. Do not point A21 upload checks at X21/V21 build directories, generic `firmware.bin` paths, or auto-selected serial ports.

Before any future firmware upload:

1. Confirm physical device identity.
2. Confirm serial/upload port.
3. Confirm `firmware/stackchan/a21-firmware.json`.
4. Confirm git commit and version.
5. Confirm generated artifact filename begins with `a21-stackchan-`.
6. Run `firmware-artifact-check`.
7. Run `firmware-upload-check`.
8. Capture `/v1/devices` from the A21 Gateway and run `firmware-device-check`.
9. Run `firmware-flash-plan`.
10. Only then may a future explicit guarded upload command run.

There is intentionally no upload target in Phase 5D.

## Serial Inventory

Before choosing an upload port, inspect the current serial state:

```bash
go run ./cmd/a21 serial-list
```

The inventory lists `/dev/cu.*` paths, marks `usbmodem` candidates, and reports whether `lsof` sees another process holding the path. A busy port is a hard stop for `firmware-upload-check`.
