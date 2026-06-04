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
- Roleplay voice-pipeline turns now carry the selected safe
  `voice_clone_profile` into the provider-neutral request and TTS boundary;
  reports expose only the profile ID and `voice_clone_sample_not_recorded`.
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
- Gateway now exposes `GET /v1/workspace-sources` as a memory-only source
  readiness registry derived from upload/import job metadata. It returns
  redacted source IDs, job IDs, user/workspace labels, source scope, source
  kind, document label, content type, size, readiness, index status, and
  source-scope counts. A narrow `mark_searchable`/`mark_indexed_metadata_only`
  job action promotes a source to `searchable_metadata_only` for internal test
  readiness without running real indexing or storing document contents.
- Gateway now exposes `POST /v1/workspace-documents` as the local upload intake
  slice. It accepts multipart file bytes, stores them under the A21 Gateway
  runtime store, and returns only safe document/source/job metadata:
  `document_id`, `source_id`, `job_id`, `document_hash`, byte size,
  `storage_status=stored_local`, and `readiness=stored_local_pending_index`.
  It does not parse, chunk, embed, OCR, index, upload to cloud storage, or
  execute V21; API responses, traces, simulator readouts, and tests keep raw
  content, base64 payloads, original private filenames, local paths,
  credentials, provider output, and evidence bodies out of the surface.
- Gateway now exposes `GET/POST /v1/workspace-index-jobs` as the no-execute
  indexing request ledger for stored local documents. It verifies the stored
  local file exists, records a redacted `index_job_id`, and promotes linked
  document/job/source readiness to `indexing_requested_no_execute` while
  keeping `execution_started=false`, `v21_execution_allowed=false`, and
  `searchable=false`. Professional workspace readiness can now distinguish
  stored-local pending index from indexing-requested-without-execution.
- Gateway now exposes a professional mode ritual contract in
  `GET/POST /v1/voice-modes`. Selecting `professional` returns a `PRO` screen
  label, evidence-first cue text, professional expression, trace marker,
  `professional_only` workspace policy, `v21_allowed=true`, and
  `physical_accepted=false`; the simulator displays the selected mode cue.
- Gateway now exposes `GET /v1/professional-read-records` as a memory-only
  professional read ledger. Mock professional turns and the stock Xiaozhi
  professional route start a record before V21 query and complete or fail it
  with only safe scope/status/count/timing metadata: no utterance text,
  retrieved text, evidence bodies, screen-card text, speech blocks, provider
  output, document text, URLs, paths, credentials, voice transcript, or audio.
- The simulator now has a Workspace Audit surface. It can create the existing
  no-execute workspace upload job, show the last job status, manually refresh
  professional read records, and refresh the read ledger after professional
  evidence appears. The surface displays only safe metadata: count, status,
  query scope, utterance bucket, source-scope counts, workspace status, and
  privacy scope.
- Explicit user-spoken trigger phrases now bridge the default companion path
  into professional mode. `/v1/mock-turn` and stock `/v1/xiaozhi` turns in
  default/roleplay/workmate/companion context route to the professional evidence
  path when the recognized text contains phrases such as "专业模式",
  "认真查一下", "帮我查 V21", or "给我证据". Negated phrases such as
  "不要进专业检索" remain out of V21. Xiaozhi trigger routing reuses the
  streaming ASR final text when present instead of running a second batch ASR.
  Traces record only safe markers:
  `professional.voice_trigger.detected` and
  `xiaozhi.professional_route.voice_trigger`.
- V21 scoped worker branch
  `origin/codex/a21-v2-workspace-scope-retrieval-guard` at commit `fccd0ac`
  now makes native V21 `/internal/v1/knowledge/voice-query` execute A21 v2
  `query_scope` against classified `source_scope=public|personal` evidence
  before answer generation. The worker passes A21 scope metadata into
  retrieval requests, carries `source_scope` through HTTP/Qdrant/Postgres
  retrieval paths, filters `public_only`, `personal_only`, and
  `personal_plus_public`, fails closed for unclassified evidence in scoped A21
  responses, and returns classified `source_scope_counts` with
  `workspace_status=searchable`. Legacy requests without `query_scope` remain
  `scope_contract_ready_acl_pending`. This is source-scope guard evidence, not
  full tenant/account ACL, real personal upload indexing, V21 merge/release, or
  physical StackChan professional consult acceptance.
