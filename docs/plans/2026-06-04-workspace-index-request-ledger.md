# 2026-06-04 - Workspace Index Request Ledger

Status: active focused plan.
Owner: A21 control tower.
Transition: `T-INTERNAL-TEST4-WORKSPACE-INDEX-REQUEST-LEDGER-001`.
Created: 2026-06-04 CST.

## Goal

Move the internal test 4 workspace from `stored_local_pending_index` to an
explicit no-execute indexing request state without parsing documents, chunking,
embedding, uploading to V21, executing V21, or exposing private document
contents.

## Current State

- `/v1/workspace-documents` stores local upload bytes in the A21 runtime store
  and creates linked document/job/source metadata with
  `stored_local_pending_index`.
- `/v1/workspace-upload-jobs` remains metadata-only and can mark sources as
  `searchable_metadata_only`, but that action is not a real index request.
- `/v1/professional-workspace` can report source/query-scope readiness while
  keeping `v21_execution_allowed=false`.

## Target State

- `POST /v1/workspace-index-jobs` creates a redacted index request ledger entry
  for an existing stored-local document.
- The linked document, upload job, and source are promoted to
  `indexing_requested_no_execute`.
- `/v1/workspace-sources` and `/v1/professional-workspace` expose
  source-scope-aware indexing-request readiness.
- API responses, traces, simulator readouts, and tests continue to exclude raw
  document text, bytes, base64 payloads, original private filenames, local
  paths, import URLs, credentials, provider output, V21 evidence, transcripts,
  and audio.

## Scope

Allowed:

- Gateway in-memory ledger and redacted HTTP contract.
- Simulator workspace audit controls/readouts.
- Gateway tests and docs/state/handoff updates.

Forbidden:

- No provider execution.
- No V21 execution or V21 internal reads.
- No document parsing, OCR, chunking, embedding, cloud storage, or import URL
  fetching.
- No Gateway service start.
- No firmware build, flash, serial, NVS write, ECS change, prune/gc, report
  deletion, or rollback of internal test 3 voice protocol work.

## Acceptance

- Focused Gateway tests cover happy path, forbidden raw payload rejection,
  source summary/readiness promotion, professional workspace readiness, and
  trace redaction.
- `git diff --check` passes.
- `GOMAXPROCS=2 make verify` passes.
- Project state and handoff log name this transition and keep physical/V21
  acceptance separate.

## Failure State

- `v21_boundary_leak`: responses or traces include document text, original
  filenames, local paths, credentials, provider output, or private evidence.
- `false_searchable_claim`: `indexing_requested_no_execute` is treated as
  searchable or as real V21 readiness.
- `internal_test3_regression`: Xiaozhi voice-main-chain protocol behavior is
  changed or reverted by this workspace cut.

## Rollback

Revert only this focused ledger/simulator/docs commit. Keep internal test 3
voice-chain, firmware product lane, provider selector, hardware parity, and
workspace upload-intake commits intact.

## Next State

- `S-INTERNAL-TEST4-WORKSPACE-INDEX-REQUEST-LEDGER-READY-V21-INDEX-EXECUTION-PENDING`
