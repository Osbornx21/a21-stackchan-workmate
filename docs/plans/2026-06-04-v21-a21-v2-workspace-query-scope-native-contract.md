# 2026-06-04 - V21 A21 v2 Workspace Query Scope Native Contract

Status: V21 worker completed; V21 merge and ACL/schema follow-up pending.
Owner: A21 control tower; execution in V21 scoped worker.
Transition: `T-V21-A21-V2-WORKSPACE-QUERY-SCOPE-NATIVE-001`.

## Goal

Move internal test 4 professional workspace support from A21-only adapter
wrapping toward native V21 support for A21 v2 professional scope fields.

A21 already sends `device_id`, `user_id`, `workspace_id`, and `query_scope` to
its adapter boundary and can wrap V21 retrieval results with redacted
`source_scope_counts` plus `workspace_status`. V21 now needs a scoped worker to
accept and report the same fields on its internal voice-query boundary without
leaking document contents or pretending unsupported personal/private corpus ACL
is already finished.

## Current State

- A21 branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- A21 current HEAD at dispatch:
  `919e4c4 docs(control): prepare mcp status parity dispatch`.
- V21 local repository:
  `/Users/jiyurun/Documents/v21-knowledge-platform`.
- V21 current branch at inspection:
  `feat/consumer-lan-discovery-ui`.
- V21 current working tree is dirty with LAN discovery / desktop connector
  changes. Those changes are source state only for this transition and must
  not be reverted or overwritten.
- V21 already has:
  - `/internal/v1/knowledge/voice-query`;
  - upload metadata/local ingest/retrieval/index APIs;
  - workspace, actor, collection, object ref, release, chunk, and citation
    tables;
  - `object_refs.access_class` with current values `INTERNAL`, `SENSITIVE`,
    and `RESTRICTED`.

## Target State

- V21 internal voice-query schema and Go request/response types accept the A21
  v2 additive fields:
  - `device_id`
  - `user_id`
  - `workspace_id`
  - `query_scope`
- Allowed `query_scope` values match A21:
  - `public_only`
  - `personal_only`
  - `personal_plus_public`
- V21 response can include:
  - `source_scope_counts`
  - `workspace_status`
- The first worker must be truthful:
  - If V21 can map current release evidence to public/personal scope using
    existing schema, implement that mapping with tests.
  - If V21 cannot yet enforce personal/public ACLs without a migration or
    collection-policy decision, accept/validate the fields but report a stable
    pending/blocker status rather than claiming personal corpus enforcement.
- A21 remains ignorant of V21 internals and continues to target only an adapter
  boundary.

## Not In Scope

- No A21 code changes in the V21 worker.
- No direct A21 reads from V21 database tables.
- No service start/stop unless the V21 worker chooses to run existing local
  smoke commands after `make boundary-check`.
- No provider API execution.
- No firmware, hardware, StackChan, serial, NVS, or Gateway runtime work.
- No deletion, prune, report cleanup, or rewrite of existing V21 LAN UI dirty
  changes.
- No storage of query text, answer text, evidence bodies, full URLs, local
  paths, credentials, transcripts, raw audio, or provider output in reports or
  docs.

## Worker Execution Steps

1. In V21, read `AGENTS.md`, `docs/core/00-INDEX.md`, relevant API/contract
   docs, and current git status.
2. Create/update V21 plan:
   `docs/plans/2026-06-04-a21-v2-workspace-query-scope-native-contract.md`.
3. Add tests around `/internal/v1/knowledge/voice-query` proving additive A21
   v2 fields are accepted, validated, and returned only as safe metadata.
4. Implement the smallest native V21 changes:
   request/response structs, schema updates, validation, response metadata,
   and redaction-safe tests.
5. If source-scope enforcement is not possible in this slice, document the
   exact schema/query gap and return a stable status such as
   `scope_contract_ready_acl_pending` instead of `searchable`.
6. Run focused tests first, then V21's required verification as feasible:
   `make test`, `make check`, `make compose-config`.
7. Commit and push a scoped V21 worker branch if verification passes.

## Acceptance Conditions

- A V21 request with A21 v2 fields does not fail as an unknown or malformed
  request.
- Invalid `query_scope` is rejected before retrieval.
- Response metadata includes only counts/status, never evidence bodies beyond
  the already approved voice-query response contract.
- `public_only` never claims personal result counts unless V21 has a real,
  tested source-scope mapping.
- Tests prove the contract behavior.
- V21 handoff states whether personal/public ACL enforcement is complete or
  pending.

## Worker Result

- V21 branch:
  `origin/codex/a21-v2-workspace-query-scope-native-contract`.
- V21 commit:
  `ad61246 feat(voice-query): accept A21 workspace scope metadata`.
- V21 `/internal/v1/knowledge/voice-query` now natively accepts
  `device_id`, `user_id`, `workspace_id`, and `query_scope`.
- V21 validates `query_scope` as `public_only`, `personal_only`, or
  `personal_plus_public`.
- V21 responses include redacted `source_scope_counts` and `workspace_status`.
- Current truthful status is `scope_contract_ready_acl_pending` because V21's
  current access classes do not prove A21 public/personal ACL enforcement.
- Worker verification:
  - `cd backend && go test ./internal/httpapi`: passed.
  - `cd backend && go test ./...`: passed.
  - JSON schema parse: passed.
  - `git diff --check`: passed.
  - `make compose-config`: passed.
  - `make test` and `make check`: blocked at `check-toolchain` because the
    installed Node runtime was `v25.8.0` while the Makefile required
    `v24.15.0`.
  - `ALLOW_BRANCH_MISMATCH=true make boundary-check`: blocked by existing
    Compose project `v21air-lan-demo` owning Jaeger UI port `16687`; no
    services were started or stopped.
- Worker avoided A21 code changes, A21 Gateway runtime, StackChan hardware,
  firmware, provider execution, V21 service startup, cleanup/prune, and dirty
  V21 LAN/desktop connector files.

## Failure States

- `v21_dirty_tree_conflict`: the existing V21 LAN UI dirty work prevents a clean
  worker branch.
- `v21_scope_schema_gap`: current schema cannot distinguish public and personal
  sources without migration/ADR.
- `v21_contract_regression`: existing X21/V21AIR voice-query callers break.
- `v21_redaction_violation`: report/stdout/docs leak query, answer, evidence,
  credential, URL, path, transcript, raw audio, or provider output.

## Rollback Path

Revert only the scoped V21 worker commit. Do not revert A21 internal test 3
voice/protocol work, A21 internal test 4 roleplay/professional/workspace
contracts, V21 LAN discovery dirty work, or V21 release/ingestion history.

## Required Worker Summary

- Transition
- What changed
- Files changed
- Tests run and results
- Runtime evidence
- Deviations from plan
- Remaining issues
- Next suggested action
- Forbidden actions avoided
- Commit and push status
