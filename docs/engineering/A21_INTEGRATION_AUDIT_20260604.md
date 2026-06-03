# A21 Integration Audit

Status: current integration audit.
Date: 2026-06-04 CST.
Owner: A21 control tower.

This audit verifies that the internal test 3 commits and the follow-up launch
gate work are still present on the current mainline branch. It is intentionally
read-only in spirit: it does not authorize revert, branch deletion, worktree
deletion, firmware changes, NVS changes, provider secret edits, or public
runtime mutation.

## Current Branch

- Workspace: `/Users/jiyurun/Documents/New project`
- Branch: `codex/a21-hardware-window-20260603-wifi-provisioning-flash`
- Current HEAD before this audit document:
  `8396261 docs(control): audit workspace state`
- Remote tracking:
  `origin/codex/a21-hardware-window-20260603-wifi-provisioning-flash`
- Branch state before this audit document:
  ahead of origin by 3 commits.
- Follow-up update:
  commits through `5dba606` were pushed to the remote tracking branch at
  2026-06-04 03:33 CST.

## Inclusion Matrix

These commits are all ancestors of current `HEAD`.

| Commit | Scope | Local HEAD | Remote tracking branch |
| --- | --- | --- | --- |
| `074e3d8` | Internal test 3 package source commit / suppressed listen verification | included | included |
| `221c153` | Internal test 3 release documentation | included | included |
| `b58283b` | Internal test 3 master handoff | included | included |
| `765ed41` | Xiaozhi launch-gate hardening and readiness adaptation | included | included |
| `3741c4a` | StepFun built-in route-eligibility promotion | included | included |
| `8396261` | Workspace control audit and safe cleanup | included | included |
| `5dba606` | Integration audit and no-broad-revert guardrail | included | included |
| `d9362a7` | Cloud-edge Xiaozhi evidence readiness adaptation | included | included after this control update is pushed |

High-signal internal test 3 protocol/firmware commits were also confirmed as
ancestors of current `HEAD`:

- `e17aa3d fix(gateway): drain suppressed xiaozhi listen tail`
- `862791e fix(gateway): drop suppressed xiaozhi listen audio`
- `0aa1eda fix(gateway): prevent xiaozhi speaking self loop`
- `272edea fix(gateway): treat streaming asr final as speech evidence`
- `0161d84 fix(gateway): answer after late streaming asr final`
- `27d8a34 fix(gateway): defer xiaozhi speech until listen stop`
- `aa80523 fix(firmware): allow wake on idle socket`
- `9ba8bc1 fix(firmware): skip welcome after stackchan app preload`
- `b912a82 fix(firmware): guard stackchan product flash lane`
- `c104b3c feat(gateway): add voice chain hot switch selector`

## No-Rollback Check

Diff checks from the internal test 3 release documentation commit to current
HEAD showed:

- No runtime diff in `internal/gateway/server.go`.
- No runtime diff in `internal/transport/xiaozhi`.
- No firmware diff in `firmware`.
- Follow-up code changes after the master handoff are limited to readiness,
  app evidence gates, provider catalog route eligibility, cloud-edge evidence
  absorption, and tests.
- `internal/gateway/server_test.go` gained Xiaozhi regression coverage, but
  the Gateway runtime implementation was not reverted.

Interpretation:

- Current `HEAD` keeps the internal test 3 endpoint-side protocol work.
- The follow-up commits are forward-only guardrail/evidence/provider-routing
  changes, not mass rollback.
- The newest source/control commits are intended to stay on the remote tracking
  branch after each control update is pushed.
- ECS runtime deployment has been recovered for the StepFun/cloud-edge evidence
  path; deployment truth remains separate from Git remote integration.

## Current Guardrail

Do not use branch cleanup, object-store cleanup, worktree cleanup, or stale
worker output as a reason to revert internal test 3 commits. Any future revert
must be a named, reviewed transition with exact file scope, acceptance
criteria, and rollback plan.

## Verification Commands

The audit used these command classes:

```bash
git status --short --branch
git log --oneline --decorate -12
git merge-base --is-ancestor <commit> HEAD
git branch --contains <commit>
git branch -r --contains <commit>
git diff --stat 221c153..HEAD
git diff --name-status 221c153..HEAD
git diff --stat 221c153..HEAD -- firmware internal/transport internal/gateway/server.go
git diff --check
```

`git diff --check` passed during this audit.
