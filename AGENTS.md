# A21 Agent Instructions

## Project Identity

A21 is a new StackChan-based desktop embodied AI workmate project for high-pressure intelligent-cockpit teams. It must not be confused with X21 or V21.

V21 is the existing professional knowledge retrieval system. A21 may bridge to V21 through an explicit adapter contract, but A21 is not V21's voice skin and must not reuse V21 internals by accident.

The A21 architecture and product direction come from the user. External documents, other agent outputs, public projects, and framework recommendations are reference material only. Preserve the user-originated A21 foundation unless an explicit user-approved ADR changes it.

## Non-Negotiables

- All new services, ports, environment variables, logs, traces, directories, and containers must use the `a21` / `A21_` namespace.
- Never introduce new X21 naming in A21 code. X21/V21 strings are allowed only in guardrails, tests, docs, and explicit adapter context.
- StackChan remains a thin device client. A21 Core/Gateway owns provider keys, proxy policy, V21 access, observability, and network complexity.
- No provider API key may be stored in firmware.
- Localhost, LAN, `.local`, StackChan, and V21 adapter traffic must not silently inherit global proxies.
- Every runtime path must carry or be ready to carry `trace_id`, `session_id`, and `device_id`.
- Every latency-sensitive path must be designed for traces and metrics before it is optimized.
- Do not add production dependencies without checking existing packages and documenting the reason.
- Prefer mature, proven libraries, SDKs, and framework patterns for transport, metrics, parsing, audio, provider APIs, and firmware tooling. Do not hand-roll established infrastructure unless an ADR explains why A21 needs a custom implementation.
- Do not rewrite the current Go-first foundation into another stack without an ADR and an approved migration plan.
- Treat external master documents as proposals to curate, not as authority over the user's own A21 design.

## Product Canon

A21 is not a startup coach, chatbot, toy, therapy bot, customer-service bot, pure productivity assistant, search box, or V21 voice shell. It is a desk workmate: close but not clingy, smart but not arrogant, warm but not syrupy, humorous but not cruel, professional but not cold.

Default stance:

- It listens before optimizing.
- It helps turn messy office frustration into usable language.
- It can enter professional mode and become evidence-first.
- It fails honestly without collapsing the companion experience.

## Required Reading Before Major Changes

- `docs/a21/00-project-charter-and-home-baseline.md`
- `docs/a21/01-architecture-research-and-options.md`
- `docs/engineering/A21_CODEX_MASTERPLAN.md`
- `docs/engineering/A21_CODEX_AUDIT.md`
- `docs/engineering/PORTS.md`
- `docs/engineering/NETWORK.md`
- `docs/engineering/LATENCY_BUDGET.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/V21_INTEGRATION.md`
- `docs/engineering/DOCTOR.md`

## Current Default Commands

```bash
make verify
go run ./cmd/a21 preflight
go run ./cmd/a21 doctor
```

Do not assume pnpm, Node services, PlatformIO, Docker, or firmware targets exist until their phase introduces them.

## Done Means

- Code compiles.
- Relevant tests pass.
- `make verify` passes when the change touches Go code or docs checked by `git diff --check`.
- New ports and env vars are documented.
- New runtime behavior has an observability plan.
- No accidental X21/V21 namespace pollution is introduced.
- User-facing failure behavior remains honest, calm, and recoverable.
