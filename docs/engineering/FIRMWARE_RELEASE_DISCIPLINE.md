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

`firmware-check` rejects unpinned or missing core firmware dependencies and rejects PlatformIO configs that omit the raw upload blocker. It also treats `firmware/stackchan/a21-firmware.json` as the firmware release identity source of truth: every `A21_FIRMWARE_ID`, `A21_FIRMWARE_VERSION`, and `A21_FIRMWARE_BOARD` build flag in the CoreS3 and native test environments must exactly match the manifest. This is intentional: firmware builds must be reproducible, must not silently drift under A21, and must fail fast before any unguarded flash path can run.

The generated commit header is ignored:

```text
firmware/stackchan/include/a21_firmware_build.generated.h
```

It is created by PlatformIO before native tests and CoreS3 builds. It must not be committed.

## Build Discipline

Before any firmware build:

```bash
make firmware-tools
go run ./cmd/a21 firmware-check
PLATFORMIO_CORE_DIR=$PWD/.a21-tools/platformio-core .a21-tools/platformio-venv/bin/pio run -d firmware/stackchan
```

The preferred wrapper is:

```bash
make firmware-test
make firmware-build
make firmware-upload-blocker-check
make firmware-package
make firmware-current-artifact-check
make firmware-artifact-prune-plan
make office-handoff
```

`firmware-test` runs the PlatformIO `native` environment and Unity tests. It must stay hardware-free.

`firmware-upload-blocker-check` intentionally invokes PlatformIO's raw upload target and expects it to fail with the A21 blocker message before any hardware write can start. A successful raw upload target is a release-blocking failure.

`firmware-current-artifact-check` validates the newest packaged artifact for the current git commit by reading `a21-firmware-release-index.jsonl`, selecting the latest matching package, and re-running the artifact, release-index, and per-artifact manifest guards. It is part of `make release-check`, so a package step is not considered release-clean until the generated candidate can be independently re-read from the release ledger.

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
- Gateway `device.event` send gating, deterministic seq/trace IDs, and interrupt/mock-turn envelope construction
- audio WebSocket begin gating after Gateway connection, full 640-byte mock `audio.frame` envelope construction, queued mic PCM `audio.frame` uplink construction, ack control-event parsing, mock `audio.playback.chunk` downlink parsing, full 20 ms 16 kHz PCM base64 payload capacity, fixed 640-byte PCM encode/decode, bounded playback buffering, M5 speaker pump queue gating, and send rejection while disconnected
- microphone capture policy and four-frame uplink queue that record one PCM16 frame only when the render state is capture-safe and the speaker queue is idle; this protects the current M5Unified internal mic/speaker half-duplex constraint
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

Upload-path dry-run guards now require both the release index record and the sibling artifact manifest. `firmware-upload-check`, `firmware-device-check`, and `firmware-flash-plan` also reject manifest, artifact, and device-report input paths containing forbidden X21/V21 identity before opening those paths, and they do not echo the polluted path back to the operator. A hand-assembled `.bin + .sha256` pair may still be inspected with `firmware-artifact-check`, but it cannot pass upload or flash-planning gates unless:

- the same directory's release index contains a matching firmware ID, version, board, commit, timestamp, artifact filename, checksum filename, SHA-256, and build provenance
- the sibling `.manifest.json` contains matching A21 project, firmware ID, version, board, commit, timestamp, artifact filename, checksum filename, SHA-256, and build provenance
- the release index entry and sibling manifest resolve to the same checked artifact path and checksum path
- the release index and sibling manifest full artifact/checksum paths do not contain forbidden X21/V21 identities

`firmware-upload-check` also rejects older packages when the release index contains a newer artifact for the same firmware ID, version, board, and commit. This keeps the dry-run path aligned with the latest A21 package for that exact checkout and avoids choosing a stale same-commit binary by accident.

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
go run ./cmd/a21 firmware-artifact-check --artifact firmware/artifacts/<a21-stackchan...bin>
```

or:

```bash
A21_FIRMWARE_ARTIFACT=firmware/artifacts/<a21-stackchan...bin> make firmware-artifact-check
```

For the current checkout, prefer the ledger-backed command:

```bash
go run ./cmd/a21 firmware-current-artifact-check --commit <git-sha>
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

