# A21 V21 Integration

## Boundary

V21 is the professional knowledge retrieval system. A21 is the embodied voice/workmate system. A21 must not become a V21 voice shell and must not read V21 internals directly.

A21 may call V21 only through an explicit adapter boundary.

## Allowed

- A21 professional mode sends a user-confirmed query to a V21 adapter.
- V21 adapter returns answer blocks, evidence, confidence, citations, and screen-card data.
- A21 presents V21 results with professional tone and evidence-first behavior.
- A21 traces V21 query latency and failure codes.

## Forbidden

- Direct reads from V21 database files or internal tables.
- Reuse of V21 service names for A21 services.
- Reuse of V21 ports as A21 defaults.
- Sending private companion/roleplay content to V21 without explicit user confirmation.
- Treating V21 retrieval output as unverified emotional advice.

## Professional Mode Triggers

Examples:

- "专业模式"
- "认真查一下"
- "帮我查 v21"
- "用专业模式回答"
- "给我证据"

The mode switch must be visible in voice and screen state.

## Current Adapter Contract

Implemented client endpoint:

```text
POST /a21/v21/query
```

Implemented health endpoint:

```text
GET /healthz
```

Request:

```json
{
  "trace_id": "a21-trace-example",
  "session_id": "a21-session-example",
  "mode": "professional",
  "utterance": "帮我查一下以前语音唤醒这块是不是讨论过误触发",
  "latency_profile": "fast_first",
  "answer_style": "voice_first_with_citations",
  "max_first_response_ms": 1200,
  "privacy_scope": "professional_only"
}
```

Response:

```json
{
  "trace_id": "a21-trace-example",
  "fast_answer": "历史讨论主要集中在车内多人说话、相似音节误唤醒、连续对话残留监听三个场景。",
  "confidence": 0.82,
  "evidence": [
    {
      "title": "语音唤醒体验复盘",
      "type": "meeting",
      "source_id": "v21-doc-example",
      "summary": "提到多人说话导致误唤醒。"
    }
  ],
  "speech_blocks": [
    "我先说结论……",
    "第一，车内多人说话……"
  ],
  "screen_cards": [
    {
      "label": "结论",
      "text": "误唤醒集中在 3 类场景"
    }
  ],
  "follow_ups": [
    "要不要按车型展开？"
  ]
}
```

## Internal Test 4 Adapter v2 Direction

Internal test 4 keeps the v1 adapter smoke as accepted local boundary evidence,
and A21 now carries a scoped v2 professional query contract. A21 remains
ignorant of V21 internals while carrying enough redacted user/workspace context
for V21 or the adapter bridge to enforce permissions.

Additional request fields:

```json
{
  "device_id": "stackchan-example",
  "user_id": "redacted-user-label",
  "workspace_id": "redacted-workspace-label",
  "query_scope": "personal_plus_public"
}
```

Allowed `query_scope` values:

- `public_only`
- `personal_only`
- `personal_plus_public`

Additional response fields:

```json
{
  "source_scope_counts": {
    "public": 2,
    "personal": 3
  },
  "workspace_status": "searchable"
}
```

Rules:

- `roleplay` mode must not call V21, even when user memory or persona hints are
  available.
- `professional` mode must be user-confirmed before A21 sends a V21 query.
- `public_only` must never return personal source IDs.
- V21 scoped worker
  `origin/codex/a21-v2-workspace-scope-retrieval-guard` commit `fccd0ac`
  implements the first native source-scope guard: V21 carries A21 v2 scope
  metadata into retrieval requests, filters classified
  `source_scope=public|personal` evidence before answer generation, fails
  closed for unclassified evidence in scoped A21 responses, and can return
  `workspace_status=searchable` with nonzero classified counts. This is not yet
  durable tenant/account ACL or real personal upload indexing.
- V21 reports may include scope labels and counts, but not uploaded document
  text, evidence bodies, full source paths, credentials, local private URLs,
  prompts, transcripts, or provider output.
- Gateway exposes `GET/POST/PUT /v1/professional-workspace` as the no-execute
  selector for redacted `user_id`, `workspace_id`, and `query_scope`. This is
  not upload/index readiness.
- Gateway exposes `GET/POST/PUT /v1/workspace-upload-jobs` as the no-execute
  upload/import/index job lifecycle contract. It can create, poll, fail, retry,
  and delete redacted metadata jobs, but it does not store file bytes or execute
  V21 indexing.
- A21 hardware may initiate a professional consult only through Gateway/Core;
  StackChan does not store workspace documents, embeddings, provider keys, or
  V21 credentials.

