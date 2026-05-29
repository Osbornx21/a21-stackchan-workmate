# A21 Firmware Release Discipline

## Purpose

A21 firmware must be treated as a device release artifact, not a casual sketch. Wrong package, wrong board, wrong port, or wrong project identity can brick time, damage trust, and contaminate X21/V21 work.

## Non-Negotiables

- A21 firmware lives under `firmware/stackchan/`.
- A21 firmware build artifacts must be named with `a21`, target board, firmware version, git commit, and build timestamp.
- A21 firmware may not use X21 or V21 project names, service names, artifact names, namespaces, or upload targets.
- Firmware upload is forbidden unless a preflight command verifies manifest identity, board ID, version, git commit, and explicit upload port.
- `pio run -t upload` must not be wrapped in a generic target until an A21 upload guard exists.
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

The StackChan Y-axis servo must eventually be clamped to the documented safe range of 5 to 85 degrees.

## Build Discipline

Before any firmware build:

```bash
go run ./cmd/a21 firmware-check
PLATFORMIO_CORE_DIR=$PWD/.a21-tools/platformio-core .a21-tools/platformio-venv/bin/pio run -d firmware/stackchan
```

The preferred wrapper is:

```bash
make firmware-build
make firmware-package
```

`make firmware-package` copies PlatformIO's generic `firmware.bin` into `firmware/artifacts/` with an A21-specific filename:

```text
a21-stackchan-<version>-m5stack-cores3-<git-sha>-<YYYYMMDD-HHMMSS>.bin
```

It also writes a sibling `.sha256` file. Only packaged artifacts should be considered candidates for future upload.

Before any future firmware upload:

1. Confirm physical device identity.
2. Confirm serial/upload port.
3. Confirm `firmware/stackchan/a21-firmware.json`.
4. Confirm git commit and version.
5. Confirm generated artifact filename begins with `a21-stackchan-`.
6. Run an A21 upload guard command.

There is intentionally no upload target in Phase 5A.
