# A21 Main Recovery

Status: executed.
Date: 2026-06-06.

This note records the recovery from the lean/partial `main` tree back to the
internal test 4 product baseline. It does not authorize new feature work,
firmware flash, NVS writes, or provider changes.

## Problem

`main` was found at commit `4fa66c8`, an older lean tree that was an ancestor of
internal test 4 but did not contain the internal test 4 product surface.

Compared with internal test 4 commit
`1387d58f364f6ae1c7258487fb5a9863567adc74`, that tree showed:

- `144` deleted files;
- `91` modified files;
- about `98k` deleted lines;
- missing control tower documents, hardware plans, product recovery tools,
  official-compatible firmware overlay, workspace console, StackChan official
  avatar/body relay, and cloud voice adapters.

`go test ./...` on that tree failed in `internal/app`.

## Recovery Action

Created and pushed recovery branch:

`codex/a21-mainline-recovery-internal-test4-20260606`

The branch points to commit `090eca6`, which is:

- internal test 4 release commit `1387d58`;
- plus stabilization baseline docs;
- plus P0 power boot RCA;
- plus P0 voice-loop RCA;
- plus the provider realtime test fake concurrency fix.

`origin/main` was fast-forwarded from `4fa66c8` to `090eca6`. This was not a
force push because `4fa66c8` was an ancestor of `090eca6`.

## Verification

- Key internal test 4 capability files are present again on `origin/main`,
  including:
  - `docs/agent_handoff_log.md`;
  - `docs/project_state_machine.md`;
  - `docs/engineering/A21_CURRENT_CONTROL.md`;
  - `firmware/stackchan-official/overlays/a21-official-xiaozhi-compatible.patch`;
  - `internal/gateway/workspace_console.go`;
  - `internal/providers/dashscope_realtime.go`;
  - `internal/app/app_stackchan_product_recovery.go`.
- `GOMAXPROCS=2 go test ./...` passed on the recovery branch.
- `make verify` passed on the recovery branch.

## Current Safe Rule

Treat `090eca6` and this recovery branch as the restored mainline floor.
Cherry-pick later voice-clone or 5080 work only after diff review proves it
does not delete internal test 4 product capabilities.
