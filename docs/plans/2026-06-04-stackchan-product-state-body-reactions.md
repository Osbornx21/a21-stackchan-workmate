# StackChan Product State Body Reactions

Date: 2026-06-04 CST.
Owner: A21 control tower.

## Transition

`T-STACKCHAN-OFFICIAL-STATE-BODY-REACTION-001`

## Current State

- Product touch body reactions are deployed and physically evidenced through
  bounded official Xiaozhi MCP robot head/LED tools.
- Gateway already records Xiaozhi state transitions such as `listening`,
  `thinking`, `speaking`, and `idle`.
- Those state transitions update registry/display traces, but do not yet drive
  the product device body unless the operator touches the device.

## Target State

- A separate server-side gate,
  `A21_XIAOZHI_PRODUCT_STATE_REACTIONS=true`, allows hardware-MAC stock
  Xiaozhi clients with `hello.features.mcp=true` to receive small bounded
  official MCP body reactions on state transitions.
- Reactions are limited to `self.robot.set_led_color` and
  `self.robot.set_head_angles`, reuse the existing MCP whitelist and redaction
  path, and record stable trace/runtime echo evidence.
- Touch reactions remain separate and unchanged.

## Not Doing

- No firmware build, flash, NVS write, serial write, provider execution, or V21
  execution.
- No camera, NFC, infrared, rich per-LED choreography, or app lifecycle changes.
- No claim of full PRD physical acceptance from tests alone.

## Acceptance

- Focused Gateway tests prove the product gate advertises
  `a21.state_reactions=true`, sends bounded MCP for `listening`, records
  state-reaction traces, and refuses clients without MCP.
- App env tests prove the new `A21_` env is wired.
- Docs/state/handoff describe the separate gate and remaining physical
  evidence requirement.
- `git diff --check` and `GOMAXPROCS=2 make verify` pass before commit.

## Rollback

- Disable `A21_XIAOZHI_PRODUCT_STATE_REACTIONS` at runtime.
- If code rollback is required, revert only this transition's commit and keep
  internal-test3 voice/protocol and product touch evidence intact.
