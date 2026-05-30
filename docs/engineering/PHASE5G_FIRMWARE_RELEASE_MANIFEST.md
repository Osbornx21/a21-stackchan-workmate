# A21 Phase 5G Firmware Artifact Manifest Guard

Phase 5G adds a per-artifact release manifest beside every packaged firmware binary.

`firmware-package` now writes:

- `<artifact>.bin`
- `<artifact>.bin.sha256`
- `<artifact>.bin.manifest.json`
- `a21-firmware-release-index.jsonl`

The manifest schema is `a21.firmware.artifact_manifest.v1`. It records:

- A21 project identity
- firmware ID
- firmware version
- board
- git commit
- package timestamp
- artifact filename
- artifact path
- checksum path
- SHA-256

Upload-path dry-run guards now require both:

- a matching `a21-firmware-release-index.jsonl` entry
- a matching sibling `.manifest.json`

This closes the gap where a manually assembled `.bin + .sha256` pair could look locally valid. A loose binary can still be inspected with `firmware-artifact-check`, but it cannot enter `firmware-upload-check`, `firmware-device-check`, or `firmware-flash-plan` unless the package ledger and the per-artifact manifest agree.

The CLI package command itself now enforces a clean git worktree before packaging. This mirrors the Makefile guard and prevents direct `a21 firmware-package` calls from producing a release artifact whose commit does not describe the source tree.

Real flashing remains locked. This phase adds no hardware-writing command.

Release verification also includes `make firmware-upload-blocker-check`, which intentionally invokes the raw PlatformIO upload target and expects the A21 blocker message. If a raw upload ever exits successfully, release verification must fail.
