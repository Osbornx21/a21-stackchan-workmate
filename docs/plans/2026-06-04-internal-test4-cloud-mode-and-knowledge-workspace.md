# 2026-06-04 - Internal Test 4 Cloud Mode And Knowledge Workspace

Status: active control plan.
Owner: A21 control tower.
Transition: `T-INTERNAL-TEST4-CLOUD-MODE-AND-KNOWLEDGE-WORKSPACE-001`.
Created: 2026-06-04 CST.

## Goal

Advance A21 from internal test 3 voice-main-chain evidence toward the full PRD
shape without regressing the accepted Xiaozhi/StackChan experience. Internal
test 4 makes two user-facing modes first-class:

- `roleplay`: the default A21 embodied role/personality mode with memory hints
  and voice-clone selection.
- `professional`: the user-confirmed evidence mode that may call the A21/V21
  adapter.

This plan also defines the target cloud product form: an A21 web/app knowledge
workspace where users upload documents, choose public-only or personal+public
query scope, and then consult through either the web/app surface or the A21
StackChan hardware.

## Current State

- Branch:
  `codex/a21-hardware-window-20260603-wifi-provisioning-flash`.
- Internal test 3 is accepted for voice-main-chain testing.
- Product StackChan Xiaozhi path, StepFun/DashScope cloud-edge evidence, and
  local V21 adapter-boundary execution evidence are preserved.
- At plan creation, the legacy mode contract still exposed `dialogue` as the
  default user-facing `voice_mode` and collapsed roleplay intent into that
  dialogue-era contract.
- V21 currently runs as a local/LAN Docker Compose service with heavy local
  inference and sensitive professional documents. The fresh A21 evidence proves
  only the adapter boundary, not a permanent public cloud topology.

## Execution Update - 2026-06-04

- A21 code now exposes `roleplay` and `professional` from
  `GET /v1/voice-modes`.
- The default selected `voice_mode` is `roleplay`.
- `dialogue` remains accepted as a backwards-compatible input alias and returns
  selected `roleplay`.
- `roleplay` is accepted by the fast-companion path, while selected
  `professional` still returns `409` before provider or V21 execution on that
  endpoint.
- Internal test 3 legacy runtime labels such as `workmate` and `companion`
  remain valid below the user-facing product mode contract.
- Gateway now exposes `GET/POST/PUT /v1/roleplay-profile` for the roleplay
  runtime slice. The endpoint selects the default roleplay profile, roleplay
  scenario/playbook, and voice-clone profile; voice-clone selection is bridged
  into `/v1/voice-chain-profiles`.
- Fast companion now returns a redacted `roleplay` runtime summary and records
  `roleplay.profile.ready` plus `roleplay.memory.ready` trace markers without
  storing prompt text, memory text, transcripts, provider output, voice-clone
  samples, or V21 evidence.
- Gateway now exposes `GET/POST/PUT /v1/professional-workspace` for the
  professional no-execute workspace contract. It selects redacted `user_id`,
  `workspace_id`, and `query_scope` (`public_only`, `personal_only`, or
  `personal_plus_public`) while keeping upload, indexing, and V21 execution
  separate gates.
- The A21/V21 adapter query contract now carries v2 fields:
  `device_id`, `user_id`, `workspace_id`, `query_scope`, source-scope counts,
  and workspace status. Gateway professional turns pass those fields to the
  adapter and trace only safe readiness/scope markers.
- Gateway now exposes `GET/POST/PUT /v1/workspace-upload-jobs` as a no-execute
  upload/import/index job skeleton. It can create, poll, mark failed, retry,
  and delete redacted metadata jobs while rejecting document text, bytes,
  base64 payloads, import URLs, local paths, credentials, and provider output.

## Product Form

Internal test 4 should be designed as a three-surface product instead of a
single voice shell:

1. **A21 Hardware Experience**
   - StackChan remains the embodied desk workmate.
   - Wake/touch/voice can switch between `roleplay` and `professional`.
   - `roleplay` uses personality constitution, role playbooks, memory hints,
     voice-clone profile, low-latency ASR/LLM/TTS, face, screen, servo, RGB,
     and barge-in.
   - `professional` gives a visible/audible checking cue, then reads a compact
     evidence-first answer and shows evidence cards.
   - The device never stores provider keys, V21 keys, uploaded documents, or
     long-term knowledge indexes.

