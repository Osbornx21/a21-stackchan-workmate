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
- Gateway records `v21.query.start`, `v21.query.first_result`, and `v21.query.error` trace markers.
- V21 failure returns an honest professional fallback instead of pretending retrieval succeeded.

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
- V21 trace markers

Run:

```bash
go test ./internal/v21adapter ./internal/gateway
make verify
```

## Next

- Add configurable `A21_V21_ADAPTER_URL` with preflight validation.
- Add V21 health check to `doctor`.
- Add timeout and latency metric for `a21_v21_query_ms`.
- Add simulator UI rendering for evidence cards.
- Add real adapter smoke only after the Shanghai/V21 runtime endpoint is explicitly identified.
