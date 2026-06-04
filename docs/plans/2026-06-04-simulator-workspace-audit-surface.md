# Simulator Workspace Audit Surface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make internal test 4 upload/read discipline visible in the A21 simulator so an operator can inspect workspace job status and professional read records from the same control surface.

**Architecture:** Reuse existing Gateway contracts. The simulator will call `GET /v1/professional-read-records` and display only ledger metadata: record count, status, query scope, source-scope counts, workspace status, and the last workspace job status. It will not display query text, evidence bodies, document text, provider output, URLs, paths, credentials, voice transcripts, or audio.

**Tech Stack:** Existing Go-served simulator HTML/JS in `internal/gateway/simulator.go`, Gateway tests, Browser/manual simulator smoke, repo docs/state/handoff discipline.

---

## Transition

Transition id: `T-INTERNAL-TEST4-SIMULATOR-WORKSPACE-AUDIT-001`

Current state:

- Gateway exposes `/v1/workspace-upload-jobs` and `/v1/professional-read-records`.
- The simulator can create a no-execute workspace upload job.
- The simulator cannot display read-record ledger state.

Target state:

- Simulator has a Workspace Audit section.
- Operator can manually refresh professional read records.
- Professional evidence returns trigger read-record refresh for the current trace.
- Workspace job create updates a visible no-execute job readout.
- Served simulator smoke test proves the new IDs and endpoint references exist.

Boundaries:

- No real document upload bytes, indexing, persistence, database, ACL work, provider execution, V21 execution, Gateway service start, firmware, flash, serial, NVS, or physical hardware action.
- Do not change accepted internal test 3 Xiaozhi audio/control behavior.

## Files

- Modify `internal/gateway/simulator.go`
- Modify `internal/gateway/server_test.go`
- Modify `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Modify `docs/project_state_machine.md`
- Append `docs/agent_handoff_log.md`

## Task 1: Simulator UI And JS

- [x] Step 1: Add a Workspace Audit section with the following IDs:
  - `workspaceJobReadout`
  - `professionalReadRecordCount`
  - `professionalReadRecordStatus`
  - `professionalReadRecordScope`
  - `professionalReadRecordSources`
  - `professionalReadRecordWorkspace`
  - `professionalReadRecordsRefresh`
- [x] Step 2: Add `refreshProfessionalReadRecords()` that calls `/v1/professional-read-records`, filtered by current `trace_id` when present.
- [x] Step 3: Render only safe metadata and update it after workspace job creation and professional evidence responses.

## Task 2: Tests

- [x] Step 1: Update `TestSimulatorPageServed` to assert the new endpoint and DOM IDs are present.
- [x] Step 2: Run:

```bash
go test ./internal/gateway -run 'TestSimulatorPageServed|TestProfessionalReadRecords' -count=1
```

Expected:

- Tests pass and prove the simulator exposes the audit surface plus existing read-record API behavior.

## Task 3: Docs, State, And Verification

- [x] Step 1: Update the internal-test4 plan, current control, project state machine, and handoff log.
- [x] Step 2: Verify:

```bash
go test ./internal/gateway -run 'TestSimulatorPageServed|TestProfessionalReadRecords' -count=1
go test ./internal/gateway -count=1
git diff --check
GOMAXPROCS=2 make verify
```

Expected:

- All commands exit 0.

Commit message:

```bash
git add internal/gateway/simulator.go internal/gateway/server_test.go docs/plans/2026-06-04-simulator-workspace-audit-surface.md docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md docs/engineering/A21_CURRENT_CONTROL.md docs/project_state_machine.md docs/agent_handoff_log.md
git commit -m "feat(gateway): surface workspace read audit in simulator"
```
