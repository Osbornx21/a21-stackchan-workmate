# 2026-06-04 - Workspace Document Upload Intake

Status: active control plan.
Owner: A21 control tower.
Transition: `T-INTERNAL-TEST4-WORKSPACE-DOCUMENT-UPLOAD-INTAKE-001`.
Created: 2026-06-04 CST.

## Execution Update - 2026-06-04

- Gateway added `POST /v1/workspace-documents` with schema
  `a21.gateway.workspace_documents.v1`.
- The endpoint accepts multipart uploads only, stores bytes in an A21 local
  runtime store, hashes by SHA-256, and links the stored document to the
  existing workspace job/source registries.
- Linked jobs and sources use `stored_local_pending_index`,
  `storage_status=stored_local`, and `index_status=not_started_no_execute`.
- `/v1/professional-workspace` now reports `stored_local_pending_index` in
  query-scope readiness when the selected scope has local stored uploads but no
  searchable source.
- Simulator Workspace Audit can upload a selected file and shows only safe
  document/job/source metadata.
- This round did not implement parsing, import URL fetch, chunking, embedding,
  indexing, cloud storage, V21 execution, durable auth, or hardware consult
  acceptance.

## Goal

Move the internal test 4 knowledge workspace one step beyond metadata-only
upload jobs by allowing A21 Gateway to accept a real user document upload into
an A21-owned local runtime store, while preserving the no-index/no-V21-execute
boundary.

This is an intake transition, not a retrieval or indexing transition. It proves
that users can upload a document into the A21 workspace surface and see a safe
source/job/readiness state without exposing document text, file bytes, local
paths, provider output, V21 internals, or credentials.

## Current State

- `GET/POST/PUT /v1/professional-workspace` selects redacted `user_id`,
  `workspace_id`, and `query_scope`.
- `GET/POST/PUT /v1/workspace-upload-jobs` manages metadata-only no-execute
  jobs.
- `GET /v1/workspace-sources` exposes memory-only source readiness derived from
  redacted job metadata.
- The simulator shows workspace job/read/readiness metadata.
- There is no endpoint that accepts file bytes yet; the plan explicitly states
  file upload/import/index execution is not implemented.

## Target State

- Gateway exposes a new A21-namespaced local upload intake endpoint:
  `POST /v1/workspace-documents`.
- The endpoint accepts `multipart/form-data` only, with a single `file` part
  and safe metadata fields: `user_id`, `workspace_id`, `source_scope`,
  `document_label`, `content_type`, `trace_id`, `session_id`, and `device_id`.
- Gateway stores the file bytes under an A21 runtime directory:
  `A21_WORKSPACE_DOCUMENT_STORE_DIR` when configured, otherwise
  `.a21-run/gateway/workspace-documents`.
- Runtime storage is content-addressed by SHA-256 and kept out of API,
  trace, report, and simulator responses.
- Responses expose only safe metadata: document/source/job IDs, document hash,
  byte size, safe content type, safe document label, status, storage status,
  timestamps, readiness, and redaction flags.
- The linked workspace job/source remain no-index and no-V21-execute:
  `stored_local_pending_index` plus `not_started_no_execute`.
- `/v1/professional-workspace` reflects the uploaded source in
  `source_scope_counts` and `query_scope_readiness`, but still keeps
  `v21_execution_allowed=false`.

## Boundary

Allowed:

- Modify Gateway HTTP handlers, in-memory workspace source/job state, tests,
  simulator UI, and docs.
- Create files under `.a21-run/gateway/workspace-documents` or a test-provided
  temporary store path only when the upload endpoint is called.
- Use standard Go multipart, hashing, and filesystem APIs.

Forbidden:

- Real indexing, embedding, parsing, chunking, OCR, V21 execution, provider
  execution, cloud storage, database schema changes, account auth, ECS changes,
  Gateway service startup, firmware builds, flash, serial, NVS writes, or Git
  prune/gc.
- Returning or tracing document text, raw bytes, base64 payloads, local storage
  paths, user private filenames, import URLs, credentials, provider output, or
  V21 evidence.
- Weakening the existing `/v1/workspace-upload-jobs` raw-payload rejection.

## Implementation Steps

1. Add failing tests for multipart upload storage, redaction, linked
   source/job readiness, trace markers, and unsafe input rejection.
2. Add `WorkspaceDocument` response types and a Gateway handler for
   `POST /v1/workspace-documents`.
3. Add server options/env support for `WorkspaceDocumentStoreDir` and a
   conservative default max upload byte limit.
4. Store upload bytes through a temp file plus SHA-256 digest, then atomically
   rename into the A21 runtime store.
5. Link each stored upload to a workspace upload job and source with
   `stored_local_pending_index` readiness.
6. Add simulator file-input controls and safe metadata readout for operator
   internal test checks.
7. Update `PROTOCOL.md`, this plan, project state, current control, and handoff
   log.
8. Verify with focused Gateway tests, full Gateway tests, `git diff --check`,
   and `GOMAXPROCS=2 make verify`.

## Acceptance Conditions

- A multipart upload stores bytes under the configured A21 local store.
- The response and traces do not include the raw document content, original
  private filename, local path, import URL, credentials, provider output, or
  V21 evidence.
- `GET /v1/workspace-sources?source_id=...` shows the linked source as
  `stored_local_pending_index`.
- `GET /v1/workspace-upload-jobs?job_id=...` shows the linked job as
  `stored_local_pending_index` with `index_status=not_started_no_execute`.
- `/v1/professional-workspace` shows source-scope counts/readiness while
  keeping `v21_execution_allowed=false`.
- Unsupported JSON/raw-payload upload attempts fail without echoing unsafe
  values.
- Existing metadata-only job tests continue to pass.

## Failure State

- `F-WORKSPACE-DOCUMENT-UPLOAD-RAW-LEAK` if API, trace, docs, or simulator
  output exposes raw document content, private filenames, local paths,
  credentials, provider output, or V21 evidence.
- `F-WORKSPACE-DOCUMENT-UPLOAD-FALSE-SEARCHABLE` if stored uploads are marked
  searchable or V21-executable before an approved indexing transition.
- `F-WORKSPACE-DOCUMENT-UPLOAD-UNSCOPED-IO` if files are written outside the
  A21 runtime store or test temp store.

## Rollback Path

Revert this scoped transition's code/docs commit. Existing metadata-only
workspace jobs and professional workspace selection remain the stable fallback.

## Next State

`S-INTERNAL-TEST4-WORKSPACE-DOCUMENT-UPLOAD-INTAKE-READY`

Follow-up candidates:

- `T-INTERNAL-TEST4-WORKSPACE-LOCAL-PARSER-INDEX-PLAN-001`
- `T-V21-UPLOAD-INDEX-ADAPTER-V2-001`
- `T-INTERNAL-TEST4-WEB-WORKSPACE-SHELL-001`