## Failure Behavior

If V21 is unavailable:

- do not pretend retrieval succeeded
- keep the user in A21 experience
- say the professional system is unavailable
- offer to remember the query and retry
- log a structured failure code

Example:

"V21 现在没接上。我先把这个问题留住，等专业系统回来再查证据。"

## Current Implementation

Current Go package:

```text
internal/v21adapter
```

Implemented clients:

- `NewHTTPClient(baseURL)` posts to `/a21/v21/query`.
- `ProbeHealth(ctx, baseURL, httpClient)` checks adapter `/healthz`.
- `NewMockClient()` returns deterministic evidence, speech blocks, screen cards, and follow-ups for Gateway/simulator tests.
- `v21-adapter-smoke` emits a redacted readiness or execution report for the adapter boundary.
- `v21-professional-readiness` emits a host-only mock report for the professional-mode bridge: it records the local checking acknowledgement separately from mock evidence/cards/follow-ups, validates redaction, and never executes the adapter query path.
- `xiaozhi-professional-bench` emits a host-only `/v1/xiaozhi` professional runtime report. Without `--gateway-url` it starts an in-process Gateway with mock ASR/TTS and a fake V21 client; with `--gateway-url` it drives an already-running A21 Gateway, may observe the real A21 V21 adapter path through traces, and still stores only redacted timing/status/count fields.

Safety rules:

- Real adapter configuration uses `A21_V21_ADAPTER_URL`.
- Physical stock-xiaozhi acceptance may set
  `A21_XIAOZHI_STOCK_PROFESSIONAL_ROUTE=professional` to route stock
  `realtime`/`auto`/`manual` listen turns into the A21 professional path. This
  is an explicit Gateway-side override for bounded acceptance windows; it does
  not change stock firmware protocol, does not require `device_events` or
  `debug_metrics`, and should be replaced by a user-confirmed professional
  intent/tool contract before broad product use.
- The Gateway professional path must synthesize the checking cue, unavailable
  fallback, and evidence result through the configured A21 TTS adapter and
  paced xiaozhi OPUS downlink. Text/evidence JSON is metadata for the stock
  session and does not count as physical audible evidence by itself.
- `doctor` skips V21 health when `A21_V21_ADAPTER_URL` is unset.
- `doctor` checks `/healthz` when `A21_V21_ADAPTER_URL` is set, uses a direct no-ambient-proxy HTTP client, and redacts URL credentials from error details.
- `v21-adapter-smoke` does not execute a professional query unless `--execute` is present.
- `v21-professional-readiness` never supports `--execute`; `adapter_executed` remains `false` and adapter disabled/misconfigured states are reported with fixed findings.
- `xiaozhi-professional-bench --gateway-url <gateway>` may mark `v21_executed=true` only when the Gateway trace proves `v21.query.start` and `v21.query.first_result`. It never marks provider or hardware execution, and it always keeps `prd_accepted=false`.
- `v21-adapter-smoke` uses a direct HTTP client when executing so ambient `HTTP_PROXY`, `HTTPS_PROXY`, and `ALL_PROXY` do not catch local/LAN adapter traffic.
- `v21-adapter-smoke --output-dir reports` writes `reports/a21-v21-adapter-smoke-YYYYMMDD-HHMMSS.json` with a basename-only `report_path`.
- `v21-professional-readiness --output-dir reports` writes `reports/a21-v21-professional-readiness-YYYYMMDD-HHMMSS.json` with a basename-only `report_path`.
- `xiaozhi-professional-bench --output-dir reports` writes `reports/a21-xiaozhi-professional-bench-YYYYMMDD-HHMMSS.NNNNNNNNN.json` with a basename-only `report_path`.
- V21 smoke reports may include adapter name, protocol, status, configured/executed flags, endpoint host, fixed health/query paths, professional request contract labels, query scope, source-scope counts, workspace status, latency, confidence, response counts, and `redaction_ok`. They must not include the user query text, response text, full adapter URL, credentials, API keys, or V21 document content.
- `product-readiness --v21-adapter-smoke-report <report.json>` ingests an executed smoke report as real adapter-boundary evidence only when it proves the same professional contract (`mode=professional`, `latency_profile=fast_first`, `answer_style=voice_first_with_citations`, `privacy_scope=professional_only`, `max_first_response_ms=1200`), positive confidence, evidence/speech/card/follow-up counts, and report redaction. Internal test 4 readiness additionally needs the v2 scope fields (`query_scope`, `source_scope_counts`, and `workspace_status`) before cloud workspace readiness can be claimed. `product-readiness --v21-professional-report <report.json>` also accepts an `a21.xiaozhi_professional_bench.v1` external-Gateway report when it proves checking feedback, evidence/cards/follow-ups, redaction, and real V21 query markers. Health alone is not enough for launch readiness; the rollup still keeps `prd_accepted=false` and requires physical StackChan and voice evidence separately.
- When either executed adapter smoke or external-Gateway professional evidence is accepted, product-readiness mirrors only redacted booleans/counts/status and the basename-only source into `v21.v21_professional_execution`. Host-mock professional reports keep that section invalid and leave canonical `missing_real_evidence=v21_professional_execution` in place.
- Professional readiness reports may include acknowledgement timing, adapter configured/executed flags, evidence/card/follow-up availability booleans, counts, low-information evidence types, redaction status, and fixed findings. They must not include query text, retrieved text, prompts, transcripts, provider output, reasoning, full URLs, proxy values, local paths, or secrets.
- HTTP client applies professional defaults: `user_id=a21_local_user`,
  `workspace_id=a21_local_workspace`, `query_scope=public_only`,
  `mode=professional`, `latency_profile=fast_first`,
  `answer_style=voice_first_with_citations`,
  `privacy_scope=professional_only`, and `max_first_response_ms=1200`.
