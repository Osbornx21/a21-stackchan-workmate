# A21 Phase 5F Firmware Release Index Guard

Phase 5F turns the firmware release index from a passive ledger into an upload-path guard.

The following dry-run checks now require the candidate artifact to appear in the same directory's `a21-firmware-release-index.jsonl`:

- `firmware-upload-check`
- `firmware-device-check`
- `firmware-flash-plan`

The guard validates that the release index entry matches the candidate artifact's:

- firmware ID
- version
- board
- git commit
- build timestamp
- artifact filename
- checksum filename
- SHA-256

This blocks hand-assembled `.bin + .sha256` pairs from entering the upload planning path. A binary can still pass the lower-level `firmware-artifact-check` for inspection, but it cannot become an upload or flash-plan candidate unless it came through the A21 packaging ledger.

Real flashing remains locked. The guard only strengthens dry-run receipts and does not add any command that writes to hardware.
