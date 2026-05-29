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
.a21-tools/platformio-venv/bin/pio run -d firmware/stackchan
```

## Upload

There is no upload target in Phase 5A. Do not run `pio run -t upload` until an A21 upload guard exists and the physical device, serial port, firmware version, and artifact name have been verified.

Phase 5B adds only dry-run guards:

```bash
go run ./cmd/a21 firmware-artifact-check --artifact firmware/artifacts/<a21-stackchan...bin>
go run ./cmd/a21 firmware-upload-check --artifact firmware/artifacts/<a21-stackchan...bin> --port /dev/cu.usbmodemXXXX --commit <expected-git-sha>
```

These commands validate artifact identity, board, version, checksum, expected git commit, explicit serial target, and whether another process is already holding the serial path. They do not flash the device.
