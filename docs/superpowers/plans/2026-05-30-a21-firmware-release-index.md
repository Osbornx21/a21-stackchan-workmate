# A21 Firmware Release Index Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an A21-only firmware release index so packaged StackChan binaries are traceable by commit, board, version, timestamp, and checksum.

**Architecture:** Keep this inside `internal/firmwarecheck.PackageArtifact`. The package step remains the single source of release candidate creation. Do not introduce a real flash command.

**Tech Stack:** Go, JSONL, existing firmware manifest and artifact packaging guard.

---

### Task 1: Red Tests

**Files:**
- Create: `internal/firmwarecheck/package_test.go`

- [x] Test `PackageArtifact` writes one release index JSONL entry.
- [x] Test the entry mirrors artifact path, sha path, sha256, firmware identity, commit, and timestamp.
- [x] Test legacy output directories are rejected.
- [x] Run targeted tests and verify missing release index support fails.

### Task 2: Package Implementation

**Files:**
- Modify: `internal/firmwarecheck/package.go`

- [x] Add `ReleaseIndexFileName`.
- [x] Add `ReleaseIndexEntry`.
- [x] Add `release_index_path` to package output.
- [x] Append JSONL release records during packaging.
- [x] Reject package input/output paths containing forbidden legacy identities.

### Task 3: Docs and Verification

**Files:**
- Modify: `docs/engineering/FIRMWARE_RELEASE_DISCIPLINE.md`
- Create: `docs/engineering/PHASE5E_FIRMWARE_RELEASE_INDEX.md`

- [x] Document the release index purpose and fields.
- [x] Document that generic PlatformIO binaries are not release candidates.
- [x] Run targeted tests, `make verify`, `make release-check`, and firmware guard checks.
