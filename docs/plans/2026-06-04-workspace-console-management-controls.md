# A21 Workspace Console Management Controls

Date: 2026-06-04
Owner: Codex main control thread
Status: completed

## Transition

`T-WORKSPACE-CONSOLE-MANAGEMENT-CONTROLS-001`

## Current State

Internal test 4 already exposes a host-local `/workspace` page over existing
safe Gateway contracts. It can select professional query scope, upload a local
document, request the no-execute indexing ledger, refresh source/read metadata,
and show the roleplay/professional boundary. Backend support already exists for
workspace source lifecycle deletion through `PUT /v1/workspace-upload-jobs`
with `action=delete`, and for professional read-record filtering by
`record_id`, `trace_id`, and `session_id`.

## Target State

The `/workspace` page becomes a minimally useful management surface without
expanding protocol risk: users can delete the selected workspace source/job,
export safe metadata from the page state, and filter professional read records
by safe IDs.

## Trigger

The user asked to keep pushing internal test 4 toward a real product surface
while preserving internal test 3 endpoint voice/protocol changes and avoiding
repeat work after context compression.

## Action

1. Add source delete and safe metadata export controls to the existing
   `/workspace` console.
2. Add read-record filters for `record_id`, `trace_id`, and `session_id`.
3. Reuse existing Gateway APIs only; do not create a new service, port, route,
   provider execution path, V21 execution path, firmware action, or hardware
   action.
4. Keep the exported object schema explicit:
   `a21.workspace_console_export.v1`.
5. Preserve redaction discipline: no document content, provider output,
   original private path, raw bytes, prompt content, voice data, or evidence
   bodies in the page state or export.

## Acceptance

- `GET /workspace` serves controls for source deletion, metadata export, and
  read-record filters.
- The delete control uses the existing upload-job lifecycle contract and keeps
  deleted source state honest as `deleted_metadata_only`.
- The export control produces only client-side safe metadata.
- Existing workspace console safety assertions still reject forbidden terms and
  external URLs.
- Focused Gateway tests pass.
- `git diff --check` passes.
- `GOMAXPROCS=2 make verify` passes.
- Browser or Playwright evidence confirms the page renders on desktop and
  mobile without obvious overflow.

## Failure State

If the console requires a new backend API, real provider execution, real V21
execution, cloud upload, firmware flash, serial/NVS access, or any content leak,
stop and return to planning before implementation.

## Rollback Path

Revert only this transition's `/workspace` page, test, and documentation
changes. Do not revert internal test 3 endpoint voice/protocol commits or
earlier internal test 4 roleplay/professional contracts.

## Next State

`WORKSPACE-CONSOLE-MANAGEMENT-CONTROLS-READY`

## Verification - 2026-06-04

- `go test ./internal/gateway -run 'TestWorkspaceConsolePageServed|TestWorkspaceDocumentUpload|TestWorkspaceIndex|TestWorkspaceSource|TestProfessionalWorkspace' -count=1`:
  passed.
- Playwright opened `http://127.0.0.1:21080/workspace` on a local Gateway,
  uploaded a dummy local text fixture, requested the no-execute index ledger,
  exported `a21.workspace_console_export.v1`, filtered read records by a
  nonexistent `trace_id`, cleared filters, deleted the selected source/job, and
  observed `deleted_metadata_only`, `deleted_no_execute`,
  `searchable=false`, and zero horizontal overflow.
- Desktop evidence: `.a21-run/evidence/workspace-console-management-desktop.png`.
- Mobile evidence: `.a21-run/evidence/workspace-console-management-mobile.png`.
- Export evidence: `.a21-run/evidence/a21-workspace-metadata.json`.

No provider, V21, ECS, firmware, serial, NVS, prune/gc, flash, or physical
hardware action occurred.
