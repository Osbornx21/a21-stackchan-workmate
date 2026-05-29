# A21 Firmware Release Index Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Require upload-path firmware candidates to be present in the A21 release index before they can pass upload, device identity, or flash-plan dry-run guards.

**Architecture:** Keep lower-level `firmware-artifact-check` useful for inspection, but make upload-path checks call `ValidateArtifact` with release-index enforcement.

**Tech Stack:** Go, existing JSONL release index, existing firmware guard receipts.

---

### Task 1: Red Tests

**Files:**
- Modify: `internal/firmwarecheck/artifact_test.go`

- [x] Test upload candidate without release index is rejected.
- [x] Test upload candidate with matching release index is accepted.
- [x] Test upload candidate with mismatched release index checksum is rejected.
- [x] Run targeted tests and verify missing enforcement fails.

### Task 2: Guard Enforcement

**Files:**
- Modify: `internal/firmwarecheck/artifact.go`
- Modify: `internal/firmwarecheck/device_identity.go`

- [x] Add optional release-index enforcement to `ArtifactOptions`.
- [x] Add `release_index_path` to guarded artifact results.
- [x] Validate release index entry identity and checksum.
- [x] Require release index for upload candidates.
- [x] Require release index for device identity candidates.

### Task 3: Test Base and Docs

**Files:**
- Modify: `internal/app/app_test.go`
- Modify: `internal/firmwarecheck/device_identity_test.go`
- Modify: `docs/engineering/FIRMWARE_RELEASE_DISCIPLINE.md`
- Create: `docs/engineering/PHASE5F_FIRMWARE_RELEASE_INDEX_GUARD.md`

- [x] Upgrade legal upload-path test fixtures to include release index records.
- [x] Document that hand-assembled `.bin + .sha256` pairs cannot pass upload planning.
- [x] Run targeted tests, `make verify`, `make release-check`, and firmware guard checks.
