# A21 Governance Remediation Plan

Status: source-only proposed remediation plan.
Date: 2026-06-03.
Owner: A21 control tower.
Scope: governance, control-source consolidation, plan-state cleanup, evidence indexing, command-surface reduction, and zombie worktree/stash handling.

This document consolidates the 2026-06-03 governance audit into one actionable
remediation plan. It is not launch evidence, not a replacement for PRD
acceptance, and not authorization to remove safety gates. It is the backlog for
reducing governance load before and after launch.

Current control note, 2026-06-04: this file is now a source-only remediation
backlog. The active first-read control surface remains
`docs/engineering/A21_CURRENT_CONTROL.md`; the current workspace facts are in
`docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`.

## 1. Verdict

A21 is not in a P0 uncontrolled state. The important safety gates still hold:

- Current product readiness remains red for full launch.
- Host, mock, static, candidate, and server-side evidence are still separated
  from physical PRD acceptance.
- Canonical readiness still requires physical StackChan and wake-word proof.
- Firmware and hardware-write lanes still require explicit guarded execution.

The governance state is nevertheless yellow-red:

- Too many documents claim to be the active control source.
- Plan status labels no longer reliably mean current, completed, blocked, or
  source-only.
- The report directory is large enough that `latest` selection can become a
  hidden decision source.
- The Makefile and CLI expose many old, deprecated, diagnostic, product, and
  hardware-window commands at the same visual level.
- Dirty worktrees and stashes contain source material that should not be merged
  accidentally.

The remediation goal is not to make A21 smaller or less ambitious. The goal is
to make full PRD launch harder to fake and easier to route.

## 2. Current Snapshot

Observed on 2026-06-03:

- Main checkout: branch `codex/network`, HEAD `2f8a63f`.
- Main worktree: dirty before this document was added.
- Worktrees: 36 total, 18 detached, 4 dirty.
- Plans: 35 markdown files under `docs/plans`.
- Reports: 1736 files under `reports`, with 1330 JSON files at the report root.
- Server-side readiness bundles observed: 30, all `server_side_blocked`.
- Latest product-readiness report: `reports/a21-product-readiness-20260603-120713.json`.
- Latest full-launch status: `launch_ready=false`, `status=server_side_blocked`.
- Durable canonical missing evidence: `physical_stackchan_online`,
  `physical_stackchan_prd_acceptance`, and `wake_word_product_ready`.

Known source-only or zombie surfaces:

| Surface | Current classification | Reason |
| --- | --- | --- |
| `/Users/jiyurun/.codex/worktrees/2687/New project` | Source-only / likely zombie | Detached `18dc553`; dirty handoff plus duplicate stock Xiaozhi cleanup plan. |
| `/Users/jiyurun/.codex/worktrees/4d95/New project` | Source-only / absorbed | Detached `cef094f`; ASR partial to LLM/TTS work appears superseded by later committed bridge and hardening. |
| `/Users/jiyurun/.codex/worktrees/bb16/New project` | Historical / source-only | Branch `codex/t-hw-volume-001-official-codec-volume`; codec-volume direction appears superseded by later audio commits. |
| `stash@{0}` | Abandoned source material | `a21-pcm-bridge-diagnostic-only-abandoned-after-xiaozhi-direction`. |
| `stash@{1}` | Superseded source material | `a21-realtime-evidence-leftover-docs`. |

Do not remove these without explicit user approval. Until then, treat them as
read-only evidence/source material, not merge targets.

## 3. Non-Negotiables

These protections are not governance waste. Keep them unless an explicit ADR
and reviewed migration plan say otherwise.

- Full PRD launch remains the target. Do not redefine success as demo,
  host-only, server-side, static, or candidate green.
- `product-readiness` remains the canonical launch decision surface.
- `canonical_decision.launch_ready=false` and
  `canonical_decision.prd_accepted=false` must override optimistic child
  reports.
- `demo_ready=true`, `mock_demo_ready`, `server_side_blocked`,
  `candidate_host_only`, and static provider-shape `passed` are not launch
  acceptance.
- Product StackChan flashing must use the official-compatible product lane and
  artifact, not the generic Xiaozhi firmware lane.