2. **A21 Cloud/Web/App Workspace**
   - Users sign in and bind devices.
   - Users upload documents into a personal workspace.
   - Users can query `public_only`, `personal_only`, or
     `personal_plus_public` scopes.
   - The same workspace controls what the hardware may consult when the user
     switches to professional mode.
   - Upload/index status, source visibility, citations, delete/export, and
     device access are visible to the user.

3. **V21 Knowledge Service / Adapter**
   - V21 is not imported into A21 internals.
   - V21 becomes the professional retrieval backend behind an A21 adapter.
   - The adapter owns the stable A21 contract:
     `workspace_id`, `user_id`, `query_scope`, `privacy_scope`,
     `latency_profile`, evidence/cards/follow-ups, redaction, and timing.
   - Heavy local inference may remain a V21 worker implementation detail for
     private deployments. Cloud A21 should be able to call a managed V21
     service, a private V21 tenant, or a local/LAN V21 bridge through the same
     adapter contract.

## V21 Refactor Plan

### V21-1: Tenant And Knowledge Scope Boundary

Target:

- Split knowledge into `public` and user/workspace-owned `personal` corpora.
- Add ACL-aware query scope:
  - `public_only`
  - `personal_only`
  - `personal_plus_public`
- Keep private documents out of public answer contexts by default.

Acceptance:

- A query with `public_only` never returns personal source IDs.
- A query with `personal_plus_public` returns source scope metadata per
  evidence item.
- Reports store only counts, source scope labels, and redacted status fields.

### V21-2: Upload And Indexing API

Target:

- Add document upload/import jobs for small private corpora around the current
  expected 2 GB scale.
- Return job status and index readiness without exposing raw document text in
  A21 reports.

Acceptance:

- Upload job can be created, polled, failed, retried, and deleted.
- Index readiness is source-scope aware.
- A21 can show "uploaded / indexing / searchable / failed" without knowing V21
  internals.

### V21-3: Adapter Contract v2

Target request fields:

```json
{
  "trace_id": "a21-trace-example",
  "session_id": "a21-session-example",
  "device_id": "stackchan-example",
  "user_id": "redacted-user-label",
  "workspace_id": "redacted-workspace-label",
  "mode": "professional",
  "query_scope": "personal_plus_public",
  "privacy_scope": "professional_only",
  "utterance": "redacted at A21 report surfaces",
  "latency_profile": "fast_first",
  "answer_style": "voice_first_with_citations",
  "max_first_response_ms": 1200
}
```

Target response additions:

```json
{
  "source_scope_counts": {
    "public": 2,
    "personal": 3
  },
  "workspace_status": "searchable"
}
```

Acceptance:

- A21 can execute the adapter smoke without storing utterance text, retrieved
  text, full URLs, credentials, document paths, or private evidence bodies.
- Old v1 adapter smoke remains accepted for local internal test 3 evidence, but
  internal test 4 readiness requires v2 scope fields before claiming cloud
  workspace readiness.

## A21 Refactor Plan

### A21-1: Product Mode Contract v2

Target:

- `roleplay` and `professional` are the two user-facing modes.
- `dialogue`, `workmate`, `companion`, and `co_creation` remain compatibility
  aliases or lower-level playbooks under `roleplay`.
- `public`, `private`, `focus`, `muted`, `local_fallback`, and `error` remain
  state/policy fields rather than product modes.
- Mode switching is user-initiated through voice, touch, app, or web control.

Acceptance:

- `GET /v1/voice-modes` lists only `roleplay` and `professional`.
- `POST /v1/voice-modes {"voice_mode":"dialogue"}` is accepted as a backwards
  compatibility alias and returns selected `roleplay`.
- Dialogue/roleplay endpoints never call V21.
- Professional endpoints reject non-`professional_only` V21 privacy.

### A21-2: Roleplay Runtime

Target:

- Roleplay mode composes:
  - role/personality prompt assets;
  - memory hints;
  - scenario playbooks;
  - selected cloud/local voice-clone profile;
  - fast ASR/LLM/TTS chain;
  - embodied expression state.
- The roleplay lane keeps internal test 3's stock Xiaozhi state machine and
  barge-in behavior.

