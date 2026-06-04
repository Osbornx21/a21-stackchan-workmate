# 2026-06-04 - A21 V21 Native Voice Query Bridge

Status: completed host-local adapter contract cut.
Owner: A21 control tower.
Transition: `T-A21-V21-NATIVE-VOICE-QUERY-BRIDGE-001`.

## Goal

Make the A21 local `v21-adapter-bridge` consume V21's native
`/internal/v1/knowledge/voice-query` contract as the primary professional
query path, so A21 benefits from the V21 `query_scope` and `source_scope`
guard instead of bypassing it with direct retrieval calls and inferred counts.

## Starting State

- A21 branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- A21 control HEAD before this cut:
  `731d4d8 docs(control): record v21 source scope guard`.
- V21 worker branch
  `origin/codex/a21-v2-workspace-scope-retrieval-guard` commit `fccd0ac`
  now filters scoped evidence before answer generation.
- A21's bridge still routes professional queries through
  `/api/v1/collections/{collection}/retrieval/query` and constructs
  `source_scope_counts` from `query_scope` plus evidence count. That bypasses
  V21 native source-scope enforcement.

## Boundary

Allowed:

- A21 app/adapter bridge code, tests, docs, and control logs.
- Host-local unit tests using `httptest`.

Forbidden:

- Do not start A21 Gateway as a runtime service.
- Do not start V21 Docker Compose or real local inference.
- Do not execute providers, ingest private documents, call ECS, flash firmware,
  touch serial, write NVS, or prune/gc.
- Do not claim durable account ACL, real personal indexing, cloud storage, or
  physical StackChan professional consult acceptance.

## Implementation Steps

1. Extend the A21 bridge's V21 voice-query DTO with `source_scope_counts` and
   `workspace_status`.
2. Send A21 v2 fields (`device_id`, `user_id`, `workspace_id`, `query_scope`)
   into V21 native voice-query.
3. Make the bridge handler call V21 native voice-query first.
4. Keep direct retrieval only as an expansion fallback for controlled
   no-evidence cases, and derive fallback counts only from result
   `source_scope` labels.
5. Add focused tests proving native voice-query is used, scope metadata is
   propagated, V21 counts/status are preserved, and direct retrieval is not
   called on the green path.

## Completed State

- A21 `v21-adapter-bridge` now calls V21 native
  `/internal/v1/knowledge/voice-query` before any direct retrieval fallback.
- Native requests carry `device_id`, `user_id`, `workspace_id`, and
  `query_scope`.
- Native responses mirror V21-returned `source_scope_counts` and
  `workspace_status`.
- Direct retrieval is retained only for controlled no-evidence expansion and
  can count source scopes only from result `source_scope` labels.

## Acceptance

- A21 `v21-adapter-bridge` calls `/internal/v1/knowledge/voice-query` for a
  normal professional query.
- The request carries redacted A21 v2 workspace and scope fields.
- The bridge response mirrors V21's safe `source_scope_counts` and
  `workspace_status`.
- The bridge no longer fabricates counts from `query_scope` on the normal path.
- Tests pass without starting external services or touching private data.

## Handoff Format

- Transition.
- What changed.
- Files changed.
- Tests run and results.
- Runtime or physical evidence.
- Deviations from plan.
- Remaining issues.
- Next suggested action.
- Forbidden actions avoided.
