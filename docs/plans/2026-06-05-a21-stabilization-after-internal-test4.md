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

## Lean Carve Override - 2026-06-05

Transition:
`T-A21-LEAN-CARVE-MAINLINE-001`.

Current state:
`S-INTERNAL-TEST4-FROZEN-HEAD-090ECA6-UNREVIEWABLE-MAINLINE`.

Target state:
`S-MAIN-LEAN-REVIEWABLE-PRODUCT-CORE`.

Trigger:
The internal test 4 branch is frozen and usable, but the repository has grown
into an unreviewable control surface: `cmd/a21` imports one `internal/app`
god-package containing product, lab, bench, demo, evidence, professional
acceptance, firmware, and report commands; `internal/gateway/server.go`,
`internal/app/app_test.go`, and `internal/gateway/server_test.go` exceed any
reasonable review unit; generated/runtime reports and local work artifacts are
still present as repository concerns.

Action:

1. Create the single allowed convergence branch `main-lean` from frozen HEAD
   `090eca6`.
2. Record the current product dependency set in `docs/lean/PRODUCT_SET.md`.
3. Split product `cmd/a21` from lab-only commands under `cmd/a21-lab` or
   non-product packages so the product binary no longer compiles
   demo/bench/evidence/professional scaffolding.
4. Split the oversized product and test files by existing responsibilities
   without changing runtime behavior.
5. Move local reports/runtime artifacts out of product governance by ignoring
   `reports/`, `.a21-run/`, `.a21-tmp/`, `.a21-tools/`, `dist/`, and
   `.playwright-cli/`; record irreversible removals in
   `docs/lean/CARVE_LOG.md` before deleting tracked files or local branches.
6. Replace dead governance references in `AGENTS.md` with the lean/current
   documents that still exist.
7. Add the local/CI `scripts/lean-gate.sh` required by the carve order.

Acceptance:

- `bash scripts/lean-gate.sh` exits 0.
- `cmd/a21` and `cmd/a21-lab` are separate; the product package set no longer
  contains files named for `bench`, `demo`, `evidence`, or `professional`.
- No production `.go` file under `internal` or `cmd` exceeds 800 lines; no
  test file exceeds 1000 lines.
- `git branch | wc -l` is at most 3 after branch convergence, and every deleted
  branch has one `CARVE_LOG.md` line.
- Product artifacts and runtime reports are ignored or out of the worktree.
- `make stackchan-fast-companion-turn` records true-provider answer first-audio
  p95 below 1500 ms and one barge-in check in `CARVE_LOG.md`.

Failure state:
Any product behavior rewrite without before/after evidence, any new feature or
provider/firmware lane, any unguarded product flash/upload command, any provider
secret or transcript stored in Git/report output, any X21 runtime identity added
outside guardrails/docs/tests/adapter context, or any deletion/rebase/branch
removal without a prior `CARVE_LOG.md` entry.

Rollback path:
Return to frozen HEAD `090eca6` or the protected internal test 4 release commit
`1387d58f364f6ae1c7258487fb5a9863567adc74`; do not reuse any partially carved
branch as a product flash source.

Execution task:
Current Codex session executes the transition on `main-lean`. No additional
Codex worker branch or parallel write-capable worktree is allowed in this
transition.

Boundary conditions:
No product feature work, no provider chain switch, no firmware build or flash,
no V21 internal access, no production dependency addition, no raw hardware write
command, no X21/V21 runtime namespace reuse, and no deletion of redline assets:
`internal/runtimeguard`, `internal/v21adapter`, `internal/protocol`,
`internal/transport/{xiaozhi,stackchan}`, real streaming provider/audio
pipeline code, and guarded official-compatible product flash discipline.

Required summary format:
What changed; files changed; tests/build/runtime results; deviations from this
plan; remaining issues; next suggested action.
