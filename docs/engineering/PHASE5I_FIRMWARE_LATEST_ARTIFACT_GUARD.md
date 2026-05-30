# A21 Phase 5I Firmware Latest Artifact Guard

## Purpose

Phase 5I removes a subtle firmware release ambiguity: a clean checkout can produce multiple valid A21 packages for the same git commit. They may all have correct names, checksums, embedded identity, release manifests, and release-index entries, but an operator should not accidentally choose an older same-commit package when a newer one exists.

## Rule

`firmware-upload-check` now accepts only the newest timestamped release-index entry for the candidate's:

- firmware ID
- version
- board
- git commit

Older same-commit packages can still be inspected with `firmware-artifact-check`, but they cannot pass the upload dry-run guard and therefore cannot move toward a flash plan.

## Non-Goals

This does not enable real flashing. Raw PlatformIO upload remains blocked, and A21 still requires the explicit guarded flash path plus physical device identity validation before any future real flash command can exist.
