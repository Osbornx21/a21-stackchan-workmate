# Professional Voice Trigger Route Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let explicit PRD trigger phrases such as "专业模式", "认真查一下", "帮我查 V21", and "给我证据" switch default roleplay/workmate voice turns into the professional evidence route.

**Architecture:** Add a small provider-neutral trigger classifier in Gateway that recognizes only explicit professional phrases, not generic serious topics. Apply it to `/v1/mock-turn` and stock `/v1/xiaozhi` roleplay/workmate turns after ASR text is available. When a trigger is detected, reuse the existing professional ritual, V21 adapter contract, read-record ledger, and downlink behavior.

**Tech Stack:** Go Gateway, existing Xiaozhi streaming ASR final, existing professional V21 adapter path, existing read-record ledger, Gateway tests, repo docs/state/handoff discipline.

---

## Transition

Transition id: `T-INTERNAL-TEST4-PROFESSIONAL-VOICE-TRIGGER-001`

Current state:

- Users can select professional mode through `/v1/voice-modes`, `/v1/mock-turn` mode, and explicit Xiaozhi listen mode.
- The default roleplay/workmate path does not intentionally route PRD trigger phrases into professional mode.
- Stock Xiaozhi professional route can be forced by a separate stock override, but that does not prove user-spoken trigger phrases in the default companion route.

Target state:

- `/v1/mock-turn` with default/roleplay/workmate/companion mode and explicit trigger text routes to professional mode.
- Stock `/v1/xiaozhi` default roleplay/workmate turns route to professional when the streaming ASR final contains an explicit trigger phrase.
- The professional path reuses the streaming ASR final when available instead of running a second ASR pass.
- Trace markers distinguish `professional.voice_trigger.detected` and `xiaozhi.professional_route.voice_trigger`.
- Privacy modes without an explicit trigger and negated phrases such as "不要进专业检索" do not call V21.

Boundaries:

- No provider or real V21 execution.
- No Gateway service start beyond optional local smoke.
- No firmware build, flash, serial, NVS, ECS change, or physical hardware action.
- Do not change accepted internal test 3 Xiaozhi audio/barge-in/Opus behavior.

## Files

- Modify `internal/gateway/server.go`
- Modify `internal/gateway/server_test.go`
- Modify `docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md`
- Modify `docs/engineering/PROTOCOL.md`
- Modify `docs/engineering/A21_CURRENT_CONTROL.md`
- Modify `docs/project_state_machine.md`
- Append `docs/agent_handoff_log.md`

## Task 1: Tests

- [x] Step 1: Add a mock-turn test proving "认真查一下" in roleplay/workmate mode reaches the professional path, calls V21, emits professional ritual/evidence, writes read records, and does not leak utterance text in traces.
- [x] Step 2: Add a classifier/negative test proving "不要进专业检索" does not trigger V21.
- [x] Step 3: Add a Xiaozhi WebSocket test proving a default `realtime` listen turn with streaming ASR final "给我证据 ..." routes to professional without enabling stock override.

Run:

```bash
go test ./internal/gateway -run 'TestProfessionalVoiceTrigger|TestXiaozhiWebSocketVoiceTrigger' -count=1
```

Observed before implementation:

- New tests failed to build at `undefined: professionalVoiceTrigger`.

## Task 2: Gateway Implementation

- [x] Step 1: Add helper functions:

```go
func professionalVoiceTrigger(text string) bool
func professionalVoiceTriggerModeAllowed(mode protocol.Mode) bool
```

Explicit trigger phrases:

- `专业模式`
- `进入专业模式`
- `认真查`
- `认真查一下`
- `帮我查 v21`
- `帮我查v21`
- `查 v21`
- `查v21`
- `给我证据`

Negative guard examples:

- `不要进专业检索`
- `不用专业模式`
- `别查 v21`

- [x] Step 2: In `mockTurnResponse`, if mode is default roleplay/workmate/companion and the trigger is present, record `professional.voice_trigger.detected`, force `req.Mode=professional`, and call `professionalTurnResponse`.
- [x] Step 3: In `writeXiaozhiTTS`, if a non-professional turn has a trigger in `streamingASRFinalText`, set the current turn mode to professional, record `professional.voice_trigger.detected` and `xiaozhi.professional_route.voice_trigger`, and call `writeXiaozhiProfessionalTTS`.
- [x] Step 4: In `xiaozhiProfessionalASRFinal`, return `task.streamingASRFinalText` first when available so the triggered professional route does not run duplicate ASR.

## Task 3: Docs, State, And Verification

- [x] Step 1: Update protocol and internal-test4 docs.
- [x] Step 2: Update current control, project state machine, and handoff log.
- [x] Step 3: Verify:

```bash
go test ./internal/gateway -run 'TestProfessionalVoiceTrigger|TestXiaozhiWebSocketVoiceTrigger|TestProfessionalMode|TestWorkmateModeDoesNotCallV21Adapter|TestOfficePrivacyModesDoNotCallV21Adapter' -count=1
go test ./internal/gateway -count=1
git diff --check
GOMAXPROCS=2 make verify
```

Expected:

- All commands exit 0.

Commit message:

```bash
git add internal/gateway/server.go internal/gateway/server_test.go docs/plans/2026-06-04-professional-voice-trigger-route.md docs/plans/2026-06-04-internal-test4-cloud-mode-and-knowledge-workspace.md docs/engineering/PROTOCOL.md docs/engineering/A21_CURRENT_CONTROL.md docs/project_state_machine.md docs/agent_handoff_log.md
git commit -m "feat(gateway): route professional voice triggers"
```
