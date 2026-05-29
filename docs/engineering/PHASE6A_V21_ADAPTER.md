# Phase 6A: V21 Adapter Boundary

## Status

Implemented as the first professional-mode slice.

## Goal

A21 can enter professional mode without becoming a V21 voice shell. The Gateway calls a narrow V21 adapter contract, receives structured evidence, and returns professional screen/voice fields to StackChan or the simulator.

## Current Scope

- `internal/v21adapter` defines the request/response contract.
- `NewHTTPClient(baseURL)` posts to `/a21/v21/query`.
- `NewMockClient()` provides deterministic professional evidence for local Gateway and simulator flows.
- Gateway calls V21 only when `mode=professional`.
- Workmate mode does not call V21.
- Gateway emits professional control events with `confidence`, `evidence`, `speech_blocks`, `screen_cards`, and `follow_ups`.
- Gateway records `v21.query.start`, `v21.query.first_result`, `v21.query.error`, and `v21.query.timeout` trace markers.
- Gateway observes professional V21 query latency with `a21_v21_query_ms_bucket`.
- Gateway applies a 3000 ms default outer timeout around V21 adapter calls.
- V21 failure returns an honest professional fallback instead of pretending retrieval succeeded.
- `doctor` reports V21 adapter health when `A21_V21_ADAPTER_URL` is configured.
- V21 health failure details redact URL credentials.

## Boundaries

- A21 does not read V21 databases, indexes, files, chunks, or embedding internals.
- A21 does not reuse V21 service names or legacy ports.
- Companion/workmate/private content is not sent to V21 by default.
- The mock client is for local development and tests only; it is not evidence retrieval.

## Verification

Current tests cover:

- HTTP client request contract and professional defaults
- rejection of legacy internal ports
- deterministic mock evidence response
- Gateway professional mode evidence payload
- workmate mode V21 isolation
- V21 unavailable fallback copy
- V21 timeout fallback and cancellation
- V21 trace markers
- V21 latency metric
- simulator Professional Evidence panel availability
- doctor V21 health configured/skipped states
- doctor V21 health credential redaction

Run:

```bash
go test ./internal/v21adapter ./internal/gateway
make verify
```

## Next

- Add real adapter smoke only after the Shanghai/V21 runtime endpoint is explicitly identified.