- Plan/execute separation for firmware, NVS, serial, hardware, and provider
  execution remains mandatory.
- Provider keys stay out of firmware.
- Localhost, LAN, `.local`, StackChan, and V21 adapter traffic must keep direct
  routing and must not silently inherit global proxies.
- X21 and V21 remain guarded references/adapters only. They must not become A21
  runtime identity.
- No cleanup may delete evidence that is still needed to explain a launch or
  safety decision.

## 4. Problem Map

### P1: Competing Current Control Sources

Multiple files currently look authoritative:

- `docs/A21_SERVER_MAINLINE.md`
- `docs/engineering/A21_PROJECT_CONTROL.md`
- `docs/engineering/A21_PRD_FULL_PUSH_CONTROL_PLAN_20260602.md`
- `docs/project_state_machine.md`
- `docs/agent_handoff_log.md`

Some of these contain stale branches, stale HEADs, and older readiness context.
That makes it easy for a recovered conversation or worker to resume from the
wrong branch, reopen a completed slice, or over-trust old launch state.

Required direction:

- One current entry document.
- Old control documents explicitly marked historical/source-only.
- State machine remains the transition ledger, not a long narrative first-read
  document.
- Handoff log remains append-only evidence, not the first routing source.

### P1: Plan Status Collapse

Many plan files still say `active`, `ready`, or `planned` after the state
machine records the transition as completed. This makes `Status:` untrustworthy.

Examples of status drift:

- `2026-06-03-xiaozhi-asr-partial-to-llm-realtime-bridge.md` still says planned
  while the state machine records the host-side candidate as completed.
- `2026-06-03-sherpa-realmodel-no-audio-smoke.md` still says ready for scoped
  worker while the state machine records a completed real-model no-audio smoke.
- `2026-06-03-streaming-tts-runtime-proof.md` still says active while the state
  machine records a truthful blocker.
- `2026-06-03-xiaozhi-streaming-tts-adapter.md` still says active while the
  state machine records the adapter seam as completed.

Required direction:

- Replace free-form `Status:` values with a small enum.
- Add a plan index.
- Make every plan point to its state-machine transition and closure evidence.

### P1: Report and Latest-Selector Sprawl

The report directory is large enough that `latest` becomes dangerous when used
as an operator decision shortcut. The current code has useful safeguards that
skip mismatched latest reports, but the operator experience is still noisy.

Risks:

- A stale `real_launch_ready` historical report can be found by raw search.
- Static provider-shape `passed` can be mistaken for runtime or physical proof.
- Failed runtime provider reports can be buried under many generated files.
- `ls -t | head` and `--use-latest-reports` are convenient but too implicit for
  launch decisions.

Required direction:

- Add a current evidence manifest.
- Keep raw reports append-only, but stop treating directory mtime as authority.
- Make accepted, blocked, rejected, stale, and source-only evidence explicit.

### P1: Command Surface Sprawl

The Makefile and CLI expose many targets at one level:

- Host gates and aliases.
- Provider smoke, provider execution, 5080lab runbooks, and realtime fixtures.
- V21 adapter bridge and smoke.
- Local TTS/ASR/voice loopback.
- Official-compatible product build/flash/NVS.
- PCM bridge diagnostic lanes.
- Generic Xiaozhi firmware lanes.
- Legacy firmware upload blocker checks.

The safety guards still matter, but the surface is too broad for safe routing.

Required direction:

- Keep safety checks.
- Reduce default discoverability of deprecated or non-product lanes.
- Group commands by tier and lane.
- Add top-level help or command inventory.
- Make product-lane commands visually and semantically distinct from
  diagnostic/dev lanes.

### P2: Handoff and State Documents Are Too Large

`docs/agent_handoff_log.md` and `docs/project_state_machine.md` are valuable,
but they are too large for first-read routing. They should be evidence backing a
short current control document.

Required direction:

- Keep append-only history.
- Add rolling summaries.
- Keep only current transitions and blockers in the first-read path.

### P2: Redline Language Is Over-Distributed

Anti-fake-green language appears in many places. The repetition helped during
high-speed work, but now it creates drift risk because each copy can age
differently.

Required direction:

- Centralize launch-readiness rules.
- Reference the canonical rule instead of rewriting it everywhere.
- Preserve critical local warnings only where operator mistakes are likely.

