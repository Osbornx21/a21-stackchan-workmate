# A21 Workflow State Machine Plan

Status: active plan.
Date: 2026-06-02.
Transition: `T-GOV-001`.

## Background And Problem Definition

A21 has been moving quickly across Gateway, providers, firmware, hardware, and
official StackChan integration. Long Codex conversations and background workers
can lose context through compaction, stalled turns, or branch/worktree drift.
The project needs repository-carried state so any new model or worker can
resume from the latest explicit transition instead of rediscovering or
re-implementing finished work.

## Current System State

- The control branch before this plan is
  `codex/a21-hardware-window-20260602-stackchan-prd`.
- The current baseline commit before governance docs is
  `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`.
- A correct A21 firmware candidate build lane exists:
  `a21-stackchan-official-xiaozhi-compatible`.
- Host-side focused firmware/app tests passed before this governance transition.
- Full physical PRD acceptance remains pending until real StackChan flash and
  evidence.
- Existing control docs describe branch/tool/hardware discipline, but there is
  no standalone repo-level handoff log or project state-machine document.

## Target State

- `AGENTS.md` defines the workflow discipline but does not become a long log.
- `docs/agent_handoff_log.md` records per-round handoffs.
- `docs/project_state_machine.md` records project/module state and transitions.
- `docs/plans/` contains detailed transition plans before large work begins.
- New workers can recover by reading the repository state rather than relying
  on one long conversation.

## Non-Goals

- Do not modify Gateway, provider, V21, firmware, test, or hardware behavior.
- Do not run provider, V21, Gateway runtime, hardware, NVS, flash, or Mac-audio
  commands.
- Do not claim PRD physical acceptance from docs, host tests, or build
  candidates.
- Do not store secrets, local credentials, raw transcripts, or private runtime
  URLs in docs.

## Impact Scope

Expected files:

- `AGENTS.md`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-a21-workflow-state-machine.md`

Expected branch:

- `codex/a21-workflow-state-machine-20260602`

## Phased Execution

### Phase 1 - Establish Rules

Actions:

- Add a concise control-tower workflow section to `AGENTS.md`.
- Keep it limited to durable rules and recovery discipline.

Acceptance:

- `AGENTS.md` points to the handoff log, state machine, and plan directory.
- No long run log is embedded in `AGENTS.md`.

### Phase 2 - Create Handoff Log

Actions:

- Create `docs/agent_handoff_log.md`.
- Add the first transition entry for `T-GOV-001`.
- Record current repo state, prior firmware candidate status, unfinished
  verification, risks, and next action.

Acceptance:

- A model with no previous conversation can identify the current state and next
  action from the log.

### Phase 3 - Create State Machine

Actions:

- Create `docs/project_state_machine.md`.
- Record project state, module states, active transition, completed
  transitions, blocked transitions, and next candidate transitions.

Acceptance:

- The document distinguishes host/build readiness from physical PRD acceptance.
- The next three transitions are explicit and actionable.

### Phase 4 - Verify And Commit

Actions:

- Run `git diff --check`.
- Confirm only governance/docs files changed.
- Commit as `docs(control): add handoff and state machine workflow`.

Acceptance:

- Worker branch is clean after commit.
- Control tower receives commit hash, file list, validation results, and next
  transitions.

## Rollback Plan

- Revert the governance docs commit if the process proves too heavy or
  conflicts with A21 execution.
- If only one file needs rollback, remove the corresponding doc and remove its
  reference from `AGENTS.md`.

## Risks

- Over-governance can slow product work if every small task becomes a planning
  ceremony.
- Logs can become stale if workers forget to update them before handoff.
- State documents can falsely green hardware if they do not separate host,
  candidate, and physical evidence.
- Background workers can still drift if they ignore branch/worktree ownership.

## Human Confirmation Points

- Confirm whether this workflow should be mandatory for every small edit or
  only for substantial transitions.
- Confirm whether the next hardware window should prioritize full official
  candidate flash or another host/provider verification pass first.
- Confirm whether future handoff logs should be appended in Markdown or split
  into dated files after the first few entries.

