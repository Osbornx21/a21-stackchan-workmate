# A21 Workspace Control Audit

Status: current workspace audit.
Date: 2026-06-04 CST.
Owner: A21 control tower.

This document records the current Git/workspace/thread control facts. It is not
launch evidence and does not authorize deleting source material, reports,
branches, worktrees, stashes, firmware artifacts, or hardware evidence.

## Current Mainline

- Workspace: `/Users/jiyurun/Documents/New project`
- Branch: `codex/a21-hardware-window-20260603-wifi-provisioning-flash`
- Current source HEAD:
  `3741c4a feat(providers): promote stepfun route eligibility`
- Remote tracking:
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`
- Local branch state:
  ahead of origin by 2 commits.
- Current first-read control:
  `docs/engineering/A21_CURRENT_CONTROL.md`
- Current evidence manifest:
  `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
- Current active plan:
  `docs/plans/2026-06-04-stepfun-route-eligibility-promotion.md`
- Source-only remediation backlog:
  `docs/engineering/A21_GOVERNANCE_REMEDIATION_PLAN.md`

## Mainline Truth

- Internal test 3 endpoint-side voice acceptance and the associated `/v1/xiaozhi`
  protocol changes are protected baseline. Do not revert them during workspace
  cleanup.
- Public Gateway has selected StepFun in the voice-chain selector, but ECS has
  not yet been updated to commit `3741c4a` from this thread because SSH
  control-plane access is unavailable.
- StepFun source-level route eligibility is locally committed.
- A fresh remote executed StepFun provider-smoke report is still required after
  ECS deploy.
- Full PRD launch still requires physical playback ack, playback stop_done, or
  trusted audible/instrument observation.

## Git Object Store

Observed `git count-objects -vH`:

- loose object count: `7903`
- loose object size: `178.80 MiB`
- packed objects: `7587`
- pack count: `7`
- pack size: `4.74 MiB`
- garbage: `0`

Interpretation:

- The Git warning is about unreachable loose objects in `.git/objects`, not
  tracked source files and not the working tree.
- Likely causes are high worker churn, abandoned temporary commits, rebases,
  stashes, and worktree/branch fan-out.
- `git prune` or aggressive GC can remove unreachable objects permanently.
  That is useful maintenance only after dirty/source-only worktrees and stashes
  are classified.
- Safe first step is `git worktree prune` for stale worktree registrations; do
  not run object pruning as part of launch-critical work.

## Worktrees And Branches

Observed before cleanup:

- `codex/*` branches: `145`
- registered worktrees: `38`
- existing worktrees: `37`
- missing/prunable worktrees: `1`
- dirty worktrees: `4`
- branched worktrees: `19`
- detached worktrees: `18`

After safe registration cleanup:

- `git worktree prune` removed the stale missing worktree registration only.
- registered worktrees: `37`
- missing/prunable worktrees: `0`
- existing worker worktree directories were not deleted.

Known dirty worktrees:

| Branch/state | HEAD | Path | Classification |
| --- | --- | --- | --- |
| `codex/a21-hardware-window-20260603-wifi-provisioning-flash` | `3741c4a` | main workspace | current mainline; must be kept clean |
| detached | `18dc553` | `/Users/jiyurun/.codex/worktrees/2687/New project` | source-only / likely zombie |
| detached | `cef094f` | `/Users/jiyurun/.codex/worktrees/4d95/New project` | source-only / likely absorbed |
| `codex/t-hw-volume-001-official-codec-volume` | `f49abde` | `/Users/jiyurun/.codex/worktrees/bb16/New project` | historical / source-only |

Current cleanup rule:

- Remove only stale worktree registrations with `git worktree prune`.
- Do not delete existing worktree directories or branches without a dedicated
  cleanup plan and explicit approval.
- Do not merge dirty worktree content into mainline unless it is re-read,
  classified, and assigned to a current transition.

## Stashes

Current stashes:

- `stash@{0}`:
  `a21-pcm-bridge-diagnostic-only-abandoned-after-xiaozhi-direction`
- `stash@{1}`:
  `a21-realtime-evidence-leftover-docs`

Current cleanup rule:

- Keep both as source-only until a post-launch cleanup window.
- Do not apply, pop, drop, or rewrite these stashes during launch-critical work.

## Threads And Subagents

Observed A21 Codex threads:

- `019e7f81-e218-7df1-8743-1ed66e7ddd37`:
  historical server-mainline/control-tower thread.
- `019e8873-52f0-7570-8efd-04b899db7d4e`:
  2026-06-03 hardware control-tower recovery thread.

Observed subagent status:

- Previously surfaced subagent
  `019e8e3e-d974-75d0-8788-664143c0e67d` is not currently known to the
  multi-agent runtime.

Current worker rule:

- No new subagents unless the current control plan records their task, write
  boundary, forbidden actions, and return format.
- One mainline owner decides integration. Workers are source producers, not
  independent launch authorities.

## Immediate Cleanup Actions

Allowed now:

- Commit current control/audit docs.
- Add `.DS_Store` to `.gitignore`.
- Delete current `.DS_Store` working-tree noise.
- Run `git worktree prune` to remove the stale missing worktree registration.

Completed in this cleanup pass:

- `.DS_Store` added to `.gitignore`.
- Current `.DS_Store` files removed from the main workspace.
- Stale missing worktree registration pruned.

Deferred:

- Branch deletion.
- Existing worktree deletion.
- Stash deletion.
- `git prune`, `git gc --prune`, or aggressive object-store cleanup.
- Report archive moves.
- Firmware, NVS, provider secret, or public Gateway runtime mutation.

## Recovery Rule

When a new thread resumes A21 work, it should start from:

1. `AGENTS.md`
2. `docs/engineering/A21_CURRENT_CONTROL.md`
3. `docs/engineering/A21_WORKSPACE_CONTROL_AUDIT_20260604.md`
4. `docs/engineering/A21_CURRENT_EVIDENCE_MANIFEST.md`
5. The active plan named in current control

Then run:

```bash
git status --short --branch
git log --oneline -5
git worktree list --porcelain
git count-objects -vH
```

Do not infer current project truth from raw branch count, report mtime, or
historical worker directories.
