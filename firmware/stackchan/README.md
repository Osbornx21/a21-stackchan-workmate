# A21 StackChan Firmware

This is the A21 firmware lane for M5Stack StackChan/CoreS3.

## Identity

- Project: `A21`
- Firmware ID: `a21-stackchan`
- Board: `m5stack-cores3`
- Version: `0.1.0`

## Build

From the repository root:

```bash
go run ./cmd/a21 firmware-check
make firmware-test
.a21-tools/platformio-venv/bin/pio run -d firmware/stackchan
```

`make firmware-test` runs host-native protocol parser tests. It does not flash hardware.

## Runtime Surface

The current firmware starts in local fallback state and renders the A21 identity, firmware version, state label, and short status text on the CoreS3 screen. Gateway `control.event` messages are parsed by `a21_firmware_protocol.h` and applied to a thin device-local state model in `a21_firmware_state.h`.

Gateway configuration is currently compile-time and A21-only:

- `A21_GATEWAY_HOST=10.21.0.1`
- `A21_GATEWAY_PORT=21080`
- control path `/ws/control`
- audio path `/ws/audio`

The firmware still does not connect to Wi-Fi or WebSocket in this slice. Network transport will be added after the state model and screen rendering path are stable under native tests.

## Upload

There is no upload target in Phase 5A. Do not run `pio run -t upload` until an A21 upload guard exists and the physical device, serial port, firmware version, and artifact name have been verified.

Phase 5B adds only dry-run guards:

```bash
go run ./cmd/a21 serial-list
go run ./cmd/a21 firmware-artifact-check --artifact firmware/artifacts/<a21-stackchan...bin>
go run ./cmd/a21 firmware-upload-check --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --commit <expected-git-sha>
```

These commands inventory serial devices and validate artifact identity, board, version, checksum, expected git commit, explicit serial target, port existence, and whether another process is already holding the serial path. They do not flash the device.
