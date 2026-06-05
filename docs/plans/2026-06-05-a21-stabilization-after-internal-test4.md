# A21 Stabilization After Internal Test 4

Status: active stabilization plan.
Created: 2026-06-05.
Owner: main control thread.

Internal test 4 is the protected fallback baseline. The next phase is not more
feature expansion. The next phase is to regain control of the project and close
the P0 product regressions without destroying the published baseline.

Recovery baseline:
`docs/engineering/A21_INTERNAL_TEST4_RECOVERY_BASELINE.md`.

## Freeze Rules

- No new product features until P0 stabilization exits.
- No product flash unless the transition is explicitly rollback, recovery, or
  acceptance for a P0 fix.
- No provider chain changes until the voice-loop and realtime race risks are
  isolated.
- No new worker branch or worktree unless it has one transition, one owner, one
  rollback path, and a short handoff.
- No side-branch merge until the diff is classified against the internal test 4
  baseline.
- No long handoff expansion. New entries must be operational and compact.
- No secrets, Wi-Fi credentials, provider keys, or private transcripts in repo
  files or release assets.

## Current State

State:
`S-CODE-FREEZE-INTERNAL-TEST4-BASELINE-PROTECTED`

Trigger:
The internal test 4 release is published and usable as the floor, but user
reports show unresolved P0 regressions in power, voice, body parity, and
configuration.

Failure state:
Any change that worsens the published internal test 4 behavior, introduces a
new product flash lane, stores secrets in the repo, or makes rollback unclear.

Rollback path:
Return to commit `1387d58f364f6ae1c7258487fb5a9863567adc74` and the
`a21-internal-test4` release assets before continuing.

## P0 Transition Order

1. `T-A21-STABILIZE-CONTROL-SURFACE-001`
   - Action: freeze baseline, classify worktrees/branches, and stop uncontrolled
     expansion.
   - Acceptance: docs identify the protected baseline, active stabilization
     state, forbidden actions, and next P0 transitions.
   - Scope: docs and git hygiene only.

2. `T-A21-P0-POWER-BOOT-RCA-001`
   - Action: compare stock M5Stack StackChan, official Xiaozhi ESP32, and A21
     product boot/power state machines from button press through PMIC,
     launcher, AI.AGENT entry, and NVS restore.
   - Acceptance: one root-cause report with evidence and a minimal fix candidate
     or a decision to retain official front-end lifecycle unchanged.
   - Scope: no product flash until the report names the exact candidate and
     rollback path.

3. `T-A21-P0-VOICE-LOOP-RCA-001`
   - Action: isolate wake sensitivity, first-token delay, and self-reply loop
     against internal test 4, Gateway logs, Xiaozhi protocol events, and
     provider realtime state.
   - Acceptance: deterministic reproduction or log-proven non-reproduction,
     race-safe tests, and one minimal Gateway/provider fix if required.
   - Scope: no provider switch, no prompt/personality rewrite.

4. `T-A21-P0-BODY-PARITY-RCA-001`
   - Action: compare official StackChan body actions with A21 touch, RGB,
     vibration, and servo amplitude.
   - Acceptance: stock-vs-A21 matrix and exact transport decision for each
     effect: official front-end, Xiaozhi MCP, body relay, or unavailable.
   - Scope: no speculative hardware effects that feel weaker than stock.

5. `T-A21-P0-PROVISIONING-RCA-001`
   - Action: verify no-preloaded-Wi-Fi setup, official app binding, device-data
     parsing, and A21 Gateway handoff.
   - Acceptance: a user path from fresh/no-Wi-Fi to A21 usable mode, or a
     documented internal-test limitation with a guarded preload path.
   - Scope: no hidden credential persistence beyond the approved NVS path.

## Immediate Next Action

Complete `T-A21-STABILIZE-CONTROL-SURFACE-001`, commit it, and push the
stabilization branch. Then start P0 power boot RCA as the first code-adjacent
transition, because a device that cannot reliably power on cannot be accepted
as an internal-test product.