- A21 local `v21-adapter-bridge` now uses V21 native
  `/internal/v1/knowledge/voice-query` as its primary professional query path,
  passes `device_id`/`user_id`/`workspace_id`/`query_scope`, mirrors
  V21-returned `source_scope_counts` and `workspace_status`, and keeps direct
  retrieval only as a controlled no-evidence expansion fallback. This prevents
  A21 from bypassing the V21 source-scope guard on the normal professional
  consult path, but it is still adapter-boundary contract evidence rather than
  V21 merge/release, real indexing, or physical consult acceptance.
- Roleplay device-state reflection now exposes the selected safe role soul,
  scenario, memory readiness/count, voice clone, and
  `roleplay_physical_accepted=false` through `/v1/devices` and the simulator
  registry panel. It records only IDs, booleans, counts, and non-storage flags;
  it does not expose prompt bodies, memory text, transcripts, provider output,
  voice-clone samples, or physical acceptance.
- Roleplay official expression planning now exposes a safe no-send
  `expression_plan` in `/v1/roleplay-profile`. The plan maps the selected role
  soul, scenario, memory readiness, and voice-clone profile to existing
  official StackChan action metadata with action phases, packet counts,
  semantic surfaces, and redaction flags. It keeps
  `delivery_policy=no_send_plan_only` and `physical_accepted=false`; it is not
  hardware delivery or roleplay physical acceptance.
- Gateway now exposes `GET /workspace` as the first product-oriented
  workspace console over existing safe APIs. It lets a user select query
  scope, upload a local document, request no-execute indexing, refresh source
  readiness, refresh professional read records, and see the
  roleplay/professional boundary without adding a new service, port,
  dependency, real indexing, provider/V21 execution, or hardware action.
- The workspace console now also has first-pass management controls: delete
  selected source/job through the existing no-execute upload-job lifecycle,
  export client-side safe metadata as `a21.workspace_console_export.v1`, and
  filter professional read records by `record_id`, `trace_id`, or
  `session_id`. These controls do not parse, index, upload, retrieve, call
  V21/provider services, or claim physical acceptance.
- The workspace console now also exposes roleplay setup controls over existing
  safe contracts: role soul, scenario, voice profile, bounded memory hint, and
  memory clear. Saving those controls writes `/v1/roleplay-profile`, refreshes
  `/v1/voice-chain-profiles`, and reflects the selected values in the
  existing runtime prompt/voice pipeline readiness without exposing memory
  text, prompt bodies, voice data, provider output, V21 evidence, or physical
  acceptance.
- The workspace console now also exposes voice-chain and wake-word setup over
  existing safe contracts. Voice-chain controls select `cascade` or
  `realtime`, ASR profile, LLM profile, realtime provider, and effective TTS
  readout through `/v1/voice-chain-profiles` without running a provider.
  Wake-word controls save/reset `/v1/wake-word` intent while showing the
  honest activation boundary: custom MultiNet requests are
  `pending_firmware_build`, built-in Xiaozhi WakeNet remains active, runtime
  hot swap is false, and physical acceptance is still separate.
- The workspace console now also exposes a safe Voice Probe panel. Roleplay
  probe uses existing `/v1/fast-companion/turn` boundary metadata to show the
  selected role soul, scenario, voice profile, memory count, prompt readiness,
  and trace markers. Professional probe uses existing `/v1/mock-turn`,
  `/v1/traces`, and `/v1/professional-read-records` to show professional
  route/read metadata. It does not add a backend route, execute real provider
  or V21 services, index documents, touch firmware/hardware, or expose raw
  utterance, prompt, evidence, document, credential, or audio content.
- Product readiness and server-side readiness now ingest the existing static
  no-execute selected voice-chain capability report
  `a21.xiaozhi_streaming_provider_readiness.v1` through
  `--voice-chain-readiness-report` or `--use-latest-reports`. The report must
  match the current Gateway-selected ASR, LLM, and TTS profiles before
  `static_capability_ready` becomes true; mismatches stay visible as safe
  findings and do not get absorbed as current-chain readiness. This is
  metadata evidence only, not provider execution, V21 execution, Gateway
  deployment, ECS acceptance, firmware/hardware action, audible playback, or
  physical StackChan PRD acceptance.
