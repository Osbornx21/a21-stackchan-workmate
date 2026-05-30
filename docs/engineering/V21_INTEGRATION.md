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

Safety rules:

- Real adapter configuration uses `A21_V21_ADAPTER_URL`.
- `doctor` skips V21 health when `A21_V21_ADAPTER_URL` is unset.
- `doctor` checks `/healthz` when `A21_V21_ADAPTER_URL` is set, uses a direct no-ambient-proxy HTTP client, and redacts URL credentials from error details.
- `v21-adapter-smoke` does not execute a professional query unless `--execute` is present.
- `v21-adapter-smoke` uses a direct HTTP client when executing so ambient `HTTP_PROXY`, `HTTPS_PROXY`, and `ALL_PROXY` do not catch local/LAN adapter traffic.
- `v21-adapter-smoke --output-dir reports` writes `reports/a21-v21-adapter-smoke-YYYYMMDD-HHMMSS.json`.
- V21 smoke reports may include adapter name, protocol, status, configured/executed flags, endpoint host, fixed health/query paths, latency, confidence, and response counts. They must not include the user query text, response text, full adapter URL, credentials, API keys, or V21 document content.
- HTTP client applies professional defaults: `mode=professional`, `latency_profile=fast_first`, `answer_style=voice_first_with_citations`, `privacy_scope=professional_only`, and `max_first_response_ms=1200`.
- HTTP client rejects adapter URLs containing credentials.
- HTTP client rejects known X21/V21 internal legacy ports such as `8000`, `8080`, `18080`, `4173`, `42173`, `16686`, and `16687`. A21 must target an adapter boundary, not V21 internals.
- Gateway calls V21 only when the request mode is `professional`.
- Workmate, companion, co-creation, roleplay, focus, public, private, muted, and local-fallback mock turns do not call V21.
- Gateway professional responses carry explicit evidence fields in `control.event` payloads rather than flattening evidence into generic chat text.
- Gateway wraps V21 queries in a 3000 ms default timeout and records `v21.query.start`, `v21.query.first_result`, `v21.query.error`, or `v21.query.timeout` trace markers.
- Gateway observes `a21_v21_query_ms_bucket` for professional-mode V21 adapter latency on success, error, and timeout.

Current Gateway professional sequence:

1. `listening`
2. `professional`
3. `speaking` with `fast_answer`, `confidence`, `evidence`, `speech_blocks`, `screen_cards`, and `follow_ups`

## Adapter Smoke

Dry-run readiness:

```bash
go run ./cmd/a21 v21-adapter-smoke --output-dir reports
make v21-adapter-smoke
```

Explicit execution after the A21 V21 adapter endpoint is identified:

```bash
A21_V21_ADAPTER_URL=http://127.0.0.1:21121 make v21-adapter-smoke-execute
go run ./cmd/a21 v21-adapter-smoke --adapter-url http://127.0.0.1:21121 --execute --output-dir reports
```

This smoke proves only the A21 adapter boundary can accept the professional query contract and return structured counts. It does not prove V21 internal retrieval quality, citation truth, embedding/rerank behavior, production latency, or permission scope. Those remain V21-side acceptance concerns and must be checked with evidence-specific tests once the real Shanghai endpoint is available.
