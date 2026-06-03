# Professional Mode Ritual Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make user-initiated professional mode switching visibly and audibly intentional without changing the accepted internal test 3 voice path.

**Architecture:** Promote the existing professional checking cue into the `/v1/voice-modes` contract so web/app/simulator/hardware operators can share one redacted ritual definition. Keep the actual V21 execution path unchanged: roleplay remains the low-latency embodied default, and professional remains the explicit evidence-first adapter route.

**Tech Stack:** Go Gateway, A21 simulator HTML/JS, existing `v21adapter.ProfessionalCheckingFeedbackText`, protocol expression states, repo docs/state/handoff discipline, `go test`, `make verify`.

---

## Transition

Transition id: `T-INTERNAL-TEST4-PROFESSIONAL-MODE-RITUAL-001`

Current state:

- `/v1/voice-modes` lists `roleplay` and `professional`.
- Professional voice execution already sends checking feedback before V21 query.
- The mode-selection API and simulator do not expose a formal mode-switch ritual contract.

Target state:

- `/v1/voice-modes` returns a selected ritual and per-mode ritual metadata.
- Selecting `professional` returns a PRO screen label, evidence-first cue text,
  expression, trace marker, V21 policy, and physical acceptance flag.
- Simulator shows the selected ritual cue without logging prompts, evidence
  bodies, provider output, credentials, URLs, paths, or private workspace text.
- Fast-companion remains blocked when the selected voice mode is professional.

Boundaries:

- No provider or V21 execution.
- No Gateway service start.
- No firmware build, flash, serial, NVS, or physical hardware action.
- Do not modify the stock `/v1/xiaozhi` audio/barge-in behavior.

## Files

- Modify `internal/gateway/server.go`
- Modify `internal/gateway/server_test.go`
- Modify `internal/gateway/simulator.go`
- Modify `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Modify `docs/project_state_machine.md`
- Append `docs/agent_handoff_log.md`

## Task 1: Gateway Ritual Contract

- [x] Step 1: Add tests proving `/v1/voice-modes` exposes professional ritual metadata.

Run:

```bash
go test ./internal/gateway -run 'TestVoiceModesCatalogDefaultsToRoleplayAndListsProfessional|TestVoiceModeSelectionProfessionalReturnsRitualContract|TestFastCompanionRejectsProfessionalVoiceModeWithoutProviderOrV21Execution' -count=1
```

Expected before implementation:

- The new professional ritual test fails because fields are missing.

- [x] Step 2: Add `VoiceModeRitual` to the Gateway response.

Required fields:

```json
{
  "mode": "professional",
  "screen_label": "PRO",
  "cue_text": "我在查，先把证据和置信度拉出来。",
  "expression": "professional",
  "trace_marker": "professional.checking_feedback.sent",
  "workspace_policy": "professional_only",
  "v21_allowed": true,
  "physical_accepted": false
}
```

- [x] Step 3: Keep roleplay as default and keep professional blocked from the fast-companion endpoint.

Expected:

- Existing fast-companion rejection test stays green.

## Task 2: Simulator Ritual Readout

- [x] Step 1: Add simulator HTML/JS readout for the current ritual.

Required behavior:

- On voice-mode refresh/save, show the selected ritual cue.
- When professional is selected, screen badge remains `PRO`.
- Do not show user query text, workspace text, evidence bodies, URLs, paths, or credentials.

- [x] Step 2: Update simulator smoke test.

Expected:

- `TestSimulatorPageServed` asserts the new readout ID exists.

## Task 3: Docs, State, And Verification

- [x] Step 1: Update protocol and internal-test4 docs.
- [x] Step 2: Update current control, state machine, and handoff log.
- [x] Step 3: Verify.

Run:

```bash
go test ./internal/gateway -run 'TestVoiceMode|TestFastCompanionRejectsProfessionalVoiceMode|TestSimulatorPageServed' -count=1
go test ./internal/gateway -count=1
git diff --check
GOMAXPROCS=2 make verify
```

Expected:

- All pass.

Commit message:

```bash
git add internal/gateway/server.go internal/gateway/server_test.go internal/gateway/simulator.go docs/plans/2026-06-04-professional-mode-ritual-contract.md docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md docs/engineering/PROTOCOL.md docs/engineering/A21_CURRENT_CONTROL.md docs/project_state_machine.md docs/agent_handoff_log.md
git commit -m "feat(gateway): expose professional mode ritual contract"
```