- HTTP client and bridge handler validate the professional query contract before
  network or retrieval execution: explicit non-`professional` mode, explicit
  non-`professional_only` privacy, invalid `query_scope`, unsafe user/workspace
  labels, and empty utterance are rejected locally.
- HTTP client rejects adapter URLs containing credentials.
- HTTP client rejects known X21/V21 internal legacy ports such as `8000`, `8080`, `18080`, `4173`, `42173`, `16686`, and `16687`. A21 must target an adapter boundary, not V21 internals.
- Gateway calls V21 only when the request mode is `professional`.
- Workmate, companion, co-creation, roleplay, focus, public, private, muted, and local-fallback mock turns do not call V21.
- Gateway professional responses carry explicit evidence fields in `control.event` payloads rather than flattening evidence into generic chat text.
- Gateway wraps V21 queries in a 3000 ms default timeout and records `v21.query.start`, `v21.query.first_result`, `v21.query.error`, or `v21.query.timeout` trace markers. Failure traces may add redacted low-information reason markers such as `v21.query.error.upstream_status`, `v21.query.error.contract_invalid`, `v21.query.error.no_evidence`, and `v21.query.error.status_5xx`; they must not include query text, response text, evidence bodies, full URLs, proxy values, local paths, or secrets.
- Gateway may record a coarse `v21.query.utterance.length_*` bucket for diagnostics, but must not store the ASR utterance text or an utterance hash in traces or reports.
- Gateway observes `a21_v21_query_ms_bucket` for professional-mode V21 adapter latency on success, error, and timeout.

Current Gateway professional sequence:

1. `listening`
2. `professional`
3. `speaking` with `fast_answer`, `confidence`, `evidence`, `speech_blocks`, `screen_cards`, and `follow_ups`

For stock xiaozhi physical turns, the shortest safe acceptance path is the
explicit stock route override above: Gateway maps the transport listen mode to
product `professional`, sends local checking feedback before ASR/V21 result
work, then queries V21 with the ASR-derived utterance under
`privacy_scope=professional_only`. Reports and traces must still omit raw ASR
text, prompt/provider output, evidence body, full URLs, proxy values, local
paths, and credentials.

## Adapter Smoke

Dry-run readiness:

```bash
go run ./cmd/a21 v21-adapter-smoke --output-dir reports
make v21-adapter-smoke
go run ./cmd/a21 v21-professional-readiness --adapter-url http://127.0.0.1:21121 --output-dir reports
```

Explicit execution after the A21 V21 adapter endpoint is identified:

```bash
A21_V21_ADAPTER_URL=http://127.0.0.1:21121 make v21-adapter-smoke-execute
go run ./cmd/a21 v21-adapter-smoke --adapter-url http://127.0.0.1:21121 --execute --output-dir reports
go run ./cmd/a21 product-readiness --v21-adapter-smoke-report reports/a21-v21-adapter-smoke-YYYYMMDD-HHMMSS.json --output-dir reports
```

This smoke proves only the A21 adapter boundary can accept the professional query contract and return structured counts. It does not prove V21 internal retrieval quality, citation truth, embedding/rerank behavior, production latency, or permission scope. Those remain V21-side acceptance concerns and must be checked with evidence-specific tests once the real Shanghai endpoint is available.