Acceptance:

- A roleplay turn can select a voice-clone profile without changing
  professional/V21 scope.
- Reports include profile IDs and counts only, not user text, memory content,
  prompt bodies, or cloned voice samples.
- Barge-in timing remains within the internal test 3 acceptance envelope.

### A21-3: Professional Cloud Workspace Route

Target:

- Add a professional request context that carries `workspace_id`,
  `query_scope`, and `source_scope_policy`.
- Hardware wake/touch can initiate professional consultation only after the
  user chooses or confirms professional mode.
- Public mode and private roleplay content do not become V21 query context
  automatically.

Acceptance:

- Hardware professional consult shows `PRO`, checking feedback, and evidence
  cards.
- Web/app professional consult and hardware consult use the same adapter
  contract.
- `public_only` and `personal_plus_public` scopes are observable in redacted
  traces and readiness reports.

### A21-4: Cloud Device Binding And Gateway Topology

Target:

- A21 Cloud owns user/device/workspace binding.
- A21 Gateway/Core remains the provider/V21/network/proxy owner.
- StackChan receives only a Gateway URL, session state, semantic expression
  events, and audio/control frames.

Acceptance:

- No provider/V21 credentials are present in firmware or OTA payloads.
- Device binding can be revoked without reflashing firmware.
- `gateway_profile` remains a transport selector and never becomes a product
  mode.

## Internal Test 4 Slice Order

1. Land the mode-contract v2 code/doc cut:
   - first-class `roleplay`;
   - `dialogue` compatibility alias;
   - roleplay profile/scenario/voice-clone runtime selector;
   - professional-only V21 boundary unchanged.
2. Add v2 adapter-plan docs for query scope and workspace fields. Completed
   for the Gateway/V21 adapter contract; V21 repository implementation remains
   a separate worker task.
3. Add a no-execute upload/workspace PRD spec and API contract. Completed for
   `/v1/professional-workspace` and `/v1/workspace-upload-jobs`; file
   upload/import/index job execution is still not implemented.
4. Add fake/fixture tests for:
   - roleplay voice-mode selection;
   - dialogue alias normalization;
   - roleplay fast-companion execution without V21 calls;
   - professional query-scope redaction.
5. Add V21-side worker tasks for upload/index/query-scope support in the V21
   repository without mutating A21 internals.
6. Re-run A21 product/server-side readiness and keep physical StackChan PRD
   acceptance separate.

## Not In Scope For This First Cut

- No firmware flash, NVS write, or product-lane artifact change.
- No rollback of internal test 3 protocol/audio changes.
- No direct V21 database reads from A21.
- No cloud account/payment/auth implementation in A21 Gateway during the mode
  contract cut.
- No repository pruning, report deletion, or worker worktree cleanup.

## Acceptance Conditions

- `git diff --check` passes.
- Focused tests for protocol/gateway mode contract pass.
- `GOMAXPROCS=2 make verify` passes.
- Current handoff log and project state machine name this transition.
- Internal test 3 evidence remains intact and is not reinterpreted as internal
  test 4 full PRD readiness.

## Failure States

- `mode_contract_regression`: `professional` can enter dialogue/roleplay
  endpoint execution or roleplay can call V21.
- `internal_test3_audio_regression`: Xiaozhi state-machine or barge-in tests
  fail after mode changes.
- `v21_boundary_leak`: A21 stores or logs V21 query/evidence text, full URLs,
  credentials, local paths, or private document contents.
- `scope_ambiguity`: professional queries cannot distinguish public-only from
  personal+public results.

## Rollback Path

Rollback only the internal test 4 mode/workspace contract cut if it breaks
tests. Do not revert internal test 3 voice-chain, StepFun/DashScope runtime,
V21 adapter-boundary evidence, firmware lane, or physical evidence commits.

## Next State

On this first cut:

- `S-INTERNAL-TEST4-MODE-CONTRACT-V2-READY-CLOUD-KNOWLEDGE-WORKSPACE-PLANNED-PHYSICAL-PENDING`

After V21 upload/query-scope and A21 professional workspace context land:

- `S-INTERNAL-TEST4-CLOUD-KNOWLEDGE-WORKSPACE-ADAPTER-READY-HARDWARE-CONSULT-PENDING`