## 5. Target Governance Shape

After remediation, a new control conversation should need to read only this
small set first:

1. `AGENTS.md`
2. `docs/engineering/A21_CURRENT_CONTROL.md`
3. `docs/project_state_machine.md` current-state section
4. `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md` or JSON equivalent
5. The one active plan named by the current control document

Everything else should be historical, reference, evidence, or source material.

Target invariants:

- Exactly one current control entry.
- Exactly one current next action.
- Every active plan has one owner and one transition ID.
- Every completed plan has closure evidence.
- Every report used for a launch decision is named in the evidence manifest.
- Deprecated or source-only commands are not listed beside product commands
  without warnings.
- Dirty worktrees and stashes are classified before any merge or cleanup.

## 6. Remediation Phases

### Phase 0: Before Launch, Do Not Destabilize

Timing: now until initial launch or field acceptance window closes.

Goal: prevent new fake green and duplicate work without doing large cleanup.

Allowed work:

- Add or maintain this remediation document.
- Add a short current-control pointer if explicitly approved.
- Add a current-evidence manifest if explicitly approved.
- Label obviously stale documents as historical if the edit is low-risk and
  approved.
- Classify worktrees and stashes read-only.

Forbidden work:

- Do not delete reports.
- Do not remove old Make targets.
- Do not collapse state machine or handoff history.
- Do not rewrite launch criteria.
- Do not change firmware, provider, V21, Gateway, or hardware behavior as part
  of governance cleanup.

Phase 0 acceptance:

- Full launch criteria are unchanged.
- Latest readiness still reports honestly.
- New workers can see that governance cleanup is intentionally deferred.

### Phase 1: Create the Single Current Control Entry

Timing: immediately after launch or after the current launch-critical hardware
window stabilizes.

Create:

- `docs/engineering/A21_CURRENT_CONTROL.md`

Required sections:

- Current branch, HEAD, and dirty-state policy.
- Current project state.
- Current active transition.
- Current blocked transitions.
- Current evidence manifest path.
- Current active plan path.
- Current owner and worker boundary.
- Commands allowed for the next transition.
- Commands explicitly forbidden for the next transition.
- Last verified command results.
- One recommended next action.

Then mark these as historical/source-only unless they are still explicitly
needed:

- `docs/A21_SERVER_MAINLINE.md`
- `docs/engineering/A21_PROJECT_CONTROL.md`
- `docs/engineering/A21_PRD_FULL_PUSH_CONTROL_PLAN_20260602.md`

Phase 1 acceptance:

- Searching `Status: active control` returns only the current entry or clearly
  historical documents.
- No stale branch/HEAD is presented as the current control checkout.
- A recovered worker can find the next transition in under two minutes.

### Phase 2: Normalize Plan Status

Timing: after Phase 1.

Create:

- `docs/plans/INDEX.md`

Use this status enum:

| Status | Meaning |
| --- | --- |
| `current` | The one plan currently authorized for active execution. |
| `active` | Approved but not the single current transition. Use sparingly. |
| `blocked` | Still desired, but waiting on named evidence, hardware, provider, or decision. |
| `completed` | Implemented or reviewed and recorded with closure evidence. |
| `superseded` | Replaced by a newer plan, commit, report, or product direction. |
| `source-only` | Useful reference material, not a merge or execution target. |
| `historical` | Preserved for audit history only. |

Every plan header should include:

```text
Status:
Transition:
Owner:
Created:
Last reviewed:
State-machine row:
Closure evidence:
Next action:
Forbidden actions:
```

Phase 2 acceptance:

- Each of the 35 current plan files is classified.
- No completed transition still has a plan marked `planned worker transition`,
  `ready for scoped worker`, or ambiguous `active` without explanation.
- `docs/project_state_machine.md` and `docs/plans/INDEX.md` agree.

### Phase 3: Add Current Evidence Manifest

Timing: after Phase 1, before any serious post-launch refactor.

Create one of:

- `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- `reports/a21-current-evidence-manifest.json`

Prefer JSON if it will later be consumed by tooling. Markdown is acceptable as
the first human-readable step.

Suggested JSON shape:

```json
{
  "schema_version": "a21.current_evidence_manifest.v1",
  "generated_for": "a21-control-tower",
  "launch_ready": false,
  "prd_accepted": false,
  "canonical_product_readiness_report": "reports/a21-product-readiness-YYYYMMDD-HHMMSS.json",
  "accepted_reports": [],
  "blocked_reports": [],
  "source_only_reports": [],
  "rejected_or_stale_reports": [],
  "missing_real_evidence": [],
  "notes": []
}
```

Manifest rules:

- The manifest names report basenames or repo-relative paths.
- It never stores provider output, credentials, raw audio, transcript text, full
  URLs, or local secret paths.
- Launch decisions read from the manifest plus explicit reports, not raw mtime.
- Historical `real_launch_ready` reports without modern canonical fields are
  marked stale.
- Static provider-shape reports are marked static-only.
- Failed runtime reports remain visible as blockers.

Phase 3 acceptance:

- A human can identify the current readiness truth without scanning 1736 report
  files.
- `--use-latest-reports` remains available for collection, but launch review
  requires explicit report paths or manifest references.

### Phase 4: Reduce Command Discoverability Risk

Timing: post-launch, after evidence manifest exists.

Do not remove guards first. Reorganize the command surface first.

Recommended command tiers:

| Tier | Meaning | Examples |
| --- | --- | --- |
| T0 | Read-only inspection | `git status`, `rg`, report reading. |
| T1 | Host-only verification | `make verify`, `make preflight`, `go test`. |
| T2 | No-execute evidence collection | readiness reports, static provider gates. |
| T3 | Local runtime | Gateway, simulator, host loopback. |
| T4 | Provider or V21 execute | explicit provider smoke, V21 adapter smoke. |
| T5 | Hardware no-write | device report, physical observation readouts. |
| T6 | Hardware write / flash / NVS | guarded product-lane flash and NVS only. |

Makefile cleanup direction:

- Keep `make verify`, `make preflight`, and `make doctor`.
- Add a `make help` or `make commands` target grouped by tier.
- Move deprecated or diagnostic-only targets below a clear warning block.
- Make the official-compatible product lane visually separate from generic
  Xiaozhi and PCM bridge lanes.
- Keep `plan` and `execute` names, but require target comments to state blast
  radius.

CLI cleanup direction:

- Add top-level `a21 help` and `a21 commands`.
- Mark deprecated aliases in help output.
- Keep deprecated aliases only as compatibility shims with warnings, or require
  an explicit `A21_ALLOW_DEPRECATED_COMMAND_ALIAS` env for dangerous old lanes.
- Ensure product flash and generic/dev flash cannot be confused by help text.

Phase 4 acceptance:

- An operator can list safe host-only commands without seeing hardware-write
  commands at the same priority.
- Deprecated aliases are still guarded, but no longer look like recommended
  paths.
- Top-level `a21 --help` or `a21 help` succeeds.

### Phase 5: Worktree and Stash Quarantine

Timing: post-launch or during a dedicated cleanup window.

Before deletion:

- Capture `git status --short --branch`.
- Capture `git rev-parse --short HEAD`.
- Capture dirty file list.
- If dirty, save a diff patch outside the active tree or record that the user
  intentionally discards it.
- Confirm whether the work is already merged, superseded, source-only, or still
  needed.

Initial classification queue:

| Item | Proposed action |
| --- | --- |
| Main worktree `codex/network` | Keep active until current owner closes or hands off. |
| Worktree `2687` | Mark source-only, then remove only after duplicate plan/handoff are confirmed captured. |
| Worktree `4d95` | Mark absorbed/source-only, preserve patch if needed, then remove after approval. |
| Worktree `bb16` | Mark historical/source-only, compare against accepted audio commits, then remove after approval. |
| `stash@{0}` | Keep as abandoned PCM diagnostic source until product lane fully stabilizes, then archive/drop with approval. |
| `stash@{1}` | Keep as leftover docs source until evidence manifest exists, then archive/drop with approval. |

Phase 5 acceptance:

- No dirty worktree disappears without an explicit classification.
- No stale branch is merged just because it contains attractive changes.
- Cleanup reduces worktree count without losing launch-critical evidence.

### Phase 6: Report Retention and Archive Policy

Timing: after the evidence manifest is adopted.

Retention policy:

- Keep launch-critical reports named by the evidence manifest.
- Keep failed reports that explain blockers until the blocker closes.
- Keep firmware flash, NVS, and hardware-write receipts indefinitely unless
  exported to a durable release archive.
- Move old mock/demo/iteration reports to dated archive folders only after the
  manifest points to current truth.
- Never delete reports as part of a normal coding slice.

Suggested directory shape:

```text
reports/
  current/
  archive/
    2026-05/
    2026-06/
  provider-live/
  provider-tts-candidate/
