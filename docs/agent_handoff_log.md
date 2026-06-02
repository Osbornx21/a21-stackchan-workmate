# A21 Agent Handoff Log

Status: active handoff document.
Last updated: 2026-06-02.

This log is the recovery surface for Codex workers and future control-tower
threads. Every work round must add or update an entry before handoff. Keep this
file operational and redacted: no provider keys, Wi-Fi credentials, raw
transcripts, full private URLs, or local secret values.

Each entry should include:

- goal;
- actual completed work;
- files changed;
- unfinished items;
- known risks or blockers;
- recommended next action;
- test, build, or runtime results;
- failure location and reason, when applicable.

## 2026-06-02 - T-GOV-001 - Establish Repo-Carried Workflow State

Goal:

- Move A21 from long conversation memory toward repository-carried state.
- Add a handoff log, project state machine, and plan directory so new workers
  can resume without redoing finished work.
- Keep this transition documentation-only.

Actual completed work:

- Added control-tower workflow rules to `AGENTS.md`.
- Created this handoff log as the canonical per-round recovery record.
- Created `docs/project_state_machine.md` with initial project/module states,
  active/completed/blocked transitions, and next candidate transitions.
- Created `docs/plans/2026-06-02-a21-workflow-state-machine.md` as the detailed
  plan for this transition.

Files changed:

- `AGENTS.md`
- `docs/agent_handoff_log.md`
- `docs/project_state_machine.md`
- `docs/plans/2026-06-02-a21-workflow-state-machine.md`

Current repository state:

- Control branch before this transition: `codex/a21-hardware-window-20260602-stackchan-prd`.
- Worker branch for this transition: `codex/a21-workflow-state-machine-20260602`.
- Baseline commit before governance docs: `987bbb0 feat(firmware): add official xiaozhi compatible stackchan build`.
- The correct A21 firmware candidate is `a21-stackchan-official-xiaozhi-compatible`.
- The firmware candidate build evidence recorded before this transition had
  `official_avatar_action_preserved=true`,
  `official_xiaozhi_start_preserved=true`, and
  `minimal_bridge_screen=false`. No flash or NVS write was executed.

Unfinished items:

- Mainline still needs a fresh `make verify` after the governance commit is
  integrated.
- Mainline still needs a fresh
  `make a21-stackchan-official-xiaozhi-compatible-build` if the control tower
  wants a report/artifact generated from the final integrated branch rather
  than the worker branch.
- Physical PRD acceptance remains pending until the official candidate is
  flashed and real StackChan evidence is recorded.

Known risks and blockers:

- Do not treat host/mock/candidate evidence as PRD physical acceptance.
- Do not mix governance documentation work with firmware, provider, Gateway, or
  hardware-write branches.
- The historical official StackChan source path may live near old X21 material;
  A21 may use it only as a read-only official-source export, never as an X21
  firmware package source.

Validation results for this transition:

- `git status --short --branch`: confirmed
  `codex/a21-workflow-state-machine-20260602` before docs edits.
- `git diff --check`: passed with no output before commit.
- Scoped secret scan over the changed docs found only redaction-rule wording,
  not actual credentials.
- No provider, V21, Gateway runtime, hardware, NVS, flash, or Mac-audio command
  should be executed by this transition.

Recommended next action:

- Integrate the governance-doc commit into the control branch.
- Run the documented host-only verification.
- Continue with the next explicit transition from `docs/project_state_machine.md`.