- Product readiness now also exposes a top-level `roleplay` readiness object
  from `GET /v1/roleplay-profile` when the Gateway exposes it. It reports safe
  role soul, scenario, voice-clone profile, soul prompt readiness, prompt
  composed status, memory configured/readiness/count, official expression-plan
  action/packet counts, redaction flags, and physical acceptance truth. This
  makes roleplay immersion measurable in the launch report while still not
  executing providers, V21, voice-clone CLI, Gateway deployment, ECS, firmware,
  serial, NVS, or physical StackChan roleplay acceptance.
- Product readiness and server-side readiness now also accept
  `--roleplay-voice-report` and latest
  `a21-roleplay-voice-probe-*.json` evidence. Matched safe reports mark
  `roleplay.voice_runtime_ready=true` only when the selected role soul,
  scenario, bounded memory prompt, voice-clone profile, prompt input,
  text-stream execution, audio downlink, and playback-start boundary reached a
  roleplay voice turn. This closes a reporting gap between workspace Voice
  Probe and launch readiness while still not executing providers/V21, accepting
  voice-clone audio quality, deploying Gateway/ECS, touching firmware, serial,
  NVS, or claiming physical StackChan roleplay acceptance.

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
- Users can see which uploaded/imported sources are metadata-only,
  searchable-metadata candidates, failed, or deleted before real indexing is
  implemented.
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
- Source registry can list public/personal source metadata and scope counts
  without document contents.
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
- `/v1/roleplay-profile` can set and clear bounded runtime memory hints for
  roleplay. Responses and simulator readouts show only readiness/counts and
  finding codes, not memory text.
- Fast-companion and stock `/v1/xiaozhi` roleplay turns pass the composed
  personality/scenario/memory prompt into the voice pipeline text provider
  while reports expose only readiness/redaction metadata.
- Fast-companion and stock `/v1/xiaozhi` roleplay turns pass the selected safe
  voice-clone profile into the TTS request, and trace only
  `roleplay.voice_clone_profile.used` when a clone profile is selected.
- Reports include profile IDs and counts only, not user text, memory content,
  prompt bodies, or cloned voice samples.
- Barge-in timing remains within the internal test 3 acceptance envelope.

Implementation slice:

- `T-ROLEPLAY-MEMORY-CONTROL-SURFACE-001` adds `memory_hints` and
  `clear_memory` to the existing roleplay profile contract. Runtime hints are
  sanitized with the personality memory policy, stored only in Gateway memory,
  and reused by fast-companion roleplay summaries without V21 execution.
- `T-ROLEPLAY-PROMPT-VOICE-PIPELINE-001` carries the composed roleplay prompt
  into the provider-neutral voice pipeline through a runtime-only prompt field;
  prompt bodies remain out of reports, traces, and API responses.
- `T-ROLEPLAY-VOICE-CLONE-PIPELINE-CONTRACT-001` carries the selected safe
  `voice_clone_profile` through the provider-neutral voice pipeline and TTS
  adapter request. Reports expose only the selected profile ID and the
  `voice_clone_sample_not_recorded` policy.

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
- Default/roleplay/workmate/companion voice turns can enter professional mode
  only through explicit user-spoken trigger phrases such as "专业模式",
  "认真查一下", "帮我查 V21", or "给我证据"; negated phrases and
  privacy/state modes must not route to V21.
- `/v1/voice-modes` exposes the same `PRO` checking cue contract used by
  hardware/web/app mode selection before V21 evidence is read aloud.
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
   - roleplay profile now selects A21-owned role-soul assets
     (`a21_roleplay_default`, `a21_roleplay_wry_peer`,
     `a21_roleplay_calm_anchor`) that compose into voice-pipeline prompt input;
   - professional-only V21 boundary unchanged.
2. Add v2 adapter-plan docs for query scope and workspace fields. Completed
   for the Gateway/V21 adapter contract; V21 repository implementation remains
   a separate worker task.
3. Add a no-execute upload/workspace PRD spec and API contract. Completed for
   `/v1/professional-workspace`, `/v1/workspace-upload-jobs`, and
   `/v1/workspace-sources`; local file upload intake is now completed through
   `/v1/workspace-documents` as `stored_local_pending_index`; no-execute index
   request readiness is now completed through `/v1/workspace-index-jobs` as
   `indexing_requested_no_execute`. Import, parsing, chunking, embedding,
   actual indexing, cloud storage, and V21 execution are still not implemented.
