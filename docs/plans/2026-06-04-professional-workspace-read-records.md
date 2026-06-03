# Professional Workspace Read Records Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a redacted professional-workspace read ledger so A21 can prove which safe scope was consulted without storing user queries, retrieved text, provider output, credentials, URLs, paths, or document bodies.

**Architecture:** Keep `/v1/professional-workspace` and `/v1/workspace-upload-jobs` as Gateway-owned contracts. Add an in-memory `professional_read_records` contract that starts before a professional V21 query and completes or fails with only safe metadata: trace/session/device IDs, redacted user/workspace labels, query scope, utterance length bucket, source-scope counts, workspace status, status code, timing, and storage redaction flags. Wire the ledger into both the mock professional turn path and the stock Xiaozhi professional route.

**Tech Stack:** Go Gateway, existing A21/V21 adapter v2 contract, existing Gateway tests, repo docs/state/handoff discipline, `go test`, `make verify`.

---

## Transition

Transition id: `T-INTERNAL-TEST4-PROFESSIONAL-WORKSPACE-READ-RECORDS-001`

Current state:

- `/v1/professional-workspace` selects redacted `user_id`, `workspace_id`, and `query_scope`.
- `/v1/workspace-upload-jobs` records no-execute upload/import metadata.
- Professional turns send v2 workspace fields to V21 and trace only safe markers.
- There is no API showing which professional reads occurred, whether they succeeded, and what redacted source scopes were used.

Target state:

- Gateway exposes `GET /v1/professional-read-records`.
- Every professional V21 query creates one read record before adapter execution.
- Successful V21 responses complete the record with redacted `source_scope_counts` and `workspace_status`.
- V21 failures and contract-invalid responses mark the record failed without storing raw utterance, evidence, cards, speech blocks, provider output, URLs, paths, credentials, or document text.
- Roleplay/fast-companion paths still create no read record and do not call V21.

Boundaries:

- No real provider or V21 execution.
- No Gateway service start.
- No document upload bytes, indexing, persistence, database, cloud storage, or ACL implementation.
- No firmware build, flash, serial, NVS, or physical hardware action.
- Do not change accepted internal test 3 Xiaozhi voice/audio/barge-in behavior.

## Files

- Modify `internal/gateway/server.go`
- Modify `internal/gateway/server_test.go`
- Modify `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Modify `docs/project_state_machine.md`
- Append `docs/agent_handoff_log.md`

## Task 1: Red-Green Gateway Contract Tests

- [x] Step 1: Add tests for successful and failed professional read records.

Run:

```bash
go test ./internal/gateway -run 'TestProfessionalReadRecords' -count=1
```

Expected before implementation:

- The tests fail because `/v1/professional-read-records` is not registered and no read record type exists.

Required assertions:

- A successful professional mock turn creates one `completed` record.
- The record has schema `a21.gateway.professional_read_records.v1`.
- The record includes safe `trace_id`, `session_id`, `device_id`, `user_id`, `workspace_id`, `query_scope`, `privacy_scope`, `utterance_bucket`, `source_scope_counts`, `workspace_status`, and timing fields.
- The response redaction booleans are all false for stored raw content.
- The response body does not contain raw user query text, evidence bodies, card text, speech blocks, URLs, local paths, credentials, or secret-looking values.
- A V21 failure creates one `failed` record with safe failure code metadata.

## Task 2: Gateway Read Ledger Implementation

- [x] Step 1: Add `ProfessionalReadRecordsSchemaVersion`, read-record response structs, and redaction structs to `internal/gateway/server.go`.
- [x] Step 2: Add `professionalReadRecords map[string]ProfessionalReadRecord` and sequence state to `Server`, initialized in `NewServerWithOptions`.
- [x] Step 3: Register `GET /v1/professional-read-records`.
- [x] Step 4: Implement read-record helpers:

```go
func (s *Server) startProfessionalReadRecord(request v21adapter.QueryRequest, utteranceBucket string) string
func (s *Server) completeProfessionalReadRecord(recordID string, response v21adapter.QueryResponse)
func (s *Server) failProfessionalReadRecord(recordID string, code string)
func (s *Server) professionalReadRecordsResponse(recordID string, traceID string, sessionID string) ProfessionalReadRecordsResponse
```

Required behavior:

- Store only safe labels already accepted by the professional adapter contract.
- Store `utterance_bucket`, never `Utterance`.
- Copy only `public` and `personal` integer counts when non-negative.
- Accept only workspace statuses `searchable`, `uploaded`, `indexing`, `failed`, or `unavailable`.
- Record trace markers `professional.read_record.started`, `professional.read_record.completed`, and `professional.read_record.failed`.

## Task 3: Wire Professional Paths

- [x] Step 1: In `professionalTurnResponse`, start a read record immediately before `s.v21.Query`.
- [x] Step 2: Complete the read record after V21 returns a valid response.
- [x] Step 3: Mark it failed on V21 errors or contract-invalid responses.
- [x] Step 4: Apply the same start/complete/fail flow to `writeXiaozhiProfessionalTTS`.

Expected:

- Mock professional mode and stock Xiaozhi professional mode share the same redacted read-record schema.
- Roleplay/workmate/fast-companion paths remain unchanged.

## Task 4: Docs, State, And Verification

- [x] Step 1: Update protocol and internal-test4 plan docs with the read-record contract.
- [x] Step 2: Update current control, state machine, and handoff log.
- [x] Step 3: Verify.

Run:

```bash
go test ./internal/gateway -run 'TestProfessionalReadRecords|TestProfessionalModeSendsExplicitV21PlaceholderContract|TestProfessionalModeV21TimeoutCancelsQueryAndFallsBack|TestWorkmateModeDoesNotCallV21Adapter' -count=1
go test ./internal/gateway -count=1
git diff --check
GOMAXPROCS=2 make verify
```

Expected:

- All commands exit 0.

Commit message:

```bash
git add internal/gateway/server.go internal/gateway/server_test.go docs/plans/2026-06-04-professional-workspace-read-records.md docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md docs/engineering/PROTOCOL.md docs/engineering/A21_CURRENT_CONTROL.md docs/project_state_machine.md docs/agent_handoff_log.md
git commit -m "feat(gateway): record professional workspace reads"
```
