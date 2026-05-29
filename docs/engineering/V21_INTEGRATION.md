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

## Future Adapter Contract

Candidate endpoint:

```text
POST /a21/v21/query
```

Candidate request:

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

Candidate response:

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
