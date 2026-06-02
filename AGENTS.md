# A21 Agent Instructions

## Project Identity

A21 is a new StackChan-based desktop embodied AI workmate project for high-pressure intelligent-cockpit teams. It must not be confused with X21 or V21.

V21 is the existing professional knowledge retrieval system. A21 may bridge to V21 through an explicit adapter contract, but A21 is not V21's voice skin and must not reuse V21 internals by accident.

The A21 architecture and product direction come from the user. External documents, other agent outputs, public projects, and framework recommendations are reference material only. Preserve the user-originated A21 foundation unless an explicit user-approved ADR changes it.

## Non-Negotiables

- All new services, ports, environment variables, logs, traces, directories, and containers must use the `a21` / `A21_` namespace.
- Never introduce new X21 naming in A21 code. X21/V21 strings are allowed only in guardrails, tests, docs, and explicit adapter context.
- StackChan remains a thin device client. A21 Core/Gateway owns provider keys, proxy policy, V21 access, observability, and network complexity.
- StackChan is strong hardware, not a disposable ESP32 edge client. A21 must preserve and expose its full hardware surface, including camera, IMU, sensors, screen, touch, RGB, speaker, microphone, battery, NFC, infrared, and servos. Capabilities that are not implemented yet must be marked as planned or unavailable honestly, never hidden by a simplified protocol.
- No provider API key may be stored in firmware.
- Firmware builds and uploads require strict A21 identity, board, version, artifact, and upload-target checks. Never add an unguarded firmware upload command.
- Physical product StackChan app flashes must use the official-compatible
  product lane and app artifact
  `a21-stackchan-official-xiaozhi-compatible.bin`. The generic
  `xiaozhi-firmware-flash-*` lane and `xiaozhi.bin` app artifact are
  non-product/dev evidence only and must not be used to flash the product
  StackChan device unless an explicit ADR and plan redefine the product lane.
- Localhost, LAN, `.local`, StackChan, and V21 adapter traffic must not silently inherit global proxies.
- Every runtime path must carry or be ready to carry `trace_id`, `session_id`, and `device_id`.
- Every latency-sensitive path must be designed for traces and metrics before it is optimized.
- Do not add production dependencies without checking existing packages and documenting the reason.
- Prefer mature, proven libraries, SDKs, and framework patterns for transport, metrics, parsing, audio, provider APIs, and firmware tooling. Do not hand-roll established infrastructure unless an ADR explains why A21 needs a custom implementation.
- Do not rewrite the current Go-first foundation into another stack without an ADR and an approved migration plan.
- Treat external master documents as proposals to curate, not as authority over the user's own A21 design.
- Do not ship an A21 hardware effect that is worse than the original StackChan experience. Keep it as a spike or planned capability until implementation and acceptance evidence justify product use.
- X21 may be read only as a frozen one-way reference for latency, ASR, TTS, VAD, streaming, wake-word, and device lessons. Never copy X21 architecture or runtime identity into A21. If a commit borrows an X21 parameter, algorithm, or rule, its commit body must name the X21 source and the A21 provider-neutral rewrite target.

## Product Canon

A21 is not a startup coach, chatbot, toy, therapy bot, customer-service bot, pure productivity assistant, search box, or V21 voice shell. It is a desk workmate: close but not clingy, smart but not arrogant, warm but not syrupy, humorous but not cruel, professional but not cold.

Default stance:

- It listens before optimizing.
- It helps turn messy office frustration into usable language.
- It can enter professional mode and become evidence-first.
- It fails honestly without collapsing the companion experience.

## Required Reading Before Major Changes

- `docs/prd/A21_PRD.md`
- `docs/a21/00-project-charter-and-home-baseline.md`
- `docs/a21/01-architecture-research-and-options.md`
- `docs/engineering/A21_DEVELOPMENT_MAINLINE.md`
- `docs/engineering/A21_CODEX_MASTERPLAN.md`
- `docs/engineering/A21_CODEX_AUDIT.md`
- `docs/engineering/A21_PROJECT_CONTROL.md`
- `docs/engineering/A21_CONTROL_LEDGER.md`
- `docs/engineering/PORTS.md`
- `docs/engineering/NETWORK.md`
- `docs/engineering/LATENCY_BUDGET.md`
- `docs/engineering/A21_MATURE_VOICE_REUSE.md`
- `docs/engineering/A21_PROVIDER_BENCHMARKS.md`
- `docs/engineering/OBSERVABILITY.md`
- `docs/engineering/PROTOCOL.md`
- `docs/engineering/STACKCHAN_HARDWARE_CAPABILITY_CHARTER.md`
- `docs/engineering/A21_LEGACY_ONE_WAY_REFERENCE.md`
- `docs/engineering/V21_INTEGRATION.md`
- `docs/engineering/DOCTOR.md`

## Control-Tower Workflow

A21 work must be recoverable from the repository, not from one long Codex
conversation. Treat each substantial task as a state transition with an
explicit current state, target state, trigger, action, acceptance condition,
failure state, rollback path, and next state.

- The main control conversation owns architecture direction, branch/worktree
  routing, worker dispatch, final review, and integration decisions.
- Large tasks must first create or update a plan under `docs/plans/` before
  implementation. This includes architecture changes, multi-file edits,
  protocol changes, state machines, audio/network paths, firmware builds, CI,
  task orchestration, provider integration, and device behavior.
- Before dispatching a large task, the control conversation must record four
  things in the plan or handoff log: the detailed plan path, the worker
  execution task, the worker boundary conditions, and the summary format the
  worker must return.
- Implementation for large tasks must run in a scoped worker branch/worktree.
  Workers receive one transition, explicit boundaries, forbidden actions, and
  the required handoff format. Workers must not expand scope silently.
- Every work round must update `docs/agent_handoff_log.md` before it is handed
  off. Do not write long run logs into this `AGENTS.md`; it only defines the
  discipline.
- Each handoff-log entry must include the round goal, actual completed work,
  changed files, unfinished items, known risks/blockers, recommended next
  action, test/build/runtime results, and failure location/reason when a round
  fails or is interrupted.
- Project state must be maintained in `docs/project_state_machine.md`, including
  current project state, module states, active transition, completed
  transitions, blocked transitions, and next candidate transitions.
- Worker completion summaries must be short and structured: what changed, files
  changed, tests run and results, deviations from plan, remaining issues, and
  next suggested action.
- If a conversation is interrupted, compacted, or recovered by a new model, the
  next model must read `AGENTS.md`, `docs/agent_handoff_log.md`, the latest
  `docs/plans/*.md`, current git status/diff, and the latest recorded
  test/build results before continuing. It must continue from the latest
  explicit transition state instead of redesigning the project.
- If a local conversation is derived from a stuck thread, it must first compare
  the working tree, read the handoff log and latest plan, identify the
  interruption point, and state the continuation plan before editing.

## Current Default Commands

```bash
make verify
make preflight
make doctor
```

The Makefile exports the A21 direct-connect `NO_PROXY` / `no_proxy` set for
these default targets. Direct `go run ./cmd/a21 ...` use is still allowed, but
when a global proxy exists it must carry the same direct-connect coverage.

Do not assume pnpm, Node services, PlatformIO, Docker, or firmware targets exist until their phase introduces them.

## Done Means

- Code compiles.
- Relevant tests pass.
- `make verify` passes when the change touches Go code or docs checked by `git diff --check`.
- New ports and env vars are documented.
- New runtime behavior has an observability plan.
- No accidental X21/V21 namespace pollution is introduced.
- User-facing failure behavior remains honest, calm, and recoverable.