4. Add fake/fixture tests for:
   - roleplay voice-mode selection;
   - dialogue alias normalization;
   - roleplay fast-companion execution without V21 calls;
   - professional query-scope redaction.
5. Add V21-side worker tasks for upload/index/query-scope support in the V21
   repository without mutating A21 internals.
6. Re-run A21 product/server-side readiness and keep physical StackChan PRD
   acceptance separate.

Execution update:

- `T-ROLEPLAY-VOICE-PROBE-REPORT-GENERATOR-001` adds
  `a21 roleplay-voice-probe`, which generates the safe
  `a21-roleplay-voice-probe-*.json` runtime report from the existing Gateway
  roleplay voice path instead of relying on a hand-authored fixture. This closes
  the host/Gateway roleplay runtime evidence-generation gap while keeping
  physical StackChan, wake, audible voice-clone quality, and PRD acceptance as
  separate gates.
- `T-SERVER-SIDE-ROLEPLAY-VOICE-RUNTIME-GATE-001` makes matched ready roleplay
  voice runtime evidence a `server-side-readiness-bundle` candidate gate and
  wires `--collect-missing` to run `a21 roleplay-voice-probe --require-ready`.
  Provider/V21/host-voice evidence alone can no longer produce a server-side
  candidate without roleplay voice runtime.
- `T-SERVER-SIDE-PROFESSIONAL-RITUAL-EXECUTION-GATE-001` makes external
  Gateway professional ritual evidence a separate server-side candidate gate.
  Adapter smoke still proves the V21 boundary, but
  `professional_ritual_ready` now requires an accepted
  `a21.xiaozhi_professional_bench.v1` report with checking feedback, result
  ordering, stale-result suppression, abort stop, V21 execution, and redaction
  checks.
- `T-PROFESSIONAL-READ-RECORD-READINESS-GATE-001` makes the Gateway
  professional read ledger part of the same readiness evidence. External
  Gateway professional bench reports now include a safe `read_record` summary,
  and product/server-side readiness require a completed matching read record
  before the professional path can satisfy the server-side candidate gate.
- `T-ROLEPLAY-VOICE-RUNTIME-PROBE-CLOSURE-001` closes the local live roleplay
  runtime probe gap. The Gateway now traces selected voice-profile use and
  host/simulator playback-start in the roleplay voice-pipeline branch, and
  product readiness accepts safe single-token prompt-part IDs such as
  `core_identity`. A fresh local `roleplay-voice-probe --require-ready` passed;
  the refreshed server-side bundle now has only real provider smoke as the
  no-hardware server-side blocker.
- `T-STEPFUN-PROVIDER-SMOKE-SERVER-CANDIDATE-CLOSURE-001` closes that remaining
  no-hardware blocker. StepFun executed streaming provider smoke passed with
  `repeat=3`, and the local bundle with
  `--provider-smoke-report reports/provider-live/a21-provider-smoke-20260604-153030-291957000.json`
  plus `--require-candidate` returned `server_side_candidate_ready`. Full PRD
  launch still requires physical StackChan online evidence and physical PRD
  acceptance.
- `T-XIAOZHI-PHYSICAL-PRD-PROMOTE-GATE-001` adds the report-only promote
  boundary for stock Xiaozhi physical evidence. The new
  `a21 xiaozhi-physical-prd-review` command and
  `make xiaozhi-physical-prd-review` target consume matching
  `a21.xiaozhi_physical_evidence.v1` and
  `a21.xiaozhi_half_duplex_acceptance.v1` reports, require
  `ACCEPT_A21_XIAOZHI_PHYSICAL_PRD`, and only then write an accepted physical
  report for `product-readiness`. This closes the tooling gap without flashing,
  writing NVS, executing providers, or reinterpreting internal test 3
  candidate evidence.
- `T-OFFICIAL-XIAOZHI-COMPATIBLE-NVS-WIFI-OVERRIDE-001` unblocks the current
  foreground hardware window when preserved Wi-Fi is unavailable. The
  official-compatible NVS writer can now take explicit Wi-Fi SSID/password env
  or CLI options while preserving calibration, mutating only the Xiaozhi
  connection keys and requested Wi-Fi keys, and redacting credentials from
  stdout/reports.

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

After server-side provider/professional/roleplay/wake/voice-chain evidence
lands:

- `S-INTERNAL-TEST4-SERVER-SIDE-CANDIDATE-READY-PHYSICAL-STACKCHAN-PENDING`
