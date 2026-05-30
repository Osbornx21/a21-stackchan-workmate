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

Phase 5G further requires a sibling per-artifact `.manifest.json` with schema `a21.firmware.artifact_manifest.v1`. The release index proves the artifact is in the package ledger; the per-artifact manifest makes the candidate self-describing beside the binary. Upload-path guards require both records to agree with the filename, checksum, embedded identity, A21 manifest, and expected git commit.

Phase 5I further tightens upload dry-runs by rejecting stale same-commit artifacts. When `a21-firmware-release-index.jsonl` contains multiple packages for the same A21 firmware ID, version, board, and commit, `firmware-upload-check` accepts only the newest timestamp. Older packages remain inspectable through `firmware-artifact-check`, but they cannot progress toward a flash plan.

Real flashing remains locked. The guard only strengthens dry-run receipts and does not add any command that writes to hardware.