```

Phase 6 acceptance:

- Root `reports/` is scannable.
- Current evidence is explicit.
- Historical evidence remains recoverable.

### Phase 7: Governance Cadence

Timing: continuous after launch.

Run this cadence weekly or before every major worker fan-out:

1. Confirm branch, HEAD, and dirty state.
2. Confirm exactly one current control entry.
3. Confirm active plan count.
4. Confirm latest product-readiness status and canonical missing evidence.
5. Confirm server-side bundle status.
6. Confirm no historical `real_launch_ready` report is being used as current.
7. Confirm deprecated flash lanes are not recommended for product use.
8. Confirm dirty worktrees and stashes are classified.

Escalation colors:

| Color | Condition |
| --- | --- |
| Green | One control entry, one current plan, explicit manifest, no unclassified dirty worktrees. |
| Yellow | Some stale docs or source-only worktrees exist, but current routing is unambiguous. |
| Red | More than one active control source, launch evidence chosen by raw latest report, or stale branch treated as merge target. |
| Stop | Any host/mock/static/candidate evidence is promoted to PRD accepted without required physical evidence. |

## 7. P0/P1/P2 Remediation Backlog

### P0: Stop Conditions

These require immediate pause:

- Any report or document claims full launch while canonical missing real evidence
  is still open.
- Generic Xiaozhi firmware lane is used as the product StackChan flash lane.
- Hardware-write command runs without branch, commit, artifact, port, and
  confirmation-token evidence.
- Provider key or proxy secret appears in firmware, reports, logs, or docs.
- A stale worktree is merged without current control approval.

### P1: First Post-Launch Cleanup

1. Create `A21_CURRENT_CONTROL.md`.
2. Mark old active control docs historical/source-only.
3. Create `docs/plans/INDEX.md` and reclassify all plans.
4. Create the evidence manifest and pin current reports.
5. Add top-level CLI/Make help grouped by tier.

### P2: Ongoing Cleanup

1. Rotate or summarize `docs/agent_handoff_log.md`.
2. Shorten `docs/project_state_machine.md` first-read section.
3. Move old reports into archive folders after manifest adoption.
4. Quarantine deprecated aliases behind warnings.
5. Clean source-only worktrees and stashes after explicit approval.

## 8. Definition of Done

Governance remediation is complete when:

- A new agent can recover current A21 state from one current-control document.
- The state machine, plan index, and evidence manifest agree.
- No stale branch/HEAD appears as current truth.
- Every active plan has one transition, one owner, and one acceptance path.
- Every completed plan has closure evidence.
- Launch readiness cannot be inferred from raw report mtime.
- Command help makes blast radius obvious.
- Deprecated and diagnostic lanes are clearly source-only or guarded.
- Dirty worktrees and stashes have explicit keep/archive/remove decisions.

## 9. Recommended First Action After Launch

Do not start by deleting files.

Start with a small documentation-only transition:

```text
Transition: T-GOV-001-CURRENT-CONTROL-ENTRY
Goal: Create one current control entry and mark older active control docs as historical/source-only.
Allowed: docs-only edits.
Forbidden: code changes, runtime start/stop, provider execute, V21 execute, hardware, flash, report deletion.
Acceptance:
- `A21_CURRENT_CONTROL.md` exists and points to current branch/HEAD/dirty policy.
- Old active control docs carry a historical/source-only banner.
- The next active plan and current evidence manifest path are explicit.
```

After that, run `T-GOV-002-PLAN-INDEX` and `T-GOV-003-EVIDENCE-MANIFEST`.

This order keeps launch safety intact while reducing the control-room noise that
caused the yellow-red governance state.
