# 2026-06-04 - Workspace Console Product Surface

Status: completed and verified.
Owner: A21 control tower.
Transition: `T-INTERNAL-TEST4-WORKSPACE-CONSOLE-PRODUCT-SURFACE-001`.

## Goal

Make internal test 4 workspace progress visible as a real web/app product
surface instead of only a hardware simulator/debug panel.

The first product console should let a user:

- choose professional query scope;
- upload a local document through the existing safe intake API;
- request indexing through the existing no-execute index ledger;
- see source readiness and read-record status;
- see the roleplay/professional boundary without executing provider, V21, or
  hardware actions.

## Boundary

Allowed:

- Host-local Gateway route and static HTML/CSS/JS.
- Existing safe Gateway APIs:
  `/v1/professional-workspace`, `/v1/workspace-documents`,
  `/v1/workspace-index-jobs`, `/v1/workspace-sources`,
  `/v1/professional-read-records`, `/v1/roleplay-profile`, and
  `/v1/voice-modes`.
- Host-local tests and control docs.

Forbidden:

- Do not add a new service, port, Node build, package dependency, database,
  cloud storage, auth provider, provider execution, V21 execution, real
  indexing, Gateway deployment, ECS action, firmware build/flash, serial, NVS,
  prune/gc, or physical hardware action.
- Do not show document text, raw bytes, local paths, credentials, prompt text,
  transcripts, provider output, evidence bodies, voice samples, or audio.
- Do not claim uploaded documents are searchable; keep
  `indexing_requested_no_execute` honest.

## Implementation Steps

1. Add `GET /workspace` as a product-oriented web console route.
2. Build a compact app shell with three work zones:
   workspace setup, upload/index pipeline, and professional read/source
   readiness.
3. Wire controls to existing safe APIs with client-side state and status
   readouts.
4. Keep copy honest: local intake, no-execute index request, and
   physical/V21 acceptance not yet complete.
5. Add tests proving the page is served, names the relevant APIs, exposes core
   controls, and does not include forbidden/private payload terms.
6. Update protocol/current control/state/handoff after verification.

## Acceptance

- `GET /workspace` returns HTML with `data-testid="workspace-console-root"`.
- The page has controls for query scope, document label/file upload,
  upload, index request, sources refresh, and read-record refresh.
- The page references only existing A21 safe APIs and no new service/port.
- The page copy distinguishes `stored_local`, `indexing_requested_no_execute`,
  `searchable=false`, and `physical_accepted=false`.
- Focused Gateway tests, `git diff --check`, and `GOMAXPROCS=2 make verify`
  pass.

## Execution Update - 2026-06-04

- Gateway now serves `GET /workspace`.
- The console has workspace setup, upload/index, readiness, and mode-boundary
  zones.
- The page calls only existing safe Gateway APIs and adds no new service, port,
  dependency, auth provider, storage backend, or runtime process.
- Browser/Playwright verification on `1270x900` and `390x900` found no visible
  overflow. A dummy local upload through the page moved the UI from
  `stored_local_pending_index` to `indexing_requested_no_execute` with
  `searchable=false` after the no-execute index request.
- No provider, V21, ECS, firmware, serial, NVS, prune/gc, or physical hardware
  action occurred.

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
