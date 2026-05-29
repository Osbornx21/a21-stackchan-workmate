# A21 Phase 5E Firmware Release Index

Phase 5E adds a machine-readable firmware release index to the A21 packaging path.

`firmware-package` now appends one JSONL record per packaged artifact to:

```text
firmware/artifacts/a21-firmware-release-index.jsonl
```

Each record uses `schema_version: "a21.firmware.release.v1"` and captures:

- firmware ID
- firmware version
- board
- git commit
- build timestamp
- artifact path
- checksum path
- SHA-256

This is not a flash permission system. It is a release-traceability ledger so operators can answer which exact A21 commit, board, version, and checksum produced a candidate binary.

Packaging now also rejects input or output paths containing forbidden legacy project identities. This prevents accidental use of X21/V21 build folders or artifact directories while A21 firmware remains isolated.

The index is generated only by A21 packaging. Generic PlatformIO `firmware.bin` files remain non-release build products and must not be used as upload candidates.