This command verifies the artifact guard and rejects ambiguous upload targets such as `auto`, `default`, `any`, non-`/dev/` paths, and non-serial `/dev/*` paths such as `/dev/null`. Accepted upload paths are deliberately narrow: macOS `cu.*`/`tty.*` names must contain `usbmodem` or `usbserial`, and Linux names must be `ttyUSB*` or `ttyACM*`. Bluetooth, debug-console, and other generic serial-looking ports are rejected before process-ownership checks. It also checks whether the selected serial path is already held by another process. It still does not flash. Actual flashing must only be introduced later as a separate guarded command after physical device identity checks are in place.

Successful `firmware-upload-check` output is intentionally a dry-run receipt. The JSON must include:

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

go run ./cmd/a21 firmware-device-check \
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

`firmware-device-check` requires `--max-device-age-ms`; `make firmware-device-report` writes `reports/a21-devices-YYYYMMDD-HHMMSS.json` and uses direct Gateway HTTP without ambient proxy inheritance. It rejects Gateway URLs that target known X21/V21 legacy ports, requires the Gateway response to declare `schema_version=a21.gateway.devices.v1` and `service=a21-gateway`, and rejects Gateway device identity fields that contain forbidden X21/V21 naming before writing the report. `make firmware-device-check` passes `A21_DEVICE_MAX_AGE_MS=300000` by default. Override that value only for an explicitly documented lab reason; physical acceptance should use a freshly captured Gateway `/v1/devices` report.

The captured report preserves Gateway operator fields such as `connection_status`, `device_age_ms`, `current_mode`, `current_expression`, and `playback_stream_id`. These fields help prove what the office operator was looking at during acceptance, but the guard still uses explicit identity, artifact, commit, and freshness checks rather than trusting display state alone. `firmware-device-check` requires the report to carry A21 Gateway identity, either as the raw Gateway response fields `schema_version=a21.gateway.devices.v1` and `service=a21-gateway`, or as the `gateway_schema_version` / `gateway_service` fields written by `firmware-device-report`. A naked hand-written `{"devices":[...]}` file is not valid acceptance evidence. It also requires `connection_status=online`; stale or unknown Gateway state is a hard stop.

The guard verifies:

- the artifact still passes `firmware-artifact-check`
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

This receipt is a stronger identity confirmation than `firmware-upload-check`, but it is still not permission to flash. It proves that Gateway has seen a StackChan-like A21 device identity matching the candidate artifact. A future real flashing command must require both the upload dry-run receipt and this device identity receipt, then perform its own final confirmation.

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

`firmware-flash-plan` requires `--max-device-age-ms`; the Makefile wrapper passes the same `A21_DEVICE_MAX_AGE_MS` freshness guard as `firmware-device-check` and writes a timestamped no-flash receipt to:

```text
reports/a21-firmware-flash-plan-YYYYMMDD-HHMMSS.json
```

This makes flash planning auditable instead of ephemeral stdout. The receipt includes `generated_at_ms` and `report_path`, but it still sets `flash_allowed: false`.

The guard verifies:

- the artifact passes `firmware-artifact-check`
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
go run ./cmd/a21 firmware-bootstrap-flash-plan \
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
go run ./cmd/a21 firmware-bootstrap-flash-execute \
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
6. Run `firmware-artifact-check`.
7. Run `firmware-upload-check`.
8. Capture `/v1/devices` from the A21 Gateway and run `firmware-device-check`.
9. Run `firmware-flash-plan`.
10. If the device already reports A21 identity, prefer `firmware-flash-plan` and keep bootstrap flashing out of the path.
11. If this is initial bring-up or recovery and no A21 identity can be captured, run `firmware-bootstrap-flash-plan`.
12. Only after the bootstrap plan passes may `firmware-bootstrap-flash-execute` run with `WRITE_A21_STACKCHAN_FIRMWARE`.

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

Only USB-looking candidates from that inventory should be used with `firmware-upload-check` or `firmware-flash-plan`. A visible `/dev/cu.*` path is not enough by itself.

The inventory lists `/dev/cu.*` paths, marks `usbmodem` candidates, and reports whether `lsof` sees another process holding the path. A busy port is a hard stop for `firmware-upload-check`.
