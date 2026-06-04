# Workspace Source Readiness Registry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the internal test 4 workspace surface from upload-job metadata only to a safe source/readiness registry so A21 can show which public/personal workspace sources exist, which are metadata-only, and which are searchable metadata candidates, without storing raw documents or claiming real V21 indexing.

**Architecture:** Keep `/v1/workspace-upload-jobs` as the job control endpoint. Add Gateway-owned memory-only source records derived from redacted job metadata, expose them through `GET /v1/workspace-sources`, and allow a narrow metadata-only `mark_searchable` job action that updates source readiness and professional workspace runtime summaries. No document bytes, document text, import URLs, provider output, real indexing, or V21 execution is introduced.

**Tech Stack:** Go Gateway, existing workspace job structs, existing professional workspace runtime, existing simulator/read-record discipline, Gateway tests, docs/state/handoff discipline.

---

## Transition

Transition id: `T-INTERNAL-TEST4-WORKSPACE-SOURCE-READINESS-001`

Current state:

- `/v1/workspace-upload-jobs` creates redacted no-execute job metadata and supports `mark_failed`, `retry`, and `delete`.
- `/v1/professional-workspace` exposes query scope and reports `UploadAPIReady=false`, `IndexingAPIReady=false`, and `WorkspaceStatus=contract_ready`.
- The simulator can show upload job status and read-record metadata, but there is no first-class source registry or source-scope readiness summary.

Target state:

- `GET /v1/workspace-sources` returns schema `a21.gateway.workspace_sources.v1` with memory-only redacted source metadata derived from upload/import jobs.
- Sources are filterable by `source_id`, `user_id`, `workspace_id`, and `source_scope`.
- Creating a workspace upload/import job creates a source record with `readiness=metadata_only`, `index_status=not_started_no_execute`, and safe redaction flags.
- `PUT /v1/workspace-upload-jobs` supports `mark_searchable` / `mark_indexed_metadata_only` to promote a metadata source to `readiness=searchable_metadata_only` and `index_status=searchable_metadata_only` without executing real indexing.
- `mark_failed`, `retry`, and `delete` keep the corresponding source readiness in sync.
- `/v1/professional-workspace` runtime includes source-scope counts and a derived readiness for the selected `query_scope`, while still keeping `v21_execution_allowed=false`.
- Traces record only safe markers such as `workspace.source.created_metadata_only` and `workspace.index_job.searchable_metadata_only`.

Boundaries:

- No document text, file bytes, base64 payloads, import URLs, local paths, credentials, provider output, or transcript storage.
- No real upload storage, no real indexing, no provider execution, no real V21 execution.
- No Gateway service start beyond optional local smoke.
- No firmware build, flash, serial, NVS, ECS change, or physical hardware action.
- Do not change accepted internal test 3 Xiaozhi audio/barge-in/Opus behavior.

## Files

- Modify `internal/gateway/server.go`
- Modify `internal/gateway/server_test.go`
- Modify `internal/gateway/simulator.go` only if the source registry can be surfaced cheaply without visual redesign
- Modify `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Modify `docs/project_state_machine.md`
- Append `docs/agent_handoff_log.md`

## Task 1: Tests

- [x] Step 1: Add a workspace-sources test proving job creation creates a redacted source record, `mark_searchable` updates it, professional workspace reports correct personal/public counts/readiness, and traces do not leak raw content.
- [x] Step 2: Add lifecycle tests proving retry/fail/delete update source readiness without storing raw document labels after delete.
- [x] Step 3: Add rejection tests proving `/v1/workspace-sources` filters reject unsafe labels and raw payload fields are still rejected by upload jobs.

Run:

```bash
go test ./internal/gateway -run 'TestWorkspaceSource|TestWorkspaceUploadJobs|TestProfessionalWorkspace' -count=1
```

Observed before implementation:

- New tests failed to build because `WorkspaceSourcesResponse`,
  `WorkspaceUploadJob.SourceID`, professional workspace source-count fields,
  and test helpers did not exist.

## Task 2: Gateway Implementation

- [x] Step 1: Add `WorkspaceSource` response structs and schema constant `a21.gateway.workspace_sources.v1`.
- [x] Step 2: Add memory-only source state keyed by `source_id`, linked to `job_id`, with safe counts and redaction flags.
- [x] Step 3: Add `GET /v1/workspace-sources` with safe filters.
- [x] Step 4: Create a source record when a workspace job is created.
- [x] Step 5: Extend upload job actions with `mark_searchable` / `mark_indexed_metadata_only`, and sync retry/fail/delete source readiness.
- [x] Step 6: Add source-scope counts/readiness to professional workspace runtime and response without enabling V21 execution.

## Task 3: Docs, State, And Verification

- [x] Step 1: Update protocol and internal-test4 docs.
- [x] Step 2: Update current control, project state machine, and handoff log.
- [x] Step 3: Verify:

```bash
go test ./internal/gateway -run 'TestWorkspaceSource|TestWorkspaceUploadJobs|TestProfessionalWorkspace' -count=1
go test ./internal/gateway -count=1
git diff --check
GOMAXPROCS=2 make verify
```

Expected:

- All commands exit 0.

Commit message:

```bash
git add internal/gateway/server.go internal/gateway/server_test.go internal/gateway/simulator.go docs/plans/2026-06-04-workspace-source-readiness-registry.md docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md docs/engineering/PROTOCOL.md docs/engineering/A21_CURRENT_CONTROL.md docs/project_state_machine.md docs/agent_handoff_log.md
git commit -m "feat(gateway): expose workspace source readiness"
```
